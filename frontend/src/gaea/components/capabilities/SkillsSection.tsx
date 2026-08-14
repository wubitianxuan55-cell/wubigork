// CapabilitiesPanel 拆分产物：技能列表区（行为零变化，T6-10.1）
import { useMemo, useState } from "react";
import { useT } from "../../lib/i18n";
import { summarizeSkillDescription } from "../../lib/capabilities";
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

function SkillRow({
  skill,
  count,
  expanded,
  onToggle,
}: {
  skill: SkillView
  count: number
  expanded: boolean
  onToggle: () => void
}) {
  const t = useT()
  const summary = summarizeSkillDescription(skill.description)
  const canExpand = summary !== skill.description
  return (
    <button
      className={`w-full text-left border border-border-soft rounded-lg p-3 bg-transparent cursor-pointer transition-[border-color,background] duration-[var(--dur-fast)] hover:border-accent/30 hover:bg-bg-soft active:bg-bg-elev ${
        expanded ? "border-accent/30 bg-bg-elev" : ""
      }`}
      type="button"
      onClick={onToggle}
      aria-expanded={expanded}
      title={skill.description}
    >
      <div className="flex items-center gap-2.5 mb-1">
        <span className="w-8 h-8 flex items-center justify-center rounded-md bg-accent-soft text-accent font-mono text-base font-bold shrink-0">/</span>
        <span className="flex-1 min-w-0 flex flex-col gap-0.5">
          <span className="text-fg text-[13px] font-semibold font-mono">{skill.name}</span>
          <span className="flex items-center gap-1">
            <span className={`badge ${
              skill.scope === "project" ? "badge--success" : "badge--muted"
            }`}>{skillScopeLabel(skill.scope, t)}</span>
            {skill.runAs === "subagent" && <span className="badge badge--accent">{t("caps.subagent")}</span>}
          </span>
        </span>
        {count > 0 && (
          <span className="shrink-0 font-mono text-[11px] font-semibold text-accent">{count}</span>
        )}
      </div>
      <div className={`text-fg-dim text-[12px] leading-snug ${expanded ? "" : "line-clamp-2"}`}>
        {expanded ? skill.description : summary}
      </div>
      {canExpand && <div className="mt-1 text-fg-faint text-[11px]">{expanded ? t("common.collapse") : t("common.expand")}</div>}
    </button>
  )
}

export interface SkillsSectionProps {
  skills: SkillView[]
  counts: Record<string, number>
  expanded: Set<string>
  onToggle: (name: string) => void
}

export function SkillsSection({ skills, counts, expanded, onToggle }: SkillsSectionProps) {
  const t = useT()
  const [query, setQuery] = useState("")
  const filteredSkills = useMemo(() => {
    const q = query.trim().toLowerCase()
    if (!q) return skills
    return skills.filter((sk) => {
      const text = [sk.name, `/${sk.name}`, sk.description, sk.scope, sk.runAs].join(" ").toLowerCase()
      return text.includes(q)
    })
  }, [skills, query])

  return (
    <section className="mb-3">
      <div className="mb-2">
        <input
          className="w-full bg-bg-soft border border-border-soft rounded-md text-fg text-[13px] px-2.5 py-1.5 outline-none placeholder:text-fg-faint focus:border-accent"
          type="search"
          placeholder={t("caps.searchSkills")}
          value={query}
          onChange={(e) => setQuery(e.target.value)}
        />
      </div>
      {skills.length === 0 ? (
        <div className="py-4 text-fg-faint text-xs text-center">{t("caps.noSkills")}</div>
      ) : filteredSkills.length === 0 ? (
        <div className="py-4 text-fg-faint text-xs text-center">{t("caps.noSkillMatches")}</div>
      ) : (
        <div className="flex flex-col gap-2">
          {filteredSkills.map((sk) => (
            <SkillRow
              key={sk.name}
              skill={sk}
              count={counts[sk.name] ?? 0}
              expanded={expanded.has(sk.name)}
              onToggle={() => onToggle(sk.name)}
            />
          ))}
        </div>
      )}
    </section>
  )
}
