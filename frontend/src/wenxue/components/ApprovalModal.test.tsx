import { beforeAll, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import type { ReactElement } from "react";
import { LocaleProvider } from "../lib/i18n";
import type { WireApproval } from "../lib/types";
import { ApprovalModal } from "./ApprovalModal";

beforeAll(() => {
  localStorage.setItem("gaea-lang", "zh");
});

function wrap(ui: ReactElement) {
  return <LocaleProvider>{ui}</LocaleProvider>;
}

describe("文学审批卡 · codex 蒸馏 T3/T11", () => {
  it("拒绝理由随 deny 回传；放行不带理由", () => {
    const approval: WireApproval = { id: "wx-a1", tool: "write_file", subject: "覆盖 章节/001.md" };
    const onAnswer = vi.fn();
    render(wrap(<ApprovalModal approval={approval} onAnswer={onAnswer} />));

    const input = screen.getByTestId("approval-deny-reason") as HTMLInputElement;
    fireEvent.change(input, { target: { value: "先保留原章节" } });
    fireEvent.click(screen.getByText("拒绝"));
    expect(onAnswer).toHaveBeenCalledWith("deny", "先保留原章节");

    onAnswer.mockClear();
    fireEvent.click(screen.getByText("允许一次"));
    expect(onAnswer).toHaveBeenCalledWith("allow_once", undefined);
  });

  it("Guardian 追认重放卡显示人工裁决说明", () => {
    const approval: WireApproval = {
      id: "wx-a2",
      tool: "write_file",
      subject: "写入 候选/第一章.md",
      source: "guardian",
    };
    render(wrap(<ApprovalModal approval={approval} onAnswer={() => {}} />));
    expect(screen.getByText(/追认重放卡/)).toBeTruthy();
  });

  it("硬确认卡按 3 与办公壳走同一终止语义", () => {
    const approval: WireApproval = { id: "wx-a3", tool: "remember", subject: "写入长期记忆" };
    const onAnswer = vi.fn();
    render(wrap(<ApprovalModal approval={approval} onAnswer={onAnswer} />));

    fireEvent.keyDown(document, { key: "3" });

    expect(onAnswer).toHaveBeenCalledWith("abort", undefined);
  });
});
