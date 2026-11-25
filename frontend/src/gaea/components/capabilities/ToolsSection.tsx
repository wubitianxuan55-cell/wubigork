// CapabilitiesPanel 拆分产物：工具列表区（行为零变化，T6-10.1）
// v3「星枢」面板语言：工具卡实底收敛（图标 + 名称 + 描述 + 状态徽标），
// 激活工具 = 主色容器底 + 左侧光条（inset shadow）+ 柔光。
import { useEffect, useId, useMemo, useRef, useState, type ReactNode } from "react";
import {
  BookOpen,
  Blocks,
  ChevronDown,
  Command,
  Cpu,
  Database,
  FileText,
  Globe,
  Search,
  Users,
  Wand2,
  Wrench,
  X,
  Zap,
} from "../../icons";
import { useGSAPCollapse } from "../../lib/useGSAPCollapse";
import { app } from "../../lib/bridge";
import type { ToolFaceViewB, ToolSurfaceEntryB, ToolSurfaceViewB } from "../../lib/bridge/model";

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
  semantic_search: "本地 bge-m3——跨库语义检索(成本/知识/办公记忆)",
  routine_llm: "通用文本兜底——纯文本摘要/归一化/抽取/改写，本地/免费优先",
  translate_text: "本地翻译——优先 Hunyuan-MT / Hy-MT 翻译模型，未安装时回退办公模型",
}

interface Section {
  title: string
  items: string[]
}

const SECTIONS: Section[] = [
  { title: "文件与命令", items: ["read_file", "write_file", "ls", "bash", "bash_output", "wait", "kill_shell"] },
  { title: "网络", items: ["web_search", "web_fetch"] },
  { title: "任务与协作", items: ["todo_write", "complete_step", "ask", "task"] },
  { title: "记忆", items: ["remember", "memory_search"] },
  { title: "技能", items: ["read_skill", "run_skill", "install_skill"] },
  { title: "文档技能", items: ["docx", "xlsx", "pdf", "pptx"] },
  { title: "本地专业模型", items: ["vision", "ocr", "semantic_search", "translate_text"] },
  { title: "通用办公", items: ["format_convert", "chart_gen", "routine_llm"] },
]

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
    "c:文件与命令": <Command size={12} aria-hidden />,
    "c:网络": <Globe size={12} aria-hidden />,
    "c:任务与协作": <Users size={12} aria-hidden />,
    "c:记忆": <BookOpen size={12} aria-hidden />,
    "c:知识库": <Database size={12} aria-hidden />,
    "c:技能": <Wand2 size={12} aria-hidden />,
    "c:文档与交付": <FileText size={12} aria-hidden />,
    "c:本地模型": <Cpu size={12} aria-hidden />,
    "c:界面": <Blocks size={12} aria-hidden />,
    "g:cost": <Zap size={12} aria-hidden />,
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
  /** 模式藏面组（Registry.Hide）：灰显+藏徽标——诚实显形「他档能力」。 */
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
    "文件与命令": "c:文件与命令",
    "网络": "c:网络",
    "任务与协作": "c:任务与协作",
    "记忆": "c:记忆",
    "知识库": "c:知识库",
    "技能": "c:技能",
    "文档与交付": "c:文档与交付",
    "本地模型": "c:本地模型",
    "界面": "c:界面",
    "造价专用": "g:cost",
  }
  return byTitle[title] ?? ""
}

