import { useCallback, useEffect, useState } from "react";
import { Database, ExternalLink, Eye, FileText, MessageSquare, RefreshCw, Trash2, X } from "../../icons";
import { app } from "../../lib/bridge";
import { usePreviewStore } from "../../lib/store";
import { DOMAIN_COLORS } from "../../lib/domainColors";
import type { DatabaseOverview, UploadFileRow } from "../../lib/types";
import type { ProjectGroup } from "../../lib/types/session";

/**
 * DatabaseLibrary 记忆中枢「数据库」库（v4.479）：管理本设备保存的聊天记录
 * 与上传的文件。聊天记录 = 各工作区 .gaea/sessions/ 的 .jsonl 会话档（含归档）；
 * 上传的文件 = 各工作区 .gaea/uploads/（attach-* 上传附件、paste-* 粘贴图）。
 * 作用域 = 当前 + 最近工作区（GaeaDatabaseOverview 与侧边栏项目视图同口径）。
 * 删除均两段确认（首点变确认态）；预览走全局 FilePreviewModal。
 */

function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  if (n < 1024 * 1024 * 1024) return `${(n / 1024 / 1024).toFixed(1)} MB`;
  return `${(n / 1024 / 1024 / 1024).toFixed(2)} GB`;
}

function formatTime(ms: number): string {
  if (!ms) return "—";
  const d = new Date(ms);
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")} ${String(d.getHours()).padStart(2, "0")}:${String(d.getMinutes()).padStart(2, "0")}`;
}

