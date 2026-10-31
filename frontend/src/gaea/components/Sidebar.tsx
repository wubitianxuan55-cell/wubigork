import type { Dispatch, KeyboardEvent, PointerEvent as ReactPointerEvent, SetStateAction } from "react";
import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import { List, type RowComponentProps } from "react-window";
import {
  Plus, Blocks, BookOpen, MessageSquare, Search,
  PanelLeftClose, PanelLeftOpen, Loader2, ChevronDown, FolderGit2,
  Pin, Inbox, Rollback, X,
} from "../icons";
import logoSvg from "../assets/logo.svg";
import logoLightSvg from "../assets/logo-light.svg";
import { useT } from "../lib/i18n";
import { sessionTitle } from "../lib/session";
import { relativeTime } from "../lib/time";
import { filterProjectGroups } from "../lib/projectGroups";
import type { JobView, ProjectGroup, SessionMeta, SubagentRunView } from "../lib/types";
import { reloadSubagentRuns, subscribeSubagentRuns } from "../lib/subagentRunsStore";
import { useDebouncedValue } from "../hooks/useDebouncedValue";
import type { SubagentThreadStatus } from "./SubagentThread";

export interface SidebarProps {
  collapsed: boolean;
  toggleSidebar: () => void;
  running: boolean;
  jobs: JobView[];
  newSessionAndReset: () => void;
  // 会话模块：按项目分组（当前工作区在前，其余为最近打开过的工作区）
  projectGroups: ProjectGroup[];
  searchQuery: string;
  onSearchChange: (q: string) => void;
  onResumeSessionInProject: (path: string, projectPath: string) => void;
  onArchiveSession: (path: string) => void;
  onRestoreSession: (path: string, projectPath: string) => void;
  onPinSession: (path: string, pinned: boolean) => void;
  onDeleteSession: (path: string) => void;
  onRenameSession: (path: string, title: string) => void;
  onOpenSubagentThread: (p: {
    sessionPath: string;
    ref: string;
    task?: string;
    model?: string;
    status: SubagentThreadStatus;
  }) => void;
  onOpenHistory: () => void;
  onOpenCaps: () => void;
  onOpenKnowledge: () => void;
  startResize: (e: ReactPointerEvent<HTMLButtonElement>) => void;
  resizeWithKeyboard: (e: KeyboardEvent<HTMLButtonElement>) => void;
  onDoubleClickResize: () => void;
  sidebarWidth: number;
  SIDEBAR_MIN_WIDTH: number;
  SIDEBAR_MAX_WIDTH: number;
}

const PROJECTS_OPEN_KEY = "gaea.sidebar.projectsOpen";
const PER_PROJECT_PAGE = 8;

/** 虚拟滚动固定行高（px）：会话项含预览两行约 46px，组头/归档行更矮，取整行估算 44px。 */
const SESSION_ROW_HEIGHT = 44;
/** jsdom/无布局环境下容器的兜底高度；真实浏览器由 ResizeObserver 测得实际高度。 */
const SIDEBAR_VIEWPORT_FALLBACK = 480;

// 过渡期：SessionMeta.interrupted 由契约层子代理统一补充（后端列表接口新增字段），
// 此处以内联类型断言读取；契约层补齐字段定义后可移除断言。
const isInterruptedSession = (s: SessionMeta): boolean =>
  (s as SessionMeta & { interrupted?: boolean }).interrupted === true;

// ── 虚拟滚动行模型：把「分组头 / 空态 / 会话 / 显示更多 / 归档头 / 归档项」拍平成等高的行 ──
type SessionRowItem =
  | { key: string; kind: "group"; g: ProjectGroup; open: boolean }
  | { key: string; kind: "empty"; g: ProjectGroup }
  | { key: string; kind: "session"; s: SessionMeta; g: ProjectGroup }
  | { key: string; kind: "subagent"; s: SessionMeta; g: ProjectGroup; r: SubagentRunView }
  | { key: string; kind: "subagentNote"; s: SessionMeta; g: ProjectGroup; note: "loading" | "empty" | "error" }
  | { key: string; kind: "showMore"; g: ProjectGroup; hidden: number }
  | { key: string; kind: "archHeader"; g: ProjectGroup; open: boolean }
  | { key: string; kind: "archived"; s: SessionMeta; g: ProjectGroup };

type SubagentChildrenState = {
  status: "loading" | "ready" | "error";
  runs: SubagentRunView[];
};

/** SubagentRunView.status → 独立子代理会话 tab 的 SubagentThread 状态口径。 */
function subagentThreadStatus(r: SubagentRunView): SubagentThreadStatus {
  if (r.status === "running") return "running";
  if (r.status === "failed") return "failed";
  return "completed";
}

