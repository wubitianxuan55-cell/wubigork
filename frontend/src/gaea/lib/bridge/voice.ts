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
  // TTSSpeakBase64WithParams 带 TTS 参数（tts.TTSParams，前端用 Record 透传）合成；
  // TTSVoicePreviewBase64 角色档案试听段（text/voiceID/voiceDescription 显式覆盖，
  // 不读当前人格预设；仅走能表达音色预设的引擎，全链不可用诚实报错）。
  TTSSpeakBase64(text: string): Promise<Record<string, unknown>>;
  TTSSpeakBase64WithParams(text: string, params: Record<string, unknown>): Promise<Record<string, unknown>>;
  TTSVoicePreviewBase64(text: string, voiceID: string, voiceDescription: string): Promise<Record<string, unknown>>;
  // ── 批次四 bridge 双轨退役终局（Go VoiceB 门面 bindings_voice.go:27）──
  // VoiceHealth 语音服务健康状态（返回 {asrReady,ttsReady,state,error}；
  // 语音设置面板「检测」按钮 VoiceHealth 直调转正）。
  VoiceHealth(): Promise<Record<string, unknown>>;
  // TTSLineAudioFor 角色台词语音文件（内容寻址缓存优先）：命中回读文件
  // {audio, mimeType, cached:true}（试听秒开）；未命中合成一次落盘回
  // {audio, mimeType, cached:false}。chJSON=角色 JSON（新角色无 id 拒绝）。
  TTSLineAudioFor(chJSON: string): Promise<Record<string, unknown>>;
  // ── Voice V2 档（M4 产品接入刀；voice.backend=v2 消费）──
  // VoiceV2Status 四依赖只读探测 {sidecar,llm,breeze,gateway,ready}。
  VoiceV2Status(): Promise<Record<string, unknown>>;
  // VoiceV2Ensure 幂等拉起 breeze→4B→sidecar 链（异步，voice-v2-status 事件上报）。
  VoiceV2Ensure(): Promise<Record<string, unknown>>;
  // VoiceV2Stop 按端口击杀 sidecar 与 4B（breeze 共享引擎不动）。
  VoiceV2Stop(): Promise<Record<string, unknown>>;
  // VoiceV2AbortTurn 生成中关闭语音台的精确回滚（P0a-A5f）：按 itemID 领走
  // 并回滚生成前预推进；{aborted, reason?, itemID?, personalityID?}。
  VoiceV2AbortTurn(itemID: string): Promise<Record<string, unknown>>;
  // VoiceV2PreviewIntent 最终转写 → 意图 dry-run（P1.9/P2）：命中且白名单内
  // 登记台账，返回 {handled, executable, reply, action, previewId?, intentHash?, ttlMs?}。
  VoiceV2PreviewIntent(userText: string): Promise<Record<string, unknown>>;
  // VoiceV2ExecuteAction 确认执行（P2）：幂等/TTL/hash/白名单校验，按台账
  // 直派执行层——确认后最多执行被预览动作一次。
  VoiceV2ExecuteAction(previewID: string, intentHash: string): Promise<Record<string, unknown>>;
  // VoiceV2CommitTurn 以 topic mode 为权威人格，单事务写回 V2 聊天并触发
  // 轻语工作记忆/状态/FactStore 异步收尾。responseID 持久判重；itemID
  // 精确关联生成前的 persona 快照，重复话术也不会串轮。
  VoiceV2CommitTurn(
    topicID: string,
    userText: string,
    assistantText: string,
    responseID: string,
    itemID: string,
  ): Promise<Record<string, unknown>>;
}
