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

  it("识别 /context", () => {
    expect(classifyComposerCommand("/context")).toEqual({ type: "context" });
    expect(classifyComposerCommand("  /context  ")).toEqual({ type: "context" });
  });

  it("识别 /panel open/clear/自定义指令（GenUI 会话面板）", () => {
    expect(classifyComposerCommand("/panel")).toEqual({ type: "panel", action: "open" });
    expect(classifyComposerCommand("/panel open")).toEqual({ type: "panel", action: "open" });
    expect(classifyComposerCommand("/panel clear")).toEqual({ type: "panel", action: "clear" });
    expect(classifyComposerCommand("/panel 把成本拆成月度趋势")).toEqual({
      type: "panel",
      action: "open",
      instruction: "把成本拆成月度趋势",
    });
  });

  it("其它输入按提交处理", () => {
    expect(classifyComposerCommand("帮我写报告")).toEqual({ type: "submit" });
    expect(classifyComposerCommand("/unknown cmd")).toEqual({ type: "submit" });
  });
});
