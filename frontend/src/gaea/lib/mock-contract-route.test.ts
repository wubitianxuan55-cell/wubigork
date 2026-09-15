// 路由学习四名（阶段七 7.1-2）契约层 mock 冒烟测试：模型中心「成本归因」tab
// （AttributionSection → api/engines getRouteLedger/getRouteSuggestions/
// applyRouteSuggestion/ignoreRouteSuggestion → Go ModelB 同名绑定）。锁定 mock
// 契约：四名存在性 + 返回形状关键键——键名与 api/engines.ts 消费方解析字段、
// bridge/model.ts 局部重述类型（RouteLedgerViewB/RouteSuggestionsViewB）三方
// 一一对应（bridge 侧局部重述不 import api 层防环，字段漂移由本测试兜底，
// herdsman 先例）。
import { describe, expect, it } from "vitest";
import { app } from "./bridge";

/** 路由绑定面窄化视图（mock 实现见 mock/model.ts；真实后端见 Go ModelB）。 */
const route = app as unknown as {
  GaeaRouteLedger(): Promise<{
    generated_at: string;
    total: {
      calls: number;
      tokens_in: number;
      tokens_out: number;
      cost_cny: number;
      avg_ms: number;
      success_rate: number;
    };
    features: Array<{
      feature: string;
      engine_id: string;
      model: string;
      calls: number;
      tokens_in: number;
      tokens_out: number;
      cost_cny: number;
      avg_ms: number;
      success_rate: number;
    }>;
  }>;
  GaeaRouteSuggestions(): Promise<{
    generated_at: string;
    suggestions: Array<{
      id: string;
      feature: string;
      reason: string;
      evidence: string;
      from: { engine_id: string; model: string };
      to: { engine_id: string; model: string; is_local: boolean; success_rate?: number; cost_cny: number };
      score_gap: number;
    }>;
  }>;
  GaeaRouteSuggestionApply(id: string): Promise<void>;
  GaeaRouteSuggestionIgnore(id: string): Promise<void>;
};

describe("mock 契约 · 路由学习四名（模型中心成本归因 tab）", () => {
  it("四名均存在于 mock 绑定面（契约：可调用、不 undefined）", () => {
    for (const n of ["GaeaRouteLedger", "GaeaRouteSuggestions", "GaeaRouteSuggestionApply", "GaeaRouteSuggestionIgnore"]) {
      expect(typeof (app as unknown as Record<string, unknown>)[n], n).toBe("function");
    }
  });

  it("GaeaRouteLedger：账本形状（total 六键 + features 行含 feature/engine_id/model 九键）", async () => {
    const v = await route.GaeaRouteLedger();
    expect(typeof v.generated_at).toBe("string");
    for (const k of ["calls", "tokens_in", "tokens_out", "cost_cny", "avg_ms", "success_rate"] as const) {
      expect(typeof v.total[k], `total.${k}`).toBe("number");
    }
    expect(Array.isArray(v.features)).toBe(true);
    expect(v.features.length).toBeGreaterThan(0);
    for (const row of v.features) {
      for (const k of ["feature", "engine_id", "model"] as const) {
        expect(typeof row[k], `features[].${k}`).toBe("string");
      }
      for (const k of ["calls", "tokens_in", "tokens_out", "cost_cny", "avg_ms", "success_rate"] as const) {
        expect(typeof row[k], `features[].${k}`).toBe("number");
      }
    }
    // 未标记桶（feature=""）排在最后——统计层定序契约。
    const feats = v.features.map((f) => f.feature);
    const lastNonEmpty = feats.reduce((acc, f, i) => (f !== "" ? i : acc), -1);
    const firstEmpty = feats.findIndex((f) => f === "");
    if (firstEmpty >= 0) expect(firstEmpty).toBeGreaterThan(lastNonEmpty);
  });

  it("GaeaRouteSuggestions：建议形状（含本地引擎样本，判据④走查样本）", async () => {
    const v = await route.GaeaRouteSuggestions();
    expect(typeof v.generated_at).toBe("string");
    expect(Array.isArray(v.suggestions)).toBe(true);
    const s = v.suggestions[0];
    expect(s).toBeDefined();
    for (const k of ["id", "feature", "reason", "evidence"] as const) {
      expect(typeof s[k], `suggestion.${k}`).toBe("string");
    }
    expect(typeof s.from.engine_id).toBe("string");
    expect(typeof s.score_gap).toBe("number");
    expect(typeof s.to.engine_id).toBe("string");
    expect(typeof s.to.model).toBe("string");
    expect(typeof s.to.is_local).toBe("boolean");
    expect(typeof s.to.cost_cny).toBe("number");
    // mock 样本给本地引擎候选（herdsman）——归因 tab「本地/云端」徽标走查锚点。
    expect(s.to.is_local).toBe(true);
  });

  it("Apply/Ignore：mock 环境为 no-op（确认语义；真实链路=SetFeatureModel/状态文件）", async () => {
    await expect(route.GaeaRouteSuggestionApply("chat|xai>grok-4>herdsman>qwen3:32b")).resolves.toBeUndefined();
    await expect(route.GaeaRouteSuggestionIgnore("chat|xai>grok-4>herdsman>qwen3:32b")).resolves.toBeUndefined();
  });
});
