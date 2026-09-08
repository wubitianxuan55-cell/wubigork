import { memo, useCallback, useEffect, useMemo, useState } from "react";
import { Archive, ClipboardList, FileText, Loader2, Paperclip } from "../icons";
import { app } from "../lib/bridge";
import type { DeliverableRegistryView, JournalChangeRecord, VerdictView, VerifyDiffRow } from "../lib/types";
import {
  groupVersionsByPath,
  versionLabel,
} from "../lib/versionTimeline";
import { loadDeliverableAutoOpen, saveDeliverableAutoOpen } from "../lib/deliverablePrefs";
import {
  acceptanceOf,
  acceptanceSummary,
  loadAcceptanceMap,
  saveAcceptanceMap,
  setAcceptance,
} from "../lib/deliverableStatus";
import type { DeliverableAcceptance, DeliverableStatusMap } from "../lib/deliverableStatus";
import {
  buildCellIndex,
  buildVerifyDiff,
  isClaimableOp,
  parseOps,
} from "../lib/verifyDiff";
import { useComposerInsertStore, usePreviewStore, useUpdatedFilesStore } from "../lib/store";
import { useT } from "../lib/i18n";
import { FRONTEND_EVENTS, emitFrontendEvent } from "../../events";
import { useToast } from "./Toast";
import { DeliverableRow } from "./deliverables/DeliverableRow";
import { EvidenceSection } from "./deliverables/EvidenceSection";
import { RegistrySection } from "./deliverables/RegistrySection";
import { baseName, iconBtn } from "./deliverables/shared";

export interface SessionDeliverable {
  path: string;
  sourceId: string;
  turn?: number;
  /** 同一文件在会话内被提及/更新的次数（≥1）；>1 显示版本徽标与步进器。 */
  versions?: number;
}

