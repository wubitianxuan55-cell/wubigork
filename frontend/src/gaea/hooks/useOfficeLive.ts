// useOfficeLive — COM 实时编辑状态喂入 + office_file_saved 保存跟随（波3）。
//
// 两路职责：
//   1. 工具卡事件流 → lib/officeLiveStore：照 usePreviewAutoFront 的
//      items 遍历模式（订阅 ControllerState.items，比对工具卡状态跃迁，
//      基线期只登记不触发——恢复会话/补拉快照里的历史 office 卡不得点亮
//      「实时编辑」），对 name 以 office_ 开头的工具卡调 noteActivity。
//      turnEnd 不清——活动文档跨回合（AI 常跨多回合持续编辑同一份文档），
//      收尾靠 office_close 回执 + store 内 5 分钟惰性过期（无定时器）。
//   2. gaea-event 的 office_file_saved 事件 → paneTabs.requestReload：
//      settle 保存的文档此刻若开着预览（pane 文件 tab / 主区大预览）则
//      静默重读（requestReload 自带归一键；路径没开着预览就是空转，绝不
//      打开任何东西）。usePreviewAutoFront 的 U4 调度器语义冻结，此处是
//      独立轻通道，不与其共用状态。
//
// 挂载模型：喂入是**引用计数单例**（本 hook 可在任意多个 FilePreview 实例
// 里挂，首个宿主启动订阅、末个宿主卸载才退订——多实例重复扫描既浪费又可
// 能互相干扰基线）。事件注销走 onEvent 封装（v4.62.2 纪律：只摘本监听者，
// 绝不 EventsOff 全清）。
//
// 纯逻辑（items 扫描、事件解析）抽为本文件导出的纯函数，单测钉死；hook
// 本体只是接线。

import { useEffect } from "react";
import { useStore } from "../lib/store";
import { usePaneTabsStore } from "../lib/paneTabs";
import { onEvent } from "../lib/bridge";
import {
  useOfficeLiveStore,
  type OfficeLiveNote,
} from "../lib/officeLiveStore";

// ── 纯函数：工具卡扫描（officeTurnProjection.test 同风格单测钉死）─────────

/** 宽容形状的工具条目（Item 联合的 tool 成员子集，免引 store 类型）。 */
export interface OfficeToolItemLike {
  kind: unknown;
  id?: string;
  name?: string;
  args?: string;
  status?: string;
  output?: string;
  error?: string;
}

export interface OfficeLiveScan {
  /** 本扫描要喂入 store 的活动（已按跃迁/基线过滤）。 */
  notes: OfficeLiveNote[];
  /** 扫描后的工具卡状态基线（id → status）。 */
  seen: Map<string, string>;
  /** 首扫建基线期=true；扫过一次后恒 false。 */
  baseline: boolean;
}

/**
 * 扫描 items → office_* 工具活动集（纯函数）。
 *   - 只看 kind==="tool" 且 name 以 office_ 开头的条目；
 *   - 状态没变的条目跳过（同一条 running 卡随流式刷新反复出现不重复喂）；
 *   - 基线期（会话装载/恢复/补拉后的首扫）只登记不触发；
 *   - now 由调用方注入（同一次扫描的活动共享同一时刻，纯函数可测）。
 */
export function officeLiveScanItems(
  items: readonly OfficeToolItemLike[],
  seen: ReadonlyMap<string, string>,
  baseline: boolean,
  now: number,
): OfficeLiveScan {
  const nextSeen = new Map(seen);
  const notes: OfficeLiveNote[] = [];
  for (const it of items) {
    if (!it || it.kind !== "tool") continue;
    if (typeof it.name !== "string" || !it.name.startsWith("office_")) continue;
    const id = typeof it.id === "string" ? it.id : "";
    if (!id) continue;
    const prev = nextSeen.get(id);
    const status = typeof it.status === "string" ? it.status : "";
    if (prev === status) continue;
    nextSeen.set(id, status);
    if (baseline && prev === undefined) continue; // 基线期只登记
    notes.push({ tool: it.name, args: it.args, status, output: it.output, error: it.error, now });
  }
  return { notes, seen: nextSeen, baseline: false };
}

// ── 纯函数：office_file_saved 事件解析 ─────────────────────────────────

export interface OfficeFileSavedPayload {
  /** 落盘绝对路径（Go emit 侧约定；可能缺省）。 */
  path: string;
  /** 工作区相对路径（优先用它请求刷新，与预览登记口径一致）。 */
  rel: string;
}

