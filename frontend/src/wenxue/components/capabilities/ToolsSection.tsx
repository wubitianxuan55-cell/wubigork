// CapabilitiesPanel 拆分产物：工具列表区（与办公 ToolsSection 同源镜像，2026-11-26 同步审计）
// v3「星枢」面板语言：工具卡实底收敛（图标 + 名称 + 描述 + 状态徽标），
// 激活工具 = 主色容器底 + 左侧光条（inset shadow）+ 柔光。
// 图文板块口径（渲染面与办公同源，数据面不同）：
//  - 静态兜底表 = 图文专属 + 通用底座（办公域工具按 disabledDomains 滤除）；
//  - 装配面取 WenxueToolSurface；分组按功能语义（沿用办公 2026-11-21 拍板：工具
//    面板按功能分组显示，不再按装配来源），wenxue_* 前缀归「图文专属」；
//  - 无运行模式，故不带 ToolFaceBar 与模式档徽标。
import { useEffect, useId, useMemo, useRef, useState, type ReactNode } from "react";
import {
  BookOpen,
  Blocks,
  ChevronDown,
  Command,
  Cpu,
  FileText,
  Globe,
  Search,
  Users,
  Wand2,
  Wrench,
  X,
} from "../../icons";
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
  read_skill: "读取指定技能的完整内容",
  run_skill: "按名称调用技能或项目命令(playbook)",
  install_skill: "编写并保存新技能",
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
  { title: "记忆", items: ["remember", "memory_search"] },
  { title: "会话", items: ["session_search", "recall_archived"] },
  { title: "技能", items: ["read_skill", "run_skill", "install_skill"] },
  { title: "文档技能", items: ["docx", "xlsx", "pdf", "pptx"] },
  { title: "本地模型", items: ["routine_llm", "vision", "ocr", "image_gen", "translate_text"] },
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
      className="tool-card"
      data-active={active}
      data-hidden={hidden}
      title={desc ? `${name} — ${desc}` : name}
    >
      <div className="tool-card__copy">
        <span className="tool-card__name">{name}</span>
        {desc && (
          <p className="tool-card__desc">{desc}</p>
        )}
      </div>
      <span className="tool-card__badges">
        {hidden && <span className="tool-card__badge tool-card__badge--hidden">本档隐藏</span>}
        {active && <span className="tool-card__badge tool-card__badge--count">{count} 次</span>}
      </span>
    </div>
  )
}

function GroupGlyph({ groupKey }: { groupKey: string }) {
  if (groupKey.startsWith("mcp:")) return <Globe size={12} aria-hidden />
  if (groupKey === "__hidden__") return <Blocks size={12} aria-hidden />
  const iconMap: Record<string, ReactNode> = {
    "c:图文专属": <BookOpen size={12} aria-hidden />,
    "c:文件与命令": <Command size={12} aria-hidden />,
    "c:网络": <Globe size={12} aria-hidden />,
    "c:任务与协作": <Users size={12} aria-hidden />,
    "c:记忆": <BookOpen size={12} aria-hidden />,
    "c:会话": <Search size={12} aria-hidden />,
    "c:技能": <Wand2 size={12} aria-hidden />,
    "c:文档与交付": <FileText size={12} aria-hidden />,
    "c:本地模型": <Cpu size={12} aria-hidden />,
    "c:界面": <Blocks size={12} aria-hidden />,
  }
  return iconMap[groupKey] ?? <Wrench size={12} aria-hidden />
}

