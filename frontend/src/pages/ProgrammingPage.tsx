// ProgrammingPage.tsx — 编程板块「DeepSeek Harness Web 进程管理」
// 启动/停止 dsh web（默认 http://127.0.0.1:3080）、打开编程工作台、实时状态轮询。
// 数据源：CoreB.GetProgrammingWebStatus/StartProgrammingWeb/StopProgrammingWeb（wailsjsCompat）。
import React, { useEffect, useState, useCallback } from 'react'
import { Button, Tooltip, message } from 'antd'
import {
  CodeOutlined, PlayCircleOutlined, StopOutlined,
  LinkOutlined, FolderOpenOutlined, CheckCircleOutlined,
  ClockCircleOutlined, ThunderboltOutlined,
} from '@ant-design/icons'
import * as App from '../../src/wailsjsCompat'
import './programming-page.css'

/** 与 Go 侧 map[string]interface{} 返回契约一致（见 internal/app/programming_web.go） */
interface ProgrammingWebStatus {
  running: boolean
  owned: boolean
  pid: number
  url: string
  root: string
}

const ProgrammingPage: React.FC = () => {
  const [status, setStatus] = useState<ProgrammingWebStatus | null>(null)
  const [busy, setBusy] = useState<'start' | 'stop' | null>(null)
  const [error, setError] = useState('')

  const refresh = useCallback(async () => {
    try {
      const s = (await App.GetProgrammingWebStatus()) as ProgrammingWebStatus
      setStatus(s)
    } catch { /* 后端未就绪时静默，等待下一次轮询 */ }
  }, [])

  useEffect(() => {
    void refresh()
    const t = window.setInterval(refresh, 3000)
    return () => window.clearInterval(t)
  }, [refresh])

  const handleStart = async () => {
    setBusy('start')
    setError('')
    try {
      await App.StartProgrammingWeb()
      message.success('DeepSeek Harness Web 已启动')
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setBusy(null)
      void refresh()
    }
  }

  const handleStop = async () => {
    setBusy('stop')
    setError('')
    try {
      await App.StopProgrammingWeb()
      message.success('已停止 gaea 自启的 Harness Web 实例')
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setBusy(null)
      void refresh()
    }
  }

  const handleOpen = () => {
    if (!status?.url) return
    if (window.runtime?.BrowserOpenURL) {
      window.runtime.BrowserOpenURL(status.url)
    } else {
      window.open(status.url, '_blank', 'noopener')
    }
  }

  const running = !!status?.running
  const owned = !!status?.owned

  return (
    <div className="prog">
      <div className="prog-shell">
        <section className="prog-hero v3-card v3-rise" aria-label="编程工作台">
          <div className="prog-hero-icon" aria-hidden="true"><CodeOutlined /></div>
          <div className="prog-hero-copy">
            <h1>编程工作台</h1>
            <p>DeepSeek Harness Web 进程管理 — 启动、停止并打开本地编程工作台。</p>
          </div>
          <span className={`prog-badge ${running ? 'is-on' : ''}`}>
            <span className="prog-badge-dot" aria-hidden="true" />
            {running ? '服务运行中' : '服务未启动'}
          </span>
        </section>

        <section className="prog-panel v3-panel v3-rise v3-rise-2" aria-label="Harness Web 状态">
          <div className="prog-panel-head">
            <ThunderboltOutlined aria-hidden="true" />
            <span>Harness Web 状态</span>
          </div>
          <div className="prog-body">
            <div className="prog-rows">
              <div className="prog-row">
                <span className="prog-key"><LinkOutlined aria-hidden="true" />访问地址</span>
                <code className="prog-val">{status?.url ?? '—'}</code>
              </div>
              <div className="prog-row">
                <span className="prog-key"><FolderOpenOutlined aria-hidden="true" />Harness 目录</span>
                <code className="prog-val prog-root">{status?.root ?? '—'}</code>
              </div>
              <div className="prog-row">
                <span className="prog-key"><CheckCircleOutlined aria-hidden="true" />实例归属</span>
                <span className="prog-val">
                  {status == null ? '检测中…' : owned ? `gaea 自启（pid ${status.pid}）` : running ? '外部实例' : '无'}
                </span>
              </div>
              <div className="prog-row">
                <span className="prog-key"><ClockCircleOutlined aria-hidden="true" />轮询</span>
                <span className="prog-val">每 3 秒自动刷新</span>
              </div>
            </div>

            <div className="prog-actions">
              <Tooltip title={running ? 'Harness Web 已在服务' : '在 Harness 目录启动 pnpm dsh web'}>
                <Button
                  type="primary"
                  icon={<PlayCircleOutlined />}
                  loading={busy === 'start'}
                  disabled={running}
                  onClick={() => void handleStart()}
                >
                  启动
                </Button>
              </Tooltip>
              <Tooltip title={owned ? '停止 gaea 自启的实例' : (running ? '外部实例，为避免误杀请手动停止' : '当前无 gaea 自启实例')}>
                <Button
                  icon={<StopOutlined />}
                  danger
                  loading={busy === 'stop'}
                  disabled={!owned}
                  onClick={() => void handleStop()}
                >
                  停止
                </Button>
              </Tooltip>
              <Button icon={<LinkOutlined />} disabled={!running} onClick={handleOpen}>
                打开编程工作台
              </Button>
            </div>

            {error && <div className="prog-error" role="alert">{error}</div>}
            <p className="prog-hint">
              首次使用前需在 Harness 目录执行过一次 <code>pnpm install</code> 与 <code>pnpm run build</code>；
              启动日志写入系统临时目录 <code>gaea-dsh-web.log</code>。
            </p>
          </div>
        </section>
      </div>
    </div>
  )
}

export default ProgrammingPage
