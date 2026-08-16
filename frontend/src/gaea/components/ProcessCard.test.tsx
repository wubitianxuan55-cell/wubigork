import { describe, expect, it } from "vitest";
import { render } from "@testing-library/react";
import { ProcessCard } from "./Transcript";
import { deriveProcessStatus } from "../lib/processStatus";
import { LocaleProvider } from "../lib/i18n";
import type { Item } from "../lib/store";

const noSubcalls = new Map<string, never>();

const wrap = (node: React.ReactNode) => <LocaleProvider>{node}</LocaleProvider>;

describe("ProcessCard 小过程卡 / 大过程卡初始状态", () => {
  it("分段小过程卡（small）默认折叠；大过程卡（含文本合并卡）默认展开", () => {
    const items: Item[] = [
      { kind: "assistant", id: "a1", text: "", reasoning: "先分析需求", streaming: false },
    ];
    const smallView = render(
      <ProcessCard items={items} toolCount={0} thoughtCount={1} small subcallsByParent={noSubcalls} />,
    );
    // 运行中的分段小过程卡：默认折叠
    const smallHeader = smallView.container.querySelectorAll("button[aria-expanded]")[0];
    expect(smallHeader?.getAttribute("aria-expanded")).toBe("false");
    smallView.unmount();

    // 大过程卡：整轮结束后以全新实例挂载（small=false），默认展开
    const bigView = render(
      <ProcessCard items={items} toolCount={0} thoughtCount={1} small={false} subcallsByParent={noSubcalls} />,
    );
    const bigHeader = bigView.container.querySelectorAll("button[aria-expanded]")[0];
    expect(bigHeader?.getAttribute("aria-expanded")).toBe("true");
  });
});

describe("deriveProcessStatus 状态四态推导", () => {
  const tool = (status: "running" | "done" | "error" | "stopped", parentId?: string): Item =>
    ({ kind: "tool", id: `t${status}${Math.random()}`, name: "write_file", args: "", readOnly: false, status, parentId }) as Item;

  it("running 优先于一切", () => {
    expect(deriveProcessStatus([tool("error"), tool("done")], true)).toBe("running");
  });

  it("有 error 工具 → error（即便同时有 stopped）", () => {
    expect(deriveProcessStatus([tool("error"), tool("stopped")], false)).toBe("error");
  });

  it("无 error 但有 stopped → stopped", () => {
    expect(deriveProcessStatus([tool("stopped"), tool("done")], false)).toBe("stopped");
  });

  it("全部完成 → done", () => {
    expect(deriveProcessStatus([tool("done"), tool("done")], false)).toBe("done");
  });

  it("无工具（纯思考/正文）→ idle；子调用不参与推导", () => {
    expect(deriveProcessStatus([{ kind: "assistant", id: "a", text: "", reasoning: "", streaming: false }], false)).toBe("idle");
    expect(deriveProcessStatus([tool("error", "parent1")], false)).toBe("idle");
  });
});

describe("ProcessCard 状态徽标（色+图标+文字三重传达）", () => {
  const tool = (status: "running" | "done" | "error" | "stopped"): Item =>
    ({ kind: "tool", id: `t${status}`, name: "write_file", args: "", readOnly: false, status }) as Item;

  it("running 显示「处理中」徽标", () => {
    const view = render(
      wrap(<ProcessCard items={[tool("running")]} toolCount={1} thoughtCount={0} small running subcallsByParent={noSubcalls} />),
    );
    expect(view.container.textContent).toContain("处理中");
  });

  it("error 工具 → 「有错误」徽标 + 语义图标", () => {
    const view = render(
      wrap(<ProcessCard items={[tool("error")]} toolCount={1} thoughtCount={0} small={false} subcallsByParent={noSubcalls} />),
    );
    expect(view.container.textContent).toContain("有错误");
  });

  it("stopped 工具 → 「已中断」徽标", () => {
    const view = render(
      wrap(<ProcessCard items={[tool("stopped")]} toolCount={1} thoughtCount={0} small={false} subcallsByParent={noSubcalls} />),
    );
    expect(view.container.textContent).toContain("已中断");
  });

  it("全部 done → 「完成」徽标", () => {
    const view = render(
      wrap(<ProcessCard items={[tool("done")]} toolCount={1} thoughtCount={0} small={false} subcallsByParent={noSubcalls} />),
    );
    expect(view.container.textContent).toContain("完成");
  });
});
