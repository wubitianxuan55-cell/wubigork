import { useCallback, useEffect, useState } from 'react'
import { Button, Modal, Spin } from 'antd'
import { ReloadOutlined } from '@ant-design/icons'
import { GenerateSceneIllustration } from '../../wailsjsCompat'
import { PortraitImg } from '../../components/characterlib/PortraitImg'

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
    } catch (err) {
      setError(errText(err, '配图生成失败'))
    } finally {
      setLoading(false)
    }
  }, [chapterNum])

  // 打开即加载：挂载即触发一次生成
  useEffect(() => {
    void run()
  }, [run])

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
      </div>
    </Modal>
  )
}
