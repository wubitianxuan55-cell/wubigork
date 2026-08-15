/**
 * events.ts — 事件名常量表 + subscribe() 统一封装（3.0 01 报告 §4）
 *
 * 后端事件 21 个（window.runtime.EventsOn 全量订阅点，§4.1）+ 前端自定义事件 4 个（§4.2）。
 * subscribe() 收敛"规范路径（wailsjs runtime 返回卸载函数）"与"手动路径（window.runtime 裸调）"
 * 两套并存（§4.3），统一返回 () => void 卸载函数。
 */

// ─── 后端事件（§4.1，21 个）────────────────────────────────────────────
export const BACKEND_EVENTS = {
  /** 顶栏模型标签刷新（MainLayout） */
  MODEL_CHANGED: 'model-changed',
  /** 功能模型变更（useFeatureModel，按 d.feature 过滤） */
  FEATURE_MODEL_CHANGED: 'feature-model-changed',
  /** 聊天流动态频道前缀：实际事件名为 `chat-stream:${runID}`（useChatStream） */
  CHAT_STREAM: 'chat-stream',
  /** 章节生成流（useChapterStream） */
  CREATE_CHAPTER_STREAM: 'create-chapter-stream',
  /** 幽灵补写（GhostText） */
  GHOST_STREAM: 'ghost-stream',
  /** AI 控制台输出（AIConsole） */
  XAI_OUTPUT: 'xai-output',
  /** TTS 语音流（TTSPlayer） */
  TTS_STREAM: 'tts-stream',
  /** 新角色发现（NewCharactersModal） */
  NEW_CHARACTERS_DISCOVERED: 'new-characters-discovered',
  /** 角色批量填充进度（CharacterLibraryPage） */
  CHARACTER_FILL_PROGRESS: 'character-fill-progress',
  /** 语音状态（useVoiceChat） */
  VOICE_STATE: 'voice:state',
  /** 语音识别文本 */
  VOICE_TRANSCRIPT: 'voice:transcript',
  /** 语音回复文本 */
  VOICE_REPLY: 'voice:reply',
  /** 语音 TTS 音频 */
  VOICE_TTS_AUDIO: 'voice:tts-audio',
  /** 语音 TTS 朗读文本 */
  VOICE_TTS_SPEAK_TEXT: 'voice:tts-speak-text',
  /** 语音 TTS 朗读取消 */
  VOICE_TTS_SPEAK_CANCEL: 'voice:tts-speak-cancel',
  /** 语音聆听状态 */
  VOICE_LISTENING: 'voice:listening',
  /** 语音思考状态 */
  VOICE_THINKING: 'voice:thinking',
  /** gaea 桥接主事件（bridge.ts onEvent） */
  GAEA_EVENT: 'gaea-event',
  /** 更新进度（bridge.ts onUpdaterProgress） */
  UPDATER_PROGRESS: 'updater:progress',
  /** 任务事件（bridge.ts onTaskEvent） */
  GAEA_TASK: 'gaea-task',
  /** 桥接就绪（bridge.ts onReady） */
  GAEA_READY: 'gaea-ready',
} as const

// ─── 前端自定义事件（§4.2，4 个，dispatchEvent 非 runtime）───────────────
export const FRONTEND_EVENTS = {
  /** 跨页面导航（chat/utils.ts / CharacterPage / ModelPanel → MainLayout 白名单校验） */
  NAVIGATE: 'navigate',
  /** 轻语人格变更（CharacterLibraryPage 内） */
  GAEA_PERSONA_CHANGED: 'gaea-persona-changed',
  /** 项目角色变更（CharacterLibraryPage / NewCharactersModal） */
  GAEA_PROJECT_CHARS_CHANGED: 'gaea-project-chars-changed',
  /** AI 辅助发送（ChapterEditor 内部） */
  AI_ASSIST_SEND: 'ai-assist-send',
} as const

export type BackendEventName = (typeof BACKEND_EVENTS)[keyof typeof BACKEND_EVENTS]
export type FrontendEventName = (typeof FRONTEND_EVENTS)[keyof typeof FRONTEND_EVENTS]

/** 聊天流动态频道：chat-stream:<runID> */
export function chatStreamChannel(runID: string): string {
  return `${BACKEND_EVENTS.CHAT_STREAM}:${runID}`
}

/**
 * 统一事件订阅：优先 wailsjs runtime（返回卸载函数），退化为 window.runtime 裸调
 * （EventsOn 返回 void，用 EventsOff 卸载）。事件通道缺失时返回 noop，不抛错。
 */
export function subscribe(event: string, handler: (data: unknown) => void): () => void {
  const rt = window.runtime
  if (!rt?.EventsOn) return () => {}
  try {
    const unsub = rt.EventsOn(event, handler) as (() => void) | void
    if (typeof unsub === 'function') return unsub
    return () => {
      try { rt.EventsOff?.(event, handler) } catch { /* 清理失败无害 */ }
    }
  } catch {
    return () => {}
  }
}

/** 分发前端自定义事件（§4.2 的 dispatchEvent 统一封装） */
export function emitFrontendEvent(name: FrontendEventName, detail?: unknown): void {
  window.dispatchEvent(new CustomEvent(name, { detail }))
}
