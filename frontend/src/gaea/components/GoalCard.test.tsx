import { describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import type { ReactElement } from "react";
import { LocaleProvider } from "../lib/i18n";
import type { Requirement } from "../lib/types";
import { GoalCard } from "./GoalCard";

function wrap(ui: ReactElement) {
  return <LocaleProvider>{ui}</LocaleProvider>;
}

const baseRequirement: Requirement = {
  text: "整理季度经营数据，输出带图表的总结报告（docx）",
  done: false,
  updatedAt: 1,
  items: [
    { text: "数据口径与来源确认", done: true },
    { text: "图表生成并嵌入报告", done: false },
    { text: "输出 docx 到交付物目录", done: false },
  ],
  autoPursue: false,
};

describe("GoalCard 任务目标卡", () => {
  it("渲染目标 + 验收清单，展示进度与状态", () => {
    render(wrap(<GoalCard requirement={baseRequirement} />));
    expect(screen.getByText(/整理季度经营数据/)).toBeTruthy();
    expect(screen.getByText("1/3")).toBeTruthy();
    expect(screen.getByText("进行中")).toBeTruthy();
    expect(screen.getByText("数据口径与来源确认")).toBeTruthy();
    expect(screen.getByText("自动追踪")).toBeTruthy();
  });

  it("全部勾选后显示已验收", () => {
    const allDone: Requirement = {
      ...baseRequirement,
      done: true,
      items: baseRequirement.items.map((it) => ({ ...it, done: true })),
    };
    render(wrap(<GoalCard requirement={allDone} />));
    expect(screen.getByText("已验收")).toBeTruthy();
    expect(screen.getByText("3/3")).toBeTruthy();
    expect(screen.getByText("重新打开")).toBeTruthy();
  });

  it("勾选验收项回调 onSetRequirementItemDone", () => {
    const onSetDone = vi.fn();
    render(wrap(<GoalCard requirement={baseRequirement} onSetRequirementItemDone={onSetDone} />));
    const checkboxes = screen.getAllByRole("checkbox");
    expect(checkboxes).toHaveLength(3);
    fireEvent.click(checkboxes[1]); // 第 2 项：false → true
    expect(onSetDone).toHaveBeenCalledWith(1, true);
  });

  it("回车添加验收项回调 onAddRequirementItem", () => {
    const onAdd = vi.fn();
    render(wrap(<GoalCard requirement={baseRequirement} onAddRequirementItem={onAdd} />));
    const input = screen.getByPlaceholderText(/添加验收标准/);
    fireEvent.change(input, { target: { value: "报告导出 PDF 版本" } });
    fireEvent.keyDown(input, { key: "Enter" });
    expect(onAdd).toHaveBeenCalledWith("报告导出 PDF 版本");
  });

  it("删除按钮回调 onRemoveRequirementItem", () => {
    const onRemove = vi.fn();
    render(wrap(<GoalCard requirement={baseRequirement} onRemoveRequirementItem={onRemove} />));
    fireEvent.click(screen.getByLabelText("删除验收项：数据口径与来源确认"));
    expect(onRemove).toHaveBeenCalledWith(0);
  });

  it("双击编辑验收项并回车提交", () => {
    const onSetItem = vi.fn();
    render(wrap(<GoalCard requirement={baseRequirement} onSetRequirementItem={onSetItem} />));
    fireEvent.doubleClick(screen.getByText("图表生成并嵌入报告"));
    const edit = screen.getByDisplayValue("图表生成并嵌入报告");
    fireEvent.change(edit, { target: { value: "生成折线图并嵌入报告" } });
    fireEvent.keyDown(edit, { key: "Enter" });
    expect(onSetItem).toHaveBeenCalledWith(1, "生成折线图并嵌入报告");
  });

  it("自动追踪开关回调并展示开启态", () => {
    const onToggle = vi.fn();
    const { rerender } = render(wrap(<GoalCard requirement={baseRequirement} onToggleRequirementAutoPursue={onToggle} />));
    const sw = screen.getByRole("switch");
    expect(sw.getAttribute("aria-checked")).toBe("false");
    fireEvent.click(sw);
    expect(onToggle).toHaveBeenCalledTimes(1);
    rerender(wrap(<GoalCard requirement={{ ...baseRequirement, autoPursue: true }} onToggleRequirementAutoPursue={onToggle} />));
    expect(screen.getByRole("switch").getAttribute("aria-checked")).toBe("true");
  });

  it("标记验收完成回调 onToggleRequirementDone", () => {
    const onToggleDone = vi.fn();
    render(wrap(<GoalCard requirement={baseRequirement} onToggleRequirementDone={onToggleDone} />));
    fireEvent.click(screen.getByText("标记验收完成"));
    expect(onToggleDone).toHaveBeenCalledTimes(1);
  });
});
