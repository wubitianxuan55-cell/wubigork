// voice.ts — VoiceBindings（AppBindings 分域接口之一，Go VoiceB 门面）：
// TTS 参数通道（情绪标签 → 结构化参数）。
import type { TTSParams } from "../types";

export interface VoiceBindings {
  // TTSVoiceParams 返回情绪标签对应的结构化 TTS 参数（预览/调试）。
  TTSVoiceParams(emotion: string): Promise<TTSParams>;
  // VoiceApplySettings 部分更新语音设置（patch 增量合并；wailsjsCompat 直调转正，
  // useChatVoice / 语音设置面板 / 模型中心 BindSection / api/settings.ts 消费）。
  VoiceApplySettings(patch: Record<string, unknown>): Promise<void>;
  // ── Whisper 轻语读取/写入族（Go VoiceB 门面，同批转正；角色记忆面板
  // CharacterMemoryModal 与轻语记忆 WhisperMemoryModal 消费）──
  // WhisperGetState 返回人格状态快照（情绪/关系/人格维度/欲望栈/总轮次）。
  WhisperGetState(personalityID: string): Promise<Record<string, unknown>>;
  // WhisperGetFacts 返回人格已沉淀事实列表。
  WhisperGetFacts(personalityID: string): Promise<Array<Record<string, unknown>>>;
  // WhisperGetTraces 返回人格最近对话追踪（TurnTrace 列表）。
  WhisperGetTraces(personalityID: string): Promise<Array<Record<string, unknown>>>;
  // WhisperDeleteFact 按 factID 删除一条事实。
  WhisperDeleteFact(personalityID: string, factID: string): Promise<void>;
  // WhisperUpdateFact 对既有事实做字段级部分更新（updates 只写给定键）。
  WhisperUpdateFact(personalityID: string, factID: string, updates: Record<string, unknown>): Promise<void>;
  // ── 批次二 wailsjsCompat 双轨退役转正（Go VoiceB 门面，同名前缀）──
  // VoiceGetSettings 读取语音设置快照（与 VoiceApplySettings patch 写配对）；
  // VoiceChatText 把文本送入语音对话管线（Voice 域发消息入口，no-op 语义
  // 由调用方结合事件流理解）。
  VoiceGetSettings(): Promise<Record<string, unknown>>;
  VoiceChatText(text: string): Promise<void>;
  // ── 批次三a legacy 直调转正（Go VoiceB 门面，同名前缀无需映射）──
  // 语音运行时族：VoiceStart 带 browserASR 启停语音会话；VoiceStop 停止；
  // VoicePlaybackDone 播放完成回执；VoiceCancelTTS 取消待合成队列；
  // VoicePushAudio 推送音频块（Go []byte，前端传 base64 字符串）；
  // VoiceSetPTTActive 设置按键说话（PTT）激活态。
  VoiceStart(browserASR: boolean): Promise<void>;
  VoiceStop(): Promise<void>;
  VoicePlaybackDone(): Promise<void>;
  VoiceCancelTTS(): Promise<void>;
  VoicePushAudio(chunk: string): Promise<void>;
  VoiceSetPTTActive(active: boolean): Promise<void>;
  // WhisperClearSession 清空指定人格的轻语会话（重开一段干净对话）。
  WhisperClearSession(personalityID: string): Promise<void>;
  // TTSSpeakBase64 合成 TTS 音频并返回 base64（text → {base64, mimeType}）；
  // TTSSpeakBase64WithParams 带 TTS 参数（tts.TTSParams，前端用 Record 透传）合成。
  TTSSpeakBase64(text: string): Promise<Record<string, unknown>>;
  TTSSpeakBase64WithParams(text: string, params: Record<string, unknown>): Promise<Record<string, unknown>>;
  // ── 批次四 bridge 双轨退役终局（Go VoiceB 门面 bindings_voice.go:27）──
  // VoiceHealth 语音服务健康状态（返回 {asrReady,ttsReady,state,error}；
  // 语音设置面板「检测」按钮 VoiceHealth 直调转正）。
  VoiceHealth(): Promise<Record<string, unknown>>;
}
