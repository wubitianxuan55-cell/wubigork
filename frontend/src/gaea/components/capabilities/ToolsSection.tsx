// CapabilitiesPanel 拆分产物：工具列表区（行为零变化，T6-10.1）
// v3「星枢」面板语言：工具卡实底收敛（图标 + 名称 + 描述 + 状态徽标），
// 激活工具 = 主色容器底 + 左侧光条（inset shadow）+ 柔光。
import { useEffect, useMemo, useRef, useState } from "react";
import { Cpu, ChevronDown, Search, X } from "../../icons";
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
  memory_get: "按ID读取单条记忆详情",
  promote_session_facts: "把当前会话事实提升为持久记忆",
  knowledge_search: "搜索工程知识库(关键词/分类/标签)",
  knowledge_add: "向知识库添加条目(标题+分类+正文)",
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
  { title: "记忆", items: ["remember", "forget", "memory_search", "memory_get", "promote_session_facts"] },
  { title: "知识库", items: ["knowledge_search", "knowledge_add"] },
  { title: "技能", items: ["read_skill", "run_skill", "install_skill", "slash_command"] },
  { title: "文档技能", items: ["docx", "xlsx", "pdf", "pptx"] },
  { title: "本地专业模型", items: ["vision", "ocr", "semantic_search", "translate_text"] },
  { title: "通用办公", items: ["format_convert", "chart_gen", "routine_llm"] },
]

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
  /** 模式藏面组（Registry.Hide）：灰显+藏徽标——诚实显形「他档能力」。 */
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
  { title: "文件与命令", names: ["read_file", "write_file", "ls", "grep", "edit_file", "edit_lines", "multi_edit", "replace", "move_file", "bash", "bash_output", "wait", "kill_shell", "read_spill", "summarize_file", "screen_capture"] },
  { title: "网络", names: ["web_search", "web_fetch"] },
  { title: "任务与协作", names: ["todo_write", "complete_step", "ask", "task", "exit_plan_mode", "request_permission", "interrupt_agent", "subagent_list", "send_message", "dag_plan"] },
  { title: "记忆", names: ["remember", "forget", "memory_search", "memory_get", "promote_session_facts"] },
  { title: "知识库", names: ["knowledge_search", "knowledge_add", "semantic_search"] },
  { title: "技能", names: ["read_skill", "run_skill", "install_skill", "slash_command"] },
  { title: "文档与交付", names: ["format_convert", "office_generate", "docx_apply", "docx_accept", "pptx_apply", "xlsx_apply", "xlsx_profile", "xlsx_recalc", "xlsx_audit", "office_batch", "doc_render", "md_validate", "ai_feedback", "fact_add", "fact_list", "fact_clear"] },
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
  const [busy, setBusy] = useState(false)
  const [notice, setNotice] = useState<string | null>(null)
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
  return (
    <div className="mx-2 my-1 px-2 py-1.5 rounded-md border" style={{ borderColor: "var(--md-sys-color-outline-variant)", background: "var(--md-sys-color-surface-container)" }}>
      <div className="flex items-center gap-1.5">
        <span className="text-[10px] font-semibold" style={{ color: "var(--md-sys-color-text-secondary)" }}>工具面</span>
        {(["lean", "standard"] as const).map((face) => (
          <button key={face} type="button"
            className={`px-2 py-0.5 text-[10.5px] rounded-md border cursor-pointer transition-[color,background,border] duration-150 ${view.face === face ? "font-semibold" : ""}`}
            style={view.face === face
              ? { color: "var(--gaea-glow)", borderColor: "color-mix(in srgb, var(--gaea-glow) 40%, transparent)", background: "color-mix(in srgb, var(--gaea-glow) 12%, transparent)" }
              : { color: "var(--md-sys-color-text-secondary)", borderColor: "var(--md-sys-color-outline-variant)", background: "transparent" }}
            disabled={busy}
            title={face === "lean" ? "精简：关浏览器自动化与 COM 实时协同（CAD 随造价档、进度随方案档）" : "标准：恢复浏览器自动化与 COM（跟随本机装机）"}
            onClick={() => { if (view.face !== face) void apply(() => app.GaeaSetToolFace(face)) }}
            aria-pressed={view.face === face}
          >
            {face === "lean" ? "精简" : "标准"}
          </button>
        ))}
        {Object.keys(view.overrides).length > 0 && (
          <button type="button"
            className="ml-auto text-[10px] border-0 bg-transparent cursor-pointer underline underline-offset-2"
            style={{ color: "var(--md-sys-color-text-secondary)" }}
            disabled={busy}
            onClick={() => { void apply(() => app.GaeaResetToolGroups()) }}
          >
            重置组覆盖
          </button>
        )}
      </div>
      <div className="flex flex-col gap-0.5 mt-1">
        {view.groups.map((g) => (
          <div key={g.id} className="flex items-center gap-1.5 text-[10.5px] leading-[1.4]" title={g.note}>
            <span style={{ color: g.enabled ? "var(--md-sys-color-text)" : "var(--md-sys-color-text-secondary)" }}>{g.label}</span>
            <span className="font-mono text-[9px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>{g.count}</span>
            <span className="ml-auto">
              {g.overridable ? (
                <button type="button"
                  role="switch"
                  aria-checked={g.enabled}
                  disabled={busy}
                  className="px-1.5 py-px text-[9.5px] rounded-full border cursor-pointer"
                  style={g.enabled
                    ? { color: "var(--gaea-glow)", borderColor: "color-mix(in srgb, var(--gaea-glow) 40%, transparent)", background: "color-mix(in srgb, var(--gaea-glow) 12%, transparent)" }
                    : { color: "var(--md-sys-color-text-secondary)", borderColor: "var(--md-sys-color-outline-variant)", background: "transparent" }}
                  onClick={() => { void apply(() => app.GaeaSetToolGroup(g.id, !g.enabled)) }}
                >
                  {g.enabled ? "开" : "关"}
                </button>
              ) : g.gate ? (
                <button type="button"
                  role="switch"
                  aria-checked={g.enabled}
                  disabled={busy}
                  className="px-1.5 py-px text-[9.5px] rounded-full border cursor-pointer"
                  style={g.enabled
                    ? { color: "var(--gaea-glow)", borderColor: "color-mix(in srgb, var(--gaea-glow) 40%, transparent)", background: "color-mix(in srgb, var(--gaea-glow) 12%, transparent)" }
                    : { color: "var(--md-sys-color-text-secondary)", borderColor: "var(--md-sys-color-outline-variant)", background: "transparent" }}
                  onClick={() => { void apply(() => app.GaeaSetComputerUseEnabled(!g.enabled)) }}
                >
                  {g.enabled ? "开" : "关"}
                </button>
              ) : (
                <span className="text-[9.5px]" style={{ color: "var(--md-sys-color-text-secondary)" }}>注册闸</span>
              )}
            </span>
          </div>
        ))}
      </div>
      {notice && <div className="mt-1 text-[10px] leading-[1.4]" style={{ color: "var(--md-sys-color-text-secondary)" }}>{notice}</div>}
    </div>
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

  return (
    <div className="flex flex-col overflow-hidden h-full" style={{minHeight: 0}}>
      <div className="flex items-center gap-1.5 px-2 py-2 font-semibold text-[11px] shrink-0" style={{ color: "var(--md-sys-color-text-secondary)" }}>
        <Cpu size={12} aria-hidden className="text-(color:--gaea-glow)" />
        <span>工具</span>
        {faceView && (
          <span className="text-[9px] rounded-full px-1.5 py-px" title={faceView.face === "lean" ? "精简面：新建会话默认（关浏览器自动化/COM）" : "标准面：一键回退（浏览器/COM 恢复）"} style={{ color: "var(--gaea-glow)", background: "color-mix(in srgb, var(--gaea-glow) 14%, transparent)" }}>
            {faceView.face === "lean" ? "精简面" : "标准面"}
          </span>
        )}
        {surface?.mode && RUNMODE_LABELS[surface.mode] && (
          <span
            className="text-[9px] rounded-full px-1.5 py-px"
            title="引擎实际装配档：本清单活取自主注册表，与模型看到的工具面同源；切档后重载生效"
            style={{ color: "var(--gaea-glow)", background: "color-mix(in srgb, var(--gaea-glow) 14%, transparent)" }}
          >
            {RUNMODE_LABELS[surface.mode]}档
          </span>
        )}
        <span className="ml-auto text-[10px] font-mono" style={{ color: "var(--md-sys-color-text-secondary)" }}>
          {surface
            ? `${visibleTools!.length}/${totalTools}`
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
      {faceView && <ToolFaceBar view={faceView} onChanged={() => setLocalTick((t) => t + 1)} />}
      <div className="flex-1 min-h-0 overflow-y-auto pb-2">
        {!hasResults ? (
          <div className="text-xs text-center py-8" style={{ color: "var(--md-sys-color-text-secondary)" }}>无匹配工具</div>
        ) : surface ? (
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
