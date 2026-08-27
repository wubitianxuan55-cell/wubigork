import { describe, expect, it, vi } from "vitest";
import { render, fireEvent } from "@testing-library/react";
import { WorkspaceTabs } from "./WorkspaceTabs";

describe("WorkspaceTabs 两级按钮条（v3.0.8 收敛为 4 个主 Tab）", () => {
  it("第一级渲染 4 个主 Tab（分组）且激活态正确", () => {
    render(<WorkspaceTabs active="files" onChange={() => {}} />);
    expect(document.querySelectorAll('[data-grouptab]')).toHaveLength(4);
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

  it("单面板组（分析）不渲染第二级", () => {
    render(<WorkspaceTabs active="stats" onChange={() => {}} />);
    expect(document.querySelector('[data-grouptab="insight"]')?.getAttribute("aria-selected")).toBe("true");
    expect(document.querySelector('[data-grouptab="files"]')?.getAttribute("aria-selected")).toBe("false");
    // 分析组只有 1 个子面板 → 无第二级小 Tab
    expect(document.querySelector('[data-subtab="stats"]')).toBeNull();
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
