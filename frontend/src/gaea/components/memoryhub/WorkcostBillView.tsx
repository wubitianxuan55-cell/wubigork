// WorkcostBillView.tsx — 分部分项工程量清单（工料法第④层，板块首屏）。
//
// 用户定调（2026-10-07）：「造价数据库只是一个数据库，你可以录入**费率、
// 清单、测算模版**，但不是把它们加起来」——本视图是清单与费率的**录入和
// 管理面**：项目（五表封面）→ 分部分项清单（编码/名称/单位/工程量/引用定额）
// + 项目费率。合价与费用汇总**不在库里算**：导出五表活公式工作簿由 Excel 算，
// 本视图的单价列只做「定额现算参考」。
//
// 三层关系：工料机=资源价格 → 定额=每单位消耗 → 清单=工程实体分项（套定额×工程量）。
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Coins, FileSpreadsheet, FolderOpen, ListTree, Plus, RefreshCw, Search, Trash2, X } from "../../icons";
import { app } from "../../lib/bridge";
import type { WorkcostBillItem, WorkcostBillProject, WorkcostCompose, WorkcostQuota } from "../../lib/types";

const fmt = new Intl.NumberFormat("zh-CN", { maximumFractionDigits: 2 });
const fmtPrice = (p: number) => "¥" + fmt.format(p);

const fieldCls =
  "w-full bg-bg border border-border-soft rounded-md text-fg text-[12px] px-2.5 py-1.5 outline-none focus:border-accent transition-colors placeholder:text-fg-faint/50";
const ghostBtn =
  "inline-flex items-center gap-1 px-2.5 h-7 rounded-lg border border-border text-fg-faint hover:text-fg hover:bg-bg-soft transition-colors text-[11.5px]";
const solidBtn =
  "inline-flex items-center gap-1 px-2.5 h-7 rounded-lg bg-accent text-accent-fg text-[11.5px] hover:opacity-90 transition-opacity disabled:opacity-50";
const iconBtn =
  "inline-flex items-center justify-center w-6 h-6 rounded-md border border-border text-fg-faint hover:text-fg hover:bg-bg-soft transition-colors";

/** 带项目归属的清单行（累计视图的行模型）。 */
interface BillRowItem extends WorkcostBillItem {
  projectName: string;
}

/**
 * WorkcostBillView — 分部分项清单（**所有项目累计** + 筛分）。
 *
 * 用户定调（2026-10-07）：清单是「所有累计的清单项的显示，且可以筛分」——
 * 主表 = 全部项目的清单项汇总，按 项目/分部/关键字 筛选；选中具体项目时
 * 显示其封面与费率录入，并可整体删除该项目（封面+清单+独占定额）。
 * 全部是数据操作——不出现任何合计/费用链。
 */
