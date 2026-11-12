import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type DragEvent,
  type KeyboardEvent,
  type ReactNode,
} from "react";
import { Dropdown, Input, Modal } from "antd";
import type { MenuProps } from "antd";
import {
  ChevronRight,
  ChevronDown,
  Download,
  FolderPlus,
  Paperclip,
  Copy,
  ExternalLink,
  FolderOpen,
  Eye,
  Pencil,
  Plus,
  Settings,
  Trash2,
  X,
} from "../icons";
import { app } from "../lib/bridge";
import type { FileSearchHit, WorkbenchEntry, WorkbenchLevel } from "../lib/types";
import { useDebouncedValue } from "../hooks/useDebouncedValue";
import { FileTypeIcon } from "../lib/fileIcon";
import { compactFolderChain, compactPreloadTargets } from "../lib/fileTreeCompact";
import {
  loadFileTreePrefs,
  normalizeFileTreeExcludes,
  saveFileTreePrefs,
  type FileTreePrefs,
} from "../lib/fileTreePrefs";
import {
  buildGitStatusIndex,
  gitPathKey,
  type GitDirInk,
  type GitFileInk,
  type GitInkTone,
} from "../lib/gitStatusInk";
import { subscribe } from "../../events";
import { acquireGitStatus, useGitStatusStore } from "../lib/gitStatusStore";

// 复制成功后行尾「已复制」反馈时长（对齐插件 rowActions copied 1.2s）。
const COPIED_MS = 1200;
// 树中定位（reveal）高亮闪烁时长：足以被注意到，又不打断后续操作。
const REVEAL_FLASH_MS = 1600;
// 树中定位滚动重试：行可能在父链展开的异步加载后才渲染，轮询兜底。
const REVEAL_SCROLL_RETRIES = 20;
const REVEAL_SCROLL_INTERVAL_MS = 100;
// 展开态 localStorage 条目上限，防膨胀。
const EXPANDED_MAX = 500;
// 树内搜索命中上限（GaeaFileSearch 服务端同样钳制，前端侧再限一次对齐插件预算封顶纪律）。
const FILE_SEARCH_LIMIT = 50;
const UPLOAD_CHUNK_BYTES = 3 * 1024 * 1024;
const GIT_TONE_COLOR: Record<GitInkTone, string> = {
  added: "var(--md-sys-color-success)",
  untracked: "var(--md-sys-color-success)",
  copied: "var(--md-sys-color-success)",
  modified: "var(--md-sys-color-warning)",
  renamed: "var(--md-sys-color-warning)",
  deleted: "var(--md-sys-color-destructive)",
  conflict: "var(--md-sys-color-destructive)",
};

function GitDirDot({ ink }: { ink?: GitDirInk }) {
  if (!ink) return null;
  const title = `目录内有 ${ink.count} 个 Git 变更`;
  return (
    <span
      className="h-1.5 w-1.5 shrink-0 rounded-full"
      data-git-rollup={ink.count}
      style={{ background: GIT_TONE_COLOR[ink.tone] }}
      title={title}
      aria-label={title}
    />
  );
}

function GitFileBadge({ ink }: { ink?: GitFileInk }) {
  if (!ink) return null;
  return (
    <span
      className="shrink-0 rounded px-1 font-mono text-[8.5px] font-semibold leading-4"
      data-git-badge={ink.tone}
      style={{
        color: GIT_TONE_COLOR[ink.tone],
        background: `color-mix(in srgb, ${GIT_TONE_COLOR[ink.tone]} 12%, transparent)`,
      }}
      title={`${ink.label} · ${ink.detail}`}
      aria-label={`Git ${ink.label}`}
    >
      {ink.letter}
    </span>
  );
}

// 每个目录层级的加载态（加载三态：loading / error+重试 / 已加载）。
interface LevelData {
  loading: boolean;
  entries?: WorkbenchEntry[];
  error?: string;
}

/** 展开集 localStorage key：按 cwd 隔离，特殊字符经 encodeURIComponent 编码。 */
function expandedKey(cwd: string): string {
  return `gaea.fileTree.expanded.${encodeURIComponent(cwd)}`;
}

/** 读取展开集：解析失败/损坏/越权静默回退空集；条目数封顶防膨胀。 */
function loadExpanded(cwd: string | undefined): Record<string, boolean> {
  if (cwd === undefined) return {};
  try {
    const raw = localStorage.getItem(expandedKey(cwd));
    if (!raw) return {};
    const parsed: unknown = JSON.parse(raw);
    if (parsed === null || typeof parsed !== "object" || Array.isArray(parsed)) return {};
    const rec: Record<string, boolean> = {};
    for (const [k, v] of Object.entries(parsed as Record<string, unknown>)) {
      if (typeof v === "boolean") rec[k] = v;
    }
    const keys = Object.keys(rec);
    if (keys.length > EXPANDED_MAX) {
      const trimmed: Record<string, boolean> = {};
      for (const k of keys.slice(keys.length - EXPANDED_MAX)) trimmed[k] = rec[k];
      return trimmed;
    }
    return rec;
  } catch {
    return {};
  }
}

/** 写展开集：保留最近 EXPANDED_MAX 条；配额/隐私模式静默失败。 */
function persistExpanded(cwd: string | undefined, rec: Record<string, boolean>): void {
  if (cwd === undefined) return;
  const keys = Object.keys(rec);
  const trimmed: Record<string, boolean> = {};
  const start = Math.max(0, keys.length - EXPANDED_MAX);
  for (let i = start; i < keys.length; i++) trimmed[keys[i]] = rec[keys[i]];
  try {
    localStorage.setItem(expandedKey(cwd), JSON.stringify(trimmed));
  } catch {
    // 静默失败，不影响主流程
  }
}

/** 树中定位请求（v4.25 A3）：nonce 变化触发一次「展开父链 + 滚动到行 + 高亮闪烁」。 */
export interface RevealRequest {
  /** 目标文件相对路径（与 FileTree onSelect 同一相对路径口径）。 */
  rel: string;
  /** 递增计数：每次定位请求 +1，变化才触发（同一目标可重复定位）。 */
  nonce: number;
}

/** 由相对路径推导需展开的父目录链：`a/b/c.md` → `["a", "a/b"]`。
 *  兼容 Windows 反斜杠（产物面板登记路径可能带 `\`）。 */
function parentDirsOf(rel: string): string[] {
  const parts = rel.replace(/\\/g, "/").split("/").filter(Boolean);
  parts.pop(); // 末段是文件名
  const dirs: string[] = [];
  let cur = "";
  for (const part of parts) {
    cur = cur === "" ? part : `${cur}/${part}`;
    dirs.push(cur);
  }
  return dirs;
}

