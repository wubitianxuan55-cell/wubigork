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

describe("ChangesDiff marker 列（v4.87 统一 diff 查看器：docx 段号 / xlsx ref）", () => {
  const hunkRows = (container: HTMLElement) =>
    Array.from(
      container.querySelectorAll("[data-testid='changes-diff-hunk'] > div > div"),
    ) as HTMLElement[];

  it("marker 存在：在 +/- 列后渲染定宽右对齐列，正文不变", () => {
    const diff: ChangeDiff = {
      kind: "diff",
      hunks: [
        {
          rows: [
            { type: "ctx", text: "标题段", marker: "1" },
            { type: "del", text: "旧段", marker: "12" },
            { type: "add", text: "新段", marker: "13" },
          ],
        },
      ],
    };
    const { container } = render(<ChangesDiff diff={diff} />);
    const rows = hunkRows(container);
    expect(rows).toHaveLength(3);
    // 行文本 = 符号列 + marker + 正文（marker 列夹在中间，缺列不占位）
    expect(rows[0].textContent).toBe(" 1标题段");
    expect(rows[1].textContent).toBe("-12旧段");
    expect(rows[2].textContent).toBe("+13新段");
  });

  it("无 marker：不渲染 marker 列（ChangesPanel/GitPanel 既有渲染零变化）", () => {
    const { container } = render(
      <ChangesDiff diff={{ kind: "diff", hunks: [{ rows: [{ type: "add", text: "纯增行" }] }] }} />,
    );
    const rows = hunkRows(container);
    expect(rows).toHaveLength(1);
    expect(rows[0].textContent).toBe("+纯增行");
    // 仅符号列 + 正文两段（无 marker 列 span）
    expect(rows[0].querySelectorAll("span")).toHaveLength(2);
  });

  it("配对行与折叠占位展开后均保留 marker", () => {
    const diff: ChangeDiff = {
      kind: "diff",
      hunks: [
        {
          rows: [
            ...Array.from({ length: 8 }, (_, i) => ({
              type: "ctx" as const,
              text: `段${i + 1}`,
              marker: String(i + 1),
            })),
            { type: "del", text: "旧段", marker: "9" },
            { type: "add", text: "新段", marker: "9" },
          ],
        },
      ],
    };
    const { container } = render(<ChangesDiff diff={diff} />);
    // 8 连续 ctx 折叠：3 + 折叠开关(button，不计入 div 行) + 3 + 配对 del/add
    const toggle = () => container.querySelector("[data-testid='diff-fold-toggle']")!;
    let rows = hunkRows(container);
    expect(toggle().textContent).toContain("已折叠 2 行"); // 8 ctx - 3*2
    expect(rows).toHaveLength(8);
    expect(rows[3].textContent).toBe(" 6段6"); // 折叠后第一可见 ctx 是段6
    // 配对行改蓝仍带 marker（pairModifications 携带原行对象）
    expect(rows[6].textContent).toBe("-9旧段");
    expect(rows[6].getAttribute("data-pair")).toBe("old");
    expect(rows[7].textContent).toBe("+9新段");
    expect(rows[7].getAttribute("data-pair")).toBe("new");
    // 展开折叠：原 ctx 行就地渲染，marker 列不丢
    fireEvent.click(toggle());
    rows = hunkRows(container);
    expect(rows).toHaveLength(10);
    expect(rows[3].textContent).toBe(" 4段4");
    expect(rows[4].textContent).toBe(" 5段5");
  });
});
