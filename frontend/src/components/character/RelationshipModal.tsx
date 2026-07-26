import React from 'react'
import { Typography, Button, Space, Select, Modal } from 'antd'
import { C } from '../../utils/theme'
import type { CharacterData, OrganizationData, RelationshipData } from '../../types'

interface RelationshipModalProps {
  open: boolean
  onClose: () => void
  characters: CharacterData[]
  organizations: OrganizationData[]
  editForm: { id: string } | null
  relTargetId: string
  onRelTargetChange: (id: string) => void
  relType: string
  onRelTypeChange: (t: string) => void
  onAdd: () => void
}

/** RelationshipModal — 添加关系弹窗 */
const RelationshipModal: React.FC<RelationshipModalProps> = ({
  open, onClose, characters, organizations, editForm,
  relTargetId, onRelTargetChange,
  relType, onRelTypeChange, onAdd,
}) => (
  <Modal title="添加关系" open={open} onCancel={onClose} footer={null}
    width={400}
    styles={{ body: { background: 'var(--bg-glass)', backdropFilter: 'blur(8px)', WebkitBackdropFilter: 'blur(8px)', padding: 24 } }}
  >
    <Space direction="vertical" size={12} style={{ width: '100%' }}>
      <div>
        <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11, display: 'block', marginBottom: 4 }}>目标角色/组织</Typography.Text>
        <Select showSearch value={relTargetId} onChange={onRelTargetChange}
          style={{ width: '100%', background: 'rgba(255,255,255,0.05)', border: '1px solid var(--border-subtle)', borderRadius: 'var(--radius-md)' }}
          placeholder="搜索并选择..."
          filterOption={(input, option) => (option?.label as string || '').includes(input)}
          options={[
            { label: '─ 角色 ─', value: '', disabled: true },
            ...characters.filter((c) => c.id !== editForm?.id).map((c) => ({ label: `${c.name} (${c.role_type})`, value: c.id })),
            { label: '─ 组织 ─', value: '', disabled: true },
            ...organizations.map((o) => ({ label: o.name, value: o.id })),
          ]}
        />
      </div>
      <div>
        <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11, display: 'block', marginBottom: 4 }}>关系类型</Typography.Text>
        <Select value={relType} onChange={onRelTypeChange}
          style={{ width: '100%', background: 'rgba(255,255,255,0.05)', borderColor: 'var(--border-subtle)', borderRadius: 'var(--radius-md)' }}
          options={[
            { value: 'friend', label: '朋友' }, { value: 'enemy', label: '敌人' },
            { value: 'family', label: '家人' }, { value: 'mentor', label: '导师' },
            { value: 'rival', label: '对手' }, { value: 'lover', label: '恋人' },
            { value: 'member', label: '成员' }, { value: 'leader', label: '领袖' },
          ]}
        />
      </div>
      <div style={{ textAlign: 'right', paddingTop: 8 }}>
        <Button onClick={onClose} style={{ marginRight: 8, background: 'var(--bg-elevated)', border: '1px solid var(--border-subtle)', borderRadius: 'var(--radius-md)' }}>取消</Button>
        <Button type="primary" onClick={onAdd} disabled={!relTargetId}
          style={{ background: 'var(--color-primary)', borderColor: 'var(--color-primary)', boxShadow: 'var(--shadow-glow)', borderRadius: 'var(--radius-md)' }}>添加</Button>
      </div>
    </Space>
  </Modal>
)

export default RelationshipModal
