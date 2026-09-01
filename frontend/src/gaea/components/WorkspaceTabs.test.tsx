import { describe, expect, it, vi } from "vitest";
import { render, fireEvent } from "@testing-library/react";
import { WorkspaceTabs } from "./WorkspaceTabs";
import { WORKSPACE_TAB_IDS, type WorkspaceTabId } from "../lib/workspaceTabs";

describe("WorkspaceTabs 一级按钮条（v4.27 扁平化：无二级标签）", () => {
  it("渲染 5 个一级 Tab（文件/产物/变更/任务/分工）且激活态正确", () => {
    render(<WorkspaceTabs active="files" onChange={() => {}} />);
    expect(document.querySelectorAll("[data-paneltab]")).toHaveLength(5);
    expect(document.querySelector('[data-paneltab="files"]')?.getAttribute("aria-selected")).toBe("true");
    expect(document.querySelector('[data-paneltab="deliverables"]')?.getAttribute("aria-selected")).toBe("false");
    // 已删除的资料/成本库不再渲染
    expect(document.querySelector('[data-paneltab="materials"]')).toBeNull();
    expect(document.querySelector('[data-paneltab="cost"]')).toBeNull();
    // 无第二级小 Tab
    expect(document.querySelector("[data-subtab]")).toBeNull();
  });

  it("点击 Tab 触发 onChange 携带对应面板 id", () => {
    const onChange = vi.fn();
    render(<WorkspaceTabs active="files" onChange={onChange} />);
    fireEvent.click(document.querySelector('[data-paneltab="changes"]') as HTMLElement);
    expect(onChange).toHaveBeenCalledWith("changes");
    fireEvent.click(document.querySelector('[data-paneltab="tasks"]') as HTMLElement);
    expect(onChange).toHaveBeenCalledWith("tasks");
  });

  it("停用的面板从 Tab 条隐藏（声明式设置收敛）", () => {
    const enabled = new Set([...ALL_ENABLED].filter((id) => id !== "changes"));
    render(<WorkspaceTabs active="deliverables" onChange={() => {}} enabledTabs={enabled} />);
    expect(document.querySelectorAll("[data-paneltab]")).toHaveLength(4);
    expect(document.querySelector('[data-paneltab="changes"]')).toBeNull();
    expect(document.querySelector('[data-paneltab="deliverables"]')?.getAttribute("aria-selected")).toBe("true");
  });
});

describe("WorkspaceTabs 运行域活动角标（C6，蒸馏 dsh-better-sidebar badge）", () => {
  it("无 badges 时不渲染角标", () => {
    render(<WorkspaceTabs active="files" onChange={() => {}} />);
    expect(document.querySelector('[data-paneltab="tasks"]')?.textContent).toBe("任务");
  });

  it("活跃任务数 >0 时任务/分工 Tab 显示计数角标（未激活者）", () => {
    render(<WorkspaceTabs active="files" onChange={() => {}} badges={{ tasks: 3, subagents: 3 }} />);
    expect(document.querySelector('[data-paneltab="tasks"]')?.textContent).toContain("任务");
    expect(document.querySelector('[data-paneltab="tasks"]')?.textContent).toContain("3");
    expect(document.querySelector('[data-paneltab="subagents"]')?.textContent).toContain("3");
  });

  it("角标 99+ 封顶", () => {
    render(<WorkspaceTabs active="files" onChange={() => {}} badges={{ tasks: 123, subagents: 123 }} />);
    expect(document.querySelector('[data-paneltab="tasks"]')?.textContent).toContain("99+");
  });

  it("任务/分工任一激活时不显示角标（视为已读）", () => {
    render(<WorkspaceTabs active="tasks" onChange={() => {}} badges={{ tasks: 3, subagents: 3 }} />);
    expect(document.querySelector('[data-paneltab="tasks"]')?.textContent).toBe("任务");
    expect(document.querySelector('[data-paneltab="subagents"]')?.textContent).toContain("3");
  });

  it("非运行面板不渲染角标（0 计数不渲染）", () => {
    render(<WorkspaceTabs active="files" onChange={() => {}} badges={{ tasks: 0, deliverables: 0 }} />);
    expect(document.querySelector('[data-paneltab="deliverables"]')?.textContent).toBe("产物");
  });
});

