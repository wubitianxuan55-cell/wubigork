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
  // FE3-04：SetSubagentTemperature/SetEffort/SetSubagentModel 三个 mock-only 死名
  // 已整体摘除——Go 侧从无对应绑定（bindingSignatures 744 名可证），前端亦零调用
  // 者；推理强度/子代理模型的实际写口见下两个真实绑定（Gaea 前缀同名直调）。
  // SetSubagentEffort sets the reasoning effort for sub-agents. "" = inherit from Effort.
  SetSubagentEffort(effort: string): Promise<void>;
  // SetSubagentModelForSkill sets a per-skill sub-agent model override.
  // skill is one of explore|research|review|security-review. Empty ref = inherit.
  SetSubagentModelForSkill(skill: string, ref: string): Promise<void>;
  // SetPermLevel controls permission strictness: "ask" (default, prompt before writes),
  // "auto" (allow writes without asking), or "yolo" (skip all prompts).
  SetPermLevel(level: string): Promise<void>;
  // 办公运行模式（2026-11-21 众神殿 pan_1795202763988 终案，Gaea 前缀同名直调）：
  // SetOfficeRunMode 只登记默认档（standard|cost|proposal，立即返回不重建）——
  // 当前会话保持现有档，GaeaNewSession 发现配置档≠引擎档时先重建再建会话
  // （「新会话生效」）。OfficeRunMode 返回 {configured, active, hide, extras}。
  GaeaSetOfficeRunMode(mode: string): Promise<void>;
  GaeaOfficeRunMode(): Promise<OfficeRunModeViewB>;
  // 能力面板「工具」tab 装配面（2026-11-21 面板缝）：从引擎主注册表活取
  // （含模式藏面 hidden 灰显）；引擎未初始化时 tools 空——前端落静态清单兜底。
  GaeaToolSurface(): Promise<ToolSurfaceViewB>;
  WenxueToolSurface(): Promise<ToolSurfaceViewB>;
  // 工具面双轴（2026-11-21 众神殿 pan_1795224680124 终案）：GaeaToolFace 返回
  // {face, mode, groups[{id,label,count,enabled,overridable,note}], overrides}；
  // 三个写口只登记默认值（新会话生效，不重建）。
  GaeaToolFace(): Promise<ToolFaceViewB>;
  GaeaSetToolFace(face: string): Promise<void>;
  GaeaSetToolGroup(group: string, enabled: boolean): Promise<void>;
  GaeaResetToolGroups(): Promise<void>;
  // 电脑操控注册闸（2026-11-21 用户问询后补 UI）：开关写 [computer_use].enabled，
  // 新会话重建后整族注册（boot 期注册闸）。
  GaeaSetComputerUseEnabled(enabled: boolean): Promise<void>;
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
  // 按目标模型口径覆盖全部本地引擎（herdsman/strata/llama）；model 省略
  // 时后端回退引擎默认模型。
  ModelSwitchEstimate(engineID: string, model?: string): Promise<ModelSwitchEstimate>;
  // Feature Model 功能级模型绑定（写侧，Go ModelB.SetFeatureModel 3 参 /
  // SetFeatureModelEnabled 2 参，wailsjsCompat 直调转正；useBindState 消费，
  // feature 取值 chat/whisper/office/gaea/wenxue/characterlib）。
  SetFeatureModel(feature: string, engineID: string, modelName: string): Promise<void>;
  SetFeatureModelEnabled(feature: string, enabled: boolean): Promise<void>;
  // GetActiveModel 当前活跃模型名（空=无可用引擎，Go ModelB.GetActiveModel，
  // 同批转正；api/settings.ts getActiveModel / 模型中心 / MainLayout 消费）。
  GetActiveModel(): Promise<string>;
  // ── 路由学习四名（阶段七 7.1-2，Go ModelB 同名；模型中心「成本归因」tab
  // 经 api/engines.ts App() 兼容通道消费）。视图形状局部重述对齐
  // api/engines.ts 的 RouteLedgerView/RouteSuggestionsView（api 层 import 会成
  // api→gaea→api 环，字段漂移由 mock-contract-route 契约测试兜底——herdsman 先例）。──
  GaeaRouteLedger(): Promise<RouteLedgerViewB>;
  GaeaRouteSuggestions(): Promise<RouteSuggestionsViewB>;
  GaeaRouteSuggestionApply(id: string): Promise<void>;
  GaeaRouteSuggestionIgnore(id: string): Promise<void>;
  // GaeaLocalServiceModels 本机内嵌/直连服务模型卡（模型中心本地服务五卡
  // 212b34a0；api/engines.ts App() 兼容通道消费，此处认领过 drift 方向二——
  // 视图形状局部重述同 RouteLedger 先例，字段漂移由消费方测试兜底）。
  GaeaLocalServiceModels(): Promise<LocalServiceModelB[]>;
}

