// sin/SinSidePanel.tsx — 右栏创作面板（v4.263，对标办公右侧面板的标签页形态）。
//
// 四个页签把故事的「工作底稿 + 产物」聚合到一处：
//   角色 = 本故事带入的角色库角色（SinCastPanel 原样入页签）；
//   大纲 = AI 用 sin_outline 工具记下的章节走向（只读）；
//   设定 = AI 用 sin_notes 工具记下的设定集（只读）；
//   插图 = 本故事已生成的全部插图缩略图（点开看大图）。
// 玩法/插图协议/边界三段静态说明收进面板头部的问号气泡（内容不删，版面降噪）。
// 开合/页签记忆在 sinPanelState.ts（gaea.sin.* 自有键名）。
// 页签不放图标：268px 面板宽度下「图标+两字」会折行成竖排（真机走查实证），
// 纯文字+计数在密度与可读性上都是对的。

import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Button, Modal, Popover } from 'antd'
import { QuestionCircleOutlined } from '@ant-design/icons'
import V3Empty from '../../components/V3Empty'
import { readFileAsDataURL } from '../../api/image'
import { collectIllustrations, type SinGalleryItem } from './storyText'
import {
  clampSinPanelWidth, readSinPanelTab, readSinPanelWidth, SIN_PANEL_DEFAULT_WIDTH,
  writeSinPanelTab, writeSinPanelWidth, type SinSideTabId,
} from './sinPanelState'
import { SinCastPanel } from './SinCastPanel'
import type { SinCastCharacter } from './useSinCast'
import type { SinNotesDoc } from './useSinNotes'
import type { SinMessageView } from './types'

/** 缩略图：本地路径经附件读取通道转 data URL（与流内插图同口径）。 */
function SinGalleryThumb({ item, onOpen }: { item: SinGalleryItem; onOpen: (item: SinGalleryItem) => void }) {
  const [url, setUrl] = useState('')
  const [failed, setFailed] = useState(false)
  useEffect(() => {
    let live = true
    setUrl('')
    setFailed(false)
    readFileAsDataURL(item.path)
      .then((u) => { if (live) setUrl(u) })
      .catch(() => { if (live) setFailed(true) })
    return () => { live = false }
  }, [item.path])

  return (
    <img
      className={`sin-gal-thumb${failed ? ' is-broken' : ''}${url ? '' : ' is-loading'}`}
      src={url || undefined}
      alt={item.prompt || '故事插图'}
      loading="lazy"
      title={failed ? '插图读取失败' : item.prompt}
      onClick={url ? () => onOpen(item) : undefined}
    />
  )
}

function EmptyHint({ description, hint }: { description: string; hint: string }) {
  return (
    <V3Empty compact description={description} style={{ padding: '26px 0' }}>
      <span className="sin-side-empty-hint">{hint}</span>
    </V3Empty>
  )
}

export interface SinSidePanelProps {
  cast: SinCastCharacter[]
  castSaving: boolean
  onOpenPicker: () => void
  onRemoveCast: (id: string) => void
  notesDoc: SinNotesDoc
  notesError: string
  notesLoading: boolean
  messages: SinMessageView[]
  /** 故事回合进行中：重新生成禁用（避免与在途插图/流式回写互相踩）。 */
  sending: boolean
  /** 画廊「重新生成」：排队→SinIllustrate 覆盖回写→消息重载；失败经 notice 透出。 */
  onRegenerate: (item: SinGalleryItem) => Promise<void>
}

