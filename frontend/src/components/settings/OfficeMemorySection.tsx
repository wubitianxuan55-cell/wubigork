import React, { useCallback, useEffect, useState } from 'react'
import { Switch, Typography, message } from 'antd'
import { DatabaseOutlined } from '@ant-design/icons'
import { app } from '../../gaea/lib/bridge'
import SettingsSection from './SettingsSection'

/**
 * OfficeMemorySection — 办公记忆四态（设置中枢批 4 从办公右栏 MemoryPanel 胶囊
 * 收编）：记忆注入/晨报预载/项目本体/自动做梦。与面板胶囊同键同源（app.Memory/
 * SetMemoryEnabled/MorningPreload/MemoryBrief/SetDreamMode），双入口单源——
 * 面板胶囊=会话内高频快捷态，本段=设置页完整面。自动做梦默认建议制
 * （办公记忆写入确认红线：建议制先例，接受才入库）。
 */
const OfficeMemorySection: React.FC = () => {
  const [memoryEnabled, setMemoryEnabled] = useState(true)
  const [dreamMode, setDreamMode] = useState('suggest')
  const [morningPreload, setMorningPreload] = useState(true)
  const [projectBrief, setProjectBrief] = useState(true)

  useEffect(() => {
    app.Memory()
      .then((v) => {
        setMemoryEnabled(v?.enabled ?? true)
        setDreamMode(v?.dreamMode ?? 'suggest')
      })
      .catch(() => {})
    app.MorningPreload().then((v) => setMorningPreload(v)).catch(() => {})
    app.MemoryBrief().then((v) => setProjectBrief(v)).catch(() => {})
  }, [])

  const patch = useCallback(
    (kind: string, next: boolean | string, okMsg: string, rollback: () => void) => {
      const call =
        kind === 'memory' ? app.SetMemoryEnabled(next as boolean)
        : kind === 'morning' ? app.SetMorningPreload(next as boolean)
        : kind === 'brief' ? app.SetMemoryBrief(next as boolean)
        : app.SetDreamMode(next as string)
      call.then(() => message.success(okMsg)).catch(() => {
        rollback()
        message.warning('设置失败，已还原')
      })
    },
    [],
  )

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
      desc="会话记忆注入策略：画像/规则/事实注入、晨报预装配、项目本体引用块、自动做梦整理（建议制=提炼进待确认，接受才入库）"
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
              patch('memory', v, v ? '记忆已开启' : '记忆已关闭', () => setMemoryEnabled(prev))
            }}
          />
        </div>
        <div style={rowStyle}>
          <div>
            <div style={titleStyle}>晨报预载</div>
            <div style={descStyle}>work 空间新会话自动预装配高频工作记忆</div>
          </div>
          <Switch
            size="small"
            checked={morningPreload}
            onChange={(v) => {
              const prev = morningPreload
              setMorningPreload(v)
              patch('morning', v, v ? '晨报预载已开启' : '晨报预载已关闭', () => setMorningPreload(prev))
            }}
          />
        </div>
        <div style={rowStyle}>
          <div>
            <div style={titleStyle}>项目本体</div>
            <div style={descStyle}>新会话注入固化/项目决策事实（带 [MEM:] 引用键）</div>
          </div>
          <Switch
            size="small"
            checked={projectBrief}
            onChange={(v) => {
              const prev = projectBrief
              setProjectBrief(v)
              patch('brief', v, v ? '项目本体已开启' : '项目本体已关闭', () => setProjectBrief(prev))
            }}
          />
        </div>
        <div style={rowStyle}>
          <div>
            <div style={titleStyle}>自动做梦</div>
            <div style={descStyle}>轮次结束后自动整理记忆：建议=提炼进待确认（默认）| 直写=直接入库 | 关=不整理</div>
          </div>
          <Typography.Text
            code
            style={{ fontSize: 11, cursor: 'pointer' }}
            onClick={() => {
              const next = dreamMode === 'off' ? 'suggest' : dreamMode === 'suggest' ? 'auto' : 'off'
              const prev = dreamMode
              setDreamMode(next)
              patch('dream', next, `自动做梦：${next === 'off' ? '关' : next === 'auto' ? '直写' : '建议'}`, () => setDreamMode(prev))
            }}
          >
            {dreamMode === 'off' ? '关' : dreamMode === 'auto' ? '直写' : '建议'}
          </Typography.Text>
        </div>
      </div>
    </SettingsSection>
  )
}

export default OfficeMemorySection
