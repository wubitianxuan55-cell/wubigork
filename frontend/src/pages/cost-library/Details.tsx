// Details — 右栏详情：清单项的综合单价分析（按挂接定额分组，带「消耗量 = 工程量 ×
// 含量」）、定额的含量行分析表与引用它的清单、工料机的价格卡（地区 · 期）与价格表。
// 数据现算（点开才取），每一级都给出下一级与上一级的入口：清单 ⇄ 定额 ⇄ 工料机。
// 2026-11-27 起带头部「编辑」钮：进编辑面直接修正录入数据（Edit.tsx），修完回
// 只读视图并整页静默重读。onSaved 没传（旧调用方/测试）就不出编辑钮。
import { useEffect, useState, type ReactNode } from "react";
import { AlertCircle, Loader2, Pencil } from "../../gaea/icons";
import { app } from "../../gaea/lib/bridge";
import type {
  WorkcostCompose, WorkcostQuota, WorkcostQuotaItem, WorkcostResource,
  WorkcostResourcePrice, WorkcostUsageRef,
} from "../../gaea/lib/types";
import {
  alignItems, consumptionOf, dimLabel, effectivePrice, fmtMoney, fmtPeriod, fmtQty, fmtRate, isCrossSpecialty,
  linkedQuotas, periodKey, quotaFactor, unitPriceOf, type BillRow,
} from "./logic";
import { BillEdit, QuotaEdit, ResourceEdit } from "./Edit";
import { KindBadge } from "./Tables";

type Phase = "idle" | "loading" | "ready" | "failed";

function message(e: unknown): string {
  return e instanceof Error ? e.message : String(e);
}

/** 详情头部的编辑钮（onSaved 给了才出）。 */
function EditButton({ onClick, testid }: { onClick: () => void; testid: string }) {
  return (
    <button type="button" className="cl-btn cl-detail-edit" data-testid={testid} onClick={onClick}>
      <Pencil size={12} aria-hidden />编辑
    </button>
  );
}

export function DetailState({ phase, error, children }: { phase: Phase; error?: string; children?: ReactNode }) {
  if (phase === "idle") return <div className="cl-state"><span className="cl-state-title">在左侧列表选一行查看详情</span></div>;
  if (phase === "loading") return <div className="cl-state" role="status"><Loader2 size={16} className="cl-spin" aria-hidden /><span className="cl-state-title">正在现算…</span></div>;
  if (phase === "failed") {
    return <div className="cl-state is-failed" role="alert"><AlertCircle size={16} aria-hidden /><span className="cl-state-title">读取失败</span><span>{error}</span></div>;
  }
  return <>{children}</>;
}

