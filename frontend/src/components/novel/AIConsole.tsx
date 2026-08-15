import { useEffect, useRef, useState } from "react";
import { ConsoleSqlOutlined, CloseOutlined } from "@ant-design/icons";
import { useAppStore } from "../../stores/appStore";

// ── 小说板块专属：AI 控制台 ──────────────────────────────────────
// 展示小说 AI 调用的实时输出（xai-output 事件）。从 MainLayout 抽出，
// 仅由 NovelPage 挂载，不再占用全局布局。
// v3.1：改为 v3 玻璃面板语言（.ai-console-* 类），状态色走令牌。

interface LogEntry {
  id: number; type: string; time: string
  model?: string; content?: string; error?: string; length?: number
  system?: string; user?: string
}

/** xai-output 事件动态载荷（最小消费面） */
interface XAIOutputEvent {
  type?: string
  model?: string
  content?: string
  error?: string
  length?: number
  system?: string
  user?: string
}

let logId = 0

/** AIConsole 小说 AI 调用监控面板（右上角悬浮，可展开/关闭）。 */
export function AIConsole() {
  const projectPath = useAppStore((s) => s.projectPath)
  const [logs, setLogs] = useState<LogEntry[]>([])
  const [consoleOpen, setConsoleOpen] = useState(false)
  const [expandedLog, setExpandedLog] = useState<number | null>(null)
  const logEnd = useRef<HTMLDivElement>(null)
  const logContainerRef = useRef<HTMLDivElement>(null)

  // 切换小说时清空旧项目的调用日志，避免把上一本书的请求混进来
  useEffect(() => {
    setLogs([])
    setExpandedLog(null)
  }, [projectPath])

  // 监听 XAI 实时输出事件
  useEffect(() => {
    if (!window.runtime?.EventsOn) return
    const handler = (ev: XAIOutputEvent) => {
      if (!ev) return
      const entry: LogEntry = {
        id: ++logId,
        type: ev.type || 'unknown',
        time: new Date().toLocaleTimeString(),
        model: ev.model,
        content: ev.content,
        error: ev.error,
        length: ev.length,
        system: ev.system,
        user: ev.user,
      }
      setLogs((prev) => [...prev.slice(-99), entry])
    }
    window.runtime.EventsOn('xai-output', handler)
    return () => {
      try {
        window.runtime?.EventsOff?.('xai-output')
      } catch (_) { /* EventsOff 可能不可用 */ }
    }
  }, [])

  // 自动滚动到底部
  useEffect(() => {
    logContainerRef.current?.scrollTo({ top: logContainerRef.current.scrollHeight, behavior: 'smooth' })
  }, [logs])

  return (
    <>
      {consoleOpen && (
        <div className="ai-console-panel" role="log" aria-label="AI 控制台">
          <div className="ai-console-head">
            <span className="ai-console-head-title">
              <i className={`ai-console-status-dot ${logs.length > 0 ? 'is-live' : 'is-idle'}`} aria-hidden />
              AI 控制台
            </span>
            <span className="ai-console-head-count">{logs.length}</span>
            <button
              type="button"
              className="ai-console-head-close"
              onClick={() => setConsoleOpen(false)}
              aria-label="关闭 AI 控制台"
            >
              <CloseOutlined />
            </button>
          </div>

          <div ref={logContainerRef} className="ai-console-body">
            {logs.length === 0 ? (
              <div className="ai-console-empty">
                <ConsoleSqlOutlined aria-hidden />
                <div>等待 AI 调用...</div>
              </div>
            ) : (
              logs.map((l) => {
                const open = expandedLog === l.id
                const borderCls = l.type === 'error' ? 'ai-console-entry-border-error'
                  : l.type === 'request' ? 'ai-console-entry-border-request'
                  : l.type === 'response' ? 'ai-console-entry-border-response'
                  : 'ai-console-entry-border-other'
                const summary = l.type === 'request' ? l.model || ''
                  : l.type === 'response' ? `${(l.length || 0).toLocaleString()} 字`
                  : l.type === 'error' ? l.error?.slice(0, 80) || ''
                  : `+${(l.content?.length || 0)} 字`
                return (
                  <div key={l.id}>
                    <div
                      className={`ai-console-entry ${borderCls}${open ? ' is-open' : ''}`}
                      onClick={() => setExpandedLog(open ? null : l.id)}
                      role="button"
                      tabIndex={0}
                      onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); setExpandedLog(open ? null : l.id) } }}
                    >
                      <span className="ai-console-entry-time">{l.time}</span>
                      <span className={`novel-tag-tone ${l.type === 'error' ? 'is-destructive' : l.type === 'request' ? 'is-primary' : l.type === 'response' ? 'is-success' : 'is-neutral'}`}>
                        {l.type === 'request' ? 'REQ' : l.type === 'response' ? 'OK' : l.type === 'error' ? 'ERR' : '···'}
                      </span>
                      <span className="ai-console-entry-summary">{summary}</span>
                    </div>
                    {open && (
                      <div className="ai-console-detail">
                        {l.type === 'request' && <>
                          {l.system && <div style={{ marginBottom: 4 }}><div className="ai-console-detail-label is-system">SYSTEM</div><pre>{l.system}</pre></div>}
                          {l.user && <div><div className="ai-console-detail-label is-user">USER</div><pre>{l.user}</pre></div>}
                        </>}
                        {l.type === 'response' && l.content && <pre>{l.content}</pre>}
                        {l.type === 'error' && l.error && <div><div className="ai-console-detail-label is-error">ERROR</div><pre>{l.error}</pre></div>}
                        {l.type === 'chunk' && l.content && <pre>{l.content}</pre>}
                      </div>
                    )}
                  </div>
                )
              })
            )}
            <div ref={logEnd} />
          </div>
        </div>
      )}

      {!consoleOpen && (
        <button
          type="button"
          className="ai-console-fab"
          onClick={() => setConsoleOpen(true)}
          aria-label="打开 AI 控制台"
          title="AI 控制台"
        >
          <ConsoleSqlOutlined aria-hidden />
        </button>
      )}
    </>
  )
}
