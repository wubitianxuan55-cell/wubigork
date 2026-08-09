import type { KeyboardEvent, PointerEvent as ReactPointerEvent } from "react";
import { useState, useEffect } from "react";
import {
  SquarePen, Brain, Blocks, BookOpen, MessageSquare,
  PanelLeftClose, PanelLeftOpen, Loader2, FileText,
} from "../icons";
import logoSvg from "../assets/logo.svg";
import logoLightSvg from "../assets/logo-light.svg";
import { useT } from "../lib/i18n";
import { sessionTitle } from "../lib/session";
import type { FactBaseView, JobView, SessionMeta } from "../lib/types";
import { app } from "../lib/bridge";
import { useToast } from "./Toast";

export interface SidebarProps {
  collapsed: boolean;
  toggleSidebar: () => void;
  running: boolean;
  jobs: JobView[];
  factBase: FactBaseView;
  onClearFactBase: () => void;
  onPromoteFactBase: () => Promise<number>;
  newSessionAndReset: () => void;
  sessions: SessionMeta[];
  searchQuery: string;
  onSearchChange: (q: string) => void;
  hasMore: boolean;
  onLoadMore: () => void;
  onResumeSession: (path: string) => void;
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

export function Sidebar({
  collapsed,
  toggleSidebar,
  running,
  jobs,
  factBase,
  onClearFactBase,
  onPromoteFactBase,
  newSessionAndReset,
  sessions,
  searchQuery,
  onSearchChange,
  hasMore,
  onLoadMore,
  onResumeSession,
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
  useEffect(() => { setLocalQuery(searchQuery); }, [searchQuery]);
  useEffect(() => {
    const timer = setTimeout(() => onSearchChange(localQuery), 200);
    return () => clearTimeout(timer);
  }, [localQuery]);

  return (
    <>
      <aside
        className={`flex flex-col min-w-0 pt-[50px] pb-3 border-r border-border-soft select-none overflow-hidden drag-region ${
          collapsed ? "items-center px-2" : "px-2.5"
        }`}
        style={{ background: "var(--ds-gradient-sidebar)" }}
        aria-label="gaea navigation"
      >
        <div className={`flex items-center gap-2.5 px-2 pb-3.5 text-fg text-[15px] font-semibold ${
          collapsed ? "flex-col gap-2 px-0 pb-3" : ""
        }`}>
          <img src={logoSvg} alt="" className="w-6 h-6 rounded-md dark:hidden" />
          <img src={logoLightSvg} alt="" className="w-6 h-6 rounded-md hidden dark:block" />
          {!collapsed && <span>gaea</span>}
          <button
            className={`inline-flex items-center justify-center w-7 h-7 border-0 rounded-md bg-transparent text-fg-faint cursor-pointer transition-[color,background] duration-[var(--dur-fast)] hover:text-fg hover:bg-sidebar-hover no-drag ${
              collapsed ? "ml-0" : "ml-auto"
            }`}
            onClick={toggleSidebar}
            title={toggleTitle}
            aria-label={toggleTitle}
          >
            {collapsed ? <PanelLeftOpen size={15} /> : <PanelLeftClose size={15} />}
          </button>
        </div>

        {/* New session button */}
        <button
          className={`w-full min-w-0 border-0 rounded-full bg-accent text-accent-fg font-semibold cursor-pointer transition-all duration-[var(--dur-fast)] hover:brightness-110 active:scale-[0.97] disabled:opacity-40 disabled:cursor-default flex items-center gap-2 h-9 px-3 mb-3 no-drag ${
            collapsed ? "justify-center w-9 h-9 !rounded-full !p-0 !gap-0" : ""
          }`}
          style={{ boxShadow: "var(--ds-shadow-accent-btn)" }}
          onClick={() => void newSessionAndReset()}
          disabled={running}
          title={running ? t("common.busyHint") : t("topbar.newSession")}
        >
          <SquarePen size={15} />
          {!collapsed && <span>{t("topbar.newSession")}</span>}
        </button>

        {/* Collapsed session indicator */}
        {collapsed && (() => {
          const cur = sessions.find(s => s.current);
          return cur ? (
            <button
              className="w-9 h-9 mb-3 rounded-full bg-sidebar-active text-accent text-[12px] font-bold flex items-center justify-center cursor-pointer hover:bg-sidebar-hover transition-colors no-drag"
              title={sessionTitle(cur, "")}
              type="button"
            >
              {(cur.title || cur.preview || "?").charAt(0).toUpperCase()}
            </button>
          ) : null;
        })()}

        {/* Sessions section (hidden when collapsed) */}
        {!collapsed && (
          <section className="flex-1 min-h-0 flex flex-col">
            <div className="flex items-center gap-2 px-1 pb-2 pl-2.5">
              <div className="flex-1 min-w-0 text-fg-faint font-mono text-[11px] uppercase tracking-wider">
                {t("sidebar.conversations")}
              </div>
              <button
                className="shrink-0 border-0 rounded-md bg-transparent text-fg-faint text-[11.5px] px-1.5 py-0.5 cursor-pointer transition-[color,background,transform] duration-[var(--dur-fast)] hover:text-fg hover:bg-sidebar-hover active:scale-[0.97] disabled:opacity-50 disabled:cursor-default disabled:hover:text-fg-faint disabled:hover:bg-transparent"
                onClick={() => void onOpenHistory()}
                disabled={running}
                title={running ? t("common.busyHint") : t("topbar.history")}
              >
                {t("sidebar.viewAll")}
              </button>
            </div>
            <input
              className="w-full bg-bg-soft border border-border-soft rounded-[5px] text-fg text-xs py-1 px-2 mb-2 outline-none focus:border-accent no-drag"
              placeholder={t("sidebar.search")}
              value={localQuery}
              onChange={e => setLocalQuery(e.target.value)}
              onKeyDown={e => e.stopPropagation()}
            />
            <div className="min-h-0 overflow-y-auto pr-0.5">
              {(() => {
                const q = localQuery.trim().toLowerCase();
                const filtered = q
                  ? sessions.filter((s: SessionMeta) =>
                      (s.title || s.preview || "").toLowerCase().includes(q) ||
                      s.path.toLowerCase().includes(q)
                    )
                  : sessions;
                if (sessions.length === 0)
                  return <div className="py-2 px-2.5 text-fg-faint text-xs">{t("sidebar.noRecent")}</div>;
                if (filtered.length === 0 && q)
                  return <div className="py-2 px-2.5 text-fg-faint text-xs">无匹配</div>;
                const renderItem = (session: SessionMeta) => (
                  <div
                    className={`flex items-start gap-1 py-1 pl-2.5 pr-1 mb-0.5 rounded-md hover:bg-sidebar-hover group ${
                      session.current
                        ? "bg-sidebar-active border-l-[3px] border-accent pl-[8px]"
                        : ""
                    }`}
                    key={session.path}
                  >
                    <button
                      className="flex items-start gap-2.5 flex-1 min-w-0 bg-transparent border-0 text-inherit cursor-pointer py-1 text-left disabled:cursor-default"
                      onClick={() => void onResumeSession(session.path)}
                      disabled={running || session.current}
                      title={session.path}
                    >
                      <MessageSquare
                        size={14}
                        className={`shrink-0 mt-0.5 ${session.current ? "text-accent" : "text-fg-faint"}`}
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
                          <span
                            className={`overflow-hidden text-ellipsis whitespace-nowrap text-fg-dim text-[12.5px] leading-[1.35] font-medium cursor-text ${
                              session.current ? "text-accent" : ""
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
                        )}
                        <span className="flex items-center gap-2 min-w-0">
                          <span className="text-fg-faint font-mono text-[10.5px] shrink-0">
                            {session.current ? t("history.current") : relativeTime(session.modTime)}
                          </span>
                          {!session.current && session.preview && (
                            <span className="flex-1 min-w-0 overflow-hidden text-ellipsis whitespace-nowrap text-fg-faint/60 text-[10.5px]">
                              {session.preview}
                            </span>
                          )}
                        </span>
                      </span>
                    </button>
                    {!session.current && (
                      deleteConfirm === session.path ? (
                        <span className="flex items-center gap-1 shrink-0">
                          <button className="bg-transparent border-0 text-[10px] text-err cursor-pointer px-1 py-0.5 rounded hover:bg-err/10" onClick={e => { e.stopPropagation(); void onDeleteSession(session.path); setDeleteConfirm(null); }}>
                            确认
                          </button>
                          <button className="bg-transparent border-0 text-[10px] text-fg-faint cursor-pointer px-1 py-0.5 rounded hover:bg-bg-soft" onClick={e => { e.stopPropagation(); setDeleteConfirm(null); }}>
                            取消
                          </button>
                        </span>
                      ) : (
                        <button
                          className="hidden group-hover:block bg-transparent border-0 text-fg-faint text-[15px] cursor-pointer px-1 py-0.5 rounded-[3px] mt-1 hover:text-err"
                          title="删除"
                          onClick={e => { e.stopPropagation(); setDeleteConfirm(session.path); }}
                        >
                          ×
                        </button>
                      )
                    )}
                  </div>
                );
                // 搜索时保持平铺；未搜索时按 Codex 风格日期分组
                if (q) return filtered.map(renderItem);
                const groups: { label: string; items: SessionMeta[] }[] = [];
                for (const s of filtered) {
                  const label = sessionBucket(s.modTime);
                  const last = groups[groups.length - 1];
                  if (last && last.label === label) last.items.push(s);
                  else groups.push({ label, items: [s] });
                }
                return groups.map((g) => (
                  <div key={g.label} className="mb-1.5">
                    <div className="flex items-center gap-1.5 px-2 pt-1.5 pb-1 text-fg-faint text-[10px] font-semibold uppercase tracking-wider">
                      <span className="w-1 h-1 rounded-full bg-fg-faint/30" />
                      {g.label}
                      <span className="text-fg-faint/40 font-mono font-normal normal-case tracking-normal">{g.items.length}</span>
                    </div>
                    {g.items.map(renderItem)}
                  </div>
                ));
              })()}
              {hasMore && !localQuery && (
                <button
                  className="w-full mt-1 py-1.5 text-fg-faint text-[11.5px] border border-border-soft rounded-md bg-transparent cursor-pointer hover:text-fg hover:bg-sidebar-hover transition-colors"
                  onClick={() => void onLoadMore()}
                  type="button"
                >
                  Show more...
                </button>
              )}
            </div>
          </section>
        )}

        {/* 后台任务（仅展开态显示运行中的任务） */}
        {!collapsed && jobs.length > 0 && (
          <section className="shrink-0 px-1 pt-2 pb-2 border-t border-border-soft">
            <div className="flex items-center gap-2 px-1 pl-2.5 pb-1.5 text-fg-faint font-mono text-[11px] uppercase tracking-wider">
              <span className="flex items-center gap-1.5">
                <Loader2 size={12} className="text-accent" />
                {t("status.jobsTitle")}
              </span>
              <span className="text-fg-faint/50">{jobs.length}</span>
            </div>
            <div className="flex flex-col gap-1 max-h-36 overflow-y-auto pr-0.5">
              {jobs.map((j) => (
                <div
                  key={j.id}
                  className="flex items-start gap-2 px-2.5 py-1.5 rounded-md bg-bg-soft/60 border border-border-soft"
                  title={j.id}
                >
                  <span
                    className={`mt-1.5 w-1.5 h-1.5 rounded-full shrink-0 ${
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

        {/* 事实底座：交付前沉淀的事实，docx/pptx/xlsx 基于同一底座生成 */}
        {!collapsed && (
          <section className="shrink-0 px-1 pt-2 pb-2 border-t border-border-soft">
            <div className="flex items-center gap-2 px-1 pl-2.5 pb-1.5 text-fg-faint font-mono text-[11px] uppercase tracking-wider">
              <span className="flex items-center gap-1.5">
                <FileText size={12} className="text-accent" />
                事实底座
              </span>
              {factBase.count > 0 && <span className="text-fg-faint/50">{factBase.count}</span>}
            </div>
            {factBase.count === 0 ? (
              <div className="px-2.5 pb-1 text-fg-faint text-[11px] leading-snug">
                交付类任务会自动沉淀事实，docx/pptx/xlsx 基于同一底座生成
              </div>
            ) : (
              <>
                <div className="flex flex-col gap-1 max-h-40 overflow-y-auto pr-0.5">
                  {factBase.facts.map((f) => (
                    <div
                      key={f.key}
                      className="px-2.5 py-1.5 rounded-md bg-bg-soft/60 border border-border-soft"
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
                      if (window.confirm("确定清空当前会话的事实底座？")) onClearFactBase();
                    }}
                  >
                    清空
                  </button>
                  </div>
                </div>
              </>
            )}
          </section>
        )}

        <nav
          className={`flex flex-col gap-0.5 shrink-0 pt-2.5 pb-2 border-t border-border-soft ${
            collapsed ? "items-center w-full !pt-0 !pb-3" : ""
          }`}>
          <button
            className={`flex items-center gap-2.5 h-8 px-2.5 rounded-md text-fg-faint text-[13px] no-drag transition-[color,background,transform] duration-[var(--dur-fast)] hover:text-fg hover:bg-sidebar-hover active:scale-[0.97] ${collapsed ? "justify-center w-10 !p-0 !gap-0" : ""}`}
            onClick={() => void onOpenMemory()}
            title={t("topbar.memory")}
          >
            <Brain size={15} />
            {!collapsed && <span>{t("topbar.memory")}</span>}
          </button>
          <button
            className={`flex items-center gap-2.5 h-8 px-2.5 rounded-md text-fg-faint text-[13px] no-drag transition-[color,background,transform] duration-[var(--dur-fast)] hover:text-fg hover:bg-sidebar-hover active:scale-[0.97] ${collapsed ? "justify-center w-10 !p-0 !gap-0" : ""}`}
            onClick={() => void onOpenKnowledge()}
            title={t("topbar.knowledge")}
          >
            <BookOpen size={15} />
            {!collapsed && <span>{t("topbar.knowledge")}</span>}
          </button>
          <button
            className={`flex items-center gap-2.5 h-8 px-2.5 rounded-md text-fg-faint text-[13px] no-drag transition-[color,background,transform] duration-[var(--dur-fast)] hover:text-fg hover:bg-sidebar-hover active:scale-[0.97] ${collapsed ? "justify-center w-10 !p-0 !gap-0" : ""}`}
            onClick={() => onOpenCaps()}
            title={t("caps.title")}
          >
            <Blocks size={15} />
            {!collapsed && <span>{t("caps.title")}</span>}
          </button>
        </nav>
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
