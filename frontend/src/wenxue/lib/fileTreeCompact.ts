import type { WorkbenchEntry } from "./types";

export const MAX_COMPACT_DEPTH = 8;

export interface CompactLevelView {
  entries?: WorkbenchEntry[];
}

export interface CompactFolderLink {
  path: string;
  entry: WorkbenchEntry;
}

function joinRel(parent: string, name: string): string {
  return parent ? `${parent}/${name}` : name;
}

/**
 * Resolve the visible breadcrumb chain from one directory entry. Backend
 * `compact` is the gate; a missing/failed/multi-child/loading level safely
 * stops the fold at the last known directory.
 */
export function compactFolderChain(
  path: string,
  entry: WorkbenchEntry,
  levelOf: (path: string) => CompactLevelView | undefined,
): CompactFolderLink[] {
  const chain: CompactFolderLink[] = [{ path, entry }];
  let current = entry;
  let currentPath = path;
  while (current.compact === true && chain.length < MAX_COMPACT_DEPTH) {
    const level = levelOf(currentPath);
    const entries = level?.entries;
    if (!entries || entries.length !== 1) break;
    const only = entries[0];
    if (!only.isDir || only.isSymlink) break;
    currentPath = joinRel(currentPath, only.name);
    current = only;
    chain.push({ path: currentPath, entry: only });
  }
  return chain;
}

/**
 * Levels that must be prefetched to make compact breadcrumbs visible. Each
 * answer may reveal the next compact frontier; the caller refetches once the
 * returned levels arrive.
 */
export function compactPreloadTargets(levels: Record<string, CompactLevelView>): string[] {
  const out = new Set<string>();
  for (const [parent, level] of Object.entries(levels)) {
    for (const entry of level.entries ?? []) {
      if (!entry.isDir || entry.compact !== true) continue;
      const path = joinRel(parent, entry.name);
      const chain = compactFolderChain(path, entry, (p) => levels[p]);
      const tail = chain[chain.length - 1];
      if (tail.entry.compact === true && chain.length < MAX_COMPACT_DEPTH && levels[tail.path] === undefined) {
        out.add(tail.path);
      }
    }
  }
  return [...out];
}