// 功能分组（2026-11-21 用户拍板：工具面板按功能分组显示，不再按装配来源）。
// 配置组（cad/schedule/browser/com/cost/longdoc/media/computer）用组名直显；
// core（无组归属）按语义小节映射（文件与命令/网络/任务/记忆/知识/技能/文档
// 交付/本地模型/界面），未命中落「其他」；MCP 按服务器分组。
const GROUP_LABELS: Record<string, string> = {
  cad: "CAD 图纸",
  schedule: "进度计划",
  browser: "浏览器自动化",
  com: "COM 实时协同",
  computer: "电脑操控",
  cost: "造价专用",
  longdoc: "长文方案",
  media: "图面与生图",
}
const CORE_SECTIONS: { title: string; names: string[] }[] = [
  { title: "文件与命令", names: ["read_file", "write_file", "ls", "grep", "edit_file", "move_file", "bash", "bash_output", "wait", "kill_shell", "read_spill", "summarize_file", "screen_capture"] },
  { title: "网络", names: ["web_search", "web_fetch"] },
  { title: "任务与协作", names: ["todo_write", "complete_step", "ask", "task", "exit_plan_mode", "request_permission", "interrupt_agent", "subagent_list", "send_message"] },
  { title: "记忆", names: ["remember", "memory_search"] },
  { title: "跨库检索", names: ["semantic_search"] },
  { title: "技能", names: ["read_skill", "run_skill", "install_skill"] },
  { title: "文档与交付", names: ["format_convert", "office_generate", "docx_apply", "pptx_apply", "xlsx_apply", "xlsx_profile", "xlsx_recalc", "xlsx_audit", "office_batch", "doc_render", "md_validate", "ai_feedback", "fact_add", "fact_list", "fact_clear"] },
  { title: "本地模型", names: ["routine_llm", "translate_text", "vision", "ocr"] },
  { title: "界面", names: ["sidebar_open", "genui_validate"] },
]
const RUNMODE_LABELS: Record<string, string> = { standard: "标准", cost: "造价", proposal: "方案" }

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
  for (const g of ["cad", "schedule", "cost", "longdoc", "media", "browser", "com", "computer"]) {
    const key = `g:${g}`
    const items = byKey.get(key)
    if (items?.length) {
      groups.push({ key, title: GROUP_LABELS[g] ?? g, items, hidden: false })
      byKey.delete(key)
    }
  }
  for (const [key, items] of byKey) {
    const title = key.startsWith("mcp:") ? `MCP：${key.slice(4)}` : key === "other" ? "其他" : key
    groups.push({ key, title, items, hidden: false })
  }
  if (hiddenList.length) {
    groups.push({
      key: "__hidden__", title: `本档隐藏（${hiddenList.length}）`, items: hiddenList, hidden: true,
      note: "模式藏面：本档 schema 不可见、协议层按名可调；切换到对应运行模式后进工具清单（新会话生效）。",
    })
  }
  return groups
}

