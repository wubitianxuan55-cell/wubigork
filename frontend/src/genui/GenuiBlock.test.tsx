import { act, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { GenuiBlock } from "./GenuiBlock";
import { GenuiActionProvider, type GenuiActionHandler } from "./GenuiActionContext";
import type { GenuiSpec } from "./spec";

function statSpec(): GenuiSpec {
  return {
    title: "测试看板",
    items: [
      { type: "stat", label: "营收", value: "¥128k", delta: "+12%" },
      { type: "progress", label: "进度", value: 66 },
    ],
  };
}

function gradingSpec(): GenuiSpec {
  return {
    items: [
      {
        type: "radio",
        label: "第 1 题",
        options: ["北京", "上海"],
        group: "q1",
        answer: "北京",
        explanation: "首都是北京",
      },
      { type: "submit", label: "交卷", groups: ["q1"], resetAction: "redo" },
    ],
  };
}

afterEach(() => {
  vi.useRealTimers();
  window.localStorage.clear();
});

describe("GenuiBlock 展示", () => {
  it("渲染 banner/stat/progress", () => {
    render(<GenuiBlock spec={statSpec()} />);
    expect(screen.getByText("测试看板")).toBeTruthy();
    expect(screen.getByText("营收")).toBeTruthy();
    expect(screen.getByText("¥128k")).toBeTruthy();
    expect(screen.getByRole("progressbar").getAttribute("aria-valuenow")).toBe("66");
  });
});

describe("GenuiBlock action 诚实交互", () => {
  it("无 provider → 带 action 按钮禁用；有 provider → 点击触发且 300ms 防抖", () => {
    const spec: GenuiSpec = { items: [{ type: "button", label: "刷新", action: "refresh" }] };
    const { unmount } = render(<GenuiBlock spec={spec} />);
    expect((screen.getByRole("button") as HTMLButtonElement).disabled).toBe(true);
    unmount();

    vi.useFakeTimers();
    const handler = vi.fn<GenuiActionHandler>();
    render(
      <GenuiActionProvider onAction={handler}>
        <GenuiBlock spec={spec} />
      </GenuiActionProvider>,
    );
    const btn = screen.getByRole("button") as HTMLButtonElement;
    expect(btn.disabled).toBe(false);
    fireEvent.click(btn);
    fireEvent.click(btn);
    expect(handler).not.toHaveBeenCalled();
    act(() => {
      vi.advanceTimersByTime(320);
    });
    expect(handler).toHaveBeenCalledTimes(1);
    expect(handler).toHaveBeenCalledWith("refresh", {});
  });
});

describe("GenuiBlock 本地判卷", () => {
  it("radio 聚合 + submit 本地判分与重做", () => {
    vi.useFakeTimers();
    render(<GenuiBlock spec={gradingSpec()} />);
    fireEvent.click(screen.getByLabelText("上海"));
    fireEvent.click(screen.getByRole("button", { name: "交卷" }));
    expect(screen.getByText("得分 0/1")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "重新作答" }));
    fireEvent.click(screen.getByLabelText("北京"));
    fireEvent.click(screen.getByRole("button", { name: "交卷" }));
    expect(screen.getByText("得分 1/1")).toBeTruthy();
  });
});

describe("GenuiBlock quiz 点选即判", () => {
  it("答错显示错误并可重试", () => {
    const spec: GenuiSpec = {
      items: [
        {
          type: "quiz",
          question: "2+2=?",
          options: [
            { label: "3", feedback: "再想想" },
            { label: "4", correct: true },
          ],
          explanation: "2+2=4",
        },
      ],
    };
    render(<GenuiBlock spec={spec} />);
    fireEvent.click(screen.getByRole("button", { name: "3" }));
    expect(screen.getByText("回答错误")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "重试" }));
    fireEvent.click(screen.getByRole("button", { name: "4" }));
    expect(screen.getByText("回答正确")).toBeTruthy();
  });
});

describe("GenuiBlock 持久化", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  it("同 stateKey 重渲染恢复 radio 选择", () => {
    const key = "genui:office:s1:m1:fp1";
    const { unmount } = render(<GenuiBlock spec={gradingSpec()} stateKey={key} />);
    fireEvent.click(screen.getByLabelText("北京"));
    act(() => {
      vi.advanceTimersByTime(400);
    });
    unmount();

    render(<GenuiBlock spec={gradingSpec()} stateKey={key} />);
    expect((screen.getByLabelText("北京") as HTMLInputElement).checked).toBe(true);
  });
});
