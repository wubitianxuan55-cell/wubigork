import { describe, expect, it, vi, beforeAll } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import type { ReactElement } from "react";
import { LocaleProvider } from "../lib/i18n";
import type { WireApproval } from "../lib/types";
import { ApprovalModal } from "./ApprovalModal";

// jsdom 的 navigator.language 是 en-US；显式钉死 zh 断言壳层主字典。
beforeAll(() => {
  localStorage.setItem("gaea-lang", "zh");
});

function wrap(ui: ReactElement) {
  return <LocaleProvider>{ui}</LocaleProvider>;
}

describe("ApprovalModal 审批卡", () => {
  it("常规工具审批保持原形态：工具标题 + subject 原文，不渲染 reason 块", () => {
    const approval: WireApproval = { id: "a1", tool: "bash", subject: "go build ./..." };
    render(wrap(<ApprovalModal approval={approval} onAnswer={() => {}} />));
    expect(screen.getByText("允许这次工具调用吗？")).toBeTruthy();
    expect(screen.getByText("go build ./...")).toBeTruthy();
    expect(screen.queryByText("申请理由（模型提供）")).toBeNull();
  });

  it("权限升级申请：显示申请规则串与 reason 原文（理由必须可见）", () => {
    const approval: WireApproval = {
      id: "a2",
      tool: "bash",
      subject: "go build*",
      request: true,
      reason: "需要运行 go build 验证改动可编译",
    };
    render(wrap(<ApprovalModal approval={approval} onAnswer={() => {}} />));
    expect(screen.getByText("模型申请放开权限")).toBeTruthy();
    // 规则以 Tool(subject) 形式完整展示
    expect(screen.getByText("申请规则")).toBeTruthy();
    expect(screen.getByText("bash(go build*)")).toBeTruthy();
    // reason 原文可见（禁止不经展示直接采信）
    expect(screen.getByText("申请理由（模型提供）")).toBeTruthy();
    expect(screen.getByText("需要运行 go build 验证改动可编译")).toBeTruthy();
  });

  it("申请缺 reason 时明确标注未提供，而不是静默渲染空块", () => {
    const approval: WireApproval = { id: "a3", tool: "web_fetch", subject: "", request: true };
    render(wrap(<ApprovalModal approval={approval} onAnswer={() => {}} />));
    expect(screen.getByText("（模型未提供理由）")).toBeTruthy();
    expect(screen.getByText("web_fetch")).toBeTruthy();
  });

  it("决策按钮经既有 GaeaApprove 通道回传五决策串", () => {
    const approval: WireApproval = {
      id: "a4",
      tool: "bash",
      subject: "go test*",
      request: true,
      reason: "跑测试",
    };
    const onAnswer = vi.fn();
    render(wrap(<ApprovalModal approval={approval} onAnswer={onAnswer} />));
    fireEvent.click(screen.getByText("拒绝"));
    fireEvent.click(screen.getByText("允许一次"));
    fireEvent.click(screen.getByText("本会话内允许"));
    fireEvent.click(screen.getByText("始终允许"));
    fireEvent.click(screen.getByText("拒绝并停止本轮"));
    expect(onAnswer.mock.calls.map((c) => c[0])).toEqual([
      "deny",
      "allow_once",
      "allow_session",
      "persist_allow",
      "abort",
    ]);
  });
});
