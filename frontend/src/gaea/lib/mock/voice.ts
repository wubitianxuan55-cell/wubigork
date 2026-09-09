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
  };
}