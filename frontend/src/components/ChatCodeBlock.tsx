import { useCallback, useState } from 'react'
import { message } from 'antd'
import { CheckOutlined, CopyOutlined } from '@ant-design/icons'
import { C } from '../utils/theme'
import { HlCode } from '../gaea/components/HlCode'
import { CodeCollapse } from '../gaea/components/CodeCollapse'
import { copyToClipboard } from '../gaea/lib/clipboard'
import { ChatMermaid } from './chat/ChatMermaid'

/** 聊天线代码块（ChatMarkdown plain 模式与 companion 模式 genuiAdapter 共用）：
 *  语言标签+复制头部 + 暗色面板 + hljs 高亮。面板为行业标准暗色专用色
 *  （#0b0e14/#e2e8f0 hex-exempt 在册，不随主题），故根节点挂 .hl-scope-dark
 *  把高亮令牌整体钉死暗色调色板——令牌随面板不随应用主题（hljs-theme.css）。
 *  C3：mermaid 围栏走图表/源码双态（ChatMermaid 渐进容错，chat-demo 配方）。 */

function CodeBlockHeader({ language, text, mermaid, showDiagram, onToggleDiagram }: {
  language?: string
  text: string
  mermaid?: boolean
  showDiagram: boolean
  onToggleDiagram: () => void
}) {
  const [copied, setCopied] = useState(false)
  const copy = useCallback(async () => {
    // v4.513 诚实反馈：成功才翻转「已复制」，失败可见提示（不再假成功）。
    if (await copyToClipboard(text)) {
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    } else {
      message.warning('复制失败：剪贴板不可用')
    }
  }, [text])
  return (
    <div style={{
      display: 'flex', alignItems: 'center', justifyContent: 'space-between',
      padding: '5px 12px',
      background: 'rgba(0,0,0,0.28)',
      borderBottom: '1px solid rgba(255,255,255,0.08)',
      fontSize: 10.5, userSelect: 'none',
    }}>
      <span style={{ color: C('color-text-secondary'), fontFamily: 'monospace', fontWeight: 500, textTransform: 'uppercase', letterSpacing: 0.5 }}>
        {mermaid ? (
          // C3：图表/源码双态切换（chat-demo CodeInfoRow 同位配方）
          <span style={{ display: 'inline-flex', gap: 2 }}>
            <button type="button" aria-pressed={showDiagram} onClick={() => !showDiagram && onToggleDiagram()}
              style={{ border: 'none', background: 'transparent', padding: '2px 6px', borderRadius: 6, cursor: 'pointer', color: showDiagram ? C('color-text') : C('color-text-secondary'), fontSize: 10.5 }}>
              图表
            </button>
            <button type="button" aria-pressed={!showDiagram} onClick={() => showDiagram && onToggleDiagram()}
              style={{ border: 'none', background: 'transparent', padding: '2px 6px', borderRadius: 6, cursor: 'pointer', color: !showDiagram ? C('color-text') : C('color-text-secondary'), fontSize: 10.5 }}>
              源码
            </button>
          </span>
        ) : (
          language || 'text'
        )}
      </span>
      <button
        onClick={copy}
        title="复制代码"
        style={{
          display: 'inline-flex', alignItems: 'center', gap: 4, cursor: 'pointer',
          border: 'none', background: 'transparent', padding: '2px 6px', borderRadius: 6,
          color: copied ? 'var(--md-sys-color-success)' : C('color-text-secondary'), fontSize: 11,
          transition: 'color 0.15s',
        }}
        onMouseEnter={(e) => { if (!copied) e.currentTarget.style.color = C('color-text') }}
        onMouseLeave={(e) => { e.currentTarget.style.color = copied ? 'var(--md-sys-color-success)' : C('color-text-secondary') }}
      >
        {copied ? <CheckOutlined style={{ fontSize: 10 }} /> : <CopyOutlined style={{ fontSize: 10 }} />}
        {copied ? '已复制' : '复制'}
      </button>
    </div>
  )
}

export function ChatCodeBlock({ language, text, mermaid = false }: { language?: string; text: string; mermaid?: boolean }) {
  // C3：mermaid 图表/源码双态，默认图表；渲染失败可在头部切「源码」看原文
  const [showDiagram, setShowDiagram] = useState(true)
  return (
    /* hex-exempt 行业标准暗色代码面板（不随主题，专用色在册）；fade 同色 */
    <div className="hl-scope-dark" style={{ margin: '10px 0', borderRadius: 10, overflow: 'hidden', border: '1px solid rgba(255,255,255,0.09)', background: '#0b0e14' }}> {/* hex-exempt 行业标准暗色代码面板（不随主题） */}
      <CodeBlockHeader language={language} text={text} mermaid={mermaid} showDiagram={showDiagram} onToggleDiagram={() => setShowDiagram(show => !show)} />
      {mermaid && showDiagram ? (
        <div style={{ background: '#0b0e14' }}> {/* hex-exempt 随专用暗色面板 */}
          <ChatMermaid code={text} />
        </div>
      ) : (
        <CodeCollapse text={text} fade="#0b0e14"> {/* hex-exempt 渐隐色随专用暗色面板 */}
          <pre style={{ padding: '10px 12px', margin: 0, overflow: 'auto', fontFamily: "'Cascadia Code', Consolas, monospace", fontSize: 12.5, lineHeight: 1.55, color: '#e2e8f0', whiteSpace: 'pre' }}><HlCode text={text} lang={language} /></pre> {/* hex-exempt 行业标准暗色代码面板（不随主题） */}
        </CodeCollapse>
      )}
    </div>
  )
}
