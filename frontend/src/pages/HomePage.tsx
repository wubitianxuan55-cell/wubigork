import { wailsApp } from '../lib/wailsApp';
import React, { useState, useEffect, useMemo } from 'react'
import {
  Button, Skeleton, message, Input, Select, Modal,
} from 'antd'
import {
  PlusOutlined, SearchOutlined, ReadOutlined, SortAscendingOutlined, UploadOutlined, GlobalOutlined,
} from '@ant-design/icons'
import { useAppStore, type ProjectCard } from '../stores/appStore'

import WelcomePage from '../components/WelcomePage'
import CreateNovelModal from '../components/novel/CreateNovelModal'
import ImportNovelModal from '../components/novel/ImportNovelModal'
import BookSearchModal from '../components/novel/BookSearchModal'
import ProjectCardItem from '../components/ProjectCardItem'
import V3Empty from '../components/V3Empty'
import { readReadingProgress } from '../utils/readingProgress'
import { formatImportSuccess, formatImportWarnings } from '../utils/novelImportReport'

type SortKey = 'recent' | 'words' | 'chapters' | 'title'

const SORT_OPTIONS: Array<{ value: SortKey; label: string }> = [
  { value: 'recent', label: '最近打开' },
  { value: 'words', label: '总字数' },
  { value: 'chapters', label: '章节数' },
  { value: 'title', label: '书名' },
]

