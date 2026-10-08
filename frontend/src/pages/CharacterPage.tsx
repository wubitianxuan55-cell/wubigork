// CharacterPage.tsx — 小说角色面板（单向使用角色库）
// 约束：小说只引用角色库的角色，不自行生成、不回写全局角色；
// 面板内可改的只有项目内覆盖（定位 / 弧线状态 / 状态），全局设定一律去角色库。
// 2026-10-04 hook 分域首刀：状态与 handler 群迁 ./character/ 七个分域 hook
// （useCharacterData/useCharFilters/useProjectCharacterEdit/useProtagonistRelations/
// useCharDraw/useLegacyWriteBack/useOrgRelation），本文件只做接线与渲染，
// JSX 与渲染函数逐字节保留（拆分等价证明 = 16 例金样零编辑复绿）。
import React, { useMemo, useState } from 'react'
import {
  Typography, Button, Input, Modal, InputNumber, Drawer,
  Select, Tabs, Tag, Switch, Popconfirm, Checkbox, Dropdown,
} from 'antd'
import {
  ThunderboltOutlined, PlusOutlined, ExperimentOutlined, CameraOutlined, MergeCellsOutlined,
  UserOutlined, ApartmentOutlined, LinkOutlined, TeamOutlined,
  ImportOutlined, SyncOutlined, EditOutlined, SwapOutlined, DeleteOutlined,
  GlobalOutlined, BookOutlined, CloseOutlined, DownOutlined,
} from '@ant-design/icons'
import RelationGraph from '../components/RelationGraph'
import V3Empty from '../components/V3Empty'
import { C, ROLE_COLORS as roleColors, ROLE_LABELS as roleLabels } from '../utils/theme'
import { CHARACTER_STATUS_OPTIONS, characterStatusLabel } from '../utils/characterStatus'
import CharacterCard from '../components/novel/character/CharacterCard'
import OrganizationCard from '../components/novel/character/OrganizationCard'
import RelationshipModal from '../components/novel/character/RelationshipModal'
import OrganizationEditModal from '../components/novel/character/OrganizationEditModal'
import PortraitLightbox from '../components/novel/character/PortraitLightbox'
import { PortraitImg } from '../components/characterlib/PortraitImg'
import type { ImportFieldConflict } from '../api/characterlib'
import { useCharacterData } from './character/useCharacterData'
import { useCharFilters } from './character/useCharFilters'
import { useProjectCharacterEdit } from './character/useProjectCharacterEdit'
import { useProtagonistRelations } from './character/useProtagonistRelations'
import { useCharDraw } from './character/useCharDraw'
import { useLegacyWriteBack } from './character/useLegacyWriteBack'
import { useOrgRelation } from './character/useOrgRelation'
import './character-page.css'

// 角色状态枚举统一来自 utils/characterStatus（T6-7.5 状态收敛：非法值回退默认）
const statusOptions = CHARACTER_STATUS_OPTIONS
const roleOptions = [
  { value: 'protagonist', label: '主角' }, { value: 'antagonist', label: '反派' },
  { value: 'supporting', label: '配角' }, { value: 'minor', label: '龙套' },
]

function navigateToCharacterLib() {
  window.dispatchEvent(new CustomEvent('navigate', { detail: { page: 'characterlib' } }))
}

