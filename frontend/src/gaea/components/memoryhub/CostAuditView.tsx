// CostAuditView.tsx — 跨项目对标查漏（11.3；复核增强而非替代）
// 确定性统计对标引擎：清单缺项（被审项目 vs 同行分部覆盖）+ 同名定额缺组成
// （同名组多数派资源）；结论逐条带溯源引用（哪些项目/定额支撑），宁缺勿误
// 闸（语料 <3 / 覆盖 <60% / 少数派一律不报）。只读零写入。
import { useCallback, useEffect, useState } from "react";
import { Alert, Spin } from "antd";
import { RefreshCw, Search } from "../../icons";
import { app } from "../../lib/bridge";
import type { WorkcostAuditFinding, WorkcostAuditReport, WorkcostBillProject } from "../../lib/types";

export function CostAuditView() {
  const [projects, setProjects] = useState<WorkcostBillProject[]>([]);
  const [projectID, setProjectID] = useState(0);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [report, setReport] = useState<WorkcostAuditReport | null>(null);

  useEffect(() => {
    (async () => {
      try {
        const ps = (await app.WorkcostBillProjects()) ?? [];
        setProjects(ps);
        if (ps.length > 0) setProjectID(ps[0].id);
      } catch {
        // 列表失败：下拉留空，扫描时报错透出。
      }
    })();
  }, []);

  const scan = useCallback(async () => {
    setBusy(true);
    setError("");
    try {
      setReport(await app.CostAuditScan(projectID));
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }, [projectID]);

  const total = report ? report.billGaps.length + report.quotaGaps.length : 0;

  return (
    <div data-testid="cost-audit-view" style={{ fontSize: 12.5, padding: 14 }}>
      <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 10 }}>
        <b>对标查漏</b>
        <span style={{ color: "var(--color-text-secondary)" }}>
          跨项目统计对标 · 结论逐条溯源 · 复核线索而非替代判断（只读零写入）
        </span>
        <div style={{ flex: 1 }} />
        <select
          className="h-7 rounded-lg border border-border bg-transparent px-2 text-[12px] text-fg"
          value={projectID}
          onChange={(e) => setProjectID(Number(e.target.value))}
          aria-label="被审项目"
        >
          {projects.length === 0 && <option value={0}>（语料库还没有项目案例）</option>}
          {projects.map((p) => (
            <option key={p.id} value={p.id}>{p.name}</option>
          ))}
        </select>
        <button
          type="button"
          data-testid="audit-run"
          className="px-2.5 h-7 rounded-lg border border-border text-fg hover:text-accent text-[11.5px] inline-flex items-center gap-1 disabled:opacity-50"
          disabled={busy || projectID === 0}
          onClick={() => void scan()}
        >
          <Search size={12} /> 开始对标
        </button>
        {report && (
          <button type="button" className="px-2.5 h-7 rounded-lg border border-border text-fg-faint hover:text-fg text-[11.5px] inline-flex items-center gap-1" onClick={() => void scan()}>
            <RefreshCw size={12} /> 重扫
          </button>
        )}
      </div>

      {busy && <div style={{ textAlign: "center", padding: 32 }}><Spin tip="对标扫描中（只读零写入）…" /></div>}
      {error && <Alert type="error" showIcon message={error} action={<a onClick={() => void scan()}>重试</a>} />}

      {report && (
        <>
          {report.projectHits < 3 ? (
            <Alert
              type="info"
              showIcon
              message={`语料项目 ${report.projectHits} 个（少于 3 个）——按宁缺勿误闸不产出发现`}
              description="导入更多项目案例（造价参考→项目文件）后，对标引擎才有「多数派」可言。"
            />
          ) : total === 0 ? (
            <Alert type="success" showIcon message="零发现：被审项目的分部覆盖与各同名定额组成都跟得上同行多数派" />
          ) : (
            <>
              <FindingSection title="清单缺项（同行多数项目都有、本项目没有的分部）" findings={report.billGaps} />
              <FindingSection title="定额缺组成（同名定额多数成员都含、该定额没有的工料机）" findings={report.quotaGaps} />
            </>
          )}
          <div style={{ marginTop: 10, color: "var(--color-text-secondary)", fontSize: 11.5 }}>
            口径：覆盖率 ≥60% 且至少 2 个同行支撑才产出（宁缺勿误）；每条发现附溯源引用，处置与否由作者复核裁决。
          </div>
        </>
      )}
    </div>
  );
}

function FindingSection({ title, findings }: { title: string; findings: WorkcostAuditFinding[] }) {
  return (
    <section style={{ marginBottom: 14 }}>
      <div style={{ fontWeight: 600, marginBottom: 6 }}>
        {title}
        <span style={{ marginLeft: 6, color: findings.length ? "var(--color-warning)" : "var(--color-text-secondary)" }}>
          {findings.length} 条
        </span>
      </div>
      {findings.length === 0 ? (
        <div style={{ color: "var(--color-text-secondary)" }}>无</div>
      ) : (
        findings.map((f, i) => (
          <div key={i} data-testid={`audit-finding-${f.kind}`} style={{ border: "1px solid var(--border-subtle, #eee)", borderRadius: 8, padding: "6px 10px", marginBottom: 6 }}>
            <div>{f.detail}</div>
            <details style={{ marginTop: 4 }}>
              <summary style={{ cursor: "pointer", color: "var(--color-text-secondary)", fontSize: 11.5 }}>
                溯源依据（{f.evidence.length} 条）
              </summary>
              <div style={{ marginTop: 4, color: "var(--color-text-secondary)", fontSize: 11.5 }}>
                {f.evidence.map((ev, j) => (
                  <div key={j}>
                    {ev.ref}{ev.source ? `（${ev.source}）` : ""}：{ev.support}
                  </div>
                ))}
              </div>
            </details>
          </div>
        ))
      )}
    </section>
  );
}
