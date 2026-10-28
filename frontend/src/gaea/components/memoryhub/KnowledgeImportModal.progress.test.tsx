import { afterEach, describe, expect, it, vi } from "vitest";
import { act, cleanup, render, screen } from "@testing-library/react";
import { KnowledgeImportModal } from "./KnowledgeImportModal";
import { ToastProvider } from "../Toast";

// 扫描件导入长达分钟级/页（uocr2）：解析期间后端逐页广播
// knowledge-import-progress，弹窗 loading 态须显示「N/总页」进度而非哑转。

const { previewSpy } = vi.hoisted(() => ({ previewSpy: vi.fn() }));

vi.mock("../../lib/bridge", () => ({
  app: {
    KnowledgeImportPreview: (...args: unknown[]) => previewSpy(...args),
    KnowledgeImportAIParse: async () => ({ rows: [] }),
    KnowledgeImportApply: async () => 0,
  },
}));

const wrap = (node: React.ReactNode) => <ToastProvider>{node}</ToastProvider>;

describe("KnowledgeImportModal OCR 进度可见", () => {
  afterEach(() => {
    cleanup();
    delete (window as { runtime?: unknown }).runtime;
    vi.restoreAllMocks();
  });

  it("loading 期间收到进度事件显示「正在 OCR 识别（N/总页）」", async () => {
    let fire: ((data: unknown) => void) | null = null;
    (window as { runtime?: unknown }).runtime = {
      EventsOn: (_ev: string, cb: (data: unknown) => void) => {
        fire = cb;
        return () => {
          fire = null;
        };
      },
    };
    previewSpy.mockReturnValue(new Promise(() => {})); // 挂起 = 解析进行中

    render(
      wrap(
        <KnowledgeImportModal
          open
          path="C:/tmp/扫描件.pdf"
          fileName="扫描件.pdf"
          onClose={() => {}}
          onImported={() => {}}
        />,
      ),
    );
    expect(screen.getByText("解析中…")).toBeTruthy();

    act(() => {
      fire?.({ path: "C:/tmp/扫描件.pdf", done: 3, total: 15 });
    });
    expect(screen.getByText(/正在 OCR 识别（3\/15 页）/)).toBeTruthy();
    expect(screen.queryByText("解析中…")).toBeNull();
  });

  it("无进度事件时保持「解析中…」（非 OCR 文档不发事件）", () => {
    (window as { runtime?: unknown }).runtime = {
      EventsOn: () => () => {},
    };
    previewSpy.mockReturnValue(new Promise(() => {}));
    render(
      wrap(
        <KnowledgeImportModal
          open
          path="C:/tmp/规范.pdf"
          fileName="规范.pdf"
          onClose={() => {}}
          onImported={() => {}}
        />,
      ),
    );
    expect(screen.getByText("解析中…")).toBeTruthy();
  });
});
