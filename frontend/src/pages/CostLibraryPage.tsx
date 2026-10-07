import { useCallback, useEffect, useState, type ReactNode } from "react";
import {
  ChevronRight, CloudUpload, Coins, FileSpreadsheet, Gauge, Layers, ListTree,
  PieChart, TrendingUp,
} from "../gaea/icons";
import { app } from "../gaea/lib/bridge";
import { CostLibraryView } from "../gaea/components/CostLibraryView";
import { PriceSourcesPanel } from "../gaea/components/memoryhub/PriceSourcesPanel";
import { PriceSourcesRepository } from "../gaea/components/memoryhub/PriceSourcesRepository";
import { CostInquiryPanel } from "../gaea/components/memoryhub/CostInquiryPanel";
import { CostIndicatorsView } from "../gaea/components/memoryhub/CostIndicatorsView";
import { WorkcostResourceView } from "../gaea/components/memoryhub/WorkcostResourceView";
import { WorkcostComposeView } from "../gaea/components/memoryhub/WorkcostComposeView";
import { WorkcostBillView } from "../gaea/components/memoryhub/WorkcostBillView";
import { CostGraphView } from "../gaea/components/memoryhub/CostGraphView";
import "../gaea/styles.css";
import "../gaea/tailwind.css";
import "../gaea/components/memoryhub/hub.css";

/**
 * CostLibraryPage — 「造价数据库」一级板块（v4.459 全新设计：工料法成本数据库）。
 *
 * 用户定调（2026-10-07）：「工料机是基本数据，综合单价分析表靠工料机和消耗定额
 * 核算出来」+「对造价数据库进行全新设计，该删除就删除」。造价数据库的主人
 * 是**清单测算**——五表模版（费用汇总/综合单价/工料机价格）的软件形态：
 *
 *   清单（首屏：分部分项工程量清单 + 项目费率，v4.460 定调「数据库不是
 *   计算器」——录入费率/清单/模版，不做加总）→ 单价分析（定额 × 资源价）→
 *   资源库（工料机主数据 + 调价）→ 价格数据（信息价飞轮 + 资料条目）→
 *   造价参考（分位对标）→ 概览（工料法仪表盘 / 关联图谱）。
 *
 * 退役（UI 删除；Go 绑定与数据表全保留，旧数据零损失）：
 * - 「测算项目」：v4.150 时代的旧五算项目体系，与清单测算工作台重叠，模块退役；
 * - 「复盘笔记」：独立模块退役（办公域已有记忆/笔记体系）；
 * - 旧「成本条目」（价格手册形态）降为「价格数据 → 资料条目」段——检索/导入/
 *   AI 解析全保留，它仍是资源库与询价的沉淀池，只是不再是板块主角。
 */
type CostModule = "bill" | "compose" | "resources" | "prices" | "refs" | "overview";
type OverviewView = "data" | "graph";
type PriceView = "sources" | "repository" | "inquiry" | "entries";

const MODULES: { key: CostModule; label: string; icon: ReactNode; hint: string }[] = [
  { key: "bill", label: "清单", icon: <ListTree size={14} />, hint: "分部分项工程量清单 · 项目费率（录入数据）" },
  { key: "compose", label: "单价分析", icon: <FileSpreadsheet size={14} />, hint: "消耗定额 · 工料机现算综合单价（只含人材机）· 导入/导出五表" },
  { key: "resources", label: "资源库", icon: <Layers size={14} />, hint: "工料机主数据 · 现行价/基准价 · 调价历史 · 存量资源化" },
  { key: "prices", label: "价格数据", icon: <CloudUpload size={14} />, hint: "价格源 · 价格仓库 · 询价库 · 资料条目" },
  { key: "refs", label: "造价参考", icon: <TrendingUp size={14} />, hint: "资料条目分位数对标（旧价格手册沉淀，不落表实时聚合）——与分部分项清单是两套数据" },
  { key: "overview", label: "概览", icon: <Gauge size={14} />, hint: "工料法规模 · 资料库速览 · 关联图谱" },
];