// LocalServiceModelB 与 api/engines.ts LocalServiceModel 同形（环规避先例）。
export interface LocalServiceModelB {
  engineId: string;
  modelId: string;
  name: string;
  kind: string;
  detail: string;
  ready: boolean;
}

// OfficeRunModeViewB 与 Go app.OfficeRunModeView 同形（「查看配置」诚实面：
// configured=默认档，active=引擎实际档；hide=该档隐藏的专用工具，extras=该档
// 追加的内嵌技能）。
export interface OfficeRunModeViewB {
  configured: string;
  active: string;
  hide: string[];
  extras: string[];
}

// ToolSurfaceViewB 与 Go app.ToolSurfaceView 同形：board=gaea|wenxue；
// mode=引擎实际运行模式（wenxue 恒空）；tools=主注册表全量装配面（含 hidden）。
export interface ToolSurfaceViewB {
  board: string;
  mode: string;
  tools: ToolSurfaceEntryB[];
}

// ToolSurfaceEntryB 与 Go control.ToolSurfaceEntry 同形：description=模型可见
// 口径（CompactDescriptor 工具给紧凑描述）；hidden=模式藏面（协议层按名可调）；
// group=功能组 id（""=core，组开关数据源）。
export interface ToolSurfaceEntryB {
  name: string;
  description: string;
  source: string;
  group: string;
  hidden: boolean;
}

// ToolFaceViewB 与 Go app.ToolFaceView 同形（工具面双轴视图）。
export interface ToolFaceViewB {
  face: string;
  mode: string;
  groups: ToolFaceGroupViewB[];
  overrides: Record<string, boolean>;
}

export interface ToolFaceGroupViewB {
  id: string;
  label: string;
  count: number;
  enabled: boolean;
  overridable: boolean;
  // gate=true=注册闸组（computer）：开关走 GaeaSetComputerUseEnabled。
  gate: boolean;
  note: string;
}

// RouteLedgerTotalB 账本总计（与 api/engines.ts RouteLedgerTotal 同形）。
export interface RouteLedgerTotalB {
  calls: number;
  tokens_in: number;
  tokens_out: number;
  cost_cny: number;
  avg_ms: number;
  success_rate: number; // 0-1；无调用时 0
}

// FeatureUsageSummaryB 单个 feature×engine×model 桶账目行。
export interface FeatureUsageSummaryB {
  feature: string; // 功能绑定键；空串=历史/未标记
  engine_id: string;
  model: string;
  calls: number;
  tokens_in: number;
  tokens_out: number;
  cost_cny: number;
  avg_ms: number;
  success_rate: number; // 0-1
}

// RouteLedgerViewB 路由账本（GaeaRouteLedger 返回形状）。
export interface RouteLedgerViewB {
  generated_at: string;
  total: RouteLedgerTotalB;
  features: FeatureUsageSummaryB[]; // feature="" 桶=未标记，排序最后
}

// RouteEndpointB 建议绑定端点。
export interface RouteEndpointB {
  engine_id: string;
  model: string;
}

// RouteSuggestionTargetB 建议目标（含质量/成本供展示）。
export interface RouteSuggestionTargetB {
  engine_id: string;
  model: string;
  is_local: boolean;
  success_rate?: number; // 无样本时缺省
  cost_cny: number;
}

// RouteSuggestionB 一条改绑建议。
export interface RouteSuggestionB {
  id: string;
  feature: string;
  reason: string;
  evidence: string;
  from: RouteEndpointB;
  to: RouteSuggestionTargetB;
  score_gap: number;
}

// RouteSuggestionsViewB 建议视图（GaeaRouteSuggestions 返回形状）。
export interface RouteSuggestionsViewB {
  generated_at: string;
  suggestions: RouteSuggestionB[];
}
