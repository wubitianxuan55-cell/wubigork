import React, { useCallback, useEffect, useState } from 'react'
import { Button, InputNumber, Select, Tag, Typography, message } from 'antd'
import { ApiOutlined, ArrowRightOutlined } from '@ant-design/icons'
import { getActiveModel, getConfig, saveConfig } from '../../api/settings'
import { getUsdCnyRate, setUsdCnyRate } from '../../api/engines'
import SettingsSection from './SettingsSection'

// 默认美元→人民币汇率（与后端 internal/config usd_cny_rate 默认值一致）
const DEFAULT_USD_CNY = 7.2

/** ModelPanel — 全局模型设置：当前模型 + 推理强度 + 费用汇率（引擎管理在「模型中心」） */
const ModelPanel: React.FC = () => {
  const [activeModel, setActiveModel] = useState('')
  const [effort, setEffort] = useState('')
  const [rate, setRate] = useState<number | null>(null)

  const load = useCallback(async () => {
    try {
      const [am, cfg] = await Promise.all([getActiveModel(), getConfig()])
      setActiveModel(am || '')
      setEffort(cfg.reasoning_effort || '')
    } catch { /* 引擎未就绪时静默 */ }
    // T6-6.2：加载汇率回填；未配置/失败时回退默认值 7.2（保存时仍会显式校验）
    try {
      const r = await getUsdCnyRate()
      setRate(Number.isFinite(r) && r > 0 ? r : DEFAULT_USD_CNY)
    } catch { /* 引擎未就绪时回退默认汇率 7.2 */ }
  }, [])

  useEffect(() => { load() }, [load])

  const handleEffort = async (v: string) => {
    setEffort(v)
    try {
      await saveConfig('reasoning_effort', v)
      message.success('推理强度已更新')
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '保存失败')
    }
  }

  // T6-6.2：保存汇率 — 校验正数；错误提示不静默
  const handleSaveRate = async () => {
    const v = Number(rate)
    if (!Number.isFinite(v) || v <= 0) {
      message.error('汇率必须是大于 0 的数字')
      return
    }
    try {
      await setUsdCnyRate(v)
      setRate(v)
      message.success('美元→人民币汇率已保存，费用折算即时生效')
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '汇率保存失败')
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

      <SettingsSection
        icon={<span style={{ fontSize: 15 }}>💱</span>}
        title="美元→人民币汇率"
        desc="模型调用费用按此汇率折算为人民币展示（模型中心「调用统计」同步生效）；仅接受大于 0 的数值。"
        instant
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 12, flexWrap: 'wrap' }}>
          <InputNumber
            aria-label="美元人民币汇率"
            min={0.01}
            step={0.1}
            precision={2}
            value={rate ?? undefined}
            onChange={(v) => setRate(v === null || v === undefined ? null : Number(v))}
            placeholder="7.2"
            style={{ width: 160 }}
          />
          <Button type="primary" size="small" onClick={handleSaveRate}>保存汇率</Button>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginTop: 12 }}>
          <Tag style={{ margin: 0, fontSize: 11 }}>USD → CNY</Tag>
          <Typography.Text style={{ fontSize: 11, color: 'var(--md-sys-color-text-secondary)' }}>
            汇率仅接受大于 0 的数值；保存失败会提示具体原因
          </Typography.Text>
        </div>
      </SettingsSection>
    </>
  )
}

export default ModelPanel
