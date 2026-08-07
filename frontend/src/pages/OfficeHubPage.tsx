// OfficeHubPage.tsx — 办公板块（独立二级导航）
// 顶栏「办公」标题右侧为水平二级导航，下方为对应工作区；
// 未来新增办公专用模块时，只需在 OFFICE_MODULES 数组中追加一项。
// 各工作区同时挂载、按需显示，切换不丢失状态；上次选择持久化。
import React, { useState } from 'react'
import { RobotOutlined, FileTextOutlined } from '@ant-design/icons'
import GaeaPage from './GaeaPage'
import OfficePage from './OfficePage'
import './office-hub.css'

const OFFICE_MODULE_KEY = 'gaea.officeModule'

interface OfficeModule {
  id: string
  label: string
  icon: React.ReactNode
  render: () => React.ReactNode
}

// ── 办公二级模块注册表：未来新增模块在此追加 ──
const OFFICE_MODULES: OfficeModule[] = [
  {
    id: 'agent',
    label: '通用办公',
    icon: <RobotOutlined />,
    render: () => <GaeaPage />,
  },
  {
    id: 'proposal',
    label: '方案编写',
    icon: <FileTextOutlined />,
    render: () => <OfficePage />,
  },
]

const OfficeHubPage: React.FC = () => {
  const [active, setActive] = useState<string>(() => {
    try {
      const saved = localStorage.getItem(OFFICE_MODULE_KEY)
      return OFFICE_MODULES.some(m => m.id === saved) ? (saved as string) : OFFICE_MODULES[0].id
    } catch {
      return OFFICE_MODULES[0].id
    }
  })

  const switchModule = (id: string) => {
    setActive(id)
    try { localStorage.setItem(OFFICE_MODULE_KEY, id) } catch { /* ignore */ }
  }

  return (
    <div className="office-hub">
      {/* 板块头 + 水平二级导航 */}
      <div className="office-hub-head">
        <span className="office-hub-title">办公</span>
        <nav className="office-subnav" aria-label="办公模块">
          {OFFICE_MODULES.map(m => (
            <button
              key={m.id}
              type="button"
              className={`office-subnav-item ${active === m.id ? 'office-subnav-item-active' : ''}`}
              onClick={() => switchModule(m.id)}
              aria-current={active === m.id ? 'page' : undefined}
            >
              <span className="office-subnav-icon">{m.icon}</span>
              {m.label}
            </button>
          ))}
        </nav>
        <span className="office-hub-hint">工作区各自保留状态，可随时切换</span>
      </div>

      {/* 工作区 */}
      <div className="office-hub-content">
        {OFFICE_MODULES.map(m => (
          <div key={m.id} style={{ display: active === m.id ? 'flex' : 'none', flex: 1, minHeight: 0 }}>
            {m.render()}
          </div>
        ))}
      </div>
    </div>
  )
}

export default OfficeHubPage
