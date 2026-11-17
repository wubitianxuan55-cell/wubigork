import { describe, expect, it } from "vitest";
import { genuiPlaceholderUrl, genuiSourceExceedsLimit } from "./sourceGate";
import { splitGenuiFences } from "./parse";

describe("GenUI source guard", () => {
  it("uses a stable .invalid placeholder and rejects oversized sources before scanning fences", () => {
    const placeholder = genuiPlaceholderUrl("fragment-1");
    expect(new URL(placeholder).hostname.endsWith(".invalid")).toBe(true);
    const huge = "a".repeat(2 * 1024 * 1024 + 1);
    expect(genuiSourceExceedsLimit(huge)).toBe(true);
    const split = splitGenuiFences(huge);
    expect(split.segments).toHaveLength(1);
    expect(split.segments[0]).toMatchObject({ kind: "text", text: huge });
    expect(split.fenceIndex).toHaveLength(0);
  });
});
