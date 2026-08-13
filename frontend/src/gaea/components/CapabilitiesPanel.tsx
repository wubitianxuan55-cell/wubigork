import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { X, Globe, Cpu, ChevronDown, Search, RefreshCw } from "../icons";
import { app } from "../lib/bridge";
import { useT } from "../lib/i18n";
import { summarizeServerError, summarizeSkillDescription } from "../lib/capabilities";
import type { CapabilitiesView, MCPServerInput, ServerView, SkillView } from "../lib/types";
import { DrawerHeader, DrawerTitle, DrawerSubtitle } from "./DrawerHeader";
import { ResizableDrawer } from "./ResizableDrawer";
import { useGSAPCollapse } from "../lib/useGSAPCollapse";

// CapabilitiesPanel is the desktop MCP & Skills drawer — the GUI counterpart to
// the CLI's /mcp + /skill, aligning with Claude Code's Customize → Connectors:
// each server shows a connected/failed dot, transport, and tool/prompt/resource
// counts, with add / remove / retry; skills list their scope and run mode.
type CapTab = "servers" | "tools" | "skills";
export function CapabilitiesPanel({
  onClose,
  toolCounts = {},
  skillCounts = {},
}: {
  onClose: () => void;
  toolCounts?: Record<string, number>;
  skillCounts?: Record<string, number>;
}) {
  const t = useT();
  const [view, setView] = useState<CapabilitiesView | null>(null);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [reloading, setReloading] = useState(false);
  const [adding, setAdding] = useState(false);
  const [addingContext7, setAddingContext7] = useState(false);
  const [confirming, setConfirming] = useState<string | null>(null);
  const [tab, setTab] = useState<CapTab>("servers");
  const [skillQuery, setSkillQuery] = useState("");
  const [expandedSkills, setExpandedSkills] = useState<Set<string>>(() => new Set());
  const [expandedErrors, setExpandedErrors] = useState<Set<string>>(() => new Set());
  const [expandedServers, setExpandedServers] = useState<Set<string>>(() => new Set());

  const reload = async () =>
    setView(await app.Capabilities().catch(() => ({ servers: [], skills: [] })));
  useEffect(() => {
    void reload();
  }, []);

  // reloadEngine 热加载办公引擎：重建 controller 后重新拉取能力快照，
  // 外部编辑的技能/工具/插件目录无需重启桌面端即可被引擎感知。
  const reloadEngine = async () => {
    setReloading(true);
    setNotice(null);
    setErr(null);
    try {
      const res = await app.Reload();
      await reload();
      setNotice(t("caps.reloaded", { tools: res.tools, skills: res.skills }));
    } catch (e) {
      setErr(String((e as Error)?.message ?? e));
    } finally {
      setReloading(false);
    }
  };

  // mutate runs an MCP edit, re-reads the snapshot, and surfaces any failure as an
  // inline banner (a connect error, a missing binary, a bad URL).
  const mutate = async (fn: () => Promise<unknown>) => {
    setBusy(true);
    setErr(null);
    try {
      await fn();
      await reload();
      return true;
    } catch (e) {
      setErr(String((e as Error)?.message ?? e));
      return false;
    } finally {
      setBusy(false);
    }
  };

  const addContext7 = async () => {
    setAddingContext7(true);
    setErr(null);
    try {
      await app.AddMCPServer({
        name: "context7",
        transport: "streamable-http",
        command: "",
        args: [],
        url: "https://mcp.context7.com/mcp",
        env: {},
      });
      await reload();
    } catch (e) {
      setErr(String((e as Error)?.message ?? e));
    } finally {
      setAddingContext7(false);
    }
  };

  const summary = useMemo(() => {
    if (!view) return "";
    return t("caps.summary", {
      connected: view.servers.filter((s) => s.status === "connected").length,
      failed: view.servers.filter((s) => s.status === "failed").length,
      skills: view.skills.length,
    });
  }, [view, t]);

  const filteredSkills = useMemo(() => {
    if (!view) return [];
    const q = skillQuery.trim().toLowerCase();
    if (!q) return view.skills;
    return view.skills.filter((sk) => {
      const text = [sk.name, `/${sk.name}`, sk.description, sk.scope, sk.runAs].join(" ").toLowerCase();
      return text.includes(q);
    });
  }, [view, skillQuery]);

  const serverGroups = useMemo(() => {
    const servers = view?.servers ?? [];
    return {
      failed: servers.filter((s) => s.status === "failed"),
      active: servers.filter((s) => s.status !== "failed"),
    };
  }, [view]);

  const toggleSkill = useCallback((name: string) => {
    setExpandedSkills((prev) => {
      const next = new Set(prev);
      if (next.has(name)) next.delete(name);
      else next.add(name);
      return next;
    });
  }, []);

  const toggleError = useCallback((name: string) => {
    setExpandedErrors((prev) => {
      const next = new Set(prev);
      if (next.has(name)) next.delete(name);
      else next.add(name);
      return next;
    });
  }, []);

  const toggleServer = useCallback((name: string) => {
    setExpandedServers((prev) => {
      const next = new Set(prev);
      if (next.has(name)) next.delete(name);
      else next.add(name);
      return next;
    });
  }, []);

  return (
    <ResizableDrawer onClose={onClose} subtle>
        <DrawerHeader onClose={onClose}>
          <div>
            <DrawerTitle text={t("caps.title")} />
            {view && <DrawerSubtitle text={summary} />}
          </div>
          <button
            className="flex items-center gap-1.5 px-2 py-1 text-xs border border-border-soft rounded text-fg-dim cursor-pointer hover:text-fg hover:bg-bg-soft transition-colors disabled:opacity-40 disabled:cursor-default"
            disabled={reloading}
            onClick={() => void reloadEngine()}
            title={t("caps.reloadHint")}
          >
            {reloading ? (
              <span className="animate-spin inline-block w-3 h-3 border border-current border-t-transparent rounded-full" />
            ) : (
              <RefreshCw size={13} />
            )}
            <span>{reloading ? t("caps.reloading") : t("caps.reload")}</span>
          </button>
        </DrawerHeader>

        {!view ? (
          <div className="empty-state">{t("caps.loading")}</div>
        ) : (
          <div className="overflow-y-auto px-4 py-3.5 flex flex-col gap-5">
            {err && <div className="shrink-0 px-4 py-2 text-[12.5px] bg-del-bg text-err border-b border-border-soft">{err}</div>}
            {notice && <div className="shrink-0 px-4 py-2 text-[12.5px] bg-ok/10 text-ok border-b border-ok/20">{notice}</div>}
            <div className="flex border-b border-border-soft mb-3" role="tablist" aria-label={t("caps.title")}>
              <button
                className={`flex-1 px-4 py-2 border-0 border-b-2 bg-transparent text-[13px] font-medium cursor-pointer transition-[color,border] duration-[var(--dur-fast)] ${
                  tab === "servers" ? "text-accent border-accent" : "text-fg-dim border-transparent hover:text-fg hover:border-fg-faint"
                }`}
                role="tab" aria-selected={tab === "servers"} onClick={() => setTab("servers")}
              >{t("caps.connectorsTab")}</button>
              <button
                className={`flex-1 px-4 py-2 border-0 border-b-2 bg-transparent text-[13px] font-medium cursor-pointer transition-[color,border] duration-[var(--dur-fast)] ${
                  tab === "tools" ? "text-accent border-accent" : "text-fg-dim border-transparent hover:text-fg hover:border-fg-faint"
                }`}
                role="tab" aria-selected={tab === "tools"} onClick={() => setTab("tools")}
              >
                <Cpu size={12} className="inline mr-1 align-middle -mt-px" />
                <span>工具</span>
              </button>
              <button
                className={`flex-1 px-4 py-2 border-0 border-b-2 bg-transparent text-[13px] font-medium cursor-pointer transition-[color,border] duration-[var(--dur-fast)] ${
                  tab === "skills" ? "text-accent border-accent" : "text-fg-dim border-transparent hover:text-fg hover:border-fg-faint"
                }`}
                role="tab" aria-selected={tab === "skills"} onClick={() => setTab("skills")}
              >{t("caps.skillsTab")}</button>
            </div>

            {tab === "servers" ? (
              <section className="mb-3">
                <div className="flex justify-end mb-2">
                  {/* Context7 一键添加 */}
                  <button
                    className="flex items-center gap-1.5 mr-2 px-2.5 py-1 text-xs border border-accent/30 rounded bg-accent/5 text-accent cursor-pointer hover:bg-accent/10 transition-colors disabled:opacity-40"
                    disabled={busy || addingContext7}
                    onClick={() => addContext7()}
                    title={t("caps.addContext7Hint")}
                  >
                    {addingContext7 ? (
                      <span className="animate-spin inline-block w-3 h-3 border border-current border-t-transparent rounded-full" />
                    ) : (
                      <Globe size={12} />
                    )}
                    <span>{addingContext7 ? t("caps.addContext7Busy") : t("caps.addContext7")}</span>
                  </button>
                  {!adding && (
                    <button className="px-2.5 py-1 text-xs" disabled={busy} onClick={() => setAdding(true)}>
                      {t("caps.addServer")}
                    </button>
                  )}
                </div>
                {serverGroups.failed.length > 0 && (
                  <FailedServersNotice
                    servers={serverGroups.failed}
                    expanded={expandedErrors}
                    onToggle={toggleError}
                    onRetry={(name) => void mutate(() => app.RetryMCPServer(name))}
                    confirming={confirming}
                    onConfirm={setConfirming}
                    onCancelConfirm={() => setConfirming(null)}
                    onRemove={(name) => mutate(() => app.RemoveMCPServer(name)).then(() => setConfirming(null))}
                    busy={busy}
                  />
                )}
                {view.servers.length === 0 && !adding && (
                  <div className="text-fg-faint text-xs text-center py-4">{t("caps.noServers")}</div>
                )}
                <ServerGroup
                  busy={busy}
                  servers={serverGroups.active}
                  expanded={expandedServers}
                  confirming={confirming}
                  onConfirm={setConfirming}
                  onCancelConfirm={() => setConfirming(null)}
                  onRemove={(name) => mutate(() => app.RemoveMCPServer(name)).then(() => setConfirming(null))}
                  onRetry={(name) => void mutate(() => app.RetryMCPServer(name))}
                  onToggle={(name, on) => void mutate(() => app.SetMCPServerEnabled(name, on))}
                  onToggleDetails={toggleServer}
                />
                {adding ? (
                  <AddServerForm busy={busy} onCancel={() => setAdding(false)} onAdd={async (input) => (await mutate(() => app.AddMCPServer(input))) && setAdding(false)} />
                ) : null}
              </section>
            ) : tab === "tools" ? (
              <ToolsTabContent toolCounts={toolCounts} />
            ) : (
              <section className="mb-3">
                <div className="mb-2">
                  <input
                    className="w-full bg-bg-soft border border-border-soft rounded-md text-fg text-[13px] px-2.5 py-1.5 outline-none placeholder:text-fg-faint focus:border-accent"
                    type="search"
                    placeholder={t("caps.searchSkills")}
                    value={skillQuery}
                    onChange={(e) => setSkillQuery(e.target.value)}
                  />
                </div>
                {view.skills.length === 0 ? (
                  <div className="py-4 text-fg-faint text-xs text-center">{t("caps.noSkills")}</div>
                ) : filteredSkills.length === 0 ? (
                  <div className="py-4 text-fg-faint text-xs text-center">{t("caps.noSkillMatches")}</div>
                ) : (
                  <div className="flex flex-col gap-2">
                    {filteredSkills.map((sk) => (
                      <SkillRow
                        key={sk.name}
                        skill={sk}
                        count={skillCounts[sk.name] ?? 0}
                        expanded={expandedSkills.has(sk.name)}
                        onToggle={() => toggleSkill(sk.name)}
                      />
                    ))}
                  </div>
                )}
              </section>
            )}
          </div>
        )}
    </ResizableDrawer>
  );
}

