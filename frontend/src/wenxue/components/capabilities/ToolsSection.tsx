// CapabilitiesPanel 拆分产物：工具列表区（行为零变化，T6-10.1）
// v3「星枢」面板语言：工具卡实底收敛（图标 + 名称 + 描述 + 状态徽标），
// 激活工具 = 主色容器底 + 左侧光条（inset shadow）+ 柔光。
import { useEffect, useMemo, useRef, useState } from "react";
import { Cpu, ChevronDown, Search, X } from "../../icons";
import { useGSAPCollapse } from "../../lib/useGSAPCollapse";
import { app } from "../../lib/bridge";
import type { ToolSurfaceEntryB, ToolSurfaceViewB } from "../../lib/bridge/wenxue";
import { isDisabledWenxueTool } from "../../lib/disabledDomains";

type Counts = Record<string, number>;

const TOOL_DESC: Record<string, string> = {
  read_file: "读取文件内容(可选行范围/分页)",
  write_file: "写入/覆盖文件(自动建父目录)",
  ls: "列目录条目(子目录带/)",
  bash: "执行shell命令(合并stdout+stderr,支持后台任务)",
  bash_output: "读取后台任务的增量输出(不阻塞)",
  wait: "阻塞等待后台任务结束(可设超时)",
  kill_shell: "终止后台任务(SIGTERM→SIGKILL)",
  web_fetch: "抓取URL纯文本(去标签,SSRF安全)",
  web_search: "搜索公开网页(返回结构化JSON)",
  todo_write: "更新任务清单(全量替换,最多一个进行中)",
  complete_step: "完成计划步骤(附验证证据,空证据拒绝)",
  ask: "向用户提供多选项问题",
  task: "派发子代理执行聚焦子任务",
  remember: "保存持久事实到项目记忆",
  forget: "通过名称删除已保存记忆",
  memory_search: "按关键词搜索已保存记忆",
  memory_get: "按ID读取单条记忆详情",
  promote_session_facts: "把当前会话事实提升为持久记忆",
  read_skill: "读取指定技能的完整内容",
  run_skill: "调用Skills索引中的playbook",
  install_skill: "编写并保存新技能",
  slash_command: "按名称调用项目斜杠命令与技能",
  format_convert: "文档格式转换(docx/xlsx/pptx/pdf→Markdown,含OCR回退)",
  chart_gen: "matplotlib图表生成(bar/line/pie/scatter)",
  docx: "Word 文档技能——创建/读取/编辑 .docx(模板、目录、修订、排版)",
  xlsx: "Excel 表格技能——创建/处理 .xlsx/.csv(公式、格式、清洗、图表)",
  pdf: "PDF 文档技能——读取/合并/拆分/水印/表单/OCR",
  pptx: "演示文稿技能——把内容做成 .pptx 幻灯片",
  ocr: "本地 OCR 模型——提取图片/扫描件中的精确文字",
  image_gen: "生图——文生图/改图（需配置图像后端；产出锚当前小说目录）",
  semantic_search: "本地 bge-m3——跨库语义检索(成本/知识/办公记忆)",
  routine_llm: "通用文本兜底——纯文本摘要/归一化/抽取/改写，本地/免费优先",
  translate_text: "本地翻译——优先 Hunyuan-MT / Hy-MT 翻译模型，未安装时回退办公模型",
  // 图文专属四族（刀3，仅图文引擎装配；与 wenxue-workbench 技能同源）
  wenxue_chapter_list: "章节清单——章号/标题/字数/修改时间（按约定目录）",
  wenxue_chapter_read: "读一章正文+上一章结尾衔接段",
  wenxue_chapter_write: "写/建一章（overwrite 显式闸；回执带字数差与前后章）",
  wenxue_candidate_write: "写 A/B/C 同章候选稿到章节/候选/，不改正文与定稿",
  wenxue_candidate_promote: "候选提升为正文；当前正文先自动归档",
  wenxue_character_list: "角色卡清单——名字/字数/摘要+角色库同名命中",
  wenxue_character_read: "读角色卡全文",
  wenxue_character_write: "建/更新角色卡（已存在即拒，自动补一级标题）",
  wenxue_character_import: "从角色库导入角色为工作空间卡（全字段落卡，库只读不回写）",
  wenxue_timeline_read: "时间线事件全量（章节序，可按起始章过滤）",
  wenxue_timeline_append: "追加时间线事件（章节锚+一句话，缺文件自动建）",
  wenxue_stats: "工作空间统计快照——总字数/章均/角色卡/设定/时间线",
  wenxue_quality_check: "全书确定性体检——章节编号/角色卡缺节/时间线锚/设定空文档/孤儿候选（零 LLM）",
  wenxue_style_report: "文风确定性指标——句长/段长/对话占比/叹问密度/高频句首+基线漂移差值（零 LLM）",
  wenxue_final_list: "定稿章节清单（发布稿域，非草稿）",
  wenxue_final_read: "读一章定稿全文（审校/前情/风格参考）",
  wenxue_final_write: "定稿入库（仅定稿目录；用户明确「定稿」时才用）",
  grep: "工作空间内容搜索（正则/文件过滤）",
  genui_validate: "GenUI 块契约校验（回答即 UI 面板）",
  session_search: "跨会话检索——按关键词搜过往会话正文",
  recall_archived: "召回已归档会话（历史取材）",
}

