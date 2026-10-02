// proxy.mockonly.test.ts — FE3-04 反向证据锁：mock-only 绑定（drift.ts
// MOCK_ONLY_NAMES 单源）在真机路径 fail-fast 显式拒绝，而不再 get→undefined→
// TypeError→invoke 归一成「失败…请重试」的假可重试错误；三名零调用者死成员
// （SetSubagentTemperature/SetEffort/SetSubagentModel）摘除后 facets/mock 运行期
// 不再出现（编译期反证 = tsc --noEmit 全绿，见审计报告）；dev mock 路径 Compact
// 行为不变（resolve）。
import { afterEach, describe, expect, it } from "vitest";
import { app, realApp, waitMockReady, BridgeError } from "./proxy";
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

describe("FE3-04 mock-only 绑定：真机 fail-fast / 死成员摘除 / mock 路径不变", () => {
  it("真机路径调 Compact → BridgeError(MockOnlyBinding) 显式拒绝，文案含 mock-only 与「不可用」", async () => {
    installRealShell();
    expect(realApp()).toBeDefined(); // 真机分支命中（window.go.app 存在）
    // realApp 代理直取：mock-only 名是拒绝函数而非 undefined（旧行为的 TypeError 根因）
    expect(typeof realApp()!.Compact).toBe("function");

    let caught: unknown;
    try {
      await app.Compact();
    } catch (e) {
      caught = e;
    }
    expect(caught).toBeInstanceOf(BridgeError);
    const err = caught as BridgeError;
    expect(err.code).toBe("MockOnlyBinding");
    expect(err.message).toContain("mock-only");
    expect(err.message).toContain("不可用");
    expect(err.message).toContain("Compact");
  });

  it("真机路径普通绑定路由不受 fail-fast 注入影响（同壳 Version 正常返回）", async () => {
    installRealShell();
    await expect(app.Version()).resolves.toBe("v-fake");
  });

  it("三名零调用者摘除后：GAEA_METHOD_FACETS 与 dev mock 均无此三名；MOCK_ONLY 收缩为 [Compact]", () => {
    const removed = ["SetSubagentTemperature", "SetEffort", "SetSubagentModel"];
    for (const name of removed) {
      expect(GAEA_METHOD_FACETS).not.toHaveProperty(name);
      expect(makeMockApp()).not.toHaveProperty(name);
    }
    // Compact 仍在类型面/facets/mock（有真实调用点走运行时拒绝，不做行为删除），
    // 且 MOCK_ONLY_NAMES 单源收缩后只钉它一个。
    expect(GAEA_METHOD_FACETS).toHaveProperty("Compact");
    expect([...MOCK_ONLY_NAMES]).toEqual(["Compact"]);
  });

  it("dev mock 路径 Compact 行为不变：resolve 而非拒绝", async () => {
    delete (window as unknown as GoWindow).go; // 无真壳 → mock 分支
    await waitMockReady();
    await expect(app.Compact()).resolves.toBeUndefined();
  });
});
