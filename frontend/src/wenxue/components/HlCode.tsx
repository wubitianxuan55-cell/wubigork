import { useEffect, useState } from "react";
import { highlightCode } from "../lib/codeHighlight";
import "../hljs-theme.css";

// 代码块语法高亮（懒加载缝）——办公板块 Markdown 与聊天线 ChatCodeBlock 共享。
// 首帧纯文本立现（不挡流式），hljs 异步就位后整体换着色 HTML；未知/失败
// 语言保持纯文本不硬造。调用点=稳定分段（MemoMarkdown 段签名缓存），流式
// 增长中的尾部走简易 HTML 不经此处，无逐 delta 重高亮问题。
export function HlCode({ text, lang }: { text: string; lang?: string }) {
  const [html, setHtml] = useState<string | null>(null);
  useEffect(() => {
    let alive = true;
    setHtml(null);
    highlightCode(text, lang).then((h) => {
      if (alive) setHtml(h);
    });
    return () => {
      alive = false;
    };
  }, [text, lang]);
  return html === null ? (
    <code>{text}</code>
  ) : (
    <code dangerouslySetInnerHTML={{ __html: html }} />
  );
}
