import React, { useCallback, useEffect, useRef, useState } from 'react'
import { message, Modal } from 'antd'
import { app } from '../../gaea/lib/bridge'
import { GetNovelState, BuildNovelStatePatch, SettleNovelState, DeSlopChapterAiTaste, RewriteChapterAiTaste, NovelOutlineReconstruct, NovelOutlineReconstructApply, NovelOutlineReconstructStart, NovelOutlineReconstructTaskGet } from '../../../wailsjs/go/app/NovelB'

// useCreateNarrativeOps — 创作页「叙事状态账本 + AI 反推大纲 + 一键去味」域
// （2026-10-28 续大道至简分域拆件：CreatePage 67 态的第六族——三者共用同一条
// rail 消息通道（stateMsg/stateBusy，来自 useCreateBanners），故同族出走。
// 7 个状态 + 6 个异步动作 + 反推轮询/取消/串联事件。页面按同名解构，全部引用点
// 零改动；statePatch 为账本内部态不外露。

export interface UseCreateNarrativeOpsParams {
  /** 本 pane 是否可见（novel:auto-reconstruct 串联按它挂起/开跑） */
  active: boolean
  activeChapterNum: number
  /** rail 消息通道（useCreateBanners 出口，账本/反推/去味共用一条） */
  setStateMsg: (msg: string) => void
  generatingRef: { current: boolean }
  /** 去味在服务端生效后的编辑区刷新闸（脏则只提示不覆盖） */
  refreshEditorAfterServerRewrite: () => void
  loadOutlines: () => unknown
}

