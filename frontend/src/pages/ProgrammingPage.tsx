import { useCallback, useEffect, useRef, useState, type ReactNode } from 'react'
import { Button, Spin, Tag } from 'antd'
import {
  ArrowRightOutlined,
  CheckCircleOutlined,
  CheckOutlined,
  CodeOutlined,
  CopyOutlined,
  DesktopOutlined,
  FolderOpenOutlined,
  GithubOutlined,
  GlobalOutlined,
  PlayCircleOutlined,
  ReloadOutlined,
  SafetyCertificateOutlined,
  StopOutlined,
  WarningOutlined,
} from '@ant-design/icons'
import { app } from '../gaea/lib/bridge'
import { openExternal } from '../gaea/lib/bridge/proxy'
import type {
  ProgrammingWebPreflight,
  ProgrammingWebStatus,
} from '../gaea/lib/types'
import './programming-page.css'

export const DSH_VERSION = '0.2.2-rc.1'
export const DSH_REPO_URL = 'https://github.com/deepseek-ai/deepseek-harness#run'
export const DSH_NPM_VERSION_URL = `https://www.npmjs.com/package/@deepseek-ai/dsh/v/${DSH_VERSION}`
export const DSH_WUBI_UPSTREAM_URL = 'https://github.com/mmparko/dsh-purge'
export const REA_VERSION = '6.8.0'
export const REA_TOOL_COUNT = 229
export const REA_REPO_URL = 'https://github.com/morluto/rea'
export const DSH_LOCAL_URL = 'http://dsh.wails.localhost:3091'
export const DSH_INSTALL_COMMAND = `npm install --prefix C:\\AI\\dsh @deepseek-ai/dsh@${DSH_VERSION}`
export const DSH_START_COMMAND = 'C:\\AI\\dsh\\start-web.cmd'

function errText(error: unknown): string {
  if (error instanceof Error) return error.message
  if (typeof error === 'string') return error
  if (error && typeof error === 'object' && 'message' in error) {
    return String((error as { message?: unknown }).message ?? '')
  }
  return ''
}

function preflightIssue(pre: ProgrammingWebPreflight): string {
  if (!pre.node_found) return '未找到 Node.js；DeepSeek Harness 0.2.2 需要 Node.js 24.9 或更高。'
  if (!pre.install_ready) return `未找到隔离安装：${pre.root}`
  if (!pre.profile_ready) return '独立 Web profile 尚未初始化。'
  if (!pre.plugin_ready) return 'Web profile 尚未链接本地 dsh-wubi 插件。'
  if (!pre.rea_ready) return 'REA MCP 注册或 reverse-engineer-anything 技能尚未就绪。'
  if (!pre.isolated) return '安装根、DSH_HOME 或 DSH_BASE 超出了隔离目录。'
  if (!pre.version_ok) return `当前 DSH ${pre.version || '未知'}，适配目标为 ${pre.expected}。`
  if (!pre.port_free) return '端口 3091 已被非 gaea 实例占用。'
  return ''
}

function ExternalLink({
  href,
  children,
  className = '',
}: {
  href: string
  children: ReactNode
  className?: string
}) {
  return (
    <a
      className={`programming-link ${className}`.trim()}
      href={href}
      onClick={(event) => {
        event.preventDefault()
        openExternal(href)
      }}
    >
      <span>{children}</span>
      <ArrowRightOutlined aria-hidden="true" />
    </a>
  )
}

