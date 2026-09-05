// AssetLibrary（T1 收口）：画室素材库独立页/视图——从 TaskCenter 的窄栏 tab
// 升级为 imagegen 模块内可全幅浏览的素材库页。
//   - 数据源：复用 ImageHubAssets 只读绑定（零新绑定），按空间读 ledger
//     （work=办公空间为缺省，play=创作空间可切换；两空间各自隔离不合并）；
//   - 原语/来源筛选：绑定面无 capability 参数，原语筛在前端对 capability
//     字段做客户端过滤（登记视图自带该字段）；
//   - 懒加载：登记列表一次取 120 条，缩略 dataURL 只转换当前可见页
//     （12 张/页，「加载更多」递增），已转换结果按 path 缓存不重读文件；
//   - 溯源元数据：模型/成本/来源/角色(params.character_id)/时间/提示词/路径。
// 图示（.mmd 代码产物）无位图缩略 → 诚实占位（缩略图不可用），不伪装。
import React, { useEffect, useMemo, useRef, useState } from 'react'
import { Button, Segmented, Tag, Typography } from 'antd'
import {
  ArrowLeftOutlined, FileImageOutlined, PlayCircleOutlined,
} from '@ant-design/icons'
import { C } from '../../utils/theme'
import { imageHubAssets, readFileAsDataURL, type ImageHubAssetView } from '../../api/image'
import { useT } from '../../gaea/lib/i18n'
import type { DictKey } from '../../gaea/locales/en'

interface AssetThumb extends ImageHubAssetView {
  url: string
}

const LIB_FETCH_LIMIT = 120
const LIB_PAGE_SIZE = 12

/** 原语 → 展示名（未知原语诚实显示原始值，不猜）。 */
function capabilityLabel(t: (k: DictKey) => string, cap?: string): string {
  if (cap === 'media.generate') return t('imagehubT1.capMediaGenerate')
  if (cap === 'media.diagram') return t('imagehubT1.capMediaDiagram')
  return cap || ''
}

/** 来源板块 → 展示名（未知板块诚实显示原始值）。 */
function boardLabel(t: (k: DictKey) => string, board?: string): string {
  if (board === 'imagegen') return t('imagehubT1.boardImagegen')
  if (board === 'novel') return t('imagehubT1.boardNovel')
  if (board === 'characterlib') return t('imagehubT1.boardCharacterlib')
  return board || ''
}

/** 溯源角色：params.character_id（参考槽/剧照登记带入）。 */
function roleOf(r: ImageHubAssetView): string {
  const id = r.params?.character_id
  return typeof id === 'string' ? id : ''
}

