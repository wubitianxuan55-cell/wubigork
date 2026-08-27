import { describe, expect, it } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { PreviewNavBar } from "./PreviewNavBar";

describe("PreviewNavBar 预览队列导航条（P1-1，C7 chip 化）", () => {
  it("队列 ≤1 时不渲染（无切换/关闭意义）", () => {
    const { container } = render(
      <PreviewNavBar files={["a.md"]} index={0} onJump={() => {}} onClose={() => {}} />,
    );
    expect(container.firstChild).toBeNull();
  });

  it("多文件渲染文件 chip 条（basename）并高亮活动项", () => {
    render(
      <PreviewNavBar files={["docs/a.md", "b.docx"]} index={1} onJump={() => {}} onClose={() => {}} />,
    );
    expect(screen.getByText("a.md")).toBeTruthy();
    expect(screen.getByText("b.docx")).toBeTruthy();
    // aria-current 挂在外层 chip span 上；getByText 命中内层 truncate span，取其父级
    expect(screen.getByText("b.docx").parentElement?.getAttribute("aria-current")).toBe("true");
    expect(screen.getByText("a.md").parentElement?.getAttribute("aria-current")).not.toBe("true");
  });

  it("点击 chip 触发 onJump", () => {
    let jumped: number | null = null;
    render(
      <PreviewNavBar
        files={["a.md", "b.docx", "c.xlsx"]}
        index={0}
        onJump={(i) => { jumped = i; }}
        onClose={() => {}}
      />,
    );
    fireEvent.click(screen.getByText("c.xlsx"));
    expect(jumped).toBe(2);
  });

  it("× 按钮触发 onClose 且不触发 onJump（stopPropagation）", () => {
    let closed: number | null = null;
    let jumped = false;
    render(
      <PreviewNavBar
        files={["a.md", "b.docx"]}
        index={0}
        onJump={() => { jumped = true; }}
        onClose={(i) => { closed = i; }}
      />,
    );
    fireEvent.click(screen.getByLabelText("关闭 a.md"));
    expect(closed).toBe(0);
    expect(jumped).toBe(false);
  });

  it("中键按下+弹起在同一 chip 才关闭（VS Code 语义）", () => {
    let closed: number | null = null;
    render(
      <PreviewNavBar
        files={["a.md", "b.docx"]}
        index={0}
        onJump={() => {}}
        onClose={(i) => { closed = i; }}
      />,
    );
    const chip = screen.getByText("b.docx");
    // 中键按下 b.docx → 弹起仍落在 b.docx → 关闭
    fireEvent.mouseDown(chip, { button: 1 });
    fireEvent.mouseUp(chip, { button: 1 });
    expect(closed).toBe(1);
  });

  it("中键按下后弹起落在其他位置不关闭", () => {
    let closed: number | null = null;
    render(
      <PreviewNavBar
        files={["a.md", "b.docx", "c.xlsx"]}
        index={0}
        onJump={() => {}}
        onClose={(i) => { closed = i; }}
      />,
    );
    const chipB = screen.getByText("b.docx");
    const chipC = screen.getByText("c.xlsx");
    fireEvent.mouseDown(chipB, { button: 1 });
    // 弹起在另一个 chip（容器 mouseUp 清空记录）→ 不关闭
    fireEvent.mouseUp(chipC, { button: 1 });
    expect(closed).toBeNull();
  });
});
