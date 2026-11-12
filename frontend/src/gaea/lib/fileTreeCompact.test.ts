import { describe, expect, it } from "vitest";
import { compactFolderChain, compactPreloadTargets, MAX_COMPACT_DEPTH } from "./fileTreeCompact";
import type { WorkbenchEntry } from "./types";

const dir = (name: string, compact = false): WorkbenchEntry => ({
  name,
  isDir: true,
  size: 0,
  compact,
});

describe("fileTreeCompact", () => {
  it("folds consecutive sole-directory children into one chain", () => {
    const levels = {
      "": { entries: [dir("src", true)] },
      src: { entries: [dir("core", true)] },
      "src/core": { entries: [dir("ui", false)] },
    };
    const chain = compactFolderChain("src", levels[""].entries![0], (p) => levels[p as keyof typeof levels]);
    expect(chain.map((link) => link.path)).toEqual(["src", "src/core", "src/core/ui"]);
  });

  it("stops when the loaded level has files, multiple children or a symlink", () => {
    expect(compactFolderChain("src", dir("src", true), () => ({ entries: [
      dir("a", true), dir("b", false),
    ] }))).toHaveLength(1);
    expect(compactFolderChain("src", dir("src", true), () => ({ entries: [
      { name: "main.ts", isDir: false, size: 1 },
    ] }))).toHaveLength(1);
    expect(compactFolderChain("src", dir("src", true), () => ({ entries: [
      { ...dir("link", true), isSymlink: true },
    ] }))).toHaveLength(1);
  });

  it("prefetches only the unresolved compact frontier", () => {
    const levels = {
      "": { entries: [dir("src", true), dir("docs", false)] },
      src: { entries: [dir("core", true)] },
    };
    expect(compactPreloadTargets(levels)).toEqual(["src/core"]);
  });

  it("caps pathological chains", () => {
    const levels: Record<string, { entries: WorkbenchEntry[] }> = {};
    levels[""] = { entries: [dir("a", true)] };
    let path = "a";
    for (let i = 0; i < MAX_COMPACT_DEPTH + 5; i++) {
      const next = `d${i}`;
      levels[path] = { entries: [dir(next, true)] };
      path = `${path}/${next}`;
    }
    const chain = compactFolderChain("a", levels[""].entries[0], (p) => levels[p]);
    expect(chain).toHaveLength(MAX_COMPACT_DEPTH);
  });
});