function CopyCommand({ command, label }: { command: string; label: string }) {
  const [copied, setCopied] = useState(false)
  const timerRef = useRef<number | null>(null)

  useEffect(() => () => {
    if (timerRef.current !== null) window.clearTimeout(timerRef.current)
  }, [])

  const copy = useCallback(async () => {
    try {
      let copiedByClipboard = false
      if (navigator.clipboard?.writeText) {
        try {
          await Promise.race([
            navigator.clipboard.writeText(command),
            new Promise<void>((_, reject) => {
              window.setTimeout(() => reject(new Error('clipboard timeout')), 600)
            }),
          ])
          copiedByClipboard = true
        } catch {
          // WebView clipboard permission can be denied; use the local fallback.
        }
      }
      if (!copiedByClipboard) {
        const area = document.createElement('textarea')
        area.value = command
        area.setAttribute('readonly', '')
        area.style.position = 'fixed'
        area.style.opacity = '0'
        document.body.appendChild(area)
        area.select()
        const ok = document.execCommand('copy')
        area.remove()
        if (!ok) throw new Error('copy failed')
      }
      setCopied(true)
      if (timerRef.current !== null) window.clearTimeout(timerRef.current)
      timerRef.current = window.setTimeout(() => setCopied(false), 1600)
    } catch {
      setCopied(false)
    }
  }, [command])

  return (
    <Button
      type="text"
      size="small"
      className="programming-copy"
      icon={copied ? <CheckOutlined /> : <CopyOutlined />}
      onClick={() => void copy()}
      aria-label={copied ? `${label}已复制` : `复制${label}`}
    >
      {copied ? '已复制' : '复制'}
    </Button>
  )
}

function CommandBlock({
  eyebrow,
  command,
  label,
}: {
  eyebrow: string
  command: string
  label: string
}) {
  return (
    <div className="programming-command">
      <div className="programming-command__head">
        <span>{eyebrow}</span>
        <CopyCommand command={command} label={label} />
      </div>
      <code>{command}</code>
    </div>
  )
}

