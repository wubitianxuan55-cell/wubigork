import sys
content = """import React, { useState, useEffect, useRef, useMemo } from 'react'
import {
  Typography, Button, Space, Tag, Input, Select, message, Tabs, Tooltip, Modal, Skeleton,
} from 'antd'
import type { TabsProps } from 'antd'
import {
  PlayCircleOutlined, SaveOutlined, LeftOutlined, RightOutlined,
  ThunderboltOutlined, BookOutlined,
  ExpandOutlined, CompressOutlined, SearchOutlined, BarChartOutlined, ReadOutlined,
  CheckCircleOutlined, BulbOutlined,
  UnorderedListOutlined,
} from '@ant-design/icons'
import { useAppStore } from '../stores/appStore'
import { useChapterStream } from '../hooks/useChapterStream'
import StatsModal from '../components/StatsModal'
import BookReviewModal from '../components/BookReviewModal'
import NextChapterModal from '../components/NextChapterModal'
import TTSPlayer from '../components/TTSPlayer'
import ChapterEditor from '../components/ChapterEditor'
import AIAssistSheet from '../components/AIAssistSheet'
import { findAllLeaves, sortNodes } from '../utils/outline'
import { loadOutlines as fetchOutlines } from '../api/outlines'
import type { OutlineNode, ChapterTabData } from '../types'
import { C, STATUS_COLORS, STATUS_LABELS } from '../utils/theme'

const ChapterPage: React.FC = () => {
  const [outlines, setOutlines] = useState<OutlineNode[]>([])
  const [storyThread, setStoryThread] = useState('')
  const [tabs, setTabs] = useState<ChapterTabData[]>([])
  const [activeKey, setActiveKey] = useState<string>('')
  const activeKeyRef = useRef(activeKey)
  const startTime = useRef(0)
  const [focusMode, setFocusMode] = useState(false)
  const [reviewVisible, setReviewVisible] = useState(false)
  const [reviewing, setReviewing] = useState(false)
  const [reviewData, setReviewData] = useState<any>(null)
  const [statsVisible, setStatsVisible] = useState(false)
  const [bookReviewVisible, setBookReviewVisible] = useState(false)
  const [nextChapterModal, setNextChapterModal] = useState(false)
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
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [focusMode])

  const projectPath = useAppStore((s) => s.projectPath)
  useEffect(() => {
    setTabs([]); setOutlines([]); setActiveKey('')
    if (projectPath) loadOutlines()
  }, [projectPath])

  useEffect(() => { activeKeyRef.current = activeKey }, [activeKey])

  const loadOutlines = async () => {
    const data = await fetchOutlines()
    if (data?.nodes) setOutlines(data.nodes)
    if (data?.story_thread !== undefined) setStoryThread(data.story_thread)
  }

  const sortedOutlines = useMemo(() => sortNodes(outlines), [outlines])
  const outlineLeaves = useMemo(() => findAllLeaves(sortedOutlines) as OutlineNode[], [sortedOutlines])

  const handleSelectNode = async (node: OutlineNode) => {
    const chNum = node.order_index || 0
    const key = node.id
    setActiveKey(key)
    if (tabs.some((t) => t.node.id === key)) return
    const newTab: ChapterTabData = { node, chapterNum: chNum, scenes: [''], summary: '', keyEvents: [], emotionTone: '', saved: false, generating: false, streamSpeed: 0, messages: [], targetWords: 3000, skillName: '', retryStatus: null }
    setTabs((prev) => [...prev, newTab])
    if (chNum > 0) {
      try {
        const result = await (window as any).go.app.App.GetChapter(chNum)
        if (result?.content) {
          updateTabByKey(key, 'scenes', [result.content]); updateTabByKey(key, 'saved', true)
        }
      } catch (e) { console.error('GetChapter failed:', e) }
    }
  }

  function updateTabByKey<K extends keyof ChapterTabData>(key: string, field: K, value: ChapterTabData[K]) {
    setTabs((prev) => { const i = prev.findIndex((t) => t.node.id === key); if (i < 0) return prev; const c = [...prev]; c[i] = { ...c[i], [field]: value }; return c })
  }

  const closeTab = (key: string) => {
    setTabs((prev) => { const n = prev.filter((t) => t.node.id !== key); if (key === activeKey && n.length > 0) setActiveKey(n[n.length - 1].node.id); return n })
  }

  const activeTab = tabs.find((t) => t.node.id === activeKey) ?? null

  function updateTab<K extends keyof ChapterTabData>(field: K, value: ChapterTabData[K]) {
    setTabs((prev) => { const i = prev.findIndex((t) => t.node.id === activeKey); if (i < 0) return prev; const c = [...prev]; c[i] = { ...c[i], [field]: value }; return c })
  }

  const handleGenerate = async () => {
    if (!activeTab) return
    updateTab('generating', true); updateTab('scenes', ['']); updateTab('streamSpeed', 0); updateTab('saved', false)
    startTime.current = Date.now()
    try { await (window as any).go.app.App.SaveOutlineNode(JSON.stringify({ ...activeTab.node, status: 'writing' })); loadOutlines() } catch (e) {}
    try { await (window as any).go.app.App.GenerateChapter(activeTab.node.id, activeTab.skillName, activeTab.targetWords) } catch (err: any) { updateTab('generating', false); updateTab('scenes', ['\\u274C ' + (err.message || err)]) }
  }

  const handleSave = async () => {
    if (!activeTab || activeTab.chapterNum < 1) return
    const c = activeTab.scenes.join('\\n\\n')
    if (!c) return
    try { await (window as any).go.app.App.SaveChapterContent(activeTab.chapterNum, c); updateTab('saved', true); message.success('Saved') } catch (e) { message.error('Save failed') }
  }

  const handleFinalize = async () => {
    if (!activeTab || activeTab.chapterNum < 1) return
    const c = activeTab.scenes.join('\\n\\n')
    if (!c) { message.warning('No content'); return }
    try { await (window as any).go.app.App.SaveChapterContent(activeTab.chapterNum, c); await (window as any).go.app.App.SaveOutlineNode(JSON.stringify({ ...activeTab.node, status: 'done' })); loadOutlines(); message.success('Finalized') } catch (e) { message.error('Failed') }
  }

  const handlePrevChapter = () => { const i = outlineLeaves.findIndex((n) => n.id === activeTab?.node.id); if (i > 0) handleSelectNode(outlineLeaves[i - 1]) }
  const handleNextChapter = () => { const i = outlineLeaves.findIndex((n) => n.id === activeTab?.node.id); if (i >= 0 && i < outlineLeaves.length - 1) handleSelectNode(outlineLeaves[i + 1]) }
  const getNextOutlineNode = (): OutlineNode | null => { const i = outlineLeaves.findIndex((n) => n.id === lastCompletedNode.current?.id); return (i >= 0 && i < outlineLeaves.length - 1) ? outlineLeaves[i + 1] : null }

  const handleReview = async () => {
    if (!activeTab?.chapterNum) return
    setReviewVisible(true); setReviewing(true); setReviewData(null)
    try { setReviewData(await (window as any).go.app.App.AnalyzeChapter(activeTab.chapterNum)) } catch (err: any) { message.error(err.message || 'Failed'); setReviewVisible(false) } finally { setReviewing(false) }
  }

  const getChapterStats = async () => {
    const stats: any[] = []
    for (const n of outlineLeaves) {
      const num = n.order_index || 0
      if (num < 1) continue
      try { const d = await (window as any).go.app.App.GetChapter(num); if (d) stats.push({ num, title: n.title, words: (d.content || '').length }) } catch (e) {}
    }
    return stats
  }

  const renderOutlineCards = () => {
    const cards: React.ReactNode[] = []
    const walk = (nodes: OutlineNode[], depth: number) => {
      nodes.forEach((n) => {
        const isVolume = (n.children?.length || 0) > 0 || n.id.startsWith('vol_')
        const isSelected = n.id === activeKey
        if (isVolume) {
          cards.push(<div key={n.id} style={{ padding: '6px 10px', marginTop: depth === 0 ? 0 : 4, background: 'rgba(192,132,252,0.08)', borderLeft: '3px solid #c084fc', borderRadius: '0 4px 4px 0', fontSize: 12, fontWeight: 600, color: '#c084fc', display: 'flex', alignItems: 'center', gap: 6 }}><BookOutlined style={{ fontSize: 11 }} /><span style={{ flex: 1 }}>{n.title}</span></div>)
        } else {
          cards.push(<div key={n.id} onClick={() => handleSelectNode(n)} style={{ padding: '6px 10px', margin: '2px 0', cursor: 'pointer', background: isSelected ? 'rgba(192,132,252,0.08)' : 'transparent', borderRadius: '0 var(--radius-sm) var(--radius-sm) 0', borderLeft: isSelected ? '3px solid #c084fc' : '3px solid transparent' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}><span style={{ color: '#c084fc', fontSize: 10, fontWeight: 600 }}>{n.order_index || '\\u00B7'}</span><span style={{ flex: 1, fontSize: 12 }}>{n.title}</span></div>
          </div>)
        }
        if (n.children?.length) walk(n.children, depth + 1)
      })
    }
    walk(sortedOutlines, 0)
    return cards
  }

  const tabItems: TabsProps['items'] = tabs.map((tab) => ({
    key: tab.node.id,
    label: <Tooltip title={tab.summary || tab.node.title}><Space size={2}>{tab.generating ? <ThunderboltOutlined style={{ color: '#4ade80', fontSize: 11 }} /> : null}<span>{tab.chapterNum > 0 ? 'Ch' + tab.chapterNum : tab.node.title.slice(0, 8)}</span></Space></Tooltip>,
    closable: true,
  }))

  const totalWords = activeTab ? [...activeTab.scenes.join('\\n\\n')].length : 0

  return (
    <div style={{ display: 'flex', flexDirection: isMobile ? 'column' : 'row', height: isMobile ? '100%' : 'calc(100vh - 112px)', gap: 12 }}>
    <div style={{ display: 'flex', flexDirection: 'row', height: 'calc(100vh - 112px)', gap: 12 }}>
      {
        <div style={{ padding: '10px 14px', borderBottom: '1px solid ' + C('color-border') }}><Typography.Text strong style={{ color: C('color-text'), fontSize: 13 }}><UnorderedListOutlined style={{ marginRight: 6 }} />Outline</Typography.Text></div>
        <div style={{ flex: 1, overflow: 'auto', padding: 6 }}>{outlines.length === 0 ? <div style={{ textAlign: 'center', color: C('color-text-secondary'), marginTop: 32, fontSize: 12 }}>No outline</div> : renderOutlineCards()}</div>
      </div>
      )}
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', gap: 8, minWidth: 0 }}>
        {!activeTab ? (
          <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', background: C('color-bg-container'), borderRadius: 8, border: '1px solid ' + C('color-border') }}>
            <div style={{ textAlign: 'center', color: C('color-text-secondary') }}><BookOutlined style={{ fontSize: 48, marginBottom: 16 }} /><div>Select a chapter from the outline</div></div>
          </div>
        ) : (
          <>
            <div style={{ flex: 1, display: 'flex', flexDirection: 'column', background: C('color-bg-container'), borderRadius: 8, border: '1px solid ' + C('color-border'), overflow: 'hidden' }}>
              <div style={{ borderBottom: '1px solid ' + C('color-border'), display: 'flex', alignItems: 'center', paddingRight: 8 }}>
                <Tabs activeKey={activeKey} onChange={setActiveKey} onEdit={(key, action) => { if (action === 'remove' && typeof key === 'string') closeTab(key) }} items={tabItems} type="editable-card" size="small" style={{ flex: 1, marginBottom: 0 }} tabBarStyle={{ marginBottom: 0 }} />
                <Space size={4} style={{ flexShrink: 0 }}>
                  <Button size="small" icon={<LeftOutlined />} onClick={handlePrevChapter} type="text" />
                  <Button size="small" icon={<RightOutlined />} onClick={handleNextChapter} type="text" />
                  <Button size="small" icon={<BulbOutlined />} onClick={() => setNextChapterModal(true)} type="text" style={{ color: '#c084fc' }}>Suggest</Button>
                  {activeTab.generating && <Tag color="green" style={{ fontSize: 9 }}>{activeTab.streamSpeed}cps</Tag>}
                </Space>
              </div>
              <div style={{ padding: '4px 12px', display: 'flex', alignItems: 'center', gap: 8, borderBottom: '1px solid ' + C('color-border') }}>
                <TTSPlayer getText={() => activeTab?.scenes?.join('\\n\\n') || ''} />
                <div style={{ flex: 1 }} />
                <Button size="small" icon={<SaveOutlined />} onClick={handleSave} disabled={!totalWords}>Save</Button>
                <Button size="small" icon={<CheckCircleOutlined />} onClick={handleFinalize} disabled={!totalWords}>Finalize</Button>
                <Button type="primary" size="small" icon={<PlayCircleOutlined />} onClick={handleGenerate} loading={activeTab.generating}>Generate</Button>
                <Button size="small" icon={<SearchOutlined />} onClick={handleReview}>Review</Button>
                <Button size="small" icon={<BarChartOutlined />} onClick={() => setStatsVisible(true)}>Stats</Button>
              </div>
              <ChapterEditor tab={activeTab} onUpdate={updateTab} sceneTextareaRefs={sceneTextareaRefs} ghostEnabled={ghostEnabled} />
            </div>
            <AIAssistSheet tab={activeTab} onUpdate={updateTab} autoSendMsg={autoSendMsg} onAutoSendDone={() => setAutoSendMsg('')} />
          </>
        )}
      </div>
      <Modal title='AI Review' open={reviewVisible} onCancel={() => setReviewVisible(false)} footer={null} width={640}>
        {reviewing ? <Skeleton active /> : reviewData ? <div>Score: {reviewData.qualityScore}/10</div> : null}
      </Modal>
      <StatsModal open={statsVisible} onClose={() => setStatsVisible(false)} getChapterStats={getChapterStats} totalWords={totalWords} chapterCount={findAllLeaves(sortNodes(outlines)).length} />
      <BookReviewModal open={bookReviewVisible} onClose={() => setBookReviewVisible(false)} />
      <NextChapterModal open={nextChapterModal} onClose={() => setNextChapterModal(false)} currentNode={lastCompletedNode.current} nextNode={getNextOutlineNode()} onGenerate={async (id: string) => { await handleSelectNode(outlineLeaves.find((n) => n.id === id)!); setTabs((prev) => { const i = prev.findIndex((t) => t.node.id === id); if (i < 0) return prev; const c = [...prev]; c[i] = { ...c[i], generating: true, scenes: [''] }; return c }); startTime.current = Date.now(); try { await (window as any).go.app.App.GenerateChapter(id, '', 3000) } catch (err: any) { message.error('Failed: ' + (err.message || err)); setTabs((prev) => { const i = prev.findIndex((t) => t.node.id === id); if (i < 0) return prev; const c = [...prev]; c[i] = { ...c[i], generating: false }; return c }) } }} />
    </div>
  )
}

export default ChapterPage
"""

import os
path = 'D:\\AI\\wubigork\\frontend\\src\\pages\\ChapterPage.tsx'
with open(path, 'w', encoding='utf-8', newline='\n') as f:
    f.write(content)
print(f'Written {len(content)} bytes to {path}')
