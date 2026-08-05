import { useEffect, useRef, useState } from "react";
import { Button, Space, Tag, Typography } from "antd";
import { ConsoleSqlOutlined } from "@ant-design/icons";

// ── 小说板块专属：AI 控制台 ──────────────────────────────────────
// 展示小说 AI 调用的实时输出（xai-output 事件）。从 MainLayout 抽出，
// 仅由 NovelPage 挂载，不再占用全局布局。

interface LogEntry {
  id: number; type: string; time: string
  model?: string; content?: string; error?: string; length?: number
  system?: string; user?: string
}

let logId = 0

/** AIConsole 小说 AI 调用监控面板（右上角悬浮，可展开/关闭）。 */
export function AIConsole() {
  const [logs, setLogs] = useState<LogEntry[]>([])
  const [consoleOpen, setConsoleOpen] = useState(false)
  const [expandedLog, setExpandedLog] = useState<number | null>(null)
  const logEnd = useRef<HTMLDivElement>(null)
  const logContainerRef = useRef<HTMLDivElement>(null)

  // 监听 XAI 实时输出事件
  useEffect(() => {
    // @ts-ignore
    if (!window.runtime?.EventsOn) return
    const handler = (ev: any) => {
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
    // @ts-ignore
    window.runtime.EventsOn('xai-output', handler)
    return () => {
      try {
        // @ts-ignore
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
        <div className="ai-console-panel" style={{
          width: 380, flexShrink: 0, alignSelf: 'stretch',
          maxHeight: 'calc(100vh - 80px)',
          margin: '8px 8px 8px 0',
          background: 'var(--gaea-glass-bg, var(--md-sys-color-surface-container))',
          WebkitBackdropFilter: 'blur(24px) saturate(140%)',
          backdropFilter: 'blur(24px) saturate(140%)',
          border: '1px solid var(--md-sys-color-outline-variant)',
          borderRadius: 'var(--md-sys-radius-lg)',
          boxShadow: 'var(--md-sys-elevation-2)',
          display: 'flex', flexDirection: 'column', fontSize: 11,
          fontFamily: 'system-ui, sans-serif',
          overflow: 'hidden',
          animation: 'slideInRight 0.25s cubic-bezier(0.16, 1, 0.3, 1)',
        }}>
          <div style={{
            padding: '8px 12px',
            borderBottom: '1px solid var(--md-sys-color-outline-variant)',
            display: 'flex', justifyContent: 'space-between', alignItems: 'center',
          }}>
            <Space size={6}>
              <span style={{
                width: 6, height: 6, borderRadius: '50%',
                background: logs.length > 0 ? 'var(--md-sys-color-primary)' : 'var(--md-sys-color-text-secondary)',
                display: 'inline-block',
                boxShadow: logs.length > 0 ? '0 0 6px var(--md-sys-color-primary)' : 'none',
              }} />
              <Typography.Text style={{ color: 'var(--gaea-glow)', fontSize: 10, fontWeight: 600, letterSpacing: '0.08em', textShadow: '0 0 10px var(--gaea-glow)' }}>
                AI 控制台
              </Typography.Text>
            </Space>
            <Space size={4}>
              <Typography.Text style={{ color: 'var(--md-sys-color-text-secondary)', fontSize: 9, opacity: 0.6 }}>
                {logs.length}
              </Typography.Text>
              <Button type="text" size="small" onClick={() => setConsoleOpen(false)}
                style={{
                  color: 'var(--md-sys-color-text-secondary)', fontSize: 12, padding: 0,
                  width: 20, height: 20, borderRadius: '50%',
                  display: 'flex', alignItems: 'center', justifyContent: 'center',
                }}>
                ✕
              </Button>
            </Space>
          </div>
          <div ref={logContainerRef} style={{ flex: 1, overflowY: 'scroll', maxHeight: 'calc(100vh - 200px)', padding: '8px 12px' }}>
            {logs.length === 0 ? (
              <div style={{ color: 'var(--md-sys-color-text-secondary)', textAlign: 'center', marginTop: 40, opacity: 0.5 }}>
                <ConsoleSqlOutlined style={{ fontSize: 24, marginBottom: 8 }} />
                <div>等待 AI 调用...</div>
              </div>
            ) : (
              logs.map((l) => {
                const open = expandedLog === l.id
                const tagColor = l.type === 'error' ? 'red' : l.type === 'request' ? 'blue' : l.type === 'response' ? 'green' : 'processing'
                const tagLabel = l.type === 'request' ? 'REQ' : l.type === 'response' ? 'OK' : l.type === 'error' ? 'ERR' : '···'
                const summary = l.type === 'request' ? l.model || ''
                  : l.type === 'response' ? `${(l.length || 0).toLocaleString()} 字`
                  : l.type === 'error' ? l.error?.slice(0, 80) || ''
                  : `+${(l.content?.length || 0)} 字`
                const borderColor = l.type === 'error' ? '#f87171' : l.type === 'request' ? '#60a5fa' : l.type === 'response' ? '#4ade80' : 'transparent'
                return (
                  <div key={l.id}>
                    <div onClick={() => setExpandedLog(open ? null : l.id)}
                      style={{ marginBottom: 1, padding: '3px 6px', cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 4, borderLeft: `2px solid ${borderColor}`, background: open ? 'var(--md-sys-color-surface-container-high)' : 'transparent', borderRadius: 2 }}>
                      <span style={{ color: 'var(--md-sys-color-text-secondary)', fontSize: 9, opacity: 0.5, minWidth: 52 }}>{l.time}</span>
                      <Tag color={tagColor} style={{ fontSize: 10, lineHeight: '16px', margin: 0 }}>{tagLabel}</Tag>
                      <span style={{ color: 'var(--md-sys-color-text-secondary)', fontSize: 10, flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{summary}</span>
                    </div>
                    {open && (
                      <div style={{ padding: '6px 8px', marginBottom: 4, marginLeft: 8, background: 'var(--md-sys-color-surface-container-high)', borderRadius: 4, fontSize: 10, overflow: 'auto', borderLeft: '2px solid var(--md-sys-color-outline-variant)' }}>
                        {l.type === 'request' && <>
                          {l.system && <div style={{ marginBottom: 4 }}><div style={{ color: '#60a5fa', fontWeight: 600, marginBottom: 2 }}>SYSTEM:</div><pre style={{ margin: 0, whiteSpace: 'pre-wrap', color: 'var(--md-sys-color-text-secondary)', fontFamily: 'monospace', fontSize: 9 }}>{l.system}</pre></div>}
                          {l.user && <div><div style={{ color: '#f59e0b', fontWeight: 600, marginBottom: 2 }}>USER:</div><pre style={{ margin: 0, whiteSpace: 'pre-wrap', color: 'var(--md-sys-color-text-secondary)', fontFamily: 'monospace', fontSize: 9 }}>{l.user}</pre></div>}
                        </>}
                        {l.type === 'response' && l.content && <pre style={{ margin: 0, whiteSpace: 'pre-wrap', color: 'var(--md-sys-color-text)', fontFamily: 'monospace', fontSize: 9 }}>{l.content}</pre>}
                        {l.type === 'error' && l.error && <pre style={{ margin: 0, whiteSpace: 'pre-wrap', color: '#f87171', fontFamily: 'monospace', fontSize: 9 }}>{l.error}</pre>}
                        {l.type === 'chunk' && l.content && <pre style={{ margin: 0, whiteSpace: 'pre-wrap', color: 'var(--md-sys-color-text)', fontFamily: 'monospace', fontSize: 9 }}>{l.content}</pre>}
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
        <Button
          onClick={() => setConsoleOpen(true)}
          style={{
            position: 'fixed', right: 12, top: 56, zIndex: 1000,
            width: 28, height: 28, borderRadius: '50%',
            background: 'var(--md-sys-color-surface-container)',
            border: '1px solid var(--md-sys-color-outline-variant)',
            color: 'var(--md-sys-color-text-secondary)',
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            boxShadow: 'var(--md-sys-elevation-1)',
          }}
          title="AI 控制台"
        >
          <ConsoleSqlOutlined style={{ fontSize: 12 }} />
        </Button>
      )}
    </>
  )
}