function SetupContent({
  error,
  preflight,
  busy,
  onRetry,
}: {
  error: string
  preflight: ProgrammingWebPreflight | null
  busy: boolean
  onRetry: () => void
}) {
  return (
    <div className="programming-scroll">
      <div className="programming-shell">
        <section className="programming-hero" aria-labelledby="programming-entry-title">
          <div className="programming-eyebrow">
            <CodeOutlined aria-hidden="true" />
            DeepSeek Harness · Web
          </div>
          <h2 id="programming-entry-title">独立编程工作台</h2>
          <p className="programming-lede">
            工作台会直接启动并内嵌 {DSH_VERSION} 网页版。安装、会话和插件补丁全部留在
            <code>C:\AI\dsh</code>，与本机官方桌面版完全分开。
          </p>
          <div className="programming-badges" aria-label="编程入口状态">
            <span className="programming-badge is-ok">
              <CheckCircleOutlined aria-hidden="true" />
              最新版 {DSH_VERSION}
            </span>
            <span className="programming-badge">
              <SafetyCertificateOutlined aria-hidden="true" />
              dsh-wubi 2.0.0 · 已适配
            </span>
            <span className="programming-badge is-ok">
              <CheckCircleOutlined aria-hidden="true" />
              REA MCP {REA_VERSION} · {REA_TOOL_COUNT} tools
            </span>
            <span className="programming-badge">
              <DesktopOutlined aria-hidden="true" />
              桌面版零改动
            </span>
          </div>
          {error && (
            <div className="programming-alert" role="alert">
              <WarningOutlined aria-hidden="true" />
              <span>{error}</span>
              <Button size="small" loading={busy} onClick={onRetry}>重试启动</Button>
            </div>
          )}
        </section>

        <div className="programming-grid">
          <section className="programming-card programming-install" aria-labelledby="programming-install-title">
            <div className="programming-card__head">
              <div>
                <span className="programming-kicker">网页版</span>
                <h3 id="programming-install-title">下载与隔离安装</h3>
              </div>
              <span className="programming-version">v{DSH_VERSION}</span>
            </div>
            <p className="programming-copy-text">
              当前 npm latest 为 {DSH_VERSION}，Node.js 要求 24.9 或更高。命令只安装到 gaea
              专用目录，不覆盖桌面版。
            </p>
            <CommandBlock eyebrow="安装到独立目录" command={DSH_INSTALL_COMMAND} label="安装命令" />
            <CommandBlock eyebrow="手动启动（可选）" command={DSH_START_COMMAND} label="启动命令" />
            <div className="programming-links">
              <ExternalLink href={DSH_REPO_URL} className="is-primary">
                <GithubOutlined aria-hidden="true" />
                打开官方网页版安装说明
              </ExternalLink>
              <ExternalLink href={DSH_NPM_VERSION_URL}>
                <GlobalOutlined aria-hidden="true" />
                查看 {DSH_VERSION} 安装包
              </ExternalLink>
              <ExternalLink href={REA_REPO_URL}>
                <GithubOutlined aria-hidden="true" />
                REA 逆向工程工具
              </ExternalLink>
            </div>
          </section>

          <aside className="programming-card programming-boundary" aria-labelledby="programming-boundary-title">
            <div className="programming-card__head">
              <div>
                <span className="programming-kicker">隔离边界</span>
                <h3 id="programming-boundary-title">两个 DSH 完全分开</h3>
              </div>
              <SafetyCertificateOutlined className="programming-boundary__icon" aria-hidden="true" />
            </div>
            <dl className="programming-paths">
              <div>
                <dt>安装根</dt>
                <dd><code>{preflight?.root || String.raw`C:\AI\dsh`}</code></dd>
              </div>
              <div>
                <dt>Web DSH_HOME</dt>
                <dd><code>{preflight?.home || String.raw`C:\AI\dsh\home`}</code></dd>
              </div>
              <div>
                <dt>DSH_AGENTS_HOME</dt>
                <dd><code>{preflight?.agents_home || String.raw`C:\AI\dsh\home\agents`}</code></dd>
              </div>
              <div>
                <dt>插件仓库</dt>
                <dd><code>C:\AI\wubigrok\clones\dsh-wubi</code></dd>
              </div>
            </dl>
            <div className="programming-isolation-note">
              <FolderOpenOutlined aria-hidden="true" />
              <p>
                <code>DSH_WUBI_ISOLATED_ROOT</code> 已锁定为安装根。启动、应用、还原和停止都不会触碰
                <code>C:\Users\wubi\.dsh</code>、共享的 <code>C:\Users\wubi\.agents</code> 或官方桌面安装目录。
              </p>
            </div>
            <ExternalLink href={DSH_WUBI_UPSTREAM_URL}>
              <GithubOutlined aria-hidden="true" />
              查看原始项目与署名
            </ExternalLink>
          </aside>
        </div>

        <section className="programming-card programming-steps" aria-labelledby="programming-steps-title">
          <div className="programming-card__head">
            <div>
              <span className="programming-kicker">自动连接</span>
              <h3 id="programming-steps-title">进入工作台前会检查</h3>
            </div>
          </div>
          <ol>
            <li>
              <span>01</span>
              <div>
                <strong>运行时</strong>
                <p>Node.js、{DSH_VERSION} 安装文件与独立 profile。</p>
              </div>
            </li>
            <li>
              <span>02</span>
              <div>
                <strong>插件</strong>
                <p>本地 dsh-wubi 2.0.0 已链接，REA MCP 与技能已注册。</p>
              </div>
            </li>
            <li>
              <span>03</span>
              <div>
                <strong>隔离</strong>
                <p>安装根、DSH_HOME、DSH_AGENTS_HOME、DSH_BASE 全部位于 C:\AI\dsh。</p>
              </div>
            </li>
            <li>
              <span>04</span>
              <div>
                <strong>内嵌</strong>
                <p>取得本进程专属访问令牌后，直接在 gaea 窗口加载。</p>
              </div>
            </li>
          </ol>
          <p className="programming-footnote">
            dsh-wubi 为本地维护的第三方插件，保留 dsh-purge 原作者署名；与本机桌面版及 DeepSeek 均无隶属关系。
          </p>
        </section>
      </div>
    </div>
  )
}