function ServerGroup({
  servers,
  expanded,
  busy,
  confirming,
  onConfirm,
  onCancelConfirm,
  onRemove,
  onRetry,
  onToggle,
  onToggleDetails,
}: {
  servers: ServerView[];
  expanded: Set<string>;
  busy: boolean;
  confirming: string | null;
  onConfirm: (name: string) => void;
  onCancelConfirm: () => void;
  onRemove: (name: string) => void;
  onRetry: (name: string) => void;
  onToggle: (name: string, on: boolean) => void;
  onToggleDetails: (name: string) => void;
}) {
  if (servers.length === 0) return null;
  return (
    <div className="flex flex-col mt-3">
      {servers.map((s) => (
        <ServerRow
          key={s.name}
          s={s}
          expanded={expanded.has(s.name)}
          busy={busy}
          confirming={confirming === s.name}
          onConfirm={() => onConfirm(s.name)}
          onCancelConfirm={onCancelConfirm}
          onRemove={() => onRemove(s.name)}
          onRetry={() => onRetry(s.name)}
          onToggle={(on) => onToggle(s.name, on)}
          onToggleDetails={() => onToggleDetails(s.name)}
        />
      ))}
    </div>
  );
}

function FailedServersNotice({
  servers,
  expanded,
  busy,
  confirming,
  onToggle,
  onRetry,
  onConfirm,
  onCancelConfirm,
  onRemove,
}: {
  servers: ServerView[];
  expanded: Set<string>;
  busy: boolean;
  confirming: string | null;
  onToggle: (name: string) => void;
  onRetry: (name: string) => void;
  onConfirm: (name: string) => void;
  onCancelConfirm: () => void;
  onRemove: (name: string) => void;
}) {
  const t = useT();
  return (
    <div className="mb-3 p-3 border border-err/20 rounded-lg" role="status" style={{background: "var(--ds-danger-soft)"}}>
      <div className="flex items-center justify-between mb-2">
        <div>
          <div className="text-err text-sm font-semibold">{t("caps.failureTitle", { failed: servers.length })}</div>
          <div className="text-fg-faint text-[11px]">{t("caps.failureHint")}</div>
        </div>
      </div>
      <div className="flex flex-col gap-2">
        {servers.map((s) => {
          const open = expanded.has(s.name);
          const error = s.error || t("caps.failed");
          return (
            <div className="border border-border-soft rounded-lg overflow-hidden" key={s.name}>
              <div className="flex items-center gap-2 px-3 py-2">
                <span className="w-2 h-2 rounded-full bg-err shrink-0" />
                <div className="flex-1 min-w-0">
                  <div className="text-fg text-[13px] font-medium">{s.name}</div>
                  <div className="text-fg-faint text-[11px] truncate">{summarizeServerError(error)}</div>
                </div>
              </div>
              <div className="flex items-center gap-1 px-3 pb-2">
                {confirming === s.name ? (
                  <>
                    <button className="px-2.5 py-1 text-xs" disabled={busy} onClick={() => onRemove(s.name)}>{t("caps.confirmRemove")}</button>
                    <button className="px-2.5 py-1 text-xs" disabled={busy} onClick={onCancelConfirm}>{t("common.cancel")}</button>
                  </>
                ) : (
                  <>
                    <button className="px-2.5 py-1 text-xs" disabled={busy} onClick={() => onRetry(s.name)}>{t("caps.retry")}</button>
                    <button className="px-2.5 py-1 text-xs border border-border-soft rounded bg-transparent text-fg-dim cursor-pointer hover:text-fg hover:bg-bg-soft transition-colors" onClick={() => void navigator.clipboard?.writeText(error)}>{t("common.copy")}</button>
                    <button className="px-2.5 py-1 text-xs" onClick={() => onToggle(s.name)} aria-expanded={open}>{open ? t("common.collapse") : t("caps.showLog")}</button>
                    <button className="px-2.5 py-1 text-xs border border-border-soft rounded bg-transparent text-fg-dim cursor-pointer hover:text-err hover:bg-bg-soft transition-colors" disabled={busy} onClick={() => onConfirm(s.name)} title={t("caps.remove")}><X size={13} /></button>
                  </>
                )}
              </div>
              {open && <pre className="m-0 p-3 bg-bg text-fg-dim text-xs leading-relaxed whitespace-pre-wrap border-t border-border-soft max-h-[200px] overflow-y-auto">{error}</pre>}
            </div>
          );
        })}
      </div>
    </div>
  );
}

