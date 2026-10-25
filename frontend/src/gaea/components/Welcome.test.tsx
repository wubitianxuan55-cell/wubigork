import { describe, expect, it, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import type { ReactElement } from "react";
import { LocaleProvider } from "../lib/i18n";
import { Welcome, FALLBACK_TEMPLATES, resolveTemplates, loadTemplates, resetTemplatesCacheForTest } from "./Welcome";

function wrap(ui: ReactElement) {
  // Welcome 走 useT：钉住 zh 让「核心能力」等中文断言继续成立
  localStorage.setItem("gaea-lang", "zh");
  return <LocaleProvider>{ui}</LocaleProvider>;
}

beforeEach(() => {
  resetTemplatesCacheForTest();
});

describe("Welcome 欢迎页", () => {
  it("任务模板空库/失败回退内置模板", () => {
    expect(resolveTemplates(null)).toHaveLength(FALLBACK_TEMPLATES.length);
    expect(resolveTemplates([])).toHaveLength(FALLBACK_TEMPLATES.length);
    const remote = [{ name: "custom", title: "自定义", description: "d", prompt: "p" }];
    expect(resolveTemplates(remote)).toEqual(remote);
  });

  it("loadTemplates 返回模板且缓存命中后结果稳定", async () => {
    const first = await loadTemplates();
    const second = await loadTemplates();
    expect(first.length).toBeGreaterThan(0);
    expect(second).toEqual(first);
  });

  it("渲染核心能力与任务模板区", async () => {
    render(wrap(<Welcome onPrompt={() => {}} />));
    expect(screen.getByText("核心能力")).toBeTruthy();
    expect(await screen.findByText("周报")).toBeTruthy();
  });

  it("点击任务模板把提示词填入输入通道", async () => {
    const prompts: string[] = [];
    render(wrap(<Welcome onPrompt={(p) => prompts.push(p)} />));
    fireEvent.click(await screen.findByText("周报"));
    expect(prompts[0]).toContain("周报");
  });

  // 2026-10 工具化（§6.4 空态工作台）：工具 tile 开 ToolSheet 不是页面；
  // onOpenTool 缺省整节不渲染（工具节是办公宿主可选接线，其他消费方零感知）。
  it("未接 onOpenTool：工具节不渲染（能力卡照常）", async () => {
    render(wrap(<Welcome onPrompt={() => {}} />));
    expect(screen.queryByTestId("welcome-tools")).toBeNull();
    expect(screen.getByText("核心能力")).toBeTruthy();
  });

  it("工具节两 tile：点 CAD/进度派发 onOpenTool 对应 kind（不触发 onPrompt）", async () => {
    const opened: string[] = [];
    const prompts: string[] = [];
    render(wrap(<Welcome onPrompt={(p) => prompts.push(p)} onOpenTool={(tool) => opened.push(tool)} />));
    expect(await screen.findByTestId("welcome-tools")).toBeTruthy();
    fireEvent.click(screen.getByTestId("welcome-tool-cad"));
    fireEvent.click(screen.getByTestId("welcome-tool-schedule"));
    expect(opened).toEqual(["cad", "schedule"]);
    expect(prompts).toEqual([]);
  });
});
