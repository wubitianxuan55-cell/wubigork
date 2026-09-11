// agentNetworkStore.ts — GaeaAgentNetwork 的共享单轮询聚合（v4.65，模式对标
// subagentRunsStore：单定时器、单在途、订阅即拉、不可见门控、loading/ready/
// error 状态机、显式 reload）。
//
// Why: 右栏子代理树的树拓扑数据源此前由 SubagentsPanel 自管 5s 轮询
// （usePollingGate 门控），TasksWorkbench 另有一份自己的 net 轮询——同一个
// 绑定 N 份请求、N 份状态互相错拍。本模块把轮询收敛为「全局一个轮询器」，
// 快照广播给所有订阅者；拉取失败保留上一份快照、status 置 error（宁旧勿断），
// 订阅者可渲染重试入口，live 轮询下个 tick 自动再试自愈。
//
// 为什么单例而不是像 subagentRunsStore 按路径建册：所有消费方（上下文页/
// 任务管理页/右栏子代理面板）看的都是同一个「UI 当前会话」，按路径建册只会
// 得到 N 份相同 payload 的重复轮询。路径跟随最近一次订阅/reload 的声明；
// 跨会话切换清空快照（旧会话树=误导数据，宁空勿错），同会话重试保留旧快照
// （宁旧勿断口径不变）。绑定按会话读取（v4.181：GaeaAgentNetwork 变参
// sessionPath，缺省=内核当前会话）。
//
// 契约：
//  - subscribeAgentNetwork(cb, opts?)：注册订阅并立即拉一次；全员共享轮询器，
//    最后一个订阅者注销即停表弃册（下次订阅重建重拉）。opts.live = false 声明
//    「快照订阅」：只参与订阅即拉与显式 reload，不驱动 5s tick（与
//    subagentRunsStore 同语义，为后续静态消费点留结构）。opts.path 声明查询
//    会话（UI 当前会话路径；缺省=内核会话），与轮询器现路径不同则清快照立即重拉。
//  - cb(net, meta)：meta.status ∈ loading/ready/error。首拉成功前快照为 null。
//  - reloadAgentNetwork(path?)：显式重拉（失败重试/手动刷新/会话切换共用），
//    path 语义同上；置 loading 通知后立即拉取；在途合并、无订阅者 no-op。
//  - getAgentNetworkSnapshot()：非响应式读取快照（非 React 场景预留）。
//  - 页面不可见（document.hidden）时 tick 跳过——与 usePollingGate 同语义。

import { app } from "./bridge";
import type { AgentNetwork } from "./types";

export type AgentNetworkStatus = "loading" | "ready" | "error";

/** 随快照广播的拉取状态：loading=拉取在途；error=最近一次拉取失败（快照为上一份成功数据）。 */
export interface AgentNetworkMeta {
  status: AgentNetworkStatus;
}

type Listener = (net: AgentNetwork | null, meta: AgentNetworkMeta) => void;

interface Subscription {
  cb: Listener;
  live: boolean;
}

/** 全局唯一轮询器：单定时器、单在途、快照 + 状态。path=当前查询的 UI 会话
 * （""=内核当前会话），跟随最近一次订阅/reload 的声明。 */
interface Poller {
  snapshot: AgentNetwork | null;
  meta: AgentNetworkMeta;
  timer: number;
  inFlight: boolean;
  path: string;
  subs: Set<Subscription>;
}

const POLL_MS = 5000;
let poller: Poller | null = null;

function notify(p: Poller): void {
  const meta = { ...p.meta };
  for (const sub of p.subs) sub.cb(p.snapshot, meta);
}

/** 声明查询路径：跨会话切换清空快照（旧会话树=误导数据，宁空勿错），
 * 同会话重试保留旧快照。路径变化即触发立即拉取。 */
function declarePath(p: Poller, path: string | undefined): void {
  const next = path ?? p.path;
  if (next === p.path) return;
  p.path = next;
  p.snapshot = null;
  p.meta = { status: "loading" };
  notify(p);
  pull();
}

function pull(): void {
  const p = poller;
  if (!p || p.inFlight) return;
  p.inFlight = true;
  void app
    .AgentNetwork(p.path || "") // Wails 要求逐参显式传：空串=内核当前会话（resolveGaeaSessionPath 跳过空值）
    .then((n) => {
      p.snapshot = n ?? null;
      p.meta = { status: "ready" };
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
function syncTicker(p: Poller): void {
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
      pull();
    }, POLL_MS);
  } else if (!live && p.timer) {
    window.clearInterval(p.timer);
    p.timer = 0;
  }
}

/** 订阅子代理树快照；返回注销函数（最后一个注销者停表弃册）。opts.path
 * 声明查询会话（UI 当前会话；缺省=内核会话），跨会话切换清快照立即重拉。 */
export function subscribeAgentNetwork(
  cb: Listener,
  opts?: { live?: boolean; path?: string },
): () => void {
  const sub: Subscription = { cb, live: opts?.live !== false };
  if (!poller) {
    poller = { snapshot: null, meta: { status: "loading" }, timer: 0, inFlight: false, path: opts?.path ?? "", subs: new Set() };
    poller.subs.add(sub);
    syncTicker(poller);
    pull(); // 订阅即拉一次，不等首个 tick
  } else {
    poller.subs.add(sub);
    syncTicker(poller);
    declarePath(poller, opts?.path);
    sub.cb(poller.snapshot, { ...poller.meta }); // 迟到订阅者：补发当前快照与状态，立即有数据
  }
  return () => {
    const p = poller;
    if (!p || !p.subs.delete(sub)) return;
    if (p.subs.size === 0) {
      if (p.timer) window.clearInterval(p.timer);
      if (poller === p) poller = null; // 弃册：下次订阅重建重拉，不陈货转售
    } else {
      syncTicker(p);
    }
  };
}

/** 显式重拉（失败重试 / 手动刷新 / 会话切换）：path 与现路径不同 → 清快照
 * 立即切换重拉（在途旧路径请求由下个 tick 自愈覆盖）；同路径 → 原语义置
 * loading 重拉，在途合并；无订阅者 no-op。 */
export function reloadAgentNetwork(path?: string): void {
  const p = poller;
  if (!p) return;
  if (path !== undefined && path !== p.path) {
    declarePath(p, path);
    return;
  }
  if (p.inFlight) return;
  p.meta = { ...p.meta, status: "loading" };
  notify(p);
  pull();
}

/** 非响应式读取快照（非 React 场景预留）。首拉成功前为 null。 */
export function getAgentNetworkSnapshot(): AgentNetwork | null {
  return poller?.snapshot ?? null;
}
