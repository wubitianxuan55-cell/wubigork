import { Fragment, memo, useCallback, useEffect, useMemo, useState } from "react";
import { Archive, ClipboardList, Coins, Copy, ExternalLink, FileText, FolderTree, ListTree, Loader2, MessageSquare, Paperclip, Rollback, Shield, Sparkles, Table } from "../icons";
import { app } from "../lib/bridge";
import type { DeliverableRegistryView, JournalChangeRecord, VerdictView, VerifyDiffRow } from "../lib/types";
import {
  groupVersionsByPath,
  normalizeVersionPath,
  versionLabel,
} from "../lib/versionTimeline";
import {
  buildCellIndex,
  buildVerifyDiff,
  describeOp,
  isClaimableOp,
  opBatchCount,
  opImpact,
  parseOps,
} from "../lib/verifyDiff";
import { useComposerInsertStore, usePreviewStore, useUpdatedFilesStore } from "../lib/store";
import { FRONTEND_EVENTS, emitFrontendEvent } from "../../events";
import { useToast } from "./Toast";
import { FileThumb } from "./FileThumb";
import { VersionTimeline } from "./VersionTimeline";

export interface SessionDeliverable {
  path: string;
  sourceId: string;
  turn?: number;
  /** 同一文件在会话内被提及/更新的次数（≥1）；>1 显示版本徽标与步进器。 */
  versions?: number;
}

const SPREADSHEET_EXT_RE = /\.(xlsx?|csv|et|ods)$/i;

function extOf(path: string): string {
  const m = /\.[^.\\/]+$/.exec(path);
  return m ? m[0].toLowerCase() : "";
}

function baseName(path: string): string {
  return path.split(/[\\/]/).pop() ?? path;
}

// 小图标操作按钮：令牌化 + 可见焦点环（全局 :focus-visible）+ aria-label
const iconBtn =
  "flex items-center justify-center w-6 h-6 rounded-md border-0 bg-transparent text-(color:--md-sys-color-text-secondary) cursor-pointer hover:text-(color:--md-sys-color-text) hover:bg-(color:--md-sys-color-surface-container-high) transition-colors";

