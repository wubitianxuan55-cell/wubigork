import React from 'react'
import ChatPanel from './ChatPanel'
import { Z_INDEX } from '../utils/zIndex'
import type { ChapterTabData } from '../types'

interface AIAssistSheetProps {
  tab: ChapterTabData | null
  onUpdate: <K extends keyof ChapterTabData>(field: K, value: ChapterTabData[K]) => void
  autoSendMsg: string
  onAutoSendDone: () => void
}

/**
 * AIAssistSheet — AI 协写聊天面板（桌面端底部 + 移动端 FAB + Sheet）
 * 提取自 ChapterPage，集中管理写作 Agent 对话交互
 */
const AIAssistSheet: React.FC<AIAssistSheetProps> = ({ tab, onUpdate, autoSendMsg, onAutoSendDone }) => {
  const [aiSheetOpen, setAiSheetOpen] = React.useState(false)

  const handleSend = async (msg: string): Promise<string> => {
    try {
      if (!tab?.chapterNum) return '请先选择章节'
      const cleanMsg = msg.replace(/##AUTO_SEND##\d+$/, '')
      // @ts-ignore
      const result = await window.go.app.App.ChatChapter(tab.chapterNum, cleanMsg)
      return result.reply as string
    } catch (err: any) {
      throw new Error(typeof err === 'string' ? err : (err?.message || '对话失败'))
    }
  }

  const messages = tab?.messages || []

  return (
    <>
      {/* 底部 Agent 面板 */}
      {tab && (
        <div style={{
          display: 'flex', flexDirection: 'column',
          background: 'var(--color-bg-container)',
          borderRadius: 8,
          border: '1px solid var(--color-border)',
          minHeight: 0,
        }}>
          <ChatPanel
            title={`写作 Agent · ${tab.node.title}`}
            messages={messages}
            onMessagesChange={(msgs) => onUpdate('messages', msgs)}
            onSend={handleSend}
            placeholder="和 AI 讨论本章修改..."
            defaultCollapsed
            autoSend={autoSendMsg}
            onAutoSendDone={onAutoSendDone}
          />
        </div>
      )}

    </>
  )
}

export default AIAssistSheet