export const AssetLibrary: React.FC<{ onClose: () => void }> = ({ onClose }) => {
  const t = useT()
  const [space, setSpace] = useState<'work' | 'play'>('work')
  const [records, setRecords] = useState<ImageHubAssetView[]>([])
  const [loading, setLoading] = useState(false)
  const [capability, setCapability] = useState('')
  const [source, setSource] = useState('')
  const [visibleCount, setVisibleCount] = useState(LIB_PAGE_SIZE)
  const [thumbs, setThumbs] = useState<AssetThumb[]>([])
  const [selected, setSelected] = useState<ImageHubAssetView | null>(null)
  const urlCacheRef = useRef<Map<string, string>>(new Map())

  // 空间切换 → 重读该空间 ledger（登记按空间隔离，play/work 各自独立）。
  useEffect(() => {
    let cancelled = false
    setLoading(true)
    void (async () => {
      const list = await imageHubAssets(space, '', LIB_FETCH_LIMIT)
      if (cancelled) return
      setRecords(list)
      setVisibleCount(LIB_PAGE_SIZE)
      setSelected(null)
      setCapability('')
      setSource('')
      setLoading(false)
    })()
    return () => { cancelled = true }
  }, [space])

  const filtered = useMemo(() => records.filter((r) =>
    (!capability || r.capability === capability) && (!source || r.source_board === source),
  ), [records, capability, source])

  const visible = useMemo(() => filtered.slice(0, visibleCount), [filtered, visibleCount])

  // 懒加载：只转换当前可见页的缩略 dataURL（失败/图示代码 = 空串占位）。
  useEffect(() => {
    let cancelled = false
    void (async () => {
      for (const r of visible) {
        const p = r.path || ''
        if (!p || urlCacheRef.current.has(p)) continue
        const url = await readFileAsDataURL(p).catch(() => '')
        if (cancelled) return
        urlCacheRef.current.set(p, url || '')
      }
      if (cancelled) return
      setThumbs(visible.map((r) => ({ ...r, url: (r.path && urlCacheRef.current.get(r.path)) || '' })))
    })()
    return () => { cancelled = true }
  }, [visible])

  const capabilityOptions = useMemo(() => {
    const caps: string[] = []
    for (const r of records) {
      if (r.capability && !caps.includes(r.capability)) caps.push(r.capability)
    }
    return [
      { label: t('imagehubT1.filterAll'), value: '' },
      ...caps.map((c) => ({ label: capabilityLabel(t, c), value: c })),
    ]
  }, [records, t])

  const sourceOptions = useMemo(() => {
    const boards: string[] = []
    for (const r of records) {
      if (r.source_board && !boards.includes(r.source_board)) boards.push(r.source_board)
    }
    return [
      { label: t('imagehubT1.filterAll'), value: '' },
      ...boards.map((b) => ({ label: boardLabel(t, b), value: b })),
    ]
  }, [records, t])

  const selectedThumb = thumbs.find((x) => x.path === selected?.path) || null

  return (
    <div className="ig-asset-library v3-zone" aria-label={t('imagehubT1.libraryTitle')}
      style={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column', overflow: 'hidden', padding: '10px 12px' }}>
      {/* 顶条：返回 + 标题 + 计数 + 空间切换 */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 10, flexShrink: 0 }}>
        <Button size="small" icon={<ArrowLeftOutlined />} onClick={onClose}
          style={{ borderRadius: 999, fontSize: 12 }}>
          {t('imagehubT1.libraryBack')}
        </Button>
        <Typography.Text style={{ fontSize: 13, fontWeight: 600 }}>
          {t('imagehubT1.libraryTitle')}
        </Typography.Text>
        <Typography.Text style={{ fontSize: 11, color: C('color-text-secondary') }}>
          {t('imagehubT1.count', { n: filtered.length })}
        </Typography.Text>
        <div style={{ marginLeft: 'auto' }}>
          <Segmented
            size="small"
            value={space}
            onChange={(v) => setSpace(v as 'work' | 'play')}
            options={[
              { label: t('imagehubT1.spaceWork'), value: 'work' },
              { label: t('imagehubT1.spacePlay'), value: 'play' },
            ]}
          />
        </div>
      </div>

      {/* 筛选行：原语 + 来源（来自登记数据的原语/板块集合） */}
      <div style={{ display: 'flex', alignItems: 'center', flexWrap: 'wrap', gap: 8, marginTop: 8, flexShrink: 0 }}>
        <Typography.Text style={{ fontSize: 11, color: C('color-text-secondary') }}>
          {t('imagehubT1.filterPrimitive')}
        </Typography.Text>
        <Segmented size="small" value={capability} onChange={(v) => setCapability(v as string)}
          options={capabilityOptions} />
        <Typography.Text style={{ fontSize: 11, color: C('color-text-secondary') }}>
          {t('imagehubT1.filterSource')}
        </Typography.Text>
        <Segmented size="small" value={source} onChange={(v) => setSource(v as string)}
          options={sourceOptions} />
      </div>

      {/* 溯源详情条（选中卡片时展示完整元数据） */}
      {selected && (
        <div style={{
          marginTop: 8, padding: '8px 10px', borderRadius: 10, flexShrink: 0,
          border: '1px solid var(--border-subtle)', background: 'rgba(255,255,255,0.03)',
        }}>
          <div style={{ display: 'flex', gap: 10 }}>
            {selectedThumb?.url && (
              <img src={selectedThumb.url} alt={selected.path || '素材'}
                style={{ maxWidth: 220, maxHeight: 140, objectFit: 'contain', borderRadius: 8, flexShrink: 0 }} />
            )}
            <div style={{ display: 'flex', flexDirection: 'column', gap: 3, minWidth: 0, fontSize: 11 }}>
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4 }}>
                {capabilityLabel(t, selected.capability) && (
                  <Tag color="geekblue" style={{ marginInlineEnd: 0 }}>{capabilityLabel(t, selected.capability)}</Tag>
                )}
                {selected.kind === 'video' && <Tag color="orange" style={{ marginInlineEnd: 0 }}>{t('imagehubT1.kindVideo')}</Tag>}
                {selected.space && <Tag style={{ marginInlineEnd: 0 }}>{selected.space === 'play' ? t('imagehubT1.spacePlay') : t('imagehubT1.spaceWork')}</Tag>}
              </div>
              <span><Typography.Text type="secondary">{t('imagehubT1.metaModel')}: </Typography.Text>{selected.model || '-'}</span>
              <span><Typography.Text type="secondary">{t('imagehubT1.metaCost')}: </Typography.Text>{selected.cost || t('imagehubT1.costUnset')}</span>
              <span><Typography.Text type="secondary">{t('imagehubT1.metaSource')}: </Typography.Text>{boardLabel(t, selected.source_board) || '-'}</span>
              <span><Typography.Text type="secondary">{t('imagehubT1.metaRole')}: </Typography.Text>{roleOf(selected) || '-'}</span>
              <span><Typography.Text type="secondary">{t('imagehubT1.metaTime')}: </Typography.Text>{selected.created_at || '-'}</span>
              {selected.prompt_truncate && (
                <span style={{ minWidth: 0 }}>
                  <Typography.Text type="secondary">{t('imagehubT1.metaPrompt')}: </Typography.Text>
                  <Typography.Text style={{ fontSize: 11, wordBreak: 'break-all' }}>{selected.prompt_truncate}</Typography.Text>
                </span>
              )}
              {selected.path && (
                <span style={{ minWidth: 0 }}>
                  <Typography.Text type="secondary">{t('imagehubT1.metaPath')}: </Typography.Text>
                  <Typography.Text style={{ fontSize: 11, wordBreak: 'break-all' }}>{selected.path}</Typography.Text>
                </span>
              )}
            </div>
          </div>
        </div>
      )}

      {/* 素材网格（懒加载分页） */}
      <div style={{ flex: 1, minHeight: 0, overflowY: 'auto', marginTop: 10 }}>
        {records.length === 0 && !loading && (
          <div className="ig-task-empty">
            <FileImageOutlined style={{ fontSize: 20, color: C('color-text-secondary'), opacity: 0.6 }} />
            <span>{t('imagehubT1.empty')}</span>
          </div>
        )}
        {records.length > 0 && filtered.length === 0 && (
          <div className="ig-task-empty">
            <FileImageOutlined style={{ fontSize: 20, color: C('color-text-secondary'), opacity: 0.6 }} />
            <span>{t('imagehubT1.emptyFiltered')}</span>
          </div>
        )}
        {filtered.length > 0 && (
          <>
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(140px, 1fr))', gap: 8 }}>
              {thumbs.map((a) => (
                <button key={a.path} type="button" onClick={() => setSelected(a)} title={a.path || ''}
                  style={{
                    padding: 4, textAlign: 'left', cursor: 'pointer',
                    border: '1px solid',
                    borderColor: selected?.path === a.path ? 'var(--color-primary)' : 'var(--border-subtle)',
                    borderRadius: 10, background: 'rgba(255,255,255,0.03)',
                  }}>
                  <div style={{ position: 'relative' }}>
                    {a.url ? (
                      <img src={a.url} alt={a.path || '素材'}
                        style={{ width: '100%', aspectRatio: '1 / 1', objectFit: 'cover', borderRadius: 7, display: 'block' }} />
                    ) : (
                      <div style={{
                        width: '100%', aspectRatio: '1 / 1', borderRadius: 7, display: 'flex',
                        flexDirection: 'column', alignItems: 'center', justifyContent: 'center', gap: 4,
                        background: 'rgba(255,255,255,0.04)',
                      }}>
                        <FileImageOutlined style={{ fontSize: 18, color: C('color-text-secondary'), opacity: 0.6 }} />
                        <span style={{ fontSize: 10, color: C('color-text-secondary') }}>
                          {t('imagehubT1.thumbUnavailable')}
                        </span>
                      </div>
                    )}
                    {a.kind === 'video' && (
                      <PlayCircleOutlined style={{
                        position: 'absolute', right: 6, bottom: 6, fontSize: 16,
                        color: 'var(--color-text)', filter: 'drop-shadow(0 1px 2px rgba(0,0,0,0.6))',
                      }} />
                    )}
                  </div>
                  <div style={{ display: 'flex', flexWrap: 'wrap', gap: 3, marginTop: 4 }}>
                    {a.model && <Tag color="blue" style={{ marginInlineEnd: 0, fontSize: 10, lineHeight: '16px', borderRadius: 999 }}>{a.model}</Tag>}
                    <Tag style={{ marginInlineEnd: 0, fontSize: 10, lineHeight: '16px', borderRadius: 999 }}>{a.cost || t('imagehubT1.costUnset')}</Tag>
                    {a.kind === 'video' && <Tag color="orange" style={{ marginInlineEnd: 0, fontSize: 10, lineHeight: '16px', borderRadius: 999 }}>{t('imagehubT1.kindVideo')}</Tag>}
                  </div>
                  <div style={{ fontSize: 10, color: C('color-text-secondary'), marginTop: 3, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                    {[boardLabel(t, a.source_board), a.created_at].filter(Boolean).join(' · ')}
                  </div>
                </button>
              ))}
            </div>
            {filtered.length > visibleCount && (
              <div style={{ display: 'flex', justifyContent: 'center', padding: '10px 0' }}>
                <Button size="small" onClick={() => setVisibleCount((n) => n + LIB_PAGE_SIZE)}
                  style={{ borderRadius: 999, fontSize: 12 }}>
                  {t('imagehubT1.loadMore')}
                </Button>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  )
}
