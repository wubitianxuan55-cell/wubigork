// subagentRunsStore.ts — GaeaSubagentRuns 的共享单轮询聚合（v4.63，对标
// dsh-better-sidebar 的 subagents.live 客户端纪律：单轮询、单在途请求）。
//
// Why: 此前同一绑定的消费点各自维护 5s 轮询（App 会话 tab 同步 / task 卡
// live 活动 / 任务树 / 左栏子行）——同一份数据 N 份请求、N 份缓存、状态
// 互相错拍。dsh 在 #298 修过同样的问题（per-child 轮询 → subagents.live
// 一次枚举整树 + 客户端单轮询）。本模块把轮询收敛为「每会话一个定时器、
// 任意时刻至多一个在途请求」，快照广播给所有订阅者；task 卡活动行等非
// React 场景走 getSubagentRunsSnapshot() 非响应式读。
//
// 契约：
//  - subscribeSubagentRuns(path, cb)：注册订阅并立即拉一次；同会话切换 /
//    最后一个订阅者注销即停表（refcount）。会话切换后首个快照照常广播。
//  - 拉取失败保留上一份快照、静默重试（宁旧勿断，与既有轮询口径一致）。
//  - 页面不可见（document.hidden）时 tick 跳过——与 usePollingGate 同语义。

import { app } from "./bridge";
import type { SubagentRunView } from "./types";

type Listener = (runs: SubagentRunView[]) => void;

let sessionPath = "";
let snapshot: SubagentRunView[] = [];
let timer = 0;
let inFlight = false;
const listeners = new Set<Listener>();

const POLL_MS = 5000;

function notify(): void {
  for (const l of listeners) l(snapshot);
}

function pull(): void {
  if (inFlight || !sessionPath) return;
  inFlight = true;
  void app
    .SubagentRuns(sessionPath)
    .then((v) => {
      snapshot = v?.runs ?? [];
      notify();
    })
    .catch(() => {
      /* 拉取失败：保留上一份快照，下个 tick 再试 */
    })
    .finally(() => {
      inFlight = false;
    });
}

function start(path: string): void {
  stop();
  sessionPath = path;
  snapshot = [];
  pull(); // 订阅/切会话立即拉一次，不等首个 tick
  timer = window.setInterval(() => {
    if (typeof document !== "undefined" && document.hidden) return;
    pull();
  }, POLL_MS);
}

function stop(): void {
  if (timer) window.clearInterval(timer);
  timer = 0;
  sessionPath = "";
  snapshot = [];
  inFlight = false;
}

/** 订阅当前会话的子代理运行快照；返回注销函数（最后一个注销者停表）。 */
export function subscribeSubagentRuns(path: string, cb: Listener): () => void {
  const first = listeners.size === 0 || sessionPath !== path;
  listeners.add(cb);
  if (first) start(path);
  else cb(snapshot); // 已有轮询：补发当前快照，订阅者立即有数据
  return () => {
    listeners.delete(cb);
    if (listeners.size === 0) stop();
  };
}

/** 非响应式读取最新快照（task 卡活动 provider 等非 React 场景）。 */
export function getSubagentRunsSnapshot(): SubagentRunView[] {
  return snapshot;
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
