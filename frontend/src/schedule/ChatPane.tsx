/**
 * schedule/ChatPane.tsx — 进度计划板块左栏 AI 对话（v4.119.0 刀10）
 *
 * 与办公板块 GaeaApp 共享同一会话 store（gaea/lib/store 模块单例）：
 * 左栏里聊的就是办公那条工作线程，schedule_* 工具的审批/回执/证据卡
 * 全部同源。事件绑定经 store.ensureEventsBound 恰好一次（多宿主不双订阅）。
 * schedule_apply 成功回执 → notifyScheduleFileChanged 即时回读（与 App.tsx
 * 刀5 接线同款，绕过 15s 轻扫）。
 */
import React, { useEffect, useRef } from 'react'
import { LocaleProvider, useT } from '../gaea/lib/i18n'
import { Transcript } from '../gaea/components/Transcript'
import { Composer } from '../gaea/components/Composer'
import { ApprovalModal } from '../gaea/components/ApprovalModal'
import { AskCard } from '../gaea/components/AskCard'
import { useController } from '../gaea/lib/store'
import { notifyScheduleFileChanged } from './store'

/** schedule_apply 回执 → 板块即时回读（监听 items 里工具卡的 status 跃迁） */
function useScheduleApplyNotify(): void {
  const { state } = useController()
  const seen = useRef(new Map<string, string>())
  useEffect(() => {
    for (const it of state.items) {
      if (it.kind !== 'tool') continue
      const prev = seen.current.get(it.id)
      if (prev === it.status) continue
      seen.current.set(it.id, it.status)
      if (it.name === 'schedule_apply' && it.status === 'done') {
        notifyScheduleFileChanged()
      }
    }
  }, [state.items])
}

const ChatPaneInner: React.FC = () => {
  const t = useT()
  const ctrl = useController()
  const state = ctrl.state
  useScheduleApplyNotify()

  return (
    <div className="sched-chat">
      <div className="sched-chat-head">
        <span className="sched-chat-title">{t('schedChat.title')}</span>
        {state.running && (
          <span className="sched-chat-running">
            <span className="sched-chat-running-dot" aria-hidden="true" />
            {t('schedChat.running')}
          </span>
        )}
        <button
          className="sched-chat-new"
          title={t('schedChat.newTip')}
          onClick={() => { void ctrl.newSession() }}
        >
          + {t('schedChat.new')}
        </button>
      </div>
      {/* .transcript 自带 flex:1/min-height:0/overflow-y（styles.css），直接作为弹性行 */}
      <Transcript
        running={state.running}
        onPrompt={(text) => ctrl.send(text)}
        cwd={state.meta?.cwd}
        meta={state.meta}
      />
      <Composer
        running={state.running}
        cwd={state.meta?.cwd}
        onSend={ctrl.send}
        onSteer={ctrl.steer}
        onCancel={ctrl.cancel}
        onPickFolder={ctrl.pickWorkspace}
        disabled={state.meta?.ready === false || state.approval != null}
      />
      {state.approval && (
        <ApprovalModal
          approval={state.approval}
          onAnswer={(decision) => ctrl.approve(state.approval!.id, decision)}
        />
      )}
      {state.ask && (
        <AskCard
          ask={state.ask}
          onAnswer={ctrl.answerQuestion}
          onDismiss={() => ctrl.answerQuestion(state.ask!.id, [])}
        />
      )}
    </div>
  )
}

/** 进度计划左栏对话宿主：LocaleProvider 必须包外层（gaea 组件顶层 useT）。 */
export const ScheduleChatPane: React.FC = () => (
  <LocaleProvider>
    <ChatPaneInner />
  </LocaleProvider>
)