export function useCreateNarrativeOps({ active, activeChapterNum, setStateMsg, generatingRef, refreshEditorAfterServerRewrite, loadOutlines }: UseCreateNarrativeOpsParams) {
  const [novelState, setNovelState] = useState<{ version?: number; entities?: Record<string, { name?: string; type?: string; status?: string }> } | null>(null)
  const [statePatch, setStatePatch] = useState<unknown>(null)
  const [stateBusy, setStateBusy] = useState(false)
  // AI 反推大纲（v4.281）：反推 → 确认 → 幂等合并进大纲
  const [reconstructBusy, setReconstructBusy] = useState(false)
  // v4.340 取消：轮询期「取消反推」可见（cancellable=已入队拿到 taskId），
  // 点击置标志 → 轮询下个检查点退出 + 请求后端协作取消任务。
  const [reconstructCancellable, setReconstructCancellable] = useState(false)
  const reconstructCancelRef = useRef(false)
  const reconstructTaskIdRef = useRef('')
  /** 已等待秒数（v4.421.0：长任务可见化——12 分钟轮询期间此前只有一个静态提示） */
  const [reconstructElapsed, setReconstructElapsed] = useState(0)
  /** 组件是否还在（v4.421.0：卸载/切走后停止轮询，避免对已卸载组件 setState 与空跑请求） */
  const reconstructAliveRef = useRef(true)
  useEffect(() => () => { reconstructAliveRef.current = false }, [])

  const loadState = useCallback(async () => {
    setStateBusy(true); setStateMsg('')
    try {
      const s = (await GetNovelState()) as { version?: number; entities?: Record<string, { name?: string; type?: string; status?: string }> }
      setNovelState(s); setStateMsg('已加载叙事状态账本')
    } catch (err: unknown) { setStateMsg(err instanceof Error ? err.message : '加载失败') }
    finally { setStateBusy(false) }
  }, [setStateMsg])
  const proposeState = useCallback(async () => {
    setStateBusy(true); setStateMsg('')
    try {
      const p = await BuildNovelStatePatch(activeChapterNum)
      setStatePatch(p); setStateMsg(`AI 已生成第 ${activeChapterNum} 章状态建议，等待你审批`)
    } catch (err: unknown) { setStateMsg(err instanceof Error ? err.message : '生成建议失败') }
    finally { setStateBusy(false) }
  }, [activeChapterNum, setStateMsg])
  const approveState = useCallback(async () => {
    if (statePatch === null || statePatch === undefined) { setStateMsg('请先生成 AI 状态建议'); return }
    setStateBusy(true); setStateMsg('')
    try {
      const r = (await SettleNovelState(JSON.stringify(statePatch), true)) as { version?: number }
      setStateMsg(`已审批结算，状态账本版本 → ${r?.version ?? '?'}`)
      await loadState()
    } catch (err: unknown) { setStateMsg(err instanceof Error ? err.message : '结算失败') }
    finally { setStateBusy(false) }
  }, [statePatch, loadState, setStateMsg])

  // 手动「一键去味」：对当前章节跑确定性 DeSlopRewrite（任意章节可用，不只生成时）。
  // v4.421.0：成功后刷新编辑区（旧实现只更新提示，作者看到的仍是旧文，以为去味没生效）；
  // 若编辑区有未保存的手写修改则不刷新，如实提示，避免把作者的字覆盖掉。
  const deslopChapter = useCallback(async () => {
    // A6：生成中途去味会把**半章**写盘，随后流式 done 再写全文——两写者的最终结果取决于
    // 到达顺序（旧实现不禁用，rail 按钮在生成中照常可点）。rail 上另外加了 disabled 门控，
    // 这里保留早退以覆盖非按钮调用点。
    if (generatingRef.current) { message.warning('正在生成，请先停止生成再去味'); return }
    setStateBusy(true); setStateMsg('')
    try {
      const r = (await DeSlopChapterAiTaste(activeChapterNum)) as { changes?: number; beforeScore?: number; afterScore?: number; done?: boolean }
      setStateMsg(r?.done ? `已去味 ${r.changes} 处，分数 ${r.beforeScore}→${r.afterScore}` : '未命中 AI 套路，无需去味')
      if (r?.done) refreshEditorAfterServerRewrite()
    } catch (err: unknown) { setStateMsg(err instanceof Error ? err.message : '去味失败') }
    finally { setStateBusy(false) }
  }, [activeChapterNum, refreshEditorAfterServerRewrite, setStateMsg, generatingRef])
  // 高级去味：LLM 受限重写命中句（质量更高，需模型调用）。
  const llmDeslop = useCallback(async () => {
    // A6：同 deslopChapter——生成中改写同一章，两个写者会互相覆盖。
    if (generatingRef.current) { message.warning('正在生成，请先停止生成再去味'); return }
    setStateBusy(true); setStateMsg('')
    try {
      const r = (await RewriteChapterAiTaste(activeChapterNum)) as { done?: boolean; rewritten?: number; beforeScore?: number; afterScore?: number; reason?: string }
      setStateMsg(r?.done ? `已高级去味 ${r.rewritten} 句，分数 ${r.beforeScore}→${r.afterScore}` : (r?.reason ?? '无命中句'))
      if (r?.done) refreshEditorAfterServerRewrite()
    } catch (err: unknown) { setStateMsg(err instanceof Error ? err.message : '高级去味失败') }
    finally { setStateBusy(false) }
  }, [activeChapterNum, refreshEditorAfterServerRewrite, setStateMsg, generatingRef])

  // ── AI 反推大纲（v4.281，拆书导入 P1）：反推当前工程立项 + 分批章节大纲（只读预览），
  // 确认后按章号合并进大纲（幂等、不覆盖章节正文）；模型不可用时后端自动规则兜底并如实回报。
  const reconstructOutlines = useCallback(async () => {
    setReconstructBusy(true)
    setStateMsg('AI 反推大纲中…')
    try {
      // v4.291 任务化：反推入任务队列（任务中心可见、页面关闭不丢），
      // 轮询取结果；队列不可用回落同步绑定。v4.340：轮询期可取消——
      // 停止等待并请求后端取消任务（任务完成竞态下取消失败被吞，任务中心可查）。
      let preview: Awaited<ReturnType<typeof NovelOutlineReconstruct>> | null = null
      let cancelled = false
      try {
        const started = await NovelOutlineReconstructStart()
        reconstructTaskIdRef.current = started.taskId || ''
        reconstructCancelRef.current = false
        setReconstructCancellable(true)
        const startedAt = Date.now()
        const deadline = startedAt + 12 * 60_000
        for (;;) {
          if (!reconstructAliveRef.current) return
          if (reconstructCancelRef.current) { cancelled = true; break }
          if (Date.now() > deadline) throw new Error('反推任务超时（12 分钟）')
          await new Promise((r) => setTimeout(r, 3000))
          if (!reconstructAliveRef.current) return
          const waited = Math.floor((Date.now() - startedAt) / 1000)
          setReconstructElapsed(waited)
          setStateMsg(`AI 反推大纲中…已等待 ${Math.floor(waited / 60)} 分 ${waited % 60} 秒`)
          if (reconstructCancelRef.current) { cancelled = true; break }
          const st = await NovelOutlineReconstructTaskGet()
          if (!reconstructAliveRef.current) return
          if (st.status === 'failed') throw new Error(st.error || '反推任务失败')
          if (st.status === 'succeeded' && st.preview) {
            preview = st.preview as Awaited<ReturnType<typeof NovelOutlineReconstruct>>
            break
          }
        }
      } catch (taskErr: unknown) {
        const msg = taskErr instanceof Error ? taskErr.message : ''
        const fallbackable = msg.includes('任务队列不可用') || msg.includes('尚无反推任务')
        if (!fallbackable) throw taskErr
        if (!reconstructAliveRef.current) return
        // 回退路径是同步绑定，没有取消面——如实告知，不让「取消反推」按钮假消失
        setStateMsg('任务队列不可用：已回退同步反推（该路径不可取消，完成后会弹确认）')
        preview = await NovelOutlineReconstruct()
        if (!reconstructAliveRef.current) return
      }
      if (cancelled) {
        // 用户取消：停止等待并请求后端协作取消（任务恰已完成时后端报错，
        // 吞掉即可——等待已停，结果可在任务中心看到）。
        const id = reconstructTaskIdRef.current
        if (id) void app.TaskCancel(id).catch(() => {})
        setStateMsg('已取消反推等待；任务已请求取消，可在任务中心查看')
        return
      }
      const items = preview?.items ?? []
      if (items.length === 0) {
        setStateMsg('反推结果为空，未做修改')
        return
      }
      const head = preview?.aiUsed ? 'AI 反推完成' : '模型不可用，已用规则兜底'
      const warn = preview?.warnings?.length ? `\n提示：${preview.warnings.slice(0, 2).join('；')}` : ''
      const meta = [preview?.genre, preview?.narrativePerspective, preview?.targetWords ? `目标 ${preview.targetWords.toLocaleString()} 字` : '']
        .filter(Boolean).join(' · ')
      // 篇幅路由（oh-story T4）：中/长篇预览按每 segmentSize 章聚合为卷级参考节点
      const seg = preview?.segmentSize ?? 0
      const tierLabel = preview?.tier === 'long' ? '长篇' : preview?.tier === 'mid' ? '中篇' : '短篇'
      const action = seg > 0
        ? `篇幅路由（${tierLabel}）：已按每 ${seg} 章聚合为 ${items.length} 个卷级参考节点，应用将新建这些节点（不影响章节正文，可重复执行）。`
        : '将把每章的概要 / 场景 / 要点 / 情感基调合并进大纲（不覆盖章节正文，可重复执行）。'
      Modal.confirm({
        title: '应用 AI 反推大纲？',
        content: `${head}：共 ${items.length} ${seg > 0 ? '个卷级节点' : '章'}。${action}${meta ? `\n推断：${meta}` : ''}${warn}`,
        okText: '应用到大纲',
        cancelText: '取消',
        onOk: async () => {
          const n = await NovelOutlineReconstructApply(JSON.stringify(items))
          setStateMsg(seg > 0 ? `已应用 ${n} 个卷级参考节点` : `已应用 ${n} 章大纲`)
          await loadOutlines()
        },
      })
    } catch (err: unknown) {
      setStateMsg(err instanceof Error ? err.message : 'AI 反推失败')
    } finally {
      setReconstructBusy(false)
      setReconstructCancellable(false)
      setReconstructElapsed(0)
    }
  }, [loadOutlines, setStateMsg])

  // tail×反推串联（v4.292）：导入向导 tail 模式完成后由书架派发本事件，
  // 本页自动开跑反推（任务化后台执行）。
  // v4.421.0 门控：本页不可见时**挂起**而不是直接开跑——书架流程里
  // `novel:goto-tab{create}` 与 `novel:auto-reconstruct` 同 tick 连发，
  // 事件到达时 active prop 还没翻转，直接按 active 拦会把正常流程拦掉。
  const pendingReconstructRef = useRef(false)
  // 事件处理器读「当前是否可见」走 ref 镜像（闭包捕获的 prop 可能滞后一拍）
  const activeRef = useRef(active)
  activeRef.current = active
  React.useEffect(() => {
    if (!active || !pendingReconstructRef.current) return
    pendingReconstructRef.current = false
    void reconstructOutlines()
  }, [active, reconstructOutlines])

  React.useEffect(() => {
    const handler = () => {
      if (activeRef.current) { void reconstructOutlines(); return }
      pendingReconstructRef.current = true
    }
    window.addEventListener('novel:auto-reconstruct', handler)
    return () => window.removeEventListener('novel:auto-reconstruct', handler)
  }, [reconstructOutlines])

  return {
    novelState, stateBusy,
    loadState, proposeState, approveState,
    deslopChapter, llmDeslop,
    reconstructBusy, reconstructCancellable, reconstructElapsed,
    reconstructCancelRef, reconstructOutlines,
  }
}