// ToolFaceBar：工具面双轴控制条（2026-11-21 众神殿 pan_1795224680124 终案）——
// 一键「精简/标准」预设 + 组级开关（登记默认值，新会话生效；文案复用 hidden
// 现成语义）。computer 组走 [computer_use] 注册闸不参与开关。
function ToolFaceBar({ view, onChanged }: { view: ToolFaceViewB; onChanged: () => void }) {
  const [open, setOpen] = useState(false)
  const [busy, setBusy] = useState(false)
  const [notice, setNotice] = useState<string | null>(null)
  const enabledGroups = view.groups.filter((group) => group.enabled).length
  const apply = async (fn: () => Promise<void>) => {
    setBusy(true)
    setNotice(null)
    try {
      await fn()
      setNotice("已登记：新会话生效（会话创建时定面，当前会话不中断）")
      onChanged()
    } catch (e) {
      setNotice(e instanceof Error ? e.message : String(e))
    } finally {
      setBusy(false)
    }
  }
  const faceLabel = view.face === "lean" ? "精简面" : "标准面"
  return (
    <section className="tool-face" aria-label="工具面设置">
      <div className="tool-face__bar">
        <div className="tool-face__identity">
          <span className="tool-face__icon" aria-hidden="true"><Wrench size={13} /></span>
          <div className="tool-face__copy">
            <strong>工具面</strong>
            <span>{faceLabel} · {enabledGroups}/{view.groups.length} 组启用</span>
          </div>
        </div>
        <div className="tool-face__presets" role="group" aria-label="工具面预设">
          {(["lean", "standard"] as const).map((face) => (
            <button
              key={face}
              type="button"
              className="tool-face__preset"
              disabled={busy}
              title={face === "lean" ? "精简：关浏览器自动化与 COM 实时协同（CAD 随造价档、进度随方案档）" : "标准：恢复浏览器自动化与 COM（跟随本机装机）"}
              onClick={() => { if (view.face !== face) void apply(() => app.GaeaSetToolFace(face)) }}
              aria-pressed={view.face === face}
            >
              {face === "lean" ? "精简" : "标准"}
            </button>
          ))}
        </div>
        <button
          className="tool-face__expand"
          type="button"
          aria-expanded={open}
          onClick={() => setOpen((value) => !value)}
        >
          <span>{open ? "收起" : "调整"}</span>
          <ChevronDown
            size={10}
            aria-hidden
            style={{ transform: open ? "rotate(0deg)" : "rotate(-90deg)", transition: "transform 150ms ease" }}
          />
        </button>
      </div>
      {open && (
        <div className="tool-face__details">
          <div className="tool-face__details-head">
            <span>工具组在新会话创建时定面，当前会话不中断</span>
            {Object.keys(view.overrides).length > 0 && (
              <button
                type="button"
                className="tool-face__reset"
                disabled={busy}
                onClick={() => { void apply(() => app.GaeaResetToolGroups()) }}
              >
                重置组覆盖
              </button>
            )}
          </div>
          {view.groups.map((g) => (
            <div key={g.id} className="tool-face__row" data-enabled={g.enabled} title={g.note}>
              <span className="tool-face__label">{g.label}</span>
              <span className="tool-face__count">{g.count} 项</span>
              {g.overridable ? (
                <button
                  type="button"
                  role="switch"
                  aria-checked={g.enabled}
                  aria-label={`${g.label}：${g.enabled ? "开" : "关"}`}
                  disabled={busy}
                  className="tool-face__state"
                  data-on={g.enabled}
                  onClick={() => { void apply(() => app.GaeaSetToolGroup(g.id, !g.enabled)) }}
                >
                  {g.enabled ? "开" : "关"}
                </button>
              ) : g.gate ? (
                <button
                  type="button"
                  role="switch"
                  aria-checked={g.enabled}
                  aria-label={`${g.label}：${g.enabled ? "开" : "关"}`}
                  disabled={busy}
                  className="tool-face__state"
                  data-on={g.enabled}
                  onClick={() => { void apply(() => app.GaeaSetComputerUseEnabled(!g.enabled)) }}
                >
                  {g.enabled ? "开" : "关"}
                </button>
              ) : (
                <button type="button" className="tool-face__state" disabled>注册闸</button>
              )}
            </div>
          ))}
        </div>
      )}
      {notice && <div className="tool-face__notice" role="status">{notice}</div>}
    </section>
  )
}

