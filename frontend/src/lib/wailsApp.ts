import type { AppAPI } from "../types/wails";

/**
 * window.go.app.App 访问器：strict 收窄 + 运行时缺失即抛错。
 * 行为对齐旧直调（缺失时同样在调用点抛异常），仅错误信息更明确；
 * 浏览器 mock 模式经 gaea/lib/bridge.ts 的 app 代理，不走此路径。
 */
export function wailsApp(): AppAPI {
  const app = window.go?.app?.App;
  if (!app) throw new Error("Wails runtime unavailable: window.go.app.App");
  return app;
}
