// CharacterLibraryPage.tsx — 角色库
// 统一整合小说角色（项目 characters.json）与轻语角色（人格预设 + 虚拟助手）。
import React, { useState, useEffect, useMemo, useCallback } from 'react'
import {
  Typography, Empty, Button, Input, Modal, Select, message, Row, Col, Tabs,
} from 'antd'
import {
  UserOutlined, HeartOutlined, PlusOutlined, SwapOutlined, TeamOutlined,
} from '@ant-design/icons'
import type { CharacterData, OrganizationData, RelationshipData } from '../types'
import { useAppStore } from '../stores/appStore'
import { C } from '../utils/theme'
import CharacterCard from '../components/novel/character/CharacterCard'
import CharacterEditor from '../components/novel/character/CharacterEditor'
import PortraitLightbox from '../components/novel/character/PortraitLightbox'
import TisorRadar from '../components/TisorRadar'
import WhisperRolePanel, { type WhisperPersonality } from '../components/characterlib/WhisperRolePanel'
import {
  getCharacters, saveCharacter, deleteCharacter,
  generateCharacters, generateSingleCharacter,
  toggleOrgMember, deleteRelationship,
} from '../components/novel/api/character'
import * as App from '../../wailsjs/go/app/App'

const statusOptions = [
  { value: 'Alive', label: '存活' }, { value: 'Dead', label: '已故' },
  { value: 'Missing', label: '失踪' }, { value: 'Transformed', label: '变身' },
]

const PERSONALITY_KEY = 'gaea_whisper_personality'

/** 设为当前轻语人格：本地持久化 + 广播（聊天页联动） */
function setCurrentPersona(id: string) {
  try { localStorage.setItem(PERSONALITY_KEY, id) } catch (_) {}
  window.dispatchEvent(new CustomEvent('gaea-persona-changed', { detail: { id } }))
}

