import type { Dispatch, KeyboardEvent, PointerEvent as ReactPointerEvent, SetStateAction } from "react";
import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import { List, type RowComponentProps } from "react-window";
import {Input, Modal, Select } from "antd";
import {
  Plus, Blocks, BookOpen, MessageSquare, Search,
  Loader2, ChevronDown,
  Pin, Inbox, Rollback, X, Clock, Trash2, SquarePen,
} from "../icons";

// 新建小说·成人等级白名单（与 Go wenxueAdultLevels 同源：存储值=展示值）。
const ADULT_LEVELS = ["全年龄", "轻度（暗示级）", "成人（限制级）", "深度成人（无限制）"] as const;

const ADULT_LEVEL_HINT: Record<string, string> = {
  "全年龄": "无成人内容；情感与冲突克制处理。",
  "轻度（暗示级）": "亲密与暴力以暗示、淡出处理，无直接描写。",
  "成人（限制级）": "可直接描写，保持文学化表达与叙事必要。",
  "深度成人（无限制）": "无任何安全底线；性/暴力/黑暗/禁忌题材全部正面直写，不回避、不淡化、不审查。随书注入云端模型破甲契约。",
};
import { useT } from "../lib/i18n";
import { PrivacySidebarSection } from "./PrivacySidebarSection";
import { WorkspaceSidebarToggleIcon } from "../../components/WorkspaceSidebarToggleIcon";
import { sessionTitle } from "../lib/session";
import { relativeTime } from "../lib/time";
import { filterProjectGroups } from "../lib/projectGroups";
import type { JobView, ProjectGroup, SessionMeta, SubagentRunView, WorkspaceView } from "../lib/types";
import { reloadSubagentRuns, subscribeSubagentRuns } from "../lib/subagentRunsStore";
import { useDebouncedValue } from "../hooks/useDebouncedValue";
import type { SubagentThreadStatus } from "./SubagentThread";

export interface SidebarProps {
  /** 首页隐身模式：只遮显示面，不影响真实路径、会话和项目数据。 */
  privacyMode?: boolean;
  collapsed: boolean;
  toggleSidebar: () => void;
  running: boolean;
  jobs: JobView[];
  newSessionAndReset: () => void;
  /** 左栏唯一的新小说入口：后端创建 tuwen/<name> 并自动切换。 */
  onCreateNovel: (name: string, infoJson: string) => Promise<WorkspaceView>;
  /** 切换左栏选中的小说工作空间。 */
  onSelectNovelWorkspace: (path: string) => Promise<string>;
  /** 删除一部小说及其完整工作空间。 */
  onDeleteNovel: (path: string) => Promise<void>;
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
  startResize: (e: ReactPointerEvent<HTMLButtonElement>) => void;
  resizeWithKeyboard: (e: KeyboardEvent<HTMLButtonElement>) => void;
  onDoubleClickResize: () => void;
  sidebarWidth: number;
  SIDEBAR_MIN_WIDTH: number;
  SIDEBAR_MAX_WIDTH: number;
}

const PROJECTS_OPEN_KEY = "gaea.sidebar.projectsOpen";
const PER_PROJECT_PAGE = 8;

