import React, { useCallback, useEffect, useRef, useState } from 'react'
import { message, Modal } from 'antd'
import * as App from '../../src/wailsjsCompat'
import { useOutlineStore } from '../stores/outlineStore'
import { useAppStore } from '../stores/appStore'
import type { OutlineNode } from '../types'
import { buildTree, flattenTree, buildPrevSummary } from '../components/novel/create/outlineTree'
import { useChapterStream } from '../components/novel/create/useChapterStream'
import ChapterTreePanel from '../components/novel/create/ChapterTreePanel'
import EditorPanel from '../components/novel/create/EditorPanel'
import CreateInspector from '../components/novel/create/CreateInspector'
import NewCharactersModal from '../components/novel/create/NewCharactersModal'
import BranchWizardModal, { type Branch } from '../components/novel/create/BranchWizardModal'

interface WizardRequest { prevChapter: number; overwriteChapter: number; branchFromID: string }
const BRAINSTORM_MSG_KEY = 'novel-brainstorm-loading'

// T6-7.5 拆分后的编排层（≤300 行）：持有页面状态与生成编排；视图拆到 5 个子组件；
// 流式事件经 useChapterStream + chapterStreamTypes 判别联合分发（T6-7.2 停止按钮 + cancelled）。
const CreatePage: React.FC = () => {
  const [setting, setSetting] = useState('')
  const outlines = useOutlineStore(s => s.outlines)
  const loadOutlines = useOutlineStore(s => s.loadOutlines)
  const projectPath = useAppStore(s => s.projectPath)

  const [activeId, setActiveId] = useState('')
  const [content, setContent] = useState('')
  const [chapterLoading, setChapterLoading] = useState(false)
  const [generating, setGenerating] = useState(false)
  const [saving, setSaving] = useState(false)
  const [genPhase, setGenPhase] = useState('')
  const [genPercent, setGenPercent] = useState(0)
  const [stopping, setStopping] = useState(false)

  const [minWords, setMinWords] = useState(5000)
  const [temperature, setTemperature] = useState(0)
  const [directPlot, setDirectPlot] = useState('')
  const [stats, setStats] = useState<{ totalWords: number; chapterCount: number } | null>(null)
  // 默认启用 story-deslop 去 AI 味润色技能；可在右侧创作设置中切换或清空
  const [selectedSkill, setSelectedSkill] = useState<string | undefined>('story-deslop')
  const [editorFontSize, setEditorFontSize] = useState<number>(() => {
    try {
      const v = Number(localStorage.getItem('gaea.novel.editorFontSize'))
      if (Number.isFinite(v) && v >= 12 && v <= 24) return v
    } catch { /* 读取失败按默认值 */ }
    return 15
  })
  const [wizard, setWizard] = useState<WizardRequest | null>(null)
  const [wizardBranches, setWizardBranches] = useState<Branch[]>([])

  const settingLoadToken = useRef(0)
  const chapterLoadToken = useRef(0)
  const generatingRef = useRef(false)
  const brainstormingRef = useRef(false)

  const handleEditorFontSizeChange = useCallback((v: number) => {
    setEditorFontSize(v)
    try { localStorage.setItem('gaea.novel.editorFontSize', String(v)) } catch { /* 持久化失败不影响使用 */ }
  }, [])

  // 当前生成任务的章节标识（取 CreateChapter 返回值，供停止按钮 CancelCreateChapter）
  const streamTargetRef = useRef({ chapterNum: 0, branch: '' })

  const { attach, detach } = useChapterStream()

  // 拉取最新小说设定；projectPath 变化（切换小说）时重新拉取
  const refreshSetting = useCallback(async () => {
    const token = ++settingLoadToken.current
    const requestedPath = useAppStore.getState().projectPath
    let fresh = ''
    try { fresh = await App.GetWorldview() || '' } catch { /* 设定拉取失败按空处理 */ }
    if (token !== settingLoadToken.current || requestedPath !== useAppStore.getState().projectPath) return ''
    setSetting(fresh)
    return fresh
  }, [])

  // 创作统计（章节数 / 总字数）
  const refreshStats = useCallback(async () => {
    try {
      const s = await App.GetStats()
      if (s) setStats(s as { totalWords: number; chapterCount: number })
    } catch { /* 统计失败不阻塞创作 */ }
  }, [])

  useEffect(() => {
    ;(async () => { loadOutlines(); await refreshSetting(); refreshStats() })()
  }, [projectPath, loadOutlines, refreshSetting, refreshStats])

  // 切换小说时清空编辑区与选中节点，避免展示上一个项目的内容
  useEffect(() => { setActiveId(''); setContent('') }, [projectPath])

  const selectChapter = async (node: OutlineNode) => {
    const token = ++chapterLoadToken.current
    const requestedPath = useAppStore.getState().projectPath
    setActiveId(node.id); setChapterLoading(true)
    try {
      const branch = node.branch || ''
      const result = await App.GetChapterBranch(node.order_index || 1, branch)
      if (token === chapterLoadToken.current && requestedPath === useAppStore.getState().projectPath) setContent(result?.content || '')
    } catch {
      if (token === chapterLoadToken.current && requestedPath === useAppStore.getState().projectPath) setContent('')
    } finally {
      if (token === chapterLoadToken.current) setChapterLoading(false)
    }
  }

  // 向导拉取 AI 构思分支（注入最新设定与前文摘要）
  const fetchWizardBranches = useCallback(async (prevChapter: number): Promise<Branch[]> => {
    const freshSetting = await refreshSetting()
    const prevSummary = prevChapter > 0 ? buildPrevSummary(outlines, prevChapter) : ''
    const res = await App.QuickBrainstormBranches(freshSetting, prevSummary || '')
    const list = res?.branches || []
    return list.map((b: { title?: string; summary?: string }) => ({ title: b.title ?? '', pitch: b.summary ?? '' }))
  }, [refreshSetting, outlines])

  // 后台构思剧情分支：不弹阻塞弹窗，构思完成后弹窗确认选择
  const openWizard = useCallback(async (prevChapter: number, overwriteChapter = 0, branchFromID = '') => {
    if (brainstormingRef.current) return
    brainstormingRef.current = true
    setWizard({ prevChapter, overwriteChapter, branchFromID })
    setWizardBranches([])
    message.open({ key: BRAINSTORM_MSG_KEY, content: 'AI 正在构思剧情分支，你可以继续操作…', duration: 0 })
    try {
      const list = await fetchWizardBranches(prevChapter)
      if (list.length === 0) {
        setWizard(null)
        message.warning('AI 未构思出剧情分支，可直接输入剧情要求')
        return
      }
      setWizardBranches(list)
    } catch (err: unknown) {
      setWizard(null)
      message.error(err instanceof Error ? err.message : '剧情构思失败，可手动输入剧情要求')
    } finally {
      message.destroy(BRAINSTORM_MSG_KEY)
      brainstormingRef.current = false
    }
  }, [fetchWizardBranches])

  // 流式生成收尾：三路终态（done/error/cancelled）与停止兜底共用
  const finishStream = useCallback(() => {
    setGenPhase(''); setGenPercent(0); setGenerating(false); setStopping(false)
    generatingRef.current = false
    detach()
  }, [detach])

  // 直接开始生成：注册流式监听并调用后端（带目标字数/温度/技能设置）
  const startGeneration = async (plotReq: string, overwriteChapter = 0, branchFromID = '') => {
    if (generatingRef.current) return
    if (!plotReq.trim()) { message.warning('请选择分支或输入剧情要求'); return }
    generatingRef.current = true
    setGenerating(true); setGenPhase('正在生成…'); setGenPercent(0); setContent(''); setStopping(false)

    attach({
      onEvent: (ev) => {
        switch (ev.type) {
          case 'phase':
            if (ev.phase === 'continuing') {
              setGenPhase(`字数不足，正在续写… 第${ev.attempt || 1}次 · ${(ev.current || 0).toLocaleString()}/${(ev.target || minWords).toLocaleString()} 字`)
            } else {
              setGenPhase(`正在生成… 目标 ${(ev.target || minWords).toLocaleString()} 字`)
            }
            break
          case 'chunk':
            setContent((prev) => prev + ev.content)
            setGenPercent(Math.min(100, Math.round(((ev.total || 0) / Math.max(minWords, 1)) * 100)))
            setGenPhase(`正在生成… ${(ev.total || 0).toLocaleString()}/${minWords.toLocaleString()} 字`)
            break
          case 'done': {
            finishStream()
            const chNum = ev.chapterNum || 0
            const branch = ev.branch || ''
            message.success(`${branch ? `第${chNum}${branch}章` : `第${chNum}章`} 生成完成（${(ev.total || 0).toLocaleString()} 字）`)
            refreshStats()
            loadOutlines().then(() => {
              const ch = useOutlineStore.getState().outlines.find(n => n.order_index === chNum && n.branch === branch)
              if (ch) setActiveId(ch.id)
            })
            break
          }
          case 'error':
            finishStream()
            message.error(ev.error || '生成失败')
            break
          case 'cancelled': {
            finishStream()
            // 取消：后端已落盘部分正文（事件携带 content）；保留编辑器已累积正文可继续编辑/保存
            if (typeof ev.content === 'string' && ev.content.length > 0) setContent(ev.content)
            const chNum = ev.chapterNum || 0
            const branch = ev.branch || ''
            message.info(`${branch ? `第${chNum}${branch}章` : `第${chNum}章`} 已停止生成（已保留 ${(ev.total || 0).toLocaleString()} 字）`)
            refreshStats()
            break
          }
        }
      },
    })

    try {
      // 生成前再读一次最新设定，确保正文提示词注入当前小说设定
      const freshSetting = await refreshSetting()
      if (!freshSetting.trim()) { throw new Error('小说设定为空，请先在「设定」页填写世界观') }
      const result = await App.CreateChapter(freshSetting, '', plotReq, overwriteChapter, branchFromID, selectedSkill || '', minWords, temperature)
      // 预创建节点已由后端同步完成，立即激活；记录章节号供停止按钮
      const nodeId = result?.nodeId
      const chapNum = result?.chapterNum
      streamTargetRef.current = { chapterNum: chapNum || 0, branch: result?.branch || '' }
      if (nodeId) {
        const store = useOutlineStore.getState()
        if (!store.outlines.find(n => n.id === nodeId)) {
          store.setOutlines([...store.outlines, { id: nodeId, order_index: chapNum, title: `第${chapNum}章`, status: 'writing', parent_id: '', summary: '' } as OutlineNode])
        }
        setActiveId(nodeId)
        setContent('')
      }
    } catch (err: unknown) {
      finishStream()
      message.error(err instanceof Error ? err.message : '生成失败')
    }
  }

  // CancelCreateChapter 契约（后端批 1）：wails build 重新生成 NovelB.d.ts 后自动可用，
  // 再生成前先用本地类型桥接，避免 tsc 报错（运行时不经过 bridge/mock，直调 window.go.app.NovelB）。
  const novelBridge = App as unknown as { CancelCreateChapter: (chapterNum: number, branch: string) => Promise<boolean> }

  // 停止生成：取 CreateChapter 返回值中的章节号 + 主线分支 ''（T6-7.2）
  const handleStop = async () => {
    const { chapterNum, branch } = streamTargetRef.current
    if (!chapterNum) { message.warning('暂无进行中的生成任务'); return }
    setStopping(true)
    try {
      const cancelled = await novelBridge.CancelCreateChapter(chapterNum, branch)
      if (cancelled) { setGenPhase('正在停止…') } // 随后收到 cancelled 事件收尾（保留部分正文）
      else { finishStream(); message.info('生成已结束') } // 幂等 false：本地兜底，避免 UI 悬挂
    } catch (err: unknown) {
      setStopping(false)
      message.error(err instanceof Error ? err.message : '取消失败')
    }
  }

  // 直接生成下一章（跳过分支向导）
  const handleDirectGenerate = () => {
    if (!directPlot.trim()) { message.warning('请输入剧情要求'); return }
    const chapNum = activeNode?.order_index || 0
    const next = outlines.find(nx => nx.order_index === chapNum + 1 && !nx.parent_id)
    if (next) {
      Modal.confirm({
        title: `第${chapNum + 1}章已存在「${next.title || `第${chapNum + 1}章`}」`,
        content: '选择生成方式：', okText: '覆盖下一章', cancelText: '作为分支追加',
        onOk: () => startGeneration(directPlot, chapNum + 1, ''),
        onCancel: () => startGeneration(directPlot, 0, next.id),
      })
    } else { startGeneration(directPlot, 0, '') }
  }

  const handleDelete = async (node: OutlineNode) => {
    try {
      await App.DeleteOutlineNode(node.id)
      if (activeId === node.id) { setActiveId(''); setContent('') }
      await loadOutlines(); refreshStats()
      message.success('已删除')
    } catch (err: unknown) { message.error(err instanceof Error ? err.message : '失败') }
  }

  const handleRegenerate = (node: OutlineNode) => openWizard((node.order_index || 1) - 1, node.order_index || 1)

  const handleSave = async () => {
    const node = outlines.find(n => n.id === activeId)
    if (!node || !content.trim()) return
    setSaving(true)
    try {
      const branch = node.branch || ''
      await App.SaveChapterBranchContent(node.order_index || 1, branch, content)
      message.success('已保存')
    } catch (err: unknown) { message.error(err instanceof Error ? err.message : '失败') }
    finally { setSaving(false) }
  }

  // 节点后生成下一章/分支（含覆盖确认）
  const handleAddNext = (node: OutlineNode) => {
    const chapNum = node.order_index || 1
    const next = outlines.find(nx => nx.order_index === chapNum + 1 && !nx.parent_id)
    if (next) {
      Modal.confirm({
        title: `第${chapNum}章后已有「${next.title || `第${chapNum + 1}章`}」`,
        content: '选择生成方式：', okText: '覆盖下一章', cancelText: '末尾追加',
        onOk: () => openWizard(chapNum, chapNum + 1),
        onCancel: () => openWizard(chapNum, 0, node.id),
      })
    } else { openWizard(chapNum, 0, '') }
  }

  const flatNodes = flattenTree(buildTree(outlines))
  const activeNode = outlines.find(n => n.id === activeId)
  const nextMainChapterNum = Math.max(0, ...outlines.filter(n => !n.parent_id).map(n => n.order_index || 0)) + 1
  const lastMainChapter = nextMainChapterNum - 1

  return (
    <div className="novel-workspace">
      <ChapterTreePanel flatNodes={flatNodes} activeId={activeId} nextChapterNum={nextMainChapterNum}
        onSelect={selectChapter} onRegenerate={handleRegenerate} onDelete={handleDelete}
        onAddNext={handleAddNext} onGenerateNext={() => openWizard(lastMainChapter)} />
      <div className="v3-grip" aria-hidden="true" />
      <EditorPanel activeNode={activeNode ?? null} content={content} onContentChange={setContent}
        chapterLoading={chapterLoading}
        generating={generating} genPhase={genPhase} genPercent={genPercent} stopping={stopping} saving={saving}
        onRegenerate={() => activeNode && handleRegenerate(activeNode)} onSave={handleSave} onStop={handleStop}
        hasChapters={flatNodes.length > 0} nextChapterNum={nextMainChapterNum} onOpenWizard={openWizard}
        editorFontSize={editorFontSize} onEditorFontSizeChange={handleEditorFontSizeChange} />
      <div className="v3-grip" aria-hidden="true" />
      <CreateInspector setting={setting} onRefreshSetting={() => void refreshSetting()}
        selectedSkill={selectedSkill} onSelectSkill={(v) => setSelectedSkill(v)}
        minWords={minWords} onMinWordsChange={setMinWords}
        temperature={temperature} onTemperatureChange={setTemperature}
        directPlot={directPlot} onDirectPlotChange={setDirectPlot}
        onDirectGenerate={handleDirectGenerate} onOpenWizard={openWizard}
        prevChapterHint={activeNode?.order_index || lastMainChapter}
        stats={stats} chapterCount={flatNodes.length} />
      <NewCharactersModal />
      <BranchWizardModal open={!!wizard && wizardBranches.length > 0}
        prevChapter={wizard?.prevChapter ?? 0} overwriteChapter={wizard?.overwriteChapter ?? 0}
        branchFromID={wizard?.branchFromID ?? ''} onClose={() => { setWizard(null); setWizardBranches([]) }}
        preloadedBranches={wizardBranches}
        onFetchBranches={fetchWizardBranches} onStart={startGeneration} />
    </div>
  )
}

export default CreatePage
