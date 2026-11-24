// TierRail — 左栏：三级切换（清单 / 定额 / 工料机）+ 当前级的上下文清单。
// 清单级列项目（可整体删除），定额级列专业，工料机级列类别。纯展示，状态在页面。
import type { ReactNode } from "react";
import { Coins, ListTree, Layers, Trash2 } from "../../gaea/icons";
import type { WorkcostBillProject } from "../../gaea/lib/types";
import { TIERS, type BillFilters, type QuotaFilters, type ResourceFilters, type Tier } from "./logic";

export interface TierCounts {
  bill: number;
  quota: number;
  resource: number;
}

interface BaseProps {
  tier: Tier;
  counts: TierCounts;
  onTier: (tier: Tier) => void;
}

const TIER_ICONS: Record<Tier, ReactNode> = {
  bill: <ListTree size={14} />,
  quota: <Coins size={14} />,
  resource: <Layers size={14} />,
};

export function TierSwitch({ tier, counts, onTier }: BaseProps) {
  return (
    <div className="cl-tier-switch" role="tablist" aria-label="造价数据三级">
      {TIERS.map((t) => (
        <button
          key={t.key}
          type="button"
          role="tab"
          aria-selected={tier === t.key}
          title={t.hint}
          data-testid={`cost-tier-${t.key}`}
          className={`cl-tier${tier === t.key ? " is-active" : ""}`}
          onClick={() => onTier(t.key)}
        >
          <span className="cl-tier-icon" aria-hidden>{TIER_ICONS[t.key]}</span>
          <span className="cl-tier-label">{t.label}</span>
          <span className="cl-tier-count">{counts[t.key]}</span>
        </button>
      ))}
    </div>
  );
}

export function TierRail(props: {
  tier: Tier;
  counts: TierCounts;
  onTier: (tier: Tier) => void;
  projects: WorkcostBillProject[];
  billFilters: BillFilters;
  onProject: (id: number) => void;
  onDeleteProject: (project: WorkcostBillProject) => void;
  specialties: string[];
  quotaFilters: QuotaFilters;
  onSpecialty: (specialty: string) => void;
  kinds: { kind: string; count: number }[];
  resourceFilters: ResourceFilters;
  onKind: (kind: string) => void;
}) {
  const { tier, projects, billFilters, specialties, quotaFilters, kinds, resourceFilters } = props;
  const counts = props.counts;
  return (
    <nav className="cl-rail" aria-label="造价库导航">
      <TierSwitch tier={tier} counts={props.counts} onTier={props.onTier} />
      <div className="cl-rail-context">
        {tier === "bill" && (
          <ul className="cl-rail-list" aria-label="项目">
            <li>
              <button
                type="button"
                className={`cl-rail-item${billFilters.projectId === 0 ? " is-active" : ""}`}
                aria-current={billFilters.projectId === 0 ? "true" : undefined}
                onClick={() => props.onProject(0)}
              >
                <span className="cl-rail-item-title">全部项目</span>
                <span className="cl-rail-item-count">{counts.bill}</span>
              </button>
            </li>
            {projects.map((p) => (
              <li key={p.id} className="cl-rail-row">
                <button
                  type="button"
                  className={`cl-rail-item${billFilters.projectId === p.id ? " is-active" : ""}`}
                  aria-current={billFilters.projectId === p.id ? "true" : undefined}
                  title={p.name}
                  onClick={() => props.onProject(p.id)}
                >
                  <span className="cl-rail-item-title">{p.name}</span>
                  <span className="cl-rail-item-count">{p.itemCount}</span>
                </button>
                <button
                  type="button"
                  className="cl-icon-btn cl-row-delete"
                  aria-label={`删除项目 ${p.name}`}
                  title="整体删除项目（封面、清单与独占定额；共享定额与工料机保留）"
                  onClick={() => props.onDeleteProject(p)}
                >
                  <Trash2 size={13} />
                </button>
              </li>
            ))}
          </ul>
        )}
        {tier === "quota" && (
          <ul className="cl-rail-list" aria-label="定额专业">
            <li>
              <button
                type="button"
                className={`cl-rail-item${!quotaFilters.specialty ? " is-active" : ""}`}
                onClick={() => props.onSpecialty("")}
              >
                <span className="cl-rail-item-title">全部专业</span>
                <span className="cl-rail-item-count">{counts.quota}</span>
              </button>
            </li>
            {specialties.map((s) => (
              <li key={s}>
                <button
                  type="button"
                  className={`cl-rail-item${quotaFilters.specialty === s ? " is-active" : ""}`}
                  onClick={() => props.onSpecialty(s)}
                >
                  <span className="cl-rail-item-title">{s}</span>
                </button>
              </li>
            ))}
          </ul>
        )}
        {tier === "resource" && (
          <ul className="cl-rail-list" aria-label="工料机类别">
            <li>
              <button
                type="button"
                className={`cl-rail-item${!resourceFilters.kind ? " is-active" : ""}`}
                onClick={() => props.onKind("")}
              >
                <span className="cl-rail-item-title">全部类别</span>
                <span className="cl-rail-item-count">{counts.resource}</span>
              </button>
            </li>
            {kinds.map((k) => (
              <li key={k.kind}>
                <button
                  type="button"
                  className={`cl-rail-item${resourceFilters.kind === k.kind ? " is-active" : ""}`}
                  onClick={() => props.onKind(k.kind)}
                >
                  <span className="cl-rail-item-title">{k.kind}</span>
                  <span className="cl-rail-item-count">{k.count}</span>
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>
    </nav>
  );
}

/** 窄窗：三级切换横排 + 上下文改下拉（放不下左栏时用）。 */
export function TierBar(props: {
  tier: Tier;
  counts: TierCounts;
  onTier: (tier: Tier) => void;
  contextOptions: { value: string; label: string }[];
  contextValue: string;
  onContext: (value: string) => void;
  contextLabel: string;
}) {
  return (
    <div className="cl-tier-bar">
      <TierSwitch tier={props.tier} counts={props.counts} onTier={props.onTier} />
      {props.contextOptions.length > 1 && (
        <select
          className="cl-select"
          aria-label={props.contextLabel}
          value={props.contextValue}
          onChange={(e) => props.onContext(e.target.value)}
        >
          {props.contextOptions.map((o) => (
            <option key={o.value} value={o.value}>{o.label}</option>
          ))}
        </select>
      )}
    </div>
  );
}
