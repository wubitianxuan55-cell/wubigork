// CharacterLibraryPage.tsx — 全局统一角色库（角色档案库 · 3 分区工作台）
// 角色是独立资产：不绑定任何一本小说，小说只是引用；聊天直接把角色当人格用。
// 布局：细条头部（统计 + 新建主操作）→ 左检索/筛选栏 + 中档案卡网格 + 右详情 inspector。
import React, { useCallback, useEffect, useState } from 'react'
import {
  Button, message, Pagination, Modal, Spin,
} from 'antd'
import {
  PlusOutlined, TeamOutlined, ImportOutlined,
} from '@ant-design/icons'
import CharacterCard from '../components/characterlib/CharacterCard'
import CharacterLibEditor from '../components/characterlib/CharacterLibEditor'
import CharacterLibInspector from '../components/characterlib/CharacterLibInspector'
import CharacterLibFilterBar from '../components/characterlib/CharacterLibFilterBar'
import CharacterMemoryModal from '../components/characterlib/CharacterMemoryModal'
import FeatureModelBar from '../components/FeatureModelBar'
import { useAppStore } from '../stores/appStore'
import {
  listCharacters, getCharacter, deleteCharacter, importProjectCharacters,
  listProjectCharacters, associateToProject, dissociateFromProject, syncProjectCharacters,
  fillAllCharacters, type LibraryCharacter,
} from '../api/characterlib'
import '../components/characterlib/character-library.css'

