import React, { useState, useEffect, useRef, useMemo } from 'react'
import {
  Typography, Button, Space, Tag, message, Tabs, Tooltip, Modal, Skeleton, Divider,
} from 'antd'
import type { TabsProps } from 'antd'
import {
  PlayCircleOutlined, SaveOutlined, LeftOutlined, RightOutlined,
  ThunderboltOutlined, BookOutlined,
  SearchOutlined, BarChartOutlined,
  CheckCircleOutlined,
} from '@ant-design/icons'
import {
  GetChapter, GenerateChapter, SaveChapterContent,
  SaveOutlineNode, AnalyzeChapter,
  GetWorldview,
} from '../../wailsjs/go/app/App'
import PlotBranchModal from '../components/PlotBranchModal'
import { useAppStore } from '../stores/appStore'
import { useChapterStream } from '../hooks/useChapterStream'
import StatsModal from '../components/StatsModal'
import BookReviewModal from '../components/BookReviewModal'
import TTSPlayer from '../components/TTSPlayer'
import ChapterEditor from '../components/ChapterEditor'
import AIAssistSheet from '../components/AIAssistSheet'
import OutlinePanel from '../components/OutlinePanel'
import ReviewResult from '../components/ReviewResult'
import { findAllLeaves, sortNodes } from '../utils/outline'
import { useOutlineStore } from '../stores/outlineStore'
import type { OutlineNode, ChapterTabData } from '../types'
import { C } from '../utils/theme'

/** 创建空的 ChapterTabData */
function createTabData(node: OutlineNode): ChapterTabData {
  return {
    node, chapterNum: node.order_index || 0,
    scenes: [''], summary: '', keyEvents: [],
    emotionTone: '', saved: false, generating: false,
    streamSpeed: 0, messages: [], targetWords: 3000,
    skillName: '', retryStatus: null,
  }
}

