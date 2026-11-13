import { describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
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

  it("选择答案后更新进度、启用提交并通过既有通道回传", () => {
    const onAnswer = vi.fn();
    const ask: WireAsk = {
      id: "ask-2",
      questions: [
        {
          id: "q1",
          header: "交付格式",
          prompt: "请选择输出格式",
          options: [{ label: "Word 文档" }, { label: "PDF" }],
        },
      ],
    };
    render(wrap(<AskCard ask={ask} onAnswer={onAnswer} onDismiss={() => {}} />));

    const option = screen.getByRole("button", { name: /Word 文档/ });
    expect(option.getAttribute("aria-pressed")).toBe("false");
    fireEvent.click(option);
    expect(option.getAttribute("aria-pressed")).toBe("true");
    expect(screen.getByRole("progressbar").getAttribute("aria-valuenow")).toBe("1");

    const submit = screen.getByRole("button", { name: "提交" }) as HTMLButtonElement;
    expect(submit.disabled).toBe(false);
    fireEvent.click(submit);
    expect(onAnswer).toHaveBeenCalledWith("ask-2", [{ questionId: "q1", selected: ["Word 文档"] }]);
  });
});
