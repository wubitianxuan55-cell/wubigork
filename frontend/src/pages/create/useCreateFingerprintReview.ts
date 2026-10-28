import { useCallback, useEffect, useState } from 'react'
import { NovelFingerprintStatus, NovelFingerprintBuild, NovelFingerprintScore, NovelReviewPlatforms, NovelChapterReview } from '../../../wailsjs/go/app/NovelB'
import type { ChapterReviewPayload, FingerprintScorePayload, FingerprintStatusPayload, ReviewPlatform } from '../../gaea/lib/bridge/novel'

// useCreateFingerprintReview — 创作页「文风指纹 + 平台评审」域（2026-10-28 续大道至简
// 分域拆件：CreatePage 67 态的第五族——9 个状态 + 5 个异步动作，纯自包含：
// 只依赖当前章号、两个弹层开关 setter 与 projectPath（B5a 切书失效内聚为本域 effect）。
// 页面按同名解构，全部引用点零改动）。

export interface UseCreateFingerprintReviewParams {
  activeChapterNum: number
  projectPath: string
  setFpOpen: (open: boolean) => void
  setReviewOpen: (open: boolean) => void
}

export function useCreateFingerprintReview({ activeChapterNum, projectPath, setFpOpen, setReviewOpen }: UseCreateFingerprintReviewParams) {
  // 文风指纹（参考档 + 章节体检）弹窗状态；Status 在打开时自动拉取（无独立刷新按钮）
  const [fpStatus, setFpStatus] = useState<FingerprintStatusPayload | null>(null)
  const [fpScore, setFpScore] = useState<FingerprintScorePayload | null>(null)
  const [fpBusy, setFpBusy] = useState(false)
  const [fpMsg, setFpMsg] = useState('')
  // 平台评审（v4.282，oh-story 蒸馏 T1）：档位清单 + 单章报告 + 动作态。
  const [reviewPlatforms, setReviewPlatforms] = useState<ReviewPlatform[]>([])
  const [reviewPlatformId, setReviewPlatformId] = useState('general')
  const [reviewReport, setReviewReport] = useState<ChapterReviewPayload | null>(null)
  const [reviewBusy, setReviewBusy] = useState(false)
  const [reviewMsg, setReviewMsg] = useState('')

  // 打开即拉参考档状态（刷新并入 open 时自动加载）；失败落弹窗内消息（fpMsg）。
  const openFingerprint = useCallback(async () => {
    setFpOpen(true); setFpBusy(true); setFpMsg(''); setFpScore(null)
    try {
      setFpStatus(await NovelFingerprintStatus())
    } catch (err: unknown) { setFpMsg(err instanceof Error ? err.message : '加载文风指纹失败') }
    finally { setFpBusy(false) }
  }, [setFpOpen])
  // 构建/重建参考档：用全部已写章节生成风格基线（样本不足时后端 reject 中文提示）。
  const buildFingerprint = useCallback(async () => {
    setFpBusy(true); setFpMsg('')
    try {
      const s = await NovelFingerprintBuild()
      setFpStatus(s)
      setFpMsg(`参考档已构建（${s?.chapters ?? 0} 章 · ${(s?.chars ?? 0).toLocaleString()} 字）`)
    } catch (err: unknown) { setFpMsg(err instanceof Error ? err.message : '构建参考档失败') }
    finally { setFpBusy(false) }
  }, [])
  // 体检当前章：AI 味评分 + 与参考档距离（无参考档时后端按通用阈值打分）。
  const scoreFingerprint = useCallback(async () => {
    setFpBusy(true); setFpMsg('')
    try {
      const r = await NovelFingerprintScore(activeChapterNum)
      setFpScore(r)
      setFpMsg(`第 ${activeChapterNum} 章体检完成`)
    } catch (err: unknown) { setFpMsg(err instanceof Error ? err.message : '章节体检失败') }
    finally { setFpBusy(false) }
  }, [activeChapterNum])

  // 平台评审：档位清单（一次拉取）+ 当前章确定性评审。打开面板即拉档位
  // （失败落面板消息，选择器回退为「通用」一档）；评审零模型调用。
  const openReview = useCallback(async () => {
    setReviewOpen(true)
    setReviewMsg('')
    if (reviewPlatforms.length === 0) {
      try {
        const list = await NovelReviewPlatforms()
        setReviewPlatforms(Array.isArray(list) ? list : [])
      } catch (err: unknown) {
        setReviewMsg(err instanceof Error ? err.message : '加载平台档位失败')
      }
    }
  }, [reviewPlatforms.length, setReviewOpen])
  const runReview = useCallback(async () => {
    setReviewBusy(true)
    setReviewMsg('')
    try {
      const rep = await NovelChapterReview(activeChapterNum, reviewPlatformId)
      setReviewReport(rep)
      const hit = (rep?.counts?.S1 ?? 0) + (rep?.counts?.S2 ?? 0) + (rep?.counts?.S3 ?? 0) + (rep?.counts?.S4 ?? 0)
      setReviewMsg(`第 ${activeChapterNum} 章评审完成：${rep?.platformLabel ?? ''} · ${rep?.verdict ?? ''} · 命中 ${hit} 项`)
    } catch (err: unknown) {
      setReviewMsg(err instanceof Error ? err.message : '评审失败')
    } finally {
      setReviewBusy(false)
    }
  }, [activeChapterNum, reviewPlatformId])

  // 换平台即弃旧报告（旧平台档位的结论对换掉的档位无效）
  const changeReviewPlatform = useCallback((id: string) => {
    setReviewPlatformId(id)
    setReviewReport(null)
  }, [])

  // B5a 切书失效：评审/指纹的报告、提示语与两个弹窗都取自上一本书——不清则切书后
  // 弹窗仍整屏显示上一本的内容（审计确证）。弹窗一并关闭：留在屏上比「需要重开一次」
  // 更糟。（原 CreatePage 切书 effect 内联语句逐字迁入；挂载首跑全为初值=无操作等价）
  useEffect(() => {
    setReviewReport(null); setReviewMsg(''); setReviewOpen(false)
    setFpStatus(null); setFpScore(null); setFpMsg(''); setFpOpen(false)
    // 仅在项目变化时执行；其余依赖全为 useState setter，引用恒稳
  }, [projectPath, setFpOpen, setReviewOpen])

  return {
    fpStatus, fpScore, fpBusy, fpMsg,
    openFingerprint, buildFingerprint, scoreFingerprint,
    reviewPlatforms, reviewPlatformId, changeReviewPlatform,
    reviewReport, reviewBusy, reviewMsg,
    openReview, runReview,
  }
}
