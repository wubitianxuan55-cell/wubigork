// CapabilitiesPanel 拆分产物：技能列表区（行为零变化，T6-10.1）
// v3「星枢」面板语言：技能卡实底收敛（/ 图标 + 名称 + 描述 + 范围徽标），
// 展开态 = 主色容器强调，hover 柔光。
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

// 范围徽标：project = 成功语义色，其余 = 中性；subagent = 主色（语义色 + 文字）
function ScopeBadge({ scope, text }: { scope: string; text: string }) {
  const isProject = scope === "project"
  return (
    <span
      className="inline-flex items-center rounded-full px-1.5 py-px text-[10px] font-medium"
      style={{
        color: isProject ? "var(--md-sys-color-success)" : "var(--md-sys-color-text-secondary)",
        background: isProject
          ? "color-mix(in srgb, var(--md-sys-color-success) 12%, transparent)"
          : "color-mix(in srgb, var(--md-sys-color-text) 8%, transparent)",
        border: `1px solid color-mix(in srgb, ${isProject ? "var(--md-sys-color-success)" : "var(--md-sys-color-text)"} 26%, transparent)`,
      }}
    >
      {text}
    </span>
  )
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
      className={`w-full text-left rounded-[var(--radius-md)] p-3 bg-transparent cursor-pointer transition-all duration-200 ${
        expanded
          ? ""
          : "shadow-[inset_0_1px_0_color-mix(in_srgb,var(--md-sys-color-text)_5%,transparent)] hover:bg-(color:--md-sys-color-surface-container-high) hover:shadow-[var(--v3-glow-soft)]"
      }`}
      type="button"
      onClick={onToggle}
      aria-expanded={expanded}
      title={skill.description}
      style={{
        background: expanded ? "var(--md-sys-color-primary-container)" : "var(--md-sys-color-surface-container)",
        border: expanded
          ? "1px solid color-mix(in srgb, var(--gaea-glow) 32%, transparent)"
          : "1px solid var(--md-sys-color-outline-variant)",
        boxShadow: expanded
          ? "inset 3px 0 0 var(--gaea-glow), inset 0 1px 0 color-mix(in srgb, var(--md-sys-color-text) 5%, transparent), var(--v3-glow-faint)"
          : undefined,
      }}
    >
      <div className="flex items-center gap-2.5 mb-1">
        <span
          className="w-8 h-8 flex items-center justify-center rounded-md font-mono text-base font-bold shrink-0"
          style={{
            background: expanded
              ? "color-mix(in srgb, var(--gaea-glow) 16%, transparent)"
              : "color-mix(in srgb, var(--gaea-glow) 11%, transparent)",
            color: "var(--gaea-glow)",
          }}
        >/</span>
        <span className="flex-1 min-w-0 flex flex-col gap-0.5">
          <span className="text-[13px] font-semibold font-mono truncate" style={{ color: "var(--md-sys-color-text)" }}>{skill.name}</span>
          <span className="flex items-center gap-1 flex-wrap">
            <ScopeBadge scope={skill.scope} text={skillScopeLabel(skill.scope, t)} />
            {skill.runAs === "subagent" && <ScopeBadge scope="project" text={t("caps.subagent")} />}
          </span>
        </span>
        {count > 0 && (
          <span
            className="shrink-0 font-mono text-[11px] font-semibold rounded-full px-1.5 py-px"
            style={{
              color: "var(--gaea-glow)",
              background: "color-mix(in srgb, var(--gaea-glow) 14%, transparent)",
            }}
          >
            {count}
          </span>
        )}
      </div>
      <div
        className={`text-[12px] leading-snug ${expanded ? "" : "line-clamp-2"}`}
        style={{ color: "var(--md-sys-color-text-secondary)" }}
      >
        {expanded ? skill.description : summary}
      </div>
      {canExpand && (
        <div className="mt-1 text-[11px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>
          {expanded ? t("common.collapse") : t("common.expand")}
        </div>
      )}
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
          className="w-full rounded-md px-2.5 py-1.5 outline-none text-[13px] transition-[border-color,box-shadow] duration-200 focus:border-[color:color-mix(in_srgb,var(--gaea-glow)_45%,var(--md-sys-color-outline-variant))] focus:shadow-[0_0_0_2px_color-mix(in_srgb,var(--gaea-glow)_14%,transparent)] placeholder:text-(color:--md-sys-color-text-secondary)"
          type="search"
          placeholder={t("caps.searchSkills")}
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          aria-label={t("caps.searchSkills")}
          style={{
            background: "var(--md-sys-color-surface-container)",
            border: "1px solid var(--md-sys-color-outline-variant)",
            color: "var(--md-sys-color-text)",
          }}
        />
      </div>
      {skills.length === 0 ? (
        <div className="py-4 text-xs text-center" style={{ color: "var(--md-sys-color-text-secondary)" }}>{t("caps.noSkills")}</div>
      ) : filteredSkills.length === 0 ? (
        <div className="py-4 text-xs text-center" style={{ color: "var(--md-sys-color-text-secondary)" }}>{t("caps.noSkillMatches")}</div>
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
