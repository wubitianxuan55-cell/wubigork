// WorkcostComposeView.tsx — 综合单价分析表（工料法第②③层的核算面）。
//
// 口径（实测模版，旺牛矿业/什邡五表产物的「综合单价」表）：
//   综合单价 = Σ(消耗量 × 资源价)，**只含人材机**；
//   管理费/利润/规费/税金只在项目合计层跑一次（见 WorkcostFeePanel，不进本表）；
//   外委是独立行类，但三费归集并入材料桶（材料 = SUMIF(材料)+SUMIF(外委)）。
//
// 左侧列定额，右侧展示该定额的综合单价分析表：汇总行（人材机小计 + 综合单价）
// 下挂工料机明细行（含量 × 单价 = 金额），逐行可追溯。定额可新建/编辑/删除
// （QuotaModal——消耗定额是全局标准，本视图就是它的管理面）；项目级覆盖
// （调量/调价）直接体现在同一张表上——这正是「全局定额 + 项目级覆盖」的可见形态。
import { useCallback, useEffect, useMemo, useState } from "react";
import {
  Calculator, FileSpreadsheet, FolderOpen, ListTree, Pencil, Plus, RefreshCw, Search, Trash2, X,
} from "../../icons";
import { app } from "../../lib/bridge";
import type { WorkcostCompose, WorkcostQuota } from "../../lib/types";
import { QuotaModal } from "./WorkcostQuotaModal";

const fmtPrice = (p: number) => "¥" + new Intl.NumberFormat("zh-CN", { maximumFractionDigits: 2 }).format(p);
const fmtQty = (q: number) => new Intl.NumberFormat("zh-CN", { maximumFractionDigits: 4 }).format(q);
const fieldCls =
  "w-full bg-bg border border-border-soft rounded-md text-fg text-[12px] px-2.5 py-1.5 outline-none focus:border-accent transition-colors placeholder:text-fg-faint/50";
const ghostBtn =
  "inline-flex items-center gap-1 px-2.5 h-7 rounded-lg border border-border text-fg-faint hover:text-fg hover:bg-bg-soft transition-colors text-[11.5px]";
const solidBtn =
  "inline-flex items-center gap-1 px-2.5 h-7 rounded-lg bg-accent text-accent-fg text-[11.5px] hover:opacity-90 transition-opacity disabled:opacity-50";
const iconBtn =
  "inline-flex items-center justify-center w-6 h-6 rounded-md border border-border text-fg-faint hover:text-fg hover:bg-bg-soft transition-colors";

function kindTone(kind: string): string {
  switch (kind) {
    case "人工":
      return "text-sky-400 bg-sky-400/10";
    case "机械":
      return "text-violet-400 bg-violet-400/10";
    case "外委":
      return "text-amber-400 bg-amber-400/10";
    default:
      return "text-ok bg-ok/10";
  }
}

// emptyQuota 新建定额初值（编码/名称由用户填，落库时后端校验唯一）。
function emptyQuota(): WorkcostQuota {
  return {
    id: 0, code: "", title: "", specialty: "", chapter: "", unit: "",
    categoryPath: "", baseLabor: 0, baseMaterial: 0, baseMachine: 0,
    source: "手工录入", region: "", priceDate: "", note: "", status: "现行", items: [],
  };
}

/**
 * WorkcostComposeView — 综合单价分析表。
 *
 * 左：定额列表（按专业筛选 + 搜索）；右：选中定额的综合单价分析（汇总 + 工料机明细）。
 * 选中即调 WorkcostQuotaCompose 现算——不在库里存派生值，避免与资源价漂移。
 */
