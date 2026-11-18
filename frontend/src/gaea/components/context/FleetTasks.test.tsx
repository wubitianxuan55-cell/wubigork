import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { LocaleProvider } from "../../lib/i18n";
import type { FamilyAgentRowView } from "../../lib/types/context";
import type { SubagentRunView } from "../../lib/types/subagent";
import { FleetTasks } from "./FleetTasks";

const RUNS: SubagentRunView[] = [
  {
    ref: "sa_1",
    status: "completed",
    task: "调研 dsh-context 功能面",
    answer: "五入口与三投影已完成",
    model: "deepseek-flash",
    toolCalls: 12,
    createdAt: "2026-11-14T09:00:00Z",
    updatedAt: "2026-11-14T09:20:00Z",
  },
  {
    ref: "sa_2",
    status: "failed",
    task: "写映射表",
    answer: "权限被拒",
    model: "deepseek-flash",
    toolCalls: 2,
    createdAt: "2026-11-14T09:02:00Z",
    updatedAt: "2026-11-14T09:03:00Z",
  },
  {
    ref: "sa_3",
    status: "running",
    task: "事件面板",
    model: "deepseek-reasoner",
    toolCalls: 5,
    createdAt: "2026-11-14T09:06:00Z",
    updatedAt: "2026-11-14T09:08:00Z",
  },
];

const AGENTS: FamilyAgentRowView[] = [
  { ref: "sa_1", tokens: 30_000, cost: 0.11, dispatches: 1, messages: 4, reports: 1 },
  { ref: "sa_2", tokens: 8_000, dispatches: 1, messages: 2 },
  { ref: "sa_3", tokens: 25_000, cost: 0.08, dispatches: 1, messages: 1 },
];

function renderTasks(overrides: Partial<React.ComponentProps<typeof FleetTasks>> = {}) {
  const props: React.ComponentProps<typeof FleetTasks> = {
    runs: RUNS,
    agents: AGENTS,
    selectedAgent: null,
    onSelectAgent: vi.fn(),
    onViewContext: vi.fn(),
    ...overrides,
  };
  render(
    <LocaleProvider>
      <FleetTasks {...props} />
    </LocaleProvider>,
  );
  return props;
}

describe("FleetTasks", () => {
  it("shows title, owner groups and real status counts", () => {
    renderTasks();
    const panel = screen.getByTestId("fleet-tasks");
    expect(panel.textContent).toContain("协作任务");
    expect(panel.textContent).toContain("3/3 个工作项");
    expect(screen.getByTestId("fleet-task-status-all").textContent).toContain("3");
    expect(screen.getByTestId("fleet-task-status-running").textContent).toContain("1");
    expect(screen.getByTestId("fleet-task-status-completed").textContent).toContain("1");
    expect(screen.getByTestId("fleet-task-status-failed").textContent).toContain("1");
  });

  it("filters by phase and text search", () => {
    renderTasks();
    fireEvent.click(screen.getByTestId("fleet-task-status-failed"));
    const panel = screen.getByTestId("fleet-tasks");
    expect(panel.textContent).toContain("写映射表");
    expect(panel.textContent).not.toContain("调研 dsh-context 功能面");

    fireEvent.click(screen.getByTestId("fleet-task-status-all"));
    fireEvent.change(screen.getByPlaceholderText("搜索…"), { target: { value: "事件面板" } });
    expect(panel.textContent).toContain("事件面板");
    expect(panel.textContent).not.toContain("写映射表");
  });

  it("expands a task and opens its context", () => {
    const props = renderTasks();
    fireEvent.click(screen.getByTestId("fleet-task-status-failed"));
    fireEvent.click(screen.getByLabelText("展开任务"));
    expect(screen.getByText("权限被拒")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: /打开会话/ }));
    expect(props.onViewContext).toHaveBeenCalledWith("sa_2");
  });

  it("分工卡侧栏钮：不展开即点即回调（ref+task），右栏打开链", () => {
    const onOpenAside = vi.fn();
    renderTasks({ onOpenAside });
    fireEvent.click(screen.getByTestId("fleet-task-status-failed"));
    const aside = screen.getByTestId("fleet-task-aside");
    fireEvent.click(aside);
    expect(onOpenAside).toHaveBeenCalledTimes(1);
    expect(onOpenAside.mock.calls[0][0]).toBe("sa_2");

    cleanup();
    renderTasks({ onOpenAside: undefined });
    fireEvent.click(screen.getByTestId("fleet-task-status-all"));
    expect(screen.queryByTestId("fleet-task-aside")).toBeNull();
  });

  it("selects and clears an owner filter from the group header", () => {
    const onSelectAgent = vi.fn();
    renderTasks({ onSelectAgent });
    const panel = screen.getByTestId("fleet-tasks");
    const owner = panel.querySelector('[data-owner="sa_1"] .fleet-task-owner') as HTMLElement;
    fireEvent.click(owner);
    expect(onSelectAgent).toHaveBeenCalledWith("sa_1");

    renderTasks({ onSelectAgent, selectedAgent: "sa_1" });
    const selectedOwner = screen
      .getAllByTestId("fleet-tasks")[1]
      .querySelector('[data-owner="sa_1"] .fleet-task-owner') as HTMLElement;
    fireEvent.click(selectedOwner);
    expect(onSelectAgent).toHaveBeenLastCalledWith(null);
  });
});