export default function ProgrammingPage() {
  const [status, setStatus] = useState<ProgrammingWebStatus | null>(null)
  const [preflight, setPreflight] = useState<ProgrammingWebPreflight | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [frameKey, setFrameKey] = useState(0)
  const bootRef = useRef(false)

  const refreshStatus = useCallback(async () => {
    const next = await app.GetProgrammingWebStatus()
    setStatus(next)
    return next
  }, [])

  const start = useCallback(async () => {
    setBusy(true)
    setError('')
    try {
      const pre = await app.GetProgrammingWebPreflight()
      setPreflight(pre)
      const current = await app.GetProgrammingWebStatus()
      setStatus(current)
      if (current.running) {
        if (!current.owned) setError('端口 3091 已被 gaea 之外的实例占用，未进行接管。')
        return
      }
      const issue = preflightIssue(pre)
      if (issue) {
        setError(issue)
        return
      }
      await app.StartProgrammingWeb()
      const next = await refreshStatus()
      if (!next.running || !next.owned || !next.url.includes('token=')) {
        setError('DSH Web 已启动，但 gaea 尚未取得可用访问入口。')
      }
    } catch (cause) {
      setError(errText(cause) || 'DSH Web 启动失败')
    } finally {
      setBusy(false)
    }
  }, [refreshStatus])

  useEffect(() => {
    if (bootRef.current) return
    bootRef.current = true
    void (async () => {
      try {
        const current = await refreshStatus()
        if (current.running) {
          if (!current.owned) setError('端口 3091 已被非 gaea 实例占用。')
          return
        }
        await start()
      } catch (cause) {
        setError(errText(cause) || '无法读取 DSH Web 状态')
      }
    })()
  }, [refreshStatus, start])

  useEffect(() => {
    const timer = window.setInterval(() => {
      if (!busy) void refreshStatus().catch(() => {})
    }, 3500)
    return () => window.clearInterval(timer)
  }, [busy, refreshStatus])

  const stop = useCallback(async () => {
    setBusy(true)
    setError('')
    try {
      await app.StopProgrammingWeb()
      await refreshStatus()
    } catch (cause) {
      setError(errText(cause) || '停止 DSH Web 失败')
    } finally {
      setBusy(false)
    }
  }, [refreshStatus])

  const live = Boolean(status?.running && status.owned && status.url.includes('token='))
  const statusText = status?.starting
    ? '正在启动'
    : live
      ? `运行中 · ${status?.version || DSH_VERSION}`
      : status?.running
        ? '端口被占用'
        : '准备启动'

  return (
    <main className={`programming-page${live ? ' is-live' : ''}`} aria-labelledby="programming-title">
      <header className="programming-toolbar">
        <div className="programming-title-lockup">
          <span className="programming-title-icon"><CodeOutlined aria-hidden="true" /></span>
          <div>
            <h1 id="programming-title">编程</h1>
            <span className={`programming-runtime-state${live ? ' is-live' : ''}`}>
              <i aria-hidden="true" />
              {statusText}
            </span>
          </div>
        </div>
        <div className="programming-toolbar-actions">
          <Tag bordered={false}>{DSH_VERSION}</Tag>
          {live && (
            <>
              <Button
                size="small"
                icon={<ReloadOutlined />}
                onClick={() => setFrameKey((key) => key + 1)}
              >
                刷新
              </Button>
              <Button
                size="small"
                icon={<GlobalOutlined />}
                onClick={() => openExternal(status?.url || DSH_LOCAL_URL)}
              >
                浏览器打开
              </Button>
            </>
          )}
          {status?.owned ? (
            <Button size="small" danger icon={<StopOutlined />} loading={busy} onClick={() => void stop()}>
              停止
            </Button>
          ) : (
            <Button
              size="small"
              type="primary"
              icon={<PlayCircleOutlined />}
              loading={busy || status?.starting}
              onClick={() => void start()}
            >
              启动
            </Button>
          )}
        </div>
      </header>

      {live ? (
        <div className="programming-frame-shell">
          <iframe
            key={frameKey}
            className="programming-frame"
            title="DeepSeek Harness Web"
            src={status?.url}
            sandbox="allow-scripts allow-same-origin allow-forms allow-popups allow-downloads allow-modals allow-popups-to-escape-sandbox"
            allow="clipboard-read; clipboard-write"
            referrerPolicy="no-referrer"
          />
        </div>
      ) : busy || status?.starting ? (
        <div className="programming-loading" role="status" aria-live="polite">
          <Spin size="large" />
          <strong>正在启动隔离的 DeepSeek Harness Web</strong>
          <span>正在校验 0.2.2-rc.1、dsh-wubi 与独立数据目录，并为当前进程取得访问令牌。</span>
        </div>
      ) : (
        <SetupContent error={error} preflight={preflight} busy={busy} onRetry={() => void start()} />
      )}
    </main>
  )
}
