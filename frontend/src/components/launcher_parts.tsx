// launcher_parts.tsx — 首页启动器展示件与数据源。
// §8.3 DeepSeek 化（2026-10-27）：最近文档行随「简洁」拍板退役（办公板块自有
// 最近面），useLauncherData 只留最近会话挂载一次读；TaskInboxEntry（待办行
// 挂点内容）随待办行退役——收件箱面板入口移命令面板空态（OPEN_TASK_INBOX）。
// 纯展示件：SectionLabel/ChatBubble/SessionList。
import React, { useState, useEffect } from 'react'
import { RobotOutlined, UserOutlined } from '@ant-design/icons'
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

/** 会话列表内容（首页「最近」区会话行；点行进聊天板块续上现场） */
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

/** 首页共享数据（useLauncherData 一次拉取） */
export interface LauncherData {
  sessions: SessionLite[]
}

function useLauncherData(): LauncherData {
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

  return { sessions }
}

export { SectionLabel, ChatBubble, useLauncherData }