const PRICE_VIEWS: { key: PriceView; label: string }[] = [
  { key: "sources", label: "价格源" },
  { key: "repository", label: "价格仓库" },
  { key: "inquiry", label: "询价库" },
  { key: "entries", label: "资料条目" },
];

// 概览统计（工料法口径）：三层规模 + 资料库/价格源速览。
interface CostOverviewStats {
  resourceTotal: number;
  labor: number;
  material: number;
  machine: number;
  outsourced: number;
  quotaTotal: number;
  entryTotal: number;
  sourceTotal: number;
}

const EMPTY_STATS: CostOverviewStats = {
  resourceTotal: 0,
  labor: 0,
  material: 0,
  machine: 0,
  outsourced: 0,
  quotaTotal: 0,
  entryTotal: 0,
  sourceTotal: 0,
};

/** 段切换小胶囊（价格数据四段 / 概览双视图共用样式）。 */
function SegChip({ active, onClick, children }: { active: boolean; onClick: () => void; children: ReactNode }) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-current={active ? "true" : undefined}
      className={`px-2.5 h-6 rounded-full text-[11px] transition-colors ${
        active ? "bg-accent text-accent-fg" : "bg-bg-elev text-fg-faint hover:text-fg border border-border"
      }`}
    >
      {children}
    </button>
  );
}