/**
 * 解析 office_file_saved 事件负载（宽容两代形状）：
 *   - Wails 顶层数组包装（EventsEmit 变参）→ 取首个元素再判；
 *   - 事件名可能在 kind 键（WireEvent 同式，usePreviewProgress 口径）或
 *     约定的 event 键。认不出返回 null（静默忽略，不打扰）。
 * path/rel 至少要有一个才有效。
 */
export function parseOfficeFileSavedEvent(payload: unknown): OfficeFileSavedPayload | null {
  let raw: unknown = payload;
  if (Array.isArray(raw)) raw = raw[0];
  if (raw === null || typeof raw !== "object" || Array.isArray(raw)) return null;
  const rec = raw as Record<string, unknown>;
  const name = typeof rec.kind === "string" ? rec.kind : typeof rec.event === "string" ? rec.event : "";
  if (name !== "office_file_saved") return null;
  const path = typeof rec.path === "string" ? rec.path : "";
  const rel = typeof rec.rel === "string" ? rec.rel : "";
  if (!path && !rel) return null;
  return { path, rel };
}

// ── 引用计数单例喂入 ───────────────────────────────────────────────────

interface FeedSession {
  /** 工具卡状态基线（id → status）。 */
  seen: Map<string, string>;
  /** 基线期标记：启动/会话整体替换后的首扫只登记。 */
  baseline: boolean;
  /** items 首条 id 锚点：变化 = 会话装载/恢复/补拉整体替换 → 重建基线。 */
  anchor: string;
  /** 退订 items 监听 + office_file_saved 监听。 */
  stop: () => void;
}

let feedSession: FeedSession | null = null;
let feedRefCount = 0;

/**
 * 扫描一次 items 并把活动喂入 store（feed 内部共用）。
 * 锚点变化（首条 id 换人）= 会话内容整体替换（usePreviewAutoFront 用
 * currentSessionKey 复位同式；这里吃不到 App 的会话 key，用锚点代偿：
 * live 回合只追加 items、首条 user 卡恒定，整体替换必然换锚点）。
 */
function scanAndFeed(session: FeedSession, now: number): void {
  const items = useStore.getState().items;
  const anchor = items.length > 0 && typeof items[0]?.id === "string" ? (items[0].id as string) : "";
  if (anchor !== session.anchor) {
    session.anchor = anchor;
    session.seen = new Map();
    session.baseline = true;
  }
  const scan = officeLiveScanItems(items, session.seen, session.baseline, now);
  session.seen = scan.seen;
  session.baseline = scan.baseline;
  const note = useOfficeLiveStore.getState().noteActivity;
  for (const n of scan.notes) note(n.tool, n.args, n.status, n.output, n.error);
}

function startOfficeLiveFeed(): () => void {
  feedRefCount++;
  if (feedSession) return () => {
    feedRefCount = Math.max(0, feedRefCount - 1);
    if (feedRefCount === 0) {
      feedSession?.stop();
      feedSession = null;
    }
  };
  const session: FeedSession = {
    seen: new Map(),
    baseline: true,
    anchor: "",
    stop: () => {},
  };
  const unItems = useStore.subscribe((s, prev) => {
    if (s.items === prev.items) return;
    scanAndFeed(session, Date.now());
  });
  // office_file_saved 保存跟随：独立订阅（注销只摘本监听者，见 onEvent 封装）。
  const unSaved = onEvent((payload: unknown) => {
    const saved = parseOfficeFileSavedEvent(payload);
    if (!saved) return;
    const target = saved.rel || saved.path;
    if (target) usePaneTabsStore.getState().requestReload(target);
  });
  session.stop = () => {
    unItems();
    unSaved();
  };
  // 启动即建基线：此刻已在 items 里的历史 office 卡只登记不触发。
  scanAndFeed(session, Date.now());
  feedSession = session;
  return () => {
    feedRefCount = Math.max(0, feedRefCount - 1);
    if (feedRefCount === 0) {
      feedSession?.stop();
      feedSession = null;
    }
  };
}

/**
 * COM 实时编辑前端接线（自包含订阅：不吃 props，宿主可零改动挂载）。
 * 预期宿主：FilePreview（本特性唯一展示位，多实例安全）。
 */
export function useOfficeLive(): void {
  useEffect(() => startOfficeLiveFeed(), []);
}
