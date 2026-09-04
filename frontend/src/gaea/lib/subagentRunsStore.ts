// subagentRunsStore.ts — GaeaSubagentRuns 的共享单轮询聚合（v4.63，对标
// dsh-better-sidebar 的 subagents.live 客户端纪律：单轮询、单在途请求）。
//
// Why: 此前同一绑定的消费点各自维护 5s 轮询（App 会话 tab 同步 / 分工面板 /
// 左栏子行）——同一份数据 N 份请求、N 份缓存、状态互相错拍。dsh 在 #298 修过
// 同样的问题（per-child 轮询 → subagents.live 一次枚举整树 + 客户端单轮询）。
// 本模块把轮询收敛为「每会话一个轮询器：单定时器、任意时刻至多一个在途请求」，
// 快照广播给该会话的所有订阅者。v4.64 起轮询器按路径建册（多会话并存：左栏
// 按需订阅历史会话子行），并补齐 loading/ready/error 状态与显式 reload——
// 拉取失败不再静默，订阅者可渲染重试入口。
//
// 契约：
//  - subscribeSubagentRuns(path, cb, opts?)：注册订阅并立即拉一次；同路径共
//    享轮询器，最后一个订阅者注销即停表。opts.live = false 声明「快照订阅」：
//    只参与订阅即拉与显式 reload，不驱动 5s tick（左栏静态子行用——当前会话
//    行的实时性由 App/面板的 live 订阅兜底，不为静态历史数据空转轮询）。
//  - cb(runs, meta)：meta.status ∈ loading/ready/error，并带 available/total/
//    running。旧消费者只读第一个参数（React setState 天然忽略多余实参），
//    向后兼容。
//  - 拉取失败保留上一份快照、status 置 error（宁旧勿断，与既有轮询口径一致），
//    订阅者据此给重试入口；live 轮询器下个 tick 自动再试自愈。
//  - reloadSubagentRuns(path)：显式重拉（失败重试/手动刷新共用），在途合并。
//  - 页面不可见（document.hidden）时 tick 跳过——与 usePollingGate 同语义。

import { app } from "./bridge";
import type { SubagentRunView } from "./types";

export type SubagentRunsStatus = "loading" | "ready" | "error";

/** 随快照广播的拉取状态：loading=拉取在途；error=最近一次拉取失败（快照为上一份成功数据）。 */
export interface SubagentRunsMeta {
  status: SubagentRunsStatus;
  /** false = 会话无 subagents 目录（从未派发子代理）；首次成功拉取前为 undefined。 */
  available?: boolean;
  total: number;
  running: number;
}

type Listener = (runs: SubagentRunView[], meta: SubagentRunsMeta) => void;

interface Subscription {
  cb: Listener;
  live: boolean;
}

/** 每会话一个轮询器：单定时器、单在途、快照 + 状态。 */
interface Poller {
  snapshot: SubagentRunView[];
  meta: SubagentRunsMeta;
  timer: number;
  inFlight: boolean;
  subs: Set<Subscription>;
}

const POLL_MS = 5000;
const pollers = new Map<string, Poller>();

function metaOf(p: Poller): SubagentRunsMeta {
  return { ...p.meta };
}

function notify(p: Poller): void {
  for (const sub of p.subs) sub.cb(p.snapshot, metaOf(p));
}

function pull(path: string, p: Poller): void {
  if (p.inFlight) return;
  p.inFlight = true;
  void app
    .SubagentRuns(path)
    .then((v) => {
      p.snapshot = v?.runs ?? [];
      p.meta = {
        status: "ready",
        available: v?.available,
        total: v?.total ?? p.snapshot.length,
        running: v?.running ?? 0,
      };
      notify(p);
    })
    .catch(() => {
      // 拉取失败：保留上一份快照，置 error 通知订阅者渲染重试入口，下个 tick 再试
      p.meta = { ...p.meta, status: "error" };
      notify(p);
    })
    .finally(() => {
      p.inFlight = false;
    });
}

/** 按订阅者的 live 诉求启停 tick：全员快照订阅（live:false）时不空转轮询。 */
function syncTicker(path: string, p: Poller): void {
  let live = false;
  for (const s of p.subs) {
    if (s.live) {
      live = true;
      break;
    }
  }
  if (live && !p.timer) {
    p.timer = window.setInterval(() => {
      if (typeof document !== "undefined" && document.hidden) return;
      pull(path, p);
    }, POLL_MS);
  } else if (!live && p.timer) {
    window.clearInterval(p.timer);
    p.timer = 0;
  }
}

/** 订阅指定会话的子代理运行快照；返回注销函数（该路径最后一个注销者停表）。 */
export function subscribeSubagentRuns(
  path: string,
  cb: Listener,
  opts?: { live?: boolean },
): () => void {
  const sub: Subscription = { cb, live: opts?.live !== false };
  let p = pollers.get(path);
  if (!p) {
    p = { snapshot: [], meta: { status: "loading", total: 0, running: 0 }, timer: 0, inFlight: false, subs: new Set() };
    pollers.set(path, p);
    p.subs.add(sub);
    syncTicker(path, p);
    pull(path, p); // 订阅即拉一次，不等首个 tick
  } else {
    p.subs.add(sub);
    syncTicker(path, p);
    cb(p.snapshot, metaOf(p)); // 迟到订阅者：补发当前快照与状态，立即有数据
  }
  return () => {
    if (!p.subs.delete(sub)) return;
    if (p.subs.size === 0) {
      if (p.timer) window.clearInterval(p.timer);
      if (pollers.get(path) === p) pollers.delete(path);
    } else {
      syncTicker(path, p);
    }
  };
}

/** 显式重拉（失败重试 / 手动刷新）：置 loading 通知后立即拉取；在途合并、未订阅路径 no-op。 */
export function reloadSubagentRuns(path: string): void {
  const p = pollers.get(path);
  if (!p || p.inFlight) return;
  p.meta = { ...p.meta, status: "loading" };
  notify(p);
  pull(path, p);
}

/** 非响应式读取快照（task 卡活动 provider 等非 React 场景）。带 path 读指定会话；省略时仅在恰有一个轮询器时返回该快照。 */
export function getSubagentRunsSnapshot(path?: string): SubagentRunView[] {
  if (path !== undefined) return pollers.get(path)?.snapshot ?? [];
  if (pollers.size === 1) {
    for (const p of pollers.values()) return p.snapshot;
  }
  return [];
}

/**
 * 纯函数：返回 runs 里 seen 集合中没有的新 ref（0→N / 新子代理检测）。
 * 不修改 seen——由调用方决定何时并入（自动展开的去抖重评估依赖这一点）。
 */
export function detectNewRunRefs(seen: ReadonlySet<string>, runs: SubagentRunView[]): string[] {
  const out: string[] = [];
  for (const r of runs) {
    if (r.ref && !seen.has(r.ref)) out.push(r.ref);
  }
  return out;
}
