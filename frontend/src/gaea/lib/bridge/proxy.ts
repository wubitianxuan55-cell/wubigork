// proxy.ts — 调用时路由代理层（S2.3「App 绑定面拆分」）：app/workApp/playApp/
// sharedApp 按方法名路由到 window.go.app 各板块门面（live binding）或 dev mock；
// resolveBinding 统一走 invoke 错误归一（BridgeError）。
// realApp 为 events 模块共享的内部接缝（非入口公开面，bridge.ts 不 re-export）。
import type { AppBindings } from "./appBindings";
import { gaeaToGaea } from "./mappings";
import {
  isBindingAllowedInSpace,
  isSharedBinding,
  type GaeaFacetBySpace,
} from "../spaceBindings";

// Resolve the Wails binding at CALL time, not module-load time: in dev the Wails
// runtime can inject window.go AFTER this module first evaluates, so snapshotting
// once would pin the browser mock for the whole session (and show fake data — the
// dev mock's model list leaking into the real app was exactly this bug).
//
// S2-3「App 绑定面拆分」：后端绑定面已从单一 window.go.app.App 拆为多个板块
// 门面（go.app.CoreB/OfficeB/MemoryB/CostB/ModelB/VoiceB/ChatB/NovelB/
// ImageB/CharLibB）。这里返回一个按方法名路由到对应门面的代理，前端调用点
// （app.Submit 等）零改动。
export function realApp(): AppBindings | undefined {
  if (typeof window === "undefined") return undefined;
  const goApp = (window as unknown as { go?: { app?: Record<string, unknown> } }).go?.app;
  if (!goApp || typeof goApp !== "object") return undefined;
  return new Proxy({} as AppBindings, {
    get(_t, prop) {
      const key = (gaeaToGaea as Record<string, string>)[String(prop)] ?? String(prop);
      for (const ns of Object.values(goApp)) {
        if (ns === null || typeof ns !== "object") continue;
        const rec = ns as Record<string, unknown>;
        const v = rec[key];
        if (typeof v === "function") return (v as (...a: unknown[]) => unknown).bind(rec);
      }
      return undefined;
    },
  });
}

// ── H6（entry 懒加载）：dev mock 独立异步 chunk ──────────────────────────
// mock 实现（约 150-190KB）不再静态导入 entry：模块顶层即触发 import()，Vite
// 会把 mock 拆为独立 chunk 异步加载，首帧 entry 主解析不阻塞。真机（Wails，
// window.go.app 存在）路径由 realApp() 恒命中，永不加载 mock chunk（预取有
// 启动探测门控）；浏览器 dev 首个绑定调用发生在 UI 挂载之后，chunk 早已就绪
// （同步 fast path）。首个调用过早的极端情形由冷路径兜底：resolveBinding 与
// events.ts 订阅在 chunk 就绪前返回「等就绪再分发」的异步 thunk，功能等价。
export type MockEventShared = Pick<
  typeof import("../mock"),
  "mockSubscribe" | "mockTaskSubscribe" | "updaterListeners"
>;

let mockSingleton: AppBindings | null = null;
let mockEventShared: MockEventShared | null = null;
let mockChunkPromise: Promise<typeof import("../mock")> | null = null;

function startMockChunk(): Promise<typeof import("../mock")> {
  if (!mockChunkPromise) {
    mockChunkPromise = import("../mock").then((m) => {
      mockSingleton = m.makeMockApp();
      mockEventShared = {
        mockSubscribe: m.mockSubscribe,
        mockTaskSubscribe: m.mockTaskSubscribe,
        updaterListeners: m.updaterListeners,
      };
      return m;
    });
  }
  return mockChunkPromise;
}

// 启动探测门控：仅当此刻探测不到 live 绑定才预取。Wails 真机 window.go.app
// 已注入 → 不触发加载（零额外开销）；浏览器 dev / vitest → 尽早拉取。
if (
  typeof window === "undefined" ||
  !(window as unknown as { go?: { app?: unknown } }).go?.app
) {
  void startMockChunk();
}

/** 事件订阅的 mock 回退面（chunk 未就绪时为 null；events.ts 冷路径等就绪后取用）。 */
export function mockEventSharedSync(): MockEventShared | null {
  return mockEventShared;
}

/** 测试/调试用：mock chunk 就绪 Promise（真机调用 resolve 且不加载 mock）。 */
export function waitMockReady(): Promise<unknown> {
  return startMockChunk();
}

// ── 错误归一化层（T6-1.2 前端错误可见性）────────────────────────────
// BridgeError 是所有绑定调用失败时归一出的结构化错误：code 机器可读
// （后端已带 code 时透传，否则 "<方法名>Error"），message 为人类可读原因
// （后端错误信息原文）。继承 Error 保证既有调用方的 e instanceof Error /
// e.message 判定不受影响（message 保留后端原文）。
export class BridgeError extends Error {
  code: string;
  constructor(code: string, message: string) {
    super(message);
    this.name = "BridgeError";
    this.code = code;
    // Error.message 默认不可枚举（JSON 序列化/结构化断言拿不到），显式
    // 重设为可枚举 own 属性，保证 { code, message } 结构对外稳定。
    Object.defineProperty(this, "message", { value: message, enumerable: true, writable: true, configurable: true });
  }
}

// normalizeError 把任意拒绝值归一为 BridgeError：已结构化（code/message）
// 的错误透传，Error 取 message，其余兜底字符串化。
function normalizeError(method: string, err: unknown): BridgeError {
  if (err instanceof BridgeError) return err;
  if (err && typeof err === "object") {
    const cand = err as { code?: unknown; message?: unknown };
    if (typeof cand.code === "string" && typeof cand.message === "string") {
      return new BridgeError(cand.code, cand.message);
    }
  }
  if (err instanceof Error) {
    return new BridgeError(`${method}Error`, err.message || String(err));
  }
  return new BridgeError(`${method}Error`, String(err ?? "unknown error"));
}