/** 综合单价分析表的一行：含量 × 单价 = 金额；带上资源编码才能下钻到工料机。 */
function AnalysisTable({ compose, items, factor, onOpenResource }: {
  compose: WorkcostCompose;
  items: WorkcostQuotaItem[];
  /** 该定额在这条清单上的工程量；给了才显示「消耗量」列（清单项下）。 */
  factor?: number;
  onOpenResource: (code: string) => void;
}) {
  const lines = compose.lines ?? [];
  const aligned = alignItems(lines, items);
  const withUse = factor !== undefined;
  if (lines.length === 0) {
    return <p className="cl-note">该定额没有工料机消耗行——综合单价为 0。补含量行走造价模式的 cost_library save_quota。</p>;
  }
  return (
    <div className="cl-analysis" data-testid={withUse ? "cost-bill-analysis" : "cost-quota-analysis"}>
      <table className="cl-table cl-table-inner">
        <thead>
          <tr>
            <th>行类</th><th>资源</th><th>单位</th><th className="cl-num">含量</th>
            {withUse && <th className="cl-num">消耗量</th>}
            <th className="cl-num">单价</th><th className="cl-num">金额</th><th className="cl-num">占比</th>
          </tr>
        </thead>
        <tbody>
          {lines.map((l, i) => {
            const code = aligned[i]?.resourceCode ?? "";
            const use = withUse ? consumptionOf(factor, l.quantity) : null;
            return (
              <tr key={i}>
                <td><KindBadge kind={l.kind} /></td>
                <td className="cl-cell-title">
                  {code ? (
                    <button type="button" className="cl-cell-link" title={`下钻到工料机 ${code}`}
                      aria-label={`下钻到工料机 ${l.title || code}`} onClick={() => onOpenResource(code)}>{l.title || code}</button>
                  ) : (l.title || <span className="cl-dim">—</span>)}
                </td>
                <td>{l.unit || "—"}</td>
                <td className="cl-num">{fmtQty(l.quantity)}</td>
                {withUse && (
                  <td className="cl-num" data-testid="cost-consumption" title="消耗量 = 该定额在这条清单上的工程量 × 定额含量">
                    {use === null ? <span className="cl-dim">—</span> : `${fmtQty(use)}${l.unit ? ` ${l.unit}` : ""}`}
                  </td>
                )}
                <td className="cl-num">{l.price > 0 ? fmtMoney(l.price) : <span className="cl-warn-text" title="资源无价，该项不计入合计">缺价</span>}</td>
                <td className="cl-num">{fmtMoney(l.amount)}</td>
                <td className="cl-num cl-dim">{l.sharePct > 0 ? `${l.sharePct.toFixed(1)}%` : "—"}</td>
              </tr>
            );
          })}
        </tbody>
        <tfoot>
          <tr className="cl-tfoot-fees">
            <td colSpan={withUse ? 8 : 7}>
              <div className="cl-fee-summary" data-testid="cost-fee-summary">
                <span className="cl-fee-summary-label">三费</span>
                <span className="cl-fee-item">人工 <b className="is-labor">{fmtMoney(compose.laborFee)}</b></span>
                <span className="cl-fee-item">材料 <b className="is-material">{fmtMoney(compose.materialFee)}</b></span>
                <span className="cl-fee-item">机械 <b className="is-machine">{fmtMoney(compose.machineFee)}</b></span>
                {compose.outsourcedFee > 0 && (
                  <span className="cl-fee-outsourced cl-dim">其中外委 {fmtMoney(compose.outsourcedFee)}（已并入材料）</span>
                )}
              </div>
            </td>
          </tr>
          <tr className="cl-tfoot-total">
            <td colSpan={withUse ? 6 : 5}>综合单价（人材机，不含管理费 / 利润 / 税金）</td>
            <td colSpan={2} className="cl-num">{fmtMoney(compose.compositePrice)}</td>
          </tr>
        </tfoot>
      </table>
      {(compose.warnings ?? []).map((w, i) => <p key={i} className="cl-warn-text">· {w}</p>)}
      {compose.zeroLines > 0 && <p className="cl-note">该定额有 {compose.zeroLines} 条零含量 / 零单价行（工序模板列出但本项未用），不计入合计。</p>}
    </div>
  );
}

// ── 清单项 ───────────────────────────────────────────────────────────

interface QuotaBlock {
  code: string;
  quota: WorkcostQuota | null;
  compose: WorkcostCompose | null;
  error: string;
}