export function CostLibraryPage() {
  // v4.459：默认落点=费用汇总（清单测算工作台首屏）。
  const [module, setModule] = useState<CostModule>("bill");
  const [overviewView, setOverviewView] = useState<OverviewView>("data");
  const [priceView, setPriceView] = useState<PriceView>("sources");
  const [stats, setStats] = useState<CostOverviewStats>(EMPTY_STATS);
  const [loading, setLoading] = useState(true);
  // 全源读取失败不装成「空库引导」——失败是读不到库，不是库是空的。
  const [statsFailed, setStatsFailed] = useState(false);

  // 概览统计只在进入概览时拉取（工作台是主路径，不为它付首屏请求）。
  const loadStats = useCallback(async () => {
    setLoading(true);
    setStatsFailed(false);
    try {
      const [resources, quotas, entries, sources] = await Promise.all([
        app.WorkcostResourceList("", "").catch(() => null),
        app.WorkcostQuotaList("", "").catch(() => null),
        app.CostSearch("", "", "").catch(() => null),
        app.PriceSources().catch(() => null),
      ]);
      if (resources === null && quotas === null && entries === null && sources === null) {
        setStats(EMPTY_STATS);
        setStatsFailed(true);
        return;
      }
      const res = resources ?? [];
      const byKind = (k: string) => res.filter((r) => r.kind === k).length;
      setStats({
        resourceTotal: res.length,
        labor: byKind("人工"),
        material: byKind("材料"),
        machine: byKind("机械"),
        outsourced: byKind("外委"),
        quotaTotal: (quotas ?? []).length,
        entryTotal: (entries ?? []).length,
        sourceTotal: (sources ?? []).length,
      });
    } catch {
      setStats(EMPTY_STATS);
      setStatsFailed(true);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (module === "overview") void loadStats();
  }, [module, loadStats]);

  return (
    <div className="h-full flex flex-col min-h-0 text-[12.5px]">
      {/* 顶栏：板块标识 + 工料法口径 */}
      <div className="shrink-0 flex items-center gap-3 px-5 h-12 border-b border-border-soft/70 bg-bg-elev/40">
        <span className="flex items-center gap-2 text-fg font-semibold text-[14px] tracking-tight">
          <span className="w-6 h-6 rounded-lg bg-accent/15 text-accent inline-flex items-center justify-center">
            <Coins size={14} />
          </span>
          造价数据库
        </span>
        <span className="hidden md:inline-flex items-center gap-1.5 px-2 h-5 rounded-md border border-border/80 text-[10.5px] text-fg-faint">
          <PieChart size={10} />
          工料法：工料机 × 消耗定额 → 综合单价（只含人材机）
        </span>
        <span className="text-fg-faint text-[11.5px] hidden lg:inline">
          录入清单 · 费率 · 模版——加总由五表活公式算
        </span>
      </div>

      {/* 模块导航：分段工作台（下划线指示，非圆角胶囊） */}
      <nav className="shrink-0 flex items-center gap-0.5 px-4 border-b border-border-soft/60" aria-label="造价数据库模块">
        {MODULES.map((m) => (
          <button
            key={m.key}
            type="button"
            onClick={() => setModule(m.key)}
            title={m.hint}
            aria-current={module === m.key ? "page" : undefined}
            className={`relative inline-flex items-center gap-1.5 px-3 h-9 text-[12px] transition-colors ${
              module === m.key
                ? "text-accent font-medium"
                : "text-fg-faint hover:text-fg hover:bg-bg-soft/50 rounded-t-lg"
            }`}
          >
            {m.icon}
            {m.label}
            {module === m.key && <span className="absolute left-2.5 right-2.5 bottom-0 h-0.5 rounded-full bg-accent" />}
          </button>
        ))}
        <div className="ml-auto flex items-center gap-1.5 pr-1">
          {module === "overview" ? (
            <>
              <SegChip active={overviewView === "data"} onClick={() => setOverviewView("data")}>数据概览</SegChip>
              <SegChip active={overviewView === "graph"} onClick={() => setOverviewView("graph")}>关联图谱</SegChip>
            </>
          ) : (
            <button
              type="button"
              className="inline-flex items-center px-2 h-6 rounded text-[11px] text-fg-faint hover:text-accent"
              onClick={() => { setModule("overview"); setOverviewView("data"); }}
              title="回到概览"
            >
              回到概览
            </button>
          )}
        </div>
      </nav>

      {/* 主区 */}
      <div className="flex-1 min-h-0">
        {module === "bill" && <WorkcostBillView />}
        {module === "compose" && <WorkcostComposeView />}
        {module === "resources" && <WorkcostResourceView />}
        {module === "refs" && <CostIndicatorsView />}
        {module === "prices" && (
          <div className="h-full flex flex-col min-h-0">
            <div className="shrink-0 flex items-center gap-1.5 px-4 py-2 border-b border-border-soft/50" role="tablist" aria-label="价格数据子视图">
              {PRICE_VIEWS.map((v) => (
                <SegChip key={v.key} active={priceView === v.key} onClick={() => setPriceView(v.key)}>{v.label}</SegChip>
              ))}
              <span className="ml-auto text-fg-faint text-[10.5px] hidden md:inline">
                信息价喂资源现行价 · 询价沉淀 · 旧价格手册归档为「资料条目」
              </span>
            </div>
            <div className="flex-1 min-h-0">
              {priceView === "sources" && <PriceSourcesPanel />}
              {priceView === "repository" && <PriceSourcesRepository />}
              {priceView === "inquiry" && <CostInquiryPanel />}
              {priceView === "entries" && <CostLibraryView />}
            </div>
          </div>
        )}
        {module === "overview" && overviewView === "graph" && <CostGraphView />}
        {module === "overview" && overviewView === "data" && (
          <div className="h-full overflow-y-auto px-5 py-4 space-y-3">
            {loading ? (
              <OverviewSkeleton />
            ) : statsFailed ? (
              <div className="v3-panel rounded-2xl p-10 flex flex-col items-center gap-3" data-testid="cost-stats-failed">
                <div className="text-[13px] text-fg">造价库读取失败</div>
                <div className="text-[11.5px] text-fg-faint">数据库暂时读不到，请重试；若持续失败请查看日志</div>
                <button
                  type="button"
                  onClick={() => void loadStats()}
                  className="px-3 h-7 rounded-full bg-accent text-accent-fg text-[11.5px]"
                >
                  重试
                </button>
              </div>
            ) : stats.resourceTotal === 0 && stats.quotaTotal === 0 ? (
              <GettingStarted
                onImportProject={() => setModule("compose")}
                onResources={() => setModule("resources")}
                onSources={() => setModule("prices")}
              />
            ) : (
              <>
                {/* 工料法三层 hero + 下一步 */}
                <div className="grid grid-cols-1 xl:grid-cols-3 gap-3">
                  <section className="v3-panel rounded-2xl p-5 xl:col-span-2 relative overflow-hidden">
                    <div className="absolute -top-20 -right-16 w-72 h-72 rounded-full bg-accent/10 blur-3xl pointer-events-none" />
                    <div className="relative flex flex-wrap items-start justify-between gap-4">
                      <div>
                        <div className="text-[10.5px] tracking-wider text-fg-faint">工料法数据库</div>
                        <div className="mt-1.5 flex items-baseline gap-2">
                          <span className="text-[34px] leading-none font-semibold text-fg tabular-nums tracking-tight">
                            {stats.resourceTotal}
                          </span>
                          <span className="text-fg-faint text-[11.5px]">条工料机资源 · {stats.quotaTotal} 条消耗定额</span>
                        </div>
                        <div className="mt-2.5 flex items-center gap-1.5 flex-wrap text-[11px] text-fg-faint">
                          <span className="inline-flex items-center gap-1 px-1.5 py-px rounded bg-bg-elev">
                            <Layers size={10} /> 资源 {stats.resourceTotal}
                          </span>
                          <span className="inline-flex items-center gap-1 px-1.5 py-px rounded bg-bg-elev">
                            <ListTree size={10} /> 定额 {stats.quotaTotal}
                          </span>
                          <span className="inline-flex items-center gap-1 px-1.5 py-px rounded bg-bg-elev">
                            <Coins size={10} /> 资料条目 {stats.entryTotal}
                          </span>
                          <span className="inline-flex items-center gap-1 px-1.5 py-px rounded bg-bg-elev">
                            <CloudUpload size={10} /> 价格源 {stats.sourceTotal}
                          </span>
                        </div>
                      </div>
                      <div className="text-right">
                        <div className="text-[10.5px] tracking-wider text-fg-faint">资源构成</div>
                        <div className="mt-1.5 text-[11.5px] text-fg-faint tabular-nums">
                          <span className="text-sky-400/90">人工 {stats.labor}</span>
                          <span className="mx-1.5 text-border">·</span>
                          <span className="text-emerald-400/90">材料 {stats.material}</span>
                          <span className="mx-1.5 text-border">·</span>
                          <span className="text-amber-400/90">机械 {stats.machine}</span>
                          <span className="mx-1.5 text-border">·</span>
                          <span className="text-amber-400/90">外委 {stats.outsourced}</span>
                        </div>
                      </div>
                    </div>
                    <KindBar
                      labor={stats.labor}
                      material={stats.material}
                      machine={stats.machine}
                      outsourced={stats.outsourced}
                      className="relative mt-4"
                    />
                    <p className="relative mt-3 text-[10.5px] text-fg-faint leading-relaxed">
                      综合单价 = Σ(消耗量 × 资源价)，只含人材机；管理费/利润/规费/税金在费用汇总的项目合计层单列。
                      资源一调价，引用它的全部综合单价同步重算。
                    </p>
                  </section>

                  {/* 下一步引导：按库里最缺的一环指路 */}
                  <section className="v3-panel rounded-2xl p-5 flex flex-col">
                    <span className="text-[12.5px] font-semibold text-fg">下一步</span>
                    {stats.quotaTotal === 0 ? (
                      <>
                        <div className="mt-3 text-[12px] text-fg">还差消耗定额</div>
                        <p className="mt-1.5 text-[11px] text-fg-faint leading-relaxed">
                          资源已就位（{stats.resourceTotal} 条），但还没有消耗定额——
                          没有定额就无法核算综合单价。导入一份五表成本测算工作簿，
                          工料机价格与综合单价两表会一次落成资源与定额。
                        </p>
                        <button
                          type="button"
                          onClick={() => setModule("compose")}
                          className="mt-auto pt-4 inline-flex items-center gap-1 text-[12px] text-accent hover:opacity-80"
                        >
                          去导入项目表 <ChevronRight size={12} />
                        </button>
                      </>
                    ) : (
                      <>
                        <div className="mt-3 text-[12px] text-fg">清单数据就绪</div>
                        <p className="mt-1.5 text-[11px] text-fg-faint leading-relaxed">
                          资源 {stats.resourceTotal} 条、定额 {stats.quotaTotal} 条都在位。
                          清单挂定额×工程量，费率随项目录入——导出五表工作簿（活公式）即得完整费用汇总。
                        </p>
                        <button
                          type="button"
                          onClick={() => setModule("bill")}
                          className="mt-auto pt-4 inline-flex items-center gap-1 text-[12px] text-accent hover:opacity-80"
                        >
                          查看分部分项清单 <ChevronRight size={12} />
                        </button>
                      </>
                    )}
                  </section>
                </div>

                {/* 快捷入口 */}
                <div className="grid grid-cols-1 xl:grid-cols-3 gap-3">
                  <section className="v3-panel rounded-2xl p-4 xl:col-span-2">
                    <div className="flex items-center justify-between mb-2">
                      <span className="text-fg text-[12.5px] font-semibold">快捷入口</span>
                      <span className="text-[10.5px] text-fg-faint">按工作流排序</span>
                    </div>
                    <div className="grid grid-cols-2 gap-1.5">
                      <QuickAction
                        label="分部分项清单"
                        hint="清单项目 · 工程量 · 费率录入"
                        icon={<ListTree size={14} className="text-accent" />}
                        onClick={() => setModule("bill")}
                      />
                      <QuickAction
                        label="导入项目表"
                        hint="五表工作簿 → 资源+定额"
                        icon={<FileSpreadsheet size={14} className="text-sky-400" />}
                        onClick={() => setModule("compose")}
                      />
                      <QuickAction
                        label="资源库"
                        hint="工料机主数据 · 调价"
                        icon={<Layers size={14} className="text-emerald-400" />}
                        onClick={() => setModule("resources")}
                      />
                      <QuickAction
                        label="资料条目"
                        hint={`旧价格手册 · ${stats.entryTotal} 条沉淀`}
                        icon={<Coins size={14} className="text-amber-400" />}
                        onClick={() => { setModule("prices"); setPriceView("entries"); }}
                      />
                    </div>
                  </section>
                  <section className="v3-panel rounded-2xl p-4 flex flex-col">
                    <span className="text-fg text-[12.5px] font-semibold">口径备忘</span>
                    <ul className="mt-2 space-y-1.5 text-[11px] text-fg-faint leading-relaxed list-disc pl-4">
                      <li>综合单价只含人材机，取费在费用汇总合计层跑一次</li>
                      <li>利润基数含不含规费按项目口径选（两份实测产物各不同）</li>
                      <li>零含量行是合法占位，不计费不告警</li>
                      <li>外委独立行类，三费归集并入材料</li>
                    </ul>
                  </section>
                </div>
              </>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

// ── 四类构成条（人工/材料/机械/外委）────────────────────────────
function KindBar({
  labor,
  material,
  machine,
  outsourced,
  className = "",
}: {
  labor: number;
  material: number;
  machine: number;
  outsourced: number;
  className?: string;
}) {
  const total = labor + material + machine + outsourced;
  if (total <= 0) {
    return <div className={`text-[10.5px] text-fg-faint ${className}`}>暂无资源构成数据</div>;
  }
  const seg = (v: number, cls: string, label: string) =>
    v > 0 ? (
      <div
        className={`h-full ${cls} transition-all duration-500`}
        style={{ width: `${Math.max(2, (v / total) * 100)}%` }}
        title={`${label} ${v} 条`}
      />
    ) : null;
  return (
    <div
      className={`flex h-2 rounded-full overflow-hidden bg-bg-elev ${className}`}
      role="img"
      aria-label={`资源构成：人工 ${labor}，材料 ${material}，机械 ${machine}，外委 ${outsourced}`}
    >
      {seg(labor, "bg-sky-400/80", "人工")}
      {seg(material, "bg-emerald-400/80", "材料")}
      {seg(machine, "bg-amber-400/80", "机械")}
      {seg(outsourced, "bg-violet-400/80", "外委")}
    </div>
  );
}

function QuickAction({ label, hint, icon, onClick }: { label: string; hint: string; icon: ReactNode; onClick: () => void }) {
  return (
    <button type="button" onClick={onClick} className="v3-card is-interactive w-full px-2.5 py-2 flex items-center gap-2.5 text-left">
      <span className="w-7 h-7 rounded-lg bg-bg-elev inline-flex items-center justify-center shrink-0">{icon}</span>
      <span className="min-w-0 flex-1">
        <span className="block text-fg text-[12px] font-medium leading-tight">{label}</span>
        <span className="block text-fg-faint text-[10.5px] truncate mt-0.5">{hint}</span>
      </span>
      <ChevronRight size={12} className="text-fg-faint shrink-0" />
    </button>
  );
}

// ── 空库引导（资源与定额都是 0）──────────────────────────────────
function GettingStarted({
  onImportProject,
  onResources,
  onSources,
}: {
  onImportProject: () => void;
  onResources: () => void;
  onSources: () => void;
}) {
  return (
    <div className="h-full flex items-center justify-center p-6">
      <div className="w-full max-w-2xl">
        <div className="text-center mb-5">
          <div className="mx-auto w-12 h-12 rounded-2xl bg-accent/15 text-accent inline-flex items-center justify-center">
            <Coins size={22} />
          </div>
          <h2 className="mt-3 text-[16px] font-semibold text-fg tracking-tight">工料法成本数据库还是空的</h2>
          <p className="mt-1 text-[12px] text-fg-faint">
            综合单价靠「工料机 × 消耗定额」核算出来——先让基本数据就位
          </p>
        </div>
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
          <StepCard
            step="01"
            title="导入项目表"
            desc="选一份五表成本测算工作簿，封面、清单、费率、资源、定额一次录入数据库"
            icon={<FileSpreadsheet size={16} className="text-sky-400" />}
            onClick={onImportProject}
            cta="去单价分析"
          />
          <StepCard
            step="02"
            title="维护资源库"
            desc="手工新增工料机资源，或把旧资料条目一键资源化归类入库"
            icon={<Layers size={16} className="text-emerald-400" />}
            onClick={onResources}
            cta="去资源库"
          />
          <StepCard
            step="03"
            title="订阅价格源"
            desc="接入固定网站信息价，定时抓取推进资源现行价"
            icon={<CloudUpload size={16} className="text-amber-400" />}
            onClick={onSources}
            cta="去配置"
          />
        </div>
      </div>
    </div>
  );
}

function StepCard({
  step,
  title,
  desc,
  icon,
  onClick,
  cta,
}: {
  step: string;
  title: string;
  desc: string;
  icon: ReactNode;
  onClick: () => void;
  cta: string;
}) {
  return (
    <button type="button" onClick={onClick} className="v3-card is-interactive p-4 text-left flex flex-col">
      <span className="text-[10px] tracking-widest text-fg-faint tabular-nums">{step}</span>
      <span className="mt-1.5 w-8 h-8 rounded-lg bg-bg-elev inline-flex items-center justify-center">{icon}</span>
      <span className="mt-2.5 text-fg text-[13px] font-semibold">{title}</span>
      <span className="mt-1 text-[11px] text-fg-faint leading-relaxed flex-1">{desc}</span>
      <span className="mt-3 text-[11.5px] text-accent inline-flex items-center gap-1">
        {cta} <ChevronRight size={11} />
      </span>
    </button>
  );
}

// ── 骨架屏（与概览布局同构）────────────────────────────────────────
function OverviewSkeleton() {
  return (
    <div className="space-y-3 animate-pulse">
      <div className="grid grid-cols-1 xl:grid-cols-3 gap-3">
        <div className="v3-panel rounded-2xl p-5 xl:col-span-2 h-40" />
        <div className="v3-panel rounded-2xl p-5 h-40" />
      </div>
      <div className="v3-panel rounded-2xl p-4 h-40" />
    </div>
  );
}

export default CostLibraryPage;
