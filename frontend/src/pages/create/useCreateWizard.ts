import { useState } from 'react'
import type { Branch, BranchCastEntry } from '../../components/novel/create/BranchWizardModal'

// useCreateWizard — 创作页分支向导域（2026-10-27 大道至简分域拆件：
// CreatePage 67 个 useState 的第三族——BranchWizardModal 的请求数据族，
// 纯数据态零逻辑；页面按同名解构消费，引用点零改动）。

export interface WizardRequest { prevChapter: number; overwriteChapter: number; branchFromID: string }

export function useCreateWizard() {
  const [wizard, setWizard] = useState<WizardRequest | null>(null)
  const [wizardBranches, setWizardBranches] = useState<Branch[]>([])
  // 选角会议：结果由页面层持有（后台构思期间弹窗收起不丢）
  const [wizardCastSelection, setWizardCastSelection] = useState<BranchCastEntry[]>([])
  // 选角会议数据源：项目名册 / 角色库候选 / 上一章出场（向导打开时拉取；失败静默降级）
  const [wizardCast, setWizardCast] = useState<string[]>([])
  const [wizardLibCast, setWizardLibCast] = useState<{ name: string; note?: string }[]>([])
  const [wizardPrevCast, setWizardPrevCast] = useState<string[]>([])

  return {
    wizard, setWizard,
    wizardBranches, setWizardBranches,
    wizardCastSelection, setWizardCastSelection,
    wizardCast, setWizardCast,
    wizardLibCast, setWizardLibCast,
    wizardPrevCast, setWizardPrevCast,
  }
}