export function BillDetail({ item, refPrices, onOpenQuota, onOpenResource, specialtyOptions, onSaved }: {
  item: BillRow;
  refPrices: Record<string, number>;
  onOpenQuota: (code: string) => void;
  onOpenResource: (code: string) => void;
  /** 编辑面：库里已有的专业取值；保存后的回执交给页面（提示 + 静默重读）。 */
  specialtyOptions?: string[];
  onSaved?: (msg: string) => void;
}) {
  const [phase, setPhase] = useState<Phase>("loading");
  const [blocks, setBlocks] = useState<QuotaBlock[]>([]);
  const [editing, setEditing] = useState(false);

  // 换行就退出编辑态——不让上一行的草稿挂在新行上。
  useEffect(() => { setEditing(false); }, [item.id]);

  useEffect(() => {
    const codes = linkedQuotas(item);
    if (codes.length === 0) {
      setBlocks([]);
      setPhase("ready");
      return;
    }
    let alive = true;
    setPhase("loading");
    void Promise.all(
      codes.map((code) =>
        Promise.all([app.WorkcostQuotaGet(code), app.WorkcostQuotaCompose(code, null)])
          .then(([quota, compose]) => ({ code, quota: quota ?? null, compose: compose ?? null, error: "" }))
          .catch((e: unknown) => ({ code, quota: null, compose: null, error: message(e) })),
      ),
    ).then((got) => {
      if (!alive) return;
      setBlocks(got);
      setPhase(got.every((b) => !b.compose) ? "failed" : "ready");
    });
    return () => { alive = false; };
  }, [item]);

  const cross = blocks.filter((b) => b.quota && isCrossSpecialty(item.specialty ?? "", b.quota.specialty)).length;

  return (
    <article className="cl-detail" data-testid="cost-detail-bill">
      <header className="cl-detail-head">
        <div className="cl-kicker">清单 · {item.projectName}</div>
        {onSaved && !editing && <EditButton testid="cost-bill-edit-open" onClick={() => setEditing(true)} />}
        <h2 className="cl-detail-title">{item.title}</h2>
        <div className="cl-meta">
          <span className="cl-code">{item.code}</span>
          {item.gcode && <span className="cl-tag" data-testid="cost-bill-gcode" title="库内自有码（Q+专业+分部+流水；与外部码一一对应）">G {item.gcode}</span>}
          {item.unit && <span>单位 {item.unit}</span>}
          {item.quantity > 0 && <span>工程量 {fmtQty(item.quantity)}</span>}
          {item.division && <span>分部 {item.division}</span>}
          {item.specialty
            ? <span data-testid="cost-bill-specialty">专业 {item.specialty}</span>
            : <span className="cl-dim" title="还没标注专业——在造价模式用 cost_library classify 归类">专业 未标注</span>}
        </div>
        {item.feature && <p className="cl-detail-feature" title={item.quantityExpr ? `${item.feature}｜计算式：${item.quantityExpr}` : undefined}>{item.feature}</p>}
        <dl className="cl-price-row">
          <div><dt>原表单价</dt><dd>{fmtMoney((item.listPrice ?? 0) > 0 ? item.listPrice : null)}</dd></div>
          <div><dt>{item.priceOverride > 0 ? "手填单价" : "现算单价"}</dt><dd className="cl-strong">{fmtMoney(unitPriceOf(item, refPrices))}</dd></div>
        </dl>
      </header>
      {editing && onSaved ? (
        <BillEdit item={item} specialtyOptions={specialtyOptions ?? []}
          onSaved={onSaved} onCancel={() => setEditing(false)} />
      ) : (
        <>
          <DetailState phase={phase} error={blocks[0]?.error}>
            {blocks.length === 0 ? (
              <p className="cl-note">这条清单项没有挂接定额（手填项）。点右上「编辑」可以直接挂接定额或改字段。</p>
            ) : (
              blocks.map((b) => {
                const crossSpec = b.quota && isCrossSpecialty(item.specialty ?? "", b.quota.specialty) ? b.quota.specialty : "";
                return (
                  <section key={b.code} className="cl-block" data-testid="cost-bill-quota-block">
                    <div className="cl-block-head">
                      <span className="cl-block-label">定额</span>
                      <button type="button" className="cl-chip" title={`下钻到定额 ${b.code}`}
                        aria-label={`下钻到定额 ${b.code}`} onClick={() => onOpenQuota(b.code)}>{b.code}</button>
                      <span className="cl-block-title">{b.quota?.title ?? ""}</span>
                      {b.quota?.origCode && b.quota.origCode !== b.quota.code && (
                        <span className="cl-tag" title="工作簿原始编码">原码 {b.quota.origCode}</span>
                      )}
                      {b.quota?.derivedFrom && (
                        <span className="cl-tag is-derived" title="借鉴生成的溯源：本专业没有可用定额，以外专业定额为底生成">{b.quota.derivedFrom}</span>
                      )}
                      {crossSpec && (
                        <span className="cl-tag is-cross" data-testid="cost-bill-cross-specialty"
                          title="清单套定额同专业优先：这条定额是别的专业的。要本专业的，用 cost_library borrow_quota 借鉴生成后再挂接">
                          外专业 · {crossSpec}
                        </span>
                      )}
                    </div>
                    {b.compose ? <AnalysisTable compose={b.compose} items={b.quota?.items ?? []} factor={quotaFactor(item, b.code)} onOpenResource={onOpenResource} /> : <p className="cl-warn-text">现算失败：{b.error}</p>}
                  </section>
                );
              })
            )}
          </DetailState>
          {cross > 0 && (
            <p className="cl-warn-text" data-testid="cost-bill-cross-note">
              {cross} 条定额属于别的专业。清单套定额同专业优先——在造价模式用 cost_library borrow_quota 以外专业定额为底生成本专业的，再把这条清单改挂过去。
            </p>
          )}
          <p className="cl-foot-note">按当前资源价现算（参考口径）；消耗量 = 该定额在这条清单上的工程量 × 定额含量；取费在项目合计层。</p>
        </>
      )}
    </article>
  );
}

