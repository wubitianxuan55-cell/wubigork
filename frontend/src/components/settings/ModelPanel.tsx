import React, { useCallback, useEffect, useState } from 'react'
import { Button, InputNumber, Select, Typography, message } from 'antd'
import {
  ApiOutlined, ArrowRightOutlined, MoneyCollectOutlined, RobotOutlined,
} from '@ant-design/icons'
import { getActiveModel, getConfig, saveConfig } from '../../api/settings'
import { getUsdCnyRate, setUsdCnyRate } from '../../api/engines'
import SettingsSection from './SettingsSection'
import { useT } from '../../gaea/lib/i18n'

// 默认美元→人民币汇率（与后端 internal/config usd_cny_rate 默认值一致）
const DEFAULT_USD_CNY = 7.2

/** ModelPanel — 模型任务页签的概览：当前模型 + 推理强度 + 费用汇率。
 *  模型一览、功能绑定与引擎管理归模型中心；设置侧只承载全局参数和低频策略。 */
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
    } catch {
      // 引擎未就绪时仍展示可用默认值；保存操作会显式校验并提示失败原因。
      setRate(DEFAULT_USD_CNY)
    }
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
        <div className={`model-current-card${activeModel ? ' is-configured' : ' is-empty'}`}>
          <span className="model-current-card__dot" aria-hidden="true" />
          <div className="model-current-card__copy">
            <span className="model-current-card__eyebrow">全局路由</span>
            <Typography.Text strong className="model-current-card__name">
              {activeModel || t('settings.model.notConfigured')}
            </Typography.Text>
            <span className="model-current-card__meta">
              {activeModel ? '当前对话与通用任务使用此模型' : '完成功能绑定后，这里会显示全局活动模型'}
            </span>
          </div>
          <div className="model-current-card__actions">
            <Button size="small" icon={<ArrowRightOutlined />} onClick={goModelCenter}>
              {t('settings.model.goModelCenter')}
            </Button>
          </div>
        </div>
        <div className="model-signal-grid" aria-label="模型概览">
          <div className="model-signal">
            <span>配置状态</span>
            <strong>{activeModel ? '已配置' : '待配置'}</strong>
          </div>
          <div className="model-signal">
            <span>推理策略</span>
            <strong>{effort || '提供方默认'}</strong>
          </div>
          <div className="model-signal">
            <span>费用折算</span>
            <strong>{rate == null ? '读取中' : `1 USD ≈ ¥${rate.toFixed(2)}`}</strong>
          </div>
        </div>
      </SettingsSection>

      <div className="settings-model-controls">
        <SettingsSection
          icon={<RobotOutlined />}
          title={t('settings.model.effortTitle')}
          desc={t('settings.model.effortDesc')}
          instant
        >
          <div className="model-control-row">
            <Select
              value={effort || undefined}
              placeholder={t('settings.model.effortPlaceholder')}
              onChange={handleEffort}
              className="model-control-row__select"
              allowClear
              options={[
                { value: 'low', label: t('settings.model.effortLow') },
                { value: 'medium', label: t('settings.model.effortMedium') },
                { value: 'high', label: t('settings.model.effortHigh') },
              ]}
            />
            <span className="model-control-tag">deepseek</span>
          </div>
          <Typography.Text className="model-control-note">
            {t('settings.model.effortNote')}
          </Typography.Text>
        </SettingsSection>

        <SettingsSection
          icon={<MoneyCollectOutlined />}
          title={t('settings.model.rateTitle')}
          desc={t('settings.model.rateDesc')}
          instant
        >
          <div className="model-control-row">
            <InputNumber
              aria-label={t('settings.model.rateAria')}
              min={0.01}
              step={0.1}
              precision={2}
              value={rate ?? undefined}
              onChange={(v) => setRate(v === null || v === undefined ? null : Number(v))}
              placeholder="7.2"
              className="model-control-row__number"
            />
            <Button type="primary" size="small" onClick={handleSaveRate}>{t('settings.model.rateSave')}</Button>
            <span className="model-control-tag">USD → CNY</span>
          </div>
          <Typography.Text className="model-control-note">
            {t('settings.model.rateNote')}
          </Typography.Text>
        </SettingsSection>
      </div>
    </>
  )
}

export default ModelPanel
