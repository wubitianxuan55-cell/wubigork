// CostHealthView.tsx — 造价数据健康页（四期 4.1/4.2）
// 五类扫描（零价/缺价子目/悬挂引用/重复防护/编码混用）+ 一键处置：
// 编码混用组=去重入口（跨身份语义合并，作者显式裁决 keep）。
import { useCallback, useEffect, useState } from "react";
import { Alert, Modal, Spin } from "antd";
import { RefreshCw } from "../../icons";
import { app } from "../../lib/bridge";

// ── wire 镜像（Go workcost.HealthReport）─────────────────────
interface HealthFinding {
  id?: number;
  code?: string;
  title?: string;
  kind?: string;
  unit?: string;
  detail?: string;
}
interface DuplicateGroup {
  kind: string;
  title: string;
  detail?: string;
}
interface CodeMismatchGroup {
  title: string;
  items: { id?: number; code?: string; kind?: string }[];
}
interface HealthReport {
  zeroPrice: HealthFinding[];
  quotaNoPrice: HealthFinding[];
  brokenRef: HealthFinding[];
  duplicates: DuplicateGroup[];
  codeMismatch: CodeMismatchGroup[];
  totalFindings: number;
}

const Section = ({
  title, count, tone, children,
}: { title: string; count: number; tone: string; children: React.ReactNode }) => (
  <section style={{ marginBottom: 14 }}>
    <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 6 }}>
      <b style={{ fontSize: 12.5 }}>{title}</b>
      <span
        className="px-1.5 rounded-full text-[10.5px]"
        style={{ background: count > 0 ? tone : "var(--bg-soft, #f5f5f5)", color: count > 0 ? "#fff" : "var(--color-text-secondary)" }}
      >
        {count}
      </span>
    </div>
    {children}
  </section>
);