// ── 定额 ─────────────────────────────────────────────────────────────

export function QuotaDetail({ quota, usage, onOpenBill, onOpenResource, specialtyOptions, chapterOptionsOf, resources, regionOptions, onSaved }: {
  quota: WorkcostQuota;
  /** 引用它的清单行（页面按全量清单算好的引用索引）。 */
  usage: BillRow[];
  onOpenBill: (row: BillRow) => void;
  onOpenResource: (code: string) => void;
  /** 编辑面：专业取值 / 各专业下的分部 / 地区取值 / 工料机全量（加含量行用）。 */
  specialtyOptions?: string[];
  chapterOptionsOf?: (specialty: string) => string[];
  resources?: WorkcostResource[];
  regionOptions?: string[];
  onSaved?: (msg: string) => void;
}) {
  const [phase, setPhase] = useState<Phase>("loading");
  const [full, setFull] = useState<WorkcostQuota | null>(null);
  const [compose, setCompose] = useState<WorkcostCompose | null>(null);
  const [error, setError] = useState("");
  const [editing, setEditing] = useState(false);

  // 换定额就退出编辑态——不让上一条的草稿挂在新定额上。
  useEffect(() => { setEditing(false); }, [quota.code]);

  useEffect(() => {
    let alive = true;
    setPhase("loading");
    setError("");
    void Promise.all([app.WorkcostQuotaGet(quota.code), app.WorkcostQuotaCompose(quota.code, null)])
      .then(([q, c]) => {
        if (!alive) return;
        setFull(q ?? null);
        setCompose(c ?? null);
        setPhase(c ? "ready" : "failed");
        if (!c) setError("综合单价现算没有返回结果");
      })
      .catch((e: unknown) => {
        if (!alive) return;
        setError(message(e));
        setPhase("failed");
      });
    return () => { alive = false; };
  }, [quota.code]);

  const cross = usage.filter((it) => isCrossSpecialty(it.specialty ?? "", quota.specialty)).length;

  return (
    <article className="cl-detail" data-testid="cost-detail-quota">
      <header className="cl-detail-head">
        <div className="cl-kicker">定额 · {dimLabel(quota.specialty)}专业</div>
        {onSaved && !editing && <EditButton testid="cost-quota-edit-open" onClick={() => setEditing(true)} />}
        <h2 className="cl-detail-title">{quota.title}</h2>
        <div className="cl-meta">
          <span className="cl-code">{quota.code}</span>
          {quota.gcode && <span className="cl-tag" data-testid="cost-quota-gcode" title="库内自有码（D+专业+分部+流水；与外部码一一对应）">G {quota.gcode}</span>}
          {quota.origCode && quota.origCode !== quota.code && <span title="工作簿原始编码">原码 {quota.origCode}</span>}
          {quota.unit && <span>计量单位 {quota.unit}</span>}
          {quota.chapter
            ? <span>分部 {quota.chapter}</span>
            : <span className="cl-dim" title="还没归分部——在造价模式用 cost_library classify 归类">分部 未归</span>}
          {quota.region
            ? <span title="地区与上游清单项目挂钩">地区 {quota.region}</span>
            : <span className="cl-dim" title="还没标地区——随上游清单项目导入带上，或点编辑补标">地区 未标</span>}
          <span>被 {usage.length} 条清单项引用</span>
        </div>
        {quota.derivedFrom && (
          <p className="cl-note">
            <span className="cl-tag is-derived" title="借鉴生成的溯源：本专业没有可用定额，以外专业定额为底生成">{quota.derivedFrom}</span>
            {" "}含量行照搬自源定额，按本项目特征核对后再用。
          </p>
        )}
      </header>
      {editing && onSaved && full ? (
        <QuotaEdit quota={full} specialtyOptions={specialtyOptions ?? []} chapterOptionsOf={chapterOptionsOf ?? (() => [])}
          resources={resources ?? []} regionOptions={regionOptions ?? []} onSaved={onSaved} onCancel={() => setEditing(false)} />
      ) : (
        <>
          <DetailState phase={phase} error={error}>
            {full && compose && <AnalysisTable compose={compose} items={full.items ?? []} onOpenResource={onOpenResource} />}
          </DetailState>
          <section className="cl-block">
            <div className="cl-block-head"><span className="cl-block-label">引用它的清单项</span><span className="cl-dim">{usage.length} 条{cross > 0 ? ` · ${cross} 条是别的专业的清单` : ""}</span></div>
            {usage.length === 0 ? (
              <p className="cl-note">还没有清单项套用这条定额。</p>
            ) : (
              <ul className="cl-usage" data-testid="cost-quota-usage">
                {usage.map((it) => {
                  const crossSpec = isCrossSpecialty(it.specialty ?? "", quota.specialty);
                  return (
                    <li key={it.id}>
                      <button type="button" className="cl-usage-item" title={`查看清单项「${it.title}」`} onClick={() => onOpenBill(it)}>
                        <span className="cl-usage-title">{it.title}</span>
                        <span className="cl-meta">
                          <span className="cl-code">{it.code}</span><span>{it.projectName}</span>
                          {it.quantity > 0 && <span>量 {fmtQty(it.quantity)}</span>}
                          {it.specialty
                            ? <span className={crossSpec ? "cl-warn-text" : ""}>{crossSpec ? "外专业清单 · " : ""}{it.specialty}</span>
                            : <span className="cl-dim">专业 未标注</span>}
                        </span>
                      </button>
                    </li>
                  );
                })}
              </ul>
            )}
          </section>
          <p className="cl-foot-note">含量行是每单位的消耗标准；资源一调价，引用它的综合单价下次读取即重算。定额专业固定——同样一件事各专业各有定额。</p>
        </>
      )}
    </article>
  );
}

