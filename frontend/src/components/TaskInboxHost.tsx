import { useCallback, useEffect, useState } from 'react'
import { TaskInboxPanel } from '../gaea/components/TaskInboxPanel'
import { requestSessionResume } from '../gaea/lib/pendingSessionResume'
import { useAppStore } from '../stores/appStore'
import { FRONTEND_EVENTS } from '../events'

/**
 * 壳层级任务收件箱宿主。
 *
 * 首页删除后仍需保证命令面板空态的「打开任务收件箱」有常驻监听者；
 * 列表保持跨空间全量（scope=''），新建任务仍按当前壳层空间落标签。
 */
export function TaskInboxHost({ onNavigate }: { onNavigate: (target: string) => void }) {
  const [open, setOpen] = useState(false)
  const space = useAppStore((state) => state.space)
  const closeInbox = useCallback(() => setOpen(false), [])

  useEffect(() => {
    const openInbox = () => setOpen(true)
    window.addEventListener(FRONTEND_EVENTS.OPEN_TASK_INBOX, openInbox)
    return () => window.removeEventListener(FRONTEND_EVENTS.OPEN_TASK_INBOX, openInbox)
  }, [])

  const resumeSession = useCallback((path: string) => {
    requestSessionResume(path)
    onNavigate('gaea')
  }, [onNavigate])

  return (
    <TaskInboxPanel
      open={open}
      onClose={closeInbox}
      space={space}
      scope=""
      onNavigate={onNavigate}
      onResumeSession={resumeSession}
    />
  )
}
