import React, { useState, useEffect, useRef, useMemo, useCallback } from 'react'
import {
  message, Modal, Spin,
} from 'antd'
import type { TabsProps } from 'antd'
import {
  BookOutlined, ReadOutlined,
} from '@ant-design/icons'
import { app } from '../gaea/lib/bridge'
import { useAppStore } from '../stores/appStore'
import ChapterEditor from '../components/novel/ChapterEditor'
import { findAllLeaves, sortNodes } from '../utils/outline'
import { useOutlineStore } from '../stores/outlineStore'
import { countTextChars } from '../utils/text'
import { readReadingProgress, writeReadingProgress } from '../utils/readingProgress'
import {
  readReadingSettings, writeReadingSettings,
  type ReadingSettings,
} from '../utils/readingSettings'
import {
  readBookmarks, writeBookmarks,
  type ReadingBookmark,
} from '../utils/readingBookmarks'
import {
  readAnnotations, writeAnnotations,
  type ReadingAnnotation, type AnnotationColor,
} from '../utils/readingAnnotations'
import { askReadingAssistant } from '../components/novel/api/readingAssistant'
import {
  buildAskHistory, rollbackLastUserMessage, type ReadingAskMessage,
} from './chapter/readingAskSession'
import {
  searchNovelAll, summarizeSearch,
  type NovelSearchHitData,
} from './chapter/novelSearchUtils'
import {
  clearReadingHighlight, highlightSearchHitAt, readSelectionInRoot, textAtScrollTop,
  // applyTextHighlight 主体已抽为模块级纯 DOM 工具（恒定身份），别名导入；
  // 组件内保留同名薄包装绑定滚动根与 readMode，流式调用入口不变。
  applyTextHighlight as highlightFirstMatch,
} from './chapter/readingHighlight'
import { renderAnnotationHighlights } from './chapter/readingAnnotation'
import { searchHitAnchor } from './chapter/searchHitAnnotation'
import {
  toggleBookmarkInList, removeBookmarkInList,
} from './chapter/readingBookmark'
import {
  readSavedScrollTop, saveScrollTop, scrollPct,
} from './chapter/readingScrollMemory'
import { createTabData, needsCloseConfirm } from './chapter/chapterTabData'
import { registerNovelDirtyProvider, takeDiscardConfirmed } from '../components/novel/novelSwitchGuard'
// 重写历史入口的未保存闸门复用共享三选原语（只消费，不改该文件语义）。
import { chooseUnsavedAction } from '../components/novel/unsavedGuard'
import { IsProjectV4, GetChapterScenes, SaveScene, CreateScene } from '../../wailsjs/go/app/NovelB'
import ExportPanel from '../components/novel/ExportPanel'
import { ChapterIllustration } from './chapter/ChapterIllustration'
import type { OutlineNode, ChapterTabData } from '../types'
import { C } from '../utils/theme'
// 瘦身 P3 拆分（沿用 v4.170 分批搬移先例）：阅读/编辑 chrome、阅读正文面板、
// 划词工具条与想法/问书弹窗、errText 均已抽至 pages/chapter/ 子目录（纯受控展示
// 组件，状态与回调经 props 传入），主文件仅保留状态/接线与空态等骨架 JSX。
import { errText } from './chapter/errText'
import ReadingChrome from './chapter/readingChrome'
import EditChrome from './chapter/editChrome'
import RewriteModal from '../components/novel/RewriteModal'
import RewriteHistoryPanel from '../components/novel/RewriteHistoryPanel'
import ReadingPanel from './chapter/readingPanel'
import ReadingOverlays from './chapter/readingOverlays'

/**
 * ChapterPage props（跨线契约，规格线2）。
 *
 * `active` 语义：**本 pane 是否为当前可见页**（由 NovelPage 的 activeTab 下发，
 * 默认 `true`）。隐藏 ≠ 卸载——小说五个子页在 NovelPage 里常驻挂载、仅靠 CSS
 * `display:none` 隐藏，本页的 window 级监听在隐藏期间依然活着；因此所有
 * **窗口级副作用**（window keydown 快捷键、novel:open-chapter 全局事件）都要按
 * `active` 门控，否则会出现：F11 在书架/设定页翻转隐藏阅读页的专注模式并
 * `preventDefault` 吞掉浏览器全屏、一次 Ctrl+S 同时保存设定与本页章节、
 * readMode 残留时 ←/→ 在别的子页翻章。
 *
 * 默认 `true`：不传即按「当前页」处理——孤立渲染本页（既有测试写法）行为不变。
 */
interface ChapterPageProps {
  active?: boolean
}