/** 虚拟滚动固定行高（px）：小说父节点需要两行信息，会话行则保持舒展。 */
const SESSION_ROW_HEIGHT = 52;
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
  privacyMode: boolean;
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
  onSelectGroup: (g: ProjectGroup) => void;
  onNewSessionInGroup: (g: ProjectGroup) => void;
  onDeleteNovel: (g: ProjectGroup) => void;
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
        <div className={`office-novel-group ${row.g.current ? "is-current" : ""}`}>
          <button
            className="office-project-row no-drag"
            onClick={() => ui.onSelectGroup(row.g)}
            title={ui.privacyMode ? "私密作品" : row.g.path}
            aria-expanded={row.open}
          >
            <span
              className="shrink-0 text-fg-faint/60 transition-transform duration-[var(--dur-fast)]"
              style={{ transform: row.open ? "rotate(90deg)" : "none" }}
            >
              <ChevronDown size={12} />
            </span>
            <span className="office-novel-spine" aria-hidden />
            <span className="flex min-w-0 flex-1 flex-col gap-0.5 text-left">
              <span className={`truncate text-[12.5px] font-semibold ${row.g.current ? "text-fg" : "text-fg-dim"}`}>
                {row.g.name}
              </span>
              <span className="flex items-center gap-1.5 text-[9px] font-normal text-fg-faint">
                {row.g.current && <span className="text-accent">{t("sidebar.currentBadge")}</span>}
                <span>{row.g.sessions.length} 个会话</span>
              </span>
            </span>
          </button>
          <span className="office-novel-actions">
            <button
              type="button"
              className="office-novel-action no-drag"
              onClick={() => ui.onNewSessionInGroup(row.g)}
              disabled={ui.running}
              title={ui.running ? t("common.busyHint") : `在《${row.g.name}》中新建会话`}
              aria-label={`在《${row.g.name}》中新建会话`}
            >
              <Plus size={13} aria-hidden />
            </button>
            <button
              type="button"
              className="office-novel-action office-novel-action--danger no-drag"
              onClick={() => ui.onDeleteNovel(row.g)}
              disabled={ui.running}
              title={ui.running ? t("common.busyHint") : `删除《${row.g.name}》`}
              aria-label={`删除《${row.g.name}》`}
            >
              <Trash2 size={12} aria-hidden />
            </button>
          </span>
        </div>
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
            title={ui.privacyMode ? "私密会话" : s.path}
          >
            <MessageSquare
              size={13}
              className={`office-session-icon ${s.current ? "text-accent" : "text-fg-faint/60"}`}
            />
            <span className="flex min-w-0 flex-1 flex-col gap-0.5">
              {ui.renameTarget === s.path && !ui.privacyMode ? (
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
                        if (s.current || ui.privacyMode) return;
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
                {ui.privacyMode ? "私密子任务" : (r.task || t("sidebar.subagentsUntitled"))}
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
  privacyMode = false,
  collapsed,
  toggleSidebar,
  running,
  jobs,
  newSessionAndReset,
  onCreateNovel,
  onSelectNovelWorkspace,
  onDeleteNovel,
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
  const [novelDialogOpen, setNovelDialogOpen] = useState(false);
  const [novelName, setNovelName] = useState("");
  const [novelGenre, setNovelGenre] = useState("");
  const [novelIntro, setNovelIntro] = useState("");
  const [novelTargetWords, setNovelTargetWords] = useState("");
  const [novelAdultLevel, setNovelAdultLevel] = useState("全年龄");
  const [novelBusy, setNovelBusy] = useState(false);
  const [novelError, setNovelError] = useState("");
  const [deleteNovelTarget, setDeleteNovelTarget] = useState<ProjectGroup | null>(null);
  const [deleteNovelBusy, setDeleteNovelBusy] = useState(false);
  const [switchingWorkspace, setSwitchingWorkspace] = useState<string | null>(null);
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

  // 隐私模式只替换渲染快照中的可见文字；path/会话动作仍使用真实对象。
  const displayedProjectGroups = useMemo(() => {
    if (!privacyMode) return projectGroups;
    return projectGroups.map((group, groupIndex) => ({
      ...group,
      name: group.current
        ? "私密作品"
        : `私密作品 ${String(groupIndex + 1).padStart(2, "0")}`,
      sessions: group.sessions.map((session, sessionIndex) => ({
        ...session,
        title: "",
        preview: `私密会话 ${sessionIndex + 1}`,
      })),
      archived: group.archived.map((session, sessionIndex) => ({
        ...session,
        title: "",
        preview: `归档会话 ${sessionIndex + 1}`,
      })),
    }));
  }, [privacyMode, projectGroups]);

  const currentSession = useMemo(() => {
    for (const g of displayedProjectGroups) {
      if (g.current) return g.sessions.find((s) => s.current);
    }
    return undefined;
  }, [displayedProjectGroups]);
  const currentProject = useMemo(
    () => displayedProjectGroups.find((g) => g.current),
    [displayedProjectGroups],
  );
  const searching = debouncedQuery.trim().length > 0;
  const filteredGroups = useMemo(
    () => filterProjectGroups(displayedProjectGroups, debouncedQuery),
    [displayedProjectGroups, debouncedQuery],
  );

  const openNovelDialog = useCallback(() => {
    if (running) return;
    setNovelName("");
    setNovelError("");
    setNovelDialogOpen(true);
  }, [running]);

  useEffect(() => {
    const openRequestedNovelDialog = () => openNovelDialog();
    window.addEventListener("wenxue:new-novel-requested", openRequestedNovelDialog);
    return () => window.removeEventListener("wenxue:new-novel-requested", openRequestedNovelDialog);
  }, [openNovelDialog]);

  const createNovel = async () => {
    const name = novelName.trim();
    if (!name || novelBusy) return;
    setNovelBusy(true);
    setNovelError("");
    try {
      const info = {
        genre: novelGenre.trim(),
        intro: novelIntro.trim(),
        targetWords: novelTargetWords.trim(),
        adultLevel: novelAdultLevel,
      };
      await onCreateNovel(name, JSON.stringify(info));
      setNovelDialogOpen(false);
      setNovelName("");
      setNovelGenre("");
      setNovelIntro("");
      setNovelTargetWords("");
      setNovelAdultLevel("全年龄");
    } catch (err) {
      setNovelError(err instanceof Error ? err.message : String(err));
    } finally {
      setNovelBusy(false);
    }
  };

  const deleteNovel = async () => {
    const target = deleteNovelTarget;
    if (!target || deleteNovelBusy) return;
    setDeleteNovelBusy(true);
    setNovelError("");
    try {
      await onDeleteNovel(target.path);
      setDeleteNovelTarget(null);
    } catch (err) {
      setNovelError(err instanceof Error ? err.message : String(err));
    } finally {
      setDeleteNovelBusy(false);
    }
  };

  const selectNovelWorkspace = async (path: string) => {
    if (running || switchingWorkspace || currentProject?.path === path) return "";
    setSwitchingWorkspace(path);
    setNovelError("");
    try {
      return await onSelectNovelWorkspace(path);
    } catch (err) {
      setNovelError(err instanceof Error ? err.message : String(err));
      return "";
    } finally {
      setSwitchingWorkspace(null);
    }
  };

  const selectGroup = async (g: ProjectGroup) => {
    if (running) return;
    if (!g.current) {
      const picked = await selectNovelWorkspace(g.path);
      if (!picked) return;
    }
    toggleProject(g.path, g.current);
  };

  const newSessionInGroup = async (g: ProjectGroup) => {
    if (running) return;
    if (!g.current) {
      const picked = await selectNovelWorkspace(g.path);
      if (!picked) return;
    }
    await newSessionAndReset();
  };

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
    privacyMode,
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
    onSelectGroup: (g) => { void selectGroup(g); },
    onNewSessionInGroup: (g) => { void newSessionInGroup(g); },
    onDeleteNovel: (g) => {
      setNovelError("");
      setDeleteNovelTarget(g);
    },
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
        <div className={`office-sidebar__brand flex items-center gap-2 ${collapsed ? "justify-center" : ""}`}>
          {!collapsed && (
            <span className="office-sidebar__brand-copy">
              <span className="office-sidebar__brand-name">
                gaea
                <span className="office-sidebar__brand-mode">文学</span>
              </span>
              {currentProject && (
                <span className="office-sidebar__workspace-name" title={privacyMode ? "私密作品" : currentProject.path}>
                  {currentProject.name}
                </span>
              )}
            </span>
          )}
          <button
            className={`office-sidebar__collapse inline-flex items-center justify-center border-0 bg-transparent text-fg-faint cursor-pointer transition-[color,background] duration-[var(--dur-fast)] hover:text-fg hover:bg-sidebar-hover no-drag ${
              collapsed ? "mx-auto" : "ml-auto"
            }`}
            onClick={toggleSidebar}
            title={toggleTitle}
            aria-label={toggleTitle}
            aria-expanded={!collapsed}
            data-testid="global-sidebar-toggle"
          >
            <WorkspaceSidebarToggleIcon collapsed={collapsed} />
            {!collapsed && (
              <span className="office-sidebar__collapse-label">{t("common.collapse")}</span>
            )}
          </button>
        </div>

        {/* 新建会话（KUN 风命令行：展开态通栏胶囊，折叠态圆钮） */}
        {collapsed ? (
          <div className="office-sidebar__collapsed-actions">
            <button
              className="office-sidebar__new office-sidebar__new--compact inline-flex items-center justify-center border-0 text-accent cursor-pointer transition-[color,background,transform] duration-[var(--dur-fast)] hover:bg-accent/22 active:scale-95 no-drag disabled:opacity-40 disabled:cursor-default"
              onClick={openNovelDialog}
              disabled={running}
              title={running ? t("common.busyHint") : "新建小说"}
              aria-label="新建小说"
            >
              <Plus size={16} />
            </button>
            <button
              className="office-sidebar__collapsed-action office-sidebar__collapsed-session no-drag"
              onClick={() => void newSessionAndReset()}
              disabled={running}
              title={running ? t("common.busyHint") : t("topbar.newSession")}
              aria-label={t("topbar.newSession")}
              data-testid="sidebar-new-session-collapsed"
            >
              <SquarePen size={16} aria-hidden />
            </button>
            {currentSession && (() => {
              const currentTitle = privacyMode ? "私密会话" : sessionTitle(currentSession, "当前会话");
              return (
                <button
                  className="office-sidebar__current sidebar-current-chip flex items-center justify-center cursor-pointer transition-colors no-drag"
                  title={`${currentTitle} · ${t("sidebar.expand")}`}
                  aria-label={`${currentTitle} · ${t("sidebar.expand")}`}
                  onClick={toggleSidebar}
                  type="button"
                  data-testid="sidebar-current-session-collapsed"
                >
                  <MessageSquare size={15} aria-hidden />
                  <span className="office-sidebar__current-dot" aria-hidden />
                </button>
              );
            })()}
          </div>
        ) : (
          <button
            className="office-sidebar__new flex items-center w-full text-accent cursor-pointer transition-[color,background,transform] duration-[var(--dur-fast)] hover:bg-accent/18 active:scale-[0.98] no-drag disabled:opacity-40 disabled:cursor-default"
            onClick={openNovelDialog}
            disabled={running}
            title={running ? t("common.busyHint") : "在 C:\\AI\\tuwen 下新建小说工作空间"}
            aria-label="新建小说"
          >
            <span className="office-sidebar__new-icon"><Plus size={14} /></span>
            <span className="office-sidebar__new-label">新建小说</span>
          </button>
        )}

        {/* 小说是父级，会话直接挂在对应小说下面。 */}
        {!collapsed && (
          <section className="office-sidebar__library flex-1 min-h-0 flex flex-col" aria-label="小说与会话">
            <div className="office-sidebar__section-head">
              <span className="office-sidebar__section-title">我的小说</span>
              <span className="office-sidebar__section-count">{projectGroups.length}</span>
              <button
                className="office-sidebar__section-action office-sidebar__section-action--icon"
                onClick={() => void onOpenHistory()}
                disabled={running}
                title={running ? t("common.busyHint") : "全部历史"}
                aria-label="全部历史"
              >
                <Clock size={12} aria-hidden />
                <span>历史</span>
              </button>
              <span className="office-sidebar__section-rule" aria-hidden />
              <span className="office-sidebar__root-label" title="C:\\AI\\tuwen">tuwen</span>
            </div>

            <div className="office-sidebar__search relative">
              <Search size={13} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-fg-faint/70 pointer-events-none" />
              <input
                className="office-sidebar__search-input no-drag"
                placeholder="搜索小说或会话"
                aria-label="搜索小说或会话"
                value={localQuery}
                onChange={e => setLocalQuery(e.target.value)}
                onKeyDown={e => e.stopPropagation()}
              />
            </div>

            <div className="office-sidebar__list min-h-0 flex-1 flex flex-col overflow-hidden" ref={sessionListRef}>
              {projectGroups.length === 0 ? (
                <div className="office-sidebar__empty">
                  <span className="office-sidebar__empty-icon" aria-hidden><BookOpen size={18} /></span>
                  <strong>从第一部小说开始</strong>
                  <p>每部小说都会拥有自己的会话、稿件和资料，不会混在一起。</p>
                  <button type="button" className="office-sidebar__empty-action" onClick={openNovelDialog} disabled={running}>
                    <Plus size={13} aria-hidden />
                    新建小说
                  </button>
                </div>
              ) : filteredGroups.length === 0 ? (
                <div className="office-sidebar__empty office-sidebar__empty--compact">
                  <span className="office-sidebar__empty-icon" aria-hidden><Search size={16} /></span>
                  <strong>没有匹配结果</strong>
                  <p>换个小说名或会话关键词再试。</p>
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
            {novelError && !novelDialogOpen && !deleteNovelTarget && (
              <div className="office-sidebar__workspace-error" role="alert">{novelError}</div>
            )}
          </section>
        )}

        {!collapsed && (
          <div className="office-sidebar__utility-dock office-sidebar__utility-dock--single shrink-0">
            <button
              className="office-sidebar__utility no-drag"
              onClick={onOpenCaps}
              title={t("caps.title")}
            >
              <Blocks size={14} className="shrink-0" />
              <span>{t("caps.title")}</span>
            </button>
          </div>
        )}

        {/* 私密保护区（2026-11-23 无限制档小说）：自含 hook 零穿线，锁定态
            占位行/解锁、解锁态逐书隐藏开关/回锁/PIN/遮罩开关。 */}
        {!collapsed && <PrivacySidebarSection />}

        <Modal
          open={novelDialogOpen}
          title="新建小说"
          okText="创建并进入"
          cancelText="取消"
          confirmLoading={novelBusy}
          okButtonProps={{ disabled: novelName.trim() === "" }}
          onCancel={() => { if (!novelBusy) setNovelDialogOpen(false); }}
          onOk={() => void createNovel()}
          destroyOnHidden
          rootClassName="wx-root"
        >
          <label className="block text-[12px] font-medium text-fg" htmlFor="wenxue-novel-name">
            小说名字
          </label>
          <Input
            id="wenxue-novel-name"
            autoFocus
            value={novelName}
            onChange={(e) => setNovelName(e.target.value)}
            onPressEnter={() => void createNovel()}
            placeholder="例如：长夜灯火"
            maxLength={64}
            spellCheck={false}
            aria-label="小说名字"
          />
                    <div className="mt-3 grid grid-cols-2 gap-2">
            <div>
              <label className="block text-[12px] font-medium text-fg" htmlFor="wenxue-novel-genre">题材</label>
              <Input id="wenxue-novel-genre" value={novelGenre} onChange={(e) => setNovelGenre(e.target.value)} placeholder="如：都市悬疑" aria-label="题材" />
            </div>
            <div>
              <label className="block text-[12px] font-medium text-fg" htmlFor="wenxue-novel-words">目标字数</label>
              <Input id="wenxue-novel-words" value={novelTargetWords} onChange={(e) => setNovelTargetWords(e.target.value)} placeholder="如：80 万" aria-label="目标字数" />
            </div>
          </div>
          <label className="mt-3 block text-[12px] font-medium text-fg" htmlFor="wenxue-novel-intro">简介</label>
          <Input.TextArea id="wenxue-novel-intro" value={novelIntro} onChange={(e) => setNovelIntro(e.target.value)} placeholder="一句话简介（注入每个会话，agent 开场即知这本书写什么）" autoSize={{ minRows: 2, maxRows: 4 }} maxLength={500} aria-label="简介" />
          <label className="mt-3 block text-[12px] font-medium text-fg" htmlFor="wenxue-novel-adult">
            成人等级<span className="text-fg-faint">（注入每个会话，agent 据此校准尺度）</span>
          </label>
          <Select id="wenxue-novel-adult" value={novelAdultLevel} onChange={(v) => setNovelAdultLevel(v)} options={ADULT_LEVELS.map((lv) => ({ value: lv, label: lv }))} aria-label="成人等级" />
          <div className="mt-1 text-[10.5px] leading-relaxed text-fg-faint">{ADULT_LEVEL_HINT[novelAdultLevel]}</div>
          <div className="mt-2 text-[11px] leading-relaxed text-fg-faint">
            将按小说名字在 <span className="font-mono text-fg-dim">C:\AI	uwen</span> 下创建独立文件夹，
            并自动切换为当前工作空间。关键信息落 <span className="font-mono text-fg-dim">书籍信息.md</span> 并自动固定注入。
          </div>
          {novelError && (
            <div className="mt-2 text-[11px] text-err" role="alert">{novelError}</div>
          )}
        </Modal>

        <Modal
          open={deleteNovelTarget !== null}
          title="删除小说"
          okText="永久删除"
          cancelText="取消"
          confirmLoading={deleteNovelBusy}
          okButtonProps={{ danger: true }}
          onCancel={() => { if (!deleteNovelBusy) setDeleteNovelTarget(null); }}
          onOk={() => void deleteNovel()}
          destroyOnHidden
          rootClassName="wx-root"
        >
          <p className="m-0 text-[12px] leading-relaxed text-fg">
            将删除《{deleteNovelTarget?.name}》及其中的全部会话、稿件和资料。
          </p>
          <p className="mt-2 mb-0 text-[11px] leading-relaxed text-fg-faint">
            删除后无法恢复。当前小说被删除时会自动回到小说根工作空间。
          </p>
          <div className="mt-3 break-all rounded-md border border-border-soft bg-bg-soft px-2 py-1.5 font-mono text-[10px] text-fg-faint">
            {deleteNovelTarget?.path}
          </div>
          {novelError && (
            <div className="mt-2 text-[11px] text-err" role="alert">{novelError}</div>
          )}
        </Modal>

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
            onClick={() => void onOpenCaps()}
            title={t("caps.title")}
            aria-label={t("caps.title")}
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
