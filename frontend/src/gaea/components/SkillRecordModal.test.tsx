// SkillRecordModal（阶段七 7.2-1 会话录制技能）：蒸馏→可编辑草稿→保存。
// 断言：打开即自动蒸馏并回填表单；步骤按行编辑后保存逐参传递；空步骤被拒；
// 蒸馏失败出 toast 不渲染表单。
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { SkillRecordModal } from "./SkillRecordModal";
import { ToastProvider } from "./Toast";

const draftFromSession = vi.fn();
const draftSave = vi.fn();

vi.mock("../lib/bridge", () => ({
  app: {
    SkillDraftFromSession: (...a: unknown[]) => draftFromSession(...a),
    SkillDraftSave: (...a: unknown[]) => draftSave(...a),
  },
  openExternal: vi.fn(),
  onEvent: vi.fn(() => () => {}),
  onReady: vi.fn(() => () => {}),
}));

function renderModal() {
  return render(
    <ToastProvider>
      <SkillRecordModal open onClose={() => {}} />
    </ToastProvider>,
  );
}

const SAMPLE = {
  draft: {
    name: "weekly-report",
    description: "生成三段式周报",
    scenario: "周五汇总本周产出",
    steps: ["读取本周会议纪要", "汇总完成事项"],
    cautions: ["控制在三百字内"],
  },
  replay: "【用户】整理周报",
  preview: "---\nname: weekly-report\n---",
};

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("SkillRecordModal 会话录制技能", () => {
  it("打开即蒸馏并回填表单（含预览与回放）", async () => {
    draftFromSession.mockResolvedValue(SAMPLE);
    renderModal();
    await waitFor(() => {
      expect((screen.getByTestId("record-name") as HTMLInputElement).value).toBe("weekly-report");
    });
    expect((screen.getByTestId("record-steps") as HTMLTextAreaElement).value).toBe(
      "读取本周会议纪要\n汇总完成事项",
    );
    expect(screen.getByTestId("record-preview").textContent).toContain("name: weekly-report");
  });

  it("编辑步骤后保存：逐行 trim 过滤空行传递", async () => {
    draftFromSession.mockResolvedValue(SAMPLE);
    draftSave.mockResolvedValue({ name: "weekly-report", reloaded: false });
    renderModal();
    await waitFor(() => expect(screen.getByTestId("record-steps")).toBeTruthy());
    fireEvent.change(screen.getByTestId("record-steps"), {
      target: { value: "读取纪要\n\n汇总事项\n" },
    });
    fireEvent.click(screen.getByRole("button", { name: /^保\s*存\s*为\s*技\s*能$/ }));
    await waitFor(() => expect(draftSave).toHaveBeenCalledTimes(1));
    expect(draftSave).toHaveBeenCalledWith(
      expect.objectContaining({
        name: "weekly-report",
        steps: ["读取纪要", "汇总事项"],
        cautions: ["控制在三百字内"],
      }),
    );
  });

  it("清空步骤后保存被拒（不落盘）", async () => {
    draftFromSession.mockResolvedValue(SAMPLE);
    renderModal();
    await waitFor(() => expect(screen.getByTestId("record-steps")).toBeTruthy());
    fireEvent.change(screen.getByTestId("record-steps"), { target: { value: "\n  \n" } });
    fireEvent.click(screen.getByRole("button", { name: /^保\s*存\s*为\s*技\s*能$/ }));
    await waitFor(() => expect(screen.getByText(/至少保留一条操作步骤/)).toBeTruthy());
    expect(draftSave).not.toHaveBeenCalled();
  });

  it("蒸馏失败：toast 报错且可手动重试", async () => {
    draftFromSession.mockRejectedValue(new Error("当前会话为空"));
    renderModal();
    await waitFor(() => expect(screen.getByText(/当前会话为空/)).toBeTruthy());
    expect(screen.queryByTestId("record-name")).toBeNull();
    expect(screen.getByRole("button", { name: /重新蒸馏/ })).toBeTruthy();
  });

  // ── preload 路径（7.2-2 journal 蒸馏收口）：跳过内部蒸馏直接回填 ──

  it("preload 提供时不再调用 SkillDraftFromSession，直接回填表单", async () => {
    render(
      <ToastProvider>
        <SkillRecordModal open preload={SAMPLE} onClose={() => {}} />
      </ToastProvider>,
    );
    await waitFor(() => {
      expect((screen.getByTestId("record-name") as HTMLInputElement).value).toBe("weekly-report");
    });
    expect((screen.getByTestId("record-steps") as HTMLTextAreaElement).value).toBe(
      "读取本周会议纪要\n汇总完成事项",
    );
    expect(draftFromSession).not.toHaveBeenCalled();
  });

  it("preload 保存成功后 onSaved 收到落盘技能名", async () => {
    const onSaved = vi.fn();
    draftSave.mockResolvedValue({ name: "weekly-report", reloaded: true });
    render(
      <ToastProvider>
        <SkillRecordModal open preload={SAMPLE} onClose={() => {}} onSaved={onSaved} />
      </ToastProvider>,
    );
    await waitFor(() => expect(screen.getByTestId("record-steps")).toBeTruthy());
    fireEvent.click(screen.getByRole("button", { name: /^保\s*存\s*为\s*技\s*能$/ }));
    await waitFor(() => expect(onSaved).toHaveBeenCalledWith("weekly-report"));
  });
});
