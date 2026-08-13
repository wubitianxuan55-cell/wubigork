import type { KeyboardEvent, PointerEvent as ReactPointerEvent } from "react";
import { useMemo, useState, useEffect } from "react";
import { Modal } from "antd";
import {
  Plus, Brain, Blocks, BookOpen, MessageSquare, Search,
  PanelLeftClose, PanelLeftOpen, Loader2, FileText, ChevronDown, FolderGit2,
  Pin, Inbox, Rollback, Check,
} from "../icons";
import logoSvg from "../assets/logo.svg";
import logoLightSvg from "../assets/logo-light.svg";
import { useT } from "../lib/i18n";
import { sessionTitle } from "../lib/session";
import type { FactBaseView, JobView, ProjectGroup, SessionMeta } from "../lib/types";
import { app } from "../lib/bridge";
import { useToast } from "./Toast";
import FeatureModelBar from "../../components/FeatureModelBar";

export interface SidebarProps {
  collapsed: boolean;
  toggleSidebar: () => void;
  running: boolean;
  jobs: JobView[];
  factBase: FactBaseView;
  onClearFactBase: () => void;
  onPromoteFactBase: () => Promise<number>;
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
  onOpenHistory: () => void;
  onOpenMemory: () => void;
  onOpenCaps: () => void;
  onOpenKnowledge: () => void;
  startResize: (e: ReactPointerEvent<HTMLButtonElement>) => void;
  resizeWithKeyboard: (e: KeyboardEvent<HTMLButtonElement>) => void;
  onDoubleClickResize: () => void;
  sidebarWidth: number;
  SIDEBAR_MIN_WIDTH: number;
  SIDEBAR_MAX_WIDTH: number;
}

// Codex 风格：按时间分桶（今天 / 昨天 / 前 7 天 / 前 30 天 / 更早）
function sessionBucket(ms: number): string {
  const startOfDay = (d: Date) => new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();
  const days = Math.round((startOfDay(new Date()) - startOfDay(new Date(ms))) / 86_400_000);
  if (days <= 0) return "今天";
  if (days === 1) return "昨天";
  if (days <= 7) return "前 7 天";
  if (days <= 30) return "前 30 天";
  return "更早";
}

// Codex 风格：相对时间（刚刚 / X 分钟前 / X 小时前 / 昨天 / M-D）
function relativeTime(ms: number): string {
  const startOfDay = (d: Date) => new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();
  const diff = Date.now() - ms;
  const min = Math.floor(diff / 60_000);
  if (min < 1) return "刚刚";
  if (min < 60) return `${min} 分钟前`;
  const days = Math.round((startOfDay(new Date()) - startOfDay(new Date(ms))) / 86_400_000);
  if (days <= 0) {
    const h = Math.floor(min / 60);
    return h < 24 ? `${h} 小时前` : new Date(ms).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
  }
  if (days === 1) return "昨天";
  const d = new Date(ms);
  return `${d.getMonth() + 1}-${d.getDate()}`;
}

const FACT_OPEN_KEY = "gaea.sidebar.factBaseOpen";
const PROJECTS_OPEN_KEY = "gaea.sidebar.projectsOpen";
const PER_PROJECT_PAGE = 8;

