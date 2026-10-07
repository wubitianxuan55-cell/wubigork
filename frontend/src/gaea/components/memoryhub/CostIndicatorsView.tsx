// CostIndicatorsView.tsx — 造价参考：**项目案例**（导入五表工作簿）+ 分位数对标。
//
// 用户定调（2026-10-07）：「把项目文件导入改在造价参考里面」——导入的项目
// 成本测算表是**对标案例**：封面/费率/清单数在此管理（可整体删除），其清单项
// 流入「分部分项清单」累计列表；下表对资料条目（沉淀的价格手册）实时聚合
// 分位数（不落表），与清单是两套数据。
import { useCallback, useEffect, useState } from "react";
import { BarChart3, ChevronDown, ChevronRight, FolderOpen, RefreshCw, Trash2, TrendingUp } from "../../icons";
import { app } from "../../lib/bridge";
import type { CostIndicator, WorkcostBillProject } from "../../lib/types";
import { RatesEditor } from "./WorkcostRatesEditor";

const fmtPrice = (p: number) => "¥" + new Intl.NumberFormat("zh-CN", { maximumFractionDigits: 2 }).format(p);
const pctText = (p: number) => (p > 0 ? `${Math.round(p * 10000) / 100}%` : "不计取");
const ghostBtn =
  "inline-flex items-center gap-1 px-2.5 h-7 rounded-lg border border-border text-fg-faint hover:text-fg hover:bg-bg-soft transition-colors text-[11.5px]";
const solidBtn =
  "inline-flex items-center gap-1 px-2.5 h-7 rounded-lg bg-accent text-accent-fg text-[11.5px] hover:opacity-90 transition-opacity disabled:opacity-50";
const iconBtn =
  "inline-flex items-center justify-center w-6 h-6 rounded-md border border-border text-fg-faint hover:text-err hover:bg-bg-soft transition-colors";

/**
 * CostIndicatorsView — 造价参考（项目案例 + 资料条目分位数对标）。
 */