// invoke 是所有绑定调用的统一入口：失败时把错误归一为 BridgeError 并记录
// 到 gaea.log（LogFrontendError），再以同样的拒绝语义抛给调用方——调用方
// 原有的 .catch 行为契约不变（仍拿到 rejected promise + 错误值）。
function invoke(method: string, fn: (...args: unknown[]) => unknown, args: unknown[]): Promise<unknown> {
  return Promise.resolve()
    .then(() => fn(...args))
    .catch((err: unknown) => {
      const normalized = normalizeError(method, err);
      // 记录到 gaea.log；日志通道自身故障不向上抛，避免掩盖原始错误。
      // LogFrontendError 在 app proxy 里不套本层（见下），不会递归。
      logFrontendError(`[${normalized.code}] ${method} 失败: ${normalized.message}`);
      throw normalized;
    });
}

// logFrontendError 上报错误到 gaea.log；日志通道不可用（dev mock 外未注入
// 绑定）或自身失败时静默降级，绝不掩盖原始错误。
function logFrontendError(message: string): void {
  const lfe = app.LogFrontendError;
  if (typeof lfe !== "function") return;
  void Promise.resolve(lfe(message)).catch(() => {});
}

/** 解析单个绑定：按方法名路由到 live binding 或 dev mock（无空间门控的通用解析）。 */
function resolveBinding(prop: string | symbol): unknown {
  const key = (gaeaToGaea as Record<string, string>)[String(prop)] ?? String(prop);
  const target = realApp() ?? mockSingleton;
  if (target) {
    const rec = target as unknown as Record<string, unknown>;
    // 真实绑定按 Gaea 前缀查找；浏览器 mock 直接暴露同名字段，需回退。
    const v = rec[key] ?? rec[String(prop)];
    if (typeof v !== "function") return v;
    const bound = (v as (...a: unknown[]) => unknown).bind(target);
    // LogFrontendError 是错误上报通道自身，不套 invoke 归一化层，避免日志
    // 通道故障时无限递归；其余方法统一走 invoke。
    if (String(prop) === "LogFrontendError") return bound;
    return (...args: unknown[]) => invoke(String(prop), bound, args);
  }
  // 冷路径（纯 dev/测试时序竞态：mock chunk 尚未就绪；真机不可达——realApp 恒命中）：
  // 返回异步兜底 thunk，等 chunk 就绪后按同一路由分发，功能等价（仅首次微异步）。
  // 注意：必须复用 mockSingleton（chunk 的 .then 在本 Promise resolve 前已赋值），
  // 不能把模块命名空间当 app 用——否则拿到 makeMockApp 而非绑定方法。
  return async (...args: unknown[]) => {
    await startMockChunk();
    if (!mockSingleton) return undefined;
    const rec = mockSingleton as unknown as Record<string, unknown>;
    const v = rec[key] ?? rec[String(prop)];
    if (typeof v !== "function") return v;
    const bound = (v as (...a: unknown[]) => unknown).bind(mockSingleton);
    if (String(prop) === "LogFrontendError") return bound;
    return invoke(String(prop), bound, args);
  };
}

// app proxies each call to the live binding (or the dev mock only when truly
// outside the shell), so a late-injected window.go is picked up transparently.
export const app: AppBindings = new Proxy({} as AppBindings, {
  get(_t, prop) {
    return resolveBinding(prop);
  },
});

// ── S2.3 bridge 分面（docs/gaea-space-shell-design.md §7）────────────────
// 类型级门面：work/play 各自只暴露「所属空间 + shared + independent」的方法，
// play 页面引用 work 专属方法会 tsc 报错；运行时同样按 spaceBindings 门控
// （越界方法返回 undefined → TypeError，双保险）。sharedApp 只暴露 shared。
function createSpaceFacade<S extends "work" | "play">(space: S): GaeaFacetBySpace[S] {
  return new Proxy({} as GaeaFacetBySpace[S], {
    get(_t, prop) {
      if (prop === "then") return undefined; // 避免被误判为 Promise
      if (!isBindingAllowedInSpace(String(prop), space)) return undefined;
      return resolveBinding(prop);
    },
  });
}

/** 工位门面（work + shared + independent）——办公工作台专用。 */
export const workApp = createSpaceFacade("work");
/** 乐园门面（play + shared + independent）——轻语/小说/绘梦等页面专用。 */
export const playApp = createSpaceFacade("play");
/** 共用门面（仅 shared）——壳层/设置等两空间共用代码专用。 */
export const sharedApp: GaeaFacetBySpace["work"] & GaeaFacetBySpace["play"] = new Proxy(
  {} as GaeaFacetBySpace["work"] & GaeaFacetBySpace["play"],
  {
    get(_t, prop) {
      if (prop === "then") return undefined;
      if (!isSharedBinding(String(prop))) return undefined;
      return resolveBinding(prop);
    },
  },
);

// openExternal opens a URL in the system browser (so links in rendered markdown
// don't navigate the webview away from the app). Falls back to window.open in the
// browser dev mock.
export function openExternal(url: string): void {
  if (typeof window !== "undefined" && window.runtime?.BrowserOpenURL) {
    window.runtime.BrowserOpenURL(url);
  } else if (typeof window !== "undefined") {
    window.open(url, "_blank", "noopener");
  }
}

