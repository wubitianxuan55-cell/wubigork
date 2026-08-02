import React, { useCallback, useEffect, useState } from 'react'
import { Button, Input, Select, Switch, Tag, Typography, message, Space, Tooltip } from 'antd'
import { ApiOutlined, CheckCircleOutlined, ExperimentOutlined, ThunderboltOutlined } from '@ant-design/icons'
import { getEngines, saveEngine, testEngineConnection, type EngineConfig } from '../../api/engines'
import { getActiveModel, getConfig, saveConfig } from '../../api/settings'
import SettingsSection from './SettingsSection'

const typeLabels: Record<string, string> = {
  xai: 'xAI', ollama: 'Ollama', herdsman: 'Herdsman', deepseek: 'DeepSeek',
}

/** EnginePanel — 模型引擎设置：激活模型 + 引擎列表 + 推理强度 */
const EnginePanel: React.FC = () => {
  const [engines, setEngines] = useState<EngineConfig[]>([])
  const [activeModel, setActiveModel] = useState('')
  const [effort, setEffort] = useState('')
  const [loading, setLoading] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const [es, am, cfg] = await Promise.all([getEngines(), getActiveModel(), getConfig()])
      setEngines(es || [])
      setActiveModel(am || '')
      setEffort(cfg.reasoning_effort || '')
    } catch { /* 引擎未就绪时静默 */ }
    setLoading(false)
  }, [])

  useEffect(() => { load() }, [load])

  const handleToggle = async (eng: EngineConfig, enabled: boolean) => {
    const next = { ...eng, enabled }
    try {
      await saveEngine(next)
      setEngines((prev) => prev.map((e) => (e.id === eng.id ? next : e)))
      if (enabled) message.success(`已启用 ${eng.name}`)
    } catch (err: any) { message.error(err?.message || '保存失败') }
  }

  const handleSave = async (eng: EngineConfig) => {
    try {
      await saveEngine(eng)
      setEngines((prev) => prev.map((e) => (e.id === eng.id ? eng : e)))
      message.success(`已保存 ${eng.name}`)
    } catch (err: any) { message.error(err?.message || '保存失败') }
  }

  const handleTest = async (id: string) => {
    try {
      const st = await testEngineConnection(id)
      if (st.connected) message.success(`连接成功 · ${st.model_count} 个模型`)
      else message.warning(st.error || '连接失败')
    } catch (err: any) { message.error(err?.message || '测试失败') }
  }

  const handleEffort = async (v: string) => {
    setEffort(v)
    try { await saveConfig('reasoning_effort', v); message.success('推理强度已更新') }
    catch (err: any) { message.error(err?.message || '保存失败') }
  }

  return (
    <>
      {/* 当前激活模型 */}
      <SettingsSection
        title={<>当前 AI 模型</>}
        desc="顶栏与 AI 助手实际使用的推理引擎。可在模型中心切换。"
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
          <span className="live-dot" />
          <Typography.Text strong style={{ fontSize: 15, color: 'var(--gaea-glow)', textShadow: '0 0 12px var(--gaea-glow)' }}>
            {activeModel || '未配置'}
          </Typography.Text>
          <Typography.Text style={{ fontSize: 12, color: 'var(--md-sys-color-text-secondary)' }}>
            在线
          </Typography.Text>
        </div>
      </SettingsSection>

      {/* 推理强度 */}
      <SettingsSection
        title={<>推理强度</>}
        desc="控制 AI 回答的思考深度。低 = 快速响应，高 = 更深入的分析。"
      >
        <Select
          value={effort || undefined}
          placeholder="选择推理强度"
          onChange={handleEffort}
          style={{ width: 220 }}
          options={[
            { value: 'low', label: '⚡ 低 — 快速响应' },
            { value: 'medium', label: '⚖️ 中 — 平衡' },
            { value: 'high', label: '🧠 高 — 深度思考' },
          ]}
        />
      </SettingsSection>

      {/* 引擎列表 */}
      <SettingsSection
        title={<>引擎配置</>}
        desc="管理模型引擎：启用状态、API 地址与默认模型。"
        instant
      >
        {engines.length === 0 && !loading && (
          <Typography.Text style={{ color: 'var(--md-sys-color-text-secondary)', fontSize: 12 }}>
            暂无引擎配置
          </Typography.Text>
        )}
        <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
          {engines.map((eng) => (
            <div
              key={eng.id}
              className="md-glass"
              style={{
                padding: '12px 14px', borderRadius: 'var(--md-sys-radius-md)',
                border: '1px solid var(--md-sys-color-outline-variant)',
                display: 'flex', flexDirection: 'column', gap: 10,
                opacity: eng.enabled ? 1 : 0.55,
              }}
            >
              <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <ApiOutlined style={{ color: 'var(--gaea-glow)' }} />
                <Typography.Text strong style={{ fontSize: 13, color: 'var(--md-sys-color-text)' }}>{eng.name}</Typography.Text>
                <Tag style={{ fontSize: 10, margin: 0 }}>{typeLabels[eng.type] || eng.type}</Tag>
                <span style={{ flex: 1 }} />
                <Tooltip title={eng.enabled ? '点击禁用' : '点击启用'}>
                  <Switch size="small" checked={eng.enabled} onChange={(v) => handleToggle(eng, v)} />
                </Tooltip>
                <Tooltip title="测试连接">
                  <Button size="small" icon={<ExperimentOutlined />} onClick={() => handleTest(eng.id)} style={{ fontSize: 11 }} />
                </Tooltip>
              </div>

              <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', alignItems: 'center' }}>
                <Input
                  size="small" value={eng.base_url}
                  placeholder="API 地址"
                  onChange={(e) => setEngines((prev) => prev.map((x) => (x.id === eng.id ? { ...x, base_url: e.target.value } : x)))}
                  style={{ flex: 2, minWidth: 200, background: 'var(--md-sys-color-surface-container)', border: '1px solid var(--md-sys-color-outline-variant)', color: 'var(--md-sys-color-text)' }}
                />
                <Select
                  size="small" value={eng.default_model || undefined}
                  placeholder="默认模型"
                  onChange={(v) => setEngines((prev) => prev.map((x) => (x.id === eng.id ? { ...x, default_model: v } : x)))}
                  style={{ flex: 1, minWidth: 140 }}
                  options={(eng.models || []).map((m) => ({ value: m.id, label: m.id }))}
                  showSearch optionFilterProp="label"
                />
                <Button
                  size="small" type="primary" icon={<CheckCircleOutlined />}
                  onClick={() => handleSave(eng)}
                  style={{ fontSize: 11, background: 'var(--md-sys-color-primary)', borderColor: 'var(--md-sys-color-primary)' }}
                >保存</Button>
              </div>
            </div>
          ))}
        </div>
        <Space size={6} style={{ marginTop: 10 }}>
          <ThunderboltOutlined style={{ color: 'var(--gaea-glow)' }} />
          <Typography.Text style={{ color: 'var(--md-sys-color-text-secondary)', fontSize: 11 }}>
            完整引擎管理请前往「模型中心」
          </Typography.Text>
        </Space>
      </SettingsSection>
    </>
  )
}

export default EnginePanel
