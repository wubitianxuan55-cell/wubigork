// HealthDrawer — 体检抽屉：只读列出库级健康发现（零价资源 / 缺价定额子目 /
// 悬挂引用 / 重复防护 / 编码未统一 / 不合规矩的数据）。点条目跳到对应行；修复在造价模式做。
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
/** 一条不合规矩的数据：某个字段的取值不在库认可的范围内（字典外 / 类别不在四类内 / 期数不是 YYYY-MM）。 */
interface RuleIssue {
  tier?: string;
  id?: number;
  code?: string;
  title?: string;
  field?: string;
  value?: string;
  fix?: string;
}
interface HealthReport {
  zeroPrice?: HealthFinding[];
  quotaNoPrice?: HealthFinding[];
  brokenRef?: HealthFinding[];
  duplicates?: { kind: string; title: string; detail?: string }[];
  codeMismatch?: { title: string; items: HealthFinding[] }[];
  ruleIssues?: RuleIssue[];
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
  /** 这一段默认跳到哪一级；条目自带 tier 时以条目为准。没有 tier 的段（悬挂引用、重复防护）不可点。 */
  tier?: HealthJump["tier"];
  findings: () => { code: string; text: string; tier?: HealthJump["tier"] }[];
}

const RULE_FIELD: Record<string, string> = {
  kind: "类别", region: "地区", category_path: "分类", specialty: "专业", price_date: "期数", price_period: "计价期",
};
const RULE_TIER: Record<string, string> = { resource: "工料机", quota: "定额", bill: "清单", project: "项目" };

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
      tier: "resource",
      findings: () => (r.zeroPrice ?? []).map((f) => ({ code: f.code ?? "", text: `「${f.title ?? ""}」${f.unit ? `（${f.unit}）` : ""} ${f.code ?? ""} ${f.detail ?? ""}`.trim() })),
    },
    {
      key: "quotaNoPrice", title: "缺价定额子目（含量行单价为 0）", fix: "先给它引用的资源补价",
      tier: "quota",
      findings: () => (r.quotaNoPrice ?? []).map((f) => ({ code: f.code ?? "", text: `定额 ${f.code ?? ""}「${f.title ?? ""}」` })),
    },
    {
      key: "brokenRef", title: "悬挂引用（含量行引用的资源不在库）", fix: "造价模式 cost_library save_resource 按该编码补建资源",
      findings: () => (r.brokenRef ?? []).map((f) => ({ code: "", text: `资源编码 ${f.code ?? ""}（定额子目「${f.title ?? ""}」）` })),
    },
    {
      key: "duplicates", title: "重复防护（身份唯一索引应在位）", fix: "属库结构问题，如实告知",
      findings: () => (r.duplicates ?? []).map((g) => ({ code: "", text: `${g.title}${g.detail ? ` — ${g.detail}` : ""}` })),
    },
    {
      key: "codeMismatch", title: "编码未统一（同地区同名资源编码风格混用）", fix: "确认保留哪一条后造价模式 cost_library dedupe 合并",
      tier: "resource",
      findings: () => (r.codeMismatch ?? []).flatMap((g) => (g.items ?? []).map((it) => ({
        code: it.code ?? "", text: `「${g.title}」 ${it.code ?? ""}`,
      }))),
    },
    {
      // 库定规矩：取值在字典外、类别不在四类内、期数不是 YYYY-MM 的数据——多为升级前的存量。
      key: "ruleIssues", title: "不合规矩的数据（取值在字典外 / 类别不在四类内 / 期数不是年月）",
      fix: "按条目提示处置：造价模式 cost_library classify 改数据，或 dict add 把值收进字典",
      findings: () => (r.ruleIssues ?? []).map((it) => ({
        code: it.code ?? "",
        text: `${RULE_TIER[it.tier ?? ""] ?? ""}${it.code ? ` ${it.code}` : ""}「${it.title ?? ""}」的${RULE_FIELD[it.field ?? ""] ?? it.field}是「${it.value ?? ""}」`,
        tier: it.tier === "resource" || it.tier === "quota" ? it.tier : undefined,
      })),
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
                          {(it.tier ?? s.tier) && it.code ? (
                            <button type="button" className="cl-usage-item" onClick={() => onJump({ tier: (it.tier ?? s.tier) as HealthJump["tier"], code: it.code })}>
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
