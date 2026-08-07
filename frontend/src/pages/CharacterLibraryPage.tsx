// CharacterLibraryPage.tsx — 全局统一角色库
// 角色是独立资产：不绑定任何一本小说，小说只是引用；聊天直接把角色当人格用。
import React, { useCallback, useEffect, useMemo, useState } from 'react'
import {
  Typography, Input, Select, Switch, Button, Tag, message, Row, Col, Pagination, Popconfirm, Empty, Space,
} from 'antd'
import {
  PlusOutlined, SearchOutlined, TeamOutlined, EditOutlined, DeleteOutlined,
  ImportOutlined, SyncOutlined, SwapOutlined, UserOutlined, ReadOutlined, DatabaseOutlined,
} from '@ant-design/icons'
import TisorRadar from '../components/TisorRadar'
import CharacterLibEditor from '../components/characterlib/CharacterLibEditor'
import CharacterMemoryModal from '../components/characterlib/CharacterMemoryModal'
import { C } from '../utils/theme'
import { useAppStore } from '../stores/appStore'
import {
  listCharacters, getCharacter, deleteCharacter, importProjectCharacters,
  listProjectCharacters, associateToProject, dissociateFromProject, syncProjectCharacters,
  type LibraryCharacter,
} from '../api/characterlib'

const PAGE_SIZE = 24
const PERSONALITY_KEY = 'gaea_whisper_personality'

const KIND_META: Record<string, { label: string; color: string }> = {
  builtin: { label: '内置', color: 'gold' },
  custom: { label: '自定义', color: 'green' },
  assistant: { label: '助手', color: 'geekblue' },
}

