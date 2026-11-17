import { describe, expect, it } from "vitest";
import {
  MAX_RENDERED_ROWS,
  OUTPUT_BUDGET,
  TOOL_CARD_OUTPUT_PREVIEW_LINES,
  TOOL_OUTPUT_MAX_PREVIEW_LINES,
} from "./outputBudget";

describe("shared output budget", () => {
  it("loads the backend-owned JSON contract", () => {
    expect(OUTPUT_BUDGET.schemaVersion).toBe(1);
    expect(TOOL_OUTPUT_MAX_PREVIEW_LINES).toBe(60);
    expect(TOOL_CARD_OUTPUT_PREVIEW_LINES).toBe(12);
    expect(MAX_RENDERED_ROWS).toBe(10000);
    expect(OUTPUT_BUDGET.display.headLines + OUTPUT_BUDGET.display.tailLines)
      .toBe(OUTPUT_BUDGET.display.toolCardPreviewLines);
    expect(OUTPUT_BUDGET.frontend.highlightByteLimit).toBe(512 * 1024);
    expect(OUTPUT_BUDGET.frontend.highlightLineLimit).toBe(10000);
    expect(OUTPUT_BUDGET.frontend.highlightLineByteLimit).toBe(4096);
    expect(OUTPUT_BUDGET.frontend.genuiFragmentByteLimit).toBe(2 * 1024 * 1024);
    expect(new URL(OUTPUT_BUDGET.frontend.genuiPlaceholderOrigin).hostname.endsWith(".invalid"))
      .toBe(true);
  });
});
