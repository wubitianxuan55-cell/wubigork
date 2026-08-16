import { describe, expect, it, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { WorkspaceTabs } from "./WorkspaceTabs";
import { WORKSPACE_TABS } from "../lib/workspaceTabs";

// 按钮 accessible name = 图标 aria-label + 文本（如 "folder-open 文件"），用正则包含匹配
const tabByName = (label: string) => screen.getByRole("tab", { name: new RegExp(label) });

describe("WorkspaceTabs 按钮条", () => {
  it("按清单渲染全部 Tab 且激活态正确", () => {
    render(<WorkspaceTabs active="files" onChange={() => {}} />);
    for (const tab of WORKSPACE_TABS) {
      const btn = tabByName(tab.label);
      expect(btn).toBeTruthy();
      expect(btn.getAttribute("aria-selected")).toBe(tab.id === "files" ? "true" : "false");
    }
  });

  it("点击触发 onChange 并携带对应 id", () => {
    const onChange = vi.fn();
    render(<WorkspaceTabs active="files" onChange={onChange} />);
    fireEvent.click(tabByName("任务"));
    expect(onChange).toHaveBeenCalledTimes(1);
    expect(onChange).toHaveBeenCalledWith("tasks");
  });

  it("激活态切换后高亮随之更新", () => {
    const { rerender } = render(<WorkspaceTabs active="files" onChange={() => {}} />);
    expect(tabByName("文件").getAttribute("aria-selected")).toBe("true");
    rerender(<WorkspaceTabs active="stats" onChange={() => {}} />);
    expect(tabByName("统计").getAttribute("aria-selected")).toBe("true");
    expect(tabByName("文件").getAttribute("aria-selected")).toBe("false");
  });
});