// ── 工料机 ───────────────────────────────────────────────────────────

export function ResourceDetail({ resource, onOpenQuota, regionOptions, periodOptions, categoryOptions, onSaved }: {
  resource: WorkcostResource;
  onOpenQuota: (code: string) => void;
  /** 编辑面：库里已有的地区 / 期数取值与该类别下的分类路径。 */
  regionOptions?: string[];
  periodOptions?: string[];
  categoryOptions?: string[];
  onSaved?: (msg: string) => void;
}) {
  const [phase, setPhase] = useState<Phase>("loading");
  const [error, setError] = useState("");
  const [history, setHistory] = useState<WorkcostResourcePrice[]>([]);
  const [usage, setUsage] = useState<WorkcostUsageRef[]>([]);
  const [editing, setEditing] = useState(false);

  // 换行就退出编辑态——不让上一行的草稿挂在新行上。
  useEffect(() => { setEditing(false); }, [resource.id]);

  useEffect(() => {
    let alive = true;
    setPhase("loading");
    setError("");
    void Promise.all([app.WorkcostResourcePrices(resource.id), app.WorkcostResourceUsage(resource.code)])
      .then(([h, u]) => {
        if (!alive) return;
        setHistory(h ?? []);
        setUsage(u ?? []);
        setPhase("ready");
      })
      .catch((e: unknown) => {
        if (!alive) return;
        setError(message(e));
        setPhase("failed");
      });
    return () => { alive = false; };
  }, [resource.id, resource.code]);

  const price = effectivePrice(resource);
  const facts: [string, string][] = [
    ["规格", resource.spec], ["单位", resource.unit], ["损耗率", resource.lossRate > 0 ? fmtRate(resource.lossRate) : ""],
    ["分类", resource.categoryPath], ["来源", resource.source], ["供应商", resource.supplier],
    ["地区", resource.region], ["价格期数", resource.priceDate ? fmtPeriod(resource.priceDate) : ""], ["价格口径", resource.priceType],
    ["有效期至", resource.validUntil], ["状态", resource.status === "现行" ? "" : resource.status],
  ].filter(([, v]) => v) as [string, string][];
  // 工料机按地区分行，时间是该行下的各期价格：地区、期、分类还没标注的如实点名。
  const missing = [
    !resource.region && "地区", !resource.priceDate && "期数", !resource.categoryPath && "分类",
  ].filter(Boolean) as string[];
  // 价格表新→旧（按期）；认不出期数的排最后。
  const priced = [...history].sort((a, b) => {
    if (!a.period) return 1;
    if (!b.period) return -1;
    return periodKey(b.period) - periodKey(a.period);
  });

  return (
    <article className="cl-detail cl-detail-resource" data-testid="cost-detail-resource">
      <header className="cl-detail-head">
        <div className="cl-kicker">
          {"工料机"}
          {resource.region ? ` · ${resource.region}` : ""}
          {resource.priceDate ? ` · ${fmtPeriod(resource.priceDate)}` : ""}
        </div>
        {onSaved && !editing && <EditButton testid="cost-resource-edit-open" onClick={() => setEditing(true)} />}
        <h2 className="cl-detail-title">{resource.title}</h2>
        <div className="cl-meta"><KindBadge kind={resource.kind} /><span className="cl-code">{resource.code}</span>
          {resource.gcode && <span className="cl-tag" data-testid="cost-resource-gcode" title="库内自有码（R+类别+分类+流水；与来源码一一对应）">G {resource.gcode}</span>}</div>
      </header>
      {editing && onSaved ? (
        <ResourceEdit resource={resource} regionOptions={regionOptions ?? []} periodOptions={periodOptions ?? []}
          categoryOptions={categoryOptions ?? []} onSaved={onSaved} onCancel={() => setEditing(false)} />
      ) : (
        <>
          <section className="cl-block cl-resource-price-block">
            <div className="cl-block-head">
              <span className="cl-block-label">核算价格</span>
              <span className="cl-dim">现行价优先，缺价回退基准价</span>
            </div>
            <div className={`cl-price-main cl-resource-price-main${price <= 0 ? " is-zero" : ""}`}>
              <span className="cl-price-label">核算取用价</span>
              <div className="cl-resource-price-number">
                <span className="cl-price-value">{fmtMoney(price > 0 ? price : null)}</span>
                {resource.unit && <span className="cl-price-unit">/ {resource.unit}</span>}
              </div>
            </div>
            <dl className="cl-price-facts cl-resource-price-pair">
              <div><dt>现行价</dt><dd>{fmtMoney(resource.currentPrice > 0 ? resource.currentPrice : null)}</dd></div>
              <div><dt>基准价</dt><dd>{fmtMoney(resource.basePrice > 0 ? resource.basePrice : null)}</dd></div>
            </dl>
            {price <= 0 && <p className="cl-warn-text">现行价与基准价均为 0——引用它的定额子目按 0 计入综合单价。点「编辑」直接补价或按期调价。</p>}
            {price > 0 && resource.currentPrice <= 0 && <p className="cl-note">现行价未录入，核算回退基准价。</p>}
            {facts.length > 0 && (
              <section className="cl-resource-price-details">
                <h3>基本资料</h3>
                <dl className="cl-facts">
                  {facts.map(([k, v]) => <div key={k}><dt>{k}</dt><dd title={v}>{v}</dd></div>)}
                </dl>
              </section>
            )}
            {missing.length > 0 && (
              <p className="cl-note" data-testid="cost-resource-pending">
                待整理：{missing.join("、")}尚未标注——点「编辑」直接归类，或去造价模式用 cost_library classify。
              </p>
            )}
            {resource.note && <p className="cl-note">{resource.note}</p>}
          </section>
          <DetailState phase={phase} error={error}>
            <section className="cl-block">
              <div className="cl-block-head"><span className="cl-block-label">被这些定额引用</span><span className="cl-dim">{usage.length} 条含量行</span></div>
              {usage.length === 0 ? (
                <p className="cl-note">还没有定额含量行引用它。</p>
              ) : (
                <ul className="cl-usage" data-testid="cost-resource-usage">
                  {usage.map((u) => (
                    <li key={`${u.code}:${u.id}`}>
                      <button type="button" className="cl-usage-item" title={`下钻到定额 ${u.code}`} onClick={() => onOpenQuota(u.code)}>
                        <span className="cl-usage-title">{u.title}</span>
                        <span className="cl-meta"><span className="cl-code">{u.code}</span><span>含量 {fmtQty(u.quantity)}</span>{u.unit && <span>/ {u.unit}</span>}</span>
                      </button>
                    </li>
                  ))}
                </ul>
              )}
            </section>
            <section className="cl-block">
              <div className="cl-block-head"><span className="cl-block-label">各期价格</span><span className="cl-dim">{history.length} 条</span></div>
              {history.length === 0 ? (
                <p className="cl-note">还没有价格记录——现行价来自初始录入。</p>
              ) : (
                <table className="cl-table cl-table-inner cl-price-table" data-testid="cost-price-history">
                  <thead>
                    <tr><th>期</th><th>地区</th><th className="cl-num">价格</th><th>口径</th><th>来源</th></tr>
                  </thead>
                  <tbody>
                    {priced.map((h) => (
                      <tr key={h.id}>
                        <td>{h.period ? fmtPeriod(h.period) : <span className="cl-dim">未记期</span>}</td>
                        <td>{h.region || <span className="cl-dim">未记地区</span>}</td>
                        <td className="cl-num cl-strong">{fmtMoney(h.price)}</td>
                        <td>{h.priceType || "—"}</td>
                        <td title={`${h.source || ""}${h.fetchedAt ? ` · ${h.fetchedAt.slice(0, 10)}` : ""}${h.note ? ` · ${h.note}` : ""}`}>
                          {h.source || (h.fetchedAt ? h.fetchedAt.slice(0, 10) : "—")}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </section>
          </DetailState>
          <p className="cl-foot-note">核算取用价优先采用现行价；现行价未录入时回退基准价。各期价格用于追溯调价来源，不改变定额含量。</p>
        </>
      )}
    </article>
  );
}
