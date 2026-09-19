import { isValidElement, memo } from 'react'
import ReactMarkdown from 'react-markdown'
import type { Components } from 'react-markdown'
import remarkGfm from 'remark-gfm'
import remarkMath from 'remark-math'
import rehypeKatex from 'rehype-katex'
import { ensureKatexCss, hasMathContent, normalizeMath } from '../gaea/lib/mathText'
import { ChatCodeBlock } from './ChatCodeBlock'

/** Markdown 渲染（GFM：表格/删除线/任务列表等，基于 react-markdown） */

// 缺省代码渲染（v4.233：调用方未传 components 时生效——NovelSettingPage 等
// 直用面与聊天线同款：块级代码走 ChatCodeBlock 暗色面板+hljs 高亮+复制头，
// 行内代码交还 .md-content code 默认样式；pre 透传防双层包裹。传了
// components（聊天 GenUI 缝）则完全尊重调用方，零变化。模块级常量保证
// 引用稳定，不破坏 memo。
const defaultComponents: Components = {
  pre: ({ children }) => (isValidElement(children) ? <>{children}</> : <pre>{children}</pre>),
  code: ({ className, children }) => {
    const text = String(children ?? '').replace(/\n$/, '')
    const match = /language-([\w-]+)/.exec(className ?? '')
    const isBlock = match !== null || text.includes('\n')
    if (isBlock) return <ChatCodeBlock language={match?.[1]} text={text} />
    return <code className={className}>{children}</code>
  },
}

type Props = {
  source: string
  className?: string
  /** 可选组件覆盖（聊天 GenUI 渲染缝用；缺省 = 代码块缺省高亮面板）。 */
  components?: Components
}

// T7-4：React.memo 包裹——source 未变化时跳过重渲染，避免父级无关
// state 刷新导致整棵 markdown 子树（大文档时开销明显）重复 diff。
export const MarkdownContent = memo(function MarkdownContent({ source, className, components }: Props) {
  // v4.239 数学公式对齐（与 ChatMarkdown 同款）。
  if (hasMathContent(source)) ensureKatexCss()
  return (
    <div
      className={className}
      style={{
        fontSize: 14,
        lineHeight: 1.7,
        wordBreak: 'break-word',
      }}
    >
      <ReactMarkdown
        remarkPlugins={[remarkGfm, remarkMath]}
        rehypePlugins={[rehypeKatex]}
        components={components ?? defaultComponents}
      >
        {normalizeMath(source)}
      </ReactMarkdown>
    </div>
  )
})

// Markdown 全局样式注入
export const mdStyles = `
.md-content h1 { font-size: 20px; font-weight: 700; margin: 16px 0 8px; }
.md-content h2 { font-size: 17px; font-weight: 600; margin: 14px 0 6px; }
.md-content h3 { font-size: 15px; font-weight: 600; margin: 10px 0 4px; }
.md-content p { margin: 6px 0; }
.md-content ul, .md-content ol { margin: 8px 0; padding-left: 20px; }
.md-content li { margin: 3px 0; }
.md-content code { 
  background: rgba(128,128,128,0.15); 
  border-radius: 4px; 
  padding: 1px 5px; 
  font-size: 12px; 
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
}
.md-content pre {
  /* v4.351 令牌化：写死 rgba(0,0,0,0.3) 在亮主题成中灰脏块；surface-container 亮暗自动正确 */
  background: var(--md-sys-color-surface-container, rgba(0,0,0,0.3));
  border-radius: 8px;
  padding: 12px 16px;
  margin: 10px 0;
  overflow-x: auto;
  font-size: 12px;
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
}
.md-content pre code { background: none; padding: 0; }
.md-content strong { font-weight: 600; }
.md-content em { font-style: italic; }
.md-content del { opacity: 0.6; }
.md-content hr { border: none; border-top: 1px solid rgba(128,128,128,0.2); margin: 16px 0; }
.md-content table {
  border-collapse: collapse;
  width: 100%;
  margin: 10px 0;
  font-size: 13px;
}
.md-content th, .md-content td {
  border: 1px solid rgba(128,128,128,0.35);
  padding: 6px 10px;
  text-align: left;
  vertical-align: top;
}
.md-content th {
  background: rgba(128,128,128,0.12);
  font-weight: 600;
}
.md-content blockquote {
  margin: 10px 0;
  padding: 2px 12px;
  border-left: 3px solid rgba(128,128,128,0.4);
  color: var(--md-sys-color-text-secondary, rgba(255,255,255,0.65));
}
.md-content a { color: var(--gaea-glow, #7c8cff); }
.md-content input[type="checkbox"] { margin-right: 6px; }
`
