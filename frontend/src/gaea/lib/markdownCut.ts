// markdownCut.ts — Markdown 流式「稳定切分点」纯函数（自 MemoMarkdown.tsx 迁出，
// v4.368 聊天流式分段渲染与办公流式共用同一语义；独立文件以满足 react-refresh
// 组件文件纯导出约束）。

/**
 * findStableCut — 找到"稳定"切分点。
 *
 * 规则（按优先级）：
 *   1. 若候选切点落在未闭合的代码围栏内（fence body 含空行时原 suffix 扫描
 *      的盲区——计数偶数误判可切，把代码块拦腰切断），切点回退到该围栏打开
 *      行之前（v4.368：改为全量行扫描 fence 状态）。
 *   2. 否则在最后一个 \n\n 处切分。
 *   3. 没有 \n\n 则全部视为不稳定。
 *
 * 返回 [stablePrefix, unstableSuffix]。导出供聊天流式行（ChatRow）复用同一
 * 切分语义（v4.368 流式分段渲染）。
 */
export function findStableCut(text: string): [string, string] {
  const lastGap = text.lastIndexOf("\n\n");
  if (lastGap < 0) return ["", text];

  // v4.368：全量行扫描候选切点之前的行首 ``` 标记，跟踪 fence 状态——
  // 原实现只扫切点之后的 suffix，fence body 含空行时计数为偶数误判「可切」，
  // 把代码块拦腰切断（盲区）。现若切点落在未闭合 fence 内，回退切点到该
  // fence 打开行之前（切分只会更保守，稳定段永不含悬挂 fence）。导出供
  // 聊天流式行（ChatRow）复用同一切分语义（v4.368 流式分段渲染）。
  const FENCE = "```";
  let inFence = false;
  let fenceOpenStart = -1;
  let pos = 0;
  while (pos <= lastGap) {
    const nl = text.indexOf("\n", pos);
    const lineEnd = nl < 0 ? text.length : nl;
    const line = text.slice(pos, lineEnd);
    if (line.trimStart().startsWith(FENCE)) {
      inFence = !inFence;
      fenceOpenStart = inFence ? pos : -1;
    }
    if (nl < 0 || nl >= lastGap) break;
    pos = nl + 1;
  }
  let cut = lastGap + 2;
  if (inFence && fenceOpenStart >= 0) {
    const preFenceNL = text.lastIndexOf("\n\n", fenceOpenStart - 1);
    cut = preFenceNL >= 0 ? preFenceNL + 2 : 0;
  }
  return [text.slice(0, cut), text.slice(cut)];
}
