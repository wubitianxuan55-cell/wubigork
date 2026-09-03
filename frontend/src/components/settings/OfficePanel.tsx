import React, { useEffect, useState } from 'react'
import { Button, Input, InputNumber, Select, Space, Switch, Typography, message } from 'antd'
import { SaveOutlined, ReloadOutlined, FileTextOutlined } from '@ant-design/icons'
import { gaeaSettings } from '../../api/settings'
import SettingsSection from './SettingsSection'
import { useT } from '../../gaea/lib/i18n'
import * as App from '../../../src/wailsjsCompat'
import { app as gaeaApp } from '../../gaea/lib/bridge'
import type { app as AppModels } from '../../../wailsjs/go/models'
import type { LintReportView } from '../../gaea/lib/types'

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
  const t = useT()
  const [view, setView] = useState<Record<string, unknown>>({})
  const [draft, setDraft] = useState<DraftView>(emptyDraft)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [reloading, setReloading] = useState(false)
  // v4.1c 规范体检（GB/T 9704 红头要素 lint）
  const [lintPath, setLintPath] = useState('')
  const [linting, setLinting] = useState(false)
  const [lintReport, setLintReport] = useState<LintReportView | null>(null)

  const runLint = async () => {
    const rel = lintPath.trim()
    if (!rel) { message.warning(t('settings.office.lintRequired')); return }
    setLinting(true)
    try {
      const r = await gaeaApp.DocumentLint(rel)
      setLintReport(r)
      message[r.passed ? 'success' : 'warning'](r.summary)
    } catch (e) {
      message.error(t('settings.office.lintFailed', { msg: e instanceof Error ? e.message : String(e) }))
    } finally {
      setLinting(false)
    }
  }

  useEffect(() => {
    gaeaSettings().then((v) => {
      setView(v || {})
      const agent = (v.agent || {}) as {
        systemPrompt?: string; temperature?: number; subagentTemperature?: number;
        maxSteps?: number; effort?: string; subagentEffort?: string
      }
      const perms = (v.permissions || {}) as { mode?: string; allow?: string[]; ask?: string[]; deny?: string[] }
      const sandbox = (v.sandbox || {}) as { bash?: string; network?: boolean; workspaceRoot?: string; allowWrite?: string[] }
      setDraft({
        defaultModel: (v.defaultModel as string) || '',
        subagentModel: (v.subagentModel as string) || '',
        systemPrompt: agent.systemPrompt || '',
        temperature: agent.temperature || 0,
        subagentTemperature: agent.subagentTemperature || 0,
        maxSteps: typeof agent.maxSteps === 'number' ? agent.maxSteps : 0,
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

  const providers = (view.providers || []) as Array<{ name?: string; models?: string[] }>
  const modelOptions = providers.flatMap((p) => (p.models || []).map((m: string) => ({ value: m, label: `${p.name} / ${m}` })))

  const handleSave = async () => {
    setSaving(true)
    try {
      await App.GaeaSaveSettings({
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
          allowWrite: ((view.sandbox as { allowWrite?: string[] } | undefined)?.allowWrite) || [],
        },
      } as AppModels.SettingsView)
      message.success(t('settings.office.saved'))
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : t('settings.saveFailed'))
    } finally {
      setSaving(false)
    }
  }

  // handleReload 从磁盘重新读取引擎配置并重建控制器——外部直接编辑
  // config.toml 或技能/工具目录后，无需重启桌面端即可生效。
  const handleReload = async () => {
    setReloading(true)
    try {
      const res = await App.GaeaReload()
      message.success(t('caps.reloaded', { tools: res.tools, skills: res.skills }))
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : t('settings.office.reloadFailed'))
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
        title={t('settings.office.title')}
        desc={t('settings.office.desc')}
        instant
      >
        {/* 模型区 */}
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, minmax(0, 1fr))', gap: '0 16px' }}>
          {field(t('settings.office.defaultModel'), (
            <Select
              placeholder={t('settings.office.defaultModelPh')} value={draft.defaultModel || undefined}
              onChange={(v) => setDraft({ ...draft, defaultModel: v })}
              options={modelOptions} style={selectStyle} size="small"
              showSearch optionFilterProp="label"
            />
          ))}
          {field(t('settings.office.maxSteps'), (
            <InputNumber size="small" min={0} max={100} value={draft.maxSteps}
              onChange={(v) => setDraft({ ...draft, maxSteps: v || 0 })} style={selectStyle} />
          ), t('settings.office.unlimited0'))}
          {field(t('settings.office.subModel'), (
            <Input size="small" placeholder={t('settings.office.subModelPh')} value={draft.subagentModel}
              onChange={(e) => setDraft({ ...draft, subagentModel: e.target.value })} />
          ))}
        </div>
      </SettingsSection>

      <SettingsSection
        title={t('settings.office.agentTitle')}
        desc={t('settings.office.agentDesc')}
      >
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, minmax(0, 1fr))', gap: '0 16px' }}>
          {field(t('settings.office.temp'), (
            <InputNumber size="small" min={0} max={2} step={0.05} value={draft.temperature}
              onChange={(v) => setDraft({ ...draft, temperature: v || 0 })} style={selectStyle} />
          ))}
          {field(t('settings.office.subTemp'), (
            <InputNumber size="small" min={0} max={2} step={0.05} value={draft.subagentTemperature}
              onChange={(v) => setDraft({ ...draft, subagentTemperature: v || 0 })} style={selectStyle} />
          ))}
          {field(t('settings.office.effort'), (
            <Select size="small" allowClear placeholder={t('settings.office.effortPh')}
              value={draft.effort || undefined}
              onChange={(v) => setDraft({ ...draft, effort: v || '' })}
              options={[{ value: 'high', label: 'high' }, { value: 'max', label: 'max' }]} style={selectStyle} />
          ))}
          {field(t('settings.office.subEffort'), (
            <Select size="small" allowClear placeholder={t('settings.office.subEffortPh')}
              value={draft.subagentEffort || undefined}
              onChange={(v) => setDraft({ ...draft, subagentEffort: v || '' })}
              options={[{ value: 'high', label: 'high' }, { value: 'max', label: 'max' }]} style={selectStyle} />
          ))}
        </div>
        {field(t('settings.office.systemPrompt'), (
          <Input.TextArea rows={3} value={draft.systemPrompt}
            onChange={(e) => setDraft({ ...draft, systemPrompt: e.target.value })}
            style={{ background: 'var(--md-sys-color-surface-container)', border: '1px solid var(--md-sys-color-outline-variant)' }} />
        ))}
      </SettingsSection>

      <SettingsSection
        title={t('settings.office.permTitle')}
        desc={t('settings.office.permDesc')}
      >
        {field(t('settings.office.permMode'), (
          <Select size="small" value={draft.permMode}
            onChange={(v) => setDraft({ ...draft, permMode: v })}
            options={[
              { value: 'ask', label: t('settings.office.modeAsk') },
              { value: 'allow', label: t('settings.office.modeAllow') },
              { value: 'deny', label: t('settings.office.modeDeny') },
            ]} style={selectStyle} />
        ))}
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, minmax(0, 1fr))', gap: '0 16px' }}>
          {field(t('settings.office.ruleAllow'), <Input size="small" value={draft.permAllow} onChange={(e) => setDraft({ ...draft, permAllow: e.target.value })} placeholder={t('settings.office.ruleAllowPh')} />)}
          {field(t('settings.office.ruleAsk'), <Input size="small" value={draft.permAsk} onChange={(e) => setDraft({ ...draft, permAsk: e.target.value })} placeholder={t('settings.office.ruleAskPh')} />)}
          {field(t('settings.office.ruleDeny'), <Input size="small" value={draft.permDeny} onChange={(e) => setDraft({ ...draft, permDeny: e.target.value })} placeholder={t('settings.office.ruleDenyPh')} />)}
        </div>
      </SettingsSection>

      <SettingsSection
        title={t('settings.office.sandboxTitle')}
        desc={t('settings.office.sandboxDesc')}
      >
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, minmax(0, 1fr))', gap: '0 16px' }}>
          {field(t('settings.office.bashMode'), (
            <Select size="small" value={draft.sandboxBash}
              onChange={(v) => setDraft({ ...draft, sandboxBash: v })}
              options={[
                { value: 'enforce', label: t('settings.office.sandboxEnforce') },
                { value: 'off', label: t('settings.office.sandboxOff') },
              ]} style={selectStyle} />
          ))}
          {field(t('settings.office.network'), (
            <Space size={8}>
              <Switch checked={draft.sandboxNetwork} onChange={(v) => setDraft({ ...draft, sandboxNetwork: v })} />
              <Typography.Text style={{ fontSize: 12, color: 'var(--md-sys-color-text-secondary)' }}>
                {draft.sandboxNetwork ? t('settings.office.netOn') : t('settings.office.netOff')}
              </Typography.Text>
            </Space>
          ))}
        </div>
        {field(t('settings.office.workspaceRoot'), (
          <Input size="small" value={draft.workspaceRoot}
            onChange={(e) => setDraft({ ...draft, workspaceRoot: e.target.value })}
            placeholder={t('settings.office.workspaceRootPh')} />
        ))}
      </SettingsSection>

      <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8, marginTop: 4 }}>
        <Button icon={<ReloadOutlined />} loading={reloading} onClick={handleReload}>
          {t('settings.office.reload')}
        </Button>
        <Button type="primary" icon={<SaveOutlined />} loading={saving} onClick={handleSave}
          style={{ background: 'var(--md-sys-color-primary)', borderColor: 'var(--md-sys-color-primary)', borderRadius: 'var(--md-sys-radius-md)' }}>
          {t('settings.office.save')}
        </Button>
      </div>

      {/* v4.1c → v4.6.1 中文规范体检（规范包机制化：红头要素 + 造价工程表式） */}
      <SettingsSection title={t('settings.office.lintTitle')}>
        <div style={{ display: 'flex', gap: 8 }}>
          <Input
            placeholder={t('settings.office.lintPh')}
            value={lintPath}
            onChange={(e) => setLintPath(e.target.value)}
            onPressEnter={() => void runLint()}
            style={{ flex: 1 }}
          />
          <Button icon={<FileTextOutlined />} loading={linting} onClick={() => void runLint()}>
            {t('settings.office.lintRun')}
          </Button>
        </div>
        {lintReport && (
          <div style={{ marginTop: 8, fontSize: 12, lineHeight: 1.8 }}>
            <Typography.Text strong style={{ color: lintReport.passed ? 'var(--md-sys-color-success)' : 'var(--md-sys-color-warning)' }}>
              {lintReport.summary}
            </Typography.Text>
            <ul style={{ margin: '6px 0 0', paddingLeft: 18 }}>
              {lintReport.issues.map((it) => (
                <li key={`${it.spec ?? '通用'}:${it.element}`} style={{ color: it.found ? 'var(--md-sys-color-text-secondary)' : 'var(--md-sys-color-text)' }}>
                  {it.spec && (
                    <span
                      style={{
                        marginRight: 6,
                        fontSize: 10,
                        padding: '0 5px',
                        borderRadius: 4,
                        color: 'var(--md-sys-color-primary)',
                        background: 'color-mix(in srgb, var(--md-sys-color-primary) 10%, transparent)',
                      }}
                    >
                      {it.spec}
                    </span>
                  )}
                  {it.element}：{it.found ? t('settings.office.lintOk') : <span style={{ color: 'var(--md-sys-color-destructive)' }}>{it.note}</span>}
                </li>
              ))}
            </ul>
          </div>
        )}
      </SettingsSection>

      {loading && <Typography.Text style={{ color: 'var(--md-sys-color-text-secondary)', fontSize: 12 }}>{t('common.loading')}</Typography.Text>}
    </>
  )
}

export default OfficePanel
