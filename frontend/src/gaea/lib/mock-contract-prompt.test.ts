// T6 提示词工坊契约层 mock 冒烟测试：PromptTemplate 五名（List/Get/Save/Reset/
// Preview，Go NovelB 门面，t6 首刀规格 §4.2）+ t6-C2 模板包两名
// （PromptBundleExport/Import，规格 进度计划/gaea-prompt-bundle-t6c2-20260916.md）。
// 锁定 AppBindings 契约 + mock 实现形状——键名与 api/prompt.ts 消费方解析字段、
// 面板（PromptWorkshopPanel）渲染分支三方一一对应（先例：mock-contract-route.test.ts）。
import { describe, expect, it } from "vitest";
import { app } from "./bridge";

/** 提示词工坊绑定面窄化视图（mock 实现见 mock/novel.ts；真实后端见 Go NovelB）。 */
const promptApi = app as unknown as {
  PromptTemplateList(): Promise<
    Array<{
      key: string;
      category: string;
      description: string;
      source: string;
      hasOverride: boolean;
      overrideActive: boolean;
      version: number;
      updatedAt: number;
    }>
  >;
  PromptTemplateGet(key: string): Promise<{
    meta: { key: string; source: string; version: number };
    template: { system: string; task: string; parameters?: string[] };
    base: { system: string; task: string };
  }>;
  PromptTemplateSave(
    key: string,
    reqJSON: string,
  ): Promise<{ saved: boolean; issues: Array<{ code: string; severity: string; message: string }>; version: number }>;
  PromptTemplateReset(key: string): Promise<void>;
  PromptTemplatePreview(reqJSON: string, varsJSON: string): Promise<{ systemPrompt: string; warnings: string[] }>;
  PromptBundleExport(): Promise<string>;
  PromptBundleImport(bundleJSON: string): Promise<{
    applied: boolean;
    statistics: Record<string, number>;
    outcomes: Array<{ key: string; action: string; reason?: string }>;
  }>;
};

