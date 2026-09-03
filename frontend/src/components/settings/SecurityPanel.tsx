import React, { useCallback, useEffect, useState } from 'react'
import { Button, Space, Switch, Tag, Typography, message } from 'antd'
import { ReloadOutlined, SafetyCertificateOutlined, LockOutlined, BugOutlined } from '@ant-design/icons'
import SettingsSection from './SettingsSection'
import { useT } from '../../gaea/lib/i18n'
import type { AppFacade } from '../../types/wails'

/**
 * SecurityPanel — 安全设置（阶段 2 安全收敛 S2-1 / S2-2 / S2-4）
 *
 *  - 敏感域本地化（S2-4/D8）：成本/报价类 AI 操作默认路由本地 Herdsman，
 *    数据不出本机；可切换回云端。
 *  - Herdsman LAN 暴露（S2-1）：实时检测结果 + 处置指引（与全局安全横幅联动）。
 *  - WebView2 远程调试（S2-2）：说明 env 开关（GAEA_WEBVIEW_DEBUG），默认关闭。
 */

interface LanExposure {
  config_path: string
  exposed: boolean
  port: number
  config_missing: boolean
  parse_error?: string
  guidance?: string
}

export const SecurityPanel: React.FC = () => {
  const go = window.go?.app?.App as AppFacade
  const t = useT()
  const [sensitiveLocal, setSensitiveLocal] = useState<boolean>(true)
  const [sensitiveLoading, setSensitiveLoading] = useState(true)
  const [officeLocal, setOfficeLocal] = useState<boolean>(true)
  const [officeLoading, setOfficeLoading] = useState(true)
  const [offlineMode, setOfflineMode] = useState<boolean>(false)
  const [offlineLoading, setOfflineLoading] = useState(true)
  const [exposure, setExposure] = useState<LanExposure | null>(null)
  const [checking, setChecking] = useState(false)

  const load = useCallback(async () => {
    try {
      // S2-4：敏感域本地化开关（后端 GetSensitiveLocal/SetSensitiveLocal）
      if (typeof go?.GetSensitiveLocal === 'function') {
        setSensitiveLocal(!!(await go.GetSensitiveLocal()))
      }
      if (typeof go?.GetOfficeLocal === 'function') {
        setOfficeLocal(!!(await go.GetOfficeLocal()))
      }
      if (typeof go?.GetOfflineMode === 'function') {
        setOfflineMode(!!(await go.GetOfflineMode()))
      }
    } catch {
      /* 后端未就绪时保持默认 */
    } finally {
      setSensitiveLoading(false)
      setOfficeLoading(false)
      setOfflineLoading(false)
    }
  }, [go])

  useEffect(() => { void load() }, [load])

  const handleToggleSensitive = async (v: boolean) => {
    const prev = sensitiveLocal
    setSensitiveLocal(v)
    try {
      await go?.SetSensitiveLocal?.(v)
      message.success(v ? t('settings.security.sensitiveOn') : t('settings.security.sensitiveOff'))
    } catch (err: unknown) {
      setSensitiveLocal(prev)
      message.error(err instanceof Error ? err.message : t('settings.saveFailed'))
    }
  }

  const handleToggleOffice = async (v: boolean) => {
    const prev = officeLocal
    setOfficeLocal(v)
    try {
      await go?.SetOfficeLocal?.(v)
      message.success(v ? t('settings.security.officeOn') : t('settings.security.officeOff'))
    } catch (err: unknown) {
      setOfficeLocal(prev)
      message.error(err instanceof Error ? err.message : t('settings.saveFailed'))
    }
  }

  const handleToggleOffline = async (v: boolean) => {
    const prev = offlineMode
    setOfflineMode(v)
    try {
      await go?.SetOfflineMode?.(v)
      message.success(v ? t('settings.security.offlineOnToast') : t('settings.security.offlineOffToast'))
    } catch (err: unknown) {
      setOfflineMode(prev)
      message.error(err instanceof Error ? err.message : t('settings.saveFailed'))
    }
  }

  const check = useCallback(async () => {
    setChecking(true)
    try {
      const res: LanExposure | undefined = await go?.HerdsmanSecurityCheck?.()
      if (res) setExposure(res)
    } catch {
      setExposure(null)
    } finally {
      setChecking(false)
    }
  }, [go])

  useEffect(() => { void check() }, [check])

  return (
    <>
      <SettingsSection
        icon={<LockOutlined />}
        title={t('settings.security.sensitiveTitle')}
        desc={t('settings.security.sensitiveDesc')}
        instant
      >
        <Space size={12}>
          <Switch
            checked={sensitiveLocal}
            loading={sensitiveLoading}
            onChange={handleToggleSensitive}
          />
          <Typography.Text style={{ fontSize: 13, color: 'var(--md-sys-color-text)' }}>
            {sensitiveLocal ? t('settings.security.localPreferred') : t('settings.security.normalRoute')}
          </Typography.Text>
        </Space>
        <div style={{ fontSize: 11, color: 'var(--md-sys-color-text-secondary)', marginTop: 6, opacity: 0.8 }}>
          {t('settings.security.sensitiveScope')}
        </div>
      </SettingsSection>

      <SettingsSection
        icon={<LockOutlined />}
        title={t('settings.security.officeTitle')}
        desc={t('settings.security.officeDesc')}
        instant
      >
        <Space size={12}>
          <Switch
            checked={officeLocal}
            loading={officeLoading}
            onChange={handleToggleOffice}
          />
          <Typography.Text style={{ fontSize: 13, color: 'var(--md-sys-color-text)' }}>
            {officeLocal ? t('settings.security.localPreferred') : t('settings.security.normalRoute')}
          </Typography.Text>
        </Space>
        <div style={{ fontSize: 11, color: 'var(--md-sys-color-text-secondary)', marginTop: 6, opacity: 0.8 }}>
          {t('settings.security.officeScope')}
        </div>
      </SettingsSection>

      <SettingsSection
        icon={<LockOutlined />}
        title={t('settings.security.offlineTitle')}
        desc={t('settings.security.offlineDesc')}
        instant
      >
        <Space size={12}>
          <Switch
            checked={offlineMode}
            loading={offlineLoading}
            onChange={handleToggleOffline}
          />
          <Typography.Text style={{ fontSize: 13, color: 'var(--md-sys-color-text)' }}>
            {offlineMode ? t('settings.security.offlineOn') : t('settings.security.offlineOff')}
          </Typography.Text>
        </Space>
        <div style={{ fontSize: 11, color: 'var(--md-sys-color-text-secondary)', marginTop: 6, opacity: 0.8 }}>
          {t('settings.security.offlineNote')}
        </div>
      </SettingsSection>

      <SettingsSection
        icon={<SafetyCertificateOutlined />}
        title={t('settings.security.lanTitle')}
        desc={t('settings.security.lanDesc')}
      >
        <Space size={8} wrap>
          {exposure?.exposed ? (
            <Tag color="error">{t('settings.security.lanExposed', { port: exposure.port })}</Tag>
          ) : exposure ? (
            <Tag color="success">{t('settings.security.lanSafe')}</Tag>
          ) : (
            <Tag>{t('settings.security.lanNoConfig')}</Tag>
          )}
          <Button size="small" icon={<ReloadOutlined spin={checking} />} loading={checking} onClick={() => void check()}>
            {t('settings.security.recheck')}
          </Button>
        </Space>
        {exposure?.exposed && (
          <div style={{
            marginTop: 8, padding: '10px 12px', borderRadius: 'var(--md-sys-radius-md)',
            background: 'var(--md-sys-color-surface-container)',
            border: '1px solid var(--md-sys-color-outline-variant)',
            fontSize: 12, lineHeight: 1.7, color: 'var(--md-sys-color-text)',
          }}>
            {exposure.guidance || t('settings.security.lanGuidance', { path: exposure.config_path })}
          </div>
        )}
        {exposure?.config_missing && (
          <div style={{ marginTop: 8, fontSize: 11, color: 'var(--md-sys-color-text-secondary)' }}>
            {exposure.parse_error}
          </div>
        )}
      </SettingsSection>

      <SettingsSection
        icon={<BugOutlined />}
        title={t('settings.security.debugTitle')}
        desc={t('settings.security.debugDesc')}
      >
        <Typography.Text style={{ fontSize: 12, color: 'var(--md-sys-color-text-secondary)' }}>
          {t('settings.security.debugNote')}
        </Typography.Text>
      </SettingsSection>
    </>
  )
}

export default SecurityPanel
