import React, { useState, useEffect, useMemo } from 'react'
import {
  Typography, Empty, Button, Input, Modal,
  Select, message, Row, Col, Tabs, Space, Tag,
} from 'antd'
import {
  ThunderboltOutlined, PlusOutlined,
  UserOutlined, ApartmentOutlined, LinkOutlined,
} from '@ant-design/icons'
import RelationGraph from '../components/RelationGraph'
import type { Message } from '../components/ChatPanel'
import type { CharacterData, OrganizationData, RelationshipData } from '../types'
import ChatPanel from '../components/ChatPanel'
import { useAppStore } from '../stores/appStore'
import { C } from '../utils/theme'
import CharacterCard from '../components/novel/character/CharacterCard'
import OrganizationCard from '../components/novel/character/OrganizationCard'
import CharacterEditor from '../components/novel/character/CharacterEditor'
import RelationshipModal from '../components/novel/character/RelationshipModal'
import OrganizationEditModal from '../components/novel/character/OrganizationEditModal'
import PortraitLightbox from '../components/novel/character/PortraitLightbox'
import {
  getCharacters, saveCharacter, deleteCharacter,
  generateCharacters, generateSingleCharacter, chatCharacter,
  saveOrganization, deleteOrganization, toggleOrgMember,
  saveRelationship, deleteRelationship,
} from '../components/novel/api/character'

const statusOptions = [
  { value: 'Alive', label: '存活' }, { value: 'Dead', label: '已故' },
  { value: 'Missing', label: '失踪' }, { value: 'Transformed', label: '变身' },
]

