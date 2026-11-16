// chatErrors — B1 错误分类前端映射（chat-demo describe.ts 语义对齐，
// docs/gaea-deepseek-chatdemo-distill-2026-11.md §1.3）。Go 侧
// classifyChatStreamError 给 error 帧补稳定 code；本表把码翻译成
// 友好文案与可选动作（跳模型中心）。未命中码回落原文展示，零行为变化。

export interface DescribedChatError {
  text: string
  /** 需要用户处理的引导动作；当前仅「打开模型中心」 */
  action?: 'modelcenter'
}

export function describeChatError(code: string | undefined, raw: string | undefined): DescribedChatError {
  switch (code) {
    case 'engine_unreachable':
      return { text: '引擎未连接：请检查本机引擎状态后重试', action: 'modelcenter' }
    case 'auth':
      return { text: '鉴权失败：请到模型中心检查 API Key', action: 'modelcenter' }
    case 'engine_busy':
      return { text: '引擎繁忙，请稍后重试' }
    case 'timeout':
      return { text: '引擎响应超时，请重试' }
    case 'context_overflow':
      return { text: '上下文超限：建议清空会话或新建对话后重试' }
    case 'persistence':
      return { text: '回复已生成但保存失败，请重试' }
    case 'empty_reply':
      return { text: '本轮未生成内容，请重试' }
    default:
      return { text: raw ? `请求失败：${raw}` : '请求失败：未知错误' }
  }
}