export function CostHealthView() {
  const [report, setReport] = useState<HealthReport | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  // 去重弹窗状态（编码混用组：选保留条目后合并其余）
  const [mergeGroup, setMergeGroup] = useState<CodeMismatchGroup | null>(null);
  const [keepCode, setKeepCode] = useState("");
  const [mergeBusy, setMergeBusy] = useState(false);
  const [mergeMsg, setMergeMsg] = useState("");

  const scan = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const raw = await app.CostHealthScan();
      setReport(raw as unknown as HealthReport);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void scan();
  }, [scan]);

  const dedupe = async () => {
    if (!mergeGroup) return;
    const keepItem = mergeGroup.items.find((it) => it.code === keepCode);
    const merges = mergeGroup.items.filter((it) => it.code !== keepCode).map((it) => it.id ?? 0);
    if (!keepItem || merges.length === 0) return;
    setMergeBusy(true);
    setMergeMsg("");
    try {
      const raw = await app.CostDedupeResources(keepItem.id ?? 0, merges);
      const r = raw as { kept: string; merged: number; redirectedItems: number; redirectedPrices: number };
      setMergeMsg(`已合并 ${r.merged} 条进「${r.kept}」：含量引用重定向 ${r.redirectedItems} 行、调价历史并入 ${r.redirectedPrices} 条`);
      setMergeGroup(null);
      void scan(); // 重扫刷新
    } catch (e) {
      setMergeMsg(e instanceof Error ? e.message : String(e));
    } finally {
      setMergeBusy(false);
    }
  };

  if (loading) return <div style={{ textAlign: "center", padding: 32 }}><Spin tip="扫描中（只读零写入）…" /></div>;
  if (error) return <Alert type="error" showIcon message={error} action={<a onClick={() => void scan()}>重试</a>} />;
  if (!report) return null;

  const rowsStyle = { maxHeight: 180, overflowY: "auto", fontSize: 12 } as const;

  return (
    <div data-testid="cost-health-view" style={{ fontSize: 12.5 }}>
      <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 10 }}>
        <b>数据健康</b>
        <span style={{ color: "var(--color-text-secondary)" }}>
          共 {report.totalFindings} 项发现（只读扫描，零写入）
        </span>
        <div style={{ flex: 1 }} />
        <button type="button" className="px-2.5 h-7 rounded-lg border border-border text-fg-faint hover:text-fg text-[11.5px] inline-flex items-center gap-1" onClick={() => void scan()}>
          <RefreshCw size={12} /> 重扫
        </button>
      </div>

      <Section title="① 零价资源（现行价与基准价均为 0）" count={report.zeroPrice.length} tone="#d46b08">
        <div style={rowsStyle}>
          {report.zeroPrice.map((f, i) => (
            <div key={i}>「{f.title}」{f.unit ? `（${f.unit}）` : ""} {f.code}</div>
          ))}
        </div>
      </Section>

      <Section title="② 缺价定额子目（含量行单价为 0，综合单价按 0 计入）" count={report.quotaNoPrice.length} tone="#d46b08">
        <div style={rowsStyle}>
          {report.quotaNoPrice.map((f, i) => (
            <div key={i}>定额 {f.code}：「{f.title}」</div>
          ))}
        </div>
      </Section>

      <Section title="③ 悬挂引用（含量行引用的资源编码不在资源表）" count={report.brokenRef.length} tone="#cf1322">
        <div style={rowsStyle}>
          {report.brokenRef.map((f, i) => (
            <div key={i}>资源编码 {f.code}（定额子目「{f.title}」）——补建资源或改含量行引用</div>
          ))}
        </div>
      </Section>

      <Section title="④ 重复资源防护（身份唯一索引在位性）" count={report.duplicates.length} tone="#cf1322">
        {report.duplicates.length === 0 ? (
          <div style={{ color: "var(--color-text-secondary)" }}>身份唯一索引在位——同身份多行已被库层防护消灭</div>
        ) : (
          report.duplicates.map((g, i) => <Alert key={i} type="error" showIcon message={g.title} description={g.detail} />)
        )}
      </Section>

      <Section title="⑤ 编码未统一（同名资源编码风格混用：助记码 vs 前缀序号）" count={report.codeMismatch.length} tone="#1677ff">
        {report.codeMismatch.length === 0 ? (
          <div style={{ color: "var(--color-text-secondary)" }}>无混用</div>
        ) : (
          <div style={rowsStyle}>
            {report.codeMismatch.map((g, i) => (
              <div key={i} style={{ padding: "4px 0", borderBottom: "1px solid var(--border-subtle, #eee)" }}>
                「{g.title}」：
                {g.items.map((it) => it.code).join(" / ")}
                <button
                  type="button"
                  className="ml-2 px-2 h-5 rounded-md border border-border text-fg-faint hover:text-fg text-[11px]"
                  data-testid={`dedupe-open-${i}`}
                  onClick={() => { setMergeGroup(g); setKeepCode(g.items[0]?.code ?? ""); }}
                >
                  去重合并
                </button>
              </div>
            ))}
          </div>
        )}
      </Section>

      <Modal
        title={`去重合并：${mergeGroup?.title ?? ""}`}
        open={!!mergeGroup}
        onCancel={() => { setMergeGroup(null); setMergeMsg(""); }}
        footer={null}
        width={520}
        destroyOnHidden
      >
        {mergeMsg && <Alert type={mergeMsg.startsWith("已合并") ? "success" : "error"} showIcon message={mergeMsg} style={{ marginBottom: 10 }} />}
        <div style={{ fontSize: 12.5 }}>
          选择保留条目（其余合并进它：含量引用重定向、调价历史并入、merged 条目删除）：
          {mergeGroup?.items.map((it) => (
            <label key={it.code} style={{ display: "block", padding: "5px 0", cursor: "pointer" }}>
              <input type="radio" name="keep" checked={keepCode === it.code} onChange={() => setKeepCode(it.code ?? "")} />
              <b style={{ marginLeft: 6 }}>{it.code}</b>（{it.kind}）
            </label>
          ))}
          <div style={{ marginTop: 12, textAlign: "right" }}>
            <button
              type="button"
              className="px-3 h-7 rounded-lg bg-accent text-accent-fg text-[12px] disabled:opacity-50"
              data-testid="dedupe-confirm"
              disabled={mergeBusy}
              onClick={dedupe}
            >
              确认合并
            </button>
          </div>
        </div>
      </Modal>
    </div>
  );
}

