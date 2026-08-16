// ProgrammingPage.tsx — 编程板块「DeepSeek Harness 编程工作台」
//
// 设计目标：桌面端直接使用 DeepSeek Harness Web——服务运行时把
// http://127.0.0.1:3080 以 iframe 内嵌在 gaea 窗口内（同一桌面应用里
// 打开 web，无需跳出到浏览器）；未运行时给出启动引导视图：一键启动 +
// 真实前置条件检查清单（GetProgrammingWebPreflight）+ 自启日志查看
// （ProgrammingWebLogTail）+ 状态行。数据源：CoreB.GetProgrammingWebStatus /
// StartProgrammingWeb / StopProgrammingWeb（wailsjsCompat）。
import React, { useEffect, useState, useCallback, useRef } from 'react'
import { Button, Tooltip, message, Spin } from 'antd'
import {
  CodeOutlined, PlayCircleOutlined, StopOutlined,
  LinkOutlined, FolderOpenOutlined, ClockCircleOutlined,
  ThunderboltOutlined, ReloadOutlined, GlobalOutlined,
  FileTextOutlined, ApiOutlined, CheckCircleOutlined,
  CloseCircleOutlined, DownOutlined, UpOutlined,
  FileSearchOutlined, SyncOutlined, CheckOutlined, WarningOutlined,
} from '@ant-design/icons'
import * as App from '../../src/wailsjsCompat'
import type {
  ProgrammingWebStatus, ProgrammingWebPreflight, ProgrammingWebLogTail,
} from '../gaea/lib/types'
import './programming-page.css'

const POLL_MS = 3000
const LOG_TAIL_LINES = 100

/** 秒 → "X 小时 Y 分" / "X 分 Y 秒" / "X 秒" */
function formatUptime(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds <= 0) return '刚刚启动'
  const s = Math.floor(seconds)
  if (s < 60) return `${s} 秒`
  const m = Math.floor(s / 60)
  if (m < 60) return `${m} 分 ${s % 60} 秒`
  return `${Math.floor(m / 60)} 小时 ${m % 60} 分`
}

interface PreflightItem {
  key: keyof ProgrammingWebPreflight
  label: string
  desc: string
}

/** 前置条件清单（与 Go GetProgrammingWebPreflight 字段一一对应） */
const PREFLIGHT_ITEMS: PreflightItem[] = [
  { key: 'harness_valid', label: 'Harness 目录有效', desc: '根目录存在 package.json（GAEA_HARNESS_DIR 可指定）' },
  { key: 'pnpm_found', label: 'pnpm 可用', desc: 'Node.js ≥22 + corepack（corepack enable）' },
  { key: 'deps_ready', label: '依赖已安装', desc: 'node_modules 存在（首次使用先执行 pnpm install）' },
  { key: 'build_ready', label: 'Web 构建产物就绪', desc: 'apps/web/dist 已构建（pnpm run build 生成）' },
  { key: 'port_free', label: '端口 3080 空闲', desc: '被其他进程占用时需手动停止后再启动' },
]

