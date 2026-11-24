// Details — 右栏详情：清单项的综合单价分析（按挂接定额分组）、定额的含量行
// 分析表、工料机的价格卡与调价历史。数据现算（点开才取），每一级都给出下一级
// 与上一级的入口：清单 ⇄ 定额 ⇄ 工料机。只读，不提供改库入口。
import { useEffect, useState, type ReactNode } from "react";
import { AlertCircle, Loader2 } from "../../gaea/icons";
import { app } from "../../gaea/lib/bridge";
import type {
  WorkcostCompose, WorkcostQuota, WorkcostQuotaItem, WorkcostResource,
  WorkcostResourcePrice, WorkcostUsageRef,
} from "../../gaea/lib/types";
import { effectivePrice, fmtMoney, fmtQty, fmtRate, linkedQuotas, unitPriceOf, type BillRow } from "./logic";
import { KindBadge } from "./Tables";

type Phase = "idle" | "loading" | "ready" | "failed";

function message(e: unknown): string {
  return e instanceof Error ? e.message : String(e);
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
function AnalysisTable({ compose, items, onOpenResource }: {
  compose: WorkcostCompose;
  items: WorkcostQuotaItem[];
  onOpenResource: (code: string) => void;
}) {
  const lines = compose.lines ?? [];
  const aligned = items.length === lines.length;
  if (lines.length === 0) {
    return <p className="cl-note">该定额没有工料机消耗行——综合单价为 0。补含量行走造价模式的 cost_library save_quota。</p>;
  }
  return (
    <div className="cl-analysis">
      <table className="cl-table cl-table-inner">
        <thead>
          <tr><th>行类</th><th>资源</th><th>单位</th><th className="cl-num">含量</th><th className="cl-num">单价</th><th className="cl-num">金额</th><th className="cl-num">占比</th></tr>
        </thead>
        <tbody>
          {lines.map((l, i) => {
            const code = aligned ? items[i]?.resourceCode : "";
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
                <td className="cl-num">{l.price > 0 ? fmtMoney(l.price) : <span className="cl-warn-text" title="资源无价，该项不计入合计">缺价</span>}</td>
                <td className="cl-num">{fmtMoney(l.amount)}</td>
                <td className="cl-num cl-dim">{l.sharePct > 0 ? `${l.sharePct.toFixed(1)}%` : "—"}</td>
              </tr>
            );
          })}
        </tbody>
        <tfoot>
          <tr className="cl-tfoot-fees">
            <td colSpan={3}>三费</td>
            <td colSpan={4} className="cl-fee-line">
              <span>人工 <b className="is-labor">{fmtMoney(compose.laborFee)}</b></span>
              <span>材料 <b className="is-material">{fmtMoney(compose.materialFee)}</b></span>
              <span>机械 <b className="is-machine">{fmtMoney(compose.machineFee)}</b></span>
              {compose.outsourcedFee > 0 && <span className="cl-dim">其中外委 {fmtMoney(compose.outsourcedFee)}（已并入材料）</span>}
            </td>
          </tr>
          <tr className="cl-tfoot-total">
            <td colSpan={5}>综合单价（人材机，不含管理费 / 利润 / 税金）</td>
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

export function BillDetail({ item, refPrices, onOpenQuota, onOpenResource }: {
  item: BillRow;
  refPrices: Record<string, number>;
  onOpenQuota: (code: string) => void;
  onOpenResource: (code: string) => void;
}) {
  const [phase, setPhase] = useState<Phase>("loading");
  const [blocks, setBlocks] = useState<QuotaBlock[]>([]);

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

  return (
    <article className="cl-detail" data-testid="cost-detail-bill">
      <header className="cl-detail-head">
        <div className="cl-kicker">清单 · {item.projectName}</div>
        <h2 className="cl-detail-title">{item.title}</h2>
        <div className="cl-meta">
          <span className="cl-code">{item.code}</span>
          {item.unit && <span>单位 {item.unit}</span>}
          {item.quantity > 0 && <span>工程量 {fmtQty(item.quantity)}</span>}
          {item.division && <span>分部 {item.division}</span>}
        </div>
        {item.feature && <p className="cl-detail-feature" title={item.quantityExpr ? `${item.feature}｜计算式：${item.quantityExpr}` : undefined}>{item.feature}</p>}
        <dl className="cl-price-row">
          <div><dt>原表单价</dt><dd>{fmtMoney((item.listPrice ?? 0) > 0 ? item.listPrice : null)}</dd></div>
          <div><dt>{item.priceOverride > 0 ? "手填单价" : "现算单价"}</dt><dd className="cl-strong">{fmtMoney(unitPriceOf(item, refPrices))}</dd></div>
        </dl>
      </header>
      <DetailState phase={phase} error={blocks[0]?.error}>
        {blocks.length === 0 ? (
          <p className="cl-note">这条清单项没有挂接定额（手填项）。挂接定额在造价模式里做：cost_project_apply save_items 重发该行并带上 quota_code。</p>
        ) : (
          blocks.map((b) => (
            <section key={b.code} className="cl-block" data-testid="cost-bill-quota-block">
              <div className="cl-block-head">
                <span className="cl-block-label">定额</span>
                <button type="button" className="cl-chip" title={`下钻到定额 ${b.code}`}
                  aria-label={`下钻到定额 ${b.code}`} onClick={() => onOpenQuota(b.code)}>{b.code}</button>
                <span className="cl-block-title">{b.quota?.title ?? ""}</span>
                {b.quota?.origCode && b.quota.origCode !== b.quota.code && (
                  <span className="cl-tag" title="工作簿原始编码">原码 {b.quota.origCode}</span>
                )}
              </div>
              {b.compose ? <AnalysisTable compose={b.compose} items={b.quota?.items ?? []} onOpenResource={onOpenResource} /> : <p className="cl-warn-text">现算失败：{b.error}</p>}
            </section>
          ))
        )}
      </DetailState>
      <p className="cl-foot-note">按当前资源价现算（参考口径）；清单合价 = Σ(定额工程量 × 定额单价)，取费在项目合计层。</p>
    </article>
  );
}

// ── 定额 ─────────────────────────────────────────────────────────────

export function QuotaDetail({ quota, usage, onOpenBill, onOpenResource }: {
  quota: WorkcostQuota;
  /** 引用它的清单行（页面按全量清单算好的引用索引）。 */
  usage: BillRow[];
  onOpenBill: (row: BillRow) => void;
  onOpenResource: (code: string) => void;
}) {
  const [phase, setPhase] = useState<Phase>("loading");
  const [full, setFull] = useState<WorkcostQuota | null>(null);
  const [compose, setCompose] = useState<WorkcostCompose | null>(null);
  const [error, setError] = useState("");

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

  return (
    <article className="cl-detail" data-testid="cost-detail-quota">
      <header className="cl-detail-head">
        <div className="cl-kicker">定额{quota.specialty ? ` · ${quota.specialty}` : ""}</div>
        <h2 className="cl-detail-title">{quota.title}</h2>
        <div className="cl-meta">
          <span className="cl-code">{quota.code}</span>
          {quota.origCode && quota.origCode !== quota.code && <span title="工作簿原始编码">原码 {quota.origCode}</span>}
          {quota.unit && <span>计量单位 {quota.unit}</span>}
          {quota.chapter && <span>章节 {quota.chapter}</span>}
          <span>被 {usage.length} 条清单项引用</span>
        </div>
      </header>
      <DetailState phase={phase} error={error}>
        {full && compose && <AnalysisTable compose={compose} items={full.items ?? []} onOpenResource={onOpenResource} />}
      </DetailState>
      <section className="cl-block">
        <div className="cl-block-head"><span className="cl-block-label">引用它的清单项</span><span className="cl-dim">{usage.length} 条</span></div>
        {usage.length === 0 ? (
          <p className="cl-note">还没有清单项套用这条定额。</p>
        ) : (
          <ul className="cl-usage" data-testid="cost-quota-usage">
            {usage.map((it) => (
              <li key={it.id}>
                <button type="button" className="cl-usage-item" title={`查看清单项「${it.title}」`} onClick={() => onOpenBill(it)}>
                  <span className="cl-usage-title">{it.title}</span>
                  <span className="cl-meta"><span className="cl-code">{it.code}</span><span>{it.projectName}</span>{it.quantity > 0 && <span>量 {fmtQty(it.quantity)}</span>}</span>
                </button>
              </li>
            ))}
          </ul>
        )}
      </section>
      <p className="cl-foot-note">含量行是每单位的消耗标准；资源一调价，引用它的综合单价下次读取即重算。</p>
    </article>
  );
}

// ── 工料机 ───────────────────────────────────────────────────────────

export function ResourceDetail({ resource, onOpenQuota }: { resource: WorkcostResource; onOpenQuota: (code: string) => void }) {
  const [phase, setPhase] = useState<Phase>("loading");
  const [error, setError] = useState("");
  const [history, setHistory] = useState<WorkcostResourcePrice[]>([]);
  const [usage, setUsage] = useState<WorkcostUsageRef[]>([]);

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
    ["地区", resource.region], ["价格期数", resource.priceDate], ["价格口径", resource.priceType],
    ["有效期至", resource.validUntil], ["状态", resource.status === "现行" ? "" : resource.status],
  ].filter(([, v]) => v) as [string, string][];

  return (
    <article className="cl-detail" data-testid="cost-detail-resource">
      <header className="cl-detail-head">
        <div className="cl-kicker">工料机</div>
        <h2 className="cl-detail-title">{resource.title}</h2>
        <div className="cl-meta"><KindBadge kind={resource.kind} /><span className="cl-code">{resource.code}</span></div>
      </header>
      <section className="cl-price-card">
        <div className={`cl-price-main${price <= 0 ? " is-zero" : ""}`}>
          <span className="cl-price-label">核算取用价</span>
          <span className="cl-price-value">{fmtMoney(price > 0 ? price : null)}</span>
          {resource.unit && <span className="cl-price-unit">/ {resource.unit}</span>}
        </div>
        <dl className="cl-price-facts">
          <div><dt>现行价</dt><dd>{fmtMoney(resource.currentPrice > 0 ? resource.currentPrice : null)}</dd></div>
          <div><dt>基准价</dt><dd>{fmtMoney(resource.basePrice > 0 ? resource.basePrice : null)}</dd></div>
        </dl>
        {price <= 0 && <p className="cl-warn-text">现行价与基准价均为 0——引用它的定额子目按 0 计入综合单价。补价走造价模式的 cost_library set_price。</p>}
        {price > 0 && resource.currentPrice <= 0 && <p className="cl-note">现行价未录入，核算回退基准价。</p>}
        {facts.length > 0 && (
          <dl className="cl-facts">
            {facts.map(([k, v]) => <div key={k}><dt>{k}</dt><dd title={v}>{v}</dd></div>)}
          </dl>
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
          <div className="cl-block-head"><span className="cl-block-label">调价历史</span><span className="cl-dim">{history.length} 条</span></div>
          {history.length === 0 ? (
            <p className="cl-note">还没有调价记录——价格来自初始录入。</p>
          ) : (
            <ul className="cl-history" data-testid="cost-price-history">
              {history.map((h) => (
                <li key={h.id}>
                  <span className="cl-history-price">{fmtMoney(h.price)}</span>
                  {h.period && <span>{h.period}</span>}
                  {h.region && <span className="cl-dim">{h.region}</span>}
                  {h.priceType && <span className="cl-tag">{h.priceType}</span>}
                  {h.note && <span className="cl-dim cl-truncate" title={h.note}>{h.note}</span>}
                  <span className="cl-history-time">{h.fetchedAt?.slice(0, 10) ?? ""}</span>
                </li>
              ))}
            </ul>
          )}
        </section>
      </DetailState>
    </article>
  );
}