interface Section {
  title: string
  items: string[]
}

const RAW_SECTIONS: Section[] = [
  { title: "图文专属", items: ["wenxue_chapter_list", "wenxue_chapter_read", "wenxue_chapter_write", "wenxue_candidate_write", "wenxue_candidate_promote", "wenxue_character_list", "wenxue_character_read", "wenxue_character_write", "wenxue_character_import", "wenxue_timeline_read", "wenxue_timeline_append", "wenxue_stats", "wenxue_quality_check", "wenxue_style_report", "wenxue_final_list", "wenxue_final_read", "wenxue_final_write"] },
  { title: "文件与命令", items: ["read_file", "write_file", "ls", "grep", "bash", "bash_output", "wait", "kill_shell"] },
  { title: "网络", items: ["web_search", "web_fetch"] },
  { title: "任务与协作", items: ["todo_write", "complete_step", "ask", "task", "genui_validate"] },
  { title: "记忆", items: ["remember", "forget", "memory_search", "memory_get", "promote_session_facts"] },
  { title: "会话", items: ["session_search", "recall_archived"] },
  { title: "技能", items: ["read_skill", "run_skill", "install_skill", "slash_command"] },
  { title: "文档技能", items: ["docx", "xlsx", "pdf", "pptx"] },
  { title: "本地专业模型", items: ["vision", "ocr", "image_gen", "semantic_search", "translate_text"] },
  { title: "通用办公", items: ["format_convert", "chart_gen", "routine_llm"] },
]

// 图文板块：办公域工具按 disabledDomains 兜底滤除（引擎侧物理卸载在刀1），
// 滤空的分组整体丢弃（如「文档技能」四件全是办公技能名）。
const SECTIONS: Section[] = RAW_SECTIONS.map((sec) => ({
  ...sec,
  items: sec.items.filter((name) => !isDisabledWenxueTool(name)),
})).filter((sec) => sec.items.length > 0)

