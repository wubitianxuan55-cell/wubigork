import type { WireShape } from "./shared";
import type { app as AppModels } from "../../../../wailsjs/go/models";

// Settings panel payloads (desktop/settings_app.go).
export type ProviderView = WireShape<AppModels.ProviderView>;

// BalanceInfo is the wallet-balance readout (desktop/app.go Balance). available
// is false when the provider declares no balanceUrl or a fetch failed; display is
// the formatted amount (e.g. "¥110.00").
export type BalanceInfo = WireShape<AppModels.BalanceInfo>;

// JobView is one running background job (desktop/app.go Jobs) for the status bar.
export type JobView = WireShape<AppModels.JobView>;


// FactBaseView: the fact-base panel view: facts + copy-ready Markdown.
export type FactBaseView = WireShape<AppModels.FactBaseView>;

// SkillCaptureInput 是一次成功对话沉淀为技能的输入（桌面端 GaeaCaptureSkill）。
export type SkillCaptureInput = WireShape<AppModels.SkillCaptureInput>;

// SkillCaptureResult 是沉淀结果；reloaded=true 表示技能已热加载进引擎。
export type SkillCaptureResult = WireShape<AppModels.SkillCaptureResult>;

// SkillDraft 是会话录制蒸馏出的可编辑技能草稿（阶段七 7.2-1）。
export type SkillDraft = WireShape<AppModels.SkillDraft>;

// SkillRecordResult 是会话录制返回：草稿+回放摘要+SKILL.md 预览。
export type SkillRecordResult = WireShape<AppModels.SkillRecordResult>;

export type PermissionsView = WireShape<AppModels.PermissionsView>;

export type SandboxView = WireShape<AppModels.SandboxView>;

export type AgentView = WireShape<AppModels.AgentView>;

export interface SettingsView {
  defaultModel: string;
  subagentModel: string;
  subagentModels: Record<string, string>; // per-skill overrides
  subagentSkills: string[]; // builtin subagent skill names
  providers: ProviderView[];
  permissions: PermissionsView;
  sandbox: SandboxView;
  agent: AgentView;
  configPath: string;
  providerKinds: string[]; // provider implementations the kernel registered (for the kind picker)
  bypass: boolean; // DEPRECATED — use permLevel instead
  permLevel?: string; // live permission level this session (\"ask\"|\"auto\"|\"yolo\")
}
