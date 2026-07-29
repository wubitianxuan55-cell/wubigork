import { useMemo } from 'react'

/** 简易 Markdown → HTML 渲染器（对齐 Ackem MarkdownContent） */
function renderMarkdown(source: string): string {
  let html = source
    // 转义 HTML
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')

  // 代码块 ```...```
  html = html.replace(/```(\w*)\n([\s\S]*?)```/g, (_, lang, code) =>
    `<pre><code class="language-${lang || 'plain'}">${code.trim()}</code></pre>`
  )
  // 行内代码 `...`
  html = html.replace(/`([^`]+)`/g, '<code>$1</code>')

  // 标题
  html = html.replace(/^### (.+)$/gm, '<h3>$1</h3>')
  html = html.replace(/^## (.+)$/gm, '<h2>$1</h2>')
  html = html.replace(/^# (.+)$/gm, '<h1>$1</h1>')

  // 粗体/斜体/删除线
  html = html.replace(/\*\*\*(.+?)\*\*\*/g, '<strong><em>$1</em></strong>')
  html = html.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
  html = html.replace(/\*(.+?)\*/g, '<em>$1</em>')
  html = html.replace(/~~(.+?)~~/g, '<del>$1</del>')

  // 无序列表
  html = html.replace(/^[\-\*] (.+)$/gm, '<li>$1</li>')
  html = html.replace(/((?:<li>.*<\/li>\n?)+)/g, '<ul>$1</ul>')

  // 有序列表
  html = html.replace(/^\d+\. (.+)$/gm, '<li>$1</li>')
  // 只处理没有 <ul> 包裹的 <li> 序列
  const lines = html.split('\n')
  const result: string[] = []
  let inOl = false
  for (let i = 0; i < lines.length; i++) {
    const line = lines[i]
    const isOlItem = /^<li>/.test(line) && !lines[i - 1]?.includes('<ul>') && !lines[i + 1]?.includes('<ul>')
    // 简化：如果一行是 <li> 且前后没有 <ul>，用 <ol> 包裹
    if (/^<li>/.test(line) && !/<ul>/.test(line) && !/<ol>/.test(line)) {
      if (!inOl) { result.push('<ol>'); inOl = true }
      result.push(line)
    } else {
      if (inOl) { result.push('</ol>'); inOl = false }
      result.push(line)
    }
  }
  if (inOl) result.push('</ol>')
  html = result.join('\n')

  // 段落：非标签行的连续文本
  html = html.replace(/^(?!<[houlp])(.+)$/gm, '<p>$1</p>')
  // 清理空标签
  html = html.replace(/<p>\s*<\/p>/g, '')
  // 合并连续 <p>
  html = html.replace(/<\/p>\n<p>/g, '<br/>')

  // 水平线
  html = html.replace(/^---+$/gm, '<hr/>')

  return html
}

type Props = {
  source: string
  className?: string
}

export function MarkdownContent({ source, className }: Props) {
  const html = useMemo(() => renderMarkdown(source), [source])

  return (
    <div
      className={className}
      style={{
        fontSize: 14,
        lineHeight: 1.7,
        wordBreak: 'break-word',
      }}
      dangerouslySetInnerHTML={{ __html: html }}
    />
  )
}

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
`