interface SessionRowUI {
  running: boolean;
  deleteConfirm: string | null;
  setDeleteConfirm: Dispatch<SetStateAction<string | null>>;
  renameTarget: string | null;
  renameDraft: string;
  setRenameDraft: Dispatch<SetStateAction<string>>;
  setRenameTarget: Dispatch<SetStateAction<string | null>>;
  revealed: Record<string, boolean>;
  setRevealed: Dispatch<SetStateAction<Record<string, boolean>>>;
  archivedOpen: Record<string, boolean>;
  setArchivedOpen: Dispatch<SetStateAction<Record<string, boolean>>>;
  toggleProject: (path: string, defaultOpen: boolean) => void;
  isProjectOpen: (g: ProjectGroup) => boolean;
  onResumeSessionInProject: (path: string, projectPath: string) => void;
  onArchiveSession: (path: string) => void;
  onRestoreSession: (path: string, projectPath: string) => void;
  onPinSession: (path: string, pinned: boolean) => void;
  onDeleteSession: (path: string) => void;
  onRenameSession: (path: string, title: string) => void;
  onOpenSubagentThread: (p: {
    sessionPath: string;
    ref: string;
    task?: string;
    model?: string;
    status: SubagentThreadStatus;
  }) => void;
  subagentOpen: Record<string, boolean>;
  subagents: Record<string, SubagentChildrenState>;
  toggleSubagents: (path: string) => void;
  reloadSubagents: (path: string) => void;
}

/**
 * 单行渲染组件（react-window rowComponent）：
 * 行内容与原有 renderSessionItem / renderProjectGroup 保持一一对应，
 * 选中恢复、双击重命名、置顶/归档/删除、归档恢复等交互不变。
 */