export function Sidebar({
  collapsed,
  toggleSidebar,
  running,
  jobs,
  factBase,
  onClearFactBase,
  onPromoteFactBase,
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
  onOpenHistory,
  onOpenMemory,
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
  const toast = useToast();
  const toggleTitle = collapsed ? t("sidebar.expand") : t("sidebar.collapse");
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null);
  const [renameTarget, setRenameTarget] = useState<string | null>(null);
  const [renameDraft, setRenameDraft] = useState("");
  const [localQuery, setLocalQuery] = useState(searchQuery);
  const [exportTemplate, setExportTemplate] = useState<"通用" | "公文" | "报告" | "合同">("报告");
  const [factOpen, setFactOpen] = useState(() => {
    try { return localStorage.getItem(FACT_OPEN_KEY) !== "0"; } catch { return true; }
  });
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

  useEffect(() => { setLocalQuery(searchQuery); }, [searchQuery]);
  useEffect(() => {
    const timer = setTimeout(() => onSearchChange(localQuery), 200);
    return () => clearTimeout(timer);
  }, [localQuery]);

  const toggleFactOpen = () => {
    setFactOpen((o) => {
      try { localStorage.setItem(FACT_OPEN_KEY, o ? "0" : "1"); } catch { /* ignore */ }
      return !o;
    });
  };

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

  const renderSessionItem = (session: SessionMeta, projectPath: string) => (
    <div
      className={`group flex items-start gap-1 rounded-md py-[6px] pl-2 pr-1.5 mb-[1px] transition-colors duration-150 hover:bg-sidebar-hover ${
        session.current ? "bg-sidebar-active" : ""
      }`}
      key={session.path}
    >
      <button
        className="flex items-start gap-2 flex-1 min-w-0 bg-transparent border-0 text-inherit cursor-pointer py-0.5 text-left disabled:cursor-default"
        onClick={() => void onResumeSessionInProject(session.path, projectPath)}
        disabled={running || session.current}
        title={session.path}
      >
        <MessageSquare
          size={13}
          className={`shrink-0 mt-[3px] ${session.current ? "text-accent" : "text-fg-faint/60"}`}
        />
        <span className="flex min-w-0 flex-1 flex-col gap-0.5">
          {renameTarget === session.path ? (
            <input
              className="w-full bg-bg border border-accent rounded px-1 py-0 text-fg text-[12.5px] outline-none"
              value={renameDraft}
              onChange={e => setRenameDraft(e.target.value)}
              onKeyDown={e => {
                if (e.key === "Enter") { e.preventDefault(); void onRenameSession(session.path, renameDraft.trim() || sessionTitle(session, "")); setRenameTarget(null); }
                if (e.key === "Escape") { e.preventDefault(); setRenameTarget(null); }
              }}
              onBlur={() => { void onRenameSession(session.path, renameDraft.trim() || sessionTitle(session, "")); setRenameTarget(null); }}
              autoFocus
              onClick={e => e.stopPropagation()}
            />
          ) : (
            <>
              <span className="flex items-baseline gap-2 min-w-0">
                <span
                  className={`overflow-hidden text-ellipsis whitespace-nowrap text-[12.5px] leading-[1.35] font-medium cursor-text ${
                    session.current ? "text-accent" : "text-fg-dim"
                  }`}
                  onDoubleClick={e => {
                    if (session.current) return;
                    e.stopPropagation();
                    setRenameTarget(session.path);
                    setRenameDraft(sessionTitle(session, ""));
                  }}
                  title="双击重命名"
                >
                  {sessionTitle(session, t("history.emptySession"))}
                </span>
                {session.hasRequirement && (
                  session.requirementDone ? (
                    <Check size={10} className="shrink-0 text-ok" aria-label="任务已验收" />
                  ) : (
                    <span className="shrink-0 self-center w-1.5 h-1.5 rounded-full bg-accent" aria-label="任务进行中" />
                  )
                )}
                {session.pinned && (
                  <Pin size={10} className="shrink-0 text-accent/80" aria-label="已置顶" />
                )}
                <span className="shrink-0 ml-auto text-fg-faint/70 font-mono text-[10px] tabular-nums">
                  {session.current ? t("history.current") : relativeTime(session.modTime)}
                </span>
              </span>
              {!session.current && session.preview && session.preview !== sessionTitle(session, "") && (
                <span className="flex-1 min-w-0 overflow-hidden text-ellipsis whitespace-nowrap text-fg-faint/60 text-[11px] leading-snug">
                  {session.preview}
                </span>
              )}
            </>
          )}
        </span>
      </button>
      {!session.current && (
        deleteConfirm === session.path ? (
          <span className="flex items-center gap-1 shrink-0 mt-1">
            <button className="bg-transparent border-0 text-[10px] text-err cursor-pointer px-1 py-0.5 rounded hover:bg-err/10" onClick={e => { e.stopPropagation(); void onDeleteSession(session.path); setDeleteConfirm(null); }}>
              确认
            </button>
            <button className="bg-transparent border-0 text-[10px] text-fg-faint cursor-pointer px-1 py-0.5 rounded hover:bg-bg-soft" onClick={e => { e.stopPropagation(); setDeleteConfirm(null); }}>
              取消
            </button>
          </span>
        ) : (
          <span className="hidden group-hover:flex items-center gap-0.5 shrink-0 mt-0.5">
            <button
              className="flex items-center justify-center w-5 h-5 rounded-md bg-transparent border-0 text-fg-faint cursor-pointer hover:text-accent hover:bg-bg-soft transition-colors"
              title={session.pinned ? "取消置顶" : "置顶"}
              onClick={e => { e.stopPropagation(); onPinSession(session.path, !session.pinned); }}
            >
              <Pin size={12} className={session.pinned ? "text-accent" : ""} />
            </button>
            <button
              className="flex items-center justify-center w-5 h-5 rounded-md bg-transparent border-0 text-fg-faint cursor-pointer hover:text-fg hover:bg-bg-soft transition-colors"
              title="归档（可恢复）"
              onClick={e => { e.stopPropagation(); onArchiveSession(session.path); }}
            >
              <Inbox size={12} />
            </button>
            <button
              className="flex items-center justify-center w-5 h-5 rounded-md bg-transparent border-0 text-fg-faint text-[13px] cursor-pointer hover:text-err hover:bg-bg-soft transition-colors"
              title="删除"
              onClick={e => { e.stopPropagation(); setDeleteConfirm(session.path); }}
            >
              ×
            </button>
          </span>
        )
      )}
    </div>
  );

  // 项目分组渲染：搜索时展开所有命中的分组，否则按折叠状态渲染
  const renderProjectGroup = (g: ProjectGroup, forceOpen: boolean) => {
    const open = forceOpen || isProjectOpen(g);
    const sessions = g.sessions;
    const visible = sessions.slice(0, revealed[g.path] ? sessions.length : PER_PROJECT_PAGE);
    return (
      <div key={g.path} className="mb-0.5">
        <button
          className="flex items-center gap-1.5 w-full h-8 pl-1 pr-1 rounded-md bg-transparent border-0 cursor-pointer transition-colors duration-[var(--dur-fast)] hover:bg-sidebar-hover no-drag"
          onClick={() => toggleProject(g.path, g.current)}
          title={g.path}
        >
          <span
            className="shrink-0 text-fg-faint/60 transition-transform duration-[var(--dur-fast)]"
            style={{ transform: open ? "rotate(90deg)" : "none" }}
          >
            <ChevronDown size={12} />
          </span>
          <FolderGit2 size={14} className={`shrink-0 ${g.current ? "text-accent" : "text-fg-faint"}`} />
          <span className={`flex-1 min-w-0 truncate text-left text-[12.5px] font-medium ${g.current ? "text-fg" : "text-fg-dim"}`}>
            {g.name}
          </span>
          {g.current && (
            <span className="shrink-0 text-accent text-[10px] font-medium">当前</span>
          )}
          <span className="shrink-0 text-fg-faint/55 font-mono text-[10px]">{g.sessions.length}</span>
        </button>

        {open && (
          <div className="ml-[15px] border-l border-border-soft/70 pl-2">
            {visible.length === 0 ? (
              <div className="py-2 px-2 text-fg-faint text-[11px] leading-snug">
                {g.archived.length === 0 ? (g.current ? "还没有会话，点击上方「新建会话」开始" : "该项目暂无会话") : "没有活动会话"}
              </div>
            ) : (
              <>
                {visible.map((s) => renderSessionItem(s, g.path))}
                {sessions.length > visible.length && (
                  <button
                    className="w-full mt-0.5 py-1 text-fg-faint text-[11.5px] rounded-md bg-transparent cursor-pointer hover:text-fg hover:bg-sidebar-hover transition-colors"
                    onClick={() => setRevealed((prev) => ({ ...prev, [g.path]: true }))}
                    type="button"
                  >
                    显示更多（{sessions.length - visible.length}）
                  </button>
                )}
              </>
            )}

            {/* 已归档分组（Kun/Codex：归档而非删除，随时可恢复） */}
            {g.archived.length > 0 && (
              <div className="mt-1">
                <button
                  className="flex items-center gap-1.5 w-full h-7 pl-1 pr-1 rounded-md bg-transparent border-0 cursor-pointer text-fg-faint transition-colors duration-[var(--dur-fast)] hover:bg-sidebar-hover hover:text-fg no-drag"
                  onClick={() => setArchivedOpen((prev) => ({ ...prev, [g.path]: !prev[g.path] }))}
                  title="已归档会话（可恢复）"
                >
                  <span
                    className="shrink-0 text-fg-faint/60 transition-transform duration-[var(--dur-fast)]"
                    style={{ transform: archivedOpen[g.path] ? "rotate(90deg)" : "none" }}
                  >
                    <ChevronDown size={11} />
                  </span>
                  <Inbox size={12} className="shrink-0" />
                  <span className="text-[11.5px]">已归档</span>
                  <span className="ml-auto font-mono text-[10px] text-fg-faint/55">{g.archived.length}</span>
                </button>

                {archivedOpen[g.path] && (
                  <div className="flex flex-col gap-px pt-0.5">
                    {g.archived.map((s) => (
                      <div
                        key={s.path}
                        className="group flex items-center gap-1 rounded-md py-[5px] pl-2 pr-1 hover:bg-sidebar-hover transition-colors"
                      >
                        <button
                          className="flex items-center gap-2 flex-1 min-w-0 bg-transparent border-0 text-left cursor-pointer"
                          onClick={() => onRestoreSession(s.path, g.path)}
                          title="恢复并继续该会话"
                        >
                          <MessageSquare size={12} className="shrink-0 text-fg-faint/50" />
                          <span className="flex-1 min-w-0 truncate text-[12px] text-fg-faint">{sessionTitle(s, t("history.emptySession"))}</span>
                          {s.hasRequirement && (
                            s.requirementDone ? (
                              <Check size={10} className="shrink-0 text-ok" />
                            ) : (
                              <span className="shrink-0 w-1.5 h-1.5 rounded-full bg-accent" />
                            )
                          )}
                          <span className="shrink-0 text-fg-faint/50 font-mono text-[10px]">{relativeTime(s.modTime)}</span>
                        </button>
                        {deleteConfirm === s.path ? (
                          <span className="flex items-center gap-1 shrink-0">
                            <button
                              className="bg-transparent border-0 text-[10px] text-err cursor-pointer px-1 py-0.5 rounded hover:bg-err/10"
                              onClick={e => { e.stopPropagation(); void onDeleteSession(s.path); setDeleteConfirm(null); }}
                            >
                              确认
                            </button>
                            <button
                              className="bg-transparent border-0 text-[10px] text-fg-faint cursor-pointer px-1 py-0.5 rounded hover:bg-bg-soft"
                              onClick={e => { e.stopPropagation(); setDeleteConfirm(null); }}
                            >
                              取消
                            </button>
                          </span>
                        ) : (
                          <span className="hidden group-hover:flex items-center gap-0.5 shrink-0">
                            <button
                              className="flex items-center justify-center w-5 h-5 rounded-md bg-transparent border-0 text-fg-faint cursor-pointer hover:text-accent hover:bg-bg-soft transition-colors"
                              title="恢复"
                              onClick={() => onRestoreSession(s.path, g.path)}
                            >
                              <Rollback size={12} />
                            </button>
                            <button
                              className="flex items-center justify-center w-5 h-5 rounded-md bg-transparent border-0 text-fg-faint text-[13px] cursor-pointer hover:text-err hover:bg-bg-soft transition-colors"
                              title="永久删除"
                              onClick={e => { e.stopPropagation(); setDeleteConfirm(s.path); }}
                            >
                              ×
                            </button>
                          </span>
                        )}
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}
          </div>
        )}
      </div>
    );
  };

  const q = localQuery.trim().toLowerCase();
  const searching = q.length > 0;
  const filteredGroups = useMemo(() => {
    if (!searching) return projectGroups;
    const out: ProjectGroup[] = [];
    for (const g of projectGroups) {
      const hits = g.sessions.filter(
        (s) =>
          (s.title || s.preview || "").toLowerCase().includes(q) ||
          s.path.toLowerCase().includes(q),
      );
      if (hits.length) out.push({ ...g, sessions: hits });
    }
    return out;
  }, [projectGroups, q, searching]);

  return (
    <>
      <aside
        className={`flex flex-col min-w-0 pt-[50px] pb-2 border-r border-border-soft select-none overflow-hidden drag-region ${
          collapsed ? "items-center px-2" : "px-2.5"
        }`}
        style={{ background: "var(--ds-gradient-sidebar)" }}
        aria-label="gaea navigation"
      >
        {/* 品牌行：logo + 名称 + 折叠 */}
        <div className={`flex items-center gap-2 px-1.5 pb-2.5 ${collapsed ? "flex-col gap-1.5 px-0 pb-3" : ""}`}>
          <img src={logoSvg} alt="" className="w-5 h-5 rounded-md dark:hidden" />
          <img src={logoLightSvg} alt="" className="w-5 h-5 rounded-md hidden dark:block" />
          {!collapsed && <span className="text-[13px] font-semibold tracking-tight text-fg">gaea</span>}
          <button
            className={`inline-flex items-center justify-center w-7 h-7 border-0 rounded-md bg-transparent text-fg-faint cursor-pointer transition-[color,background] duration-[var(--dur-fast)] hover:text-fg hover:bg-sidebar-hover no-drag ${
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
            className="inline-flex items-center justify-center w-8 h-8 mb-3 border-0 rounded-full text-accent bg-accent/12 cursor-pointer transition-[color,background,transform] duration-[var(--dur-fast)] hover:bg-accent/22 active:scale-95 no-drag disabled:opacity-40 disabled:cursor-default"
            onClick={() => void newSessionAndReset()}
            disabled={running}
            title={running ? t("common.busyHint") : t("topbar.newSession")}
          >
            <Plus size={15} />
          </button>
        ) : (
          <button
            className="flex items-center gap-2.5 min-h-9 w-full px-3 mb-1.5 rounded-full bg-accent/10 text-accent border border-accent/15 cursor-pointer transition-[color,background,transform] duration-[var(--dur-fast)] hover:bg-accent/18 active:scale-[0.98] no-drag disabled:opacity-40 disabled:cursor-default"
            onClick={() => void newSessionAndReset()}
            disabled={running}
            title={running ? t("common.busyHint") : t("topbar.newSession")}
          >
            <Plus size={14} />
            <span className="text-[13px] font-medium">{t("topbar.newSession")}</span>
          </button>
        )}

        {/* KUN 风命令行：功能入口常驻会话区上方（展开态） */}
        {!collapsed && (
          <div className="flex flex-col gap-0.5 mb-2">
            <button
              className="flex items-center gap-2.5 min-h-9 w-full px-3 rounded-full border border-transparent text-fg-dim text-[13px] no-drag cursor-pointer transition-[color,background] duration-[var(--dur-fast)] hover:bg-sidebar-hover hover:text-fg active:scale-[0.985]"
              onClick={() => void onOpenMemory()}
              title={t("topbar.memory")}
            >
              <Brain size={15} className="shrink-0 text-fg-faint" />
              <span className="flex-1 min-w-0 truncate text-left">{t("topbar.memory")}</span>
            </button>
            <button
              className="flex items-center gap-2.5 min-h-9 w-full px-3 rounded-full border border-transparent text-fg-dim text-[13px] no-drag cursor-pointer transition-[color,background] duration-[var(--dur-fast)] hover:bg-sidebar-hover hover:text-fg active:scale-[0.985]"
              onClick={() => void onOpenKnowledge()}
              title={t("topbar.knowledge")}
            >
              <BookOpen size={15} className="shrink-0 text-fg-faint" />
              <span className="flex-1 min-w-0 truncate text-left">{t("topbar.knowledge")}</span>
            </button>
            <button
              className="flex items-center gap-2.5 min-h-9 w-full px-3 rounded-full border border-transparent text-fg-dim text-[13px] no-drag cursor-pointer transition-[color,background] duration-[var(--dur-fast)] hover:bg-sidebar-hover hover:text-fg active:scale-[0.985]"
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
            className="w-8 h-8 mb-3 rounded-lg bg-sidebar-active text-accent text-[12px] font-bold flex items-center justify-center cursor-pointer hover:bg-sidebar-hover transition-colors no-drag"
            title={sessionTitle(currentSession, "")}
            type="button"
          >
            {(currentSession.title || currentSession.preview || "?").charAt(0).toUpperCase()}
          </button>
        )}

        {/* 会话区（展开态，占主要空间） */}
        {!collapsed && (
          <section className="flex-1 min-h-0 flex flex-col">
            {/* 搜索框 */}
            <div className="relative mb-2">
              <Search size={13} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-fg-faint/70 pointer-events-none" />
              <input
                className="w-full h-8 pl-8 pr-2.5 bg-bg-soft/60 border border-border-soft rounded-lg text-fg text-xs placeholder:text-fg-faint/60 outline-none focus:border-accent/50 transition-colors no-drag"
                placeholder={t("sidebar.search")}
                value={localQuery}
                onChange={e => setLocalQuery(e.target.value)}
                onKeyDown={e => e.stopPropagation()}
              />
            </div>

            {/* 项目头（KUN 风小节标签） */}
            <div className="flex items-center gap-2 px-1.5 pb-1.5">
              <span className="flex-1 min-w-0 text-fg-faint text-[11px] font-medium tracking-[0.02em]">
                项目
              </span>
              <button
                className="shrink-0 border-0 rounded-md bg-transparent text-fg-faint/70 text-[11px] px-1.5 py-0.5 cursor-pointer transition-colors duration-[var(--dur-fast)] hover:text-fg hover:bg-sidebar-hover disabled:opacity-50 disabled:cursor-default"
                onClick={() => void onOpenHistory()}
                disabled={running}
                title={running ? t("common.busyHint") : t("topbar.history")}
              >
                {t("sidebar.viewAll")}
              </button>
            </div>

            <div className="min-h-0 overflow-y-auto pr-0.5 sidebar-session-scroll">
              {projectGroups.length === 0 ? (
                <div className="py-3 px-2.5 text-fg-faint text-xs">
                  还没有最近会话
                  <div className="mt-1 text-[11px] leading-snug opacity-80">打开一个项目开始办公，会话会按项目自动归类</div>
                </div>
              ) : filteredGroups.length === 0 ? (
                <div className="py-3 px-2.5 text-fg-faint text-xs">无匹配</div>
              ) : searching ? (
                filteredGroups.map((g) => renderProjectGroup(g, true))
              ) : (
                filteredGroups.map((g) => renderProjectGroup(g, false))
              )}
            </div>
          </section>
        )}

        {/* 后台任务（仅展开态、有运行任务时显示；紧凑单行） */}
        {!collapsed && jobs.length > 0 && (
          <section className="shrink-0 px-1 pt-2 pb-1.5 border-t border-border-soft">
            <div className="flex items-center gap-2 px-1.5 pb-1 text-fg-faint text-[11px] font-medium tracking-[0.02em]">
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

        {/* 事实底座：可折叠，默认展开；交付类任务沉淀的事实底座 */}
        {!collapsed && (
          <section className="shrink-0 px-1 pt-1.5 pb-1.5 border-t border-border-soft">
            <button
              className="flex items-center gap-2 w-full h-8 px-2 rounded-lg bg-transparent border-0 cursor-pointer transition-colors duration-[var(--dur-fast)] hover:bg-sidebar-hover no-drag"
              onClick={toggleFactOpen}
              title={factOpen ? "收起事实底座" : "展开事实底座"}
            >
              <FileText size={13} className={`shrink-0 ${factBase.count > 0 ? "text-accent" : "text-fg-faint"}`} />
              <span className="text-[12px] font-medium text-fg-dim">事实底座</span>
              {factBase.count > 0 && (
                <span className="text-fg-faint/60 font-mono text-[10px]">{factBase.count}</span>
              )}
              <span className="ml-auto text-fg-faint/60 transition-transform duration-[var(--dur-fast)]" style={{ transform: factOpen ? "rotate(180deg)" : "none" }}>
                <ChevronDown size={12} />
              </span>
            </button>

            {factOpen && (
              <div className="pt-0.5">
                {factBase.count === 0 ? (
                  <div className="px-2.5 pb-1 text-fg-faint text-[11px] leading-snug">
                    交付类任务会自动沉淀事实，docx/pptx/xlsx 基于同一底座生成
                  </div>
                ) : (
                  <>
                    <div className="flex flex-col gap-0.5 max-h-32 overflow-y-auto pr-0.5">
                      {factBase.facts.map((f) => (
                        <div
                          key={f.key}
                          className="px-2 py-1 rounded-md bg-bg-soft/40"
                          title={f.value}
                        >
                          <span className="block truncate text-fg-dim text-[12px] leading-snug">{f.key}</span>
                          <span className="block truncate text-fg-faint text-[11px] leading-snug">{f.value}</span>
                          {f.source ? (
                            <span className="block truncate text-fg-faint/60 text-[10px] font-mono">来源：{f.source}</span>
                          ) : null}
                        </div>
                      ))}
                    </div>
                    <div className="flex flex-col gap-1.5 px-1 pt-1.5">
                      <div className="flex items-center gap-1.5">
                        <select
                          className="h-6 rounded-md text-[11px] text-fg-dim bg-bg-soft/60 border border-border-soft outline-none cursor-pointer"
                          value={exportTemplate}
                          onChange={(e) => setExportTemplate(e.target.value as "通用" | "公文" | "报告" | "合同")}
                          title="选择交付模板（公文/报告/合同/通用）"
                        >
                          <option value="报告">报告</option>
                          <option value="公文">公文</option>
                          <option value="合同">合同</option>
                          <option value="通用">通用</option>
                        </select>
                        <button
                          className="flex-1 h-6 rounded-md text-[11px] text-accent bg-accent/10 border border-accent/25 cursor-pointer transition-[background,color] hover:bg-accent/20"
                          title="把当前事实底座一键导出为所选模板的 Word 报告（docx/pptx/xlsx 同管线，一稿多用）"
                          onClick={() => {
                            void app.ExportDeliverable({
                              markdown: factBase.markdown,
                              format: "docx",
                              title: "事实底座报告",
                              template: exportTemplate,
                              cover: exportTemplate === "报告" || exportTemplate === "通用",
                              toc: exportTemplate === "报告",
                            })
                              .then((r) => {
                                toast.show(`已导出 ${r.name}`, "info");
                                void app.RevealWorkspacePath(r.path).catch(() => {});
                              })
                              .catch((e) => toast.show(e?.message || "导出失败", "warn"));
                          }}
                        >
                          导出报告
                        </button>
                      </div>
                      <button
                        className="h-6 rounded-md text-[11px] text-accent bg-accent/10 border border-accent/25 cursor-pointer transition-[background,color] hover:bg-accent/20"
                        title="把当前会话事实写入长期记忆，后续对话自动加载"
                        onClick={() => {
                          void onPromoteFactBase().then((n) => {
                            toast.show(n > 0 ? `已沉淀 ${n} 条事实到长期记忆` : "暂无事实可沉淀", "info");
                          });
                        }}
                      >
                        沉淀为长期记忆
                      </button>
                      <div className="flex items-center gap-2">
                        <button
                          className="flex-1 h-6 rounded-md text-[11px] text-fg-dim bg-bg-soft/60 border border-border-soft cursor-pointer transition-[background,color] hover:text-fg hover:bg-sidebar-hover"
                          onClick={() => {
                            void navigator.clipboard?.writeText(factBase.markdown).then(
                              () => toast.show("事实底座 Markdown 已复制", "info"),
                              () => {},
                            );
                          }}
                        >
                          复制 Markdown
                        </button>
                        <button
                          className="flex-1 h-6 rounded-md text-[11px] text-fg-faint bg-transparent border border-border-soft cursor-pointer transition-[background,color] hover:text-warning hover:border-warning/50"
                          onClick={() => {
                            Modal.confirm({
                              title: "清空事实底座",
                              content: "确定清空当前会话的事实底座？",
                              okText: "清空",
                              okButtonProps: { danger: true },
                              cancelText: "取消",
                              onOk: () => onClearFactBase(),
                            });
                          }}
                        >
                          清空
                        </button>
                      </div>
                    </div>
                  </>
                )}
              </div>
            )}
          </section>
        )}

        {/* 绑定模型（左下角，折叠时隐藏） */}
        {!collapsed && (
          <section className="shrink-0 px-1 pt-1.5 pb-2 border-t border-border-soft">
            <div className="side-model-wrap">
              <FeatureModelBar feature="gaea" label="办公" />
            </div>
          </section>
        )}

        {/* 折叠态底部导航图标 */}
        {collapsed && (
        <nav className="flex flex-col gap-1 items-center w-full mt-auto !pt-2.5 !pb-3">
          <button
            className="flex items-center justify-center w-8 h-8 rounded-lg text-fg-dim no-drag cursor-pointer transition-[color,background] duration-[var(--dur-fast)] hover:text-fg hover:bg-sidebar-hover active:scale-[0.98]"
            onClick={() => void onOpenMemory()}
            title={t("topbar.memory")}
          >
            <Brain size={15} />
          </button>
          <button
            className="flex items-center justify-center w-8 h-8 rounded-lg text-fg-dim no-drag cursor-pointer transition-[color,background] duration-[var(--dur-fast)] hover:text-fg hover:bg-sidebar-hover active:scale-[0.98]"
            onClick={() => void onOpenKnowledge()}
            title={t("topbar.knowledge")}
          >
            <BookOpen size={15} />
          </button>
          <button
            className="flex items-center justify-center w-8 h-8 rounded-lg text-fg-dim no-drag cursor-pointer transition-[color,background] duration-[var(--dur-fast)] hover:text-fg hover:bg-sidebar-hover active:scale-[0.98]"
            onClick={() => onOpenCaps()}
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
