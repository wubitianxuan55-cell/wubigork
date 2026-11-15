// memoryPeek.ts — 提炼记忆弹窗的呈现解析纯函数（波3.5 五卡实弹反馈第三条：
// remember/fact_add 工具行点击弹窗查看完整内容）。纯函数独立成 lib 文件，
// 让组件文件满足 react-refresh only-export-components（组件与函数分文件）。

/** 解析 remember/fact_add 的 args → 弹窗呈现字段（纯函数，可测）。 */
export function parseMemoryPeek(
  tool: string,
  args?: string,
): { title: string; typeLabel: string; hook: string; body: string; bodyLabel: string } | null {
  if (!args) return null;
  let a: Record<string, unknown>;
  try {
    a = JSON.parse(args) as Record<string, unknown>;
  } catch {
    return null;
  }
  if (tool === "remember") {
    const title = typeof a.title === "string" && a.title ? a.title : typeof a.name === "string" ? a.name : "";
    const type = typeof a.type === "string" ? a.type : "";
    const kind = typeof a.kind === "string" && a.kind ? ` · ${a.kind}` : "";
    const body = typeof a.body === "string" ? a.body : "";
    if (!title && !body) return null;
    return {
      title: title || "…",
      typeLabel: `remember · ${type}${kind}`,
      hook: typeof a.description === "string" ? a.description : "",
      body,
      bodyLabel: "body",
    };
  }
  if (tool === "fact_add") {
    const key = typeof a.key === "string" ? a.key : "";
    const value = typeof a.value === "string" ? a.value : "";
    if (!key && !value) return null;
    const cat = typeof a.category === "string" && a.category ? ` · ${a.category}` : "";
    return {
      title: key || "…",
      typeLabel: `fact_add${cat}`,
      hook: typeof a.source === "string" ? a.source : "",
      body: value,
      bodyLabel: "value",
    };
  }
  return null;
}
