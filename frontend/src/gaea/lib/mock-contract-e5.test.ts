// 阶段 5「进料与质量」契约层 mock 冒烟测试：锁定 E5 新增绑定方法的行为，
// 供 E2 组件（PDF/图片成本导入、比价）与统一搜索入口联调时参考。
// T6-10.4 起 RetrievalEvalRun 改为契约校验：断言与 Go 侧
// internal/app/gaea_retrieval_eval.go + docs/retrieval-eval-set.md 一致的结构与
// 数值锚点（total=12、threshold=0.8、passed 与 recallAt10 关系、perQuery 形状），
// 不再锁定 0.85 虚构值——若 mock 与 Go 契约不一致，本用例会在数值锚点或
// 结构断言处失败，从而暴露漂移（见阶段报告「契约校验如何暴露不一致」）。
import { describe, expect, it } from "vitest";
import { app } from "./bridge";

describe("E5 契约层 mock（进料与质量）", () => {
  it("CostImportVisionPreview 返回 source=pdf_text 的候选行", async () => {
    const pv = await app.CostImportVisionPreview("C:/tmp/报价单.pdf");
    // 字段对齐 Go gaea_cost_import_vision.go（source=pdf_text / image；aiUsed 标记）
    expect(pv.source).toBe("pdf_text");
    expect(pv.aiUsed).toBe(true);
    expect(pv.rows.length).toBeGreaterThanOrEqual(2);
    expect(pv.rows[0].name).toBe("rebar");
  });

  it("CostCompare 返回多源比价明细（现价/历史/抓取）", async () => {
    const rows = await app.CostCompare("rebar");
    // kind 枚举对齐 Go CostCompareRow（current/history/fetch）
    expect(rows.length).toBeGreaterThanOrEqual(2);
    expect(rows[0].kind).toBe("current");
    expect(rows.some((r) => r.kind === "fetch")).toBe(true);
    expect(rows.some((r) => r.kind === "history")).toBe(true);
    rows.forEach((r) => {
      expect(["current", "history", "fetch"]).toContain(r.kind);
      expect(typeof r.price).toBe("number");
      expect(typeof r.diffPct).toBe("number");
      expect(typeof r.source).toBe("string");
    });
  });

  it("UnifiedSearch 一次返回关键词 + 语义两组，scope + topN 生效", async () => {
    // S1.2-C：签名变为 (query, scope, topN)；scope="work" 走 work 分区演示命中。
    const v = await app.UnifiedSearch("打桩", "work", 20);
    // 结构对齐 Go gaea_unified_search.go（keyword/semantic 两组）
    expect(Array.isArray(v.keyword)).toBe(true);
    expect(Array.isArray(v.semantic)).toBe(true);
    expect(v.keyword.length + v.semantic.length).toBeGreaterThan(0);
    // 语义命中带库徽标字段
    v.semantic.forEach((h) => {
      expect(["cost", "knowledge", "office", "file"]).toContain(h.kind);
    });
  });

  it("UnifiedSearch 记忆统一层扩展：brain/files 两组结构对齐 Go", async () => {
    const v = await app.UnifiedSearch("振动锤", "", 8);
    // 记忆统一层第一刀：GaeaUnifiedSearch 四组视图（keyword/semantic/brain/files）
    expect(Array.isArray(v.brain)).toBe(true);
    expect(Array.isArray(v.files)).toBe(true);
    // brain 命中：三脑命名空间 + entity/text
    (v.brain ?? []).forEach((h) => {
      expect(typeof h.brain).toBe("string");
      expect(typeof h.entity).toBe("string");
      expect(typeof h.text).toBe("string");
      expect(typeof h.score).toBe("number");
    });
    // files 命中：path/snippet（供 hub 预览 / @引用）
    (v.files ?? []).forEach((h) => {
      expect(typeof h.path).toBe("string");
      expect(typeof h.snippet).toBe("string");
    });
  });

  it("UnifiedSearch scope=work：三脑滤 brain.right（轻语=play 专属），语义只含 work 域", async () => {
    const v = await app.UnifiedSearch("振动锤", "work", 8);
    // 设计检索面地图：brain.left/main 属 work，brain.right 属 play → work scope 滤 right。
    expect((v.brain ?? []).some((h) => h.brain === "brain.right")).toBe(false);
    expect((v.brain ?? []).some((h) => h.brain === "brain.left")).toBe(true);
    // semantic 只对最终 hits 过滤：work 演示域为 cost/knowledge（非 play 分区命中）。
    expect((v.semantic ?? []).some((h) => h.name.includes("娱乐"))).toBe(false);
  });

  it("UnifiedSearch scope=play：三脑只留 brain.right，语义为 play 分区命中", async () => {
    const v = await app.UnifiedSearch("游戏", "play", 8);
    expect((v.brain ?? []).every((h) => h.brain === "brain.right")).toBe(true);
    expect((v.brain ?? []).length).toBeGreaterThan(0);
    expect((v.semantic ?? []).some((h) => h.name.includes("娱乐"))).toBe(true);
  });

  it("UnifiedSearch scope=\"\"（全部）：work + play 两分区命中都返回（显式选择才跨空间）", async () => {
    const v = await app.UnifiedSearch("振动锤", "", 8);
    expect((v.brain ?? []).some((h) => h.brain === "brain.left")).toBe(true);
    expect((v.brain ?? []).some((h) => h.brain === "brain.right")).toBe(true);
    expect((v.semantic ?? []).some((h) => h.name.includes("娱乐"))).toBe(true);
  });

  it("RetrievalEvalRun 契约对齐 Go：12 条真实查询集、threshold=0.8、recall 自洽", async () => {
    const r = await app.RetrievalEvalRun();
    // 数值锚点（Go GaeaRetrievalEvalRun + docs/retrieval-eval-set.md）：
    // 查询集条数 12（非 0.85 时代的虚构 4 条）
    expect(r.total).toBe(12);
    expect(r.threshold).toBe(0.8); // Go retrievalEvalThreshold
    // passed 与 recallAt10 的关系（Go: passed = recallAt10 >= threshold）
    expect(r.passed).toBe(r.recallAt10 >= r.threshold);
    expect(r.recallAt10).toBeGreaterThanOrEqual(0);
    expect(r.recallAt10).toBeLessThanOrEqual(1);
    // 结构（Go RetrievalEvalReport / RetrievalEvalQuery）
    expect(r.perQuery).toHaveLength(r.total);
    r.perQuery.forEach((q) => {
      expect(typeof q.query).toBe("string");
      expect(Array.isArray(q.expected)).toBe(true);
      expect(Array.isArray(q.topHits)).toBe(true);
      expect(typeof q.recall).toBe("number");
      expect(q.recall).toBeGreaterThanOrEqual(0);
      expect(q.recall).toBeLessThanOrEqual(1);
      // expected/topHits 均为 "kind:name" 形式（Go RetrievalEvalQuery 约定）
      q.expected.forEach((e) => expect(e).toMatch(/^[a-z]+:.+$/));
      q.topHits.forEach((h) => expect(h).toMatch(/^[a-z]+:.+$/));
    });
    // 首条查询锚点：与 docs/retrieval-eval-set.md 第 1 条一致
    expect(r.perQuery[0].query).toBe("打桩设备 台班价");
    expect(r.perQuery[0].expected).toEqual(["cost:HP300 高频液压振动锤", "cost:桩机台班费"]);
    // 演示命中下整体通过门槛（recall@10 = 11/12 ≈ 0.9167 ≥ 0.8）
    expect(r.passed).toBe(true);
  });
});
