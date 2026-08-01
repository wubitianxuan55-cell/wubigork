import React, { useState } from 'react'
import { Typography, Tag, Space, Button, Row, Col, Input, Tabs, Spin, Skeleton } from 'antd'
import {
  DeleteOutlined, ThunderboltOutlined, ReloadOutlined, HeartOutlined,
  SmileOutlined, AimOutlined, BarChartOutlined, BookOutlined,
} from '@ant-design/icons'
import { C, ROLE_COLORS as roleColors, ROLE_LABELS as roleLabels, RELATION_LABELS as relationLabels } from '../../utils/theme'
import type { CharacterData, OrganizationData, RelationshipData } from '../../types'
import { Block, Field, extractTags } from './CharacterFormHelpers'

export interface CharacterEditorProps {
  editForm: CharacterData
  setEditForm: (f: CharacterData) => void
  onClose: () => void
  onDelete: (id: string) => void
  onSave: () => void
  onRandomGenerate: () => void
  onGeneratePortrait: () => void
  generatingPortrait: boolean
  genSingle: boolean
  onImportToWhisper?: () => void
  characters: CharacterData[]
  organizations: OrganizationData[]
  relationships: RelationshipData[]
  onNewOrg: () => void
  onToggleOrgMember: (orgID: string) => void
  onOpenRelModal: () => void
  onDeleteRel: (rel: RelationshipData) => void
  statusOptions: { value: string; label: string }[]
  onPortraitFullscreen: (url: string) => void
}

const roleTypeOptions = [
  { value: 'supporting', label: '配角' }, { value: 'minor', label: '次要' },
]