const CharacterPage: React.FC = () => {
  const [portraitFullscreen, setPortraitFullscreen] = useState('')
  const data = useCharacterData()
  const filters = useCharFilters()
  const edit = useProjectCharacterEdit(data)
  const relations = useProtagonistRelations({
    characters: data.characters,
    refreshAll: data.refreshAll,
    projectEdit: edit.projectEdit,
    reloadProjectEdit: edit.reloadProjectEdit,
  })
  const draw = useCharDraw({ projectRefs: data.projectRefs, refreshAll: data.refreshAll })
  const writeBack = useLegacyWriteBack({ loadRefs: data.loadRefs })
  const orgRel = useOrgRelation({
    characters: data.characters,
    organizations: data.organizations,
    loadData: data.loadData,
  })

  const { characters, organizations, relationships, projectRefs, loading, setLoading, loadError, unimported, syncing, loadData, handleSync } = data
  const { filterGender, setFilterGender, filterRole, setFilterRole, filterStatus, setFilterStatus, filterOrg, setFilterOrg } = filters
  const {
    projectEdit, setProjectEdit, openProjectEdit,
    peRole, setPeRole, peArc, setPeArc, peStatus, setPeStatus,
    peCareerMain, setPeCareerMain, peCareerMainStage, setPeCareerMainStage,
    peNewSub, setPeNewSub, peNewSubStage, setPeNewSubStage,
    filling, genPortrait, mergeOpen, setMergeOpen, mergeTargetId, setMergeTargetId,
    handleSetMainCareer, handleRemoveMainCareer, handleAddSubCareer, handleRemoveSubCareer,
    handleSaveProjectState, handleRemoveFromProject,
    handleProjectFill, handleProjectPortrait, handleMergeConfirm,
  } = edit
  const { relBusy, relProgress, runProtagonistRelations, handleGenRelations } = relations
  const {
    drawOpen, setDrawOpen,
    drawCount, setDrawCount,
    drawGender, setDrawGender,
    drawTags, setDrawTags,
    drawChatOnly, setDrawChatOnly,
    drawResult, drawLoading,
    handleDraw, handleAddDrawn, handleAddAllDrawn,
  } = draw
  const { wbOpen, setWbOpen, wbBusy, wbPreview, wbChecked, setWbChecked, handleImportLegacy, doWriteBack, collectOverwrites } = writeBack
  const {
    modalOrg, setModalOrg, editOrg, setEditOrg,
    relTargetId, setRelTargetId, relFromId, setRelFromId,
    relType, setRelType, relModalOpen, setRelModalOpen,
    getCharName, handleNewOrg, handleSaveOrg, handleDeleteOrg,
    handleAddRel, handleDeleteRel,
  } = orgRel

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
                {ch.protagonist_relation && (
                  <Tag color="gold" data-testid="char-detail-relation">与主角：{ch.protagonist_relation}</Tag>
                )}
                <span className="char-detail-meta">{genderText}{ch.age ? ` · ${ch.age}岁` : ''}</span>
                <span className="char-detail-meta"><LinkOutlined aria-hidden /> {relCount} 个关系</span>
                {orgs.length > 0 && (
                  <span className="char-detail-meta"><ApartmentOutlined aria-hidden /> {orgs.map(o => o.name).join('、')}</span>
                )}
              </div>
            </div>
            <div className="char-detail-head-actions">
              {ch.role_type !== 'protagonist' && (
                <Button size="small" icon={<UserOutlined />} loading={relBusy}
                  data-testid="char-detail-gen-relation"
                  onClick={() => void runProtagonistRelations('one', ch.name)}
                  title="AI 随机生成该角色与主角的关系短语（覆盖重写；只写本书）">
                  AI 主角关系
                </Button>
              )}
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
                {/* 阶段四出口③：关联即快照——项目内弧线/状态以本书为准 */}
                <Tag className="char-detail-section-tag">仅本小说生效 · 与角色库互不影响</Tag>
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

            <section className="char-detail-col char-detail-col--local">
              <div className="char-detail-section-title">
                <ThunderboltOutlined />职业体系
                <Tag className="char-detail-section-tag">t5 状态机 · 即时保存 · 生成时自动携带</Tag>
              </div>
              <div className="char-detail-form">
                <div className="char-detail-form-item">
                  <label>主职业{ch.main_career_id ? `（当前：${ch.main_career_id}·${ch.main_career_stage || 1}阶）` : '（未设）'}</label>
                  <div style={{ display: 'flex', gap: 6 }}>
                    <Input size="small" value={peCareerMain} onChange={e => setPeCareerMain(e.target.value)}
                      placeholder="如：剑修（名称即 ID）" style={{ flex: 1 }} />
                    <InputNumber size="small" min={1} max={99} value={peCareerMainStage}
                      onChange={v => setPeCareerMainStage(v || 1)} addonAfter="阶" style={{ width: 96 }} />
                    <Button size="small" onClick={handleSetMainCareer}>设定</Button>
                    {ch.main_career_id && (
                      <Popconfirm title={`移除主职业「${ch.main_career_id}」？`} okText="移除" cancelText="取消"
                        onConfirm={handleRemoveMainCareer}>
                        <Button size="small" danger icon={<DeleteOutlined />} />
                      </Popconfirm>
                    )}
                  </div>
                </div>
                <div className="char-detail-form-item">
                  <label>副职业（最多 2 个；阶段由章节分析自动推进）</label>
                  {(ch.sub_careers || []).map(sc => (
                    <div key={sc.career_id} style={{ display: 'flex', gap: 6, marginBottom: 4, alignItems: 'center' }}>
                      <Tag color="blue">{sc.career_name || sc.career_id}·{sc.stage}阶</Tag>
                      <Popconfirm title={`移除副职业「${sc.career_name || sc.career_id}」？`} okText="移除" cancelText="取消"
                        onConfirm={() => handleRemoveSubCareer(sc.career_id)}>
                        <Button size="small" type="text" danger icon={<CloseOutlined />} />
                      </Popconfirm>
                    </div>
                  ))}
                  <div style={{ display: 'flex', gap: 6 }}>
                    <Input size="small" value={peNewSub} onChange={e => setPeNewSub(e.target.value)}
                      placeholder="如：炼丹师" style={{ flex: 1 }} />
                    <InputNumber size="small" min={1} max={99} value={peNewSubStage}
                      onChange={v => setPeNewSubStage(v || 1)} addonAfter="阶" style={{ width: 96 }} />
                    <Button size="small" onClick={handleAddSubCareer}
                      disabled={(ch.sub_careers || []).length >= 2}>添加</Button>
                  </div>
                </div>
                <div className="char-detail-hint">
                  写入本书 characters.json，与章节分析差分同库——章节分析会按剧情自动推进阶段，手工设置定初值或纠偏；章节生成时职业与状态自动注入 Prompt。
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
        <V3Empty description="抽到的角色会显示在这里，点「加入本书」引用" style={{ marginTop: 40 }} />
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
          ) : loadError && characters.length === 0 ? (
            // 诚实错误态：失败 ≠ 空库（对齐 HomePage shelf-load-error 口径）
            <V3Empty description={`角色数据读取失败：${loadError}`} style={{ marginTop: 80 }}>
              <Button icon={<SyncOutlined />} onClick={() => { setLoading(true); loadData().finally(() => setLoading(false)) }}>重试</Button>
            </V3Empty>
          ) : characters.length === 0 ? (
            <V3Empty description="本书还没有角色，去角色库抽卡吧" style={{ marginTop: 80 }}>
              <Button type="primary" icon={<ThunderboltOutlined />} onClick={() => setDrawOpen(true)}>抽卡</Button>
            </V3Empty>
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
        <V3Empty description="暂无组织，新建一个试试" style={{ marginTop: 80 }}>
          <Button icon={<PlusOutlined />} onClick={handleNewOrg}>新建组织</Button>
        </V3Empty>
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
            <V3Empty description="暂无关系" style={{ marginTop: 60 }} />
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
                    <Popconfirm
                      title="删除这条关系？"
                      okText="删除"
                      cancelText="取消"
                      okButtonProps={{ danger: true }}
                      onConfirm={() => void handleDeleteRel(r)}
                    >
                      <Button size="small" type="text" danger icon={<DeleteOutlined />} title="删除关系"
                        style={{ fontSize: 11, padding: '0 4px' }} />
                    </Popconfirm>
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

      {relBusy && relProgress && (
        <div className="char-migration-banner" data-testid="char-rel-progress">
          <span style={{ color: C('color-text') }}>{relProgress}（AI 逐个生成主角关系中，可继续操作）</span>
        </div>
      )}

      {/* 头部信息栏（收敛：无重复板块标题，保留统计与操作） */}
      <div className="char-panel-header">
        <div className="char-panel-stats">
          <span className="char-panel-stat">角色 <strong>{characters.length}</strong></span>
          <span className="char-panel-stat">组织 <strong>{organizations.length}</strong></span>
          <span className="char-panel-stat">关系 <strong>{relationships.length}</strong></span>
          {unimported.length > 0 && (
            <span className="char-panel-stat is-warn">未入库 <strong>{unimported.length}</strong></span>
          )}
        </div>
        <div className="char-panel-actions">
          <Dropdown
            menu={{
              items: [
                { key: 'all', label: '全部角色（覆盖重写）' },
                { key: 'missing', label: '剩余全部（只补空白）' },
              ],
              onClick: ({ key }) => handleGenRelations(key as 'all' | 'missing'),
            }}
            trigger={['click']}
          >
            <Button size="small" icon={<UserOutlined />} loading={relBusy} data-testid="char-gen-relations"
              title="AI 随机生成各角色与主角的关系短语（锚定本书「主角」定位的角色；只写本书，不进通用角色库）">
              AI 主角关系
            </Button>
          </Dropdown>
          <Button size="small" type="primary" icon={<ThunderboltOutlined />} onClick={() => setDrawOpen(true)}>抽卡</Button>
          {/* 简化批：低频运维三件（去角色库 / 回写 / 同步）收进「更多」菜单，功能零删除；
              原 loading 态以菜单项 disabled 表达 */}
          <Dropdown
            trigger={['click']}
            menu={{
              items: [
                { key: 'lib', label: '去角色库' },
                { key: 'writeback', label: '回写到角色库', disabled: wbBusy },
                { key: 'sync', label: '同步角色', disabled: syncing || unimported.length > 0 },
              ],
              onClick: ({ key }) => {
                if (key === 'lib') navigateToCharacterLib()
                else if (key === 'writeback') void handleImportLegacy()
                else if (key === 'sync') void handleSync()
              },
            }}
          >
            <Button size="small" icon={<DownOutlined />} iconPosition="end" aria-label="更多角色操作">更多</Button>
          </Dropdown>
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

      {/* 副本→库回写确认：非空冲突逐字段勾选，不勾选=保持角色库原值（关联即快照：定位/弧线/状态始终以本书为准） */}
      <Modal
        open={wbOpen}
        title="回写角色库：确认要覆盖的设定"
        width={640}
        onCancel={() => setWbOpen(false)}
        footer={[
          <Button key="cancel" onClick={() => setWbOpen(false)}>取消</Button>,
          <Button key="fill" onClick={() => { setWbOpen(false); void doWriteBack({}) }}>不覆盖，仅补全空缺</Button>,
          <Button key="ok" type="primary" disabled={Object.keys(wbChecked).length === 0}
            onClick={() => { const ov = collectOverwrites(); setWbOpen(false); void doWriteBack(ov) }}>
            覆盖勾选项并回写
          </Button>,
        ]}
      >
        <Typography.Paragraph type="secondary" style={{ fontSize: 12, marginTop: 0 }}>
          以下字段角色库与本书副本都有内容且不同。只有勾选的字段才会被副本值覆盖，未勾选的保持角色库原值；
          空缺字段无论如何都会补全。定位/弧线/状态始终以本书为准，不进角色库。
        </Typography.Paragraph>
        <div style={{ maxHeight: 400, overflow: 'auto' }}>
          {(Object.entries(wbPreview?.conflicts.reduce<Record<string, ImportFieldConflict[]>>((acc, c) => {
            (acc[`${c.characterId}::${c.characterName}`] ||= []).push(c)
            return acc
          }, {}) ?? []).map(([key, items]) => (
            <div key={key} style={{ marginBottom: 12 }}>
              <div style={{ fontWeight: 600, marginBottom: 4 }}>{items[0].characterName}</div>
              {items.map(c => {
                const ck = `${c.characterId}::${c.field}`
                return (
                  <label key={ck} style={{ display: 'flex', gap: 8, alignItems: 'flex-start', padding: '3px 0', cursor: 'pointer' }}>
                    <Checkbox
                      checked={!!wbChecked[ck]}
                      onChange={e => setWbChecked(prev => {
                        const next = { ...prev }
                        if (e.target.checked) next[ck] = true
                        else delete next[ck]
                        return next
                      })}
                      style={{ marginTop: 2 }}
                    />
                    <span style={{ flexShrink: 0, width: 44 }}>{c.fieldLabel}</span>
                    <span style={{ flex: 1, minWidth: 0 }}>
                      <span style={{ display: 'block', color: C('color-text-tertiary'), wordBreak: 'break-all' }}>库内：{c.libraryValue}</span>
                      <span style={{ display: 'block', wordBreak: 'break-all' }}>副本：{c.projectValue}</span>
                    </span>
                  </label>
                )
              })}
            </div>
          )))}
        </div>
      </Modal>
    </div>
  )
}

export default CharacterPage

