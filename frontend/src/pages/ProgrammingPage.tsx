// ProgrammingPage.tsx — 编程板块「DeepSeek Harness 编程工作台」
//
// 设计目标：桌面端直接使用 DeepSeek Harness Web——服务运行时把
// http://127.0.0.1:3080 以 iframe 内嵌在 gaea 窗口内（同一桌面应用里
// 打开 web，无需跳出到浏览器）；未运行时给出启动/状态/前置条件的
// 引导视图。数据源：CoreB.GetProgrammingWebStatus / StartProgrammingWeb /
// StopProgrammingWeb（wailsjsCompat）。
import React, { useEffect, useState, useCallback } from 'react'
import { Button, Tooltip, message, Spin } from 'antd'
import {
  CodeOutlined, PlayCircleOutlined, StopOutlined,
  LinkOutlined, FolderOpenOutlined, ClockCircleOutlined,
  ThunderboltOutlined, ReloadOutlined, GlobalOutlined,
  FileTextOutlined, ApiOutlined, CheckCircleOutlined,
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
  /** 自启日志路径（Go 侧返回；旧二进制缺失时前端兜底显示文件名） */
  log?: string
}

const POLL_MS = 3000

const ProgrammingPage: React.FC = () => {
  const [status, setStatus] = useState<ProgrammingWebStatus | null>(null)
  const [busy, setBusy] = useState<'start' | 'stop' | null>(null)
  const [error, setError] = useState('')
  const [frameKey, setFrameKey] = useState(0)
  const [frameLoaded, setFrameLoaded] = useState(false)

  const refresh = useCallback(async () => {
    try {
      const s = (await App.GetProgrammingWebStatus()) as ProgrammingWebStatus
      setStatus(s)
    } catch {
      /* 后端未就绪时静默，等待下一次轮询 */
    }
  }, [])

  useEffect(() => {
    void refresh()
    const t = window.setInterval(refresh, POLL_MS)
    return () => window.clearInterval(t)
  }, [refresh])

  const handleStart = async () => {
    setBusy('start')
    setError('')
    try {
      await App.StartProgrammingWeb()
      message.success('DeepSeek Harness Web 已启动')
      setFrameLoaded(false)
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

  /** 桌面内嵌工作台 / 启动视图共用的外部浏览器打开 */
  const handleOpenBrowser = () => {
    if (!status?.url) return
    if (window.runtime?.BrowserOpenURL) {
      window.runtime.BrowserOpenURL(status.url)
    } else {
      window.open(status.url, '_blank', 'noopener')
    }
  }

  /** 打开 Harness 目录（Wails BrowserOpenURL 支持 file:// 目录 → 系统资源管理器） */
  const handleOpenFolder = () => {
    const root = status?.root
    if (!root) return
    const fileUrl = 'file:///' + root.replace(/\\/g, '/').replace(/^\/+/, '')
    if (window.runtime?.BrowserOpenURL) {
      window.runtime.BrowserOpenURL(fileUrl)
    } else {
      window.open(fileUrl, '_blank', 'noopener')
    }
  }

  const handleReload = () => {
    setFrameLoaded(false)
    setFrameKey((k) => k + 1)
  }

  const running = !!status?.running
  const owned = !!status?.owned

  // ── 运行中：桌面内嵌工作台（核心视图） ───────────────────────────
  if (running) {
    return (
      <div className="prog">
        <div className="prog-workbench">
          <div className="prog-frame-bar">
            <span className="prog-badge is-on" title="服务运行中，工作台已内嵌到桌面窗口">
              <span className="prog-badge-dot" aria-hidden="true" />
              Harness Web 运行中
            </span>
            <code className="prog-url-chip" title={status?.url}>{status?.url}</code>
            <span className="prog-frame-spacer" aria-hidden="true" />
            <Tooltip title="重新加载工作台">
              <Button
                type="text"
                size="small"
                icon={<ReloadOutlined />}
                onClick={handleReload}
                aria-label="重新加载工作台"
              />
            </Tooltip>
            <Tooltip title="在浏览器中打开">
              <Button
                type="text"
                size="small"
                icon={<GlobalOutlined />}
                onClick={handleOpenBrowser}
                aria-label="在浏览器中打开"
              />
            </Tooltip>
            <Tooltip title={owned ? '停止 gaea 自启的实例' : '外部实例，为避免误杀请手动停止'}>
              <Button
                type="text"
                danger
                size="small"
                icon={<StopOutlined />}
                loading={busy === 'stop'}
                disabled={!owned}
                onClick={() => void handleStop()}
                aria-label="停止服务"
              />
            </Tooltip>
          </div>

          <div className="prog-frame-wrap">
            {!frameLoaded && (
              <div className="prog-frame-overlay" role="status">
                <Spin size="large" />
                <span className="prog-frame-overlay-text">正在加载 DeepSeek Harness 工作台…</span>
              </div>
            )}
            <iframe
              key={frameKey}
              src={status?.url}
              title="DeepSeek Harness 编程工作台"
              className="prog-frame"
              onLoad={() => setFrameLoaded(true)}
              allow="clipboard-read; clipboard-write; fullscreen; autoplay; camera; microphone"
            />
          </div>
        </div>
      </div>
    )
  }

  // ── 未运行：启动引导视图 ────────────────────────────────────────
  return (
    <div className="prog">
      <div className="prog-shell">
        <section className="prog-hero v3-card v3-rise" aria-label="编程工作台">
          <div className="prog-hero-icon" aria-hidden="true"><CodeOutlined /></div>
          <div className="prog-hero-copy">
            <h1>编程工作台</h1>
            <p>DeepSeek Harness Web 进程管理 — 启动后直接在桌面窗口内使用编程工作台，无需跳出浏览器。</p>
          </div>
          <span className={`prog-badge ${running ? 'is-on' : ''}`}>
            <span className="prog-badge-dot" aria-hidden="true" />
            {status == null ? '检测中…' : running ? '服务运行中' : '服务未启动'}
          </span>
        </section>

        <section className="prog-launch v3-panel v3-rise v3-rise-1" aria-label="启动 Harness Web">
          <div className="prog-launch-main">
            <div className="prog-launch-orb" aria-hidden="true"><ThunderboltOutlined /></div>
            <div className="prog-launch-copy">
              <h2>一键启动，桌面内嵌</h2>
              <p>
                在 Harness 目录执行 <code>pnpm dsh web</code> 并等待端口就绪后，
                工作台会直接嵌入当前窗口。
              </p>
            </div>
            <div className="prog-launch-actions">
              <Tooltip title="在 Harness 目录启动 pnpm dsh web（已运行幂等）">
                <Button
                  type="primary"
                  size="large"
                  icon={<PlayCircleOutlined />}
                  loading={busy === 'start'}
                  disabled={running}
                  onClick={() => void handleStart()}
                >
                  启动编程工作台
                </Button>
              </Tooltip>
              <Tooltip title="在资源管理器中打开 Harness 目录">
                <Button
                  size="large"
                  icon={<FolderOpenOutlined />}
                  disabled={!status?.root}
                  onClick={handleOpenFolder}
                >
                  打开 Harness 目录
                </Button>
              </Tooltip>
              <Button size="large" icon={<GlobalOutlined />} disabled={!running} onClick={handleOpenBrowser}>
                在浏览器中打开
              </Button>
            </div>
          </div>

          <div className="prog-launch-side">
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
                <span className="prog-key"><ApiOutlined aria-hidden="true" />服务端口</span>
                <code className="prog-val">3080</code>
              </div>
              <div className="prog-row">
                <span className="prog-key"><FileTextOutlined aria-hidden="true" />启动日志</span>
                <code className="prog-val prog-root">{status?.log ?? 'gaea-dsh-web.log'}</code>
              </div>
              <div className="prog-row">
                <span className="prog-key"><CheckCircleOutlined aria-hidden="true" />实例归属</span>
                <span className="prog-val">
                  {status == null ? '检测中…' : owned ? `gaea 自启（pid ${status.pid}）` : running ? '外部实例' : '无'}
                </span>
              </div>
              <div className="prog-row">
                <span className="prog-key"><ClockCircleOutlined aria-hidden="true" />状态轮询</span>
                <span className="prog-val">每 3 秒自动刷新</span>
              </div>
            </div>
          </div>
        </section>

        {error && <div className="prog-error v3-rise v3-rise-2" role="alert">{error}</div>}

        <section className="prog-hint-card v3-card v3-rise v3-rise-2" aria-label="使用说明">
          <span className="prog-hint-icon" aria-hidden="true"><ThunderboltOutlined /></span>
          <p className="prog-hint">
            首次使用前需在 Harness 目录执行过一次 <code>pnpm install</code> 与 <code>pnpm run build</code>；
            启动日志写入系统临时目录（默认 <code>gaea-dsh-web.log</code>）。
            若 3080 端口已被其他进程占用，gaea 会明确提示而不会抢端口。
          </p>
        </section>
      </div>
    </div>
  )
}

export default ProgrammingPage
