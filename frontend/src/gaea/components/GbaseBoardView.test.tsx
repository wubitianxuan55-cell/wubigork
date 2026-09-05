// GbaseBoardView.test.tsx — B2 看板视图渲染与拖拽改值回归
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { GbaseBoardView } from "./GbaseBoardView";
import { gbaseSheetModel, type GbaseView } from "../lib/gbase";
import type { XlsxSheet } from "../lib/types";

const SHEET: XlsxSheet = {
  name: "Sheet1",
  rows: [
    [
      { ref: "A1", value: "状态" },
      { ref: "B1", value: "事项" },
      { ref: "C1", value: "负责人" },
    ],
    [
      { ref: "A2", value: "进行中" },
      { ref: "B2", value: "审计底稿" },
      { ref: "C2", value: "张三" },
    ],
    [
      { ref: "A3", value: "完成" },
      { ref: "B3", value: "询价函" },
      { ref: "C3", value: "李四" },
    ],
  ],
};

const VIEW: GbaseView = {
  id: "v1",
  name: "看板",
  type: "board",
  groupBy: "状态",
  cardFields: ["事项", "负责人"],
};

function renderBoard(onMoveCard = vi.fn()) {
  const model = gbaseSheetModel(SHEET);
  render(<GbaseBoardView model={model} view={VIEW} onMoveCard={onMoveCard} />);
  return onMoveCard;
}

describe("GbaseBoardView（B2 看板）", () => {
  it("按 groupBy 列值分泳道渲染卡片（首字段=卡标题）", () => {
    renderBoard();
    const lanes = screen.getAllByTestId("gbase-lane");
    expect(lanes.map((l) => l.getAttribute("data-lane-key"))).toEqual(["进行中", "完成"]);
    const cards = screen.getAllByTestId("gbase-card");
    expect(cards).toHaveLength(2);
    expect(screen.getByText("审计底稿")).toBeTruthy();
    expect(screen.getAllByText("负责人：").length).toBeGreaterThanOrEqual(1); // 标签与值分属两个文本节点
    expect(screen.getByText("李四")).toBeTruthy();
  });

  it("卡片拖到目标泳道触发 onMoveCard(rowIndex, 泳道值)", () => {
    const onMoveCard = renderBoard();
    const card = screen.getAllByTestId("gbase-card").find((c) => c.getAttribute("data-row-index") === "2")!;
    const doneLane = screen.getAllByTestId("gbase-lane").find((l) => l.getAttribute("data-lane-key") === "完成")!;
    fireEvent.dragStart(card);
    fireEvent.dragOver(doneLane);
    fireEvent.drop(doneLane);
    expect(onMoveCard).toHaveBeenCalledWith(2, "完成");
  });

  it("原地放下不触发写盘", () => {
    const onMoveCard = renderBoard();
    const card = screen.getAllByTestId("gbase-card").find((c) => c.getAttribute("data-row-index") === "2")!;
    const srcLane = screen.getAllByTestId("gbase-lane").find((l) => l.getAttribute("data-lane-key") === "进行中")!;
    fireEvent.dragStart(card);
    fireEvent.dragOver(srcLane);
    fireEvent.drop(srcLane);
    expect(onMoveCard).not.toHaveBeenCalled();
  });

  it("moving 期间禁用拖拽（不触发 onMoveCard）", () => {
    const onMoveCard = vi.fn();
    const model = gbaseSheetModel(SHEET);
    render(<GbaseBoardView model={model} view={VIEW} onMoveCard={onMoveCard} moving />);
    const card = screen.getAllByTestId("gbase-card")[0]!;
    const lane = screen.getAllByTestId("gbase-lane")[1]!;
    fireEvent.dragStart(card);
    fireEvent.drop(lane);
    expect(onMoveCard).not.toHaveBeenCalled();
  });
});
