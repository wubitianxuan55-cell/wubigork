// WorkcostBillView.tsx — 分部分项清单（**所有项目累计的清单列表**）。
//
// 用户定调（2026-10-07）：「清单是所有项目累计的清单列表，它不需要工程量」
// ——清单=清单项主列表（编码/名称/特征/单位/引用定额/单价参考），工程量是
// 项目使用的属性不在本视图；项目文件的导入/费率/删除在「造价参考」的项目
// 案例区。综合单价按当前资源价现算（参考口径），加总由导出的五表活公式算。
//
// 三层关系：工料机=资源价格 → 定额=每单位消耗 → 清单=工程实体分项（套定额）。
import { useCallback, useEffect, useMemo, useState } from "react";
import { ListTree, RefreshCw, Search, X } from "../../icons";
import { app } from "../../lib/bridge";
import type { WorkcostBillItem, WorkcostBillProject, WorkcostBillQuotaLink, WorkcostCompose, WorkcostQuota } from "../../lib/types";

export const fmtPrice = (p: number) =>
  "¥" + new Intl.NumberFormat("zh-CN", { maximumFractionDigits: 2 }).format(p);

export const fieldCls =
  "w-full bg-bg border border-border-soft rounded-md text-fg text-[12px] px-2.5 py-1.5 outline-none focus:border-accent transition-colors placeholder:text-fg-faint/50";
export const ghostBtn =
  "inline-flex items-center gap-1 px-2.5 h-7 rounded-lg border border-border text-fg-faint hover:text-fg hover:bg-bg-soft transition-colors text-[11.5px]";
export const solidBtn =
  "inline-flex items-center gap-1 px-2.5 h-7 rounded-lg bg-accent text-accent-fg text-[11.5px] hover:opacity-90 transition-opacity disabled:opacity-50";
export const iconBtn =
  "inline-flex items-center justify-center w-6 h-6 rounded-md border border-border text-fg-faint hover:text-fg hover:bg-bg-soft transition-colors";

/** 带项目归属的清单行（累计视图的行模型）。 */
interface BillRowItem extends WorkcostBillItem {
  projectName: string;
}

/**
 * WorkcostBillView — **清单库**（所有项目累计的分部分项清单 + 筛分）。
 *
 * 双视图：
 *   明细 = 每个项目的清单项逐条陈列（项目/编码/名称/特征/单位/定额/单价参考）；
 *   库项归并 = 同名+同特征+同单位的清单项聚合成一条库项，显形「被 N 个项目
 *   使用」与引用的定额码——这是企业清单积累的主数据视角。
 * 点定额编码看综合单价分析。项目文件的导入/费率/删除 → 「造价参考」。
 */
