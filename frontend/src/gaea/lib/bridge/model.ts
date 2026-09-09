// model.ts — ModelBindings（AppBindings 分域接口之一，Go ModelB 门面）：
// 模型/供应商设置/权限/本地模型调度（保活/预载/换模预估）域。
import type {
  ModelInfo,
  ModelSwitchEstimate,
  ProviderView,
  SettingsView,
} from "../types";

export interface ModelBindings {
  Models(): Promise<ModelInfo[]>;
  SetModel(name: string): Promise<void>;
  // Settings panel: read the resolved config and apply edits (each writes config
  // and rebuilds the controller live). Secrets go through SetProviderKey (→ .env).
  Settings(): Promise<SettingsView>;
  SetDefaultModel(ref: string): Promise<void>;
  SaveProvider(p: ProviderView): Promise<void>;
  DeleteProvider(name: string): Promise<void>;
  LoginProvider(name: string): Promise<void>;
  LogoutProvider(name: string): Promise<void>;
  SetProviderKey(apiKeyEnv: string, value: string): Promise<void>;
  SetPermissionMode(mode: string): Promise<void>;
  AddPermissionRule(list: string, rule: string): Promise<void>;
  RemovePermissionRule(list: string, rule: string): Promise<void>;
  SetSandbox(bash: string, network: boolean, workspaceRoot: string, allowWrite: string[]): Promise<void>;
  SetAgentParams(temperature: number, maxSteps: number, systemPrompt: string): Promise<void>;
  // SetSubagentTemperature sets the subagent-specific temperature override.
  // 0 means "use the global temperature".
  SetSubagentTemperature(temp: number): Promise<void>;
  // SetEffort sets the reasoning effort for the executor. "" = provider default.
  SetEffort(effort: string): Promise<void>;
  // SetSubagentEffort sets the reasoning effort for sub-agents. "" = inherit from Effort.
  SetSubagentEffort(effort: string): Promise<void>;
  // SetSubagentModel sets the default model for spawned sub-agents. An empty string
  // clears it so sub-agents inherit the parent's provider.
  SetSubagentModel(ref: string): Promise<void>;
  // SetSubagentModelForSkill sets a per-skill sub-agent model override.
  // skill is one of explore|research|review|security-review. Empty ref = inherit.
  SetSubagentModelForSkill(skill: string, ref: string): Promise<void>;
  // SetPermLevel controls permission strictness: "ask" (default, prompt before writes),
  // "auto" (allow writes without asking), or "yolo" (skip all prompts).
  SetPermLevel(level: string): Promise<void>;
  // ── 阶段 5 T5-3 本地模型调度纵深 ──
  // KeepWarmGet/KeepWarmSet 保活开关：空闲时定期轻量探测，防止本地模型被卸载
  // （~/.gaea_config.json 持久化，重启后仍生效）。
  KeepWarmGet(): Promise<boolean>;
  KeepWarmSet(enabled: boolean): Promise<void>;
  // PreloadPlanGet/PreloadPlanSet 启动自动预载开关：启动时后台预载常用本地模型，
  // 减少首次对话等待。
  PreloadPlanGet(): Promise<boolean>;
  PreloadPlanSet(enabled: boolean): Promise<void>;
  // ModelSwitchEstimate 换模预估：目标本地模型 hot/cold/download/unknown，
  // 前端在非 hot 时提示预计等待并让用户确认是否继续切换。v4.126 刀2 起
  // 按目标模型口径覆盖全部本地引擎（herdsman/ollama/modelhub）；model 省略
  // 时后端回退引擎默认模型。
  ModelSwitchEstimate(engineID: string, model?: string): Promise<ModelSwitchEstimate>;
  // Feature Model 功能级模型绑定（写侧，Go ModelB.SetFeatureModel 3 参 /
  // SetFeatureModelEnabled 2 参，wailsjsCompat 直调转正；useBindState 消费，
  // feature 取值 chat/whisper/novel/office/gaea/characterlib）。
  SetFeatureModel(feature: string, engineID: string, modelName: string): Promise<void>;
  SetFeatureModelEnabled(feature: string, enabled: boolean): Promise<void>;
  // GetActiveModel 当前活跃模型名（空=无可用引擎，Go ModelB.GetActiveModel，
  // 同批转正；api/settings.ts getActiveModel / 模型中心 / MainLayout 消费）。
  GetActiveModel(): Promise<string>;
}
