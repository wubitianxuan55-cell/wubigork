// Model Hub（Unsloth）与 OpenCode Key 的 dev mock 契约测试（6cd891df 落库后的
// mock 补齐，v4.58「引擎管理 is-not-a-function」同族欠账销账）。
// 锁定：方法存在性 + 状态查询形状（configured/masked）+ Set→Get 内存态联动 +
// 空 Key 诚实拒绝 + StartModelHubModel 浏览器 mock 诚实失败。
// 调用路径与 mock-contract-herdsman 同款：bridge 单例 app 窄化视图（与
// window.go.app.App 兼容代理路由方法名一致，api/engines.ts App() 同名）。
import { describe, expect, it } from "vitest";
import { app } from "./bridge";

/** legacy 直调面窄化视图：Model Hub + OpenCode Key 方法族（mock 见 mock/model.ts）。 */
const keys = app as unknown as {
  SetModelHubKey(apiKey: string): Promise<void>;
  GetModelHubKeyStatus(): Promise<{ configured: boolean; masked: string }>;
  StartModelHubModel(modelID: string): Promise<void>;
  SetOpencodeGoKey(apiKey: string): Promise<void>;
  GetOpencodeGoKeyStatus(): Promise<{ configured: boolean; masked: string }>;
  SetOpencodeZenKey(apiKey: string): Promise<void>;
  GetOpencodeZenKeyStatus(): Promise<{ configured: boolean; masked: string }>;
};

describe("mock 契约：Model Hub / OpenCode Key 方法族", () => {
  it("初始中性空态：未配置（configured=false, masked 空）", async () => {
    await expect(keys.GetModelHubKeyStatus()).resolves.toEqual({ configured: false, masked: "" });
    await expect(keys.GetOpencodeGoKeyStatus()).resolves.toEqual({ configured: false, masked: "" });
    await expect(keys.GetOpencodeZenKeyStatus()).resolves.toEqual({ configured: false, masked: "" });
  });

  it("Set→Get 内存态联动，脱敏口径对齐 maskKeyStatus（>8 位=前4+****+后4）", async () => {
    await keys.SetModelHubKey("sk-unsloth-abcd1234efgh5678");
    await expect(keys.GetModelHubKeyStatus()).resolves.toEqual({
      configured: true,
      masked: "sk-u****5678",
    });
    await keys.SetOpencodeGoKey("short");
    await expect(keys.GetOpencodeGoKeyStatus()).resolves.toEqual({ configured: true, masked: "****" });
  });

  it("空 Key 诚实拒绝（rejected），不污染状态", async () => {
    await expect(keys.SetOpencodeZenKey("   ")).rejects.toThrow("API Key 不能为空");
    await expect(keys.GetOpencodeZenKeyStatus()).resolves.toEqual({ configured: false, masked: "" });
  });

  it("StartModelHubModel 浏览器 mock 诚实失败（无本地 Unsloth Studio）", async () => {
    await expect(keys.StartModelHubModel("ollama-manifest:qwen3")).rejects.toThrow(/mock/);
  });
});
