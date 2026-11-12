import type { SessionChange } from "./changes";
import type { GitFileStatus, GitStatusView } from "./types";

export interface UnifiedChangeFile {
  kind: "file";
  path: string;
  name: string;
  git?: GitFileStatus;
  agent?: SessionChange;
}

export interface UnifiedChangeDir {
  kind: "dir";
  path: string;
  name: string;
  children: UnifiedChangeNode[];
  fileCount: number;
  gitCount: number;
  agentCount: number;
}

export type UnifiedChangeNode = UnifiedChangeFile | UnifiedChangeDir;

export function countUnifiedChangeFiles(nodes: UnifiedChangeNode[]): number {
  let count = 0;
  for (const node of nodes) {
    count += node.kind === "file" ? 1 : countUnifiedChangeFiles(node.children);
  }
  return count;
}

interface MutableDir {
  path: string;
  name: string;
  dirs: Map<string, MutableDir>;
  files: Map<string, UnifiedChangeFile>;
}

function slash(path: string): string {
  return String(path || "").trim().replace(/\\/g, "/").replace(/^\.\/+/, "").replace(/\/+$/, "");
}

function workspaceRel(path: string, cwd?: string): string {
  const p = slash(path);
  const base = slash(cwd ?? "");
  if (!p) return "";
  if (base) {
    const lowerP = p.toLowerCase();
    const lowerBase = base.toLowerCase();
    if (lowerP === lowerBase) return "";
    if (lowerP.startsWith(`${lowerBase}/`)) return p.slice(base.length + 1);
  }
  if (/^[A-Za-z]:\//.test(p) || p.startsWith("//")) return "";
  return p.replace(/^\/+/, "");
}

/**
 * Merge workspace Git state and this-session Agent changes into one directory
 * tree. Git paths are already workspace-relative; Agent paths may be absolute
 * and are only accepted when they sit below cwd.
 */
export function buildUnifiedChangeTree(
  status: GitStatusView | null | undefined,
  changes: SessionChange[],
  cwd?: string,
): UnifiedChangeNode[] {
  const merged = new Map<string, UnifiedChangeFile>();
  const put = (displayPath: string) => {
    const key = displayPath.toLowerCase();
    const existing = merged.get(key);
    if (existing) return existing;
    const parts = displayPath.split("/").filter(Boolean);
    if (parts.length === 0) return null;
    const file: UnifiedChangeFile = {
      kind: "file",
      path: displayPath,
      name: parts[parts.length - 1],
    };
    merged.set(key, file);
    return file;
  };

  for (const git of status?.isRepo ? status.files : []) {
    const path = workspaceRel(git.path, cwd);
    if (!path) continue;
    const file = put(path);
    if (file) file.git = git;
  }
  for (const change of changes) {
    const path = workspaceRel(change.path, cwd);
    if (!path) continue;
    const file = put(path);
    if (file) file.agent = change;
  }
  if (merged.size === 0) return [];

  const root: MutableDir = { path: "", name: "", dirs: new Map(), files: new Map() };
  for (const file of merged.values()) {
    const parts = file.path.split("/").filter(Boolean);
    let dir = root;
    for (let i = 0; i < parts.length - 1; i++) {
      const name = parts[i];
      const path = parts.slice(0, i + 1).join("/");
      let next = dir.dirs.get(name.toLowerCase());
      if (!next) {
        next = { path, name, dirs: new Map(), files: new Map() };
        dir.dirs.set(name.toLowerCase(), next);
      }
      dir = next;
    }
    dir.files.set(file.name.toLowerCase(), file);
  }

  const finalize = (dir: MutableDir): UnifiedChangeDir => {
    const dirs = [...dir.dirs.values()].map(finalize);
    const files = [...dir.files.values()];
    const children: UnifiedChangeNode[] = [
      ...dirs.sort((a, b) => a.name.localeCompare(b.name, undefined, { numeric: true, sensitivity: "base" })),
      ...files.sort((a, b) => a.name.localeCompare(b.name, undefined, { numeric: true, sensitivity: "base" })),
    ];
    return {
      kind: "dir",
      path: dir.path,
      name: dir.name,
      children,
      fileCount: files.length + dirs.reduce((n, d) => n + d.fileCount, 0),
      gitCount: files.filter((f) => !!f.git).length + dirs.reduce((n, d) => n + d.gitCount, 0),
      agentCount: files.reduce((n, f) => n + (f.agent?.count ?? 0), 0) +
        dirs.reduce((n, d) => n + d.agentCount, 0),
    };
  };

  const roots: UnifiedChangeNode[] = [
    ...[...root.dirs.values()].map(finalize),
    ...root.files.values(),
  ];
  return roots.sort(
    (a, b) => Number(a.kind === "file") - Number(b.kind === "file") ||
      a.name.localeCompare(b.name, undefined, { numeric: true, sensitivity: "base" }),
  );
}
