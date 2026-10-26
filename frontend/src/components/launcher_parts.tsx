// launcher_parts.tsx — 首页启动器展示件与数据源。
// 方向一「输入即首页」（2026-10-26 拍板）后本文件随首页六区退役大幅瘦身：
// 遥测轮询（3s）/记忆脉搏/写作进度环随状态卡排与侧列退役——内核状态归壳层
// 恒显遥测轨与运行监测板块，写作进度归小说板块；useLauncherData 只留首页
// 仍然消费的两源（recentFiles 订阅单源 + 最近会话挂载一次读）。
// 纯展示件：SectionLabel/ChatBubble/TaskInboxEntry/SessionList。
// 主文件以命名导入回接；外部消费（TasksFirstHome）经 ModuleLauncher.tsx 的
// SessionList/LauncherData re-export 通道，导入路径不变。
import React, { useState, useEffect, useMemo, useSyncExternalStore } from 'react'
import { RobotOutlined, UserOutlined } from '@ant-design/icons'
import { type ShellSpace } from '../boards/space'
import { loadRecentFiles, subscribeRecentFiles } from '../gaea/lib/recentFiles'
import type { AtEntry } from '../gaea/lib/types'
import { useT, type Translator } from '../gaea/lib/i18n'
import { app } from '../gaea/lib/bridge'

// ── 会话的最小类型（对齐 wails 生成的 d.ts，避免引入重型类型）──
interface SessionLite {
  title?: string
  preview?: string
  turns?: number
  modTime?: number
  current?: boolean
}

/** 相对时间（unix ms → 「刚刚 / N 分钟前 / N 小时前 / N 天前」） */
function fmtRel(ms: number, t: Translator): string {
  if (!ms) return '—'
  const diff = Date.now() - ms
  const min = Math.floor(diff / 60000)
  if (min < 1) return t('shell.launcher.fmtJustNow')
  if (min < 60) return t('shell.launcher.fmtMin', { n: min })
  const h = Math.floor(min / 60)
  if (h < 24) return t('shell.launcher.fmtHour', { n: h })
  const d = Math.floor(h / 24)
  if (d < 30) return t('shell.launcher.fmtDay', { n: d })
  return new Date(ms).toLocaleDateString()
}

// ════════════════════════════════════════════════════════════════════
//  排版原语：节标 / 气泡
// ════════════════════════════════════════════════════════════════════

/** 节标：宽字距小标 + 渐隐细线（仪器面板式节眉，不带卡片） */
const SectionLabel: React.FC<{
  icon: React.ReactNode
  title: string
  sub?: string
}> = ({ icon, title, sub }) => (
  <header className="ml-eyebrow">
    <span className="ml-eyebrow-icon" aria-hidden="true">{icon}</span>
    <span className="ml-eyebrow-title">{title}</span>
    <span className="ml-eyebrow-rule" aria-hidden="true" />
    {sub && <span className="ml-eyebrow-sub">{sub}</span>}
  </header>
)

/** 语音/对话气泡 */
const ChatBubble: React.FC<{ role: 'user' | 'assistant'; text: string }> = ({ role, text }) => {
  const isUser = role === 'user'
  return (
    <div className={`ml-bubble-row ${isUser ? 'ml-bubble-user' : 'ml-bubble-ai'}`}>
      {!isUser && (
        <div className="ml-avatar ml-avatar-ai"><RobotOutlined /></div>
      )}
      <div className="ml-bubble">{text}</div>
      {isUser && (
        <div className="ml-avatar ml-avatar-user"><UserOutlined /></div>
      )}
    </div>
  )
}

/**
 * 任务收件箱挂点内容（7.3-1）：待处理计数 + 「打开收件箱」。
 * 计数取 GaeaTaskInboxList(space) 的 pending 数——挂载时一次读（与
 * desk-recent-docs 同款首屏单次读）；inboxTick 在面板收起时 +1，用户动作
 * 触发补一次重读（判据④：零轮询零定时器，读写均由用户动作触发）。
 * row=true 为首页待办行的横排形态（方向一：状态收为一行），默认纵排。
 */
