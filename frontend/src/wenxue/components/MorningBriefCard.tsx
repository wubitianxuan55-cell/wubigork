// MorningBriefCard — 「今日晨报」（做梦 2.0 主动预取 MVP，纯本地晨报）。
//
// 挂载即拉取 app.MemoryMorningBrief()（Go 侧 GaeaMemoryMorningBrief 返回
// JSON 串）：work 空间记忆 top5 + 常驻规则 + 近 24h dream 沉淀计数。
//
// 降级红线：
//   - 解析失败 / 空 items → 静默隐藏（渲染 null，不弹 toast，不打扰用户）；
//   - 仅当前空间为 work 时由调用方渲染（play 不渲染 = 双空间红线）；
//   - 零副作用：本组件只读，无写库/落审计/LLM 调用。
import { useEffect, useState } from "react";
import { app } from "../lib/bridge";
import { useT } from "../lib/i18n";
import "./morning-brief-card.css";

interface BriefItem {
  name: string;
  description?: string;
}

interface MorningBrief {
  items?: BriefItem[];
  rules?: string[];
  dreamed24h?: number;
}

// 静默隐藏哨兵：解析失败与空 items 共用，统一渲染 null。
const HIDDEN: MorningBrief | null = null;

export default function MorningBriefCard() {
  const t = useT();
  const [brief, setBrief] = useState<MorningBrief | null>(null);

  useEffect(() => {
    let alive = true;
    app
      .MemoryMorningBrief()
      .then((raw) => JSON.parse(raw) as MorningBrief)
      .then((b) => {
        if (alive) setBrief(b);
      })
      // 绑定失败/JSON 非法/字段缺失一律静默隐藏，不弹 toast。
      .catch(() => {
        if (alive) setBrief(HIDDEN);
      });
    return () => {
      alive = false;
    };
  }, []);

  if (!brief || !Array.isArray(brief.items) || brief.items.length === 0) {
    return null;
  }

  return (
    <div className="mbc" aria-label={t("home.morningBrief.title")}>
      <div className="mbc-head">
        <span className="mbc-title">{t("home.morningBrief.title")}</span>
        <span className="mbc-dreamed">{t("home.morningBrief.dreamed", { n: brief.dreamed24h ?? 0 })}</span>
      </div>
      <ul className="mbc-list">
        {brief.items.slice(0, 5).map((it) => (
          <li key={it.name} className="mbc-item" title={it.description}>
            <span className="mbc-item-name">{it.name}</span>
            {it.description ? <span className="mbc-item-desc">{it.description}</span> : null}
          </li>
        ))}
      </ul>
      {brief.rules && brief.rules.length > 0 && (
        <div className="mbc-rules">
          <div className="mbc-rules-title">{t("home.morningBrief.rulesTitle")}</div>
          <ul className="mbc-rules-list">
            {brief.rules.slice(0, 3).map((r, i) => (
              <li key={i} className="mbc-rule">
                {r}
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}
