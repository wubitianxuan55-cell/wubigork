// 文件链接工具：统一识别"输出文本里的本地文件引用"，让所有涉及文件的输出
// （聊天正文 / 流式尾部 / 工具输出）都能渲染成可点击预览。
//
// 识别三类文件引用：
//   1. 绝对路径：C:\AI\xx.docx、C:/AI/xx.docx、/data/xx.pdf、\tmp\xx.txt
//   2. 相对路径（含目录分隔符）：exports/方案.docx、.gaea/uploads/截图.png、docs/a.md
//   3. 关键词引导的裸文件名：输出文件：成本测算.xlsx、已生成：报告.docx
//
// 纯逻辑模块，不依赖 React（渲染层分别在 Markdown.tsx / MemoMarkdown.tsx /
// components/FileLinkText.tsx）。

// 常见文件扩展名（含全部办公格式 + 代码/文本格式）。
export const FILE_EXT_RE = /\.(md|markdown|mmd|mermaid|txt|json|jsonl|csv|tsv|xml|yaml|yml|toml|ini|log|docx?|xlsx?|pdf|pptx?|odt|ods|odp|rtf|wps|et|dps|ofd|pages|numbers|key|png|jpe?g|gif|webp|svg|bmp|ico|html?|css|js|jsx|ts|tsx|go|py|java|c|h|cpp|hpp|rs|rb|php|sh|bat|ps1|sql)$/i;

// 交付物扩展名：办公文档 / PDF / 文本表格 / 图片。代码文件不算交付物，
// 避免把"提到了某个 .go 文件"也渲染成交付卡片。
export const DELIVERABLE_EXT_RE = /\.(docx?|xlsx?|pptx?|pdf|md|markdown|txt|csv|json|html?|odt|ods|odp|rtf|wps|et|dps|ofd|png|jpe?g|gif|webp|svg|bmp|ico)$/i;

