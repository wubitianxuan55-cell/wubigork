// CostLibraryPage — 一级板块「造价数据库」（2026-11-25 推倒重建；同日库定规矩）。
//
// 定位：三级造价数据的浏览面——清单 → 定额 → 工料机。导入、解析、维护全部在
// 办公板块的「造价」运行模式里由模型完成（cost_library / cost_project_apply）；
// 这里只给人看，外加导出与删除。与方案库同一模式：库页只读，改库经产品。
//
// 维度：清单与定额按专业分（定额在专业内按章节），工料机按种类（类别 → 分类）、
// 地区、期数分。规范值只认库里的取值；没标注的归「未标注」一桶，状态行的「待整理」
// 指路去造价模式归类。
//
// 布局：左栏三级切换与两级上下文（专业 → 项目 / 专业 → 章节 / 类别 → 分类）→
// 中栏列表（筛选、当前位置、批量删除、下钻入口）→ 右栏详情（综合单价分析与
// 消耗量、含量行分析表、价格卡与各期价格），三级之间可以来回下钻。
// 四态分明：加载 / 空库（三步引导）/ 无匹配 / 读取失败。
import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import {
  AlertCircle, Coins, Copy, FolderOpen, Inbox, Loader2, RefreshCw, Search, Shield, Trash2, X,
} from "../gaea/icons";
import { app } from "../gaea/lib/bridge";
import { copyToClipboard } from "../gaea/lib/clipboard";
import { useBoardActive } from "../lib/boardActive";
import type { WorkcostBillProject, WorkcostLibraryFilter, WorkcostQuota, WorkcostResource } from "../gaea/lib/types";
import { BillDetail, DetailState, QuotaDetail, ResourceDetail } from "./cost-library/Details";
import { HealthDrawer, type HealthJump } from "./cost-library/HealthDrawer";
import { DeleteDialog, ExportMenu, type DeleteRequest, type ExportItem } from "./cost-library/Overlays";
import { BillTable, QuotaTable, ResourceTable } from "./cost-library/Tables";
import { TierBar, TierRail, type TierCounts } from "./cost-library/TierRail";
import {
  NONE, OUTSIDE, PENDING_PROMPT, SAMPLE_PROMPTS, TIERS, capRows, categoryTreeOf, chaptersOf, billSpecialties, dimLabel,
  emptyBillFilters, emptyQuotaFilters, emptyResourceFilters, divisionsOf, facetMatch, filterBill, filterQuotas, filterResources,
  fmtPeriod, fmtRate, kindLabel, kindsOf, linkedQuotas, parseSelKey, pendingOf, pendingSummary, periodsOf, quotaUsageIndex, regionsOf,
  selKey, specialtiesOf,
  type BillFilters, type BillRow, type CategoryNode, type Facet, type QuotaFilters, type ResourceFilters, type SelKey, type Tier,
} from "./cost-library/logic";
import "../gaea/styles.css";
import "../gaea/tailwind.css";
import "./cost-database.css";

type Phase = "loading" | "ready" | "failed";
type ExportKind = "template" | "bill" | "quota" | "resource" | "entry";
type LibraryRow = BillRow | WorkcostQuota | WorkcostResource;

const ROW_CAP = 400;
const NARROW_WIDTH = 980;
const EXPORT_LABEL: Record<ExportKind, string> = { template: "测算模版", bill: "清单", quota: "定额", resource: "工料机", entry: "资料条目" };
const TIER_LABEL: Record<Tier, string> = { bill: "清单", quota: "定额", resource: "工料机" };
const EXPORT_ITEMS: ExportItem[] = [
  { key: "template", label: "测算模版（按库预填）", hint: "六表空白测算表：工料机价格按当前专业/地区/期预填、费率预填最近项目口径；填完可在造价模式导回库" },
  { key: "bill", label: "清单（当前筛选）", hint: "项目、编码、名称、特征、挂接定额、原表单价、现算单价" },
  { key: "quota", label: "定额（当前筛选）", hint: "定额主表 + 含量明细附表" },
  { key: "resource", label: "工料机（当前筛选）", hint: "主数据与取用价、被定额引用次数" },
  { key: "entry", label: "资料条目（全部）", hint: "资料条目没有界面了，这里是唯一的导出口" },
];

/** 一次读取的三级数据快照（含现算参考价）。 */
interface Snapshot {
  projects: WorkcostBillProject[];
  rows: BillRow[];
  quotas: WorkcostQuota[];
  resources: WorkcostResource[];
  refPrices: Record<string, number>;
  /** 哪些来源读失败——读失败不是空库，界面要点名。 */
  failedSources: string[];
}

const EMPTY_SNAPSHOT: Snapshot = { projects: [], rows: [], quotas: [], resources: [], refPrices: {}, failedSources: [] };

function message(e: unknown): string {
  return e instanceof Error ? e.message : String(e);
}

function openOfficeBoard(): void {
  window.dispatchEvent(new CustomEvent("navigate", { detail: { page: "gaea" } }));
}

/** 维度取值的下拉：「全部」+ 各取值（带条数）；NONE 显示成「未标注」。 */
function FacetSelect({ label, allLabel, value, facets, onChange, emptyLabel, format, testid }: {
  label: string;
  allLabel: string;
  value: string;
  facets: Facet[];
  onChange: (next: string) => void;
  emptyLabel: string;
  /** 取值的显示写法（默认原样；期数用 fmtPeriod 显示成年月，与列表同一口径）。 */
  format?: (name: string) => string;
  testid?: string;
}) {
  // 没有可筛的（一个取值都没有）且当前也没在筛：不出这个下拉。
  if (facets.length === 0 && !value) return null;
  const show = (name: string) => (format ? format(name) : dimLabel(name, emptyLabel));
  return (
    <select className="cl-select" aria-label={label} data-testid={testid} value={value} onChange={(e) => onChange(e.target.value)}>
      <option value="">{allLabel}</option>
      {facets.map((f) => (
        <option key={f.name} value={f.name}>{show(f.name)}（{f.count}）</option>
      ))}
    </select>
  );
}

