import { useState } from 'react'

// useCreatePanels — 创作页弹层开关域（2026-10-27 大道至简分域拆件：
// CreatePage 67 个 useState 的第一族——15 个弹层/面板 open 态，纯开关零逻辑。
// 页面按同名解构消费，全部引用点零改动）。

/** 15 个弹层 open 态族（用途注释随迁自 CreatePage 各声明处） */
export function useCreatePanels() {
  // 整章重写弹窗（t4-C3 前端消费）
  const [rwOpen, setRwOpen] = useState(false)
  // 重写版本历史面板（t4-C3 余项：版本库 List/Get 前端消费）
  const [rwHistOpen, setRwHistOpen] = useState(false)
  // 提示词工坊面板（t6 首刀：模板可编辑覆盖层；打开才拉一次）
  const [promptWsOpen, setPromptWsOpen] = useState(false)
  // 对标拆解面板（oh-story story-long-analyze 蒸馏：写法学习导向，与章节分析分面）
  const [dissectOpen, setDissectOpen] = useState(false)
  // 技法库面板（webnovel-writer 九表思想蒸馏 N11：零 LLM 检索，只读不落盘）
  const [techniqueOpen, setTechniqueOpen] = useState(false)
  // 章节计划卡（刀1 线D）：工具轨「章节计划」与硬闸弹窗共用同一展开位
  const [planOpen, setPlanOpen] = useState(false)
  const [analysisOpen, setAnalysisOpen] = useState(false)
  const [healthOpen, setHealthOpen] = useState(false)
  const [spineOpen, setSpineOpen] = useState(false)
  const [convergeOpen, setConvergeOpen] = useState(false)
  const [ctxInvOpen, setCtxInvOpen] = useState(false)
  // 叙事状态账本（narrative 审批制结算）UI
  const [stateOpen, setStateOpen] = useState(false)
  // 文风指纹（参考档 + 章节体检）弹窗状态
  const [fpOpen, setFpOpen] = useState(false)
  // 平台评审（v4.282，oh-story 蒸馏 T1）弹窗状态
  const [reviewOpen, setReviewOpen] = useState(false)

  return {
    rwOpen, setRwOpen,
    rwHistOpen, setRwHistOpen,
    promptWsOpen, setPromptWsOpen,
    dissectOpen, setDissectOpen,
    techniqueOpen, setTechniqueOpen,
    planOpen, setPlanOpen,
    analysisOpen, setAnalysisOpen,
    healthOpen, setHealthOpen,
    spineOpen, setSpineOpen,
    convergeOpen, setConvergeOpen,
    ctxInvOpen, setCtxInvOpen,
    stateOpen, setStateOpen,
    fpOpen, setFpOpen,
    reviewOpen, setReviewOpen,
  }
}
