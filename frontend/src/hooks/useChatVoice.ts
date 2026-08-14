// ChatPage 拆分产物：语音对话集成（行为零变化，T6-10.1）。
// T6-3.3：语音消息持久化。识别文本/回复经 ChatAppendMessages 落库（单事务），
// 与正常 ChatSend 的落库互不重复（语音管道不走 ChatSend）；无活跃话题时仅内存展示。
import { useCallback, useEffect, useState } from 'react'
import * as App from '../wailsjsCompat'
import { useVoiceChat } from './useVoiceChat'
import { VOICE_LAUNCH_FLAG } from '../components/ModuleLauncher'
import { logFrontendError, nextMsgKey, nowStr } from '../pages/chat/utils'
import type { ChatMsg } from '../pages/chat/types'

export interface UseChatVoiceOptions {
  setMessages: React.Dispatch<React.SetStateAction<ChatMsg[]>>
  getActiveId: () => string
  getMode: () => string
}

export function useChatVoice({ setMessages, getActiveId, getMode }: UseChatVoiceOptions) {
  const [voiceOn, setVoiceOn] = useState(false)

  const persistVoiceMessages = useCallback((msgs: Array<{ Role: string; Content: string; Extra: string }>) => {
    const topicID = getActiveId()
    if (!topicID) return
    // ChatAppendMessages 为 T6-3 新绑定：wailsjs/go 生成物由 wails build 再生成，
    // 绑定已声明（Array<app.ChatMessageInput>），直接调用。
    App.ChatAppendMessages(topicID, msgs).catch((err: unknown) => {
      // 落库失败不静默：记录到 gaea.log（界面不受影响，内存消息已展示）
      logFrontendError('语音消息落库失败: ' + (err instanceof Error ? err.message : String(err)))
    })
  }, [getActiveId])

  const onVoiceTranscript = useCallback((t: string) => {
    const text = (t || '').trim(); if (!text) return
    setMessages(prev => [...prev, { key: nextMsgKey(), role: 'user', content: text, createdAt: nowStr() }])
    persistVoiceMessages([{ Role: 'user', Content: text, Extra: '' }])
  }, [persistVoiceMessages, setMessages])

  const onVoiceReply = useCallback((t: string) => {
    const text = (t || '').trim(); if (!text) return
    setMessages(prev => [...prev, { key: nextMsgKey(), role: 'assistant', content: text, createdAt: nowStr() }])
    persistVoiceMessages([{ Role: 'assistant', Content: text, Extra: '' }])
  }, [persistVoiceMessages, setMessages])

  const { state: voice, start: startVoice, stop: stopVoice } = useVoiceChat({ onTranscript: onVoiceTranscript, onReply: onVoiceReply })

  const toggleVoice = useCallback(async () => {
    if (voiceOn) { stopVoice(); setVoiceOn(false); return }
    try { await App.VoiceApplySettings?.({ personalityPresetId: getMode() }) } catch (_) {}
    setVoiceOn(true)
    await startVoice()
  }, [voiceOn, getMode, startVoice, stopVoice])

  // 首页语音入口兼容：进入聊天板块自动开启收听
  useEffect(() => {
    let flag = false
    try { flag = sessionStorage.getItem(VOICE_LAUNCH_FLAG) === '1' } catch (_) {}
    if (flag) {
      try { sessionStorage.removeItem(VOICE_LAUNCH_FLAG) } catch (_) {}
      toggleVoice()
    }
  }, [toggleVoice])

  return { voice, voiceOn, toggleVoice }
}
