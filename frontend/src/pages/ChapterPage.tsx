import React, { useState, useEffect, useRef, useMemo } from 'react'
import {
  Button, Space, message, Modal, Tabs, Tooltip,
} from 'antd'
import type { TabsProps } from 'antd'
import {
  SaveOutlined, LeftOutlined, RightOutlined,
  BookOutlined, EditOutlined, ReadOutlined, ExpandOutlined, ShrinkOutlined,
} from '@ant-design/icons'
import {
  GetChapter, GetChapterBranch, SaveChapterContent, SaveChapterBranchContent,
} from '../../src/wailsjsCompat'
import { useAppStore } from '../stores/appStore'
import TTSPlayer from '../components/TTSPlayer'
import ChapterEditor from '../components/novel/ChapterEditor'
import { findAllLeaves, sortNodes } from '../utils/outline'
import { useOutlineStore } from '../stores/outlineStore'
import { countTextChars } from '../utils/text'
import { readReadingProgress, writeReadingProgress } from '../utils/readingProgress'
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
  const [readMode, setReadMode] = useState(false)
  // 当前激活章节（须在首个引用它的 useEffect 依赖数组之前定义，避免 TDZ）
  const activeTab = tabs.find((t) => t.node.id === activeKey) ?? null
  const handleSaveRef = useRef<() => void>(() => {})
  const sceneTextareaRefs = useRef<Map<number, HTMLTextAreaElement>>(new Map())

  // 世界构建工作台：大纲树位于壳层左 zone，点击经 novel:open-chapter 事件进入
  const handleSelectNodeRef = useRef<(node: OutlineNode) => void>(() => {})
  useEffect(() => { handleSelectNodeRef.current = handleSelectNode })
  useEffect(() => {
    const handler = (e: Event) => {
      const node = (e as CustomEvent<{ node?: OutlineNode }>).detail?.node
      if (node) void handleSelectNodeRef.current(node)
    }
    window.addEventListener('novel:open-chapter', handler)
    return () => window.removeEventListener('novel:open-chapter', handler)
  }, [])

  // 上报当前章节属性 → 壳层右 zone（属性检查器）与大纲激活项
  useEffect(() => {
    if (!activeTab) {
      window.dispatchEvent(new CustomEvent('novel:chapter-active', { detail: {} }))
      return
    }
    window.dispatchEvent(new CustomEvent('novel:chapter-active', {
      detail: {
        id: activeTab.node.id,
        title: activeTab.node.title,
        words: countTextChars(activeTab.scenes.join('\n')),
        saved: activeTab.saved,
        status: activeTab.node.status,
      },
    }))
  }, [activeTab])

  // 专注模式：通知壳层收起左右 zone
  useEffect(() => {
    window.dispatchEvent(new CustomEvent('novel:focus-mode', { detail: { active: focusMode } }))
  }, [focusMode])

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        if (readMode) { setReadMode(false); return }
        if (focusMode) { setFocusMode(false); return }
      }
      if (e.key === 'F11') { e.preventDefault(); setFocusMode((p) => !p) }
      if ((e.ctrlKey || e.metaKey) && e.key === 's') {
        e.preventDefault()
        handleSaveRef.current()
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [focusMode, readMode])

  const projectPath = useAppStore((s) => s.projectPath)
  useEffect(() => {
    setTabs([]); setActiveKey(''); setReadMode(false)
    if (projectPath) {
      void loadOutlines()
        .then(() => {
          const progress = readReadingProgress(projectPath)
          if (!progress) return
          const node = findAllLeaves(sortNodes(useOutlineStore.getState().outlines))
            .find((n) => n.id === progress.nodeId)
          if (!node) return
          setActiveKey(node.id)
          setTabs([createTabData(node)])
          const chNum = node.order_index || 0
          if (chNum > 0) {
            const load = node.branch ? GetChapterBranch(chNum, node.branch) : GetChapter(chNum)
            load.then((result) => {
              if (useAppStore.getState().projectPath !== projectPath || !result?.content) return
              updateTabByKey(node.id, 'scenes', [result.content])
              updateTabByKey(node.id, 'saved', true)
            }).catch((e) => console.error('GetChapter failed:', e))
          }
        })
        .catch((e) => console.error('loadOutlines failed:', e))
    }
  }, [projectPath, loadOutlines])

  // 记住当前项目最后阅读的章节，下一次切回该书时自动恢复
  useEffect(() => {
    if (!projectPath || !activeKey) return
    const node = findAllLeaves(sortNodes(outlines)).find((n) => n.id === activeKey)
    if (!node) return
    writeReadingProgress(projectPath, {
      nodeId: node.id,
      chapterNum: node.order_index || 0,
      title: node.title || `第${node.order_index || '?'}章`,
    })
  }, [projectPath, activeKey, outlines])

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
      const requestedPath = projectPath
      try {
        const result = node.branch ? await GetChapterBranch(chNum, node.branch) : await GetChapter(chNum)
        if (requestedPath !== useAppStore.getState().projectPath) return
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
    const n = tabs.filter((t) => t.node.id !== key)
    setTabs(n)
    if (key === activeKey) setActiveKey(n.length > 0 ? n[n.length - 1].node.id : '')
  }

  const requestCloseTab = (key: string) => {
    const target = tabs.find((t) => t.node.id === key)
    if (target && !target.saved && target.scenes.some((s) => s.trim())) {
      Modal.confirm({
        title: '章节尚未保存',
        content: `确定关闭「${target.node.title || key}」吗？未保存的修改会丢失。`,
        okText: '关闭',
        okButtonProps: { danger: true },
        cancelText: '取消',
        onOk: () => closeTab(key),
      })
      return
    }
    closeTab(key)
  }

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
      if (activeTab.node.branch) {
        await SaveChapterBranchContent(activeTab.chapterNum, activeTab.node.branch, c)
      } else {
        await SaveChapterContent(activeTab.chapterNum, c)
      }
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

  const totalWords = countTextChars(activeTab?.scenes?.join('\n') || '')

  const tabItems: TabsProps['items'] = tabs.map((t) => ({
    key: t.node.id, label: t.node.title,
    closable: true,
  }))

  const atFirst = activeTab ? outlineLeaves.findIndex((n) => n.id === activeTab.node.id) <= 0 : true
  const atLast = activeTab ? outlineLeaves.findIndex((n) => n.id === activeTab.node.id) >= outlineLeaves.length - 1 : true

  return (
    <div style={{ height: '100%', display: 'flex', flexDirection: 'column', gap: 8 }}>

      <div style={{ flex: 1, display: 'flex', gap: 8, minHeight: 0 }}>
        {!activeTab ? (
          /* 空态：从左侧大纲选择章节 */
          <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', color: C('color-text-secondary'), opacity: 0.5 }}>
            <div style={{ textAlign: 'center' }}>
              <BookOutlined style={{ fontSize: 48, marginBottom: 16 }} />
              <div>从左侧大纲选择章节开始阅读</div>
            </div>
          </div>
        ) : (
          <div className="novel-editor-panel" style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
            {/* ── 单条 chrome（收敛原 tab/信息/工具栏 三层） ── */}
            {!focusMode && (
              <div className="novel-chrome">
                {readMode ? (
                  <>
                    <span className="novel-chrome-title">
                      <ReadOutlined aria-hidden />{activeTab.node.title || '未命名章节'}
                    </span>
                    <span className="novel-chrome-sub">· {totalWords.toLocaleString()} 字</span>
                    <span className="novel-chrome-spacer" />
                    <span className="novel-chrome-save-state">
                      <i className={`novel-dot ${activeTab.saved ? 'ok' : 'dirty'}`} aria-hidden />
                      <span style={{ color: activeTab.saved ? 'var(--color-success)' : 'var(--color-warning)' }}>
                        {activeTab.saved ? '已保存' : '未保存'}
                      </span>
                    </span>
                    <div className="novel-chrome-nav">
                      <Tooltip title="上一章"><Button size="small" icon={<LeftOutlined />} onClick={handlePrevChapter} type="text" disabled={atFirst} aria-label="上一章" /></Tooltip>
                      <Tooltip title="下一章"><Button size="small" icon={<RightOutlined />} onClick={handleNextChapter} type="text" disabled={atLast} aria-label="下一章" /></Tooltip>
                    </div>
                    <Tooltip title="返回编辑">
                      <Button size="small" icon={<EditOutlined />} onClick={() => setReadMode(false)}>编辑</Button>
                    </Tooltip>
                  </>
                ) : (
                  <>
                    <Tabs
                      className="novel-editor-tabs"
                      activeKey={activeKey}
                      onChange={setActiveKey}
                      onEdit={(key, action) => { if (action === 'remove' && typeof key === 'string') requestCloseTab(key) }}
                      items={tabItems}
                      type="editable-card"
                      size="small"
                      hideAdd
                      style={{ flex: 1, minWidth: 0, marginBottom: 0 }}
                      tabBarStyle={{ marginBottom: 0 }}
                    />
                    <span className="novel-chrome-spacer" />
                    <span className="novel-chrome-save-state">
                      <i className={`novel-dot ${activeTab.saved ? 'ok' : 'dirty'}`} aria-hidden />
                      <span style={{ color: activeTab.saved ? 'var(--color-success)' : 'var(--color-warning)', fontSize: 11 }}>
                        {activeTab.saved ? '已保存' : '未保存'}
                      </span>
                    </span>
                    <div className="novel-chrome-nav">
                      <Tooltip title="上一章"><Button size="small" icon={<LeftOutlined />} onClick={handlePrevChapter} type="text" disabled={atFirst} aria-label="上一章" /></Tooltip>
                      <Tooltip title="下一章"><Button size="small" icon={<RightOutlined />} onClick={handleNextChapter} type="text" disabled={atLast} aria-label="下一章" /></Tooltip>
                    </div>
                    <TTSPlayer getText={() => activeTab?.scenes?.join('\n\n') || ''} />
                    <Tooltip title="阅读模式（沉浸排版）">
                      <Button size="small" icon={<ReadOutlined />} onClick={() => setReadMode(true)} className="is-readmode" aria-label="进入阅读模式" />
                    </Tooltip>
                    <Tooltip title={focusMode ? '退出专注模式' : '专注模式 F11'}>
                      <Button size="small" icon={focusMode ? <ShrinkOutlined /> : <ExpandOutlined />} onClick={() => setFocusMode((p) => !p)} type="text" aria-label="专注模式" />
                    </Tooltip>
                    <Tooltip title="Ctrl+S">
                      <Button size="small" icon={<SaveOutlined />} onClick={handleSave} disabled={!totalWords}>保存</Button>
                    </Tooltip>
                  </>
                )}
              </div>
            )}

            {/* ── 阅读模式：居中限宽衬线排版 ── */}
            {readMode ? (
              <>
                <div className="novel-reading-scroll">
                  <div className="novel-reading-column">
                    {activeTab.scenes.map((scene, i) => (
                      <React.Fragment key={i}>
                        {i > 0 && <div className="novel-reading-scene-sep" aria-hidden>＊ ＊ ＊</div>}
                        <p style={{ margin: 0 }}>{scene || '（本章暂无内容）'}</p>
                      </React.Fragment>
                    ))}
                  </div>
                </div>
                {/* 阅读页脚：章节导航 */}
                <div className="novel-reading-foot">
                  <Button size="small" icon={<LeftOutlined />} onClick={handlePrevChapter} disabled={atFirst}>上一章</Button>
                  <span style={{ fontSize: 11, color: 'var(--color-text-secondary)', fontVariantNumeric: 'tabular-nums' }}>
                    {activeTab.node.title || '未命名章节'} · {totalWords.toLocaleString()} 字
                  </span>
                  <Button size="small" onClick={handleNextChapter} disabled={atLast}>下一章<RightOutlined /></Button>
                </div>
              </>
            ) : (
              /* ── 编辑模式：场景多文本框 ── */
              <>
                <ChapterEditor
                  tab={activeTab}
                  onUpdate={updateTab}
                  sceneTextareaRefs={sceneTextareaRefs}
                  ghostEnabled={false}
                />
                {focusMode && (
                  <div style={{ padding: '2px 12px', display: 'flex', alignItems: 'center', gap: 16, fontSize: 11, color: C('color-text-secondary'), opacity: 0.7 }}>
                    <span><kbd className="novel-kbd">F11</kbd> 专注模式</span>
                    <span><kbd className="novel-kbd">Ctrl+S</kbd> 保存</span>
                    <div style={{ flex: 1 }} />
                    <span>{activeTab.node.title} · {totalWords.toLocaleString()} 字</span>
                    <span style={{ color: 'var(--color-warning)' }}>专注模式已开启 · Esc 退出</span>
                  </div>
                )}
              </>
            )}
          </div>
        )}
      </div>

      {/* 底部快捷键提示栏（阅读模式隐藏，避免噪音） */}
      {activeTab && !readMode && !focusMode && (
        <div style={{ padding: '2px 12px', display: 'flex', alignItems: 'center', gap: 16, fontSize: 11, color: C('color-text-secondary'), opacity: 0.7 }}>
          <span><kbd className="novel-kbd">F11</kbd> 专注模式</span>
          <span><kbd className="novel-kbd">Ctrl+S</kbd> 保存</span>
          <span><kbd className="novel-kbd">Ctrl+K</kbd> AI 编辑选中段落</span>
          <span style={{ marginLeft: 'auto' }}><ReadOutlined style={{ marginRight: 4 }} />阅读模式 = 沉浸排版</span>
        </div>
      )}
    </div>
  )
}

export default ChapterPage
