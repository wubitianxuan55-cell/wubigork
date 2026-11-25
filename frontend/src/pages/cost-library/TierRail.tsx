// TierRail — 左栏：三级切换（清单 / 定额 / 工料机）+ 当前级的两级上下文。
// 清单级：专业 → 项目；定额级：专业 → 章节；工料机级：类别 → 分类。
// 纯展示，状态在页面。规范值只认库里的取值；空值归「未标注」一桶。
import type { ReactNode } from "react";
import { Coins, ListTree, Layers, Trash2 } from "../../gaea/icons";
import type { WorkcostBillProject } from "../../gaea/lib/types";
import {
  NONE, OUTSIDE, dimLabel, kindLabel, type BillFilters, type CategoryNode, type Facet, type QuotaFilters, type ResourceFilters, type Tier, TIERS,
} from "./logic";

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

/** 左栏的一段上下文：小标题 + 列表（或列表 + 树）。 */
function RailSection({ label, listId, hint, testid, children }: {
  label: string;
  listId: string;
  hint?: string;
  testid?: string;
  children: ReactNode;
}) {
  return (
    <section className="cl-rail-section" data-testid={testid}>
      <p className="cl-rail-cap" id={listId} title={hint}>{label}</p>
      {children}
    </section>
  );
}

function RailItem({ active, onClick, title, children }: {
  active: boolean;
  onClick: () => void;
  title: string;
  children: ReactNode;
}) {
  return (
    <button
      type="button"
      className={`cl-rail-item${active ? " is-active" : ""}`}
      aria-current={active ? "true" : undefined}
      title={title}
      onClick={onClick}
    >
      {children}
    </button>
  );
}

function RailCount({ n }: { n: number }) {
  return <span className="cl-rail-item-count">{n}</span>;
}

/** 一列维度取值：「全部」+ 各取值；NONE 显示成 emptyLabel（未标注）。 */
function FacetList({ listId, allLabel, allCount, facets, value, onPick, emptyLabel, testid }: {
  listId: string;
  allLabel: string;
  allCount: number;
  facets: Facet[];
  value: string;
  onPick: (name: string) => void;
  emptyLabel: string;
  testid?: string;
}) {
  return (
    <ul className="cl-rail-list" aria-labelledby={listId} data-testid={testid}>
      <li>
        <RailItem active={!value} onClick={() => onPick("")} title={allLabel}>
          <span className="cl-rail-item-title">{allLabel}</span>
          <RailCount n={allCount} />
        </RailItem>
      </li>
      {facets.map((f) => (
        <li key={f.name}>
          <RailItem active={value === f.name} onClick={() => onPick(f.name)} title={dimLabel(f.name, emptyLabel)}>
            <span className={`cl-rail-item-title${f.name === NONE ? " cl-dim" : ""}`}>{dimLabel(f.name, emptyLabel)}</span>
            <RailCount n={f.count} />
          </RailItem>
        </li>
      ))}
    </ul>
  );
}

/** 分类树：每一级都是筛选值（完整路径），全部展开——层级不超过三四级，折叠只会多一次点击。 */
function CategoryList({ nodes, value, onPick }: { nodes: CategoryNode[]; value: string; onPick: (path: string) => void }) {
  return (
    <ul className="cl-rail-list cl-rail-sub">
      {nodes.map((n) => (
        <li key={n.path}>
          <RailItem active={value === n.path} onClick={() => onPick(n.path)} title={n.path}>
            <span className="cl-rail-item-title">{n.label}</span>
            <RailCount n={n.count} />
          </RailItem>
          {n.children.length > 0 && <CategoryList nodes={n.children} value={value} onPick={onPick} />}
        </li>
      ))}
    </ul>
  );
}