function ToolGroup({
  title,
  items,
  counts,
  defaultOpen,
  descs,
  hidden = false,
  groupNote,
  forceOpen = false,
}: {
  title: string
  items: string[]
  counts: Counts
  defaultOpen: boolean
  /** 活取描述表（装配面缝）：有则优先生效，缺项回落静态 TOOL_DESC。 */
  descs?: Record<string, string>
  /** 藏面组：灰显+藏徽标（图文引擎模式中立，此组常态为空；MCP deferred 藏面可达）。 */
  hidden?: boolean
  groupNote?: string
  forceOpen?: boolean
}) {
  const [open, setOpen] = useState(defaultOpen)
  const groupId = useId()
  const ref = useRef<HTMLDivElement>(null)
  useGSAPCollapse(ref, open, { duration: 0.18 })
  useEffect(() => {
    if (forceOpen) setOpen(true)
  }, [forceOpen])

  const activeCount = items.filter((n) => (counts[n] ?? 0) > 0).length

  return (
    <section className="tool-group" data-hidden={hidden}>
      <button
        className="tool-group__toggle"
        type="button"
        aria-expanded={open}
        aria-controls={groupId}
        onClick={() => setOpen((v) => !v)}
      >
        <span className="tool-group__icon"><GroupGlyph groupKey={groupKeyForTitle(title, hidden)} /></span>
        <span className="tool-group__title">{title}</span>
        <span className="tool-group__meta">
          {activeCount > 0 ? `${activeCount}/${items.length} 已用` : `${items.length} 项`}
        </span>
        <ChevronDown className="tool-group__chevron" size={11} aria-hidden />
      </button>
      <div ref={ref} className="tool-group__body" id={groupId}>
        <div>
          {groupNote && (
            <div className="tool-group__note">{groupNote}</div>
          )}
          <div className="tool-group__cards">
            {items.map((name) => (
              <ToolCard key={name} name={name} count={counts[name] ?? 0} desc={descs?.[name] ?? TOOL_DESC[name]} hidden={hidden} />
            ))}
          </div>
        </div>
      </div>
    </section>
  )
}

function groupKeyForTitle(title: string, hidden: boolean): string {
  if (hidden) return "__hidden__"
  if (title.startsWith("MCP：")) return `mcp:${title.slice(4)}`
  const byTitle: Record<string, string> = {
    "图文专属": "c:图文专属",
    "文件与命令": "c:文件与命令",
    "网络": "c:网络",
    "任务与协作": "c:任务与协作",
    "记忆": "c:记忆",
    "会话": "c:会话",
    "技能": "c:技能",
    "文档与交付": "c:文档与交付",
    "本地模型": "c:本地模型",
    "界面": "c:界面",
  }
  return byTitle[title] ?? ""
}

// 功能分组（与办公同源：2026-11-21 用户拍板，工具面板按功能分组显示，不再按
// 装配来源）。配置组（media 等）用组名直显；core（无组归属）按语义小节映射，
// wenxue_* 前缀恒归「图文专属」，未命中落「其他」；MCP 按服务器分组。
const GROUP_LABELS: Record<string, string> = {
  media: "图面与生图",
}
const CORE_SECTIONS: { title: string; names: string[] }[] = [
  { title: "图文专属", names: [] },
  { title: "文件与命令", names: ["read_file", "write_file", "ls", "grep", "bash", "bash_output", "wait", "kill_shell"] },
  { title: "网络", names: ["web_search", "web_fetch"] },
  { title: "任务与协作", names: ["todo_write", "complete_step", "ask", "task", "genui_validate"] },
  { title: "记忆", names: ["remember", "memory_search"] },
  { title: "会话", names: ["session_search", "recall_archived"] },
  { title: "技能", names: ["read_skill", "run_skill", "install_skill"] },
  { title: "本地模型", names: ["routine_llm", "translate_text", "vision", "ocr", "image_gen"] },
]

type SurfaceGroup = { key: string; title: string; items: ToolSurfaceEntryB[]; hidden: boolean; note?: string }