/** 行的选择键：清单用行 id，定额 / 工料机用编码。 */
function rowKeyOf(tier: Tier, row: LibraryRow): SelKey {
  return tier === "bill" ? selKey("bill", (row as BillRow).id) : selKey(tier, (row as WorkcostQuota | WorkcostResource).code);
}

// 逐条执行删除并收集失败原因（顺序执行：一次只碰一行，失败不影响其余行）。
async function runDeletes(tasks: { label: string; run: () => Promise<unknown> }[]): Promise<{ ok: number; failed: string[] }> {
  const failed: string[] = [];
  let ok = 0;
  for (const task of tasks) {
    try {
      await task.run();
      ok++;
    } catch (e) {
      failed.push(`${task.label}：${message(e)}`);
    }
  }
  return { ok, failed };
}

function StatePanel({ icon, title, error, action }: { icon: "loading" | "failed"; title: string; error?: string; action?: ReactNode }) {
  return (
    <div className="cl-state-panel" role={icon === "loading" ? "status" : "alert"} data-testid={`cost-${icon}`}>
      {icon === "loading" && <Loader2 size={20} className="cl-spin" aria-hidden />}
      {icon === "failed" && <AlertCircle size={20} aria-hidden />}
      <span className="cl-state-title">{title}</span>
      {error && <span>{error}；这不是空库。</span>}
      {action && <div className="cl-state-action">{action}</div>}
    </div>
  );
}

/** 空库：三步引导去造价模式录入，附可复制的示例提示词。 */
function EmptyLibrary({ copyState, onCopy }: { copyState: "idle" | "done" | "failed"; onCopy: (tier: Tier) => void }) {
  return (
    <div className="cl-empty" data-testid="cost-empty">
      <div className="cl-empty-inner">
        <div className="cl-empty-mark" aria-hidden><Coins size={22} /></div>
        <h2 className="cl-empty-title">造价库还是空的</h2>
        <p className="cl-empty-lead">
          三级造价数据——清单、定额、工料机——在办公板块的「造价」运行模式里录入和维护；
          这里用来看、下钻、导出和删除。
        </p>
        <ol className="cl-steps">
          <li className="cl-step">
            <span className="cl-step-index">01</span>
            <div>
              <div className="cl-step-title">在办公板块切到「造价」运行模式</div>
              <div className="cl-step-desc">输入框旁的模式钮里选档位，开启新会话生效。</div>
            </div>
          </li>
          <li className="cl-step">
            <span className="cl-step-index">02</span>
            <div>
              <div className="cl-step-title">让 AI 把数据录进来</div>
              <div className="cl-step-desc">造价资料格式不限：AI 读资料、整理后提交，勾稽核对由你确认后落库。</div>
              {(Object.keys(SAMPLE_PROMPTS) as Tier[]).map((t) => (
                <div key={t} className="cl-prompt">
                  <code>{SAMPLE_PROMPTS[t]}</code>
                  <button type="button" className="cl-btn cl-btn-ghost" aria-label={`复制${TIER_LABEL[t]}示例提示词`} onClick={() => onCopy(t)}>
                    <Copy size={13} aria-hidden />{copyState === "done" ? "已复制" : copyState === "failed" ? "复制失败" : "复制"}
                  </button>
                </div>
              ))}
            </div>
          </li>
          <li className="cl-step">
            <span className="cl-step-index">03</span>
            <div>
              <div className="cl-step-title">回到这里浏览与导出</div>
              <div className="cl-step-desc">清单点进去看综合单价，定额点进去看含量行，工料机点进去看价格与调价历史。</div>
            </div>
          </li>
        </ol>
        <div className="cl-empty-actions">
          <button type="button" className="cl-btn cl-btn-primary" onClick={openOfficeBoard}>
            <FolderOpen size={14} aria-hidden />去办公板块
          </button>
        </div>
      </div>
    </div>
  );
}

/** 选中项目时的封面卡（只读）：封面信息 + 口径（专业 / 地区 / 计价期）+ 四费率 + 整体删除。 */
function ProjectCard({ project, onDeleted }: { project: WorkcostBillProject; onDeleted: (p: WorkcostBillProject) => void }) {
  // 口径是库里的规范值（空=还没标注，在造价模式用 cost_library classify 归类）。
  const facts: [string, string][] = [
    ["专业", project.specialty || "未标注"], ["地区", project.region || "未标注"],
    ["计价期", project.pricePeriod ? fmtPeriod(project.pricePeriod) : "未标注"],
    ["封面地址", project.location], ["源资料", project.fileName], ["工期", project.duration], ["计价口径", project.pricing],
  ].filter(([, v]) => v) as [string, string][];
  return (
    <article className="cl-detail" data-testid="cost-project-card">
      <header className="cl-detail-head">
        <div className="cl-kicker">项目案例 · {project.itemCount} 条清单</div>
        <h2 className="cl-detail-title">{project.name}</h2>
        {facts.length > 0 && (
          <dl className="cl-facts">
            {facts.map(([k, v]) => <div key={k}><dt>{k}</dt><dd title={v}>{v}</dd></div>)}
          </dl>
        )}
      </header>
      <section className="cl-block">
        <div className="cl-block-head"><span className="cl-block-label">取费（录入数据，0=不计取）</span></div>
        <dl className="cl-price-facts">
          <div><dt>企业管理费</dt><dd>{fmtRate(project.managementRate)}</dd></div>
          <div><dt>规费</dt><dd>{fmtRate(project.regulatoryRate)}</dd></div>
          <div><dt>利润</dt><dd>{fmtRate(project.profitRate)}</dd></div>
          <div><dt>增值税</dt><dd>{fmtRate(project.taxRate)}</dd></div>
        </dl>
        {project.controlPrice > 0 && <p className="cl-note">控制价 {project.controlPrice.toLocaleString("zh-CN", { maximumFractionDigits: 2 })} 元</p>}
        <p className="cl-note">点左侧清单行看综合单价分析；改费率走造价模式的 cost_project_apply update_rates。</p>
        <button type="button" className="cl-btn cl-btn-danger-ghost" onClick={() => onDeleted(project)}>
          <Trash2 size={12} aria-hidden />删除这个项目
        </button>
      </section>
    </article>
  );
}

