// voice.ts — VoiceBindings（AppBindings 分域接口之一，Go VoiceB 门面）：
// TTS 参数通道（情绪标签 → 结构化参数）。
import type { TTSParams } from "../types";

export interface VoiceBindings {
  // TTSVoiceParams 返回情绪标签对应的结构化 TTS 参数（预览/调试）。
  TTSVoiceParams(emotion: string): Promise<TTSParams>;
}