export function WorkcostBillView() {
  const [projects, setProjects] = useState<WorkcostBillProject[]>([]);
  const [loading, setLoading] = useState(true);
  const [readError, setReadError] = useState("");
  // 项目筛选：0 = 全部项目（累计视图）；>0 = 只看该项目（并显示封面费率）。
  const [selected, setSelected] = useState<number>(0);
  const [division, setDivision] = useState("");
  const [keyword, setKeyword] = useState("");
  const [allItems, setAllItems] = useState<BillRowItem[]>([]);
  const [busy, setBusy] = useState(false);
  const [toast, setToast] = useState("");
  const [pickerOpen, setPickerOpen] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState<WorkcostBillProject | null>(null);
  // 清单分析：点定额 chip 打开该清单项的综合单价分析（工料机明细+三费）。
  const [analyzeCode, setAnalyzeCode] = useState("");

  const loadProjects = useCallback(async () => {
    setLoading(true);
    setReadError("");
    try {
      const r = (await app.WorkcostBillProjects()) ?? [];
      setProjects(r);
    } catch (e) {
      setProjects([]);
      setReadError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, []);

  // 累计清单：全部项目的清单项一次拉平（各自带项目名）。
  const loadItems = useCallback(async (projs: WorkcostBillProject[]) => {
    try {
      const batches = await Promise.all(
        projs.map(async (p) =>
          ((await app.WorkcostBillItems(p.id)) ?? []).map((it) => ({ ...it, projectName: p.name })),
        ),
      );
      setAllItems(batches.flat());
    } catch (e) {
      setAllItems([]);
      setToast(`清单读取失败：${e instanceof Error ? e.message : String(e)}`);
    }
  }, []);

  const reload = useCallback(async () => {
    await loadProjects();
  }, [loadProjects]);

  useEffect(() => {
    loadProjects();
  }, [loadProjects]);

  useEffect(() => {
    loadItems(projects);
  }, [projects, loadItems]);

  const current = useMemo(() => projects.find((p) => p.id === selected) ?? null, [projects, selected]);

  // 筛分：项目 → 分部 → 关键字（名称/编码/特征）。
  const divisions = useMemo(() => {
    const s = new Set<string>();
    for (const it of allItems) if (it.division) s.add(it.division);
    return [...s].sort();
  }, [allItems]);

  const visible = useMemo(() => {
    const kw = keyword.trim().toLowerCase();
    return allItems.filter((it) => {
      if (selected && it.projectId !== selected) return false;
      if (division && it.division !== division) return false;
      if (kw && ![it.title, it.code, it.feature, it.quotaCode].some((v) => (v ?? "").toLowerCase().includes(kw))) return false;
      return true;
    });
  }, [allItems, selected, division, keyword]);

  // 导入项目表：解析→落库（资源+定额+**清单+费率**）→ 刷新项目列表。
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
      await loadProjects();
      await loadItems((await app.WorkcostBillProjects()) ?? []);
      const warn = bundle.warnings.length ? `；${bundle.warnings.length} 条告警` : "";
      setToast(
        `已录入「${bundle.project || bundle.fileName}」：清单 ${bundle.items.length} 条、` +
          `资源 新建 ${res.resourceNew}/更新 ${res.resourceUpd}、定额 新建 ${res.quotaNew}/更新 ${res.quotaUpd}${warn}`,
      );
    } catch (e) {
      setToast(`导入失败：${e instanceof Error ? e.message : String(e)}`);
    } finally {
      setBusy(false);
    }
  }, [loadProjects, loadItems, selected]);

  // addManualRow 手填清单行（外委包干类：无定额，可手填单价）。仅在选中具体
  // 项目时可用（累计视图没有「落点项目」）。
  const addManualRow = useCallback(async () => {
    if (!selected) return;
    setBusy(true);
    try {
      await app.WorkcostBillItemSave({
        id: 0, projectId: selected, code: "", title: "新清单项", unit: "", division: "",
        quantity: 0, quantityExpr: "", quotaCode: "", feature: "", priceOverride: 0, sort: 0,
      });
      await reload();
      await loadItems(projects);
    } catch (e) {
      setToast(`新增失败：${e instanceof Error ? e.message : String(e)}`);
    } finally {
      setBusy(false);
    }
  }, [selected, reload, loadItems, projects]);

  const patchItem = useCallback(
    async (item: WorkcostBillItem, patch: Partial<WorkcostBillItem>) => {
      setAllItems((ls) => ls.map((x) => (x.id === item.id ? { ...x, ...patch } : x)));
      try {
        await app.WorkcostBillItemSave({ ...item, ...patch });
      } catch (e) {
        setToast(`保存失败：${e instanceof Error ? e.message : String(e)}`);
        await loadItems(projects);
      }
    },
    [loadItems, projects],
  );

  const removeItem = useCallback(
    async (item: WorkcostBillItem) => {
      try {
        await app.WorkcostBillItemDelete(item.id);
        await loadItems(projects);
        await reload();
      } catch (e) {
        setToast(`删除失败：${e instanceof Error ? e.message : String(e)}`);
      }
    },
    [loadItems, reload, projects],
  );

  // deleteProject 整体删除导入的项目（封面+清单+独占定额；共享定额与资源保留）。
  const deleteProject = useCallback(async () => {
    if (!confirmDelete) return;
    setBusy(true);
    try {
      const removed = await app.WorkcostBillProjectDelete(confirmDelete.id);
      setConfirmDelete(null);
      setSelected(0);
      await reload();
      await loadItems(projects);
      setToast(`已删除项目「${confirmDelete.name}」（连带独占定额 ${removed} 条；共享定额与资源保留）`);
    } catch (e) {
      setToast(`删除失败：${e instanceof Error ? e.message : String(e)}`);
    } finally {
      setBusy(false);
    }
  }, [confirmDelete, reload, loadItems, projects]);

  return (
    <div className="h-full flex flex-col min-h-0 text-[12.5px]">
      {/* 工具条 */}
      <div className="shrink-0 flex flex-wrap items-center gap-2 px-5 h-auto min-h-12 py-2 border-b border-border-soft/60">
        <span className="text-fg font-semibold text-[13px] flex items-center gap-1.5">
          <ListTree size={14} className="text-accent" /> 分部分项清单
        </span>
        <span className="text-[11px] text-fg-faint hidden lg:inline">
          数据库：录入清单 · 工程量 · 费率 · 模版——加总由导出的五表活公式算，不在库里算
        </span>
        <div className="ml-auto flex items-center gap-1.5">
          <button type="button" className={ghostBtn} onClick={importWorkbook} disabled={busy} title="导入五表工作簿：清单+费率+资源+定额一次录入">
            <FolderOpen size={12} /> 导入项目表
          </button>
          <button type="button" className={ghostBtn} onClick={addManualRow} disabled={busy || !selected} title={selected ? "手填一条清单项（外委包干类，可无定额）" : "先在下方选择具体项目"}>
            <Plus size={12} /> 手填行
          </button>
          <button type="button" className={solidBtn} onClick={() => setPickerOpen(true)} disabled={busy || !selected} title={selected ? "从定额库选一条，作为清单项录入" : "先在下方选择具体项目"}>
            <FileSpreadsheet size={12} /> 从定额库添加
          </button>
          <button type="button" className={ghostBtn} onClick={reload} title="刷新">
            <RefreshCw size={12} />
          </button>
        </div>
      </div>

      {toast && (
        <div className="shrink-0 flex items-center gap-2 px-5 py-1.5 border-b border-border-soft/60 bg-accent/5 text-[11px] text-fg-dim">
          <span className="flex-1 min-w-0 break-all">{toast}</span>
          <button type="button" className={iconBtn} onClick={() => setToast("")} title="关闭提示">
            <X size={10} />
          </button>
        </div>
      )}

      {readError && (
        <div role="alert" data-testid="workcost-bill-read-error" className="shrink-0 flex items-center gap-2 px-5 py-1.5 border-b border-amber-400/30 bg-amber-400/10 text-amber-200 text-[11px]">
          <span className="flex-1 min-w-0">项目列表读取失败：{readError}</span>
          <button type="button" className={ghostBtn} onClick={loadProjects}>重试</button>
        </div>
      )}

      <div className="flex-1 min-h-0 flex flex-col">
        {/* 筛选工具条：项目 / 分部 / 关键字（累计视图的筛分） */}
        <div className="shrink-0 flex flex-wrap items-center gap-2 px-5 py-2 border-b border-border-soft/60">
          <select
            className={`${fieldCls} !w-auto min-w-[14rem]`}
            value={selected}
            aria-label="筛选项目"
            data-testid="workcost-bill-project-filter"
            onChange={(e) => { setSelected(Number(e.target.value)); setDivision(""); }}
          >
            <option value={0}>全部项目（{projects.length} 个累计）</option>
            {projects.map((p) => (
              <option key={p.id} value={p.id}>{p.name}（{p.itemCount} 条）</option>
            ))}
          </select>
          <select className={`${fieldCls} !w-auto`} value={division} aria-label="筛选分部" onChange={(e) => setDivision(e.target.value)}>
            <option value="">全部分部</option>
            {divisions.map((d) => (
              <option key={d} value={d}>{d}</option>
            ))}
          </select>
          <div className="relative">
            <Search size={11} className="absolute left-2 top-1/2 -translate-y-1/2 text-fg-faint" />
            <input
              className={`${fieldCls} !pl-6 !w-56`}
              value={keyword}
              placeholder="搜索名称/编码/特征/定额"
              aria-label="搜索清单"
              onChange={(e) => setKeyword(e.target.value)}
            />
          </div>
          {current && (
            <>
              <span className="text-[10.5px] text-fg-faint hidden md:inline">
                {current.location ? `${current.location} · ` : ""}{current.fileName}
              </span>
              <button
                type="button"
                className="ml-auto inline-flex items-center gap-1 px-2.5 h-7 rounded-lg border border-err/40 text-err hover:bg-err/10 transition-colors text-[11.5px]"
                onClick={() => setConfirmDelete(current)}
                title="整体删除该项目：封面+清单+独占定额（共享定额与资源保留）"
              >
                <Trash2 size={12} /> 删除该项目
              </button>
            </>
          )}
          <span className={`text-[10.5px] text-fg-faint ${current ? "" : "ml-auto"}`} data-testid="workcost-bill-count">
            {visible.length}/{allItems.length} 条
          </span>
        </div>

        {/* 项目封面 + 费率（选中具体项目时显示） */}
        {current && (
          <div className="shrink-0 px-5 pt-3">
            <section className="v3-panel rounded-2xl p-4">
              <div className="flex flex-wrap items-baseline gap-x-4 gap-y-1">
                <span className="text-[14px] text-fg font-semibold">{current.name}</span>
                {current.location && <span className="text-[11px] text-fg-faint">地点：{current.location}</span>}
                {current.duration && <span className="text-[11px] text-fg-faint">工期：{current.duration}</span>}
              </div>
              {current.pricing && (
                <p className="mt-1.5 text-[10.5px] text-fg-faint leading-relaxed" title={current.pricing}>
                  口径：{current.pricing}
                </p>
              )}
              <RatesEditor project={current} onSaved={() => { reload(); setToast("费率已保存"); }} />
            </section>
          </div>
        )}

        {/* 累计清单表 */}
        <div className="flex-1 min-h-0 overflow-y-auto px-5 py-3">
          {loading ? (
            <div className="space-y-2 animate-pulse">
              <div className="v3-panel rounded-xl h-10" />
              <div className="v3-panel rounded-xl h-64" />
            </div>
          ) : readError ? (
            <div role="alert" className="v3-panel rounded-xl px-4 py-3 text-amber-200 text-[11.5px]">
              项目读取失败：{readError}
              <button type="button" className={ghostBtn + " ml-3"} onClick={reload}>重试</button>
            </div>
          ) : projects.length === 0 ? (
            <Empty
              title="还没有清单项目"
              hint="点「导入项目表」选一份五表成本测算工作簿——封面、分部分项清单、取费费率、工料机价格、消耗定额一次录入数据库。"
            />
          ) : visible.length === 0 ? (
            <Empty title="没有匹配的清单项" hint="调整上方筛选（项目/分部/关键字）后重试。" />
          ) : (
            <section className="v3-panel rounded-2xl p-4" data-testid="workcost-bill-items">
              <table className="w-full text-[11.5px]">
                <thead>
                  <tr className="text-left text-[10px] text-fg-faint border-b border-border-soft/50">
                    <th className="py-2 px-2 w-24">项目</th>
                    <th className="py-2 px-2 w-16">编码</th>
                    <th className="py-2 px-2">名称</th>
                    <th className="py-2 px-2">项目特征</th>
                    <th className="py-2 px-2 w-14">单位</th>
                    <th className="py-2 px-2 w-24 text-right">工程量</th>
                    <th className="py-2 px-2 w-24">引用定额</th>
                    <th className="py-2 px-2 w-24 text-right">单价参考</th>
                    <th className="py-2 px-2 w-10" />
                  </tr>
                </thead>
                <tbody>
                  {visible.map((it) => (
                    <BillRow key={it.id} item={it} showProject={!selected} onPatch={patchItem} onRemove={removeItem} onAnalyze={setAnalyzeCode} />
                  ))}
                </tbody>
              </table>
            </section>
          )}
        </div>
      </div>

      {analyzeCode && <QuotaAnalysisModal quotaCode={analyzeCode} onClose={() => setAnalyzeCode("")} />}
      {confirmDelete && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" onClick={() => setConfirmDelete(null)}>
          <div className="v3-panel rounded-2xl p-5 space-y-3 w-[28rem]" onClick={(e) => e.stopPropagation()}>
            <div className="text-[13px] text-fg font-semibold">删除清单项目</div>
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
      {pickerOpen && current && (
        <QuotaPickerModal
          onCancel={() => setPickerOpen(false)}
          onPick={(q) => {
            setPickerOpen(false);
            void patchItem(
              { id: 0, projectId: current.id, code: q.code, title: q.title, unit: q.unit, division: "", quantity: 1, quantityExpr: "", quotaCode: q.code, feature: "", priceOverride: 0, sort: 0 },
              {},
            ).then(() => {
              loadItems(projects);
              reload();
            });
          }}
        />
      )}
    </div>
  );
}

