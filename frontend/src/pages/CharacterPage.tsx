// CharacterPage.tsx — 小说角色面板（单向使用角色库）
// 约束：小说只引用角色库的角色，不自行生成、不回写全局角色；
// 面板内可改的只有项目内覆盖（定位 / 弧线状态 / 状态），全局设定一律去角色库。
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import {
  Typography, Empty, Button, Input, Modal, InputNumber, Drawer,
  Select, message, Tabs, Tag, Switch, Popconfirm,
} from 'antd'
import {
  ThunderboltOutlined, PlusOutlined, ExperimentOutlined, CameraOutlined, MergeCellsOutlined,
  UserOutlined, ApartmentOutlined, LinkOutlined, TeamOutlined,
  ImportOutlined, SyncOutlined, EditOutlined, SwapOutlined, DeleteOutlined,
  GlobalOutlined, BookOutlined, CloseOutlined,
} from '@ant-design/icons'
import RelationGraph from '../components/RelationGraph'
import type { CharacterData, OrganizationData, RelationshipData } from '../types'

/** 提取错误消息（unknown 收窄；无 message 用 fallback） */
function errText(err: unknown, fallback: string): string {
  return (err instanceof Error && err.message) || fallback
}
import { useAppStore } from '../stores/appStore'
import { C, ROLE_COLORS as roleColors, ROLE_LABELS as roleLabels } from '../utils/theme'
import { CHARACTER_STATUS_OPTIONS, characterStatusLabel, normalizeCharacterStatus } from '../utils/characterStatus'
import CharacterCard from '../components/novel/character/CharacterCard'
import OrganizationCard from '../components/novel/character/OrganizationCard'
import RelationshipModal from '../components/novel/character/RelationshipModal'
import OrganizationEditModal from '../components/novel/character/OrganizationEditModal'
import PortraitLightbox from '../components/novel/character/PortraitLightbox'
import { PortraitImg } from '../components/characterlib/PortraitImg'
import {
  getCharacters, saveOrganization, deleteOrganization, toggleOrgMember,
  saveRelationship, deleteRelationship,
  generateCharacterFill, generateCharacterPortrait, mergeCharacters,
} from '../components/novel/api/character'
import {
  listProjectCharacters, associateToProject, dissociateFromProject,
  syncProjectCharacters, importProjectCharacters, drawRandom, setProjectState,
  type LibraryCharacter,
} from '../api/characterlib'
import './character-page.css'

// 角色状态枚举统一来自 utils/characterStatus（T6-7.5 状态收敛：非法值回退默认）
const statusOptions = CHARACTER_STATUS_OPTIONS
const roleOptions = [
  { value: 'protagonist', label: '主角' }, { value: 'antagonist', label: '反派' },
  { value: 'supporting', label: '配角' }, { value: 'minor', label: '龙套' },
]

const CHAR_FILTER_KEY = 'gaea.novel.charFilters.'

interface CharFilterState {
  gender: string
  role: string
  status: string
  org: string
}

function readCharFilters(projectPath: string): CharFilterState {
  try {
    const raw = localStorage.getItem(CHAR_FILTER_KEY + projectPath)
    if (!raw) return { gender: '', role: '', status: '', org: '' }
    const value = JSON.parse(raw) as CharFilterState
    return value || { gender: '', role: '', status: '', org: '' }
  } catch {
    return { gender: '', role: '', status: '', org: '' }
  }
}

