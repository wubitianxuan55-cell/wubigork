import { describe, it, expect } from "vitest";
import { summarizeServerError, summarizeSkillDescription } from "./capabilities";

describe("summarizeServerError", () => {
  it("提取 npm error code 与 errno", () => {
    expect(summarizeServerError('plugin "mcp-x" install failed\nnpm error code E404\nerrno -404')).toBe("mcp-x: npm E404 (-404)");
  });

  it("无 npm 码时取首句", () => {
    expect(summarizeServerError("Command failed. 请检查命令路径")).toBe("Command failed");
  });
});

describe("summarizeSkillDescription", () => {
  it("短描述原样返回", () => {
    expect(summarizeSkillDescription("  转换文档为 Markdown  ")).toBe("转换文档为 Markdown");
  });

  it("长描述截断到句末", () => {
    expect(summarizeSkillDescription("a".repeat(50) + "。" + "b".repeat(200))).toBe("a".repeat(50));
  });

  it("无句末标点时按 128 字符截断", () => {
    expect(summarizeSkillDescription("a".repeat(200))).toBe("a".repeat(128) + "…");
  });
});
