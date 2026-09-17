// image.ts — ImageBindings（AppBindings 分域接口之一，Go ImageB 门面）：
// 图像域 legacy 直调族转正（v4.102）+ 画室素材库/截图/OCR（无类型导入，全 Record）。

export interface ImageBindings {
  // ── 图像域 legacy 直调族转正（v4.102，api/image.ts 消费收口；从 LegacySurfaceNames 摘除）──
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
  // CU1 绘梦页首入预热（64×64 空跑预加载默认模型；后端武装位+一次闸+静默降级）
  WarmComfyUI(): Promise<{ started: boolean; reason?: string; model?: string }>;
  GetSystemStats(): Promise<Record<string, unknown>>;
  OpenImageSaveDir(): Promise<void>;
  OpenNovelImagesDir(): Promise<void>;
  // ImageHubAssets 图像域登记只读视图（按空间/来源过滤，T1 画室素材库；
  // 原 legacy wailsjsCompat 直调转正，浏览器 dev mock 可达；从 LegacySurfaceNames 摘除）。
  ImageHubAssets(space: string, sourceBoard: string, limit: number): Promise<Array<Record<string, unknown>>>;
  // ImageHubMonthlyUsage 当月画室消耗聚合（只读台账；刀 E）。
  ImageHubMonthlyUsage(space: string): Promise<StudioUsageView>;
  // ChapterArtList 项目章节配图清单（chapter-art.json 只读取，同批转正）。
  ChapterArtList(chapterNum: number): Promise<Array<Record<string, unknown>>>;
  // CaptureScreen 捕获整个屏幕（返回 PNG data URL）；RecognizeImage 用本地
  // 视觉模型识别图片内容，返回文本描述。
  CaptureScreen(): Promise<string>;
  RecognizeImage(imagePath: string, prompt: string): Promise<string>;
  // OCRText 用本地 OvisOCR2 常驻服务提取图片中的文字（办公「提取文字」入口）。
  OCRText(imagePath: string): Promise<string>;
  // StartLocalTTSService 启动本地 TTS 服务（模型中心「启动」按钮；Go ImageB
  // 门面同名幂等启动，返回 {engine, ready, starting}；wailsjsCompat 直调转正）。
  StartLocalTTSService(engineID: string): Promise<Record<string, unknown>>;
  // SetImageBackend 设置图像生成后端（backend/comfyUIURL/imageModel/imageSaveDir
  // 四参；Go ImageB 同名，批次二直调转正；绘梦域配置写入口）。
  SetImageBackend(backend: string, comfyUIURL: string, imageModel: string, imageSaveDir: string): Promise<void>;
  // ── 批次三a legacy 直调转正（Go ImageB 门面 bindings_image.go:30/32，同名前缀）──
  // GetTTSSpeakers 查询指定 TTS 模型的可用音色列表；GetVoicePipelineConfig 读取
  // 语音管线配置快照（采样率/编码等）——模型中心/语音设置两空间共用。
  GetTTSSpeakers(model: string): Promise<Array<string>>;
  GetVoicePipelineConfig(): Promise<Record<string, unknown>>;
  // ── 批次四 bridge 双轨退役终局（Go ImageB 门面 bindings_image.go:37/38/40，
  // 同名前缀）──语音模型设置写口：SetActiveASRModel/SetActiveTTSModel 设置
  // 语音管道激活的识别/合成模型；SetChatVoiceModel 设置功能绑定「聊天语音」
  // （空串=清除绑定回退全局 TTS）。模型中心 useVoiceState 消费，
  // 同 GetTTSSpeakers 模型配置读写面归 shared。
  SetActiveASRModel(engineID: string, modelID: string): Promise<void>;
  SetActiveTTSModel(engineID: string, modelID: string): Promise<void>;
  SetChatVoiceModel(engineID: string, modelID: string): Promise<void>;
}

// ── 画室消耗视图（刀 E；Go ImageHubMonthlyUsageReport 直连）──
export interface StudioUsageModelRow {
  model: string;
  backend?: string;
  count: number;
  unitCost?: string;
  estCost?: string;
}
export interface StudioUsageView {
  month: string;
  space: string;
  total: number;
  freeCount: number;
  unpriced: number;
  estimated: string;
  byModel: StudioUsageModelRow[];
}
