// RecommendSection.tsx — 模型推荐位三档陈列（阶段十 10.2）
// 数据源：survey §5.3 谱系表（静态，标注口径诚实——口碑含负面原文）；
// 量化建议联动 10.1 档位口径；「本机模型适用面」调 QuantSuitability 绑定。
import { useEffect, useState } from "react";
import { Collapse, Spin } from "antd";
import { REC_TIERS } from "./recommendations";
import { getEngines, getQuantSuitability } from "../../api/engines";

interface EngineLite {
  id: string;
  type: string;
  models?: { id: string }[];
}

/** 本机已装本地模型的适用面速览（调 QuantSuitability 绑定逐个标注）。 */
function LocalModelSuitability() {
  const [rows, setRows] = useState<{ model: string; advice: string }[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let alive = true;
    (async () => {
      try {
        const engines = (await getEngines()) as unknown as EngineLite[];
        if (!alive) return;
        const locals = (engines ?? []).filter((e) => e.models && e.models.length > 0);
        const out: { model: string; advice: string }[] = [];
        for (const e of locals) {
          for (const m of e.models ?? []) {
            try {
              const advice = await getQuantSuitability(m.id);
              if (advice && alive) out.push({ model: m.id, advice });
            } catch {
              /* 单模型失败跳过 */
            }
          }
        }
        if (alive) setRows(out);
      } catch {
        /* 不可用静默（推荐位是陈列面，不报错打扰） */
      } finally {
        if (alive) setLoading(false);
      }
    })();
    return () => {
      alive = false;
    };
  }, []);

  if (loading) return <Spin size="small" />;
  if (rows.length === 0)
    return (
      <div style={{ color: "var(--color-text-secondary)", fontSize: 12 }}>
        本机暂无可标注的量化模型
      </div>
    );
  return (
    <div data-testid="local-suitability" style={{ fontSize: 12 }}>
      <div style={{ marginBottom: 4, fontWeight: 600 }}>本机模型适用面</div>
      {rows.map((r) => (
        <div key={r.model} style={{ padding: "2px 0" }}>
          <b>{r.model}</b>：{r.advice}
        </div>
      ))}
    </div>
  );
}

/** 推荐位三档陈列（静态数据驱动；本机适用面速览挂顶部）。 */
export function RecommendSection() {
  return (
    <div data-testid="recommend-section" style={{ display: "flex", flexDirection: "column", gap: 12, fontSize: 12.5 }}>
      <div>
        <div style={{ fontWeight: 600, marginBottom: 4 }}>模型推荐位（社区实战谱系 · 2026-10）</div>
        <div style={{ color: "var(--color-text-secondary)" }}>
          量化共识：文风对低量化的容忍度远高于知识（知识低量化非线性崩塌）；agent 工具调用建议 ≥Q4（低比特崩溃率高，见量化红线）。
        </div>
      </div>
      <Collapse
        items={REC_TIERS.map((t) => ({
          key: t.tier,
          label: (
            <span>
              {t.label}{" "}
              <span style={{ color: "var(--color-text-secondary)", fontWeight: 400 }}>— {t.blurb}</span>
            </span>
          ),
          children: (
            <div style={{ display: "flex", flexDirection: "column", gap: 10 }}>
              {t.models.map((m) => (
                <div key={m.name} data-testid={`rec-${t.tier}-${m.name.split(" ")[0]}`}>
                  <div>
                    <b>{m.name}</b>{" "}
                    <span style={{ color: "var(--color-text-secondary)" }}>({m.base})</span>
                  </div>
                  <div>口碑：{m.notes}</div>
                  <div>
                    量化建议：<b>{m.quant}</b>
                    {m.multilingual ? ` · ${m.multilingual}` : ""}
                  </div>
                </div>
              ))}
            </div>
          ),
        }))}
      />
      <LocalModelSuitability />
    </div>
  );
}

