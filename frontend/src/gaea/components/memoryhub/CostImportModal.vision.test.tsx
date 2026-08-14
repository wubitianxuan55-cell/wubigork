import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { CostImportModal } from "./CostImportModal";
import { ToastProvider } from "../Toast";

// PDF/图片导入走视觉识别管线（CostImportVisionPreview）→ 预览确认 → 入库。
const { visionSpy, applySpy } = vi.hoisted(() => ({
  visionSpy: vi.fn(),
  applySpy: vi.fn(),
}));

vi.mock("../../lib/bridge", () => ({
  app: {
    CostImportPreview: async () => ({ rows: [] }),
    CostImportVisionPreview: (...args: unknown[]) => visionSpy(...args),
    CostImportAIParse: async () => ({ rows: [] }),
    CostImportApply: (...args: unknown[]) => applySpy(...args),
  },
}));

const wrap = (node: React.ReactNode) => <ToastProvider>{node}</ToastProvider>;

// 识别预览样例：source 契约（pdf_text/pdf_scan/image/xlsx/csv）落盘后可移除内联构造。
const visionPreview = (source: string, over: Record<string, unknown> = {}) => ({
  path: "C:/tmp/报价单.pdf",
  fileName: "报价单.pdf",
  columns: [],
  unmapped: [],
  message: "已从扫描件识别 2 条报价行，请核对后确认导入。",
  aiUsed: false,
  source,
  rows: [
    {
      name: "", title: "HP300 高频液压振动锤", category: "机械", unit: "台班",
      price: 3200, spec: "300kW", source: "XX租赁", status: "现行",
      existingName: "", existingPrice: 0, matchNote: "新增",
      raw: "HP300 高频液压振动锤 | 台班 | 3200 | XX租赁", skip: false, skipReason: "",
    },
    {
      name: "", title: "P.O 42.5 水泥", category: "材料", unit: "吨",
      price: 480, spec: "", source: "海螺", status: "现行",
      existingName: "", existingPrice: 0, matchNote: "新增",
      raw: "P.O 42.5 水泥 | 吨 | 480 | 海螺", skip: false, skipReason: "",
    },
  ],
  ...over,
});

describe("CostImportModal PDF/图片识别导入", () => {
  beforeEach(() => {
    visionSpy.mockReset();
    applySpy.mockReset();
  });

  it("PDF 走视觉识别管线：识别中提示 → 预览表 + 来源标注 → 确认导入", async () => {
    let resolveVision!: (pv: unknown) => void;
    visionSpy.mockReturnValue(
      new Promise((res) => {
        resolveVision = res;
      }),
    );

    render(
      wrap(
        <CostImportModal
          open
          path="C:/tmp/报价单.pdf"
          fileName="报价单.pdf"
          onClose={() => {}}
          onImported={() => {}}
        />,
      ),
    );

    // 识别进行中：表格区显示「正在识别报价单…」，并确实调用视觉识别绑定。
    expect(screen.getByText("正在识别报价单…")).toBeTruthy();
    expect(visionSpy).toHaveBeenCalledWith("C:/tmp/报价单.pdf");

    resolveVision(visionPreview("pdf_scan"));
    // 预览沿用现有表格流程（名称可编辑、可勾选）。
    expect(await screen.findByDisplayValue("HP300 高频液压振动锤")).toBeTruthy();
    expect(screen.getByDisplayValue("P.O 42.5 水泥")).toBeTruthy();
    // 识别来源中文标注：PDF 扫描件 OCR。
    expect(screen.getByText("PDF 扫描件 OCR")).toBeTruthy();

    // 确认后走统一入库（无确认不落库：此时才调 Apply）。
    fireEvent.click(screen.getByText("确认导入 2 条"));
    await waitFor(() => expect(applySpy).toHaveBeenCalledTimes(1));
    expect(applySpy.mock.calls[0][0]).toHaveLength(2);
  });

  it("图片与 PDF 文本分别标注「图片 OCR」「PDF 文本」，按扩展名路由", async () => {
    visionSpy.mockResolvedValueOnce(visionPreview("image", { path: "C:/tmp/报价单.png", fileName: "报价单.png" }));
    const first = render(
      wrap(
        <CostImportModal
          open
          path="C:/tmp/报价单.png"
          fileName="报价单.png"
          onClose={() => {}}
          onImported={() => {}}
        />,
      ),
    );
    expect(await screen.findByText("图片 OCR")).toBeTruthy();
    expect(visionSpy).toHaveBeenLastCalledWith("C:/tmp/报价单.png");
    first.unmount();

    visionSpy.mockResolvedValueOnce(visionPreview("pdf_text", { path: "C:/tmp/报价单.pdf", fileName: "报价单.pdf" }));
    render(
      wrap(
        <CostImportModal
          open
          path="C:/tmp/报价单.pdf"
          fileName="报价单.pdf"
          onClose={() => {}}
          onImported={() => {}}
        />,
      ),
    );
    expect(await screen.findByText("PDF 文本")).toBeTruthy();
  });
});
