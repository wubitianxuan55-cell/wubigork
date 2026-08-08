// CharacterPage.tsx — 小说角色面板（单向使用角色库）
// 约束：小说只引用角色库的角色，不自行生成、不回写全局角色；
// 面板内可改的只有项目内覆盖（定位 / 弧线状态 / 状态），全局设定一律去角色库。
import React, { useCallback, useEffect, useMemo, useState } from 'react'
import {
  Typography, Empty, Button, Input, Modal, InputNumber,
  Select, message, Row, Col, Tabs, Space, Tag, Switch, Popconfirm,
} from 'antd'
import {
  ThunderboltOutlined, PlusOutlined,
  UserOutlined, ApartmentOutlined, LinkOutlined, TeamOutlined,
  ImportOutlined, SyncOutlined, EditOutlined, SwapOutlined, DeleteOutlined,
} from '@ant-design/icons'
import RelationGraph from '../components/RelationGraph'
import type { CharacterData, OrganizationData, RelationshipData } from '../types'
import { useAppStore } from '../stores/appStore'
import { C } from '../utils/theme'
import CharacterCard from '../components/novel/character/CharacterCard'
import OrganizationCard from '../components/novel/character/OrganizationCard'
import RelationshipModal from '../components/novel/character/RelationshipModal'
import OrganizationEditModal from '../components/novel/character/OrganizationEditModal'
import PortraitLightbox from '../components/novel/character/PortraitLightbox'
import {
  getCharacters, saveOrganization, deleteOrganization, toggleOrgMember,
  saveRelationship, deleteRelationship,
} from '../components/novel/api/character'
import {
  listProjectCharacters, associateToProject, dissociateFromProject,
  syncProjectCharacters, importProjectCharacters, drawRandom, setProjectState,
  type LibraryCharacter,
} from '../api/characterlib'

const statusOptions = [
  { value: 'Alive', label: '存活' }, { value: 'Dead', label: '已故' },
  { value: 'Missing', label: '失踪' }, { value: 'Transformed', label: '变身' },
]
const roleOptions = [
  { value: 'protagonist', label: '主角' }, { value: 'antagonist', label: '反派' },
  { value: 'supporting', label: '配角' }, { value: 'minor', label: '龙套' },
]

function navigateToCharacterLib() {
  window.dispatchEvent(new CustomEvent('navigate', { detail: { page: 'characterlib' } }))
}