export function WorkcostComposeView() {
  const [specialty, setSpecialty] = useState("");
  const [keyword, setKeyword] = useState("");
  const [quotas, setQuotas] = useState<WorkcostQuota[]>([]);
  const [loading, setLoading] = useState(true);
  const [readError, setReadError] = useState("");
  const [selected, setSelected] = useState<string>("");
  const [compose, setCompose] = useState<WorkcostCompose | null>(null);
  const [composing, setComposing] = useState(false);
  const [composeError, setComposeError] = useState("");
  // 导入/导出：五表项目工作簿的往返。importedPath 记住最近导入的源文件，
  // 供「导出五表」按其口径重生成（导出的前提是有一份可解析的模版源）。
  const [busy, setBusy] = useState(false);
  const [toast, setToast] = useState("");
  const [importedPath, setImportedPath] = useState("");
  // 定额管理：编辑/新建弹窗与删除确认。编辑必须先 QuotaGet 取全量——
  // ListQuotas 是轻量查询不含含量行，直接拿列表对象编辑会把含量行清空。
  const [editing, setEditing] = useState<WorkcostQuota | null>(null);
  const [confirmDelete, setConfirmDelete] = useState<WorkcostQuota | null>(null);
  // composeTick 定额保存后自增——selected 编码不变时核算 effect 不会重跑，
  // 靠它强制按新含量行重算（改价场景同理）。
  const [composeTick, setComposeTick] = useState(0);

  // load 必须在导入/导出回调之前定义——那两个回调依赖它刷新列表。
  // （早期顺序写反过：`await load()` 引用了 const 声明前的 TDZ 变量，tsc 直接拦下。）
  const load = useCallback(async () => {
    setLoading(true);
    setReadError("");
    try {
      const r = (await app.WorkcostQuotaList(specialty, keyword)) ?? [];
      setQuotas(r);
      setSelected((prev) => (prev && r.some((q) => q.code === prev) ? prev : (r[0]?.code ?? "")));
    } catch (e) {
      setQuotas([]);
      setReadError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, [specialty, keyword]);

  // importWorkbook 选一份五表工作簿 → 解析 → 落库（资源 + 消耗定额）。
  // 两步都在后端做，前端只负责选文件与展示结果；失败原因原样显形。
  // 注：本回调定义在 load 之后（依赖它刷新列表）——定义顺序即依赖顺序。
  const importWorkbook = useCallback(async () => {
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
      setImportedPath(path);
      await load();
      const warn = bundle.warnings.length ? `；${bundle.warnings.length} 条告警` : "";
      setToast(
        `已导入「${bundle.project || bundle.fileName}」：` +
          `资源 新建 ${res.resourceNew}/更新 ${res.resourceUpd}、` +
          `定额 新建 ${res.quotaNew}/更新 ${res.quotaUpd}、消耗行 ${res.lines}${warn}`,
      );
      if (res.errors.length) {
        setToast((t) => `${t}；${res.errors.length} 条失败：${res.errors[0]}`);
      }
    } catch (e) {
      setToast(`导入失败：${e instanceof Error ? e.message : String(e)}`);
    } finally {
      setBusy(false);
    }
  }, [load]);

  // exportWorkbookAs 选目录导出五表（活公式）：用户挑位置，文件名 = 源名+时间戳。
  // 取消对话框（空串）静默返回——不是错误。
  const exportWorkbookAs = useCallback(async () => {
    if (!importedPath) return;
    setBusy(true);
    setToast("");
    try {
      const dir = await app.PickDirectory();
      if (!dir) {
        setBusy(false);
        return;
      }
      const srcBase = importedPath.replace(/\\/g, "/").split("/").pop() ?? "成本测算表";
      const base = srcBase.replace(/\.xlsx$/i, "");
      const n = new Date();
      const p2 = (v: number) => String(v).padStart(2, "0");
      const stamp = `${n.getFullYear()}${p2(n.getMonth() + 1)}${p2(n.getDate())}-${p2(n.getHours())}${p2(n.getMinutes())}${p2(n.getSeconds())}`;
      const out = `${dir.replace(/[\\/]+$/, "")}/${base}-${stamp}.xlsx`;
      const written = await app.WorkcostProjectExport(importedPath, out, "", "", "");
      setToast(`已导出五表工作簿：${written}`);
    } catch (e) {
      setToast(`导出失败：${e instanceof Error ? e.message : String(e)}`);
    } finally {
      setBusy(false);
    }
  }, [importedPath]);

  // exportToWorkspace 导出到工作区 .gaea/exports（固定约定位置的快捷路径）。
  const exportToWorkspace = useCallback(async () => {
    if (!importedPath) return;
    setBusy(true);
    setToast("");
    try {
      const out = await app.WorkcostProjectExportToWorkspace(importedPath, "成本测算表");
      setToast(`已导出到工作区：${out}`);
    } catch (e) {
      setToast(`导出失败：${e instanceof Error ? e.message : String(e)}`);
    } finally {
      setBusy(false);
    }
  }, [importedPath]);

  // openEdit 编辑定额：先取全量（含含量行），读取失败如实显形不装成空定额。
  const openEdit = useCallback(async () => {
    if (!selected) return;
    setBusy(true);
    try {
      const full = await app.WorkcostQuotaGet(selected);
      if (full) {
        setEditing(full);
      } else {
        setToast(`定额 ${selected} 读取失败（不存在）——请刷新列表`);
      }
    } catch (e) {
      setToast(`读取定额失败：${e instanceof Error ? e.message : String(e)}`);
    } finally {
      setBusy(false);
    }
  }, [selected]);

  // onQuotaSaved 保存后的统一收尾：刷新列表 + 强制重算当前选中。
  const onQuotaSaved = useCallback(
    async (saved: WorkcostQuota) => {
      setEditing(null);
      setToast(`已保存定额「${saved.title}」（${saved.code}）`);
      await load();
      setSelected(saved.code);
      setComposeTick((t) => t + 1);
    },
    [load],
  );

  // deleteQuota 删除定额（二次确认后）。
  const deleteQuota = useCallback(async () => {
    if (!confirmDelete) return;
    setBusy(true);
    try {
      await app.WorkcostQuotaDelete(confirmDelete.code);
      setToast(`已删除定额 ${confirmDelete.code}`);
      setConfirmDelete(null);
      await load();
      setComposeTick((t) => t + 1);
    } catch (e) {
      setToast(`删除失败：${e instanceof Error ? e.message : String(e)}`);
    } finally {
      setBusy(false);
    }
  }, [confirmDelete, load]);

  useEffect(() => {
    load();
  }, [load]);

  // 选中定额 → 核算综合单价（只含人材机）。
  useEffect(() => {
    if (!selected) {
      setCompose(null);
      return;
    }
    let alive = true;
    setComposing(true);
    setComposeError("");
    app
      .WorkcostQuotaCompose(selected, null)
      .then((c) => {
        if (alive) setCompose(c);
      })
      .catch((e) => {
        if (alive) {
          setCompose(null);
          setComposeError(e instanceof Error ? e.message : String(e));
        }
      })
      .finally(() => {
        if (alive) setComposing(false);
      });
    return () => {
      alive = false;
    };
  }, [selected, composeTick]);

  const current = useMemo(() => quotas.find((q) => q.code === selected) ?? null, [quotas, selected]);
  const specialties = useMemo(() => {
    const s = new Set<string>();
    for (const q of quotas) if (q.specialty) s.add(q.specialty);
    return [...s];
  }, [quotas]);

  return (
    <div className="h-full flex flex-col min-h-0 text-[12.5px]">
      {/* 工具条 */}
      <div className="shrink-0 flex flex-wrap items-center gap-2 px-5 min-h-12 py-2 border-b border-border-soft/60">
        <span className="text-fg font-semibold text-[13px] flex items-center gap-1.5">
          <Calculator size={14} className="text-accent" /> 综合单价分析
        </span>
        <span className="text-[11px] text-fg-faint hidden lg:inline">
          综合单价 = Σ(消耗量 × 资源价)，只含人材机 · 管理费/利润/税在项目合计层单列
        </span>
        <div className="ml-auto flex items-center gap-1.5">
          <button
            type="button"
            className={ghostBtn}
            onClick={importWorkbook}
            disabled={busy}
            title="选择五表项目工作簿（成本测算表），解析后入库为资源与消耗定额"
          >
            <FolderOpen size={12} /> 导入项目表
          </button>
          <button
            type="button"
            className={ghostBtn}
            onClick={exportWorkbookAs}
            disabled={busy || !importedPath}
            title="选择目录，按当前工料法口径导出五表模版工作簿（活公式）"
          >
            <FileSpreadsheet size={12} /> 导出五表到…
          </button>
          <button
            type="button"
            className={ghostBtn}
            onClick={exportToWorkspace}
            disabled={busy || !importedPath}
            title="导出到工作区 .gaea/exports（文件名带时间戳，活公式）"
          >
            <FileSpreadsheet size={12} /> → 工作区
          </button>
          <button
            type="button"
            className={solidBtn}
            onClick={() => setEditing(emptyQuota())}
            disabled={busy}
            title="新建消耗定额（编码 + 工料机含量行）"
          >
            <Plus size={12} /> 新建定额
          </button>
          {specialties.length > 1 && (
            <select className={`${fieldCls} !w-28`} value={specialty} onChange={(e) => setSpecialty(e.target.value)}>
              <option value="">全部专业</option>
              {specialties.map((s) => (
                <option key={s} value={s}>{s}</option>
              ))}
            </select>
          )}
          <div className="relative">
            <Search size={11} className="absolute left-2 top-1/2 -translate-y-1/2 text-fg-faint" />
            <input
              className={`${fieldCls} !pl-6 !w-40`}
              value={keyword}
              placeholder="搜索定额名称/编码"
              onChange={(e) => setKeyword(e.target.value)}
            />
          </div>
          <button type="button" className={ghostBtn} onClick={load} title="刷新">
            <RefreshCw size={12} />
          </button>
        </div>
      </div>

      {toast && (
        <div className="shrink-0 flex items-center gap-2 px-5 py-1.5 border-b border-border-soft/60 bg-accent/5 text-[11px] text-fg-dim">
          <span className="flex-1 min-w-0 break-all">{toast}</span>
          <button
            type="button"
            className="inline-flex items-center justify-center w-5 h-5 rounded border border-border text-fg-faint hover:text-fg"
            onClick={() => setToast("")}
            title="关闭提示"
          >
            <X size={10} />
          </button>
        </div>
      )}

      {readError && (
        <div
          role="alert"
          data-testid="workcost-compose-read-error"
          className="shrink-0 flex items-center gap-2 px-5 py-1.5 border-b border-amber-400/30 bg-amber-400/10 text-amber-200 text-[11px]"
        >
          <span className="flex-1 min-w-0">消耗定额读取失败：{readError}——结果可能不完整。</span>
          <button type="button" className={ghostBtn} onClick={load}>重试</button>
        </div>
      )}

      <div className="flex-1 min-h-0 flex">
        {/* 左：定额列表 */}
        <div className="w-72 shrink-0 border-r border-border-soft/60 overflow-y-auto">
          {loading ? (
            <div className="p-3 space-y-2 animate-pulse">
              <div className="v3-panel rounded-lg h-9" />
              <div className="v3-panel rounded-lg h-9" />
              <div className="v3-panel rounded-lg h-9" />
            </div>
          ) : quotas.length === 0 ? (
            <div className="p-3 text-[11.5px] text-fg-faint leading-relaxed">
              {readError ? "读取失败，未取到定额。" : "还没有消耗定额。导入项目工作簿（工料机价格 + 综合单价两表）后，每个清单项会自动成为一条定额。"}
            </div>
          ) : (
            <ul>
              {quotas.map((q) => (
                <li key={q.code}>
                  <button
                    type="button"
                    className={`w-full text-left px-3 py-2 border-b border-border-soft/25 transition-colors ${
                      q.code === selected ? "bg-accent/10" : "hover:bg-bg-elev/40"
                    }`}
                    onClick={() => setSelected(q.code)}
                  >
                    <span className="flex items-center gap-1.5">
                      <span className="font-mono text-[10px] text-fg-faint shrink-0">{q.code}</span>
                      {q.specialty && (
                        <span className="text-[9.5px] px-1 py-px rounded bg-bg-soft text-fg-faint shrink-0">{q.specialty}</span>
                      )}
                    </span>
                    <span className="block truncate text-[11.5px] text-fg mt-0.5">{q.title}</span>
                    <span className="block truncate text-[10px] text-fg-faint">
                      {q.unit || "-"} · {q.items?.length ?? 0} 条工料机
                    </span>
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>

        {/* 右：分析表 */}
        <div className="flex-1 min-w-0 overflow-y-auto px-5 py-4">
          {!selected ? (
            <Empty
              title="选择左侧一条定额"
              hint="右侧会展示该清单项的综合单价分析：工料机明细 → 人工/材料/机械三费 → 综合单价。"
            />
          ) : composeError ? (
            <Empty title="核算失败" hint={composeError} tone="warn" />
          ) : composing || !compose ? (
            <div className="space-y-2 animate-pulse">
              <div className="v3-panel rounded-xl h-20" />
              <div className="v3-panel rounded-xl h-64" />
            </div>
          ) : (
            <AnalysisTable quota={current} compose={compose} onEdit={openEdit} onDelete={() => current && setConfirmDelete(current)} />
          )}
        </div>
      </div>

      {editing && <QuotaModal value={editing} onCancel={() => setEditing(null)} onSaved={onQuotaSaved} />}
      {confirmDelete && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" onClick={() => setConfirmDelete(null)}>
          <div className="v3-panel rounded-2xl p-5 space-y-3 w-[26rem]" onClick={(e) => e.stopPropagation()}>
            <div className="text-[13px] text-fg font-semibold">删除消耗定额</div>
            <p className="text-[11.5px] text-fg-faint leading-relaxed">
              将删除定额 <span className="font-mono text-fg-dim">{confirmDelete.code}</span>「{confirmDelete.title}」
              及其全部工料机含量行。该操作不可撤销；引用它的费用汇总试算行会失去现算依据。
            </p>
            <div className="flex items-center justify-end gap-2">
              <button type="button" className={ghostBtn} onClick={() => setConfirmDelete(null)} disabled={busy}>
                取消
              </button>
              <button
                type="button"
                className="inline-flex items-center gap-1 px-2.5 h-7 rounded-lg bg-err text-white text-[11.5px] hover:opacity-90 transition-opacity disabled:opacity-50"
                onClick={() => void deleteQuota()}
                disabled={busy}
              >
                {busy ? "删除中…" : "确认删除"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

// AnalysisTable 单条定额的综合单价分析表（汇总行 + 工料机明细行）。
function AnalysisTable({
  quota,
  compose,
  onEdit,
  onDelete,
}: {
  quota: WorkcostQuota | null;
  compose: WorkcostCompose;
  onEdit: () => void;
  onDelete: () => void;
}) {
  return (
    <div className="space-y-3">
      {/* 表头信息 */}
      <div className="flex flex-wrap items-baseline gap-x-4 gap-y-1">
        <span className="text-[14px] text-fg font-semibold">
          {quota?.title ?? ""}
        </span>
        {quota?.code && <span className="font-mono text-[11px] text-fg-faint">{quota.code}</span>}
        {quota?.unit && <span className="text-[11px] text-fg-faint">计量单位：{quota.unit}</span>}
        {quota?.chapter && <span className="text-[11px] text-fg-faint">章节：{quota.chapter}</span>}
        {quota && (
          <span className="ml-auto flex items-center gap-1">
            <button type="button" className={iconBtn} title="编辑定额与含量行" onClick={onEdit}>
              <Pencil size={11} />
            </button>
            <button type="button" className={`${iconBtn} hover:text-err`} title="删除定额" onClick={onDelete}>
              <Trash2 size={11} />
            </button>
          </span>
        )}
      </div>

      {compose.warnings.length > 0 && (
        <div role="alert" className="v3-panel rounded-lg px-3 py-2 border border-amber-400/30 bg-amber-400/10 text-amber-200 text-[11px] space-y-0.5">
          {compose.warnings.map((w, i) => (
            <div key={i}>· {w}</div>
          ))}
        </div>
      )}

      {/* 综合单价分析表 */}
      <div className="v3-panel rounded-xl overflow-x-auto">
        <table className="w-full text-[11.5px]">
          <thead>
            <tr className="text-left text-[10px] text-fg-faint border-b border-border-soft/50 bg-bg-soft/30">
              <th className="py-2 px-3 w-16">行类</th>
              <th className="py-2 px-2">资源名称</th>
              <th className="py-2 px-2 w-16">消耗单位</th>
              <th className="py-2 px-2 w-24 text-right">消耗量</th>
              <th className="py-2 px-2 w-24 text-right">单价</th>
              <th className="py-2 px-2 w-24 text-right">金额</th>
              <th className="py-2 px-2 w-16 text-right">占比</th>
            </tr>
          </thead>
          <tbody>
            {compose.lines.length === 0 ? (
              <tr>
                <td colSpan={7} className="py-8 text-center text-fg-faint text-[11.5px]">
                  该定额没有工料机消耗行——综合单价为 0，需先录入消耗量。
                </td>
              </tr>
            ) : (
              compose.lines.map((l, i) => (
                <tr key={i} className="border-b border-border-soft/25 last:border-0 hover:bg-bg-elev/30">
                  <td className="py-1.5 px-3">
                    <span className={`px-1.5 py-px rounded text-[10px] ${kindTone(l.kind)}`}>{l.kind}</span>
                  </td>
                  <td className="py-1.5 px-2 text-fg-dim">{l.title || "-"}</td>
                  <td className="py-1.5 px-2 text-fg-faint">{l.unit || "-"}</td>
                  <td className="py-1.5 px-2 text-right tabular-nums">{l.quantity > 0 ? fmtQty(l.quantity) : "—"}</td>
                  <td className="py-1.5 px-2 text-right tabular-nums">
                    {l.price > 0 ? fmtPrice(l.price) : <span className="text-amber-300" title="资源无价，该项不计入合计">缺价</span>}
                  </td>
                  <td className="py-1.5 px-2 text-right tabular-nums">{fmtPrice(l.amount)}</td>
                  <td className="py-1.5 px-2 text-right tabular-nums text-fg-faint">{l.sharePct > 0 ? `${l.sharePct.toFixed(1)}%` : "-"}</td>
                </tr>
              ))
            )}
          </tbody>
          {/* 汇总：人材机三费 + 综合单价 */}
          <tfoot>
            <tr className="border-t border-border-soft/50 bg-bg-soft/30 text-[11.5px]">
              <td className="py-2 px-3 text-fg-faint">三费</td>
              <td className="py-2 px-2 text-fg-dim" colSpan={4}>
                人工 <span className="tabular-nums text-sky-400">{fmtPrice(compose.laborFee)}</span>
                <span className="mx-1.5 text-fg-faint">·</span>
                材料 <span className="tabular-nums text-ok">{fmtPrice(compose.materialFee)}</span>
                <span className="mx-1.5 text-fg-faint">·</span>
                机械 <span className="tabular-nums text-violet-400">{fmtPrice(compose.machineFee)}</span>
                {compose.outsourcedFee > 0 && (
                  <>
                    <span className="mx-1.5 text-fg-faint">·</span>
                    其中外委 <span className="tabular-nums text-amber-400">{fmtPrice(compose.outsourcedFee)}</span>
                    <span className="text-[10px] text-fg-faint ml-1">（已并入材料）</span>
                  </>
                )}
              </td>
              <td className="py-2 px-2 text-right tabular-nums text-fg-dim">{fmtPrice(compose.subtotal)}</td>
              <td className="py-2 px-2" />
            </tr>
            <tr className="bg-accent/5 text-[12.5px]">
              <td className="py-2.5 px-3 font-semibold text-fg" colSpan={5}>
                综合单价（人材机，不含管理费/利润/税金）
              </td>
              <td className="py-2.5 px-2 text-right tabular-nums font-semibold text-accent">
                {fmtPrice(compose.compositePrice)}
              </td>
              <td className="py-2.5 px-2" />
            </tr>
          </tfoot>
        </table>
      </div>

      {compose.zeroLines > 0 && (
        <p className="text-[11px] text-fg-faint">
          该定额有 {compose.zeroLines} 条零含量/零单价行（工序模板列出但本项未用），不计入合计——属合法占位，不是错误。
        </p>
      )}
      {compose.otherFee > 0 && (
        <p className="text-[11px] text-amber-300">
          有 {fmtPrice(compose.otherFee)} 的资源行类别未识别，已计入小计但未进入人工/材料/机械拆分，请检查资源类别。
        </p>
      )}
    </div>
  );
}

function Empty({ title, hint, tone }: { title: string; hint: string; tone?: "warn" }) {
  return (
    <div className="v3-panel rounded-2xl py-16 text-center">
      <ListTree size={26} className={`mx-auto ${tone === "warn" ? "text-amber-400" : "text-fg-faint"}`} />
      <div className="mt-3 text-[13px] text-fg font-medium">{title}</div>
      <p className="mt-1.5 text-[11.5px] text-fg-faint leading-relaxed max-w-md mx-auto">{hint}</p>
    </div>
  );
}

