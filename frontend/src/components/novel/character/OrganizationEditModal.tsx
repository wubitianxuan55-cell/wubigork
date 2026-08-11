import React from 'react'
import { Typography, Button, Space, Tag, Modal } from 'antd'
import { ApartmentOutlined, DeleteOutlined } from '@ant-design/icons'
import { C } from '../../../utils/theme'
import type { OrganizationData } from '../../../types'
import { OrgField } from './CharacterFormHelpers'

interface OrganizationEditModalProps {
  open: boolean
  org: OrganizationData | null
  onClose: () => void
  onSave: () => void
  onDelete: (id: string) => void
  onEditOrgChange: (org: OrganizationData) => void
  getCharName: (id: string) => string
}

/** OrganizationEditModal — 组织编辑弹窗 */
const OrganizationEditModal: React.FC<OrganizationEditModalProps> = ({
  open, org, onClose, onSave, onDelete, onEditOrgChange, getCharName,
}) => (
  <Modal title={null} open={open} onCancel={onClose} footer={null}
    width={480}
    // WebView2 冻结 rAF 时关闭动画不结束会残留全屏 wrap 拦截点击：关闭即卸载。
    destroyOnHidden transitionName="" maskTransitionName=""
    styles={{ body: { background: 'var(--bg-glass)', backdropFilter: 'blur(8px)', WebkitBackdropFilter: 'blur(8px)', padding: 24 } }}
  >
    {org && (
      <div>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
          <Typography.Title level={4} style={{ color: C('color-text'), margin: 0 }}>
            <ApartmentOutlined style={{ marginRight: 8, color: '#c084fc' }} />{org.name}
          </Typography.Title>
          <Space>
            <Button type="primary" onClick={onSave}
              style={{ background: 'var(--color-primary)', borderColor: 'var(--color-primary)', boxShadow: 'var(--shadow-glow)', borderRadius: 'var(--radius-md)' }}>保存</Button>
            <Button danger icon={<DeleteOutlined />} onClick={() => { onDelete(org.id); onClose() }}>删除</Button>
          </Space>
        </div>
        <Space direction="vertical" size={8} style={{ width: '100%' }}>
          <OrgField l="组织名称" v={org.name} onChange={(v) => onEditOrgChange({ ...org, name: v })} />
          <OrgField l="类型" v={org.type} onChange={(v) => onEditOrgChange({ ...org, type: v })} />
          <OrgField l="实力等级" v={org.power_level} onChange={(v) => onEditOrgChange({ ...org, power_level: v })} />
          <OrgField l="描述" type="textarea" rows={2} v={org.description} onChange={(v) => onEditOrgChange({ ...org, description: v })} />
          <OrgField l="位置" type="textarea" rows={2} v={org.location || ''} onChange={(v) => onEditOrgChange({ ...org, location: v })} />
          <OrgField l="格言" type="textarea" rows={2} v={org.motto || ''} onChange={(v) => onEditOrgChange({ ...org, motto: v })} />
          {org.members && org.members.length > 0 && (
            <div>
              <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11, display: 'block', marginBottom: 4 }}>成员</Typography.Text>
              <Space wrap size={4}>
                {org.members.map((mid) => (
                  <Tag key={mid} color="green">{getCharName(mid)}</Tag>
                ))}
              </Space>
            </div>
          )}
        </Space>
      </div>
    )}
  </Modal>
)

export default OrganizationEditModal
