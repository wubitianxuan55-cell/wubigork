import { describe, expect, it } from "vitest";
import { render } from "@testing-library/react";
import { ProcessCard, buildSegments } from "./Transcript";
import { deriveProcessStatus } from "../lib/processStatus";
import { LocaleProvider } from "../lib/i18n";
import type { Item } from "../lib/store";

const noSubcalls = new Map<string, never>();

const wrap = (node: React.ReactNode) => {
  // ProcessCard 标签走 t()；显式钉住 zh，断言用中文文案（不依赖其他文件的 localStorage 残留）
  localStorage.setItem("gaea-lang", "zh");
  return <LocaleProvider>{node}</LocaleProvider>;
};

describe("ProcessCard 小过程卡 / 展开态初始状态", () => {
  // 含已完成工具的段（非 bare）：卡头 button[aria-expanded] 即过程卡头
  const toolDone = (): Item =>
    ({ kind: "tool", id: `t${Math.random()}`, name: "write_file", args: "", readOnly: false, status: "done" } as Item);

  it("完成态过程卡默认折叠（small/非 small 一致；2026-09-18 Codex 对齐：完成即收起）", () => {
    const items: Item[] = [
      { kind: "assistant", id: "a1", text: "", reasoning: "先分析需求", streaming: false },
      toolDone(),
    ];
    const smallView = render(
      wrap(
<ProcessCard items={items} toolCount={1} thoughtCount={1} small subcallsByParent={noSubcalls} />),
    );
    const smallHeader = smallView.container.querySelectorAll("button[aria-expanded]")[0];
    expect(smallHeader?.getAttribute("aria-expanded")).toBe("false");
    smallView.unmount();

    // 旧「展开态默认展开」已废：非 small 完成态同样收起为一行
    const bigView = render(
      wrap(
<ProcessCard items={items} toolCount={1} thoughtCount={1} small={false} subcallsByParent={noSubcalls} />),
    );
    const bigHeader = bigView.container.querySelectorAll("button[aria-expanded]")[0];
    expect(bigHeader?.getAttribute("aria-expanded")).toBe("false");
  });

  it("运行中过程卡自动展开（实时活动可见）", () => {
    const items: Item[] = [
      { kind: "assistant", id: "a1", text: "", reasoning: "先分析需求", streaming: false },
      { kind: "tool", id: "t-run", name: "write_file", args: "", readOnly: false, status: "running" } as Item,
    ];
    const view = render(
      wrap(
<ProcessCard items={items} toolCount={1} thoughtCount={1} small running subcallsByParent={noSubcalls} />),
    );
    const header = view.container.querySelectorAll("button[aria-expanded]")[0];
    expect(header?.getAttribute("aria-expanded")).toBe("true");
  });

  it("纯思考段走 bare 模式：无过程卡容器，只渲染裸思考行", () => {
    const items: Item[] = [
      { kind: "assistant", id: "a1", text: "", reasoning: "先分析需求", streaming: false },
    ];
    const view = render(
      wrap(
<ProcessCard items={items} toolCount={0} thoughtCount={1} small={false} subcallsByParent={noSubcalls} />),
    );
    expect(view.container.querySelector("[data-bare-process]")).not.toBeNull();
    // 唯一的折叠头是思考行自己的 .reasoning__head，无卡头
    const heads = view.container.querySelectorAll("button[aria-expanded]");
    expect(heads.length).toBe(1);
    expect(heads[0].className).toContain("reasoning__head");
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

// v4.419 notice 摘出过程卡分组：动词回执（/plan on 等）与失败告警此前被折叠
// 进过程卡默认不可见（真机走查实锤）——notice 必须独立成行渲染（可见 chip 行）。
describe("notice 独立可见（v4.419 摘出过程卡分组）", () => {
  it("buildSegments：notice 落 outsideItems，绝不进 processItems", () => {
    const items: Item[] = [
      { kind: "user", id: "u1", text: "/plan on" } as Item,
      { kind: "notice", id: "n1", level: "info", text: "计划模式已开启：模型只做研究和方案设计" } as Item,
      { kind: "tool", id: "t1", name: "list_dir", args: "", readOnly: true, status: "done" } as Item,
      { kind: "notice", id: "n2", level: "warn", text: "写入失败：请重试" } as Item,
    ];
    const segs = buildSegments(items);
    const noticeInProcess = segs.some((s) => s.processItems.some((i) => i.kind === "notice"));
    const noticeOutside = segs.some((s) => s.outsideItems.some((i) => i.kind === "notice"));
    expect(noticeInProcess).toBe(false);
    expect(noticeOutside).toBe(true);
  });
});
