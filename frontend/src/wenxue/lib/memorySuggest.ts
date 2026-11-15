// memorySuggest.ts — 提炼记忆卡的 notice 判定/计数纯函数（与组件文件分离，
// react-refresh only-export-components 纪律；同 memoryPeek.ts 模式）。

const DREAM_NOTICE_RE = /本轮提炼出 (\d+) 条记忆建议/;

/** notice 文本是否为提炼记忆提示（Transcript notice 分支特化判定，可测）。 */
export function isDreamSuggestNotice(text: string): boolean {
  return DREAM_NOTICE_RE.test(text);
}

/** 从 notice 文本提取建议条数（匹配失败回 0）。 */
export function dreamSuggestCount(text: string): number {
  const m = DREAM_NOTICE_RE.exec(text);
  return m ? Number(m[1]) : 0;
}
