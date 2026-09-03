import { describe, expect, it } from "vitest";
import { render } from "@testing-library/react";
import { MergedPanel } from "./MergedPanel";

// v4.53 合并面板壳：两块内容直接并成一个面板（上下分区同屏全可见），
// 不引入二级标签/段切换——零额外点击。

describe("MergedPanel 合并面板壳（v4.53 产物与变更/任务与分工直接合并）", () => {
  it("两块内容同时渲染（同屏全可见，无段切换）", () => {
    render(
      <MergedPanel
        primary={<div data-testid="part-primary">产物区</div>}
        secondary={<div data-testid="part-secondary">变更区</div>}
      />,
    );
    expect(document.querySelector('[data-testid="part-primary"]')).toBeTruthy();
    expect(document.querySelector('[data-testid="part-secondary"]')).toBeTruthy();
  });

  it("主区占大头（flex-1），次区带分隔线、封顶高度且可滚动", () => {
    const { container } = render(
      <MergedPanel
        primary={<div>主</div>}
        secondary={<div>次</div>}
      />,
    );
    const sections = container.firstElementChild!.children;
    expect(sections[0].className).toContain("flex-1");
    expect(sections[1].className).toContain("shrink-0");
    expect(sections[1].className).toContain("max-h-[45%]");
    expect(sections[1].className).toContain("overflow-y-auto");
    expect(sections[1].className).toContain("border-t");
  });
});
