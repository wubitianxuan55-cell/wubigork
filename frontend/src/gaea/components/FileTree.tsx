import { useCallback, useEffect, useRef, useState, type KeyboardEvent, type ReactNode } from "react";
import { Dropdown } from "antd";
import type { MenuProps } from "antd";
import {
  ChevronRight,
  ChevronDown,
  File,
  Folder,
  Image,
  FileText,
  Paperclip,
  Copy,
  ExternalLink,
  FolderOpen,
  Eye,
  X,
} from "../icons";
import { app } from "../lib/bridge";
import type { DirEntry, FileSearchHit } from "../lib/types";
import { useDebouncedValue } from "../hooks/useDebouncedValue";

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

// 文件图标映射（按扩展名着色，办公文件优先）
function fileIcon(name: string, isDir: boolean) {
  if (isDir) return <Folder size={14} className="text-accent shrink-0" />;
  const ext = name.split(".").pop()?.toLowerCase() ?? "";
  if (["doc", "docx"].includes(ext))
    return <FileText size={14} className="text-sky-400 shrink-0" />;
  if (["xls", "xlsx", "csv"].includes(ext))
    return <FileText size={14} className="text-emerald-400 shrink-0" />;
  if (["ppt", "pptx"].includes(ext))
    return <FileText size={14} className="text-orange-400 shrink-0" />;
  if (ext === "pdf")
    return <FileText size={14} className="text-red-400 shrink-0" />;
  if (["png", "jpg", "jpeg", "gif", "webp", "bmp", "svg"].includes(ext))
    return <Image size={14} className="text-violet-400 shrink-0" />;
  if (["md", "txt", "json", "toml", "yaml", "yml", "xml", "html", "css", "js", "ts", "tsx", "jsx", "go", "py"].includes(ext))
    return <FileText size={14} className="text-fg-dim shrink-0" />;
  return <File size={14} className="text-fg-faint shrink-0" />;
}

