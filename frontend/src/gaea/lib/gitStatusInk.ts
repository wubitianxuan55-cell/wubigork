import type { GitFileStatus, GitStatusView } from "./types";

export type GitInkTone =
  | "conflict"
  | "deleted"
  | "renamed"
  | "untracked"
  | "added"
  | "copied"
  | "modified";

export interface GitFileInk {
  tone: GitInkTone;
  letter: string;
  label: string;
  detail: string;
}

export interface GitDirInk {
  tone: GitInkTone;
  count: number;
}

export interface GitStatusIndex {
  files: Map<string, GitFileInk>;
  dirs: Map<string, GitDirInk>;
}

const TONE_RANK: Record<GitInkTone, number> = {
  modified: 1,
  copied: 2,
  added: 3,
  untracked: 4,
  renamed: 5,
  deleted: 6,
  conflict: 7,
};

const LETTER_LABEL: Record<string, string> = {
  A: "新增",
  C: "复制",
  D: "删除",
  M: "修改",
  R: "重命名",
  T: "类型变更",
  U: "冲突",
  "?": "未跟踪",
};

/** Git paths and FileTree paths use one comparison key: slash-normalized, case-folded. */
export function gitPathKey(path: string): string {
  let p = String(path || "").replace(/\\/g, "/");
  const renamed = p.lastIndexOf(" -> ");
  if (renamed >= 0) p = p.slice(renamed + 4);
  return p.replace(/^\.\/+/, "").replace(/\/+/g, "/").replace(/\/+$/, "").toLowerCase();
}

function statusIsDir(path: string): boolean {
  const p = String(path || "").replace(/\\/g, "/");
  const renamed = p.lastIndexOf(" -> ");
  return /\/$/.test(renamed >= 0 ? p.slice(renamed + 4) : p);
}

// 未合并冲突六形态（与 Go 侧 isConflictXY 同一规则，G1.7 起补齐 AU/UA/DU/UD）。
const CONFLICT_XY = new Set(["AA", "DD", "AU", "UA", "DU", "UD"]);

function classify(x: string, y: string, f: GitFileStatus): GitInkTone {
  const xs = (x || " ").trim();
  const ys = (y || " ").trim();
  if (f.untracked || (xs === "?" && ys === "?")) return "untracked";
  if (f.conflict || xs === "U" || ys === "U" || CONFLICT_XY.has(`${xs}${ys}`)) return "conflict";
  if (f.deleted || xs === "D" || ys === "D") return "deleted";
  if (f.renamed || xs === "R" || ys === "R") return "renamed";
  if (xs === "A" || ys === "A") return "added";
  if (xs === "C" || ys === "C") return "copied";
  return "modified";
}

function letterOf(x: string, y: string, tone: GitInkTone): string {
  if (tone === "untracked") return "?";
  if (tone === "conflict") return "!";
  const letters = [...new Set([(x || "").trim(), (y || "").trim()].filter(Boolean))];
  return letters.join("") || "M";
}

function sideLabel(letter: string): string {
  if (!letter.trim()) return "无";
  return LETTER_LABEL[letter.trim()] ?? letter.trim();
}

/** Convert one porcelain X/Y row to one visible tint/badge. */
export function fileGitInk(f: GitFileStatus): GitFileInk {
  const x = f.x || " ";
  const y = f.y || " ";
  const tone = classify(x, y, f);
  const letter = letterOf(x, y, tone);
  const hasX = x.trim() !== "";
  const hasY = y.trim() !== "";
  const xLabel = sideLabel(x);
  const yLabel = sideLabel(y);
  const label = tone === "untracked"
    ? "未跟踪"
    : tone === "conflict"
      ? "冲突"
      : hasX && hasY
        ? `暂存${xLabel} / 工作区${yLabel}`
        : hasX
          ? `暂存${xLabel}`
          : `工作区${yLabel}`;
  return {
    tone,
    letter,
    label,
    detail: tone === "untracked"
      ? "Git：未跟踪"
      : `Git：暂存 ${xLabel} · 工作区 ${yLabel}`,
  };
}

function stronger(a: GitInkTone, b: GitInkTone): GitInkTone {
  return TONE_RANK[b] > TONE_RANK[a] ? b : a;
}

/**
 * Build O(1) file lookups and directory rollups from one Git status snapshot.
 * Directory rows report a dot when any descendant changed, not a copy of file state.
 */
export function buildGitStatusIndex(status?: GitStatusView | null): GitStatusIndex {
  const files = new Map<string, GitFileInk>();
  const dirs = new Map<string, GitDirInk>();
  const directoryOnly = new Map<string, GitFileInk>();
  if (!status?.isRepo || !Array.isArray(status.files)) return { files, dirs };

  for (const f of status.files) {
    const key = gitPathKey(f.path);
    if (!key) continue;
    const ink = fileGitInk(f);
    if (statusIsDir(f.path)) {
      const prev = directoryOnly.get(key);
      if (!prev || TONE_RANK[ink.tone] > TONE_RANK[prev.tone]) directoryOnly.set(key, ink);
      continue;
    }
    const prev = files.get(key);
    if (!prev || TONE_RANK[ink.tone] > TONE_RANK[prev.tone]) files.set(key, ink);
  }

  const bump = (key: string, ink: GitFileInk, includeSelf: boolean) => {
    const parts = key.split("/").filter(Boolean);
    const ancestors = [""];
    let cur = "";
    const upto = includeSelf ? parts.length : parts.length - 1;
    for (let i = 0; i < upto; i++) {
      cur = cur ? `${cur}/${parts[i]}` : parts[i];
      ancestors.push(cur);
    }
    for (const dir of ancestors) {
      const prev = dirs.get(dir);
      dirs.set(dir, {
        tone: prev ? stronger(prev.tone, ink.tone) : ink.tone,
        count: (prev?.count ?? 0) + 1,
      });
    }
  };

  for (const [key, ink] of files) {
    bump(key, ink, false);
  }
  for (const [key, ink] of directoryOnly) {
    if (files.has(key)) continue;
    bump(key, ink, true);
  }
  return { files, dirs };
}
