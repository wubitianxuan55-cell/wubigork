// diffHighlight — diff 行内语法着色（2c 遗留项，CodeMirror 依赖 3a 已就位）。
//
// 按文件扩展名选 Lezer parser，对单行文本做 token 切分，输出 tok-* 类名
// 片段（配色见 context-view.css 的 .tok-* 规则）。单行独立解析没有跨行
// 上下文（多行字符串/块注释会失真）——展示层可接受的近似，如实注明。
// 解析器按扩展名模块级缓存（parser 实例复用，parse 结果不缓存）。
import { parser as jsParser } from "@lezer/javascript";
import { parser as pyParser } from "@lezer/python";
import { parser as jsonParser } from "@lezer/json";
import { parser as cssParser } from "@lezer/css";
import { parser as htmlParser } from "@lezer/html";
import { parser as mdParser } from "@lezer/markdown";
import { highlightTree, classHighlighter } from "@lezer/highlight";
import type { Parser } from "@lezer/common";

const PARSERS: [RegExp, Parser][] = [
  [/\.(ts|tsx)$/i, jsParser],
  [/\.(js|jsx|mjs|cjs)$/i, jsParser],
  [/\.(py)$/i, pyParser],
  [/\.(json)$/i, jsonParser],
  [/\.(css)$/i, cssParser],
  [/\.(html?|htm)$/i, htmlParser],
  [/\.(md|markdown|mdx)$/i, mdParser],
];

const parserCache = new Map<string, Parser | null>();

/** 按路径找 parser（找不到= null，行文本原样渲染）。 */
export function parserForPath(path: string): Parser | null {
  if (parserCache.has(path)) return parserCache.get(path)!;
  let found: Parser | null = null;
  for (const [re, p] of PARSERS) {
    if (re.test(path)) {
      found = p;
      break;
    }
  }
  parserCache.set(path, found);
  return found;
}

export interface TokenSeg {
  text: string;
  /** Lezer 高亮类名（tok-keyword 等；空串=无着色）。 */
  cls: string;
}

// 单行 token 上限：超长行不做 token 切分（防退化），整行无着色。
const MAX_HIGHLIGHT_LEN = 600;

/**
 * 单行语法着色：path 决定语言，text 切成带 tok-* 类名的片段。
 * 无 parser / 空行 / 超长行 → 单片段 cls=""（原样渲染）。
 */
export function highlightLine(text: string, path: string): TokenSeg[] {
  const parser = parserForPath(path);
  if (!parser || !text || text.length > MAX_HIGHLIGHT_LEN) {
    return [{ text, cls: "" }];
  }
  try {
    const tree = parser.parse(text);
    const spans: { from: number; to: number; cls: string }[] = [];
    highlightTree(tree, [classHighlighter], (from, to, cls) => {
      if (to > from) spans.push({ from, to, cls });
    });
    if (spans.length === 0) return [{ text, cls: "" }];
    // 合并区间 → 片段（含未着色间隙）
    const out: TokenSeg[] = [];
    let pos = 0;
    for (const s of spans) {
      if (s.from > pos) out.push({ text: text.slice(pos, s.from), cls: "" });
      out.push({ text: text.slice(s.from, s.to), cls: s.cls });
      pos = s.to;
    }
    if (pos < text.length) out.push({ text: text.slice(pos), cls: "" });
    return out;
  } catch {
    return [{ text, cls: "" }];
  }
}