function ServerRow({
  s,
  expanded,
  busy,
  confirming,
  onConfirm,
  onCancelConfirm,
  onRemove,
  onRetry,
  onToggle,
  onToggleDetails,
}: {
  s: ServerView;
  expanded: boolean;
  busy: boolean;
  confirming: boolean;
  onConfirm: () => void;
  onCancelConfirm: () => void;
  onRemove: () => void;
  onRetry: () => void;
  onToggle: (on: boolean) => void;
  onToggleDetails: () => void;
}) {
  const t = useT();
  const actionLabel = serverActionLabel(s, t);
  const tools = s.toolList ?? [];
  const hasTools = tools.length > 0;
  const sub =
    s.status === "failed"
      ? s.error || t("caps.failed")
      : s.status === "disabled"
        ? t("caps.disabled")
        : t("caps.counts", { tools: s.tools, prompts: s.prompts, resources: s.resources });
  return (
    <div className={`border border-border-soft rounded-lg ${s.status === "disabled" ? "opacity-60" : ""}`}>
      <div className="flex items-center gap-2 px-3 py-2" title={s.error || undefined}>
        <button
          className="w-5 h-5 border-0 bg-transparent text-fg-faint cursor-pointer flex items-center justify-center text-sm disabled:opacity-30 disabled:cursor-default"
          disabled={!hasTools}
          aria-expanded={hasTools ? expanded : undefined}
          onClick={onToggleDetails}
          title={hasTools ? (expanded ? t("caps.collapseTools") : t("caps.expandTools")) : t("caps.noToolDetails")}
        >
          {hasTools ? (expanded ? "⌄" : "›") : ""}
        </button>
        <span className={`w-2 h-2 rounded-full shrink-0 ${s.status === "connected" ? "bg-ok" : s.status === "failed" ? "bg-err" : "bg-fg-faint"}`} />
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className="text-fg text-[13px] font-medium">{s.name}</span>
            <span className="text-fg-faint text-[11px] font-mono">{s.transport}</span>
          </div>
          <div className={`text-[11px] truncate ${s.status === "disabled" ? "text-fg-faint opacity-60" : "text-fg-faint"}`}>{sub}</div>
        </div>
        <div className="flex items-center gap-1 shrink-0">
          {confirming ? (
            <>
              <button className="px-2.5 py-1 text-xs" disabled={busy} onClick={onRemove}>{t("caps.confirmRemove")}</button>
              <button className="px-2.5 py-1 text-xs" disabled={busy} onClick={onCancelConfirm}>{t("common.cancel")}</button>
            </>
          ) : (
            <>
              {s.status === "failed" ? (
                <button className="px-2.5 py-1 text-xs" disabled={busy} onClick={onRetry}>{actionLabel}</button>
              ) : (
                <label className="inline-flex cursor-pointer no-drag" title={s.status === "connected" ? t("caps.disable") : t("caps.enable")}>
                  <input type="checkbox" className="peer absolute opacity-0 w-0 h-0" checked={s.status === "connected"} disabled={busy} onChange={(e) => onToggle(e.target.checked)} />
                  <span className="relative w-[30px] h-[17px] rounded-full bg-border transition-colors duration-[var(--dur-base)] peer-checked:bg-ok peer-disabled:opacity-50 peer-checked:[&>span]:translate-x-[13px]">
                    <span className="absolute top-0.5 left-0.5 w-[13px] h-[13px] rounded-full bg-bg-elev transition-transform duration-[var(--dur-base)]" />
                  </span>
                </label>
              )}
              <button className="px-2.5 py-1 text-xs border border-border-soft rounded bg-transparent text-fg-dim cursor-pointer hover:text-err hover:bg-bg-soft transition-colors" disabled={busy} onClick={onConfirm} title={t("caps.remove")}><X size={13} /></button>
            </>
          )}
        </div>
      </div>
      {hasTools && expanded && (
        <div className="border-t border-border-soft px-3 py-2">
          <div className="text-fg-faint text-[11px] font-medium mb-1">{t("caps.tools")}</div>
          {tools.map((tool) => (
            <div className="flex items-center gap-2 px-2 py-1" key={tool.name}>
              <span className="font-mono text-fg text-[13px]">{tool.name}</span>
              {tool.description && <span className="text-fg-faint text-[11px] truncate">{tool.description}</span>}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function serverActionLabel(s: ServerView, t: ReturnType<typeof useT>): string {
  const err = (s.error || "").toLowerCase();
  if (err.includes("401") || err.includes("unauthorized")) return t("caps.reauthorize");
  if (
    err.includes("command not found") ||
    err.includes("executable file not found") ||
    err.includes("no such file") ||
    err.includes("enoent")
  ) {
    return t("caps.checkCommand");
  }
  return t("caps.retry");
}
// ── Tool list (from RuntimePanel, inlined) ──────────────────────────────

type Counts = Record<string, number>;

const TOOL_DESC: Record<string, string> = {
  read_file: "读取文件内容(可选行范围/分页)",
  write_file: "写入/覆盖文件(自动建父目录)",
  ls: "列目录条目(子目录带/)",
  bash: "执行shell命令(合并stdout+stderr,支持后台任务)",
  bash_output: "读取后台任务的增量输出(不阻塞)",
  wait: "阻塞等待后台任务结束(可设超时)",
  kill_shell: "终止后台任务(SIGTERM→SIGKILL)",
  web_fetch: "抓取URL纯文本(去标签,SSRF安全)",
  web_search: "搜索公开网页(返回结构化JSON)",
  todo_write: "更新任务清单(全量替换,最多一个进行中)",
  complete_step: "完成计划步骤(附验证证据,空证据拒绝)",
  ask: "向用户提供多选项问题",
  task: "派发子代理执行聚焦子任务",
  remember: "保存持久事实到项目记忆",
  forget: "通过名称删除已保存记忆",
  memory_search: "按关键词搜索已保存记忆",
  memory_get: "按ID读取单条记忆详情",
  promote_session_facts: "把当前会话事实提升为持久记忆",
  knowledge_search: "搜索工程知识库(关键词/分类/标签)",
  knowledge_add: "向知识库添加条目(标题+分类+正文)",
  read_skill: "读取指定技能的完整内容",
  run_skill: "调用Skills索引中的playbook",
  install_skill: "编写并保存新技能",
  slash_command: "按名称调用项目斜杠命令与技能",
  format_convert: "文档格式转换(docx/xlsx/pptx/pdf→Markdown,含OCR回退)",
  chart_gen: "matplotlib图表生成(bar/line/pie/scatter)",
  "format-convert": "子代理——格式转换(统一为可编辑Markdown)",
  "chart-builder": "子代理——图表生成(数据可视化)",
  "doc-assemble": "子代理——文档拼装(多份素材→完整报告)",
  docx: "Word 文档技能——创建/读取/编辑 .docx(模板、目录、修订、排版)",
  xlsx: "Excel 表格技能——创建/处理 .xlsx/.csv(公式、格式、清洗、图表)",
  pdf: "PDF 文档技能——读取/合并/拆分/水印/表单/OCR",
  pptx: "演示文稿技能——把内容做成 .pptx 幻灯片",
  ocr: "本地 OCR 模型——提取图片/扫描件中的精确文字",
  semantic_search: "本地 bge-m3——跨库语义检索(成本/知识/办公记忆)",
  routine_llm: "通用文本兜底——纯文本摘要/归一化/抽取/改写，本地/免费优先",
  translate_text: "本地翻译——优先 Hunyuan-MT / Hy-MT 翻译模型，未安装时回退办公模型",
};

interface Section {
  title: string;
  items: string[];
}

const SECTIONS: Section[] = [
  { title: "文件与命令", items: ["read_file", "write_file", "ls", "bash", "bash_output", "wait", "kill_shell"] },
  { title: "网络", items: ["web_search", "web_fetch"] },
  { title: "任务与协作", items: ["todo_write", "complete_step", "ask", "task"] },
  { title: "记忆", items: ["remember", "forget", "memory_search", "memory_get", "promote_session_facts"] },
  { title: "知识库", items: ["knowledge_search", "knowledge_add"] },
  { title: "技能", items: ["read_skill", "run_skill", "install_skill", "slash_command"] },
  { title: "文档技能", items: ["docx", "xlsx", "pdf", "pptx"] },
  { title: "本地专业模型", items: ["vision", "ocr", "semantic_search", "translate_text"] },
  { title: "通用办公", items: ["format_convert", "chart_gen", "routine_llm", "format-convert", "chart-builder", "doc-assemble"] },
];

function ToolCard({ name, count }: { name: string; count: number }) {
  const active = count > 0;
  const desc = TOOL_DESC[name];
  return (
    <div
      className={`flex items-start gap-1.5 px-2 py-1.5 rounded-md border border-border-soft bg-bg cursor-default ${
        active ? "border-accent-soft bg-sidebar-active" : ""
      }`}
      title={desc ?? name}
    >
      <span className={`w-1.5 h-1.5 mt-[5px] rounded-full shrink-0 ${active ? "bg-accent" : "bg-border-soft"}`} />
      <span className="flex-1 min-w-0 flex flex-col gap-0.5 leading-[1.25]">
        <span className={`font-mono text-[10.5px] truncate ${active ? "text-accent font-semibold" : "text-fg-dim"}`}>
          {name}
        </span>
        {desc && <span className="text-[10px] text-fg-faint leading-[1.3] line-clamp-1">{desc}</span>}
      </span>
      <span className={`shrink-0 font-mono text-[11px] font-semibold mt-px ${active ? "text-accent" : "text-fg-faint"}`}>
        {count}
      </span>
    </div>
  );
}

function ToolGroup({
  title,
  items,
  counts,
  defaultOpen,
}: {
  title: string;
  items: string[];
  counts: Counts;
  defaultOpen: boolean;
}) {
  const [open, setOpen] = useState(defaultOpen);
  const ref = useRef<HTMLDivElement>(null);
  useGSAPCollapse(ref, open, { duration: 0.18 });

  const activeCount = items.filter((n) => (counts[n] ?? 0) > 0).length;

  return (
    <div className="px-1.5 py-0.5">
      <button
        className="flex items-center gap-1 w-full px-1 py-1.5 bg-transparent border-0 text-left cursor-pointer hover:bg-bg-soft rounded transition-colors"
        onClick={() => setOpen((v) => !v)}
      >
        <ChevronDown
          size={10}
          className={`text-fg-faint transition-transform duration-150 ${open ? "rotate-0" : "-rotate-90"}`}
        />
        <span className="text-[10px] font-semibold uppercase tracking-[0.5px] text-fg-faint">{title}</span>
        {activeCount > 0 && (
          <span className="ml-auto text-[9px] font-mono text-accent">{activeCount}</span>
        )}
      </button>
      <div ref={ref} style={{ overflow: "hidden" }}>
        <div className="flex flex-col gap-0.5 pt-0.5 pb-1">
          {items.map((name) => (
            <ToolCard key={name} name={name} count={counts[name] ?? 0} />
          ))}
        </div>
      </div>
    </div>
  );
}

function ToolsTabContent({ toolCounts }: { toolCounts: Counts }) {
  const [query, setQuery] = useState("");
  const inputRef = useRef<HTMLInputElement>(null);

  const totalTools = SECTIONS.reduce((sum, s) => sum + s.items.length, 0);
  const activeTotal = useMemo(
    () => SECTIONS.reduce((sum, s) => sum + s.items.filter((n) => (toolCounts[n] ?? 0) > 0).length, 0),
    [toolCounts],
  );

  const filteredSections = useMemo(() => {
    if (!query.trim()) return SECTIONS;
    const q = query.toLowerCase();
    return SECTIONS
      .map((sec) => ({
        ...sec,
        items: sec.items.filter(
          (name) =>
            name.toLowerCase().includes(q) ||
            (TOOL_DESC[name] ?? "").toLowerCase().includes(q),
        ),
      }))
      .filter((sec) => sec.items.length > 0);
  }, [query]);

  const hasResults = filteredSections.length > 0;

  return (
    <div className="flex flex-col overflow-hidden h-full" style={{minHeight: 0}}>
      <div className="flex items-center gap-1.5 px-2 py-2 text-fg-dim font-semibold text-[11px] shrink-0">
        <Cpu size={12} />
        <span>工具</span>
        <span className="ml-auto text-[10px] font-mono text-fg-faint/50">
          {activeTotal > 0 ? `${activeTotal}/${totalTools}` : totalTools}
        </span>
      </div>
      <div className="flex items-center gap-1.5 mx-2 my-1 px-2 h-7 border border-border rounded-md bg-bg text-fg-faint shrink-0">
        <Search size={12} />
        <input
          ref={inputRef}
          className="flex-1 min-w-0 border-0 outline-none bg-transparent text-fg text-[11.5px] placeholder:text-fg-faint"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="搜索工具…"
        />
        {query && (
          <button
            className="border-0 bg-transparent text-fg-faint cursor-pointer p-0 leading-none hover:text-fg"
            onClick={() => { setQuery(""); inputRef.current?.focus(); }}
          >
            ✕
          </button>
        )}
      </div>
      <div className="flex-1 min-h-0 overflow-y-auto pb-2">
        {!hasResults ? (
          <div className="text-fg-faint text-xs text-center py-8">无匹配工具</div>
        ) : (
          filteredSections.map((sec) => (
            <ToolGroup
              key={sec.title}
              title={sec.title}
              items={sec.items}
              counts={toolCounts}
              defaultOpen={sec.title === "文件" || filteredSections.length <= 3}
            />
          ))
        )}
      </div>
    </div>
  );
}

function SkillRow({
  skill,
  count,
  expanded,
  onToggle,
}: {
  skill: SkillView;
  count: number;
  expanded: boolean;
  onToggle: () => void;
}) {
  const t = useT();
  const summary = summarizeSkillDescription(skill.description);
  const canExpand = summary !== skill.description;
  return (
    <button
      className={`w-full text-left border border-border-soft rounded-lg p-3 bg-transparent cursor-pointer transition-[border-color,background] duration-[var(--dur-fast)] hover:border-accent/30 hover:bg-bg-soft active:bg-bg-elev ${
        expanded ? "border-accent/30 bg-bg-elev" : ""
      }`}
      type="button"
      onClick={onToggle}
      aria-expanded={expanded}
      title={skill.description}
    >
      <div className="flex items-center gap-2.5 mb-1">
        <span className="w-8 h-8 flex items-center justify-center rounded-md bg-accent-soft text-accent font-mono text-base font-bold shrink-0">/</span>
        <span className="flex-1 min-w-0 flex flex-col gap-0.5">
          <span className="text-fg text-[13px] font-semibold font-mono">{skill.name}</span>
          <span className="flex items-center gap-1">
            <span className={`badge ${
              skill.scope === "project" ? "badge--success" : "badge--muted"
            }`}>{skillScopeLabel(skill.scope, t)}</span>
            {skill.runAs === "subagent" && <span className="badge badge--accent">{t("caps.subagent")}</span>}
          </span>
        </span>
        {count > 0 && (
          <span className="shrink-0 font-mono text-[11px] font-semibold text-accent">{count}</span>
        )}
      </div>
      <div className={`text-fg-dim text-[12px] leading-snug ${expanded ? "" : "line-clamp-2"}`}>
        {expanded ? skill.description : summary}
      </div>
      {canExpand && <div className="mt-1 text-fg-faint text-[11px]">{expanded ? t("common.collapse") : t("common.expand")}</div>}
    </button>
  );
}

function skillScopeLabel(scope: string, t: ReturnType<typeof useT>): string {
  switch (scope) {
    case "builtin":
      return t("caps.skillScopeBuiltin");
    case "project":
      return t("caps.skillScopeProject");
    case "custom":
      return t("caps.skillScopeCustom");
    case "global":
      return t("caps.skillScopeGlobal");
    default:
      return scope;
  }
}

function AddServerForm({
  busy,
  onCancel,
  onAdd,
}: {
  busy: boolean;
  onCancel: () => void;
  onAdd: (input: MCPServerInput) => void;
}) {
  const t = useT();
  const [name, setName] = useState("");
  const [transport, setTransport] = useState("stdio");
  const [command, setCommand] = useState("");
  const [url, setUrl] = useState("");
  const [env, setEnv] = useState("");

  const isStdio = transport === "stdio";
  const ready = name.trim() !== "" && (isStdio ? command.trim() !== "" : url.trim() !== "");

  const submit = () => {
    const parts = command.trim().split(/\s+/).filter(Boolean);
    const envMap: Record<string, string> = {};
    for (const line of env.split("\n")) {
      const eq = line.indexOf("=");
      if (eq > 0) envMap[line.slice(0, eq).trim()] = line.slice(eq + 1).trim();
    }
    onAdd({
      name: name.trim(),
      transport,
      command: isStdio ? (parts[0] ?? "") : "",
      args: isStdio ? parts.slice(1) : [],
      url: isStdio ? "" : url.trim(),
      env: envMap,
    });
  };

  return (
    <div className="flex flex-col gap-2 p-3 border border-border-soft rounded-lg mb-2">
      <input className="flex-1 bg-bg-soft border border-border-soft rounded-md text-fg text-[13px] px-2.5 py-1.5 outline-none placeholder:text-fg-faint focus:border-accent" placeholder={t("caps.namePlaceholder")} value={name} onChange={(e) => setName(e.target.value)} />
      <label className="text-fg-dim text-[13px] shrink-0">{t("caps.transport")}</label>
      <select className="bg-bg-soft border border-border-soft rounded-md text-fg text-[13px] px-2.5 py-1.5 outline-none focus:border-accent" value={transport} onChange={(e) => setTransport(e.target.value)}>
        <option value="stdio">stdio</option>
        <option value="http">http</option>
        <option value="sse">sse</option>
      </select>
      {isStdio ? (
        <input className="flex-1 bg-bg-soft border border-border-soft rounded-md text-fg text-[13px] px-2.5 py-1.5 outline-none placeholder:text-fg-faint focus:border-accent" placeholder={t("caps.commandPlaceholder")} value={command} onChange={(e) => setCommand(e.target.value)} />
      ) : (
        <input className="flex-1 bg-bg-soft border border-border-soft rounded-md text-fg text-[13px] px-2.5 py-1.5 outline-none placeholder:text-fg-faint focus:border-accent" placeholder={t("caps.urlPlaceholder")} value={url} onChange={(e) => setUrl(e.target.value)} />
      )}
      <label className="text-fg-dim text-[13px] shrink-0">{t("caps.envLabel")}</label>
      <textarea className="bg-bg-soft border border-border-soft rounded-md text-fg text-[13px] p-2 outline-none resize-y min-h-[60px] focus:border-accent" value={env} onChange={(e) => setEnv(e.target.value)} placeholder={t("caps.envPlaceholder")} spellCheck={false} />
      <div className="flex justify-end gap-2 mt-2">
        <button className="px-2.5 py-1 text-xs border border-border-soft rounded bg-transparent text-fg-dim cursor-pointer hover:text-fg hover:bg-bg-soft transition-colors" onClick={onCancel} disabled={busy}>
          {t("common.cancel")}
        </button>
        <button className="btn--primary" onClick={submit} disabled={busy || !ready}>
          {t("caps.add")}
        </button>
      </div>
    </div>
  );
}
