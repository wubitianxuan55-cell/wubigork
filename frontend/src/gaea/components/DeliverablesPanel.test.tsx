import { describe, expect, it, vi } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { DeliverablesPanel } from "./DeliverablesPanel";
import { ToastProvider } from "./Toast";
import { useComposerInsertStore, usePreviewStore, useUpdatedFilesStore } from "../lib/store";

describe("DeliverablesPanel 会话产物面板", () => {
  it("展示会话交付文件，点击打开预览", () => {
    usePreviewStore.setState({ previewFile: null });
    render(
      <DeliverablesPanel
        items={[
          { path: "exports/成本测算.xlsx", sourceId: "a1" },
          { path: ".gaea/exports/方案.docx", sourceId: "a2" },
        ]}
        onOpenFile={(p) => usePreviewStore.setState({ previewFile: p })}
      />,
    );
    expect(screen.getByText("成本测算.xlsx")).toBeTruthy();
    expect(screen.getByText("方案.docx")).toBeTruthy();
    fireEvent.click(screen.getByText("成本测算.xlsx"));
    expect(usePreviewStore.getState().previewFile).toBe("exports/成本测算.xlsx");
  });

  it("无交付文件时显示空状态", () => {
    render(<DeliverablesPanel items={[]} onOpenFile={() => {}} />);
    expect(screen.getByText(/暂无交付文件/)).toBeTruthy();
  });

  it("编辑过的文件显示「已更新」徽标", () => {
    useUpdatedFilesStore.setState({ updatedAt: { "exports/成本测算.xlsx": Date.now() } });
    render(
      <DeliverablesPanel
        items={[{ path: "exports/成本测算.xlsx", sourceId: "a1" }]}
        onOpenFile={() => {}}
      />,
    );
    expect(screen.getByText("已更新")).toBeTruthy();
    useUpdatedFilesStore.setState({ updatedAt: {} });
  });

  it("点击「跳转到生成它的消息」回调对应轮次", () => {
    const calls: number[] = [];
    render(
      <DeliverablesPanel
        items={[{ path: "exports/成本测算.xlsx", sourceId: "a1", turn: 2 }]}
        onOpenFile={() => {}}
        onLocateSource={(turn) => calls.push(turn)}
      />,
    );
    fireEvent.click(screen.getByTitle("跳转到生成它的消息"));
    expect(calls).toEqual([2]);
  });

  it("表格产物提供「沉淀到成本库」操作，指令进入输入框通道", () => {
    useComposerInsertStore.setState({ pendingText: null });
    render(
      <DeliverablesPanel
        items={[{ path: "exports/成本测算.xlsx", sourceId: "a1" }]}
        onOpenFile={() => {}}
      />,
    );
    fireEvent.click(screen.getByTitle("沉淀到成本库：把单价明细用 cost_save 写回成本库"));
    const text = useComposerInsertStore.getState().pendingText ?? "";
    expect(text).toContain("cost_save");
    expect(text).toContain("[成本测算.xlsx](exports/成本测算.xlsx)");
    useComposerInsertStore.getState().consumeText();
  });

  it("非表格产物不显示「沉淀到成本库」操作", () => {
    render(
      <DeliverablesPanel
        items={[{ path: ".gaea/exports/方案.docx", sourceId: "a2" }]}
        onOpenFile={() => {}}
      />,
    );
    expect(screen.queryByTitle("沉淀到成本库：把单价明细用 cost_save 写回成本库")).toBeNull();
  });

  it("图片产物渲染缩略图，非图片保留类型图标", async () => {
    const { container } = render(
      <DeliverablesPanel
        items={[
          { path: "exports/趋势.png", sourceId: "a1" },
          { path: ".gaea/exports/方案.docx", sourceId: "a2" },
        ]}
        onOpenFile={() => {}}
      />,
    );
    await waitFor(() => expect(container.querySelector("img")).toBeTruthy());
    const img = container.querySelector("img");
    expect(img?.getAttribute("src")).toContain("data:image/png");
    // 非图片产物不渲染 <img> 缩略图（仅一张图片，共两个产物）
    expect(container.querySelectorAll("img")).toHaveLength(1);
  });

  it("一键复制全部文件路径（最新在前）", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText } });
    render(
      <DeliverablesPanel
        items={[
          { path: "exports/成本测算.xlsx", sourceId: "a1" },
          { path: ".gaea/exports/方案.docx", sourceId: "a2" },
        ]}
        onOpenFile={() => {}}
      />,
    );
    fireEvent.click(screen.getByTitle("复制全部文件路径"));
    await waitFor(() =>
      expect(writeText).toHaveBeenCalledWith(".gaea/exports/方案.docx\nexports/成本测算.xlsx"),
    );
  });

  it("会话内多次出现的文件显示版本徽标（P1-2 产物版本时间线）", () => {
    render(
      <DeliverablesPanel
        items={[
          { path: "exports/周报.docx", sourceId: "a1", versions: 3 },
          { path: "exports/成本测算.xlsx", sourceId: "a2", versions: 1 },
        ]}
        onOpenFile={() => {}}
      />,
    );
    // 更新 3 次 → v3 徽标；仅 1 次不显示
    expect(screen.getByTitle("会话内更新了 3 次（产物版本时间线）")).toBeTruthy();
    expect(screen.getByText("v3")).toBeTruthy();
    expect(screen.queryByText("v1")).toBeNull();
  });

  it("一键打包下载全部交付文件（P0-1，对标 Kimi/WorkBuddy 会话产物打包）", async () => {
    const first = render(
      <ToastProvider>
        <DeliverablesPanel
          items={[
            { path: "exports/成本测算.xlsx", sourceId: "a1" },
            { path: ".gaea/exports/方案.docx", sourceId: "a2" },
          ]}
          onOpenFile={() => {}}
        />
      </ToastProvider>,
    );
    const btn = screen.getByTitle("打包下载：把本次会话全部交付文件打成一个 zip");
    expect(btn).toBeTruthy();
    // 点击后走 mock ZipDeliverables（返回 2 个条目）并触发定位，不应抛错
    fireEvent.click(btn);
    expect(await screen.findByText(/已打包 2 个文件/)).toBeTruthy();
    first.unmount();

    // 无产物时不显示打包按钮
    render(
      <ToastProvider>
        <DeliverablesPanel items={[]} onOpenFile={() => {}} />
      </ToastProvider>,
    );
    expect(screen.queryByTitle("打包下载：把本次会话全部交付文件打成一个 zip")).toBeNull();
  });
});

