// CostLibraryPage — 一级板块「造价数据库」（2026-11-25 推倒重建；同日库定规矩）。
//
// 定位：三级造价数据的浏览面——清单 → 定额 → 工料机。导入、解析、批量维护在
// 办公板块的「造价」运行模式里由模型完成（cost_library / cost_project_apply）；
// 这里给人看，外加导出与删除；2026-11-27 起每条数据可点「编辑」直接修正录入
// 错误（字段 / 含量行 / 挂接 / 调价 / 口径 / 费率），修完静默重读。
//
// 维度：清单与定额按专业分（定额在专业内按分部，字段沿用 chapter；定额地区与
// 上游清单项目挂钩），工料机按种类（类别 → 分类）、地区、期数分。规范值只认库里
// 的取值；没标注的归「未标注」一桶，状态行的「待整理」指路去造价模式归类，
// 也可以点编辑直接标注。
//
// 布局：左栏三级切换与两级上下文（专业 → 项目 / 专业 → 分部 / 类别 → 分类）→
// 中栏列表（筛选、当前位置、批量删除、下钻入口）→ 右栏详情（综合单价分析与
// 消耗量、含量行分析表、价格卡与各期价格），三级之间可以来回下钻。
// 四态分明：加载 / 空库（三步引导）/ 无匹配 / 读取失败。
// 搜索：本地子串过滤在前；该级向量索引已建时，后台混合检索把「意思相近」的条目补在
// 后面。本地语义模型不可用时在搜索框旁明示（2026-11-28），不再静默只剩关键词。
import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { Popover } from "antd";
import {
  AlertCircle, ClipboardList, Coins, Copy, FolderOpen, Globe, Inbox, Loader2, Pencil, Plus, RefreshCw, Search, Shield, Trash2, Wand2, X,
} from "../gaea/icons";
import { app } from "../gaea/lib/bridge";
import { copyToClipboard } from "../gaea/lib/clipboard";
import { useBoardActive } from "../lib/boardActive";
import type { CostIndexStatusView, WorkcostBillProject, WorkcostDictItem, WorkcostLibraryFilter, WorkcostQuota, WorkcostResource } from "../gaea/lib/types";
import { BillDetail, DetailState, QuotaDetail, ResourceDetail } from "./cost-library/Details";
import { BillEdit, ProjectEdit, QuotaEdit, ResourceEdit, type EditDicts } from "./cost-library/Edit";
import { HealthDrawer, type HealthJump } from "./cost-library/HealthDrawer";
import { DeleteDialog, ExportMenu, type DeleteRequest, type ExportItem } from "./cost-library/Overlays";
import { BatchEditDialog } from "./cost-library/BatchEdit";
import { AliasDialog, RegionBackfillDialog } from "./cost-library/Maintain";
import { BillTable, QuotaTable, ResourceTable } from "./cost-library/Tables";
import { TierBar, TierRail, type TierCounts } from "./cost-library/TierRail";
import { IndexStatus } from "./cost-library/IndexStatus";
import {
  NONE, OUTSIDE, PENDING_ITEMS, PENDING_PROMPT, SAMPLE_PROMPTS, STATE_LABELS, TIERS, capRows, categoryTreeOf, chaptersOf, billSpecialties, dimLabel,
  emptyBillFilters, emptyQuotaFilters, emptyResourceFilters, divisionsOf, facetMatch, filterBill, filterQuotas, filterResources,
  fmtPeriod, fmtRate, kindLabel, kindsOf, keywordTerms, linkedQuotas, parseSelKey, pendingFilter, pendingOf, pendingSummary, periodsOf, projectCounts, quotaUsageIndex,
  quotaRegionsOf, regionsOf, selKey, specialtiesOf,
  type BillFilters, type BillRow, type CategoryNode, type Facet, type FilterContext, type PendingCounts, type QuotaFilters,
  type ResourceFilters, type RowState, type SelKey, type Tier,
} from "./cost-library/logic";
import "../gaea/styles.css";
import "../gaea/tailwind.css";
import "./cost-database.css";

type Phase = "loading" | "ready" | "failed";
type ExportKind = "template" | "bill" | "quota" | "resource";
type LibraryRow = BillRow | WorkcostQuota | WorkcostResource;

