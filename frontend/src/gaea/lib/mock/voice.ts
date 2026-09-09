// mock/voice.ts — 语音（TTS/ASR/轻语）域 dev mock（v4.171 批次一 legacy 直调转正）。
// Go 侧对应 VoiceB 门面（internal/app/bindings_voice.go，VoiceApplySettings /
// WhisperGetState/GetFacts/GetTraces/DeleteFact/UpdateFact 全部同名）。
// 口径：查询类中性空态（无轻语人格状态库/事实/追踪）、动作类 no-op（浏览器
// 开发无语音服务实例可应用、无事实库可删改）。消费方（CharacterMemoryModal /
// WhisperMemoryModal / 语音设置面板）对空值均有兜底渲染，不编造数据。
import type { AppBindings } from "../bridge";

type VoiceMethods = Pick<
  AppBindings,
  | "VoiceApplySettings"
  | "WhisperGetState" | "WhisperGetFacts" | "WhisperGetTraces"
  | "WhisperDeleteFact" | "WhisperUpdateFact"
  // 批次二 legacy 直调转正（VoiceB 门面）：设置读 + 语音对话文本。
  | "VoiceGetSettings" | "VoiceChatText"
  // 批次三a legacy 直调转正（VoiceB 门面）：语音运行时族 + 轻语清会话 + TTS base64 合成。
  | "VoiceStart" | "VoiceStop" | "VoicePlaybackDone" | "VoiceCancelTTS"
  | "VoicePushAudio" | "VoiceSetPTTActive" | "WhisperClearSession"
  | "TTSSpeakBase64" | "TTSSpeakBase64WithParams"
  // 批次四 bridge 双轨退役终局（VoiceB 门面 bindings_voice.go:27）：语音服务
  // 健康状态（语音设置面板「检测」按钮消费，同 Voice 运行时族就近）。
  | "VoiceHealth"
>;

export function buildVoice(): VoiceMethods {
  return {
    async VoiceApplySettings(_patch: Record<string, unknown>) {
      // mock: no-op——浏览器开发无语音服务实例可应用（真实实现部分更新语音配置，
      // 校验失败报错；空补丁语义 = 不改变任何设置）。
    },
    async WhisperGetState(_personalityID: string) {
      // 无轻语人格状态：CharacterMemoryModal 对空 state 渲染默认值
      //（relationship/emotion/personality 均走 ?? 兜底），不编造维度读数。
      return {};
    },
    async WhisperGetFacts(_personalityID: string) {
      return [];
    },
    async WhisperGetTraces(_personalityID: string) {
      return [];
    },
    async WhisperDeleteFact(_personalityID: string, _factID: string) {
      // mock: no-op——无真实事实库可删（消费方 onFactsChange 过滤本地列表）。
    },
    async WhisperUpdateFact(_personalityID: string, _factID: string, _updates: Record<string, unknown>) {
      // mock: no-op——同上，无事实可更新。
    },
    // ── 批次二 legacy 直调转正（Go VoiceB，同名前缀）────────────────
    async VoiceGetSettings() {
      // 未配置语音设置：空对象（消费方 ?? 兜底，不编造引擎/参数）。
      return {};
    },
    async VoiceChatText(_text: string) {
      // mock: no-op——浏览器开发无语音服务实例（真实实现把文本送入语音对话管线）。
    },
    // ── 批次三a legacy 直调转正（Go VoiceB，同名前缀）────────────────
    async VoiceStart(_browserASR: boolean) {
      // mock: no-op——浏览器开发无语音会话实例可启动（真实实现按 browserASR 启停）。
    },
    async VoiceStop() {
      // mock: no-op——同 VoiceStart，无会话可停。
    },
    async VoicePlaybackDone() {
      // mock: no-op——无播放中的 TTS 音频，回执空转。
    },
    async VoiceCancelTTS() {
      // mock: no-op——无待合成队列可取消。
    },
    async VoicePushAudio(_chunk: string) {
      // mock: no-op——无语音服务接收音频块（真实实现把 base64 解码 []byte 推送）。
    },
    async VoiceSetPTTActive(_active: boolean) {
      // mock: no-op——无 PTT 会话态可设置。
    },
    async WhisperClearSession(_personalityID: string) {
      // mock: no-op——无真实轻语会话可清空（消费方按成功语义继续，不编造记忆）。
    },
    async TTSSpeakBase64(_text: string) {
      // 无本地 TTS 服务：返回空合成结果（消费方对空 base64 兜底，不编造音频）。
      return { base64: "", mimeType: "" };
    },
    async TTSSpeakBase64WithParams(_text: string, _params: Record<string, unknown>) {
      // 同上：参数透传但无服务可合成 → 空 base64。
      return { base64: "", mimeType: "" };
    },
    // ── 批次四 bridge 双轨退役终局（Go VoiceB，同名前缀）────────────────
    async VoiceHealth() {
      // 浏览器开发无语音服务实例：按「未就绪/空闲」形状返回，面板图标不谎报就绪。
      return { asrReady: false, ttsReady: false, state: "idle", error: "" };
    },
  };
}