// ── v4.8 Verifier 产品化：证据链「三步展开」（卡面 → 声明↔实况 diff → 操作回放）──
// 证据数据来自 mock GaeaJournalList / VerifyRecord / RollbackRecord（office.ts），
// 实况预览来自 mock Preview 的 MOCK_XLSX_BODY（预算!B2=120.50、B4 公式 SUM(B2:B3)）。
describe("DeliverablesPanel 证据链三步展开（v4.8）", () => {
  const openEvidence = async () => {
    render(
      <ToastProvider>
        <DeliverablesPanel items={[{ path: "docs/成本测算.xlsx", sourceId: "a1" }]} onOpenFile={() => {}} />
      </ToastProvider>,
    );
    fireEvent.click(screen.getByText("证据链"));
    await screen.findByText("3 条变更证据卡");
  };

  it("展开证据链：渲染证据卡 + tool 徽标 + 「可复核明细」小徽标", async () => {
    await openEvidence();
    expect(screen.getAllByText("xlsx_apply")).toHaveLength(2); // 带 opsJson 卡 + 历史旧卡
    expect(screen.getByText("edit_file")).toBeTruthy();
    expect(screen.getByText("可复核明细")).toBeTruthy(); // 仅带 opsJson 的卡
    // 产物列表行也展示同一路径 → 至少 2 处（产物 + 证据卡）
    expect(screen.getAllByText("docs/成本测算.xlsx").length).toBeGreaterThanOrEqual(2);
    expect(screen.getByText("docs/说明.md")).toBeTruthy();
    expect(screen.getByText("docs/旧表.xlsx")).toBeTruthy();
  });

  it("无 baselinePath 的证据卡：回滚按钮 disabled + 提示「无基线快照，无法回滚」", async () => {
    await openEvidence();
    const blocked = screen.getAllByTitle("无基线快照，无法回滚");
    expect(blocked).toHaveLength(2); // edit_file 卡 + 历史 xlsx_apply 旧卡
    blocked.forEach((b) => expect((b as HTMLButtonElement).disabled).toBe(true));
    const ok = screen.getByTitle("用基线快照回滚（目标被手工修改时拒绝）");
    expect((ok as HTMLButtonElement).disabled).toBe(false);
  });

  it("展开 xlsx_apply 卡：第 1 层「声明↔实况」diff 渲染（mock GaeaPreview 实况）", async () => {
    await openEvidence();
    fireEvent.click(screen.getByRole("button", { name: "展开 xlsx_apply docs/成本测算.xlsx 证据详情" }));
    await screen.findByText("声明 ↔ 实况（前端近似比对）");
    // 声明列（删除线 before）| → | 实况列（after）：set_value B2 数值容差 match
    // （预算!B2 / 120.50 同时出现在 diff 表、回放影响区域与产物缩略图 → ≥1 处）
    expect(screen.getAllByText("预算!B2").length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText("120.5").length).toBeGreaterThanOrEqual(1); // 声明值
    expect(screen.getAllByText("120.50").length).toBeGreaterThanOrEqual(1); // 实况值（预览）
    // set_formula B4：公式 fx = 前缀，声明与实况一致
    expect(screen.getAllByText("fx =SUM(B2:B3)")).toHaveLength(2);
    // replace 批量格 → 跳过标注
    expect(screen.getByText("跳过")).toBeTruthy();
    expect(screen.getByText("前端近似比对，权威结论以复核为准")).toBeTruthy();
  });

  it("展开 xlsx_apply 卡：第 2 层操作回放时间线（描述文案 + 影响区域 + type 徽标）", async () => {
    await openEvidence();
    fireEvent.click(screen.getByRole("button", { name: "展开 xlsx_apply docs/成本测算.xlsx 证据详情" }));
    await screen.findByText("操作回放");
    expect(screen.getByText("写入值 B2=120.5")).toBeTruthy();
    expect(screen.getByText("写入公式 B4=SUM(B2:B3)")).toBeTruthy();
    expect(screen.getByText("替换 A1:A3：设备 → 机械")).toBeTruthy();
    expect(screen.getByText("set_value")).toBeTruthy();
    expect(screen.getByText("set_formula")).toBeTruthy();
    expect(screen.getByText("replace")).toBeTruthy();
    // 影响区域列（与 diff 表单元格同文案，出现 ≥1 次即可）
    expect(screen.getAllByText("预算!B2").length).toBeGreaterThanOrEqual(1);
  });

  it("旧卡无 opsJson：展开回退 beforeSummary 文本块（不渲染 diff/回放）", async () => {
    await openEvidence();
    fireEvent.click(screen.getByText("docs/旧表.xlsx"));
    await screen.findByText(/历史卡无 opsJson/);
    expect(screen.queryByText("操作回放")).toBeNull();
    expect(screen.queryByText("声明 ↔ 实况（前端近似比对）")).toBeNull();
  });

  it("复核证据卡：内联 verdict 徽标（复核通过）", async () => {
    await openEvidence();
    fireEvent.click(screen.getAllByTitle("双通道复核（结构/引用完整性 + 视觉健全性）")[0]);
    await screen.findByText("复核通过"); // ev_1003 → verified（徽标文本精确匹配）
    // 结论 note 同时出现在内联区与 toast（toast 前缀「复核通过：」）→ ≥1 处
    expect(screen.getAllByText(/（mock）双通道复核通过/).length).toBeGreaterThanOrEqual(1);
  });

  it("回滚成功路径：toast 透出后端成功文案，可继续复核（现状保留）", async () => {
    await openEvidence();
    fireEvent.click(screen.getByTitle("用基线快照回滚（目标被手工修改时拒绝）"));
    expect(await screen.findByText(/已回滚 docs\/成本测算\.xlsx/)).toBeTruthy();
    // 回滚后可再次复核（按钮仍在）
    expect(screen.getAllByTitle("双通道复核（结构/引用完整性 + 视觉健全性）").length).toBeGreaterThan(0);
  });
});