export function WorkcostBillView() {
  const [projects, setProjects] = useState<WorkcostBillProject[]>([]);
  const [loading, setLoading] = useState(true);
  const [readError, setReadError] = useState("");
  // 项目筛选：0 = 全部项目（累计视图）。
  const [selected, setSelected] = useState<number>(0);
  const [division, setDivision] = useState("");
  const [keyword, setKeyword] = useState("");
  const [allItems, setAllItems] = useState<BillRowItem[]>([]);
  // 视图：detail=按项目明细；grouped=按库项归并（同名+同特征+同单位聚合）。
  const [mode, setMode] = useState<"detail" | "grouped">("detail");
  const [expandedGroups, setExpandedGroups] = useState<Set<string>>(new Set());
  const [toast, setToast] = useState("");
  // 清单分析：点清单行打开该清单项的综合单价分析（按挂接定额分组，1:N）。
  const [analyzeItem, setAnalyzeItem] = useState<WorkcostBillItem | null>(null);
  // 参考价批量缓存：一次 ComposeMany（key=定额编码），替代每行一次核算——
  // 逐行调用在 63 行清单上=63 次全资源扫描，真机首屏 longtask ~700ms。
  const [refPrices, setRefPrices] = useState<Record<string, number>>({});

  const loadProjects = useCallback(async () => {
    setLoading(true);
    setReadError("");
    try {
      setProjects((await app.WorkcostBillProjects()) ?? []);
    } catch (e) {
      setProjects([]);
      setReadError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, []);

  // 累计清单：全部项目的清单项一次拉平（各自带项目名），参考价一次批量核算。
  const loadItems = useCallback(async (projs: WorkcostBillProject[]) => {
    try {
      const batches = await Promise.all(
        projs.map(async (p) =>
          ((await app.WorkcostBillItems(p.id)) ?? []).map((it) => ({ ...it, projectName: p.name })),
        ),
      );
      const items = batches.flat();
      setAllItems(items);
      const codes = [...new Set(items.map((it) => it.quotaCode).filter(Boolean))];
      if (codes.length > 0) {
        try {
          const many = (await app.WorkcostQuotaComposeMany(codes)) ?? {};
          const prices: Record<string, number> = {};
          for (const [code, c] of Object.entries(many)) prices[code] = c.compositePrice;
          setRefPrices(prices);
        } catch {
          setRefPrices({}); // 参考价失败不拦清单列表——单价列回退占位
        }
      } else {
        setRefPrices({});
      }
    } catch (e) {
      setAllItems([]);
      setToast(`清单读取失败：${e instanceof Error ? e.message : String(e)}`);
    }
  }, []);

  const reload = useCallback(async () => {
    const projs = (await app.WorkcostBillProjects().catch(() => [])) ?? [];
    setProjects(projs);
    await loadItems(projs);
  }, [loadItems]);

  useEffect(() => {
    loadProjects();
  }, [loadProjects]);

  useEffect(() => {
    loadItems(projects);
  }, [projects, loadItems]);

  // 筛分：项目 → 分部 → 关键字（名称/编码/特征/定额）。
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

  // 库项归并：同名+同特征+同单位 → 一条库项（跨项目复用的企业清单积累）。
  const groups = useMemo(() => {
    const m = new Map<string, BillRowItem[]>();
    for (const it of visible) {
      const k = `${it.title}|${it.unit}|${it.feature}`;
      const arr = m.get(k);
      if (arr) arr.push(it);
      else m.set(k, [it]);
    }
    return [...m.entries()]
      .map(([key, items]) => ({
        key,
        title: items[0].title,
        feature: items[0].feature,
        unit: items[0].unit,
        projects: [...new Set(items.map((x) => x.projectName))],
        quotaCodes: [...new Set(items.map((x) => x.quotaCode).filter(Boolean))],
        items,
      }))
      .sort((a, b) => b.projects.length - a.projects.length);
  }, [visible]);

  const toggleGroup = useCallback((key: string) => {
    setExpandedGroups((s) => {
      const n = new Set(s);
      if (n.has(key)) n.delete(key);
      else n.add(key);
      return n;
    });
  }, []);

  return (
    <div className="h-full flex flex-col min-h-0 text-[12.5px]">
      {/* 工具条 */}
      <div className="shrink-0 flex flex-wrap items-center gap-2 px-5 h-auto min-h-12 py-2 border-b border-border-soft/60">
        <span className="text-fg font-semibold text-[13px] flex items-center gap-1.5">
          <ListTree size={14} className="text-accent" /> 清单库
        </span>
        <span className="text-[11px] text-fg-faint hidden lg:inline">
          所有项目累计的分部分项清单 · 项目文件导入与费率在「造价参考」· 加总由五表活公式算
        </span>
        <div className="ml-auto flex items-center rounded-lg border border-border bg-bg p-0.5 text-[11px]">
          <button
            type="button"
            className={`px-2 h-6 rounded-md transition-colors ${mode === "detail" ? "bg-accent text-accent-fg" : "text-fg-faint hover:text-fg"}`}
            onClick={() => setMode("detail")}
          >
            按明细
          </button>
          <button
            type="button"
            className={`px-2 h-6 rounded-md transition-colors ${mode === "grouped" ? "bg-accent text-accent-fg" : "text-fg-faint hover:text-fg"}`}
            onClick={() => setMode("grouped")}
            title="同名+同特征+同单位聚合为一条库项"
          >
            按库项归并
          </button>
        </div>
        <button type="button" className={ghostBtn + " ml-auto"} onClick={reload} title="刷新">
          <RefreshCw size={12} />
        </button>
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
          <span className="flex-1 min-w-0">清单读取失败：{readError}</span>
          <button type="button" className={ghostBtn} onClick={reload}>重试</button>
        </div>
      )}

      {/* 筛选工具条 */}
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
        <span className="ml-auto text-[10.5px] text-fg-faint" data-testid="workcost-bill-count">
          {visible.length}/{allItems.length} 条
        </span>
      </div>

      {/* 累计清单表 */}
      <div className="flex-1 min-h-0 overflow-y-auto px-5 py-3">
        {loading ? (
          <div className="space-y-2 animate-pulse">
            <div className="v3-panel rounded-xl h-10" />
            <div className="v3-panel rounded-xl h-64" />
          </div>
        ) : projects.length === 0 ? (
          <div className="v3-panel rounded-2xl py-16 text-center">
            <ListTree size={26} className="mx-auto text-fg-faint" />
            <div className="mt-3 text-[13px] text-fg font-medium">还没有清单</div>
            <p className="mt-1.5 text-[11.5px] text-fg-faint leading-relaxed max-w-md mx-auto">
              到「造价参考」导入项目文件（五表成本测算工作簿）——清单、费率、资源、定额一次录入，
              这里随即出现所有项目累计的清单列表。
            </p>
          </div>
        ) : visible.length === 0 ? (
          <div className="v3-panel rounded-2xl py-16 text-center">
            <ListTree size={26} className="mx-auto text-fg-faint" />
            <div className="mt-3 text-[13px] text-fg font-medium">没有匹配的清单项</div>
            <p className="mt-1.5 text-[11.5px] text-fg-faint">调整上方筛选（项目/分部/关键字）后重试。</p>
          </div>
        ) : (
          mode === "detail" ? (
            <section className="v3-panel rounded-2xl p-4" data-testid="workcost-bill-items">
              <table className="w-full text-[11.5px]">
                <thead>
                  <tr className="text-left text-[10px] text-fg-faint border-b border-border-soft/50">
                    <th className="py-2 px-2 w-24">项目</th>
                    <th className="py-2 px-2 w-16">编码</th>
                    <th className="py-2 px-2">名称</th>
                    <th className="py-2 px-2">项目特征</th>
                    <th className="py-2 px-2 w-14">单位</th>
                    <th className="py-2 px-2 w-24">引用定额</th>
                    <th className="py-2 px-2 w-24 text-right">单价参考</th>
                  </tr>
                </thead>
                <tbody>
                  {visible.map((it) => (
                    <BillRow key={it.id} item={it} showProject={!selected} onAnalyze={setAnalyzeItem} refPrice={it.quotaCode ? refPrices[it.quotaCode] : undefined} />
                  ))}
                </tbody>
              </table>
            </section>
          ) : (
            <section className="space-y-2" data-testid="workcost-bill-grouped">
              {groups.map((g) => {
                const open = expandedGroups.has(g.key);
                return (
                  <div key={g.key} className="v3-panel rounded-xl">
                    <button
                      type="button"
                      className="w-full flex items-center gap-3 px-4 py-2.5 text-left hover:bg-bg-elev/30 transition-colors"
                      onClick={() => toggleGroup(g.key)}
                    >
                      <span className="min-w-0 flex-1">
                        <span className="block truncate text-[12px] text-fg font-medium">
                          {g.title}
                          {g.feature && <span className="ml-2 text-[10.5px] text-fg-faint font-normal">{g.feature}</span>}
                        </span>
                        <span className="block truncate text-[10px] text-fg-faint mt-0.5">
                          {g.unit || "—"} · 被 {g.projects.length} 个项目使用 · {g.quotaCodes.length > 0 ? `定额 ${g.quotaCodes.join(" / ")}` : "未套定额"}
                        </span>
                      </span>
                      <span className="shrink-0 px-1.5 py-px rounded bg-accent/10 text-accent text-[10px]">
                        {g.projects.length} 项目 · {g.items.length} 条
                      </span>
                    </button>
                    {open && (
                      <ul className="px-6 pb-3 space-y-1 text-[11px]">
                        {g.items.map((it) => (
                          <li key={it.id} className="flex items-center gap-2 border-t border-border-soft/20 pt-1 first:border-0">
                            <span className="text-fg-faint w-40 truncate" title={it.projectName}>{it.projectName}</span>
                            <span className="font-mono text-[10px] text-fg-faint">{it.code}</span>
                            {it.quantityExpr && <span className="text-fg-faint truncate">计算式：{it.quantityExpr}</span>}
                            {it.quotaCode && (
                              <button
                                type="button"
                                className="ml-auto font-mono text-[10px] px-1.5 py-px rounded bg-accent/10 text-accent hover:bg-accent/20 transition-colors shrink-0"
                                title={`引用定额 ${it.quotaCode}——点开综合单价分析（按挂接定额分组）`}
                                onClick={() => setAnalyzeItem(it)}
                              >
                                {it.quotaCode}
                              </button>
                            )}
                          </li>
                        ))}
                      </ul>
                    )}
                  </div>
                );
              })}
            </section>
          )
        )}
      </div>

      {analyzeItem && <QuotaAnalysisModal item={analyzeItem} onClose={() => setAnalyzeItem(null)} />}
    </div>
  );
}

