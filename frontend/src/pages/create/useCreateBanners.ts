import { useState } from 'react'
import type { AiTasteResult } from '../../components/novel/create/chapterStreamTypes'

// useCreateBanners — 创作页写作反馈横幅域（2026-10-27 大道至简分域拆件：
// CreatePage 67 个 useState 的第二族——顶部 Alert 横幅的 5 个状态源。
// 呈现面收敛为 WritingStatusBar 单条状态条，状态语义零变化）。

/** 章级变更声明门禁（阶段八 8.5）：拒收时人可读问题清单 */
export interface DeclGateState { declared: boolean; accepted: boolean; issues: string[] }

export function useCreateBanners() {
  // 生成完成后 novelstyle 的 AI 味检测结果（分数 + 命中问题），展示给作者
  const [aiTaste, setAiTaste] = useState<AiTasteResult | null>(null)
  const [declGate, setDeclGate] = useState<DeclGateState | null>(null)
  const [samplerNote, setSamplerNote] = useState('')
  // 阶段收尾提示（阶段八 8.8）：收官章 done 时出现，建议跑体检（建议制，不自动跑）
  const [stageClosing, setStageClosing] = useState<{ stage: number } | null>(null)
  // 叙事状态消息（工具轨旁短提示）
  const [stateMsg, setStateMsg] = useState('')

  return { aiTaste, setAiTaste, declGate, setDeclGate, samplerNote, setSamplerNote, stageClosing, setStageClosing, stateMsg, setStateMsg }
}