// 装配面 → 功能分组（visible 按功能语义、hidden 收尾组）。
function groupSurface(tools: ToolSurfaceEntryB[]): SurfaceGroup[] {
  const byKey = new Map<string, ToolSurfaceEntryB[]>()
  const hiddenList: ToolSurfaceEntryB[] = []
  const put = (key: string, t: ToolSurfaceEntryB) => {
    if (!byKey.has(key)) byKey.set(key, [])
    byKey.get(key)!.push(t)
  }
  for (const t of tools) {
    if (t.hidden) { hiddenList.push(t); continue }
    if (t.source.startsWith("mcp:")) { put(t.source, t); continue }
    if (t.group) { put(`g:${t.group}`, t); continue }
    if (t.name.startsWith("wenxue_")) { put("c:图文专属", t); continue }
    const sec = CORE_SECTIONS.find((sec) => sec.names.includes(t.name))
    put(sec ? `c:${sec.title}` : "other", t)
  }
  const groups: SurfaceGroup[] = []
  for (const sec of CORE_SECTIONS) {
    const items = byKey.get(`c:${sec.title}`)
    if (items?.length) {
      groups.push({ key: `c:${sec.title}`, title: sec.title, items, hidden: false })
      byKey.delete(`c:${sec.title}`)
    }
  }
  for (const [key, items] of byKey) {
    const title = key.startsWith("mcp:")
      ? `MCP：${key.slice(4)}`
      : key === "other"
        ? "其他"
        : key.startsWith("g:")
          ? (GROUP_LABELS[key.slice(2)] ?? key.slice(2))
          : key
    groups.push({ key, title, items, hidden: false })
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

  const liveGroups = useMemo(
    () => (surface ? groupSurface(surface.tools) : null),
    [surface],
  )
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
  const visibleTools = surface ? surface.tools.filter((t) => !t.hidden) : null
  const visibleTotal = visibleTools?.length ?? totalTools
  const activeTotal = useMemo(
    () => (surface
      ? visibleTools!.filter((t) => (toolCounts[t.name] ?? 0) > 0).length
      : sections.reduce((sum, s) => sum + s.items.filter((n) => (toolCounts[n] ?? 0) > 0).length, 0)),
    [surface, visibleTools, toolCounts, sections],
  )
  // 活取描述表（含 hidden——藏面卡同样给描述）。
  const liveDescs = useMemo(() => {
    if (!surface) return undefined
    const m: Record<string, string> = {}
    for (const t of surface.tools) if (t.description) m[t.name] = t.description
    return m
  }, [surface])

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
  const filteredCount = filteredGroups?.reduce((sum, group) => sum + group.items.length, 0) ?? 0

  return (
    <div className="tools-tab">
      <header className="tools-overview">
        <div className="tools-overview__copy">
          <div className="tools-overview__eyebrow"><Cpu size={11} aria-hidden /><span>能力目录</span></div>
          <h3 className="tools-overview__title">可用工具</h3>
          <p className="tools-overview__sub">
            {activeTotal > 0
              ? `${visibleTotal} 个工具可见 · ${activeTotal} 个已有调用记录`
              : `${visibleTotal} 个工具可见 · 按功能分组浏览`}
          </p>
        </div>
        <div
          className="tools-overview__metric"
          title={surface ? `模型可见 ${visibleTotal} / 注册表共 ${totalTools}` : `当前清单 ${totalTools}`}
          aria-label={surface ? `可见 ${visibleTools!.length} / 共 ${totalTools}` : `当前清单 ${totalTools}`}
        >
          <strong>{surface ? visibleTools!.length : totalTools}</strong>
          <span>{surface ? `/ ${totalTools}` : "项"}</span>
        </div>
      </header>

      <div className="tools-search">
        <Search size={13} aria-hidden />
        <input
          ref={inputRef}
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="搜索工具名或说明…"
          aria-label="搜索工具"
        />
        {query && (
          <button
            className="tools-search__clear"
            type="button"
            onClick={() => { setQuery(""); inputRef.current?.focus() }}
            aria-label="清除搜索"
          >
            <X size={12} />
          </button>
        )}
      </div>

      <div className="tools-results-meta" aria-live="polite">
        <span>{query.trim() ? `“${query.trim()}” 的匹配结果` : "按功能分组 · 展开查看用途"}</span>
        <span><strong>{filteredCount}</strong> 项</span>
      </div>

      <div className="tools-list">
        {!hasResults ? (
          <div className="tools-empty">
            <span className="tools-empty__icon" aria-hidden="true"><Search size={17} /></span>
            <strong>{query.trim() ? "没有找到工具" : "暂无可展示工具"}</strong>
            <p>{query.trim() ? "试试更短的名称，或搜索“文件”“记忆”“章节”等用途关键词。" : "引擎尚未返回工具目录；稍后热加载再试。"}</p>
            {query.trim() && (
              <button type="button" onClick={() => { setQuery(""); inputRef.current?.focus() }}>清除搜索</button>
            )}
          </div>
        ) : liveGroups ? (
          (filteredGroups as SurfaceGroup[]).map((g, index) => (
            <ToolGroup
              key={g.key}
              title={g.title}
              items={g.items.map((t) => t.name)}
              counts={toolCounts}
              descs={liveDescs}
              hidden={g.hidden}
              groupNote={g.note}
              forceOpen={!!query.trim()}
              defaultOpen={!g.hidden && (!!query.trim() || index < 2 || filteredGroups.length <= 4)}
            />
          ))
        ) : (
          (filteredGroups as Section[]).map((sec, index) => (
            <ToolGroup
              key={sec.title}
              title={sec.title}
              items={sec.items}
              counts={toolCounts}
              forceOpen={!!query.trim()}
              defaultOpen={!!query.trim() || index < 2 || filteredGroups.length <= 3}
            />
          ))
        )}
      </div>
    </div>
  )
}
