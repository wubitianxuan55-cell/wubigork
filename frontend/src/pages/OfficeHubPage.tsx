// OfficeHubPage.tsx — 办公板块（单一入口：通用办公工作台）。
// 方案编写模块已下线，办公板块不再需要二级导航，直接承载通用办公工作台。
import React from 'react'
import GaeaPage from './GaeaPage'
import './office-hub.css'

const OfficeHubPage: React.FC = () => {
  return (
    <div className="office-hub">
      {/* 板块头 */}
      <div className="office-hub-head">
        <span className="office-hub-title">办公</span>
      </div>

      {/* 工作区：通用办公 */}
      <div className="office-hub-content">
        <div style={{ display: 'flex', flex: 1, minHeight: 0 }}>
          <GaeaPage />
        </div>
      </div>
    </div>
  )
}

export default OfficeHubPage
