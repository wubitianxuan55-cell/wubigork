// 数学公式文本处理（跨板块共享）：normalizeMath 把模型常见的 \(...\)/\[...\]
// 定界符归一成 remark-math 的 $...$/$$...$$；hasMathContent 供 CSS 懒注入
// 门控；ensureKatexCss 按需加载 KaTeX 样式（link 注入，vite 走共享 chunk，
// 与 gaea Markdown 既有先例同款）。办公板块 Markdown 与聊天线两处消费。

let katexCssLoaded = false;

export function ensureKatexCss(): void {
  if (katexCssLoaded) return;
  katexCssLoaded = true;
  const link = document.createElement("link");
  link.rel = "stylesheet";
  link.href = new URL("katex/dist/katex.min.css", import.meta.url).href;
  document.head.appendChild(link);
}

export function hasMathContent(text: string): boolean {
  return text.includes("$$") || (text.includes("$") && /\$\S[^$]*\S\$/.test(text));
}

const LB = "\x00LB\x00";

export function normalizeMath(s: string): string {
  let r = s.replace(/\\\\\[/g, LB);
  r = r
    .replace(/\\\[/g, () => "$$")
    .replace(/\\\]/g, () => "$$")
    .replace(/\\\(/g, () => "$")
    .replace(/\\\)/g, () => "$");
  // 用字面量字符串恢复哨兵（等价于全局替换，避免在正则中书写 \x00 控制字符）
  r = r.split(LB).join("\\\\[");
  const vert = (m: string) => m.replace(/\|/g, "\\vert ");
  r = r.replace(/\$\$([\s\S]*?)\$\$/g, (_m, m) => `$$${vert(m)}$$`);
  r = r.replace(/\$([^$\n]+)\$/g, (_m, m) => `$${vert(m)}$`);
  return r;
}
