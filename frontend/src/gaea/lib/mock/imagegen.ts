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
  // v4.171 批次一 legacy 直调转正（ImageB 门面，与图像域同门面就近）。
  | "StartLocalTTSService"
  // 批次二 legacy 直调转正（ImageB 门面）：绘梦后端配置。
  | "SetImageBackend"
  // 批次三a legacy 直调转正（ImageB 门面 bindings_image.go:30/32）：TTS 音色/
  // 语音管线配置读取（模型中心/语音设置消费，同 StartLocalTTSService 就近）。
  | "GetTTSSpeakers" | "GetVoicePipelineConfig"
  // 批次四 bridge 双轨退役终局（ImageB 门面 bindings_image.go:37/38/40）：
  // 语音模型设置写口（激活识别/合成模型 + 功能绑定聊天语音），同批转正。
  | "SetActiveASRModel" | "SetActiveTTSModel" | "SetChatVoiceModel"
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
    async StartLocalTTSService(engineID: string) {
      // 契约对齐 Go ensureLocalTTSService（internal/app/tts_service.go）：幂等
      // 启动返回 {engine, ready:false, starting:true}。浏览器开发无真实本地 TTS
      // 服务，按「正在启动」形状返回——useEngineState 的 res?.ready 分支走
      // 「正在启动」提示，不谎报已就绪。
      return { engine: engineID, ready: false, starting: true };
    },
    // ── 批次二 legacy 直调转正（Go ImageB，同名前缀）────────────────
    async SetImageBackend(_backend: string, _comfyUIURL: string, _imageModel: string, _imageSaveDir: string) {
      // mock: 浏览器内存 no-op——无后端可配置（与 SetPortraitConfig 同口径）。
    },
    // ── 批次三a legacy 直调转正（Go ImageB，同名前缀）────────────────
    async GetTTSSpeakers(_model: string) {
      // 浏览器开发无本地 TTS 引擎：空音色列表（模型中心音色下拉空态，不编造）。
      return [];
    },
    async GetVoicePipelineConfig() {
      // 最小样例：无语音管线实例 → 仅 samples 键（消费方按空态兜底渲染）。
      return { samples: [] };
    },
    // ── 批次四 bridge 双轨退役终局（Go ImageB，同名前缀）────────────────
    async SetActiveASRModel(_engineID: string, _modelID: string) {
      // mock: 浏览器内存 no-op——无语音管道实例可设置（与 SetPortraitConfig 同口径）。
    },
    async SetActiveTTSModel(_engineID: string, _modelID: string) {
      // mock: 同上 no-op（合成模型设置空转）。
    },
    async SetChatVoiceModel(_engineID: string, _modelID: string) {
      // mock: 同上 no-op（功能绑定「聊天语音」设置/清除均空转，不编造落盘）。
    },
  };
}
