import { afterEach, describe, expect, it } from "vitest";
import { loadBrowserAutoOpen, saveBrowserAutoOpen, shouldAutoOpenBrowser } from "./browserPrefs";

// browserPrefs 测试（v4.28 A2）：自动弹出偏好的持久化与坏值兜底。
// 模式对齐 subagentPrefs.test（同款 localStorage 直读 + 宽容净化）。

const KEY = "gaea.browserAutoOpen";

describe("browserPrefs 自动弹出偏好（gaea.browserAutoOpen，默认开）", () => {
  afterEach(() => {
    try { localStorage.removeItem(KEY); } catch { /* ignore */ }
  });

  it("未设置时默认开", () => {
    expect(loadBrowserAutoOpen()).toBe(true);
  });

  it("保存后读取往返（落盘 1/0，可读可手改）", () => {
    saveBrowserAutoOpen(false);
    expect(localStorage.getItem(KEY)).toBe("0");
    expect(loadBrowserAutoOpen()).toBe(false);
    saveBrowserAutoOpen(true);
    expect(localStorage.getItem(KEY)).toBe("1");
    expect(loadBrowserAutoOpen()).toBe(true);
  });

  it("坏值兜底：只有显式 0/false 视为关，垃圾值一律回落开", () => {
    for (const bad of ["garbage", "1", "true", "yes", ""]) {
      try { localStorage.setItem(KEY, bad); } catch { /* ignore */ }
      expect(loadBrowserAutoOpen(), `值 ${JSON.stringify(bad)} 应兜底为开`).toBe(true);
    }
    try { localStorage.setItem(KEY, "0"); } catch { /* ignore */ }
    expect(loadBrowserAutoOpen()).toBe(false);
    try { localStorage.setItem(KEY, "false"); } catch { /* ignore */ }
    expect(loadBrowserAutoOpen()).toBe(false);
  });

  it("shouldAutoOpenBrowser 是 App 接线入口：与偏好读数一致", () => {
    expect(shouldAutoOpenBrowser()).toBe(true);
    saveBrowserAutoOpen(false);
    expect(shouldAutoOpenBrowser()).toBe(false);
    // 坏值同样兜底（App 侧不做二次净化，入口保证语义）
    try { localStorage.setItem(KEY, "{{{"); } catch { /* ignore */ }
    expect(shouldAutoOpenBrowser()).toBe(true);
  });
});