// ── v4.23 声明式设置（蒸馏 dsh-better-sidebar「侧边卡片」）──────────────────

const ALL_ENABLED = new Set<WorkspaceTabId>(WORKSPACE_TAB_IDS);

function openSettings(): void {
  fireEvent.click(document.querySelector('[data-testid="workspace-tabs-gear"]') as HTMLElement);
}

describe("WorkspaceTabs 声明式设置（v4.23 蒸馏 dsh-better-sidebar 侧边卡片）", () => {
  it("齿轮按钮打开设置弹层，5 个面板各一张卡（名称 + 开关）", () => {
    render(<WorkspaceTabs active="files" onChange={() => {}} enabledTabs={ALL_ENABLED} />);
    openSettings();
    expect(document.querySelector('[data-testid="workspace-tabs-settings"]')).toBeTruthy();
    expect(document.querySelectorAll("[data-settings-card]")).toHaveLength(5);
    expect(document.querySelectorAll('[role="switch"]')).toHaveLength(5);
    // 默认全启用：开关 aria-checked 全 true（NodeList 无迭代器，用 forEach）
    document.querySelectorAll('[role="switch"]').forEach((sw) => {
      expect(sw.getAttribute("aria-checked")).toBe("true");
    });
  });

  it("再次点击齿轮关闭弹层", () => {
    render(<WorkspaceTabs active="files" onChange={() => {}} enabledTabs={ALL_ENABLED} />);
    openSettings();
    expect(document.querySelector('[data-testid="workspace-tabs-settings"]')).toBeTruthy();
    openSettings();
    expect(document.querySelector('[data-testid="workspace-tabs-settings"]')).toBeNull();
  });

  it("点击遮罩（弹层外部）关闭弹层", () => {
    render(<WorkspaceTabs active="files" onChange={() => {}} enabledTabs={ALL_ENABLED} />);
    openSettings();
    fireEvent.click(document.querySelector(".fixed.inset-0") as HTMLElement);
    expect(document.querySelector('[data-testid="workspace-tabs-settings"]')).toBeNull();
  });

  it("开关点击触发 onToggleTab(id, next)", () => {
    const onToggleTab = vi.fn();
    render(<WorkspaceTabs active="files" onChange={() => {}} enabledTabs={ALL_ENABLED} onToggleTab={onToggleTab} />);
    openSettings();
    fireEvent.click(document.querySelector('[data-tabswitch="changes"]') as HTMLElement);
    expect(onToggleTab).toHaveBeenCalledWith("changes", false);
    fireEvent.click(document.querySelector('[data-tabswitch="tasks"]') as HTMLElement);
    expect(onToggleTab).toHaveBeenCalledWith("tasks", false);
  });

  it("至少保留一个启用面板：最后剩下的开关锁定（不可关）", () => {
    render(<WorkspaceTabs active="files" onChange={() => {}} enabledTabs={new Set<WorkspaceTabId>(["files"])} />);
    openSettings();
    const filesSwitch = document.querySelector('[data-tabswitch="files"]') as HTMLButtonElement;
    expect(filesSwitch.getAttribute("aria-checked")).toBe("true");
    expect(filesSwitch.hasAttribute("disabled")).toBe(true);
  });

  it("非最后一个启用面板的开关不锁定", () => {
    render(<WorkspaceTabs active="files" onChange={() => {}} enabledTabs={ALL_ENABLED} />);
    openSettings();
    const filesSwitch = document.querySelector('[data-tabswitch="files"]') as HTMLButtonElement;
    expect(filesSwitch.hasAttribute("disabled")).toBe(false);
  });

  it("不传 enabledTabs 时（旧调用方兼容）弹层同样可用且全部启用", () => {
    render(<WorkspaceTabs active="files" onChange={() => {}} />);
    openSettings();
    expect(document.querySelectorAll("[data-settings-card]")).toHaveLength(5);
    document.querySelectorAll('[role="switch"]').forEach((sw) => {
      expect(sw.hasAttribute("disabled")).toBe(false);
    });
  });
});
