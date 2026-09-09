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
}