// DeliverablesPanel — 右侧「会话产物」视图（Codex 式工作区收尾）：
// 展示本次会话交付的全部文件（去重、最新在前），点击行经注入回调打开
// （pane 语义下 = 右栏文件 tab，与资源管理器同 tab 条），悬停提供外部打开 /
// 定位 / 复制路径 / 沉淀成本库；预览内编辑过的文件显示「已更新」徽标。
// v4.28 B1 版本时间线：vN 次数徽标可点，展开该文件的逐版本列表（预览/恢复，
// 数据为挂载时自拉的 JournalList(200)，见 VersionTimeline）。
// v4.31 A1 单版本入口：versions≤1 但有 journal 快照的产物同样渲染「版本」入口
// 徽标（收 v4.28 欠账「B1 单版本无入口」）；无快照记录不渲染，保持空态。
// v4.32 线B：头部「自动弹出」胶囊（默认关 opt-in，对标 BrowserPanel 同款交互，
// 键 gaea.deliverableAutoOpen）——新产物出现时 App 自动切到本面板；触发接线在
// App（shouldAutoOpenDeliverables），面板只负责偏好读写。单版本徽标 title 细化
// 为带快照数（收 v4.31 欠账「静态文案」）。
// v3「星枢」面板语言：v3-panel-head 细条头部 + 低边框 hover 高亮行。
// A1 交付验收闭环：每行验收徽标（绿「已验收」/警示「要求修改」；open 为缺省态
// 不显示徽标，视觉最安静）+ 悬停标记操作（标记已验收/要求修改/重新查看）；
// 头部「已验收 n/m」汇总。数据层 lib/deliverableStatus.ts，持久化键
// gaea.deliverableAcceptance.v1（面板级 map，同会话所有行共享）；versionAt 取
// 该路径登记表 updatedAt，登记表前进时 acceptanceOf 自动重置回待查看
// （新版本不看旧结论）。
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
  const t = useT();

  // ── v4.32 线B「自动弹出」偏好（默认关 opt-in）：新产物出现时 App 自动切到
  // 本面板；开关 UI 在头部胶囊，持久化键 gaea.deliverableAutoOpen。新产物检测
  // 与切换时机在 App（shouldAutoOpenDeliverables），面板只负责读写偏好。
  const [autoOpen, setAutoOpen] = useState(() => loadDeliverableAutoOpen());
  const toggleAutoOpen = useCallback(() => {
    setAutoOpen((prev) => {
      const next = !prev; // 先算 next 再落盘（BrowserPanel 同款），避免双调重复写
      saveDeliverableAutoOpen(next);
      return next;
    });
  }, []);

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
  // A1 验收版 useMemo 化：稳定引用供下方 registryUpdatedAt 索引依赖
  // （原为每次渲染新数组，行为不变）。
  const registryEntries = useMemo(
    () => (registry?.available ? registry.entries : []),
    [registry],
  );
  const registryTotal = registry?.total ?? 0;

  // ── A1 交付验收闭环：会话键复用面板拉登记表的 sessionPath prop；prop 缺省
  // （未保存草稿 / 旧入口）时与 lib/deliverablesTurn.ts 同式回退
  // ListSessions().find(current)?.path；两者都拿不到则整节功能收起（验收必须
  // 挂会话，避免同相对路径跨会话串状态）。
  const [accSessionPath, setAccSessionPath] = useState<string | undefined>(sessionPath);
  useEffect(() => {
    if (sessionPath) {
      setAccSessionPath(sessionPath);
      return;
    }
    let cancelled = false;
    void app
      .ListSessions()
      .then((sessions) => { if (!cancelled) setAccSessionPath(sessions?.find((s) => s.current)?.path); })
      .catch(() => { if (!cancelled) setAccSessionPath(undefined); });
    return () => { cancelled = true };
  }, [sessionPath]);

  // 验收 map：面板级一份（同会话所有行共享），初始化从 localStorage 读入，
  // 标记即写（setAcceptance + saveAcceptanceMap，薄 IO 壳异常静默）。
  const [acceptMap, setAcceptMap] = useState<DeliverableStatusMap>(() => loadAcceptanceMap());

  // 路径 → 登记表 updatedAt（unix 秒）：既是标记时的 versionAt（「标记时所见
  // 版本」），也是读状态时的 currentUpdatedAt（登记表前进 → acceptanceOf 自动
  // 重置回 open，新版本不看旧结论）；登记表没有的路径兜底 0。归一口径与
  // statusKeyOf 一致（反斜杠→/、小写）。
  const registryUpdatedAt = useMemo(() => {
    const m = new Map<string, number>();
    for (const e of registryEntries) m.set(e.path.replace(/\\/g, "/").toLowerCase(), e.updatedAt);
    return m;
  }, [registryEntries]);
  const updatedAtOf = useCallback(
    (path: string): number => registryUpdatedAt.get(path.replace(/\\/g, "/").toLowerCase()) ?? 0,
    [registryUpdatedAt],
  );

  const applyAcceptance = useCallback(
    (path: string, status: DeliverableAcceptance) => {
      if (!accSessionPath) return;
      const sp = accSessionPath;
      setAcceptMap((prev) => {
        // 先算 next 再落盘（autoOpen 胶囊同款），StrictMode 双调同值覆写无害
        const next = setAcceptance(prev, sp, path, status, Date.now(), updatedAtOf(path));
        saveAcceptanceMap(next);
        return next;
      });
    },
    [accSessionPath, updatedAtOf],
  );

  // 头部汇总：已验收 n / 总数 m（轻量文案 accept.statusConfirmed + 计数，
  // title 走 accept.title）；无会话键或无产物时不渲染。
  const acceptSummary = useMemo(() => {
    if (!accSessionPath || items.length === 0) return null;
    return acceptanceSummary(acceptMap, accSessionPath, items.map((d) => d.path), updatedAtOf);
  }, [acceptMap, accSessionPath, items, updatedAtOf]);
  // v4.6 失败回 Plan：逐证据卡内联展示复核结论（不再只弹 toast 一闪而过）
  const [verdicts, setVerdicts] = useState<Record<string, VerdictView>>({});

  const open = onOpenFile ?? openFilePreview;
  const copyPath = useCallback(async (path: string) => {
    try {
      await navigator.clipboard.writeText(path);
      toast.show(t("deliver.copyPathDone"), "info");
    } catch {
      toast.show(t("deliver.copyFail"), "warn");
    }
  }, [t, toast]);

  // 沉淀到成本库：把测算/表格产物一键转为 cost_save 指令进入输入框，
  // agent 读取文件后将单价明细写回成本库（来源标注该文件，同名覆盖）。
  const depositToCost = useCallback((path: string) => {
    const name = baseName(path);
    // 发给 LLM 的指令文本（非 UI 文案），不进字典
    const prompt = `请读取 [${name}](${path})，用 cost_save 把其中的单价明细沉淀到成本库：逐行提取科目/单位/单价/规格，来源标注该文件；同名条目覆盖更新，完成后汇报新增/更新条数。`;
    useComposerInsertStore.getState().requestText(prompt);
    toast.show(t("deliverPanel.depositDone"), "info");
  }, [t, toast]);

  // 最新在前
  const list = [...items].reverse();

  // 复制全部路径：一次拿到本次会话全部交付物清单，便于归档或继续引用。
  const copyAllPaths = useCallback(async () => {
    const paths = list.map((d) => d.path);
    try {
      await navigator.clipboard.writeText(paths.join("\n"));
      toast.show(t("deliverPanel.copyAllDone", { n: paths.length }), "info");
    } catch {
      toast.show(t("deliver.copyFail"), "warn");
    }
  }, [list, t, toast]);

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
      toast.show(t("deliverPanel.restoreDone", { path: r.target, label: versionLabel(r) }), "info");
      void loadJournal();
    } catch (e) {
      toast.show(t("deliverPanel.restoreFail", { msg: e instanceof Error ? e.message : String(e) }), "warn");
    }
  }, [loadJournal, t, toast]);

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
      const label = v.status === "verified" ? t("deliverPanel.verifyPass") : v.status === "warned" ? t("deliverPanel.verifyWarn") : t("deliverPanel.verifyFail");
      toast.show(t("deliverPanel.verifyToast", { label, note: v.note ?? "", a: v.channelA ?? "n/a", b: v.channelB ?? "n/a" }), v.status === "failed" ? "warn" : "info");
    } catch (e) {
      toast.show(t("deliverPanel.verifyFailToast", { msg: e instanceof Error ? e.message : String(e) }), "warn");
    }
  }, [t, toast]);

  // v4.6 失败回 Plan：xlsx_apply 复核未通过 → 一键回到办公板块重新规划。
  // NAVIGATE 的 page 是壳层板块 id（manifest），办公板块 id 为 gaea，
  // 不能沿用历史模块名 office（不在 MainLayout 导航白名单内）。
  const replanFailed = useCallback((r: JournalChangeRecord) => {
    emitFrontendEvent(FRONTEND_EVENTS.NAVIGATE, { page: "gaea" });
    toast.show(t("deliverPanel.replanToast", { path: r.target }), "info");
  }, [t, toast]);

  const rollbackRecord = useCallback(async (r: JournalChangeRecord) => {
    try {
      await app.RollbackRecord(r.id);
      toast.show(t("deliverPanel.rollbackDone", { path: r.target }), "info");
    } catch (e) {
      toast.show(t("deliverPanel.rollbackFail", { msg: e instanceof Error ? e.message : String(e) }), "warn");
    }
  }, [t, toast]);

  // 打包下载：把本次会话全部交付文件打成一个 zip（对标 Kimi 工作空间 /
  // WorkBuddy 会话产物打包），完成后在文件管理器中定位 zip。
  const [zipping, setZipping] = useState(false);
  const zipDeliverables = useCallback(async () => {
    if (zipping || list.length === 0) return;
    setZipping(true);
    try {
      const r = await app.ZipDeliverables(list.map((d) => d.path));
      toast.show(t("deliverPanel.zipDone", { n: r.entries, kb: (r.bytes / 1024).toFixed(1) }), "info");
      void app.RevealWorkspacePath(r.path).catch(() => {});
    } catch (e) {
      toast.show(t("deliverPanel.zipFail", { msg: e instanceof Error ? e.message : String(e) }), "warn");
    } finally {
      setZipping(false);
    }
  }, [zipping, list, t, toast]);

  return (
    <div className="flex flex-col h-full min-h-0 text-xs" style={{ color: "var(--md-sys-color-text-secondary)" }}>
      {/* v3 细条头部：标题 + 计数徽标 + 复制全部 */}
      <div className="v3-panel-head">
        <FileText size={13} aria-hidden style={{ color: "var(--gaea-glow)" }} />
        <span className="v3-panel-title">{t("deliverPanel.title")}</span>
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
        {/* A1 验收汇总：轻量「已验收 n/m」（accept.statusConfirmed + 计数，
            title=accept.title）；无会话键 / 无产物时不渲染，不与计数徽标挤位。 */}
        {acceptSummary && (
          <span
            data-testid="deliverable-acceptance-summary"
            className="shrink-0 text-[10px] font-mono tabular-nums"
            title={t("accept.title")}
            style={{ color: acceptSummary.confirmed > 0 ? "var(--md-sys-color-success)" : "var(--md-sys-color-text-secondary)" }}
          >
            {t("accept.statusConfirmed")} {acceptSummary.confirmed}/{acceptSummary.total}
          </span>
        )}
        <span className="v3-panel-spacer" />
        {/* v4.32 线B：自动弹出胶囊（默认关 opt-in；形状/交互对齐 BrowserPanel
            头部同款）——开=亮色点关、关=灰态点开；触发接线在 App，面板只管偏好。 */}
        <button
          type="button"
          data-testid="deliverable-auto-open-toggle"
          className="inline-flex shrink-0 cursor-pointer items-center gap-1 rounded-full px-1.5 py-px text-[10px] leading-none transition-colors"
          aria-pressed={autoOpen}
          title={autoOpen
            ? t("deliverPanel.autoOpenOn")
            : t("deliverPanel.autoOpenOff")}
          onClick={toggleAutoOpen}
          style={autoOpen
            ? {
                background: "color-mix(in srgb, var(--gaea-glow) 12%, transparent)",
                color: "var(--gaea-glow)",
                border: "1px solid color-mix(in srgb, var(--gaea-glow) 30%, transparent)",
              }
            : {
                background: "transparent",
                color: "var(--md-sys-color-text-secondary)",
                border: "1px solid var(--md-sys-color-outline-variant)",
              }}
        >
          <span
            className="inline-block h-1.5 w-1.5 rounded-full"
            style={{ background: autoOpen ? "var(--gaea-glow)" : "var(--md-sys-color-outline-variant)" }}
            aria-hidden
          />
          {t("deliverPanel.autoOpenLabel", { state: autoOpen ? t("deliverPanel.on") : t("deliverPanel.off") })}
        </button>
        {items.length > 0 && (
          <>
            <button
              type="button"
              className={iconBtn}
              onClick={() => void zipDeliverables()}
              disabled={zipping}
              title={t("deliverPanel.zipTitle")}
              aria-label={t("deliverPanel.zipAria")}
            >
              {zipping ? <Loader2 size={12} className="animate-spin" /> : <Archive size={12} />}
            </button>
            <button
              type="button"
              className={iconBtn}
              onClick={() => void copyAllPaths()}
              title={t("deliverPanel.copyAllPaths")}
              aria-label={t("deliverPanel.copyAllPaths")}
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
            {t("deliverPanel.empty")}
            <br />
            {t("deliverPanel.emptyHint")}
          </span>
        </div>
      ) : (
        <div className="flex-1 min-h-0 overflow-y-auto p-2 flex flex-col gap-1.5">
          {list.map((item) => (
            <DeliverableRow
              key={item.path}
              item={item}
              fresh={freshPaths?.includes(item.path) ?? false}
              updated={updatedAt[item.path] != null}
              accept={accSessionPath ? acceptanceOf(acceptMap, accSessionPath, item.path, updatedAtOf(item.path)) : "open"}
              accAvailable={!!accSessionPath}
              journal={journal}
              groupedVersions={groupedVersions}
              timelinePath={timelinePath}
              onOpen={open}
              onCopyPath={copyPath}
              onDeposit={depositToCost}
              onLocateSource={onLocateSource}
              onRevealInTree={onRevealInTree}
              onToggleTimeline={(norm) => setTimelinePath((cur) => (cur === norm ? null : norm))}
              onApplyAcceptance={applyAcceptance}
              onRestoreVersion={restoreVersion}
            />
          ))}
        </div>
      )}

      {/* v4.24 C1 权威产物登记表：后端从事件日志折叠的写类/生成类落盘登记，
          只读展示（含启发式漏登的非常规扩展名产物）。无会话路径或后端
          Available=false（legacy 会话）整节收起。 */}
      {registry !== null && registryEntries.length > 0 && (
        <RegistrySection
          open={registryOpen}
          onToggleOpen={() => setRegistryOpen((v) => !v)}
          total={registryTotal}
          entries={registryEntries}
          onOpen={open}
        />
      )}

      {/* v4.1 证据链入口：最近变更证据卡（Apply→Verify→Journal 的 Journal 面） */}
      <EvidenceSection
        open={evidenceOpen}
        onToggleOpen={() => setEvidenceOpen((v) => !v)}
        evidence={evidence}
        verdicts={verdicts}
        expandedId={expandedId}
        diffs={diffs}
        diffStates={diffStates}
        onToggleExpand={toggleExpand}
        onVerify={verifyRecord}
        onRollback={rollbackRecord}
        onReplan={replanFailed}
      />
    </div>
  );
});