const CharacterLibraryPage: React.FC = () => {
  // ── 小说角色状态 ──
  const [characters, setCharacters] = useState<CharacterData[]>([])
  const [organizations, setOrganizations] = useState<OrganizationData[]>([])
  const [relationships, setRelationships] = useState<RelationshipData[]>([])
  const [modalChar, setModalChar] = useState<CharacterData | null>(null)
  const [editForm, setEditForm] = useState<CharacterData | null>(null)
  const [generating, setGenerating] = useState(false)
  const [genCount, setGenCount] = useState(5)
  const [genSingle, setGenSingle] = useState(false)
  const [generatingPortrait, setGeneratingPortrait] = useState(false)
  const [portraitFullscreen, setPortraitFullscreen] = useState('')
  const [filterGender, setFilterGender] = useState('')
  const [filterRole, setFilterRole] = useState('')
  const [filterStatus, setFilterStatus] = useState('')
  const [filterOrg, setFilterOrg] = useState('')
  const [portraitModelOpen, setPortraitModelOpen] = useState(false)
  const [portraitModel, setPortraitModel] = useState('')
  const [portraitModels, setPortraitModels] = useState<{ engine: string; model: string }[]>([])
  const [portraitPrompt, setPortraitPrompt] = useState('')

  // ── 轻语角色状态 ──
  const [personalities, setPersonalities] = useState<WhisperPersonality[]>([])
  const [activePersonality, setActivePersonality] = useState<string>(() => {
    try { return localStorage.getItem(PERSONALITY_KEY) || 'gaea' } catch { return 'gaea' }
  })
  const projectPath = useAppStore((s) => s.projectPath)

  const loadNovelData = useCallback(async () => {
    try {
      const data = await getCharacters()
      setCharacters(data.characters || [])
      setOrganizations(data.organizations || [])
      setRelationships(data.relationships || [])
    } catch (err) { console.error('[CharacterLibrary] loadData:', err) }
  }, [])

  useEffect(() => {
    setCharacters([]); setOrganizations([]); setRelationships([])
    if (projectPath) loadNovelData()
  }, [projectPath, loadNovelData])

  useEffect(() => {
    App.WhisperGetPersonalities().then((ps: any) => setPersonalities(ps || [])).catch(() => {})
    const onPersona = (e: Event) => {
      const id = (e as CustomEvent).detail?.id
      if (id) setActivePersonality(id)
    }
    window.addEventListener('gaea-persona-changed', onPersona)
    return () => window.removeEventListener('gaea-persona-changed', onPersona)
  }, [])

  const refreshFromResult = (result: any) => {
    if (result.characters) setCharacters(result.characters)
    if (result.organizations) setOrganizations(result.organizations)
    if (result.relationships) setRelationships(result.relationships)
    if (editForm && result.characters) {
      const u = result.characters.find((c: CharacterData) => c.id === editForm.id)
      if (u) setEditForm(u)
    }
  }

  // ── 角色操作 ──
  const openCharModal = (ch: CharacterData) => {
    setModalChar(ch); setEditForm({ ...ch })
  }

  const handleNewCharacter = () => {
    const blank: CharacterData = {
      id: 'char_' + Date.now(), name: '新角色', role_type: 'supporting', gender: '', age: '',
      personality: '', background: '', appearance: '', figure: '',
      motivation: '', arc: '', status: 'Alive',
    }
    setModalChar(blank); setEditForm({ ...blank })
  }

  const handleBatchGenerate = async () => {
    setGenerating(true)
    try {
      const result = await generateCharacters(genCount)
      refreshFromResult(result)
      message.success(`已生成${genCount}个角色`)
    } catch (err: any) { message.error(err.message || '生成失败') }
    finally { setGenerating(false) }
  }

  const handleSaveChar = async () => {
    if (!editForm) return
    try {
      await saveCharacter(editForm)
      setCharacters((prev) => {
        const idx = prev.findIndex((c) => c.id === editForm.id)
        if (idx >= 0) { const n = [...prev]; n[idx] = editForm; return n }
        return [...prev, editForm]
      })
      setModalChar(editForm); message.success('已保存')
    } catch (err) { console.error('[CharacterLibrary] SaveCharacter:', err); message.error('保存失败') }
  }

  const handleDeleteChar = async (id: string) => {
    try {
      await deleteCharacter(id)
      setCharacters((prev) => prev.filter((c) => c.id !== id))
      if (modalChar?.id === id) setModalChar(null)
    } catch (err) { console.error('[CharacterLibrary] DeleteCharacter:', err) }
  }

  const handleRandomGenerate = async () => {
    if (!editForm) return
    setGenSingle(true)
    try {
      const result = await generateSingleCharacter(editForm)
      if (result.character) setEditForm({ ...editForm, ...JSON.parse(result.character) })
    } catch (err: any) { message.error(err.message || '生成失败') }
    finally { setGenSingle(false) }
  }

  // 小说 → 轻语（打通互传通道）
  const handleImportToWhisper = async () => {
    if (!editForm) return
    try {
      const ast = {
        id: `char_${editForm.id}`,
        name: editForm.name,
        personalityId: `char_${editForm.id}`,
        voiceGuide: [editForm.personality, editForm.background ? `背景：${editForm.background}` : ''].filter(Boolean).join('。'),
        gender: editForm.gender || 'neutral',
        tags: editForm.role_type ? [editForm.role_type] : undefined,
        portraitUrl: editForm.portrait_url || '',
        enabled: true,
      }
      await (App as any).WhisperAssistantSave(ast)
      message.success(`已导入「${editForm.name}」为轻语角色`)
    } catch (err: any) {
      message.error(err?.message || '导入失败')
    }
  }

  // 组织/关系（编辑器内维护，供角色卡参数使用）
  const handleNewOrg = () => { /* 组织管理保留在小说模块 */ }
  const handleToggleOrgMember = async (orgID: string) => {
    if (!editForm) return
    try {
      await toggleOrgMember(editForm.id, orgID)
      setOrganizations((prev) => prev.map((o) => {
        if (o.id !== orgID) return o
        const m = o.members || []
        return { ...o, members: m.includes(editForm!.id) ? m.filter((x) => x !== editForm!.id) : [...m, editForm!.id] }
      }))
    } catch (err) { console.error('[CharacterLibrary] ToggleOrgMember:', err) }
  }
  const handleDeleteRel = async (rel: RelationshipData) => {
    try {
      await deleteRelationship(rel.from_id, rel.to_id)
      setRelationships((prev) => prev.filter((r) => r.from_id !== rel.from_id || r.to_id !== rel.to_id))
    } catch (err) { console.error('[CharacterLibrary] DeleteRelationship:', err) }
  }

  const buildPortraitPrompt = (ch: any) => {
    const parts: string[] = []
    const gl: Record<string, string> = { '男': '男性', '女': '女性', 'male': '男性', 'female': '女性' }
    parts.push(ch.gender && gl[ch.gender] ? gl[ch.gender] + '角色 ' + ch.name : ch.name)
    const rm: Record<string, string> = { protagonist: '主角', antagonist: '反派', supporting: '配角', minor: '次要角色' }
    if (ch.role_type && rm[ch.role_type]) parts.push(rm[ch.role_type])
    if (ch.appearance) parts.push(ch.appearance)
    if (ch.figure) parts.push(ch.figure)
    if (ch.personality) parts.push(ch.personality)
    if (ch.background) parts.push('背景：' + ch.background)
    if (ch.age) parts.push('年龄' + ch.age + '岁')
    parts.push('电影级光影，8K超高清，半身肖像，深色氛围背景。')
    return parts.join('。')
  }

  const handleGeneratePortrait = async () => {
    if (!editForm) return
    setPortraitPrompt(buildPortraitPrompt(editForm))
    try {
      const config: any = await App.GetImageBackendConfig()
      const models: { engine: string; model: string }[] = config?.availableModels || []
      const current = config?.currentModel || ''
      setPortraitModels(models)
      setPortraitModel(current)
      setPortraitModelOpen(true)
    } catch (_) { doGenerate() }
  }

  const doGenerate = async () => {
    setPortraitModelOpen(false)
    setGeneratingPortrait(true)
    try {
      const { generateImage } = await import('../api/image')
      const res = await generateImage(portraitPrompt, '', '1024x1024', portraitModel, 0, 1)
      if (res?.error) { message.error(res.error) }
      else if (res?.images?.[0]?.image) {
        const url = res.images[0].image
        setEditForm({ ...editForm!, portrait_url: url })
        setCharacters((prev) => prev.map((c) => c.id === editForm!.id ? { ...c, portrait_url: url } : c))
        message.success('剧照已生成')
      } else { message.error('生成失败') }
    } catch (err: any) { message.error(err?.message || '生成失败') }
    finally { setGeneratingPortrait(false) }
  }

  const filteredCharacters = useMemo(() => {
    return characters.filter((ch) => {
      if (filterGender && ch.gender !== filterGender) return false
      if (filterRole && ch.role_type !== filterRole) return false
      if (filterStatus && ch.status !== filterStatus) return false
      if (filterOrg) {
        const org = organizations.find((o) => o.id === filterOrg)
        if (!org || !org.members?.includes(ch.id)) return false
      }
      return true
    })
  }, [characters, filterGender, filterRole, filterStatus, filterOrg, organizations])

  const novelTab = (
    <div>
      {!projectPath ? (
        <Empty description="请先打开小说项目以管理小说角色" image={Empty.PRESENTED_IMAGE_SIMPLE} style={{ marginTop: 72 }}>
          <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 12, display: 'block', marginBottom: 12 }}>
            轻语角色不依赖项目，可直接在「轻语角色」页签管理
          </Typography.Text>
        </Empty>
      ) : (
        <>
          {/* 工具栏 */}
          <div style={{ display: 'flex', gap: 8, alignItems: 'center', marginBottom: 12, flexWrap: 'wrap' }}>
            <Select size="small" value={filterGender} onChange={setFilterGender} style={{ width: 90 }}
              options={[{ label: '全部性别', value: '' }, { label: '♂ 男', value: 'male' }, { label: '♀ 女', value: 'female' }]} />
            <Select size="small" value={filterRole} onChange={setFilterRole} style={{ width: 100 }}
              options={[{ label: '全部阵营', value: '' }, { label: '主角', value: 'protagonist' }, { label: '反派', value: 'antagonist' }, { label: '配角', value: 'supporting' }, { label: '次要', value: 'minor' }]} />
            <Select size="small" value={filterStatus} onChange={setFilterStatus} style={{ width: 100 }}
              options={[{ label: '全部状态', value: '' }, { label: '存活', value: 'Alive' }, { label: '已故', value: 'Dead' }, { label: '失踪', value: 'Missing' }, { label: '变身', value: 'Transformed' }]} />
            <Select size="small" value={filterOrg} onChange={setFilterOrg} style={{ width: 120 }}
              options={[{ label: '全部组织', value: '' }, ...organizations.map((o) => ({ label: o.name, value: o.id }))]} />
            {(filterGender || filterRole || filterStatus || filterOrg) && (
              <Button size="small" onClick={() => { setFilterGender(''); setFilterRole(''); setFilterStatus(''); setFilterOrg('') }} style={{ fontSize: 10, padding: '0 6px' }}>
                ✕ 清除
              </Button>
            )}
            <div style={{ flex: 1 }} />
            <Select size="small" value={genCount} onChange={setGenCount}
              style={{ width: 76 }}
              options={[1, 3, 5, 8].map(n => ({ value: n, label: `${n} 个` }))} />
            <Button size="small" icon={<PlusOutlined />} loading={generating} onClick={handleBatchGenerate}>
              AI 生成
            </Button>
            <Button size="small" type="primary" icon={<PlusOutlined />} onClick={handleNewCharacter}>
              新建角色
            </Button>
          </div>

          {characters.length === 0 ? (
            <Empty description="暂无角色" image={Empty.PRESENTED_IMAGE_SIMPLE} style={{ marginTop: 80 }}>
              <Button type="primary" icon={<PlusOutlined />} onClick={handleNewCharacter}>新建角色</Button>
            </Empty>
          ) : filteredCharacters.length === 0 ? (
            <div style={{ textAlign: 'center', padding: 40, color: C('color-text-secondary'), fontSize: 12 }}>
              没有匹配筛选条件的角色
            </div>
          ) : (
            <Row gutter={[12, 12]}>
              {filteredCharacters.map((ch) => {
                const relCount = relationships.filter((r) => r.from_id === ch.id || r.to_id === ch.id).length
                return (
                  <Col key={ch.id} xs={24} sm={12} lg={8} xl={6}>
                    <CharacterCard
                      character={ch} relationCount={relCount}
                      onClick={() => openCharModal(ch)}
                      onPortraitFullscreen={setPortraitFullscreen}
                    />
                  </Col>
                )
              })}
            </Row>
          )}
        </>
      )}
    </div>
  )

  const whisperTab = (
    <div>
      {/* 人格预设：只读浏览 + 设为当前 */}
      <div style={{ marginBottom: 18 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 10 }}>
          <Typography.Text strong style={{ color: C('color-text'), fontSize: 13 }}>
            <HeartOutlined style={{ marginRight: 6, color: '#f472b6' }} />人格预设
          </Typography.Text>
          <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11 }}>
            {personalities.length} 种 · 点击「设为当前」切换聊天人格
          </Typography.Text>
        </div>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(150px, 1fr))', gap: 10 }}>
          {personalities.map(p => {
            const active = p.id === activePersonality
            return (
              <div key={p.id}
                style={{
                  padding: '12px 12px 10px', borderRadius: 14,
                  border: active ? '1px solid color-mix(in srgb, #f472b6 45%, transparent)' : '1px solid var(--md-sys-color-outline-variant)',
                  background: active ? 'color-mix(in srgb, #f472b6 9%, transparent)' : 'var(--gaea-glass-bg, var(--md-sys-color-surface-container))',
                  display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 6,
                  transition: 'transform 180ms, border-color 180ms',
                }}
                onMouseEnter={(e) => { e.currentTarget.style.transform = 'translateY(-2px)' }}
                onMouseLeave={(e) => { e.currentTarget.style.transform = 'translateY(0)' }}
              >
                <TisorRadar dims={p.dims} size={64} color={active ? '#f472b6' : '#8899aa'} showLabels={false} />
                <Typography.Text style={{ color: C('color-text'), fontSize: 12.5, fontWeight: 600 }}>{p.label}</Typography.Text>
                <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 10, textAlign: 'center', lineHeight: 1.5, minHeight: 30 }}>
                  {p.tags?.slice(0, 3).join(' · ') || p.gender === 'female' ? '女性' : p.gender === 'male' ? '男性' : '中性'}
                </Typography.Text>
                <Button size="small" type={active ? 'primary' : 'default'}
                  icon={<SwapOutlined style={{ fontSize: 10 }} />}
                  disabled={active}
                  onClick={() => { setCurrentPersona(p.id); message.success(`已设为当前人格：${p.label}`) }}
                  style={{ borderRadius: 999, fontSize: 11, height: 26 }}>
                  {active ? '当前人格' : '设为当前'}
                </Button>
              </div>
            )
          })}
        </div>
      </div>

      {/* 虚拟助手（轻语角色）管理 */}
      <WhisperRolePanel
        activePersonality={activePersonality}
        onSwitchPersonality={(id) => {
          setActivePersonality(id)
          message.success('已切换轻语人格，可在聊天板块继续对话')
        }}
      />
    </div>
  )

  return (
    <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minHeight: 0, position: 'relative' }}>
      <style>{`
        .characterlib-page .ant-tabs-content-holder { flex: 1; min-height: 0; }
      `}</style>
      <div className="characterlib-page" style={{ flex: 1, display: 'flex', flexDirection: 'column', minHeight: 0 }}>
        <div style={{ display: 'flex', alignItems: 'baseline', gap: 10, marginBottom: 8, flexShrink: 0 }}>
          <Typography.Title level={4} style={{ margin: 0, color: C('color-text'), fontSize: 17 }}>
            <TeamOutlined style={{ marginRight: 8, color: 'var(--gaea-glow)' }} />角色库
          </Typography.Title>
          <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 12 }}>
            小说角色 {characters.length} · 轻语人格 {personalities.length}
          </Typography.Text>
        </div>
        <Tabs
          size="small"
          items={[
            { key: 'novel', label: <span><UserOutlined /> 小说角色</span>, children: novelTab },
            { key: 'whisper', label: <span><HeartOutlined /> 轻语角色</span>, children: whisperTab },
          ]}
          tabBarStyle={{ borderColor: C('color-border') }}
        />
      </div>

      {/* 小说角色编辑 */}
      {editForm && (
        <CharacterEditor
          editForm={editForm} setEditForm={setEditForm}
          onClose={() => setModalChar(null)}
          onDelete={handleDeleteChar} onSave={handleSaveChar}
          onRandomGenerate={handleRandomGenerate}
          onGeneratePortrait={handleGeneratePortrait}
          onImportToWhisper={handleImportToWhisper}
          generatingPortrait={generatingPortrait} genSingle={genSingle}
          characters={characters} organizations={organizations}
          relationships={relationships}
          onNewOrg={handleNewOrg} onToggleOrgMember={handleToggleOrgMember}
          onOpenRelModal={() => { message.info('关系管理保留在小说模块') }}
          onDeleteRel={handleDeleteRel}
          statusOptions={statusOptions} onPortraitFullscreen={setPortraitFullscreen}
        />
      )}

      {/* 剧照生成弹窗 */}
      <Modal title="生成角色剧照" open={portraitModelOpen}
        onOk={() => doGenerate()} onCancel={() => setPortraitModelOpen(false)}
        okText="生成" width={520}>
        <Select value={portraitModel} onChange={setPortraitModel} style={{ width: '100%', marginBottom: 10 }}
          options={portraitModels.map(m => ({ value: m.model, label: `${m.model} (${m.engine})` }))} />
        <Input.TextArea value={portraitPrompt} onChange={e => setPortraitPrompt(e.target.value)}
          rows={4} placeholder="输入图像生成提示词..."
          style={{ background: 'rgba(0,0,0,0.2)', border: '1px solid var(--border-subtle)', color: C('color-text'), borderRadius: 6, fontSize: 13 }} />
      </Modal>

      <PortraitLightbox imageUrl={portraitFullscreen} onClose={() => setPortraitFullscreen('')} />
    </div>
  )
}

export default CharacterLibraryPage