const ROLE_LABELS: Record<string, string> = {
  protagonist: '主角', antagonist: '反派', supporting: '配角', minor: '龙套',
}

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
  const [memoryChar, setMemoryChar] = useState<LibraryCharacter | null>(null)

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

  const openNew = () => {
    setEditing(null)
    setEditingProjects([])
    setEditorOpen(true)
  }

  const openEdit = async (c: LibraryCharacter) => {
    try {
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
    <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minHeight: 0, position: 'relative' }}>
      {/* ── 头部：搜索 + 筛选 + 操作 ── */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 8, flexShrink: 0, flexWrap: 'wrap' }}>
        <Typography.Title level={4} style={{ margin: 0, color: C('color-text'), fontSize: 17 }}>
          <TeamOutlined style={{ marginRight: 8, color: 'var(--gaea-glow)' }} />角色库
        </Typography.Title>
        <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 12 }}>全局资产 · 小说引用 · 聊天直用（{total}）</Typography.Text>
        <div style={{ flex: 1 }} />
        <Input size="small" allowClear prefix={<SearchOutlined style={{ color: C('color-text-secondary') }} />}
          placeholder="搜索名称 / 标签 / 性格" value={query}
          onChange={e => { setQuery(e.target.value); setPage(1) }}
          style={{ width: 190 }} />
        <Select size="small" value={kind} onChange={v => { setKind(v); setPage(1) }} style={{ width: 100 }}
          options={[
            { value: '', label: '全部类型' },
            { value: 'builtin', label: '内置' },
            { value: 'custom', label: '自定义' },
            { value: 'assistant', label: '助手' },
          ]} />
        <span style={{ display: 'inline-flex', alignItems: 'center', gap: 5, fontSize: 12, color: C('color-text-secondary') }}>
          <Switch size="small" checked={chatOnly} onChange={v => { setChatOnly(v); setPage(1) }} />
          可聊天
        </span>
        <Button size="small" icon={<ImportOutlined />} disabled={!hasProject} onClick={handleImportProject}
          title="把当前项目的 characters.json 导入全局库">导入项目</Button>
        <Button size="small" icon={<SyncOutlined />} disabled={!hasProject} onClick={handleSyncProject}
          title="把项目引用写回 characters.json，小说 Agent 生效">同步到项目</Button>
        <Button size="small" type="primary" icon={<PlusOutlined />} onClick={openNew}>新建角色</Button>
      </div>

      {!hasProject && (
        <div style={{ fontSize: 11.5, color: C('color-text-secondary'), marginBottom: 8, flexShrink: 0 }}>
          当前未打开小说项目：角色仍可全局管理、设为聊天人格；打开项目后可把角色加入小说。
        </div>
      )}

      {/* ── 统一角色列表 ── */}
      <div style={{ flex: 1, overflowY: 'auto', minHeight: 0, paddingRight: 4 }}>
        {items.length === 0 && !loading ? (
          <Empty description="还没有角色" style={{ marginTop: 90 }}>
            <Button type="primary" icon={<PlusOutlined />} onClick={openNew}>新建第一个角色</Button>
            {hasProject && (
              <Button icon={<ImportOutlined />} style={{ marginLeft: 8 }} onClick={handleImportProject}>从当前项目导入</Button>
            )}
          </Empty>
        ) : (
          <Row gutter={[12, 12]}>
            {items.map(c => {
              const km = KIND_META[c.kind] || KIND_META.custom
              const inProject = projectRefs.has(c.id)
              return (
                <Col key={c.id} xs={24} sm={12} lg={8} xl={6}>
                  <div
                    style={{
                      padding: '12px 12px 10px', borderRadius: 14, height: '100%',
                      border: '1px solid var(--md-sys-color-outline-variant)',
                      background: 'var(--gaea-glass-bg, var(--md-sys-color-surface-container))',
                      display: 'flex', flexDirection: 'column', gap: 6,
                      transition: 'transform 180ms, border-color 180ms',
                    }}
                    onMouseEnter={e => { e.currentTarget.style.transform = 'translateY(-2px)'; e.currentTarget.style.borderColor = '#f472b6' }}
                    onMouseLeave={e => { e.currentTarget.style.transform = 'translateY(0)'; e.currentTarget.style.borderColor = 'var(--md-sys-color-outline-variant)' }}
                  >
                    <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                      {c.portraitUrl
                        ? <img src={c.portraitUrl} alt={c.name} style={{ width: 48, height: 48, borderRadius: 12, objectFit: 'cover', flexShrink: 0 }} />
                        : <div style={{ width: 48, height: 48, borderRadius: 12, background: 'rgba(244,114,182,0.12)', display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
                            <UserOutlined style={{ fontSize: 18, color: '#f472b6' }} />
                          </div>}
                      <div style={{ flex: 1, minWidth: 0 }}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                          <Typography.Text strong style={{ color: C('color-text'), fontSize: 13.5, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{c.name}</Typography.Text>
                          <Tag color={km.color} style={{ fontSize: 9, margin: 0, lineHeight: '14px' }}>{km.label}</Tag>
                          {c.chatEnabled && <Tag color="pink" style={{ fontSize: 9, margin: 0, lineHeight: '14px' }}>可聊天</Tag>}
                        </div>
                        <div style={{ fontSize: 10.5, color: C('color-text-secondary'), whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                          {c.roleType ? `${ROLE_LABELS[c.roleType] || c.roleType}` : '未设定位'}
                          {c.status ? ` · ${c.status}` : ''}
                          {c.gender ? ` · ${c.gender === 'female' ? '女' : c.gender === 'male' ? '男' : c.gender}` : ''}
                        </div>
                      </div>
                      {c.chatEnabled && <TisorRadar dims={c.dims} size={46} color="#f472b6" showLabels={false} />}
                    </div>

                    {(c.tags?.length > 0 || c.arc) && (
                      <div style={{ fontSize: 10.5, color: C('color-text-secondary'), lineHeight: 1.5, minHeight: 16 }}>
                        {c.tags?.slice(0, 3).map(t => <Tag key={t} style={{ fontSize: 9, margin: '0 4px 2px 0', lineHeight: '14px' }}>{t}</Tag>)}
                        {c.arc && <span style={{ opacity: 0.75 }}>弧线：{c.arc}</span>}
                      </div>
                    )}

                    <div style={{ display: 'flex', gap: 4, marginTop: 'auto', paddingTop: 4 }}>
                      <Button size="small" type="text" icon={<EditOutlined />} title="编辑"
                        onClick={() => openEdit(c)} style={{ fontSize: 12, padding: '0 6px' }} />
                      <Button size="small" type="text" icon={<SwapOutlined />} title="设为当前聊天人格" disabled={!c.chatEnabled}
                        onClick={() => handleSetPersona(c)} style={{ fontSize: 12, padding: '0 6px' }} />
                      {c.chatEnabled && (
                        <Button size="small" type="text" icon={<DatabaseOutlined />} title="查看状态 / 记忆 / 追踪"
                          onClick={() => setMemoryChar(c)} style={{ fontSize: 12, padding: '0 6px' }} />
                      )}
                      {hasProject && (
                        inProject
                          ? <Button size="small" type="text" title="从当前项目移除（角色保留在库）"
                              onClick={() => handleDissociate(c)} style={{ fontSize: 12, padding: '0 6px', color: C('color-text-secondary') }}>
                              <ReadOutlined /> 已加入
                            </Button>
                          : <Button size="small" type="text" title="加入当前项目"
                              onClick={() => handleAssociate(c)} style={{ fontSize: 12, padding: '0 6px' }}>
                              加入项目
                            </Button>
                      )}
                      <div style={{ flex: 1 }} />
                      <Popconfirm
                        title={c.kind === 'builtin' ? `隐藏「${c.name}」？` : `删除「${c.name}」？删除会同时清理项目引用与聊天通道`}
                        okText={c.kind === 'builtin' ? '隐藏' : '删除'} cancelText="取消"
                        onConfirm={() => handleDelete(c)}>
                        <Button size="small" type="text" danger icon={<DeleteOutlined />} title="删除"
                          style={{ fontSize: 12, padding: '0 6px' }} />
                      </Popconfirm>
                    </div>
                  </div>
                </Col>
              )
            })}
          </Row>
        )}
      </div>

      {total > PAGE_SIZE && (
        <div style={{ display: 'flex', justifyContent: 'center', paddingTop: 10, flexShrink: 0 }}>
          <Pagination size="small" current={page} pageSize={PAGE_SIZE} total={total} onChange={setPage}
            showSizeChanger={false} />
        </div>
      )}

      <CharacterLibEditor
        open={editorOpen}
        character={editing}
        projects={editingProjects}
        onClose={() => setEditorOpen(false)}
        onSaved={handleSaved}
      />
      <CharacterMemoryModal
        open={!!memoryChar}
        character={memoryChar}
        onClose={() => setMemoryChar(null)}
      />
    </div>
  )
}

export default CharacterLibraryPage
