import React, { useState, useEffect, useRef, useMemo } from 'react'
import {
  Button, message, Modal, Tabs, Tooltip, Popover, Segmented, Slider, Input,
} from 'antd'
import type { TabsProps } from 'antd'
import {
  SaveOutlined, LeftOutlined, RightOutlined,
  BookOutlined, EditOutlined, ReadOutlined, ExpandOutlined, ShrinkOutlined, FontSizeOutlined,
  PushpinOutlined, PushpinFilled, PlayCircleOutlined, PauseCircleOutlined, CloseOutlined,
  HighlightOutlined, CommentOutlined, ThunderboltOutlined, DownOutlined, LoadingOutlined, SearchOutlined,
  ExportOutlined, PictureOutlined,
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
import {
  clampFontSize, clampBrightness, clampAutoScrollSpeed,
  readReadingSettings, writeReadingSettings, READING_COLUMN_WIDTH,
  type ReadingColumn, type ReadingSettings, type ReadingTheme,
} from '../utils/readingSettings'
import {
  readBookmarks, writeBookmarks,
  type ReadingBookmark,
} from '../utils/readingBookmarks'
import {
  readAnnotations, writeAnnotations, ANNOTATION_COLORS,
  type ReadingAnnotation, type AnnotationColor,
} from '../utils/readingAnnotations'
import { askReadingAssistant } from '../components/novel/api/readingAssistant'
import { searchNovel, type NovelSearchHit } from '../components/novel/api/search'
import ExportPanel from '../components/novel/ExportPanel'
import { ChapterIllustration } from './chapter/ChapterIllustration'
import type { OutlineNode, ChapterTabData } from '../types'
import { C } from '../utils/theme'

const READ_SCROLL_PREFIX = 'gaea.novel.reading.scroll.'

function errText(err: unknown, fallback: string): string {
  return (err instanceof Error && err.message) || fallback
}

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
  const [readPrefs, setReadPrefs] = useState<ReadingSettings>(readReadingSettings)
  const [readProgress, setReadProgress] = useState(0)
  const [bookmarks, setBookmarks] = useState<ReadingBookmark[]>([])
  const [bookmarkOpen, setBookmarkOpen] = useState(false)
  const [autoScrolling, setAutoScrolling] = useState(false)
  const [annotations, setAnnotations] = useState<ReadingAnnotation[]>([])
  const [selToolbar, setSelToolbar] = useState<{ x: number; y: number } | null>(null)
  const [noteTarget, setNoteTarget] = useState<ReadingAnnotation | null>(null)
  const [noteDraft, setNoteDraft] = useState('')
  const [selText, setSelText] = useState('')
  const [summaryOpen, setSummaryOpen] = useState(false)
  const [summaryText, setSummaryText] = useState<string | null>(null)
  const [summaryLoading, setSummaryLoading] = useState(false)
  const [summaryError, setSummaryError] = useState<string | null>(null)
  const summaryCache = useRef<Record<string, string>>({})
  const [askTarget, setAskTarget] = useState<{ selection: string } | null>(null)
  const [askQuestion, setAskQuestion] = useState('')
  const [askAnswer, setAskAnswer] = useState('')
  const [askLoading, setAskLoading] = useState(false)
  const [searchOpen, setSearchOpen] = useState(false)
  const [searchQuery, setSearchQuery] = useState('')
  const [searchLoading, setSearchLoading] = useState(false)
  const [searchHits, setSearchHits] = useState<NovelSearchHit[]>([])
  const [searchError, setSearchError] = useState<string | null>(null)
  const [exportOpen, setExportOpen] = useState(false)
  const [illusOpen, setIllusOpen] = useState(false)
  const pendingSearch = useRef<{ nodeId: string; query: string } | null>(null)
  const readingScrollRef = useRef<HTMLDivElement | null>(null)
  const readScrollTimer = useRef<ReturnType<typeof setTimeout> | null>(null)
  const autoScrollRef = useRef<number | null>(null)
  // 当前激活章节（须在首个引用它的 useEffect 依赖数组之前定义，避免 TDZ）
  const activeTab = tabs.find((t) => t.node.id === activeKey) ?? null
  const handleSaveRef = useRef<() => void>(() => {})
  const handleNavRef = useRef<{ prev: () => void; next: () => void }>({ prev: () => {}, next: () => {} })
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
      if (readMode && (e.key === 'ArrowLeft' || e.key === 'ArrowRight')) {
        e.preventDefault()
        if (e.key === 'ArrowLeft') handleNavRef.current.prev()
        else handleNavRef.current.next()
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
    } catch { message.error('保存失败') }
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
  useEffect(() => {
    handleNavRef.current = { prev: handlePrevChapter, next: handleNextChapter }
  })

  const totalWords = countTextChars(activeTab?.scenes?.join('\n') || '')
  const readNodeId = activeTab?.node.id ?? ''

  // ── 阅读偏好（字号 / 行距 / 版宽，全局持久化） ──
  const patchReadPrefs = (p: Partial<ReadingSettings>) => {
    setReadPrefs((prev) => {
      const next = { ...prev, ...p }
      writeReadingSettings(next)
      return next
    })
  }
  const changeReadFont = (delta: number) => {
    setReadPrefs((prev) => {
      const next = { ...prev, fontSize: clampFontSize(prev.fontSize + delta) }
      writeReadingSettings(next)
      return next
    })
  }

  // ── 阅读滚动：进度条 + 章节位置记忆 ──
  const handleReadScroll = () => {
    const el = readingScrollRef.current
    if (!el || !activeTab) return
    setSelToolbar(null)
    const max = el.scrollHeight - el.clientHeight
    setReadProgress(max > 0 ? Math.min(100, Math.round((el.scrollTop / max) * 100)) : 0)
    if (readScrollTimer.current) clearTimeout(readScrollTimer.current)
    readScrollTimer.current = setTimeout(() => {
      try { localStorage.setItem(READ_SCROLL_PREFIX + activeTab.node.id, String(el.scrollTop)) } catch { /* ignore */ }
    }, 250)
  }
  useEffect(() => {
    if (!readMode || !readNodeId) return
    const el = readingScrollRef.current
    let saved = 0
    try { saved = Number(localStorage.getItem(READ_SCROLL_PREFIX + readNodeId) || 0) } catch { /* ignore */ }
    // 切换章节时先回到顶部，再恢复该章保存的位置
    if (el) el.scrollTop = 0
    const raf = requestAnimationFrame(() => {
      if (el && saved > 0) el.scrollTop = saved
    })
    setSummaryOpen(false)
    setSummaryText(null)
    setSummaryError(null)
    return () => {
      cancelAnimationFrame(raf)
      if (el && el.scrollTop > 0) {
        try { localStorage.setItem(READ_SCROLL_PREFIX + readNodeId, String(el.scrollTop)) } catch { /* ignore */ }
      }
      if (readScrollTimer.current) { clearTimeout(readScrollTimer.current); readScrollTimer.current = null }
    }
  }, [readMode, readNodeId])

  // ── 书签：按项目持久化，列表归属当前章节 ──
  useEffect(() => {
    if (!readMode) return
    setBookmarks(readBookmarks(projectPath).filter((b) => b.nodeId === readNodeId))
  }, [projectPath, readMode, readNodeId])

  const persistBookmarks = (list: ReadingBookmark[]) => {
    setBookmarks(list)
    writeBookmarks(projectPath, list)
  }

  const textAtScrollTop = (el: HTMLDivElement, scrollTop: number): string => {
    const paras = el.querySelectorAll<HTMLElement>('.novel-reading-p')
    let best: HTMLElement | null = null
    for (const p of Array.from(paras)) {
      if (p.offsetTop <= scrollTop + 48) best = p
      else break
    }
    return (best?.textContent || '').trim().slice(0, 48)
  }

  const toggleBookmark = () => {
    const el = readingScrollRef.current
    if (!el || !activeTab) return
    const scrollTop = Math.max(0, Math.round(el.scrollTop))
    const hit = bookmarks.find((b) => b.nodeId === activeTab.node.id && Math.abs(b.scrollTop - scrollTop) < 48)
    let next: ReadingBookmark[]
    if (hit) {
      next = bookmarks.filter((b) => b !== hit)
    } else {
      const max = el.scrollHeight - el.clientHeight
      next = [
        ...bookmarks,
        {
          nodeId: activeTab.node.id,
          title: activeTab.node.title || '未命名章节',
          scrollTop,
          pct: max > 0 ? Math.round((scrollTop / max) * 100) : 0,
          text: textAtScrollTop(el, scrollTop),
          createdAt: Date.now(),
        },
      ].sort((a, b) => a.scrollTop - b.scrollTop)
    }
    persistBookmarks(next)
  }

  const jumpBookmark = (b: ReadingBookmark) => {
    const el = readingScrollRef.current
    if (!el) return
    el.scrollTop = b.scrollTop
    setBookmarkOpen(false)
  }

  const removeBookmark = (b: ReadingBookmark) => {
    persistBookmarks(bookmarks.filter((x) => x.nodeId !== b.nodeId || x.scrollTop !== b.scrollTop))
  }

  // ── 自动滚屏：40ms 步进，滚轮手动干预即暂停 ──
  const startAutoScroll = () => {
    if (autoScrollRef.current !== null) clearInterval(autoScrollRef.current)
    setAutoScrolling(true)
    autoScrollRef.current = window.setInterval(() => {
      const el = readingScrollRef.current
      if (!el) return
      const max = el.scrollHeight - el.clientHeight
      if (el.scrollTop >= max) { stopAutoScroll(); return }
      el.scrollTop = Math.min(max, el.scrollTop + readPrefs.autoScrollSpeed)
    }, 40)
  }

  const stopAutoScroll = () => {
    if (autoScrollRef.current !== null) { clearInterval(autoScrollRef.current); autoScrollRef.current = null }
    setAutoScrolling(false)
  }

  useEffect(() => {
    if (!autoScrolling) return
    const el = readingScrollRef.current
    const onWheel = () => stopAutoScroll()
    el?.addEventListener('wheel', onWheel, { passive: true })
    return () => el?.removeEventListener('wheel', onWheel)
  }, [autoScrolling])

  useEffect(() => () => {
    if (autoScrollRef.current !== null) clearInterval(autoScrollRef.current)
  }, [])

  // ── 划线 / 高亮 / 想法 ──
  useEffect(() => {
    if (!projectPath) return
    setAnnotations(readAnnotations(projectPath))
  }, [projectPath])

  const chapterAnns = annotations.filter((a) => a.nodeId === readNodeId)

  const persistAnnotations = (list: ReadingAnnotation[]) => {
    setAnnotations(list)
    writeAnnotations(projectPath, list)
  }

  const paraOf = (node: Node): HTMLElement | null => {
    let el: HTMLElement | null = node instanceof HTMLElement ? node : node.parentElement
    while (el && !el.classList.contains('novel-reading-p')) el = el.parentElement
    return el
  }

  const addHighlight = (color: AnnotationColor, withNote: boolean, textOverride?: string) => {
    const root = readingScrollRef.current
    if (!root) return
    let text = textOverride ?? ''
    if (!text) {
      const sel = window.getSelection()
      if (!sel || sel.isCollapsed) return
      const range = sel.getRangeAt(0)
      if (!root.contains(range.commonAncestorContainer)) return
      if (paraOf(range.startContainer) !== paraOf(range.endContainer)) return
      text = sel.toString().replace(/\s+/g, ' ').trim()
      sel.removeAllRanges()
    }
    if (!text || text.length > 200) return
    const ann: ReadingAnnotation = {
      id: `ann_${Date.now()}_${Math.random().toString(36).slice(2, 7)}`,
      nodeId: readNodeId,
      title: activeTab?.node.title || '未命名章节',
      color,
      text,
      note: '',
      createdAt: Date.now(),
    }
    persistAnnotations([...annotations, ann])
    setSelToolbar(null)
    if (withNote) { setNoteTarget(ann); setNoteDraft('') }
  }

  const openAnnotation = (ann: ReadingAnnotation) => {
    setNoteTarget(ann)
    setNoteDraft(ann.note || '')
  }

  const saveNote = () => {
    if (!noteTarget) return
    persistAnnotations(annotations.map((a) => (a.id === noteTarget.id ? { ...a, note: noteDraft.trim() } : a)))
    setNoteTarget(null)
  }

  const deleteAnnotation = (id: string) => {
    persistAnnotations(annotations.filter((a) => a.id !== id))
    setNoteTarget(null)
  }

  const jumpToAnnotation = (ann: ReadingAnnotation) => {
    const root = readingScrollRef.current
    const mark = root
      ? Array.from(root.querySelectorAll<HTMLElement>('mark[data-ann-id]'))
        .find((m) => m.getAttribute('data-ann-id') === ann.id)
      : null
    mark?.scrollIntoView({ block: 'center', behavior: 'smooth' })
  }

  // 选中文本 → 浮动工具条（仅限阅读列内、单段落内选择）
  const handleReadingMouseUp = () => {
    const sel = window.getSelection()
    const root = readingScrollRef.current
    if (!sel || sel.isCollapsed || !root) { setSelToolbar(null); return }
    const range = sel.getRangeAt(0)
    if (!root.contains(range.commonAncestorContainer)) { setSelToolbar(null); return }
    if (paraOf(range.startContainer) !== paraOf(range.endContainer)) { setSelToolbar(null); return }
    const text = sel.toString().trim()
    if (!text) { setSelToolbar(null); return }
    setSelText(text.replace(/\s+/g, ' '))
    const rect = range.getBoundingClientRect()
    setSelToolbar({ x: rect.left + rect.width / 2, y: rect.top })
  }

  // ── AI 伴读：章节摘要 + 划线提问 ──
  const chapterText = activeTab?.scenes?.join('\n\n') || ''

  const runSummary = async () => {
    if (summaryLoading || !activeTab || !readNodeId) return
    const cached = summaryCache.current[readNodeId]
    if (cached) { setSummaryText(cached); return }
    setSummaryLoading(true)
    setSummaryError(null)
    try {
      const text = await askReadingAssistant('summary', activeTab.node.title || '', chapterText, '', '')
      summaryCache.current[readNodeId] = text
      setSummaryText(text)
    } catch (err) {
      setSummaryError(errText(err, '摘要生成失败'))
    } finally {
      setSummaryLoading(false)
    }
  }

  const toggleSummary = () => {
    if (summaryOpen) { setSummaryOpen(false); return }
    setSummaryOpen(true)
    if (!summaryText && !summaryLoading) void runSummary()
  }

  const openAsk = (selection: string) => {
    setAskTarget({ selection })
    setAskQuestion('')
    setAskAnswer('')
    setAskLoading(false)
  }

  const runAsk = async () => {
    const q = askQuestion.trim()
    if (!q || !askTarget || askLoading || !activeTab) return
    setAskLoading(true)
    setAskAnswer('')
    try {
      const answer = await askReadingAssistant('ask', activeTab.node.title || '', chapterText, askTarget.selection, q)
      setAskAnswer(answer)
    } catch (err) {
      setAskAnswer(`出错了：${errText(err, '提问失败')}`)
    } finally {
      setAskLoading(false)
    }
  }

  // 高亮回渲染：清除旧 mark 后按摘录文本在段落内重新定位（内容变更时自然失效）
  useEffect(() => {
    const root = readingScrollRef.current
    if (!root) return
    root.querySelectorAll<HTMLElement>('mark[data-ann-id]').forEach((m) => {
      const parent = m.parentNode
      if (!parent) return
      parent.replaceChild(document.createTextNode(m.textContent ?? ''), m)
    })
    root.normalize()
    const current = annotations.filter((a) => a.nodeId === readNodeId)
    if (current.length === 0) return
    const paras = Array.from(root.querySelectorAll<HTMLElement>('.novel-reading-p'))
    for (const p of paras) {
      const text = p.textContent ?? ''
      if (!text) continue
      const matches: { start: number; end: number; ann: ReadingAnnotation }[] = []
      const occupied = new Set<number>()
      for (const ann of [...current].sort((a, b) => b.text.length - a.text.length)) {
        if (!ann.text) continue
        let idx = text.indexOf(ann.text)
        while (idx !== -1) {
          const end = idx + ann.text.length
          let conflict = false
          for (let k = idx; k < end; k++) {
            if (occupied.has(k)) { conflict = true; break }
          }
          if (!conflict) {
            matches.push({ start: idx, end, ann })
            for (let k = idx; k < end; k++) occupied.add(k)
            break
          }
          idx = text.indexOf(ann.text, idx + 1)
        }
      }
      if (matches.length === 0) continue
      matches.sort((a, b) => b.start - a.start)
      for (const m of matches) {
        const node = p.firstChild
        if (!node || node.nodeType !== Node.TEXT_NODE) continue
        try {
          const range = document.createRange()
          range.setStart(node, m.start)
          range.setEnd(node, m.end)
          const mark = document.createElement('mark')
          mark.className = 'novel-reading-mark'
          mark.dataset.annId = m.ann.id
          mark.dataset.annColor = m.ann.color
          mark.addEventListener('click', () => openAnnotation(m.ann))
          mark.appendChild(range.extractContents())
          range.insertNode(mark)
        } catch { /* ignore */ }
      }
    }
  }, [readMode, readNodeId, annotations])

  // ── 朗读/搜索定位高亮：按文本在段落 DOM 中回定位 ──
  const clearReadingHighlight = (className: string) => {
    const root = readingScrollRef.current
    if (!root) return
    root.querySelectorAll<HTMLElement>(`.${className}`).forEach((el) => {
      const parent = el.parentNode
      if (!parent) return
      while (el.firstChild) parent.insertBefore(el.firstChild, el)
      parent.removeChild(el)
      parent.normalize()
    })
  }

  const textNodesOf = (el: HTMLElement): { node: Text; start: number; end: number }[] => {
    const walker = document.createTreeWalker(el, NodeFilter.SHOW_TEXT)
    const out: { node: Text; start: number; end: number }[] = []
    let pos = 0
    let n: Node | null = walker.nextNode()
    while (n) {
      const len = (n.textContent || '').length
      if (len > 0) out.push({ node: n as Text, start: pos, end: pos + len })
      pos += len
      n = walker.nextNode()
    }
    return out
  }

  const applyTextHighlight = (rawText: string, className: string): boolean => {
    const root = readingScrollRef.current
    if (!root || !readMode) return false
    const target = rawText.trim()
    if (!target) return false
    clearReadingHighlight(className)
    const paras = Array.from(root.querySelectorAll<HTMLElement>('.novel-reading-p'))
    for (const p of paras) {
      const full = p.textContent ?? ''
      const idx = full.indexOf(target)
      if (idx === -1) continue
      const nodes = textNodesOf(p)
      const endAt = idx + target.length
      const startHolder = nodes.find((t) => idx < t.end)
      const endHolder = nodes.find((t) => endAt <= t.end)
      if (!startHolder || !endHolder) break
      try {
        const range = document.createRange()
        range.setStart(startHolder.node, Math.max(0, idx - startHolder.start))
        range.setEnd(endHolder.node, endAt - endHolder.start)
        const span = document.createElement('span')
        span.className = className
        span.appendChild(range.extractContents())
        range.insertNode(span)
        span.scrollIntoView({ block: 'center', behavior: 'smooth' })
        return true
      } catch { /* ignore */ }
    }
    return false
  }
  const applyTextHighlightRef = useRef(applyTextHighlight)
  applyTextHighlightRef.current = applyTextHighlight

  const handleTtsSentence = (sentence: string) => {
    applyTextHighlight(sentence, 'novel-reading-current')
  }
  const handleTtsClear = () => {
    clearReadingHighlight('novel-reading-current')
  }

  // ── 全文搜索（防抖 300ms，点击结果打开章节并定位） ──
  useEffect(() => {
    if (!searchOpen) return
    const q = searchQuery.trim()
    if (!q) { setSearchHits([]); setSearchError(null); setSearchLoading(false); return }
    setSearchLoading(true)
    const timer = setTimeout(async () => {
      try {
        const hits = await searchNovel(q)
        setSearchHits(hits || [])
        setSearchError(null)
      } catch (err) {
        setSearchHits([])
        setSearchError(errText(err, '搜索失败'))
      } finally {
        setSearchLoading(false)
      }
    }, 300)
    return () => clearTimeout(timer)
  }, [searchOpen, searchQuery])

  const openSearchHit = (hit: NovelSearchHit) => {
    const node = outlineLeaves.find((n) => n.id === hit.node_id)
    if (!node) return
    pendingSearch.current = { nodeId: hit.node_id, query: searchQuery.trim() }
    setSearchOpen(false)
    handleSelectNode(node)
    setReadMode(true)
  }

  // 打开目标章节后等待正文渲染，再定位并高亮首个命中
  useEffect(() => {
    if (!readMode || !readNodeId || !pendingSearch.current || pendingSearch.current.nodeId !== readNodeId) return
    const query = pendingSearch.current.query
    let tries = 0
    const timer = window.setInterval(() => {
      tries++
      const root = readingScrollRef.current
      const found = root
        ? Array.from(root.querySelectorAll('.novel-reading-p')).some((p) => (p.textContent || '').includes(query))
        : false
      if (found || tries > 30) {
        window.clearInterval(timer)
        if (found) applyTextHighlightRef.current(query, 'novel-reading-search-hit')
        pendingSearch.current = null
      }
    }, 120)
    return () => window.clearInterval(timer)
  }, [readMode, readNodeId])

  const tabItems: TabsProps['items'] = tabs.map((t) => ({
    key: t.node.id, label: t.node.title,
    closable: true,
  }))

  const atFirst = activeTab ? outlineLeaves.findIndex((n) => n.id === activeTab.node.id) <= 0 : true
  const atLast = activeTab ? outlineLeaves.findIndex((n) => n.id === activeTab.node.id) >= outlineLeaves.length - 1 : true

  return (
    <div style={{ height: '100%', display: 'flex', flexDirection: 'column', gap: 8, flex: 1, minWidth: 0 }}>

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
                    <Popover
                      trigger="click"
                      placement="bottomRight"
                      open={bookmarkOpen}
                      onOpenChange={setBookmarkOpen}
                      content={(
                        <div className="novel-read-bookmarks">
                          <div className="novel-read-bookmarks-head">
                            <span>本章书签（{bookmarks.length}）</span>
                            <Button size="small" type="text" onClick={toggleBookmark} aria-label="在当前位置添加书签">＋ 此处</Button>
                          </div>
                          {bookmarks.length === 0 ? (
                            <div className="novel-read-bookmarks-empty">滚动到想记住的位置，点「＋ 此处」添加书签</div>
                          ) : (
                            <div className="novel-read-bookmarks-list">
                              {bookmarks.map((b) => (
                                <div
                                  key={b.createdAt}
                                  className="novel-read-bookmark"
                                  role="button"
                                  tabIndex={0}
                                  onClick={() => jumpBookmark(b)}
                                  onKeyDown={(e) => { if (e.key === 'Enter') jumpBookmark(b) }}
                                >
                                  <span className="novel-read-bookmark-pct">{b.pct}%</span>
                                  <span className="novel-read-bookmark-text">{b.text || '（无摘录）'}</span>
                                  <Button
                                    size="small"
                                    type="text"
                                    icon={<CloseOutlined />}
                                    aria-label="删除书签"
                                    onClick={(e) => { e.stopPropagation(); removeBookmark(b) }}
                                  />
                                </div>
                              ))}
                            </div>
                          )}
                        </div>
                      )}
                    >
                      <Tooltip title={bookmarks.length > 0 ? `书签（${bookmarks.length}）` : '书签'}>
                        <Button
                          size="small"
                          type="text"
                          icon={bookmarks.length > 0 ? <PushpinFilled /> : <PushpinOutlined />}
                          className={bookmarks.length > 0 ? 'is-active' : ''}
                          aria-label="书签"
                        />
                      </Tooltip>
                    </Popover>
                    <Tooltip title={autoScrolling ? '停止自动滚屏' : '自动滚屏'}>
                      <Button
                        size="small"
                        type="text"
                        icon={autoScrolling ? <PauseCircleOutlined /> : <PlayCircleOutlined />}
                        className={autoScrolling ? 'is-active' : ''}
                        aria-label="自动滚屏"
                        onClick={() => (autoScrolling ? stopAutoScroll() : startAutoScroll())}
                      />
                    </Tooltip>
                    <Popover
                      trigger="click"
                      placement="bottomRight"
                      content={(
                        <div className="novel-read-anns">
                          <div className="novel-read-anns-head">
                            <span>本章划线 / 想法（{chapterAnns.length}）</span>
                            <span className="novel-read-anns-hint">选中文字即可划线</span>
                          </div>
                          {chapterAnns.length === 0 ? (
                            <div className="novel-read-anns-empty">拖动选中正文 → 高亮或写想法</div>
                          ) : (
                            <div className="novel-read-anns-list">
                              {chapterAnns.map((a) => (
                                <div
                                  key={a.id}
                                  className="novel-read-ann"
                                  role="button"
                                  tabIndex={0}
                                  onClick={() => jumpToAnnotation(a)}
                                  onKeyDown={(e) => { if (e.key === 'Enter') jumpToAnnotation(a) }}
                                >
                                  <i className="novel-read-ann-dot" style={{ background: ANNOTATION_COLORS[a.color] }} />
                                  <span className="novel-read-ann-text">{a.text}</span>
                                  {a.note && <CommentOutlined className="novel-read-ann-note" />}
                                  <Button
                                    size="small"
                                    type="text"
                                    icon={<CloseOutlined />}
                                    aria-label="删除划线"
                                    onClick={(e) => { e.stopPropagation(); deleteAnnotation(a.id) }}
                                  />
                                </div>
                              ))}
                            </div>
                          )}
                        </div>
                      )}
                    >
                      <Tooltip title={chapterAnns.length > 0 ? `划线 / 想法（${chapterAnns.length}）` : '划线 / 想法'}>
                        <Button
                          size="small"
                          type="text"
                          icon={<HighlightOutlined />}
                          className={chapterAnns.length > 0 ? 'is-active' : ''}
                          aria-label="划线 / 想法"
                        />
                      </Tooltip>
                    </Popover>
                    <Popover
                      trigger="click"
                      placement="bottomRight"
                      open={searchOpen}
                      onOpenChange={(v) => { setSearchOpen(v); if (!v) { setSearchHits([]); setSearchError(null) } }}
                      content={(
                        <div className="novel-read-search">
                          <Input
                            size="small"
                            value={searchQuery}
                            onChange={(e) => setSearchQuery(e.target.value)}
                            placeholder="搜索全书（标题 + 正文）"
                            allowClear
                          />
                          <div className="novel-read-search-body">
                            {searchLoading ? (
                              <div className="novel-read-search-hint"><LoadingOutlined spin /> 搜索中…</div>
                            ) : searchError ? (
                              <div className="novel-read-search-hint">{searchError}</div>
                            ) : searchQuery.trim() && searchHits.length === 0 ? (
                              <div className="novel-read-search-hint">没有找到「{searchQuery.trim()}」</div>
                            ) : (
                              <div className="novel-read-search-list">
                                {searchHits.map((h) => (
                                  <div
                                    key={h.node_id}
                                    className="novel-read-search-hit-row"
                                    role="button"
                                    tabIndex={0}
                                    onClick={() => openSearchHit(h)}
                                    onKeyDown={(e) => { if (e.key === 'Enter') openSearchHit(h) }}
                                  >
                                    <span className="novel-read-search-hit-title">{h.title}</span>
                                    <span className="novel-read-search-hit-snippet">{h.snippet}</span>
                                  </div>
                                ))}
                              </div>
                            )}
                          </div>
                        </div>
                      )}
                    >
                      <Tooltip title="全文搜索">
                        <Button size="small" type="text" icon={<SearchOutlined />} aria-label="全文搜索" />
                      </Tooltip>
                    </Popover>
                    <Popover
                      trigger="click"
                      placement="bottomRight"
                      content={(
                        <div className="novel-read-prefs">
                          <div className="novel-read-prefs-row">
                            <span className="novel-read-prefs-label">字号</span>
                            <Button size="small" onClick={() => changeReadFont(-1)} disabled={readPrefs.fontSize <= 14} aria-label="减小字号">A−</Button>
                            <span className="novel-read-prefs-val">{readPrefs.fontSize}</span>
                            <Button size="small" onClick={() => changeReadFont(1)} disabled={readPrefs.fontSize >= 24} aria-label="增大字号">A+</Button>
                          </div>
                          <div className="novel-read-prefs-row">
                            <span className="novel-read-prefs-label">行距</span>
                            <Segmented
                              size="small"
                              value={String(readPrefs.lineHeight)}
                              options={[
                                { label: '紧凑', value: '1.8' },
                                { label: '标准', value: '2' },
                                { label: '宽松', value: '2.3' },
                              ]}
                              onChange={(v) => patchReadPrefs({ lineHeight: Number(v) as ReadingSettings['lineHeight'] })}
                            />
                          </div>
                          <div className="novel-read-prefs-row">
                            <span className="novel-read-prefs-label">版宽</span>
                            <Segmented
                              size="small"
                              value={readPrefs.column}
                              options={[
                                { label: '窄', value: 'narrow' },
                                { label: '标准', value: 'standard' },
                                { label: '铺满', value: 'wide' },
                              ]}
                              onChange={(v) => patchReadPrefs({ column: v as ReadingColumn })}
                            />
                          </div>
                          <div className="novel-read-prefs-row">
                            <span className="novel-read-prefs-label">主题</span>
                            <Segmented
                              size="small"
                              value={readPrefs.theme}
                              options={[
                                { label: '跟随', value: 'auto' },
                                { label: '米黄', value: 'sepia' },
                                { label: '护眼绿', value: 'green' },
                                { label: '夜间', value: 'dark' },
                              ]}
                              onChange={(v) => patchReadPrefs({ theme: v as ReadingTheme })}
                            />
                          </div>
                          <div className="novel-read-prefs-row">
                            <span className="novel-read-prefs-label">亮度</span>
                            <Slider
                              min={70}
                              max={120}
                              step={5}
                              value={readPrefs.brightness}
                              onChange={(v) => patchReadPrefs({ brightness: clampBrightness(Number(v)) })}
                              style={{ flex: 1, margin: '0 6px' }}
                              tooltip={{ formatter: (v?: number) => `${v}%` }}
                            />
                            <span className="novel-read-prefs-val">{readPrefs.brightness}%</span>
                          </div>
                          <div className="novel-read-prefs-row">
                            <span className="novel-read-prefs-label">滚屏</span>
                            <Segmented
                              size="small"
                              value={String(readPrefs.autoScrollSpeed)}
                              options={[
                                { label: '慢', value: '1' },
                                { label: '2', value: '2' },
                                { label: '3', value: '3' },
                                { label: '4', value: '4' },
                                { label: '快', value: '5' },
                              ]}
                              onChange={(v) => patchReadPrefs({ autoScrollSpeed: clampAutoScrollSpeed(Number(v)) })}
                            />
                          </div>
                        </div>
                      )}
                    >
                      <Tooltip title="排版 / 主题 / 亮度 / 滚屏">
                        <Button size="small" type="text" icon={<FontSizeOutlined />} aria-label="阅读排版" />
                      </Tooltip>
                    </Popover>
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
                    <Tooltip title="为当前章节生成配图">
                      <Button
                        size="small"
                        icon={<PictureOutlined />}
                        onClick={() => setIllusOpen(true)}
                        disabled={!activeTab || activeTab.chapterNum < 1}
                        aria-label="生成配图"
                      >
                        配图
                      </Button>
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
                <div className="novel-reading-progress" aria-hidden>
                  <i style={{ width: `${readProgress}%` }} />
                </div>
                <div
                  className="novel-reading-scroll"
                  ref={readingScrollRef}
                  onScroll={handleReadScroll}
                  onMouseUp={handleReadingMouseUp}
                  data-read-theme={readPrefs.theme}
                  style={{ filter: `brightness(${readPrefs.brightness}%)` }}
                >
                  <div
                    className="novel-reading-column"
                    style={{
                      fontSize: readPrefs.fontSize,
                      lineHeight: readPrefs.lineHeight,
                      maxWidth: READING_COLUMN_WIDTH[readPrefs.column],
                    }}
                  >
                    <h2 className="novel-reading-title">{activeTab.node.title || '未命名章节'}</h2>
                    <div className="novel-read-summary">
                      <button
                        type="button"
                        className="novel-read-summary-head"
                        onClick={toggleSummary}
                        aria-expanded={summaryOpen}
                      >
                        <ThunderboltOutlined className="novel-read-summary-ic" />
                        <span>AI 摘要</span>
                        {summaryLoading
                          ? <LoadingOutlined className="novel-read-summary-loading" />
                          : <DownOutlined className={`novel-read-summary-chev${summaryOpen ? ' is-open' : ''}`} />}
                      </button>
                      {summaryOpen && (
                        <div className="novel-read-summary-body">
                          {summaryLoading ? (
                            <span className="novel-read-summary-hint">AI 正在阅读本章…</span>
                          ) : summaryText ? (
                            <div className="novel-read-summary-text">{summaryText}</div>
                          ) : summaryError ? (
                            <div className="novel-read-summary-error">
                              <span>{summaryError}</span>
                              <Button size="small" type="text" onClick={() => void runSummary()}>重试</Button>
                            </div>
                          ) : (
                            <span className="novel-read-summary-hint">展开即生成，仅使用本章本地文本</span>
                          )}
                        </div>
                      )}
                    </div>
                    {activeTab.scenes.map((scene, i) => {
                      const paras = scene.split(/\n\s*\n/).map((s) => s.trim()).filter(Boolean)
                      return (
                        <React.Fragment key={i}>
                          {i > 0 && <div className="novel-reading-scene-sep" aria-hidden>＊ ＊ ＊</div>}
                          {paras.length === 0
                            ? <p className="novel-reading-p">{scene || '（本章暂无内容）'}</p>
                            : paras.map((p, j) => <p key={j} className="novel-reading-p">{p}</p>)}
                        </React.Fragment>
                      )
                    })}
                  </div>
                </div>
                {/* 阅读页脚：章节导航 */}
                <div className="novel-reading-foot">
                  <TTSPlayer
                    getText={() => activeTab?.scenes?.join('\n\n') || ''}
                    onSentence={handleTtsSentence}
                    onClear={handleTtsClear}
                  />
                  <Button size="small" icon={<LeftOutlined />} onClick={handlePrevChapter} disabled={atFirst}>上一章</Button>
                  <span style={{ fontSize: 11, color: 'var(--color-text-secondary)', fontVariantNumeric: 'tabular-nums' }}>
                    {activeTab.node.title || '未命名章节'} · {totalWords.toLocaleString()} 字
                  </span>
                  <Button size="small" onClick={handleNextChapter} disabled={atLast}>下一章<RightOutlined /></Button>
                  <Tooltip title="导出全部格式">
                    <Button size="small" type="text" icon={<ExportOutlined />} aria-label="导出小说" onClick={() => setExportOpen(true)} />
                  </Tooltip>
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

      {/* 划词工具条：选中文字后浮动在选区上方 */}
      {readMode && selToolbar && (
        <div
          className="novel-read-selbar"
          style={{ left: selToolbar.x, top: selToolbar.y }}
          role="toolbar"
          aria-label="划词工具"
        >
          {(Object.keys(ANNOTATION_COLORS) as AnnotationColor[]).map((c) => (
            <button
              key={c}
              type="button"
              className="novel-read-selbar-swatch"
              style={{ background: ANNOTATION_COLORS[c] }}
              title={`高亮（${c}）`}
              aria-label={`高亮（${c}）`}
              onClick={() => addHighlight(c, false, selText)}
            />
          ))}
          <span className="novel-read-selbar-divider" aria-hidden />
          <button type="button" className="novel-read-selbar-note" onClick={() => addHighlight('yellow', true, selText)}>
            <CommentOutlined /> 想法
          </button>
          <span className="novel-read-selbar-divider" aria-hidden />
          <button
            type="button"
            className="novel-read-selbar-note"
            onClick={() => { window.getSelection()?.removeAllRanges(); setSelToolbar(null); openAsk(selText) }}
          >
            <ThunderboltOutlined /> 问书
          </button>
        </div>
      )}

      {/* 想法编辑弹窗 */}
      <Modal
        open={!!noteTarget}
        onCancel={() => setNoteTarget(null)}
        title={noteTarget ? `想法 · ${noteTarget.title}` : '想法'}
        footer={null}
        width={420}
      >
        {noteTarget && (
          <div className="novel-read-note">
            <blockquote className="novel-read-note-quote">{noteTarget.text}</blockquote>
            <Input.TextArea
              rows={4}
              value={noteDraft}
              onChange={(e) => setNoteDraft(e.target.value)}
              placeholder="写点什么…（保存后随高亮展示）"
            />
            <div className="novel-read-note-actions">
              <Button danger size="small" onClick={() => deleteAnnotation(noteTarget.id)}>删除高亮</Button>
              <div style={{ flex: 1 }} />
              <Button size="small" onClick={() => setNoteTarget(null)}>取消</Button>
              <Button type="primary" size="small" onClick={saveNote}>保存想法</Button>
            </div>
          </div>
        )}
      </Modal>

      {/* AI 问书弹窗 */}
      <Modal
        open={!!askTarget}
        onCancel={() => setAskTarget(null)}
        title="AI 问书"
        footer={null}
        width={560}
      >
        {askTarget && (
          <div className="novel-read-ask">
            <blockquote className="novel-read-note-quote">{askTarget.selection}</blockquote>
            <Input.TextArea
              rows={2}
              value={askQuestion}
              onChange={(e) => setAskQuestion(e.target.value)}
              placeholder="针对摘选内容提问，例如：这句话暗示了什么？"
            />
            <div className="novel-read-ask-actions">
              <div style={{ flex: 1 }} />
              <Button size="small" onClick={() => setAskTarget(null)}>关闭</Button>
              <Button
                type="primary"
                size="small"
                loading={askLoading}
                disabled={!askQuestion.trim()}
                onClick={() => void runAsk()}
              >
                提问
              </Button>
            </div>
            {askAnswer && (
              <div className="novel-read-ask-answer">
                <div className="novel-read-ask-answer-title">回答</div>
                <div className="novel-read-ask-answer-body">{askAnswer}</div>
              </div>
            )}
          </div>
        )}
      </Modal>

      {/* 导出弹窗（原「导出」标签页合并进阅读面板） */}
      <Modal
        open={exportOpen}
        onCancel={() => setExportOpen(false)}
        title="导出小说"
        footer={null}
        width={520}
      >
        <ExportPanel />
      </Modal>

      {/* 章节配图（v4.3g 图文联动）：打开即加载，父级卸载即关闭 */}
      {illusOpen && activeTab && activeTab.chapterNum >= 1 && (
        <ChapterIllustration
          chapterNum={activeTab.chapterNum}
          onClose={() => setIllusOpen(false)}
        />
      )}
    </div>
  )
}

export default ChapterPage
