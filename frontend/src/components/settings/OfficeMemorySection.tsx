import React, { useCallback, useEffect, useState } from 'react'
import { Switch, message } from 'antd'
import { DatabaseOutlined } from '@ant-design/icons'
import { app } from '../../gaea/lib/bridge'
import SettingsSection from './SettingsSection'

/**
 * OfficeMemorySection — 办公记忆开关（设置中枢批 4 从办公右栏 MemoryPanel 胶囊
 * 收编）：记忆注入。与面板胶囊同键同源（app.Memory/SetMemoryEnabled），双入口
 * 单源。自动做梦已删（2026-11-20 用户拍板：自动提炼建议质量差，记忆写入走
 * remember+确认卡）。晨报预载/项目本体两个开关 2026-11-25 撤下——那两块与
 * 记忆索引逐条重复，已停注（办公工具梳理批）。
 */
const OfficeMemorySection: React.FC = () => {
  const [memoryEnabled, setMemoryEnabled] = useState(true)

  useEffect(() => {
    app.Memory()
      .then((v) => {
        setMemoryEnabled(v?.enabled ?? true)
      })
      .catch(() => {})
  }, [])

  const patch = useCallback((next: boolean, okMsg: string, rollback: () => void) => {
    app.SetMemoryEnabled(next)
      .then(() => message.success(okMsg))
      .catch(() => {
        rollback()
        message.warning('设置失败，已还原')
      })
  }, [])

  const rowStyle: React.CSSProperties = {
    display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 12,
    padding: '8px 0',
  }
  const titleStyle: React.CSSProperties = { fontSize: 13, color: 'var(--md-sys-color-text)' }
  const descStyle: React.CSSProperties = { fontSize: 11, color: 'var(--md-sys-color-text-secondary)' }

  return (
    <SettingsSection
      icon={<span style={{ fontSize: 15 }}><DatabaseOutlined /></span>}
      title="办公记忆"
      desc="会话记忆注入策略：画像/规则/事实按工作区与运行模式隔离注入"
    >
      <div style={{ display: 'flex', flexDirection: 'column' }}>
        <div style={rowStyle}>
          <div>
            <div style={titleStyle}>记忆注入</div>
            <div style={descStyle}>关闭后不再注入画像/规则/事实（磁盘记忆保留）</div>
          </div>
          <Switch
            size="small"
            checked={memoryEnabled}
            onChange={(v) => {
              const prev = memoryEnabled
              setMemoryEnabled(v)
              patch(v, v ? '记忆已开启' : '记忆已关闭', () => setMemoryEnabled(prev))
            }}
          />
        </div>
      </div>
    </SettingsSection>
  )
}

export default OfficeMemorySection