function ToolCard({ name, count, desc, hidden = false }: { name: string; count: number; desc?: string; hidden?: boolean }) {
  const active = !hidden && count > 0
  return (
    <div
      className={`flex items-start gap-1.5 px-2 py-1.5 rounded-[var(--radius-sm)] border cursor-default transition-all duration-150 ${
        active ? "" : "hover:bg-(color:--md-sys-color-surface-container-high)"
      }`}
      title={desc ?? name}
      style={{
        background: active ? "var(--md-sys-color-primary-container)" : "var(--md-sys-color-surface-container)",
        borderColor: active
          ? "color-mix(in srgb, var(--gaea-glow) 30%, transparent)"
          : "var(--md-sys-color-outline-variant)",
        boxShadow: active
          ? "inset 3px 0 0 var(--gaea-glow), inset 0 1px 0 color-mix(in srgb, var(--md-sys-color-text) 5%, transparent), var(--v3-glow-faint)"
          : "inset 0 1px 0 color-mix(in srgb, var(--md-sys-color-text) 5%, transparent)",
      }}
    >
      {/* 状态点：语义色 + 光晕（激活/未激活三重传达之一） */}
      <span
        className="w-1.5 h-1.5 mt-[5px] rounded-full shrink-0"
        style={{
          background: active ? "var(--gaea-glow)" : "var(--md-sys-color-outline-variant)",
          boxShadow: active ? "0 0 6px var(--gaea-glow)" : "none",
        }}
      />
      <span className="flex-1 min-w-0 flex flex-col gap-0.5 leading-[1.25]">
        <span
          className={`font-mono text-[10.5px] truncate ${active ? "font-semibold" : ""}`}
          style={{ color: active ? "var(--gaea-glow)" : "var(--md-sys-color-text-secondary)" }}
        >
          {name}
        </span>
        {desc && (
          <span className="text-[10px] leading-[1.3] line-clamp-1" style={{ color: "var(--md-sys-color-text-secondary)" }}>
            {desc}
          </span>
        )}
      </span>
      {hidden && (
        <span
          className="shrink-0 text-[9px] rounded-full px-1.5 py-px mt-px"
          style={{ color: "var(--md-sys-color-text-secondary)", background: "color-mix(in srgb, var(--md-sys-color-text) 8%, transparent)" }}
        >
          藏
        </span>
      )}
      {/* 状态徽标：次数 + 语义色容器 */}
      <span
        className="shrink-0 font-mono text-[11px] font-semibold mt-px rounded-full px-1.5 py-px"
        style={{
          color: active ? "var(--gaea-glow)" : "var(--md-sys-color-text-secondary)",
          background: active
            ? "color-mix(in srgb, var(--gaea-glow) 16%, transparent)"
            : "color-mix(in srgb, var(--md-sys-color-text) 8%, transparent)",
        }}
      >
        {count}
      </span>
    </div>
  )
}

function ToolGroup({
  title,
  items,
  counts,
  defaultOpen,
  descs,
  hidden = false,
  groupNote,
}: {
  title: string
  items: string[]
  counts: Counts
  defaultOpen: boolean
  /** 活取描述表（装配面缝）：有则优先生效，缺项回落静态 TOOL_DESC。 */
  descs?: Record<string, string>
  /** 模式藏面组：灰显+藏徽标（图文引擎模式中立，此组常态为空；MCP deferred 藏面可达）。 */
  hidden?: boolean
  groupNote?: string
}) {
  const [open, setOpen] = useState(defaultOpen)
  const ref = useRef<HTMLDivElement>(null)
  useGSAPCollapse(ref, open, { duration: 0.18 })

  const activeCount = items.filter((n) => (counts[n] ?? 0) > 0).length

  return (
    <div className="px-1.5 py-0.5">
      <button
        className="flex items-center gap-1 w-full px-1 py-1.5 bg-transparent border-0 text-left cursor-pointer rounded transition-colors hover:bg-(color:--md-sys-color-surface-container-high)"
        onClick={() => setOpen((v) => !v)}
      >
        <ChevronDown
          size={10}
          aria-hidden
          className={`transition-transform duration-150 ${open ? "rotate-0" : "-rotate-90"}`}
          style={{ color: "var(--md-sys-color-text-secondary)" }}
        />
        <span className="text-[10px] font-semibold uppercase tracking-[0.5px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>{title}</span>
        {activeCount > 0 && (
          <span className="ml-auto text-[9px] font-mono" style={{ color: "var(--gaea-glow)" }}>{activeCount}</span>
        )}
      </button>
      <div ref={ref} style={{ overflow: "hidden" }}>
        <div className="flex flex-col gap-0.5 pt-0.5 pb-1">
          {groupNote && (
            <div className="px-2 pb-1 text-[10px] leading-[1.4]" style={{ color: "var(--md-sys-color-text-secondary)" }}>{groupNote}</div>
          )}
          {items.map((name) => (
            <ToolCard key={name} name={name} count={counts[name] ?? 0} desc={descs?.[name] ?? TOOL_DESC[name]} hidden={hidden} />
          ))}
        </div>
      </div>
    </div>
  )
}