/** 提取错误消息（unknown 收窄；无 message 用 fallback） */
function errText(err: unknown, fallback: string): string {
  return (err instanceof Error && err.message) || fallback
}

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
  const [fillingAll, setFillingAll] = useState(false)
  const [fillProgress, setFillProgress] = useState('')

  const [projectRefs, setProjectRefs] = useState<Set<string>>(new Set())
  const [editorOpen, setEditorOpen] = useState(false)
  const [editing, setEditing] = useState<LibraryCharacter | null>(null)
  const [editingProjects, setEditingProjects] = useState<string[]>([])
  const [editingIndex, setEditingIndex] = useState(0)
  const [memoryChar, setMemoryChar] = useState<LibraryCharacter | null>(null)

  // 右侧详情 inspector 选中项 + 折叠状态
  const [selected, setSelected] = useState<LibraryCharacter | null>(null)
  const [selectedIndex, setSelectedIndex] = useState(0)
  const [inspectorOpen, setInspectorOpen] = useState(true)

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
    } catch (err: unknown) {
      message.error(`加载角色库失败：${errText(err, String(err))}`)
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

  /** 点击档案卡 → 右侧 inspector 预览（编辑走 inspector/卡片上的「编辑」） */
  const selectForInspector = (c: LibraryCharacter) => {
    const idx = items.findIndex(i => i.id === c.id)
    setSelected(c)
    setSelectedIndex(idx)
    setInspectorOpen(true)
  }

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
    } catch (err: unknown) {
      message.error(`读取角色失败：${errText(err, String(err))}`)
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
    } catch (err: unknown) {
      message.error(errText(err, '加入项目失败'))
    }
  }

  const handleDissociate = async (c: LibraryCharacter) => {
    try {
      await dissociateFromProject(c.id)
      message.success(`「${c.name}」已从当前项目移除（角色保留在库中）`)
      loadProjectRefs()
      // 通知已挂载的小说角色面板刷新（MainLayout 切换页面不销毁组件）
      window.dispatchEvent(new CustomEvent('gaea-project-chars-changed'))
    } catch (err: unknown) {
      message.error(errText(err, '移除失败'))
    }
  }

  const handleDelete = async (c: LibraryCharacter) => {
    try {
      await deleteCharacter(c.id)
      message.success(c.kind === 'builtin' ? `「${c.name}」已隐藏` : `「${c.name}」已删除`)
      if (selected?.id === c.id) { setSelected(null); setSelectedIndex(0) }
      load()
      loadProjectRefs()
    } catch (err: unknown) {
      message.error(errText(err, '删除失败'))
    }
  }

  const handleImportProject = async () => {
    if (!projectPath) { message.warning('请先打开小说项目'); return }
    try {
      const n = await importProjectCharacters()
      message.success(`已从项目导入 ${n} 个角色到全局库`)
      load()
      loadProjectRefs()
    } catch (err: unknown) {
      message.error(errText(err, '导入失败'))
    }
  }

  // ── 一键补齐：为全部角色填充空缺字段（基于已有设定推断） ──
  const handleFillAll = () => {
    if (!total) return
    Modal.confirm({
      title: '一键补齐全部角色信息？',
      content: `将为 ${total} 位角色补齐空缺字段（定位/性别/年龄/性格/外貌/背景/动机/弧线等），
基于已有设定（如人格）推断，只填空缺、不改已有内容。角色较多时耗时较长。`,
      okText: '开始补齐',
      cancelText: '取消',
      onOk: () => { void runFillAll() },
    })
  }

  const runFillAll = async () => {
    const onProgress = (ev: unknown) => {
      const raw = ev as { detail?: unknown } | null | undefined
      const d = (raw && typeof raw === 'object' && 'detail' in raw && raw.detail ? raw.detail : raw) as { current?: number; total?: number; name?: string } | null | undefined
      if (d && d.current && d.total) setFillProgress(`正在补齐 ${d.current}/${d.total}：${d.name || ''}`)
    }
    try {
      window.runtime?.EventsOn?.('character-fill-progress', onProgress)
      setFillingAll(true)
      setFillProgress('准备中…')
      const res = await fillAllCharacters()
      const { filled, skipped, failed, failNames } = res || {}
      if (failed > 0) {
        message.warning(
          `补齐完成：填充 ${filled} 位，无空缺跳过 ${skipped} 位，失败 ${failed} 位` +
          (failNames?.length ? `（${failNames.slice(0, 3).join('、')}${failNames.length > 3 ? '…' : ''}）` : ''),
        )
      } else {
        message.success(`已补齐 ${filled} 位角色，${skipped} 位无空缺`)
      }
      load()
      loadProjectRefs()
    } catch (err: unknown) {
      message.error(`补齐失败：${errText(err, String(err))}`)
    } finally {
      window.runtime?.EventsOff?.('character-fill-progress')
      setFillingAll(false)
      setFillProgress('')
    }
  }

  const handleSyncProject = async () => {
    if (!projectPath) { message.warning('请先打开小说项目'); return }
    try {
      await syncProjectCharacters()
      message.success('已把项目引用物化到 characters.json（小说 Agent 生效）')
    } catch (err: unknown) {
      message.error(errText(err, '同步失败'))
    }
  }

  const hasProject = !!projectPath
  const inProjectOf = (id: string) => projectRefs.has(id)

  return (
    <div className="clib-page">
      {/* ── 细条头部：统计 + 主操作（板块名已在左侧轨道） ── */}
      <div className="clib-header-bar">
        <div className="clib-count">
          <TeamOutlined aria-hidden />
          角色档案库
          <span className="clib-count-num">{total}</span>
          <span className="clib-count-unit">位</span>
        </div>
        <div className="clib-actions">
          <Button type="primary" icon={<PlusOutlined />} onClick={openNew}
            className="clib-new-btn">新建角色</Button>
        </div>
      </div>

      {/* ── 3 分区工作台 ── */}
      <div className="clib-workbench">
        {/* 左：检索 / 筛选栏 */}
        <CharacterLibFilterBar
          query={query}
          kind={kind}
          chatOnly={chatOnly}
          total={total}
          hasProject={hasProject}
          fillingAll={fillingAll}
          onQueryChange={q => { setQuery(q); setPage(1) }}
          onKindChange={k => { setKind(k); setPage(1) }}
          onChatOnlyChange={v => { setChatOnly(v); setPage(1) }}
          onFillAll={handleFillAll}
          onImportProject={handleImportProject}
          onSyncProject={handleSyncProject}
        />

        {/* 中：档案卡网格 */}
        <main className="clib-main" aria-label="角色档案网格">
          {!hasProject && (
            <div className="clib-hint">
              当前未打开小说项目：角色仍可全局管理、设为聊天人格；打开项目后可将角色加入小说。
            </div>
          )}

          {fillingAll && fillProgress && (
            <div className="clib-hint">
              <Spin size="small" /> {fillProgress}（AI 逐个补齐中，可继续浏览其他页面）
            </div>
          )}

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
                    <Button icon={<ImportOutlined className="clib-empty-import" />} onClick={handleImportProject}>从当前项目导入</Button>
                  )}
                </div>
              </div>
            ) : (
              <div className="clib-grid">
                {items.map((c, idx) => (
                  <CharacterCard
                    key={c.id}
                    character={c}
                    index={idx}
                    inProject={inProjectOf(c.id)}
                    isCurrentPersona={currentPersona === c.id}
                    hasProject={hasProject}
                    onClick={selectForInspector}
                    onEdit={openEdit}
                    onSetPersona={handleSetPersona}
                    onMemory={setMemoryChar}
                    onAssociate={handleAssociate}
                    onDissociate={handleDissociate}
                    onDelete={handleDelete}
                  />
                ))}
              </div>
            )}
          </div>

          {total > PAGE_SIZE && (
            <div className="clib-pager">
              <Pagination size="small" current={page} pageSize={PAGE_SIZE} total={total}
                onChange={setPage} showSizeChanger={false} />
            </div>
          )}
        </main>

        {/* 右：详情 inspector（可折叠） */}
        <CharacterLibInspector
          character={selected}
          index={selectedIndex}
          inProject={selected ? inProjectOf(selected.id) : false}
          isCurrentPersona={selected ? currentPersona === selected.id : false}
          hasProject={hasProject}
          collapsed={!inspectorOpen}
          onToggleCollapse={() => setInspectorOpen(v => !v)}
          onEdit={openEdit}
          onSetPersona={handleSetPersona}
          onMemory={setMemoryChar}
          onAssociate={handleAssociate}
          onDissociate={handleDissociate}
          onDelete={handleDelete}
        />
      </div>

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

      {/* 模型状态统一由顶栏轨道条展示（3.0 定制：移除左下角悬浮模型卡） */}
    </div>
  )
}

export default CharacterLibraryPage
