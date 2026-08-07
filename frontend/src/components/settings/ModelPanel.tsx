import React, { useCallback, useEffect, useState } from 'react'
import { Button, Select, Tag, Typography, message } from 'antd'
import { ApiOutlined, ArrowRightOutlined } from '@ant-design/icons'
import { getActiveModel, getConfig, saveConfig } from '../../api/settings'
import SettingsSection from './SettingsSection'

/** ModelPanel — 全局模型设置：当前模型 + 推理强度（引擎管理在「模型中心」） */
const ModelPanel: React.FC = () => {
  const [activeModel, setActiveModel] = useState('')
  const [effort, setEffort] = useState('')

  const load = useCallback(async () => {
    try {
      const [am, cfg] = await Promise.all([getActiveModel(), getConfig()])
      setActiveModel(am || '')
      setEffort(cfg.reasoning_effort || '')
    } catch { /* 引擎未就绪时静默 */ }
  }, [])

  useEffect(() => { load() }, [load])

  const handleEffort = async (v: string) => {
    setEffort(v)
    try {
      await saveConfig('reasoning_effort', v)
      message.success('推理强度已更新')
    } catch (err: any) {
      message.error(err?.message || '保存失败')
    }
  }

  const goModelCenter = () => {
    window.dispatchEvent(new CustomEvent('navigate', { detail: { page: 'modelcenter' } }))
  }

  return (
    <>
      <SettingsSection
        icon={<ApiOutlined />}
        title="当前模型"
        desc="全局 AI 助手实际使用的推理引擎与模型；引擎启停、密钥与功能绑定请在「模型中心」管理。"
      >
        <div style={{
          display: 'flex', alignItems: 'center', gap: 12, flexWrap: 'wrap',
          padding: '12px 14px', borderRadius: 'var(--md-sys-radius-md)',
          background: 'var(--md-sys-color-surface-container)',
          border: '1px solid var(--md-sys-color-outline-variant)',
        }}>
          <span className="live-dot" />
          <Typography.Text strong style={{ fontSize: 15, color: 'var(--md-sys-color-text)' }}>
            {activeModel || '未配置'}
          </Typography.Text>
          <span style={{ flex: 1 }} />
          <Button size="small" icon={<ArrowRightOutlined />} onClick={goModelCenter}>前往模型中心</Button>
        </div>
      </SettingsSection>

      <SettingsSection
        icon={<span style={{ fontSize: 15 }}>🧠</span>}
        title="推理强度"
        desc="控制 AI 回答的思考深度：低 = 快速响应，高 = 更深入的分析；留空 = 提供方默认。"
        instant
      >
        <Select
          value={effort || undefined}
          placeholder="选择推理强度"
          onChange={handleEffort}
          style={{ width: 220 }}
          allowClear
          options={[
            { value: 'low', label: '⚡ 低 — 快速响应' },
            { value: 'medium', label: '⚖️ 中 — 均衡' },
            { value: 'high', label: '🛰️ 高 — 深度思考' },
          ]}
        />
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginTop: 12 }}>
          <Tag style={{ margin: 0, fontSize: 11 }}>deepseek</Tag>
          <Typography.Text style={{ fontSize: 11, color: 'var(--md-sys-color-text-secondary)' }}>
            推理强度对支持该参数的引擎生效（如 DeepSeek：high / max）
          </Typography.Text>
        </div>
      </SettingsSection>
    </>
  )
}

export default ModelPanel
