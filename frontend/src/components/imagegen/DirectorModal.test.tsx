// DirectorModal.test.tsx — 绘梦导演弹窗（11.6）
// 契约：开拍=需求透传后端；回执渲染轨迹/成片/计划/自评；accepted=false 显式
// 未达标警示；错误透出；空需求禁用。window.runtime 缺省（jsdom）时事件订阅
// 静默跳过，轨迹以回执 steps 为准。
import { describe, expect, it, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { DirectorModal } from "./DirectorModal";

const state = vi.hoisted(() => ({
  directorCalls: [] as Record<string, unknown>[],
  directorResult: null as Record<string, unknown> | null,
  directorError: "" as string,
}));

vi.mock("../../gaea/lib/bridge", () => ({
  app: {
    GaeaImageDirector: async (req: Record<string, unknown>) => {
      state.directorCalls.push(req);
      if (state.directorError) throw new Error(state.directorError);
      if (!state.directorResult) throw new Error("no result seeded");
      return state.directorResult;
    },
  },
}));

vi.mock("../../gaea/lib/wailsEvents", () => ({
  subscribeWailsEvent: vi.fn(() => () => undefined),
}));

import { LocaleProvider } from "../../gaea/lib/i18n";

const okResult = {
  images: [{ image: "data:image/png;base64,AAA", seed: 42, prompt: "优化后提示词", model: "krea2", size: "1024x1024", file_path: "C:/img/a.png" }],
  plan: { model: "krea2", prompt: "雨夜天台，冷色调", negative: "", style: "", loras: [], reason: "写实人像选 krea2" },
  verdict: { pass: true, score: 88, issues: [], fix: "" },
  steps: [
    { stage: "plan", attempt: 0, detail: "计划就绪：krea2（写实人像选 krea2）" },
    { stage: "generate", attempt: 1, detail: "生成中：krea2（1024x1024，seed 42）" },
    { stage: "eval", attempt: 1, detail: "自评 88 分（pass=true）" },
    { stage: "done", attempt: 1, detail: "第 1 轮自评通过（88 分）" },
  ],
  attempts: 1,
  accepted: true,
};

const renderModal = () =>
  render(
    <LocaleProvider>
      <DirectorModal open onClose={vi.fn()} />
    </LocaleProvider>,
  );

beforeEach(() => {
  state.directorCalls = [];
  state.directorResult = null;
  state.directorError = "";
});

describe("DirectorModal 绘梦导演", () => {
  it("空需求禁用开拍；填需求后开拍透传参数并渲染回执（轨迹/成片/计划/自评）", async () => {
    state.directorResult = okResult;
    renderModal();
    const run = screen.getByTestId("director-run") as HTMLButtonElement;
    expect(run.disabled).toBe(true);
    fireEvent.change(screen.getByTestId("director-need"), { target: { value: "雨夜的都市天台" } });
    expect(run.disabled).toBe(false);
    fireEvent.click(run);
    await waitFor(() => expect(screen.getByTestId("director-result")).toBeDefined());
    expect(state.directorCalls).toEqual([{ need: "雨夜的都市天台", aspect: "1:1", maxAttempts: 2 }]);
    const text = document.body.textContent ?? "";
    expect(text).toContain("导演轨迹");
    expect(text).toContain("计划就绪：krea2");
    expect(text).toContain("自评 88 分");
    expect(text).toContain("写实人像选 krea2");
    expect(document.querySelector('img[alt="导演成片"]')).not.toBeNull();
    // 通过态不出现未达标警示。
    expect(text).not.toContain("未达标");
  });

  it("accepted=false 显式未达标警示（诚实口径）", async () => {
    state.directorResult = {
      ...okResult,
      verdict: { pass: false, score: 40, issues: ["人脸崩坏"], fix: "减少特写" },
      steps: [...okResult.steps, { stage: "done", attempt: 2, detail: "已用尽 2 轮自评未过，返回末轮结果（诚实口径：未达标）" }],
      attempts: 2,
      accepted: false,
    };
    renderModal();
    fireEvent.change(screen.getByTestId("director-need"), { target: { value: "立绘" } });
    fireEvent.click(screen.getByTestId("director-run"));
    await waitFor(() => expect(screen.getByTestId("director-result")).toBeDefined());
    const text = document.body.textContent ?? "";
    expect(text).toContain("未达标");
    expect(text).toContain("人脸崩坏");
  });

  it("后端失败：错误透出", async () => {
    state.directorError = "未找到可用 LLM（模型中心未配置或离线）";
    renderModal();
    fireEvent.change(screen.getByTestId("director-need"), { target: { value: "x" } });
    fireEvent.click(screen.getByTestId("director-run"));
    await waitFor(() => expect(document.body.textContent).toContain("未找到可用 LLM"));
    expect(state.directorCalls).toHaveLength(1);
  });
});