// BillRow 一条清单项（只读）：**整行可点**开综合单价分析（用户心智=「点击清单
// 看组价明细」，此前只有 10px 定额码徽章可点，真机反馈「点击清单无法查看组价
// 明细」——可点面太小）；徽章保留为视觉锚点。手填行（未套定额）不可点。
function BillRow({
  item,
  showProject,
  onAnalyze,
  refPrice,
}: {
  item: WorkcostBillItem & { projectName?: string };
  showProject?: boolean;
  onAnalyze: (item: WorkcostBillItem) => void;
  refPrice?: number;
}) {
  const clickable = Boolean(item.quotaCode);
  return (
    <tr
      className={`border-b border-border-soft/25 last:border-0 hover:bg-bg-elev/30 ${clickable ? "cursor-pointer" : ""}`}
      onClick={clickable ? () => onAnalyze(item) : undefined}
      title={clickable ? `点开综合单价分析：${item.title}（按挂接定额分组展示工料机明细）` : "未套定额（手填项）——单价即行内手填价"}
    >
      {showProject && (
        <td className="py-1.5 px-2 max-w-[9rem]" title={item.projectName}>
          <span className="block truncate text-[10px] text-fg-faint">{item.projectName}</span>
        </td>
      )}
      <td className="py-1.5 px-2 font-mono text-[10px] text-fg-faint">{item.code}</td>
      <td className="py-1.5 px-2 text-fg">{item.title}</td>
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
      <td className="py-1.5 px-2 text-fg-faint">{item.unit || "-"}</td>
      <td className="py-1.5 px-2">
        {item.quotaCode ? (
          <button
            type="button"
            className="font-mono text-[10px] px-1.5 py-px rounded bg-accent/10 text-accent hover:bg-accent/20 transition-colors"
            title={`引用定额 ${item.quotaCode}——点开综合单价分析（按挂接定额分组）`}
            onClick={() => onAnalyze(item)}
          >
            {item.quotaCode}
          </button>
        ) : (
          <span className="text-[10px] text-fg-faint" title="未套定额（手填项）">手填</span>
        )}
      </td>
      <td className="py-1.5 px-2 text-right">
        {item.quotaCode ? (
          refPrice === undefined ? (
            <span className="text-fg-faint">…</span>
          ) : (
            <span className="tabular-nums text-fg-dim" title="按当前资源价现算的综合单价（人材机）——参考值，导出后由 Excel 活公式重算">
              {fmtPrice(refPrice)}
            </span>
          )
        ) : item.priceOverride > 0 ? (
          <span className="tabular-nums text-fg-dim" title="手填单价（录入数据）">{fmtPrice(item.priceOverride)}</span>
        ) : (
          <span className="text-fg-faint">—</span>
        )}
      </td>
    </tr>
  );
}

