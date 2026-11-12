export interface FileTreePrefs {
  excludes: string[];
  compactFolders: boolean;
}

export const FILE_TREE_EXCLUDE_LIMIT = 64;
export const FILE_TREE_EXCLUDE_MAX_LENGTH = 256;

const DEFAULT_PREFS: FileTreePrefs = { excludes: [], compactFolders: true };

export function fileTreePrefsKey(cwd: string): string {
  return `gaea.fileTree.prefs.${encodeURIComponent(cwd)}`;
}

export function normalizeFileTreeExcludes(raw: unknown): string[] {
  if (!Array.isArray(raw)) return [];
  const out: string[] = [];
  const seen = new Set<string>();
  for (const value of raw) {
    if (typeof value !== "string") continue;
    let pattern = value.trim().replace(/\\/g, "/");
    if (!pattern || pattern.length > FILE_TREE_EXCLUDE_MAX_LENGTH) continue;
    while (pattern.startsWith("./")) pattern = pattern.slice(2);
    pattern = pattern.replace(/^\/+|\/+$/g, "");
    if (!pattern || pattern === "**") continue;
    const key = pattern.toLowerCase();
    if (seen.has(key)) continue;
    seen.add(key);
    out.push(pattern);
    if (out.length >= FILE_TREE_EXCLUDE_LIMIT) break;
  }
  return out;
}

function normalizePrefs(raw: unknown): FileTreePrefs {
  if (raw === null || typeof raw !== "object" || Array.isArray(raw)) return { ...DEFAULT_PREFS };
  const rec = raw as Record<string, unknown>;
  return {
    excludes: normalizeFileTreeExcludes(rec.excludes),
    compactFolders: typeof rec.compactFolders === "boolean" ? rec.compactFolders : DEFAULT_PREFS.compactFolders,
  };
}

export function loadFileTreePrefs(cwd?: string): FileTreePrefs {
  if (!cwd) return { ...DEFAULT_PREFS };
  try {
    const raw = localStorage.getItem(fileTreePrefsKey(cwd));
    if (!raw) return { ...DEFAULT_PREFS };
    return normalizePrefs(JSON.parse(raw));
  } catch {
    return { ...DEFAULT_PREFS };
  }
}

export function saveFileTreePrefs(cwd: string | undefined, prefs: FileTreePrefs): FileTreePrefs {
  const next = normalizePrefs(prefs);
  if (!cwd) return next;
  try {
    localStorage.setItem(fileTreePrefsKey(cwd), JSON.stringify(next));
  } catch {
    // Quota/private mode: keep in-memory preferences for this mount.
  }
  return next;
}
