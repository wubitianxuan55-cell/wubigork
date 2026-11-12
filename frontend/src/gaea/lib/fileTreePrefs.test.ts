import { beforeEach, describe, expect, it } from "vitest";
import {
  FILE_TREE_EXCLUDE_LIMIT,
  fileTreePrefsKey,
  loadFileTreePrefs,
  normalizeFileTreeExcludes,
  saveFileTreePrefs,
} from "./fileTreePrefs";

describe("fileTreePrefs", () => {
  beforeEach(() => localStorage.clear());

  it("normalizes separators, decorations, duplicates and unsafe bare globs", () => {
    expect(normalizeFileTreeExcludes([
      " ./generated/ ",
      ".\\generated\\",
      "**",
      "",
      "docs\\private",
      "docs/private",
    ])).toEqual(["generated", "docs/private"]);
  });

  it("loads safe defaults for missing or corrupt storage", () => {
    expect(loadFileTreePrefs("C:/ws")).toEqual({ excludes: [], compactFolders: true });
    localStorage.setItem(fileTreePrefsKey("C:/ws"), "{bad json");
    expect(loadFileTreePrefs("C:/ws")).toEqual({ excludes: [], compactFolders: true });
  });

  it("persists per-workspace exclusions and compact-folder choice", () => {
    const saved = saveFileTreePrefs("C:/ws", { excludes: ["dist", "docs/private"], compactFolders: false });
    expect(saved).toEqual({ excludes: ["dist", "docs/private"], compactFolders: false });
    expect(loadFileTreePrefs("C:/ws")).toEqual(saved);
    expect(loadFileTreePrefs("C:/other")).toEqual({ excludes: [], compactFolders: true });
  });

  it("caps the pattern count", () => {
    const patterns = Array.from({ length: FILE_TREE_EXCLUDE_LIMIT + 5 }, (_, i) => `dir-${i}`);
    expect(normalizeFileTreeExcludes(patterns)).toHaveLength(FILE_TREE_EXCLUDE_LIMIT);
  });
});
