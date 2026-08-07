import React, { useEffect, useState } from 'react'
import { Button, Input, InputNumber, Select, Space, Switch, Tag, Typography, message } from 'antd'
import { SaveOutlined, RobotOutlined, ApiOutlined, CheckCircleOutlined, ReloadOutlined } from '@ant-design/icons'
import { gaeaSettings } from '../../api/settings'
import SettingsSection from './SettingsSection'
import * as App from '../../../wailsjs/go/app/App'

interface DraftView {
  defaultModel: string
  subagentModel: string
  systemPrompt: string
  temperature: number
  subagentTemperature: number
  maxSteps: number
  effort: string
  subagentEffort: string
  permMode: string
  permAllow: string
  permAsk: string
  permDeny: string
  sandboxBash: string
  sandboxNetwork: boolean
  workspaceRoot: string
}

const emptyDraft: DraftView = {
  defaultModel: '', subagentModel: '', systemPrompt: '',
  temperature: 0, subagentTemperature: 0, maxSteps: 0,
  effort: '', subagentEffort: '', permMode: 'ask', permAllow: '', permAsk: '', permDeny: '',
  sandboxBash: 'enforce', sandboxNetwork: true, workspaceRoot: '',
}

/** OfficePanel — 办公引擎完整设置（模型 / Agent 参数 / 权限 / 沙箱，持久化到 ~/.config/gaea/config.toml） */
const OfficePanel: React.FC = () => {
  const [view, setView] = useState<Record<string, any>>({})
  const [draft, setDraft] = useState<DraftView>(emptyDraft)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [reloading, setReloading] = useState(false)

  useEffect(() => {
    gaeaSettings().then((v) => {
      setView(v || {})
      const agent = (v as any)?.agent || {}
      const perms = (v as any)?.permissions || {}
      const sandbox = (v as any)?.sandbox || {}
      setDraft({
        defaultModel: (v as any)?.defaultModel || '',
        subagentModel: (v as any)?.subagentModel || '',
        systemPrompt: agent.systemPrompt || '',
        temperature: agent.temperature || 0,
        subagentTemperature: agent.subagentTemperature || 0,
        maxSteps: agent.maxSteps || 0,
        effort: agent.effort || '',
        subagentEffort: agent.subagentEffort || '',
        permMode: perms.mode || 'ask',
        permAllow: (perms.allow || []).join(', '),
        permAsk: (perms.ask || []).join(', '),
        permDeny: (perms.deny || []).join(', '),
        sandboxBash: sandbox.bash || 'enforce',
        sandboxNetwork: sandbox.network !== false,
        workspaceRoot: sandbox.workspaceRoot || '',
      })
    }).catch(() => {}).finally(() => setLoading(false))
  }, [])

  const providers: any[] = view.providers || []
  const modelOptions = providers.flatMap((p: any) => (p.models || []).map((m: string) => ({ value: m, label: `${p.name} / ${m}` })))

  const handleSave = async () => {
    setSaving(true)
    try {
      await (App as any).GaeaSaveSettings({
        defaultModel: draft.defaultModel,
        subagentModel: draft.subagentModel,
        agent: {
          systemPrompt: draft.systemPrompt,
          temperature: draft.temperature,
          subagentTemperature: draft.subagentTemperature,
          maxSteps: draft.maxSteps,
          effort: draft.effort,
          subagentEffort: draft.subagentEffort,
        },
        permissions: {
          mode: draft.permMode,
          allow: draft.permAllow.split(',').map(s => s.trim()).filter(Boolean),
          ask: draft.permAsk.split(',').map(s => s.trim()).filter(Boolean),
          deny: draft.permDeny.split(',').map(s => s.trim()).filter(Boolean),
        },
        sandbox: {
          bash: draft.sandboxBash,
          network: draft.sandboxNetwork,
          workspaceRoot: draft.workspaceRoot,
          allowWrite: ((view as any)?.sandbox?.allowWrite) || [],
        },
      })
      message.success('办公设置已保存并生效')
    } catch (err: any) {
      message.error(err?.message || '保存失败')
    } finally {
      setSaving(false)
    }
  }

  // handleReload 从磁盘重新读取引擎配置并重建控制器——外部直接编辑
  // config.toml 或技能/工具目录后，无需重启桌面端即可生效。
  const handleReload = async () => {
    setReloading(true)
    try {
      const res = await (App as any).GaeaReload()
      message.success(`引擎已热加载：${res.tools} 个工具 · ${res.skills} 个技能`)
    } catch (err: any) {
      message.error(err?.message || '热加载失败')
    } finally {
      setReloading(false)
    }
  }

  const field = (label: string, node: React.ReactNode, extra?: string) => (
    <div style={{ marginBottom: 12 }}>
      <div style={{ fontSize: 12, color: 'var(--md-sys-color-text-secondary)', marginBottom: 4 }}>{label}</div>
      {node}
      {extra && <div style={{ fontSize: 11, color: 'var(--md-sys-color-text-secondary)', marginTop: 2, opacity: 0.8 }}>{extra}</div>}
    </div>
  )

  const selectStyle = { width: '100%' }

  return (
    <>
      <SettingsSection
        title={<>办公引擎设置</>}
        desc="完整设置（模型 / Agent 参数 / 权限 / 沙箱），持久化到 ~/.config/gaea/config.toml。"
        instant
      >
        {/* 模型区 */}
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, minmax(0, 1fr))', gap: '0 16px' }}>
          {field('默认模型', (
            <Select
              placeholder="选择默认模型" value={draft.defaultModel || undefined}
              onChange={(v) => setDraft({ ...draft, defaultModel: v })}
              options={modelOptions} style={selectStyle} size="small"
              showSearch optionFilterProp="label"
            />
          ))}
          {field('最大步骤', (
            <InputNumber size="small" min={0} max={100} value={draft.maxSteps}
              onChange={(v) => setDraft({ ...draft, maxSteps: v || 0 })} style={selectStyle} />
          ), '0 = 不限制')}
          {field('Subagent 模型', (
            <Input size="small" placeholder="如 xai / grok-4.20" value={draft.subagentModel}
              onChange={(e) => setDraft({ ...draft, subagentModel: e.target.value })} />
          ))}
        </div>
      </SettingsSection>

      <SettingsSection
        title={<>Agent 参数</>}
        desc="对话温度与推理强度（DeepSeek：high / max；空 = 提供方默认）。"
      >
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, minmax(0, 1fr))', gap: '0 16px' }}>
          {field('温度', (
            <InputNumber size="small" min={0} max={2} step={0.05} value={draft.temperature}
              onChange={(v) => setDraft({ ...draft, temperature: v || 0 })} style={selectStyle} />
          ))}
          {field('Subagent 温度', (
            <InputNumber size="small" min={0} max={2} step={0.05} value={draft.subagentTemperature}
              onChange={(v) => setDraft({ ...draft, subagentTemperature: v || 0 })} style={selectStyle} />
          ))}
          {field('推理强度', (
            <Select size="small" allowClear placeholder="提供方默认"
              value={draft.effort || undefined}
              onChange={(v) => setDraft({ ...draft, effort: v || '' })}
              options={[{ value: 'high', label: 'high' }, { value: 'max', label: 'max' }]} style={selectStyle} />
          ))}
          {field('Subagent 推理强度', (
            <Select size="small" allowClear placeholder="同主推理强度"
              value={draft.subagentEffort || undefined}
              onChange={(v) => setDraft({ ...draft, subagentEffort: v || '' })}
              options={[{ value: 'high', label: 'high' }, { value: 'max', label: 'max' }]} style={selectStyle} />
          ))}
        </div>
        {field('系统提示词（SystemPrompt）', (
          <Input.TextArea rows={3} value={draft.systemPrompt}
            onChange={(e) => setDraft({ ...draft, systemPrompt: e.target.value })}
            style={{ background: 'var(--md-sys-color-surface-container)', border: '1px solid var(--md-sys-color-outline-variant)' }} />
        ))}
      </SettingsSection>

      <SettingsSection
        title={<>权限</>}
        desc="工具调用权限模式与规则（逗号分隔）。模式：ask = 每次询问 / allow = 直接允许 / deny = 直接拒绝。"
      >
        {field('权限模式', (
          <Select size="small" value={draft.permMode}
            onChange={(v) => setDraft({ ...draft, permMode: v })}
            options={[
              { value: 'ask', label: 'ask（询问）' },
              { value: 'allow', label: 'allow（允许）' },
              { value: 'deny', label: 'deny（拒绝）' },
            ]} style={selectStyle} />
        ))}
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, minmax(0, 1fr))', gap: '0 16px' }}>
          {field('允许规则', <Input size="small" value={draft.permAllow} onChange={(e) => setDraft({ ...draft, permAllow: e.target.value })} placeholder="如 run_skill, file_write" />)}
          {field('询问规则', <Input size="small" value={draft.permAsk} onChange={(e) => setDraft({ ...draft, permAsk: e.target.value })} placeholder="如 bash" />)}
          {field('拒绝规则', <Input size="small" value={draft.permDeny} onChange={(e) => setDraft({ ...draft, permDeny: e.target.value })} placeholder="如 network" />)}
        </div>
      </SettingsSection>

      <SettingsSection
        title={<>沙箱</>}
        desc="工具执行的 OS 边界（Bash 隔离模式、网络访问、工作目录）。"
      >
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, minmax(0, 1fr))', gap: '0 16px' }}>
          {field('Bash 模式', (
            <Select size="small" value={draft.sandboxBash}
              onChange={(v) => setDraft({ ...draft, sandboxBash: v })}
              options={[
                { value: 'enforce', label: 'enforce（隔离）' },
                { value: 'off', label: 'off（不隔离）' },
              ]} style={selectStyle} />
          ))}
          {field('网络访问', (
            <Space size={8}>
              <Switch checked={draft.sandboxNetwork} onChange={(v) => setDraft({ ...draft, sandboxNetwork: v })} />
              <Typography.Text style={{ fontSize: 12, color: 'var(--md-sys-color-text-secondary)' }}>
                {draft.sandboxNetwork ? '允许出网' : '禁止出网'}
              </Typography.Text>
            </Space>
          ))}
        </div>
        {field('工作目录（WorkspaceRoot）', (
          <Input size="small" value={draft.workspaceRoot}
            onChange={(e) => setDraft({ ...draft, workspaceRoot: e.target.value })}
            placeholder="文件写入根目录" />
        ))}
      </SettingsSection>

      <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8, marginTop: 4 }}>
        <Button icon={<ReloadOutlined />} loading={reloading} onClick={handleReload}>
          从磁盘热加载
        </Button>
        <Button type="primary" icon={<SaveOutlined />} loading={saving} onClick={handleSave}
          style={{ background: 'var(--md-sys-color-primary)', borderColor: 'var(--md-sys-color-primary)', borderRadius: 'var(--md-sys-radius-md)' }}>
          保存并生效
        </Button>
      </div>

      {loading && <Typography.Text style={{ color: 'var(--md-sys-color-text-secondary)', fontSize: 12 }}>加载中…</Typography.Text>}
    </>
  )
}

export default OfficePanel
