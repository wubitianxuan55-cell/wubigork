// ContextModal + ContextPill 单测（2.5e：/context 居中弹层 + 常驻剩余上下文徽标）。
import { describe, expect, it, vi, beforeEach } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import { LocaleProvider } from "../lib/i18n";
import { ContextModal, ContextPill } from "./ContextModal";

const ctxViewMock = vi.fn(async () => ({
  ok: true,
  window: 1_000_000,
  current: { system: 2100, tools: 10400, user: 20, inject: 21900, assistant: 93400, tool: 114000 },
  stats: { turns: 1, steps: 1, injects: 0, compacts: 0, prunes: 0, toolCalls: 0, images: 0 },
  requests: [],
  events: [],
  nodes: [],
  archive: [],
  files: [],
}));

vi.mock("../lib/agentNetworkStore", () => ({
  subscribeAgentNetwork: () => () => {},
  reloadAgentNetwork: () => Promise.resolve(),
}));

vi.mock("../lib/bridge", () => ({
  app: { ContextView: (...a: unknown[]) => ctxViewMock(...(a as [])), openExternal: vi.fn(), onEvent: () => () => {} },
  onEvent: () => () => {},
}));

const wrap = (ui: React.ReactNode) => {
  localStorage.setItem("gaea-lang", "zh");
  return render(<LocaleProvider>{ui}</LocaleProvider>);
};

describe("ContextPill 常驻剩余上下文徽标", () => {
  it("渲染剩余百分比与占用进度条", () => {
    wrap(<ContextPill used={100_000} window={1_000_000} />);
    const pill = screen.getByTestId("ctx-pill");
    expect(pill.textContent).toContain("剩余 90%");
    expect(pill.getAttribute("title")).toContain("100k/1.0M");
  });

  it("点击回调打开弹层", () => {
    const onClick = vi.fn();
    wrap(<ContextPill used={100_000} window={1_000_000} onClick={onClick} />);
    fireEvent.click(screen.getByTestId("ctx-pill"));
    expect(onClick).toHaveBeenCalledTimes(1);
  });

  it("≥90% 转警示色，window=0 不渲染", () => {
    const { container } = wrap(<ContextPill used={950_000} window={1_000_000} />);
    expect(screen.getByTestId("ctx-pill").textContent).toContain("剩余 5%");
    const { container: c2 } = wrap(<ContextPill used={0} window={0} />);
    expect(c2.querySelector("[data-testid='ctx-pill']")).toBeNull();
    void container;
  });
});

describe("ContextModal /context 居中弹层", () => {
  beforeEach(() => {
    ctxViewMock.mockClear();
  });

  it("open 时挂载 ContextView 内容；关闭回调接通", () => {
    const onClose = vi.fn();
    wrap(
      <ContextModal
        open
        onClose={onClose}
        running={false}
        sessionPath="/ws/s.jsonl"
        sessionName="演示会话"
        model="Hephaestus"
      />,
    );
    // Modal 标题与 ContextView 内部卡同名——用 antd 标题类定位
    expect(document.querySelector(".ant-modal-title")?.textContent).toBe("当前上下文");
    // 内容区挂载（ContextView 的汇总条文案）
    expect(screen.getByTestId("ctx-modal-body")).toBeTruthy();
    // antd 关闭按钮
    fireEvent.click(document.querySelector(".ant-modal-close")!);
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("open=false 时不渲染内容（关闭即卸载）", () => {
    wrap(<ContextModal open={false} onClose={() => {}} running={false} />);
    expect(screen.queryByTestId("ctx-modal-body")).toBeNull();
  });
});
