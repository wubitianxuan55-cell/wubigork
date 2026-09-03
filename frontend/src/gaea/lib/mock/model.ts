// mock/model.ts — 模型中心域（T6-10.1 拆分自 lib/mock.ts，方法体零改动；
// Models/ModelSwitchEstimate 契约对齐 Go，T6-10.4）。
// 注意 keepWarm/preloadPlan/engineFailover 会被整体重绑，必须经 s.keepWarm /
// s.preloadPlan / s.engineFailover 读写（setter 经 state.setKeepWarm 等
// 重绑），不能解构快照。
import type { AppBindings } from "../bridge";
import type { ModelSwitchEstimate } from "../types";
import type { MakeMockState } from "./state";

// GetEngineFailover/SetEngineFailover 是 legacy 绑定面（挂在后端 ModelB 门面，
// Go 同名，经 window.go.app.App 兼容代理按方法名路由），按 bridge.ts 的
// LegacySurfaceNames 约定不进 AppBindings——这里用局部方法类型补足，避免牵动
// AppBindings 双向类型锁（bindingNames 已含两名，drift 校验零变更）。
type LegacyModelMethods = {
  GetEngineFailover(): Promise<boolean>;
  SetEngineFailover(enabled: boolean): Promise<void>;
};

type ModelMethods = Pick<
  AppBindings,
  | "Models" | "SetModel"
  | "KeepWarmGet" | "KeepWarmSet" | "PreloadPlanGet" | "PreloadPlanSet"
  | "ModelSwitchEstimate" | "Balance"
> & LegacyModelMethods;

export function buildModel(s: MakeMockState): ModelMethods {
  return {
    async Models() {
      // 契约对齐 Go GaeaModels（internal/app/gaea_ui_meta.go）：ref = 引擎ID + "/" + 模型，
      // provider = 引擎 ID，model 为空回退 "(默认)"，current = 活跃引擎；引擎默认模型取
      // internal/modelengine/engine.go 预置值（deepseek→deepseek-v4-pro、xai→grok-4.20、
      // herdsman/ollama 默认模型为空 → "(默认)"）。
      return [
        { ref: "xai/grok-4.20", provider: "xai", model: "grok-4.20", current: false },
        { ref: "ollama/(默认)", provider: "ollama", model: "(默认)", current: false },
        { ref: "herdsman/(默认)", provider: "herdsman", model: "(默认)", current: false },
        { ref: "deepseek/deepseek-v4-pro", provider: "deepseek", model: "deepseek-v4-pro", current: true },
      ];
    },
    async SetModel() {},
    async KeepWarmGet() {
      return s.keepWarm;
    },
    async KeepWarmSet(enabled: boolean) {
      s.setKeepWarm(enabled);
    },
    async PreloadPlanGet() {
      return s.preloadPlan;
    },
    async PreloadPlanSet(enabled: boolean) {
      s.setPreloadPlan(enabled);
    },
    // 故障转移开关（C 刀）：Get 读 / Set 写回 mock state（默认 false，会话内
    // 一致）。与 KeepWarmGet/Set 同构；真实后端持久化与失败换引擎重试在
    // ModelB 门面（并行线），mock 只需锁「可读布尔 + 写回一致」契约。
    async GetEngineFailover() {
      return s.engineFailover;
    },
    async SetEngineFailover(enabled: boolean) {
      s.setEngineFailover(enabled);
    },
    async ModelSwitchEstimate(engineID: string): Promise<ModelSwitchEstimate> {
      // 契约对齐 Go GaeaModelSwitchEstimate（internal/app/gaea_schedule.go）：
      // 非 herdsman 引擎恒为 hot（引擎常驻，waitSeconds=1）；herdsman 演示冷启动。
      if (engineID !== "herdsman") {
        return {
          engine: engineID,
          model: "",
          status: "hot",
          waitSeconds: 1,
          note: "引擎已就绪",
        };
      }
      return {
        engine: engineID,
        model: "",
        status: "cold",
        waitSeconds: 20,
        note: "本地模型需冷启动，实测约 15-20 秒（mock）",
      };
    },
    async Balance() {
      // Mirror the active mock provider: deepseek-flash carries a balance_url.
      const p = s.settings.providers.find((x) => x.name === s.settings.defaultModel);
      if (!p?.balanceUrl) return { available: false, display: "" };
      return { available: true, display: "¥128.50" };
    },
  };
}
