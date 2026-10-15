// CostAuditView.test.tsx — 跨项目对标查漏面板（11.3）
// 契约：项目选择透传；语料不足显式宁缺勿误提示；发现渲染+溯源引用展开；
// 零发现成功态。
import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { CostAuditView } from "./CostAuditView";
import type { WorkcostAuditReport, WorkcostBillProject } from "../../lib/types";

const state = vi.hoisted(() => ({
  projects: [] as WorkcostBillProject[],
  scanCalls: [] as number[],
  report: null as WorkcostAuditReport | null,
  scanError: "",
}));

vi.mock("../../lib/bridge", () => ({
  app: {
    WorkcostBillProjects: async () => state.projects,
    CostAuditScan: async (projectID: number) => {
      state.scanCalls.push(projectID);
      if (state.scanError) throw new Error(state.scanError);
      if (!state.report) throw new Error("no report seeded");
      return state.report;
    },
  },
}));

import { LocaleProvider } from "../../lib/i18n";

const renderView = () =>
  render(
    <LocaleProvider>
      <CostAuditView />
    </LocaleProvider>,
  );

const ps: WorkcostBillProject[] = [
  { id: 1, name: "甲项目", fileName: "甲.xlsx", source: "", location: "", duration: "", pricing: "", managementRate: 0, regulatoryRate: 0, profitRate: 0, taxRate: 0, profitIncludesRegulatory: false, controlPrice: 0, itemCount: 5, createdAt: "", updatedAt: "" },
  { id: 2, name: "乙项目", fileName: "乙.xlsx", source: "", location: "", duration: "", pricing: "", managementRate: 0, regulatoryRate: 0, profitRate: 0, taxRate: 0, profitIncludesRegulatory: false, controlPrice: 0, itemCount: 5, createdAt: "", updatedAt: "" },
];

const fullReport: WorkcostAuditReport = {
  billGaps: [{
    kind: "bill_gap",
    target: "B支护降水",
    detail: "「B支护降水」分部在本项目缺失：4/4 个同行项目都有（占 100%），疑似漏项",
    evidence: [{ ref: "甲项目", source: "甲.xlsx", support: "「B支护降水」分部 2 项清单" }],
  }],
  quotaGaps: [{
    kind: "quota_gap",
    target: "污染土壤开挖（m3）",
    detail: "定额 WP-100-4「污染土壤开挖」未含资源 CL0001「水」：同名的 4 个定额里 3 个都含（占 75%），疑似缺组成行",
    evidence: [{ ref: "WP-100", source: "项目导入：语料.xlsx", support: "「污染土壤开挖」（m3）含 水" }],
  }],
  auditedProj: "甲项目",
  projectHits: 5,
};

beforeEach(() => {
  state.projects = ps;
  state.scanCalls = [];
  state.report = null;
  state.scanError = "";
});

describe("CostAuditView 对标查漏", () => {
  it("扫描透传项目 ID；发现分区渲染+溯源引用可展开", async () => {
    state.report = fullReport;
    renderView();
    const run = await screen.findByTestId("audit-run");
    fireEvent.click(run);
    await waitFor(() => expect(screen.getByTestId("audit-finding-bill_gap")).toBeDefined());
    expect(state.scanCalls).toEqual([1]);
    const text = document.body.textContent ?? "";
    expect(text).toContain("清单缺项");
    expect(text).toContain("4/4 个同行项目都有");
    expect(text).toContain("定额缺组成");
    expect(text).toContain("疑似缺组成行");
    // 溯源引用经 <details> 折叠披露（details 内容常驻 DOM，点击摘要展开可见）。
    fireEvent.click(screen.getAllByText(/溯源依据/)[0]);
    expect(document.body.textContent).toContain("「B支护降水」分部 2 项清单");
  });

  it("语料不足：宁缺勿误显式提示，不产发现", async () => {
    state.report = { billGaps: [], quotaGaps: [], auditedProj: "甲项目", projectHits: 2 };
    renderView();
    fireEvent.click(await screen.findByTestId("audit-run"));
    await waitFor(() => expect(document.body.textContent).toContain("宁缺勿误"));
    expect(document.body.textContent).toContain("少于 3 个");
  });

  it("零发现：成功态", async () => {
    state.report = { billGaps: [], quotaGaps: [], auditedProj: "甲项目", projectHits: 5 };
    renderView();
    fireEvent.click(await screen.findByTestId("audit-run"));
    await waitFor(() => expect(document.body.textContent).toContain("零发现"));
  });

  it("扫描失败：错误透出", async () => {
    state.scanError = "数据库未打开";
    renderView();
    fireEvent.click(await screen.findByTestId("audit-run"));
    await waitFor(() => expect(document.body.textContent).toContain("数据库未打开"));
  });
});
