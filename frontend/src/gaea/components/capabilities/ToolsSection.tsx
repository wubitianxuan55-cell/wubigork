// CapabilitiesPanel 拆分产物：工具列表区（行为零变化，T6-10.1）
import { useMemo, useRef, useState } from "react";
import { Cpu, ChevronDown, Search } from "../../icons";
import { useGSAPCollapse } from "../../lib/useGSAPCollapse";

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
  "format-convert": "子代理——格式转换(统一为可编辑Markdown)",
  "chart-builder": "子代理——图表生成(数据可视化)",
  "doc-assemble": "子代理——文档拼装(多份素材→完整报告)",
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
  { title: "通用办公", items: ["format_convert", "chart_gen", "routine_llm", "format-convert", "chart-builder", "doc-assemble"] },
]

function ToolCard({ name, count }: { name: string; count: number }) {
  const active = count > 0
  const desc = TOOL_DESC[name]
  return (
    <div
      className={`flex items-start gap-1.5 px-2 py-1.5 rounded-md border border-border-soft bg-bg cursor-default ${
        active ? "border-accent-soft bg-sidebar-active" : ""
      }`}
      title={desc ?? name}
    >
      <span className={`w-1.5 h-1.5 mt-[5px] rounded-full shrink-0 ${active ? "bg-accent" : "bg-border-soft"}`} />
      <span className="flex-1 min-w-0 flex flex-col gap-0.5 leading-[1.25]">
        <span className={`font-mono text-[10.5px] truncate ${active ? "text-accent font-semibold" : "text-fg-dim"}`}>
          {name}
        </span>
        {desc && <span className="text-[10px] text-fg-faint leading-[1.3] line-clamp-1">{desc}</span>}
      </span>
      <span className={`shrink-0 font-mono text-[11px] font-semibold mt-px ${active ? "text-accent" : "text-fg-faint"}`}>
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
}: {
  title: string
  items: string[]
  counts: Counts
  defaultOpen: boolean
}) {
  const [open, setOpen] = useState(defaultOpen)
  const ref = useRef<HTMLDivElement>(null)
  useGSAPCollapse(ref, open, { duration: 0.18 })

  const activeCount = items.filter((n) => (counts[n] ?? 0) > 0).length

  return (
    <div className="px-1.5 py-0.5">
      <button
        className="flex items-center gap-1 w-full px-1 py-1.5 bg-transparent border-0 text-left cursor-pointer hover:bg-bg-soft rounded transition-colors"
        onClick={() => setOpen((v) => !v)}
      >
        <ChevronDown
          size={10}
          className={`text-fg-faint transition-transform duration-150 ${open ? "rotate-0" : "-rotate-90"}`}
        />
        <span className="text-[10px] font-semibold uppercase tracking-[0.5px] text-fg-faint">{title}</span>
        {activeCount > 0 && (
          <span className="ml-auto text-[9px] font-mono text-accent">{activeCount}</span>
        )}
      </button>
      <div ref={ref} style={{ overflow: "hidden" }}>
        <div className="flex flex-col gap-0.5 pt-0.5 pb-1">
          {items.map((name) => (
            <ToolCard key={name} name={name} count={counts[name] ?? 0} />
          ))}
        </div>
      </div>
    </div>
  )
}

export function ToolsTabContent({ toolCounts }: { toolCounts: Counts }) {
  const [query, setQuery] = useState("")
  const inputRef = useRef<HTMLInputElement>(null)

  const totalTools = SECTIONS.reduce((sum, s) => sum + s.items.length, 0)
  const activeTotal = useMemo(
    () => SECTIONS.reduce((sum, s) => sum + s.items.filter((n) => (toolCounts[n] ?? 0) > 0).length, 0),
    [toolCounts],
  )

  const filteredSections = useMemo(() => {
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
  }, [query])

  const hasResults = filteredSections.length > 0

  return (
    <div className="flex flex-col overflow-hidden h-full" style={{minHeight: 0}}>
      <div className="flex items-center gap-1.5 px-2 py-2 text-fg-dim font-semibold text-[11px] shrink-0">
        <Cpu size={12} />
        <span>工具</span>
        <span className="ml-auto text-[10px] font-mono text-fg-faint/50">
          {activeTotal > 0 ? `${activeTotal}/${totalTools}` : totalTools}
        </span>
      </div>
      <div className="flex items-center gap-1.5 mx-2 my-1 px-2 h-7 border border-border rounded-md bg-bg text-fg-faint shrink-0">
        <Search size={12} />
        <input
          ref={inputRef}
          className="flex-1 min-w-0 border-0 outline-none bg-transparent text-fg text-[11.5px] placeholder:text-fg-faint"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="搜索工具…"
        />
        {query && (
          <button
            className="border-0 bg-transparent text-fg-faint cursor-pointer p-0 leading-none hover:text-fg"
            onClick={() => { setQuery(""); inputRef.current?.focus() }}
          >
            ✕
          </button>
        )}
      </div>
      <div className="flex-1 min-h-0 overflow-y-auto pb-2">
        {!hasResults ? (
          <div className="text-fg-faint text-xs text-center py-8">无匹配工具</div>
        ) : (
          filteredSections.map((sec) => (
            <ToolGroup
              key={sec.title}
              title={sec.title}
              items={sec.items}
              counts={toolCounts}
              defaultOpen={sec.title === "文件" || filteredSections.length <= 3}
            />
          ))
        )}
      </div>
    </div>
  )
}
