import { describe, it, expect } from "vitest";
import { shouldSubmitOnEnter } from "./chatComposer";

describe("shouldSubmitOnEnter", () => {
  it("普通 Enter 提交", () => {
    expect(shouldSubmitOnEnter("Enter", false, false)).toBe(true);
  });

  it("Shift+Enter 换行不提交", () => {
    expect(shouldSubmitOnEnter("Enter", true, false)).toBe(false);
  });

  it("输入法组合态 Enter 不提交", () => {
    expect(shouldSubmitOnEnter("Enter", false, true)).toBe(false);
  });

  it("非 Enter 键不提交", () => {
    expect(shouldSubmitOnEnter("a", false, false)).toBe(false);
    expect(shouldSubmitOnEnter("NumpadEnter", false, false)).toBe(false);
  });
});
