import { useCallback, useEffect, useMemo, useState, type ReactNode } from "react";
import {
  BookOpen, Box, Calculator, ChevronRight, CloudUpload, Coins, FileSpreadsheet, FolderPlus,
  FolderTree, Gauge, PieChart, Plus, Shield, TrendingUp,
} from "../gaea/icons";
import { app } from "../gaea/lib/bridge";
import type { CostCategory, CostSummary, FilePickResult, PriceSource } from "../gaea/lib/types";
import { CostLibraryView } from "../gaea/components/CostLibraryView";
import { CostEntryModal } from "../gaea/components/memoryhub/CostEntryModal";
import { CostImportModal } from "../gaea/components/memoryhub/CostImportModal";
import { PriceSourcesPanel } from "../gaea/components/memoryhub/PriceSourcesPanel";
import { PriceSourcesRepository } from "../gaea/components/memoryhub/PriceSourcesRepository";
import { CostInquiryPanel } from "../gaea/components/memoryhub/CostInquiryPanel";
import { CostProjectsView } from "../gaea/components/memoryhub/CostProjectsView";
import { CostIndicatorsView } from "../gaea/components/memoryhub/CostIndicatorsView";
import { CostNotesView } from "../gaea/components/memoryhub/CostNotesView";
import { CostGraphView } from "../gaea/components/memoryhub/CostGraphView";
import "../gaea/styles.css";
import "../gaea/tailwind.css";
import "../gaea/components/memoryhub/hub.css";

/**
 * CostLibraryPage — 「造价数据库」一级板块（2026-09-03 化繁为简重构图）。
 *
 * IA 定调（v4.50）：平级 8 模块收敛为 6——
 * - 概览：数据概览 / 关联图谱 双视图（图谱从平级模块降为概览的分析镜头）；
 * - 价格数据：价格源 / 价格仓库 / 询价库 三段同域（询价库从「成本条目」
 *   隐藏的第三个 icon 视图升格为一等子页——询价飞轮本就是价格数据域）；
 * - 重组零删减：所有功能保留，只改归属与可见性。
 * 数据模型不变：综合单价=一级，人材机=二级组成。
 */
type CostModule = "overview" | "entries" | "projects" | "prices" | "refs" | "notes";
type OverviewView = "data" | "graph";
type PriceView = "sources" | "repository" | "inquiry";

const MODULES: { key: CostModule; label: string; icon: ReactNode; hint: string }[] = [
  { key: "overview", label: "概览", icon: <Gauge size={14} />, hint: "库规模 · 人材机构成 · 数据健康 · 关联图谱" },
  { key: "entries", label: "成本条目", icon: <Coins size={14} />, hint: "分类树 + 列表/表格管理" },
  { key: "projects", label: "测算项目", icon: <Calculator size={14} />, hint: "报价/测算工作 · 版本留痕 · 沉淀回库" },
  { key: "prices", label: "价格数据", icon: <CloudUpload size={14} />, hint: "价格源 · 价格仓库 · 询价库" },
  { key: "refs", label: "造价参考", icon: <TrendingUp size={14} />, hint: "案例分位数对标（不落表实时聚合）" },
  { key: "notes", label: "复盘笔记", icon: <BookOpen size={14} />, hint: "结论/边界/风险/证据沉淀判断" },
];

const PRICE_VIEWS: { key: PriceView; label: string }[] = [
  { key: "sources", label: "价格源" },
  { key: "repository", label: "价格仓库" },
  { key: "inquiry", label: "询价库" },
];

interface CostOverviewStats {
  total: number;
  missingPrice: number;
  draft: number;
  categoryCount: number;
  sourceCount: number;
  specialtyCount: number; // 综合单价下的专业数
  divisionCount: number; // 分部数
  componentTotal: number; // 人材机组成行总数
  laborSum: number; // 人工费合计
  materialSum: number;
  machineSum: number;
  coveragePct: number; // 引用完备度
  recent: CostSummary[];
  pending: CostSummary[];
}

