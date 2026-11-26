// CapabilitiesPanel 拆分产物：技能列表区（行为零变化，T6-10.1）
// v3「星枢」面板语言：技能卡实底收敛（/ 图标 + 名称 + 描述 + 范围徽标），
// 展开态 = 主色容器强调，hover 柔光。
// 图文板块：办公域技能按 disabledDomains 兜底过滤（引擎侧子集在刀2）。
import { useMemo, useState } from "react";
import { ChevronDown, Search, Wand2, X } from "../../icons";
import { useT } from "../../lib/i18n";
import { summarizeSkillDescription } from "../../lib/capabilities";
import { isDisabledWenxueSkill } from "../../lib/disabledDomains";
import type { SkillView } from "../../lib/types";

function skillScopeLabel(scope: string, t: ReturnType<typeof useT>): string {
  switch (scope) {
    case "builtin":
      return t("caps.skillScopeBuiltin")
    case "project":
      return t("caps.skillScopeProject")
    case "custom":
      return t("caps.skillScopeCustom")
    case "global":
      return t("caps.skillScopeGlobal")
    default:
      return scope
  }
}

function ScopeBadge({ scope, text, tone }: { scope: string; text: string; tone?: "project" | "accent" }) {
  const resolvedTone = tone ?? (scope === "project" ? "project" : undefined)
  return (
    <span className="skill-badge" data-tone={resolvedTone}>{text}</span>
  )
}

function SkillRow({
  skill,
  count,
  usage,
  expanded,
  onToggle,
}: {
  skill: SkillView
  count: number
  usage?: { calls: number; ok: number }
  expanded: boolean
  onToggle: () => void
}) {
  const t = useT()
  const summary = summarizeSkillDescription(skill.description)
  const canExpand = summary !== skill.description
  return (
    <button
      className="skill-card"
      type="button"
      onClick={onToggle}
      aria-expanded={expanded}
      title={skill.description}
      data-expanded={expanded}
    >
      <div className="skill-card__top">
        <span className="skill-card__icon" aria-hidden="true"><Wand2 size={13} /></span>
        <span className="skill-card__identity">
          <span className="skill-card__name">{skill.name}</span>
          <span className="skill-card__badges">
            <ScopeBadge scope={skill.scope} text={skillScopeLabel(skill.scope, t)} />
            {skill.runAs === "subagent" && <ScopeBadge scope="project" tone="accent" text={t("caps.subagent")} />}
            {count > 0 && <span className="skill-badge" data-tone="accent">{count} 次调用</span>}
          </span>
        </span>
        {usage && usage.calls > 0 && (
          <span
            data-testid="skill-usage-stat"
            className="skill-card__stats"
          >
            {t("caps.skillUsage", { n: usage.calls })} · {t("caps.skillSuccessRate", { p: Math.round((usage.ok / usage.calls) * 100) })}
          </span>
        )}
      </div>
      <div className="skill-card__desc">
        {expanded ? skill.description : summary}
      </div>
      {canExpand && (
        <div className="skill-card__expand">
          <span>{expanded ? t("common.collapse") : t("common.expand")}</span>
          <ChevronDown size={10} aria-hidden />
        </div>
      )}
    </button>
  )
}

export interface SkillsSectionProps {
  skills: SkillView[]
  counts: Record<string, number>
  // usage 持久化调用计数（7.2-2 判据②，GaeaSkillStats 视图按名索引）；
  // 缺省不渲染统计行（旧调用方/旧测试零变化）。
  usage?: Record<string, { calls: number; ok: number }>
  expanded: Set<string>
  onToggle: (name: string) => void
}

export function SkillsSection({ skills, counts, usage, expanded, onToggle }: SkillsSectionProps) {
  const t = useT()
  const [query, setQuery] = useState("")
  const scopedSkills = useMemo(() => skills.filter((sk) => !isDisabledWenxueSkill(sk.name)), [skills])
  const filteredSkills = useMemo(() => {
    const q = query.trim().toLowerCase()
    if (!q) return scopedSkills
    return scopedSkills.filter((sk) => {
      const text = [sk.name, `/${sk.name}`, sk.description, sk.scope, sk.runAs].join(" ").toLowerCase()
      return text.includes(q)
    })
  }, [scopedSkills, query])

  return (
    <section className="skills-section">
      <div className="skills-head">
        <div className="skills-head__identity">
          <span className="skills-head__icon" aria-hidden="true"><Wand2 size={14} /></span>
          <div>
            <h3>{t("caps.skillsTab")}</h3>
            <p>可运行的 playbook、专用流程与子代理能力</p>
          </div>
        </div>
        <span className="skills-head__count" aria-live="polite">
          {query.trim() ? `${filteredSkills.length}/${scopedSkills.length}` : scopedSkills.length}
        </span>
      </div>

      <div className="skills-search">
        <Search size={13} aria-hidden />
        <input
          className="skills-search__input"
          type="search"
          placeholder={t("caps.searchSkills")}
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          aria-label={t("caps.searchSkills")}
        />
        {query && (
          <button type="button" onClick={() => setQuery("")} aria-label="清除技能搜索">
            <X size={12} aria-hidden />
          </button>
        )}
      </div>

      {scopedSkills.length === 0 ? (
        <div className="skills-empty">
          <span className="skills-empty__icon" aria-hidden="true"><Wand2 size={16} /></span>
          <strong>暂无可用技能</strong>
          <p>{t("caps.noSkills")}</p>
        </div>
      ) : filteredSkills.length === 0 ? (
        <div className="skills-empty">
          <span className="skills-empty__icon" aria-hidden="true"><Search size={16} /></span>
          <strong>没有匹配的技能</strong>
          <p>{t("caps.noSkillMatches")}</p>
          <button type="button" onClick={() => setQuery("")}>清除搜索</button>
        </div>
      ) : (
        <div className="skills-list">
          {filteredSkills.map((sk) => (
            <SkillRow
              key={sk.name}
              skill={sk}
              count={counts[sk.name] ?? 0}
              usage={usage?.[sk.name]}
              expanded={expanded.has(sk.name)}
              onToggle={() => onToggle(sk.name)}
            />
          ))}
        </div>
      )}
    </section>
  )
}
