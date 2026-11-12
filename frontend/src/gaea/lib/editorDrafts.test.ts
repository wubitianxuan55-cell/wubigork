import { afterEach, describe, expect, it, vi } from "vitest";
import {
  hasDirtyEditor,
  isEditorDirty,
  registerEditorDraft,
  unregisterEditorDraft,
} from "./editorDrafts";

describe("editorDrafts", () => {
  afterEach(() => vi.restoreAllMocks());

  it("tracks multiple preview owners by normalized path", () => {
    const a = Symbol("a");
    const b = Symbol("b");
    registerEditorDraft("Docs\\Plan.md", a);
    registerEditorDraft("docs/plan.md", b);
    expect(isEditorDirty("docs/plan.md")).toBe(true);
    unregisterEditorDraft("docs/plan.md", a);
    expect(isEditorDirty("docs/plan.md")).toBe(true);
    unregisterEditorDraft("docs/plan.md", b);
    expect(isEditorDirty("docs/plan.md")).toBe(false);
    expect(hasDirtyEditor()).toBe(false);
  });
});
