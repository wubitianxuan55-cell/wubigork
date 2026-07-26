import React, { useState, useEffect } from 'react'
import {
  Typography, Button, Space, Tag, Skeleton, message,
} from 'antd'
import { PlusOutlined } from '@ant-design/icons'
import { useAppStore, type ProjectCard } from '../stores/appStore'
import { C } from '../utils/theme'

import WelcomePage from '../components/WelcomePage'
import CreateNovelModal from '../components/CreateNovelModal'
import ProjectCardItem from '../components/ProjectCardItem'

const HomePage: React.FC = () => {
  const {
    loggedIn, login, projectOpen, projectPath, projectTitle, novelsDir,
    projects, loadProjects, openProject, closeProject, deleteProject, loadNovelsDir,
  } = useAppStore()


  // 新建小说表单
  const [newModal, setNewModal] = useState(false)
  const [newTitle, setNewTitle] = useState('')
  const [newGenre, setNewGenre] = useState<string[]>([])
  const [newStyle, setNewStyle] = useState<string[]>([])

  // 加载状态
  const [loadingProjects, setLoadingProjects] = useState(true)

  useEffect(() => {
    if (loggedIn) {
      loadNovelsDir()
      loadProjects().finally(() => setLoadingProjects(false))
    } else {
      setLoadingProjects(false)
    }
  }, [loggedIn])

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

      // @ts-ignore
      await window.go.app.App.CreateProject(dir, newTitle, genreStr, styleStr)

      openProject(dir, newTitle)
      await loadProjects()
      setNewModal(false)
      resetForm()
    } catch (err: any) {
      message.error(err.message || '创建失败')
    }
  }

  // ── 打开/关闭项目 ──
  const handleOpen = async (card: ProjectCard) => {
    if (projectOpen && card.path === projectPath) {
      try {
        // @ts-ignore
        await window.go.app.App.CloseProject()
        closeProject()
      } catch (err) { console.error('[HomePage] CloseProject:', err) }
      return
    }
    try {
      // @ts-ignore
      await window.go.app.App.OpenProject(card.path)
      openProject(card.path, card.title)
    } catch (err: any) {
      message.error(err.message || '打开失败')
    }
  }

  // ── 删除项目 ──
  const handleDelete = async (card: ProjectCard) => {
    try {
      await deleteProject(card.path)
      message.success(`已删除「${card.title}」`)
    } catch (err: any) {
      message.error(err.message || '删除失败')
    }
  }

  // --- 未登录：品牌欢迎页 ---
  if (!loggedIn) {
    return <WelcomePage onLogin={login} />
  }

  // --- 书架视图 ---
  return (
    <div>
      {/* 顶部栏 */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
        <Space direction="vertical" size={0}>
          <Typography.Title level={3} style={{ color: C('color-text'), margin: 0, fontWeight: 600 }}>
            我的书架
          </Typography.Title>
          {projectOpen && (
            <Tag color="green" style={{ fontSize: 11, marginTop: 4 }}>
              正在编辑：{projectTitle}
            </Tag>
          )}
        </Space>
        <Button
          type="primary" icon={<PlusOutlined />}
          onClick={() => { resetForm(); setNewModal(true) }}
          style={{
            background: 'var(--color-primary)', borderColor: 'var(--color-primary)',
            boxShadow: 'var(--shadow-glow)', borderRadius: 'var(--radius-md)',
          }}
        >
          新建小说
        </Button>
      </div>

      {/* 书架主区域 */}
      {loadingProjects ? (
        <div style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(4, 1fr)',
          gap: 16,
        }}>
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} style={{ padding: 20 }}>
              <Skeleton active paragraph={{ rows: 3 }} />
            </div>
          ))}
        </div>
      ) : projects.length === 0 ? (
        /* 空书架 */
        <div style={{
          display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center',
          height: '60vh', opacity: 0.3, userSelect: 'none', pointerEvents: 'none',
        }}>
          <img src="/favicon.svg" alt="" style={{ width: 80, height: 80, marginBottom: 16 }} />
          <Typography.Text style={{ color: 'var(--color-text-secondary)', fontSize: 24, fontWeight: 200, letterSpacing: '0.1em' }}>
            wubigork
          </Typography.Text>
          <Typography.Text style={{ color: 'var(--color-text-secondary)', fontSize: 12, marginTop: 8 }}>
            Ctrl+N 新建小说
          </Typography.Text>
        </div>
      ) : (
        /* Bento Grid 书架 */
        <div style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(4, 1fr)',
          gridAutoRows: 'minmax(160px, auto)',
          gap: 16,
        }}>
          {projects.filter(Boolean).map((card, idx) => (
            <ProjectCardItem
              key={card.path}
              card={card}
              isActive={projectOpen && card.path === projectPath}
              isHero={idx === 0 && projects.length > 1}
              isMobile={false}
              onOpen={handleOpen}
              onDelete={handleDelete}
            />
          ))}
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
    </div>
  )
}

export default HomePage
