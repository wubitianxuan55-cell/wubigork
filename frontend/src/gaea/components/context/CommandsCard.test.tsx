// 刀4 组件行为测试：命令执行卡（状态筛选/程序回填/搜索）+ 浏览器 DNA 视图
// + 助手 kind 筛选 + skill 节点消费。mock 桥同 ContextView.test 模式。
import { fireEvent, render, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { ReactElement } from "react";
import { LocaleProvider } from "../../lib/i18n";
import type { ContextCommandRecord, ContextSurfaceNode } from "../../lib/types/context";

vi.setConfig({ testTimeout: 20_000 });

const renderT = (ui: ReactElement) => {
  localStorage.setItem("gaea-lang", "zh");
  return render(<LocaleProvider>{ui}</LocaleProvider>);
};

import { CommandsCard } from "./CommandsCard";
import { ContextBrowserTree } from "./inspector";

const CMDS: ContextCommandRecord[] = [
  { seq: 9, resultSeq: 10, time: 1750000009, tool: "bash", cmd: "go build ./...", turn: 1, step: 2, start: 1750000009, end: 1750000013, status: "completed", exit: 0, outChars: 1240, derived: true },
  { seq: 14, resultSeq: 15, time: 1750000014, tool: "bash", cmd: "npm run dev", turn: 1, step: 3, start: 1750000014, end: 1750000015, status: "completed", background: true, outChars: 88, derived: true },
  { seq: 18, resultSeq: 19, time: 1750000018, tool: "bash", cmd: "go vet ./internal/...", turn: 2, step: 4, start: 1750000018, end: 1750000021, status: "failed", err: true, exit: 1, outChars: 640, derived: true },
  { seq: 22, time: 1750000022, tool: "bash", cmd: "sleep 999", turn: 2, step: 5, start: 1750000022, status: "killed" },
];

describe("CommandsCard 命令执行卡", () => {
  it("渲染生命周期行（三态点/退出码/后台徽标）", () => {
    renderT(<CommandsCard commands={CMDS} stats={{ runs: 4, failed: 1, ms: 8000, bg: 1 }} />);
    expect(screen.getByTestId("ctx-commands")).toBeTruthy();
    const list = within(screen.getByTestId("ctx-commands-list"));
    expect(list.getByText("go build ./...")).toBeTruthy();
    // JSX 文本节点拆分：exit 与码值分属两个 text node，用 textContent 匹配
    const exitEl = screen.getByText((_, el) => el?.textContent === "exit 1" && el.tagName === "SPAN");
    expect(exitEl).toBeTruthy();
    expect(screen.getByTestId("ctx-cmd-dot-killed")).toBeTruthy();
    expect(screen.getByText("bg")).toBeTruthy();
    // 程序分布（go ×2 / npm ×1）
    expect(screen.getByTitle("go ×2")).toBeTruthy();
  });

  it("状态筛选 failed 只留失败行", () => {
    renderT(<CommandsCard commands={CMDS} />);
    fireEvent.click(screen.getByTestId("ctx-cmd-status-failed"));
    const list = within(screen.getByTestId("ctx-commands-list"));
    expect(list.getByText("go vet ./internal/...")).toBeTruthy();
    expect(list.queryByText("go build ./...")).toBeNull();
  });

  it("程序分布 chip 回填过滤（go 只留两条）", () => {
    renderT(<CommandsCard commands={CMDS} />);
    fireEvent.click(screen.getByTitle("go ×2"));
    const list = within(screen.getByTestId("ctx-commands-list"));
    expect(list.getByText("go build ./...")).toBeTruthy();
    expect(list.getByText("go vet ./internal/...")).toBeTruthy();
    expect(list.queryByText("npm run dev")).toBeNull();
  });

  it("搜索框过滤", () => {
    renderT(<CommandsCard commands={CMDS} />);
    fireEvent.change(screen.getByTestId("ctx-cmd-search"), { target: { value: "npm" } });
    const list = within(screen.getByTestId("ctx-commands-list"));
    expect(list.getByText("npm run dev")).toBeTruthy();
    expect(list.queryByText("go build ./...")).toBeNull();
  });
});

// ─── inspector：DNA 视图 + kind 筛选 + skill 行 ───

const NODES: ContextSurfaceNode[] = [
  { seq: 1, cat: "system", tokens: 2100, text: "system prompt here" },
  { seq: 2, cat: "tools", tokens: 10400, text: "read_file · bash" },
  { seq: 3, cat: "user", tokens: 120, text: "帮我看看这段代码" },
  { seq: 4, cat: "assistant", tokens: 900, text: "thinking about it", reasoning: 600, answerTokens: 300 },
  { seq: 5, cat: "assistant", tokens: 400, text: "run tool", callTokens: 400 },
  { seq: 6, cat: "tool", tokens: 2200, text: "package main", tool: "read_file" },
  { seq: 7, cat: "skill", tokens: 1500, text: "skill body", tool: "run_skill", name: "pdf-toolkit" },
];

function Browser(props: Partial<Parameters<typeof ContextBrowserTree>[0]>) {
  return (
    <ContextBrowserTree
      nodes={NODES}
      archive={[]}
      {...props}
    />
  );
}

describe("ContextBrowserTree 刀4 增强", () => {
  it("DNA 视图：顺序编号平铺 + 系统提示词/工具定义收尾两行", () => {
    renderT(<Browser />);
    fireEvent.click(screen.getByTestId("ctx-browser-view-dna"));
    const dna = screen.getByTestId("ctx-browser-dna");
    expect(dna).toBeTruthy();
    // 编号行（#1..#5：user/assistant×2/tool/skill——system/tools 不进编号）
    expect(dna.textContent).toContain("#1");
    expect(dna.textContent).toContain("#5");
    expect(dna.textContent).not.toContain("#6");
    // 收尾两行
    expect(screen.getByText("系统提示词（最新头）")).toBeTruthy();
    expect(screen.getByText("工具定义（最新头）")).toBeTruthy();
  });

  it("助手 kind 筛选：思考 chip 只留带 reasoning 的行；正文/工具计数正确", () => {
    renderT(<Browser />);
    fireEvent.click(screen.getByTestId("ctx-browser-view-tree"));
    // 展开助手组
    fireEvent.click(screen.getByText("助手消息"));
    const kinds = screen.getByTestId("ctx-assistant-kinds");
    expect(kinds.textContent).toContain("思考 1");
    expect(kinds.textContent).toContain("工具 1");
    expect(kinds.textContent).toContain("正文 1");
    fireEvent.click(screen.getByText(/思考 1/));
    // 过滤后只显示带 reasoning 的助手行
    expect(screen.getByText("thinking about it")).toBeTruthy();
    expect(screen.queryByText("run tool")).toBeNull();
  });

  it("skill 行：技能名 chip 展示 + 可展开详情按钮", () => {
    renderT(<Browser />);
    fireEvent.click(screen.getByTestId("ctx-browser-view-tree"));
    fireEvent.click(screen.getByText("技能内容"));
    expect(screen.getByText("pdf-toolkit")).toBeTruthy();
    expect(screen.getByText("run_skill")).toBeTruthy();
    // DETAILABLE 含 skill：行上有详情按钮
    expect(screen.getAllByTestId("ctx-node-detail-btn").length).toBeGreaterThan(0);
  });
});