const HomePage: React.FC = () => {
  const {
    loggedIn, login, projectOpen, projectPath, projectTitle, novelsDir,
    projects, loadProjects, openProject, deleteProject, loadNovelsDir,
  } = useAppStore()

  // 新建小说表单
  const [newModal, setNewModal] = useState(false)
  const [newTitle, setNewTitle] = useState('')
  const [newGenre, setNewGenre] = useState<string[]>([])
  const [newStyle, setNewStyle] = useState<string[]>([])

  // 导入成品小说
  const [importModal, setImportModal] = useState(false)
  const [importFile, setImportFile] = useState<{ path: string; name: string } | null>(null)
  const [importTitle, setImportTitle] = useState('')
  const [importGenre, setImportGenre] = useState<string[]>([])
  const [importStyle, setImportStyle] = useState<string[]>([])
  const [importing, setImporting] = useState(false)
  const [importExtractRange, setImportExtractRange] = useState('full')

  // 在线搜书（书源取书→拆书导入 t2）
  const [bookSearchModal, setBookSearchModal] = useState(false)

  // 书架工具条：搜索 / 排序
  const [query, setQuery] = useState('')
  const [sortKey, setSortKey] = useState<SortKey>('recent')

  // 加载状态
  const [loadingProjects, setLoadingProjects] = useState(true)

  useEffect(() => {
    if (loggedIn) {
      loadNovelsDir()
      loadProjects().finally(() => setLoadingProjects(false))
    } else {
      setLoadingProjects(false)
    }
  }, [loggedIn, loadNovelsDir, loadProjects])

  // Ctrl+N 快捷键
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key === 'n') {
        e.preventDefault()
        if (loggedIn) { resetForm(); setNewModal(true) }
      }
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [loggedIn])

  // ── 新建小说 ──
  const resetForm = () => {
    setNewTitle('')
    setNewGenre([])
    setNewStyle([])
  }

  const handleCreate = async () => {
    if (!newTitle.trim()) return
    try {
      const dir = `${novelsDir}\\${newTitle.replace(/[/\\:*?"<>|]/g, '_')}`
      const genreStr = newGenre.join('、') || '未分类'
      const styleStr = newStyle.join('、') || '默认'

      await wailsApp().CreateProject(dir, newTitle, genreStr, styleStr)

      openProject(dir, newTitle)
      await loadProjects()
      setNewModal(false)
      resetForm()
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '创建失败')
    }
  }

  // ── 导入成品小说 ──
  const handlePickImport = async () => {
    try {
      const picked = await wailsApp().GaeaPickFiles()
      const file = picked && picked.length > 0 ? picked[0] : null
      if (!file?.path) return
      setImportFile({ path: file.path, name: file.name || file.path })
      setImportTitle(file.name ? file.name.replace(/\.[^.]+$/, '') : '')
      setImportGenre([])
      setImportStyle([])
      setImportModal(true)
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '选择文件失败')
    }
  }

  const handleImport = async () => {
    if (!importFile || !importTitle.trim() || importing) return
    setImporting(true)
    try {
      const [mode, tailStr] = importExtractRange.split(':')
      const res = await wailsApp().ImportNovelBookEx(
        importFile.path,
        importTitle.trim(),
        importGenre.join('、') || '未分类',
        importStyle.join('、') || '默认',
        mode,
        tailStr ? Number(tailStr) : 0,
      )
      openProject(res.path, res.title)
      await loadProjects()
      setImportModal(false)
      // v4.279：解析报告直显（编码 + 切分策略），便于判断「这本书是不是被切错了」
      message.success(formatImportSuccess(res))
      const warnText = formatImportWarnings(res)
      if (warnText) message.warning(warnText)
      // tail×反推串联（v4.292）：按提取范围（末 N 章）导入的，引导一键反推
      // 续写大纲——flag 经 sessionStorage 交接给 CreatePage（跨页面挂载时序）。
      if (mode === 'tail') {
        const tailN = tailStr ? Number(tailStr) : 0
        Modal.confirm({
          title: '立即反推这部分大纲？',
          content: `已按提取范围导入末 ${tailN || res.chapter_count} 章。去创作间跑一次「AI 反推大纲」，即可得到这部分的结构参考（任务化后台执行）。`,
          okText: '去创作间反推',
          cancelText: '稍后再说',
          onOk: () => {
            window.dispatchEvent(new CustomEvent('novel:goto-tab', { detail: { tab: 'create' } }))
            window.dispatchEvent(new CustomEvent('novel:auto-reconstruct'))
          },
        })
      }
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '导入失败')
    } finally {
      setImporting(false)
    }
  }

  // ── 在线搜书导入完成：与文件导入同款落书架 + 报告直显（t2）。
  // 不在此关 Modal：全部成功由 Modal 自关，带失败章时留失败面板可重试（t3）。
  const handleOnlineImported = async (res: Parameters<typeof formatImportSuccess>[0]) => {
    openProject(res.path, res.title)
    await loadProjects()
    message.success(formatImportSuccess(res))
    const warnText = formatImportWarnings(res)
    if (warnText) message.warning(warnText)
  }

  // ── 失败章补下完成（t3）：刷新书架 + 增量提示 ──
  const handleChaptersAppended = async (res: { appended: number; totalChapters: number; failed?: unknown[] }) => {
    await loadProjects()
    if (res.failed && res.failed.length > 0) {
      message.warning(`补下 ${res.appended} 章，仍有 ${res.failed.length} 章未取到`)
    } else {
      message.success(`已补下 ${res.appended} 章，本书共 ${res.totalChapters} 章`)
    }
  }

  // ── 打开/关闭项目 ──
  const handleOpen = async (card: ProjectCard, goRead = false) => {
    if (projectOpen && card.path === projectPath) {
      if (goRead) {
        window.dispatchEvent(new CustomEvent('novel:goto-tab', { detail: { tab: 'chapter' } }))
      }
      return
    }
    try {
      await wailsApp().OpenProject(card.path)
      openProject(card.path, card.title)
      if (goRead) {
        window.dispatchEvent(new CustomEvent('novel:goto-tab', { detail: { tab: 'chapter' } }))
      }
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '打开失败')
    }
  }

  // ── 删除项目 ──
  const handleDelete = async (card: ProjectCard) => {
    try {
      await deleteProject(card.path)
      message.success(`已删除「${card.title}」`)
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '删除失败')
    }
  }

  // ── 书架过滤 + 排序（纯前端，不依赖后端） ──
  const visibleProjects = useMemo(() => {
    const q = query.trim().toLowerCase()
    let list = projects.filter(Boolean)
    if (q) {
      list = list.filter((card) =>
        card.title.toLowerCase().includes(q) ||
        (card.genre || '').toLowerCase().includes(q) ||
        (card.style || '').toLowerCase().includes(q),
      )
    }
    const sorted = [...list]
    switch (sortKey) {
      case 'words': sorted.sort((a, b) => (b.word_count || 0) - (a.word_count || 0)); break
      case 'chapters': sorted.sort((a, b) => (b.chapter_count || 0) - (a.chapter_count || 0)); break
      case 'title': sorted.sort((a, b) => a.title.localeCompare(b.title, 'zh')); break
      default: {
        const t = (s: ProjectCard) => Date.parse(s.last_opened_at) || 0
        sorted.sort((a, b) => t(b) - t(a))
      }
    }
    return sorted
  }, [projects, query, sortKey])

  const currentCard = useMemo(
    () => projects.find((p) => p.path === projectPath) ?? null,
    [projects, projectPath],
  )
  const currentProgress = projectPath ? readReadingProgress(projectPath) : null

  // --- 未登录：品牌欢迎页 ---
  if (!loggedIn) {
    return <WelcomePage onLogin={login} />
  }

  // --- 书架视图 ---
  return (
    <div className="novel-shelf">
      <div className="novel-shelf-head">
        <div className="novel-shelf-identity">
          <h1 className="novel-shelf-title">书架</h1>
          <span className="novel-shelf-count">
            {loadingProjects ? '…' : `${projects.length} 部`}
          </span>
        </div>
      </div>

      {projectOpen && (
        <section className="novel-now-reading" aria-label="正在编辑">
          <div>
            <span className="novel-now-reading-kicker">正在编辑</span>
            <h2 className="novel-now-reading-title">{projectTitle || '未命名小说'}</h2>
            <p className="novel-now-reading-meta">
              {currentCard
                ? `${currentCard.chapter_count} 章 · ${currentCard.word_count.toLocaleString()} 字`
                : '打开后可继续阅读或去创作'}
              {currentProgress ? ` · 读到第${currentProgress.chapterNum}章` : ''}
            </p>
          </div>
          <div className="novel-now-reading-actions">
            <Button
              type="primary"
              icon={<ReadOutlined aria-hidden />}
              onClick={() => window.dispatchEvent(new CustomEvent('novel:goto-tab', { detail: { tab: 'chapter' } }))}
            >
              继续阅读
            </Button>
            <Button
              onClick={() => window.dispatchEvent(new CustomEvent('novel:goto-tab', { detail: { tab: 'create' } }))}
            >
              去创作
            </Button>
          </div>
        </section>
      )}

      <div className="novel-shelf-toolbar">
        <Input
          allowClear
          prefix={<SearchOutlined style={{ color: 'var(--color-text-secondary)' }} />}
          placeholder="搜索书名 / 题材 / 风格…"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          className="novel-shelf-search"
          aria-label="搜索书架"
        />
        <Select
          value={sortKey}
          onChange={setSortKey}
          options={SORT_OPTIONS}
          size="middle"
          suffixIcon={<SortAscendingOutlined style={{ color: 'var(--color-text-secondary)' }} />}
          popupMatchSelectWidth={false}
          className="novel-shelf-sort"
          aria-label="书架排序"
        />
        <span className="novel-shelf-toolbar-spacer" />
        <Button
          icon={<GlobalOutlined aria-hidden />}
          onClick={() => setBookSearchModal(true)}
          className="novel-shelf-btn"
        >
          在线搜书
        </Button>
        <Button
          icon={<UploadOutlined aria-hidden />}
          onClick={handlePickImport}
          className="novel-shelf-btn"
        >
          导入小说
        </Button>
        <Button
          type="primary"
          icon={<PlusOutlined aria-hidden />}
          onClick={() => { resetForm(); setNewModal(true) }}
          className="novel-shelf-btn is-primary"
        >
          新建小说
        </Button>
      </div>

      {loadingProjects ? (
        <div className="novel-shelf-grid">
          {Array.from({ length: 6 }).map((_, i) => (
            <div key={i} className="novel-shelf-card is-skeleton" aria-hidden>
              <Skeleton active paragraph={{ rows: 3 }} />
            </div>
          ))}
        </div>
      ) : visibleProjects.length === 0 ? (
        query.trim() ? (
          <div className="novel-shelf-empty">
            <SearchOutlined aria-hidden />
            <div className="novel-shelf-empty-title">没有匹配的书</div>
            <div className="novel-shelf-empty-hint">换个关键词试试</div>
          </div>
        ) : projects.length === 0 ? (
          <div className="novel-shelf-empty">
            <ReadOutlined aria-hidden />
            <div className="novel-shelf-empty-title">书架空空如也</div>
            <div className="novel-shelf-empty-hint">Ctrl+N 新建你的第一本小说</div>
            <div className="novel-shelf-empty-actions">
              <Button type="primary" icon={<PlusOutlined aria-hidden />} onClick={() => { resetForm(); setNewModal(true) }}>
                新建小说
              </Button>
              <Button icon={<UploadOutlined aria-hidden />} onClick={handlePickImport}>
                导入成品小说
              </Button>
            </div>
          </div>
        ) : (
          <V3Empty description="没有可显示的小说" className="novel-shelf-antd-empty" />
        )
      ) : (
        <div className="novel-shelf-grid">
          {visibleProjects.map((card) => {
            const progress = readReadingProgress(card.path)
            const chapterCount = card.chapter_count || 0
            return (
              <ProjectCardItem
                key={card.path}
                card={card}
                isActive={projectOpen && card.path === projectPath}
                isMobile={false}
                readingChapter={progress
                  ? `第${progress.chapterNum}章 · ${progress.title}`
                  : undefined}
                readingProgress={progress && chapterCount > 0
                  ? progress.chapterNum / chapterCount
                  : undefined}
                onOpen={(c) => handleOpen(c, false)}
                onContinueReading={(c) => handleOpen(c, true)}
                onDelete={handleDelete}
              />
            )
          })}
        </div>
      )}

      {/* Create Novel Modal */}
      <CreateNovelModal
        open={newModal}
        onClose={() => { setNewModal(false); resetForm() }}
        onCreate={handleCreate}
        title={newTitle} onTitleChange={setNewTitle}
        genre={newGenre} onGenreChange={setNewGenre}
        style={newStyle} onStyleChange={setNewStyle}
      />
      <ImportNovelModal
        open={importModal}
        fileName={importFile?.name || ''}
        title={importTitle}
        genre={importGenre}
        style={importStyle}
        importing={importing}
        extractRange={importExtractRange}
        onExtractRangeChange={setImportExtractRange}
        onTitleChange={setImportTitle}
        onGenreChange={setImportGenre}
        onStyleChange={setImportStyle}
        onImport={() => void handleImport()}
        onClose={() => { if (!importing) setImportModal(false) }}
      />
      <BookSearchModal
        open={bookSearchModal}
        onClose={() => setBookSearchModal(false)}
        onImported={(res) => void handleOnlineImported(res)}
        onAppended={(res) => void handleChaptersAppended(res)}
      />
    </div>
  )
}

export default HomePage