const ROW_CAP = 400;
const NARROW_WIDTH = 980;
const EXPORT_LABEL: Record<ExportKind, string> = { template: "测算模版", bill: "清单", quota: "定额", resource: "工料机" };
const TIER_LABEL: Record<Tier, string> = { bill: "清单", quota: "定额", resource: "工料机" };
// 「资料条目」导出口随旧成本库退役（2026-11-26 后端已删）——菜单里不再列。
const EXPORT_ITEMS: ExportItem[] = [
  { key: "template", label: "测算模版（样表 + 使用说明书）", hint: "六表测算样表，旁边带一份使用说明书；版式以样表为准、程序不预填，综合单价表是唯一导入硬契约。填完经造价模式导回库" },
  { key: "bill", label: "清单（当前筛选）", hint: "项目、编码、名称、特征、挂接定额、原表单价、现算单价" },
  { key: "quota", label: "定额（当前筛选）", hint: "定额主表 + 含量明细附表" },
  { key: "resource", label: "工料机（当前筛选）", hint: "主数据与取用价、被定额引用次数" },
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
// 语义补充段最多显示的条数（补充段是「意思相近」的提示，不是主列表）。
const SEM_CAP = 30;

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

/** 选中项目时的封面卡：封面信息 + 口径（专业 / 地区 / 计价期）+ 四费率 + 编辑 + 整体删除。 */
function ProjectCard({ project, onDeleted, specialtyOptions, regionOptions, periodOptions, onSaved, dicts, onDirty }: {
  project: WorkcostBillProject;
  onDeleted: (p: WorkcostBillProject) => void;
  /** 编辑面：口径与费率的可选值（库里已有的规范值）与保存回执。 */
  specialtyOptions: string[];
  regionOptions: string[];
  periodOptions: string[];
  onSaved: (msg: string) => void;
  dicts?: EditDicts;
  onDirty?: (dirty: boolean) => void;
}) {
  const [editing, setEditing] = useState(false);
  // 换项目就退出编辑态——不让上一个项目的草稿挂在新项目上。
  useEffect(() => { setEditing(false); }, [project.id]);
  // 口径是库里的规范值（空=还没标注，可在这里直接标注）。
  const facts: [string, string][] = [
    ["专业", project.specialty || "未标注"], ["地区", project.region || "未标注"],
    ["计价期", project.pricePeriod ? fmtPeriod(project.pricePeriod) : "未标注"],
    ["封面地址", project.location], ["源资料", project.fileName], ["工期", project.duration], ["计价口径", project.pricing],
  ].filter(([, v]) => v) as [string, string][];
  return (
    <article className="cl-detail" data-testid="cost-project-card">
      <header className="cl-detail-head">
        <div className="cl-kicker">项目案例 · {project.itemCount} 条清单</div>
        {!editing && (
          <button type="button" className="cl-btn cl-detail-edit" data-testid="cost-project-edit-open" onClick={() => setEditing(true)}>
            <Pencil size={12} aria-hidden />编辑
          </button>
        )}
        <h2 className="cl-detail-title">{project.name}</h2>
        {facts.length > 0 && (
          <dl className="cl-facts">
            {facts.map(([k, v]) => <div key={k}><dt>{k}</dt><dd title={v}>{v}</dd></div>)}
          </dl>
        )}
      </header>
      {editing ? (
        <ProjectEdit project={project} specialtyOptions={specialtyOptions} regionOptions={regionOptions}
          periodOptions={periodOptions} dicts={dicts} onDirty={onDirty} onSaved={onSaved} onCancel={() => setEditing(false)} />
      ) : (
        <section className="cl-block">
          <div className="cl-block-head"><span className="cl-block-label">取费（录入数据，0=不计取）</span></div>
          <dl className="cl-price-facts">
            <div><dt>企业管理费</dt><dd>{fmtRate(project.managementRate)}</dd></div>
            <div><dt>规费</dt><dd>{fmtRate(project.regulatoryRate)}</dd></div>
            <div><dt>利润</dt><dd>{fmtRate(project.profitRate)}</dd></div>
            <div><dt>增值税</dt><dd>{fmtRate(project.taxRate)}</dd></div>
          </dl>
          {project.controlPrice > 0 && <p className="cl-note">控制价 {project.controlPrice.toLocaleString("zh-CN", { maximumFractionDigits: 2 })} 元</p>}
          <p className="cl-note">点「编辑」直接改口径与费率；清单行的综合单价在它的详情里。</p>
          <button type="button" className="cl-btn cl-btn-danger-ghost" onClick={() => onDeleted(project)}>
            <Trash2 size={12} aria-hidden />删除这个项目
          </button>
        </section>
      )}
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
  // 「待整理」浮层（逐项点开即筛出对应的行）。
  const [pendingOpen, setPendingOpen] = useState(false);
  // 勾选批量改维度 / 定额地区回填 / 检索别名管理（2026-11-28）。
  const [batchOpen, setBatchOpen] = useState(false);
  const [regionOpen, setRegionOpen] = useState(false);
  const [aliasOpen, setAliasOpen] = useState(false);
  const [exportBusy, setExportBusy] = useState(false);
  const [exportNote, setExportNote] = useState<{ text: string; isError: boolean } | null>(null);
  const [notice, setNotice] = useState("");
  // 语义索引（2026-11-26 造价库接入本地模型索引）：状态栏的状态钮读它，三级关键词检索也靠它
  // 判断「该级的向量索引建好没有」；semRows 是语义补充段（名称不含关键词、意思相近的条目）。
  const [indexStatus, setIndexStatus] = useState<CostIndexStatusView | null>(null);
  const [semRows, setSemRows] = useState<LibraryRow[]>([]);
  const semSeq = useRef(0);
  // 本地语义模型不可用的原因（非空 = 向量路坏了，搜索只按关键词匹配）：搜完读一次索引
  // 状态得到，明示在搜索框旁，不再静默降级；semNonce 是「重试」计数，点一下重新检索。
  const [semDown, setSemDown] = useState("");
  const [semNonce, setSemNonce] = useState(0);
  // 混合检索给出的相关度名次（行键 → 名次）：搜索时主列表按它排，不在检索结果里的保持原序跟在后面。
  const [rankOf, setRankOf] = useState<Map<string, number>>(() => new Map());
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

  // 索引状态读不到（老版本后端 / 绑定不可用）按「没有语义索引」处理，不影响本地过滤。
  const readIndexStatus = useCallback(async (): Promise<CostIndexStatusView | null> => {
    if (typeof app.CostIndexStatus !== "function") return null;
    try {
      return await app.CostIndexStatus();
    } catch {
      return null;
    }
  }, []);
  const loadIndexStatus = useCallback(async () => {
    setIndexStatus(await readIndexStatus());
  }, [readIndexStatus]);
  useEffect(() => { void loadIndexStatus(); }, [loadIndexStatus]);

  // 规范值字典（2026-11-28）：编辑面的专业 / 地区 / 分类下拉读它——字典里规划了、还没有行
  // 在用的值也列得出来（定额专业此前只列得出「土壤修复」）。读不到（老版本后端 / 失败）时
  // 各处退回「库里已有取值」，也不出「新增」。
  const [dicts, setDicts] = useState<WorkcostDictItem[]>([]);
  const [dictsOk, setDictsOk] = useState(false);
  const loadDicts = useCallback(async () => {
    if (typeof app.WorkcostDictList !== "function") return;
    try {
      setDicts((await app.WorkcostDictList("")) ?? []);
      setDictsOk(true);
    } catch {
      setDictsOk(false);
    }
  }, []);
  useEffect(() => { void loadDicts(); }, [loadDicts]);

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
    void loadIndexStatus();
    void loadDicts();
  }, [boardActive, load, loadIndexStatus, loadDicts]);

  // ── 派生：引用索引、筛选项、当前级的可见行 ──
  const usageIndex = useMemo(() => quotaUsageIndex(snap.rows), [snap.rows]);
  const usageCount = useMemo(() => new Map<string, number>([...usageIndex].map(([code, list]) => [code, list.length])), [usageIndex]);
  // 状态筛选要用的引用关系（被定额含量行引用的工料机编码要另取一次定额含量行：列表不带它，
  // 用到才取、取到即缓存，每次重读后作废）。
  const [usedResourceCodes, setUsedResourceCodes] = useState<Set<string> | null>(null);
  const usedCodesSeq = useRef(0);
  const needUsedCodes = tier === "resource" && resourceF.state === "unused";
  useEffect(() => {
    if (!needUsedCodes || usedResourceCodes) return;
    const seq = ++usedCodesSeq.current;
    void Promise.all(snap.quotas.map(async (q) => ((await app.WorkcostQuotaGet(q.code))?.items ?? [])))
      .then((lists) => {
        if (seq === usedCodesSeq.current) setUsedResourceCodes(new Set(lists.flat().map((it) => it.resourceCode).filter(Boolean)));
      })
      .catch(() => { if (seq === usedCodesSeq.current) setUsedResourceCodes(new Set()); });
  }, [needUsedCodes, usedResourceCodes, snap.quotas]);
  useEffect(() => { setUsedResourceCodes(null); }, [snap]);
  const filterCtx = useMemo<FilterContext>(() => ({
    quotaSpecialty: new Map(snap.quotas.map((q) => [q.code, (q.specialty ?? "").trim()])),
    quotaUsage: usageCount,
    usedResourceCodes: usedResourceCodes ?? undefined,
  }), [snap.quotas, usageCount, usedResourceCodes]);
  // 分面（2026-11-28 分面联动）：各级的计数随搜索词与状态走——搜「混凝土」时左栏只数
  // 命中的行；每一维自己的筛选值不套进自己的计数，否则选了一个值就只剩它一个可选项。
  const divisions = useMemo(
    () => divisionsOf(snap.rows, { projectId: billF.projectId, division: billF.division, keyword: billF.keyword, state: billF.state }, filterCtx),
    [snap.rows, billF.projectId, billF.division, billF.keyword, billF.state, filterCtx],
  );
  // 维度（库定规矩）：规范值只认库里的取值，空值归「未标注」一桶。
  const specialties = useMemo(
    () => specialtiesOf(snap.quotas, { keyword: quotaF.keyword, state: quotaF.state }, filterCtx),
    [snap.quotas, quotaF.keyword, quotaF.state, filterCtx],
  );
  const quotaRegions = useMemo(
    () => quotaRegionsOf(snap.quotas, { keyword: quotaF.keyword, state: quotaF.state }, filterCtx),
    [snap.quotas, quotaF.keyword, quotaF.state, filterCtx],
  );
  const billSpecFacets = useMemo(
    () => billSpecialties(snap.rows, { division: billF.division, keyword: billF.keyword, state: billF.state }, filterCtx),
    [snap.rows, billF.division, billF.keyword, billF.state, filterCtx],
  );
  // 每个项目在搜索下各剩多少行——左栏项目列表的计数随之变，没有命中行的项目不出。
  const projectMatch = useMemo(
    () => projectCounts(snap.rows, { division: billF.division, keyword: billF.keyword, state: billF.state }, filterCtx),
    [snap.rows, billF.division, billF.keyword, billF.state, filterCtx],
  );
  // 当前专业下有清单行的项目——左栏的项目列表随专业变。
  const specProjects = useMemo(() => {
    let list = snap.projects;
    if (billF.specialty) {
      const ids = new Set(snap.rows.filter((it) => facetMatch(billF.specialty, it.specialty)).map((it) => it.projectId));
      list = list.filter((p) => ids.has(p.id));
    }
    if (billF.keyword.trim() || billF.state) list = list.filter((p) => (projectMatch.get(p.id) ?? 0) > 0);
    return list;
  }, [snap.projects, snap.rows, billF.specialty, billF.keyword, billF.state, projectMatch]);
  // 分部只在专业内有意义：没选专业时不出分部这一级。
  const chapters = useMemo(
    () => (quotaF.specialty ? chaptersOf(snap.quotas, quotaF.specialty, quotaF.keyword) : []),
    [snap.quotas, quotaF.specialty, quotaF.keyword],
  );
  // 四类里有行的按固定顺序；四类外有行时末尾加一桶「类别待整理」（存量里类别不合规矩的行）。
  const kinds = useMemo(
    () => kindsOf(snap.resources, { keyword: resourceF.keyword, state: resourceF.state }, filterCtx),
    [snap.resources, resourceF.keyword, resourceF.state, filterCtx],
  );
  const resourceCategories = useMemo(
    () => categoryTreeOf(snap.resources, resourceF.kind, { keyword: resourceF.keyword, state: resourceF.state }, filterCtx),
    [snap.resources, resourceF.kind, resourceF.keyword, resourceF.state, filterCtx],
  );
  const resourceRegions = useMemo(
    () => regionsOf(snap.resources, { keyword: resourceF.keyword, state: resourceF.state }, filterCtx),
    [snap.resources, resourceF.keyword, resourceF.state, filterCtx],
  );
  const resourcePeriods = useMemo(
    () => periodsOf(snap.resources, { keyword: resourceF.keyword, state: resourceF.state }, filterCtx),
    [snap.resources, resourceF.keyword, resourceF.state, filterCtx],
  );
  const pending = useMemo(() => pendingOf(snap.rows, snap.quotas, snap.resources), [snap.rows, snap.quotas, snap.resources]);

  // 状态筛选「没有被定额引用」在引用关系取到之前先不给行（给了也是错的）。
  const usedCodesLoading = needUsedCodes && !usedResourceCodes;
  const billRows = useMemo(() => filterBill(snap.rows, billF, filterCtx), [snap.rows, billF, filterCtx]);
  const quotaRows = useMemo(() => filterQuotas(snap.quotas, quotaF, filterCtx), [snap.quotas, quotaF, filterCtx]);
  const resourceRows = useMemo(
    () => (usedCodesLoading ? [] : filterResources(snap.resources, resourceF, filterCtx)),
    [snap.resources, resourceF, filterCtx, usedCodesLoading],
  );
  const matched: LibraryRow[] = tier === "bill" ? billRows : tier === "quota" ? quotaRows : resourceRows;
  const totalOfTier = tier === "bill" ? snap.rows.length : tier === "quota" ? snap.quotas.length : snap.resources.length;
  // 搜索时主列表按混合检索的相关度排（2026-11-28）：本地命中的行里，检索结果靠前的靠前，
  // 不在检索结果里的保持原序跟在后面——「名称含关键词」与「意思相近」共用一套相关度序。
  const searching = !!(tier === "bill" ? billF.keyword : tier === "quota" ? quotaF.keyword : resourceF.keyword).trim();
  const ordered = useMemo(() => {
    if (!searching || rankOf.size === 0) return matched;
    const rank = (row: LibraryRow) => rankOf.get(rowKeyOf(tier, row)) ?? Number.MAX_SAFE_INTEGER;
    return [...matched].sort((a, b) => rank(a) - rank(b));
  }, [matched, rankOf, searching, tier]);
  const view = useMemo(() => capRows(ordered, cap), [ordered, cap]);

  // 切级或改筛选回到首批行数；切级清空勾选（勾选只在当前级内有意义）。
  useEffect(() => { setCap(ROW_CAP); setChecked(new Set()); }, [tier]);
  useEffect(() => { setCap(ROW_CAP); }, [billF, quotaF, resourceF]);

  const counts: TierCounts = { bill: snap.rows.length, quota: snap.quotas.length, resource: snap.resources.length };
  const currentProject = snap.projects.find((p) => p.id === billF.projectId) ?? null;
  const libraryEmpty = phase === "ready" && counts.bill === 0 && counts.quota === 0 && counts.resource === 0;
  const keyword = tier === "bill" ? billF.keyword : tier === "quota" ? quotaF.keyword : resourceF.keyword;
  const tierMeta = TIERS.find((t) => t.key === tier) ?? TIERS[0];

  // ── 语义检索补充 ──
  // 搜索框照旧做本地子串过滤；关键词非空且该级向量索引已建时，另调一次后台混合检索，把「名称不含
  // 关键词、意思相近」的条目（如「挖机」对「挖掘机」）放进补充段。索引没建 / 绑定不可用 / 请求失败：
  // 没有补充段，本地过滤照常。其它筛选（专业 / 分部 / 类别…）对补充段同样生效。
  // 向量路要是坏了（本地模型不可用），搜完读一次索引状态拿到原因，明示在搜索框旁——不能让
  // 「没有意思相近的条目」和「模型坏了」看起来一样。
  useEffect(() => {
    const kw = keyword.trim();
    const kind = tier === "bill" ? "wc_bill" : tier === "quota" ? "wc_quota" : "wc_res";
    const ready = (indexStatus?.kinds.find((k) => k.kind === kind)?.count ?? 0) > 0;
    if (!kw || !ready) {
      setSemRows([]);
      setRankOf(new Map());
      setSemDown("");
      return;
    }
    const seq = ++semSeq.current;
    const timer = window.setTimeout(async () => {
      let extra: LibraryRow[] = [];
      // 后台混合检索回来的是相关度序：记下每行的名次，主列表按它排（名次相同的保持原序）。
      const rank = new Map<string, number>();
      try {
        if (tier === "quota") {
          const local = new Set(quotaRows.map((q) => q.code));
          const found = (await app.WorkcostQuotaList(quotaF.specialty, kw)) ?? [];
          found.forEach((q, i) => rank.set(q.code, i));
          extra = found.filter((q) => !local.has(q.code) && filterQuotas([q], { ...quotaF, keyword: "" }, filterCtx).length > 0);
        } else if (tier === "resource") {
          const local = new Set(resourceRows.map((r) => r.code));
          const found = (await app.WorkcostResourceList(resourceF.kind, kw)) ?? [];
          found.forEach((r, i) => rank.set(r.code, i));
          extra = found.filter((r) => !local.has(r.code) && filterResources([r], { ...resourceF, keyword: "" }, filterCtx).length > 0);
        } else if (typeof app.WorkcostBillSearch === "function") {
          const local = new Set(billRows.map((r) => String(r.id)));
          const found = ((await app.WorkcostBillSearch(kw, billF.specialty, billF.projectId)) ?? [])
            .map((it) => ({ ...it, projectName: snap.projects.find((p) => p.id === it.projectId)?.name ?? "" }));
          found.forEach((r, i) => rank.set(String(r.id), i));
          extra = found.filter((row) => !local.has(String(row.id)) && filterBill([row], { ...billF, keyword: "" }, filterCtx).length > 0);
        }
      } catch {
        extra = [];
      }
      if (seq !== semSeq.current) return;
      // 向量路坏没坏，以这次检索自己的结果为准（索引器那边要有文档要嵌才会报错）。
      const st = await readIndexStatus();
      if (seq !== semSeq.current) return;
      setSemRows(extra);
      setRankOf(rank);
      setSemDown(st?.kinds.find((k) => k.kind === kind)?.searchError ?? "");
    }, 250);
    return () => window.clearTimeout(timer);
  }, [tier, keyword, indexStatus, quotaRows, resourceRows, billRows, quotaF, resourceF, billF, snap.projects, snap.rows, filterCtx, readIndexStatus, semNonce]);

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
  // 编辑面有未保存改动时（Edit 组件经 onDirty 回报），换行 / 换级 / 换筛选前先问一句。
  const dirtyEdit = useRef(false);
  const guardDirty = (): boolean =>
    !dirtyEdit.current || window.confirm("当前编辑框里有未保存的改动，切走后会被丢弃。仍要切走吗？");
  const changeTier = (next: Tier) => {
    if (next !== tier && !guardDirty()) return;
    setCreating(null);
    setTier(next);
    const parsed = selected ? parseSelKey(selected) : null;
    if (parsed && parsed.tier !== next) setSelected(null);
    setPane("list");
  };

  // 跳到某一级的某一行。目标行过不了该级当前筛选时才清空筛选（保证它在列表里可见，
  // 跨级下钻、体检跳转都是这种情况）；它本来就在列表里时保留筛选——同级点一行看详情，
  // 人刚筛到这儿，不能把上下文冲掉。
  const jump = (next: Tier, key: SelKey) => {
    if (!guardDirty()) return;
    setCreating(null);
    const id = parseSelKey(key)?.id;
    if (next === "bill") {
      const row = snap.rows.find((r) => String(r.id) === id);
      if (!row || filterBill([row], billF, filterCtx).length === 0) setBillF(emptyBillFilters);
    } else if (next === "quota") {
      const quota = snap.quotas.find((q) => q.code === id);
      if (!quota || filterQuotas([quota], quotaF, filterCtx).length === 0) setQuotaF(emptyQuotaFilters);
    } else {
      const resource = snap.resources.find((r) => r.code === id);
      if (!resource || filterResources([resource], resourceF, filterCtx).length === 0) setResourceF(emptyResourceFilters);
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
  // 「待整理」逐项点开（2026-11-28）：落到对应那一级，只留这一项的筛选，就地改或勾选批量改。
  const applyPending = (key: keyof PendingCounts) => {
    if (!guardDirty()) return;
    setCreating(null);
    const target = pendingFilter(key);
    setBillF(target.tier === "bill" ? { ...(target.filters as BillFilters), grouped: billF.grouped } : emptyBillFilters);
    setQuotaF(target.tier === "quota" ? (target.filters as QuotaFilters) : emptyQuotaFilters);
    setResourceF(target.tier === "resource" ? (target.filters as ResourceFilters) : emptyResourceFilters);
    setSelected(null);
    setTier(target.tier);
    setPane("list");
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
  // 当前级勾选的行（批量修改用）。
  const checkedRows: LibraryRow[] = tier === "bill" ? checkedBills : tier === "quota" ? checkedQuotas : checkedResources;

  const deleteBillRows = (rows: BillRow[]) => setDeleteRequest({
    title: "删除清单项",
    lines: [
      `将删除 ${rows.length} 条清单项：${rows.slice(0, 3).map((r) => `「${r.title}」`).join("、")}${rows.length > 3 ? " 等" : ""}`,
      "定额挂接随行清理；项目封面与其余清单项不受影响。不可恢复。",
    ],
    confirmLabel: `删除 ${rows.length} 条`,
    run: () => runDeletes(rows.map((r) => ({ label: `${r.code} ${r.title}`, run: () => app.WorkcostBillItemDelete(r.id) }))),
  });
  const quotaUsed = (q: WorkcostQuota) => usageCount.get(q.code) ?? 0;
  const deleteQuotaRows = (rows: WorkcostQuota[]) => {
    const used = rows.filter((q) => quotaUsed(q) > 0);
    const links = rows.reduce((n, q) => n + quotaUsed(q), 0);
    return setDeleteRequest({
      title: "删除定额",
      lines: [
        `将删除 ${rows.length} 条消耗定额：${rows.slice(0, 3).map((q) => `「${q.title}」`).join("、")}${rows.length > 3 ? " 等" : ""}（含全部含量行）。`,
        "仍被清单项引用的定额会被拒绝，不留悬空引用。不可恢复。",
        ...(used.length > 0
          ? [`其中 ${used.length} 条仍被清单项引用（共 ${links} 条挂接）：要删它们，用「解除挂接后删除」——先把这些定额从清单项上摘掉，那些清单项随之变成手填项。`]
          : []),
      ],
      confirmLabel: `删除 ${rows.length} 条`,
      run: () => runDeletes(rows.map((q) => ({ label: `${q.code} ${q.title}`, run: () => app.WorkcostQuotaDelete(q.code) }))),
      force: used.length > 0 ? {
        label: "解除挂接后删除",
        hint: "先解除这些定额在全部清单项上的挂接（清单项变成手填项），再删除定额",
        run: () => runDeletes(rows.map((q) => ({
          label: `${q.code} ${q.title}`,
          run: async () => {
            if (quotaUsed(q) > 0) await app.WorkcostQuotaDetachEverywhere(q.code);
            await app.WorkcostQuotaDelete(q.code);
          },
        }))),
      } : undefined,
    });
  };
  const deleteResourceRows = (rows: WorkcostResource[]) => setDeleteRequest({
    title: "删除工料机",
    lines: [
      `将删除 ${rows.length} 条工料机：${rows.slice(0, 3).map((r) => `「${r.title}」`).join("、")}${rows.length > 3 ? " 等" : ""}。`,
      "仍被定额含量行引用的工料机会被拒绝；删除后调价历史一并清除。不可恢复。",
    ],
    confirmLabel: `删除 ${rows.length} 条`,
    run: () => runDeletes(rows.map((r) => ({ label: `${r.code} ${r.title}`, run: () => app.WorkcostResourceDelete(r.id, false) }))),
    force: {
      label: "连同解除引用删除",
      hint: "仍被定额含量行引用的工料机：连带解除那些含量行对它的引用后删除（对应定额里的含量行会消失，综合单价随之变化）",
      run: () => runDeletes(rows.map((r) => ({ label: `${r.code} ${r.title}`, run: () => app.WorkcostResourceDelete(r.id, true) }))),
    },
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

  // ── 编辑面（2026-11-27）：保存后的回执 = 提示 + 静默重读（选中行/详情保持）──
  const onEditSaved = useCallback((msg: string) => {
    showNotice(msg);
    void load(true);
    void loadIndexStatus();
    void loadDicts(); // 编辑面里可能就地新增了字典值
  }, [load, loadIndexStatus, loadDicts]);
  // 规范值可选面（2026-11-28 改读字典）：字典值在前，库里已有取值兜底（不含「未标注」哨兵）。
  const unionNames = (a: string[], b: string[]): string[] => [...new Set([...a, ...b])];
  const dictSpecialty = useMemo(() => dicts.filter((d) => d.kind === "specialty").map((d) => d.name), [dicts]);
  const dictRegion = useMemo(() => dicts.filter((d) => d.kind === "region").map((d) => d.name), [dicts]);
  const billSpecNames = useMemo(
    () => unionNames(dictSpecialty, billSpecFacets.filter((f) => f.name !== NONE).map((f) => f.name)), [dictSpecialty, billSpecFacets],
  );
  const quotaSpecNames = useMemo(
    () => unionNames(dictSpecialty, specialties.filter((f) => f.name !== NONE).map((f) => f.name)), [dictSpecialty, specialties],
  );
  const chaptersOfSpec = useCallback((spec: string) => chaptersOf(snap.quotas, spec).map((f) => f.name), [snap.quotas]);
  const resourceRegionNames = useMemo(
    () => unionNames(dictRegion, resourceRegions.filter((f) => f.name !== NONE).map((f) => f.name)), [dictRegion, resourceRegions],
  );
  const resourcePeriodNames = useMemo(() => resourcePeriods.filter((f) => f.name !== NONE).map((f) => f.name), [resourcePeriods]);
  const categoriesOfKind = useCallback((kind: string) => {
    const flat = (nodes: CategoryNode[], depth = 0): string[] =>
      nodes.flatMap((n) => (depth === 0 ? [n.path, ...flat(n.children, 1)] : flat(n.children, depth + 1)));
    return unionNames(
      dicts.filter((d) => d.kind === "category" && (d.name === kind || d.name.startsWith(`${kind}/`))).map((d) => d.name),
      flat(categoryTreeOf(snap.resources, kind).tree),
    );
  }, [snap.resources, dicts]);
  // 项目口径可选面：资源与项目两侧已有的取值并集。
  const projectRegionNames = useMemo(() =>
    unionNames(resourceRegionNames, snap.projects.map((p) => p.region).filter(Boolean) as string[]), [resourceRegionNames, snap.projects]);
  const projectPeriodNames = useMemo(() =>
    [...new Set([...resourcePeriodNames, ...snap.projects.map((p) => p.pricePeriod).filter(Boolean)])], [resourcePeriodNames, snap.projects]);
  const projectSpecNames = useMemo(() =>
    unionNames(billSpecNames, snap.projects.map((p) => p.specialty).filter(Boolean) as string[]), [billSpecNames, snap.projects]);
  // 编辑面的字典接入：就地新增走字典写入；地区会被归一成省级全称，所以加完回读、取库里最终的写法。
  const addDict = useCallback(async (kind: "specialty" | "region" | "category", name: string, parent = ""): Promise<string> => {
    await app.WorkcostDictSet({ id: 0, kind, name, parent }, false);
    const items = (await app.WorkcostDictList(kind)) ?? [];
    await loadDicts();
    const typed = name.trim();
    const newest = items.reduce<WorkcostDictItem | undefined>((m, d) => (!m || d.id > m.id ? d : m), undefined);
    return items.find((d) => d.name === typed)?.name ?? newest?.name ?? typed;
  }, [loadDicts]);
  const editDicts = useMemo<EditDicts>(
    () => (dictsOk ? { specialty: dictSpecialty, region: dictRegion, categoryOf: categoriesOfKind, add: addDict } : {}),
    [dictsOk, dictSpecialty, dictRegion, categoriesOfKind, addDict],
  );
  // 新建 / 复制另存（2026-11-28）：右栏换成新建表单；base 给了 = 以它为底复制。
  const [creating, setCreating] = useState<{ tier: Tier; base?: LibraryRow } | null>(null);
  const blankBill = (): BillRow => ({
    id: 0, projectId: billF.projectId, projectName: "", code: "", gcode: "", title: "", unit: "", division: "", quantity: 0,
    quantityExpr: "", quotaCode: "", feature: "", priceOverride: 0, listPrice: 0, sort: 0, quotaCodes: [], specialty: "",
  });
  const blankQuota = (): WorkcostQuota => ({
    id: 0, code: "", gcode: "", origCode: "", title: "", specialty: quotaF.specialty === NONE ? "" : quotaF.specialty,
    chapter: quotaF.chapter === NONE ? "" : quotaF.chapter, unit: "", categoryPath: "", baseLabor: 0, baseMaterial: 0,
    baseMachine: 0, source: "", region: quotaF.region === NONE ? "" : (quotaF.region ?? ""), priceDate: "", note: "",
    status: "现行", items: [],
  });
  const blankResource = (): WorkcostResource => ({
    id: 0, code: "", gcode: "", kind: resourceF.kind && resourceF.kind !== OUTSIDE ? resourceF.kind : "材料", title: "", spec: "",
    unit: "", basePrice: 0, currentPrice: 0, categoryPath: resourceF.category === NONE ? "" : resourceF.category, source: "",
    supplier: "", region: resourceF.region === NONE ? "" : resourceF.region, priceDate: "", priceType: "", validUntil: "",
    lossRate: 0, note: "", tags: [], status: "现行",
  });

  const startCreate = (next: Tier, base?: LibraryRow) => {
    if (!guardDirty()) return;
    setCreating({ tier: next, base });
    setSelected(null);
    setTier(next);
    setPane("detail");
  };
  // 新建成功：静默重读后定位到新那一行（详情仍是它，方便接着改）。
  const onCreated = async (key: SelKey) => {
    const created = creating?.tier;
    setCreating(null);
    await load(true);
    if (created) {
      setSelected(key);
      setPane("detail");
    }
  };

  // 语义索引显式补齐（状态栏的钮）：模型不可用时绑定会 reject，原因留在提示里。
  const backfillIndex = async () => {
    if (typeof app.CostIndexBackfill !== "function") return;
    try {
      await app.CostIndexBackfill("");
      await loadIndexStatus();
      showNotice("语义索引已补齐");
    } catch (e) {
      await loadIndexStatus();
      showNotice(`语义索引补齐失败：${message(e)}`);
    }
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
      // 状态筛选与定额地区筛选是板块里的视图口径，导出的筛选结构体不承载——回执里说清楚，
      // 免得导出的行数与列表对不上时以为是丢了数据。
      const notExported = [
        (kind === "bill" ? billF.state : kind === "quota" ? quotaF.state : kind === "resource" ? resourceF.state : "") && "状态",
        kind === "quota" && quotaF.region && "定额地区",
      ].filter(Boolean);
      const suffix = notExported.length > 0 ? `（${notExported.join("、")}筛选不参与导出）` : "";
      // 测算模版旁边绑着一份使用说明书（后端一并落盘）：提示里带上它，别让人以为只出来一个表。
      setExportNote({ text: (kind === "template" ? `已导出：${written}（使用说明书：同目录同名 -使用说明.md）` : `已导出：${written}`) + suffix, isError: false });
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
  const setFilterState = (state: RowState) => {
    if (tier === "bill") setBillF({ ...billF, state });
    else if (tier === "quota") setQuotaF({ ...quotaF, state });
    else setResourceF({ ...resourceF, state });
  };
  const stateOf = tier === "bill" ? billF.state : tier === "quota" ? quotaF.state : resourceF.state;
  const stateLabel = (state?: RowState) => (STATE_LABELS[tier].find((s) => s.value === state && state)?.label ?? "");
  const hasFilter = tier === "bill"
    ? billF.projectId !== 0 || !!billF.division || !!billF.specialty || !!billF.keyword || !!billF.state
    : tier === "quota"
      ? !!quotaF.specialty || !!quotaF.chapter || !!quotaF.region || !!quotaF.keyword || !!quotaF.state
      : !!resourceF.kind || !!resourceF.category || !!resourceF.region || !!resourceF.period || !!resourceF.keyword || !!resourceF.state;

  // 当前位置：下钻到三层以上时让人知道自己在哪（专业 › 项目 › 分部）。
  // 分类路径已含类别名（材料/混凝土/泵送），前一级已显示类别，这里只列类别以下的各段。
  const categoryCrumbs = (() => {
    const c = resourceF.category;
    if (!c) return [] as string[];
    if (c === NONE) return ["未细分"];
    if (c === resourceF.kind || c.startsWith(`${resourceF.kind}/`)) return c.slice(resourceF.kind.length).split("/").filter(Boolean);
    return [c];
  })();
  const crumbs = (tier === "bill"
    ? [billF.specialty ? dimLabel(billF.specialty) : "", currentProject?.name ?? "", billF.division]
    : tier === "quota"
      ? [
        quotaF.specialty ? dimLabel(quotaF.specialty) : "", quotaF.chapter ? dimLabel(quotaF.chapter, "未归分部") : "",
        quotaF.region ? dimLabel(quotaF.region, "未标地区") : "",
      ]
      : [
        resourceF.kind === OUTSIDE ? kindLabel(OUTSIDE) : resourceF.kind, ...categoryCrumbs,
        resourceF.region ? dimLabel(resourceF.region) : "",
        resourceF.period ? (resourceF.period === NONE ? "无期" : fmtPeriod(resourceF.period)) : "",
      ]).filter(Boolean).concat(stateLabel(stateOf) || []);

  // 「待整理」：文件不一定带专业 / 地区 / 分类。2026-11-28 起逐项点开即筛出对应的行，
  // 在板块里直接改或勾选批量改；仍可以把整理提示词复制给造价模式的模型。
  const copyPending = async () => {
    setCopyState(await copyToClipboard(PENDING_PROMPT) ? "done" : "failed");
    window.setTimeout(() => setCopyState("idle"), 1600);
  };
  const pendingItems = PENDING_ITEMS.filter((it) => pending[it.key] > 0).map((it) => ({ ...it, count: pending[it.key] }));
  // 选专业 = 换一个专业看：回到全部项目（原项目在这个专业下可能没有行）。
  const setBillSpecialty = (specialty: string) => setBillF({ ...billF, specialty, projectId: 0, division: "" });

  // ── 右栏详情：按选择键落到对应级 ──
  const parsedSel = selected ? parseSelKey(selected) : null;
  const selectedBill = parsedSel?.tier === "bill" ? (snap.rows.find((r) => String(r.id) === parsedSel.id) ?? null) : null;
  const selectedQuota = parsedSel?.tier === "quota" ? (snap.quotas.find((q) => q.code === parsedSel.id) ?? null) : null;
  const selectedResource = parsedSel?.tier === "resource" ? (snap.resources.find((r) => r.code === parsedSel.id) ?? null) : null;
  const activeKey = selected && parsedSel?.tier === tier ? selected : null;
  // 宽窗下右栏按需展开：选中了行，或清单级正看着某个项目的封面卡。
  const detailActive = !!selected || !!creating || (tier === "bill" && !!currentProject);

  // 一批行按当前级渲染成表（主列表与语义补充段共用同一套表组件）。semantic = 这张表是
  // 「意思相近」补充段（行上标来源）；搜索时命中的词条高亮。
  const searchTerms = searching ? keywordTerms(keyword) : undefined;
  const renderTables = (rows: LibraryRow[], semantic = false) => (
    <>
            {tier === "bill" && (
              <BillTable rows={rows as BillRow[]} grouped={billF.grouped} showProject={billF.projectId === 0}
                refPrices={snap.refPrices} checked={checked} active={activeKey} expanded={expandedGroups}
                highlight={searchTerms} semantic={semantic}
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
              <QuotaTable rows={rows as WorkcostQuota[]} usage={usageCount}
                refPrices={snap.refPrices} checked={checked} active={activeKey} highlight={searchTerms} semantic={semantic}
                onToggle={toggle} onToggleAll={toggleAll} onOpen={(q) => jump("quota", selKey("quota", q.code))}
                onDelete={(q) => deleteQuotaRows([q])} />
            )}
            {tier === "resource" && (
              <ResourceTable rows={rows as WorkcostResource[]} checked={checked} active={activeKey} highlight={searchTerms} semantic={semantic}
                onToggle={toggle} onToggleAll={toggleAll} onOpen={(r) => jump("resource", selKey("resource", r.code))}
                onDelete={(r) => deleteResourceRows([r])} />
            )}
    </>
  );
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
        {/* 新建（2026-11-28）：在板块里直接新建一条，不用绕回造价模式；复制另存在详情头部。 */}
        <button type="button" className="cl-btn" data-testid="cost-create-btn"
          title="在板块里直接新建一条（不用绕回造价模式）" onClick={() => startCreate(tier)}>
          <Plus size={12} aria-hidden />新建{tierMeta.label}
        </button>
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
        {tier === "quota" && (
          <FacetSelect label="筛选定额地区" allLabel="全部地区" value={quotaF.region ?? ""} facets={quotaRegions} emptyLabel="未标地区"
            testid="cost-select-quota-region" onChange={(v) => setQuotaF({ ...quotaF, region: v })} />
        )}
        {/* 状态筛选（2026-11-28）：把「该处理的那一类行」一键筛出来，就地改或勾选批量改。 */}
        <select className="cl-select" aria-label="筛选状态" data-testid="cost-select-state" value={stateOf ?? ""}
          onChange={(e) => setFilterState((e.target.value || "") as RowState)}>
          <option value="">全部状态</option>
          {STATE_LABELS[tier].map((s) => <option key={s.value} value={s.value}>{s.label}</option>)}
        </select>
        {/* 定额地区回填（2026-11-28）：没标地区的定额按上游清单项目所在的省补上，先预览后落库。 */}
        {tier === "quota" && typeof app.WorkcostQuotaRegionBackfill === "function" && (
          <button type="button" className="cl-btn" data-testid="cost-region-open"
            title="没标地区的定额，按引用它的清单项目所在的省补上（只被一个省引用的才补；先预览，确认后才写）"
            onClick={() => setRegionOpen(true)}>
            <Globe size={12} aria-hidden />回填地区
          </button>
        )}
        <span className="cl-spacer" />
        {checkedKeys.length > 0 && (
          <>
            {/* 勾选批量改维度（2026-11-28）：专业 / 分部 / 地区 / 分类 / 期数，逐行回报成败。 */}
            <button type="button" className="cl-btn" data-testid="cost-batch-open"
              title="把勾选的行批量改成同一个专业 / 分部 / 地区 / 分类 / 期数"
              onClick={() => setBatchOpen(true)}>
              <ClipboardList size={12} aria-hidden />批量修改 {checkedKeys.length}
            </button>
            <button type="button" className="cl-btn cl-btn-danger-ghost" data-testid="cost-delete-checked" onClick={deleteChecked}>
              <Trash2 size={12} aria-hidden />删除所选 {checkedKeys.length}
            </button>
          </>
        )}
      </div>
      <div className="cl-list">
        {usedCodesLoading ? (
          <div className="cl-state" role="status" data-testid="cost-list-wait">
            <Loader2 size={16} className="cl-spin" aria-hidden />
            <span className="cl-state-title">正在核对哪些工料机被定额引用…</span>
          </div>
        ) : matched.length === 0 && semRows.length === 0 ? (
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
            {matched.length > 0 && (
              <>
                {renderTables(view.shown)}
            {view.hidden > 0 && (
              <div className="cl-more">
                <button type="button" className="cl-btn" onClick={() => setCap(cap + ROW_CAP)}>
                  再显示 {Math.min(ROW_CAP, view.hidden)} 条（余 {view.hidden} 条）
                </button>
              </div>
            )}
              </>
            )}
            {semRows.length > 0 && (
              <section className="cl-sem" data-testid="cost-semantic" aria-label="语义相近的条目">
                <div className="cl-sem-head">
                  名称不含「{keyword}」、意思相近
                  <span>语义检索 {semRows.length} 条{semRows.length > SEM_CAP ? `，显示前 ${SEM_CAP} 条` : ""}</span>
                </div>
                {renderTables(semRows.slice(0, SEM_CAP), true)}
              </section>
            )}
          </>
        )}
      </div>
    </>
  );

  const reportDirty = (d: boolean) => { dirtyEdit.current = d; };
  const cancelCreate = () => setCreating(null);
  const detailPane = (
    <>
      <div className="cl-col-head">
        <span className="cl-col-head-title">详情</span>
        {narrow && (
          <button type="button" className="cl-btn cl-btn-ghost" onClick={() => setPane("list")}>返回列表</button>
        )}
      </div>
      <div className="cl-detail-scroll">
        {creating ? (
          // 新建 / 复制另存（2026-11-28）：右栏换成新建表单；新建成功后重读并定位到新那一行。
          <article className="cl-detail" data-testid="cost-create">
            <header className="cl-detail-head">
              <div className="cl-kicker">{TIER_LABEL[creating.tier]} · {creating.base ? "复制另存" : "新建"}</div>
              <h2 className="cl-detail-title">
                {creating.base ? `以「${creating.base.title}」为底新建一条` : `新建一条${TIER_LABEL[creating.tier]}`}
              </h2>
            </header>
            {creating.tier === "bill" && (
              <BillEdit mode="create" item={blankBill()} projects={snap.projects} specialtyOptions={billSpecNames} dicts={editDicts}
                onDirty={reportDirty} onSaved={onEditSaved}
                onCreated={(s) => void onCreated(selKey("bill", s.id))} onCancel={cancelCreate} />
            )}
            {creating.tier === "quota" && (
              <QuotaEdit mode={creating.base ? "duplicate" : "create"} quota={(creating.base as WorkcostQuota | undefined) ?? blankQuota()}
                specialtyOptions={quotaSpecNames} chapterOptionsOf={chaptersOfSpec} resources={snap.resources} regionOptions={projectRegionNames}
                dicts={editDicts} onDirty={reportDirty} onSaved={onEditSaved}
                onCreated={(s) => void onCreated(selKey("quota", s.code))} onCancel={cancelCreate} />
            )}
            {creating.tier === "resource" && (
              <ResourceEdit mode={creating.base ? "duplicate" : "create"}
                resource={(creating.base as WorkcostResource | undefined) ?? blankResource()}
                regionOptions={resourceRegionNames} periodOptions={resourcePeriodNames}
                categoryOptions={creating.base ? categoriesOfKind((creating.base as WorkcostResource).kind) : []}
                dicts={editDicts} onDirty={reportDirty} onSaved={onEditSaved}
                onCreated={(s) => void onCreated(selKey("resource", s.code))} onCancel={cancelCreate} />
            )}
          </article>
        ) : selectedBill ? (
          <BillDetail item={selectedBill} refPrices={snap.refPrices} onOpenQuota={openQuota} onOpenResource={openResource}
            specialtyOptions={billSpecNames} dicts={editDicts} onDirty={reportDirty} onSaved={onEditSaved} />
        ) : selectedQuota ? (
          <QuotaDetail quota={selectedQuota} usage={usageIndex.get(selectedQuota.code) ?? []} onOpenBill={openBill} onOpenResource={openResource}
            specialtyOptions={quotaSpecNames} chapterOptionsOf={chaptersOfSpec} resources={snap.resources} regionOptions={projectRegionNames}
            dicts={editDicts} onDirty={reportDirty} onSaved={onEditSaved} onDuplicate={(q) => startCreate("quota", q)} />
        ) : selectedResource ? (
          <ResourceDetail resource={selectedResource} onOpenQuota={openQuota}
            regionOptions={resourceRegionNames} periodOptions={resourcePeriodNames} categoryOptions={categoriesOfKind(selectedResource.kind)}
            dicts={editDicts} onDirty={reportDirty} onSaved={onEditSaved} onDuplicate={(r) => startCreate("resource", r)} />
        ) : tier === "bill" && currentProject ? (
          <ProjectCard project={currentProject} onDeleted={deleteProject}
            specialtyOptions={projectSpecNames} regionOptions={projectRegionNames} periodOptions={projectPeriodNames}
            dicts={editDicts} onDirty={reportDirty} onSaved={onEditSaved} />
        ) : (
          <DetailState phase="idle" />
        )}
      </div>
    </>
  );

  // 窄窗放不下左栏：两级上下文各一个下拉（清单：专业 → 项目；定额：专业 → 分部；
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
  // 第二级只在选了第一级后出现（分部在专业内、分类在类别内才有意义；「类别待整理」下面没有分类）；清单的项目级恒在。
  const firstChosen = tier === "quota" ? quotaF.specialty : resourceF.kind;
  const secondChosen = tier === "bill" || (!!firstChosen && firstChosen !== OUTSIDE);
  const secondOptions = tier === "bill"
    ? [{
      value: "0", label: "全部项目",
    }, ...specProjects.map((p) => ({ value: String(p.id), label: `${p.name}（${projectMatch.get(p.id) ?? p.itemCount}）` }))]
    : tier === "quota"
      ? [{ value: "", label: "全部分部" }, ...facetOptions(chapters, "未归分部")]
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
            {dictsOk && (
              // 检索别名（2026-11-28）：行话 / 缩写 ↔ 标准说法，搜索时自动双向展开。
              <button type="button" className="cl-btn" aria-label="检索别名" data-testid="cost-alias-open"
                title="行话 ↔ 标准说法（如 砼 → 混凝土），搜索时自动双向展开" onClick={() => setAliasOpen(true)}>
                <Wand2 size={12} aria-hidden />别名
              </button>
            )}
            {semDown && (
              // 本地语义模型不可用：这次搜索只有关键词匹配。原因放悬停明细；「重试」给模型
              // 重载之后用（重新检索一次，好了提示自己消失）。
              <span className="cl-search-warn" role="status" data-testid="cost-semantic-down" title={semDown}>
                <AlertCircle size={13} aria-hidden />
                <span>本地语义模型不可用，只按关键词匹配</span>
                <button
                  type="button"
                  className="cl-icon-btn"
                  aria-label="重试语义检索"
                  title="重新检索一次（模型重载后点这里，好了提示会消失）"
                  onClick={() => setSemNonce((n) => n + 1)}
                >
                  <RefreshCw size={12} aria-hidden />
                </button>
              </span>
            )}
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
              // 2026-11-28：逐项点开即筛出对应的行（在板块里改或批量改）；整理提示词仍可复制给造价模式。
              <Popover
                trigger="click"
                placement="bottomLeft"
                open={pendingOpen}
                onOpenChange={setPendingOpen}
                destroyOnHidden
                content={(
                  <div className="cl-pending-pop" data-testid="cost-pending-pop">
                    <p className="cl-note">点一项直接筛出对应的行，就地改或勾选批量改：</p>
                    <ul>
                      {pendingItems.map((it) => (
                        <li key={it.key}>
                          <button
                            type="button"
                            className="cl-pending-item"
                            data-testid={`cost-pending-${it.key}`}
                            onClick={() => { setPendingOpen(false); applyPending(it.key); }}
                          >
                            <span>{it.label}</span><b>{it.count}</b>
                          </button>
                        </li>
                      ))}
                    </ul>
                    <button
                      type="button"
                      className="cl-btn cl-btn-ghost"
                      data-testid="cost-pending-copy"
                      onClick={() => void copyPending()}
                    >
                      <Copy size={12} aria-hidden />
                      {copyState === "done" ? "已复制提示词" : copyState === "failed" ? "复制失败" : "复制整理提示词（交给造价模式）"}
                    </button>
                  </div>
                )}
              >
                <button type="button" className="cl-pending" data-testid="cost-pending" title={pendingSummary(pending)}>
                  待整理 {pending.total}
                </button>
              </Popover>
            )}
            <IndexStatus status={indexStatus} onBackfill={backfillIndex} />
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
                secondLabel={tier === "bill" ? "筛选项目" : tier === "quota" ? "筛选分部" : "筛选分类"} />
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
                projects={specProjects} projectCounts={projectMatch} billFilters={billF}
                billSpecialties={billSpecFacets} onBillSpecialty={setBillSpecialty}
                onProject={(id) => setBillF({ ...billF, projectId: id, division: "" })}
                onDeleteProject={deleteProject}
                specialties={specialties} chapters={chapters} quotaRegions={quotaRegions} quotaFilters={quotaF}
                onSpecialty={(specialty) => setQuotaF({ ...quotaF, specialty, chapter: "" })}
                onChapter={(chapter) => setQuotaF({ ...quotaF, chapter })}
                onQuotaRegion={(region) => setQuotaF({ ...quotaF, region })}
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
      {batchOpen && (
        <BatchEditDialog tier={tier} rows={checkedRows} open dicts={editDicts} chaptersOf={chaptersOfSpec} divisions={divisions}
          onClose={() => setBatchOpen(false)} onDone={(summary) => { showNotice(summary); void load(true); }} />
      )}
      {regionOpen && (
        <RegionBackfillDialog open onClose={() => setRegionOpen(false)}
          onDone={(summary) => { showNotice(summary); void load(true); }} />
      )}
      {aliasOpen && (
        <AliasDialog open aliases={dicts.filter((d) => d.kind === "alias")} onClose={() => setAliasOpen(false)} onChanged={loadDicts} />
      )}
    </div>
  );
}

export default CostLibraryPage;
