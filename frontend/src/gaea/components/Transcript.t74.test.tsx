// T7-4（v2.37.0）Transcript 轮级 memo：TurnBlock/ProcessCard 稳定化验证。
// 子组件（UserMessage/AssistantMessage/ToolCard）用 vi.mock 换成计数函数，
// 从而验证「同 props 重渲染不调用子组件、内容变化才调用」的 memo 语义。
import { describe, expect, it, vi } from "vitest";
import { render } from "@testing-library/react";
import { ProcessCard, TurnBlock } from "./Transcript";
import type { Item } from "../lib/store";

vi.mock("./Message", () => ({
  UserMessage: vi.fn(() => null),
  AssistantMessage: vi.fn(() => null),
}));
vi.mock("./ToolCard", () => ({
  ToolCard: vi.fn(() => null),
}));

import * as MessageModule from "./Message";
import * as ToolCardModule from "./ToolCard";

function u(id: string, text: string): Item { return { kind: "user", id, text }; }
function a(id: string, text: string): Item { return { kind: "assistant", id, text, reasoning: "", streaming: false }; }
function tool(id: string, name: string): Item { return { kind: "tool", id, name, args: "", readOnly: false, status: "done" }; }

const noop = () => {};
const turnElsRef = { current: new Map<number, HTMLElement>() };
const noSubcalls = new Map<string, never>();

describe("TurnBlock 轮级 memo（T7-4）", () => {
  it("props 引用不变时 rerender 不重渲染子消息；段内容变化才重渲染", () => {
    const userMock = vi.mocked(MessageModule.UserMessage);
    userMock.mockClear();
    const seg = { processItems: [] as Item[], outsideItems: [u("u1", "第一问")] };
    const props = {
      seg, running: false, isLast: false, turnNo: 0, openTurn: null,
      onToggleTurn: noop, onRewindTurn: noop, onCollapse: noop,
      dismissedErrors: new Set<string>(), onDismissError: noop,
      captureForId: () => undefined, turnElsRef,
    };
    const { rerender } = render(<TurnBlock {...props} />);
    expect(userMock).toHaveBeenCalledTimes(1);

    // 相同 props 重渲染：memo 跳过，UserMessage 不重渲染
    rerender(<TurnBlock {...props} />);
    expect(userMock).toHaveBeenCalledTimes(1);

    // 段内容变化：才重新渲染
    rerender(<TurnBlock {...props} seg={{ processItems: [], outsideItems: [u("u2", "第二问")] }} />);
    expect(userMock).toHaveBeenCalledTimes(2);
  });

  it("assistant 段：同一段 rerender 不重渲染 AssistantMessage", () => {
    const assistantMock = vi.mocked(MessageModule.AssistantMessage);
    assistantMock.mockClear();
    const seg = { processItems: [] as Item[], outsideItems: [a("a1", "回答内容")] };
    const props = {
      seg, running: false, isLast: true, turnNo: undefined, openTurn: null,
      onToggleTurn: noop, onRewindTurn: noop, onCollapse: noop,
      dismissedErrors: new Set<string>(), onDismissError: noop,
      captureForId: () => undefined, turnElsRef,
    };
    const { rerender } = render(<TurnBlock {...props} />);
    expect(assistantMock).toHaveBeenCalledTimes(1);

    rerender(<TurnBlock {...props} />);
    expect(assistantMock).toHaveBeenCalledTimes(1);

    rerender(<TurnBlock {...props} seg={{ processItems: [], outsideItems: [a("a2", "新回答")] }} />);
    expect(assistantMock).toHaveBeenCalledTimes(2);
  });

  it("TurnBlock 是 React.memo 组件", () => {
    expect((TurnBlock as unknown as { $$typeof?: unknown }).$$typeof).toBe(Symbol.for("react.memo"));
  });
});

describe("ProcessCard memo（T7-4）", () => {
  it("相同 props 不重渲染工具卡；items 变化才重渲染", () => {
    const toolMock = vi.mocked(ToolCardModule.ToolCard);
    toolMock.mockClear();
    const items = [tool("t1", "read_file")];
    const { rerender } = render(
      <ProcessCard items={items} toolCount={1} thoughtCount={0} small={false} subcallsByParent={noSubcalls} />,
    );
    expect(toolMock).toHaveBeenCalledTimes(1);

    rerender(
      <ProcessCard items={items} toolCount={1} thoughtCount={0} small={false} subcallsByParent={noSubcalls} />,
    );
    expect(toolMock).toHaveBeenCalledTimes(1);

    rerender(
      <ProcessCard items={[tool("t2", "write_file")]} toolCount={1} thoughtCount={0} small={false} subcallsByParent={noSubcalls} />,
    );
    expect(toolMock).toHaveBeenCalledTimes(2);
  });

  it("子调用收集：父工具卡拿到段内子调用列表", () => {
    const toolMock = vi.mocked(ToolCardModule.ToolCard);
    toolMock.mockClear();
    const parent: Item = { kind: "tool", id: "p1", name: "run_bash", args: "", readOnly: false, status: "done" };
    const child: Item = { kind: "tool", id: "c1", name: "read_file", args: "", readOnly: false, status: "done", parentId: "p1" };
    render(
      <ProcessCard items={[parent, child]} toolCount={2} thoughtCount={0} small={false} subcallsByParent={new Map([["p1", [child]]])} />,
    );
    const first = toolMock.mock.calls[0];
    expect(first?.[0]?.item?.id).toBe("p1");
    expect(first?.[0]?.subcalls).toHaveLength(1);
    expect(first?.[0]?.subcalls?.[0]?.id).toBe("c1");
  });

  it("ProcessCard 是 React.memo 组件", () => {
    expect((ProcessCard as unknown as { $$typeof?: unknown }).$$typeof).toBe(Symbol.for("react.memo"));
  });
});
