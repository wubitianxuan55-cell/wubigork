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
}
