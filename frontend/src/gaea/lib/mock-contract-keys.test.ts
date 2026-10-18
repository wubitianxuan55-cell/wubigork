// OpenCode Go/Zen Key 的 dev mock 契约测试（原 mock-contract-modelhub.test.ts
// 的 OpenCode 半边：Model Hub（Unsloth）引擎已退役删除，其 mock/断言随之移除）。
// 锁定：方法存在性 + 状态查询形状（configured/masked）+ Set→Get 内存态联动 +
// 空 Key 诚实拒绝。
// 调用路径与 mock-contract-herdsman 同款：bridge 单例 app 窄化视图（与
// window.go.app.App 兼容代理路由方法名一致，api/engines.ts App() 同名）。
import { describe, expect, it } from "vitest";
import { app } from "./bridge";

/** legacy 直调面窄化视图：OpenCode Key 方法族（mock 见 mock/model.ts）。 */
const keys = app as unknown as {
  SetOpencodeGoKey(apiKey: string): Promise<void>;
  GetOpencodeGoKeyStatus(): Promise<{ configured: boolean; masked: string }>;
  SetOpencodeZenKey(apiKey: string): Promise<void>;
  GetOpencodeZenKeyStatus(): Promise<{ configured: boolean; masked: string }>;
};

describe("mock 契约：OpenCode Key 方法族", () => {
  it("初始中性空态：未配置（configured=false, masked 空）", async () => {
    await expect(keys.GetOpencodeGoKeyStatus()).resolves.toEqual({ configured: false, masked: "" });
    await expect(keys.GetOpencodeZenKeyStatus()).resolves.toEqual({ configured: false, masked: "" });
  });

  it("Set→Get 内存态联动，脱敏口径对齐 maskKeyStatus（>8 位=前4+****+后4，≤8 位=****）", async () => {
    await keys.SetOpencodeGoKey("short");
    await expect(keys.GetOpencodeGoKeyStatus()).resolves.toEqual({ configured: true, masked: "****" });
    await keys.SetOpencodeGoKey("oc-abcd1234efgh5678");
    await expect(keys.GetOpencodeGoKeyStatus()).resolves.toEqual({
      configured: true,
      masked: "oc-a****5678",
    });
  });

  it("空 Key 诚实拒绝（rejected），不污染状态", async () => {
    await expect(keys.SetOpencodeZenKey("   ")).rejects.toThrow("API Key 不能为空");
    await expect(keys.GetOpencodeZenKeyStatus()).resolves.toEqual({ configured: false, masked: "" });
  });
});