const CharacterEditor: React.FC<CharacterEditorProps> = ({
  editForm, setEditForm, onClose, onDelete, onSave, onRandomGenerate,
  onGeneratePortrait, generatingPortrait, genSingle,
  onImportToWhisper,
  characters, organizations, relationships,
  onNewOrg, onToggleOrgMember, onOpenRelModal, onDeleteRel,
  statusOptions, onPortraitFullscreen,
}) => {
  const [querying, setQuerying] = useState(false)
  const [chapterResult, setChapterResult] = useState<{ num: number; title: string }[]>([])

  const getCharName = (id: string) =>
    characters.find((c) => c.id === id)?.name || organizations.find((o) => o.id === id)?.name || id

  return (
    <>
      {/* ═══ 概览卡（剧照 + 信息） ═══ */}
      <div style={{
        display: 'flex', gap: 20, marginBottom: 16, padding: 20,
        background: 'var(--bg-glass)', backdropFilter: 'blur(8px)',
        border: '1px solid var(--border-subtle)', borderRadius: 'var(--radius-lg)',
        position: 'relative',
      }}>
        {/* 右上操作：导入为轻语角色 + 删除 */}
        <div style={{ position: 'absolute', top: 8, right: 8, display: 'flex', gap: 4 }}>
          {onImportToWhisper && (
            <Button type="text" size="small" icon={<HeartOutlined />}
              onClick={onImportToWhisper}
              title="导入为轻语角色（可在角色中心对话）"
              style={{ color: '#e85388', fontSize: 13 }}>
              导入为轻语
            </Button>
          )}
          <Button type="text" size="small" danger icon={<DeleteOutlined />}
            onClick={() => { onDelete(editForm.id); onClose() }} />
        </div>

        {/* 剧照 */}
        <div style={{ flexShrink: 0 }}>
          {editForm.portrait_url ? (
            <div style={{ position: 'relative' }}>
              <img src={editForm.portrait_url} alt={editForm.name}
                onDoubleClick={() => onPortraitFullscreen(editForm.portrait_url || '')}
                style={{ width: 160, height: 160, borderRadius: 'var(--radius-md)', objectFit: 'cover', border: '2px solid var(--border-subtle)', cursor: 'zoom-in' }} />
              <Button size="small" onClick={onGeneratePortrait} loading={generatingPortrait}
                style={{ position: 'absolute', bottom: 6, right: 6, fontSize: 10, opacity: 0.8 }}>
                <ReloadOutlined />
              </Button>
            </div>
          ) : generatingPortrait ? (
            <div style={{ width: 160, height: 160, borderRadius: 'var(--radius-md)', background: 'rgba(255,255,255,0.03)', border: '2px dashed var(--border-subtle)', display: 'flex', alignItems: 'center', justifyContent: 'center', flexDirection: 'column', gap: 8 }}>
              <Spin size="small" />
              <span style={{ color: C('color-text-secondary'), fontSize: 10 }}>生成中...</span>
            </div>
          ) : (
            <div onClick={onGeneratePortrait} style={{ width: 160, height: 160, borderRadius: 'var(--radius-md)', background: 'rgba(255,255,255,0.03)', border: '2px dashed var(--border-subtle)', display: 'flex', alignItems: 'center', justifyContent: 'center', cursor: 'pointer', flexDirection: 'column', gap: 6 }}>
              <ThunderboltOutlined style={{ fontSize: 24, color: C('color-text-secondary') }} />
              <span style={{ color: C('color-text-secondary'), fontSize: 11 }}>生成剧照</span>
            </div>
          )}
        </div>

        {/* 信息 */}
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap', marginBottom: 4 }}>
            <Typography.Title level={4} style={{ color: C('color-text'), margin: 0, fontSize: 20 }}>{editForm.name}</Typography.Title>
            <Tag color={roleColors[editForm.role_type]}>{roleLabels[editForm.role_type]}</Tag>
            <Tag color={editForm.status === 'Alive' ? 'green' : '#f87171'}>{statusOptions.find((o) => o.value === editForm.status)?.label || editForm.status}</Tag>
          </div>
          <div style={{ color: C('color-text-secondary'), fontSize: 12, marginBottom: 8 }}>
            {(editForm.gender || '?')} · {(editForm.age || '?')}岁 · {relationships.filter((r) => r.from_id === editForm.id || r.to_id === editForm.id).length} 段关系
          </div>
          {editForm.personality && (() => {
            const tags = extractTags(editForm.personality)
            if (tags.length < 2) return null
            return (
              <Space wrap size={2}>
                {tags.slice(0, 10).map((t) => <Tag key={t} style={{ fontSize: 10, lineHeight: '18px', padding: '0 8px', opacity: 0.85 }}>{t}</Tag>)}
              </Space>
            )
          })()}

          {/* 操作按钮 */}
          <Space style={{ marginTop: 12 }}>
            <Button size="small" icon={<ReloadOutlined />} onClick={onRandomGenerate} loading={genSingle}
              style={{ background: 'var(--bg-elevated)', border: '1px solid rgba(245, 158, 11, 0.3)', color: '#f59e0b', borderRadius: 'var(--radius-md)' }}>
              随机生成
            </Button>
            <Button type="primary" size="small" onClick={onSave}
              style={{ background: 'var(--color-primary)', borderColor: 'var(--color-primary)', boxShadow: 'var(--shadow-glow)', borderRadius: 'var(--radius-md)' }}>
              保存
            </Button>
          </Space>
        </div>
      </div>

      {/* ═══ Tab 内容区 ═══ */}
      <Tabs size="small" defaultActiveKey="profile" items={[
        {
          key: 'profile',
          label: '📋 档案',
          children: (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
              <Row gutter={[12, 8]}>
                <Col span={12}><Field l="姓名" v={editForm.name} onChange={(v) => setEditForm({ ...editForm, name: v })} noBlock /></Col>
                <Col span={12}><Field l="角色类型" type="select" value={editForm.role_type} onChange={(v) => setEditForm({ ...editForm, role_type: v })} options={roleTypeOptions} noBlock /></Col>
                <Col span={8}><Field l="性别" v={editForm.gender} onChange={(v) => setEditForm({ ...editForm, gender: v })} noBlock /></Col>
                <Col span={8}><Field l="年龄" v={editForm.age} onChange={(v) => setEditForm({ ...editForm, age: v })} noBlock /></Col>
                <Col span={8}><Field l="状态" type="select" value={editForm.status || 'Alive'} onChange={(v) => setEditForm({ ...editForm, status: v })} options={statusOptions} noBlock /></Col>
              </Row>

              <Block title={<span><SmileOutlined style={{ marginRight: 6 }} />性格特征</span>} extra={
                editForm.personality && extractTags(editForm.personality).length >= 2 ? (
                  <Space wrap size={2}>
                    {extractTags(editForm.personality).slice(0, 8).map((t) => (
                      <Tag key={t} style={{ fontSize: 9, lineHeight: '16px', padding: '0 6px' }}>{t}</Tag>
                    ))}
                  </Space>
                ) : null
              }>
                <Input.TextArea value={editForm.personality} onChange={(e) => setEditForm({ ...editForm, personality: e.target.value })} rows={3}
                  style={{ background: 'rgba(0,0,0,0.15)', border: '1px solid var(--border-subtle)', borderRadius: 'var(--radius-md)', color: 'var(--color-text)', fontSize: 13, boxShadow: 'inset 0 1px 4px rgba(0,0,0,0.2)' }} />
              </Block>

              <Row gutter={[12, 8]}>
                <Col span={12}>
                  <Block title="👤 外貌特征">
                    <Input.TextArea value={editForm.appearance} onChange={(e) => setEditForm({ ...editForm, appearance: e.target.value })} rows={3}
                      style={{ background: 'rgba(0,0,0,0.15)', border: '1px solid var(--border-subtle)', borderRadius: 'var(--radius-md)', color: 'var(--color-text)', fontSize: 13, boxShadow: 'inset 0 1px 4px rgba(0,0,0,0.2)' }} />
                  </Block>
                </Col>
                <Col span={12}>
                  <Block title="🏃 身材体型">
                    <Input.TextArea value={editForm.figure} onChange={(e) => setEditForm({ ...editForm, figure: e.target.value })} rows={3}
                      style={{ background: 'rgba(0,0,0,0.15)', border: '1px solid var(--border-subtle)', borderRadius: 'var(--radius-md)', color: 'var(--color-text)', fontSize: 13, boxShadow: 'inset 0 1px 4px rgba(0,0,0,0.2)' }} />
                  </Block>
                </Col>
              </Row>

              <Row gutter={[12, 8]}>
                <Col span={12}>
                  <Block title={<span><AimOutlined style={{ marginRight: 6 }} />核心动机</span>}>
                    <Input.TextArea value={editForm.motivation} onChange={(e) => setEditForm({ ...editForm, motivation: e.target.value })} rows={3}
                      style={{ background: 'rgba(0,0,0,0.15)', border: '1px solid var(--border-subtle)', borderRadius: 'var(--radius-md)', color: 'var(--color-text)', fontSize: 13, boxShadow: 'inset 0 1px 4px rgba(0,0,0,0.2)' }} />
                  </Block>
                </Col>
                <Col span={12}>
                  <Block title={<span><BarChartOutlined style={{ marginRight: 6 }} />角色弧光</span>}>
                    <Input.TextArea value={editForm.arc} onChange={(e) => setEditForm({ ...editForm, arc: e.target.value })} rows={3}
                      style={{ background: 'rgba(0,0,0,0.15)', border: '1px solid var(--border-subtle)', borderRadius: 'var(--radius-md)', color: 'var(--color-text)', fontSize: 13, boxShadow: 'inset 0 1px 4px rgba(0,0,0,0.2)' }} />
                  </Block>
                </Col>
              </Row>

              <Block title={<span><BookOutlined style={{ marginRight: 6 }} />背景故事</span>}>
                <Input.TextArea value={editForm.background} onChange={(e) => setEditForm({ ...editForm, background: e.target.value })} rows={4}
                  style={{ background: 'rgba(0,0,0,0.15)', border: '1px solid var(--border-subtle)', borderRadius: 'var(--radius-md)', color: 'var(--color-text)', fontSize: 13, boxShadow: 'inset 0 1px 4px rgba(0,0,0,0.2)' }} />
              </Block>
            </div>
          ),
        },
        {
          key: 'relations',
          label: `🔗 关系 (${relationships.filter((r) => r.from_id === editForm.id || r.to_id === editForm.id).length})`,
          children: (
            <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
              <Block title="🔗 所属组织" extra={<Button size="small" type="link" onClick={onNewOrg} style={{ color: '#60a5fa' }}>+新建</Button>}>
                <Space wrap size={4}>
                  {organizations.slice(0, 10).map((org) => {
                    const isMember = org.members?.includes(editForm.id)
                    return (
                      <Tag key={org.id} color={isMember ? C('color-primary') : C('color-border')}
                        style={{ cursor: 'pointer', opacity: isMember ? 1 : 0.5 }}
                        onClick={() => onToggleOrgMember(org.id)}>
                        {org.name} {isMember ? '✓' : '+'}
                      </Tag>
                    )
                  })}
                  {organizations.length > 10 && <Tag color={C('color-text-secondary')}>+{organizations.length - 10} 更多</Tag>}
                </Space>
              </Block>

              <Block title="人物关系" extra={<Button size="small" type="link" onClick={onOpenRelModal} style={{ color: '#60a5fa' }}>+添加</Button>}>
                {(() => {
                  const rels = relationships.filter((r) => r.from_id === editForm.id || r.to_id === editForm.id)
                  if (rels.length === 0) return <span style={{ color: C('color-text-secondary'), fontSize: 12 }}>暂无关系</span>
                  return (
                    <Space direction="vertical" size={2} style={{ width: '100%' }}>
                      {rels.slice(0, 12).map((rel, i) => (
                        <div key={i} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                          <Space size={4}>
                            <span style={{ color: C('color-text'), fontSize: 12 }}>{getCharName(rel.from_id)}</span>
                            <Tag color="#f59e0b" style={{ fontSize: 9 }}>{relationLabels[rel.relation_type] || rel.relation_type}</Tag>
                            <span style={{ color: C('color-text'), fontSize: 12 }}>{getCharName(rel.to_id)}</span>
                          </Space>
                          <Button type="text" size="small" danger icon={<DeleteOutlined />} onClick={() => onDeleteRel(rel)} />
                        </div>
                      ))}
                      {rels.length > 12 && <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 10, textAlign: 'center', display: 'block' }}>+{rels.length - 12} 更多关系</Typography.Text>}
                    </Space>
                  )
                })()}
              </Block>

              <Block title={<span><BookOutlined style={{ marginRight: 6 }} />出场章节</span>} extra={
                <Button size="small" type="link" style={{ color: '#60a5fa', fontSize: 10, padding: 0 }}
                  disabled={querying}
                  onClick={async () => {
                    setQuerying(true)
                    setChapterResult([])
                    try {
                      // @ts-ignore
                      const data = await window.go.app.App.GetOutlines()
                      const outlines: any[] = data?.nodes || []
                      const leaves: any[] = []
                      function find(nodes: any[]) { for (const n of nodes) { if (n.children?.length) find(n.children); else leaves.push(n) } }
                      find(outlines)
                      const found: { num: number; title: string }[] = []
                      for (const n of leaves) {
                        const num = n.order_index || 0
                        if (num < 1) continue
                        try {
                          // @ts-ignore
                          const ch = await window.go.app.App.GetChapter(num)
                          if (ch?.summary?.characters_appeared?.includes(editForm.name)) {
                            found.push({ num, title: n.title })
                          }
                        } catch (_) { }
                      }
                      setChapterResult(found)
                    } catch (_) { }
                    finally { setQuerying(false) }
                  }}>
                  {querying ? '查询中...' : '查询'}
                </Button>
              }>
                {querying ? <Skeleton active paragraph={{ rows: 2 }} /> : chapterResult.length > 0 ? (
                  <Row gutter={[4, 4]}>
                    {chapterResult.map((c) => (
                      <Col key={c.num}>
                        <Tag color="var(--color-primary)" style={{ fontSize: 10, cursor: 'pointer' }}>第{c.num}章 {c.title}</Tag>
                      </Col>
                    ))}
                  </Row>
                ) : (
                  <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11 }}>点击「查询」扫描全书章节出场记录</Typography.Text>
                )}
              </Block>
            </div>
          ),
        },
      ]} />
    </>
  )
}

export default CharacterEditor
