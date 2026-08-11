import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { CostImportModal } from "./CostImportModal";
import { ToastProvider } from "../Toast";

const { previewSpy, aiSpy, applySpy } = vi.hoisted(() => ({
  previewSpy: vi.fn(),
  aiSpy: vi.fn(),
  applySpy: vi.fn(),
}));

vi.mock("../../lib/bridge", () => ({
  app: {
    CostImportPreview: (...args: unknown[]) => previewSpy(...args),
    CostImportAIParse: (...args: unknown[]) => aiSpy(...args),
    CostImportApply: (...args: unknown[]) => applySpy(...args),
  },
}));

const wrap = (node: React.ReactNode) => <ToastProvider>{node}</ToastProvider>;

describe("CostImportModal 解析失败场景", () => {
  beforeEach(() => {
    previewSpy.mockReset();
    aiSpy.mockReset();
    applySpy.mockReset();
    previewSpy.mockRejectedValue(new Error("暂不支持 .pdf 格式导入"));
  });

  it("解析失败时展示持久错误提示，确认导入保持禁用且不会反复重解析", async () => {
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

    // 错误信息持久展示在弹窗内（不只是瞬时 toast）。
    expect(await screen.findByText(/可点击「AI 智能解析」重试/)).toBeTruthy();
    expect(screen.getAllByText(/暂不支持 .pdf 格式导入/).length).toBeGreaterThan(0);
    expect(screen.getByText(/确认导入 0 条/)).toBeTruthy();
    const confirm = screen.getByText(/确认导入/) as HTMLButtonElement;
    expect(confirm.disabled).toBe(true);

    // 等待 toast 生命周期结束：解析失败不应触发无限重试循环。
    await new Promise((r) => setTimeout(r, 500));
    expect(previewSpy).toHaveBeenCalledTimes(1);
  });

  it("解析失败后点击 AI 智能解析成功，错误清除且确认导入可用", async () => {
    aiSpy.mockResolvedValue({
      path: "C:/tmp/报价单.pdf",
      fileName: "报价单.pdf",
      columns: ["材料名称", "单价"],
      unmapped: [],
      message: "AI 智能解析完成，请核对后确认导入。",
      aiUsed: true,
      rows: [
        {
          name: "",
          title: "P.O 42.5 水泥",
          category: "材料",
          unit: "吨",
          price: 480,
          spec: "",
          source: "海螺",
          status: "现行",
          existingName: "",
          existingPrice: 0,
          matchNote: "新增",
          raw: "P.O 42.5 水泥 | 吨 | 480",
          skip: false,
          skipReason: "",
        },
      ],
    });

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
    await screen.findByText(/可点击「AI 智能解析」重试/);

    fireEvent.click(screen.getByText("AI 智能解析"));
    await waitFor(() => expect(screen.getByText(/AI 智能解析完成，请核对后确认导入。/)).toBeTruthy());
    expect(screen.queryByText(/可点击「AI 智能解析」重试/)).toBeNull();
    expect(screen.getByText("确认导入 1 条")).toBeTruthy();
  });

  it("AI 解析进行中显示过程提示，失败后错误持久展示", async () => {
    let rejectAI!: (e: Error) => void;
    aiSpy.mockReturnValue(
      new Promise((_resolve, reject) => {
        rejectAI = reject;
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
    await screen.findByText(/可点击「AI 智能解析」重试/);

    fireEvent.click(screen.getByText("AI 智能解析"));
    // 进行中：弹窗内出现过程提示（角色 status），按钮保持禁用。
    expect(await screen.findByRole("status")).toBeTruthy();
    expect(screen.getByText(/AI 智能解析中…/)).toBeTruthy();
    expect((screen.getByText("AI 解析中…") as HTMLButtonElement).disabled).toBe(true);

    // 失败：过程提示消失，错误持久展示，确认导入仍不可用。
    rejectAI(new Error("模型服务不可用"));
    await waitFor(() => expect(screen.getByText(/解析失败：AI 解析失败：/)).toBeTruthy());
    expect(screen.queryByRole("status")).toBeNull();
    expect((screen.getByText(/确认导入 0 条/) as HTMLButtonElement).disabled).toBe(true);
  });
});
