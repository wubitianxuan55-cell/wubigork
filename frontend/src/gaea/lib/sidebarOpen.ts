// sidebarOpen.ts — v4.25「模型主动打开」前端结果解析器（对标 dsh-better-sidebar）。
//
// 后端内置工具 sidebar_open（internal/gaea/tool/builtin/sidebar_open.go）把
// 工作区内的文件/目录推到右面板打开：envelope 结构化结果 data 形如
//   {"kind":"file|directory","path_abs":"...","path_rel":"<相对工作区根>","message":"..."}
// 本解析器从 trajectory 工具项（store 里 kind:"tool" 的 name/args/output）
// 折叠出该动作，供 App 接线驱动右面板。纯函数、绝不抛错——坏 JSON / 失败
// code / 字段缺失一律返回 null，与 changes.ts 等折叠器同口径。

export interface SidebarOpenResult {
  kind: "file" | "directory";
  pathRel: string;
}

interface SidebarEnvelopeData {
  kind?: unknown;
  path_abs?: unknown;
  path_rel?: unknown;
}

// parseSidebarOpenResult 判定一个工具项是否为 sidebar_open 打开动作且成功，
// 命中返回 { kind, pathRel }（pathRel 优先取 envelope data.path_rel，缺省依次
// 回退 data.path_abs、模型原始参数 path），否则返回 null。
export function parseSidebarOpenResult(
  toolName: string,
  argsJson: string,
  resultJson: string,
): SidebarOpenResult | null {
  if (toolName !== "sidebar_open") return null;

  let env: { ok?: unknown; code?: unknown; data?: unknown };
  try {
    env = JSON.parse(resultJson || "{}") as typeof env;
  } catch {
    return null;
  }
  if (!env || typeof env !== "object") return null;
  if (env.ok !== true) return null;
  if (typeof env.code === "string" && env.code !== "ok") return null;

  const data = env.data as SidebarEnvelopeData | null;
  if (!data || typeof data !== "object") return null;
  if (data.kind !== "file" && data.kind !== "directory") return null;

  const pathRel = firstNonEmptyString(data.path_rel, data.path_abs, argPath(argsJson));
  if (pathRel === null) return null;
  return { kind: data.kind, pathRel };
}

// argPath 提取模型原始参数里的 path（data.path_rel/path_abs 缺失时的兜底）。
function argPath(argsJson: string): string | null {
  try {
    const args = JSON.parse(argsJson || "{}") as { path?: unknown } | null;
    if (!args || typeof args !== "object") return null;
    return typeof args.path === "string" && args.path.trim() !== "" ? args.path.trim() : null;
  } catch {
    return null;
  }
}

function firstNonEmptyString(...vals: unknown[]): string | null {
  for (const v of vals) {
    if (typeof v === "string" && v.trim() !== "") return v.trim();
  }
  return null;
}
