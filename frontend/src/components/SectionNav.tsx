import React, { useMemo } from 'react'
import { Typography, Button } from 'antd'
import {
  HistoryOutlined, GlobalOutlined, TeamOutlined,
  SafetyOutlined, SmileOutlined, FileTextOutlined,
  DownOutlined, RightOutlined,
} from '@ant-design/icons'
import type { WorldviewSectionData } from '../types'

// 维度图标
const sectionIcons: Record<string, React.ReactNode> = {
  era: <HistoryOutlined />, geography: <GlobalOutlined />, factions: <TeamOutlined />,
  rules: <SafetyOutlined />, culture: <SmileOutlined />, history: <FileTextOutlined />,
}

interface SectionNavProps {
  sections: WorldviewSectionData[]
  activeSection: string
  onSelect: (sectionId: string) => void
  /** 移动端：下拉面板；桌面端：常驻侧栏 */
  variant: 'desktop' | 'mobile'
}

/** SectionNavItem — 单个维度条目（桌面/移动端共用） */
const SectionNavItem: React.FC<{
  sec: WorldviewSectionData
  isActive: boolean
  onClick: () => void
}> = ({ sec, isActive, onClick }) => (
  <div
    onClick={onClick}
    style={{
      display: 'flex', alignItems: 'center', gap: 8,
      padding: '8px 10px', borderRadius: 'var(--radius-md)',
      cursor: 'pointer',
      background: isActive ? 'rgba(var(--accent-rgb), 0.1)' : 'transparent',
      borderLeft: `3px solid ${isActive ? 'var(--color-primary)' : 'transparent'}`,
    }}
  >
    <span style={{ fontSize: 14 }}>{sectionIcons[sec.id] || <FileTextOutlined />}</span>
    <div style={{ flex: 1, minWidth: 0 }}>
      <Typography.Text style={{
        color: isActive ? 'var(--color-primary)' : 'var(--color-text)', fontSize: 12,
      }}>
        {sec.title}
      </Typography.Text>
      <div style={{
        fontSize: 9,
        color: sec.content ? 'var(--color-primary)' : 'var(--color-text-secondary)',
        opacity: 0.7,
      }}>
        {sec.content ? '已填写' : '待补充'}
      </div>
    </div>
  </div>
)

/**
 * SectionNav — 世界观维度导航
 *
 * 桌面端：常驻 180px 侧栏
 * 移动端：下拉面板（Button + 弹出列表）
 */
const SectionNav: React.FC<SectionNavProps> = ({ sections, activeSection, onSelect, variant }) => {
  const [mobileOpen, setMobileOpen] = React.useState(false)
  const navSections = useMemo(() => sections.filter((s) => s.id !== 'legacy'), [sections])

  if (variant === 'mobile') {
    const currentTitle = navSections.find((s) => s.id === activeSection)?.title || '维度导航'
    return (
      <div style={{ flexShrink: 0, position: 'relative' }}>
        <Button
          size="small"
          icon={mobileOpen ? <DownOutlined /> : <RightOutlined />}
          onClick={() => setMobileOpen(!mobileOpen)}
          style={{
            background: 'var(--md-sys-color-surface-container)',
            border: '1px solid var(--border-subtle)',
            borderRadius: 'var(--radius-md)',
            color: 'var(--color-text)',
            fontSize: 12,
            whiteSpace: 'nowrap',
          }}
        >
          {currentTitle}
        </Button>
        {mobileOpen && (
          <div style={{
            position: 'absolute', top: 0, left: 0, right: 0, zIndex: 50,
            background: 'var(--md-sys-color-surface-container-high)',
            borderBottom: '1px solid var(--border-subtle)',
            borderRadius: '0 0 var(--radius-lg) var(--radius-lg)',
            padding: '8px',
            boxShadow: 'var(--md-sys-elevation-3)',
            display: 'flex', flexDirection: 'column', gap: 2,
          }}>
            {navSections.map((sec) => (
              <SectionNavItem
                key={sec.id}
                sec={sec}
                isActive={activeSection === sec.id}
                onClick={() => {
                  if (activeSection === sec.id) onSelect('')
                  else onSelect(sec.id)
                  setMobileOpen(false)
                }}
              />
            ))}
          </div>
        )}
      </div>
    )
  }

  return (
    <div style={{ width: 180, flexShrink: 0, display: 'flex', flexDirection: 'column', gap: 2, overflow: 'auto' }}>
      {navSections.map((sec) => (
        <SectionNavItem
          key={sec.id}
          sec={sec}
          isActive={activeSection === sec.id}
          onClick={() => {
            if (activeSection === sec.id) onSelect('')
            else onSelect(sec.id)
          }}
        />
      ))}
    </div>
  )
}

export default SectionNav
