import { describe, it, expect } from "vitest";
import { classifyComposerCommand } from "./command";

describe("classifyComposerCommand 斜杠命令分类", () => {
  it("识别 /model <ref>", () => {
    expect(classifyComposerCommand("/model deepseek-chat")).toEqual({ type: "model", ref: "deepseek-chat" });
    expect(classifyComposerCommand("  /model g1  ")).toEqual({ type: "model", ref: "g1" });
  });

  it("识别 /memory", () => {
    expect(classifyComposerCommand("/memory")).toEqual({ type: "memory" });
  });

  it("其它输入按提交处理", () => {
    expect(classifyComposerCommand("帮我写报告")).toEqual({ type: "submit" });
    expect(classifyComposerCommand("/unknown cmd")).toEqual({ type: "submit" });
  });
});
