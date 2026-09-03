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
});
