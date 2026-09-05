// mock/imagegen.ts — 图像域 legacy 直调族转正后的 dev mock（v4.102）。
// 口径：查询类中性空态（后端信息/ComfyUI/系统统计=未配置不编造）、动作类诚实
// 失败（浏览器无生成后端/无文件管理器）；GetCharacters 给最小样例（参考槽走查），
// SetCharacterPortrait 内存 no-op。真实绑定见 internal/app/bindings_image.go。
import type { AppBindings } from "../bridge";

type ImagegenMethods = Pick<
  AppBindings,
  | "GenerateFreeImage" | "CancelImageGeneration" | "GenerateMedia" | "GenerateDiagram"
  | "GetImageBackendInfo" | "GetPortraitConfig" | "SetPortraitConfig"
  | "GetComfyUIStatus" | "GetComfyUILoras" | "GetComfyUITaskProgress"
  | "StartComfyUI" | "StopComfyUI" | "GetSystemStats"
  | "OpenImageSaveDir" | "OpenNovelImagesDir" | "GetCharacters" | "SetCharacterPortrait"
>;

export function buildImagegenTools(): ImagegenMethods {
  return {
    async GenerateFreeImage(_prompt: string, _negative: string, _size: string, _initImage: string, _model: string, _seed: number, _count: number, _lora: string) {
      throw new Error("浏览器 dev mock 无图像生成后端，生成不可用（mock）");
    },
    async CancelImageGeneration() {
      return false;
    },
    async GenerateMedia(_params: string) {
      throw new Error("浏览器 dev mock 无媒体生成后端，生成不可用（mock）");
    },
    async GenerateDiagram(_prompt: string) {
      throw new Error("浏览器 dev mock 无图示生成后端，生成不可用（mock）");
    },
    async GetImageBackendInfo() {
      // 未配置后端：消费方按空串降级（绘梦引擎 pill 显示未设置）。
      return {};
    },
    async GetPortraitConfig() {
      return {};
    },
    async SetPortraitConfig(_backend: string, _model: string) {
      // 浏览器内存 no-op：无持久化（诚实语义，配置面板保存后不假装已落盘）。
    },
    async GetComfyUIStatus() {
      return {};
    },
    async GetComfyUILoras() {
      return [];
    },
    async GetComfyUITaskProgress() {
      return {};
    },
    async StartComfyUI(): Promise<void> {
      throw new Error("浏览器 dev mock 无 ComfyUI 进程，启停不可用（mock）");
    },
    async StopComfyUI(): Promise<void> {
      throw new Error("浏览器 dev mock 无 ComfyUI 进程，启停不可用（mock）");
    },
    async GetSystemStats() {
      return {};
    },
    async OpenImageSaveDir(): Promise<void> {
      throw new Error("浏览器 dev mock 无文件管理器，打开目录不可用（mock）");
    },
    async OpenNovelImagesDir(): Promise<void> {
      throw new Error("浏览器 dev mock 无文件管理器，打开目录不可用（mock）");
    },
    async GetCharacters() {
      // 最小样例：参考槽/角色带入走查用（与 mock/weixin CharacterList 同人物宇宙）。
      return {
        characters: [
          { id: "linwan", name: "林晚" },
          { id: "gucheng", name: "顾城" },
        ],
      };
    },
    async SetCharacterPortrait(_characterId: string, _portraitPath: string): Promise<void> {
      // 内存 no-op（同 SetPortraitConfig）。
    },
  };
}