const EMPTY_STATS: CostOverviewStats = {
  total: 0,
  missingPrice: 0,
  draft: 0,
  categoryCount: 0,
  sourceCount: 0,
  specialtyCount: 0,
  divisionCount: 0,
  componentTotal: 0,
  laborSum: 0,
  materialSum: 0,
  machineSum: 0,
  coveragePct: 100,
  recent: [],
  pending: [],
};

function countCategories(nodes: CostCategory[] | undefined): number {
  let n = 0;
  for (const c of nodes ?? []) {
    n += 1 + countCategories(c.children);
  }
  return n;
}

/** 段切换小胶囊（价格数据三段 / 概览双视图共用样式）。 */
function SegChip({ active, onClick, children }: { active: boolean; onClick: () => void; children: ReactNode }) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-current={active ? "true" : undefined}
      className={`px-2.5 h-6 rounded-full text-[11px] transition-colors ${
        active ? "bg-accent text-white" : "bg-bg-elev text-fg-faint hover:text-fg border border-border"
      }`}
    >
      {children}
    </button>
  );
}

export function CostLibraryPage() {
  const [module, setModule] = useState<CostModule>("overview");
  const [overviewView, setOverviewView] = useState<OverviewView>("data");
  const [priceView, setPriceView] = useState<PriceView>("sources");
  const [stats, setStats] = useState<CostOverviewStats>(EMPTY_STATS);
  const [loading, setLoading] = useState(true);
  const [entryOpen, setEntryOpen] = useState(false);
  const [importFile, setImportFile] = useState<FilePickResult | null>(null);

  const loadStats = useCallback(async () => {
    setLoading(true);
    try {
      const [entries, categories, sources] = await Promise.all([
        app.CostSearch("", "", "").catch(() => [] as CostSummary[]),
        app.CostCategories().catch(() => [] as CostCategory[]),
        app.PriceSources().catch(() => [] as PriceSource[]),
      ]);
      const list = entries ?? [];
      const sorted = [...list].sort((a, b) => (b.updatedAt ?? "").localeCompare(a.updatedAt ?? ""));
      const missingPrice = list.filter((e) => e.price <= 0).length;
      const draft = list.filter((e) => e.status === "草稿").length;
      // 专业/分部：综合单价 → 专业 → 分部 三级树统计。
      let specialtyCount = 0;
      let divisionCount = 0;
      for (const root of categories ?? []) {
        if (root.name !== "综合单价") continue;
        for (const sp of root.children ?? []) {
          specialtyCount++;
          divisionCount += (sp.children ?? []).length;
        }
      }
      const pending = sorted.filter((e) => e.price <= 0 || e.status === "草稿").slice(0, 8);
      const covered = Math.max(0, list.length - missingPrice - draft);
      setStats({
        total: list.length,
        missingPrice,
        draft,
        categoryCount: countCategories(categories ?? []),
        sourceCount: (sources ?? []).length,
        specialtyCount,
        divisionCount,
        componentTotal: list.reduce((s, e) => s + (e.componentCount ?? 0), 0),
        laborSum: list.reduce((s, e) => s + (e.laborFee ?? 0), 0),
        materialSum: list.reduce((s, e) => s + (e.materialFee ?? 0), 0),
        machineSum: list.reduce((s, e) => s + (e.machineFee ?? 0), 0),
        coveragePct: list.length ? Math.round((covered / list.length) * 100) : 100,
        recent: sorted.slice(0, 6),
        pending,
      });
    } catch {
      setStats(EMPTY_STATS);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadStats();
  }, [loadStats]);

  const priceText = useMemo(() => {
    const fmt = new Intl.NumberFormat("zh-CN", { maximumFractionDigits: 2 });
    return (p: number) => "¥" + fmt.format(p);
  }, []);

  const pickImport = useCallback(async () => {
    try {
      const files = await app.PickFiles();
      const f = files?.[0];
      if (f) setImportFile(f);
    } catch {
      // 原生对话框不可用时静默
    }
  }, []);

  return (
    <div className="h-full flex flex-col min-h-0 text-[12.5px]">
      {/* 顶栏：造价数据库标识 + 架构标签 + 快捷动作 */}
      <div className="shrink-0 flex items-center gap-3 px-5 h-12 border-b border-border-soft/70 bg-bg-elev/40">
        <span className="flex items-center gap-2 text-fg font-semibold text-[14px] tracking-tight">
          <span className="w-6 h-6 rounded-lg bg-accent/15 text-accent inline-flex items-center justify-center">
            <Coins size={14} />
          </span>
          造价数据库
        </span>
        <span className="hidden md:inline-flex items-center gap-1.5 px-2 h-5 rounded-md border border-border/80 text-[10.5px] text-fg-faint">
          <PieChart size={10} />
          综合单价一级 · 人材机二级
        </span>
        <span className="text-fg-faint text-[11.5px] hidden lg:inline">
          本地造价数据与经验沉淀 · 供方案测算与预结算复用
        </span>
        <div className="ml-auto flex items-center gap-2">
          <button
            type="button"
            className="inline-flex items-center gap-1.5 px-3 h-7 rounded-lg border border-border text-fg-dim hover:text-fg hover:border-accent/60 hover:bg-accent/5 transition-colors"
            onClick={pickImport}
            title="导入 Excel/CSV/PDF/图片报价单，预览确认后入库"
          >
            <CloudUpload size={13} />
            导入文件
          </button>
          <button
            type="button"
            className="inline-flex items-center gap-1.5 px-3 h-7 rounded-lg bg-accent text-white hover:opacity-90 hover:brightness-110 active:scale-[0.98] transition-all"
            onClick={() => setEntryOpen(true)}
            title="新建一条综合单价子目"
          >
            <Plus size={13} />
            新建条目
          </button>
        </div>
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
        {module === "entries" && <CostLibraryView />}
        {module === "projects" && <CostProjectsView onChanged={loadStats} />}
        {module === "refs" && <CostIndicatorsView />}
        {module === "notes" && <CostNotesView />}
        {module === "prices" && (
          <div className="h-full flex flex-col min-h-0">
            <div className="shrink-0 flex items-center gap-1.5 px-4 py-2 border-b border-border-soft/50" role="tablist" aria-label="价格数据子视图">
              {PRICE_VIEWS.map((v) => (
                <SegChip key={v.key} active={priceView === v.key} onClick={() => setPriceView(v.key)}>{v.label}</SegChip>
              ))}
              <span className="ml-auto text-fg-faint text-[10.5px] hidden md:inline">
                四源归一：信息价（价格源）· OCR 报价（导入）· 供应商比价 · 手动询价
              </span>
            </div>
            <div className="flex-1 min-h-0">
              {priceView === "sources" && <PriceSourcesPanel onChanged={loadStats} />}
              {priceView === "repository" && <PriceSourcesRepository />}
              {priceView === "inquiry" && <CostInquiryPanel />}
            </div>
          </div>
        )}
        {module === "overview" && overviewView === "graph" && <CostGraphView />}
        {module === "overview" && overviewView === "data" && (
          <div className="h-full overflow-y-auto px-5 py-4 space-y-3">
            {loading ? (
              <OverviewSkeleton />
            ) : stats.total === 0 ? (
              <GettingStarted onImport={pickImport} onNew={() => setEntryOpen(true)} onSources={() => setModule("prices")} />
            ) : (
              <>
                {/* 库规模 + 人材机构成（hero 占两列，非对称） */}
                <div className="grid grid-cols-1 xl:grid-cols-3 gap-3">
                  <section className="v3-panel rounded-2xl p-5 xl:col-span-2 relative overflow-hidden">
                    <div className="absolute -top-20 -right-16 w-72 h-72 rounded-full bg-accent/10 blur-3xl pointer-events-none" />
                    <div className="relative flex flex-wrap items-start justify-between gap-4">
                      <div>
                        <div className="text-[10.5px] tracking-wider text-fg-faint">库规模</div>
                        <div className="mt-1.5 flex items-baseline gap-2">
                          <span className="text-[34px] leading-none font-semibold text-fg tabular-nums tracking-tight">
                            {stats.total}
                          </span>
                          <span className="text-fg-faint text-[11.5px]">条综合单价子目</span>
                        </div>
                        <div className="mt-2.5 flex items-center gap-1.5 flex-wrap text-[11px] text-fg-faint">
                          <span className="inline-flex items-center gap-1 px-1.5 py-px rounded bg-bg-elev">
                            <FolderTree size={10} /> 专业 {stats.specialtyCount}
                          </span>
                          <span className="inline-flex items-center gap-1 px-1.5 py-px rounded bg-bg-elev">
                            <FolderPlus size={10} /> 分部 {stats.divisionCount}
                          </span>
                          <span className="inline-flex items-center gap-1 px-1.5 py-px rounded bg-bg-elev">
                            <Box size={10} /> 组成行 {stats.componentTotal}
                          </span>
                          <span className="inline-flex items-center gap-1 px-1.5 py-px rounded bg-bg-elev">
                            <CloudUpload size={10} /> 价格源 {stats.sourceCount}
                          </span>
                        </div>
                      </div>
                      <div className="text-right">
                        <div className="text-[10.5px] tracking-wider text-fg-faint">人材机构成（累计金额）</div>
                        <div className="mt-1.5 text-[11.5px] text-fg-faint tabular-nums">
                          <span className="text-sky-400/90">人工 {priceText(stats.laborSum)}</span>
                          <span className="mx-1.5 text-border">·</span>
                          <span className="text-emerald-400/90">材料 {priceText(stats.materialSum)}</span>
                          <span className="mx-1.5 text-border">·</span>
                          <span className="text-amber-400/90">机械 {priceText(stats.machineSum)}</span>
                        </div>
                      </div>
                    </div>
                    <CompositionBar
                      labor={stats.laborSum}
                      material={stats.materialSum}
                      machine={stats.machineSum}
                      className="relative mt-4"
                    />
                  </section>

                  {/* 数据健康 */}
                  <section className="v3-panel rounded-2xl p-5 flex flex-col">
                    <div className="flex items-center justify-between">
                      <span className="text-[12.5px] font-semibold text-fg">数据健康</span>
                      <Shield size={13} className={stats.missingPrice === 0 && stats.draft === 0 ? "text-ok" : "text-amber-400"} />
                    </div>
                    <div className="mt-3 space-y-2">
                      <HealthRow label="待补单价" value={String(stats.missingPrice)} tone={stats.missingPrice > 0 ? "warn" : "ok"} />
                      <HealthRow label="草稿待复核" value={String(stats.draft)} tone={stats.draft > 0 ? "warn" : "ok"} />
                      <HealthRow label="可放心引用" value={String(Math.max(0, stats.total - stats.missingPrice - stats.draft)) + " 条"} tone="ok" />
                    </div>
                    <div className="mt-auto pt-4">
                      <div className="flex justify-between items-center text-[10.5px] text-fg-faint mb-1.5">
                        <span>引用完备度</span>
                        <span className="tabular-nums">{stats.coveragePct}%</span>
                      </div>
                      <div className="h-1.5 rounded-full bg-bg-elev overflow-hidden">
                        <div
                          className="h-full rounded-full bg-accent transition-all duration-500"
                          style={{ width: `${stats.coveragePct}%` }}
                        />
                      </div>
                    </div>
                  </section>
                </div>

                {/* 最近更新（带人材机 mini 条）+ 快捷入口 */}
                <div className="grid grid-cols-1 xl:grid-cols-3 gap-3">
                  <section className="v3-panel rounded-2xl p-4 xl:col-span-2">
                    <div className="flex items-center justify-between mb-2">
                      <span className="text-fg text-[12.5px] font-semibold">最近更新</span>
                      <button
                        type="button"
                        className="text-[11px] text-accent hover:opacity-80 inline-flex items-center gap-0.5"
                        onClick={() => setModule("entries")}
                      >
                        查看全部 <ChevronRight size={10} />
                      </button>
                    </div>
                    {stats.recent.length === 0 ? (
                      <EmptyHint text="还没有条目，先导入资料或新建条目" />
                    ) : (
                      <ul className="space-y-1">
                        {stats.recent.map((e) => (
                          <li
                            key={e.name}
                            className="flex items-center gap-2 px-2.5 py-2 rounded-xl bg-bg-elev/50 hover:bg-bg-elev transition-colors"
                            title={e.categoryPath || e.category || ""}
                          >
                            <Coins size={12} className="text-amber-400 shrink-0" />
                            <span className="min-w-0 flex-1">
                              <span className="block truncate text-fg-dim text-[12px] leading-tight">{e.title}</span>
                              <span className="block truncate text-fg-faint text-[10px] mt-0.5">
                                {e.categoryPath || e.category || "未分类"}
                              </span>
                            </span>
                            {e.laborFee || e.materialFee || e.machineFee ? (
                              <span className="shrink-0 w-16">
                                <CompositionBar labor={e.laborFee ?? 0} material={e.materialFee ?? 0} machine={e.machineFee ?? 0} className="!h-1" />
                              </span>
                            ) : null}
                            <span className="shrink-0 text-[11.5px] tabular-nums text-fg font-medium">
                              {priceText(e.price)}
                              {e.unit ? <span className="text-fg-faint font-normal">/{e.unit}</span> : ""}
                            </span>
                          </li>
                        ))}
                      </ul>
                    )}
                  </section>

                  {/* 快捷入口（只留动作与高频去处，其余走导航） */}
                  <section className="v3-panel rounded-2xl p-4">
                    <div className="flex items-center justify-between mb-2">
                      <span className="text-fg text-[12.5px] font-semibold">快捷入口</span>
                      <span className="text-[10.5px] text-fg-faint">继续最近工作</span>
                    </div>
                    <div className="space-y-1.5">
                      <QuickAction
                        label="导入资料"
                        hint="Excel / CSV / PDF / 图片报价单"
                        icon={<FileSpreadsheet size={14} className="text-sky-400" />}
                        onClick={pickImport}
                      />
                      <QuickAction
                        label="新建条目"
                        hint="手动录入综合单价子目"
                        icon={<Plus size={14} className="text-emerald-400" />}
                        onClick={() => setEntryOpen(true)}
                      />
                      <QuickAction
                        label="价格数据"
                        hint="价格源 · 价格仓库 · 询价库"
                        icon={<CloudUpload size={14} className="text-amber-400" />}
                        onClick={() => setModule("prices")}
                      />
                      <QuickAction
                        label="测算项目"
                        hint="报价/测算工作 · 版本留痕 · 沉淀回库"
                        icon={<Calculator size={14} className="text-accent" />}
                        onClick={() => setModule("projects")}
                      />
                    </div>
                  </section>
                </div>
              </>
            )}
          </div>
        )}
      </div>

      {/* 新建/导入弹窗（无确认不落库） */}
      <CostEntryModal open={entryOpen} editing={null} onClose={() => setEntryOpen(false)} onSaved={() => { setEntryOpen(false); loadStats(); }} />
      {importFile && (
        <CostImportModal
          open
          path={importFile.path}
          fileName={importFile.name}
          onClose={() => setImportFile(null)}
          onImported={() => { setImportFile(null); loadStats(); }}
        />
      )}
    </div>
  );
}

// ── 人材机构成条（人工/材料/机械 占比）────────────────────────────
function CompositionBar({
  labor,
  material,
  machine,
  className = "",
}: {
  labor: number;
  material: number;
  machine: number;
  className?: string;
}) {
  const total = labor + material + machine;
  if (total <= 0) {
    return <div className={`text-[10.5px] text-fg-faint ${className}`}>暂无组成数据（人工/材料/机械合计为空）</div>;
  }
  const seg = (v: number, cls: string, label: string) =>
    v > 0 ? (
      <div
        className={`h-full ${cls} transition-all duration-500`}
        style={{ width: `${Math.max(2, (v / total) * 100)}%` }}
        title={`${label} ${v.toFixed(2)}`}
      />
    ) : null;
  return (
    <div
      className={`flex h-2 rounded-full overflow-hidden bg-bg-elev ${className}`}
      role="img"
      aria-label={`人材机构成：人工 ${labor.toFixed(2)}，材料 ${material.toFixed(2)}，机械 ${machine.toFixed(2)}`}
    >
      {seg(labor, "bg-sky-400/80", "人工")}
      {seg(material, "bg-emerald-400/80", "材料")}
      {seg(machine, "bg-amber-400/80", "机械")}
    </div>
  );
}

function HealthRow({ label, value, tone }: { label: string; value: string; tone: "ok" | "warn" }) {
  return (
    <div className="flex items-center justify-between text-[11.5px]">
      <span className="text-fg-faint">{label}</span>
      <span className={`tabular-nums font-medium ${tone === "ok" ? "text-ok" : "text-amber-400"}`}>{value}</span>
    </div>
  );
}

function QuickAction({ label, hint, icon, onClick }: { label: string; hint: string; icon: ReactNode; onClick: () => void }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="v3-card is-interactive w-full px-2.5 py-2 flex items-center gap-2.5 text-left"
    >
      <span className="w-7 h-7 rounded-lg bg-bg-elev inline-flex items-center justify-center shrink-0">{icon}</span>
      <span className="min-w-0 flex-1">
        <span className="block text-fg text-[12px] font-medium leading-tight">{label}</span>
        <span className="block text-fg-faint text-[10.5px] truncate mt-0.5">{hint}</span>
      </span>
      <ChevronRight size={12} className="text-fg-faint shrink-0" />
    </button>
  );
}

