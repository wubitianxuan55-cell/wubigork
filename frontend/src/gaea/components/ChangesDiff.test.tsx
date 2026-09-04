import { describe, expect, it } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { ChangesDiff } from "./ChangesDiff";
import { diffLines } from "../lib/diff";
import type { ChangeDiff } from "../lib/planDiff";

describe("ChangesDiff 变更行 diff 渲染", () => {
  it("kind=diff：渲染红绿行（del/- 与 add/+）与上下文行", () => {
    const diff: ChangeDiff = {
      kind: "diff",
      hunks: [{ rows: diffLines("旧标题\n共有", "新标题\n共有") }],
    };
    const { container } = render(<ChangesDiff diff={diff} />);
    const hunk = container.querySelector('[data-testid="changes-diff-hunk"]')!;
    const rows = Array.from(hunk.querySelectorAll(":scope > div > div"));
    expect(rows.length).toBe(3);
    expect(rows[0].textContent).toContain("-旧标题");
    expect(rows[1].textContent).toContain("+新标题");
    expect(rows[2].textContent).toContain(" 共有");
    // 2c 改蓝配对：del/add 成对 → 蓝底（glow 混色）替代红/绿，且带 data-pair 标记
    expect((rows[0] as HTMLElement).getAttribute("data-pair")).toBe("old");
    expect((rows[1] as HTMLElement).getAttribute("data-pair")).toBe("new");
    expect((rows[0] as HTMLElement).style.background).toContain("color-mix");
  });

  it("kind=content：诚实降级为写入内容预览 + 降级原因，不出现红绿行", () => {
    const diff: ChangeDiff = {
      kind: "content",
      hunks: [],
      content: "写入的新内容",
      note: "覆盖写入：写入前内容未记录，以下为写入内容",
    };
    render(<ChangesDiff diff={diff} />);
    expect(screen.getByTestId("changes-content-preview")).toBeTruthy();
    expect(screen.getByText(/覆盖写入/)).toBeTruthy();
    expect(screen.getByText(/写入的新内容/)).toBeTruthy();
    expect(document.querySelector('[data-testid="changes-diff-hunk"]')).toBeNull();
  });

  it("kind=none：只显示诚实说明", () => {
    render(<ChangesDiff diff={{ kind: "none", hunks: [], note: "移动/重命名操作，无内容变化记录" }} />);
    expect(screen.getByTestId("changes-diff-none").textContent).toContain("移动/重命名操作");
  });

  it("行数超上限截断并标注总数", () => {
    const rows = diffLines(
      Array.from({ length: 350 }, (_, i) => `old-${i}`).join("\n"),
      Array.from({ length: 350 }, (_, i) => `new-${i}`).join("\n"),
    );
    render(<ChangesDiff diff={{ kind: "diff", hunks: [{ rows }] }} />);
    expect(screen.getByText(/已截断：共 700 行/)).toBeTruthy();
  });
});

describe("ChangesDiff 2c 渲染升级：改蓝配对 + 上下文折叠", () => {
  const withPairAndFold: ChangeDiff = {
    kind: "diff",
    hunks: [
      {
        label: "demo.txt",
        rows: [
          { type: "ctx", text: "第一行" },
          ...Array.from({ length: 10 }, (_, i) => ({ type: "ctx" as const, text: `上下文${i}` })),
          { type: "del", text: "旧标题" },
          { type: "add", text: "新标题" },
        ],
      },
    ],
  };

  it("配对行带 data-pair 标记（改蓝），未配对上下文行无标记", () => {
    const longCtx: ChangeDiff = {
      kind: "diff",
      hunks: [{ label: "t", rows: [{ type: "del", text: "旧标题" }, { type: "add", text: "新标题" }] }],
    };
    const { container } = render(<ChangesDiff diff={longCtx} />);
    expect(container.querySelector("[data-pair='old']")).toBeTruthy();
    expect(container.querySelector("[data-pair='new']")).toBeTruthy();
    expect(container.querySelector("[data-pair]")).toBeTruthy();
    // 配对行内做字符分段（多个 span）
    expect(container.querySelector("[data-pair='old']")!.querySelectorAll("span").length).toBeGreaterThan(0);
  });

  it("长上下文折叠：折叠占位可点击展开原行", () => {
    const { container } = render(<ChangesDiff diff={withPairAndFold} />);
    const toggles = container.querySelectorAll("[data-testid='diff-fold-toggle']");
    expect(toggles.length).toBe(1);
    expect(toggles[0]!.textContent).toContain("已折叠 5 行"); // 11 ctx - 3*2
    expect(screen.queryByText("上下文4")).toBeNull();
    fireEvent.click(toggles[0]!);
    expect(screen.getByText("上下文4")).toBeTruthy();
    expect(container.querySelectorAll("[data-testid='diff-fold-toggle']").length).toBe(0);
  });
});