function writeCharFilters(projectPath: string, state: CharFilterState) {
  try {
    if (projectPath) localStorage.setItem(CHAR_FILTER_KEY + projectPath, JSON.stringify(state))
  } catch { /* ignore */ }
}

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
  const [loading, setLoading] = useState(true)

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
  const [filling, setFilling] = useState(false)
  const [genPortrait, setGenPortrait] = useState(false)
  const [mergeOpen, setMergeOpen] = useState(false)
  const [mergeTargetId, setMergeTargetId] = useState('')
  const [peRole, setPeRole] = useState('')
  const [peArc, setPeArc] = useState('')
  const [peStatus, setPeStatus] = useState('')

  const projectPath = useAppStore(s => s.projectPath)
  const dataLoadToken = useRef(0)
  const refsLoadToken = useRef(0)

  // 项目切换时恢复/重置筛选条件
  useEffect(() => {
    const saved = readCharFilters(projectPath)
    setFilterGender(saved.gender || '')
    setFilterRole(saved.role || '')
    setFilterStatus(saved.status || '')
    setFilterOrg(saved.org || '')
  }, [projectPath])

  // 筛选变化按项目记忆
  useEffect(() => {
    if (!projectPath) return
    writeCharFilters(projectPath, {
      gender: filterGender,
      role: filterRole,
      status: filterStatus,
      org: filterOrg,
    })
  }, [projectPath, filterGender, filterRole, filterStatus, filterOrg])

  const loadData = useCallback(async () => {
    const token = ++dataLoadToken.current
    const requestedPath = useAppStore.getState().projectPath
    setLoading(true)
    try {
      const data = await getCharacters()
      if (token !== dataLoadToken.current || requestedPath !== useAppStore.getState().projectPath) return
      setCharacters(data.characters || [])
      setOrganizations(data.organizations || [])
      setRelationships(data.relationships || [])
    } catch (err) { console.error('[CharacterPage] loadData:', err) }
    finally {
      if (token === dataLoadToken.current) setLoading(false)
    }
  }, [])

  const loadRefs = useCallback(async () => {
    const token = ++refsLoadToken.current
    if (!projectPath) { setProjectRefs(new Set()); return }
    try {
      const refs = await listProjectCharacters()
      if (token !== refsLoadToken.current || projectPath !== useAppStore.getState().projectPath) return
      setProjectRefs(new Set(refs.map(r => r.characterId)))
    } catch (_) {
      if (token === refsLoadToken.current && projectPath === useAppStore.getState().projectPath) setProjectRefs(new Set())
    }
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

  // 角色库侧移出本书后，已挂载的本面板保持同步刷新（MainLayout 不销毁组件）
  useEffect(() => {
    const handler = () => { refreshAll() }
    window.addEventListener('gaea-project-chars-changed', handler)
    return () => window.removeEventListener('gaea-project-chars-changed', handler)
  }, [refreshAll])

  // ── 抽卡：从角色库随机抽取，加入当前项目 ──
  const handleDraw = async () => {
    setDrawLoading(true)
    try {
      const items = await drawRandom(drawCount, drawGender, drawTags.trim(), drawChatOnly)
      setDrawResult(items || [])
      if (!items?.length) message.info('没有抽到符合条件的角色，换个条件试试')
    } catch (err: unknown) {
      message.error(errText(err, '抽卡失败'))
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
    } catch (err: unknown) {
      message.error(errText(err, '加入失败'))
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
    } catch (err: unknown) {
      message.error(errText(err, '加入失败'))
    }
  }

  // ── 项目内状态编辑（只写关联表）──
  const openProjectEdit = (ch: CharacterData) => {
    setProjectEdit(ch)
    setPeRole(ch.role_type || 'supporting')
    setPeArc(ch.arc || '')
    setPeStatus(normalizeCharacterStatus(ch.status))
  }

  const handleSaveProjectState = async () => {
    if (!projectEdit) return
    try {
      await setProjectState(projectEdit.id, peRole, peArc, peStatus)
      await syncProjectCharacters()
      await loadData()
      message.success(`已更新「${projectEdit.name}」在本书的状态（全局角色未动）`)
      setProjectEdit(null)
    } catch (err: unknown) {
      message.error(errText(err, '保存失败'))
    }
  }

  const handleRemoveFromProject = async (ch: CharacterData) => {
    try {
      await dissociateFromProject(ch.id)
      await syncProjectCharacters()
      await refreshAll()
      message.success(`「${ch.name}」已从本书移除（角色保留在角色库）`)
      setProjectEdit(null)
    } catch (err: unknown) {
      message.error(errText(err, '移除失败'))
    }
  }

  const handleSync = async () => {
    setSyncing(true)
    try {
      await syncProjectCharacters()
      await loadData()
      message.success('已把本书引用的角色同步到 characters.json')
    } catch (err: unknown) {
      message.error(errText(err, '同步失败'))
    } finally {
      setSyncing(false)
    }
  }

  const handleImportLegacy = async () => {
    try {
      const n = await importProjectCharacters()
      await loadRefs()
      message.success(`已将 ${n} 个旧项目角色一次性迁入角色库（后续本书只引用角色库）`)
    } catch (err: unknown) {
      message.error(errText(err, '迁移失败'))
    }
  }

  // ── 章节捕获角色的补齐 / 剧照 / 合并（未入库时可用，只写本书） ──
  const handleProjectFill = async (ch: CharacterData) => {
    if (filling) return
    setFilling(true)
    try {
      const updated = await generateCharacterFill(ch)
      setProjectEdit(updated)
      await loadData()
      message.success('已补齐空缺字段（只写本书，未动角色库）')
    } catch (err: unknown) {
      message.error(errText(err, '补齐失败'))
    } finally {
      setFilling(false)
    }
  }

  const handleProjectPortrait = async (ch: CharacterData) => {
    if (genPortrait) return
    setGenPortrait(true)
    try {
      await generateCharacterPortrait(ch.id)
      await loadData()
      message.success('剧照已生成')
    } catch (err: unknown) {
      message.error(errText(err, '剧照生成失败'))
    } finally {
      setGenPortrait(false)
    }
  }

  const handleMergeConfirm = async () => {
    if (!projectEdit || !mergeTargetId) return
    try {
      // 当前角色（A）并入目标角色（B），保留 B
      await mergeCharacters(mergeTargetId, projectEdit.id)
      setMergeOpen(false)
      setMergeTargetId('')
      setProjectEdit(null)
      await refreshAll()
      message.success('已合并：空缺信息已补充，关系与组织引用已重定向')
    } catch (err: unknown) {
      message.error(errText(err, '合并失败'))
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

  // ── 角色详情抽屉：左全局信息（只读） + 右本书局部设定（可编辑） ──
  const renderProjectDetail = () => {
    if (!projectEdit) return null
    const ch = projectEdit
    const orgs = organizations.filter(o => o.members?.includes(ch.id))
    const relCount = relationships.filter(r => r.from_id === ch.id || r.to_id === ch.id).length
    const roleLabel = roleLabels[ch.role_type] || ch.role_type
    const roleColor = roleColors[ch.role_type] || 'default'
    // 非法状态回退默认（'Alive'），不泄露原始英文串
    const statusLabel = characterStatusLabel(ch.status)
    const genderText = ch.gender === 'male' ? '♂ 男' : ch.gender === 'female' ? '♀ 女' : (ch.gender || '未设')

    const globalFields: Array<{ label: string; value: string }> = [
      { label: '性格', value: ch.personality },
      { label: '背景', value: ch.background },
      { label: '外貌', value: ch.appearance },
      { label: '身材', value: ch.figure },
      { label: '动机', value: ch.motivation },
      { label: '全局弧线', value: ch.arc },
    ]

    return (
      <Drawer
        open={!!projectEdit}
        onClose={() => setProjectEdit(null)}
        width={780}
        closable={false}
        className="char-detail-drawer"
        footer={
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <Popconfirm title={`把「${ch.name}」从本书移除？角色保留在角色库`}
              okText="移除" cancelText="取消" onConfirm={() => handleRemoveFromProject(ch)}>
              <Button danger size="small" icon={<DeleteOutlined />}>移出本书</Button>
            </Popconfirm>
            <div style={{ flex: 1 }} />
            <Button size="small" onClick={() => setProjectEdit(null)}>取消</Button>
            <Button size="small" type="primary" icon={<EditOutlined />} onClick={handleSaveProjectState}>
              保存本书状态
            </Button>
          </div>
        }
      >
        <div className="char-detail">
          {/* 头部：剧照 + 名称 + 元信息 + 跳转角色库 */}
          <div className="char-detail-head">
            {ch.portrait_url
              ? <PortraitImg className="char-detail-avatar" src={ch.portrait_url} alt={ch.name} />
              : <div className="char-detail-avatar char-detail-avatar--empty"><UserOutlined /></div>}
            <div className="char-detail-head-main">
              <div className="char-detail-name">{ch.name}</div>
              <div className="char-detail-chips">
                <Tag color={roleColor}>{roleLabel}</Tag>
                <Tag>{statusLabel}</Tag>
                <span className="char-detail-meta">{genderText}{ch.age ? ` · ${ch.age}岁` : ''}</span>
                <span className="char-detail-meta"><LinkOutlined aria-hidden /> {relCount} 个关系</span>
                {orgs.length > 0 && (
                  <span className="char-detail-meta"><ApartmentOutlined aria-hidden /> {orgs.map(o => o.name).join('、')}</span>
                )}
              </div>
            </div>
            <div className="char-detail-head-actions">
              {!projectRefs.has(ch.id) && (
                <>
                  <Button size="small" icon={<ExperimentOutlined />} loading={filling}
                    onClick={() => handleProjectFill(ch)}>AI 补齐</Button>
                  <Button size="small" icon={<CameraOutlined />} loading={genPortrait}
                    onClick={() => handleProjectPortrait(ch)}>生成剧照</Button>
                  <Button size="small" icon={<MergeCellsOutlined />}
                    onClick={() => { setMergeTargetId(''); setMergeOpen(true) }}>合并</Button>
                </>
              )}
              <Button size="small" icon={<TeamOutlined />} onClick={() => { setProjectEdit(null); navigateToCharacterLib() }}>
                在角色库编辑
              </Button>
            </div>
          </div>

          {/* 双栏：全局信息 / 本书局部设定 */}
          <div className="char-detail-cols">
            <section className="char-detail-col">
              <div className="char-detail-section-title">
                <GlobalOutlined />全局信息
                <Tag className="char-detail-section-tag">只读 · 来自角色库</Tag>
              </div>
              <div className="char-detail-fields">
                {globalFields.map(f => (
                  <div key={f.label} className="char-detail-field">
                    <div className="char-detail-field-label">{f.label}</div>
                    <div className={`char-detail-field-value${f.value ? '' : ' is-empty'}`}>
                      {f.value || '（未填写）'}
                    </div>
                  </div>
                ))}
              </div>
            </section>

            <section className="char-detail-col char-detail-col--local">
              <div className="char-detail-section-title">
                <BookOutlined />本书局部设定
                <Tag className="char-detail-section-tag">仅本小说生效</Tag>
              </div>
              <div className="char-detail-form">
                <div className="char-detail-form-item">
                  <label>本书定位</label>
                  <Select size="small" value={peRole} onChange={setPeRole} style={{ width: '100%' }} options={roleOptions} />
                </div>
                <div className="char-detail-form-item">
                  <label>本书状态</label>
                  <Select size="small" value={peStatus} onChange={setPeStatus} style={{ width: '100%' }} options={statusOptions} />
                </div>
                <div className="char-detail-form-item">
                  <label>本书弧线状态（覆盖全局弧线）</label>
                  <Input.TextArea size="small" rows={4} value={peArc} onChange={e => setPeArc(e.target.value)}
                    placeholder="如：第一卷结尾黑化；第二卷开始救赎…"
                    style={{ fontSize: 12.5 }} />
                </div>
                <div className="char-detail-hint">
                  未填写项沿用全局值；这里的修改只影响本书，不会改动角色库。
                </div>
              </div>
            </section>
          </div>
        </div>
      </Drawer>
    )
  }

  // ── 抽卡弹窗 ──
  const renderDrawModal = () => (
    <Modal open={drawOpen} title="从角色库抽卡" onCancel={() => setDrawOpen(false)}
      footer={null} width={680}
      // WebView2 冻结 rAF 时关闭动画不结束会残留全屏 wrap 拦截点击：关闭即卸载。
      destroyOnHidden transitionName="" maskTransitionName="">
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
        <div className="char-grid char-grid--slim">
          {drawResult.map(c => {
            const added = projectRefs.has(c.id)
            return (
              <div key={c.id} className="char-draw-card">
                {c.portraitUrl
                  ? <PortraitImg className="char-draw-avatar" src={c.portraitUrl} alt={c.name} />
                  : <div className="char-draw-fallback"><UserOutlined /></div>}
                <div className="char-draw-info">
                  <div className="char-draw-name">{c.name}</div>
                  <div className="char-draw-sub">
                    {c.roleType ? roleOptions.find(r => r.value === c.roleType)?.label || c.roleType : '未设定位'}
                    {c.tags?.length ? ` · ${c.tags.slice(0, 2).join('、')}` : ''}
                  </div>
                </div>
                {added
                  ? <Tag style={{ margin: 0 }}>已加入</Tag>
                  : <Button size="small" type="primary" icon={<SwapOutlined />} onClick={() => handleAddDrawn(c)}>加入</Button>}
              </div>
            )
          })}
        </div>
      )}
    </Modal>
  )

  // ── 合并角色弹窗：同一人的不同称呼合并为一张卡 ──
  const renderMergeModal = () => (
    <Modal open={mergeOpen} title={`合并「${projectEdit?.name || ''}」到其他角色`}
      onCancel={() => { setMergeOpen(false); setMergeTargetId('') }}
      onOk={handleMergeConfirm} okText="合并" okButtonProps={{ disabled: !mergeTargetId }}
      width={460}
      destroyOnHidden transitionName="" maskTransitionName="">
      <Typography.Paragraph type="secondary" style={{ fontSize: 12, marginTop: 0 }}>
        用于同一人不同称呼的角色卡（如只是换了名字）。合并后保留目标角色，当前角色的空缺信息会被补充，关系与组织引用自动重定向。
      </Typography.Paragraph>
      <Select
        placeholder="选择要保留的角色…"
        style={{ width: '100%' }}
        value={mergeTargetId || undefined}
        onChange={setMergeTargetId}
        options={characters
          .filter(c => c.id !== projectEdit?.id)
          .map(c => ({ value: c.id, label: `${c.name}${c.role_type ? `（${roleLabels[c.role_type] || c.role_type}）` : ''}` }))}
      />
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
            <div className="char-toolbar">
              <Typography.Text className="char-toolbar-label">筛选</Typography.Text>
              <Select size="small" value={filterGender} onChange={setFilterGender} style={{ width: 90 }}
                options={[{ label: '全部性别', value: '' }, { label: '♂ 男', value: 'male' }, { label: '♀ 女', value: 'female' }]} />
              <Select size="small" value={filterRole} onChange={setFilterRole} style={{ width: 100 }}
                options={[{ label: '全部阵营', value: '' }, ...roleOptions]} />
              <Select size="small" value={filterStatus} onChange={setFilterStatus} style={{ width: 100 }}
                options={[{ label: '全部状态', value: '' }, ...statusOptions]} />
              <Select size="small" value={filterOrg} onChange={setFilterOrg} style={{ width: 120 }}
                options={[{ label: '全部组织', value: '' }, ...organizations.map(o => ({ label: o.name, value: o.id }))]} />
              {(filterGender || filterRole || filterStatus || filterOrg) && (
                <Button size="small" icon={<CloseOutlined />} onClick={() => { setFilterGender(''); setFilterRole(''); setFilterStatus(''); setFilterOrg('') }}
                  style={{ fontSize: 10, padding: '0 6px' }}>清除</Button>
              )}
            </div>
          )}
          {loading ? (
            <div className="char-grid">
              {Array.from({ length: 6 }).map((_, i) => (
                <div key={i} className="char-card-skeleton">
                  <div className="char-card-skeleton--portrait" />
                  <div className="char-card-skeleton--line" />
                  <div className="char-card-skeleton--line short" />
                </div>
              ))}
            </div>
          ) : characters.length === 0 ? (
            <Empty description="本书还没有角色，去角色库抽卡吧" image={Empty.PRESENTED_IMAGE_SIMPLE} style={{ marginTop: 80 }}>
              <Button type="primary" icon={<ThunderboltOutlined />} onClick={() => setDrawOpen(true)}>抽卡</Button>
            </Empty>
          ) : filteredCharacters.length === 0 ? (
            <div className="char-empty" style={{ textAlign: 'center', color: C('color-text-secondary'), fontSize: 12 }}>
              没有匹配筛选条件的角色
            </div>
          ) : (
            <div className="char-grid">
              {filteredCharacters.map(ch => {
                const relCount = relationships.filter(r => r.from_id === ch.id || r.to_id === ch.id).length
                return (
                  <CharacterCard
                    key={ch.id}
                    character={ch} relationCount={relCount}
                    onClick={() => openProjectEdit(ch)}
                    onPortraitFullscreen={setPortraitFullscreen}
                  />
                )
              })}
            </div>
          )}
        </>
      ),
    },
    {
      key: 'orgs',
      label: <span><ApartmentOutlined /> 组织 ({organizations.length})</span>,
      children: organizations.length === 0 ? (
        <Empty description="暂无组织，新建一个试试" image={Empty.PRESENTED_IMAGE_SIMPLE} style={{ marginTop: 80 }}>
          <Button icon={<PlusOutlined />} onClick={handleNewOrg}>新建组织</Button>
        </Empty>
      ) : (
        <div className="char-grid char-grid--slim">
          {organizations.map(org => (
            <OrganizationCard key={org.id} organization={org} onClick={() => { setModalOrg(org); setEditOrg({ ...org }) }} />
          ))}
        </div>
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
              onClick={() => { setRelTargetId(''); setRelType('friend'); setRelModalOpen(true) }}>添加关系</Button>
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
    <div className="char-panel-root">
      {/* 旧数据迁移提示：小说只引用角色库，旧项目角色需一次性入库 */}
      {unimported.length > 0 && (
        <div className="char-migration-banner">
          <span style={{ color: C('color-text') }}>检测到 {unimported.length} 个旧项目角色尚未进入角色库（{unimported.slice(0, 3).map(c => c.name).join('、')}…）。迁入后本书只引用角色库，不再自己生成角色。</span>
          <Button size="small" type="primary" ghost icon={<ImportOutlined />} onClick={handleImportLegacy}>一次性迁移</Button>
        </div>
      )}

      {/* 头部信息栏 */}
      <div className="char-panel-header">
        <span className="char-panel-title"><TeamOutlined />角色面板</span>
        <div className="char-panel-stats">
          <span className="char-panel-stat">角色 <strong>{characters.length}</strong></span>
          <span className="char-panel-stat">组织 <strong>{organizations.length}</strong></span>
          <span className="char-panel-stat">关系 <strong>{relationships.length}</strong></span>
          {unimported.length > 0 && (
            <span className="char-panel-stat is-warn">未入库 <strong>{unimported.length}</strong></span>
          )}
        </div>
        <div className="char-panel-actions">
          <Button size="small" icon={<TeamOutlined />} onClick={navigateToCharacterLib}>去角色库</Button>
          <Button size="small" icon={<SyncOutlined />} onClick={handleSync} loading={syncing} disabled={unimported.length > 0}>同步</Button>
          <Button size="small" type="primary" icon={<ThunderboltOutlined />} onClick={() => setDrawOpen(true)}>抽卡</Button>
        </div>
      </div>

      {/* 主面板 */}
      <div className="char-panel-main">
        <Tabs className="char-tabs novel-tabs" items={tabItems} size="small" style={{ color: C('color-text'), flex: 1, minHeight: 0 }} tabBarStyle={{ borderColor: C('color-border') }} />
      </div>

      {renderProjectDetail()}
      {renderDrawModal()}
      {renderMergeModal()}

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
