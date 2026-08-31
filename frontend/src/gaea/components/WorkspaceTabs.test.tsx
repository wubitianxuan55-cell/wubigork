import { describe, expect, it, vi } from "vitest";
import { render, fireEvent } from "@testing-library/react";
import { WorkspaceTabs } from "./WorkspaceTabs";
import { WORKSPACE_TAB_IDS, type WorkspaceTabId } from "../lib/workspaceTabs";

describe("WorkspaceTabs 两级按钮条（v3.0.8 收敛分组；v4.23 统计迁主区后为 3 个主 Tab）", () => {
  it("第一级渲染 3 个主 Tab（分组）且激活态正确", () => {
    render(<WorkspaceTabs active="files" onChange={() => {}} />);
    expect(document.querySelectorAll('[data-grouptab]')).toHaveLength(3);
    expect(document.querySelector('[data-grouptab="files"]')?.getAttribute("aria-selected")).toBe("true");
    expect(document.querySelector('[data-grouptab="running"]')?.getAttribute("aria-selected")).toBe("false");
  });

  it("第二级渲染当前组内子面板小 Tab（用 data-subtab 定位）", () => {
    render(<WorkspaceTabs active="files" onChange={() => {}} />);
    // 文件组：文件 / 资料 两个子 Tab
    expect(document.querySelector('[data-subtab="files"]')).toBeTruthy();
    expect(document.querySelector('[data-subtab="materials"]')).toBeTruthy();
    expect(document.querySelector('[data-subtab="files"]')?.getAttribute("aria-selected")).toBe("true");
    expect(document.querySelector('[data-subtab="materials"]')?.getAttribute("aria-selected")).toBe("false");
  });

  it("点击主 Tab 触发 onChange 落到该组默认子面板", () => {
    const onChange = vi.fn();
    render(<WorkspaceTabs active="files" onChange={onChange} />);
    fireEvent.click(document.querySelector('[data-grouptab="running"]') as HTMLElement);
    expect(onChange).toHaveBeenCalledWith("tasks"); // 运行组默认 = 任务
  });

  it("点击组内子 Tab 触发 onChange 携带对应子面板 id", () => {
    const onChange = vi.fn();
    render(<WorkspaceTabs active="files" onChange={onChange} />);
    fireEvent.click(document.querySelector('[data-subtab="materials"]') as HTMLElement);
    expect(onChange).toHaveBeenCalledWith("materials");
  });

  it("组内只剩一个启用面板时不渲染第二级（停用收敛，学 sanitizeState）", () => {
    const enabled = new Set([...ALL_ENABLED].filter((id) => id !== "changes"));
    render(<WorkspaceTabs active="deliverables" onChange={() => {}} enabledTabs={enabled} />);
    expect(document.querySelector('[data-grouptab="outputs"]')?.getAttribute("aria-selected")).toBe("true");
    expect(document.querySelector('[data-grouptab="files"]')?.getAttribute("aria-selected")).toBe("false");
    // 成果组只剩 1 个启用子面板 → 无第二级小 Tab
    expect(document.querySelector('[data-subtab="deliverables"]')).toBeNull();
  });
});

describe("WorkspaceTabs 运行域活动角标（C6，蒸馏 dsh-better-sidebar badge）", () => {
  it("无 badges 时不渲染角标", () => {
    render(<WorkspaceTabs active="files" onChange={() => {}} />);
    expect(document.querySelector('[data-grouptab="running"]')?.textContent).toBe("运行");
  });

  it("活跃任务数 >0 时运行组显示计数角标", () => {
    render(<WorkspaceTabs active="files" onChange={() => {}} badges={{ running: 3 }} />);
    const runningTab = document.querySelector('[data-grouptab="running"]');
    expect(runningTab?.textContent).toContain("运行");
    expect(runningTab?.textContent).toContain("3");
  });

  it("角标 99+ 封顶", () => {
    render(<WorkspaceTabs active="files" onChange={() => {}} badges={{ running: 123 }} />);
    const runningTab = document.querySelector('[data-grouptab="running"]');
    expect(runningTab?.textContent).toContain("99+");
  });

  it("运行组激活时不显示角标（视为已读）", () => {
    render(<WorkspaceTabs active="tasks" onChange={() => {}} badges={{ running: 3 }} />);
    const runningTab = document.querySelector('[data-grouptab="running"]');
    expect(runningTab?.textContent).toBe("运行");
  });

  it("非运行组不渲染角标（0 计数不渲染）", () => {
    render(<WorkspaceTabs active="files" onChange={() => {}} badges={{ running: 0, outputs: 2 }} />);
    const filesTab = document.querySelector('[data-grouptab="files"]');
    expect(filesTab?.textContent).toBe("文件");
  });
});

// ── v4.23 声明式设置（蒸馏 dsh-better-sidebar「侧边卡片」）──────────────────

const ALL_ENABLED = new Set<WorkspaceTabId>(WORKSPACE_TAB_IDS);

function openSettings(): void {
  fireEvent.click(document.querySelector('[data-testid="workspace-tabs-gear"]') as HTMLElement);
}

describe("WorkspaceTabs 声明式设置（v4.23 蒸馏 dsh-better-sidebar 侧边卡片）", () => {
  it("齿轮按钮打开设置弹层，7 个面板各一张卡（名称 + 开关）", () => {
    render(<WorkspaceTabs active="files" onChange={() => {}} enabledTabs={ALL_ENABLED} />);
    openSettings();
    expect(document.querySelector('[data-testid="workspace-tabs-settings"]')).toBeTruthy();
    expect(document.querySelectorAll("[data-settings-card]")).toHaveLength(7);
    expect(document.querySelectorAll('[role="switch"]')).toHaveLength(7);
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
    fireEvent.click(document.querySelector('[data-tabswitch="cost"]') as HTMLElement);
    expect(onToggleTab).toHaveBeenCalledWith("cost", false);
  });

  it("停用整组面板 → 对应主 Tab 从第一级隐藏", () => {
    const enabled = new Set([...ALL_ENABLED].filter((id) => id !== "deliverables" && id !== "changes"));
    render(<WorkspaceTabs active="files" onChange={() => {}} enabledTabs={enabled} />);
    expect(document.querySelectorAll("[data-grouptab]")).toHaveLength(2);
    expect(document.querySelector('[data-grouptab="outputs"]')).toBeNull();
  });

  it("组内部分停用 → 第二级只渲染启用的子面板", () => {
    const enabled = new Set([...ALL_ENABLED].filter((id) => id !== "materials"));
    render(<WorkspaceTabs active="files" onChange={() => {}} enabledTabs={enabled} />);
    expect(document.querySelector('[data-subtab="materials"]')).toBeNull();
    expect(document.querySelector('[data-subtab="files"]')).toBeTruthy();
    expect(document.querySelector('[data-subtab="cost"]')).toBeTruthy();
  });

  it("点击主 Tab 落到该组第一个启用面板（跳过已停用项）", () => {
    const onChange = vi.fn();
    const enabled = new Set([...ALL_ENABLED].filter((id) => id !== "tasks"));
    render(<WorkspaceTabs active="files" onChange={onChange} enabledTabs={enabled} />);
    fireEvent.click(document.querySelector('[data-grouptab="running"]') as HTMLElement);
    expect(onChange).toHaveBeenCalledWith("subagents");
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
    expect(document.querySelectorAll("[data-settings-card]")).toHaveLength(7);
    document.querySelectorAll('[role="switch"]').forEach((sw) => {
      expect(sw.hasAttribute("disabled")).toBe(false);
    });
  });
});
