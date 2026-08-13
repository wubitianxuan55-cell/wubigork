import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import type { ReactElement } from "react";
import { LocaleProvider } from "../lib/i18n";
import type { WireAsk } from "../lib/types";
import { AskCard } from "./AskCard";

function wrap(ui: ReactElement) {
  return <LocaleProvider>{ui}</LocaleProvider>;
}

const planAsk: WireAsk = {
  id: "plan-1",
  questions: [
    {
      id: "plan",
      header: "开工计划",
      prompt: "**任务理解**：整理成本测算",
      options: [
        { label: "确认执行", description: "按此计划开始干活" },
        { label: "先调整", description: "取消本轮，补充说明后重发" },
      ],
    },
  ],
  plan: {
    goal: "整理成本测算",
    steps: [
      {
        title: "读取成本数据",
        detail: "从工作区 docs 读取成本明细",
        resources: ["docs/成本.xlsx"],
        tools: ["read_file", "xlsx_edit"],
        deliverable: "成本明细表",
      },
      { title: "生成周报", tools: ["write_file"] },
    ],
    questions: ["统计口径按本月还是本季？"],
  },
};

describe("AskCard 开工计划卡片", () => {
  it("有 plan 时渲染结构化计划：目标/步骤/资料/工具/产出物/待确认", () => {
    render(wrap(<AskCard ask={planAsk} onAnswer={() => {}} onDismiss={() => {}} />));
    expect(screen.getByText("整理成本测算")).toBeTruthy();
    expect(screen.getByText("读取成本数据")).toBeTruthy();
    expect(screen.getByText("生成周报")).toBeTruthy();
    expect(screen.getByText("docs/成本.xlsx")).toBeTruthy();
    expect(screen.getByText("read_file")).toBeTruthy();
    expect(screen.getByText("xlsx_edit")).toBeTruthy();
    expect(screen.getByTitle("Deliverable: 成本明细表")).toBeTruthy();
    expect(screen.getByText(/统计口径按本月还是本季？/)).toBeTruthy();
    // 操作选项仍然可用
    expect(screen.getByText("确认执行")).toBeTruthy();
    expect(screen.getByText("先调整")).toBeTruthy();
  });

  it("无 plan 时回退纯文本 Markdown 问题", () => {
    const plainAsk: WireAsk = {
      id: "ask-1",
      questions: [
        {
          id: "q1",
          header: "澄清",
          prompt: "请确认目标格式",
          options: [{ label: "Word" }, { label: "PDF" }],
        },
      ],
    };
    render(wrap(<AskCard ask={plainAsk} onAnswer={() => {}} onDismiss={() => {}} />));
    expect(screen.getByText("请确认目标格式")).toBeTruthy();
  });
});