// 装配面来源分组（2026-11-21 面板缝，与 gaea ToolsSection 同构）。
const SOURCE_ORDER: string[] = ["builtin-domain", "extra-app", "build-internal"]
const SOURCE_LABELS: Record<string, string> = {
  "builtin-domain": "内置域工具",
  "extra-app": "应用工具",
  "build-internal": "会话元工具",
}

function groupSurface(tools: ToolSurfaceEntryB[]): { key: string; title: string; items: ToolSurfaceEntryB[]; hidden: boolean; note?: string }[] {
  const bySource = new Map<string, ToolSurfaceEntryB[]>()
  const hiddenList: ToolSurfaceEntryB[] = []
  for (const t of tools) {
    if (t.hidden) { hiddenList.push(t); continue }
    const src = t.source || "other"
    if (!bySource.has(src)) bySource.set(src, [])
    bySource.get(src)!.push(t)
  }
  const groups: { key: string; title: string; items: ToolSurfaceEntryB[]; hidden: boolean; note?: string }[] = []
  for (const src of SOURCE_ORDER) {
    const items = bySource.get(src)
    if (items?.length) {
      groups.push({ key: src, title: SOURCE_LABELS[src] ?? src, items, hidden: false })
      bySource.delete(src)
    }
  }
  for (const [src, items] of bySource) {
    const title = src.startsWith("mcp:") ? `MCP：${src.slice(4)}` : src === "other" ? "其他" : src
    groups.push({ key: src, title, items, hidden: false })
  }
  if (hiddenList.length) {
    groups.push({
      key: "__hidden__", title: `本档隐藏（${hiddenList.length}）`, items: hiddenList, hidden: true,
      note: "藏面：schema 不可见、协议层按名可调（图文引擎模式中立，常态为空）。",
    })
  }
  return groups
}