function SessionRow({ index, style, ariaAttributes, rows, ui }: RowComponentProps<{ rows: SessionRowItem[]; ui: SessionRowUI }>) {
  const t = useT();
  const row = rows[index];

  if (row.kind === "group") {
    return (
      <div style={style} {...ariaAttributes} className="office-tree-row" data-sidebar-row="group">
        <button
          className={`office-project-row no-drag ${row.g.current ? "is-current" : ""}`}
          onClick={() => ui.toggleProject(row.g.path, row.g.current)}
          title={row.g.path}
        >
          <span
            className="shrink-0 text-fg-faint/60 transition-transform duration-[var(--dur-fast)]"
            style={{ transform: row.open ? "rotate(90deg)" : "none" }}
          >
            <ChevronDown size={12} />
          </span>
          <FolderGit2 size={14} className={`shrink-0 ${row.g.current ? "text-accent" : "text-fg-faint"}`} />
          <span className={`flex-1 min-w-0 truncate text-left text-[12.5px] font-medium ${row.g.current ? "text-fg" : "text-fg-dim"}`}>
            {row.g.name}
          </span>
          {row.g.current && (
            <span className="office-project-current shrink-0 text-accent text-[10px] font-medium">{t("sidebar.currentBadge")}</span>
          )}
          <span className="office-project-count shrink-0 text-fg-faint/55 font-mono text-[10px]">{row.g.sessions.length}</span>
        </button>
      </div>
    );
  }

  if (row.kind === "empty") {
    return (
      <div style={style} {...ariaAttributes} className="office-tree-row" data-sidebar-row="empty">
        <div className="office-tree-empty">
          {row.g.archived.length === 0 ? (row.g.current ? t("sidebar.emptyCurrent") : t("sidebar.emptyProject")) : t("sidebar.noActive")}
        </div>
      </div>
    );
  }

  if (row.kind === "showMore") {
    return (
      <div style={style} {...ariaAttributes} className="office-tree-row" data-sidebar-row="showMore">
        <button
          className="office-tree-more"
          onClick={() => ui.setRevealed((prev) => ({ ...prev, [row.g.path]: true }))}
          type="button"
        >
          {t("sidebar.showMore", { n: row.hidden })}
        </button>
      </div>
    );
  }

  if (row.kind === "archHeader") {
    return (
      <div style={style} {...ariaAttributes} className="office-tree-row" data-sidebar-row="archHeader">
        <button
          className="office-archive-row no-drag"
          onClick={() => ui.setArchivedOpen((prev) => ({ ...prev, [row.g.path]: !prev[row.g.path] }))}
          title={t("sidebar.archivedTitle")}
        >
          <span
            className="shrink-0 text-fg-faint/60 transition-transform duration-[var(--dur-fast)]"
            style={{ transform: row.open ? "rotate(90deg)" : "none" }}
          >
            <ChevronDown size={11} />
          </span>
          <Inbox size={12} className="shrink-0" />
          <span className="text-[11.5px]">{t("sidebar.archived")}</span>
          <span className="ml-auto font-mono text-[10px] text-fg-faint/55">{row.g.archived.length}</span>
        </button>
      </div>
    );
  }

  if (row.kind === "session") {
    const s = row.s;
    const projectPath = row.g.path;
    const openSubagents = !!ui.subagentOpen[s.path];
    return (
      <div style={style} {...ariaAttributes} className="office-tree-row" data-sidebar-row="session">
        <div
          className={`office-session-row group ${
            s.current ? "sidebar-session-active" : ""
          }`}
        >
          <button
            className={`office-session-branch-toggle ${openSubagents ? "is-open" : ""}`}
            onClick={(e) => { e.stopPropagation(); ui.toggleSubagents(s.path); }}
            title={openSubagents ? t("sidebar.subagentsCollapse") : t("sidebar.subagentsExpand")}
            aria-expanded={openSubagents}
            data-sidebar-subagent-toggle={s.path}
            type="button"
          >
            <ChevronDown
              size={11}
              className="transition-transform duration-[var(--dur-fast)]"
              style={{ transform: openSubagents ? "rotate(180deg)" : "none" }}
            />
          </button>
          <button
            className="office-session-main"
            onClick={() => void ui.onResumeSessionInProject(s.path, projectPath)}
            disabled={ui.running || s.current}
            title={s.path}
          >
            <MessageSquare
              size={13}
              className={`office-session-icon ${s.current ? "text-accent" : "text-fg-faint/60"}`}
            />
            <span className="flex min-w-0 flex-1 flex-col gap-0.5">
              {ui.renameTarget === s.path ? (
                <input
                  className="w-full bg-bg border border-accent rounded px-1 py-0 text-fg text-[12.5px] outline-none"
                  value={ui.renameDraft}
                  onChange={e => ui.setRenameDraft(e.target.value)}
                  onKeyDown={e => {
                    if (e.key === "Enter") { e.preventDefault(); void ui.onRenameSession(s.path, ui.renameDraft.trim() || sessionTitle(s, "")); ui.setRenameTarget(null); }
                    if (e.key === "Escape") { e.preventDefault(); ui.setRenameTarget(null); }
                  }}
                  onBlur={() => { void ui.onRenameSession(s.path, ui.renameDraft.trim() || sessionTitle(s, "")); ui.setRenameTarget(null); }}
                  autoFocus
                  onClick={e => e.stopPropagation()}
                />
              ) : (
                <>
                  <span className="flex items-baseline gap-2 min-w-0">
                    <span
                      className={`office-session-title overflow-hidden text-ellipsis whitespace-nowrap text-[12.5px] leading-[1.35] font-medium cursor-text ${
                        s.current ? "text-accent" : "text-fg-dim"
                      }`}
                      onDoubleClick={e => {
                        if (s.current) return;
                        e.stopPropagation();
                        ui.setRenameTarget(s.path);
                        ui.setRenameDraft(sessionTitle(s, ""));
                      }}
                      title={t("sidebar.renameHint")}
                    >
                      {sessionTitle(s, t("history.emptySession"))}
                    </span>
                    {isInterruptedSession(s) && (
                      <span
                        className="office-session-state shrink-0 self-center bg-warning/15 text-[10px] leading-[1.5] font-medium text-warning"
                        title={t("sidebar.interruptedTitle")}
                      >
                        {t("sidebar.interrupted")}
                      </span>
                    )}
                    {s.pinned && (
                      <Pin size={10} className="shrink-0 text-accent/80" aria-label={t("sidebar.pinnedAria")} />
                    )}
                    <span className="office-session-time shrink-0 ml-auto text-fg-faint/70 font-mono text-[10px] tabular-nums">
                      {s.current ? t("history.current") : relativeTime(s.modTime)}
                    </span>
                  </span>
                  {!s.current && s.preview && s.preview !== sessionTitle(s, "") && (
                    <span className="office-session-preview flex-1 min-w-0 overflow-hidden text-ellipsis whitespace-nowrap text-fg-faint/60 text-[11px] leading-snug">
                      {s.preview}
                    </span>
                  )}
                </>
              )}
            </span>
          </button>
          {!s.current && (
            ui.deleteConfirm === s.path ? (
              <span className="flex items-center gap-1 shrink-0 mt-1">
                <button className="bg-transparent border-0 text-[10px] text-err cursor-pointer px-1 py-0.5 rounded hover:bg-err/10" onClick={e => { e.stopPropagation(); void ui.onDeleteSession(s.path); ui.setDeleteConfirm(null); }}>
                  {t("common.confirm")}
                </button>
                <button className="bg-transparent border-0 text-[10px] text-fg-faint cursor-pointer px-1 py-0.5 rounded hover:bg-bg-soft" onClick={e => { e.stopPropagation(); ui.setDeleteConfirm(null); }}>
                  {t("common.cancel")}
                </button>
              </span>
            ) : (
              <span className="office-session-actions hidden group-hover:flex items-center gap-0.5 shrink-0 mt-0.5">
                <button
                  className="flex items-center justify-center w-5 h-5 rounded-md bg-transparent border-0 text-fg-faint cursor-pointer hover:text-accent hover:bg-bg-soft transition-colors"
                  title={s.pinned ? t("sidebar.unpin") : t("sidebar.pin")}
                  onClick={e => { e.stopPropagation(); ui.onPinSession(s.path, !s.pinned); }}
                >
                  <Pin size={12} className={s.pinned ? "text-accent" : ""} />
                </button>
                <button
                  className="flex items-center justify-center w-5 h-5 rounded-md bg-transparent border-0 text-fg-faint cursor-pointer hover:text-fg hover:bg-bg-soft transition-colors"
                  title={t("sidebar.archiveTitle")}
                  onClick={e => { e.stopPropagation(); ui.onArchiveSession(s.path); }}
                >
                  <Inbox size={12} />
                </button>
                <button
                  className="flex items-center justify-center w-5 h-5 rounded-md bg-transparent border-0 text-fg-faint text-[13px] cursor-pointer hover:text-err hover:bg-bg-soft transition-colors"
                  title={t("common.delete")}
                  onClick={e => { e.stopPropagation(); ui.setDeleteConfirm(s.path); }}
                >
                  <X size={12} />
                </button>
              </span>
            )
          )}
        </div>
      </div>
    );
  }

  if (row.kind === "subagentNote") {
    const s = row.s;
    const loading = row.note === "loading";
    const error = row.note === "error";
    return (
      <div style={style} {...ariaAttributes} className="office-tree-row" data-sidebar-row="subagentNote">
        <div
          className={`office-subagent-note flex items-center gap-1.5 text-[11px] leading-snug ${
            error ? "text-err/85 cursor-pointer hover:text-err" : "text-fg-faint/80"
          }`}
          onClick={error ? () => ui.reloadSubagents(s.path) : undefined}
          role={error ? "button" : undefined}
          data-sidebar-subagent-note={`${s.path}:${row.note}`}
        >
          {loading && <Loader2 size={11} className="shrink-0 animate-spin text-accent" />}
          <span className="min-w-0 truncate">
            {loading
              ? t("sidebar.subagentsLoading")
              : error
                ? t("sidebar.subagentsError")
                : t("sidebar.subagentsEmpty")}
          </span>
        </div>
      </div>
    );
  }

  if (row.kind === "subagent") {
    const s = row.s;
    const r = row.r;
    const statusLabel =
      r.status === "running"
        ? t("subagent.statusRunning")
        : r.status === "failed"
          ? t("subagent.statusFailed")
          : t("subagent.statusDone");
    const updated = new Date(r.updatedAt).getTime();
    const timeText = Number.isFinite(updated) ? relativeTime(updated) : "";
    return (
      <div style={style} {...ariaAttributes} className="office-tree-row" data-sidebar-row="subagent">
        <div className="office-subagent-row group">
          <button
            className="office-subagent-main"
            onClick={() =>
              ui.onOpenSubagentThread({
                sessionPath: s.path,
                ref: r.ref,
                task: r.task || r.lastText || undefined,
                model: r.model,
                status: subagentThreadStatus(r),
              })
            }
            title={t("sidebar.subagentsOpen")}
            data-sidebar-subagent-row={`${s.path}:${r.ref}`}
            type="button"
          >
            <span
              className={`mt-[5px] h-1.5 w-1.5 rounded-full shrink-0 ${
                r.status === "running"
                  ? "bg-accent animate-pulse"
                  : r.status === "failed"
                    ? "bg-err"
                    : "bg-ok"
              }`}
            />
            <span className="flex min-w-0 flex-1 flex-col gap-0.5">
              <span className="overflow-hidden text-ellipsis whitespace-nowrap text-[12px] leading-[1.35] text-fg-dim">
                {r.task || t("sidebar.subagentsUntitled")}
              </span>
              <span className="overflow-hidden text-ellipsis whitespace-nowrap text-[10.5px] leading-snug text-fg-faint/70">
                <span className={r.status === "running" ? "text-accent" : ""}>{statusLabel}</span>
                {" · "}
                {r.kind === "model_tool"
                  ? `${t("subagent.modelToolLabel")} · `
                  : r.model
                    ? `${r.model} · `
                    : ""}
                {timeText}
              </span>
            </span>
          </button>
        </div>
      </div>
    );
  }

  // archived
  const s = row.s;
  return (
    <div style={style} {...ariaAttributes} className="office-tree-row" data-sidebar-row="archived">
      <div
        className="office-archived-row group"
      >
        <button
          className="flex items-center gap-2 flex-1 min-w-0 bg-transparent border-0 text-left cursor-pointer"
          onClick={() => ui.onRestoreSession(s.path, row.g.path)}
          title={t("sidebar.restoreRunTitle")}
        >
          <MessageSquare size={12} className="shrink-0 text-fg-faint/50" />
          <span className="flex-1 min-w-0 truncate text-[12px] text-fg-faint">{sessionTitle(s, t("history.emptySession"))}</span>
          {isInterruptedSession(s) && (
            <span
              className="shrink-0 rounded-full bg-warning/15 px-1.5 py-px text-[10px] leading-[1.5] font-medium text-warning"
              title={t("sidebar.interruptedTitle")}
            >
              {t("sidebar.interrupted")}
            </span>
          )}
          <span className="shrink-0 text-fg-faint/50 font-mono text-[10px]">{relativeTime(s.modTime)}</span>
        </button>
        {ui.deleteConfirm === s.path ? (
          <span className="flex items-center gap-1 shrink-0">
            <button
              className="bg-transparent border-0 text-[10px] text-err cursor-pointer px-1 py-0.5 rounded hover:bg-err/10"
              onClick={e => { e.stopPropagation(); void ui.onDeleteSession(s.path); ui.setDeleteConfirm(null); }}
            >
              {t("common.confirm")}
            </button>
            <button
              className="bg-transparent border-0 text-[10px] text-fg-faint cursor-pointer px-1 py-0.5 rounded hover:bg-bg-soft"
              onClick={e => { e.stopPropagation(); ui.setDeleteConfirm(null); }}
            >
              {t("common.cancel")}
            </button>
          </span>
        ) : (
          <span className="hidden group-hover:flex items-center gap-0.5 shrink-0">
            <button
              className="flex items-center justify-center w-5 h-5 rounded-md bg-transparent border-0 text-fg-faint cursor-pointer hover:text-accent hover:bg-bg-soft transition-colors"
              title={t("sidebar.restoreTitle")}
              onClick={() => ui.onRestoreSession(s.path, row.g.path)}
            >
              <Rollback size={12} />
            </button>
            <button
              className="flex items-center justify-center w-5 h-5 rounded-md bg-transparent border-0 text-fg-faint text-[13px] cursor-pointer hover:text-err hover:bg-bg-soft transition-colors"
              title={t("sidebar.deleteForever")}
              onClick={e => { e.stopPropagation(); ui.setDeleteConfirm(s.path); }}
            >
              <X size={12} />
            </button>
          </span>
        )}
      </div>
    </div>
  );
}

