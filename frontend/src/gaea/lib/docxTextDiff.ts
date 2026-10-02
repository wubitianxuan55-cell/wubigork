// docxTextDiff.ts — docx 版本结构化对比（v4.28 A2「线 C」交付验收闭环）纯函数层。
//
// Why: 版本时间线的「与当前对比」此前对 .docx 只能给 kind:"unsupported" 降级
// （双版本并排预览）。本模块把两份 docx 的正文段落序列做段级 LCS diff，
// 收掉这笔记账。
//
// How: 段落提取复用 lib/docxText.extractDocxParagraphs（docx = zip 包，JSZip
// 解包 + DOMParser 解 word/document.xml，与 docxOutline/docxText 同套路），
// 本模块只做给定两个段落序列的纯函数 diff，无 React 依赖、不触碰 bridge；
// dataUrl 取数与 async 编排在 versionCompare.ts。
//
// 口径说明：
//   - 表格每格按独立段落计。word/document.xml 的 w:p 天然按文档顺序展开、
//     含表格内段落（docxText 提取即此口径），不额外按表格聚合——聚合会把
//     「改了表里一格」放大成「改了整张表」，粒度失真。
//   - 段内改写（同一段改几个字）在段级 diff 里如实呈现为一对相邻 del+add
//     （与文本行级 diff 的「改一行 = −1 +1」同一语义），不做字词级对齐。
//   - 空段落保留参与 diff（空行是版面结构的一部分，删除分隔行的改动不该被吞）。

import { lcsDiff } from "./diff";

/** 段级 diff 一行：type 语义与 DiffRow 一致（ctx/add/del），index 为段落序号（1 起）。 */
export interface DocxRow {
  type: "ctx" | "add" | "del";
  /** 段落序号（1 起）：ctx/add 取当前文档中的序号，del 取基线文档中的序号。 */
  index: number;
  text: string;
}

/**
 * 纯函数：两份 docx 的正文段落序列 → 段级 LCS diff（带序号）。
 * 算法即 lib/diff.lcsDiff 的经典 LCS DP（FE4-01 收敛：行级/段级/页签名/字符级
 * 同一份泛型，文档段落数量级（千级）矩阵内存可接受），本函数只做「段落」元素
 * 类型与取值适配：ctx/add 序号取当前文档、del 取基线文档（1 起）。不走
 * join("\n") 复用行级入口，该技巧要求元素不含换行符，对通用序列是隐式脆弱约束。
 */
export function diffDocxParagraphs(baseParas: string[], curParas: string[]): DocxRow[] {
  return lcsDiff(baseParas, curParas, (p, q) => p === q).map((op): DocxRow => {
    if (op.type === "ctx") return { type: "ctx", index: op.bi + 1, text: curParas[op.bi]! };
    if (op.type === "del") return { type: "del", index: op.ai + 1, text: baseParas[op.ai]! };
    return { type: "add", index: op.bi + 1, text: curParas[op.bi]! };
  });
}

/** 段级 diff 的增删计数（对齐 versionCompare.diffStatOf 的口径）。 */
export function docxDiffStat(rows: DocxRow[]): { add: number; del: number } {
  let add = 0;
  let del = 0;
  for (const r of rows) {
    if (r.type === "add") add++;
    else if (r.type === "del") del++;
  }
  return { add, del };
}