const ProgrammingPage: React.FC = () => {
  const [status, setStatus] = useState<ProgrammingWebStatus | null>(null)
  const [preflight, setPreflight] = useState<ProgrammingWebPreflight | null>(null)
  const [log, setLog] = useState<ProgrammingWebLogTail | null>(null)
  const [logExpanded, setLogExpanded] = useState(false)
  const [busy, setBusy] = useState<'start' | 'stop' | 'preflight' | 'log' | null>(null)
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

  const refreshPreflight = useCallback(async () => {
    setBusy((b) => b ?? 'preflight')
    try {
      setPreflight((await App.GetProgrammingWebPreflight()) as ProgrammingWebPreflight)
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setBusy((b) => (b === 'preflight' ? null : b))
    }
  }, [])

  const refreshLog = useCallback(async () => {
    setBusy((b) => b ?? 'log')
    try {
      setLog((await App.ProgrammingWebLogTail(LOG_TAIL_LINES)) as ProgrammingWebLogTail)
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setBusy((b) => (b === 'log' ? null : b))
    }
  }, [])

  useEffect(() => {
    void refresh()
    void refreshPreflight()
    const t = window.setInterval(refresh, POLL_MS)
    return () => window.clearInterval(t)
  }, [refresh, refreshPreflight])

  // 服务停止后端口重新空闲：预检 port_free 状态需要跟随刷新
  // （仅「运行中 → 停止」方向，挂载/轮询不重复拉取）。
  const running = !!status?.running
  const prevRunning = useRef(running)
  useEffect(() => {
    if (prevRunning.current && !running) void refreshPreflight()
    prevRunning.current = running
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [running])

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
      void refreshPreflight()
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

  const toggleLog = () => {
    if (!logExpanded) void refreshLog()
    setLogExpanded((v) => !v)
  }

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
            {owned ? (
              <span className="prog-uptime-chip" title="gaea 自启实例运行时长">
                <ClockCircleOutlined aria-hidden="true" />
                {formatUptime(status?.uptime_s ?? 0)}
              </span>
            ) : (
              <span className="prog-uptime-chip is-external" title="非 gaea 自启实例，停止需手动操作">
                <WarningOutlined aria-hidden="true" />
                外部实例
              </span>
            )}
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
  const allReady = preflight?.all_ready ?? false
  const checking = preflight == null || busy === 'preflight'

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
                前置条件全部就绪后，在 Harness 目录执行 <code>pnpm dsh web</code> 并等待端口
                就绪，工作台会直接嵌入当前窗口。
              </p>
            </div>
            <div className="prog-launch-actions">
              <Tooltip title={allReady ? '在 Harness 目录启动 pnpm dsh web（已运行幂等）' : '前置条件未全部就绪，请先按下方清单处理'}>
                <Button
                  type="primary"
                  size="large"
                  icon={<PlayCircleOutlined />}
                  loading={busy === 'start'}
                  disabled={running || !allReady}
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

        {/* 启动前置条件：真实逐项检查（替代静态使用说明） */}
        <section className="prog-preflight v3-panel v3-rise v3-rise-2" aria-label="启动前置条件">
          <div className="prog-panel-head">
            <span className="prog-panel-title">
              <FileSearchOutlined aria-hidden="true" />
              启动前置条件
            </span>
            <Button
              size="small"
              icon={<SyncOutlined />}
              loading={busy === 'preflight'}
              onClick={() => void refreshPreflight()}
              aria-label="重新检查前置条件"
            >
              重新检查
            </Button>
          </div>

          {checking ? (
            <div className="prog-preflight-loading" role="status">
              <Spin size="small" />
              <span>正在检查 Harness 环境…</span>
            </div>
          ) : (
            <>
              <ul className="prog-preflight-list">
                {PREFLIGHT_ITEMS.map((item) => {
                  const ok = preflight?.[item.key] === true
                  return (
                    <li key={item.key} className={`prog-preflight-item ${ok ? 'is-ok' : 'is-fail'}`}>
                      <span className="prog-preflight-icon" aria-hidden="true">
                        {ok ? <CheckCircleOutlined /> : <CloseCircleOutlined />}
                      </span>
                      <span className="prog-preflight-copy">
                        <span className="prog-preflight-label">{item.label}</span>
                        <span className="prog-preflight-desc">{item.desc}</span>
                      </span>
                    </li>
                  )
                })}
              </ul>
              <div className={`prog-preflight-foot ${allReady ? 'is-ready' : 'is-pending'}`}>
                {allReady ? (
                  <><CheckOutlined aria-hidden="true" /> 全部就绪，可一键启动</>
                ) : (
                  <><WarningOutlined aria-hidden="true" /> 存在未满足项，按提示处理后「重新检查」</>
                )}
              </div>
            </>
          )}
        </section>

        {/* 启动日志：可展开查看自启日志尾部 */}
        <section className="prog-log v3-panel v3-rise v3-rise-2" aria-label="启动日志">
          <button type="button" className="prog-panel-head prog-log-toggle" onClick={toggleLog} aria-expanded={logExpanded}>
            <span className="prog-panel-title">
              <FileTextOutlined aria-hidden="true" />
              启动日志
              <code className="prog-log-path">{status?.log ?? 'gaea-dsh-web.log'}</code>
            </span>
            <span className="prog-log-toggle-actions">
              {logExpanded && (
                <Button
                  size="small"
                  icon={<ReloadOutlined />}
                  loading={busy === 'log'}
                  onClick={(e) => { e.stopPropagation(); void refreshLog() }}
                  aria-label="刷新日志"
                />
              )}
              <span className="prog-log-chevron" aria-hidden="true">
                {logExpanded ? <UpOutlined /> : <DownOutlined />}
              </span>
            </span>
          </button>
          {logExpanded && (
            <div className="prog-log-body">
              {log == null ? (
                <div className="prog-log-empty" role="status"><Spin size="small" /> 读取日志…</div>
              ) : !log.exists ? (
                <div className="prog-log-empty">{log.error}</div>
              ) : (
                <pre className="prog-log-pre" aria-label="日志内容">
                  {log.lines.length ? log.lines.join('\n') : '（日志为空）'}
                </pre>
              )}
            </div>
          )}
        </section>
      </div>
    </div>
  )
}

export default ProgrammingPage