export function Sidebar({
  collapsed,
  toggleSidebar,
  running,
  jobs,
  newSessionAndReset,
  projectGroups,
  searchQuery,
  onSearchChange,
  onResumeSessionInProject,
  onArchiveSession,
  onRestoreSession,
  onPinSession,
  onDeleteSession,
  onRenameSession,
  onOpenSubagentThread,
  onOpenHistory,
  onOpenCaps,
  onOpenKnowledge,
  resizeWithKeyboard,
  startResize,
  onDoubleClickResize,
  sidebarWidth,
  SIDEBAR_MIN_WIDTH,
  SIDEBAR_MAX_WIDTH,
}: SidebarProps) {
  const t = useT();
  const toggleTitle = collapsed ? t("sidebar.expand") : t("sidebar.collapse");
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null);
  const [renameTarget, setRenameTarget] = useState<string | null>(null);
  const [renameDraft, setRenameDraft] = useState("");
  const [localQuery, setLocalQuery] = useState(searchQuery);
  const [projectsOpen, setProjectsOpen] = useState<Record<string, boolean>>(() => {
    try {
      const raw = localStorage.getItem(PROJECTS_OPEN_KEY);
      if (raw) return JSON.parse(raw) as Record<string, boolean>;
    } catch { /* ignore */ }
    return {};
  });
  // 分组内「显示更多」：一次会话超过 PER_PROJECT_PAGE 时先折叠
  const [revealed, setRevealed] = useState<Record<string, boolean>>({});
  // 各项目「已归档」分组默认折叠
  const [archivedOpen, setArchivedOpen] = useState<Record<string, boolean>>({});
  // 父会话行展开状态（better-sidebar 子代理入口：会话 → 子代理子行）
  const [subagentOpen, setSubagentOpen] = useState<Record<string, boolean>>({});
  // 各展开行的 GaeaSubagentRuns 快照（v4.64 迁共享单轮询 store：本组件不再
  // 自管懒加载缓存与 running 门控轮询定时器——展开即订阅、收起即注销，
  // loading/ready/error 状态由 store 随快照广播，失败行点击即 store 重拉）。
  const [subagents, setSubagents] = useState<Record<string, SubagentChildrenState>>({});

  const toggleSubagents = useCallback(
    (path: string) => {
      setSubagentOpen((prev) => ({ ...prev, [path]: !prev[path] }));
    },
    [],
  );

  const reloadSubagents = useCallback(
    (path: string) => {
      setSubagentOpen((prev) => ({ ...prev, [path]: true }));
      reloadSubagentRuns(path); // 未订阅（行未展开）时为 no-op，展开订阅即拉已覆盖
    },
    [],
  );

  // 展开行的 store 订阅管理：按 subagentOpen diff 出新增/收起路径，稳定订阅
  // 不因无关行开合而重建。live:false（快照订阅）——历史会话行只在展开/重试时
  // 拉取、不驱动 5s tick；当前会话行的实时刷新由 App/分工面板的 live 订阅
  // 共享同一路径轮询器提供，请求零重复。
  const subsRef = useRef<Map<string, () => void>>(new Map());
  useEffect(() => {
    const want = new Set<string>();
    for (const [path, open] of Object.entries(subagentOpen)) {
      if (open) want.add(path);
    }
    for (const [path, off] of subsRef.current) {
      if (!want.has(path)) {
        off();
        subsRef.current.delete(path);
      }
    }
    for (const path of want) {
      if (subsRef.current.has(path)) continue;
      subsRef.current.set(
        path,
        subscribeSubagentRuns(
          path,
          (runs, meta) => {
            setSubagents((prev) => ({ ...prev, [path]: { status: meta.status, runs } }));
          },
          { live: false },
        ),
      );
    }
  }, [subagentOpen]);
  useEffect(() => {
    const subs = subsRef.current;
    return () => {
      for (const off of subs.values()) off();
      subs.clear();
    };
  }, []);

  useEffect(() => { setLocalQuery(searchQuery); }, [searchQuery]);
  useEffect(() => {
    const timer = setTimeout(() => onSearchChange(localQuery), 200);
    return () => clearTimeout(timer);
  }, [localQuery, onSearchChange]);

  // 高频搜索防抖：输入框值即时更新（localQuery），过滤/搜索消费防抖后的值
  const debouncedQuery = useDebouncedValue(localQuery, 250);

  const isProjectOpen = (g: ProjectGroup): boolean =>
    g.path in projectsOpen ? projectsOpen[g.path] : g.current;

  const toggleProject = (path: string, defaultOpen: boolean) => {
    setProjectsOpen((prev) => {
      const next = { ...prev, [path]: !(path in prev ? prev[path] : defaultOpen) };
      try { localStorage.setItem(PROJECTS_OPEN_KEY, JSON.stringify(next)); } catch { /* ignore */ }
      return next;
    });
  };

  const currentSession = useMemo(() => {
    for (const g of projectGroups) {
      if (g.current) return g.sessions.find((s) => s.current);
    }
    return undefined;
  }, [projectGroups]);
  const currentProject = useMemo(
    () => projectGroups.find((g) => g.current),
    [projectGroups],
  );

  const searching = debouncedQuery.trim().length > 0;
  const filteredGroups = useMemo(
    () => filterProjectGroups(projectGroups, debouncedQuery),
    [projectGroups, debouncedQuery],
  );

  // 虚拟滚动：把当前可见的「分组头/空态/会话/显示更多/归档头/归档项」拍平成等高行
  const rows = useMemo<SessionRowItem[]>(() => {
    const out: SessionRowItem[] = [];
    for (const g of filteredGroups) {
      const open = searching || (g.path in projectsOpen ? projectsOpen[g.path] : g.current);
      out.push({ key: `g:${g.path}`, kind: "group", g, open });
      if (!open) continue;
      const sessions = g.sessions;
      const visible = sessions.slice(0, revealed[g.path] ? sessions.length : PER_PROJECT_PAGE);
      if (visible.length === 0) {
        out.push({ key: `empty:${g.path}`, kind: "empty", g });
      } else {
        for (const s of visible) {
          out.push({ key: `s:${s.path}`, kind: "session", s, g });
          if (subagentOpen[s.path]) {
            const st = subagents[s.path];
            if (!st || st.status === "loading") {
              out.push({ key: `sn:${s.path}:loading`, kind: "subagentNote", s, g, note: "loading" });
            } else if (st.status === "error") {
              out.push({ key: `sn:${s.path}:error`, kind: "subagentNote", s, g, note: "error" });
            } else if (st.runs.length === 0) {
              out.push({ key: `sn:${s.path}:empty`, kind: "subagentNote", s, g, note: "empty" });
            } else {
              for (const r of st.runs) {
                out.push({ key: `sa:${s.path}:${r.ref}`, kind: "subagent", s, g, r });
              }
            }
          }
        }
        if (sessions.length > visible.length) {
          out.push({ key: `more:${g.path}`, kind: "showMore", g, hidden: sessions.length - visible.length });
        }
      }
      if (g.archived.length > 0) {
        out.push({ key: `ah:${g.path}`, kind: "archHeader", g, open: !!archivedOpen[g.path] });
        if (archivedOpen[g.path]) {
          for (const s of g.archived) out.push({ key: `a:${s.path}`, kind: "archived", s, g });
        }
      }
    }
    return out;
  }, [filteredGroups, searching, projectsOpen, revealed, archivedOpen, subagentOpen, subagents]);

  // 会话列表容器高度：真实浏览器由 ResizeObserver 测量，jsdom 无布局时走兜底高度
  const sessionListRef = useRef<HTMLDivElement | null>(null);
  const [sessionViewH, setSessionViewH] = useState(0);
  useLayoutEffect(() => {
    const el = sessionListRef.current;
    if (!el) return;
    const update = () => {
      const h = el.clientHeight;
      if (h > 0) setSessionViewH((prev) => (prev === h ? prev : h));
    };
    update();
    if (typeof ResizeObserver !== "undefined") {
      const ro = new ResizeObserver(update);
      ro.observe(el);
      return () => ro.disconnect();
    }
  }, []);

  const ui: SessionRowUI = {
    running,
    deleteConfirm,
    setDeleteConfirm,
    renameTarget,
    renameDraft,
    setRenameDraft,
    setRenameTarget,
    revealed,
    setRevealed,
    archivedOpen,
    setArchivedOpen,
    toggleProject,
    isProjectOpen,
    onResumeSessionInProject,
    onArchiveSession,
    onRestoreSession,
    onPinSession,
    onDeleteSession,
    onRenameSession,
    onOpenSubagentThread,
    subagentOpen,
    subagents,
    toggleSubagents,
    reloadSubagents,
  };

  return (
    <>
      <aside
        className={`office-sidebar flex flex-col min-w-0 pt-[50px] pb-2 border-r border-border-soft select-none overflow-hidden drag-region ${
          collapsed ? "items-center px-2" : "px-2.5"
        }`}
        data-collapsed={collapsed ? "true" : "false"}
        data-narrow={sidebarWidth < 250 ? "true" : "false"}
        style={{ background: "var(--v3-rail-bg, var(--md-sys-color-shell-sidebar, var(--gaea-glass-bg, var(--md-sys-color-surface))))" }}
        aria-label="gaea navigation"
      >
        {/* 工作区身份：产品、当前项目与侧栏开关 */}
        <div className={`office-sidebar__brand flex items-center gap-2 ${collapsed ? "flex-col gap-1.5" : ""}`}>
          <span className="office-sidebar__brand-mark" aria-hidden>
            <img src={logoSvg} alt="" className="dark:hidden" />
            <img src={logoLightSvg} alt="" className="hidden dark:block" />
          </span>
          {!collapsed && (
            <span className="office-sidebar__brand-copy">
              <span className="office-sidebar__brand-name">gaea</span>
              {currentProject && (
                <span className="office-sidebar__workspace-name" title={currentProject.path}>
                  {t("sidebar.projects")} · {currentProject.name}
                </span>
              )}
            </span>
          )}
          <button
            className={`office-sidebar__collapse inline-flex items-center justify-center border-0 bg-transparent text-fg-faint cursor-pointer transition-[color,background] duration-[var(--dur-fast)] hover:text-fg hover:bg-sidebar-hover no-drag ${
              collapsed ? "" : "ml-auto"
            }`}
            onClick={toggleSidebar}
            title={toggleTitle}
            aria-label={toggleTitle}
          >
            {collapsed ? <PanelLeftOpen size={15} /> : <PanelLeftClose size={15} />}
          </button>
        </div>

        {/* 新建会话（KUN 风命令行：展开态通栏胶囊，折叠态圆钮） */}
        {collapsed ? (
          <button
            className="office-sidebar__new office-sidebar__new--compact inline-flex items-center justify-center border-0 text-accent cursor-pointer transition-[color,background,transform] duration-[var(--dur-fast)] hover:bg-accent/22 active:scale-95 no-drag disabled:opacity-40 disabled:cursor-default"
            onClick={() => void newSessionAndReset()}
            disabled={running}
            title={running ? t("common.busyHint") : t("topbar.newSession")}
            aria-label={t("topbar.newSession")}
          >
            <Plus size={15} />
          </button>
        ) : (
          <button
            className="office-sidebar__new flex items-center w-full text-accent cursor-pointer transition-[color,background,transform] duration-[var(--dur-fast)] hover:bg-accent/18 active:scale-[0.98] no-drag disabled:opacity-40 disabled:cursor-default"
            onClick={() => void newSessionAndReset()}
            disabled={running}
            title={running ? t("common.busyHint") : t("topbar.newSession")}
          >
            <span className="office-sidebar__new-icon"><Plus size={14} /></span>
            <span className="office-sidebar__new-label">{t("topbar.newSession")}</span>
          </button>
        )}

        {/* 次级工作入口：保持两列紧凑，不与会话争夺视觉权重 */}
        {!collapsed && (
          <div className="office-sidebar__utilities">
            <button
              className="office-sidebar__utility no-drag"
              onClick={() => void onOpenKnowledge()}
              title={t("topbar.knowledge")}
            >
              <BookOpen size={15} className="shrink-0 text-fg-faint" />
              <span className="flex-1 min-w-0 truncate text-left">{t("topbar.knowledge")}</span>
            </button>
            <button
              className="office-sidebar__utility no-drag"
              onClick={() => onOpenCaps()}
              title={t("caps.title")}
            >
              <Blocks size={15} className="shrink-0 text-fg-faint" />
              <span className="flex-1 min-w-0 truncate text-left">{t("caps.title")}</span>
            </button>
          </div>
        )}

        {/* 折叠态当前会话指示 */}
        {collapsed && currentSession && (
          <button
            className="office-sidebar__current sidebar-current-chip flex items-center justify-center cursor-pointer transition-colors no-drag"
            title={sessionTitle(currentSession, "")}
            type="button"
          >
            {(currentSession.title || currentSession.preview || "?").charAt(0).toUpperCase()}
          </button>
        )}

        {/* 会话区（展开态，占主要空间） */}
        {!collapsed && (
          <section className="office-sidebar__sessions flex-1 min-h-0 flex flex-col">
            {/* 搜索框 */}
            <div className="office-sidebar__search relative">
              <Search size={13} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-fg-faint/70 pointer-events-none" />
              <input
                className="office-sidebar__search-input no-drag"
                placeholder={t("sidebar.search")}
                aria-label={t("sidebar.search")}
                value={localQuery}
                onChange={e => setLocalQuery(e.target.value)}
                onKeyDown={e => e.stopPropagation()}
              />
            </div>

            {/* 项目头（KUN 风小节标签） */}
            <div className="office-sidebar__section-head">
              <span className="office-sidebar__section-title">
                {t("sidebar.projects")}
              </span>
              <span className="office-sidebar__section-count">{filteredGroups.length}</span>
              <span className="office-sidebar__section-rule" aria-hidden />
              <button
                className="office-sidebar__section-action"
                onClick={() => void onOpenHistory()}
                disabled={running}
                title={running ? t("common.busyHint") : t("topbar.history")}
              >
                {t("sidebar.viewAll")}
              </button>
            </div>

            {/* 会话列表：虚拟滚动（仅渲染可见窗口 + overscan） */}
            <div className="office-sidebar__list min-h-0 flex-1 flex flex-col overflow-hidden" ref={sessionListRef}>
              {projectGroups.length === 0 ? (
                <div className="office-sidebar__empty">
                  <span className="office-sidebar__empty-icon" aria-hidden><MessageSquare size={18} /></span>
                  <strong>{t("sidebar.noRecentLong")}</strong>
                  <p>{t("sidebar.noRecentHint")}</p>
                </div>
              ) : filteredGroups.length === 0 ? (
                <div className="office-sidebar__empty office-sidebar__empty--compact">
                  <span className="office-sidebar__empty-icon" aria-hidden><Search size={16} /></span>
                  <strong>{t("sidebar.noMatch")}</strong>
                </div>
              ) : (
                <List
                  className="sidebar-session-scroll"
                  style={{ height: sessionViewH || SIDEBAR_VIEWPORT_FALLBACK }}
                  rowCount={rows.length}
                  rowHeight={SESSION_ROW_HEIGHT}
                  rowProps={{ rows, ui }}
                  rowKey={(index) => rows[index].key}
                  rowComponent={SessionRow}
                  overscanCount={8}
                />
              )}
            </div>
          </section>
        )}

        {/* 后台任务（仅展开态、有运行任务时显示；紧凑单行） */}
        {!collapsed && jobs.length > 0 && (
          <section className="office-sidebar__jobs shrink-0">
            <div className="office-sidebar__group-label">
              <span className="flex items-center gap-1.5">
                <Loader2 size={11} className="text-accent" />
                {t("status.jobsTitle")}
              </span>
              <span className="text-fg-faint/45">{jobs.length}</span>
            </div>
            <div className="flex flex-col gap-0.5 max-h-28 overflow-y-auto pr-0.5">
              {jobs.map((j) => (
                <div
                  key={j.id}
                  className="flex items-start gap-2 px-2 py-1 rounded-md bg-bg-soft/40"
                  title={j.id}
                >
                  <span
                    className={`mt-[5px] w-1.5 h-1.5 rounded-full shrink-0 ${
                      j.status === "running" ? "bg-accent animate-pulse" : "bg-ok"
                    }`}
                  />
                  <span className="min-w-0 flex-1">
                    <span className="block truncate text-fg-dim text-[12px] leading-snug">{j.label}</span>
                    <span className="block text-fg-faint text-[10px] font-mono">
                      {j.kind} · {relativeTime(j.startedAt)}
                    </span>
                  </span>
                </div>
              ))}
            </div>
          </section>
        )}

        {/* 折叠态底部导航图标 */}
        {collapsed && (
        <nav className="office-sidebar__collapsed-nav flex flex-col items-center mt-auto">
          <button
            className="office-sidebar__collapsed-action no-drag"
            onClick={() => void onOpenKnowledge()}
            title={t("topbar.knowledge")}
          >
            <BookOpen size={15} />
          </button>
          <button
            className="office-sidebar__collapsed-action no-drag"
            onClick={() => void onOpenCaps()}
            title={t("caps.title")}
          >
            <Blocks size={15} />
          </button>
        </nav>
        )}
      </aside>

      {/* Resizer handle */}
      <button
        className="sidebar-resizer"
        type="button"
        role="separator"
        aria-orientation="vertical"
        aria-label={t("sidebar.resize")}
        aria-valuemin={SIDEBAR_MIN_WIDTH}
        aria-valuemax={SIDEBAR_MAX_WIDTH}
        aria-valuenow={sidebarWidth}
        onPointerDown={startResize}
        onKeyDown={resizeWithKeyboard}
        onDoubleClick={() => onDoubleClickResize()}
        title={t("sidebar.resize")}
      />
    </>
  );
}
