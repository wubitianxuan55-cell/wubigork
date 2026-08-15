import { memo } from 'react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'

/** Markdown 渲染（GFM：表格/删除线/任务列表等，基于 react-markdown） */

type Props = {
  source: string
  className?: string
}

// T7-4：React.memo 包裹——source 未变化时跳过重渲染，避免父级无关
// state 刷新导致整棵 markdown 子树（大文档时开销明显）重复 diff。
export const MarkdownContent = memo(function MarkdownContent({ source, className }: Props) {
  return (
    <div
      className={className}
      style={{
        fontSize: 14,
        lineHeight: 1.7,
        wordBreak: 'break-word',
      }}
    >
      <ReactMarkdown remarkPlugins={[remarkGfm]}>{source}</ReactMarkdown>
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
  background: rgba(0,0,0,0.3);
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
