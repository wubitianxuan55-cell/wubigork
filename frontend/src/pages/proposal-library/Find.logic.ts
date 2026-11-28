// Find.logic.ts — 阅读窗查找条的定位函数（纯 DOM 读取，零 React）：在渲染后的正文里
// 按词找全部出现处，或按命中摘要的一小段原文定位到具体那一段。组件见 Find.tsx。

/** 一处命中所覆盖的文本片段（可跨文本节点）。 */
export interface Piece {
  node: Text;
  start: number;
  end: number;
}

export interface FindHit {
  pieces: Piece[];
}

/** 正文里的文本节点；跳过 script / style 与已有高亮里的文字。 */
export function textNodes(root: Node): Text[] {
  const out: Text[] = [];
  const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT);
  for (let n = walker.nextNode(); n; n = walker.nextNode()) {
    if (n.parentElement && /^(SCRIPT|STYLE|MARK)$/.test(n.parentElement.tagName)) continue;
    out.push(n as Text);
  }
  return out;
}

/** 正文平铺文本上的 [from, to) 落到各个文本节点上的片段。 */
function toPieces(nodes: Text[], from: number, to: number): Piece[] {
  const out: Piece[] = [];
  let pos = 0;
  for (const node of nodes) {
    const len = node.data.length;
    if (to > pos && from < pos + len) {
      const start = Math.max(from, pos) - pos;
      const end = Math.min(to, pos + len) - pos;
      if (end > start) out.push({ node, start, end });
    }
    pos += len;
    if (pos >= to) break;
  }
  return out;
}

/** 拼平铺文本；小写化若改变了长度（罕见字符），退回大小写敏感匹配。 */
function flatten(nodes: Text[]): { raw: string; low: string } {
  const raw = nodes.map((n) => n.data).join("");
  const low = raw.toLowerCase();
  return low.length === raw.length ? { raw, low } : { raw, low: raw };
}

/** 正文里检索词的全部出现位置（大小写不敏感），按文档顺序。 */
export function findTerm(root: Node, term: string): FindHit[] {
  const needle = term.trim().toLowerCase();
  if (!needle) return [];
  const nodes = textNodes(root);
  const { low } = flatten(nodes);
  const hits: FindHit[] = [];
  for (let at = low.indexOf(needle); at >= 0; at = low.indexOf(needle, at + needle.length)) {
    const pieces = toPieces(nodes, at, at + needle.length);
    if (pieces.length) hits.push({ pieces });
  }
  return hits;
}

// 定位时两边都去掉的字符：空白、Markdown 记号、表格竖线。摘要取自 Markdown 原文，
// 阅读窗是渲染后的文字，带着记号去比对会对不上。（test 用的那份不带 g——带 g 的
// 正则 test 是有状态的。）
const LOCATE_DROP = /[\s*_`#>|~]/g;
const LOCATE_DROP_ONE = /[\s*_`#>|~]/;

/** 按一小段原文定位（忽略空白与 Markdown 记号，可跨文本节点）；找不到返回 null。 */
export function findNeedle(root: Node, needle: string): FindHit | null {
  const want = needle.replace(LOCATE_DROP, "").toLowerCase();
  if (want.length < 2) return null;
  const nodes = textNodes(root);
  let compact = "";
  const map: number[] = [];
  let pos = 0;
  for (const node of nodes) {
    for (let i = 0; i < node.data.length; i++) {
      const ch = node.data[i];
      if (LOCATE_DROP_ONE.test(ch)) continue;
      compact += ch.toLowerCase();
      map.push(pos + i);
    }
    pos += node.data.length;
  }
  const at = compact.indexOf(want);
  if (at < 0) return null;
  return { pieces: toPieces(nodes, map[at], map[at + want.length - 1] + 1) };
}

/** 摘要的开头可能落在记号上：由长到短试几档，取第一档找得到的。 */
export function locateSnippet(root: Node, snippet: string, lengths = [16, 10, 6]): FindHit | null {
  const plain = snippet.replace(/[【】]/g, "").replace(/\s+/g, "");
  for (const n of lengths) {
    if (plain.length < n) continue;
    const hit = findNeedle(root, plain.slice(0, n));
    if (hit) return hit;
  }
  return plain.length >= 2 ? findNeedle(root, plain) : null;
}
