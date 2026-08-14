// 阶段 5「进料与质量」契约层 mock 冒烟测试：锁定 E5 新增绑定方法的行为，
// 供 E2 组件（PDF/图片成本导入、比价）与统一搜索入口联调时参考。
import { describe, expect, it } from "vitest";
import { app } from "./bridge";

describe("E5 契约层 mock（进料与质量）", () => {
  it("CostImportVisionPreview 返回 source=pdf_text 的候选行", async () => {
    const pv = await app.CostImportVisionPreview("C:/tmp/报价单.pdf");
    expect(pv.source).toBe("pdf_text");
    expect(pv.aiUsed).toBe(true);
    expect(pv.rows.length).toBeGreaterThanOrEqual(2);
    expect(pv.rows[0].name).toBe("rebar");
  });

  it("CostCompare 返回多源比价明细（现价/历史/抓取）", async () => {
    const rows = await app.CostCompare("rebar");
    expect(rows.length).toBeGreaterThanOrEqual(2);
    expect(rows[0].kind).toBe("current");
    expect(rows.some((r) => r.kind === "fetch")).toBe(true);
    expect(rows.some((r) => r.kind === "history")).toBe(true);
    rows.forEach((r) => {
      expect(typeof r.price).toBe("number");
      expect(typeof r.diffPct).toBe("number");
      expect(typeof r.source).toBe("string");
    });
  });

  it("UnifiedSearch 一次返回关键词 + 语义两组，topN 生效", async () => {
    const v = await app.UnifiedSearch("打桩", 20);
    expect(Array.isArray(v.keyword)).toBe(true);
    expect(Array.isArray(v.semantic)).toBe(true);
    expect(v.keyword.length + v.semantic.length).toBeGreaterThan(0);
    // 语义命中带库徽标字段
    v.semantic.forEach((h) => {
      expect(["cost", "knowledge", "office"]).toContain(h.kind);
    });
  });

  it("RetrievalEvalRun 返回 recall@10=0.85 且通过门槛（无参数，门槛后端固定）", async () => {
    const r = await app.RetrievalEvalRun();
    expect(r.total).toBe(4);
    expect(r.threshold).toBe(0.8);
    expect(r.recallAt10).toBeCloseTo(0.85, 2);
    expect(r.passed).toBe(true);
    expect(r.perQuery.length).toBe(r.total);
    r.perQuery.forEach((q) => {
      expect(Array.isArray(q.expected)).toBe(true);
      expect(Array.isArray(q.topHits)).toBe(true);
      expect(typeof q.recall).toBe("number");
    });
  });
});
