import { describe, expect, it } from "vitest";
import { buildUnifiedChangeTree, type UnifiedChangeDir, type UnifiedChangeFile } from "./unifiedChangesTree";

describe("buildUnifiedChangeTree", () => {
  it("merges Git and Agent state by path and builds directory rollups", () => {
    const tree = buildUnifiedChangeTree({
      isRepo: true,
      files: [
        { path: "docs/a.md", x: "M", y: " ", staged: true },
        { path: "README.md", x: " ", y: "M" },
      ],
    }, [
      { path: "docs/a.md", count: 3 },
      { path: "docs/b.md", count: 1 },
    ], "C:/ws");
    const docs = tree.find((node) => node.name === "docs") as UnifiedChangeDir;
    expect(docs.fileCount).toBe(2);
    expect(docs.gitCount).toBe(1);
    expect(docs.agentCount).toBe(4);
    const a = docs.children.find((node) => node.name === "a.md") as UnifiedChangeFile;
    expect(a.git?.staged).toBe(true);
    expect(a.agent?.count).toBe(3);
  });

  it("normalizes absolute Agent paths below cwd and drops outside/absolute leakage", () => {
    const tree = buildUnifiedChangeTree(null, [
      { path: "C:\\ws\\src\\inside.ts", count: 1 },
      { path: "D:\\outside\\escape.ts", count: 1 },
    ], "c:/ws");
    expect(tree[0]).toMatchObject({ kind: "dir", name: "src", fileCount: 1 });
    expect(JSON.stringify(tree)).not.toContain("escape.ts");
  });

  it("allows a non-repository workspace to still show Agent changes", () => {
    const tree = buildUnifiedChangeTree({ isRepo: false, files: [] }, [{ path: "notes.md", count: 2 }]);
    expect(tree[0]).toMatchObject({ kind: "file", path: "notes.md", agent: { count: 2 } });
  });
});