describe("mock 契约 · 提示词工坊五名（t6 首刀：模板可编辑覆盖层）", () => {
  it("五名均存在于 mock 绑定面（契约：可调用、不 undefined）", () => {
    for (const n of ["PromptTemplateList", "PromptTemplateGet", "PromptTemplateSave", "PromptTemplateReset", "PromptTemplatePreview", "PromptBundleExport", "PromptBundleImport"]) {
      expect(typeof (app as unknown as Record<string, unknown>)[n], n).toBe("function");
    }
  });

  it("List：数组形态 + Meta 八键，且含一个 override 态样本", async () => {
    const rows = await promptApi.PromptTemplateList();
    expect(Array.isArray(rows)).toBe(true);
    expect(rows.length).toBeGreaterThanOrEqual(2);
    for (const row of rows) {
      for (const k of ["key", "category", "description", "source"] as const) {
        expect(typeof row[k], `meta.${k}`).toBe("string");
      }
      for (const k of ["hasOverride", "overrideActive"] as const) {
        expect(typeof row[k], `meta.${k}`).toBe("boolean");
      }
      for (const k of ["version", "updatedAt"] as const) {
        expect(typeof row[k], `meta.${k}`).toBe("number");
      }
    }
    const override = rows.find((r) => r.source === "override");
    expect(override, "create-chapter override 样本").toBeDefined();
    expect(override!.hasOverride).toBe(true);
    expect(override!.overrideActive).toBe(true);
    expect(override!.version).toBeGreaterThan(0);
    // 内置态样本：version=0、updatedAt=0
    const builtin = rows.find((r) => r.source === "builtin");
    expect(builtin).toBeDefined();
    expect(builtin!.version).toBe(0);
  });

  it("Get：meta/template/base 三段，Base.system 与生效 Template.system 不同（面板对照锚点）", async () => {
    const d = await promptApi.PromptTemplateGet("create-chapter");
    expect(typeof d.meta.key).toBe("string");
    expect(typeof d.meta.version).toBe("number");
    expect(typeof d.template.system).toBe("string");
    expect(typeof d.template.task).toBe("string");
    expect(typeof d.base.system).toBe("string");
    expect(Array.isArray(d.template.parameters)).toBe(true);
    expect(d.template.system).not.toBe(d.base.system);
  });

  it("Save：正常保存 saved=true 带 legacy-brace warn 样本；System 为空 error 阻断；未知键拒绝", async () => {
    const ok = await promptApi.PromptTemplateSave(
      "create-chapter",
      JSON.stringify({ isActive: true, category: "chapter", description: "d", content: { system: "S", task: "T" } }),
    );
    expect(ok.saved).toBe(true);
    expect(ok.version).toBeGreaterThan(0);
    expect(Array.isArray(ok.issues)).toBe(true);
    expect(ok.issues.some((i) => i.code === "legacy-brace" && i.severity === "warn")).toBe(true);
    for (const i of ok.issues) {
      expect(typeof i.code).toBe("string");
      expect(typeof i.message).toBe("string");
    }

    const blocked = await promptApi.PromptTemplateSave(
      "create-chapter",
      JSON.stringify({ isActive: true, category: "", description: "", content: { system: "  ", task: "T" } }),
    );
    expect(blocked.saved).toBe(false);
    expect(blocked.version).toBe(0);
    expect(blocked.issues.some((i) => i.code === "empty-system" && i.severity === "error")).toBe(true);

    await expect(
      promptApi.PromptTemplateSave("no-such-template", JSON.stringify({ isActive: true, content: { system: "S" } })),
    ).rejects.toThrow();
  });

  it("Reset：幂等无副作用（resolve undefined）", async () => {
    await expect(promptApi.PromptTemplateReset("create-chapter")).resolves.toBeUndefined();
    await expect(promptApi.PromptTemplateReset("unknown-key")).resolves.toBeUndefined();
  });

  it("Preview：{{name}} 命中替换、缺失保留原文并进 warnings（含未解析变量样本）", async () => {
    const res = await promptApi.PromptTemplatePreview(
      JSON.stringify({ content: { system: "围绕 {{plot}} 创作第 {{chapter_num}} 章，约 {{word_count}} 字。" } }),
      JSON.stringify({ plot: "主角觉醒", chapter_num: "7", word_count: "" }),
    );
    expect(typeof res.systemPrompt).toBe("string");
    expect(res.systemPrompt).toContain("主角觉醒");
    expect(res.systemPrompt).toContain("第 7 章");
    // word_count 空串=未提供：保留原文 + 进 warnings
    expect(res.systemPrompt).toContain("{{word_count}}");
    expect(Array.isArray(res.warnings)).toBe(true);
    expect(res.warnings).toContain("word_count");
  });

  it("BundleExport：回合法 JSON 字符串（version=1 + templates + statistics）", async () => {
    const raw = await promptApi.PromptBundleExport();
    expect(typeof raw).toBe("string");
    const bundle = JSON.parse(raw) as { version: number; templates: unknown[]; statistics: { total: number } };
    expect(bundle.version).toBe(1);
    expect(Array.isArray(bundle.templates)).toBe(true);
    expect(bundle.templates.length).toBe(bundle.statistics.total);
  });

  it("BundleImport：回 applied/statistics/outcomes 三段；坏 JSON 抛错", async () => {
    const res = await promptApi.PromptBundleImport('{"version":1,"templates":[{"key":"a"},{"key":"b"}]}');
    expect(typeof res.applied).toBe("boolean");
    expect(typeof res.statistics.total).toBe("number");
    expect(res.outcomes.length).toBe(2);
    for (const o of res.outcomes) {
      expect(typeof o.key).toBe("string");
      expect(typeof o.action).toBe("string");
    }
    await expect(promptApi.PromptBundleImport("not-json")).rejects.toThrow();
  });
});