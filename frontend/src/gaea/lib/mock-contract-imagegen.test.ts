// 图像域 legacy 直调族转正（v4.102）后的 dev mock 契约测试。
// 锁定：17 名存在性 + 查询类中性空态形状 + 动作类诚实失败（生成/ComfyUI 启停/
// 打开目录 rejected）+ GetCharacters 样例形状（消费方 getCharacters 读
// cf.characters，api/image.ts）。调用路径同 mock-contract-herdsman：bridge 单例
// app 窄化视图（与 window.go.app.App 兼容代理路由方法名一致）。
import { describe, expect, it } from "vitest";
import { app } from "./bridge";

const ig = app as unknown as {
  GenerateFreeImage(prompt: string, negative: string, size: string, initImage: string, model: string, seed: number, count: number, lora: string): Promise<Record<string, unknown>>;
  CancelImageGeneration(): Promise<boolean>;
  GenerateMedia(params: string): Promise<Record<string, unknown>>;
  GenerateDiagram(prompt: string): Promise<Record<string, unknown>>;
  GetImageBackendInfo(): Promise<Record<string, string>>;
  GetPortraitConfig(): Promise<Record<string, string>>;
  SetPortraitConfig(backend: string, model: string): Promise<void>;
  GetComfyUIStatus(): Promise<Record<string, unknown>>;
  GetComfyUILoras(): Promise<Array<string>>;
  GetComfyUITaskProgress(): Promise<Record<string, unknown>>;
  StartComfyUI(): Promise<void>;
  StopComfyUI(): Promise<void>;
  GetSystemStats(): Promise<Record<string, unknown>>;
  OpenImageSaveDir(): Promise<void>;
  OpenNovelImagesDir(): Promise<void>;
  GetCharacters(): Promise<{ characters?: Array<{ id: string; name: string }> }>;
  SetCharacterPortrait(characterId: string, portraitPath: string): Promise<void>;
};

describe("mock 契约：图像域直调族（转正后 mock 可达）", () => {
  it("查询类中性空态：后端信息/ComfyUI/统计=未配置不编造", async () => {
    await expect(ig.GetImageBackendInfo()).resolves.toEqual({});
    await expect(ig.GetComfyUIStatus()).resolves.toEqual({});
    await expect(ig.GetComfyUILoras()).resolves.toEqual([]);
    await expect(ig.GetSystemStats()).resolves.toEqual({});
    await expect(ig.GetPortraitConfig()).resolves.toEqual({});
    await expect(ig.CancelImageGeneration()).resolves.toBe(false);
  });

  it("动作类诚实失败：生成/ComfyUI 启停/打开目录 rejected", async () => {
    await expect(ig.GenerateFreeImage("p", "", "1:1", "", "m", 0, 1, "")).rejects.toThrow(/mock/);
    await expect(ig.GenerateMedia("{}")).rejects.toThrow(/mock/);
    await expect(ig.GenerateDiagram("登录时序")).rejects.toThrow(/mock/);
    await expect(ig.StartComfyUI()).rejects.toThrow(/mock/);
    await expect(ig.OpenImageSaveDir()).rejects.toThrow(/mock/);
    await expect(ig.OpenNovelImagesDir()).rejects.toThrow(/mock/);
  });

  it("GetCharacters 样例形状（消费方读 cf.characters）与 Set 内存 no-op", async () => {
    const cf = await ig.GetCharacters();
    expect(cf.characters?.length).toBeGreaterThan(0);
    expect(cf.characters?.[0]).toHaveProperty("id");
    expect(cf.characters?.[0]).toHaveProperty("name");
    await expect(ig.SetCharacterPortrait("linwan", ".gaea/uploads/x.png")).resolves.toBeUndefined();
    await expect(ig.SetPortraitConfig("herdsman", "z-image-turbo")).resolves.toBeUndefined();
  });
});
