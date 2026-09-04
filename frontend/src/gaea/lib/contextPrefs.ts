// contextPrefs — 上下文页「设置中心默认值」偏好（蒸馏规划 2.5d）。
//
// 趋势卡粒度/模式、浏览器分类内排序、文件活动排序此前是卡内瞬时 toggle
// （刷新/重开即丢），本模块把它们持久化到 localStorage 单键 JSON，并在设置
// 中心「办公工作台偏好」补入口（ContextPrefsPanel）。既有 toggle 交互不变，
// 只是把「初值」改为读偏好、「变更时」写回——默认值即用户上次选择。
//
// 模式对齐 lib/subagentPrefs：localStorage 直读 + try/catch 静默降级（隐私
// 模式/存储禁用不影响页面主功能），逐字段校验、损坏值单字段回落默认。

export type CtxTrendGranularity = "step" | "turn";
export type CtxTrendMode = "total" | "delta";
export type CtxBrowserSort = "time" | "size";
export type CtxFileSort = "count" | "recent" | "path";

export interface ContextPrefs {
  trendGranularity: CtxTrendGranularity;
  trendMode: CtxTrendMode;
  browserSort: CtxBrowserSort;
  fileSort: CtxFileSort;
}

const STORAGE_KEY = "gaea.context.prefs";

export const CONTEXT_PREFS_DEFAULTS: ContextPrefs = {
  trendGranularity: "step",
  trendMode: "total",
  browserSort: "time",
  fileSort: "count",
};

const VALID: { [K in keyof ContextPrefs]: readonly ContextPrefs[K][] } = {
  trendGranularity: ["step", "turn"],
  trendMode: ["total", "delta"],
  browserSort: ["time", "size"],
  fileSort: ["count", "recent", "path"],
};

/** 读取上下文页偏好；未设置/无 window 时返回全默认，损坏值逐字段回落。 */
export function loadContextPrefs(): ContextPrefs {
  if (typeof window === "undefined") return { ...CONTEXT_PREFS_DEFAULTS };
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) return { ...CONTEXT_PREFS_DEFAULTS };
    const parsed: unknown = JSON.parse(raw);
    if (typeof parsed !== "object" || parsed === null) return { ...CONTEXT_PREFS_DEFAULTS };
    const obj = parsed as Record<string, unknown>;
    const out = { ...CONTEXT_PREFS_DEFAULTS };
    for (const k of Object.keys(CONTEXT_PREFS_DEFAULTS) as (keyof ContextPrefs)[]) {
      const v = obj[k];
      if (typeof v === "string" && (VALID[k] as readonly string[]).includes(v)) {
        // 单字段合法才覆盖；其余字段保持默认（逐字段校验，不整包拒绝）。
        (out as Record<string, unknown>)[k] = v;
      }
    }
    return out;
  } catch {
    return { ...CONTEXT_PREFS_DEFAULTS };
  }
}

/** 写回单个偏好项（读取-合并-落盘；存储失败静默降级）。 */
export function saveContextPref<K extends keyof ContextPrefs>(key: K, value: ContextPrefs[K]): void {
  if (typeof window === "undefined") return;
  try {
    const cur = loadContextPrefs();
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify({ ...cur, [key]: value }));
  } catch {
    /* 存储失败静默降级：偏好是增强项，不阻塞页面 */
  }
}
