import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import type { ReactElement } from "react";
import { DeliverablesPanel } from "./DeliverablesPanel";
import { ToastProvider } from "./Toast";
import { LocaleProvider } from "../lib/i18n";
import { useComposerInsertStore, usePreviewStore, useUpdatedFilesStore } from "../lib/store";

// DeliverablesPanel 走 useT；钉住 zh 让既有中文文案断言继续成立（默认 zh，
// i18n 抽查用例可显式传 "en" 验证英文键值）
const renderT = (ui: ReactElement, lang: "zh" | "en" | "zh-TW" = "zh") => {
  localStorage.setItem("gaea-lang", lang);
  return render(<LocaleProvider>{ui}</LocaleProvider>);
};

describe("DeliverablesPanel 会话产物面板", () => {
  it("展示会话交付文件，点击打开预览", () => {
    usePreviewStore.setState({ previewFile: null });
    renderT(
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
    renderT(<DeliverablesPanel items={[]} onOpenFile={() => {}} />);
    expect(screen.getByText(/暂无交付文件/)).toBeTruthy();
  });

  it("编辑过的文件显示「已更新」徽标", () => {
    useUpdatedFilesStore.setState({ updatedAt: { "exports/成本测算.xlsx": Date.now() } });
    renderT(
      <DeliverablesPanel
        items={[{ path: "exports/成本测算.xlsx", sourceId: "a1" }]}
        onOpenFile={() => {}}
      />,
    );
    expect(screen.getByText("已更新")).toBeTruthy();
    useUpdatedFilesStore.setState({ updatedAt: {} });
  });

  it("v4.30 新产物显示「新」徽标（freshPaths 命中行，其余行不显示）", () => {
    renderT(
      <DeliverablesPanel
        items={[
          { path: "exports/成本测算.xlsx", sourceId: "a1" },
          { path: ".gaea/exports/方案.docx", sourceId: "a2" },
        ]}
        onOpenFile={() => {}}
        freshPaths={[".gaea/exports/方案.docx"]}
      />,
    );
    // 命中行：名称旁有「新」徽标
    const freshName = screen.getByText("方案.docx");
    const freshRow = freshName.closest("div[data-fresh]");
    expect(freshRow).not.toBeNull();
    expect(freshRow!.getAttribute("data-fresh")).toBe("true");
    expect(screen.getByText("新")).toBeTruthy();
    // 未命中行：无 data-fresh 锚点
    const otherName = screen.getByText("成本测算.xlsx");
    expect(otherName.closest("div[data-fresh]")).toBeNull();
  });

  it("v4.30 freshPaths 缺省/空时不显示「新」徽标", () => {
    renderT(
      <DeliverablesPanel
        items={[{ path: "exports/成本测算.xlsx", sourceId: "a1" }]}
        onOpenFile={() => {}}
      />,
    );
    expect(screen.queryByText("新")).toBeNull();
  });

  it("点击「跳转到生成它的消息」回调对应轮次", () => {
    const calls: number[] = [];
    renderT(
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
    renderT(
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
    renderT(
      <DeliverablesPanel
        items={[{ path: ".gaea/exports/方案.docx", sourceId: "a2" }]}
        onOpenFile={() => {}}
      />,
    );
    expect(screen.queryByTitle("沉淀到成本库：把单价明细用 cost_save 写回成本库")).toBeNull();
  });

  // ── v4.25 A3 树中定位：产物行「树中定位」小按钮（→ 文件 tab 树中闪烁）──
  it("onRevealInTree 直传：点击「树中定位」回调产物相对路径", () => {
    const onRevealInTree = vi.fn();
    renderT(
      <DeliverablesPanel
        items={[{ path: "exports/成本测算.xlsx", sourceId: "a1" }]}
        onOpenFile={() => {}}
        onRevealInTree={onRevealInTree}
      />,
    );
    fireEvent.click(screen.getByTitle("树中定位：在文件树中展开并高亮该文件"));
    expect(onRevealInTree).toHaveBeenCalledTimes(1);
    expect(onRevealInTree).toHaveBeenCalledWith("exports/成本测算.xlsx");
  });

  it("未传 onRevealInTree：不渲染「树中定位」按钮（向后兼容）", () => {
    renderT(
      <DeliverablesPanel
        items={[{ path: "exports/成本测算.xlsx", sourceId: "a1" }]}
        onOpenFile={() => {}}
      />,
    );
    expect(screen.queryByTitle("树中定位：在文件树中展开并高亮该文件")).toBeNull();
  });

  it("图片产物渲染缩略图，非图片保留类型图标", async () => {
    const { container } = renderT(
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
    expect(img?.getAttribute("src")).toMatch(/^data:image\//);
    // 非图片产物不渲染 <img> 缩略图（仅一张图片，共两个产物）
    expect(container.querySelectorAll("img")).toHaveLength(1);
  });

  it("一键复制全部文件路径（最新在前）", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText } });
    renderT(
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
    renderT(
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
    const first = renderT(
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
    renderT(
      <ToastProvider>
        <DeliverablesPanel items={[]} onOpenFile={() => {}} />
      </ToastProvider>,
    );
    expect(screen.queryByTitle("打包下载：把本次会话全部交付文件打成一个 zip")).toBeNull();
  });
});

// ── v4.28 B1 文件版本时间线：vN 徽标可点 → 内联 VersionTimeline（预览/恢复）──
// 数据为 panel 挂载时自拉的 mock GaeaJournalList(200)：3 条证据卡中仅 ev_1003
// （docs/成本测算.xlsx，baselinePath=.gaea/snapshots/…）有基线快照可进时间线；
// 另两条无 baselinePath，被 groupVersionsByPath 过滤。
describe("DeliverablesPanel 版本时间线（v4.28 B1）", () => {
  const renderWithTimelineItem = (onOpenFile: (p: string) => void = () => {}) =>
    renderT(
      <ToastProvider>
        <DeliverablesPanel
          items={[{ path: "docs/成本测算.xlsx", sourceId: "a1", versions: 2 }]}
          onOpenFile={onOpenFile}
        />
      </ToastProvider>,
    );

  it("vN 徽标可点：点开内联版本时间线，只显示有基线快照的版本记录；再点收起", async () => {
    renderWithTimelineItem();
    // 初始收起
    expect(screen.queryByTestId("version-timeline")).toBeNull();
    fireEvent.click(screen.getByTitle("会话内更新了 2 次（产物版本时间线）"));
    // mock 3 条证据卡中仅 ev_1003 带 baselinePath → 时间线 1 行
    const rows = await screen.findAllByTestId("version-timeline-row");
    expect(rows).toHaveLength(1);
    expect(rows[0].textContent).toContain("xlsx_apply");
    // 恢复语义常驻说明
    expect(screen.getByText("恢复会把该文件写回所选版本，当前内容成为新版本")).toBeTruthy();
    // 再次点击徽标 → 收起
    fireEvent.click(screen.getByTitle("会话内更新了 2 次（产物版本时间线）"));
    expect(screen.queryByTestId("version-timeline")).toBeNull();
  });

  it("预览回调：点「预览」用基线快照路径打开文件预览", async () => {
    usePreviewStore.setState({ previewFile: null });
    renderWithTimelineItem((p) => usePreviewStore.setState({ previewFile: p }));
    fireEvent.click(screen.getByTitle("会话内更新了 2 次（产物版本时间线）"));
    fireEvent.click(await screen.findByTitle("预览该版本快照"));
    await waitFor(() =>
      expect(usePreviewStore.getState().previewFile).toBe(".gaea/snapshots/docs/成本测算.xlsx.snap"),
    );
  });

  it("恢复回调：点「恢复」走 RollbackRecord，成功 toast 透出恢复语义", async () => {
    renderWithTimelineItem();
    fireEvent.click(screen.getByTitle("会话内更新了 2 次（产物版本时间线）"));
    fireEvent.click(await screen.findByTitle(/^恢复到 \d{2}:\d{2} 版本：将回滚到该时间版本$/));
    // toast 文案带恢复语义（「当前内容成为新版本」；时间线说明也含该句，故用完整正则单匹配 toast）
    expect(
      await screen.findByText(/已恢复 docs\/成本测算\.xlsx 到 \d{2}:\d{2} · \S+ 版本；当前内容成为新版本/),
    ).toBeTruthy();
  });

  it("无基线快照的产物：时间线展示空态，不渲染预览/恢复按钮", async () => {
    renderT(
      <ToastProvider>
        <DeliverablesPanel
          items={[{ path: "exports/别的.docx", sourceId: "a9", versions: 2 }]}
          onOpenFile={() => {}}
        />
      </ToastProvider>,
    );
    fireEvent.click(screen.getByTitle("会话内更新了 2 次（产物版本时间线）"));
    expect(await screen.findByTestId("version-timeline-empty")).toBeTruthy();
    expect(screen.queryByTestId("version-timeline-row")).toBeNull();
    expect(screen.queryByTitle("预览该版本快照")).toBeNull();
  });
});

// ── v4.31 A1 单版本时间线入口：versions≤1 但 journal 有该路径快照时，产物行
// 也渲染「版本」入口徽标（收 v4.28 欠账「B1 单版本无入口」）；无快照不渲染。
// mock GaeaJournalList(200) 中仅 ev_1003（docs/成本测算.xlsx，有 baselinePath）
// 可进时间线，其余卡被 groupVersionsByPath 过滤。
// v4.32：非 rev 分支 title 从静态文案细化为带快照数（mock 中 n=1 →
// 「有 1 个历史快照，可预览/恢复」），原静态文案仅作 n 取不到时的回落。
describe("DeliverablesPanel 单版本时间线入口（v4.31 A1）", () => {
  it("versions=1 且 journal 有快照：渲染「版本」入口，点击展开时间线，预览/恢复可用", async () => {
    renderT(
      <ToastProvider>
        <DeliverablesPanel
          items={[{ path: "docs/成本测算.xlsx", sourceId: "a1", versions: 1 }]}
          onOpenFile={() => {}}
        />
      </ToastProvider>,
    );
    // 初始收起；等待 journal 异步就绪后入口徽标出现
    expect(screen.queryByTestId("version-timeline")).toBeNull();
    const entry = await screen.findByTitle("有 1 个历史快照，可预览/恢复");
    expect(entry.textContent).toContain("版本");
    expect(screen.queryByText("v1")).toBeNull(); // 单版本入口用「版本」而非次数徽标
    fireEvent.click(entry);
    const rows = await screen.findAllByTestId("version-timeline-row");
    expect(rows).toHaveLength(1); // mock 中 docs/成本测算.xlsx 仅 ev_1003 带基线快照
    // 单条记录下预览/恢复按钮可用（有 baselinePath）
    expect((screen.getByTitle("预览该版本快照") as HTMLButtonElement).disabled).toBe(false);
    expect(screen.getByTitle(/^恢复到 \d{2}:\d{2} 版本：将回滚到该时间版本$/)).toBeTruthy();
  });

  it("versions 省略（undefined）且 journal 有快照：同样渲染单版本入口并展开", async () => {
    renderT(
      <ToastProvider>
        <DeliverablesPanel
          items={[{ path: "docs/成本测算.xlsx", sourceId: "a1" }]}
          onOpenFile={() => {}}
        />
      </ToastProvider>,
    );
    const entry = await screen.findByTitle("有 1 个历史快照，可预览/恢复");
    fireEvent.click(entry);
    expect(await screen.findAllByTestId("version-timeline-row")).toHaveLength(1);
    expect(screen.getByText("恢复会把该文件写回所选版本，当前内容成为新版本")).toBeTruthy();
  });

  it("无快照（journal 无该路径）：不渲染时间线入口（有快照的行正常渲染）", async () => {
    renderT(
      <ToastProvider>
        <DeliverablesPanel
          items={[
            { path: "docs/成本测算.xlsx", sourceId: "a1", versions: 1 },
            { path: "exports/无快照.docx", sourceId: "a2", versions: 1 },
          ]}
          onOpenFile={() => {}}
        />
      </ToastProvider>,
    );
    // 等 journal 就绪（有快照行出现入口）后再断言：入口仅 1 个，无快照行不渲染
    await screen.findByTitle("有 1 个历史快照，可预览/恢复");
    expect(screen.getAllByTitle("有 1 个历史快照，可预览/恢复")).toHaveLength(1);
    const noSnapshotRow = screen.getByText("无快照.docx").closest(".group");
    expect(noSnapshotRow).not.toBeNull();
    expect(noSnapshotRow!.querySelector('button[title="有 1 个历史快照，可预览/恢复"]')).toBeNull();
    expect(screen.getByText("成本测算.xlsx")).toBeTruthy();
  });
});

// ── v4.32：单版本「版本」徽标 title 带快照数（收 v4.31 欠账「静态文案」）──
// mock GaeaJournalList(200) 中 docs/成本测算.xlsx 仅 ev_1003 有基线快照 → n=1。
describe("DeliverablesPanel 单版本徽标 title 带快照数（v4.32）", () => {
  it("title 细化为「有 1 个历史快照，可预览/恢复」（含具体快照数）", async () => {
    renderT(
      <ToastProvider>
        <DeliverablesPanel
          items={[{ path: "docs/成本测算.xlsx", sourceId: "a1", versions: 1 }]}
          onOpenFile={() => {}}
        />
      </ToastProvider>,
    );
    const entry = await screen.findByTitle(/个历史快照，可预览\/恢复/);
    expect(entry.getAttribute("title")).toBe("有 1 个历史快照，可预览/恢复");
  });
});

// ── v4.32 线B 产物自动弹出偏好：头部「自动弹出」胶囊（默认关 opt-in；对标
// browserAutoOpen 默认开的差异——产物更新更频繁，抢焦点代价更高）。持久化键
// gaea.deliverableAutoOpen；触发接线在 App（新产物 diff effect 调
// shouldAutoOpenDeliverables），面板只负责开关 UI。形状对齐 BrowserPanel 胶囊
// （aria-pressed 翻转 + 先落盘再变亮）。
describe("DeliverablesPanel 自动弹出胶囊（v4.32 线B）", () => {
  const KEY = "gaea.deliverableAutoOpen";

  afterEach(() => {
    try { localStorage.removeItem(KEY); } catch { /* ignore */ }
  });

  const renderPanel = () =>
    renderT(
      <DeliverablesPanel
        items={[{ path: "exports/成本测算.xlsx", sourceId: "a1" }]}
        onOpenFile={() => {}}
      />,
    );

  it("默认灰态（关）：aria-pressed=false + 关态 title，未写 localStorage", () => {
    renderPanel();
    const pill = screen.getByTestId("deliverable-auto-open-toggle");
    expect(pill.getAttribute("aria-pressed")).toBe("false");
    expect(pill.getAttribute("title")).toBe(
      "自动弹出已关：新产物出现时不切换面板，仅列表内「新」徽标提示（点击开启）",
    );
    expect(pill.textContent).toContain("自动弹出 关");
    expect(localStorage.getItem(KEY)).toBeNull();
  });

  it("点击开启：先落盘 1 再变亮（aria-pressed=true），title 反向说明", () => {
    renderPanel();
    const pill = screen.getByTestId("deliverable-auto-open-toggle");
    fireEvent.click(pill);
    expect(localStorage.getItem(KEY)).toBe("1");
    expect(pill.getAttribute("aria-pressed")).toBe("true");
    expect(pill.getAttribute("title")).toBe("自动弹出已开：新产物出现时自动切到本面板（点击关闭）");
    expect(pill.textContent).toContain("自动弹出 开");
  });

  it("已存偏好（1）启动即亮；再点关闭落盘 0 回灰态", () => {
    try { localStorage.setItem(KEY, "1"); } catch { /* ignore */ }
    renderPanel();
    const pill = screen.getByTestId("deliverable-auto-open-toggle");
    expect(pill.getAttribute("aria-pressed")).toBe("true");
    fireEvent.click(pill);
    expect(localStorage.getItem(KEY)).toBe("0");
    expect(pill.getAttribute("aria-pressed")).toBe("false");
  });
});

// ── v4.8 Verifier 产品化：证据链「三步展开」（卡面 → 声明↔实况 diff → 操作回放）──
// 证据数据来自 mock GaeaJournalList / VerifyRecord / RollbackRecord（office.ts），
// 实况预览来自 mock Preview 的 MOCK_XLSX_BODY（预算!B2=120.50、B4 公式 SUM(B2:B3)）。
describe("DeliverablesPanel 证据链三步展开（v4.8）", () => {
  const openEvidence = async () => {
    renderT(
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

  // ── v4.16 通道 B 结果产品化：verdict 携带像素差异率时渲染「视觉复核」行 ──
  it("通道 B 结果进前端：渲染像素差异率行 + 查看复核产物按钮", async () => {
    await openEvidence();
    // ev_1003（mock 携带 channelBRatio 0.013 / 3 页 / 产物目录）
    fireEvent.click(screen.getAllByTitle("双通道复核（结构/引用完整性 + 视觉健全性）")[0]);
    await screen.findByText("视觉复核：像素差异率 1.3% · 3 页");
    // 产物目录存在 → 「查看复核产物」按钮（OpenWorkspacePath 打开目录）
    expect(screen.getByTitle("查看复核产物（before/after PDF + 逐页 PNG）")).toBeTruthy();
  });

  it("旧 verdict / 无通道 B：不渲染「视觉复核」行（向后兼容）", async () => {
    await openEvidence();
    // ev_1001（mock 保持旧形态，无 channelBRatio/channelBPages/channelBArtifacts）
    fireEvent.click(screen.getAllByTitle("双通道复核（结构/引用完整性 + 视觉健全性）")[2]);
    await screen.findByText("复核警告");
    expect(screen.queryByText(/视觉复核/)).toBeNull();
    expect(screen.queryByTitle("查看复核产物（before/after PDF + 逐页 PNG）")).toBeNull();
  });
});

// ── v4.24 C1 权威产物登记表（后端事件日志折叠，前端只读）──
// 数据来自 mock DeliverableRegistry（office.ts）：3 条登记
// （write_file / format_convert / diagram_gen，含启发式漏登的 svg/xlsx）。
describe("DeliverablesPanel 权威产物登记表（v4.24 C1）", () => {
  it("有 sessionPath：渲染登记表入口，展开展示工具徽标/路径/轮次", async () => {
    renderT(
      <ToastProvider>
        <DeliverablesPanel items={[]} sessionPath="s1.jsonl" onOpenFile={() => {}} />
      </ToastProvider>,
    );
    // 入口收起态：标题 + 计数（3 条落盘登记）
    expect(await screen.findByText("权威产物登记")).toBeTruthy();
    expect(screen.getByText("3 条落盘登记")).toBeTruthy();
    // 展开 → 登记条目（tool 徽标 + 路径 + 轮次）
    fireEvent.click(screen.getByText("权威产物登记"));
    const rows = await screen.findAllByTestId("deliverable-registry-row");
    expect(rows).toHaveLength(3);
    expect(screen.getByText("docs/竞品调研报告.md")).toBeTruthy();
    expect(screen.getByText(".gaea/exports/表格方案-mock.xlsx")).toBeTruthy();
    expect(screen.getByText("docs/架构图.svg")).toBeTruthy();
    expect(screen.getByText("write_file")).toBeTruthy();
    expect(screen.getByText("format_convert")).toBeTruthy();
    expect(screen.getByText("diagram_gen")).toBeTruthy();
    // mock 登记条目轮次为 1/3/4（office.ts 演示会话唯一轮对齐 turn:1）
    expect(screen.getByText("第 1 轮")).toBeTruthy();
    expect(screen.getByText("第 4 轮")).toBeTruthy();
  });

  it("登记条目点击 → 打开对应文件预览", async () => {
    usePreviewStore.setState({ previewFile: null });
    renderT(
      <ToastProvider>
        <DeliverablesPanel items={[]} sessionPath="s1.jsonl" onOpenFile={(p) => usePreviewStore.setState({ previewFile: p })} />
      </ToastProvider>,
    );
    fireEvent.click(await screen.findByText("权威产物登记"));
    fireEvent.click(await screen.findByText("docs/架构图.svg"));
    expect(usePreviewStore.getState().previewFile).toBe("docs/架构图.svg");
  });

  it("无 sessionPath：不拉取登记表、不渲染入口（向后兼容）", () => {
    renderT(<DeliverablesPanel items={[]} onOpenFile={() => {}} />);
    expect(screen.queryByText("权威产物登记")).toBeNull();
  });
});

// ── 三语字典接线抽查：面板文案走 deliverPanel.* 键（zh 钉住 / en 抽查）──
describe("DeliverablesPanel i18n（deliverPanel.* 键）", () => {
  it("zh：面板标题与空态文案来自字典", () => {
    renderT(<DeliverablesPanel items={[]} onOpenFile={() => {}} />);
    expect(screen.getByText("会话产物")).toBeTruthy();
    // 空态 span 含 <br/> 分隔的两段文本 → 用正则匹配
    expect(screen.getByText(/本轮会话暂无交付文件/)).toBeTruthy();
    expect(screen.getByText(/生成\/保存文件后会出现在这里/)).toBeTruthy();
  });

  it("en：切到 en 后标题/空态/证据链入口走英文键值（结束还原 zh）", () => {
    renderT(<DeliverablesPanel items={[]} onOpenFile={() => {}} />, "en");
    expect(screen.getByText("Session deliverables")).toBeTruthy();
    expect(screen.getByText(/No deliverables in this session yet/)).toBeTruthy();
    expect(screen.getByText("Evidence chain")).toBeTruthy();
    localStorage.setItem("gaea-lang", "zh"); // 还原，避免影响其他用例
  });
});

// ── A1 交付验收闭环：产物行验收徽标 + 悬停标记操作 + 头部「已验收 n/m」汇总 ──
// 数据层 lib/deliverableStatus.ts（键 gaea.deliverableAcceptance.v1）。徽标只显
// 示已验收/要求修改两态（open 为缺省态不显示，视觉最安静）；「重新查看」恢复
// open（记录删除，操作位复原为标记已验收/要求修改）。
describe("DeliverablesPanel 验收状态（A1 交付验收闭环）", () => {
  const KEY = "gaea.deliverableAcceptance.v1";

  beforeEach(() => {
    try { localStorage.removeItem(KEY); } catch { /* ignore */ }
  });
  afterEach(() => {
    try { localStorage.removeItem(KEY); } catch { /* ignore */ }
  });

  const renderAcceptPanel = () =>
    renderT(
      <DeliverablesPanel
        items={[
          { path: "exports/成本测算.xlsx", sourceId: "a1" },
          { path: ".gaea/exports/方案.docx", sourceId: "a2" },
        ]}
        sessionPath="acc-sess.jsonl"
        onOpenFile={() => {}}
      />,
    );

  // 行作用域取按钮：产物行 .group 容器内按 title 定位（多行同款按钮互不干扰）
  const rowBtn = (rowText: string, title: string) => {
    const row = screen.getByText(rowText).closest(".group");
    expect(row).not.toBeNull();
    return row!.querySelector(`button[title="${title}"]`);
  };

  const savedMap = () =>
    JSON.parse(localStorage.getItem(KEY) ?? "{}") as Record<
      string,
      { status: string; at: number; versionAt: number }
    >;

  it("缺省 open：行内无徽标，悬停操作为「标记已验收」「要求修改」，头部汇总 0/2", () => {
    renderAcceptPanel();
    expect(screen.queryByText("已验收")).toBeNull(); // 行内徽标无（汇总文案为「已验收 0/2」，全文不精确等于「已验收」）
    expect(screen.queryByText("要求修改")).toBeNull();
    expect(screen.getByText("已验收 0/2")).toBeTruthy();
    expect(rowBtn("成本测算.xlsx", "标记已验收")).toBeTruthy();
    expect(rowBtn("成本测算.xlsx", "要求修改")).toBeTruthy();
  });

  it("标记已验收：徽标出现 + localStorage 写入 confirmed 记录 + 汇总计数前进 + 操作位换「重新查看」", () => {
    renderAcceptPanel();
    fireEvent.click(rowBtn("成本测算.xlsx", "标记已验收") as HTMLElement);
    // 徽标（精确文本「已验收」，区别于汇总「已验收 1/2」）
    expect(screen.getByText("已验收")).toBeTruthy();
    expect(screen.getByText("已验收 1/2")).toBeTruthy();
    // localStorage：statusKeyOf 归一化键 → confirmed 记录；未登记路径 versionAt 兜底 0
    const rec = savedMap()["acc-sess.jsonl::exports/成本测算.xlsx"];
    expect(rec?.status).toBe("confirmed");
    expect(rec?.versionAt).toBe(0);
    expect(rec?.at).toBeGreaterThan(0);
    // 已标记行操作位换「重新查看」
    expect(rowBtn("成本测算.xlsx", "重新查看")).toBeTruthy();
    expect(rowBtn("成本测算.xlsx", "标记已验收")).toBeNull();
  });

  it("要求修改：徽标变「要求修改」且 localStorage 记 redo；汇总只数已验收", () => {
    renderAcceptPanel();
    fireEvent.click(rowBtn("方案.docx", "要求修改") as HTMLElement);
    expect(screen.getByText("要求修改")).toBeTruthy();
    expect(savedMap()["acc-sess.jsonl::.gaea/exports/方案.docx"].status).toBe("redo");
    expect(screen.getByText("已验收 0/2")).toBeTruthy();
  });

  it("重新查看：恢复 open（徽标消失、记录删除、操作位复原）", () => {
    renderAcceptPanel();
    fireEvent.click(rowBtn("成本测算.xlsx", "标记已验收") as HTMLElement);
    expect(screen.getByText("已验收")).toBeTruthy();
    fireEvent.click(rowBtn("成本测算.xlsx", "重新查看") as HTMLElement);
    expect(screen.queryByText("已验收")).toBeNull(); // open 不显示徽标
    expect(savedMap()["acc-sess.jsonl::exports/成本测算.xlsx"]).toBeUndefined();
    expect(rowBtn("成本测算.xlsx", "标记已验收")).toBeTruthy();
    expect(rowBtn("成本测算.xlsx", "要求修改")).toBeTruthy();
    expect(screen.getByText("已验收 0/2")).toBeTruthy();
  });

  it("持久化：重挂载后徽标保持（面板级 loadAcceptanceMap 初始化）", () => {
    const first = renderAcceptPanel();
    fireEvent.click(rowBtn("成本测算.xlsx", "标记已验收") as HTMLElement);
    first.unmount();
    renderAcceptPanel();
    expect(screen.getByText("已验收")).toBeTruthy();
    expect(screen.getByText("已验收 1/2")).toBeTruthy();
  });

  it("versionAt 取登记表条目 updatedAt（标记时所见版本）；重挂载后登记表未前进仍保持已验收", async () => {
    const first = renderT(
      <DeliverablesPanel
        items={[{ path: "docs/架构图.svg", sourceId: "a1" }]}
        sessionPath="s1.jsonl"
        onOpenFile={() => {}}
      />,
    );
    // 等 mock DeliverableRegistry 就绪（登记表入口出现）再标记，确保 versionAt 取到登记表 updatedAt
    await screen.findByText("权威产物登记");
    fireEvent.click(rowBtn("架构图.svg", "标记已验收") as HTMLElement);
    expect(screen.getByText("已验收")).toBeTruthy();
    expect(savedMap()["s1.jsonl::docs/架构图.svg"].versionAt).toBe(1754439600); // mock 登记表 updatedAt
    first.unmount();
    // 登记表 updatedAt 未前进（=versionAt）→ 不触发新版本重置，徽标保持
    renderT(
      <DeliverablesPanel
        items={[{ path: "docs/架构图.svg", sourceId: "a1" }]}
        sessionPath="s1.jsonl"
        onOpenFile={() => {}}
      />,
    );
    await screen.findByText("权威产物登记");
    expect(screen.getByText("已验收")).toBeTruthy();
    expect(screen.getByText("已验收 1/1")).toBeTruthy();
  });
});