const ChapterPage: React.FC = () => {
  const outlines = useOutlineStore((s) => s.outlines)
  const storyThread = useOutlineStore((s) => s.storyThread)
  const loadOutlines = useOutlineStore((s) => s.loadOutlines)
  const [tabs, setTabs] = useState<ChapterTabData[]>([])
  const [activeKey, setActiveKey] = useState<string>('')
  const activeKeyRef = useRef(activeKey)
  const startTime = useRef(0)
  const [focusMode, setFocusMode] = useState(false)
  const [outlineCollapsed, setOutlineCollapsed] = useState(false)
  const handleSaveRef = useRef<() => void>(() => {})
  const [reviewVisible, setReviewVisible] = useState(false)
  const [reviewing, setReviewing] = useState(false)
  const [reviewData, setReviewData] = useState<any>(null)
  const [statsVisible, setStatsVisible] = useState(false)
  const [bookReviewVisible, setBookReviewVisible] = useState(false)
  const [branchModalOpen, setBranchModalOpen] = useState(false)
  const [branchNodeID, setBranchNodeID] = useState('')
  const lastCompletedNode = useRef<OutlineNode | null>(null)
  const modalTimerRef = useRef<number>(0)
  const [ghostEnabled, setGhostEnabled] = useState(true)
  const [autoSendMsg, setAutoSendMsg] = useState('')
  const sceneTextareaRefs = useRef<Map<number, HTMLTextAreaElement>>(new Map())

  useChapterStream(tabs, setTabs, activeKeyRef, startTime, modalTimerRef, lastCompletedNode)

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && focusMode) { setFocusMode(false); return }
      if (e.key === 'F11') { e.preventDefault(); setFocusMode((p) => !p) }
      if ((e.ctrlKey || e.metaKey) && e.key === 's') {
        e.preventDefault()
        handleSaveRef.current()
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [focusMode])

  const projectPath = useAppStore((s) => s.projectPath)
  useEffect(() => {
    setTabs([]); setActiveKey('')
    if (projectPath) {
      loadOutlines()
      // 检查小说设定，无则跳转
      ;(async () => {
        try {
          const wv = await GetWorldview()
          if (!wv || !wv.trim()) {
            message.warning('请先完善小说设定')
            window.dispatchEvent(new CustomEvent('navigate', { detail: { page: 'novelsetting' } }))
          }
        } catch (_) { /* 无设定则忽略 */ }
      })()
    }
  }, [projectPath])

  useEffect(() => { activeKeyRef.current = activeKey }, [activeKey])

  const sortedOutlines = useMemo(() => sortNodes(outlines), [outlines])
  const outlineLeaves = useMemo(() => findAllLeaves(sortedOutlines) as OutlineNode[], [sortedOutlines])

  const handleSelectNode = async (node: OutlineNode) => {
    const chNum = node.order_index || 0
    const key = node.id
    setActiveKey(key)
    if (tabs.some((t) => t.node.id === key)) return
    const newTab = createTabData(node)
    setTabs((prev) => [...prev, newTab])
    if (chNum > 0) {
      try {
        const result = await GetChapter(chNum)
        if (result?.content) {
          updateTabByKey(key, 'scenes', [result.content])
          updateTabByKey(key, 'saved', true)
        }
      } catch (e) { console.error('GetChapter failed:', e) }
    }
  }

  function updateTabByKey<K extends keyof ChapterTabData>(key: string, field: K, value: ChapterTabData[K]) {
    setTabs((prev) => {
      const i = prev.findIndex((t) => t.node.id === key)
      if (i < 0) return prev
      const c = [...prev]
      c[i] = { ...c[i], [field]: value }
      return c
    })
  }

  const closeTab = (key: string) => {
    setTabs((prev) => {
      const n = prev.filter((t) => t.node.id !== key)
      if (key === activeKey && n.length > 0) setActiveKey(n[n.length - 1].node.id)
      return n
    })
  }

  const activeTab = tabs.find((t) => t.node.id === activeKey) ?? null

  function updateTab<K extends keyof ChapterTabData>(field: K, value: ChapterTabData[K]) {
    setTabs((prev) => {
      const i = prev.findIndex((t) => t.node.id === activeKey)
      if (i < 0) return prev
      const c = [...prev]
      c[i] = { ...c[i], [field]: value }
      return c
    })
  }

  const handleGenerate = async () => {
    if (!activeTab) return
    setBranchNodeID(activeTab.node.id)
    setBranchModalOpen(true)
  }

  /** 分支选择后 → 生成章节 */
  const handleBranchApplied = async () => {
    setBranchModalOpen(false)
    if (!branchNodeID) return
    const node = outlineLeaves.find((n) => n.id === branchNodeID)
    if (!node) return
    await handleSelectNode(node)
    setTabs((prev) => {
      const i = prev.findIndex((t) => t.node.id === branchNodeID)
      if (i < 0) return prev
      const c = [...prev]
      c[i] = { ...c[i], generating: true, scenes: [''], streamSpeed: 0, saved: false }
      return c
    })
    startTime.current = Date.now()
    try {
      await SaveOutlineNode(JSON.stringify({ ...node, status: 'writing' }))
      loadOutlines()
    } catch (e) {}
    try {
      await GenerateChapter(branchNodeID, '', 3000)
    } catch (err: any) {
      setTabs((prev) => {
        const i = prev.findIndex((t) => t.node.id === branchNodeID)
        if (i < 0) return prev
        const c = [...prev]
        c[i] = { ...c[i], generating: false, scenes: ['❌ ' + (err.message || err)] }
        return c
      })
    }
  }

  const handleSave = async () => {
    if (!activeTab || activeTab.chapterNum < 1) return
    const c = activeTab.scenes.join('\n\n')
    if (!c) return
    try {
      await SaveChapterContent(activeTab.chapterNum, c)
      updateTab('saved', true)
      message.success('已保存')
    } catch (e) { message.error('保存失败') }
  }
  handleSaveRef.current = handleSave

  const handleFinalize = async () => {
    if (!activeTab || activeTab.chapterNum < 1) return
    const c = activeTab.scenes.join('\n\n')
    if (!c) { message.warning('暂无内容'); return }
    try {
      await SaveChapterContent(activeTab.chapterNum, c)
      await SaveOutlineNode(JSON.stringify({ ...activeTab.node, status: 'done' }))
      loadOutlines()
      message.success('已定稿')
    } catch (e) { message.error('操作失败') }
  }

  const handlePrevChapter = () => {
    const i = outlineLeaves.findIndex((n) => n.id === activeTab?.node.id)
    if (i > 0) handleSelectNode(outlineLeaves[i - 1])
  }

  const handleNextChapter = () => {
    const i = outlineLeaves.findIndex((n) => n.id === activeTab?.node.id)
    if (i >= 0 && i < outlineLeaves.length - 1) handleSelectNode(outlineLeaves[i + 1])
  }

  const handleReview = async () => {
    if (!activeTab?.chapterNum) return
    setReviewVisible(true)
    setReviewing(true)
    setReviewData(null)
    try {
      setReviewData(await AnalyzeChapter(activeTab.chapterNum))
    } catch (err: any) {
      message.error(err.message || '操作失败')
      setReviewVisible(false)
    } finally {
      setReviewing(false)
    }
  }

  const getChapterStats = async () => {
    const stats: any[] = []
    for (const n of outlineLeaves) {
      const num = n.order_index || 0
      if (num < 1) continue
      try {
        const d = await GetChapter(num)
        if (d) stats.push({ num, title: n.title, words: (d.content || '').length })
      } catch (e) {}
    }
    return stats
  }

  const tabItems: TabsProps['items'] = tabs.map((tab) => ({
    key: tab.node.id,
    label: (
      <Tooltip title={tab.summary || tab.node.title}>
        <Space size={2}>
          {tab.generating ? <ThunderboltOutlined style={{ color: '#4ade80', fontSize: 11 }} /> : null}
          <span>{tab.chapterNum > 0 ? '第' + tab.chapterNum + '章' : tab.node.title.slice(0, 8)}</span>
        </Space>
      </Tooltip>
    ),
    closable: true,
  }))

  const totalWords = activeTab ? [...activeTab.scenes.join('\n\n')].length : 0
  const outlineWidth = outlineCollapsed ? 40 : 220

  return (
    <div style={{ display: 'flex', flexDirection: 'row', height: 'calc(100vh - 112px)', gap: 12 }}>
      {/* ── 大纲面板（桌面端，专注模式下隐藏）── */}
      {!focusMode && (
        <OutlinePanel
          outlines={sortedOutlines}
          activeKey={activeKey}
          onSelectNode={handleSelectNode}
          collapsed={outlineCollapsed}
          onToggleCollapse={() => setOutlineCollapsed(!outlineCollapsed)}
        />
      )}

      {/* ── 主编辑区 ── */}
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', gap: 8, minWidth: 0 }}>
        {!activeTab ? (
          <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', background: C('color-bg-container'), borderRadius: 8, border: '1px solid ' + C('color-border') }}>
            <div style={{ textAlign: 'center', color: C('color-text-secondary') }}>
              <BookOutlined style={{ fontSize: 48, marginBottom: 16 }} />
              <div>从左侧大纲选择章节开始写作</div>
            </div>
          </div>
        ) : (
          <>
            <div style={{ flex: 1, display: 'flex', flexDirection: 'column', background: C('color-bg-container'), borderRadius: 8, border: '1px solid ' + C('color-border'), overflow: 'hidden' }}>
              {/* 标签栏 + 前后章节按钮 */}
              {!focusMode && (
                <div style={{ borderBottom: '1px solid ' + C('color-border'), display: 'flex', alignItems: 'center', paddingRight: 8 }}>
                  <Tabs activeKey={activeKey} onChange={setActiveKey}
                    onEdit={(key, action) => { if (action === 'remove' && typeof key === 'string') closeTab(key) }}
                    items={tabItems} type="editable-card" size="small"
                    style={{ flex: 1, marginBottom: 0 }} tabBarStyle={{ marginBottom: 0 }}
                  />
                  <Space size={4} style={{ flexShrink: 0 }}>
                    <Button size="small" icon={<LeftOutlined />} onClick={handlePrevChapter} type="text" />
                    <Button size="small" icon={<RightOutlined />} onClick={handleNextChapter} type="text" />
                    {activeTab.generating && <Tag color="green" style={{ fontSize: 9 }}>{activeTab.streamSpeed}cps</Tag>}
                  </Space>
                </div>
              )}

              {/* 章节信息摘要栏 */}
              {!focusMode && (
                <div style={{ padding: '6px 12px', display: 'flex', alignItems: 'center', gap: 8, borderBottom: '1px solid ' + C('color-border'), fontSize: 12, background: 'rgba(192,132,252,0.03)' }}>
                  <BookOutlined style={{ color: '#c084fc', fontSize: 11 }} />
                  <span style={{ fontWeight: 600, color: C('color-text') }}>{activeTab.node.title}</span>
                  <span style={{ color: C('color-text-secondary') }}>· {totalWords.toLocaleString()} 字</span>
                  <div style={{ flex: 1 }} />
                  <Tooltip title={activeTab.saved ? '内容已保存' : '内容有未保存的修改'}>
                    <span style={{ display: 'flex', alignItems: 'center', gap: 5, color: activeTab.saved ? '#4ade80' : '#f59e0b', fontSize: 11 }}>
                      <span style={{ width: 6, height: 6, borderRadius: '50%', background: activeTab.saved ? '#4ade80' : '#f59e0b', display: 'inline-block' }} />
                      {activeTab.saved ? '已保存' : '未保存'}
                    </span>
                  </Tooltip>
                </div>
              )}

              {/* 工具栏 */}
              {!focusMode && (
                <div style={{ padding: '4px 12px', display: 'flex', alignItems: 'center', gap: 4, borderBottom: '1px solid ' + C('color-border') }}>
                  <TTSPlayer getText={() => activeTab?.scenes?.join('\n\n') || ''} />
                  <div style={{ flex: 1 }} />
                  <Tooltip title="Ctrl+S"><Button size="small" icon={<SaveOutlined />} onClick={handleSave} disabled={!totalWords}>保存</Button></Tooltip>
                  <Button size="small" icon={<CheckCircleOutlined />} onClick={handleFinalize} disabled={!totalWords}>定稿</Button>
                  <Divider type="vertical" style={{ margin: '0 4px', borderColor: C('color-border'), height: 18 }} />
                  <Button type="primary" size="small" icon={<PlayCircleOutlined />} onClick={handleGenerate} loading={activeTab.generating}>生成</Button>
                  <Button size="small" icon={<SearchOutlined />} onClick={handleReview}>审稿</Button>
                  <Divider type="vertical" style={{ margin: '0 4px', borderColor: C('color-border'), height: 18 }} />
                  <Button size="small" icon={<BarChartOutlined />} onClick={() => setStatsVisible(true)}>统计</Button>
                </div>
              )}

              {/* 编辑器主体 */}
              <ChapterEditor tab={activeTab} onUpdate={updateTab} sceneTextareaRefs={sceneTextareaRefs} ghostEnabled={ghostEnabled} />
            </div>

            {/* AI 协写面板 */}
            {!focusMode && (
              <AIAssistSheet tab={activeTab} onUpdate={updateTab} autoSendMsg={autoSendMsg} onAutoSendDone={() => setAutoSendMsg('')} />
            )}

            {/* 底部快捷键提示栏 */}
            <div style={{ padding: '2px 12px', display: 'flex', alignItems: 'center', gap: 16, fontSize: 11, color: C('color-text-secondary'), opacity: 0.7 }}>
              <span><kbd style={{ border: '1px solid ' + C('color-border'), borderRadius: 3, padding: '0 4px', fontSize: 10, fontFamily: 'monospace' }}>F11</kbd> 专注模式</span>
              <span><kbd style={{ border: '1px solid ' + C('color-border'), borderRadius: 3, padding: '0 4px', fontSize: 10, fontFamily: 'monospace' }}>Ctrl+S</kbd> 保存</span>
              <span><kbd style={{ border: '1px solid ' + C('color-border'), borderRadius: 3, padding: '0 4px', fontSize: 10, fontFamily: 'monospace' }}>Ctrl+K</kbd> AI 编辑</span>
              {focusMode && (
                <>
                  <div style={{ flex: 1 }} />
                  <span>{activeTab.node.title} · {totalWords.toLocaleString()} 字</span>
                  <span style={{ color: '#f59e0b' }}>专注模式已开启 · Esc 退出</span>
                </>
              )}
            </div>
          </>
        )}
      </div>

      {/* ── 弹窗层 ── */}
      <Modal title="AI 审稿" open={reviewVisible} onCancel={() => setReviewVisible(false)} footer={null} width={640}>
        {reviewing ? <Skeleton active /> : reviewData ? <ReviewResult data={reviewData} /> : null}
      </Modal>
      <StatsModal open={statsVisible} onClose={() => setStatsVisible(false)}
        getChapterStats={getChapterStats}
        totalWords={totalWords}
        chapterCount={findAllLeaves(sortNodes(outlines)).length}
      />
      <BookReviewModal open={bookReviewVisible} onClose={() => setBookReviewVisible(false)} />

      {/* 剧情分支选择 */}
      <PlotBranchModal
        open={branchModalOpen}
        onClose={() => setBranchModalOpen(false)}
        nodeID={branchNodeID}
        nodeTitle={activeTab?.node.title || ''}
        onApplied={handleBranchApplied}
      />
    </div>
  )
}

export default ChapterPage