// ── 首次使用引导（空库态）──────────────────────────────────────────
function GettingStarted({
  onImport,
  onNew,
  onSources,
}: {
  onImport: () => void;
  onNew: () => void;
  onSources: () => void;
}) {
  return (
    <div className="h-full flex items-center justify-center p-6">
      <div className="w-full max-w-2xl">
        <div className="text-center mb-5">
          <div className="mx-auto w-12 h-12 rounded-2xl bg-accent/15 text-accent inline-flex items-center justify-center">
            <Coins size={22} />
          </div>
          <h2 className="mt-3 text-[16px] font-semibold text-fg tracking-tight">造价数据库还是空的</h2>
          <p className="mt-1 text-[12px] text-fg-faint">
            按「综合单价一级 · 人材机二级」沉淀本地造价数据，三步就能开始
          </p>
        </div>
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
          <StepCard
            step="01"
            title="导入资料"
            desc="整本导入《市政成本测算手册》或报价单，自动识别专业/分部与组成行"
            icon={<FileSpreadsheet size={16} className="text-sky-400" />}
            onClick={onImport}
            cta="选择文件"
          />
          <StepCard
            step="02"
            title="新建子目"
            desc="手动录入综合单价，补充人工/材料/机械组成明细"
            icon={<Plus size={16} className="text-emerald-400" />}
            onClick={onNew}
            cta="新建条目"
          />
          <StepCard
            step="03"
            title="订阅价格源"
            desc="接入固定网站信息价，定时抓取刷新材料市场价"
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

function EmptyHint({ text }: { text: string }) {
  return <div className="py-8 text-center text-[11.5px] text-fg-faint">{text}</div>;
}

// ── 骨架屏（与概览布局同构）────────────────────────────────────────
function OverviewSkeleton() {
  return (
    <div className="space-y-3 animate-pulse">
      <div className="grid grid-cols-1 xl:grid-cols-3 gap-3">
        <div className="v3-panel rounded-2xl p-5 xl:col-span-2 h-36" />
        <div className="v3-panel rounded-2xl p-5 h-36" />
      </div>
      <div className="grid grid-cols-1 xl:grid-cols-3 gap-3">
        <div className="v3-panel rounded-2xl p-4 xl:col-span-2 h-52" />
        <div className="v3-panel rounded-2xl p-4 h-52" />
      </div>
    </div>
  );
}

export default CostLibraryPage;