export function DatabaseLibrary() {
  const [overview, setOverview] = useState<DatabaseOverview | null>(null);
  const [groups, setGroups] = useState<ProjectGroup[]>([]);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  // 上传文件按项目懒加载（details 首次展开时拉取）；confirmId 两段删除确认。
  const [uploads, setUploads] = useState<Record<string, UploadFileRow[]>>({});
  const [confirmId, setConfirmId] = useState<string | null>(null);

  const load = useCallback(() => {
    setLoading(true);
    setError("");
    setConfirmId(null); // 刷新后未确认的删除态一并复位
    Promise.all([app.DatabaseOverview(), app.ListProjectSessions()])
      .then(([ov, gs]) => {
        setOverview(ov ?? null);
        setGroups(gs ?? []);
        setUploads({});
      })
      .catch(() => setError("数据库存档读取失败（绑定不可达或磁盘错误）"))
      .finally(() => setLoading(false));
  }, []);
  useEffect(() => { load(); }, [load]);

  const loadUploads = useCallback((root: string) => {
    if (uploads[root]) return;
    setUploads((prev) => ({ ...prev, [root]: [] })); // 先占位防重复拉取
    app.UploadsList(root)
      .then((rows) => setUploads((prev) => ({ ...prev, [root]: rows ?? [] })))
      .catch(() => setUploads((prev) => {
        // 失败清占位键：下次展开可重试（占位 [] 永久短路重试是懒加载暗坑）
        const next = { ...prev };
        delete next[root];
        return next;
      }));
  }, [uploads]);

  const deleteSession = useCallback((path: string) => {
    app.DeleteSession(path)
      .then(() => load())
      .catch(() => setError("会话删除失败（当前会话不可删除或路径非法）"));
    setConfirmId(null);
  }, [load]);

  const deleteUpload = useCallback((path: string) => {
    app.DeleteUpload(path)
      .then(() => load()) // load() 重置 uploads 缓存并刷新统计
      .catch(() => setError("附件删除失败（路径不在上传目录白名单内）"));
    setConfirmId(null);
  }, [load]);

  const reveal = useCallback((p: string) => {
    void app.RevealWorkspacePath(p).catch(() => {});
  }, []);

  const preview = useCallback((p: string) => {
    usePreviewStore.getState().openFilePreview(p);
  }, []);

  const stats = overview
    ? [
        { label: "工作区", value: String(overview.projects.length) },
        { label: "会话", value: String(overview.totalSessions) },
        { label: "归档", value: String(overview.totalArchived) },
        { label: "上传文件", value: String(overview.totalUploads) },
        { label: "占用", value: formatBytes(overview.totalBytes) },
      ]
    : [];

  return (
    <div className="h-full flex flex-col text-fg-dim text-xs">
      <div className="shrink-0 flex items-center gap-2 px-4 py-2.5 border-b border-border-soft">
        <span className="flex items-center gap-1.5 font-semibold text-fg text-sm">
          <Database size={13} style={{ color: DOMAIN_COLORS.database }} />
          数据库
        </span>
        <span className="text-fg-faint text-[10.5px]">本设备聊天记录 · 上传文件（跨工作区）</span>
        <button
          type="button"
          data-testid="db-refresh"
          className="ml-auto inline-flex items-center gap-1 px-2 h-6 rounded-md border border-border-soft bg-transparent text-fg-faint cursor-pointer hover:text-fg hover:bg-bg-soft transition-colors text-[11px]"
          onClick={load}
          title="重新扫描各工作区存档"
        >
          <RefreshCw size={11} className={loading ? "animate-spin" : ""} />
          刷新
        </button>
      </div>

      <div className="flex-1 min-h-0 overflow-y-auto px-4 py-3 flex flex-col gap-4">
        {error && (
          <div data-testid="db-error" className="px-3 py-2 rounded-md border border-err/30 bg-err/5 text-err text-[11px] flex items-center gap-2">
            <span className="flex-1">{error}</span>
            <button
              type="button"
              className="shrink-0 border-0 bg-transparent text-err/70 cursor-pointer hover:text-err p-0.5"
              onClick={() => setError("")}
              aria-label="关闭错误提示"
              title="关闭"
            >
              <X size={12} />
            </button>
          </div>
        )}

        {/* 总览统计带（overview 未到时显示占位，避免布局跳动；错误态显「—」） */}
        <div className="grid grid-cols-5 gap-2" data-testid="db-stats">
          {(stats.length > 0 ? stats : [
            { label: "工作区", value: error ? "—" : "…" },
            { label: "会话", value: error ? "—" : "…" },
            { label: "归档", value: error ? "—" : "…" },
            { label: "上传文件", value: error ? "—" : "…" },
            { label: "占用", value: error ? "—" : "…" },
          ]).map((s) => (
            <div key={s.label} className="rounded-lg border border-border-soft/60 bg-bg-soft/30 px-2.5 py-2">
              <div className="text-[10px] text-fg-faint leading-none">{s.label}</div>
              <div className="mt-1 font-mono text-[15px] font-semibold text-fg leading-none tabular-nums truncate">{s.value}</div>
            </div>
          ))}
        </div>

        {/* 聊天记录（按项目折叠） */}
        <section>
          <div className="text-[10px] uppercase tracking-wider text-fg-faint/70 font-medium mb-1.5 flex items-center gap-1.5">
            <MessageSquare size={10} style={{ color: DOMAIN_COLORS.database }} />
            聊天记录 · {overview?.totalSessions ?? 0} 会话 / {overview?.totalArchived ?? 0} 归档
          </div>
          {!overview && !error ? (
            <div className="px-3 py-4 text-fg-faint/60 text-[11px]">读取中…</div>
          ) : error ? null : groups.length === 0 ? (
            <div className="px-3 py-4 rounded-lg border border-dashed border-border-soft text-fg-faint/60 text-center text-[11px]">
              本设备还没有保存的会话
            </div>
          ) : (
            <div className="flex flex-col gap-1.5">
              {groups.map((g) => {
                const st = overview?.projects.find((p) => p.path === g.path);
                // 侧边栏聚合绑定每组上限 50 条（maxProjectSessionsPerGroup）：
                // 总览统计（全量走查）大于行数 = 被截断，如实提示防「少了几条」困惑。
                const shown = g.sessions.length + g.archived.length;
                const capped = st && st.sessionCount + st.archivedCount > shown;
                return (
                  <details key={g.path} className="rounded-lg border border-border-soft/60 bg-bg-soft/25 open:bg-bg-soft/40 transition-colors" data-testid="db-project">
                    <summary className="cursor-pointer select-none list-none px-2.5 py-2 flex items-center gap-2 hover:border-accent/30">
                      <span className="min-w-0 flex-1 flex items-center gap-1.5">
                        <span className="truncate text-[12px] text-fg font-medium">{g.name}</span>
                        {g.current && <span className="shrink-0 px-1.5 rounded-full bg-accent/15 text-accent text-[9px] leading-4">当前</span>}
                      </span>
                      <span className="shrink-0 text-[10px] text-fg-faint font-mono text-right">
                        {g.sessions.length} 会话{g.archived.length > 0 ? ` · ${g.archived.length} 归档` : ""}
                        {st ? ` · ${formatBytes(st.sessionBytes)}` : ""}
                        {st && st.latestMod > 0 ? ` · ${formatTime(st.latestMod)}` : ""}
                      </span>
                    </summary>
                    <div className="px-2.5 pb-2 flex flex-col gap-0.5 border-t border-border-soft/40 pt-1.5">
                      {capped && (
                        <div className="px-1.5 py-1 text-[10px] text-fg-faint/80" data-testid="db-capped-note">
                          项目较大，仅显示最近 50 条；更早的请到办公板块该项目分组查看
                        </div>
                      )}
                      {[...g.sessions, ...g.archived].map((s) => {
                        const isArchived = !!s.archived;
                        return (
                          <div key={s.path} className="group flex items-center gap-2 px-1.5 py-1 rounded-md hover:bg-bg-soft/60">
                            <span className="min-w-0 flex-1">
                              <span className="block truncate text-[11.5px] text-fg leading-tight">
                                {s.title || s.preview || "(无标题)"}
                                {isArchived && <span className="ml-1.5 text-[9px] text-fg-faint/70">归档</span>}
                                {s.spaceId && s.spaceId !== "work" && <span className="ml-1.5 text-[9px] text-fg-faint/70">{s.spaceId}</span>}
                              </span>
                              <span className="block truncate text-[9.5px] text-fg-faint/70 font-mono leading-tight">
                                {formatTime(s.modTime)} · {s.turns} 轮
                              </span>
                            </span>
                            <div className="shrink-0 flex items-center gap-0.5 opacity-0 group-hover:opacity-100 group-focus-within:opacity-100 transition-opacity">
                              <button
                                type="button"
                                className="flex items-center justify-center w-6 h-6 rounded-md border-0 bg-transparent text-fg-faint cursor-pointer hover:text-fg hover:bg-bg transition-colors"
                                onClick={() => reveal(s.path)}
                                title="在文件夹中定位"
                              >
                                <ExternalLink size={11} />
                              </button>
                              <button
                                type="button"
                                data-testid="db-session-delete"
                                className={`flex items-center justify-center gap-1 h-6 px-1.5 rounded-md border-0 bg-transparent cursor-pointer transition-colors text-[10px] ${
                                  confirmId === s.path ? "text-err bg-err/10 font-semibold" : "text-fg-faint hover:text-err hover:bg-err/10"
                                }`}
                                onClick={() => (confirmId === s.path ? deleteSession(s.path) : setConfirmId(s.path))}
                                title={confirmId === s.path ? "再点一次确认删除" : "删除会话"}
                              >
                                <Trash2 size={11} />
                                {confirmId === s.path ? "确认删除" : ""}
                              </button>
                            </div>
                          </div>
                        );
                      })}
                    </div>
                  </details>
                );
              })}
            </div>
          )}
        </section>

        {/* 上传的文件（按项目折叠，展开时懒加载） */}
        <section>
          <div className="text-[10px] uppercase tracking-wider text-fg-faint/70 font-medium mb-1.5">
            上传的文件 · {overview?.totalUploads ?? 0}
          </div>
          {error ? null : (overview?.totalUploads ?? 0) === 0 ? (
            <div className="px-3 py-4 rounded-lg border border-dashed border-border-soft text-fg-faint/60 text-center text-[11px]">
              各工作区还没有上传的附件或粘贴图片
            </div>
          ) : (
            <div className="flex flex-col gap-1.5">
              {overview!.projects.filter((p) => p.uploadCount > 0).map((p) => (
                <details
                  key={p.path}
                  data-testid="db-uploads"
                  className="rounded-lg border border-border-soft/60 bg-bg-soft/25 transition-colors"
                  onToggle={(e) => { if ((e.target as HTMLDetailsElement).open) loadUploads(p.path); }}
                >
                  <summary className="cursor-pointer select-none list-none px-2.5 py-2 flex items-center gap-2">
                    <span className="min-w-0 flex-1 truncate text-[12px] text-fg font-medium">{p.name}</span>
                    <span className="shrink-0 text-[10px] text-fg-faint font-mono">{p.uploadCount} 个 · {formatBytes(p.uploadBytes)}</span>
                  </summary>
                  <div className="px-2.5 pb-2 flex flex-col gap-0.5 border-t border-border-soft/40 pt-1.5">
                    {(uploads[p.path] ?? []).map((f) => (
                      <div key={f.path} className="group flex items-center gap-2 px-1.5 py-1 rounded-md hover:bg-bg-soft/60">
                        <span className="shrink-0 w-5 h-5 rounded bg-bg-soft text-fg-faint flex items-center justify-center">
                          <FileText size={10} />
                        </span>
                        <button
                          type="button"
                          onClick={() => preview(f.path)}
                          className="min-w-0 flex-1 text-left cursor-pointer bg-transparent border-0 p-0"
                          title={`点击预览 ${f.name}`}
                        >
                          <span className="block truncate text-[11.5px] text-fg leading-tight">{f.name}</span>
                          <span className="block truncate text-[9.5px] text-fg-faint/70 font-mono leading-tight">
                            {formatBytes(f.size)} · {formatTime(f.modTime)} · {f.kind === "paste" ? "粘贴图" : f.kind === "attach" ? "附件" : "文件"}
                          </span>
                        </button>
                        <div className="shrink-0 flex items-center gap-0.5 opacity-0 group-hover:opacity-100 group-focus-within:opacity-100 transition-opacity">
                          <button type="button" className="flex items-center justify-center w-6 h-6 rounded-md border-0 bg-transparent text-fg-faint cursor-pointer hover:text-fg hover:bg-bg transition-colors" onClick={() => preview(f.path)} title="预览">
                            <Eye size={11} />
                          </button>
                          <button type="button" className="flex items-center justify-center w-6 h-6 rounded-md border-0 bg-transparent text-fg-faint cursor-pointer hover:text-fg hover:bg-bg transition-colors" onClick={() => reveal(f.path)} title="在文件夹中定位">
                            <ExternalLink size={11} />
                          </button>
                          <button
                            type="button"
                            data-testid="db-upload-delete"
                            className={`flex items-center justify-center gap-1 h-6 px-1.5 rounded-md border-0 bg-transparent cursor-pointer transition-colors text-[10px] ${
                              confirmId === f.path ? "text-err bg-err/10 font-semibold" : "text-fg-faint hover:text-err hover:bg-err/10"
                            }`}
                            onClick={() => (confirmId === f.path ? deleteUpload(f.path) : setConfirmId(f.path))}
                            title={confirmId === f.path ? "再点一次确认删除" : "删除文件"}
                          >
                            <Trash2 size={11} />
                            {confirmId === f.path ? "确认删除" : ""}
                          </button>
                        </div>
                      </div>
                    ))}
                  </div>
                </details>
              ))}
            </div>
          )}
        </section>
      </div>
    </div>
  );
}