const TaskInboxEntry: React.FC<{
  /** 列表口径：壳层空间或 ''（首页合一 v4.475：首页挂点传 '' = 跨空间全量） */
  space: ShellSpace | ''
  tick: number
  onOpen: () => void
  row?: boolean
}> = ({ space, tick, onOpen, row }) => {
  const t = useT()
  const [pending, setPending] = useState<number | null>(null)
  useEffect(() => {
    let alive = true
    app
      .GaeaTaskInboxList(space)
      .then((list: unknown) => {
        if (!alive) return
        const rows = ((list ?? []) as { status?: string }[]).filter((x) => x.status === 'pending').length
        setPending(rows)
      })
      .catch(() => { if (alive) setPending(null) })
    return () => { alive = false }
  }, [space, tick])
  return (
    <div className="ml-memory" style={row ? { flexDirection: 'row', alignItems: 'center', gap: 14 } : undefined}>
      <span className="ml-krow-strong">
        {pending === null ? '—' : t('tasks.inbox.pendingCount', { n: pending })}
      </span>
      {/* 打开按钮：中性 hairline 胶囊（v11：去金点睛滥用，hover 才点亮） */}
      <button
        type="button"
        onClick={onOpen}
        data-testid="task-inbox-open-btn"
        className="ml-inline-cta"
        style={{ alignSelf: row ? 'center' : 'flex-start' }}
      >
        {t('tasks.inbox.open')}
      </button>
    </div>
  )
}

/** 会话列表内容（首页「最近」区会话行 / 任务优先首页共用；点行进聊天板块续上现场） */
export const SessionList: React.FC<{ sessions: SessionLite[]; onOpen: () => void }> = ({ sessions, onOpen }) => {
  const t = useT()
  if (sessions.length === 0) return <div className="ml-panel-empty">{t('shell.launcher.noSessions')}</div>
  return (
    <ul className="ml-sess">
      {sessions.map((s, i) => (
        <li key={s.modTime ?? i}>
          <button type="button" className="ml-sess-item" onClick={onOpen} title={s.preview || s.title || ''}>
            <span className="ml-sess-name">{s.title || s.preview || t('shell.launcher.unnamed')}</span>
            <span className="ml-sess-meta">{t('shell.launcher.sessionTurns', { turns: s.turns ?? 0, time: fmtRel(s.modTime || 0, t) })}</span>
          </button>
        </li>
      ))}
    </ul>
  )
}

/** 首页共享数据（useLauncherData 一次拉取，两变体各自选用） */
export interface LauncherData {
  recentFiles: AtEntry[]
  sessions: SessionLite[]
}

function useLauncherData(): LauncherData {
  // 最近文档（localStorage 单源，零新 binding）
  // v4.349：订阅单源而非挂载时读一次——本页在 keepAlive（visitedPages）下常驻，
  // 原先「在办公打开文档 → 回首页」看到的仍是首次挂载快照。
  // 方向一（2026-10-26）：首页只答「现在开始做什么」，最近行收 4 条。
  const recentFilesRaw = useSyncExternalStore(subscribeRecentFiles, loadRecentFiles, () => [])
  const recentFiles = useMemo(() => recentFilesRaw.slice(0, 4), [recentFilesRaw])

  // 最近会话
  const [sessions, setSessions] = useState<SessionLite[]>([])
  useEffect(() => {
    let alive = true
    const load = async () => {
      try {
        const list = await app.ListSessions()
        if (!alive) return
        const rows = (list || [])
          .filter((s) => !s.current)
          .sort((a, b) => (b.modTime || 0) - (a.modTime || 0))
          .slice(0, 3)
        setSessions(rows as SessionLite[])
      } catch (_) { /* 静默 */ }
    }
    load()
    return () => { alive = false }
  }, [])

  return { recentFiles, sessions }
}

export { SectionLabel, ChatBubble, TaskInboxEntry, useLauncherData }