function baseNameOf(rel: string): string {
  return rel.replace(/\\/g, "/").split("/").filter(Boolean).pop() ?? rel;
}

function joinRel(parent: string, name: string): string {
  return parent ? `${parent.replace(/\\/g, "/")}/${name}` : name;
}

function underPath(path: string, root: string): boolean {
  if (root === "") return true;
  const p = path.replace(/\\/g, "/");
  const r = root.replace(/\\/g, "/").replace(/\/+$/, "");
  return p === r || p.startsWith(`${r}/`);
}

function parentLevelOf(rel: string): string {
  const parts = rel.replace(/\\/g, "/").split("/").filter(Boolean);
  parts.pop();
  return parts.join("/");
}

function bytesToBase64(bytes: Uint8Array): string {
  let binary = "";
  const step = 0x8000;
  for (let i = 0; i < bytes.length; i += step) {
    binary += String.fromCharCode(...bytes.subarray(i, i + step));
  }
  return btoa(binary);
}

type NameDialogState =
  | { kind: "rename"; target: string; parent: string; value: string; isDir: boolean }
  | { kind: "file" | "folder"; parent: string; value: string };

export function FileTree({
  cwd,
  onSelect,
  selectedFile,
  onReference,
  onOpenExternal,
  onReveal,
  onOpenMainPreview,
  revealRequest,
  onPathRenamed,
  onPathDeleted,
}: {
  cwd?: string;
  onSelect: (rel: string) => void;
  selectedFile?: string;
  onReference?: (rel: string) => void;
  onOpenExternal?: (rel: string) => void;
  onReveal?: (rel: string) => void;
  /** 主区预览入口（v4.25 A3 双入口保留）：右键菜单「预览」走这里开主区
   *  pane；缺省回退 onSelect（向后兼容旧调用方）。 */
  onOpenMainPreview?: (rel: string) => void;
  /** 树中定位请求：nonce 变化触发一次展开父链 + 滚动 + 高亮闪烁。 */
  revealRequest?: RevealRequest | null;
  /** 文件/目录改名后同步上层打开态（WorkbenchPane 用它改道文件 tab）。 */
  onPathRenamed?: (oldRel: string, newRel: string, isDir: boolean) => void;
  /** 删除后关闭上层已打开的文件 tab。 */
  onPathDeleted?: (paths: string[]) => void;
}) {
  // 展开集提升到 FileTree 顶层并按 cwd 持久化：重挂载（refreshKey）后恢复，根行默认展开。
  const [expanded, setExpanded] = useState<Record<string, boolean>>(() => loadExpanded(cwd));
  // 各目录层级数据缓存（加载三态）。刷新 = WorkspacePanel 换 key 重挂载，缓存自然清空。
  const [data, setData] = useState<Record<string, LevelData>>({});
  const inflight = useRef<Set<string>>(new Set());
  // 树内搜索（C8 蒸馏插件 TreePanel 的 host fs.search）：300ms 防抖。
  // 非空查询 → GaeaFileSearch 工作区递归文件名搜索（跨目录、跳过噪音目录、
  // 深度/数量封顶）；空查询 → 树。
  const [query, setQuery] = useState("");
  const debouncedQuery = useDebouncedValue(query, 300);
  const needle = debouncedQuery.trim().toLowerCase();
  const [searching, setSearching] = useState(false);
  const [searchHits, setSearchHits] = useState<FileSearchHit[] | null>(null);
  const [searchError, setSearchError] = useState<string | null>(null);
  // 多选、CRUD、上传与工作台反馈（选择只影响树，不改变主对话）。
  const [selection, setSelection] = useState<Set<string>>(() => new Set());
  const selectionAnchorRef = useRef<string | null>(null);
  const [nameDialog, setNameDialog] = useState<NameDialogState | null>(null);
  const [nameBusy, setNameBusy] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  const [actionNotice, setActionNotice] = useState<string | null>(null);
  const [uploading, setUploading] = useState<string | null>(null);
  const filesInputRef = useRef<HTMLInputElement | null>(null);
  const folderInputRef = useRef<HTMLInputElement | null>(null);
  const uploadTargetRef = useRef("");
  const [dragTarget, setDragTarget] = useState<string | null>(null);
  const [treePrefs, setTreePrefs] = useState<FileTreePrefs>(() => loadFileTreePrefs(cwd));
  const [treePrefsOpen, setTreePrefsOpen] = useState(false);
  const [excludeDraft, setExcludeDraft] = useState("");
  const [compactDraft, setCompactDraft] = useState(true);
  const gitStatus = useGitStatusStore((s) => s.status);
  const refreshGitStatus = useGitStatusStore((s) => s.refresh);

  useEffect(() => {
    setTreePrefs(loadFileTreePrefs(cwd));
    setTreePrefsOpen(false);
    setExpanded(loadExpanded(cwd));
    setData({});
    setSelection(new Set());
    selectionAnchorRef.current = null;
    inflight.current.clear();
  }, [cwd]);

  useEffect(() => acquireGitStatus(), []);

  useEffect(() => {
    if (needle === "") {
      setSearching(false);
      setSearchHits(null);
      setSearchError(null);
      return;
    }
    let cancelled = false;
    setSearching(true);
    setSearchError(null);
    void app
      .FileSearch(needle, FILE_SEARCH_LIMIT, treePrefs.excludes)
      .then((hits) => {
        if (cancelled) return;
        setSearchHits(hits ?? []);
        setSearching(false);
      })
      .catch((err) => {
        if (cancelled) return;
        setSearchError(err instanceof Error ? err.message : String(err));
        setSearching(false);
      });
    return () => { cancelled = true; };
  }, [needle, treePrefs.excludes]);
  // 复制成功反馈：行尾「已复制」标签替换 @ 按钮 1.2s。
  const [copiedPath, setCopiedPath] = useState<string | null>(null);
  // 树中定位（v4.25 A3）：目标行短暂高亮闪烁 + 滚动容器引用。
  const [flashPath, setFlashPath] = useState<string | null>(null);
  const scrollRef = useRef<HTMLDivElement | null>(null);
  const lastRevealNonceRef = useRef<number>(-1);

  // 树中定位：nonce 变化触发一次。展开父链（落盘持久化）、退出搜索模式回树、
  // 行渲染后滚动到可见并闪烁 REVEAL_FLASH_MS。行可能在父链目录异步加载完成
  // 后才出现，滚动用轮询兜底（最多 2s）。
  useEffect(() => {
    if (!revealRequest || !revealRequest.rel) return;
    if (revealRequest.nonce === lastRevealNonceRef.current) return;
    lastRevealNonceRef.current = revealRequest.nonce;
    const rel = revealRequest.rel;
    // 若正处于树内搜索模式，目标行不在树中 → 清空查询回树
    setQuery("");
    // 展开父链（含根行），与 toggle 同路径持久化
    setExpanded((prev) => {
      const next: Record<string, boolean> = { ...prev, "": true };
      for (const dir of parentDirsOf(rel)) next[dir] = true;
      persistExpanded(cwd, next);
      return next;
    });
    // 先置高亮（行渲染时即带闪烁样式），滚动等行出现后执行
    setFlashPath(rel);
    let tries = 0;
    let scrollTimer: number | undefined;
    const tryScroll = () => {
      const root = scrollRef.current;
      if (root) {
        let row: HTMLElement | null = null;
        for (const el of Array.from(root.querySelectorAll<HTMLElement>("[data-path]"))) {
          if (el.dataset.path === rel || el.dataset.chainPaths?.split("|").includes(rel)) {
            row = el;
            break;
          }
        }
        if (row) {
          if (typeof row.scrollIntoView === "function") row.scrollIntoView({ block: "nearest" });
          return;
        }
      }
      if (++tries <= REVEAL_SCROLL_RETRIES) {
        scrollTimer = window.setTimeout(tryScroll, REVEAL_SCROLL_INTERVAL_MS);
      }
    };
    tryScroll();
    const flashTimer = window.setTimeout(() => {
      setFlashPath((cur) => (cur === rel ? null : cur));
    }, REVEAL_FLASH_MS);
    return () => {
      if (scrollTimer !== undefined) window.clearTimeout(scrollTimer);
      window.clearTimeout(flashTimer);
    };
  }, [revealRequest, cwd]);

  const toggle = useCallback((rel: string, chainPaths?: string[]) => {
    setExpanded((prev) => {
      const links = chainPaths && chainPaths.length > 0 ? chainPaths : [rel];
      const isOpen = links.every((path) => path === "" ? (prev[""] ?? true) : prev[path] === true);
      const next = { ...prev };
      for (const path of links) next[path] = !isOpen;
      persistExpanded(cwd, next);
      return next;
    });
  }, [cwd]);

  const fetchDirs = useCallback(async (dirs: string[]) => {
    const wanted = [...new Set(dirs)].filter((dir) => !inflight.current.has(dir));
    if (wanted.length === 0) return;
    wanted.forEach((dir) => inflight.current.add(dir));
    setData((prev) => {
      const next = { ...prev };
      for (const dir of wanted) {
        next[dir] = prev[dir]?.entries !== undefined
          ? { loading: true, entries: prev[dir].entries }
          : { loading: true };
      }
      return next;
    });
    try {
      const batch = app.GaeaListDirBatch;
      if (typeof batch === "function") {
        const levels: WorkbenchLevel[] = await batch(wanted, treePrefs.excludes);
        const byPath = new Map((levels ?? []).map((level) => [level.path.replace(/\\/g, "/"), level]));
        setData((prev) => {
          const next = { ...prev };
          for (const dir of wanted) {
            const level = byPath.get(dir.replace(/\\/g, "/"));
            next[dir] = level
              ? (level.error
                  ? { loading: false, error: level.error }
                  : { loading: false, entries: level.entries ?? [] })
              : { loading: false, error: "批量列目录未返回该层级" };
          }
          return next;
        });
        return;
      }
      const rows = await Promise.all(
        wanted.map(async (dir) => {
          try {
            return { dir, entries: (await app.ListDir(dir)) ?? [] };
          } catch (error) {
            return { dir, error: error instanceof Error ? error.message : String(error) };
          }
        }),
      );
      setData((prev) => {
        const next = { ...prev };
        for (const row of rows) {
          next[row.dir] = "error" in row
            ? { loading: false, error: row.error }
            : { loading: false, entries: row.entries };
        }
        return next;
      });
    } finally {
      wanted.forEach((dir) => inflight.current.delete(dir));
    }
  }, [treePrefs.excludes]);

  // 加载根层 + 所有展开目录的层级。真实桌面优先一次 batch；浏览器 mock
  // 或旧后端缺少 batch 时逐层回退，开发态与历史测试保持可用。
  const visibleDirs = useMemo(() => {
    const dirs = [""];
    for (const [rel, open] of Object.entries(expanded)) {
      if (open === true && rel !== "") dirs.push(rel);
    }
    return dirs;
  }, [expanded]);

  useEffect(() => {
    const missing = visibleDirs.filter((dir) => data[dir] === undefined);
    if (missing.length > 0) void fetchDirs(missing);
  }, [visibleDirs, data, fetchDirs]);

  const compactTargets = useMemo(
    () => treePrefs.compactFolders ? compactPreloadTargets(data) : [],
    [data, treePrefs.compactFolders],
  );

  useEffect(() => {
    const missing = compactTargets.filter((dir) => data[dir] === undefined);
    if (missing.length > 0) void fetchDirs(missing);
  }, [compactTargets, data, fetchDirs]);

  const retry = useCallback((dir: string) => {
    void fetchDirs([dir]);
  }, [fetchDirs]);

  // 展开目录监听：后端按集合重建 fsnotify，事件只失效对应层级，前端再批量重列。
  useEffect(() => {
    const watch = app.GaeaFileTreeWatch;
    if (typeof watch !== "function") return;
    void watch(visibleDirs, treePrefs.excludes).catch((error) => {
      setActionError(`实时刷新不可用：${error instanceof Error ? error.message : String(error)}`);
    });
    return () => {
      void watch([], treePrefs.excludes).catch(() => {});
    };
  }, [visibleDirs, treePrefs.excludes]);

  useEffect(() => subscribe("gaea-filetree", (payload) => {
    const paths = (payload as { paths?: unknown } | null)?.paths;
    if (!Array.isArray(paths)) return;
    setData((prev) => {
      const next = { ...prev };
      for (const raw of paths) {
        if (typeof raw !== "string") continue;
        delete next[raw];
      }
      return next;
    });
    void refreshGitStatus();
  }), [refreshGitStatus]);

  const copyPath = useCallback((path: string) => {
    const p = navigator.clipboard?.writeText(path);
    if (!p) return;
    void p.then(() => {
      setCopiedPath(path);
      window.setTimeout(() => {
        setCopiedPath((cur) => (cur === path ? null : cur));
      }, COPIED_MS);
    }).catch(() => {
      // 剪贴板被拒绝/不可用：不给反馈，静默
    });
  }, []);

  const invalidateLevels = useCallback((paths: string[]) => {
    setData((prev) => {
      const next = { ...prev };
      for (const key of Object.keys(next)) {
        if (paths.some((path) => underPath(key, path))) delete next[key];
      }
      return next;
    });
  }, []);

  const visiblePaths = useMemo(() => {
    const out: string[] = [];
    const walk = (dir: string) => {
      const entries = (data[dir]?.entries ?? [])
        .slice()
        .sort((a, b) => {
          if (a.isDir !== b.isDir) return a.isDir ? -1 : 1;
          return a.name.localeCompare(b.name, undefined, { numeric: true, sensitivity: "base" });
        });
      for (const entry of entries) {
        const path = joinRel(dir, entry.name);
        out.push(path);
        if (entry.isDir) {
          const chain = treePrefs.compactFolders
            ? compactFolderChain(path, entry, (p) => data[p])
            : [{ path, entry }];
          if (chain.some((link) => expanded[link.path] === true)) {
            walk(chain[chain.length - 1].path);
          }
        }
      }
    };
    walk("");
    return out;
  }, [data, expanded, treePrefs.compactFolders]);

  const gitIndex = useMemo(() => buildGitStatusIndex(gitStatus), [gitStatus]);

  const handleRowClick = useCallback((
    event: { ctrlKey: boolean; metaKey: boolean; shiftKey: boolean },
    path: string,
    isDir: boolean,
    chainPaths?: string[],
  ) => {
    setActionError(null);
    setActionNotice(null);
    if (event.shiftKey && selectionAnchorRef.current) {
      const anchor = selectionAnchorRef.current;
      const a = visiblePaths.indexOf(anchor);
      const b = visiblePaths.indexOf(path);
      if (a >= 0 && b >= 0) {
        const start = Math.min(a, b);
        const end = Math.max(a, b);
        setSelection(new Set(visiblePaths.slice(start, end + 1)));
        return;
      }
    }
    if (event.ctrlKey || event.metaKey) {
      setSelection((prev) => {
        const next = new Set(prev);
        if (next.has(path)) next.delete(path);
        else next.add(path);
        return next;
      });
      selectionAnchorRef.current = path;
      return;
    }
    setSelection(new Set([path]));
    selectionAnchorRef.current = path;
    if (isDir) toggle(path, chainPaths);
    else onSelect(path);
  }, [onSelect, toggle, visiblePaths]);

  const openNameDialog = useCallback((state: NameDialogState) => {
    setActionError(null);
    setActionNotice(null);
    setNameDialog(state);
  }, []);

  const submitNameDialog = useCallback(async () => {
    if (!nameDialog || nameBusy) return;
    const name = nameDialog.value.trim();
    if (!name) {
      setActionError("名称不能为空");
      return;
    }
    setNameBusy(true);
    setActionError(null);
    try {
      if (nameDialog.kind === "rename") {
        const result = await app.GaeaRenameWorkspacePath(nameDialog.target, name);
        const oldParent = nameDialog.parent;
        invalidateLevels([oldParent, result.path]);
        if (nameDialog.isDir) {
          setExpanded((prev) => {
            const next: Record<string, boolean> = {};
            for (const [key, open] of Object.entries(prev)) {
              next[underPath(key, nameDialog.target) ? result.path + key.slice(nameDialog.target.length) : key] = open;
            }
            persistExpanded(cwd, next);
            return next;
          });
        }
        setSelection(new Set([result.path]));
        onPathRenamed?.(nameDialog.target, result.path, result.isDir);
        setActionNotice(`已重命名为 ${name}`);
      } else {
        const result = nameDialog.kind === "folder"
          ? await app.GaeaCreateWorkspaceFolder(nameDialog.parent, name)
          : await app.GaeaCreateWorkspaceFile(nameDialog.parent, name);
        invalidateLevels([nameDialog.parent]);
        setSelection(new Set([result.path]));
        selectionAnchorRef.current = result.path;
        setActionNotice(`已创建 ${result.path}`);
      }
      void refreshGitStatus();
      setNameDialog(null);
    } catch (error) {
      setActionError(error instanceof Error ? error.message : String(error));
    } finally {
      setNameBusy(false);
    }
  }, [cwd, invalidateLevels, nameBusy, nameDialog, onPathRenamed, refreshGitStatus]);

  const deletePaths = useCallback((paths: string[]) => {
    const selected = [...new Set(paths)].filter(Boolean);
    if (selected.length === 0) return;
    Modal.confirm({
      title: selected.length === 1 ? `删除 ${baseNameOf(selected[0])}？` : `删除所选 ${selected.length} 项？`,
      content: "将直接写入磁盘并永久删除，目录会连同其中内容一起移除。",
      okText: "永久删除",
      cancelText: "取消",
      okButtonProps: { danger: true },
      centered: true,
      onOk: async () => {
        try {
          await app.GaeaDeleteWorkspacePaths(selected);
          invalidateLevels([...selected.map(parentLevelOf), ...selected]);
          setExpanded((prev) => {
            const next = Object.fromEntries(
              Object.entries(prev).filter(([key]) => !selected.some((path) => underPath(key, path))),
            );
            persistExpanded(cwd, next);
            return next;
          });
          setSelection(new Set());
          selectionAnchorRef.current = null;
          onPathDeleted?.(selected);
          setActionNotice(`已删除 ${selected.length} 项`);
          void refreshGitStatus();
        } catch (error) {
          setActionError(error instanceof Error ? error.message : String(error));
          throw error;
        }
      },
    });
  }, [cwd, invalidateLevels, onPathDeleted, refreshGitStatus]);

  const savePathsAs = useCallback(async (paths: string[]) => {
    const selected = [...new Set(paths)].filter(Boolean);
    if (selected.length === 0) return;
    setActionError(null);
    try {
      const fn = selected.length === 1 ? app.GaeaSaveWorkspacePathAs : app.GaeaSaveWorkspacePathsAs;
      if (typeof fn !== "function") throw new Error("文件导出绑定不可用");
      const saved = selected.length === 1
        ? await app.GaeaSaveWorkspacePathAs(selected[0])
        : await app.GaeaSaveWorkspacePathsAs(selected);
      if (saved) setActionNotice(`已保存到 ${saved}`);
    } catch (error) {
      setActionError(error instanceof Error ? error.message : String(error));
    }
  }, []);

  const uploadFiles = useCallback(async (targetDir: string, files: File[]) => {
    if (files.length === 0) return;
    if (typeof app.GaeaWorkspaceUploadBegin !== "function") {
      setActionError("文件上传绑定不可用");
      return;
    }
    setActionError(null);
    setActionNotice(null);
    let completed = 0;
    try {
      for (const file of files) {
        setUploading(`上传 ${completed + 1}/${files.length}：${file.name}`);
        const relativePath =
          (file as File & { webkitRelativePath?: string }).webkitRelativePath || file.name;
        const token = await app.GaeaWorkspaceUploadBegin(targetDir, relativePath);
        try {
          for (let offset = 0; offset < file.size; offset += UPLOAD_CHUNK_BYTES) {
            const end = Math.min(file.size, offset + UPLOAD_CHUNK_BYTES);
            const bytes = new Uint8Array(await file.slice(offset, end).arrayBuffer());
            await app.GaeaWorkspaceUploadAppend(token, bytesToBase64(bytes));
          }
          await app.GaeaWorkspaceUploadFinish(token);
        } catch (error) {
          await app.GaeaWorkspaceUploadAbort(token).catch(() => {});
          throw error;
        }
        completed += 1;
      }
      invalidateLevels([targetDir]);
      setActionNotice(`已上传 ${completed} 个文件`);
      void refreshGitStatus();
    } catch (error) {
      setActionError(`上传失败：${error instanceof Error ? error.message : String(error)}`);
    } finally {
      setUploading(null);
    }
  }, [invalidateLevels, refreshGitStatus]);

  const chooseUpload = useCallback(async (targetDir: string, folder: boolean) => {
    uploadTargetRef.current = targetDir;
    const native = folder ? app.GaeaPickWorkspaceUploadFolder : app.GaeaPickWorkspaceUploadFiles;
    if (typeof native === "function") {
      setActionError(null);
      setActionNotice(null);
      setUploading(folder ? "正在上传文件夹…" : "正在上传文件…");
      try {
        const results = await native(targetDir);
        if (results.length > 0) {
          invalidateLevels([targetDir]);
          setActionNotice(`已上传 ${results.length} 个文件`);
          void refreshGitStatus();
        }
      } catch (error) {
        setActionError(`上传失败：${error instanceof Error ? error.message : String(error)}`);
      } finally {
        setUploading(null);
      }
      return;
    }
    const input = folder ? folderInputRef.current : filesInputRef.current;
    if (!input) {
      setActionError("上传入口不可用");
      return;
    }
    input.value = "";
    input.click();
  }, [invalidateLevels, refreshGitStatus]);

  const openTreePrefs = useCallback(() => {
    setExcludeDraft(treePrefs.excludes.join("\n"));
    setCompactDraft(treePrefs.compactFolders);
    setTreePrefsOpen(true);
  }, [treePrefs]);

  const applyTreePrefs = useCallback(() => {
    const next = saveFileTreePrefs(cwd, {
      excludes: normalizeFileTreeExcludes(excludeDraft.split(/\r?\n/)),
      compactFolders: compactDraft,
    });
    setTreePrefs(next);
    setTreePrefsOpen(false);
    setData({});
    setActionError(null);
    setActionNotice(next.excludes.length > 0
      ? `已应用 ${next.excludes.length} 条排除规则`
      : "已清除用户排除规则");
  }, [compactDraft, cwd, excludeDraft]);

  // 行尾操作区：复制成功 → 「已复制」标签（1.2s）替换 @ 按钮；仅 onReference 存在时渲染 @ 按钮。
  const rowActions = (path: string): ReactNode => {
    if (copiedPath === path) {
      return <span className="shrink-0 text-[9px] text-fg-faint">已复制</span>;
    }
    if (onReference === undefined) return null;
    return (
      <span className="shrink-0 opacity-0 group-hover:opacity-100 transition-opacity">
        <button
          type="button"
          aria-label="引用到输入框"
          title="引用到输入框"
          className="flex items-center justify-center w-5 h-5 rounded-md border-0 bg-transparent text-fg-faint cursor-pointer hover:text-accent hover:bg-bg-soft transition-colors"
          onClick={(e) => {
            e.stopPropagation();
            onReference(path);
          }}
        >
          <Paperclip size={11} />
        </button>
      </span>
    );
  };

  // 目录行右键菜单：创建/上传/下载/复制/改名/删除。
  const dirMenu = (path: string): MenuProps => ({
    items: [
      ...(path === ""
        ? [
            { key: "treePrefs", label: `文件树设置${treePrefs.excludes.length > 0 ? `（排除 ${treePrefs.excludes.length}）` : ""}`, icon: <Settings size={12} /> },
            { type: "divider" as const },
          ]
        : []),
      { key: "newFile", label: "新建文件", icon: <Plus size={12} /> },
      { key: "newFolder", label: "新建文件夹", icon: <FolderPlus size={12} /> },
      { key: "uploadFile", label: "上传文件", icon: <Plus size={12} /> },
      { key: "uploadFolder", label: "上传文件夹", icon: <FolderPlus size={12} /> },
      { type: "divider" as const },
      { key: "save", label: "下载 / 打包", icon: <Download size={12} />, disabled: path === "" },
      { key: "copy", label: "复制相对路径", icon: <Copy size={12} /> },
      ...((onOpenExternal !== undefined || onReveal !== undefined)
        ? [{
            key: "openWith",
            label: "打开方式",
            icon: <ExternalLink size={12} />,
            children: [
              ...(onOpenExternal !== undefined ? [{ key: "external", label: "系统默认程序" }] : []),
              ...(onReveal !== undefined ? [{ key: "reveal", label: "在资源管理器中显示", icon: <FolderOpen size={12} /> }] : []),
            ],
          }]
        : []),
      { type: "divider" as const },
      { key: "rename", label: "重命名", icon: <Pencil size={12} />, disabled: path === "" },
      { key: "delete", label: "永久删除", icon: <Trash2 size={12} />, danger: true, disabled: path === "" },
    ],
    onClick: ({ key }) => {
      if (key === "treePrefs") openTreePrefs();
      else if (key === "copy") copyPath(path);
      else if (key === "external") onOpenExternal?.(path);
      else if (key === "reveal") onReveal?.(path);
      else if (key === "newFile") openNameDialog({ kind: "file", parent: path, value: "未命名.md" });
      else if (key === "newFolder") openNameDialog({ kind: "folder", parent: path, value: "新建文件夹" });
      else if (key === "uploadFile") void chooseUpload(path, false);
      else if (key === "uploadFolder") void chooseUpload(path, true);
      else if (key === "save") void savePathsAs([path]);
      else if (key === "rename") {
        openNameDialog({ kind: "rename", target: path, parent: parentLevelOf(path), value: baseNameOf(path), isDir: true });
      } else if (key === "delete") deletePaths([path]);
    },
  });

  // 文件行右键菜单：预览/外部打开/定位/下载/复制/改名/删除。
  const fileMenu = (path: string): MenuProps => ({
    items: [
      { key: "preview", label: "预览", icon: <Eye size={12} /> },
      ...((onOpenExternal !== undefined || onReveal !== undefined)
        ? [{
            key: "openWith",
            label: "打开方式",
            icon: <ExternalLink size={12} />,
            children: [
              ...(onOpenExternal !== undefined ? [{ key: "external", label: "系统默认程序" }] : []),
              ...(onReveal !== undefined ? [{ key: "reveal", label: "在资源管理器中显示", icon: <FolderOpen size={12} /> }] : []),
            ],
          }]
        : []),
      { key: "save", label: "下载副本", icon: <Download size={12} /> },
      { key: "copy", label: "复制相对路径", icon: <Copy size={12} /> },
      { type: "divider" as const },
      { key: "rename", label: "重命名", icon: <Pencil size={12} /> },
      { key: "delete", label: "永久删除", icon: <Trash2 size={12} />, danger: true },
    ],
    onClick: ({ key }) => {
      // 「预览」= 主区预览入口（双入口保留）；行点击的右栏内 tab 打开由
      // onSelect 承担（v4.25 A3 起语义分叉，缺省回退保持旧调用方兼容）。
      if (key === "preview") (onOpenMainPreview ?? onSelect)(path);
      else if (key === "external") onOpenExternal?.(path);
      else if (key === "reveal") onReveal?.(path);
      else if (key === "save") void savePathsAs([path]);
      else if (key === "copy") copyPath(path);
      else if (key === "rename") {
        openNameDialog({ kind: "rename", target: path, parent: parentLevelOf(path), value: baseNameOf(path), isDir: false });
      } else if (key === "delete") deletePaths([path]);
    },
  });

  const onRowKeyDown = (e: KeyboardEvent<HTMLDivElement>, action: () => void): void => {
    if (e.key === "Enter" || e.key === " ") {
      e.preventDefault();
      action();
    }
  };

  const rootOpen = expanded[""] ?? true;
  const rootLoading = data[""]?.loading === true && data[""]?.entries === undefined;
  const rootName = cwd ? cwd.split(/[/\\]/).pop() || "工作区" : "工作区";
  const rootGitInk = gitIndex.dirs.get("");

  const dropProps = (targetDir: string) => ({
    onDragEnter: (event: DragEvent) => {
      if (!event.dataTransfer.types.includes("Files")) return;
      event.preventDefault();
      setDragTarget(targetDir);
    },
    onDragOver: (event: DragEvent) => {
      if (!event.dataTransfer.types.includes("Files")) return;
      event.preventDefault();
      event.dataTransfer.dropEffect = "copy";
      setDragTarget(targetDir);
    },
    onDragLeave: (event: DragEvent) => {
      if (!event.currentTarget.contains(event.relatedTarget as Node | null)) setDragTarget(null);
    },
    onDrop: (event: DragEvent) => {
      if (!event.dataTransfer.files.length) return;
      event.preventDefault();
      event.stopPropagation();
      setDragTarget(null);
      void uploadFiles(targetDir, Array.from(event.dataTransfer.files));
    },
  });

  const renderLevel = (dir: string, depth: number): ReactNode => {
    const level = data[dir];
    if (level !== undefined && level.error !== undefined && level.entries === undefined) {
      return (
        <div
          className="flex items-center gap-1 px-2 py-1 text-[10px] text-red-400"
          style={{ paddingLeft: `${8 + (depth + 1) * 14}px` }}
        >
          <span className="truncate">加载失败：{level.error}</span>
          <button
            className="shrink-0 border-0 bg-transparent text-accent cursor-pointer hover:underline"
            onClick={() => retry(dir)}
            type="button"
          >
            重试
          </button>
        </div>
      );
    }
    if (level === undefined || (level.loading === true && level.entries === undefined)) {
      return (
        <div
          className="text-fg-faint/50 text-[10px] text-center py-1"
          style={{ paddingLeft: `${8 + (depth + 1) * 14}px` }}
        >
          加载中…
        </div>
      );
    }
    const entries = level.entries ?? [];
    if (entries.length === 0) {
      return (
        <div
          className="text-fg-faint/40 text-[10px] text-center py-2"
          style={{ paddingLeft: `${22 + (depth + 1) * 14}px` }}
        >
          空目录
        </div>
      );
    }
    return (
      <div>
        {entries
          .sort((a, b) => {
            if (a.isDir !== b.isDir) return a.isDir ? -1 : 1;
            return a.name.localeCompare(b.name, undefined, { numeric: true, sensitivity: "base" });
          })
          .map((e) => {
            const childPath = joinRel(dir, e.name);
            if (e.isDir) {
              const chain = treePrefs.compactFolders
                ? compactFolderChain(childPath, e, (p) => data[p])
                : [{ path: childPath, entry: e }];
              const chainPaths = chain.map((link) => link.path);
              const tail = chain[chain.length - 1];
              const tailPath = tail.path;
              const isOpen = chainPaths.some((path) => expanded[path] === true);
              const childLoading = data[tailPath]?.loading === true && data[tailPath]?.entries === undefined;
              // v4.28：目录行也带 data-path 锚点 + 树中定位闪烁（sidebar_open
              // directory → revealRequest 高亮目录行），闪烁样式与文件行同款；
              // 展开箭头 / 目录菜单逻辑不变。
              const isFlashed = chainPaths.some((path) => path === flashPath);
              const isSelected = selection.has(childPath);
              const gitInk = gitIndex.dirs.get(gitPathKey(tailPath));
              return (
                <div key={childPath}>
                  <Dropdown trigger={["contextMenu"]} menu={dirMenu(tailPath)}>
                    <div
                      role="button"
                      tabIndex={0}
                      data-path={childPath}
                      data-chain-paths={chainPaths.length > 1 ? chainPaths.join("|") : undefined}
                      data-flash={isFlashed ? "true" : undefined}
                      data-selected={isSelected ? "true" : undefined}
                      data-git-tone={gitInk?.tone}
                      className={`group w-full flex items-center gap-1 px-2 py-1 border-0 bg-transparent text-left cursor-pointer transition-colors hover:bg-bg-soft ${
                        isFlashed
                          ? "bg-accent/25 text-accent" // 树中定位闪烁：REVEAL_FLASH_MS 后自动消退
                          : isSelected
                            ? "bg-accent/10 text-accent"
                            : dragTarget === childPath
                              ? "bg-accent/15 text-accent"
                              : "text-fg-dim"
                      }`}
                      style={{ paddingLeft: `${8 + depth * 14}px` }}
                      onClick={(event) => handleRowClick(event, childPath, true, chainPaths)}
                      onContextMenu={() => {
                        if (!selection.has(childPath)) setSelection(new Set([childPath]));
                        selectionAnchorRef.current = childPath;
                      }}
                      onKeyDown={(ev) => onRowKeyDown(ev, () => handleRowClick(
                        { ctrlKey: ev.ctrlKey, metaKey: ev.metaKey, shiftKey: ev.shiftKey },
                        childPath,
                        true,
                        chainPaths,
                      ))}
                      {...dropProps(childPath)}
                    >
                      {isOpen ? (
                        <ChevronDown size={10} className="shrink-0 text-fg-faint" />
                      ) : (
                        <ChevronRight size={10} className="shrink-0 text-fg-faint" />
                      )}
                      <FileTypeIcon name={tail.entry.name} isDir open={isOpen} size={14} />
                      <span className="truncate flex-1" title={tailPath}>{chain.map((link) => link.entry.name).join("/")}</span>
                      <GitDirDot ink={gitInk} />
                      {e.isSymlink && <span className="shrink-0 text-[9px] text-fg-faint">{e.broken ? "断链" : "链接"}</span>}
                      {childLoading && <span className="text-fg-faint text-[9px]">⋯</span>}
                      {rowActions(childPath)}
                    </div>
                  </Dropdown>
                  {isOpen && renderLevel(tailPath, depth + 1)}
                </div>
              );
            }
            const isSelected = selectedFile === childPath || selection.has(childPath);
            const isFlashed = flashPath === childPath;
            const gitInk = gitIndex.files.get(gitPathKey(childPath));
            return (
              <Dropdown key={childPath} trigger={["contextMenu"]} menu={fileMenu(childPath)}>
                <div
                  role="button"
                  tabIndex={0}
                  data-path={childPath}
                  data-flash={isFlashed ? "true" : undefined}
                  data-selected={selection.has(childPath) ? "true" : undefined}
                  data-git-tone={gitInk?.tone}
                  data-git-letter={gitInk?.letter}
                  className={`group w-full flex items-center gap-1 px-2 py-0.5 border-0 text-left cursor-pointer transition-colors ${
                    isFlashed
                      ? "bg-accent/25 text-accent" // 树中定位闪烁：比选中态更醒目，REVEAL_FLASH_MS 后自动消退
                      : isSelected
                        ? "bg-accent/10 text-accent"
                        : dragTarget === dir
                          ? "bg-accent/10 text-accent"
                          : "text-fg-dim hover:bg-bg-soft"
                  }`}
                  style={{ paddingLeft: `${22 + (depth + 1) * 14}px` }}
                  onClick={(event) => handleRowClick(event, childPath, false)}
                  onContextMenu={() => {
                    if (!selection.has(childPath)) setSelection(new Set([childPath]));
                    selectionAnchorRef.current = childPath;
                  }}
                  onKeyDown={(ev) => onRowKeyDown(ev, () => handleRowClick(
                    { ctrlKey: ev.ctrlKey, metaKey: ev.metaKey, shiftKey: ev.shiftKey },
                    childPath,
                    false,
                  ))}
                  {...dropProps(dir)}
                >
                  <FileTypeIcon name={e.name} size={14} />
                  <span
                    className="truncate flex-1"
                    style={gitInk ? { color: GIT_TONE_COLOR[gitInk.tone] } : undefined}
                  >
                    {e.name}
                  </span>
                  <GitFileBadge ink={gitInk} />
                  {e.isSymlink && <span className="shrink-0 text-[9px] text-fg-faint">{e.broken ? "断链" : "链接"}</span>}
                  {rowActions(childPath)}
                </div>
              </Dropdown>
            );
          })}
      </div>
    );
  };

  return (
    <div className="flex flex-col h-full text-[12px]">
      <div className="flex items-center px-2 py-1.5 text-fg-faint text-[10px] font-semibold uppercase tracking-wider">
        <span>文件</span>
        <button
          type="button"
          aria-label="文件树设置"
          title="文件树设置"
          className="ml-auto flex h-5 w-5 cursor-pointer items-center justify-center rounded border-0 bg-transparent text-fg-faint hover:bg-bg-soft hover:text-fg"
          onClick={openTreePrefs}
        >
          <Settings size={11} />
        </button>
      </div>
      <div className="relative px-2 pb-1.5">
        <input
          type="text"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="过滤文件名"
          spellCheck={false}
          aria-label="过滤文件名"
          className="w-full px-2 py-1 pr-6 text-[11px] border border-border-soft rounded-md bg-transparent text-fg placeholder:text-fg-faint/70 outline-none focus:border-accent transition-colors"
        />
        {query !== "" && (
          <button
            type="button"
            aria-label="清空搜索"
            title="清空搜索"
            className="absolute right-3 top-[calc(50%+3px)] -translate-y-1/2 flex items-center justify-center w-4 h-4 rounded-full border-0 bg-transparent text-fg-faint/70 cursor-pointer hover:text-fg hover:bg-bg-soft transition-colors"
            onClick={() => setQuery("")}
          >
            <X size={10} />
          </button>
        )}
      </div>
      {selection.size > 0 && (
        <div
          data-testid="filetree-selection-bar"
          className="mx-2 mb-1.5 flex items-center gap-1.5 rounded-md border border-accent/25 bg-accent/8 px-2 py-1 text-[10.5px] text-fg-dim"
        >
          <span className="font-medium text-accent">已选 {selection.size}</span>
          <button
            type="button"
            className="ml-auto rounded px-1.5 py-0.5 border-0 bg-transparent cursor-pointer hover:bg-bg-soft hover:text-fg"
            onClick={() => {
              void navigator.clipboard?.writeText([...selection].join("\n")).catch(() => {
                setActionError("剪贴板不可用");
              });
            }}
          >
            复制路径
          </button>
          <button
            type="button"
            className="rounded px-1.5 py-0.5 border-0 bg-transparent cursor-pointer hover:bg-bg-soft hover:text-fg"
            onClick={() => void savePathsAs([...selection])}
          >
            下载
          </button>
          <button
            type="button"
            className="rounded px-1.5 py-0.5 border-0 bg-transparent text-err cursor-pointer hover:bg-err/10"
            onClick={() => deletePaths([...selection])}
          >
            删除
          </button>
          <button
            type="button"
            aria-label="清空选择"
            className="flex h-4 w-4 items-center justify-center rounded border-0 bg-transparent text-fg-faint cursor-pointer hover:bg-bg-soft hover:text-fg"
            onClick={() => {
              setSelection(new Set());
              selectionAnchorRef.current = null;
            }}
          >
            <X size={9} />
          </button>
        </div>
      )}
      {(actionError || actionNotice || uploading) && (
        <div
          className={`mx-2 mb-1.5 rounded-md px-2 py-1 text-[10.5px] ${
            actionError ? "border border-err/25 bg-err/5 text-err" : "border border-border-soft bg-bg-soft/40 text-fg-dim"
          }`}
          role={actionError ? "alert" : undefined}
        >
          {actionError || uploading || actionNotice}
        </div>
      )}
      <div className="flex-1 overflow-y-auto" ref={scrollRef}>
        {needle === "" ? (
          <>
            <Dropdown trigger={["contextMenu"]} menu={dirMenu("")}>
              <div
                role="button"
                tabIndex={0}
                data-drop-root="true"
                data-git-tone={rootGitInk?.tone}
                className={`group w-full flex items-center gap-1 px-2 py-1 border-0 text-left cursor-pointer transition-colors font-semibold ${
                  dragTarget === "" ? "bg-accent/15 text-accent" : "bg-transparent text-fg hover:bg-bg-soft"
                }`}
                style={{ paddingLeft: 8 }}
                onClick={() => toggle("")}
                onContextMenu={() => {
                  setSelection(new Set());
                  selectionAnchorRef.current = null;
                }}
                onKeyDown={(e) => onRowKeyDown(e, () => toggle(""))}
                {...dropProps("")}
              >
                <FileTypeIcon name={rootName} isDir open={rootOpen} size={14} />
                <span className="truncate flex-1">{rootName}</span>
                <GitDirDot ink={rootGitInk} />
                {rootLoading && <span className="text-fg-faint text-[9px]">⋯</span>}
                {rowActions("")}
              </div>
            </Dropdown>
            {rootOpen && renderLevel("", 1)}
          </>
        ) : (
          <>
            {searching && searchHits === null && searchError === null && (
              <div className="px-2 py-1.5 text-fg-faint/60 text-[10px] text-center">搜索中…</div>
            )}
            {searchError !== null && (
              <div className="px-2 py-1.5 text-red-400 text-[10px] text-center">搜索失败：{searchError}</div>
            )}
            {!searching && searchError === null && searchHits !== null && (
              searchHits.length === 0 ? (
                <div className="px-2 py-1.5 text-fg-faint text-[10px] text-center">无匹配文件</div>
              ) : (
                searchHits.map((h) => {
                  const key = gitPathKey(h.path);
                  const fileInk = h.isDir ? undefined : gitIndex.files.get(key);
                  const dirInk = h.isDir ? gitIndex.dirs.get(key) : undefined;
                  return (
                    <div
                      key={h.path}
                      role="button"
                      tabIndex={0}
                      data-git-tone={(fileInk ?? dirInk)?.tone}
                      data-git-letter={fileInk?.letter}
                      className={`group w-full flex items-center gap-1 px-2 py-1 border-0 text-left cursor-pointer transition-colors ${
                        h.isDir ? "text-fg-dim cursor-default" : "text-fg-dim hover:bg-bg-soft"
                      }`}
                      style={{ paddingLeft: 8 }}
                      onClick={() => { if (!h.isDir) onSelect(h.path); }}
                      onKeyDown={(ev) => {
                        if ((ev.key === "Enter" || ev.key === " ") && !h.isDir) {
                          ev.preventDefault();
                          onSelect(h.path);
                        }
                      }}
                      title={h.isDir ? "目录命中：点击预览对目录无意义，可在树中浏览" : h.path}
                    >
                      <FileTypeIcon name={h.name} isDir={h.isDir} size={14} />
                      <span
                        className="truncate flex-1"
                        style={fileInk ? { color: GIT_TONE_COLOR[fileInk.tone] } : undefined}
                      >
                        {h.path}
                      </span>
                      {fileInk && <GitFileBadge ink={fileInk} />}
                      {dirInk && <GitDirDot ink={dirInk} />}
                      {!h.isDir && rowActions(h.path)}
                    </div>
                  );
                })
              )
            )}
            {!searching && searchError === null && searchHits !== null && (
              <div className="px-2 py-1 text-fg-faint/50 text-[9.5px] text-center">
                搜索范围：整个工作区（最多 {FILE_SEARCH_LIMIT} 条命中）
              </div>
            )}
          </>
        )}
      </div>
      <input
        ref={filesInputRef}
        type="file"
        multiple
        className="hidden"
        aria-label="上传文件"
        onChange={(event) => {
          const files = Array.from(event.target.files ?? []);
          void uploadFiles(uploadTargetRef.current, files);
        }}
      />
      <input
        ref={folderInputRef}
        type="file"
        multiple
        className="hidden"
        aria-label="上传文件夹"
        {...({ webkitdirectory: "" } as Record<string, string>)}
        onChange={(event) => {
          const files = Array.from(event.target.files ?? []);
          void uploadFiles(uploadTargetRef.current, files);
        }}
      />
      <Modal
        open={nameDialog !== null}
        title={
          nameDialog?.kind === "rename"
            ? "重命名"
            : nameDialog?.kind === "folder"
              ? "新建文件夹"
              : "新建文件"
        }
        okText={nameDialog?.kind === "rename" ? "重命名" : "创建"}
        cancelText="取消"
        confirmLoading={nameBusy}
        onOk={() => void submitNameDialog()}
        onCancel={() => {
          if (!nameBusy) {
            setNameDialog(null);
            setActionError(null);
          }
        }}
        destroyOnHidden
        centered
      >
        <Input
          autoFocus
          value={nameDialog?.value ?? ""}
          onChange={(event) => {
            if (!nameDialog) return;
            setNameDialog({ ...nameDialog, value: event.target.value });
          }}
          onPressEnter={() => void submitNameDialog()}
          aria-label="文件或文件夹名称"
        />
      </Modal>
      <Modal
        open={treePrefsOpen}
        title="文件树设置"
        okText="应用"
        cancelText="取消"
        onOk={applyTreePrefs}
        onCancel={() => setTreePrefsOpen(false)}
        destroyOnHidden
        centered
        width={520}
      >
        <div className="flex flex-col gap-3 text-[12px] text-fg-dim">
          <label htmlFor="filetree-excludes" className="font-medium text-fg">排除规则</label>
          <Input.TextArea
            id="filetree-excludes"
            aria-label="文件树排除规则"
            value={excludeDraft}
            onChange={(event) => setExcludeDraft(event.target.value)}
            rows={6}
            spellCheck={false}
            placeholder={"每行一条，例如：\ngenerated\n**/tmp\ndocs/private"}
          />
          <div className="-mt-1 text-[10.5px] leading-relaxed text-fg-faint">
            支持名称、工作区相对路径以及 *、?、** 通配符；最多 64 条。默认仍会排除 .git、node_modules、dist、build 等运行时目录。
          </div>
          <label className="flex cursor-pointer items-center gap-2 rounded-md border border-border-soft bg-bg-soft/30 px-2.5 py-2">
            <input
              type="checkbox"
              checked={compactDraft}
              onChange={(event) => setCompactDraft(event.target.checked)}
            />
            <span>
              <span className="block font-medium text-fg">折叠单子目录链</span>
              <span className="block text-[10.5px] text-fg-faint">把只有一层子目录的路径合并为 a/b/c 一行。</span>
            </span>
          </label>
        </div>
      </Modal>
    </div>
  );
}
