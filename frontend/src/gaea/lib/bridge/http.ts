// http.ts — initBridge（S2-2 移动端 HTTP 桥接 + S2-3 旧形态兼容层）。
import { getHttpToken } from "../../../api/httpToken";
import { gaeaToGaea } from "./mappings";

// ── 桥接归一（T6-10.6）─────────────────────────────────────────────────
// initBridge 原为 frontend/src/api/bridge.ts（S2-2 移动端 HTTP 桥接 + S2-3 旧
// 形态兼容层），T6-10.6 并入本文件，由 App.tsx 在模块作用域最早时机调用：
//   · Wails 原生环境：window.go.app 下是各板块门面（CoreB/OfficeB/...），补一个
//     window.go.app.App 兼容代理，旧调用点（window.go.app.App.Xxx()）按方法名路由；
//   · HTTP 环境（移动端/调试页，非 Wails）：把 window.go.app.App 挂为
//     fetch('/api/rpc') 的 RPC 代理（Bearer token 鉴权，token 来源见 httpToken.ts）。
// 本文件顶层的 app 代理在调用时经 realApp() 感知上述两种注入，无需额外适配。

const API_BASE = "";

interface RPCResponse {
  result?: unknown;
  error?: string;
}

/** 是否运行在 Wails 原生环境：window.go.app 下存在任一板块门面绑定对象。 */
function isWailsNative(): boolean {
  const goApp = (window as unknown as { go?: { app?: Record<string, unknown> } }).go?.app;
  return !!goApp && typeof goApp === "object" && Object.keys(goApp).length > 0;
}

/**
 * S2-3 兼容层：为旧调用点补 window.go.app.App 代理，按方法名路由到对应门面。
 *
 * v4.257.1 修复：兼容代理原先只按**字面名**在各门面里找方法，认不出
 * gaeaToGaea 的短名映射——例如旧调用点写 `window.go.app.App.AttachmentDataURL()`，
 * 而 OfficeB 上的真名是 `GaeaAttachmentDataURL`，查找落空返回 undefined，
 * 调用即抛「AttachmentDataURL is not a function」。现在字面名查不到时回退一次
 * 映射名（短名 → Go 方法名），与 bridge 代理同口径；仍未命中才返回 undefined
 * （诚实失败，不静默兜底假实现）。
 */
function ensureLegacyAppProxy(): void {
  const goApp = (window as unknown as { go?: { app?: Record<string, unknown> } }).go?.app;
  if (!goApp || typeof goApp !== "object") return;
  if (goApp.App) return;
  goApp.App = new Proxy(
    {},
    {
      get(_t, prop: string) {
        if (prop === "then") return undefined; // 避免被误判为 Promise
        // 候选名：字面名优先，其次 gaeaToGaea 映射出的 Go 方法名
        const candidates = [String(prop)];
        const mapped = (gaeaToGaea as Record<string, string>)[String(prop)];
        if (mapped && mapped !== prop) candidates.push(mapped);
        for (const name of candidates) {
          for (const ns of Object.values(goApp)) {
            if (ns === goApp.App || ns === null || typeof ns !== "object") continue;
            const v = (ns as Record<string, unknown>)[name];
            if (typeof v === "function") return (v as (...a: unknown[]) => unknown).bind(ns);
          }
        }
        return undefined;
      },
    },
  );
}

/** 移动端 RPC 调用：POST /api/rpc，携带一次性 token（S2-2）。 */
async function rpcCall(method: string, ...args: unknown[]): Promise<unknown> {
  const headers: Record<string, string> = { "Content-Type": "application/json" };
  const token = getHttpToken();
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }
  const res = await fetch(`${API_BASE}/api/rpc`, {
    method: "POST",
    headers,
    body: JSON.stringify({ method, args }),
  });
  if (res.status === 401) {
    throw new Error("桥接鉴权失败：请携带正确的 token（__GAEA_HTTP_TOKEN / localStorage gaea_http_token）");
  }
  if (!res.ok) {
    throw new Error(`RPC 请求失败: ${res.status} ${res.statusText}`);
  }
  const data: RPCResponse = await res.json();
  if (data.error) {
    throw new Error(data.error);
  }
  return data.result;
}

/** HTTP 模式 App 代理：拦截所有方法调用并转发到 /api/rpc。 */
function createAppProxy(): Record<string, (...args: unknown[]) => Promise<unknown>> {
  return new Proxy(
    {},
    {
      get(_target, prop: string) {
        return (...args: unknown[]) => rpcCall(prop, ...args);
      },
    },
  ) as Record<string, (...args: unknown[]) => Promise<unknown>>;
}

/**
 * 初始化桥接层（App.tsx 模块作用域最早时机调用，幂等）：
 *  - Wails 环境：补齐 window.go.app.App 旧形态兼容代理（板块门面路由）；
 *  - HTTP 环境：创建 window.go.app.App 的 /api/rpc 代理。
 */
export function initBridge(): void {
  // 避免重复初始化
  if ((window as unknown as Record<string, unknown>).__bridge_initialized) return;
  (window as unknown as Record<string, unknown>).__bridge_initialized = true;

  // 显式 ?mock= 场景优先（评审 03-office-frontend.md 缺陷 8）：浏览器开发时
  // 用 URL 参数进入 mock 模式（approval/ask/compaction 等场景），不创建
  // RPC 代理——保持 window.go 为空，realApp() 返回 undefined，app 代理
  // 走 getMock()。此前 initBridge 在非 Wails 环境无条件创建 RPC 代理，
  // ?mock= 从未生效，审批/提问/压缩卡无法离线开发。
  const mockParam = new URLSearchParams(window.location.search).get("mock")?.trim().toLowerCase();
  if (mockParam && !isWailsNative()) {
    console.log(`[bridge] 浏览器 mock 模式（?mock=${mockParam}）`);
    return;
  }

  if (isWailsNative()) {
    ensureLegacyAppProxy();
    console.log("[bridge] Wails 原生环境，已就绪板块门面路由");
    return;
  }

  console.log("[bridge] 移动端 HTTP 模式，创建 RPC 代理");
  const w = window as unknown as { go?: { app?: Record<string, unknown> } };
  if (!w.go) w.go = {};
  if (!w.go.app) w.go.app = {};
  w.go.app.App = createAppProxy();
  // HTTP 模式也补齐板块门面代理（CoreB/OfficeB/MemoryB/CostB/ModelB/VoiceB/ChatB/NovelB/ImageB/CharlibB）：
  // wailsjsCompat 从 '../wailsjs/go/app/<门面>' 直调 window.go.app.<门面>.<方法>（如
  // useVoiceChat 的 App.VoiceStop），HTTP 桥接下这些门面不存在会抛
  // "Cannot read properties of undefined (reading 'VoiceStop')"。RPC 端点按方法名路由，
  // 与门面无关，故每门面挂同一 RPC 代理即可（方法名冲突时后注册者生效，方法与门面一一对应无冲突）。
  const FACADES = ["CoreB", "OfficeB", "MemoryB", "CostB", "ModelB", "VoiceB", "ChatB", "NovelB", "ImageB", "CharlibB", "SinB"] as const;
  for (const ns of FACADES) {
    if (!w.go.app[ns]) w.go.app[ns] = createAppProxy();
  }
}