export function ToolsTabContent({ toolCounts, liveTools, refreshKey = 0 }: { toolCounts: Counts; liveTools?: string[]; refreshKey?: number }) {
  const [query, setQuery] = useState("")
  const inputRef = useRef<HTMLInputElement>(null)
  // 装配面自取（2026-11-21 缝）：引擎主注册表全量（描述=模型可见口径，含
  // 藏面）——面板与模型同源；未就绪/旧后端时沿用 liveTools/静态清单兜底。
  const [surface, setSurface] = useState<ToolSurfaceViewB | null>(null)
  useEffect(() => {
    let alive = true
    try {
      app.WenxueToolSurface()
        .then((v) => { if (alive && v.tools.length > 0) setSurface(v) })
        .catch(() => { /* 未就绪：保留兜底 */ })
    } catch {
      /* mock/旧壳无此方法：TypeError 同样落兜底 */
    }
    return () => { alive = false }
  }, [refreshKey])

  const liveGroups = useMemo(() => (surface ? groupSurface(surface.tools) : null), [surface])
  const liveDescs = useMemo(() => {
    if (!surface) return undefined
    const m: Record<string, string> = {}
    for (const t of surface.tools) if (t.description) m[t.name] = t.description
    return m
  }, [surface])

  // 兜底源（刀3 续）：引擎实时名单优先，不在任何分组的归「其他」组；
  // 实时名单也缺席（旧后端/加载前）退回静态+deny 滤。
  const sections = useMemo(() => {
    if (!liveTools || liveTools.length === 0) return SECTIONS
    const allowed = new Set(liveTools)
    const covered = new Set<string>()
    const base = RAW_SECTIONS.map((sec) => ({
      ...sec,
      items: sec.items.filter((name) => !isDisabledWenxueTool(name) && allowed.has(name)),
    })).filter((sec) => sec.items.length > 0)
    for (const sec of base) for (const n of sec.items) covered.add(n)
    const extras = liveTools.filter((n) => !covered.has(n) && !isDisabledWenxueTool(n))
    return extras.length > 0 ? [...base, { title: "其他", items: extras }] : base
  }, [liveTools])

  const totalTools = surface
    ? surface.tools.length
    : sections.reduce((sum, s) => sum + s.items.length, 0)
  const visibleCount = surface ? surface.tools.filter((t) => !t.hidden).length : null
  const activeTotal = useMemo(
    () => sections.reduce((sum, s) => sum + s.items.filter((n) => (toolCounts[n] ?? 0) > 0).length, 0),
    [sections, toolCounts],
  )

  const filteredGroups = useMemo(() => {
    if (liveGroups) {
      if (!query.trim()) return liveGroups
      const q = query.toLowerCase()
      return liveGroups
        .map((g) => ({
          ...g,
          items: g.items.filter(
            (t) =>
              t.name.toLowerCase().includes(q) ||
              (t.description ?? "").toLowerCase().includes(q),
          ),
        }))
        .filter((g) => g.items.length > 0)
    }
    if (!query.trim()) return sections
    const q = query.toLowerCase()
    return sections
      .map((sec) => ({
        ...sec,
        items: sec.items.filter(
          (name) =>
            name.toLowerCase().includes(q) ||
            (TOOL_DESC[name] ?? "").toLowerCase().includes(q),
        ),
      }))
      .filter((sec) => sec.items.length > 0)
  }, [liveGroups, sections, query])

  const hasResults = filteredGroups !== null && filteredGroups.length > 0

  return (
    <div className="flex flex-col overflow-hidden h-full" style={{minHeight: 0}}>
      <div className="flex items-center gap-1.5 px-2 py-2 font-semibold text-[11px] shrink-0" style={{ color: "var(--md-sys-color-text-secondary)" }}>
        <Cpu size={12} aria-hidden className="text-(color:--gaea-glow)" />
        <span>工具</span>
        <span className="ml-auto text-[10px] font-mono" style={{ color: "var(--md-sys-color-text-secondary)" }}>
          {surface
            ? `${visibleCount}/${totalTools}`
            : activeTotal > 0 ? `${activeTotal}/${totalTools}` : totalTools}
        </span>
      </div>
      <div
        className="flex items-center gap-1.5 mx-2 my-1 px-2 h-7 rounded-md border shrink-0 transition-[border-color,box-shadow] duration-200 focus-within:border-[color:color-mix(in_srgb,var(--gaea-glow)_45%,var(--md-sys-color-outline-variant))] focus-within:shadow-[0_0_0_2px_color-mix(in_srgb,var(--gaea-glow)_14%,transparent)]"
        style={{
          borderColor: "var(--md-sys-color-outline-variant)",
          background: "var(--md-sys-color-surface-container)",
          color: "var(--md-sys-color-text-secondary)",
        }}
      >
        <Search size={12} aria-hidden />
        <input
          ref={inputRef}
          className="flex-1 min-w-0 border-0 outline-none bg-transparent text-[11.5px] placeholder:text-(color:--md-sys-color-text-secondary)"
          style={{ color: "var(--md-sys-color-text)" }}
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="搜索工具…"
          aria-label="搜索工具"
        />
        {query && (
          <button
            className="border-0 bg-transparent p-0 leading-none cursor-pointer hover:opacity-80"
            style={{ color: "var(--md-sys-color-text-secondary)" }}
            onClick={() => { setQuery(""); inputRef.current?.focus() }}
            aria-label="清除搜索"
          >
            <X size={12} />
          </button>
        )}
      </div>
      <div className="flex-1 min-h-0 overflow-y-auto pb-2">
        {!hasResults ? (
          <div className="text-xs text-center py-8" style={{ color: "var(--md-sys-color-text-secondary)" }}>无匹配工具</div>
        ) : liveGroups ? (
          (filteredGroups as { key: string; title: string; items: ToolSurfaceEntryB[]; hidden: boolean; note?: string }[]).map((g) => (
            <ToolGroup
              key={g.key}
              title={g.title}
              items={g.items.map((t) => t.name)}
              counts={toolCounts}
              descs={liveDescs}
              hidden={g.hidden}
              groupNote={g.note}
              defaultOpen={!g.hidden && (filteredGroups!.length <= 4 || g.key === "builtin-domain")}
            />
          ))
        ) : (
          (filteredGroups as Section[]).map((sec) => (
            <ToolGroup
              key={sec.title}
              title={sec.title}
              items={sec.items}
              counts={toolCounts}
              defaultOpen={sec.title === "文件" || filteredGroups!.length <= 3}
            />
          ))
        )}
      </div>
    </div>
  )
}
