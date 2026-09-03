import React, { useEffect, useState } from 'react'
import { Collapse, Tag, Typography } from 'antd'
import { GlobalOutlined, FolderOutlined, FileTextOutlined } from '@ant-design/icons'
import { getConfig } from '../../api/settings'
import SettingsSection from './SettingsSection'
import { useT } from '../../gaea/lib/i18n'
import * as App from '../../../src/wailsjsCompat'

interface ReleaseInfo {
  version: string
  title: string
  date: string
  intro: string
  points: string
}

/** AboutPanel — 关于：版本信息 + 系统信息（配置路径）+ 可折叠更新日志 */
const AboutPanel: React.FC = () => {
  const t = useT()
  const [config, setConfig] = useState<Record<string, string>>({})
  const [appInfo, setAppInfo] = useState<{ name: string; version: string; tagline: string; releases: ReleaseInfo[] } | null>(null)

  useEffect(() => {
    getConfig().then(setConfig).catch(() => {})
    try {
      App.GetAppInfo().then((r) => setAppInfo(r as { name: string; version: string; tagline: string; releases: ReleaseInfo[] })).catch(() => {})
    } catch (_) { /* 忽略 */ }
  }, [])

  return (
    <>
      <SettingsSection
        icon={<span style={{ fontSize: 15 }}><GlobalOutlined /></span>}
        title={t('settings.about.title')}
        desc={t('settings.about.desc')}
      >
        <div style={{
          display: 'flex', alignItems: 'center', gap: 14,
          padding: '14px 16px', borderRadius: 'var(--md-sys-radius-md)',
          background: 'var(--md-sys-color-surface-container)',
          border: '1px solid var(--md-sys-color-outline-variant)',
        }}>
          <div style={{
            width: 44, height: 44, borderRadius: 12, flexShrink: 0,
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            background: 'color-mix(in srgb, var(--gaea-glow) 16%, var(--md-sys-color-surface-container-high))',
            border: '1px solid color-mix(in srgb, var(--gaea-glow) 30%, transparent)',
            fontSize: 18, fontWeight: 700, color: 'var(--gaea-glow)',
          }}>{t('settings.about.logoGlyph')}</div>
          <div style={{ flex: 1, minWidth: 0 }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
              <Typography.Text strong style={{ color: 'var(--md-sys-color-text)', fontSize: 15 }}>gaea</Typography.Text>
              <Tag style={{ margin: 0, borderRadius: 8 }}>v{appInfo?.version || '—'}</Tag>
            </div>
            <Typography.Text style={{ color: 'var(--md-sys-color-text-secondary)', fontSize: 12, display: 'block', marginTop: 2 }}>
              {appInfo?.tagline || t('settings.about.taglineFallback')}
            </Typography.Text>
          </div>
        </div>
      </SettingsSection>

      <SettingsSection
        icon={<span style={{ fontSize: 15 }}><FolderOutlined /></span>}
        title={t('settings.about.storageTitle')}
        desc={t('settings.about.storageDesc')}
      >
        <div style={{ display: 'flex', flexDirection: 'column', gap: 0 }}>
          {[
            { label: t('settings.about.tokenPath'), value: config.tokenPath || '-' },
            { label: t('settings.about.imageDir'), value: config.image_save_dir || t('settings.about.notConfigured') },
          ].map((row) => (
            <div key={row.label} style={{
              display: 'flex', alignItems: 'baseline', gap: 12,
              padding: '7px 0', borderBottom: '1px dashed var(--md-sys-color-outline-variant)',
            }}>
              <Typography.Text style={{ color: 'var(--md-sys-color-text-secondary)', fontSize: 12, width: 110, flexShrink: 0 }}>
                {row.label}
              </Typography.Text>
              <Typography.Text style={{ color: 'var(--md-sys-color-text)', fontSize: 12, wordBreak: 'break-all' }}>
                {row.value}
              </Typography.Text>
            </div>
          ))}
        </div>
      </SettingsSection>

      <SettingsSection
        icon={<span style={{ fontSize: 15 }}><FileTextOutlined /></span>}
        title={t('settings.about.changelogTitle')}
        desc={t('settings.about.changelogDesc')}
      >
        <Collapse
          ghost
          items={[{
            key: 'changelog',
            label: t('settings.about.changelogLabel', { version: appInfo?.version || '—' }),
            children: appInfo?.releases?.length ? (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
                {appInfo.releases.map((r) => (
                  <div key={r.version} style={{ paddingBottom: 4, borderBottom: '1px dashed var(--md-sys-color-outline-variant)' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
                      <Typography.Text strong style={{ color: 'var(--md-sys-color-text)', fontSize: 13 }}>{r.version}</Typography.Text>
                      <Typography.Text style={{ color: 'var(--md-sys-color-text)', fontSize: 13 }}>{r.title}</Typography.Text>
                      <Typography.Text style={{ color: 'var(--md-sys-color-text-secondary)', fontSize: 11 }}>{r.date}</Typography.Text>
                    </div>
                    {r.intro && (
                      <Typography.Text style={{ color: 'var(--md-sys-color-text-secondary)', fontSize: 11, display: 'block', marginTop: 2 }}>
                        {r.intro}
                      </Typography.Text>
                    )}
                    {r.points && (
                      <ul style={{ margin: '6px 0 0', paddingLeft: 18 }}>
                        {r.points.split('\n').filter(Boolean).map((p, idx) => (
                          <li key={idx} style={{ color: 'var(--md-sys-color-text-secondary)', fontSize: 11, lineHeight: 1.7 }}>{p}</li>
                        ))}
                      </ul>
                    )}
                  </div>
                ))}
              </div>
            ) : (
              <Typography.Text style={{ color: 'var(--md-sys-color-text-secondary)', fontSize: 12 }}>{t('settings.about.changelogEmpty')}</Typography.Text>
            ),
          }]}
        />
      </SettingsSection>
    </>
  )
}

export default AboutPanel
