// proxy.mockonly.test.ts — FE3-04 反向证据锁（批 52 B6① 后态）：mock-only
// 绑定清单（drift.ts MOCK_ONLY_NAMES 单源）摘除 Compact 后**暂空**——本文件钉
// 「摘除后状态」防回潮：四名死成员（SetSubagentTemperature/SetEffort/
// SetSubagentModel/Compact）在 facets 与 dev mock 均不得复现；MOCK_ONLY_NAMES
// 非空时（未来新登记 mock-only 名）本文件须随登记理由一起更新。
import { afterEach, describe, expect, it } from "vitest";
import { app, realApp } from "./proxy";
import { MOCK_ONLY_NAMES } from "./drift";
import { GAEA_METHOD_FACETS } from "../spaceBindings";
import { makeMockApp } from "../mock";

type GoWindow = { go?: unknown };

// 与 Go 实况同构的最小真壳假件：window.go.app 有门面对象，但其中没有任何
// mock-only 名（bindingSignatures 744 名无 Compact/SetEffort/…）。
function installRealShell() {
  (window as unknown as GoWindow).go = {
    app: {
      CoreB: { GaeaVersion: async () => "v-fake" },
    },
  };
}

afterEach(() => {
  delete (window as unknown as GoWindow).go;
});

describe("FE3-04 mock-only 机制：死成员摘除后态 / 普通绑定路由不受影响", () => {
  it("真机路径普通绑定路由不受 fail-fast 注入影响（同壳 Version 正常返回）", async () => {
    installRealShell();
    expect(realApp()).toBeDefined(); // 真机分支命中（window.go.app 存在）
    await expect(app.Version()).resolves.toBe("v-fake");
  });

  it("MOCK_ONLY_NAMES 暂空（B6① 摘除 Compact 后）；四名死成员在 facets 与 dev mock 均不得复现", () => {
    expect([...MOCK_ONLY_NAMES]).toEqual([]);
    const removed = [
      "SetSubagentTemperature", "SetEffort", "SetSubagentModel",
      "Compact", // B6①：mock-only 零调用者+自动压缩已在，绑定面整体摘除
    ];
    for (const name of removed) {
      expect(GAEA_METHOD_FACETS).not.toHaveProperty(name);
      expect(makeMockApp()).not.toHaveProperty(name);
    }
  });
});
