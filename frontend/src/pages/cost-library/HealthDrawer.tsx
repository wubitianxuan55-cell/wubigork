// HealthDrawer — 体检抽屉：只读列出库级健康发现（零价资源 / 缺价定额子目 /
// 悬挂引用 / 重复防护 / 编码未统一）。点条目跳到对应行；修复在造价模式做。
import { useEffect, useState } from "react";
import { Drawer } from "antd";
import { AlertCircle, Loader2 } from "../../gaea/icons";
import { app } from "../../gaea/lib/bridge";

interface HealthFinding {
  id?: number;
  code?: string;
  title?: string;
  kind?: string;
  unit?: string;
  detail?: string;
}
interface HealthReport {
  zeroPrice?: HealthFinding[];
  quotaNoPrice?: HealthFinding[];
  brokenRef?: HealthFinding[];
  duplicates?: { kind: string; title: string; detail?: string }[];
  codeMismatch?: { title: string; items: HealthFinding[] }[];
  totalFindings?: number;
}

/** 点一条发现要跳去哪一级的哪一行。 */
export interface HealthJump {
  tier: "quota" | "resource";
  code: string;
}

type Phase = "loading" | "ready" | "failed";

interface Section {
  key: keyof HealthReport;
  title: string;
  fix: string;
  tier?: HealthJump["tier"];
  /** 这一段的条目能不能点（悬挂引用指向的资源不在库，跳不过去）。 */
  clickable: boolean;
  findings: () => { code: string; text: string }[];
}

export function HealthDrawer({ open, onClose, onJump }: {
  open: boolean;
  onClose: () => void;
  onJump: (jump: HealthJump) => void;
}) {
  const [phase, setPhase] = useState<Phase>("loading");
  const [error, setError] = useState("");
  const [report, setReport] = useState<HealthReport | null>(null);

  useEffect(() => {
    if (!open) return;
    let alive = true;
    setPhase("loading");
    setError("");
    // CostHealthScan 的声明类型是通用记录；结构以 Go workcost.HealthReport 为准。
    app.CostHealthScan()
      .then((raw) => {
        if (!alive) return;
        setReport((raw ?? {}) as unknown as HealthReport);
        setPhase("ready");
      })
      .catch((e: unknown) => {
        if (!alive) return;
        setError(e instanceof Error ? e.message : String(e));
        setPhase("failed");
      });
    return () => { alive = false; };
  }, [open]);

  const r = report ?? {};
  const sections: Section[] = [
    {
      key: "zeroPrice", title: "零价资源（现行价与基准价均为 0）", fix: "造价模式 cost_library set_price 补价",
      tier: "resource", clickable: true,
      findings: () => (r.zeroPrice ?? []).map((f) => ({ code: f.code ?? "", text: `「${f.title ?? ""}」${f.unit ? `（${f.unit}）` : ""} ${f.code ?? ""} ${f.detail ?? ""}`.trim() })),
    },
    {
      key: "quotaNoPrice", title: "缺价定额子目（含量行单价为 0）", fix: "先给它引用的资源补价",
      tier: "quota", clickable: true,
      findings: () => (r.quotaNoPrice ?? []).map((f) => ({ code: f.code ?? "", text: `定额 ${f.code ?? ""}「${f.title ?? ""}」` })),
    },
    {
      key: "brokenRef", title: "悬挂引用（含量行引用的资源不在库）", fix: "造价模式 cost_library save_resource 按该编码补建资源",
      clickable: false,
      findings: () => (r.brokenRef ?? []).map((f) => ({ code: "", text: `资源编码 ${f.code ?? ""}（定额子目「${f.title ?? ""}」）` })),
    },
    {
      key: "duplicates", title: "重复防护（身份唯一索引应在位）", fix: "属库结构问题，如实告知",
      clickable: false,
      findings: () => (r.duplicates ?? []).map((g) => ({ code: "", text: `${g.title}${g.detail ? ` — ${g.detail}` : ""}` })),
    },
    {
      key: "codeMismatch", title: "编码未统一（同名资源编码风格混用）", fix: "确认保留哪一条后造价模式 cost_library dedupe 合并",
      tier: "resource", clickable: true,
      findings: () => (r.codeMismatch ?? []).flatMap((g) => (g.items ?? []).map((it) => ({
        code: it.code ?? "", text: `「${g.title}」 ${it.code ?? ""}`,
      }))),
    },
  ];

  return (
    <Drawer
      title="体检 · 造价库健康扫描"
      open={open}
      onClose={onClose}
      width={440}
      destroyOnHidden
      data-testid="cost-health-drawer"
    >
      {phase === "loading" && <div className="cl-state" role="status"><Loader2 size={16} className="cl-spin" aria-hidden /><span className="cl-state-title">正在扫描（只读，零写入）…</span></div>}
      {phase === "failed" && (
        <div className="cl-state is-failed" role="alert"><AlertCircle size={16} aria-hidden /><span className="cl-state-title">扫描失败</span><span>{error}</span></div>
      )}
      {phase === "ready" && (
        <div className="cl-health" data-testid="cost-health-report">
          <p className="cl-health-total">共 <strong>{r.totalFindings ?? 0}</strong> 项发现{((r.totalFindings ?? 0) === 0) && "，库级扫描全绿"}。</p>
          {sections.map((s) => {
            const items = s.findings();
            return (
              <section key={s.key} className="cl-health-section" data-testid={`cost-health-${s.key}`}>
                <div className="cl-block-head">
                  <span className="cl-block-label">{s.title}</span>
                  <span className="cl-tag">{items.length}</span>
                </div>
                {items.length === 0 ? (
                  <p className="cl-note">无</p>
                ) : (
                  <>
                    <ul className="cl-usage">
                      {items.slice(0, 60).map((it, i) => (
                        <li key={`${it.code}:${i}`}>
                          {s.clickable && s.tier && it.code ? (
                            <button type="button" className="cl-usage-item" onClick={() => onJump({ tier: s.tier as HealthJump["tier"], code: it.code })}>
                              <span className="cl-usage-title">{it.text}</span>
                            </button>
                          ) : (
                            <span className="cl-usage-static">{it.text}</span>
                          )}
                        </li>
                      ))}
                    </ul>
                    {items.length > 60 && <p className="cl-note">仅列前 60 条，共 {items.length} 条。</p>}
                    <p className="cl-note">处置：{s.fix}</p>
                  </>
                )}
              </section>
            );
          })}
        </div>
      )}
    </Drawer>
  );
}
