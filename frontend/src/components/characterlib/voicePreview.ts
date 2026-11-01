// voicePreview.ts — 角色档案试听段播放器（角色库编辑器/检查器共用）。
// 试听主链走 TTSLineAudioFor（内容寻址文件缓存：命中直接播文件秒开，未命中
// 合成一次落盘，之后点试听=纯播文件）；TTSVoicePreviewBase64 只服务编辑器里
// 未保存新角色的临时试听（无 id 无法落盘）。播放用单例 HTMLAudioElement：
// 再点一次即停（旧段未结束就播新段会叠音）。
import { message } from 'antd'
import { app } from '../../gaea/lib/bridge'

let current: HTMLAudioElement | null = null

function stopCurrent() {
  if (current) {
    current.pause()
    current.src = ''
    current = null
  }
}

/** 角色音色描述的性别缺省（与编辑器占位符、后端聊天链 gender 兜底同口径）。 */
export function genderDefaultVoiceDesc(gender?: string): string {
  switch ((gender || '').toLowerCase()) {
    case 'male': return '沉稳清晰的青年男性声线'
    case 'female': return '温柔清晰的青年女性声线'
    default: return ''
  }
}

/**
 * 试听台词解析（专属语音台词，单一口径）：角色 ttsLine > 对话样本首条 >
 * 姓名问候。编辑器（表单值）与检查器（档案保存值）共用。
 */
export function resolveVoiceLine(c: {
  name?: string
  ttsLine?: string
  dialogueSamples?: string[]
}): string {
  const line = (c.ttsLine || '').trim()
  if (line) return line
  const sample = (c.dialogueSamples || []).map(s => (s || '').trim()).find(Boolean)
  if (sample) return sample
  return `你好，我是「${(c.name || '').trim() || '…'}」，很高兴认识你。`
}

function playDataUrl(mimeType: string, b64: string): HTMLAudioElement {
  const audio = new Audio(`data:${mimeType || 'audio/wav'};base64,${b64}`)
  current = audio
  audio.onended = () => { if (current === audio) current = null }
  return audio
}

/**
 * 台词试听主链（文件缓存优先）：后端探键命中直接回文件字节（秒开），未命中
 * 合成一次落盘（cached=false 时提示「语音文件已生成」）。返回 false=失败已弹错。
 */
export async function playCharacterVoiceLine(c: {
  id?: string
  name?: string
  ttsLine?: string
  ttsVoiceId?: string
  ttsDesc?: string
  gender?: string
  dialogueSamples?: string[]
}): Promise<boolean> {
  stopCurrent()
  try {
    const res = await app.TTSLineAudioFor(JSON.stringify({
      id: c.id || '', name: c.name || '', ttsLine: c.ttsLine || '',
      ttsVoiceId: c.ttsVoiceId || '', ttsDesc: c.ttsDesc || '',
      gender: c.gender || '', dialogueSamples: c.dialogueSamples || [],
    })) as { audio?: string; mimeType?: string; cached?: boolean }
    const b64 = res?.audio || ''
    if (!b64) throw new Error('返回空音频')
    playDataUrl(res.mimeType || 'audio/wav', b64)
    await current!.play()
    if (res.cached === false) message.success('语音文件已生成，之后试听秒开')
    return true
  } catch (err) {
    if (current) stopCurrent()
    message.error(`试听失败：${err instanceof Error ? err.message : String(err)}`)
    return false
  }
}

/**
 * 播放一段角色音色试听（实时合成，不落文件；仅编辑器未保存新角色回退用）。
 * 失败（引擎不可用/合成失败/空音频）弹错并返回 false，调用方只管按钮 loading 态。
 */
export async function playVoicePreview(
  text: string, voiceId: string, voiceDesc: string,
): Promise<boolean> {
  stopCurrent()
  try {
    const res = await app.TTSVoicePreviewBase64(text, voiceId, voiceDesc) as { base64?: string; mimeType?: string }
    const b64 = res?.base64 || ''
    if (!b64) throw new Error('返回空音频')
    playDataUrl(res.mimeType || 'audio/wav', b64)
    await current!.play()
    return true
  } catch (err) {
    if (current) stopCurrent()
    message.error(`试听失败：${err instanceof Error ? err.message : String(err)}`)
    return false
  }
}

/** 停掉当前试听段（抽屉关闭/组件卸载时调用，防界面关了声音还在）。 */
export function stopVoicePreview() {
  stopCurrent()
}
