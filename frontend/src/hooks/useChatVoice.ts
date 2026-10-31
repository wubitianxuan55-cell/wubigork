// ChatPage 语音对话聚合：把识别文本、AI 回复和聊天话题接到同一条会话链。
// 语音回合绑定启动时的话题，消息经 ChatAppendMessages 落库并同步话题标题/
// 预览/活跃时间；切话题或结束会话后，迟到事件不得串入新话题。
import { useCallback, useRef, useState } from 'react'
import { app } from '../gaea/lib/bridge'
import { useVoiceChat } from './useVoiceChat'
import { logFrontendError, nextMsgKey, nowStr } from '../pages/chat/utils'
import type { ChatMsg } from '../pages/chat/types'

export interface UseChatVoiceOptions {
  setMessages: React.Dispatch<React.SetStateAction<ChatMsg[]>>
  getActiveId: () => string
  getMode: () => string
  finalizeTopicAfterSend?: (topicId: string, firstUserText: string) => Promise<void>
}

export function useChatVoice({
  setMessages,
  getActiveId,
  getMode,
  finalizeTopicAfterSend,
}: UseChatVoiceOptions) {
  const [voiceOn, setVoiceOn] = useState(false)
  // 一轮语音会话绑定启动时的话题：即使后端事件与切话题交错，也不能把旧
  // 声音串进新会话。清空代表当前没有可接收的语音回合。
  const sessionTopicRef = useRef('')
  const sessionGenerationRef = useRef(0)

  const persistVoiceMessages = useCallback(async (
    topicID: string,
    msgs: Array<{ role: string; content: string; extra: string }>,
  ): Promise<boolean> => {
    if (!topicID) return false
    // ChatAppendMessages 为 T6-3 新绑定：wailsjs/go 生成物由 wails build 再生成，
    // 绑定已声明（Array<app.ChatMessageInput>），直接调用。
    try {
      await app.ChatAppendMessages(topicID, msgs)
      return true
    } catch (err: unknown) {
      // 落库失败不静默：记录到 gaea.log（界面不受影响，内存消息已展示）
      logFrontendError('语音消息落库失败: ' + (err instanceof Error ? err.message : String(err)))
      return false
    }
  }, [])

  const onVoiceTranscript = useCallback((t: string) => {
    const text = (t || '').trim(); if (!text) return
    const topicID = sessionTopicRef.current
    if (!topicID) return
    // 事件可能晚于切话题到达：只写入发起本轮语音的会话；当前视图已切走时
    // 不再把旧声音插进新会话。
    if (getActiveId() === topicID) {
      setMessages(prev => [...prev, { key: nextMsgKey(), role: 'user', content: text, createdAt: nowStr() }])
    }
    void persistVoiceMessages(topicID, [{ role: 'user', content: text, extra: '' }]).then((saved) => {
      if (saved) void finalizeTopicAfterSend?.(topicID, text)
    })
  }, [finalizeTopicAfterSend, getActiveId, persistVoiceMessages, setMessages])

  const onVoiceReply = useCallback((t: string) => {
    const text = (t || '').trim(); if (!text) return
    const topicID = sessionTopicRef.current
    if (!topicID) return
    if (getActiveId() === topicID) {
      setMessages(prev => [...prev, { key: nextMsgKey(), role: 'assistant', content: text, createdAt: nowStr() }])
    }
    void persistVoiceMessages(topicID, [{ role: 'assistant', content: text, extra: '' }])
  }, [getActiveId, persistVoiceMessages, setMessages])

  const {
    state: voice,
    start: startVoice,
    stop: stopVoice,
    interrupt: interruptVoice,
  } = useVoiceChat({ onTranscript: onVoiceTranscript, onReply: onVoiceReply })

  const stopVoiceSession = useCallback(() => {
    sessionGenerationRef.current += 1
    sessionTopicRef.current = ''
    stopVoice()
    setVoiceOn(false)
  }, [stopVoice])

  const toggleVoice = useCallback(async (): Promise<boolean> => {
    if (voiceOn) {
      stopVoiceSession()
      return false
    }
    const topicID = getActiveId()
    if (!topicID) return false
    const generation = ++sessionGenerationRef.current
    // 先登记会话再启动：Web Speech 的最终结果可能紧跟在 rec.start() 后到达。
    sessionTopicRef.current = topicID
    try { await app.VoiceApplySettings({ personalityPresetId: getMode() }) } catch (_) {}
    const started = await startVoice()
    // 启动期间用户切话题/关闭语音台：结束后立即回收这一次迟到的启动结果。
    if (generation !== sessionGenerationRef.current || sessionTopicRef.current !== topicID) {
      if (started) stopVoice()
      return false
    }
    if (started) {
      setVoiceOn(true)
    } else {
      sessionTopicRef.current = ''
      setVoiceOn(false)
    }
    return started
  }, [voiceOn, getMode, getActiveId, startVoice, stopVoice, stopVoiceSession])

  return { voice, voiceOn, toggleVoice, stopVoiceSession, interruptVoice }
}
