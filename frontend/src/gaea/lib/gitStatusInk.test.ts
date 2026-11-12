import { describe, expect, it } from "vitest";
import { buildGitStatusIndex, fileGitInk, gitPathKey } from "./gitStatusInk";
import type { GitStatusView } from "./types";

describe("gitStatusInk", () => {
  it("classifies untracked, added, modified and deleted rows", () => {
    expect(fileGitInk({ path: "new.txt", x: "?", y: "?", untracked: true })).toMatchObject({
      tone: "untracked",
      letter: "?",
      label: "未跟踪",
    });
    expect(fileGitInk({ path: "added.ts", x: "A", y: " " })).toMatchObject({
      tone: "added",
      letter: "A",
    });
    expect(fileGitInk({ path: "modified.ts", x: " ", y: "M", modified: true })).toMatchObject({
      tone: "modified",
      letter: "M",
    });
    expect(fileGitInk({ path: "deleted.ts", x: " ", y: "D", deleted: true })).toMatchObject({
      tone: "deleted",
      letter: "D",
    });
  });

  it("conflict outranks staged/working letters and keeps both letters", () => {
    expect(fileGitInk({ path: "both.txt", x: "M", y: "M" })).toMatchObject({
      tone: "modified",
      letter: "M",
    });
    expect(fileGitInk({ path: "conflict.txt", x: "U", y: "U" })).toMatchObject({
      tone: "conflict",
      letter: "!",
    });
    expect(fileGitInk({ path: "both-different.txt", x: "A", y: "M" })).toMatchObject({
      tone: "added",
      letter: "AM",
    });
  });

  it("normalizes Windows separators, leading ./ and rename arrows", () => {
    expect(gitPathKey(".\\Docs\\Plan.md")).toBe("docs/plan.md");
    expect(gitPathKey("old/name.md -> new/name.md")).toBe("new/name.md");
  });

  it("builds unique directory rollups and strongest descendant tone", () => {
    const status: GitStatusView = {
      isRepo: true,
      files: [
        { path: "docs/a.md", x: " ", y: "M" },
        { path: "docs/b.md", x: "?", y: "?", untracked: true },
        { path: "src/c.ts", x: "U", y: "U" },
        { path: "SRC\\c.ts", x: "U", y: "U" },
      ],
    };
    const index = buildGitStatusIndex(status);

    expect(index.files.get("docs/a.md")?.tone).toBe("modified");
    expect(index.files.get("src/c.ts")?.tone).toBe("conflict");
    expect(index.dirs.get("docs")).toEqual({ tone: "untracked", count: 2 });
    expect(index.dirs.get("src")).toEqual({ tone: "conflict", count: 1 });
    expect(index.dirs.get("")).toEqual({ tone: "conflict", count: 3 });
  });

  it("supports porcelain's collapsed untracked-directory row", () => {
    const index = buildGitStatusIndex({
      isRepo: true,
      files: [{ path: "reports/", x: "?", y: "?", untracked: true }],
    });
    expect(index.files.size).toBe(0);
    expect(index.dirs.get("reports")).toEqual({ tone: "untracked", count: 1 });
    expect(index.dirs.get("")).toEqual({ tone: "untracked", count: 1 });
  });

  it("returns empty maps for non-repositories", () => {
    const index = buildGitStatusIndex({ isRepo: false, files: [], error: "not a repo" });
    expect(index.files.size).toBe(0);
    expect(index.dirs.size).toBe(0);
  });
});