export function ToolsTabContent({ toolCounts, refreshKey = 0, board = "gaea" }: { toolCounts: Counts; refreshKey?: number; board?: "gaea" | "wenxue" }) {
  const [query, setQuery] = useState("")
  const inputRef = useRef<HTMLInputElement>(null)
  const [localTick, setLocalTick] = useState(0)
  // 装配面活取（2026-11-21 缝）：引擎主注册表全量（含模式藏面）——面板与
  // 模型同源。引擎未就绪（首装配前/mock dev）时 null=静态清单兜底（现状）。
  const [surface, setSurface] = useState<ToolSurfaceViewB | null>(null)
  const [faceView, setFaceView] = useState<ToolFaceViewB | null>(null)
  useEffect(() => {
    let alive = true
    try {
      const fetcher = board === "wenxue" ? app.WenxueToolSurface() : app.GaeaToolSurface()
      fetcher
        .then((v) => { if (alive && v.tools.length > 0) setSurface(v) })
        .catch(() => { /* 未就绪：保留兜底 */ })
    } catch {
      /* mock/旧壳无此方法：TypeError 同样落兜底（静态清单） */
    }
    if (board === "gaea") {
      try {
        app.GaeaToolFace()
          .then((v) => { if (alive) setFaceView(v) })
          .catch(() => { /* 未就绪：不显控制条 */ })
      } catch { /* mock 同上 */ }
    }
    return () => { alive = false }
  }, [refreshKey, localTick, board])

  const groups = useMemo(
    () => (surface ? groupSurface(surface.tools) : null),
    [surface],
  )
  const totalTools = surface ? surface.tools.length : SECTIONS.reduce((sum, s) => sum + s.items.length, 0)
  const visibleTools = surface ? surface.tools.filter((t) => !t.hidden) : null
  const activeTotal = useMemo(
    () => (surface
      ? visibleTools!.filter((t) => (toolCounts[t.name] ?? 0) > 0).length
      : SECTIONS.reduce((sum, s) => sum + s.items.filter((n) => (toolCounts[n] ?? 0) > 0).length, 0)),
    [surface, visibleTools, toolCounts],
  )
  // 活取描述表（含 hidden——藏面卡同样给描述）。
  const liveDescs = useMemo(() => {
    if (!surface) return undefined
    const m: Record<string, string> = {}
    for (const t of surface.tools) if (t.description) m[t.name] = t.description
    return m
  }, [surface])

  const filteredGroups = useMemo(() => {
    if (!groups) {
      // 兜底：无查询=静态清单全量（现状行为）；有查询=按名/描述过滤。
      if (!query.trim()) return SECTIONS
      const q = query.toLowerCase()
      return SECTIONS
        .map((sec) => ({
          ...sec,
          items: sec.items.filter(
            (name) =>
              name.toLowerCase().includes(q) ||
              (TOOL_DESC[name] ?? "").toLowerCase().includes(q),
          ),
        }))
        .filter((sec) => sec.items.length > 0)
    }
    if (!query.trim()) return groups
    const q = query.toLowerCase()
    return groups
      .map((g) => ({
        ...g,
        items: g.items.filter(
          (t) =>
            t.name.toLowerCase().includes(q) ||
            (t.description ?? "").toLowerCase().includes(q),
        ),
      }))
      .filter((g) => g.items.length > 0)
  }, [groups, query])

  const hasResults = !!filteredGroups && filteredGroups.length > 0
  const filteredCount = filteredGroups?.reduce((sum, group) => sum + group.items.length, 0) ?? 0
  const visibleTotal = visibleTools?.length ?? totalTools

  return (
    <div className="tools-tab">
      <header className="tools-overview">
        <div className="tools-overview__copy">
          <div className="tools-overview__eyebrow"><Cpu size={11} aria-hidden /><span>能力目录</span></div>
          <h3 className="tools-overview__title">
            可用工具
            {surface?.mode && RUNMODE_LABELS[surface.mode] && (
              <span
                className="tools-mode-badge"
                title="引擎实际装配档：本清单活取自主注册表，与模型看到的工具面同源；切档后重载生效"
              >
                {RUNMODE_LABELS[surface.mode]}档
              </span>
            )}
          </h3>
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

      {faceView && <ToolFaceBar view={faceView} onChanged={() => setLocalTick((t) => t + 1)} />}

      <div className="tools-list">
        {!hasResults ? (
          <div className="tools-empty">
            <span className="tools-empty__icon" aria-hidden="true"><Search size={17} /></span>
            <strong>{query.trim() ? "没有找到工具" : "暂无可展示工具"}</strong>
            <p>{query.trim() ? "试试更短的名称，或搜索“文件”“记忆”“文档”等用途关键词。" : "引擎尚未返回工具目录；稍后热加载再试。"}</p>
            {query.trim() && (
              <button type="button" onClick={() => { setQuery(""); inputRef.current?.focus() }}>清除搜索</button>
            )}
          </div>
        ) : surface ? (
          (filteredGroups as { key: string; title: string; items: ToolSurfaceEntryB[]; hidden: boolean; note?: string }[]).map((g, index) => (
            <ToolGroup
              key={g.key}
              title={g.title}
              items={g.items.map((t) => t.name)}
              counts={toolCounts}
              descs={liveDescs}
              hidden={g.hidden}
              groupNote={g.note}
              forceOpen={!!query.trim()}
              defaultOpen={!g.hidden && (!!query.trim() || index < 2 || filteredGroups!.length <= 4)}
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
              defaultOpen={!!query.trim() || index < 2 || filteredGroups!.length <= 3}
            />
          ))
        )}
      </div>
    </div>
  )
}
