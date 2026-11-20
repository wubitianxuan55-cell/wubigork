// ImageGenCard — 生图作业卡：状态三态、提示词展示与终态文件面。
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { LocaleProvider } from "../lib/i18n";
import type { ImageGenItem } from "./ImageGenCard";
import { ImageGenCard } from "./ImageGenCard";

const renderT = (ui: React.ReactElement) => {
  localStorage.setItem("gaea-lang", "zh");
  return render(<LocaleProvider>{ui}</LocaleProvider>);
};

const base: ImageGenItem = {
  kind: "imagegen",
  id: "imagegen:imggen-1",
  prompt: "治理工艺流程示意，蓝色科技风",
  model: "qwen-image",
  stage: "running",
  files: [],
  err: "",
};

describe("ImageGenCard 生图作业卡", () => {
  afterEach(cleanup);

  it("生成中：标题/模型/提示词/运行态可见", () => {
    const { container } = renderT(<ImageGenCard item={base} />);
    const card = screen.getByTestId("imggen-card");
    expect(card.getAttribute("data-stage")).toBe("running");
    expect(screen.getByText("生图")).toBeTruthy();
    expect(screen.getByText("qwen-image")).toBeTruthy();
    expect(screen.getByText("生成中")).toBeTruthy();
    expect(screen.getByText(/治理工艺流程示意/)).toBeTruthy();
    expect(container.querySelectorAll(".imggen-card-files")).toHaveLength(0);
  });

  it("完成：文件清单展示，不再有运行态", () => {
    renderT(<ImageGenCard item={{ ...base, stage: "done", files: [".gaea/uploads/gen-1.png"] }} />);
    const card = screen.getByTestId("imggen-card");
    expect(card.getAttribute("data-stage")).toBe("done");
    expect(screen.getByText("生成完成")).toBeTruthy();
    expect(screen.getByText("gen-1.png")).toBeTruthy();
    expect(screen.queryByText("生成中")).toBeNull();
  });

  it("失败：错误文案透出", () => {
    renderT(<ImageGenCard item={{ ...base, stage: "failed", err: "后端超时" }} />);
    expect(screen.getByText("失败")).toBeTruthy();
    expect(screen.getByText("后端超时")).toBeTruthy();
  });
});
