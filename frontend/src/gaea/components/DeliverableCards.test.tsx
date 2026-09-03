import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { DeliverableCards } from "./DeliverableCards";
import { usePreviewStore, useUpdatedFilesStore } from "../lib/store";
import { LocaleProvider } from "../lib/i18n";
import { invalidateTurnCaches } from "../lib/deliverablesTurn";
import type { DeliverableEntry, DirEntry, SessionMeta } from "../lib/types";

// useT 需要 Provider；钉住 zh 让断言用中文文案（沿 Message.test.tsx 模式）。
const wrap = (node: React.ReactNode) => {
  localStorage.setItem("gaea-lang", "zh");
  return <LocaleProvider>{node}</LocaleProvider>;
};

// ── 登记/列目录绑定注入（realApp 按 Gaea* 名路由，走 window.go 门面）──
const sessionOf = (patch: Partial<SessionMeta>): SessionMeta => ({
  path: "s1.jsonl", preview: "", turns: 2, modTime: 0, current: false, ...patch,
});
const entryOf = (patch: Partial<DeliverableEntry>): DeliverableEntry => ({
  path: "p.md", tool: "write_file", turn: 1, updatedAt: 100, touches: 1, ...patch,
});
const dirEntry = (name: string): DirEntry => ({ name, isDir: false, size: 1 });

function injectBindings(impl: {
  ListSessions?: () => Promise<SessionMeta[]>;
  DeliverableRegistry?: () => Promise<unknown>;
  ListDir?: (dir: string) => Promise<DirEntry[]>;
}): Record<string, ReturnType<typeof vi.fn>> {
  const facade: Record<string, ReturnType<typeof vi.fn>> = {
    GaeaListSessions: vi.fn(impl.ListSessions ?? (() => Promise.resolve([sessionOf({ current: true })]))),
    GaeaDeliverableRegistry: vi.fn(impl.DeliverableRegistry ?? (() => Promise.resolve({ available: true, entries: [], total: 0 }))),
    GaeaListDir: vi.fn(impl.ListDir ?? (() => Promise.resolve([]))),
    // FileThumb 渲染缩略图会调 Preview（文档预览）与 AttachmentDataURL（图片），
    // 注入门面补齐防空引用（返回不可识别 kind → 组件静默回退图标）。
    GaeaPreview: vi.fn(() => Promise.resolve({ kind: "empty" })),
    AttachmentDataURL: vi.fn(() => Promise.resolve("")),
    GaeaAttachmentDataURL: vi.fn(() => Promise.resolve("")),
  };
  (window as unknown as Record<string, unknown>).go = { app: { TestB: facade } };
  return facade;
}

afterEach(() => {
  delete (window as unknown as Record<string, unknown>).go;
});

describe("DeliverableCards 交付文件卡片", () => {
  beforeEach(() => {
    invalidateTurnCaches();
  });

  it("把正文中的办公文件渲染为可点击预览卡片并去重", () => {
    usePreviewStore.setState({ previewFile: null });
    render(
      wrap(
        <DeliverableCards
          text="已生成：exports/成本测算.xlsx，方案见 .gaea/exports/方案.docx，图表 exports/趋势.png。再次提到 exports/成本测算.xlsx。"
        />,
      ),
    );
    const cards = screen.getAllByRole("button", { name: /成本测算\.xlsx|方案\.docx|趋势\.png/ });
    expect(cards).toHaveLength(3);

    fireEvent.click(screen.getByRole("button", { name: /成本测算\.xlsx/ }));
    expect(usePreviewStore.getState().previewFile).toBe("exports/成本测算.xlsx");
  });

  it("代码文件不算交付物，纯文本无文件时不渲染", () => {
    const { container } = render(wrap(<DeliverableCards text="已生成 exports/main.ts，无其他交付" />));
    expect(container.querySelectorAll("button")).toHaveLength(0);
  });

  it("预览内编辑过的文件显示「已更新」徽标", () => {
    useUpdatedFilesStore.setState({ updatedAt: { "exports/成本测算.xlsx": Date.now() } });
    render(wrap(<DeliverableCards text="已生成：exports/成本测算.xlsx" />));
    expect(screen.getByText("已更新")).toBeTruthy();
    useUpdatedFilesStore.setState({ updatedAt: {} });
  });

  // ── 权威登记表合并（后端 Turn = 前端 turnNo + 1）──
  it("正文未提及但登记表本轮有 → 渲染出该文件卡（正文卡在前、登记卡在后）", async () => {
    const bindings = injectBindings({
      DeliverableRegistry: () =>
        Promise.resolve({
          available: true,
          total: 3,
          entries: [
            entryOf({ path: "reports/漏提的方案.docx", turn: 2, updatedAt: 300 }), // 本轮（turnNo=1 → Turn 2）
            entryOf({ path: "exports/上一轮.md", turn: 1, updatedAt: 200 }), // 前一轮：不并入
            entryOf({ path: "exports/图表.png", turn: 2, updatedAt: 250 }), // 正文已提及：去重
          ],
        }),
      // 缺失态探测：两个文件都在（mock 忽略目录参数），不出「未生成」徽标
      ListDir: () => Promise.resolve([dirEntry("漏提的方案.docx"), dirEntry("图表.png")]),
    });
    const view = render(
      wrap(<DeliverableCards text="已生成：exports/图表.png，详见 exports/成本测算.xlsx。" turnNo={1} />),
    );
    // 漏提文件补出卡片；前一轮/正文已提及的不重复出现
    await screen.findByText("漏提的方案.docx");
    expect(screen.queryByText("上一轮.md")).toBeNull();
    const textIdx = view.container.textContent!.indexOf("成本测算.xlsx");
    const regIdx = view.container.textContent!.indexOf("漏提的方案.docx");
    expect(textIdx).toBeGreaterThanOrEqual(0);
    expect(regIdx).toBeGreaterThan(textIdx);
    // 计数徽标 = 正文 2 张 + 登记-only 1 张
    expect(screen.getByText("3")).toBeTruthy();
    // 目录探测（缺失态）随登记拉取而触发，且确认存在 → 无徽标
    await waitFor(() => expect(bindings.GaeaListDir).toHaveBeenCalled());
    expect(screen.queryByText("未生成")).toBeNull();
  });

  it("登记-only 且列目录确认不存在 → 灰色淡化 + 「未生成」徽标", async () => {
    injectBindings({
      DeliverableRegistry: () =>
        Promise.resolve({
          available: true,
          total: 2,
          entries: [
            entryOf({ path: "reports/写失败的.docx", turn: 1 }),
            entryOf({ path: "reports/写成了的.docx", turn: 1, updatedAt: 200 }),
          ],
        }),
      ListDir: () => Promise.resolve([dirEntry("写成了的.docx"), dirEntry("子目录")]),
    });
    const view = render(wrap(<DeliverableCards text="处理完毕，结果如下。" turnNo={0} />));
    expect(await screen.findByText("写失败的.docx")).toBeTruthy();
    // 确认不存在的登记卡标「未生成」；存在的与正文卡（无登记路径）不标
    expect(screen.getByText("未生成")).toBeTruthy();
    const missingRow = screen.getByText("写失败的.docx").closest("div.group");
    expect(missingRow?.getAttribute("style")).toContain("opacity: 0.55");
    const okRow = screen.getByText("写成了的.docx").closest("div.group");
    expect(okRow?.getAttribute("style") ?? "").not.toContain("opacity");
    expect(view.container.textContent).toContain("2");
  });
});
