import React from 'react'
import { Tag } from 'antd'
import { ApartmentOutlined } from '@ant-design/icons'
import type { OrganizationData } from '../../../types'

export interface OrganizationCardProps {
  organization: OrganizationData
  onClick: () => void
}

const OrganizationCard: React.FC<OrganizationCardProps> = ({ organization, onClick }) => {
  const org = organization
  return (
    <div
      role="button"
      tabIndex={0}
      onClick={onClick}
      onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onClick() } }}
      className="char-card"
    >
      <div className="org-card-body">
        <div className="org-card-icon"><ApartmentOutlined /></div>
        <div className="org-card-main">
          <div className="org-card-title-row">
            <span className="org-card-name">{org.name}</span>
            <span className="org-card-tags">
              {org.members && org.members.length > 0 && (
                <Tag color="var(--color-primary)">{org.members.length}人</Tag>
              )}
              {org.type && <Tag>{org.type}</Tag>}
            </span>
          </div>
          <div className="org-card-desc">
            {org.description ? org.description.slice(0, 40) + (org.description.length > 40 ? '...' : '') : (org.power_level || '')}
          </div>
        </div>
      </div>
    </div>
  )
}

export default OrganizationCard
