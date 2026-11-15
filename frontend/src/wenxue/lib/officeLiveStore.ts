// officeLiveStore — dsh-office COM 实时编辑状态（波3「壳内迷你卡替代」）。
//
// 背景：波1 拍板不启 helper 的 WinForms 迷你卡（card=false），「AI 正在改哪个
// 文档 + 改到第几步」改由 gaea 壳内呈现——数据源就是工具卡事件流（office_open/
// office_edit/office_save 的 dispatch/回执），本 store 只做归约与暂存，喂入在
// hooks/useOfficeLive（照 usePreviewAutoFront 的 state.items 遍历模式），展示在
// components/OfficeLiveBadge（FilePreview 头部胶囊）。
//
// 生命周期：活动文档跨回合（turnEnd 不清）——AI 常常跨多回合持续编辑同一份
// 文档；收尾靠 office_close 回执清空 + updatedAt 超过 5 分钟的惰性过期（无
// 定时器：组件/读取方判新鲜度时顺手清，见 expireIfStale）。
//
// 结构：归约逻辑全部抽成纯函数 officeLiveReduce（officeTurnProjection 同风格，
// 单测钉死），zustand store 只是薄壳（setState 包一层）。

import { create } from "zustand";

/** COM 套件应用（与 Go 侧 office_* 工具 app 枚举一致）。 */
export type OfficeLiveApp = "word" | "excel" | "ppt";

/** 编辑批次进度：done=已应用操作数，total=本次 ops 总数。 */
export interface OfficeLiveStep {
  done: number;
  total: number;
}

export interface OfficeLiveState {
  /** 当前正被 AI 编辑的文档（args 原样：绝对或工作区相对路径）。 */
  activeDoc: string | null;
  activeApp: OfficeLiveApp | null;
  /** 最近一次 office_edit 批次进度（保存后清空，无进行中批次为 null）。 */
  step: OfficeLiveStep | null;
  /** 最近一次 office_* 工具活动时刻（ms）；0=从未活动。 */
  updatedAt: number;
}

/** 活动过期窗口：5 分钟内无任何 office_* 工具活动即视为「不再实时编辑」。 */
export const OFFICE_LIVE_TTL_MS = 5 * 60 * 1000;

export const initialOfficeLiveState: OfficeLiveState = {
  activeDoc: null,
  activeApp: null,
  step: null,
  updatedAt: 0,
};

// ── 纯函数层（单测钉死）─────────────────────────────────────────────────

/** 宽容解析工具卡 args（JSON 串；坏形状回空对象，与 officeTurnProjection 同式）。 */
function parseArgs(args: string | undefined): Record<string, unknown> {
  if (!args) return {};
  try {
    const parsed = JSON.parse(args) as unknown;
    return parsed && typeof parsed === "object" && !Array.isArray(parsed)
      ? (parsed as Record<string, unknown>)
      : {};
  } catch {
    return {};
  }
}

const APP_KINDS: ReadonlySet<string> = new Set(["word", "excel", "ppt"]);

/** 扩展名 → 应用推断（对齐 COM 族可打开的文档类型；认不出返回 null）。 */
export function inferOfficeAppFromPath(path: string): OfficeLiveApp | null {
  const m = /\.(docx?|docm|wps|odt|rtf|xlsx?|xlsm|csv|et|ods|pptx?|dps|odp)$/i.exec(path ?? "");
  if (!m) return null;
  const ext = m[0].toLowerCase();
  if (/^\.(doc|docx|docm|wps|odt|rtf)$/.test(ext)) return "word";
  if (/^\.(xls|xlsx|xlsm|csv|et|ods)$/.test(ext)) return "excel";
  return "ppt";
}

function appOfArg(v: unknown): OfficeLiveApp | null {
  return typeof v === "string" && APP_KINDS.has(v) ? (v as OfficeLiveApp) : null;
}

