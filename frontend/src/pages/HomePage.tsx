import React, { useState, useEffect, useMemo } from 'react'
import {
  Button, Skeleton, message, Input, Select, Empty,
} from 'antd'
import {
  PlusOutlined, SearchOutlined, ReadOutlined, SortAscendingOutlined, UploadOutlined,
} from '@ant-design/icons'
import { useAppStore, type ProjectCard } from '../stores/appStore'

import WelcomePage from '../components/WelcomePage'
import CreateNovelModal from '../components/novel/CreateNovelModal'
import ImportNovelModal from '../components/novel/ImportNovelModal'
import ProjectCardItem from '../components/ProjectCardItem'
import { readReadingProgress } from '../utils/readingProgress'

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

      await window.go.app.App.CreateProject(dir, newTitle, genreStr, styleStr)

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
      const picked = await window.go.app.App.GaeaPickFiles()
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
      const res = await window.go.app.App.ImportNovelBook(
        importFile.path,
        importTitle.trim(),
        importGenre.join('、') || '未分类',
        importStyle.join('、') || '默认',
      )
      openProject(res.path, res.title)
      await loadProjects()
      setImportModal(false)
      message.success(`已导入「${res.title}」：${res.chapter_count} 章，${res.total_words.toLocaleString()} 字`)
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '导入失败')
    } finally {
      setImporting(false)
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
      await window.go.app.App.OpenProject(card.path)
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

  // --- 未登录：品牌欢迎页 ---
  if (!loggedIn) {
    return <WelcomePage onLogin={login} />
  }

  // --- 书架视图 ---
  return (
    <div style={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column' }}>
      {/* 工具条：搜索 / 排序 / 新建 */}
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
          style={{ width: 128 }}
          aria-label="书架排序"
        />
        {projectOpen && (
          <span className="novel-tag-tone is-success" style={{ height: 24 }}>
            正在编辑：{projectTitle}
          </span>
        )}
        <span style={{ flex: 1 }} />
        <Button
          icon={<UploadOutlined />}
          onClick={handlePickImport}
          style={{
            color: 'var(--color-primary)',
            borderColor: 'var(--color-primary)',
            borderRadius: 'var(--radius-md)',
          }}
        >
          导入小说
        </Button>
        <Button
          type="primary" icon={<PlusOutlined />}
          onClick={() => { resetForm(); setNewModal(true) }}
          style={{
            background: 'var(--color-primary)', borderColor: 'var(--color-primary)',
            boxShadow: 'var(--v3-glow-faint)', borderRadius: 'var(--radius-md)',
          }}
        >
          新建小说
        </Button>
      </div>

      {/* 书架主区域 */}
      {loadingProjects ? (
        <div className="novel-shelf-grid">
          {Array.from({ length: 6 }).map((_, i) => (
            <div key={i} className="novel-shelf-card" style={{ pointerEvents: 'none' }}>
              <Skeleton active paragraph={{ rows: 3 }} style={{ padding: 16 }} />
            </div>
          ))}
        </div>
      ) : visibleProjects.length === 0 ? (
        query.trim() ? (
          /* 搜索无结果 */
          <div className="novel-shelf-empty">
            <SearchOutlined aria-hidden />
            <div className="novel-shelf-empty-title">没有匹配的书</div>
            <div className="novel-shelf-empty-hint">换个关键词试试</div>
          </div>
        ) : projects.length === 0 ? (
          /* 空书架 */
          <div className="novel-shelf-empty">
            <ReadOutlined aria-hidden />
            <div className="novel-shelf-empty-title">书架空空如也</div>
            <div className="novel-shelf-empty-hint">Ctrl+N 新建你的第一本小说</div>
            <Button type="primary" icon={<PlusOutlined />} onClick={() => { resetForm(); setNewModal(true) }}>
              新建小说
            </Button>
            <Button icon={<UploadOutlined />} onClick={handlePickImport}>
              导入成品小说
            </Button>
          </div>
        ) : (
          <Empty description="没有可显示的小说" style={{ marginTop: 80 }} />
        )
      ) : (
        /* 书架栅格 */
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
        onTitleChange={setImportTitle}
        onGenreChange={setImportGenre}
        onStyleChange={setImportStyle}
        onImport={() => void handleImport()}
        onClose={() => { if (!importing) setImportModal(false) }}
      />
    </div>
  )
}

export default HomePage
