import { afterEach, describe, expect, it } from "vitest";
import { loadDeliverableAutoOpen, saveDeliverableAutoOpen, shouldAutoOpenDeliverables } from "./deliverablePrefs";

// deliverablePrefs 测试（v4.32 线B）：产物自动弹出偏好的持久化与坏值兜底。
// 用例形状对齐 browserPrefs.test（同款 localStorage 直读 + 宽容净化），
// 差异只在默认值：产物自动弹出默认关（激进版 opt-in），只有显式 "1"/"true" 视为开。

const KEY = "gaea.deliverableAutoOpen";

describe("deliverablePrefs 自动弹出偏好（gaea.deliverableAutoOpen，默认关）", () => {
  afterEach(() => {
    try { localStorage.removeItem(KEY); } catch { /* ignore */ }
  });

  it("未设置时默认关", () => {
    expect(loadDeliverableAutoOpen()).toBe(false);
  });

  it("保存后读取往返（落盘 1/0，可读可手改）", () => {
    saveDeliverableAutoOpen(true);
    expect(localStorage.getItem(KEY)).toBe("1");
    expect(loadDeliverableAutoOpen()).toBe(true);
    saveDeliverableAutoOpen(false);
    expect(localStorage.getItem(KEY)).toBe("0");
    expect(loadDeliverableAutoOpen()).toBe(false);
  });

  it("坏值兜底：只有显式 1/true 视为开，垃圾值一律回落关", () => {
    for (const bad of ["garbage", "0", "false", "yes", ""]) {
      try { localStorage.setItem(KEY, bad); } catch { /* ignore */ }
      expect(loadDeliverableAutoOpen(), `值 ${JSON.stringify(bad)} 应兜底为关`).toBe(false);
    }
    try { localStorage.setItem(KEY, "1"); } catch { /* ignore */ }
    expect(loadDeliverableAutoOpen()).toBe(true);
    try { localStorage.setItem(KEY, "true"); } catch { /* ignore */ }
    expect(loadDeliverableAutoOpen()).toBe(true);
  });

  it("shouldAutoOpenDeliverables 是 App 接线入口：与偏好读数一致", () => {
    expect(shouldAutoOpenDeliverables()).toBe(false);
    saveDeliverableAutoOpen(true);
    expect(shouldAutoOpenDeliverables()).toBe(true);
    // 坏值同样兜底（App 侧不做二次净化，入口保证语义）
    try { localStorage.setItem(KEY, "{{{"); } catch { /* ignore */ }
    expect(shouldAutoOpenDeliverables()).toBe(false);
  });
});