/**
 * office_edit 回执文本 → 编辑进度。
 *
 * 解析依据（internal/office/comhelper/format.go FormatEdit，波1 落地的回执口径）：
 *   - 全成：首行 `All N operation(s) applied (not saved yet):` → done=total=N；
 *   - 中止：`Stopped at operation i (...)` → 第 i 条失败，前 i-1 条已应用，
 *     即 done=i-1（total 文本里没有，由调用方用 args.ops 长度补）。
 * 认不出任何一种形态返回 null（诚实缺省，不猜）。
 */
export function parseEditStepFromOutput(text: string | undefined): { done: number; total: number | null } | null {
  if (!text) return null;
  const all = /All (\d+) operation\(s\) applied/.exec(text);
  if (all) {
    const n = Number(all[1]);
    if (Number.isFinite(n) && n >= 0) return { done: n, total: n };
  }
  const stopped = /Stopped at operation (\d+)/.exec(text);
  if (stopped) {
    const i = Number(stopped[1]);
    if (Number.isFinite(i) && i >= 1) return { done: i - 1, total: null };
  }
  return null;
}

function clampStep(done: number, total: number | null, fallbackTotal: number): OfficeLiveStep | null {
  // ops 长度是权威上界：helper 的回执计数恒等于真实 ops 数（Edit 循环
  // ops.Count），回执超出 ops 长度=不可能形态，以 ops 封顶（2026-11-13
  // clamp 测试钉死）；done 再以 t 封顶。
  let t = total !== null && total > 0 ? total : fallbackTotal;
  if (fallbackTotal > 0) t = Math.min(t, fallbackTotal);
  if (t <= 0) return null;
  return { done: Math.max(0, Math.min(done, t)), total: t };
}

/** 一次工具活动事件（hook 从工具卡提取；now 由调用方注入保证纯函数可测）。 */
export interface OfficeLiveNote {
  tool: string;
  /** 工具卡 args（JSON 串原样）。 */
  args?: string;
  /** 工具卡状态（running/done/error/stopped）。 */
  status: string;
  /** 回执输出（office_edit 的进度文本在这里）。 */
  output?: string;
  /** 回执错误文本（中止信息可能在 error 里）。 */
  error?: string;
  /** 活动时刻（ms）。 */
  now: number;
}

/**
 * 归约一次 office_* 工具活动 → 新状态（纯函数）。
 *
 * 口径：
 *   - 非 office_* 工具：原样返回（hook 侧已过滤，双保险）；
 *   - 任何 office_* 活动先做惰性过期（上一次活动超过 TTL → 先清空再应用），
 *     并刷新 updatedAt（读/渲染等工具也是「文档还开着」的心跳）；
 *   - office_open 回执成功：activeDoc=args.path（绝对/相对原样；空 path=新建
 *     未保存文档，保留现值不覆盖），activeApp=args.app 或扩展名推断；
 *   - office_edit：activeDoc=args.doc（缺省保留现值）；step——running 时
 *     {done:0,total}（total=args.ops 长度）、回执后从输出解析
 *     （parseEditStepFromOutput；done 状态解析不出按全成计）、error/stopped
 *     解析不出保持现值（不假装知道进度）；
 *   - office_save 回执成功：args.path 非空（另存为）则 activeDoc 切到落点，
 *     step 清空（保存后无进行中批次，文档继续开着）；
 *   - office_close 回执成功：整组清空（编辑会话结束）。
 */
