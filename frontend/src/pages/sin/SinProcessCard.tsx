// sin/SinProcessCard.tsx — 原罪过程卡：思考过程 + 工具调用轨迹。
//
// 视觉语言对齐办公过程卡（可折叠头部 + 工具行 + 展开明细 + 色/图标/文字三重
// 状态传达），但不搬办公的过程卡组件（ToolCard 绑 store 的 Item 与任务卡活动
// 注入），只复用纯函数 boundedOutput；展示映射在 sinToolMeta.ts。
//
// 诚实口径（与插图进度同纪律）：不编造百分比；运行中只给不定态脉冲与「运行中」，
// 耗时只显示后端回传的 elapsed_ms；输出过长用 boundedOutput 留可见折叠标记。
import { useEffect, useMemo, useRef, useState } from 'react'
import { AlertCircle, CheckCircle, ChevronRight, Loader, Wrench } from '../../gaea/icons'
import { boundedOutput } from '../../gaea/lib/tools'
import { readFileAsDataURL } from '../../api/image'
import { TOOL_ICONS, fmtToolElapsed, sinToolLabel, sinToolSubject, sinToolSummary } from './sinToolMeta'
import type { SinToolTraceView } from './types'

export interface SinProcessCardProps {
  reasoning?: string
  tools: SinToolTraceView[]
  /** 本轮是否仍在生成（运行中：头部脉冲 + 工具行运行态）。 */
  running: boolean
}

export function SinProcessCard({ reasoning, tools, running }: SinProcessCardProps) {
  const text = (reasoning ?? '').trim()
  const hasContent = text !== '' || tools.length > 0
  const [open, setOpen] = useState(running)
  const userToggled = useRef(false)

  // 运行中自动展开、收尾自动折叠（用户手动操作过则不干预）——与办公过程卡同规矩。
  useEffect(() => {
    if (userToggled.current) return
    setOpen(running)
  }, [running])

  if (!hasContent) return null

  const summary = sinToolSummary(tools)
  const head: string[] = []
  if (text) head.push(`思考过程 · ${Array.from(text).length} 字`)
  if (summary) head.push(summary)
  const live = running || tools.some((t) => t.status === 'running')

  return (
    <div className="sin-proc" data-running={running ? '' : undefined}>
      <button
        type="button"
        className="sin-proc-head"
        onClick={() => { userToggled.current = true; setOpen((v) => !v) }}
        aria-expanded={open}
      >
        {live
          ? <Loader size={12} className="sin-proc-spin" />
          : <CheckCircle size={12} className="sin-proc-done" />}
        <span className="sin-proc-head-text">{head.join(' · ') || (running ? '工作中' : '过程')}</span>
        <ChevronRight size={12} className={`sin-proc-chevron${open ? ' is-open' : ''}`} />
      </button>
      {open && (
        <div className="sin-proc-body">
          {text && <div className="sin-proc-reason">{text}</div>}
          {tools.map((t, i) => (
            <SinToolRow key={`${t.id || t.name || 'tool'}_${i}`} tool={t} />
          ))}
        </div>
      )}
    </div>
  )
}

function SinToolRow({ tool }: { tool: SinToolTraceView }) {
  const [open, setOpen] = useState(false)
  const Icon = TOOL_ICONS[tool.name as keyof typeof TOOL_ICONS] ?? Wrench
  const args = tool.args ?? ''
  const output = tool.output ?? ''
  const error = tool.error ?? ''
  const expandable = Boolean(args || output || error)

  const status = tool.status === 'running'
    ? '运行中'
    : tool.status === 'failed' ? '失败' : fmtToolElapsed(tool.elapsed_ms)
  const pretty = useMemo(() => {
    if (!args) return ''
    try {
      return JSON.stringify(JSON.parse(args), null, 2)
    } catch {
      return args // 参数不是合法 JSON：原样显示（不假装能解析）
    }
  }, [args])
  const bounded = useMemo(() => boundedOutput(output), [output])

  return (
    <div className={`sin-proc-tool is-${tool.status}`}>
      <button
        type="button"
        className="sin-proc-tool-head"
        onClick={() => { if (expandable) setOpen((v) => !v) }}
        aria-expanded={expandable ? open : undefined}
      >
        <span className="sin-proc-tool-icon">
          {tool.status === 'failed'
            ? <AlertCircle size={12} />
            : tool.status === 'running' ? <Loader size={12} className="sin-proc-spin" /> : <Icon size={12} />}
        </span>
        <span className="sin-proc-tool-label">{sinToolLabel(tool.name ?? '')}</span>
        <span className="sin-proc-tool-subject">{sinToolSubject(tool)}</span>
        <span className="sin-proc-tool-status">{status}</span>
        {expandable && <ChevronRight size={11} className={`sin-proc-chevron${open ? ' is-open' : ''}`} />}
      </button>
      {expandable && open && (
        <div className="sin-proc-tool-body">
          {(tool.artifacts ?? []).filter((a) => a.kind === 'image').map((a, i) => (
            <SinProcArtifact key={`${a.path}_${i}`} path={a.path} caption={a.caption} />
          ))}
          {pretty && (
            <div className="sin-proc-kv">
              <span className="sin-proc-k">参数</span>
              <pre className="sin-proc-pre">{pretty}</pre>
            </div>
          )}
          {error
            ? <div className="sin-proc-err">{error}</div>
            : output && (
              <div className="sin-proc-kv">
                <span className="sin-proc-k">结果</span>
                <pre className="sin-proc-pre">{bounded.preview}</pre>
              </div>
            )}
        </div>
      )}
    </div>
  )
}

/** 工具产物缩略图（sin_illustrate 图片）：本地路径经附件通道转 data URL。 */
function SinProcArtifact({ path, caption }: { path: string; caption?: string }) {
  const [url, setUrl] = useState('')
  const [failed, setFailed] = useState(false)
  useEffect(() => {
    let live = true
    readFileAsDataURL(path)
      .then((u) => { if (live) setUrl(u) })
      .catch(() => { if (live) setFailed(true) })
    return () => { live = false }
  }, [path])
  return (
    <figure className="sin-proc-art">
      {failed
        ? <span className="sin-proc-art-missing">缩略图读取失败</span>
        : <img className="sin-proc-art-img" src={url || undefined} alt={caption || '工具产物插图'} loading="lazy" />}
      {caption && <figcaption className="sin-proc-art-cap">{caption}</figcaption>}
    </figure>
  )
}
