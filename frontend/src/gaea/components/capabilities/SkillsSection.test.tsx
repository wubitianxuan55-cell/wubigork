// 7.2-2 判据②：技能行持久调用统计（usage prop 有数据渲染、无数据零渲染）。
import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { SkillsSection } from "./SkillsSection";

vi.mock("../../lib/i18n", () => ({
  useT: () => (key: string, params?: Record<string, unknown>) => {
    const dict: Record<string, string> = {
      "caps.skillUsage": "{n} 次",
      "caps.skillSuccessRate": "成功率 {p}%",
      "caps.searchSkills": "搜索技能…",
      "caps.noSkills": "暂无技能",
      "caps.noSkillMatches": "无匹配技能",
      "caps.skillScopeBuiltin": "内置",
      "caps.skillScopeProject": "项目",
      "caps.skillScopeCustom": "自定义",
      "caps.skillScopeGlobal": "全局",
      "caps.subagent": "子代理",
      "common.expand": "展开",
      "common.collapse": "收起",
    };
    let s = dict[key] ?? key;
    for (const [k, v] of Object.entries(params ?? {})) s = s.replaceAll(`{${k}}`, String(v));
    return s;
  },
}));

const skills = [
  { name: "cost-compose", description: "AI 组价流水线", scope: "project", runAs: "" },
  { name: "explore", description: "探查目录结构", scope: "builtin", runAs: "subagent" },
] as never[];

function setup(usage?: Record<string, { calls: number; ok: number }>) {
  return render(
    <SkillsSection
      skills={skills}
      counts={{}}
      usage={usage}
      expanded={new Set()}
      onToggle={() => {}}
    />,
  );
}

describe("SkillsSection 持久调用统计", () => {
  it("usage 有数据时渲染「N 次 · 成功率 X%」", () => {
    setup({ "cost-compose": { calls: 7, ok: 6 } });
    const stat = screen.getByTestId("skill-usage-stat");
    expect(stat.textContent).toContain("7 次");
    expect(stat.textContent).toContain("成功率 86%");
  });

  it("usage 缺省/零调用不渲染统计行", () => {
    const { rerender } = setup();
    expect(screen.queryByTestId("skill-usage-stat")).toBeNull();
    rerender(
      <SkillsSection
        skills={skills}
        counts={{}}
        usage={{ explore: { calls: 0, ok: 0 } }}
        expanded={new Set()}
        onToggle={() => {}}
      />,
    );
    expect(screen.queryByTestId("skill-usage-stat")).toBeNull();
  });
});