export function officeLiveReduce(state: OfficeLiveState, note: OfficeLiveNote): OfficeLiveState {
  if (typeof note.tool !== "string" || !note.tool.startsWith("office_")) return state;
  const ttl = OFFICE_LIVE_TTL_MS;
  // 惰性过期：上一次活动已超出 TTL → 旧文档不再算「正在编辑」，先清空再应用。
  const base: OfficeLiveState =
    state.updatedAt > 0 && note.now - state.updatedAt > ttl ? initialOfficeLiveState : state;
  const args = parseArgs(note.args);
  const next: OfficeLiveState = { ...base, updatedAt: note.now };

  switch (note.tool) {
    case "office_open": {
      if (note.status !== "done") return next;
      const path = typeof args.path === "string" ? args.path.trim() : "";
      if (path) next.activeDoc = path;
      next.activeApp =
        appOfArg(args.app) ??
        inferOfficeAppFromPath(path || next.activeDoc || "") ??
        next.activeApp;
      return next;
    }
    case "office_edit": {
      const doc = typeof args.doc === "string" && args.doc.trim() !== "" ? args.doc.trim() : null;
      if (doc) next.activeDoc = doc;
      next.activeApp =
        appOfArg(args.app) ??
        inferOfficeAppFromPath(next.activeDoc ?? "") ??
        next.activeApp;
      const ops = Array.isArray(args.ops) ? args.ops.length : 0;
      if (note.status === "running") {
        next.step = ops > 0 ? { done: 0, total: ops } : null;
        return next;
      }
      // 回执（done/error/stopped 都可能带进度文本：中止是工具正常返回，
      // 不是 Go error，波1 FormatEdit 的口径）。total 口径 = args.ops 长度
      // （本次批次的计划规模）；ops 缺省时才用回执文本里的 total 补。
      const parsed = parseEditStepFromOutput(note.output) ?? parseEditStepFromOutput(note.error);
      if (parsed) {
        next.step = clampStep(parsed.done, ops > 0 ? ops : parsed.total, ops);
        return next;
      }
      if (note.status === "done") {
        // 解析不出进度按全成计（工具回执成功 = 整批应用完成）。
        next.step = clampStep(ops, null, ops);
      }
      // error/stopped 且解析不出：保持现值（不假装知道进度）。
      return next;
    }
    case "office_save": {
      if (note.status !== "done") return next;
      const path = typeof args.path === "string" ? args.path.trim() : "";
      if (path) next.activeDoc = path; // 另存为：活动文档切到落点
      next.activeApp =
        appOfArg(args.app) ??
        inferOfficeAppFromPath(next.activeDoc ?? "") ??
        next.activeApp;
      next.step = null; // 保存后无进行中的编辑批次（文档继续开着）
      return next;
    }
    case "office_close": {
      if (note.status !== "done") return next;
      return { activeDoc: null, activeApp: null, step: null, updatedAt: note.now };
    }
    default:
      // office_read/status/render/help：纯心跳（文档可能仍是当前文档）。
      return next;
  }
}

/** 该状态此刻是否代表一次「活的」实时编辑（有文档 + TTL 内有活动）。 */
export function isOfficeLiveFresh(state: OfficeLiveState, now: number, ttlMs = OFFICE_LIVE_TTL_MS): boolean {
  return state.activeDoc !== null && state.updatedAt > 0 && now - state.updatedAt <= ttlMs;
}

// ── zustand 薄壳 ────────────────────────────────────────────────────────

export interface OfficeLiveStoreState extends OfficeLiveState {
  /** 工具卡回执喂入（非 office_* 工具被归约层忽略）。 */
  noteActivity: (tool: string, args: string | undefined, status: string, output?: string, error?: string) => void;
  /** 手动清空（close 语义之外的收口预留）。 */
  clear: () => void;
  /** 惰性过期：超 TTL 的活动文档就地清空（组件读取前调用，无定时器）。 */
  expireIfStale: (now?: number) => void;
}

export const useOfficeLiveStore = create<OfficeLiveStoreState>()((set, get) => ({
  ...initialOfficeLiveState,
  noteActivity: (tool, args, status, output, error) =>
    set((s) => officeLiveReduce(s, { tool, args, status, output, error, now: Date.now() })),
  clear: () => set({ ...initialOfficeLiveState }),
  expireIfStale: (now = Date.now()) => {
    const s = get();
    if (s.activeDoc !== null && !isOfficeLiveFresh(s, now)) {
      set({ ...initialOfficeLiveState });
    }
  },
}));

/** 测试辅助：复位状态（vitest 用例间隔离）。 */
export function resetOfficeLiveStoreForTest(): void {
  useOfficeLiveStore.setState({ ...initialOfficeLiveState });
}