// BillRow 一条清单行：工程量/单位/定额引用可编辑（失焦保存）；单价=定额现算参考。
function BillRow({
  item,
  showProject,
  onPatch,
  onRemove,
  onAnalyze,
}: {
  item: WorkcostBillItem & { projectName?: string };
  showProject?: boolean;
  onPatch: (item: WorkcostBillItem, patch: Partial<WorkcostBillItem>) => void;
  onRemove: (item: WorkcostBillItem) => void;
  onAnalyze: (quotaCode: string) => void;
}) {
  // 单价参考：挂定额的行按当前资源价现算（异步、带缓存不需要——行组件内自查）。
  const [refPrice, setRefPrice] = useState<number | null>(null);
  const seq = useRef(0);
  useEffect(() => {
    if (!item.quotaCode) {
      setRefPrice(null);
      return;
    }
    const my = ++seq.current;
    app
      .WorkcostQuotaCompose(item.quotaCode, null)
      .then((c) => {
        if (my === seq.current) setRefPrice(c.compositePrice);
      })
      .catch(() => {
        if (my === seq.current) setRefPrice(null);
      });
  }, [item.quotaCode]);

  return (
    <tr className="border-b border-border-soft/25 last:border-0 hover:bg-bg-elev/30">
      {showProject && (
        <td className="py-1.5 px-2 max-w-[9rem]" title={item.projectName}>
          <span className="block truncate text-[10px] text-fg-faint">{item.projectName}</span>
        </td>
      )}
      <td className="py-1.5 px-2 font-mono text-[10px] text-fg-faint">{item.code}</td>
      <td className="py-1.5 px-2">
        <input
          className="w-full bg-transparent border-0 outline-none text-fg focus:border-b focus:border-accent"
          defaultValue={item.title}
          aria-label="清单项名称"
          onBlur={(e) => {
            if (e.target.value !== item.title) onPatch(item, { title: e.target.value });
          }}
        />
      </td>
      <td className="py-1.5 px-2">
        {item.feature || item.quantityExpr ? (
          <span
            className="block truncate max-w-[14rem] text-[10.5px] text-fg-faint"
            title={item.quantityExpr ? `${item.feature}｜计算式：${item.quantityExpr}` : item.feature}
          >
            {item.feature || `计算式：${item.quantityExpr}`}
          </span>
        ) : (
          <span className="text-fg-faint/50 text-[10.5px]">—</span>
        )}
      </td>
      <td className="py-1.5 px-2">
        <input
          className="w-12 bg-transparent border-0 outline-none text-fg-faint focus:border-b focus:border-accent"
          defaultValue={item.unit}
          aria-label="单位"
          onBlur={(e) => {
            if (e.target.value !== item.unit) onPatch(item, { unit: e.target.value });
          }}
        />
      </td>
      <td className="py-1.5 px-2 text-right">
        <input
          className="w-24 ml-auto bg-transparent border-0 outline-none text-right tabular-nums focus:border-b focus:border-accent"
          type="number"
          min={0}
          defaultValue={item.quantity}
          aria-label="工程量"
          title="工程量是录入数据；合价由导出的五表工作簿用活公式算"
          onBlur={(e) => {
            const v = Number(e.target.value);
            if (v !== item.quantity) onPatch(item, { quantity: v });
          }}
        />
      </td>
      <td className="py-1.5 px-2">
        {item.quotaCode ? (
          <button
            type="button"
            className="font-mono text-[10px] px-1.5 py-px rounded bg-accent/10 text-accent hover:bg-accent/20 transition-colors"
            title={`引用定额 ${item.quotaCode}——点开综合单价分析（工料机明细+三费）`}
            onClick={() => onAnalyze(item.quotaCode)}
          >
            {item.quotaCode}
          </button>
        ) : (
          <span className="text-[10px] text-fg-faint" title="未套定额：可在下方手填单价（录入数据）">手填</span>
        )}
      </td>
      <td className="py-1.5 px-2 text-right">
        {item.quotaCode ? (
          refPrice === null ? (
            <span className="text-fg-faint">…</span>
          ) : (
            <span className="tabular-nums text-fg-dim" title="按当前资源价现算的综合单价（人材机）——参考值，导出后由 Excel 活公式重算">
              {fmtPrice(refPrice)}
            </span>
          )
        ) : (
          <input
            className="w-24 ml-auto bg-transparent border-0 outline-none text-right tabular-nums focus:border-b focus:border-accent"
            type="number"
            min={0}
            step="0.01"
            defaultValue={item.priceOverride || ""}
            aria-label="手填单价"
            title="未套定额的手填单价（如外委包干）——录入数据"
            onBlur={(e) => {
              const v = Number(e.target.value);
              if (v !== item.priceOverride) onPatch(item, { priceOverride: v });
            }}
          />
        )}
      </td>
      <td className="py-1.5 px-2">
        <button type="button" className={`${iconBtn} hover:text-err`} title="删除本行" onClick={() => onRemove(item)}>
          <Trash2 size={11} />
        </button>
      </td>
    </tr>
  );
}