export function SinSidePanel({
  cast, castSaving, onOpenPicker, onRemoveCast,
  notesDoc, notesError, notesLoading, messages, sending, onRegenerate,
}: SinSidePanelProps) {
  const [tab, setTab] = useState<SinSideTabId>(() => readSinPanelTab())
  const [preview, setPreview] = useState<SinGalleryItem | null>(null)
  const [previewUrl, setPreviewUrl] = useState('')
  // 面板宽度：拖左缘手柄实时跟手，松手持久化（与办公 useWorkspaceLayout 同范式）
  const [width, setWidth] = useState<number>(() => readSinPanelWidth())
  const widthRef = useRef(width)
  const [resizing, setResizing] = useState(false)
  const gallery = useMemo(() => collectIllustrations(messages), [messages])
  // 重新生成进行中的条目（key 集合）：缩略图/大图按钮进 busy，防重复点火
  const [regenKeys, setRegenKeys] = useState<ReadonlySet<string>>(new Set())
  const regenerate = useCallback((item: SinGalleryItem) => {
    if (sending || regenKeys.has(item.key) || !item.prompt.trim()) return
    setRegenKeys((prev) => new Set(prev).add(item.key))
    void onRegenerate(item).finally(() => {
      setRegenKeys((prev) => {
        if (!prev.has(item.key)) return prev
        const next = new Set(prev)
        next.delete(item.key)
        return next
      })
    })
  }, [onRegenerate, regenKeys, sending])

  const startResize = useCallback((e: React.PointerEvent<HTMLDivElement>) => {
    e.preventDefault()
    const startX = e.clientX
    const startWidth = widthRef.current
    setResizing(true)
    const onMove = (me: PointerEvent) => {
      // 手柄在面板左缘：向左拖（clientX 变小）= 变宽
      const next = clampSinPanelWidth(startWidth + (startX - me.clientX))
      widthRef.current = next
      setWidth(next)
    }
    const onDone = () => {
      writeSinPanelWidth(widthRef.current)
      setResizing(false)
      window.removeEventListener('pointermove', onMove)
      window.removeEventListener('pointerup', onDone)
      window.removeEventListener('pointercancel', onDone)
      document.body.style.cursor = ''
      document.body.style.userSelect = ''
    }
    document.body.style.cursor = 'col-resize'
    document.body.style.userSelect = 'none'
    window.addEventListener('pointermove', onMove)
    window.addEventListener('pointerup', onDone)
    window.addEventListener('pointercancel', onDone)
  }, [])

  // 大图预览：点击缩略图打开；关闭即丢弃 data URL
  useEffect(() => {
    if (!preview) return
    let live = true
    setPreviewUrl('')
    readFileAsDataURL(preview.path)
      .then((u) => { if (live) setPreviewUrl(u) })
      .catch(() => { if (live) setPreviewUrl('') })
    return () => { live = false }
  }, [preview])

  // 重新生成成功后消息重载、画廊重算——大图预览若还开着，跟到新条目（新图）
  useEffect(() => {
    if (!preview) return
    const next = gallery.find((g) => g.key === preview.key)
    if (next && next.path !== preview.path) setPreview(next)
  }, [gallery, preview])

  useEffect(() => {
    writeSinPanelTab(tab)
  }, [tab])

  const tabs: Array<{ id: SinSideTabId; label: string; count: number }> = [
    { id: 'cast', label: '角色', count: cast.length },
    { id: 'outline', label: '大纲', count: notesDoc.outline ? 1 : 0 },
    { id: 'notes', label: '设定', count: notesDoc.notes.length },
    { id: 'gallery', label: '插图', count: gallery.length },
  ]

  return (
    <aside className="sin-side" style={{ width, flexBasis: width }}>
      <div
        className={`sin-side-resizer${resizing ? ' is-resizing' : ''}`}
        role="separator"
        aria-orientation="vertical"
        aria-label="调整面板宽度"
        title="拖拽调整宽度 · 双击复位"
        onPointerDown={startResize}
        onDoubleClick={() => {
          setWidth(SIN_PANEL_DEFAULT_WIDTH)
          widthRef.current = SIN_PANEL_DEFAULT_WIDTH
          writeSinPanelWidth(SIN_PANEL_DEFAULT_WIDTH)
        }}
      />
      <div className="sin-side-tabs" role="tablist" aria-label="创作面板">
        {tabs.map((t) => (
          <button
            key={t.id}
            type="button"
            role="tab"
            aria-selected={tab === t.id}
            className={`sin-side-tab${tab === t.id ? ' is-active' : ''}`}
            onClick={() => setTab(t.id)}
          >
            <span>{t.label}</span>
            {t.count > 0 && <span className="sin-side-tab-count">{t.count}</span>}
          </button>
        ))}
        <Popover
          placement="leftTop"
          trigger="click"
          overlayClassName="sin-help-popover"
          content={
            <div className="sin-help-body">
              <div className="sin-help-title">玩法</div>
              <ol className="sin-card-list">
                <li>先给设定：谁、在哪、什么关系、从哪一幕开始。</li>
                <li>原罪写完一段会停下等你——说「继续」「换场景」「她来主导」即可接着走。</li>
                <li>正文里出现插图位时会自动出图；也可点「重新生成」换一张。</li>
              </ol>
              <div className="sin-help-title">插图协议</div>
              <p className="sin-card-text">
                模型在需要配图处另起一行输出 <code>@@插图|画面描述@@</code>，
                原罪就地生成插图并嵌进正文；导出的 Markdown 会把这些位置写成图片链接。
              </p>
              <div className="sin-help-title">边界</div>
              <p className="sin-card-text">
                成人内容默认开启（个人非商用桌面应用）：故事角色一律成年人，合意基调，
                不写真实在世人物；你说停就停。
              </p>
            </div>
          }
        >
          <button type="button" className="sin-side-help" aria-label="玩法与协议说明" title="玩法与协议说明">
            <QuestionCircleOutlined />
          </button>
        </Popover>
      </div>

      <div
        className="sin-side-body"
        role="tabpanel"
        aria-label={tabs.find((t) => t.id === tab)?.label}
      >
        {tab === 'cast' && (
          <SinCastPanel
            cast={cast}
            saving={castSaving}
            onOpenPicker={onOpenPicker}
            onRemove={onRemoveCast}
          />
        )}

        {tab === 'outline' && (
          <section className="sin-card">
            {notesDoc.outline ? (
              <div className="sin-outline-text">{notesDoc.outline}</div>
            ) : (
              <EmptyHint
                description="还没有大纲"
                hint="对话里说「先出个大纲」，AI 会把章节走向、时间线和伏笔记在这里。"
              />
            )}
          </section>
        )}

        {tab === 'notes' && (
          <section className="sin-card">
            {notesError ? (
              <p className="sin-side-error">{notesError}</p>
            ) : notesDoc.notes.length > 0 ? (
              <div className="sin-note-list">
                {notesDoc.notes.map((n, i) => (
                  <div className="sin-note-item" key={i}>
                    <span className="sin-note-idx">#{i}</span>
                    <span className="sin-note-text">{n}</span>
                  </div>
                ))}
              </div>
            ) : (
              <EmptyHint
                description="还没有设定便签"
                hint="故事里定下的人名、关系、伏笔，AI 会用便签记在这里，写作时自动对齐。"
              />
            )}
          </section>
        )}

        {tab === 'gallery' && (
          <section className="sin-card">
            {gallery.length > 0 ? (
              <div className="sin-gal-grid">
                {gallery.map((item) => (
                  <figure className="sin-gal-cell" key={item.key}>
                    <div className={`sin-gal-media${regenKeys.has(item.key) ? ' is-regen' : ''}`}>
                      <SinGalleryThumb item={item} onOpen={setPreview} />
                      {regenKeys.has(item.key) ? (
                        <span className="sin-gal-regen is-busy">重新生成中…</span>
                      ) : item.prompt ? (
                        <button
                          type="button"
                          className="sin-gal-regen"
                          disabled={sending}
                          title={sending ? '故事生成中，稍后再试' : '按原画面描述重新生成这张插图'}
                          onClick={() => regenerate(item)}
                        >
                          重新生成
                        </button>
                      ) : null}
                    </div>
                    {item.prompt && (
                      <figcaption className="sin-gal-cap" title={item.prompt}>{item.prompt}</figcaption>
                    )}
                  </figure>
                ))}
              </div>
            ) : (
              <EmptyHint
                description="还没有插图"
                hint={notesLoading ? '正在读取…' : '正文里出现插图位时会自动出图，生成后都收在这里。'}
              />
            )}
          </section>
        )}
      </div>

      <Modal
        open={preview !== null}
        title={preview?.prompt || '故事插图'}
        footer={preview && (
          <Button
            size="small"
            disabled={sending || !preview.prompt.trim() || regenKeys.has(preview.key)}
            loading={regenKeys.has(preview.key)}
            title={!preview.prompt.trim() ? '该插图没有画面描述，无法重新生成' : undefined}
            onClick={() => regenerate(preview)}
          >
            重新生成
          </Button>
        )}
        width={720}
        onCancel={() => setPreview(null)}
        centered
      >
        {previewUrl
          ? <img className="sin-gal-preview-img" src={previewUrl} alt={preview?.prompt || '故事插图'} />
          : <div className="sin-gal-preview-loading">正在读取插图…</div>}
      </Modal>
    </aside>
  )
}