const ChapterPage: React.FC<ChapterPageProps> = ({ active = true }) => {
  const outlines = useOutlineStore((s) => s.outlines)
  const loadOutlines = useOutlineStore((s) => s.loadOutlines)
  const [tabs, setTabs] = useState<ChapterTabData[]>([])
  const [activeKey, setActiveKey] = useState<string>('')
  // 跨页切书闸门（novelSwitchGuard）的同步探针需要读最新 tabs/activeKey；state 闭包
  // 在登记时是旧值，故镜像到 ref（与 activeRef 同款纪律）。
  const tabsRef = useRef<ChapterTabData[]>([])
  tabsRef.current = tabs
  const activeKeyRef = useRef('')
  activeKeyRef.current = activeKey
  // 载入在途的标签（key = node.id）：置位期间该标签的编辑区渲染 loading 占位而非编辑区。
  // 为什么需要：章节是「先塞空 tab 让编辑器立即可输入、再异步灌正文」，载入完成时无条件
  // 覆盖 scenes 并把 saved 置 true——作者在载入窗口里敲的字无声消失、脏标志还被抹掉
  // （此后再关标签连确认都不弹）。用独立 state 而不进 ChapterTabData，是为了不动跨线共享的
  // 类型契约。
  const [loadingKeys, setLoadingKeys] = useState<Record<string, boolean>>({})
  const setTabLoading = (key: string, loading: boolean) => {
    setLoadingKeys((prev) => {
      const next = { ...prev }
      if (loading) next[key] = true
      else delete next[key]
      return next
    })
  }
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
  // 摘要/问书都是「发起章 → 迟到响应」的异步面：发起时自增 epoch 并快照章号，回填前
  // 两者都要校验（守卫形态对齐同页 searchSeqRef）。切章会自增 epoch，旧响应因此必然
  // 被丢弃——否则上一章的摘要被回填进新章后，toggleSummary 见 summaryText 非空就不再
  // 发请求，错章内容长期驻留；问书答案同理会串进新章的会话。
  const summarySeqRef = useRef(0)
  const askSeqRef = useRef(0)
  const [askTarget, setAskTarget] = useState<{ selection: string } | null>(null)
  const [askQuestion, setAskQuestion] = useState('')
  // 会话式问书：同一章内的问答按序累积（user/assistant 区分样式），追问时随请求回传后端
  const [askMessages, setAskMessages] = useState<ReadingAskMessage[]>([])
  const [askLoading, setAskLoading] = useState(false)
  const [askError, setAskError] = useState<string | null>(null)
  const [searchOpen, setSearchOpen] = useState(false)
  const [searchQuery, setSearchQuery] = useState('')
  const [searchLoading, setSearchLoading] = useState(false)
  const [searchHits, setSearchHits] = useState<NovelSearchHitData[]>([])
  const [searchError, setSearchError] = useState<string | null>(null)
  const [exportOpen, setExportOpen] = useState(false)
  const [illusOpen, setIllusOpen] = useState(false)
  // v4 场景章整章重写（t4-C3 收官）：场景工程章在 ChapterPage 编辑，重写入口挂这里
  const [rwOpen, setRwOpen] = useState(false)
  const [rwHistOpen, setRwHistOpen] = useState(false)
  const pendingSearch = useRef<{ nodeId: string; query: string; paragraphIndex: number; charOffset: number } | null>(null)
  // 每次点击搜索命中自增：同章内重复点命中时 readMode/readNodeId 均不变，
  // 定位 effect 若只依赖二者则不会重跑、pendingSearch 永不被消费（回归缺陷修复）。
  const [searchLocateSeq, setSearchLocateSeq] = useState(0)
  const readingScrollRef = useRef<HTMLDivElement | null>(null)
  const readScrollTimer = useRef<ReturnType<typeof setTimeout> | null>(null)
  const autoScrollRef = useRef<number | null>(null)
  // 当前激活章节（须在首个引用它的 useEffect 依赖数组之前定义，避免 TDZ）
  const activeTab = tabs.find((t) => t.node.id === activeKey) ?? null
  const handleSaveRef = useRef<() => Promise<boolean>>(() => Promise.resolve(false))
  const handleNavRef = useRef<{ prev: () => void; next: () => void }>({ prev: () => {}, next: () => {} })
  const sceneTextareaRefs = useRef<Map<number, HTMLTextAreaElement>>(new Map())

  // active 的最新值镜像（每渲染同步写）。window 级监听的注册只做一次（或只依赖
  // 少量状态），闭包若直接捕获 active 会永远读到首渲染的值 → 门控形同虚设；
  // 经 ref 读取即可保证「当前是否可见」总是最新，且不因切 tab 重挂监听。
  const activeRef = useRef(active)
  activeRef.current = active

  // 世界构建工作台：大纲树位于壳层左 zone，点击经 novel:open-chapter 事件进入
  const handleSelectNodeRef = useRef<(node: OutlineNode) => void>(() => {})
  useEffect(() => { handleSelectNodeRef.current = handleSelectNode })
  useEffect(() => {
    const handler = (e: Event) => {
      // 按 active 门控（规格线2）：隐藏时收下事件会在不可见的阅读页里凭空多出 tab。
      // 风险判断（已读码核实，不是静默不改）：本事件**唯一**派发点是
      // NovelPage.handleOpenChapter，而目录树所在的 .novel-side-zone 在书架/设定/
      // 角色/创作 tab 上是 `display:none`（novel-workspace.css:272-291），只有阅读
      // tab 才可见——即真人可点到的路径上 active 早已为 true，所以这项门控是**防御性**
      // 的（挡程序化派发 / 将来 CSS 放开左侧 zone 的情形），不是当下可复现的缺陷。
      // 与之配套：NovelPage 把派发推迟到切 tab commit 之后（pendingChapter effect），
      // 否则同 tick 同步派发时隐藏页拿到的 active 仍是 false，会被这里丢掉、点目录树失效；
      // NovelPage.test 有一条守卫用例锁这个时序。
      if (!activeRef.current) return
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
    // v4.365：上报防抖 250ms——原每击键（activeTab 引用变）全文 join+countTextChars
    // 并派发事件拖动壳层 inspector 重渲染；防抖后输入期间不付全文遍历。
    const timer = window.setTimeout(() => {
    window.dispatchEvent(new CustomEvent('novel:chapter-active', {
      detail: {
        id: activeTab.node.id,
        title: activeTab.node.title,
        words: countTextChars(activeTab.scenes.join('\n')),
        saved: activeTab.saved,
        status: activeTab.node.status,
        chapterNum: activeTab.chapterNum,
      },
    }))
    }, 250)
    return () => window.clearTimeout(timer)
  }, [activeTab])

  // 专注模式：通知壳层收起左右 zone
  useEffect(() => {
    window.dispatchEvent(new CustomEvent('novel:focus-mode', { detail: { active: focusMode } }))
  }, [focusMode])

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      // ── 窗口级快捷键的 active 门控（规格线2）──────────────────────────
      // 「一个 Ctrl+S 只能有一个归属」：阅读 tab ＝ 保存当前阅读章、设定 tab ＝
      // 保存设定、其余子页不响应。本页常驻挂载，若不门控就会与设定页各存一份；
      // 且创作页正在写的正文反而不保存。F11 同理（隐藏页吞掉浏览器全屏），
      // readMode 残留时 ←/→ 同理（在书架/设定页翻隐藏阅读页的章）。
      //
      // 非当前页时**直接返回**：连 preventDefault 都不做——否则隐藏页仍会吞掉
      // 浏览器 F11 全屏与 Ctrl+S 的默认行为，只是「自己什么都不干」，用户侧
      // 表现依然是快捷键失效。（Esc 一并门控：隐藏页不该被别页的 Esc 改状态。）
      if (!activeRef.current) return
      if (e.key === 'Escape') {
        if (readMode) { setReadMode(false); return }
        if (focusMode) { setFocusMode(false); return }
      }
      if (e.key === 'F11') { e.preventDefault(); setFocusMode((p) => !p) }
      if ((e.ctrlKey || e.metaKey) && e.key === 's') {
        e.preventDefault()
        void handleSaveRef.current()
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
  // V4 场景制探测（每项目一次）：浏览器 mock/旧壳无此绑定 → 保持 blob 模式。
  // ref 承载结论：检测完成前打开章节也拿到最新值（避免竞态落到 blob 模式）。
  const projectV4Ref = useRef(false)
  useEffect(() => {
    projectV4Ref.current = false
    if (!projectPath) return
    let alive = true
    try {
      IsProjectV4().then((v) => {
        if (!alive) return
        projectV4Ref.current = !!v
      }).catch(() => { /* blob 模式 */ })
    } catch { /* 浏览器/mock 下 window.go 不存在：同步抛 → blob 模式 */ }
    return () => { alive = false }
  }, [projectPath])

  // 章节载入：V4 主线章读场景拼装（逐场景框 + id 按索引对齐，blob 已是 Go 侧投影）；
  // 分支/V3/未知/场景读取失败 → 整章 blob 单框（宁显示勿空白）。
  const loadChapterIntoTab = useCallback(async (key: string, node: OutlineNode, chNum: number, sceneMode: boolean) => {
    const requestedPath = useAppStore.getState().projectPath
    if (sceneMode) {
      try {
        const scenes = await GetChapterScenes(chNum)
        if (useAppStore.getState().projectPath !== requestedPath) return
        const list = Array.isArray(scenes) ? scenes : []
        const boxes = list.map((s) => {
          const c = (s as Record<string, unknown>)?.content
          return typeof c === 'string' ? c : ''
        })
        const ids = list.map((s) => {
          const v = (s as Record<string, unknown>)?.id
          return typeof v === 'string' ? v : ''
        })
        updateTabByKey(key, 'scenes', boxes.length > 0 ? boxes : [''])
        updateTabByKey(key, 'sceneIds', ids)
        updateTabByKey(key, 'sceneBacked', boxes.length > 0)
        updateTabByKey(key, 'saved', true)
        return
      } catch { /* 落回整章 blob */ }
    }
    try {
      const result = node.branch ? await app.GetChapterBranch(chNum, node.branch) : await app.GetChapter(chNum)
      if (useAppStore.getState().projectPath !== requestedPath || !result?.content) return
      const content = typeof result.content === 'string' ? result.content : ''
      updateTabByKey(key, 'scenes', [content])
      updateTabByKey(key, 'sceneIds', [])
      updateTabByKey(key, 'sceneBacked', false)
      updateTabByKey(key, 'saved', true)
    } catch (e) {
      // v4.350：载入失败可见化——此前只 console.error，章节 tab 白板且未被
      // 标为未保存，用户误以为正文丢了。
      console.error('load chapter failed:', e)
      message.error(`章节 ${chNum} 载入失败：${e instanceof Error ? e.message : String(e)}`)
    }
  }, [])

  useEffect(() => {
    // 跨页切书（书架经 `novelSwitchGuard` 确认过）→ 不再重复提示；未经闸门的切书
    // 路径仍如实告知。此前这里是**无条件** `setTabs([])`：多标签里未保存的正文
    // 静默蒸发，是全板块最后一条没有任何提示的丢稿路径。
    const confirmed = takeDiscardConfirmed('chapter-tabs')
    if (!confirmed && tabsRef.current.some(needsCloseConfirm)) {
      message.warning('已切换小说：阅读页未保存的章节修改未保留')
    }
    setTabs([]); setActiveKey(''); setReadMode(false); setLoadingKeys({})
    if (!projectPath) return
    void (async () => {
      let v4 = false
      try { v4 = await IsProjectV4() } catch { v4 = false }
      projectV4Ref.current = v4
      await loadOutlines()
      const progress = readReadingProgress(projectPath)
      if (!progress) return
      const node = findAllLeaves(sortNodes(useOutlineStore.getState().outlines))
        .find((n) => n.id === progress.nodeId)
      if (!node) return
      // 「恢复上次阅读章」是迟到的异步回填：上面两处 await（IsProjectV4 / loadOutlines）
      // 期间作者完全可能已经从目录点了章（handleSelectNode 往缓冲里追加了标签）。此处
      // `setTabs([...])` 是**整体替换**缓冲——照写就会把作者刚打开的标签连同其载入结果
      // 一起换掉（updateTabByKey 随后找不到 key 而静默 no-op）。已有标签＝用户意图优先，
      // 恢复动作放弃（同族纪律：迟到的异步结果不得覆盖更新的状态）。
      if (tabsRef.current.length > 0) return
      setActiveKey(node.id)
      setTabs([createTabData(node)])
      const chNum = node.order_index || 0
      if (chNum > 0) {
        // 自动恢复的章节同样是「空 tab → 异步灌正文」：载入在途期该 tab 只读 + loading
        // （见 handleSelectNode 的因果说明）。
        setTabLoading(node.id, true)
        try { await loadChapterIntoTab(node.id, node, chNum, v4 && !node.branch) } finally { setTabLoading(node.id, false) }
      }
    })().catch((e) => console.error('load project failed:', e))
  }, [projectPath, loadOutlines, loadChapterIntoTab])

  // 跨页切书未保存保护（`novelSwitchGuard` 的登记端）：阅读页是**多标签缓冲**，
  // 脏的可能是非当前标签——只有「恰好一个脏标签且它就是当前标签」才具备一键保存
  // 能力，其余情况 canSave=false，闸门如实降级为两选（放弃修改并切换 / 取消），
  // 不假装能一次存齐。
  useEffect(() => registerNovelDirtyProvider({
    id: 'chapter-tabs',
    label: () => {
      const n = tabsRef.current.filter(needsCloseConfirm).length
      return n > 1 ? `阅读页 ${n} 个未保存章节` : '阅读页未保存的章节'
    },
    dirty: () => tabsRef.current.some(needsCloseConfirm),
    canSave: () => {
      const d = tabsRef.current.filter(needsCloseConfirm)
      return d.length === 1 && d[0].node.id === activeKeyRef.current
    },
    save: () => handleSaveRef.current(),
  }), [])

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
    if (chNum <= 0) return
    // 载入在途期该 tab 的编辑区只读 + loading（渲染占位而非编辑区）：空 tab 先落到
    // 屏幕上，编辑器立即可输入，而载入完成会无条件覆盖 scenes 并置 saved=true——
    // 作者的输入被静默覆盖后连脏标志都没有。占位渲染从根上消除「可输入窗口」，
    // 比「载入完成时检测并保留输入」更不易漏（后者仍要面对输入与回填的交错时序）。
    setTabLoading(key, true)
    try {
      await loadChapterIntoTab(key, node, chNum, projectV4Ref.current && !node.branch)
    } finally {
      setTabLoading(key, false)
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
    // 关掉的标签若仍在载入，其 loading 标记留着会污染同名节点日后的复用
    setTabLoading(key, false)
    if (key === activeKey) setActiveKey(n.length > 0 ? n[n.length - 1].node.id : '')
  }

  const requestCloseTab = (key: string) => {
    const target = tabs.find((t) => t.node.id === key)
    if (target && needsCloseConfirm(target)) {
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

  // ── 重写历史 / 整章重写：服务端写盘后的前端刷新护栏 ──
  /** 重写历史面板按「章号」工作，而阅读页是**多标签缓冲**：脏的可能是非当前标签。
   *  判脏一律按章号扫整个缓冲——只看 activeTab 既会漏掉别的标签的未保存正文，
   *  也会把别的章的脏算到当前章头上（弹窗给不出对得上的理由）。 */
  const dirtyTabsOfChapter = (chNum: number): ChapterTabData[] =>
    tabsRef.current.filter((t) => t.chapterNum === chNum && needsCloseConfirm(t))

  // 面板按 activeTab.chapterNum 取章，而它可能在面板打开期间被切走；用 ref 记住
  // 「面板当前操作的章」，onApplied 时按它判脏，而不是读回填时刻的 activeTab。
  const rwHistChapterRef = useRef<number | null>(null)
  if (rwHistOpen) rwHistChapterRef.current = activeTab?.chapterNum ?? null
  const rwModalChapterRef = useRef<number | null>(null)
  if (rwOpen) rwModalChapterRef.current = activeTab?.chapterNum ?? null

  /** 应用重写 / 恢复原文后刷新正文。脏则**不刷新**并如实提示：`loadChapterIntoTab`
   *  会把磁盘正文整章灌回缓冲并把 saved 置 true，作者的本地版本被无声覆盖、脏标志被
   *  抹掉（此后再关标签连确认都不弹）。口径对齐创作页 `refreshEditorAfterServerRewrite`
   *  （同样脏则不重载）；「重写已在服务端生效」这句必须留着——否则作者会以为重写失败。 */
  const refreshChapterAfterServerRewrite = (chNum: number | null) => {
    if (chNum == null) return
    if (dirtyTabsOfChapter(chNum).length > 0) {
      message.warning('重写已在服务端生效；当前标签有未保存的修改，已保留你的本地版本，未刷新')
      return
    }
    const tab = tabsRef.current.find((t) => t.node.id === activeKeyRef.current && t.chapterNum === chNum)
      ?? tabsRef.current.find((t) => t.chapterNum === chNum)
    if (!tab) return
    void loadChapterIntoTab(tab.node.id, tab.node, chNum, true)
  }

  /** 打开「重写历史」前过脏闸：面板里的「应用此版本 / 恢复原文」会覆盖本章正文，
   *  先让作者保存（与创作页 `openRewriteModal` 同款口径；✕/Esc 一律＝取消）。 */
  const openRewriteHistory = () => {
    if (!activeTab) return
    const chNum = activeTab.chapterNum
    const dirty = dirtyTabsOfChapter(chNum)
    const proceedToPanel = () => setRwHistOpen(true)
    if (dirty.length === 0) { proceedToPanel(); return }
    // 「先保存」只在「唯一脏标签且正是当前标签」时成立（与切书闸门 canSave 同口径）：
    // 脏在别的标签上无法一次存齐，如实降级为两选（不提供 onSave），不假装能一次存齐。
    const canSaveOneShot = dirty.length === 1 && dirty[0].node.id === activeKeyRef.current
    chooseUnsavedAction({
      title: '正文有未保存的修改',
      message: `第 ${chNum} 章有未保存的修改，重写历史里的「应用此版本 / 恢复原文」会覆盖正文。先保存，或继续打开（本地版本仍会保留）。`,
      onSave: canSaveOneShot
        ? () => { void handleSaveRef.current().then((ok) => { if (ok) proceedToPanel() }) }
        : undefined,
      onDiscard: proceedToPanel,
    })
  }

  // v4.365：useCallback 稳定 onUpdate 引用——ChapterEditor memo 才能命中
  const updateTab = useCallback(function updateTab<K extends keyof ChapterTabData>(field: K, value: ChapterTabData[K]) {
    setTabs((prev) => {
      const i = prev.findIndex((t) => t.node.id === activeKey)
      if (i < 0) return prev
      const c = [...prev]
      c[i] = { ...c[i], [field]: value }
      return c
    })
  }, [activeKey])

  const handleSave = async (): Promise<boolean> => {
    // 返回值供跨页切书闸门（novelSwitchGuard 的 save）判断「先保存再切换」是否成立；
    // 同时把静默 return 改成可见提示——点了保存却什么都没发生是 v4.421 已修过的
    // 同类缺陷（CreatePage.saveActive 口径）。
    if (!activeTab || activeTab.chapterNum < 1) { message.warning('当前没有可保存的章节'); return false }
    const c = activeTab.scenes.join('\n\n')
    if (!c) { message.warning('正文为空，无需保存'); return false }
    // v4.354：保存快照——V4 场景章逐场景串行 N 次后端往返（数百 ms 起步），
    // 期间继续打字后「完成侧强制 saved=true」会把未保存保护打掉（关闭不弹
    // 确认→新增文字静默丢失）。完成侧只有快照仍与当前缓冲一致才置 saved。
    const savedSnapshot = activeTab.scenes.join('\u0000')
    try {
      if (activeTab.node.branch) {
        await app.SaveChapterBranchContent(activeTab.chapterNum, activeTab.node.branch, c)
      } else if (activeTab.sceneBacked) {
        // V4 场景制：逐场景 SaveScene（blob 投影由 Go 侧同一调用内同步）；
        // 无 id 的框（罕见兜底）先 CreateScene 补建再存。
        const ids = [...(activeTab.sceneIds ?? [])]
        for (let i = 0; i < activeTab.scenes.length; i++) {
          let id = ids[i]
          if (!id) {
            const created = await CreateScene(activeTab.chapterNum, `scene-${i + 1}`, `场景 ${i + 1}`)
            const cid = (created as Record<string, unknown> | null)?.id
            if (typeof cid === 'string' && cid) { id = cid; ids[i] = id }
          }
          if (!id) throw new Error(`场景 ${i + 1} 缺少 id`)
          await SaveScene(activeTab.chapterNum, id, activeTab.scenes[i])
        }
        updateTab('sceneIds', ids)
      } else {
        await app.SaveChapterContent(activeTab.chapterNum, c)
      }
      // 完成侧：只有「保存时快照 == 当前缓冲」才置 saved（setTabs 函数式读最新，
      // 保存期间继续打字则保持 saved=false，未保存保护不丢）
      const keyAtSave = activeTab.node.id
      setTabs((prev) => prev.map((t) => (
        t.node.id === keyAtSave && t.scenes.join('\u0000') === savedSnapshot ? { ...t, saved: true } : t
      )))
      const unchanged = tabs.some((t) => t.node.id === keyAtSave && t.scenes.join('\u0000') === savedSnapshot)
      message.success(unchanged ? '已保存' : '已保存（保存期间有新改动，请再次保存）')
      return true
    } catch (e: unknown) {
      message.error(`保存失败：${e instanceof Error ? e.message : String(e)}`)
      return false
    }
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
  // 迟到的异步响应只能读到发起那一刻的闭包值，无法据此判断「还是不是同一章」——
  // 镜像到 ref，回填前与发起时快照比对（epoch 之外的第二道守卫：epoch 只保证
  // 「没有更新的请求」，章号还保证「不是同一标签换章后的一系列请求」）。
  const readNodeIdRef = useRef('')
  readNodeIdRef.current = readNodeId

  // ── 阅读偏好（字号 / 行距 / 版宽，全局持久化） ──
  const patchReadPrefs = (p: Partial<ReadingSettings>) => {
    setReadPrefs((prev) => {
      const next = { ...prev, ...p }
      writeReadingSettings(next)
      return next
    })
  }

  // ── 阅读滚动：进度条 + 章节位置记忆 ──
  const handleReadScroll = () => {
    const el = readingScrollRef.current
    if (!el || !activeTab) return
    setSelToolbar(null)
    setReadProgress(scrollPct(el))
    if (readScrollTimer.current) clearTimeout(readScrollTimer.current)
    readScrollTimer.current = setTimeout(() => {
      saveScrollTop(activeTab.node.id, el.scrollTop)
    }, 250)
  }
  useEffect(() => {
    if (!readMode || !readNodeId) return
    const el = readingScrollRef.current
    // 切换章节时先回到顶部，再恢复该章保存的位置
    if (el) el.scrollTop = 0
    const saved = readSavedScrollTop(readNodeId)
    const raf = requestAnimationFrame(() => {
      if (el && saved > 0) el.scrollTop = saved
    })
    setSummaryOpen(false)
    setSummaryText(null)
    setSummaryError(null)
    return () => {
      cancelAnimationFrame(raf)
      if (el && el.scrollTop > 0) saveScrollTop(readNodeId, el.scrollTop)
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

  const toggleBookmark = () => {
    const el = readingScrollRef.current
    if (!el || !activeTab) return
    const scrollTop = Math.max(0, Math.round(el.scrollTop))
    persistBookmarks(toggleBookmarkInList(bookmarks, {
      nodeId: activeTab.node.id,
      title: activeTab.node.title || '未命名章节',
      scrollTop,
      pct: scrollPct(el),
      text: textAtScrollTop(el, scrollTop),
      createdAt: Date.now(),
    }))
  }

  const jumpBookmark = (b: ReadingBookmark) => {
    const el = readingScrollRef.current
    if (!el) return
    el.scrollTop = b.scrollTop
    setBookmarkOpen(false)
  }

  const removeBookmark = (b: ReadingBookmark) => {
    persistBookmarks(removeBookmarkInList(bookmarks, b))
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

  // target：归属覆盖。划词路径省略 → 归当前阅读章节；搜索命中「落为划线」传命中所在章节，
  // 该章未打开时标注先落库，待其进入阅读模式由既有回渲染 effect 呈现。
  const addHighlight = (
    color: AnnotationColor,
    withNote: boolean,
    textOverride?: string,
    target?: { nodeId: string; title?: string },
  ): boolean => {
    let text = textOverride ?? ''
    if (!text) {
      const selInfo = readSelectionInRoot(readingScrollRef.current)
      if (!selInfo) return false
      text = selInfo.text.trim()
      window.getSelection()?.removeAllRanges()
    }
    if (!text || text.length > 200) return false
    const ann: ReadingAnnotation = {
      id: `ann_${Date.now()}_${Math.random().toString(36).slice(2, 7)}`,
      nodeId: target?.nodeId ?? readNodeId,
      title: target?.title ?? (activeTab?.node.title || '未命名章节'),
      color,
      text,
      note: '',
      createdAt: Date.now(),
    }
    persistAnnotations([...annotations, ann])
    setSelToolbar(null)
    if (withNote) { setNoteTarget(ann); setNoteDraft('') }
    return true
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

  // 选中文本 → 浮动工具条（仅限阅读列内、单段落内选择；校验与划线共用 readSelectionInRoot）
  const handleReadingMouseUp = () => {
    const selInfo = readSelectionInRoot(readingScrollRef.current)
    if (!selInfo) { setSelToolbar(null); return }
    setSelText(selInfo.text)
    setSelToolbar({ x: selInfo.rect.left + selInfo.rect.width / 2, y: selInfo.rect.top })
  }

  // ── AI 伴读：章节摘要 + 划线提问 ──
  const chapterText = activeTab?.scenes?.join('\n\n') || ''

  const runSummary = async () => {
    if (summaryLoading || !activeTab || !readNodeId) return
    // 发起时快照：缓存键与回填校验都用这一章，绝不读执行时刻的 readNodeId
    // （否则 A 章的摘要会写进 B 章的缓存键，B 章再也拿不到自己的摘要）。
    const nodeId = readNodeId
    const cached = summaryCache.current[nodeId]
    if (cached) { setSummaryText(cached); return }
    const seq = ++summarySeqRef.current
    setSummaryLoading(true)
    setSummaryError(null)
    try {
      const text = await askReadingAssistant('summary', activeTab.node.title || '', chapterText, '', '')
      if (seq !== summarySeqRef.current || readNodeIdRef.current !== nodeId) return
      summaryCache.current[nodeId] = text
      setSummaryText(text)
    } catch (err) {
      if (seq !== summarySeqRef.current || readNodeIdRef.current !== nodeId) return
      setSummaryError(errText(err, '摘要生成失败'))
    } finally {
      // loading 按发起时的 seq 收尾：切章时已由切章 effect 清掉，旧响应不得再动它
      // （否则会清掉新请求的 loading——ChapterAnalysisPanel 的 cmpSeqRef 同款先例）。
      if (seq === summarySeqRef.current) setSummaryLoading(false)
    }
  }

  const toggleSummary = () => {
    if (summaryOpen) { setSummaryOpen(false); return }
    setSummaryOpen(true)
    if (!summaryText && !summaryLoading) void runSummary()
  }

  // 会话清空策略：问书会话按章保留（同章内划不同段落也能连续追问），切章即清空；
  // 关闭弹窗保留会话，弹窗内提供「清空会话」手动重置。
  // 切章同时作废在途请求（epoch 自增 → 回填校验必然失败）并在此收尾 loading：
  // 旧响应被守卫拦下后不会再走到 finally，否则新章会永远停在「问书加载中」。
  useEffect(() => {
    summarySeqRef.current++
    askSeqRef.current++
    setSummaryLoading(false)
    setAskLoading(false)
    setAskMessages([])
    setAskError(null)
  }, [readNodeId])

  const openAsk = (selection: string) => {
    setAskTarget({ selection })
    setAskQuestion('')
    setAskError(null)
    setAskLoading(false)
  }

  const runAsk = async () => {
    const q = askQuestion.trim()
    if (!q || !askTarget || askLoading || !activeTab) return
    // 发起时快照章号：迟到答案只能落回它提问的那一章（切章后直接丢弃）。
    const nodeId = readNodeId
    const history = buildAskHistory(askMessages)
    setAskQuestion('')
    setAskError(null)
    setAskMessages((m) => [...m, { role: 'user', content: q }])
    const seq = ++askSeqRef.current
    setAskLoading(true)
    try {
      const answer = await askReadingAssistant('ask', activeTab.node.title || '', chapterText, askTarget.selection, q, history)
      if (seq !== askSeqRef.current || readNodeIdRef.current !== nodeId) return
      setAskMessages((m) => [...m, { role: 'assistant', content: answer }])
    } catch (err) {
      // 失败回滚本轮提问（半截问答不混入后续历史），问题放回输入框便于重试；
      // 已切章则整体丢弃——回滚/提示都只对新章的会话有意义。
      if (seq !== askSeqRef.current || readNodeIdRef.current !== nodeId) return
      setAskMessages(rollbackLastUserMessage)
      setAskQuestion(q)
      setAskError(errText(err, '提问失败'))
    } finally {
      if (seq === askSeqRef.current) setAskLoading(false)
    }
  }

  const clearAskSession = () => {
    setAskMessages([])
    setAskError(null)
    setAskLoading(false)
  }

  // 新回答到达时把会话列表滚到底部
  const askThreadRef = useRef<HTMLDivElement | null>(null)
  useEffect(() => {
    const el = askThreadRef.current
    if (el) el.scrollTop = el.scrollHeight
  }, [askMessages, askLoading, askTarget])

  // 划线回渲染：清除旧 mark 后按摘录文本在段落内重新定位（内容变更时自然失效）。
  // 主体已抽为 chapter/readingAnnotation 模块级 DOM 工具（readingHighlight 同款搬移法）：
  // 组件内只保留接线——按当前章节过滤划线，滚动根与 openAnnotation 经参数传入。
  useEffect(() => {
    const root = readingScrollRef.current
    if (!root) return
    renderAnnotationHighlights(
      root,
      annotations.filter((a) => a.nodeId === readNodeId),
      openAnnotation,
    )
  }, [readMode, readNodeId, annotations])

  // ── 朗读/搜索定位高亮：按文本在段落 DOM 中回定位 ──
  // 高亮主体（textNodesOf/clearReadingHighlight/highlightSearchHitAt/applyTextHighlight）
  // 均已抽为 chapter/readingHighlight 模块级纯 DOM 工具（滚动根与 readMode 经参数传入，
  // 无组件状态依赖），模块导入身份天然恒定。此处薄包装沿用原闭包的读取时机——调用时
  // 才取滚动根 ref 与最新 readMode；并继续走 ref 供流式调用（TTS 逐句 / 搜索定位 effect）
  // 持恒定入口：若在 effect 里直接传捕获值，readMode 变化后的重渲染与 effect 清理之间存在
  // 定时器已触发但旧闭包仍存活的微小窗口，ref 每渲染同步可彻底消除该差异。
  const applyTextHighlight = (rawText: string, className: string): boolean =>
    highlightFirstMatch(readingScrollRef.current, readMode, rawText, className)
  const applyTextHighlightRef = useRef(applyTextHighlight)
  applyTextHighlightRef.current = applyTextHighlight

  const handleTtsSentence = (sentence: string) => {
    applyTextHighlight(sentence, 'novel-reading-current')
  }
  const handleTtsClear = () => {
    clearReadingHighlight(readingScrollRef.current, 'novel-reading-current')
  }

  // ── 全文搜索（防抖 300ms；回车立即搜索；点击结果打开章节并按段落定位） ──
  // 每次搜索带自增序号，防止慢响应晚到覆盖新结果。
  const searchSeqRef = useRef(0)
  const runSearchRef = useRef<() => void>(() => {})
  runSearchRef.current = () => {
    const q = searchQuery.trim()
    if (!q) { setSearchHits([]); setSearchError(null); setSearchLoading(false); return }
    const seq = ++searchSeqRef.current
    setSearchLoading(true)
    searchNovelAll(q)
      .then((hits) => {
        if (seq !== searchSeqRef.current) return
        setSearchHits(hits)
        setSearchError(null)
      })
      .catch((err) => {
        if (seq !== searchSeqRef.current) return
        setSearchHits([])
        setSearchError(errText(err, '搜索失败'))
      })
      .finally(() => {
        if (seq === searchSeqRef.current) setSearchLoading(false)
      })
  }
  useEffect(() => {
    if (!searchOpen) return
    if (!searchQuery.trim()) { setSearchHits([]); setSearchError(null); setSearchLoading(false); return }
    const timer = setTimeout(() => runSearchRef.current(), 300)
    return () => clearTimeout(timer)
  }, [searchOpen, searchQuery])

  const searchSummary = useMemo(() => summarizeSearch(searchHits), [searchHits])

  const openSearchHit = (hit: NovelSearchHitData) => {
    const node = outlineLeaves.find((n) => n.id === hit.node_id)
    if (!node) return
    pendingSearch.current = {
      nodeId: hit.node_id,
      query: searchQuery.trim(),
      paragraphIndex: hit.paragraph_index,
      charOffset: hit.char_offset,
    }
    setSearchOpen(false)
    setSearchLocateSeq((v) => v + 1)
    handleSelectNode(node)
    setReadMode(true)
  }

  // 搜索命中「落为划线」：命中是区间口径（段落索引 + 段内 rune 偏移），划线是摘录文本
  // 口径（段落内回定位），经 searchHitAnchor 适配为 addHighlight 入参后走既有持久化与
  // 回渲染管线（持久化与回渲染路径零新增）；成功后 message 反馈。
  const addSearchHitHighlight = (hit: NovelSearchHitData) => {
    const anchor = searchHitAnchor(hit, searchQuery)
    if (!anchor) return
    if (addHighlight('yellow', false, anchor.text, anchor)) {
      const excerpt = anchor.text.length > 12 ? `${anchor.text.slice(0, 12)}…` : anchor.text
      message.success(`已落为划线「${excerpt}」`)
    }
  }

  // 打开目标章节后等待正文渲染，再按段落索引定位该处命中并短暂高亮；
  // 定位失败（章节被编辑/标题命中等）时降级为全文首个命中。
  useEffect(() => {
    const target = pendingSearch.current
    if (!readMode || !readNodeId || !target || target.nodeId !== readNodeId) return
    let tries = 0
    const timer = window.setInterval(() => {
      tries++
      const root = readingScrollRef.current
      const found = root
        ? Array.from(root.querySelectorAll('.novel-reading-p')).some((p) => (p.textContent || '').includes(target.query))
        : false
      if (!found && tries <= 30) return
      window.clearInterval(timer)
      pendingSearch.current = null
      if (!found || !root) return
      const paras = Array.from(root.querySelectorAll<HTMLElement>('.novel-reading-p'))
      const flashed = (target.paragraphIndex >= 0
        && highlightSearchHitAt(root, paras, target.paragraphIndex, target.query, target.charOffset))
        || applyTextHighlightRef.current(target.query, 'novel-reading-search-hit')
      // 短暂高亮：2.6s 后自动清除（触发时再读 ref，章节已切换则清在新根上，与原闭包一致）
      if (flashed) window.setTimeout(() => clearReadingHighlight(readingScrollRef.current, 'novel-reading-search-hit'), 2600)
    }, 120)
    return () => window.clearInterval(timer)
    // highlightSearchHitAt/clearReadingHighlight 为模块级常量身份（exhaustive-deps 豁免），
    // effect 实际触发时机由 readMode/readNodeId 变化 + searchLocateSeq 自增共同决定：
    // 仅靠前者时，同章内（阅读模式已开、章节未换）再次点命中不重跑、无法重新定位。
  }, [readMode, readNodeId, searchLocateSeq])

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
                  <ReadingChrome
                    title={activeTab.node.title}
                    totalWords={totalWords}
                    saved={activeTab.saved}
                    atFirst={atFirst}
                    atLast={atLast}
                    onPrev={handlePrevChapter}
                    onNext={handleNextChapter}
                    bookmarks={bookmarks}
                    bookmarkOpen={bookmarkOpen}
                    onBookmarkOpenChange={setBookmarkOpen}
                    onToggleBookmark={toggleBookmark}
                    onJumpBookmark={jumpBookmark}
                    onRemoveBookmark={removeBookmark}
                    autoScrolling={autoScrolling}
                    onToggleAutoScroll={() => (autoScrolling ? stopAutoScroll() : startAutoScroll())}
                    anns={chapterAnns}
                    onJumpAnn={jumpToAnnotation}
                    onDeleteAnn={deleteAnnotation}
                    searchOpen={searchOpen}
                    onSearchOpenChange={(v) => { setSearchOpen(v); if (!v) { setSearchHits([]); setSearchError(null) } }}
                    searchQuery={searchQuery}
                    onSearchQueryChange={setSearchQuery}
                    onSearchSubmit={() => runSearchRef.current()}
                    searchSummary={searchSummary}
                    searchHits={searchHits}
                    searchLoading={searchLoading}
                    searchError={searchError}
                    onOpenSearchHit={openSearchHit}
                    onAddSearchHitHighlight={addSearchHitHighlight}
                    prefs={readPrefs}
                    onPrefsChange={patchReadPrefs}
                    onEdit={() => setReadMode(false)}
                  />
                ) : (
                  <EditChrome
                    tabItems={tabItems}
                    activeKey={activeKey}
                    onTabChange={setActiveKey}
                    onTabRemove={requestCloseTab}
                    saved={activeTab.saved}
                    onPrev={handlePrevChapter}
                    onNext={handleNextChapter}
                    atFirst={atFirst}
                    atLast={atLast}
                    getText={() => activeTab?.scenes?.join('\n\n') || ''}
                    onRead={() => setReadMode(true)}
                    focusMode={focusMode}
                    onToggleFocus={() => setFocusMode((p) => !p)}
                    canIllustrate={activeTab.chapterNum >= 1}
                    onIllustrate={() => setIllusOpen(true)}
                    onSave={handleSave}
                    canSave={totalWords > 0}
                    canRewrite={!!activeTab.sceneBacked && activeTab.chapterNum >= 1}
                    onRewrite={() => setRwOpen(true)}
                    onRewriteHistory={openRewriteHistory}
                  />
                )}
              </div>
            )}

            {/* ── 阅读模式：居中限宽衬线排版 ── */}
            {readMode ? (
              <ReadingPanel
                scrollRef={readingScrollRef}
                onScroll={handleReadScroll}
                onMouseUp={handleReadingMouseUp}
                progress={readProgress}
                prefs={readPrefs}
                title={activeTab.node.title}
                scenes={activeTab.scenes}
                summaryOpen={summaryOpen}
                onToggleSummary={toggleSummary}
                summaryLoading={summaryLoading}
                summaryText={summaryText}
                summaryError={summaryError}
                onRetrySummary={runSummary}
                onTtsSentence={handleTtsSentence}
                onTtsClear={handleTtsClear}
                atFirst={atFirst}
                atLast={atLast}
                onPrev={handlePrevChapter}
                onNext={handleNextChapter}
                totalWords={totalWords}
                onExport={() => setExportOpen(true)}
              />
            ) : (
              /* ── 编辑模式：场景多文本框 ── */
              <>
                {loadingKeys[activeTab.node.id] ? (
                  /* 载入在途：渲染只读占位而不渲染编辑区（见 handleSelectNode 的因果说明）。
                     载入完成前没有可输入的表面，「作者输入被服务端正文静默覆盖」不可达。 */
                  <div
                    data-testid="chapter-editor-loading"
                    role="status"
                    aria-busy="true"
                    style={{
                      flex: 1, display: 'flex', flexDirection: 'column',
                      alignItems: 'center', justifyContent: 'center', gap: 8,
                      color: C('color-text-secondary'),
                    }}
                  >
                    <Spin size="small" />
                    <span>正在载入第 {activeTab.chapterNum} 章正文…</span>
                    <span style={{ fontSize: 11, opacity: 0.7 }}>载入完成前编辑区不可用，以免你的输入被正文覆盖</span>
                  </div>
                ) : (
                  <ChapterEditor
                    tab={activeTab}
                    onUpdate={updateTab}
                    sceneTextareaRefs={sceneTextareaRefs}
                    ghostEnabled={false}
                  />
                )}
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

      {/* 划词工具条 / 想法编辑弹窗 / AI 问书弹窗（自 ChapterPage 拆出为受控组件） */}
      <ReadingOverlays
        readMode={readMode}
        selToolbar={selToolbar}
        selText={selText}
        onClearSel={() => setSelToolbar(null)}
        onHighlight={addHighlight}
        onAskSelection={(sel) => openAsk(sel)}
        noteTarget={noteTarget}
        onNoteClose={() => setNoteTarget(null)}
        noteDraft={noteDraft}
        onNoteDraftChange={setNoteDraft}
        onSaveNote={saveNote}
        onDeleteAnnotation={deleteAnnotation}
        askTarget={askTarget}
        onAskClose={() => setAskTarget(null)}
        askMessages={askMessages}
        askLoading={askLoading}
        askError={askError}
        askQuestion={askQuestion}
        onAskQuestionChange={setAskQuestion}
        onRunAsk={runAsk}
        onClearAsk={clearAskSession}
        askThreadRef={askThreadRef}
      />

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
      <RewriteModal
        open={rwOpen}
        chapterNum={activeTab?.chapterNum ?? null}
        onClose={() => setRwOpen(false)}
        onApplied={() => refreshChapterAfterServerRewrite(rwModalChapterRef.current)}
      />
      <RewriteHistoryPanel
        open={rwHistOpen}
        chapterNum={activeTab?.chapterNum ?? null}
        onClose={() => setRwHistOpen(false)}
        onApplied={() => refreshChapterAfterServerRewrite(rwHistChapterRef.current)}
      />
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