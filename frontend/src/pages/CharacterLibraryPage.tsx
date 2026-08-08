// CharacterLibraryPage.tsx — 全局统一角色库（档案墙）
// 角色是独立资产：不绑定任何一本小说，小说只是引用；聊天直接把角色当人格用。
import React, { useCallback, useEffect, useState } from 'react'
import {
  Input, Segmented, Switch, Button, message, Pagination,
} from 'antd'
import {
  PlusOutlined, SearchOutlined, TeamOutlined, ImportOutlined, SyncOutlined,
} from '@ant-design/icons'
import CharacterCard from '../components/characterlib/CharacterCard'
import CharacterLibEditor from '../components/characterlib/CharacterLibEditor'
import CharacterMemoryModal from '../components/characterlib/CharacterMemoryModal'
import FeatureModelBar from '../components/FeatureModelBar'
import { C } from '../utils/theme'
import { useAppStore } from '../stores/appStore'
import {
  listCharacters, getCharacter, deleteCharacter, importProjectCharacters,
  listProjectCharacters, associateToProject, dissociateFromProject, syncProjectCharacters,
  type LibraryCharacter,
} from '../api/characterlib'
import '../components/characterlib/character-library.css'

const PAGE_SIZE = 24
const PERSONALITY_KEY = 'gaea_whisper_personality'

/** 设为当前聊天人格：本地持久化 + 广播（聊天板块联动） */
function setCurrentPersona(id: string) {
  try { localStorage.setItem(PERSONALITY_KEY, id) } catch (_) {}
  window.dispatchEvent(new CustomEvent('gaea-persona-changed', { detail: { id } }))
}