// DeliverablesPanel — 右侧「会话产物」视图（Codex 式工作区收尾）：
// 展示本次会话交付的全部文件（去重、最新在前），点击预览，悬停提供
// 外部打开 / 定位 / 复制路径 / 沉淀成本库；预览内编辑过的文件显示「已更新」徽标。
// v4.28 B1 版本时间线：vN 次数徽标可点，展开该文件的逐版本列表（预览/恢复，
// 数据为挂载时自拉的 JournalList(200)，见 VersionTimeline）。
// v3「星枢」面板语言：v3-panel-head 细条头部 + 低边框 hover 高亮行。
export const DeliverablesPanel = memo(function DeliverablesPanel({
  items,
  sessionPath,
  onOpenFile,
  onLocateSource,
  onRevealInTree,
  freshPaths,
}: {
  items: SessionDeliverable[];
  /** 当前会话路径（v4.24 C1：非空时拉取权威产物登记表）。 */
  sessionPath?: string;
  onOpenFile: (path: string) => void;
  onLocateSource?: (turn: number) => void;
  /** 树中定位（v4.25 A3）：产物行小按钮 → 切到文件 tab 并在文件树中
   *  展开父链 + 滚动 + 高亮该文件（接线由 App/sidebarRegistry 完成）。 */
  onRevealInTree?: (rel: string) => void;
  /** v4.30 产物自动置前：本会话内新出现的产物路径（App 侧 diff 出，激活
   *  产物 tab 即清零）。命中路径的行显示「新」徽标 + 短暂高亮（Devin
   *  Auto-open 式：产物生成自动提示，不打断对话）。 */
  freshPaths?: string[];
}) {
  const openFilePreview = usePreviewStore((s) => s.openFilePreview);
  const updatedAt = useUpdatedFilesStore((s) => s.updatedAt);
  const toast = useToast();

  // ── v4.24 C1 权威产物登记表（后端从事件日志折叠，前端只读）──
  // 覆盖写类 8 种 + 生成/导出类 3 种工具的落盘登记，补正文扩展名白名单
  // 启发式漏登（非常规扩展名 / format_convert / chart_gen / diagram_gen）。
  // 无 sessionPath 或后端 Available=false（legacy 会话无事件日志）时整节收起。
  const [registry, setRegistry] = useState<DeliverableRegistryView | null>(null);
  const [registryOpen, setRegistryOpen] = useState(false);
  useEffect(() => {
    if (!sessionPath) {
      setRegistry(null);
      return;
    }
    let cancelled = false;
    void app
      .DeliverableRegistry(sessionPath)
      .then((v) => { if (!cancelled) setRegistry(v ?? null); })
      .catch(() => { if (!cancelled) setRegistry(null); });
    return () => { cancelled = true };
  }, [sessionPath]);
  const registryEntries = registry?.available ? registry.entries : [];
  const registryTotal = registry?.total ?? 0;
  // v4.6 失败回 Plan：逐证据卡内联展示复核结论（不再只弹 toast 一闪而过）
  const [verdicts, setVerdicts] = useState<Record<string, VerdictView>>({});

  const open = onOpenFile ?? openFilePreview;
  const copyPath = useCallback(async (path: string) => {
    try {
      await navigator.clipboard.writeText(path);
      toast.show("已复制文件路径", "info");
    } catch {
      toast.show("复制失败：剪贴板不可用", "warn");
    }
  }, [toast]);

  // 沉淀到成本库：把测算/表格产物一键转为 cost_save 指令进入输入框，
  // agent 读取文件后将单价明细写回成本库（来源标注该文件，同名覆盖）。
  const depositToCost = useCallback((path: string) => {
    const name = baseName(path);
    const prompt = `请读取 [${name}](${path})，用 cost_save 把其中的单价明细沉淀到成本库：逐行提取科目/单位/单价/规格，来源标注该文件；同名条目覆盖更新，完成后汇报新增/更新条数。`;
    useComposerInsertStore.getState().requestText(prompt);
    toast.show(`已把沉淀指令插入输入框，可编辑后发送`, "info");
  }, [toast]);

  // 最新在前
  const list = [...items].reverse();

  // 复制全部路径：一次拿到本次会话全部交付物清单，便于归档或继续引用。
  const copyAllPaths = useCallback(async () => {
    const paths = list.map((d) => d.path);
    try {
      await navigator.clipboard.writeText(paths.join("\n"));
      toast.show(`已复制 ${paths.length} 个文件路径`, "info");
    } catch {
      toast.show("复制失败：剪贴板不可用", "warn");
    }
  }, [list, toast]);

  // ── v4.28 B1 文件版本时间线：挂载时自拉一次 GaeaJournalList(200)（证据链与
  // 回滚同源的自动快照），按 target 聚合成「路径 → 版本记录」索引；失败静默
  // 降级空态，不引入轮询。恢复成功后主动重拉一次，把恢复动作生成的新证据卡
  // 纳入时间线（恢复=新增版本不丢历史）。
  const [journal, setJournal] = useState<JournalChangeRecord[] | null>(null);
  const loadJournal = useCallback(async () => {
    try {
      const recs = await app.GaeaJournalList(200);
      setJournal(recs ?? []);
    } catch {
      setJournal([]); // 静默：时间线降级为空态，不打扰产物主流程
    }
  }, []);
  useEffect(() => { void loadJournal(); }, [loadJournal]);

  // 路径 → 版本记录索引：target 反斜杠归一后聚合、at 倒序、只留有 baselinePath 的卡
  // （无基线快照不能预览/恢复，不进时间线）。
  const groupedVersions = useMemo(() => groupVersionsByPath(journal ?? []), [journal]);

  // 当前展开时间线的产物路径（归一化 key；再次点击同一路径收起）
  const [timelinePath, setTimelinePath] = useState<string | null>(null);

  // 恢复到所选版本：RollbackRecord 按证据卡把基线快照写回目标（DeliverablesPanel
  // 证据卡回滚同款先例）；恢复动作本身也会生成新证据卡，成功后重拉 JournalList。
  const restoreVersion = useCallback(async (r: JournalChangeRecord) => {
    try {
      await app.RollbackRecord(r.id);
      toast.show(`已恢复 ${r.target} 到 ${versionLabel(r)} 版本；当前内容成为新版本`, "info");
      void loadJournal();
    } catch (e) {
      toast.show(`恢复失败：${e instanceof Error ? e.message : String(e)}`, "warn");
    }
  }, [loadJournal, toast]);

  // ── v4.1 证据链：最近证据卡（「证据」入口，复用产物面板挂载点）──
  const [evidenceOpen, setEvidenceOpen] = useState(false);
  const [evidence, setEvidence] = useState<JournalChangeRecord[] | null>(null);
  useEffect(() => {
    if (!evidenceOpen) return;
    let cancelled = false;
    void app
      .GaeaJournalList(15)
      .then((recs) => { if (!cancelled) setEvidence(recs ?? []); })
      .catch(() => { if (!cancelled) setEvidence([]); });
    return () => { cancelled = true };
  }, [evidenceOpen]);

  const fmtEvidenceTime = (at: number): string => {
    if (!at) return "—";
    const diff = Date.now() - at;
    const min = Math.floor(diff / 60000);
    if (min < 1) return "刚刚";
    if (min < 60) return `${min} 分钟前`;
    const h = Math.floor(min / 60);
    if (h < 24) return `${h} 小时前`;
    return new Date(at).toLocaleString();
  };

  // 登记表时间：unix 秒 → 本地时间（跨会话仍是绝对时间，不随会话漂移）。
  const fmtRegistryTime = (at: number): string => {
    if (!at) return "—";
    const d = new Date(at * 1000);
    if (Number.isNaN(d.getTime())) return "—";
    return d.toLocaleTimeString("zh-CN", { hour12: false, month: "numeric", day: "numeric" });
  };

  // ── v4.8 Verifier 产品化：证据卡「三步展开」──
  // 卡面（tool 徽标+target+相对时间+复核/回滚+verdict 内联）→ 展开第 1 层
  // 「声明↔实况」diff（opsJson × GaeaPreview 现取）→ 第 2 层操作回放时间线。
  // diff 按卡 id 缓存；预览不可用降级「仅声明回放」；旧卡无 opsJson 回退
  // beforeSummary 文本块。
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [diffs, setDiffs] = useState<Record<string, VerifyDiffRow[]>>({});
  const [diffStates, setDiffStates] = useState<Record<string, "loading" | "ok" | "none">>({});

  const toggleExpand = useCallback((r: JournalChangeRecord) => {
    setExpandedId((cur) => (cur === r.id ? null : r.id));
  }, []);

  // 展开 xlsx_apply 且 opsJson 可解析的卡时：现取 GaeaPreview 实况并比对。
  useEffect(() => {
    if (!expandedId) return;
    const r = (evidence ?? []).find((x) => x.id === expandedId);
    if (!r) return;
    const ops = r.tool === "xlsx_apply" ? parseOps(r.opsJson) : [];
    if (!ops.some(isClaimableOp)) {
      setDiffStates((p) => (p[expandedId] === "none" ? p : { ...p, [expandedId]: "none" }));
      return;
    }
    const st = diffStates[expandedId];
    if (st === "ok" || st === "none") return;
    let cancelled = false;
    if (st !== "loading") setDiffStates((p) => ({ ...p, [expandedId]: "loading" }));
    void app
      .Preview(r.target)
      .then((res) => {
        if (cancelled) return;
        const index = res.kind === "xlsx" ? buildCellIndex(res.body) : {};
        setDiffs((p) => ({ ...p, [expandedId]: buildVerifyDiff(ops, index) }));
        setDiffStates((p) => ({ ...p, [expandedId]: "ok" }));
      })
      .catch(() => { if (!cancelled) setDiffStates((p) => ({ ...p, [expandedId]: "none" })); });
    return () => { cancelled = true; };
  }, [expandedId, diffStates, evidence]);

  const verifyRecord = useCallback(async (r: JournalChangeRecord) => {
    try {
      const v = await app.VerifyRecord(r.id);
      setVerdicts((prev) => ({ ...prev, [r.id]: v }));
      const label = v.status === "verified" ? "复核通过" : v.status === "warned" ? "复核警告" : "复核未通过";
      toast.show(`${label}：${v.note ?? ""}（A:${v.channelA ?? "n/a"} / B:${v.channelB ?? "n/a"}）`, v.status === "failed" ? "warn" : "info");
    } catch (e) {
      toast.show(`复核失败：${e instanceof Error ? e.message : String(e)}`, "warn");
    }
  }, [toast]);

  // v4.6 失败回 Plan：xlsx_apply 复核未通过 → 一键回到办公板块重新规划
  const replanFailed = useCallback((r: JournalChangeRecord) => {
    emitFrontendEvent(FRONTEND_EVENTS.NAVIGATE, { page: "office" });
    toast.show(`已打开办公面板——可对 ${r.target} 重新规划后再应用`, "info");
  }, [toast]);

  const rollbackRecord = useCallback(async (r: JournalChangeRecord) => {
    try {
      await app.RollbackRecord(r.id);
      toast.show(`已回滚 ${r.target}（基线快照恢复）`, "info");
    } catch (e) {
      toast.show(`回滚失败：${e instanceof Error ? e.message : String(e)}`, "warn");
    }
  }, [toast]);

  // 打包下载：把本次会话全部交付文件打成一个 zip（对标 Kimi 工作空间 /
  // WorkBuddy 会话产物打包），完成后在文件管理器中定位 zip。
  const [zipping, setZipping] = useState(false);
  const zipDeliverables = useCallback(async () => {
    if (zipping || list.length === 0) return;
    setZipping(true);
    try {
      const r = await app.ZipDeliverables(list.map((d) => d.path));
      toast.show(`已打包 ${r.entries} 个文件（${(r.bytes / 1024).toFixed(1)} KB）`, "info");
      void app.RevealWorkspacePath(r.path).catch(() => {});
    } catch (e) {
      toast.show(`打包失败：${e instanceof Error ? e.message : String(e)}`, "warn");
    } finally {
      setZipping(false);
    }
  }, [zipping, list, toast]);

  return (
    <div className="flex flex-col h-full min-h-0 text-xs" style={{ color: "var(--md-sys-color-text-secondary)" }}>
      {/* v3 细条头部：标题 + 计数徽标 + 复制全部 */}
      <div className="v3-panel-head">
        <FileText size={13} aria-hidden style={{ color: "var(--gaea-glow)" }} />
        <span className="v3-panel-title">会话产物</span>
        {items.length > 0 && (
          <span
            className="rounded-full px-1.5 py-px text-[10px] font-mono"
            style={{
              background: "color-mix(in srgb, var(--gaea-glow) 10%, transparent)",
              color: "var(--gaea-glow)",
              border: "1px solid color-mix(in srgb, var(--gaea-glow) 26%, transparent)",
            }}
          >
            {items.length}
          </span>
        )}
        <span className="v3-panel-spacer" />
        {items.length > 0 && (
          <>
            <button
              type="button"
              className={iconBtn}
              onClick={() => void zipDeliverables()}
              disabled={zipping}
              title="打包下载：把本次会话全部交付文件打成一个 zip"
              aria-label="打包下载全部交付文件"
            >
              {zipping ? <Loader2 size={12} className="animate-spin" /> : <Archive size={12} />}
            </button>
            <button
              type="button"
              className={iconBtn}
              onClick={() => void copyAllPaths()}
              title="复制全部文件路径"
              aria-label="复制全部文件路径"
            >
              <ClipboardList size={12} />
            </button>
          </>
        )}
      </div>

      {items.length === 0 ? (
        <div
          className="flex flex-col items-center justify-center flex-1 gap-2 px-6 text-center"
          style={{ color: "var(--md-sys-color-text-secondary)" }}
        >
          <Paperclip size={24} aria-hidden className="opacity-40" />
          <span className="text-[11px] leading-relaxed">
            本轮会话暂无交付文件
            <br />
            生成/保存文件后会出现在这里
          </span>
        </div>
      ) : (
        <div className="flex-1 min-h-0 overflow-y-auto p-2 flex flex-col gap-1.5">
          {list.map(({ path, turn, versions }) => {
            const ext = extOf(path);
            const updated = updatedAt[path] != null;
            const rev = versions && versions > 1 ? versions : undefined;
            // v4.30 产物自动置前：本会话新产物行加「新」徽标 + 高亮闪烁提示
            //（Devin Auto-open 式；激活产物 tab 后 App 清零 freshPaths，徽标消退）。
            const fresh = freshPaths?.includes(path) ?? false;
            // v4.28 B1：时间线展开 key 用归一化路径（与 JournalList target 对齐）
            const normPath = normalizeVersionPath(path);
            const timelineOpen = timelinePath === normPath;
            return (
              <Fragment key={path}>
                <div
                  className="group flex items-center gap-2 px-2 py-1.5 rounded-lg transition-colors duration-150 hover:bg-(color:--md-sys-color-surface-container-high)"
                  data-fresh={fresh ? "true" : undefined}
                  style={fresh ? { background: "color-mix(in srgb, var(--md-sys-color-primary) 10%, transparent)" } : undefined}
                >
                  <span
                    className="shrink-0 w-8 h-8 rounded-md flex items-center justify-center overflow-hidden"
                    style={{
                      background: "color-mix(in srgb, var(--gaea-glow) 10%, transparent)",
                      color: "var(--gaea-glow)",
                      border: "1px solid color-mix(in srgb, var(--md-sys-color-outline-variant) 60%, transparent)",
                    }}
                  >
                    <FileThumb path={path} ext={ext} imgClassName="w-8 h-8 object-cover rounded-md" />
                  </span>
                  <button
                    type="button"
                    onClick={() => open(path)}
                    title={`点击预览 ${path}`}
                    className="min-w-0 flex-1 text-left cursor-pointer"
                  >
                    <span className="flex items-center gap-1">
                      <span className="truncate text-[12px] font-medium leading-tight" style={{ color: "var(--md-sys-color-text)" }}>
                        {baseName(path)}
                      </span>
                      {fresh && (
                        <span
                          className="shrink-0 inline-flex items-center gap-0.5 rounded-full px-1 py-px text-[9px] leading-none"
                          style={{
                            color: "var(--md-sys-color-primary)",
                            background: "color-mix(in srgb, var(--md-sys-color-primary) 14%, transparent)",
                            border: "1px solid color-mix(in srgb, var(--md-sys-color-primary) 34%, transparent)",
                          }}
                        >
                          <Sparkles size={8} aria-hidden />
                          新
                        </span>
                      )}
                      {updated && (
                        <span
                          className="shrink-0 inline-flex items-center gap-0.5 rounded-full px-1 py-px text-[9px] leading-none"
                          style={{
                            color: "var(--md-sys-color-success)",
                            background: "color-mix(in srgb, var(--md-sys-color-success) 12%, transparent)",
                            border: "1px solid color-mix(in srgb, var(--md-sys-color-success) 32%, transparent)",
                          }}
                        >
                          <FileText size={8} aria-hidden />
                          已更新
                        </span>
                      )}
                    </span>
                    <span className="block truncate text-[10px] font-mono leading-tight transition-opacity duration-150 group-hover:opacity-100 opacity-0" style={{ color: "var(--md-sys-color-text-secondary)" }}>
                      {path}
                    </span>
                  </button>
                  {/* v4.28 B1：vN 次数徽标改为可点按钮——点开内联版本时间线
                      （逐版本列表 + 预览 + 恢复）；作为名称按钮的兄弟节点避免
                      button 嵌套。展开时徽标加浓提示当前处于时间线视图。 */}
                  {rev && (
                    <button
                      type="button"
                      onClick={() => setTimelinePath((cur) => (cur === normPath ? null : normPath))}
                      aria-expanded={timelineOpen}
                      title={`会话内更新了 ${rev} 次（产物版本时间线）`}
                      aria-label={`查看 ${baseName(path)} 的版本时间线`}
                      className="shrink-0 inline-flex cursor-pointer items-center gap-0.5 rounded-full px-1.5 py-px text-[9px] leading-none font-mono transition-colors"
                      style={{
                        color: "var(--md-sys-color-primary)",
                        background: timelineOpen
                          ? "color-mix(in srgb, var(--md-sys-color-primary) 24%, transparent)"
                          : "color-mix(in srgb, var(--md-sys-color-primary) 12%, transparent)",
                        border: "1px solid color-mix(in srgb, var(--md-sys-color-primary) 32%, transparent)",
                      }}
                    >
                      <Rollback size={8} aria-hidden />
                      v{rev}
                    </button>
                  )}
                  <div className="shrink-0 flex items-center gap-0.5 opacity-0 group-hover:opacity-100 group-focus-within:opacity-100 transition-opacity">
                  {turn != null && onLocateSource && (
                    <button
                      type="button"
                      className={iconBtn}
                      onClick={() => onLocateSource(turn)}
                      title="跳转到生成它的消息"
                      aria-label="跳转到生成它的消息"
                    >
                      <MessageSquare size={12} />
                    </button>
                  )}
                  {onRevealInTree && (
                    <button
                      type="button"
                      className={iconBtn}
                      onClick={() => onRevealInTree(path)}
                      title="树中定位：在文件树中展开并高亮该文件"
                      aria-label="树中定位"
                    >
                      <ListTree size={12} />
                    </button>
                  )}
                  <button
                    type="button"
                    className={iconBtn}
                    onClick={() => void copyPath(path)}
                    title="复制文件路径"
                    aria-label="复制文件路径"
                  >
                    <Copy size={12} />
                  </button>
                  <button
                    type="button"
                    className={iconBtn}
                    onClick={() => void app.OpenWorkspacePath(path).catch(() => {})}
                    title="在外部程序中打开"
                    aria-label="在外部程序中打开"
                  >
                    <ExternalLink size={12} />
                  </button>
                  <button
                    type="button"
                    className={iconBtn}
                    onClick={() => void app.RevealWorkspacePath(path).catch(() => {})}
                    title="在文件管理器中定位"
                    aria-label="在文件管理器中定位"
                  >
                    <FolderTree size={12} />
                  </button>
                  {SPREADSHEET_EXT_RE.test(ext) && (
                    <button
                      type="button"
                      className={iconBtn}
                      onClick={() => depositToCost(path)}
                      title="沉淀到成本库：把单价明细用 cost_save 写回成本库"
                      aria-label="沉淀到成本库"
                      style={{ color: "var(--md-sys-color-warning)" }}
                    >
                      <Coins size={12} />
                    </button>
                  )}
                </div>
              </div>
              {/* v4.28 B1 内联版本时间线：徽标点开后在该产物行下方展开。
                  records 三态——journal=null 加载态 / 空数组空态 / 有记录列表；
                  预览/恢复经回调注入（open=onOpenFile ?? openFilePreview，
                  restoreVersion=RollbackRecord + toast + 重拉时间线）。 */}
              {timelineOpen && (
                <VersionTimeline
                  path={path}
                  records={journal === null ? null : (groupedVersions.get(normPath) ?? [])}
                  onPreview={open}
                  onRestore={restoreVersion}
                />
              )}
            </Fragment>
          );
        })}
        </div>
      )}

      {/* v4.24 C1 权威产物登记表：后端从事件日志折叠的写类/生成类落盘登记，
          只读展示（含启发式漏登的非常规扩展名产物）。无会话路径或后端
          Available=false（legacy 会话）整节收起。 */}
      {registry !== null && registryEntries.length > 0 && (
        <div className="shrink-0 border-t border-(color:--md-sys-color-outline-variant)" data-testid="deliverable-registry">
          <button
            type="button"
            className="w-full flex items-center gap-2 px-3 py-2 text-[11px] cursor-pointer hover:bg-(color:--md-sys-color-surface-container-high)"
            onClick={() => setRegistryOpen((v) => !v)}
            aria-expanded={registryOpen}
          >
            <Table size={13} aria-hidden style={{ color: "var(--md-sys-color-warning)" }} />
            <span className="font-medium" style={{ color: "var(--md-sys-color-text)" }}>权威产物登记</span>
            <span className="truncate text-[10px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
              {registryTotal > registryEntries.length
                ? `${registryEntries.length}/${registryTotal} 条（最近 ${registryEntries.length} 条）`
                : `${registryTotal} 条落盘登记`}
            </span>
            <span className="v3-panel-spacer" />
            <span className="text-[10px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>{registryOpen ? "收起" : "展开"}</span>
          </button>
          {registryOpen && (
            <div className="max-h-44 overflow-y-auto px-2 pb-2 flex flex-col gap-1">
              {registryEntries.map((e) => (
                <div
                  key={e.path}
                  data-testid="deliverable-registry-row"
                  className="flex items-center gap-1.5 px-1.5 py-1 rounded-md bg-(color:--md-sys-color-surface-container) border border-(color:--md-sys-color-outline-variant)"
                >
                  <span className="shrink-0 font-mono text-[9px] px-1 py-px rounded" style={{
                    color: "var(--md-sys-color-warning)",
                    background: "color-mix(in srgb, var(--md-sys-color-warning) 12%, transparent)",
                  }} title={`落盘工具 ${e.tool}`}>
                    {e.tool}
                  </span>
                  <button
                    type="button"
                    className="min-w-0 flex-1 text-left cursor-pointer truncate font-mono text-[10px]"
                    style={{ color: "var(--md-sys-color-text)" }}
                    title={`点击预览 ${e.path}`}
                    onClick={() => open(e.path)}
                  >
                    {e.path}
                  </button>
                  <span className="shrink-0 text-[9px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
                    {e.turn > 0 ? `第 ${e.turn} 轮` : "轮外"}
                  </span>
                  {e.touches > 1 && (
                    <span className="shrink-0 text-[9px] font-mono" style={{ color: "var(--md-sys-color-text-secondary)" }}>
                      ×{e.touches}
                    </span>
                  )}
                  <span className="shrink-0 text-[9px] font-mono tabular-nums" style={{ color: "var(--md-sys-color-text-secondary)" }}>
                    {fmtRegistryTime(e.updatedAt)}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* v4.1 证据链入口：最近变更证据卡（Apply→Verify→Journal 的 Journal 面） */}
      <div className="shrink-0 border-t border-(color:--md-sys-color-outline-variant)">
        <button
          type="button"
          className="w-full flex items-center gap-2 px-3 py-2 text-[11px] cursor-pointer hover:bg-(color:--md-sys-color-surface-container-high)"
          onClick={() => setEvidenceOpen((v) => !v)}
          aria-expanded={evidenceOpen}
        >
          <Shield size={13} aria-hidden style={{ color: "var(--md-sys-color-primary)" }} />
          <span className="font-medium" style={{ color: "var(--md-sys-color-text)" }}>证据链</span>
          <span className="truncate text-[10px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
            {evidence ? `${evidence.length} 条变更证据卡` : "最近变更审计"}
          </span>
          <span className="v3-panel-spacer" />
          <span className="text-[10px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>{evidenceOpen ? "收起" : "展开"}</span>
        </button>
        {evidenceOpen && (
          <div className="max-h-44 overflow-y-auto px-2 pb-2 flex flex-col gap-1">
            {evidence && evidence.length === 0 ? (
              <div className="px-2 py-2 text-[10px] text-center" style={{ color: "var(--md-sys-color-text-secondary)" }}>
                暂无证据卡——AI 改文件后这里会记录变更原文摘要
              </div>
            ) : (
              (evidence ?? []).map((r) => {
                const v = verdicts[r.id];
                const ops = r.tool === "xlsx_apply" ? parseOps(r.opsJson) : [];
                const claimable = ops.some(isClaimableOp);
                const hasOps = !!r.opsJson;
                const expanded = expandedId === r.id;
                const canRollback = !!r.baselinePath;
                const diffRows = diffs[r.id];
                const diffState = diffStates[r.id];
                return (
                <div key={r.id} className="flex flex-col gap-1 px-2 py-1.5 rounded-md bg-(color:--md-sys-color-surface-container) border border-(color:--md-sys-color-outline-variant)">
                  {/* 卡面：点击展开/收起（三步展开第 0 层）；带 opsJson 的卡加「可复核明细」小徽标 */}
                  <div
                    role="button"
                    tabIndex={0}
                    aria-expanded={expanded}
                    aria-label={expanded ? `收起 ${r.tool} ${r.target} 证据详情` : `展开 ${r.tool} ${r.target} 证据详情`}
                    onClick={() => toggleExpand(r)}
                    onKeyDown={(e) => {
                      if (e.key === "Enter" || e.key === " ") {
                        e.preventDefault();
                        toggleExpand(r);
                      }
                    }}
                    className="flex items-center gap-2 cursor-pointer outline-none focus-visible:ring-1 focus-visible:ring-(color:--md-sys-color-primary)"
                    title={hasOps ? "点击展开「声明↔实况」diff 与操作回放" : "点击展开证据详情"}
                  >
                    <span className="shrink-0 font-mono text-[9px] px-1 py-px rounded" style={{
                      color: "var(--md-sys-color-primary)",
                      background: "color-mix(in srgb, var(--md-sys-color-primary) 12%, transparent)",
                    }}>
                      {r.tool}
                    </span>
                    {hasOps && (
                      <span
                        className="shrink-0 font-mono text-[9px] px-1 py-px rounded"
                        style={{
                          color: "var(--md-sys-color-warning)",
                          background: "color-mix(in srgb, var(--md-sys-color-warning) 12%, transparent)",
                        }}
                        title="该卡携带 AI 原始操作集，可复核明细"
                      >
                        可复核明细
                      </span>
                    )}
                    <span className="min-w-0 flex-1 truncate text-[10px] font-mono" style={{ color: "var(--md-sys-color-text)" }} title={r.target}>
                      {r.target}
                    </span>
                    <span className="shrink-0 text-[9px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
                      {fmtEvidenceTime(r.at)}
                    </span>
                    <span className="shrink-0 flex items-center gap-1">
                      <button
                        type="button"
                        className={iconBtn}
                        onClick={(e) => { e.stopPropagation(); void verifyRecord(r); }}
                        title="双通道复核（结构/引用完整性 + 视觉健全性）"
                        aria-label="复核该证据卡"
                      >
                        <Shield size={11} />
                      </button>
                      <button
                        type="button"
                        className={iconBtn + (canRollback ? "" : " disabled:opacity-40 disabled:cursor-not-allowed")}
                        disabled={!canRollback}
                        onClick={(e) => { e.stopPropagation(); void rollbackRecord(r); }}
                        title={canRollback ? "用基线快照回滚（目标被手工修改时拒绝）" : "无基线快照，无法回滚"}
                        aria-label={canRollback ? "回滚该证据卡" : "无基线快照，无法回滚"}
                      >
                        <Rollback size={11} />
                      </button>
                    </span>
                    <span aria-hidden className="shrink-0 text-[9px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
                      {expanded ? "▾" : "▸"}
                    </span>
                  </div>
                  {/* 展开区：第 1 层「声明↔实况」diff → 第 2 层操作回放时间线 */}
                  {expanded && (
                    <div className="flex flex-col gap-1.5 pl-1">
                      {hasOps && ops.length > 0 ? (
                        <>
                          {claimable && diffState === "loading" && (
                            <div className="flex items-center gap-1 text-[9px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
                              <Loader2 size={9} className="animate-spin" />
                              正在读取实况预览…
                            </div>
                          )}
                          {claimable && diffState === "ok" && diffRows && diffRows.length > 0 && (
                            <div className="flex flex-col gap-0.5">
                              <span className="text-[9px] font-medium" style={{ color: "var(--md-sys-color-text-secondary)" }}>
                                声明 ↔ 实况（前端近似比对）
                              </span>
                              <div className="rounded-md border border-(color:--md-sys-color-outline-variant) overflow-hidden">
                                <table className="w-full text-[9px] font-mono border-collapse">
                                  <tbody>
                                    {diffRows.map((row, i) => (
                                      <tr key={`${row.sheet}!${row.cell}-${i}`} className="border-b border-(color:--md-sys-color-outline-variant) last:border-b-0">
                                        <td className="px-1.5 py-0.5 whitespace-nowrap align-top" style={{ color: "var(--md-sys-color-text-secondary)" }}>
                                          {row.sheet}!{row.cell}
                                        </td>
                                        <td className="px-1.5 py-0.5 line-through max-w-[120px] truncate align-top" style={{ color: "var(--md-sys-color-text-secondary)" }} title={row.claimed}>
                                          {row.claimed || "（空）"}
                                        </td>
                                        <td className="px-0.5 py-0.5 align-top" style={{ color: "var(--md-sys-color-primary)" }}>→</td>
                                        <td className="px-1.5 py-0.5 max-w-[160px] truncate align-top" style={{ color: row.ok === "mismatch" ? "var(--md-sys-color-destructive)" : "var(--md-sys-color-text)" }} title={row.actual}>
                                          {row.actual || "（空）"}
                                        </td>
                                        <td className="px-1.5 py-0.5 align-top whitespace-nowrap" style={{ color: row.ok === "match" ? "var(--md-sys-color-success)" : row.ok === "mismatch" ? "var(--md-sys-color-destructive)" : "var(--md-sys-color-text-secondary)" }}>
                                          {row.ok === "match" ? "✓" : row.ok === "mismatch" ? "✗" : "跳过"}
                                        </td>
                                      </tr>
                                    ))}
                                  </tbody>
                                </table>
                              </div>
                              <span className="text-[9px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
                                前端近似比对，权威结论以复核为准
                              </span>
                            </div>
                          )}
                          {claimable && diffState === "none" && (
                            <div className="text-[9px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
                              预览不可用，仅声明回放
                            </div>
                          )}
                          {claimable && diffState === "ok" && diffRows && diffRows.length === 0 && (
                            <div className="text-[9px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
                              无可比对单元格（声明不含单格写入）
                            </div>
                          )}
                          {/* 第 2 层：操作回放时间线（fill_range/transform 等批量 op 折叠为单行 + 计数徽标） */}
                          <div className="flex flex-col gap-0.5">
                            <span className="text-[9px] font-medium" style={{ color: "var(--md-sys-color-text-secondary)" }}>
                              操作回放
                            </span>
                            <ol className="flex flex-col gap-0.5">
                              {ops.map((op, i) => {
                                const count = opBatchCount(op);
                                return (
                                  <li key={i} className="flex items-center gap-1.5">
                                    <span className="shrink-0 font-mono text-[8px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>{i + 1}</span>
                                    <span className="shrink-0 font-mono text-[8px] px-1 py-px rounded" style={{
                                      color: "var(--md-sys-color-primary)",
                                      background: "color-mix(in srgb, var(--md-sys-color-primary) 12%, transparent)",
                                    }}>
                                      {op.type}
                                    </span>
                                    <span className="min-w-0 flex-1 truncate text-[9px]" style={{ color: "var(--md-sys-color-text)" }} title={describeOp(op)}>
                                      {describeOp(op)}
                                    </span>
                                    <span className="shrink-0 font-mono text-[8px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
                                      {opImpact(op)}
                                    </span>
                                    {count && (
                                      <span className="shrink-0 font-mono text-[8px] px-1 py-px rounded" style={{
                                        color: "var(--md-sys-color-warning)",
                                        background: "color-mix(in srgb, var(--md-sys-color-warning) 12%, transparent)",
                                      }}>
                                        {count}
                                      </span>
                                    )}
                                  </li>
                                );
                              })}
                            </ol>
                          </div>
                        </>
                      ) : (
                        <div className="text-[9px] font-mono leading-relaxed" style={{ color: "var(--md-sys-color-text-secondary)" }}>
                          {r.beforeSummary || "（无变更摘要）"}
                        </div>
                      )}
                    </div>
                  )}
                  {/* v4.6：内联复核结论——failed 卡常驻显示「回滚 + 重新规划」入口。
                      v4.16：通道 B 结果产品化——verdict 携带像素差异率时追加
                      「视觉复核」行（差异率 + 页数 + 查看产物按钮；产物目录是
                      绝对路径，OpenWorkspacePath 直接打开目录）。旧 verdict /
                      无通道 B（无 channelBRatio）不渲染该行，向后兼容。 */}
                  {v && (
                    <div className="flex flex-col gap-1">
                      <div className="flex flex-wrap items-center gap-1.5">
                        <span className="shrink-0 text-[9px] px-1 py-px rounded font-mono" style={{
                          color: v.status === "verified"
                            ? "var(--md-sys-color-success)"
                            : v.status === "warned"
                              ? "var(--md-sys-color-warning)"
                              : "var(--md-sys-color-destructive)",
                          background: "color-mix(in srgb, currentColor 10%, transparent)",
                        }}>
                          {v.status === "verified" ? "复核通过" : v.status === "warned" ? "复核警告" : "复核未通过"}
                        </span>
                        <span className="min-w-0 flex-1 text-[9px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
                          {v.note ?? `${v.channelA ?? ""} / ${v.channelB ?? ""}`}
                        </span>
                        {v.status === "failed" && r.tool === "xlsx_apply" && (
                          <button
                            type="button"
                            className={iconBtn}
                            onClick={() => replanFailed(r)}
                            title="回办公面板重新规划后再应用（失败回 Plan）"
                            aria-label="重新规划"
                          >
                            <ClipboardList size={11} />
                          </button>
                        )}
                      </div>
                      {typeof v.channelBRatio === "number" && (
                        <div className="flex flex-wrap items-center gap-1.5 pl-1">
                          <span className="text-[9px] font-mono" style={{ color: "var(--md-sys-color-text-secondary)" }}>
                            视觉复核：像素差异率 {(v.channelBRatio * 100).toFixed(1)}% · {v.channelBPages ?? 0} 页
                          </span>
                          {v.channelBArtifacts && (
                            <button
                              type="button"
                              className={iconBtn}
                              onClick={() => void app.OpenWorkspacePath(v.channelBArtifacts as string).catch(() => {})}
                              title="查看复核产物（before/after PDF + 逐页 PNG）"
                              aria-label="查看复核产物"
                            >
                              <FolderTree size={11} />
                            </button>
                          )}
                        </div>
                      )}
                    </div>
                  )}
                </div>
                );
              })
            )}
          </div>
        )}
      </div>
    </div>
  );
});