export function CostIndicatorsView() {
  const [group, setGroup] = useState<"title" | "category">("title");
  const [rows, setRows] = useState<CostIndicator[]>([]);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const r = (await app.CostIndicators(group)) ?? [];
      setRows(r);
    } catch {
      setRows([]);
    } finally {
      setLoading(false);
    }
  }, [group]);

  useEffect(() => {
    load();
  }, [load]);

  return (
    <div className="h-full flex flex-col min-h-0 text-[12.5px]">
      <div className="shrink-0 flex items-center gap-3 px-5 h-12 border-b border-border-soft/60">
        <span className="text-fg font-semibold text-[13px] flex items-center gap-1.5">
          <BarChart3 size={14} className="text-accent" /> 造价参考
        </span>
        <span className="text-[11px] text-fg-faint hidden md:inline">
          项目案例（导入的成本测算表）+ 资料条目分位数对标 · 与分部分项清单是两套数据
        </span>
        <div className="ml-auto flex items-center gap-1.5">
          <ProjectImportButton onDone={load} />
          <div className="flex items-center rounded-lg border border-border bg-bg p-0.5 text-[11px]">
            <button
              type="button"
              className={`px-2.5 h-6 rounded-md transition-colors ${group === "title" ? "bg-accent text-accent-fg" : "text-fg-faint hover:text-fg"}`}
              onClick={() => setGroup("title")}
              title="按科目标题聚合"
            >
              按科目
            </button>
            <button
              type="button"
              className={`px-2.5 h-6 rounded-md transition-colors ${group === "category" ? "bg-accent text-accent-fg" : "text-fg-faint hover:text-fg"}`}
              onClick={() => setGroup("category")}
              title="按一级分类聚合"
            >
              按分类
            </button>
          </div>
          <button type="button" className={ghostBtn} onClick={load} title="刷新">
            <RefreshCw size={12} />
          </button>
        </div>
      </div>

      <div className="flex-1 min-h-0 overflow-y-auto px-5 py-4 space-y-3">
        {/* 项目案例：导入的成本测算表 */}
        <ProjectCasesSection />

        {/* 资料条目分位数对标 */}
        <section className="space-y-2">
          <div className="flex items-center gap-1.5">
            <TrendingUp size={13} className="text-fg-faint" />
            <span className="text-fg text-[12.5px] font-semibold">资料条目分位数对标</span>
            <span className="text-[10.5px] text-fg-faint">样本来自「价格数据 → 资料条目」的沉淀——与项目案例是两套数据</span>
          </div>
          {loading ? (
            <div className="space-y-2 animate-pulse">
              <div className="v3-panel rounded-xl h-10" />
              <div className="v3-panel rounded-xl h-48" />
            </div>
          ) : rows.length === 0 ? (
            <div className="v3-panel rounded-2xl py-12 text-center">
              <TrendingUp size={26} className="mx-auto text-fg-faint" />
              <div className="mt-3 text-[13px] text-fg font-medium">暂无对标样本</div>
              <p className="mt-1.5 text-[11.5px] text-fg-faint leading-relaxed max-w-md mx-auto">
                到「价格数据 → 资料条目」导入报价单或沉淀条目后，这里按科目给出
                样本数/极值/P25/中位数/均值/P75——样本越多，分位数越可信。
              </p>
            </div>
          ) : (
            <div className="v3-panel rounded-xl overflow-x-auto">
              <table className="w-full text-[11.5px]">
                <thead>
                  <tr className="text-left text-[10px] text-fg-faint border-b border-border-soft/50 bg-bg-soft/30">
                    <th className="py-2 px-3">{group === "title" ? "科目" : "一级分类"}</th>
                    <th className="py-2 px-2 w-16 text-right">样本数</th>
                    <th className="py-2 px-2 w-22 text-right">最小值</th>
                    <th className="py-2 px-2 w-22 text-right">P25</th>
                    <th className="py-2 px-2 w-24 text-right">中位数</th>
                    <th className="py-2 px-2 w-24 text-right">均值</th>
                    <th className="py-2 px-2 w-22 text-right">P75</th>
                    <th className="py-2 px-2 w-22 text-right">最大值</th>
                    <th className="py-2 px-2 w-12 text-right">单位</th>
                  </tr>
                </thead>
                <tbody>
                  {rows.map((r) => (
                    <tr key={r.key} className="border-b border-border-soft/25 last:border-0 hover:bg-bg-elev/30">
                      <td className="py-1.5 px-3 text-fg-dim font-medium">{r.key}</td>
                      <td className="py-1.5 px-2 text-right tabular-nums text-fg-faint">{r.samples}</td>
                      <td className="py-1.5 px-2 text-right tabular-nums">{fmtPrice(r.min)}</td>
                      <td className="py-1.5 px-2 text-right tabular-nums text-fg-dim">{fmtPrice(r.p25)}</td>
                      <td className="py-1.5 px-2 text-right tabular-nums text-fg font-semibold">{fmtPrice(r.median)}</td>
                      <td className="py-1.5 px-2 text-right tabular-nums text-accent">{fmtPrice(r.mean)}</td>
                      <td className="py-1.5 px-2 text-right tabular-nums text-fg-dim">{fmtPrice(r.p75)}</td>
                      <td className="py-1.5 px-2 text-right tabular-nums">{fmtPrice(r.max)}</td>
                      <td className="py-1.5 px-2 text-right text-fg-faint">{r.unit || "-"}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>
      </div>
    </div>
  );
}

// ── 项目案例（导入的五表工作簿）────────────────────────────────────

// ProjectImportButton 导入项目表（五表工作簿 → 项目案例+清单+费率+资源+定额）。
export function ProjectImportButton({ onDone }: { onDone: () => void }) {
  const [busy, setBusy] = useState(false);
  const [toast, setToast] = useState("");
  const run = useCallback(async () => {
    setBusy(true);
    setToast("");
    try {
      const picked = await app.PickFiles("xlsx");
      const path = picked?.[0]?.path;
      if (!path) {
        setBusy(false);
        return;
      }
      const bundle = await app.WorkcostProjectParse(path);
      const res = await app.WorkcostProjectApply(path);
      onDone();
      const warn = bundle.warnings.length ? `；${bundle.warnings.length} 条告警` : "";
      setToast(
        `已导入「${bundle.project || bundle.fileName}」：清单 ${bundle.items.length} 条、` +
          `资源 新建 ${res.resourceNew}/更新 ${res.resourceUpd}、定额 新建 ${res.quotaNew}/更新 ${res.quotaUpd}${warn}`,
      );
    } catch (e) {
      setToast(`导入失败：${e instanceof Error ? e.message : String(e)}`);
    } finally {
      setBusy(false);
    }
  }, [onDone]);
  return (
    <>
      <button type="button" className={solidBtn} onClick={() => void run()} disabled={busy} title="导入五表成本测算工作簿：项目案例+清单+费率+资源+定额一次录入">
        <FolderOpen size={12} /> 导入项目表
      </button>
      {toast && (
        <div className="fixed bottom-6 right-6 z-50 max-w-md v3-panel rounded-xl px-4 py-3 text-[11.5px] text-fg-dim shadow-lg">
          {toast}
          <button type="button" className={ghostBtn + " ml-2"} onClick={() => setToast("")}>关闭</button>
        </div>
      )}
    </>
  );
}

// ProjectCasesSection 项目案例列表：封面/费率摘要 + 展开（费率编辑）+ 删除。
export function ProjectCasesSection() {
  const [projects, setProjects] = useState<WorkcostBillProject[]>([]);
  const [expanded, setExpanded] = useState<number>(0);
  const [confirmDelete, setConfirmDelete] = useState<WorkcostBillProject | null>(null);
  const [busy, setBusy] = useState(false);
  const [toast, setToast] = useState("");

  const load = useCallback(async () => {
    try {
      setProjects((await app.WorkcostBillProjects()) ?? []);
    } catch {
      setProjects([]);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const deleteProject = useCallback(async () => {
    if (!confirmDelete) return;
    setBusy(true);
    try {
      const removed = await app.WorkcostBillProjectDelete(confirmDelete.id);
      setToast(`已删除项目「${confirmDelete.name}」（连带独占定额 ${removed} 条；共享定额与资源保留）`);
      setConfirmDelete(null);
      await load();
    } catch (e) {
      setToast(`删除失败：${e instanceof Error ? e.message : String(e)}`);
    } finally {
      setBusy(false);
    }
  }, [confirmDelete, load]);

  return (
    <section className="space-y-2" data-testid="cost-ref-projects">
      <div className="flex items-center gap-1.5">
        <FolderOpen size={13} className="text-fg-faint" />
        <span className="text-fg text-[12.5px] font-semibold">项目案例</span>
        <span className="text-[10.5px] text-fg-faint">
          导入的成本测算表（{projects.length} 个）· 费率是录入数据 · 清单项流入「分部分项清单」
        </span>
      </div>
      {projects.length === 0 ? (
        <div className="v3-panel rounded-xl px-4 py-6 text-center text-[11.5px] text-fg-faint">
          还没有项目案例。点右上「导入项目表」选一份五表成本测算工作簿。
        </div>
      ) : (
        <div className="space-y-2">
          {projects.map((p) => (
            <div key={p.id} className="v3-panel rounded-xl">
              <button
                type="button"
                className="w-full flex items-center gap-3 px-4 py-2.5 text-left hover:bg-bg-elev/30 transition-colors"
                onClick={() => setExpanded((x) => (x === p.id ? 0 : p.id))}
                title="展开/收起费率"
              >
                {expanded === p.id ? <ChevronDown size={13} className="text-fg-faint shrink-0" /> : <ChevronRight size={13} className="text-fg-faint shrink-0" />}
                <span className="min-w-0 flex-1">
                  <span className="block truncate text-[12px] text-fg font-medium">{p.name}</span>
                  <span className="block truncate text-[10px] text-fg-faint mt-0.5">
                    {p.itemCount} 条清单{p.location ? ` · ${p.location}` : ""}{p.duration ? ` · ${p.duration}` : ""}
                    {p.fileName ? ` · ${p.fileName}` : ""}
                  </span>
                </span>
                <span className="shrink-0 hidden md:inline text-[10px] text-fg-faint tabular-nums">
                  企管 {pctText(p.managementRate)} · 规费 {pctText(p.regulatoryRate)} · 利润 {pctText(p.profitRate)} · 税 {pctText(p.taxRate)}
                </span>
                <button
                  type="button"
                  className={iconBtn}
                  title="整体删除该项目：封面+清单+独占定额（共享定额与资源保留）"
                  onClick={(e) => {
                    e.stopPropagation();
                    setConfirmDelete(p);
                  }}
                >
                  <Trash2 size={11} />
                </button>
              </button>
              {expanded === p.id && (
                <div className="px-4 pb-3">
                  {p.pricing && (
                    <p className="text-[10.5px] text-fg-faint leading-relaxed mb-2" title={p.pricing}>
                      口径：{p.pricing}
                    </p>
                  )}
                  <RatesEditor
                    project={p}
                    onSaved={() => {
                      load();
                      setToast("费率已保存");
                    }}
                  />
                </div>
              )}
            </div>
          ))}
        </div>
      )}
      {toast && (
        <div className="v3-panel rounded-xl px-4 py-2 text-[11px] text-fg-dim">
          {toast}
          <button type="button" className={ghostBtn + " ml-2"} onClick={() => setToast("")}>关闭</button>
        </div>
      )}
      {confirmDelete && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" onClick={() => setConfirmDelete(null)}>
          <div className="v3-panel rounded-2xl p-5 space-y-3 w-[28rem]" onClick={(e) => e.stopPropagation()}>
            <div className="text-[13px] text-fg font-semibold">删除项目案例</div>
            <p className="text-[11.5px] text-fg-faint leading-relaxed">
              将删除项目「{confirmDelete.name}」的封面、全部分部分项清单，以及本项目导入且不被
              其他项目引用的定额。共享定额与工料机资源**保留**。该操作不可撤销。
            </p>
            <div className="flex items-center justify-end gap-2">
              <button type="button" className={ghostBtn} onClick={() => setConfirmDelete(null)} disabled={busy}>取消</button>
              <button
                type="button"
                className="inline-flex items-center gap-1 px-2.5 h-7 rounded-lg bg-err text-white text-[11.5px] hover:opacity-90 disabled:opacity-50"
                onClick={() => void deleteProject()}
                disabled={busy}
              >
                {busy ? "删除中…" : "确认删除"}
              </button>
            </div>
          </div>
        </div>
      )}
    </section>
  );
}

