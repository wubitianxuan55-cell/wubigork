// mock/model.ts — 模型中心域（T6-10.1 拆分自 lib/mock.ts，方法体零改动；
// Models/ModelSwitchEstimate 契约对齐 Go，T6-10.4）。
// 注意 keepWarm/preloadPlan 会被整体重绑，必须经 s.keepWarm / s.preloadPlan
// 读写（setter 经 state.setKeepWarm/setPreloadPlan 重绑），不能解构快照。
import type { AppBindings } from "../bridge";
import type { ModelSwitchEstimate } from "../types";
import type { MakeMockState } from "./state";

type ModelMethods = Pick<
  AppBindings,
  | "Models" | "SetModel"
  | "KeepWarmGet" | "KeepWarmSet" | "PreloadPlanGet" | "PreloadPlanSet"
  | "ModelSwitchEstimate" | "Balance"
>;

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