// markdown 链接 href 是否为本地文件（排除 URL / 锚点 / 协议）。
export function isLocalFilePath(href: string): boolean {
  const trimmed = href.trim();
  if (!trimmed || /^(https?:|mailto:|tel:|data:|javascript:|#|\/\/)/i.test(trimmed)) return false;
  const clean = trimmed.replace(/^\.{0,2}\//, "");
  return FILE_EXT_RE.test(clean);
}

// 路径体：允许除空白与常见标点外的任意字符（不含空格，避免吞掉整句）。
const PATH_BODY = "[^\\s，。；、（）【】《》\"“”‘’'<>|)\\]}！？…]";

// 相对路径首段：排除分隔符、空白与标点（含冒号/括号/方括号），
// 避免"输出文件：exports/x.docx"从句子开头跨词误配。
const FIRST_SEG = "[^\\\\/\\s，。；、（）【】《》\"“”‘’'<>|)\\]}！？…:：(（\\[\\{]";

// 路径边界：行首 / 空白 / 中文英文标点 / 引号括号 / 箭头。
const PATH_BOUNDARY = "^|[\\s,，:：(（\"'“”>→]";

// 绝对 + 相对（含目录分隔符）路径。
const FILE_PATH_RE = new RegExp(
  "(?:[A-Za-z]:[\\\\/]" +
    "|(?<=" + PATH_BOUNDARY + ")[\\\\/]" +
    "|(?<=" + PATH_BOUNDARY + ")(?:\\.{1,2}[\\\\/]|" + FIRST_SEG + "*?[\\\\/]))" +
    PATH_BODY +
    "+\\.[A-Za-z0-9]{1,12}",
  "g",
);

// 关键词引导的裸文件名：输出文件：x.docx / 已生成：报告.docx / output: a.csv 等。
// 关键词与文件名之间要求冒号或空格，避免误伤"文件夹：xxx"这类普通文本。
const BARE_FILE_RE = new RegExp(
  "(?:输出文件|已生成|保存到|导出|生成|文件|路径|附件|output|file|saved|export|generated|result)" +
    "(?:\\s*[:：]\\s*|\\s+)" +
    "(" + PATH_BODY + "+\\.(?:[A-Za-z0-9]{1,12}))",
  "gi",
);

// 形如 example.com / www.example.com.cn 的域名前缀不当作相对路径；
// 裸文件名（无目录分隔符）不适用域名判定（如 a.csv 是文件不是域名）。
function looksDomainLike(path: string): boolean {
  if (!/[\\/]/.test(path)) return false;
  const first = path.split(/[\\/]/)[0];
  return /^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?(\.[a-z0-9](?:[a-z0-9-]*[a-z0-9])?)+$/i.test(first);
}

export interface FileMention {
  path: string;
  label: string;
  start: number;
  end: number;
}

// 扫描纯文本中的本地文件引用，返回按位置排序的命中列表（自动去重/去重叠）。
export function findFileMentions(text: string): FileMention[] {
  const out: FileMention[] = [];
  const push = (path: string, start: number, end: number) => {
    if (!isLocalFilePath(path)) return;
    if (path.includes("://") || looksDomainLike(path)) return;
    out.push({ path, label: path, start, end });
  };

  FILE_PATH_RE.lastIndex = 0;
  let m: RegExpExecArray | null;
  while ((m = FILE_PATH_RE.exec(text)) !== null) {
    push(m[0], m.index, m.index + m[0].length);
  }

  BARE_FILE_RE.lastIndex = 0;
  while ((m = BARE_FILE_RE.exec(text)) !== null) {
    const file = m[1];
    const start = m.index + m[0].indexOf(file);
    const end = start + file.length;
    if (out.some((o) => start < o.end && o.start < end)) continue;
    push(file, start, end);
  }

  out.sort((a, b) => a.start - b.start);
  return out;
}

// 从一段文本中提取交付物文件引用（去重，保持出现顺序）。
export function deliverableMentions(text: string): string[] {
  const seen = new Set<string>();
  const out: string[] = [];
  for (const m of findFileMentions(text)) {
    if (!DELIVERABLE_EXT_RE.test(m.path) || seen.has(m.path)) continue;
    seen.add(m.path);
    out.push(m.path);
  }
  return out;
}

function esc(s: string): string {
  return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
}

function escAttr(s: string): string {
  return esc(s).replace(/"/g, "&quot;");
}

export function escapeHtml(s: string): string {
  return esc(s);
}

/**
 * fileChipHtml — 行内文件引用 chip 的 HTML 形态（与 FileChip 组件同构，
 * 供流式尾部 / MemoMarkdown 等字符串渲染使用）。P0-2 视觉统一：
 * 文件名 + 扩展名 badge，data-file-preview 由外层容器事件委托打开预览。
 */
export function fileChipHtml(path: string, label?: string): string {
  const name = label ?? path.split(/[/\\]/).pop() ?? path;
  const badge = (name.match(/\.([a-z0-9]+)$/i)?.[1] ?? "file").toLowerCase();
  return (
    `<button type="button" data-file-preview="${escAttr(path)}" title="点击预览 ${escAttr(path)}" ` +
    `class="inline-flex items-center gap-1 align-middle mx-0.5 px-1.5 py-0.5 rounded-md border border-accent/25 bg-accent/5 ` +
    `text-accent text-[0.86em] font-medium cursor-pointer hover:bg-accent/15 transition-colors">` +
    `<span class="max-w-[220px] truncate font-mono">${esc(name)}</span>` +
    `<span class="shrink-0 text-[9px] uppercase text-fg-faint/70 border border-border-soft/60 rounded px-1 py-px font-mono">${esc(badge)}</span>` +
    `</button>`
  );
}

// 把纯文本中的文件引用转成 HTML 可点击 chip（流式尾部 / 简单 HTML 渲染用）。
// 视觉与 FileChip 组件同构（P0-2 统一：文件名 + 扩展名 badge）。
// 路径通过 data-file-preview 携带，由外层容器事件委托打开预览。
export function htmlFileLinks(text: string): string {
  const mentions = findFileMentions(text);
  if (mentions.length === 0) return esc(text);
  let out = "";
  let last = 0;
  for (const m of mentions) {
    out += esc(text.slice(last, m.start));
    out += fileChipHtml(m.path, m.label.split(/[/\\]/).pop());
    last = m.end;
  }
  out += esc(text.slice(last));
  return out;
}
