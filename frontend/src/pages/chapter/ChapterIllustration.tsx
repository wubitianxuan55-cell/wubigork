import { useCallback, useEffect, useState } from 'react'
import { Button, Modal, Spin } from 'antd'
import { ReloadOutlined } from '@ant-design/icons'
import { GenerateSceneIllustration } from '../../wailsjsCompat'
import { PortraitImg } from '../../components/characterlib/PortraitImg'
import { chapterArtList, readFileAsDataURL } from '../../api/image'

/**
 * 章节配图（v4.3g 图文联动前端）：为指定章节生成场景插图。
 *
 * 调用层：复用既有死绑定 GenerateSceneIllustration —— wails 生成物 NovelB 门面
 * 已含该方法（wailsjsCompat 直调 window.go.app.NovelB.GenerateSceneIllustration），
 * 后端返回契约 { url, revised_prompt }。
 * url 可能是远端地址 / data URL / 本地路径，统一经 PortraitImg 预览
 * （本地路径走 bridge 的 AttachmentDataURL 读取兜底）。
 */

/** GenerateSceneIllustration 返回契约的最小收窄（生成 d.ts 为 Record<string, any>）。 */
interface SceneIllustrationResult {
  url?: unknown
  revised_prompt?: unknown
}

interface ChapterIllustrationProps {
  /** 章节号（须 >= 1；< 1 视为无效输入，直接展示守卫错误，不调用后端） */
  chapterNum: number
  /** 关闭弹窗（父组件负责卸载本组件） */
  onClose: () => void
}

function errText(err: unknown, fallback: string): string {
  return (err instanceof Error && err.message) || fallback
}

export function ChapterIllustration({ chapterNum, onClose }: ChapterIllustrationProps) {
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [result, setResult] = useState<{ url: string; revisedPrompt: string } | null>(null)
  // T1 画室素材：本章历史配图（后端落盘后登记，本地路径转 data URL 展示）。
  const [history, setHistory] = useState<{ path: string; url: string; createdAt: string }[]>([])
  const [activeArt, setActiveArt] = useState(0)

  const loadHistory = useCallback(async () => {
    if (chapterNum < 1) return
    const entries = await chapterArtList(chapterNum)
    const items: { path: string; url: string; createdAt: string }[] = []
    for (const e of entries.slice(0, 6)) {
      if (!e.path) continue
      const url = await readFileAsDataURL(e.path).catch(() => '')
      if (url) items.push({ path: e.path, url, createdAt: e.created_at || '' })
    }
    setHistory(items)
  }, [chapterNum])

  const run = useCallback(async () => {
    if (chapterNum < 1) {
      setLoading(false)
      setError('无章节号，无法生成配图')
      setResult(null)
      return
    }
    setLoading(true)
    setError(null)
    setResult(null)
    try {
      const data = (await GenerateSceneIllustration(chapterNum)) as SceneIllustrationResult
      const url = typeof data.url === 'string' ? data.url : ''
      if (!url) throw new Error('未生成图片')
      const revisedPrompt = typeof data.revised_prompt === 'string' ? data.revised_prompt : ''
      setResult({ url, revisedPrompt })
      setActiveArt(0)
      void loadHistory()
    } catch (err) {
      setError(errText(err, '配图生成失败'))
    } finally {
      setLoading(false)
    }
  }, [chapterNum, loadHistory])

  // 打开即加载：挂载即触发一次生成
  useEffect(() => {
    void run()
    void loadHistory()
  }, [run, loadHistory])

  return (
    <Modal
      open
      title={`章节配图 · 第${chapterNum}章`}
      onCancel={onClose}
      footer={null}
      width={460}
    >
      <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 12, paddingTop: 8 }}>
        {loading ? (
          <div style={{ padding: '48px 0', textAlign: 'center' }}>
            <Spin size="large" />
            <div style={{ marginTop: 12, fontSize: 12, color: 'var(--color-text-secondary)' }}>
              正在为第{chapterNum}章生成配图…
            </div>
          </div>
        ) : error ? (
          <div style={{ textAlign: 'center', padding: '16px 0' }}>
            <div style={{ color: 'var(--color-destructive)', marginBottom: 12 }} role="alert">{error}</div>
            <Button icon={<ReloadOutlined />} onClick={() => void run()}>重试</Button>
          </div>
        ) : result ? (
          <div style={{ textAlign: 'center' }}>
            <PortraitImg
              src={result.url}
              alt={`第${chapterNum}章配图`}
              style={{ maxWidth: 360, maxHeight: 360, borderRadius: 8 }}
            />
            {result.revisedPrompt && (
              <div
                style={{
                  marginTop: 12,
                  fontSize: 12,
                  color: 'var(--color-text-secondary)',
                  textAlign: 'left',
                  wordBreak: 'break-all',
                }}
              >
                {result.revisedPrompt}
              </div>
            )}
            <div style={{ marginTop: 12 }}>
              <Button icon={<ReloadOutlined />} onClick={() => void run()}>重试</Button>
            </div>
          </div>
        ) : null}
        {!loading && history.length > 0 && (
          <div style={{ width: '100%', marginTop: 12 }}>
            <div style={{ fontSize: 11, color: 'var(--color-text-secondary)', marginBottom: 6 }}>
              本章历史配图（{history.length}）
            </div>
            <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap', justifyContent: 'center' }}>
              {history.map((h, i) => (
                <button
                  key={h.path}
                  type="button"
                  onClick={() => setActiveArt(i)}
                  title={`${h.createdAt} · ${h.path}`}
                  style={{
                    padding: 2, border: '1px solid',
                    borderColor: i === activeArt ? 'var(--color-primary)' : 'var(--border-subtle)',
                    borderRadius: 8, background: 'rgba(255,255,255,0.03)', cursor: 'pointer',
                  }}
                >
                  <img
                    src={h.url}
                    alt={`历史配图 ${i + 1}`}
                    style={{ width: 56, height: 42, objectFit: 'cover', borderRadius: 6, display: 'block' }}
                  />
                </button>
              ))}
            </div>
            {history[activeArt] && (
              <div style={{ marginTop: 8, textAlign: 'center' }}>
                <PortraitImg
                  src={history[activeArt].url}
                  alt="历史配图预览"
                  style={{ maxWidth: 320, maxHeight: 240, borderRadius: 8 }}
                />
              </div>
            )}
          </div>
        )}
      </div>
    </Modal>
  )
}