const CharacterLibraryPage: React.FC = () => {
  const projectPath = useAppStore(s => s.projectPath)
  const [items, setItems] = useState<LibraryCharacter[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [query, setQuery] = useState('')
  const [kind, setKind] = useState('')
  const [chatOnly, setChatOnly] = useState(false)
  const [loading, setLoading] = useState(false)

  const [projectRefs, setProjectRefs] = useState<Set<string>>(new Set())
  const [editorOpen, setEditorOpen] = useState(false)
  const [editing, setEditing] = useState<LibraryCharacter | null>(null)
  const [editingProjects, setEditingProjects] = useState<string[]>([])
  const [editingIndex, setEditingIndex] = useState(0)
  const [memoryChar, setMemoryChar] = useState<LibraryCharacter | null>(null)
  const [currentPersona, setCurrentPersonaState] = useState<string>(() => {
    try { return localStorage.getItem(PERSONALITY_KEY) || '' } catch { return '' }
  })

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const res = await listCharacters(query.trim(), kind, chatOnly, page, PAGE_SIZE)
      setItems(res.items || [])
      setTotal(res.total || 0)
      if (res.error) message.warning(res.error)
    } catch (err: any) {
      message.error(`加载角色库失败：${err?.message || String(err)}`)
    } finally {
      setLoading(false)
    }
  }, [query, kind, chatOnly, page])

  const loadProjectRefs = useCallback(async () => {
    if (!projectPath) { setProjectRefs(new Set()); return }
    try {
      const refs = await listProjectCharacters()
      setProjectRefs(new Set(refs.map(r => r.characterId)))
    } catch (_) { setProjectRefs(new Set()) }
  }, [projectPath])

  useEffect(() => { load() }, [load])
  useEffect(() => { loadProjectRefs() }, [loadProjectRefs])
  useEffect(() => {
    const onPersona = (e: Event) => {
      setCurrentPersonaState((e as CustomEvent<{ id: string }>).detail?.id || '')
    }
    window.addEventListener('gaea-persona-changed', onPersona)
    return () => window.removeEventListener('gaea-persona-changed', onPersona)
  }, [])

  const openNew = () => {
    setEditing(null)
    setEditingProjects([])
    setEditingIndex(0)
    setEditorOpen(true)
  }

  const openEdit = async (c: LibraryCharacter) => {
    try {
      const idx = items.findIndex(i => i.id === c.id)
      setEditingIndex(idx >= 0 ? idx : 0)
      const detail = await getCharacter(c.id)
      setEditing(detail.character)
      setEditingProjects(detail.projects || [])
      setEditorOpen(true)
    } catch (err: any) {
      message.error(`读取角色失败：${err?.message || String(err)}`)
    }
  }

  const handleSaved = (c: LibraryCharacter) => {
    load()
    loadProjectRefs()
    if (c.chatEnabled) {
      message.info(`「${c.name}」已可聊天，可在聊天板块选择`)
    }
  }

  const handleSetPersona = (c: LibraryCharacter) => {
    setCurrentPersona(c.id)
    message.success(`已将「${c.name}」设为当前聊天人格`)
  }

  const handleAssociate = async (c: LibraryCharacter) => {
    try {
      await associateToProject(c.id, c.roleType || 'supporting')
      message.success(`「${c.name}」已加入当前项目`)
      loadProjectRefs()
    } catch (err: any) {
      message.error(err?.message || '加入项目失败')
    }
  }

  const handleDissociate = async (c: LibraryCharacter) => {
    try {
      await dissociateFromProject(c.id)
      message.success(`「${c.name}」已从当前项目移除（角色保留在库中）`)
      loadProjectRefs()
    } catch (err: any) {
      message.error(err?.message || '移除失败')
    }
  }

  const handleDelete = async (c: LibraryCharacter) => {
    try {
      await deleteCharacter(c.id)
      message.success(c.kind === 'builtin' ? `「${c.name}」已隐藏` : `「${c.name}」已删除`)
      load()
      loadProjectRefs()
    } catch (err: any) {
      message.error(err?.message || '删除失败')
    }
  }

  const handleImportProject = async () => {
    if (!projectPath) { message.warning('请先打开小说项目'); return }
    try {
      const n = await importProjectCharacters()
      message.success(`已从项目导入 ${n} 个角色到全局库`)
      load()
      loadProjectRefs()
    } catch (err: any) {
      message.error(err?.message || '导入失败')
    }
  }

  const handleSyncProject = async () => {
    if (!projectPath) { message.warning('请先打开小说项目'); return }
    try {
      await syncProjectCharacters()
      message.success('已把项目引用物化到 characters.json（小说 Agent 生效）')
    } catch (err: any) {
      message.error(err?.message || '同步失败')
    }
  }

  const hasProject = !!projectPath

  return (
    <div className="clib-page">
      {/* ── 页头：标题 + 统计 + 操作 ── */}
      <div className="clib-header">
        <div className="clib-title-wrap">
          <div className="clib-icon"><TeamOutlined /></div>
          <div className="clib-title-col">
            <h2 className="clib-title">角色库</h2>
            <div className="clib-sub">全局资产 · 小说引用 · 聊天直用 · 共 {total} 位</div>
          </div>
        </div>
        <div className="clib-actions">
          <Button size="small" icon={<ImportOutlined />} disabled={!hasProject}
            onClick={handleImportProject}
            title="把当前项目的 characters.json 导入全局库">导入项目</Button>
          <Button size="small" icon={<SyncOutlined />} disabled={!hasProject}
            onClick={handleSyncProject}
            title="把项目引用写回 characters.json，小说 Agent 生效">同步到项目</Button>
          <Button size="small" type="primary" icon={<PlusOutlined />} onClick={openNew}
            className="clib-new-btn">新建角色</Button>
        </div>
      </div>

      {/* ── 工具栏：搜索 + 类型 + 可聊天 ── */}
      <div className="clib-toolbar">
        <Input size="middle" allowClear prefix={<SearchOutlined style={{ color: C('color-text-secondary') }} />}
          placeholder="搜索名称 / 标签 / 性格" value={query}
          onChange={e => { setQuery(e.target.value); setPage(1) }}
          className="clib-search" />
        <Segmented
          size="small"
          value={kind}
          onChange={v => { setKind(v as string); setPage(1) }}
          options={[
            { value: '', label: '全部类型' },
            { value: 'builtin', label: '内置' },
            { value: 'custom', label: '自定义' },
            { value: 'assistant', label: '助手' },
          ]}
        />
        <span className="clib-chat-only">
          <Switch size="small" checked={chatOnly}
            onChange={v => { setChatOnly(v); setPage(1) }} />
          仅可聊天
        </span>
      </div>

      {!hasProject && (
        <div className="clib-hint">
          当前未打开小说项目：角色仍可全局管理、设为聊天人格；打开项目后可将角色加入小说。
        </div>
      )}

      {/* ── 档案墙 ── */}
      <div className="clib-wall">
        {loading ? (
          <div className="clib-grid">
            {Array.from({ length: 12 }).map((_, i) => (
              <div key={i} className="clib-skel">
                <div className="clib-skel-portrait" />
                <div className="clib-skel-body">
                  <div className="clib-skel-line" />
                  <div className="clib-skel-line short" />
                </div>
              </div>
            ))}
          </div>
        ) : items.length === 0 ? (
          <div className="clib-empty">
            <div className="clib-empty-icon"><TeamOutlined /></div>
            <h3 className="clib-empty-title">档案库还是空的</h3>
            <p className="clib-empty-desc">创建第一个角色档案，或从当前小说项目导入已有角色。</p>
            <div className="clib-empty-actions">
              <Button type="primary" icon={<PlusOutlined />} onClick={openNew}>新建第一个角色</Button>
              {hasProject && (
                <Button icon={<ImportOutlined />} onClick={handleImportProject}>从当前项目导入</Button>
              )}
            </div>
          </div>
        ) : (
          <div className="clib-grid">
            {items.map((c, idx) => {
              const inProject = projectRefs.has(c.id)
              return (
                <CharacterCard
                  key={c.id}
                  character={c}
                  index={idx}
                  inProject={inProject}
                  isCurrentPersona={currentPersona === c.id}
                  hasProject={hasProject}
                  onClick={openEdit}
                  onEdit={openEdit}
                  onSetPersona={handleSetPersona}
                  onMemory={setMemoryChar}
                  onAssociate={handleAssociate}
                  onDissociate={handleDissociate}
                  onDelete={handleDelete}
                />
              )
            })}
          </div>
        )}
      </div>

      {total > PAGE_SIZE && (
        <div className="clib-pager">
          <Pagination size="small" current={page} pageSize={PAGE_SIZE} total={total}
            onChange={setPage} showSizeChanger={false} />
        </div>
      )}

      <CharacterLibEditor
        open={editorOpen}
        character={editing}
        projects={editingProjects}
        index={editingIndex}
        isCurrentPersona={currentPersona === editing?.id}
        onClose={() => setEditorOpen(false)}
        onSaved={handleSaved}
      />
      <CharacterMemoryModal
        open={!!memoryChar}
        character={memoryChar}
        onClose={() => setMemoryChar(null)}
      />

      {/* 绑定模型卡（左下角浮动）：角色库 AI 生成/补全走独立绑定 */}
      <div style={{ position: 'absolute', left: 12, bottom: 12, zIndex: 50 }}>
        <FeatureModelBar feature="characterlib" label="角色库" />
      </div>
    </div>
  )
}

export default CharacterLibraryPage
