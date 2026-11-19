import { Switch } from 'antd'
import { AppstoreOutlined } from '@ant-design/icons'
import SettingsSection from './SettingsSection'

/**
 * PaneManagementSection — 右栏工作台面板启停（设置中枢批 4 从 WorkspacePane
 * 「管理」弹层收编）。办公工位与文学壳层各有独立清单/store（gaea/lib 与
 * wenxue/lib 双副本），故参数化：调用方传 tabs+useStore，本组件零域知识。
 * 与面板原位「管理」弹层同键同源（zustand store 即时一致）——弹层保留为
 * 会话内高频入口，本段=设置页完整面（CAD 等可选卡的常驻开关自此双入口可达）。
 */
export interface PaneManagementSectionProps<T extends string> {
  title: string
  desc: string
  tabs: { id: T; label: string }[]
  useStore: () => {
    enabled: Record<T, boolean>
    setEnabled: (id: T, on: boolean) => void
  }
}

const PaneManagementSection = <T extends string>({ title, desc, tabs, useStore }: PaneManagementSectionProps<T>) => {
  const { enabled, setEnabled } = useStore()
  return (
    <SettingsSection
      icon={<span style={{ fontSize: 15 }}><AppstoreOutlined /></span>}
      title={title}
      desc={desc}
    >
      <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
        {tabs.map((tab) => (
          <div
            key={tab.id}
            style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 12, padding: '6px 0' }}
          >
            <span style={{ fontSize: 13, color: 'var(--md-sys-color-text)' }}>{tab.label}</span>
            <Switch
              size="small"
              checked={enabled[tab.id]}
              aria-label={`启用 ${tab.label}`}
              onChange={(v) => setEnabled(tab.id, v)}
            />
          </div>
        ))}
      </div>
    </SettingsSection>
  )
}

export default PaneManagementSection
