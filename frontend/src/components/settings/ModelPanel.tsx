import React, { useCallback, useEffect, useState } from 'react'
import { Button, InputNumber, Select, Tag, Typography, message } from 'antd'
import { ApiOutlined, ArrowRightOutlined, RobotOutlined, MoneyCollectOutlined } from '@ant-design/icons'
import { getActiveModel, getConfig, saveConfig } from '../../api/settings'
import { getUsdCnyRate, setUsdCnyRate } from '../../api/engines'
import SettingsSection from './SettingsSection'
import { useT } from '../../gaea/lib/i18n'

// 默认美元→人民币汇率（与后端 internal/config usd_cny_rate 默认值一致）
const DEFAULT_USD_CNY = 7.2

/** ModelPanel — 全局模型设置：当前模型 + 推理强度 + 费用汇率（引擎管理在「模型中心」） */
const ModelPanel: React.FC = () => {
  const t = useT()
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
      message.success(t('settings.model.effortSaved'))
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : t('settings.saveFailed'))
    }
  }

  // T6-6.2：保存汇率 — 校验正数；错误提示不静默
  const handleSaveRate = async () => {
    const v = Number(rate)
    if (!Number.isFinite(v) || v <= 0) {
      message.error(t('settings.model.rateInvalid'))
      return
    }
    try {
      await setUsdCnyRate(v)
      setRate(v)
      message.success(t('settings.model.rateSaved'))
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : t('settings.model.rateSaveFailed'))
    }
  }

  const goModelCenter = () => {
    window.dispatchEvent(new CustomEvent('navigate', { detail: { page: 'modelcenter' } }))
  }

  return (
    <>
      <SettingsSection
        icon={<ApiOutlined />}
        title={t('settings.model.currentTitle')}
        desc={t('settings.model.currentDesc')}
      >
        <div style={{
          display: 'flex', alignItems: 'center', gap: 12, flexWrap: 'wrap',
          padding: '12px 14px', borderRadius: 'var(--md-sys-radius-md)',
          background: 'var(--md-sys-color-surface-container)',
          border: '1px solid var(--md-sys-color-outline-variant)',
        }}>
          <span className="live-dot" />
          <Typography.Text strong style={{ fontSize: 15, color: 'var(--md-sys-color-text)' }}>
            {activeModel || t('settings.model.notConfigured')}
          </Typography.Text>
          <span style={{ flex: 1 }} />
          <Button size="small" icon={<ArrowRightOutlined />} onClick={goModelCenter}>{t('settings.model.goModelCenter')}</Button>
        </div>
      </SettingsSection>

      <SettingsSection
        icon={<span style={{ fontSize: 15 }}><RobotOutlined /></span>}
        title={t('settings.model.effortTitle')}
        desc={t('settings.model.effortDesc')}
        instant
      >
        <Select
          value={effort || undefined}
          placeholder={t('settings.model.effortPlaceholder')}
          onChange={handleEffort}
          style={{ width: 220 }}
          allowClear
          options={[
            { value: 'low', label: t('settings.model.effortLow') },
            { value: 'medium', label: t('settings.model.effortMedium') },
            { value: 'high', label: t('settings.model.effortHigh') },
          ]}
        />
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginTop: 12 }}>
          <Tag style={{ margin: 0, fontSize: 11 }}>deepseek</Tag>
          <Typography.Text style={{ fontSize: 11, color: 'var(--md-sys-color-text-secondary)' }}>
            {t('settings.model.effortNote')}
          </Typography.Text>
        </div>
      </SettingsSection>

      <SettingsSection
        icon={<span style={{ fontSize: 15 }}><MoneyCollectOutlined /></span>}
        title={t('settings.model.rateTitle')}
        desc={t('settings.model.rateDesc')}
        instant
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 12, flexWrap: 'wrap' }}>
          <InputNumber
            aria-label={t('settings.model.rateAria')}
            min={0.01}
            step={0.1}
            precision={2}
            value={rate ?? undefined}
            onChange={(v) => setRate(v === null || v === undefined ? null : Number(v))}
            placeholder="7.2"
            style={{ width: 160 }}
          />
          <Button type="primary" size="small" onClick={handleSaveRate}>{t('settings.model.rateSave')}</Button>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginTop: 12 }}>
          <Tag style={{ margin: 0, fontSize: 11 }}>USD → CNY</Tag>
          <Typography.Text style={{ fontSize: 11, color: 'var(--md-sys-color-text-secondary)' }}>
            {t('settings.model.rateNote')}
          </Typography.Text>
        </div>
      </SettingsSection>
    </>
  )
}

export default ModelPanel