// 每个目录层级的加载态（加载三态：loading / error+重试 / 已加载）。
interface LevelData {
  loading: boolean;
  entries?: DirEntry[];
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

export function FileTree({
  cwd,
  onSelect,
  selectedFile,
  onReference,
  onOpenExternal,
  onReveal,
  onOpenMainPreview,
  revealRequest,
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
      .FileSearch(needle, FILE_SEARCH_LIMIT)
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
  }, [needle]);
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
          if (el.dataset.path === rel) {
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

  const toggle = useCallback((rel: string) => {
    setExpanded((prev) => {
      const isOpen = rel === "" ? (prev[""] ?? true) : prev[rel] === true;
      const next = { ...prev, [rel]: !isOpen };
      persistExpanded(cwd, next);
      return next;
    });
  }, [cwd]);

  const fetchDir = useCallback((dir: string) => {
    if (inflight.current.has(dir)) return;
    inflight.current.add(dir);
    setData((prev) => (prev[dir] !== undefined ? prev : { ...prev, [dir]: { loading: true } }));
    void app.ListDir(dir)
      .then((es) => {
        inflight.current.delete(dir);
        setData((prev) => ({ ...prev, [dir]: { loading: false, entries: es ?? [] } }));
      })
      .catch((err) => {
        inflight.current.delete(dir);
        setData((prev) => ({
          ...prev,
          [dir]: { loading: false, error: err instanceof Error ? err.message : String(err) },
        }));
      });
  }, []);

  // 加载根层 + 所有展开目录的层级（已加载/加载中跳过；失败不自动重试，等待重试按钮）。
  useEffect(() => {
    const dirs = [""];
    for (const [rel, v] of Object.entries(expanded)) {
      if (v === true) dirs.push(rel);
    }
    for (const dir of dirs) {
      const level = data[dir];
      if (level === undefined || (level.loading === true && level.entries === undefined && level.error === undefined)) {
        fetchDir(dir);
      }
    }
  }, [expanded, data, fetchDir]);

  const retry = useCallback((dir: string) => {
    setData((prev) => ({ ...prev, [dir]: { loading: true } }));
    fetchDir(dir);
  }, [fetchDir]);

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

  // 目录行右键菜单：复制相对路径。
  const dirMenu = (path: string): MenuProps => ({
    items: [{ key: "copy", label: "复制相对路径", icon: <Copy size={12} /> }],
    onClick: ({ key }) => {
      if (key === "copy") copyPath(path);
    },
  });

  // 文件行右键菜单：预览 · 在外部程序中打开（可选）· 在文件夹中显示（可选）· 复制相对路径。
  const fileMenu = (path: string): MenuProps => ({
    items: [
      { key: "preview", label: "预览", icon: <Eye size={12} /> },
      ...(onOpenExternal !== undefined
        ? [{ key: "external", label: "在外部程序中打开", icon: <ExternalLink size={12} /> }]
        : []),
      ...(onReveal !== undefined
        ? [{ key: "reveal", label: "在文件夹中显示", icon: <FolderOpen size={12} /> }]
        : []),
      { key: "copy", label: "复制相对路径", icon: <Copy size={12} /> },
    ],
    onClick: ({ key }) => {
      // 「预览」= 主区预览入口（双入口保留）；行点击的右栏内 tab 打开由
      // onSelect 承担（v4.25 A3 起语义分叉，缺省回退保持旧调用方兼容）。
      if (key === "preview") (onOpenMainPreview ?? onSelect)(path);
      else if (key === "external") onOpenExternal?.(path);
      else if (key === "reveal") onReveal?.(path);
      else if (key === "copy") copyPath(path);
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
          .filter((e) => !e.name.startsWith("."))
          .sort((a, b) => {
            if (a.isDir !== b.isDir) return a.isDir ? -1 : 1;
            return a.name.localeCompare(b.name, undefined, { numeric: true, sensitivity: "base" });
          })
          .map((e) => {
            const childPath = dir === "" ? e.name : `${dir}/${e.name}`;
            if (e.isDir) {
              const isOpen = expanded[childPath] === true;
              const childLoading = data[childPath]?.loading === true && data[childPath]?.entries === undefined;
              return (
                <div key={childPath}>
                  <Dropdown trigger={["contextMenu"]} menu={dirMenu(childPath)}>
                    <div
                      role="button"
                      tabIndex={0}
                      className="group w-full flex items-center gap-1 px-2 py-1 border-0 bg-transparent text-left cursor-pointer transition-colors hover:bg-bg-soft text-fg-dim"
                      style={{ paddingLeft: `${8 + depth * 14}px` }}
                      onClick={() => toggle(childPath)}
                      onKeyDown={(ev) => onRowKeyDown(ev, () => toggle(childPath))}
                    >
                      {isOpen ? (
                        <ChevronDown size={10} className="shrink-0 text-fg-faint" />
                      ) : (
                        <ChevronRight size={10} className="shrink-0 text-fg-faint" />
                      )}
                      {fileIcon(e.name, true)}
                      <span className="truncate flex-1">{e.name}</span>
                      {childLoading && <span className="text-fg-faint text-[9px]">⋯</span>}
                      {rowActions(childPath)}
                    </div>
                  </Dropdown>
                  {isOpen && renderLevel(childPath, depth + 1)}
                </div>
              );
            }
            const isSelected = selectedFile === childPath;
            const isFlashed = flashPath === childPath;
            return (
              <Dropdown key={childPath} trigger={["contextMenu"]} menu={fileMenu(childPath)}>
                <div
                  role="button"
                  tabIndex={0}
                  data-path={childPath}
                  data-flash={isFlashed ? "true" : undefined}
                  className={`group w-full flex items-center gap-1 px-2 py-0.5 border-0 text-left cursor-pointer transition-colors ${
                    isFlashed
                      ? "bg-accent/25 text-accent" // 树中定位闪烁：比选中态更醒目，REVEAL_FLASH_MS 后自动消退
                      : isSelected
                        ? "bg-accent/10 text-accent"
                        : "text-fg-dim hover:bg-bg-soft"
                  }`}
                  style={{ paddingLeft: `${22 + (depth + 1) * 14}px` }}
                  onClick={() => onSelect(childPath)}
                  onKeyDown={(ev) => onRowKeyDown(ev, () => onSelect(childPath))}
                >
                  {fileIcon(e.name, false)}
                  <span className="truncate flex-1">{e.name}</span>
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
      <div className="px-2 py-1.5 text-fg-faint text-[10px] font-semibold uppercase tracking-wider">文件</div>
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
      <div className="flex-1 overflow-y-auto" ref={scrollRef}>
        {needle === "" ? (
          <>
            <Dropdown trigger={["contextMenu"]} menu={dirMenu("")}>
              <div
                role="button"
                tabIndex={0}
                className="group w-full flex items-center gap-1 px-2 py-1 border-0 bg-transparent text-left cursor-pointer transition-colors hover:bg-bg-soft font-semibold text-fg"
                style={{ paddingLeft: 8 }}
                onClick={() => toggle("")}
                onKeyDown={(e) => onRowKeyDown(e, () => toggle(""))}
              >
                {fileIcon(rootName, true)}
                <span className="truncate flex-1">{rootName}</span>
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
                searchHits.map((h) => (
                  <div
                    key={h.path}
                    role="button"
                    tabIndex={0}
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
                    {h.isDir ? (
                      <Folder size={14} className="text-accent shrink-0" />
                    ) : (
                      fileIcon(h.name, false)
                    )}
                    <span className="truncate flex-1">{h.path}</span>
                    {!h.isDir && rowActions(h.path)}
                  </div>
                ))
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
    </div>
  );
}
