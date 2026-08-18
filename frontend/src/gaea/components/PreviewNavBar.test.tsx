import { describe, expect, it } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { PreviewNavBar } from "./PreviewNavBar";

describe("PreviewNavBar 多文件预览导航条（P1-1）", () => {
  it("队列 ≤1 时不渲染（无切换意义）", () => {
    const { container } = render(
      <PreviewNavBar index={0} total={1} onPrev={() => {}} onNext={() => {}} />,
    );
    expect(container.firstChild).toBeNull();
  });

  it("多文件时显示位置指示 2/3", () => {
    render(
      <PreviewNavBar index={1} total={3} onPrev={() => {}} onNext={() => {}} />,
    );
    expect(screen.getByText("2/3")).toBeTruthy();
  });

  it("首元素禁用上一个,末元素禁用下一个", () => {
    const { rerender } = render(
      <PreviewNavBar index={0} total={3} onPrev={() => {}} onNext={() => {}} />,
    );
    expect(screen.getByTitle("上一个文件")).toHaveProperty("disabled", true);
    expect(screen.getByTitle("下一个文件")).toHaveProperty("disabled", false);
    rerender(<PreviewNavBar index={2} total={3} onPrev={() => {}} onNext={() => {}} />);
    expect(screen.getByTitle("上一个文件")).toHaveProperty("disabled", false);
    expect(screen.getByTitle("下一个文件")).toHaveProperty("disabled", true);
  });

  it("点击触发 onPrev/onNext", () => {
    let prev = 0;
    let next = 0;
    render(
      <PreviewNavBar
        index={1}
        total={3}
        onPrev={() => { prev++; }}
        onNext={() => { next++; }}
      />,
    );
    fireEvent.click(screen.getByTitle("上一个文件"));
    fireEvent.click(screen.getByTitle("下一个文件"));
    expect(prev).toBe(1);
    expect(next).toBe(1);
  });
});