export function TierRail(props: {
  tier: Tier;
  counts: TierCounts;
  onTier: (tier: Tier) => void;
  // 清单级：专业 → 项目。
  /** 当前专业下有清单行的项目（专业变了项目列表跟着变）。 */
  projects: WorkcostBillProject[];
  billSpecialties: Facet[];
  billFilters: BillFilters;
  onBillSpecialty: (specialty: string) => void;
  onProject: (id: number) => void;
  onDeleteProject: (project: WorkcostBillProject) => void;
  // 定额级：专业 → 章节。
  specialties: Facet[];
  /** 当前专业下的章节分布；没选专业时为空（章节只在专业内有意义）。 */
  chapters: Facet[];
  quotaFilters: QuotaFilters;
  onSpecialty: (specialty: string) => void;
  onChapter: (chapter: string) => void;
  // 工料机级：类别 → 分类。
  kinds: { kind: string; count: number }[];
  /** 当前类别下的分类树与未细分条数。 */
  categories: { tree: CategoryNode[]; uncategorized: number };
  resourceFilters: ResourceFilters;
  onKind: (kind: string) => void;
  onCategory: (category: string) => void;
}) {
  const { tier, counts, projects, billFilters, specialties, chapters, quotaFilters, kinds, categories, resourceFilters } = props;
  return (
    <nav className="cl-rail" aria-label="造价库导航">
      <TierSwitch tier={tier} counts={counts} onTier={props.onTier} />
      <div className="cl-rail-context">
        {tier === "bill" && (
          <>
            <RailSection label="专业" listId="cl-cap-bill-specialty" hint="清单按专业分；专业是库里的规范值">
              <FacetList listId="cl-cap-bill-specialty" allLabel="全部专业" allCount={counts.bill}
                facets={props.billSpecialties} value={billFilters.specialty} onPick={props.onBillSpecialty}
                emptyLabel="未标注专业" testid="cost-rail-bill-specialties" />
            </RailSection>
            <RailSection label="项目" listId="cl-cap-bill-project">
              <ul className="cl-rail-list" aria-labelledby="cl-cap-bill-project" data-testid="cost-rail-projects">
                <li>
                  <RailItem active={billFilters.projectId === 0} onClick={() => props.onProject(0)} title="全部项目">
                    <span className="cl-rail-item-title">全部项目</span>
                    <RailCount n={counts.bill} />
                  </RailItem>
                </li>
                {projects.map((p) => (
                  <li key={p.id} className="cl-rail-row">
                    <RailItem active={billFilters.projectId === p.id} onClick={() => props.onProject(p.id)} title={p.name}>
                      <span className="cl-rail-item-title">{p.name}</span>
                      <RailCount n={p.itemCount} />
                    </RailItem>
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
            </RailSection>
          </>
        )}
        {tier === "quota" && (
          <>
            <RailSection label="专业" listId="cl-cap-quota-specialty" hint="定额专业固定：一条定额只属于一个专业">
              <FacetList listId="cl-cap-quota-specialty" allLabel="全部专业" allCount={counts.quota}
                facets={specialties} value={quotaFilters.specialty} onPick={props.onSpecialty}
                emptyLabel="未标注专业" testid="cost-rail-quota-specialties" />
            </RailSection>
            {quotaFilters.specialty && (
              <RailSection label="章节" listId="cl-cap-quota-chapter" hint="章节在专业内划分">
                <FacetList listId="cl-cap-quota-chapter" allLabel="全部章节" allCount={chapters.reduce((n, c) => n + c.count, 0)}
                  facets={chapters} value={quotaFilters.chapter} onPick={props.onChapter}
                  emptyLabel="未归章节" testid="cost-rail-quota-chapters" />
              </RailSection>
            )}
          </>
        )}
        {tier === "resource" && (
          <>
            <RailSection label="类别" listId="cl-cap-resource-kind">
              <ul className="cl-rail-list" aria-labelledby="cl-cap-resource-kind" data-testid="cost-rail-kinds">
                <li>
                  <RailItem active={!resourceFilters.kind} onClick={() => props.onKind("")} title="全部类别">
                    <span className="cl-rail-item-title">全部类别</span>
                    <RailCount n={counts.resource} />
                  </RailItem>
                </li>
                {kinds.map((k) => (
                  <li key={k.kind}>
                    <RailItem active={resourceFilters.kind === k.kind} onClick={() => props.onKind(k.kind)}
                      title={k.kind === OUTSIDE ? "类别不是人工 / 材料 / 机械 / 外委的工料机（多为把说明行等错记成了工料机）——待整理" : k.kind}>
                      <span className={`cl-rail-item-title${k.kind === OUTSIDE ? " cl-dim" : ""}`}>{kindLabel(k.kind)}</span>
                      <RailCount n={k.count} />
                    </RailItem>
                  </li>
                ))}
              </ul>
            </RailSection>
            {/* 「类别待整理」这一桶下面没有分类一级（这些行的类别本身就要改）。 */}
            {resourceFilters.kind && resourceFilters.kind !== OUTSIDE && (
              <RailSection label="分类" listId="cl-cap-resource-category" hint="按库里的分类路径细分；选上级连带下级" testid="cost-rail-categories">
                <ul className="cl-rail-list" aria-labelledby="cl-cap-resource-category">
                  <li>
                    <RailItem active={!resourceFilters.category} onClick={() => props.onCategory("")} title="全部分类">
                      <span className="cl-rail-item-title">全部分类</span>
                      <RailCount n={(kinds.find((k) => k.kind === resourceFilters.kind)?.count ?? 0)} />
                    </RailItem>
                  </li>
                  {categories.uncategorized > 0 && (
                    <li>
                      <RailItem active={resourceFilters.category === NONE} onClick={() => props.onCategory(NONE)} title="还没细分的工料机">
                        <span className="cl-rail-item-title cl-dim">未细分</span>
                        <RailCount n={categories.uncategorized} />
                      </RailItem>
                    </li>
                  )}
                </ul>
                <CategoryList nodes={categories.tree} value={resourceFilters.category} onPick={props.onCategory} />
              </RailSection>
            )}
          </>
        )}
      </div>
    </nav>
  );
}

/** 窄窗：三级切换横排 + 两级上下文各一个下拉（放不下左栏时用）。 */
export function TierBar(props: {
  tier: Tier;
  counts: TierCounts;
  onTier: (tier: Tier) => void;
  contextOptions: { value: string; label: string }[];
  contextValue: string;
  onContext: (value: string) => void;
  contextLabel: string;
  /** 第二级上下文（清单的项目 / 定额的章节 / 工料机的分类）；没有可选项时不出现。 */
  secondOptions?: { value: string; label: string }[];
  secondValue?: string;
  onSecond?: (value: string) => void;
  secondLabel?: string;
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
      {props.secondOptions && props.secondOptions.length > 0 && props.onSecond && (
        <select
          className="cl-select"
          aria-label={props.secondLabel}
          value={props.secondValue ?? ""}
          onChange={(e) => props.onSecond?.(e.target.value)}
        >
          {props.secondOptions.map((o) => (
            <option key={o.value} value={o.value}>{o.label}</option>
          ))}
        </select>
      )}
    </div>
  );
}