const CharacterPage: React.FC = () => {
  const [characters, setCharacters] = useState<CharacterData[]>([])
  const [organizations, setOrganizations] = useState<OrganizationData[]>([])
  const [relationships, setRelationships] = useState<RelationshipData[]>([])
  const [modalChar, setModalChar] = useState<CharacterData | null>(null)
  const [editForm, setEditForm] = useState<CharacterData | null>(null)
  const [modalOrg, setModalOrg] = useState<OrganizationData | null>(null)
  const [editOrg, setEditOrg] = useState<OrganizationData | null>(null)
  const [generating, setGenerating] = useState(false)
  const [genCount, setGenCount] = useState(5)
  const [genSingle, setGenSingle] = useState(false)
  const [chatMessages, setChatMessages] = useState<Message[]>([])
  const [relTargetId, setRelTargetId] = useState<string>('')
  const [relType, setRelType] = useState<string>('friend')
  const [relModalOpen, setRelModalOpen] = useState(false)
  const [generatingPortrait, setGeneratingPortrait] = useState(false)
  const [portraitFullscreen, setPortraitFullscreen] = useState('')
  const [filterGender, setFilterGender] = useState<string>('')
  const [filterRole, setFilterRole] = useState<string>('')
  const [filterStatus, setFilterStatus] = useState<string>('')
  const [filterOrg, setFilterOrg] = useState<string>('')
  const [portraitModelOpen, setPortraitModelOpen] = useState(false)
  const [portraitModel, setPortraitModel] = useState('')
  const [portraitModels, setPortraitModels] = useState<{engine: string; model: string}[]>([])
  const [portraitPrompt, setPortraitPrompt] = useState('')

  const projectPath = useAppStore((s) => s.projectPath)
  useEffect(() => {
    setCharacters([])
    setOrganizations([])
    if (projectPath) { loadData() }
  }, [projectPath])

  const loadData = async () => {
    try {
      const data = await getCharacters()
      setCharacters(data.characters || [])
      setOrganizations(data.organizations || [])
      setRelationships(data.relationships || [])
    } catch (err) { console.error('[CharacterPage] loadData:', err) }
  }

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
    setModalChar(ch); setEditForm({ ...ch }); setChatMessages([])
  }

  const handleNewCharacter = () => {
    const blank: CharacterData = {
      id: 'char_' + Date.now(), name: '新角色', role_type: 'supporting', gender: '', age: '',
      personality: '', background: '', appearance: '', figure: '',
      motivation: '', arc: '', status: 'Alive',
    }
    setModalChar(blank); setEditForm({ ...blank }); setChatMessages([])
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
    } catch (err) { console.error('[CharacterPage] SaveCharacter:', err); message.error('保存失败') }
  }

  const handleCharacterChat = async (userMsg: string): Promise<string> => {
    try {
      const result = await chatCharacter(userMsg)
      if (result.characters) setCharacters(result.characters)
      if (result.organizations) setOrganizations(result.organizations)
      if (result.relationships) setRelationships(result.relationships)
      return result.reply || ''
    } catch (err: any) {
      throw new Error(typeof err === 'string' ? err : (err?.message || '对话失败'))
    }
  }

  const handleDeleteChar = async (id: string) => {
    try {
      await deleteCharacter(id)
      setCharacters((prev) => prev.filter((c) => c.id !== id))
      if (modalChar?.id === id) setModalChar(null)
    } catch (err) { console.error('[CharacterPage] DeleteCharacter:', err) }
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

  // ── 组织操作 ──
  const openOrgModal = (org: OrganizationData) => { setModalOrg(org); setEditOrg({ ...org }) }

  const handleNewOrg = () => {
    const blank: OrganizationData = { id: 'org_' + Date.now(), name: '新组织', type: '', description: '', power_level: '' }
    setModalOrg(blank); setEditOrg({ ...blank })
  }

  const handleSaveOrg = async () => {
    if (!editOrg) return
    try {
      await saveOrganization(editOrg)
      setOrganizations((prev) => {
        const idx = prev.findIndex((o) => o.id === editOrg.id)
        if (idx >= 0) { const n = [...prev]; n[idx] = editOrg!; return n }
        return [...prev, editOrg!]
      })
      setModalOrg(editOrg); message.success('已保存')
    } catch (err) { console.error('[CharacterPage] SaveOrg:', err); message.error('保存失败') }
  }

  const handleDeleteOrg = async (id: string) => {
    try {
      await deleteOrganization(id)
      setOrganizations((prev) => prev.filter((o) => o.id !== id))
      setModalOrg(null)
    } catch (err) { console.error('[CharacterPage] DeleteOrganization:', err) }
  }

  const handleToggleOrgMember = async (orgID: string) => {
    if (!editForm) return
    try {
      await toggleOrgMember(editForm.id, orgID)
      setOrganizations((prev) => prev.map((o) => {
        if (o.id !== orgID) return o
        const m = o.members || []
        return { ...o, members: m.includes(editForm!.id) ? m.filter((x) => x !== editForm!.id) : [...m, editForm!.id] }
      }))
    } catch (err) { console.error('[CharacterPage] ToggleOrgMember:', err) }
  }

  // ── 关系操作 ──
  const openRelModal = () => {
    setRelTargetId('')
    setRelType('friend')
    setRelModalOpen(true)
  }

  const handleAddRel = async () => {
    if (!editForm || !relTargetId) return
    const rel: RelationshipData = { from_id: editForm.id, to_id: relTargetId, relation_type: relType, description: '', intimacy: 0 }
    try {
      await saveRelationship(rel)
      setRelationships((prev) => [...prev, rel])
      setRelModalOpen(false)
      message.success('关系已添加')
    } catch (err) { console.error('[CharacterPage] CreateRelationship:', err); message.error('创建失败') }
  }

  const handleDeleteRel = async (rel: RelationshipData) => {
    try {
      await deleteRelationship(rel.from_id, rel.to_id)
      setRelationships((prev) => prev.filter((r) => r.from_id !== rel.from_id || r.to_id !== rel.to_id))
    } catch (err) { console.error('[CharacterPage] DeleteRelationship:', err) }
  }

  const buildPortraitPrompt = (ch: any) => {
    const parts: string[] = []
    const gl: Record<string,string> = {'男':'男性','女':'女性','male':'男性','female':'女性'}
    parts.push(ch.gender && gl[ch.gender] ? gl[ch.gender]+'角色 '+ch.name : ch.name)
    const rm: Record<string,string> = {protagonist:'主角',antagonist:'反派',supporting:'配角',minor:'次要角色'}
    if (ch.role_type && rm[ch.role_type]) parts.push(rm[ch.role_type])
    if (ch.appearance) parts.push(ch.appearance)
    if (ch.figure) parts.push(ch.figure)
    if (ch.personality) parts.push(ch.personality)
    if (ch.background) parts.push('背景：'+ch.background)
    if (ch.age) parts.push('年龄'+ch.age+'岁')
    parts.push('电影级光影，8K超高清，半身肖像，深色氛围背景。')
    return parts.join('。')
  }

  const handleGeneratePortrait = async () => {
    if (!editForm) return
    setPortraitPrompt(buildPortraitPrompt(editForm))
    try {
      // @ts-ignore
      const config = await window.go.app.App.GetImageBackendConfig()
      const models: {engine: string; model: string}[] = (config as any)?.availableModels || []
      const current = (config as any)?.currentModel || ''
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

  // 导入为轻语角色（小说 → 轻语，打通互传通道）
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
      // @ts-ignore
      await window.go.app.App.WhisperAssistantSave(ast)
      message.success(`已导入「${editForm.name}」为轻语角色`)
    } catch (err: any) {
      message.error(err?.message || '导入失败')
    }
  }

  const getCharName = (id: string) =>
    characters.find((c) => c.id === id)?.name || organizations.find((o) => o.id === id)?.name || id

  // 筛选后的角色列表
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
  // ── 剧照生成弹窗 ──
  const renderPortraitModal = () => (
    <Modal title="生成角色剧照" open={portraitModelOpen}
      onOk={() => doGenerate()}
      onCancel={() => setPortraitModelOpen(false)}
      okText="生成" width={520}>
      <Select value={portraitModel} onChange={setPortraitModel}
        style={{width:'100%',marginBottom:10}}
        options={portraitModels.map(m=>({value:m.model,label:m.model+' ('+m.engine+')'}))} />
      <Input.TextArea value={portraitPrompt} onChange={e=>setPortraitPrompt(e.target.value)}
        rows={4} placeholder="输入图像生成提示词..."
        style={{background:'rgba(0,0,0,0.2)',border:'1px solid var(--border-subtle)',color:'var(--color-text)',borderRadius:6,fontSize:13}} />
    </Modal>
  )

  // ── 共享 CharacterEditor 包装（消除桌面/移动端 40 行重复）──
  const renderCharEditor = () => {
    if (!editForm) return null
    return (
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
        onOpenRelModal={openRelModal} onDeleteRel={handleDeleteRel}
        statusOptions={statusOptions} onPortraitFullscreen={setPortraitFullscreen}
      />
    )
  }

  // ── Tab 内容 ──
  const tabItems = [
    {
      key: 'chars',
      label: <span><UserOutlined /> 角色 ({filteredCharacters.length}/{characters.length})</span>,
      children: (
        <>
          {/* 筛选栏 */}
          {characters.length > 0 && (
            <div style={{
              display: 'flex', gap: 8, alignItems: 'center',
              marginBottom: 12, flexWrap: 'wrap',
            }}>
              <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11, flexShrink: 0 }}>
                筛选:
              </Typography.Text>
              <Select size="small" value={filterGender} onChange={setFilterGender}
                style={{ width: 90 }}
                options={[
                  { label: '全部性别', value: '' },
                  { label: '♂ 男', value: 'male' },
                  { label: '♀ 女', value: 'female' },
                ]} />
              <Select size="small" value={filterRole} onChange={setFilterRole}
                style={{ width: 100 }}
                options={[
                  { label: '全部阵营', value: '' },
                  { label: '主角', value: 'protagonist' },
                  { label: '反派', value: 'antagonist' },
                  { label: '配角', value: 'supporting' },
                  { label: '次要', value: 'minor' },
                ]} />
              <Select size="small" value={filterStatus} onChange={setFilterStatus}
                style={{ width: 100 }}
                options={[
                  { label: '全部状态', value: '' },
                  { label: '存活', value: 'Alive' },
                  { label: '已故', value: 'Dead' },
                  { label: '失踪', value: 'Missing' },
                  { label: '变身', value: 'Transformed' },
                ]} />
              <Select size="small" value={filterOrg} onChange={setFilterOrg}
                style={{ width: 120 }}
                options={[
                  { label: '全部组织', value: '' },
                  ...organizations.map((o) => ({ label: o.name, value: o.id })),
                ]} />
              {(filterGender || filterRole || filterStatus || filterOrg) && (
                <Button size="small" onClick={() => {
                  setFilterGender(''); setFilterRole(''); setFilterStatus(''); setFilterOrg('')
                }} style={{ fontSize: 10, padding: '0 6px' }}>
                  ✕ 清除
                </Button>
              )}
            </div>
          )}
          {characters.length === 0 ? (
            <Empty description="暂无角色" image={Empty.PRESENTED_IMAGE_SIMPLE} style={{ marginTop: 80 }} />
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
      ),
    },
    {
      key: 'orgs',
      label: <span><ApartmentOutlined /> 组织 ({organizations.length})</span>,
      children: organizations.length === 0 ? (
        <Empty description="暂无组织" image={Empty.PRESENTED_IMAGE_SIMPLE} style={{ marginTop: 80 }}>
          <Button icon={<PlusOutlined />} onClick={handleNewOrg}>新建组织</Button>
        </Empty>
      ) : (
        <Row gutter={[12, 12]}>
          {organizations.map((org) => (
            <Col key={org.id} xs={24} sm={12} lg={8} xl={6}>
              <OrganizationCard organization={org} onClick={() => openOrgModal(org)} />
            </Col>
          ))}
        </Row>
      ),
    },
    {
      key: 'rels',
      label: <span><LinkOutlined /> 关系图 ({relationships.length})</span>,
      children: relationships.length === 0 ? (
        <Empty description="暂无关系" image={Empty.PRESENTED_IMAGE_SIMPLE} style={{ marginTop: 80 }} />
      ) : (
        <div style={{ height: '100%', minHeight: 400, overflow: 'hidden' }}>
          <RelationGraph
            characters={characters} organizations={organizations} relationships={relationships}
          />
        </div>
      ),
    },
  ]

  return (
    <div style={{ height: 'calc(100vh - 112px)', display: 'flex', flexDirection: 'column', gap: 12 }}>
      <div style={{ display: 'flex', justifyContent: 'flex-end', alignItems: 'center' }}>
        <Space>
          <Button size="small" icon={<PlusOutlined />} onClick={handleNewCharacter}
            style={{ background: 'var(--bg-elevated)', border: '1px solid rgba(96, 165, 250, 0.3)', color: '#60a5fa', borderRadius: 'var(--radius-md)' }}>新建角色</Button>
          <Input size="small" type="number" value={genCount} onChange={(e) => setGenCount(+e.target.value)}
            style={{ width: 50, background: 'rgba(255,255,255,0.05)', border: '1px solid var(--border-subtle)', borderRadius: 'var(--radius-md)', color: 'var(--color-text)' }} min={1} max={10} />
          <Button size="small" icon={<ThunderboltOutlined />} onClick={handleBatchGenerate} loading={generating}
            style={{ background: 'var(--color-primary)', borderColor: 'var(--color-primary)', boxShadow: 'var(--shadow-glow)', borderRadius: 'var(--radius-md)', color: '#000' }}>批量生成</Button>
        </Space>
      </div>

      <div style={{ flex: 1, overflow: 'auto' }}>
        <Tabs items={tabItems} size="small" style={{ color: C('color-text'), flex: 1, minHeight: 0 }} tabBarStyle={{ borderColor: C('color-border') }} />
      </div>

      {/* 角色 Agent */}
      <div style={{ flexShrink: 0, borderTop: '1px solid var(--border-subtle)', paddingTop: 8 }}>
        <ChatPanel
          title="💬 角色 Agent"
          messages={chatMessages}
          onMessagesChange={setChatMessages}
          onSend={handleCharacterChat}
          placeholder="描述你需要的角色，AI 会自动创建并保存…"
          fillHeight
          defaultCollapsed
        />
      </div>

      {/* 角色详情弹窗 / 移动端 Sheet */}
      {editForm && (
        <Modal title={null} open={!!modalChar} onCancel={() => setModalChar(null)} footer={null}
          width={960} styles={{ body: { background: 'var(--bg-glass)', backdropFilter: 'blur(8px)', WebkitBackdropFilter: 'blur(8px)', padding: 0 } }}>
          <div style={{ display: 'flex', height: 'calc(90vh - 120px)', minHeight: 520 }}>
            <div style={{ flex: 1, overflow: 'auto', padding: 20 }}>
              {renderCharEditor()}
            </div>
          </div>
        </Modal>
      )}

      {/* 组织编辑弹窗 */}
      <OrganizationEditModal
        open={!!modalOrg}
        org={editOrg}
        onClose={() => setModalOrg(null)}
        onSave={handleSaveOrg}
        onDelete={handleDeleteOrg}
        onEditOrgChange={setEditOrg}
        getCharName={getCharName}
      />

      {/* 关系添加弹窗 */}
      <RelationshipModal
        open={relModalOpen}
        onClose={() => setRelModalOpen(false)}
        characters={characters}
        organizations={organizations}
        editForm={editForm}
        relTargetId={relTargetId}
        onRelTargetChange={setRelTargetId}
        relType={relType}
        onRelTypeChange={setRelType}
        onAdd={handleAddRel}
      />

      {/* 剧照全屏 */}
      {portraitFullscreen && (
        <PortraitLightbox
          imageUrl={portraitFullscreen}
          onClose={() => setPortraitFullscreen('')}
        />
      )}

      {renderPortraitModal()}
    </div>
  )
}

export default CharacterPage
