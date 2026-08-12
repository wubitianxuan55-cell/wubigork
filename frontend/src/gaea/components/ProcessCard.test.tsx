import { describe, expect, it } from "vitest";
import { render } from "@testing-library/react";
import { ProcessCard } from "./Transcript";
import type { Item } from "../lib/store";

const noSubcalls = new Map<string, never>();

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
