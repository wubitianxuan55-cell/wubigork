import React, { useEffect, useState } from 'react'
import { Button, Input, message } from 'antd'
import { TeamOutlined } from '@ant-design/icons'
import SettingsSection from './SettingsSection'
import { pantheonApi } from '../../pages/pantheon/api'
import type { PantheonAgentStatus } from '../../gaea/lib/types/pantheon'

/**
 * PantheonAgentsSection — 众神殿外部参与者路径管理（设置中枢批 5 从
 * PantheonPage「设置 Agent」Modal 收编）。PantheonAgents 列表+每行路径编辑
 * +PantheonSetExecutable 保存——与板块页同键同源；板块页参与者卡上的路径
 * 入口保留（讨论进行中的就近配置），本段=设置页常驻完整面。
 */
const PantheonAgentsSection: React.FC = () => {
  const [agents, setAgents] = useState<PantheonAgentStatus[]>([])
  const [drafts, setDrafts] = useState<Record<string, string>>({})
  const [saving, setSaving] = useState<string | null>(null)

  useEffect(() => {
    pantheonApi.PantheonAgents()
      .then((list: PantheonAgentStatus[] | null) => {
        setAgents(list || [])
        const init: Record<string, string> = {}
        for (const a of list || []) init[a.id] = a.executable || ''
        setDrafts(init)
      })
      .catch(() => {})
  }, [])

  const save = async (id: string, name: string) => {
    setSaving(id)
    try {
      const updated = await pantheonApi.PantheonSetExecutable(id, (drafts[id] ?? '').trim())
      setAgents(updated || [])
      const init: Record<string, string> = {}
      for (const a of updated || []) init[a.id] = a.executable || ''
      setDrafts(init)
      message.success(`${name} 路径已更新`)
    } catch (err: unknown) {
      message.error(`设置 Agent 路径失败：${err instanceof Error ? err.message : String(err)}`)
    } finally {
      setSaving(null)
    }
  }

  return (
    <SettingsSection
      icon={<span style={{ fontSize: 15 }}><TeamOutlined /></span>}
      title="外部参与者"
      desc="可执行文件或 CLI 脚本绝对路径——检测完成后该参与者可以加入下一次讨论"
    >
      <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
        {agents.length === 0 && (
          <div style={{ fontSize: 12, color: 'var(--md-sys-color-text-secondary)' }}>暂无外部参与者</div>
        )}
        {agents.map((a) => (
          <div key={a.id} style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <span style={{ fontSize: 13, color: 'var(--md-sys-color-text)' }}>{a.name}</span>
              <span
                style={{
                  fontSize: 10.5,
                  color: a.available ? 'var(--md-sys-color-primary)' : 'var(--md-sys-color-text-secondary)',
                }}
              >
                {a.available ? '可用' : '未配置'}
              </span>
            </div>
            <div style={{ display: 'flex', gap: 8 }}>
              <Input
                size="small"
                value={drafts[a.id] ?? ''}
                placeholder="例如 C:\Tools\agent-cli.exe"
                aria-label={`${a.name} 可执行文件路径`}
                onChange={(e) => setDrafts((prev) => ({ ...prev, [a.id]: e.target.value }))}
              />
              <Button
                size="small"
                loading={saving === a.id}
                disabled={(drafts[a.id] ?? '').trim() === (a.executable || '')}
                onClick={() => void save(a.id, a.name)}
              >
                保存
              </Button>
            </div>
            {a.reason && (
              <span style={{ fontSize: 11, color: 'var(--md-sys-color-text-secondary)' }}>{a.reason}</span>
            )}
          </div>
        ))}
      </div>
    </SettingsSection>
  )
}

export default PantheonAgentsSection
