import React, { useState, useEffect, useRef, useMemo } from 'react'
import {
  Typography, Button, Space, Tag, message, Tabs, Tooltip,
} from 'antd'
import type { TabsProps } from 'antd'
import {
  SaveOutlined, LeftOutlined, RightOutlined,
  BookOutlined,
} from '@ant-design/icons'
import {
  GetChapter, SaveChapterContent,
} from '../../wailsjs/go/app/App'
import { useAppStore } from '../stores/appStore'
import TTSPlayer from '../components/TTSPlayer'
import ChapterEditor from '../components/ChapterEditor'
import OutlinePanel from '../components/OutlinePanel'
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
  const loadOutlines = useOutlineStore((s) => s.loadOutlines)
  const [tabs, setTabs] = useState<ChapterTabData[]>([])
  const [activeKey, setActiveKey] = useState<string>('')
  const [focusMode, setFocusMode] = useState(false)
  const [outlineCollapsed, setOutlineCollapsed] = useState(false)
  const handleSaveRef = useRef<() => void>(() => {})
  const sceneTextareaRefs = useRef<Map<number, HTMLTextAreaElement>>(new Map())

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
    }
  }, [projectPath])

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

  // 前后章节导航
  const handlePrevChapter = async () => {
    if (!activeTab) return
    const idx = outlineLeaves.findIndex((n) => n.id === activeTab.node.id)
    if (idx > 0) handleSelectNode(outlineLeaves[idx - 1])
  }
  const handleNextChapter = async () => {
    if (!activeTab) return
    const idx = outlineLeaves.findIndex((n) => n.id === activeTab.node.id)
    if (idx < outlineLeaves.length - 1) handleSelectNode(outlineLeaves[idx + 1])
  }

  const totalWords = activeTab?.scenes?.join('\n').length || 0

  const tabItems: TabsProps['items'] = tabs.map((t) => ({
    key: t.node.id, label: t.node.title,
    closable: true,
  }))

  return (
    <div style={{ height: '100%', display: 'flex', flexDirection: 'column', gap: 8 }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <Typography.Title level={4} style={{ color: C('color-text'), margin: 0 }}>
          <BookOutlined style={{ marginRight: 8 }} />阅读
        </Typography.Title>
      </div>

      <div style={{ flex: 1, display: 'flex', gap: 8, minHeight: 0 }}>
        {/* 左：大纲 */}
        {!focusMode && (
          <OutlinePanel
            outlines={sortedOutlines}
            activeKey={activeTab?.node.id || ''}
            onSelectNode={handleSelectNode}
            collapsed={outlineCollapsed}
            onToggleCollapse={() => setOutlineCollapsed((p) => !p)}
          />
        )}

        {/* 右：正文 */}
        {!activeTab ? (
          <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', color: C('color-text-secondary'), opacity: 0.5 }}>
            <div style={{ textAlign: 'center' }}>
              <BookOutlined style={{ fontSize: 48, marginBottom: 16 }} />
              <div>从左侧大纲选择章节开始阅读</div>
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
                  </Space>
                </div>
              )}

              {/* 章节信息栏 */}
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
                </div>
              )}

              {/* 编辑器主体 */}
              <ChapterEditor tab={activeTab} onUpdate={updateTab} sceneTextareaRefs={sceneTextareaRefs} ghostEnabled={false} />
            </div>

            {/* 底部快捷键提示栏 */}
            <div style={{ padding: '2px 12px', display: 'flex', alignItems: 'center', gap: 16, fontSize: 11, color: C('color-text-secondary'), opacity: 0.7 }}>
              <span><kbd style={{ border: '1px solid ' + C('color-border'), borderRadius: 3, padding: '0 4px', fontSize: 10, fontFamily: 'monospace' }}>F11</kbd> 专注模式</span>
              <span><kbd style={{ border: '1px solid ' + C('color-border'), borderRadius: 3, padding: '0 4px', fontSize: 10, fontFamily: 'monospace' }}>Ctrl+S</kbd> 保存</span>
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
    </div>
  )
}

export default ChapterPage