const CharacterPage: React.FC = () => {
  const [characters, setCharacters] = useState<CharacterData[]>([])
  const [organizations, setOrganizations] = useState<OrganizationData[]>([])
  const [relationships, setRelationships] = useState<RelationshipData[]>([])
  const [projectRefs, setProjectRefs] = useState<Set<string>>(new Set())

  const [modalOrg, setModalOrg] = useState<OrganizationData | null>(null)
  const [editOrg, setEditOrg] = useState<OrganizationData | null>(null)
  const [relTargetId, setRelTargetId] = useState<string>('')
  const [relFromId, setRelFromId] = useState<string>('')
  const [relType, setRelType] = useState<string>('friend')
  const [relModalOpen, setRelModalOpen] = useState(false)
  const [portraitFullscreen, setPortraitFullscreen] = useState('')
  const [filterGender, setFilterGender] = useState<string>('')
  const [filterRole, setFilterRole] = useState<string>('')
  const [filterStatus, setFilterStatus] = useState<string>('')
  const [filterOrg, setFilterOrg] = useState<string>('')

  // 抽卡
  const [drawOpen, setDrawOpen] = useState(false)
  const [drawCount, setDrawCount] = useState(5)
  const [drawGender, setDrawGender] = useState('')
  const [drawTags, setDrawTags] = useState('')
  const [drawChatOnly, setDrawChatOnly] = useState(false)
  const [drawResult, setDrawResult] = useState<LibraryCharacter[]>([])
  const [drawLoading, setDrawLoading] = useState(false)
  const [syncing, setSyncing] = useState(false)

  // 项目内状态编辑（唯一允许的小说侧写入）
  const [projectEdit, setProjectEdit] = useState<CharacterData | null>(null)
  const [peRole, setPeRole] = useState('')
  const [peArc, setPeArc] = useState('')
  const [peStatus, setPeStatus] = useState('')

  const projectPath = useAppStore(s => s.projectPath)

  const loadData = useCallback(async () => {
    try {
      const data = await getCharacters()
      setCharacters(data.characters || [])
      setOrganizations(data.organizations || [])
      setRelationships(data.relationships || [])
    } catch (err) { console.error('[CharacterPage] loadData:', err) }
  }, [])

  const loadRefs = useCallback(async () => {
    if (!projectPath) { setProjectRefs(new Set()); return }
    try {
      const refs = await listProjectCharacters()
      setProjectRefs(new Set(refs.map(r => r.characterId)))
    } catch (_) { setProjectRefs(new Set()) }
  }, [projectPath])

  useEffect(() => {
    setCharacters([]); setOrganizations([]); setRelationships([])
    if (projectPath) { loadData(); loadRefs() }
  }, [projectPath, loadData, loadRefs])

  // 旧项目数据检测：characters.json 里有未入库角色（未关联）
  const unimported = useMemo(
    () => characters.filter(c => !projectRefs.has(c.id)),
    [characters, projectRefs],
  )

  const refreshAll = useCallback(async () => {
    await loadData()
    await loadRefs()
  }, [loadData, loadRefs])

  // ── 抽卡：从角色库随机抽取，加入当前项目 ──
  const handleDraw = async () => {
    setDrawLoading(true)
    try {
      const items = await drawRandom(drawCount, drawGender, drawTags.trim(), drawChatOnly)
      setDrawResult(items || [])
      if (!items?.length) message.info('没有抽到符合条件的角色，换个条件试试')
    } catch (err: any) {
      message.error(err?.message || '抽卡失败')
    } finally {
      setDrawLoading(false)
    }
  }

  const handleAddDrawn = async (c: LibraryCharacter) => {
    try {
      await associateToProject(c.id, c.roleType || 'supporting')
      await syncProjectCharacters()
      await refreshAll()
      message.success(`「${c.name}」已加入本书`)
    } catch (err: any) {
      message.error(err?.message || '加入失败')
    }
  }

  const handleAddAllDrawn = async () => {
    const pending = drawResult.filter(c => !projectRefs.has(c.id))
    if (!pending.length) return
    try {
      for (const c of pending) await associateToProject(c.id, c.roleType || 'supporting')
      await syncProjectCharacters()
      await refreshAll()
      message.success(`已加入 ${pending.length} 个角色`)
      setDrawResult([])
    } catch (err: any) {
      message.error(err?.message || '加入失败')
    }
  }

  // ── 项目内状态编辑（只写关联表）──
  const openProjectEdit = (ch: CharacterData) => {
    setProjectEdit(ch)
    setPeRole(ch.role_type || 'supporting')
    setPeArc(ch.arc || '')
    setPeStatus(ch.status || 'Alive')
  }

  const handleSaveProjectState = async () => {
    if (!projectEdit) return
    try {
      await setProjectState(projectEdit.id, peRole, peArc, peStatus)
      await syncProjectCharacters()
      await loadData()
      message.success(`已更新「${projectEdit.name}」在本书的状态（全局角色未动）`)
      setProjectEdit(null)
    } catch (err: any) {
      message.error(err?.message || '保存失败')
    }
  }

  const handleRemoveFromProject = async (ch: CharacterData) => {
    try {
      await dissociateFromProject(ch.id)
      await syncProjectCharacters()
      await refreshAll()
      message.success(`「${ch.name}」已从本书移除（角色保留在角色库）`)
      setProjectEdit(null)
    } catch (err: any) {
      message.error(err?.message || '移除失败')
    }
  }

  const handleSync = async () => {
    setSyncing(true)
    try {
      await syncProjectCharacters()
      await loadData()
      message.success('已把本书引用的角色同步到 characters.json')
    } catch (err: any) {
      message.error(err?.message || '同步失败')
    } finally {
      setSyncing(false)
    }
  }

  const handleImportLegacy = async () => {
    try {
      const n = await importProjectCharacters()
      await loadRefs()
      message.success(`已将 ${n} 个旧项目角色一次性迁入角色库（后续本书只引用角色库）`)
    } catch (err: any) {
      message.error(err?.message || '迁移失败')
    }
  }

  // ── 组织 / 关系（项目内数据，保留原能力）──
  const getCharName = (id: string) =>
    characters.find(c => c.id === id)?.name || organizations.find(o => o.id === id)?.name || id

  const handleNewOrg = () => {
    const blank: OrganizationData = { id: 'org_' + Date.now(), name: '新组织', type: '', description: '', power_level: '' }
    setModalOrg(blank); setEditOrg({ ...blank })
  }

  const handleSaveOrg = async () => {
    if (!editOrg) return
    try {
      await saveOrganization(editOrg)
      await loadData()
      message.success('组织已保存')
      setModalOrg(null)
    } catch (err) { message.error('保存失败') }
  }
  const handleDeleteOrg = async (id: string) => {
    try {
      await deleteOrganization(id)
      await loadData()
      message.success('组织已删除')
    } catch (err) { message.error('删除失败') }
  }
  const handleToggleOrgMember = async (charID: string, orgID: string) => {
    try {
      await toggleOrgMember(charID, orgID)
      await loadData()
    } catch (err) { message.error('切换成员失败') }
  }
  const handleAddRel = async () => {
    if (!relFromId || !relTargetId) return
    const rel: RelationshipData = { from_id: relFromId, to_id: relTargetId, relation_type: relType, description: '', intimacy: 0 }
    try {
      await saveRelationship(rel)
      await loadData()
      message.success('关系已建立')
      setRelModalOpen(false)
    } catch (err) { message.error('建立关系失败') }
  }
  const handleDeleteRel = async (rel: RelationshipData) => {
    try {
      await deleteRelationship(rel.from_id, rel.to_id)
      await loadData()
    } catch (err) { message.error('删除关系失败') }
  }

  const filteredCharacters = useMemo(() => characters.filter(ch => {
    if (filterGender && ch.gender !== filterGender) return false
    if (filterRole && ch.role_type !== filterRole) return false
    if (filterStatus && ch.status !== filterStatus) return false
    if (filterOrg) {
      const org = organizations.find(o => o.id === filterOrg)
      if (!org || !org.members?.includes(ch.id)) return false
    }
    return true
  }), [characters, filterGender, filterRole, filterStatus, filterOrg, organizations])

  // ── 项目角色状态弹窗 ──
  const renderProjectEditModal = () => {
    if (!projectEdit) return null
    return (
      <Modal
        open={!!projectEdit}
        title={`「${projectEdit.name}」· 本书状态`}
        onCancel={() => setProjectEdit(null)}
        footer={
          <Space>
            <Popconfirm title={`把「${projectEdit.name}」从本书移除？角色保留在角色库`}
              okText="移除" cancelText="取消" onConfirm={() => handleRemoveFromProject(projectEdit)}>
              <Button danger size="small" icon={<DeleteOutlined />}>移出本书</Button>
            </Popconfirm>
            <div style={{ flex: 1 }} />
            <Button size="small" icon={<TeamOutlined />} onClick={() => { setProjectEdit(null); navigateToCharacterLib() }}>
              在角色库编辑全局设定
            </Button>
            <Button size="small" type="primary" icon={<EditOutlined />} onClick={handleSaveProjectState}>
              保存本书状态
            </Button>
          </Space>
        }
        width={620}
      >
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 10, marginBottom: 12 }}>
          <div>
            <Typography.Text style={{ fontSize: 11, color: C('color-text-secondary') }}>本书定位</Typography.Text>
            <Select size="small" value={peRole} onChange={setPeRole} style={{ width: '100%', marginTop: 4 }}
              options={roleOptions} />
          </div>
          <div>
            <Typography.Text style={{ fontSize: 11, color: C('color-text-secondary') }}>本书状态</Typography.Text>
            <Select size="small" value={peStatus} onChange={setPeStatus} style={{ width: '100%', marginTop: 4 }}
              options={statusOptions} />
          </div>
        </div>
        <div style={{ marginBottom: 12 }}>
          <Typography.Text style={{ fontSize: 11, color: C('color-text-secondary') }}>本书弧线状态（覆盖全局弧线，仅本小说生效）</Typography.Text>
          <Input.TextArea size="small" rows={2} value={peArc} onChange={e => setPeArc(e.target.value)}
            placeholder="如：第一卷结尾黑化；第二卷开始救赎…"
            style={{ marginTop: 4, fontSize: 12.5, background: 'rgba(0,0,0,0.18)', border: '1px solid var(--border-subtle)', color: C('color-text') }} />
        </div>
        <div style={{ fontSize: 11, color: C('color-text-secondary'), borderTop: '1px solid var(--border-subtle)', paddingTop: 10 }}>
          全局设定（只读）：{(projectEdit.personality || projectEdit.background || projectEdit.appearance || projectEdit.motivation)
            ? [projectEdit.personality && `性格：${projectEdit.personality}`, projectEdit.background && `背景：${projectEdit.background}`,
               projectEdit.appearance && `外貌：${projectEdit.appearance}`, projectEdit.motivation && `动机：${projectEdit.motivation}`]
              .filter(Boolean).join('；')
            : '（未填写）'}
        </div>
      </Modal>
    )
  }

  // ── 抽卡弹窗 ──
  const renderDrawModal = () => (
    <Modal open={drawOpen} title="从角色库抽卡" onCancel={() => setDrawOpen(false)}
      footer={null} width={680}>
      <div style={{ display: 'flex', gap: 8, alignItems: 'center', marginBottom: 12, flexWrap: 'wrap' }}>
        <Typography.Text style={{ fontSize: 11, color: C('color-text-secondary') }}>数量</Typography.Text>
        <InputNumber size="small" min={1} max={10} value={drawCount} onChange={v => setDrawCount(v || 5)} />
        <Select size="small" value={drawGender} onChange={setDrawGender} style={{ width: 90 }}
          options={[
            { value: '', label: '全部性别' },
            { value: 'female', label: '女性' },
            { value: 'male', label: '男性' },
          ]} />
        <Input size="small" placeholder="标签（如：剑修）" value={drawTags}
          onChange={e => setDrawTags(e.target.value)} style={{ width: 130 }} />
        <span style={{ display: 'inline-flex', alignItems: 'center', gap: 5, fontSize: 11, color: C('color-text-secondary') }}>
          <Switch size="small" checked={drawChatOnly} onChange={setDrawChatOnly} /> 可聊天
        </span>
        <Button size="small" type="primary" icon={<ThunderboltOutlined />} loading={drawLoading} onClick={handleDraw}>
          抽卡
        </Button>
        {drawResult.length > 0 && (
          <Button size="small" icon={<PlusOutlined />} onClick={handleAddAllDrawn}>
            全部加入本书（{drawResult.filter(c => !projectRefs.has(c.id)).length}）
          </Button>
        )}
      </div>
      {drawResult.length === 0 ? (
        <Empty description="抽到的角色会显示在这里，点「加入本书」引用" image={Empty.PRESENTED_IMAGE_SIMPLE} style={{ marginTop: 40 }} />
      ) : (
        <Row gutter={[10, 10]}>
          {drawResult.map(c => {
            const added = projectRefs.has(c.id)
            return (
              <Col key={c.id} xs={24} sm={12} lg={8}>
                <div style={{
                  padding: 10, borderRadius: 12, border: '1px solid var(--md-sys-color-outline-variant)',
                  background: 'var(--gaea-glass-bg, var(--md-sys-color-surface-container))',
                  display: 'flex', gap: 10, alignItems: 'center',
                }}>
                  {c.portraitUrl
                    ? <img src={c.portraitUrl} alt={c.name} style={{ width: 42, height: 42, borderRadius: 10, objectFit: 'cover', flexShrink: 0 }} />
                    : <div style={{ width: 42, height: 42, borderRadius: 10, background: 'rgba(244,114,182,0.12)', display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
                        <UserOutlined style={{ color: '#f472b6' }} />
                      </div>}
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ fontSize: 12.5, fontWeight: 600, color: C('color-text') }}>{c.name}</div>
                    <div style={{ fontSize: 10.5, color: C('color-text-secondary'), whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                      {c.roleType ? roleOptions.find(r => r.value === c.roleType)?.label || c.roleType : '未设定位'}
                      {c.tags?.length ? ` · ${c.tags.slice(0, 2).join('、')}` : ''}
                    </div>
                  </div>
                  {added
                    ? <Tag style={{ margin: 0 }}>已加入</Tag>
                    : <Button size="small" type="primary" icon={<SwapOutlined />} onClick={() => handleAddDrawn(c)}>加入</Button>}
                </div>
              </Col>
            )
          })}
        </Row>
      )}
    </Modal>
  )

  // ── Tab 内容 ──
  const tabItems = [
    {
      key: 'chars',
      label: <span><UserOutlined /> 角色 ({filteredCharacters.length}/{characters.length})</span>,
      children: (
        <>
          {characters.length > 0 && (
            <div style={{ display: 'flex', gap: 8, alignItems: 'center', marginBottom: 12, flexWrap: 'wrap' }}>
              <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11, flexShrink: 0 }}>筛选:</Typography.Text>
              <Select size="small" value={filterGender} onChange={setFilterGender} style={{ width: 90 }}
                options={[{ label: '全部性别', value: '' }, { label: '♂ 男', value: 'male' }, { label: '♀ 女', value: 'female' }]} />
              <Select size="small" value={filterRole} onChange={setFilterRole} style={{ width: 100 }}
                options={[{ label: '全部阵营', value: '' }, ...roleOptions]} />
              <Select size="small" value={filterStatus} onChange={setFilterStatus} style={{ width: 100 }}
                options={[{ label: '全部状态', value: '' }, ...statusOptions]} />
              <Select size="small" value={filterOrg} onChange={setFilterOrg} style={{ width: 120 }}
                options={[{ label: '全部组织', value: '' }, ...organizations.map(o => ({ label: o.name, value: o.id }))]} />
              {(filterGender || filterRole || filterStatus || filterOrg) && (
                <Button size="small" onClick={() => { setFilterGender(''); setFilterRole(''); setFilterStatus(''); setFilterOrg('') }}
                  style={{ fontSize: 10, padding: '0 6px' }}>✕ 清除</Button>
              )}
            </div>
          )}
          {characters.length === 0 ? (
            <Empty description="本书还没有角色，去角色库抽卡吧" image={Empty.PRESENTED_IMAGE_SIMPLE} style={{ marginTop: 80 }}>
              <Button type="primary" icon={<ThunderboltOutlined />} onClick={() => setDrawOpen(true)}>抽卡</Button>
            </Empty>
          ) : filteredCharacters.length === 0 ? (
            <div style={{ textAlign: 'center', padding: 40, color: C('color-text-secondary'), fontSize: 12 }}>
              没有匹配筛选条件的角色
            </div>
          ) : (
            <Row gutter={[12, 12]}>
              {filteredCharacters.map(ch => {
                const relCount = relationships.filter(r => r.from_id === ch.id || r.to_id === ch.id).length
                return (
                  <Col key={ch.id} xs={24} sm={12} lg={8} xl={6}>
                    <CharacterCard
                      character={ch} relationCount={relCount}
                      onClick={() => openProjectEdit(ch)}
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
          {organizations.map(org => (
            <Col key={org.id} xs={24} sm={12} lg={8} xl={6}>
              <OrganizationCard organization={org} onClick={() => { setModalOrg(org); setEditOrg({ ...org }) }} />
            </Col>
          ))}
        </Row>
      ),
    },
    {
      key: 'rels',
      label: <span><LinkOutlined /> 关系图 ({relationships.length})</span>,
      children: (
        <>
          <div style={{ display: 'flex', gap: 8, alignItems: 'center', marginBottom: 12, flexWrap: 'wrap' }}>
            <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11 }}>起点</Typography.Text>
            <Select size="small" value={relFromId || undefined} onChange={setRelFromId} style={{ width: 130 }} placeholder="选择角色"
              options={characters.map(c => ({ value: c.id, label: c.name }))} />
            <Button size="small" icon={<LinkOutlined />} disabled={!relFromId}
              onClick={() => setRelModalOpen(true)}>添加关系</Button>
          </div>
          {relationships.length === 0 ? (
            <Empty description="暂无关系" image={Empty.PRESENTED_IMAGE_SIMPLE} style={{ marginTop: 60 }} />
          ) : (
            <div style={{ height: 'calc(100% - 56px)', minHeight: 360, display: 'flex', flexDirection: 'column', gap: 10 }}>
              <div style={{ flex: 1, overflow: 'hidden' }}>
                <RelationGraph characters={characters} organizations={organizations} relationships={relationships} />
              </div>
              <div style={{ maxHeight: 150, overflowY: 'auto', borderTop: '1px solid var(--border-subtle)', paddingTop: 8 }}>
                {relationships.map((r, i) => (
                  <div key={i} style={{ display: 'flex', alignItems: 'center', gap: 8, fontSize: 11.5, padding: '3px 0', color: C('color-text-secondary') }}>
                    <span>{getCharName(r.from_id)}</span>
                    <Tag style={{ margin: 0 }}>{r.relation_type}</Tag>
                    <span>{getCharName(r.to_id)}</span>
                    <span style={{ flex: 1 }} />
                    <Button size="small" type="text" danger icon={<DeleteOutlined />} title="删除关系"
                      onClick={() => handleDeleteRel(r)} style={{ fontSize: 11, padding: '0 4px' }} />
                  </div>
                ))}
              </div>
            </div>
          )}
        </>
      ),
    },
  ]

  return (
    <div style={{ height: 'calc(100vh - 112px)', display: 'flex', flexDirection: 'column', gap: 12 }}>
      {/* 旧数据迁移提示：小说只引用角色库，旧项目角色需一次性入库 */}
      {unimported.length > 0 && (
        <div style={{
          display: 'flex', alignItems: 'center', gap: 10, padding: '8px 12px', borderRadius: 10, flexShrink: 0,
          background: 'rgba(250,204,21,0.08)', border: '1px solid rgba(250,204,21,0.25)', fontSize: 12,
        }}>
          <span style={{ color: C('color-text') }}>检测到 {unimported.length} 个旧项目角色尚未进入角色库（{unimported.slice(0, 3).map(c => c.name).join('、')}…）。迁入后本书只引用角色库，不再自己生成角色。</span>
          <Button size="small" icon={<ImportOutlined />} onClick={handleImportLegacy}>一次性迁移</Button>
        </div>
      )}

      <div style={{ display: 'flex', justifyContent: 'flex-end', alignItems: 'center', flexShrink: 0 }}>
        <Space>
          <Button size="small" icon={<TeamOutlined />} onClick={navigateToCharacterLib}
            style={{ background: 'var(--bg-elevated)', border: '1px solid rgba(96, 165, 250, 0.3)', color: '#60a5fa', borderRadius: 'var(--radius-md)' }}>去角色库</Button>
          <Button size="small" icon={<SyncOutlined />} onClick={handleSync} loading={syncing}
            disabled={unimported.length > 0}
            style={{ background: 'var(--bg-elevated)', border: '1px solid rgba(96, 165, 250, 0.3)', color: '#60a5fa', borderRadius: 'var(--radius-md)' }}>同步</Button>
          <Button size="small" type="primary" icon={<ThunderboltOutlined />} onClick={() => setDrawOpen(true)}
            style={{ boxShadow: 'var(--shadow-glow)', borderRadius: 'var(--radius-md)' }}>抽卡</Button>
        </Space>
      </div>

      <div style={{ flex: 1, overflow: 'auto' }}>
        <Tabs className="novel-tabs" items={tabItems} size="small" style={{ color: C('color-text'), flex: 1, minHeight: 0 }} tabBarStyle={{ borderColor: C('color-border') }} />
      </div>

      {renderProjectEditModal()}
      {renderDrawModal()}

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
        editForm={relFromId ? { id: relFromId } : null}
        relTargetId={relTargetId}
        onRelTargetChange={setRelTargetId}
        relType={relType}
        onRelTypeChange={setRelType}
        onAdd={handleAddRel}
      />

      {/* 剧照全屏（仅查看，生成请去角色库） */}
      {portraitFullscreen && (
        <PortraitLightbox imageUrl={portraitFullscreen} onClose={() => setPortraitFullscreen('')} />
      )}
    </div>
  )
}

export default CharacterPage