// ── 综合单价分析（1:N 多定额组合）──────────────────────────────────

// QuotaAnalysisModal 清单项的综合单价分析（「清单分析」入口，SchemaV30 1:N）：
// 按挂接的定额分组展示——每条定额一个区块（定额头 + 工料机明细 + 该定额综合
// 单价）；底部「挂接定额」区支持从定额库搜索挂接/解挂/改工程量。数据全部现算
// （WorkcostQuotaCompose），资源调价即最新口径；加总仍归五表 Excel。
export function QuotaAnalysisModal({ item, onClose }: { item: WorkcostBillItem; onClose: () => void }) {
  const [links, setLinks] = useState<WorkcostBillQuotaLink[]>([]);
  const [composes, setComposes] = useState<Record<string, WorkcostCompose>>({});
  const [quotaHeads, setQuotaHeads] = useState<Record<string, WorkcostQuota>>({});
  const [linksLoading, setLinksLoading] = useState(true);
  const [error, setError] = useState("");
  const [kw, setKw] = useState("");
  const [candidates, setCandidates] = useState<WorkcostQuota[]>([]);
  const [busy, setBusy] = useState(false);
  const [toast, setToast] = useState("");

  const reloadLinks = useCallback(async () => {
    try {
      const ls = (await app.WorkcostBillQuotaLinks(item.id)) ?? [];
      setLinks(ls);
      return ls;
    } catch (e) {
      setError(`定额引用读取失败：${e instanceof Error ? e.message : String(e)}`);
      setLinks([]);
      return [];
    } finally {
      setLinksLoading(false);
    }
  }, [item.id]);

  useEffect(() => {
    void reloadLinks();
  }, [reloadLinks]);

  // 引用变化 → 批量现算各定额明细 + 拉定额头。
  useEffect(() => {
    if (links.length === 0) {
      setComposes({});
      setQuotaHeads({});
      return;
    }
    let alive = true;
    Promise.all(links.map((l) => app.WorkcostQuotaCompose(l.quotaCode, null).catch(() => null)))
      .then((cs) => {
        if (!alive) return;
        const m: Record<string, WorkcostCompose> = {};
        cs.forEach((c, i) => {
          if (c) m[links[i].quotaCode] = c;
        });
        setComposes(m);
      });
    Promise.all(links.map((l) => Promise.resolve(app.WorkcostQuotaGet(l.quotaCode)).catch(() => null)))
      .then((qs) => {
        if (!alive) return;
        const m: Record<string, WorkcostQuota> = {};
        qs.forEach((q, i) => {
          if (q) m[links[i].quotaCode] = q;
        });
        setQuotaHeads(m);
      });
    return () => {
      alive = false;
    };
  }, [links]);

  const attach = useCallback(async (code: string) => {
    setBusy(true);
    try {
      await app.WorkcostBillQuotaAttach(item.id, code, 0);
      await reloadLinks();
      setToast(`已挂接定额 ${code}`);
    } catch (e) {
      setToast(`挂接失败：${e instanceof Error ? e.message : String(e)}`);
    } finally {
      setBusy(false);
    }
  }, [item.id, reloadLinks]);

  const detach = useCallback(async (linkId: number, code: string) => {
    setBusy(true);
    try {
      await app.WorkcostBillQuotaDetach(item.id, linkId);
      await reloadLinks();
      setToast(`已解挂定额 ${code}`);
    } catch (e) {
      setToast(`解挂失败：${e instanceof Error ? e.message : String(e)}`);
    } finally {
      setBusy(false);
    }
  }, [item.id, reloadLinks]);

  const setQuantity = useCallback(async (code: string, quantity: number) => {
    try {
      await app.WorkcostBillQuotaAttach(item.id, code, quantity);
      await reloadLinks();
    } catch (e) {
      setToast(`工程量保存失败：${e instanceof Error ? e.message : String(e)}`);
    }
  }, [item.id, reloadLinks]);

  const search = useCallback(async () => {
    const k = kw.trim();
    if (!k) {
      setCandidates([]);
      return;
    }
    try {
      const list = (await app.WorkcostQuotaList("", k)) ?? [];
      setCandidates(list.slice(0, 8));
    } catch {
      setCandidates([]);
    }
  }, [kw]);

  const f = new Intl.NumberFormat("zh-CN", { maximumFractionDigits: 2 });
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" onClick={onClose}>
      <div
        className="v3-panel rounded-2xl p-5 space-y-3 w-[46rem] max-h-[84vh] overflow-y-auto"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center gap-2">
          <span className="text-[13px] text-f font-semibold flex-1 min-w-0 truncate">
            综合单价分析：{item.title}
          </span>
          <button type="button" className={iconBtn} onClick={onClose} title="关闭">
            <X size={11} />
          </button>
        </div>
        <div className="flex flex-wrap gap-x-4 gap-y-1 text-[11px] text-fg-faint">
          <span className="font-mono text-accent">{item.code}</span>
          {item.unit && <span>单位：{item.unit}</span>}
          {item.quantity > 0 && <span>工程量：{f.format(item.quantity)}</span>}
          {item.feature && <span className="max-w-[24rem] truncate" title={item.feature}>特征：{item.feature}</span>}
        </div>
        {error ? (
          <div role="alert" className="text-[11px] text-amber-300 border border-amber-400/30 bg-amber-400/10 rounded-lg px-2.5 py-2">
            {error}
          </div>
        ) : linksLoading ? (
          <div className="space-y-2 animate-pulse">
            <div className="v3-panel rounded-lg h-16" />
            <div className="v3-panel rounded-lg h-40" />
          </div>
        ) : links.length === 0 ? (
          <div className="v3-panel rounded-xl px-4 py-6 text-center text-[11.5px] text-fg-faint">
            该清单项还没有挂接定额——下方搜索定额库挂接后，这里按定额展示工料机明细。
          </div>
        ) : (
          <div className="space-y-3">
            {links.map((l, idx) => {
              const c = composes[l.quotaCode];
              const head = quotaHeads[l.quotaCode];
              const rows = c?.lines ?? [];
              const qty = l.quantity > 0 ? l.quantity : item.quantity;
              return (
                <div key={l.id} className="v3-panel rounded-xl p-3">
                  <div className="flex items-center gap-2 mb-2">
                    <span className="text-[11px] text-fg-faint shrink-0">定额 {idx + 1}</span>
                    <span className="font-mono text-[11px] text-accent shrink-0">{l.quotaCode}</span>
                    <span className="text-[11.5px] text-fg font-medium truncate flex-1 min-w-0">{head?.title ?? ""}</span>
                    {head?.origCode && head.origCode !== l.quotaCode && (
                      <span className="font-mono text-[10px] px-1.5 py-px rounded bg-bg-soft text-fg-faint shrink-0" title="工作簿原始编码（定额编码统一前的溯源）">
                        原码 {head.origCode}
                      </span>
                    )}
                    <label className="text-[10px] text-fg-faint shrink-0" title="该定额参与合价的工程量；0 = 跟随清单工程量">
                      {"工程量 "}
                      <input
                        className="w-20 px-1.5 h-6 rounded-md bg-bg-elev border border-border text-[11px] tabular-nums text-fg"
                        defaultValue={l.quantity}
                        onBlur={(e) => {
                          const v = Number(e.target.value);
                          if (!Number.isNaN(v) && v !== l.quantity) void setQuantity(l.quotaCode, v);
                        }}
                      />
                    </label>
                    <button
                      type="button"
                      className={iconBtn}
                      title={`解挂定额 ${l.quotaCode}`}
                      disabled={busy}
                      onClick={() => void detach(l.id, l.quotaCode)}
                    >
                      <X size={11} />
                    </button>
                  </div>
                  {!c ? (
                    <div className="text-[11px] text-fg-faint animate-pulse">现算中…</div>
                  ) : rows.length === 0 ? (
                    <div className="text-[11px] text-fg-faint">该定额没有工料机消耗行——综合单价为 0，需先录入消耗量。</div>
                  ) : (
                    <>
                      <div className="v3-panel rounded-lg overflow-x-auto">
                        <table className="w-full text-[11px]">
                          <thead>
                            <tr className="text-left text-[10px] text-fg-faint border-b border-border-soft/50">
                              <th className="py-1 px-2 w-14">行类</th>
                              <th className="py-1 px-2">资源</th>
                              <th className="py-1 px-2 w-12">单位</th>
                              <th className="py-1 px-2 w-20 text-right">含量</th>
                              <th className="py-1 px-2 w-20 text-right">单价</th>
                              <th className="py-1 px-2 w-24 text-right">金额</th>
                            </tr>
                          </thead>
                          <tbody>
                            {rows.map((row, i) => (
                              <tr key={i} className="border-b border-border-soft/20 last:border-0">
                                <td className="py-1 px-2 text-fg-dim">{row.kind}</td>
                                <td className="py-1 px-2 text-fg-dim">{row.title || "-"}</td>
                                <td className="py-1 px-2 text-fg-faint">{row.unit || "-"}</td>
                                <td className="py-1 px-2 text-right tabular-nums">{row.quantity > 0 ? f.format(row.quantity) : "—"}</td>
                                <td className="py-1 px-2 text-right tabular-nums">
                                  {row.price > 0 ? fmtPrice(row.price) : <span className="text-amber-300">缺价</span>}
                                </td>
                                <td className="py-1 px-2 text-right tabular-nums">{fmtPrice(row.amount)}</td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      </div>
                      <div className="mt-1.5 flex flex-wrap items-center justify-end gap-x-3 text-[11px] text-fg-dim px-1">
                        <span>
                          该定额综合单价（人材机）
                          <span className="tabular-nums text-accent ml-1">{fmtPrice(c.compositePrice)}</span>
                        </span>
                        <span className="text-fg-faint">
                          合价参考 <span className="tabular-nums">{fmtPrice(c.compositePrice * (qty > 0 ? qty : 0))}</span>
                          （{f.format(qty)} × 单价，导出后由 Excel 重算）
                        </span>
                      </div>
                    </>
                  )}
                </div>
              );
            })}
          </div>
        )}

        {/* 挂接定额：从定额库搜索 */}
        <div className="v3-panel rounded-xl p-3 space-y-2">
          <div className="flex items-center gap-1.5">
            <span className="text-[11.5px] text-fg font-medium">挂接定额</span>
            <span className="text-[10px] text-fg-faint">从定额库搜索后挂接到本清单项（可挂多条，各带工程量）</span>
          </div>
          <div className="flex items-center gap-1.5">
            <input
              className={`${fieldCls} !w-64`}
              value={kw}
              placeholder="搜定额编码/名称"
              aria-label="搜索定额"
              onChange={(e) => setKw(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") void search();
              }}
            />
            <button type="button" className={ghostBtn} onClick={() => void search()}>搜索</button>
          </div>
          {candidates.length > 0 && (
            <ul className="space-y-1">
              {candidates.map((q) => (
                <li key={q.code} className="flex items-center gap-2 text-[11px]">
                  <span className="font-mono text-accent shrink-0">{q.code}</span>
                  <span className="text-fg-dim truncate flex-1 min-w-0">{q.title}</span>
                  <span className="text-fg-faint shrink-0">{q.unit || "-"}</span>
                  <button
                    type="button"
                    className="inline-flex items-center px-2 h-6 rounded-md bg-accent text-accent-fg text-[10.5px] hover:opacity-90 disabled:opacity-50 shrink-0"
                    disabled={busy || links.some((l) => l.quotaCode === q.code)}
                    onClick={() => void attach(q.code)}
                  >
                    {links.some((l) => l.quotaCode === q.code) ? "已挂接" : "挂接"}
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>

        {toast && (
          <div className="v3-panel rounded-xl px-3 py-2 text-[11px] text-fg-dim">
            {toast}
            <button type="button" className={ghostBtn + " ml-2"} onClick={() => setToast("")}>关闭</button>
          </div>
        )}
        <p className="text-[10px] text-fg-faint">
          按当前资源价现算（参考口径）；清单合价 = Σ(定额工程量 × 定额单价)，由导出的五表工作簿活公式计算。
        </p>
      </div>
    </div>
  );
}