function CostLibraryPage() {
  const rootRef = useRef<HTMLDivElement>(null);
  const listRef = useRef<HTMLDivElement>(null);
  const loadSeq = useRef(0);

  const [narrow, setNarrow] = useState(false);
  const [pane, setPane] = useState<"list" | "detail">("list");

  const [phase, setPhase] = useState<Phase>("loading");
  const [loadError, setLoadError] = useState("");
  const [snap, setSnap] = useState<Snapshot>(EMPTY_SNAPSHOT);

  const [tier, setTier] = useState<Tier>("bill");
  const [selected, setSelected] = useState<SelKey | null>(null);
  const [checked, setChecked] = useState<Set<SelKey>>(new Set());
  const [cap, setCap] = useState(ROW_CAP);
  const [expandedGroups, setExpandedGroups] = useState<Set<string>>(new Set());
  const [billF, setBillF] = useState<BillFilters>(emptyBillFilters);
  const [quotaF, setQuotaF] = useState<QuotaFilters>(emptyQuotaFilters);
  const [resourceF, setResourceF] = useState<ResourceFilters>(emptyResourceFilters);

  const [deleteRequest, setDeleteRequest] = useState<DeleteRequest | null>(null);
  const [healthOpen, setHealthOpen] = useState(false);
  const [exportBusy, setExportBusy] = useState(false);
  const [exportNote, setExportNote] = useState<{ text: string; isError: boolean } | null>(null);
  const [notice, setNotice] = useState("");
  const [copyState, setCopyState] = useState<"idle" | "done" | "failed">("idle");

  // 窄窗（三栏放不下）改单栏：三级切换横排，列表与详情用页签切换。按容器宽度判定。
  useEffect(() => {
    const el = rootRef.current;
    if (!el) return;
    const apply = () => {
      const w = el.getBoundingClientRect().width;
      setNarrow(w > 0 && w <= NARROW_WIDTH);
    };
    apply();
    if (typeof ResizeObserver === "undefined") return;
    const ro = new ResizeObserver(apply);
    ro.observe(el);
    return () => ro.disconnect();
  }, []);

  // silent=true：切回板块时的后台重读——不清空正在看的内容，失败也保留旧快照。
  const load = useCallback(async (silent = false) => {
    const seq = ++loadSeq.current;
    if (!silent) setPhase("loading");
    setLoadError("");
    const [pRes, qRes, rRes] = await Promise.allSettled([
      app.WorkcostBillProjects(), app.WorkcostQuotaList("", ""), app.WorkcostResourceList("", ""),
    ]);
    if (seq !== loadSeq.current) return;
    const failedSources: string[] = [];
    const value = <T,>(r: PromiseSettledResult<T> | undefined, name: string, fallback: T): T => {
      if (r && r.status === "fulfilled") return (r.value ?? fallback) as T;
      failedSources.push(name);
      return fallback;
    };
    const projects = value(pRes, "项目", [] as WorkcostBillProject[]);
    const quotas = value(qRes, "定额", [] as WorkcostQuota[]);
    const resources = value(rRes, "工料机", [] as WorkcostResource[]);
    if (failedSources.length === 3) {
      const reason = [pRes, qRes, rRes].find((r) => r?.status === "rejected");
      setLoadError(reason && reason.status === "rejected" ? message(reason.reason) : "数据库暂时读不到");
      if (!silent) setPhase("failed");
      return;
    }
    let rows: BillRow[] = [];
    try {
      const batches = await Promise.all(
        projects.map(async (p) => ((await app.WorkcostBillItems(p.id)) ?? []).map((it) => ({ ...it, projectName: p.name }))),
      );
      rows = batches.flat();
    } catch {
      failedSources.push("清单行");
    }
    if (seq !== loadSeq.current) return;
    const codes = [...new Set([
      ...quotas.map((q) => q.code),
      ...rows.flatMap((it) => linkedQuotas(it)),
    ])];
    let refPrices: Record<string, number> = {};
    if (codes.length > 0) {
      try {
        // 现算参考价失败不拦列表——单价列回退占位，但要点名。
        const many = (await app.WorkcostQuotaComposeMany(codes)) ?? {};
        for (const [code, c] of Object.entries(many)) refPrices[code] = c.compositePrice;
      } catch {
        refPrices = {};
        failedSources.push("现算单价");
      }
    }
    if (seq !== loadSeq.current) return;
    setSnap({ projects, rows, quotas, resources, refPrices, failedSources });
    setPhase("ready");
  }, []);

  useEffect(() => { void load(); }, [load]);

  // 本板块 keepAlive 常驻：去造价模式录完数据再切回来，组件不会重挂，挂载时
  // 那一次读取就成了旧快照。板块由隐藏转回可见时静默重读（方案库同款修法）。
  const boardActive = useBoardActive("cost");
  const awayRef = useRef(false);
  useEffect(() => {
    if (!boardActive) {
      awayRef.current = true;
      return;
    }
    if (!awayRef.current) return;
    awayRef.current = false;
    void load(true);
  }, [boardActive, load]);

  // ── 派生：引用索引、筛选项、当前级的可见行 ──
  const usageIndex = useMemo(() => quotaUsageIndex(snap.rows), [snap.rows]);
  const usageCount = useMemo(() => new Map<string, number>([...usageIndex].map(([code, list]) => [code, list.length])), [usageIndex]);
  const divisions = useMemo(() => divisionsOf(filterBill(snap.rows, { ...billF, division: "", keyword: "", grouped: false })), [snap.rows, billF]);
  // 维度（库定规矩）：规范值只认库里的取值，空值归「未标注」一桶。
  const specialties = useMemo(() => specialtiesOf(snap.quotas), [snap.quotas]);
  const billSpecFacets = useMemo(() => billSpecialties(snap.rows, { projectId: 0, division: billF.division }), [snap.rows, billF.division]);
  // 当前专业下有清单行的项目——左栏的项目列表随专业变。
  const specProjects = useMemo(() => {
    if (!billF.specialty) return snap.projects;
    const ids = new Set(snap.rows.filter((it) => facetMatch(billF.specialty, it.specialty)).map((it) => it.projectId));
    return snap.projects.filter((p) => ids.has(p.id));
  }, [snap.projects, snap.rows, billF.specialty]);
  // 章节只在专业内有意义：没选专业时不出章节这一级。
  const chapters = useMemo(() => (quotaF.specialty ? chaptersOf(snap.quotas, quotaF.specialty) : []), [snap.quotas, quotaF.specialty]);
  // 四类里有行的按固定顺序；四类外有行时末尾加一桶「类别待整理」（存量里类别不合规矩的行）。
  const kinds = useMemo(() => kindsOf(snap.resources), [snap.resources]);
  const resourceCategories = useMemo(() => categoryTreeOf(snap.resources, resourceF.kind), [snap.resources, resourceF.kind]);
  const resourceRegions = useMemo(() => regionsOf(snap.resources), [snap.resources]);
  const resourcePeriods = useMemo(() => periodsOf(snap.resources), [snap.resources]);
  const pending = useMemo(() => pendingOf(snap.rows, snap.quotas, snap.resources), [snap.rows, snap.quotas, snap.resources]);

  const billRows = useMemo(() => filterBill(snap.rows, billF), [snap.rows, billF]);
  const quotaRows = useMemo(() => filterQuotas(snap.quotas, quotaF), [snap.quotas, quotaF]);
  const resourceRows = useMemo(() => filterResources(snap.resources, resourceF), [snap.resources, resourceF]);
  const matched: LibraryRow[] = tier === "bill" ? billRows : tier === "quota" ? quotaRows : resourceRows;
  const totalOfTier = tier === "bill" ? snap.rows.length : tier === "quota" ? snap.quotas.length : snap.resources.length;
  const view = useMemo(() => capRows(matched, cap), [matched, cap]);

  // 切级或改筛选回到首批行数；切级清空勾选（勾选只在当前级内有意义）。
  useEffect(() => { setCap(ROW_CAP); setChecked(new Set()); }, [tier]);
  useEffect(() => { setCap(ROW_CAP); }, [billF, quotaF, resourceF]);

  const counts: TierCounts = { bill: snap.rows.length, quota: snap.quotas.length, resource: snap.resources.length };
  const currentProject = snap.projects.find((p) => p.id === billF.projectId) ?? null;
  const libraryEmpty = phase === "ready" && counts.bill === 0 && counts.quota === 0 && counts.resource === 0;
  const keyword = tier === "bill" ? billF.keyword : tier === "quota" ? quotaF.keyword : resourceF.keyword;
  const tierMeta = TIERS.find((t) => t.key === tier) ?? TIERS[0];

  // 选择键对应的行还在不在（重读后别处被删，就不能让它继续挂在界面上）。
  const keyAlive = useCallback((key: SelKey): boolean => {
    const p = parseSelKey(key);
    if (!p) return false;
    if (p.tier === "bill") return snap.rows.some((r) => String(r.id) === p.id);
    if (p.tier === "quota") return snap.quotas.some((q) => q.code === p.id);
    return snap.resources.some((r) => r.code === p.id);
  }, [snap]);
  useEffect(() => {
    if (selected && !keyAlive(selected)) setSelected(null);
    if (checked.size > 0) {
      const next = new Set([...checked].filter(keyAlive));
      if (next.size !== checked.size) setChecked(next);
    }
  }, [snap, selected, checked, keyAlive]);

  // 当前行滚进列表视野（下钻跳转后能看见它）。
  useEffect(() => {
    if (!selected || (narrow && pane !== "list")) return;
    const el = listRef.current?.querySelector<HTMLElement>(`[data-row-key="${selected.replace(/"/g, '\\"')}"]`);
    el?.scrollIntoView?.({ block: "nearest" });
  }, [selected, tier, pane, narrow, view.hidden]);

  // ── 三级下钻 ──
  const changeTier = (next: Tier) => {
    setTier(next);
    const parsed = selected ? parseSelKey(selected) : null;
    if (parsed && parsed.tier !== next) setSelected(null);
    setPane("list");
  };

  // 跳到某一级的某一行。目标行过不了该级当前筛选时才清空筛选（保证它在列表里可见，
  // 跨级下钻、体检跳转都是这种情况）；它本来就在列表里时保留筛选——同级点一行看详情，
  // 人刚筛到这儿，不能把上下文冲掉。
  const jump = (next: Tier, key: SelKey) => {
    const id = parseSelKey(key)?.id;
    if (next === "bill") {
      const row = snap.rows.find((r) => String(r.id) === id);
      if (!row || filterBill([row], billF).length === 0) setBillF(emptyBillFilters);
    } else if (next === "quota") {
      const quota = snap.quotas.find((q) => q.code === id);
      if (!quota || filterQuotas([quota], quotaF).length === 0) setQuotaF(emptyQuotaFilters);
    } else {
      const resource = snap.resources.find((r) => r.code === id);
      if (!resource || filterResources([resource], resourceF).length === 0) setResourceF(emptyResourceFilters);
    }
    setTier(next);
    setSelected(key);
    setPane("detail");
  };
  const openBill = (row: BillRow) => jump("bill", selKey("bill", row.id));
  const openQuota = (code: string) => jump("quota", selKey("quota", code));
  const openResource = (code: string) => jump("resource", selKey("resource", code));
  const onHealthJump = (j: HealthJump) => {
    setHealthOpen(false);
    jump(j.tier, selKey(j.tier, j.code));
  };

  // ── 勾选与删除 ──
  const toggle = (key: SelKey, next: boolean) => setChecked((prev) => {
    const n = new Set(prev);
    if (next) n.add(key);
    else n.delete(key);
    return n;
  });
  const toggleAll = (next: boolean) => setChecked((prev) => {
    const n = new Set(prev);
    for (const row of matched) {
      const key = rowKeyOf(tier, row);
      if (next) n.add(key);
      else n.delete(key);
    }
    return n;
  });
  const checkedKeys = [...checked].filter((k) => k.startsWith(`${tier}:`)).sort();
  const checkedIds = new Set(checkedKeys.map((k) => k.slice(k.indexOf(":") + 1)));
  const checkedBills = snap.rows.filter((r) => checkedIds.has(String(r.id)));
  const checkedQuotas = snap.quotas.filter((q) => checkedIds.has(q.code));
  const checkedResources = snap.resources.filter((r) => checkedIds.has(r.code));

  const deleteBillRows = (rows: BillRow[]) => setDeleteRequest({
    title: "删除清单项",
    lines: [
      `将删除 ${rows.length} 条清单项：${rows.slice(0, 3).map((r) => `「${r.title}」`).join("、")}${rows.length > 3 ? " 等" : ""}`,
      "定额挂接随行清理；项目封面与其余清单项不受影响。不可恢复。",
    ],
    confirmLabel: `删除 ${rows.length} 条`,
    run: () => runDeletes(rows.map((r) => ({ label: `${r.code} ${r.title}`, run: () => app.WorkcostBillItemDelete(r.id) }))),
  });
  const deleteQuotaRows = (rows: WorkcostQuota[]) => setDeleteRequest({
    title: "删除定额",
    lines: [
      `将删除 ${rows.length} 条消耗定额：${rows.slice(0, 3).map((q) => `「${q.title}」`).join("、")}${rows.length > 3 ? " 等" : ""}（含全部含量行）。`,
      "仍被清单项引用的定额会被拒绝，不留悬空引用；要删先在造价模式解除挂接。不可恢复。",
    ],
    confirmLabel: `删除 ${rows.length} 条`,
    run: () => runDeletes(rows.map((q) => ({ label: `${q.code} ${q.title}`, run: () => app.WorkcostQuotaDelete(q.code) }))),
  });
  const deleteResourceRows = (rows: WorkcostResource[]) => setDeleteRequest({
    title: "删除工料机",
    lines: [
      `将删除 ${rows.length} 条工料机：${rows.slice(0, 3).map((r) => `「${r.title}」`).join("、")}${rows.length > 3 ? " 等" : ""}。`,
      "仍被定额含量行引用的工料机会被拒绝；删除后调价历史一并清除。不可恢复。",
    ],
    confirmLabel: `删除 ${rows.length} 条`,
    run: () => runDeletes(rows.map((r) => ({ label: `${r.code} ${r.title}`, run: () => app.WorkcostResourceDelete(r.id, false) }))),
  });
  const deleteProject = (p: WorkcostBillProject) => setDeleteRequest({
    title: "删除项目",
    lines: [
      `将删除项目「${p.name}」的封面与全部 ${p.itemCount} 条分部分项清单。`,
      "本项目独占的定额一并删除；其他项目仍在引用的定额与全部工料机保留。不可恢复。",
    ],
    confirmLabel: "删除项目",
    run: async () => {
      try {
        await app.WorkcostBillProjectDelete(p.id);
        return { ok: 1, failed: [] as string[] };
      } catch (e) {
        return { ok: 0, failed: [message(e)] };
      }
    },
  });
  const deleteChecked = () => {
    if (tier === "bill") deleteBillRows(checkedBills);
    else if (tier === "quota") deleteQuotaRows(checkedQuotas);
    else deleteResourceRows(checkedResources);
  };

  const showNotice = (text: string) => {
    setNotice(text);
    window.setTimeout(() => setNotice(""), 4000);
  };

  // ── 导出 ──
  const runExport = async (rawKey: string) => {
    const kind = rawKey as ExportKind;
    setExportBusy(true);
    try {
      const dir = await app.PickDirectory();
      if (!dir) return; // 取消选目录不是错误：不清上一次的导出结果
      const n = new Date();
      const p2 = (v: number) => String(v).padStart(2, "0");
      const stamp = `${n.getFullYear()}${p2(n.getMonth() + 1)}${p2(n.getDate())}-${p2(n.getHours())}${p2(n.getMinutes())}${p2(n.getSeconds())}`;
      const out = `${dir.replace(/[\\/]+$/, "")}/造价库-${EXPORT_LABEL[kind]}-${stamp}.xlsx`;
      // 导出跟随当前筛选：filter 结构体承载全部筛选口径（与列表同口径，NONE 哨兵=只要未标注的行）。
      const none = { projectId: 0, keyword: "", specialty: "", division: "", chapter: "", kind: "", category: "", region: "", period: "" };
      // 测算模版的口径：专业取当前级的专业筛选，地区/期/类别/分类取工料机级筛选
      // （「未标注」哨兵后端当没给——模版没有「只要未标注」这种用法）。
      const scope = (v: string) => (v === NONE || v === OUTSIDE ? "" : v);
      const filter: WorkcostLibraryFilter = kind === "template"
        ? { ...none, specialty: scope(tier === "bill" ? billF.specialty : quotaF.specialty), kind: scope(resourceF.kind),
            category: scope(resourceF.category), region: scope(resourceF.region), period: scope(resourceF.period) }
        : kind === "bill"
        ? { ...none, projectId: billF.projectId, keyword: billF.keyword, specialty: billF.specialty, division: billF.division }
        : kind === "quota"
          ? { ...none, keyword: quotaF.keyword, specialty: quotaF.specialty, chapter: quotaF.chapter }
          : kind === "resource"
            ? { ...none, keyword: resourceF.keyword, kind: resourceF.kind, category: resourceF.category, region: resourceF.region, period: resourceF.period }
            : none;
      const written = await app.CostLibraryExport(kind, filter, out);
      setExportNote({ text: `已导出：${written}`, isError: false });
    } catch (e) {
      setExportNote({ text: `导出失败：${message(e)}`, isError: true });
    } finally {
      setExportBusy(false);
    }
  };

  const copyPrompt = async (t: Tier) => {
    // 复制走仓内单源：成功才显示「已复制」，失败如实提示。
    setCopyState(await copyToClipboard(SAMPLE_PROMPTS[t]) ? "done" : "failed");
    window.setTimeout(() => setCopyState("idle"), 1600);
  };

  const setKeyword = (kw: string) => {
    if (tier === "bill") setBillF((f) => ({ ...f, keyword: kw }));
    else if (tier === "quota") setQuotaF((f) => ({ ...f, keyword: kw }));
    else setResourceF((f) => ({ ...f, keyword: kw }));
  };
  const clearFilters = () => {
    if (tier === "bill") setBillF(emptyBillFilters);
    else if (tier === "quota") setQuotaF(emptyQuotaFilters);
    else setResourceF(emptyResourceFilters);
  };
  const hasFilter = tier === "bill"
    ? billF.projectId !== 0 || !!billF.division || !!billF.specialty || !!billF.keyword
    : tier === "quota"
      ? !!quotaF.specialty || !!quotaF.chapter || !!quotaF.keyword
      : !!resourceF.kind || !!resourceF.category || !!resourceF.region || !!resourceF.period || !!resourceF.keyword;

  // 当前位置：下钻到三层以上时让人知道自己在哪（专业 › 项目 › 分部）。
  // 分类路径已含类别名（材料/混凝土/泵送），前一级已显示类别，这里只列类别以下的各段。
  const categoryCrumbs = (() => {
    const c = resourceF.category;
    if (!c) return [] as string[];
    if (c === NONE) return ["未细分"];
    if (c === resourceF.kind || c.startsWith(`${resourceF.kind}/`)) return c.slice(resourceF.kind.length).split("/").filter(Boolean);
    return [c];
  })();
  const crumbs = tier === "bill"
    ? [billF.specialty ? dimLabel(billF.specialty) : "", currentProject?.name ?? "", billF.division].filter(Boolean)
    : tier === "quota"
      ? [quotaF.specialty ? dimLabel(quotaF.specialty) : "", quotaF.chapter ? dimLabel(quotaF.chapter, "未归章节") : ""].filter(Boolean)
      : [
        resourceF.kind === OUTSIDE ? kindLabel(OUTSIDE) : resourceF.kind, ...categoryCrumbs,
        resourceF.region ? dimLabel(resourceF.region) : "",
        resourceF.period ? (resourceF.period === NONE ? "无期" : fmtPeriod(resourceF.period)) : "",
      ].filter(Boolean);

  // 「待整理」：文件不一定带专业 / 地区 / 分类，由造价模式的模型归类。点一下复制提示词。
  const copyPending = async () => {
    setCopyState(await copyToClipboard(PENDING_PROMPT) ? "done" : "failed");
    window.setTimeout(() => setCopyState("idle"), 1600);
  };
  // 选专业 = 换一个专业看：回到全部项目（原项目在这个专业下可能没有行）。
  const setBillSpecialty = (specialty: string) => setBillF({ ...billF, specialty, projectId: 0, division: "" });

  // ── 右栏详情：按选择键落到对应级 ──
  const parsedSel = selected ? parseSelKey(selected) : null;
  const selectedBill = parsedSel?.tier === "bill" ? (snap.rows.find((r) => String(r.id) === parsedSel.id) ?? null) : null;
  const selectedQuota = parsedSel?.tier === "quota" ? (snap.quotas.find((q) => q.code === parsedSel.id) ?? null) : null;
  const selectedResource = parsedSel?.tier === "resource" ? (snap.resources.find((r) => r.code === parsedSel.id) ?? null) : null;
  const activeKey = selected && parsedSel?.tier === tier ? selected : null;
  // 宽窗下右栏按需展开：选中了行，或清单级正看着某个项目的封面卡。
  const detailActive = !!selected || (tier === "bill" && !!currentProject);

  const listPane = (
    <>
      <div className="cl-col-head">
        <span className="cl-col-head-title">{tierMeta.label}</span>
        {crumbs.length > 0 && (
          <nav className="cl-crumb" aria-label="当前位置" data-testid="cost-crumb">
            {crumbs.map((c, i) => (
              <span key={`${i}:${c}`} className="cl-crumb-item">{i > 0 && <span className="cl-crumb-sep" aria-hidden>›</span>}{c}</span>
            ))}
          </nav>
        )}
        <span className="cl-count" data-testid="cost-list-count">
          {matched.length !== totalOfTier ? `匹配 ${matched.length} / ${totalOfTier}` : `${totalOfTier} 条`}
        </span>
        {narrow && <span className="cl-spacer" />}
      </div>
      <div className="cl-list-tools">
        {tier === "bill" && (
          <>
            <div className="cl-seg" role="group" aria-label="清单视图">
              <button type="button" aria-pressed={!billF.grouped} onClick={() => setBillF({ ...billF, grouped: false })}>按明细</button>
              <button type="button" aria-pressed={billF.grouped} onClick={() => setBillF({ ...billF, grouped: true })}>按库项归并</button>
            </div>
            {divisions.length > 0 && (
              <select className="cl-select" aria-label="筛选分部" value={billF.division}
                onChange={(e) => setBillF({ ...billF, division: e.target.value })}>
                <option value="">全部分部</option>
                {divisions.map((d) => <option key={d} value={d}>{d}</option>)}
              </select>
            )}
          </>
        )}
        {tier === "resource" && (
          <>
            <FacetSelect label="筛选地区" allLabel="全部地区" value={resourceF.region} facets={resourceRegions} emptyLabel="未标注"
              testid="cost-select-region" onChange={(v) => setResourceF({ ...resourceF, region: v })} />
            <FacetSelect label="筛选期数" allLabel="全部期" value={resourceF.period} facets={resourcePeriods} emptyLabel="无期"
              format={(name) => (name === NONE ? "无期" : fmtPeriod(name))}
              testid="cost-select-period" onChange={(v) => setResourceF({ ...resourceF, period: v })} />
          </>
        )}
        <span className="cl-spacer" />
        {checkedKeys.length > 0 && (
          <button type="button" className="cl-btn cl-btn-danger-ghost" data-testid="cost-delete-checked" onClick={deleteChecked}>
            <Trash2 size={12} aria-hidden />删除所选 {checkedKeys.length}
          </button>
        )}
      </div>
      <div className="cl-list">
        {matched.length === 0 ? (
          <div className="cl-state" data-testid="cost-tier-empty">
            <Inbox size={18} aria-hidden />
            <span className="cl-state-title">{totalOfTier === 0 ? `${tierMeta.label}库还是空的` : "没有匹配的行"}</span>
            {totalOfTier === 0 ? (
              <>
                <span>在造价模式里录入后会出现在这里。</span>
                <div className="cl-state-action">
                  <button type="button" className="cl-btn" onClick={openOfficeBoard}><FolderOpen size={13} aria-hidden />去办公板块</button>
                </div>
              </>
            ) : hasFilter ? (
              <div className="cl-state-action">
                <button type="button" className="cl-btn" onClick={clearFilters}>清除筛选</button>
              </div>
            ) : null}
          </div>
        ) : (
          <>
            {tier === "bill" && (
              <BillTable rows={view.shown as BillRow[]} grouped={billF.grouped} showProject={billF.projectId === 0}
                refPrices={snap.refPrices} checked={checked} active={activeKey} expanded={expandedGroups}
                onToggleGroup={(key) => setExpandedGroups((prev) => {
                  const n = new Set(prev);
                  if (n.has(key)) n.delete(key);
                  else n.add(key);
                  return n;
                })}
                onToggle={toggle} onToggleAll={toggleAll} onOpen={openBill} onOpenQuota={openQuota}
                onDelete={(row) => deleteBillRows([row])} />
            )}
            {tier === "quota" && (
              <QuotaTable rows={view.shown as WorkcostQuota[]} usage={usageCount}
                refPrices={snap.refPrices} checked={checked} active={activeKey}
                onToggle={toggle} onToggleAll={toggleAll} onOpen={(q) => jump("quota", selKey("quota", q.code))}
                onDelete={(q) => deleteQuotaRows([q])} />
            )}
            {tier === "resource" && (
              <ResourceTable rows={view.shown as WorkcostResource[]} checked={checked} active={activeKey}
                onToggle={toggle} onToggleAll={toggleAll} onOpen={(r) => jump("resource", selKey("resource", r.code))}
                onDelete={(r) => deleteResourceRows([r])} />
            )}
            {view.hidden > 0 && (
              <div className="cl-more">
                <button type="button" className="cl-btn" onClick={() => setCap(cap + ROW_CAP)}>
                  再显示 {Math.min(ROW_CAP, view.hidden)} 条（余 {view.hidden} 条）
                </button>
              </div>
            )}
          </>
        )}
      </div>
    </>
  );

  const detailPane = (
    <>
      <div className="cl-col-head">
        <span className="cl-col-head-title">详情</span>
        {narrow && (
          <button type="button" className="cl-btn cl-btn-ghost" onClick={() => setPane("list")}>返回列表</button>
        )}
      </div>
      <div className="cl-detail-scroll">
        {selectedBill ? (
          <BillDetail item={selectedBill} refPrices={snap.refPrices} onOpenQuota={openQuota} onOpenResource={openResource} />
        ) : selectedQuota ? (
          <QuotaDetail quota={selectedQuota} usage={usageIndex.get(selectedQuota.code) ?? []} onOpenBill={openBill} onOpenResource={openResource} />
        ) : selectedResource ? (
          <ResourceDetail resource={selectedResource} onOpenQuota={openQuota} />
        ) : tier === "bill" && currentProject ? (
          <ProjectCard project={currentProject} onDeleted={deleteProject} />
        ) : (
          <DetailState phase="idle" />
        )}
      </div>
    </>
  );

  // 窄窗放不下左栏：两级上下文各一个下拉（清单：专业 → 项目；定额：专业 → 章节；
  // 工料机：类别 → 分类）。
  const facetOptions = (facets: Facet[], emptyLabel: string) =>
    facets.map((f) => ({ value: f.name, label: `${dimLabel(f.name, emptyLabel)}（${f.count}）` }));
  const flattenCategories = (nodes: CategoryNode[], depth = 0): { value: string; label: string }[] =>
    nodes.flatMap((n) => [{ value: n.path, label: `${"　".repeat(depth)}${n.label}（${n.count}）` }, ...flattenCategories(n.children, depth + 1)]);
  const contextOptions = tier === "bill"
    ? [{ value: "", label: "全部专业" }, ...facetOptions(billSpecFacets, "未标注专业")]
    : tier === "quota"
      ? [{ value: "", label: "全部专业" }, ...facetOptions(specialties, "未标注专业")]
      : [{ value: "", label: "全部类别" }, ...kinds.map((k) => ({ value: k.kind, label: `${kindLabel(k.kind)}（${k.count}）` }))];
  const contextValue = tier === "bill" ? billF.specialty : tier === "quota" ? quotaF.specialty : resourceF.kind;
  const onContext = (v: string) => {
    if (tier === "bill") setBillSpecialty(v);
    else if (tier === "quota") setQuotaF({ ...quotaF, specialty: v, chapter: "" });
    else setResourceF({ ...resourceF, kind: v, category: "" });
  };
  // 第二级只在选了第一级后出现（章节在专业内、分类在类别内才有意义；「类别待整理」下面没有分类）；清单的项目级恒在。
  const firstChosen = tier === "quota" ? quotaF.specialty : resourceF.kind;
  const secondChosen = tier === "bill" || (!!firstChosen && firstChosen !== OUTSIDE);
  const secondOptions = tier === "bill"
    ? [{ value: "0", label: "全部项目" }, ...specProjects.map((p) => ({ value: String(p.id), label: `${p.name}（${p.itemCount}）` }))]
    : tier === "quota"
      ? [{ value: "", label: "全部章节" }, ...facetOptions(chapters, "未归章节")]
      : [
        { value: "", label: "全部分类" },
        ...(resourceCategories.uncategorized > 0 ? [{ value: NONE, label: `未细分（${resourceCategories.uncategorized}）` }] : []),
        ...flattenCategories(resourceCategories.tree),
      ];
  const secondValue = tier === "bill" ? String(billF.projectId) : tier === "quota" ? quotaF.chapter : resourceF.category;
  const onSecond = (v: string) => {
    if (tier === "bill") setBillF((f) => ({ ...f, projectId: Number(v), division: "" }));
    else if (tier === "quota") setQuotaF((f) => ({ ...f, chapter: v }));
    else setResourceF((f) => ({ ...f, category: v }));
  };

  return (
    <div ref={rootRef} className="cost-library" data-narrow={narrow ? "true" : "false"} data-testid="cost-library">
      <a className="cl-skip-link" href="#cost-library-workspace">跳到造价库工作区</a>

      <header className="cl-masthead">
        <div className="cl-brand">
          <span className="cl-mark" aria-hidden><Coins size={18} /></span>
          <div>
            <div className="cl-kicker">COST LIBRARY</div>
            <div className="cl-title">造价数据库</div>
            <div className="cl-sub">清单 → 定额 → 工料机 · 三级造价数据</div>
          </div>
        </div>
        {phase === "ready" && !libraryEmpty && (
          <div className="cl-search" role="search">
            <div className="cl-search-box">
              <Search size={14} aria-hidden />
              <input
                aria-label={`搜索${tierMeta.label}`}
                placeholder={tierMeta.searchPlaceholder}
                value={keyword}
                onChange={(e) => setKeyword(e.target.value)}
              />
              {keyword && (
                <button type="button" className="cl-icon-btn" aria-label="清空搜索词" onClick={() => setKeyword("")}>
                  <X size={13} />
                </button>
              )}
            </div>
          </div>
        )}
        <div className="cl-actions">
          {phase === "ready" && !libraryEmpty && (
            <>
              <button type="button" className="cl-btn" title="只读扫描数据健康发现（含不合规矩的数据）" onClick={() => setHealthOpen(true)}>
                <Shield size={13} aria-hidden />体检
              </button>
              <ExportMenu
                items={EXPORT_ITEMS}
                onPick={(key) => void runExport(key)}
                busy={exportBusy}
                // 导出进行中先收起上一次的结果，避免旧路径被当成这次的。
                note={exportBusy ? "" : exportNote?.text ?? ""}
                isError={!!exportNote?.isError}
              />
            </>
          )}
          <button type="button" className="cl-btn" aria-label="刷新造价库" title="刷新" disabled={phase === "loading"}
            onClick={() => void load()}>
            <RefreshCw size={13} aria-hidden />
          </button>
        </div>
      </header>

      <div className="cl-status" aria-live="polite" data-testid="cost-status">
        {phase === "ready" && (
          <>
            <span>项目 <strong>{snap.projects.length}</strong></span>
            <span>清单 <strong>{snap.rows.length}</strong></span>
            <span>定额 <strong>{snap.quotas.length}</strong></span>
            <span>工料机 <strong>{snap.resources.length}</strong></span>
            {pending.total > 0 && (
              <button
                type="button"
                className="cl-pending"
                data-testid="cost-pending"
                title={`${pendingSummary(pending)}。点击复制整理提示词——到办公板块的「造价」运行模式粘贴发送，由模型归类、你确认后落库`}
                onClick={() => void copyPending()}
              >
                {copyState === "done" ? "已复制提示词" : copyState === "failed" ? "复制失败" : `待整理 ${pending.total}`}
              </button>
            )}
            {snap.failedSources.length > 0 && (
              <span className="cl-status-warn" role="alert">{snap.failedSources.join("、")}读取失败，对应内容可能不完整</span>
            )}
            {loadError && <span className="cl-status-warn" role="alert">刷新失败：{loadError}（显示的是上次读到的内容）</span>}
            {notice && <span data-testid="cost-notice">{notice}</span>}
          </>
        )}
      </div>

      {phase === "loading" && <StatePanel icon="loading" title="正在读取造价库…" />}
      {phase === "failed" && (
        <StatePanel icon="failed" title="造价库读取失败" error={loadError}
          action={<button type="button" className="cl-btn" onClick={() => void load()}><RefreshCw size={13} aria-hidden />重试</button>} />
      )}
      {libraryEmpty && <EmptyLibrary copyState={copyState} onCopy={(t) => void copyPrompt(t)} />}

      {phase === "ready" && !libraryEmpty && (
        <section
          id="cost-library-workspace"
          className="cl-body"
          aria-label="造价库工作区"
          tabIndex={-1}
          // 详情按需展开：宽窗下没有选中行、也没有项目封面卡可看时，右栏不占位。
          data-detail={detailActive ? "shown" : "none"}
        >
          {narrow && (
            <div className="cl-bar-row">
              <TierBar tier={tier} counts={counts} onTier={changeTier} contextOptions={contextOptions}
                contextValue={contextValue} onContext={onContext}
                contextLabel={tier === "bill" || tier === "quota" ? "筛选专业" : "筛选类别"}
                secondOptions={secondChosen ? secondOptions : undefined} secondValue={secondValue} onSecond={onSecond}
                secondLabel={tier === "bill" ? "筛选项目" : tier === "quota" ? "筛选章节" : "筛选分类"} />
              <div className="cl-seg" role="tablist" aria-label="造价库视图">
                <button type="button" role="tab" aria-selected={pane === "list"} onClick={() => setPane("list")}>列表</button>
                <button type="button" role="tab" aria-selected={pane === "detail"} onClick={() => setPane("detail")}>详情</button>
              </div>
            </div>
          )}
          {!narrow && (
            <nav className="cl-col cl-rail-col">
              <TierRail
                tier={tier} counts={counts} onTier={changeTier}
                projects={specProjects} billFilters={billF}
                billSpecialties={billSpecFacets} onBillSpecialty={setBillSpecialty}
                onProject={(id) => setBillF({ ...billF, projectId: id, division: "" })}
                onDeleteProject={deleteProject}
                specialties={specialties} chapters={chapters} quotaFilters={quotaF}
                onSpecialty={(specialty) => setQuotaF({ ...quotaF, specialty, chapter: "" })}
                onChapter={(chapter) => setQuotaF({ ...quotaF, chapter })}
                kinds={kinds} categories={resourceCategories} resourceFilters={resourceF}
                onKind={(kind) => setResourceF({ ...resourceF, kind, category: "" })}
                onCategory={(category) => setResourceF({ ...resourceF, category })}
              />
            </nav>
          )}
          <div className="cl-col cl-list-col" ref={listRef} hidden={narrow && pane !== "list"}>{listPane}</div>
          <div className="cl-col cl-detail-col" hidden={narrow ? pane !== "detail" : !detailActive}>{detailPane}</div>
        </section>
      )}

      <HealthDrawer open={healthOpen} onClose={() => setHealthOpen(false)} onJump={onHealthJump} />
      <DeleteDialog
        request={deleteRequest}
        onClose={() => setDeleteRequest(null)}
        onDone={(summary) => { showNotice(summary); void load(true); }}
      />
    </div>
  );
}

export default CostLibraryPage;
