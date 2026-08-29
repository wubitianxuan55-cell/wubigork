import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import type { ReactElement } from "react";
import { LocaleProvider } from "../lib/i18n";
import type { WireAsk } from "../lib/types";
import { AskCard } from "./AskCard";

function wrap(ui: ReactElement) {
  return <LocaleProvider>{ui}</LocaleProvider>;
}

describe("AskCard 提问卡", () => {
  it("渲染 Markdown 问题与选项", () => {
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
    expect(screen.getByText("Word")).toBeTruthy();
    expect(screen.getByText("PDF")).toBeTruthy();
  });
});
