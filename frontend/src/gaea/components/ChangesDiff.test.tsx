import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
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
    // 红绿底色走 diff 令牌
    expect((rows[0] as HTMLElement).style.background).toContain("--del-bg");
    expect((rows[1] as HTMLElement).style.background).toContain("--add-bg");
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