// RatesEditor 费率录入区（企管/规费/利润/税率 + 利润基数含规费 + 控制价）。
// 全部是数据——不做任何计算展示。
function RatesEditor({ project, onSaved }: { project: WorkcostBillProject; onSaved: () => void }) {
  const [mgmt, setMgmt] = useState(round2(project.managementRate * 100));
  const [reg, setReg] = useState(round2(project.regulatoryRate * 100));
  const [profit, setProfit] = useState(round2(project.profitRate * 100));
  const [tax, setTax] = useState(round2(project.taxRate * 100));
  const [inclReg, setInclReg] = useState(project.profitIncludesRegulatory);
  const [control, setControl] = useState(project.controlPrice);
  const [busy, setBusy] = useState(false);

  // 项目切换时同步表单值。
  useEffect(() => {
    setMgmt(round2(project.managementRate * 100));
    setReg(round2(project.regulatoryRate * 100));
    setProfit(round2(project.profitRate * 100));
    setTax(round2(project.taxRate * 100));
    setInclReg(project.profitIncludesRegulatory);
    setControl(project.controlPrice);
  }, [project]);

  const save = async () => {
    setBusy(true);
    try {
      await app.WorkcostBillProjectRatesSave(
        project.id,
        { managementRate: mgmt / 100, regulatoryRate: reg / 100, profitRate: profit / 100, taxRate: tax / 100 },
        inclReg,
        control,
      );
      onSaved();
    } catch {
      // 保存失败静默重试由用户触发；错误极少（本地库）。
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="mt-3 pt-3 border-t border-border-soft/40">
      <div className="flex items-center gap-2 mb-2">
        <Coins size={12} className="text-fg-faint" />
        <span className="text-[11.5px] text-fg font-medium">取费费率（录入数据）</span>
        <span className="text-[10px] text-fg-faint">费率 0 = 显式不计取该段；加总在导出的五表里由 Excel 算</span>
      </div>
      <div className="grid grid-cols-2 md:grid-cols-6 gap-2 items-end">
        <RateInput label="企管 %" value={mgmt} onChange={setMgmt} />
        <RateInput label="规费 %" value={reg} onChange={setReg} />
        <RateInput label="利润 %" value={profit} onChange={setProfit} />
        <RateInput label="增值税 %" value={tax} onChange={setTax} />
        <label className="block">
          <span className="block text-[10.5px] text-fg-faint mb-1">控制价（元）</span>
          <input className={fieldCls} type="number" min={0} value={control || ""} aria-label="控制价" onChange={(e) => setControl(Number(e.target.value))} />
        </label>
        <div className="flex items-center gap-2">
          <label className="flex items-center gap-1.5 text-[11px] text-fg-dim cursor-pointer select-none" title="两份实测产物口径差异：百锦路不含规费 / 市政道路含规费">
            <input type="checkbox" checked={inclReg} onChange={(e) => setInclReg(e.target.checked)} />
            利润基数含规费
          </label>
          <button type="button" className={solidBtn} onClick={() => void save()} disabled={busy}>
            {busy ? "保存中…" : "保存费率"}
          </button>
        </div>
      </div>
    </div>
  );
}

function RateInput({ label, value, onChange }: { label: string; value: number; onChange: (v: number) => void }) {
  return (
    <label className="block">
      <span className="block text-[10.5px] text-fg-faint mb-1">{label}</span>
      <input className={fieldCls} type="number" min={0} step="0.1" aria-label={label} value={value} onChange={(e) => onChange(Number(e.target.value))} />
    </label>
  );
}

function round2(v: number): number {
  return Math.round(v * 100) / 100;
}

// QuotaPickerModal 从定额库选一条作为清单项录入。
function QuotaPickerModal({ onCancel, onPick }: { onCancel: () => void; onPick: (q: WorkcostQuota) => void }) {
  const [keyword, setKeyword] = useState("");
  const [rows, setRows] = useState<WorkcostQuota[]>([]);
  const [loading, setLoading] = useState(true);
  const reqSeq = useRef(0);

  useEffect(() => {
    const seq = ++reqSeq.current;
    setLoading(true);
    app
      .WorkcostQuotaList("", keyword)
      .then((r) => {
        if (seq === reqSeq.current) setRows(r ?? []);
      })
      .catch(() => {
        if (seq === reqSeq.current) setRows([]);
      })
      .finally(() => {
        if (seq === reqSeq.current) setLoading(false);
      });
  }, [keyword]);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" onClick={onCancel}>
      <div className="v3-panel rounded-2xl p-5 space-y-3 w-[40rem] max-h-[80vh] overflow-y-auto" onClick={(e) => e.stopPropagation()}>
        <div className="flex items-center gap-2">
          <span className="text-[13px] text-fg font-semibold flex-1">从定额库选一条录入清单</span>
          <button type="button" className={iconBtn} onClick={onCancel} title="关闭">
            <X size={11} />
          </button>
        </div>
        <div className="relative">
          <Search size={11} className="absolute left-2 top-1/2 -translate-y-1/2 text-fg-faint" />
          <input className={`${fieldCls} !pl-6`} value={keyword} placeholder="搜索定额名称/编码/章节" autoFocus onChange={(e) => setKeyword(e.target.value)} />
        </div>
        {loading ? (
          <div className="space-y-2 animate-pulse">
            <div className="v3-panel rounded-lg h-9" />
            <div className="v3-panel rounded-lg h-9" />
            <div className="v3-panel rounded-lg h-9" />
          </div>
        ) : rows.length === 0 ? (
          <div className="py-8 text-center text-[11.5px] text-fg-faint">没有匹配的定额。先在「单价分析」导入或新建定额。</div>
        ) : (
          <ul className="v3-panel rounded-lg divide-y divide-border-soft/25" data-testid="workcost-bill-picker">
            {rows.slice(0, 50).map((q) => (
              <li key={q.code} className="flex items-center gap-2 px-3 py-2">
                <span className="font-mono text-[10px] text-fg-faint w-14 shrink-0">{q.code}</span>
                <span className="min-w-0 flex-1">
                  <span className="block truncate text-[11.5px] text-fg">{q.title}</span>
                  <span className="block truncate text-[10px] text-fg-faint">{q.unit || "-"} · {q.items?.length ?? 0} 条工料机</span>
                </span>
                <button type="button" className={ghostBtn} onClick={() => onPick(q)}>录入清单</button>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}

function Empty({ title, hint }: { title: string; hint: string }) {
  return (
    <div className="v3-panel rounded-2xl py-16 text-center">
      <ListTree size={26} className="mx-auto text-fg-faint" />
      <div className="mt-3 text-[13px] text-fg font-medium">{title}</div>
      <p className="mt-1.5 text-[11.5px] text-fg-faint leading-relaxed max-w-md mx-auto">{hint}</p>
    </div>
  );
}

// QuotaAnalysisModal 清单项的综合单价分析（「清单分析」入口）：
// 定额头 + 工料机明细行（含量×单价=金额）+ 人材机三费汇总 + 综合单价（只含
// 人材机）。数据全部现算（WorkcostQuotaCompose），资源调价即最新口径。
function QuotaAnalysisModal({ quotaCode, onClose }: { quotaCode: string; onClose: () => void }) {
  const [quota, setQuota] = useState<WorkcostQuota | null>(null);
  const [lines, setLines] = useState<WorkcostCompose | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    let alive = true;
    app
      .WorkcostQuotaGet(quotaCode)
      .then((q) => {
        if (alive) setQuota(q);
      })
      .catch(() => {
        /* 定额头缺失不阻塞分析表 */
      });
    app
      .WorkcostQuotaCompose(quotaCode, null)
      .then((c) => {
        if (alive) setLines(c);
      })
      .catch((e) => {
        if (alive) setError(e instanceof Error ? e.message : String(e));
      });
    return () => {
      alive = false;
    };
  }, [quotaCode]);

  const f = new Intl.NumberFormat("zh-CN", { maximumFractionDigits: 2 });
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" onClick={onClose}>
      <div
        className="v3-panel rounded-2xl p-5 space-y-3 w-[46rem] max-h-[84vh] overflow-y-auto"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center gap-2">
          <span className="text-[13px] text-fg font-semibold flex-1 min-w-0 truncate">
            综合单价分析：{quota?.title ?? quotaCode}
          </span>
          {quota?.origCode && quota.origCode !== quotaCode && (
            <span className="font-mono text-[10px] px-1.5 py-px rounded bg-bg-soft text-fg-faint" title="工作簿原始编码（定额编码统一前的溯源）">
              原码 {quota.origCode}
            </span>
          )}
          <button type="button" className={iconBtn} onClick={onClose} title="关闭">
            <X size={11} />
          </button>
        </div>
        <div className="flex flex-wrap gap-x-4 gap-y-1 text-[11px] text-fg-faint">
          <span className="font-mono text-accent">{quotaCode}</span>
          {quota?.unit && <span>单位：{quota.unit}</span>}
          {quota?.specialty && <span>专业：{quota.specialty}</span>}
          {quota?.note && <span className="max-w-[24rem] truncate" title={quota.note}>特征：{quota.note}</span>}
        </div>
        {error ? (
          <div role="alert" className="text-[11px] text-amber-300 border border-amber-400/30 bg-amber-400/10 rounded-lg px-2.5 py-2">
            核算失败：{error}
          </div>
        ) : !lines ? (
          <div className="space-y-2 animate-pulse">
            <div className="v3-panel rounded-lg h-16" />
            <div className="v3-panel rounded-lg h-40" />
          </div>
        ) : (
          <>
            <div className="v3-panel rounded-lg overflow-x-auto">
              <table className="w-full text-[11.5px]">
                <thead>
                  <tr className="text-left text-[10px] text-fg-faint border-b border-border-soft/50">
                    <th className="py-1.5 px-2 w-14">行类</th>
                    <th className="py-1.5 px-2">资源</th>
                    <th className="py-1.5 px-2 w-14">单位</th>
                    <th className="py-1.5 px-2 w-20 text-right">含量</th>
                    <th className="py-1.5 px-2 w-20 text-right">单价</th>
                    <th className="py-1.5 px-2 w-24 text-right">金额</th>
                  </tr>
                </thead>
                <tbody>
                  {lines.lines.length === 0 ? (
                    <tr>
                      <td colSpan={6} className="py-6 text-center text-fg-faint text-[11.5px]">
                        该定额没有工料机消耗行——综合单价为 0。
                      </td>
                    </tr>
                  ) : (
                    lines.lines.map((l, i) => (
                      <tr key={i} className="border-b border-border-soft/20 last:border-0">
                        <td className="py-1 px-2 text-fg-dim">{l.kind}</td>
                        <td className="py-1 px-2 text-fg-dim">{l.title || "-"}</td>
                        <td className="py-1 px-2 text-fg-faint">{l.unit || "-"}</td>
                        <td className="py-1 px-2 text-right tabular-nums">{l.quantity > 0 ? f.format(l.quantity) : "—"}</td>
                        <td className="py-1 px-2 text-right tabular-nums">
                          {l.price > 0 ? fmtPrice(l.price) : <span className="text-amber-300">缺价</span>}
                        </td>
                        <td className="py-1 px-2 text-right tabular-nums">{fmtPrice(l.amount)}</td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
            <div className="flex flex-wrap items-center justify-end gap-x-4 gap-y-1 text-[11.5px] text-fg-dim px-1">
              <span>人工 <span className="tabular-nums text-sky-400">{fmtPrice(lines.laborFee)}</span></span>
              <span>材料 <span className="tabular-nums text-ok">{fmtPrice(lines.materialFee)}</span></span>
              <span>机械 <span className="tabular-nums text-violet-400">{fmtPrice(lines.machineFee)}</span></span>
              <span className="text-fg font-semibold">
                综合单价（人材机）<span className="tabular-nums text-accent ml-1">{fmtPrice(lines.compositePrice)}</span>
              </span>
            </div>
            {lines.warnings.length > 0 && (
              <div role="alert" className="text-[10.5px] text-amber-300 border border-amber-400/30 bg-amber-400/10 rounded-lg px-2.5 py-1.5">
                {lines.warnings.map((w, i) => (
                  <div key={i}>· {w}</div>
                ))}
              </div>
            )}
            <p className="text-[10px] text-fg-faint">
              按当前资源价现算（参考口径）；清单工程量 × 此单价 = 合价，由导出的五表工作簿活公式计算。
            </p>
          </>
        )}
      </div>
    </div>
  );
}
