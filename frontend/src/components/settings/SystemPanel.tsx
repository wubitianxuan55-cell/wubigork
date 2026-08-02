import React, { useEffect, useState } from 'react'
import { Button, Descriptions, Tag, Timeline, Typography, message } from 'antd'
import { ThunderboltOutlined } from '@ant-design/icons'
import { getConfig } from '../../api/settings'
import SettingsSection from './SettingsSection'
import * as App from '../../../wailsjs/go/app/App'

interface ReleaseInfo {
  version: string
  title: string
  date: string
  intro: string
  points: string
}

/** SystemPanel — 更新信息 + 系统信息 + 数据迁移 */
const SystemPanel: React.FC = () => {
  const [config, setConfig] = useState<Record<string, string>>({})
  const [appInfo, setAppInfo] = useState<{ name: string; version: string; tagline: string; releases: ReleaseInfo[] } | null>(null)

  useEffect(() => {
    try {
      App.GetAppInfo().then((r: any) => setAppInfo(r)).catch(() => {})
    } catch (_) {}
  }, [])



  return (
    <>
      {/* ── 更新信息：关于 + 版本 + 更新日志 ── */}
      <SettingsSection
        title={<>更新信息</>}
        desc="当前版本与最近更新记录。"
      >
        {/* 关于卡 */}
        <div style={{
          display: 'flex', alignItems: 'center', gap: 14,
          padding: '16px 18px', borderRadius: 12, marginBottom: 16,
          background: 'linear-gradient(135deg, rgba(99,102,241,0.08), rgba(37,99,235,0.04))',
          border: '1px solid rgba(99,102,241,0.2)',
        }}>
          <div style={{
            width: 48, height: 48, borderRadius: 14, flexShrink: 0,
            background: 'linear-gradient(135deg, #6366f1, #2563eb)',
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            boxShadow: '0 6px 20px rgba(99,102,241,0.35)',
          }}>
            <ThunderboltOutlined style={{ fontSize: 24, color: '#fff' }} />
          </div>
          <div style={{ flex: 1, minWidth: 0 }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
              <Typography.Text strong style={{ color: 'var(--md-sys-color-text)', fontSize: 16 }}>gaea</Typography.Text>
              <Tag color="geekblue" style={{ margin: 0, borderRadius: 8 }}>v{appInfo?.version || '…'}</Tag>
            </div>
            <Typography.Text style={{ color: 'var(--md-sys-color-text-secondary)', fontSize: 12, display: 'block', marginTop: 2 }}>
              {appInfo?.tagline || '多功能 AI 助手'}
            </Typography.Text>
          </div>
        </div>

        {/* 更新日志 */}
        {appInfo?.releases && appInfo.releases.length > 0 && (
          <Timeline
            items={appInfo.releases.map((r) => ({
              color: r.version === `v${appInfo.version}` ? 'blue' : 'gray',
              children: (
                <div style={{ paddingBottom: 6 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
                    <Typography.Text strong style={{ color: 'var(--md-sys-color-text)', fontSize: 13 }}>
                      {r.version}
                    </Typography.Text>
                    <Typography.Text style={{ color: 'var(--md-sys-color-text)', fontSize: 13 }}>
                      {r.title}
                    </Typography.Text>
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
              ),
            }))}
          />
        )}
      </SettingsSection>

      <SettingsSection
        title={<>系统信息</>}
        desc="当前引擎、API 与凭证存储路径。"
      >
        <Descriptions
          column={1}
          size="small"
          labelStyle={{ color: 'var(--md-sys-color-text-secondary)', width: 120 }}
          contentStyle={{ color: 'var(--md-sys-color-text)', wordBreak: 'break-all' }}
        >
          <Descriptions.Item label="引擎">{config.model || '-'}</Descriptions.Item>
          <Descriptions.Item label="API 地址">{config.baseURL || '-'}</Descriptions.Item>
          <Descriptions.Item label="Token 存储">{config.tokenPath || '-'}</Descriptions.Item>
          <Descriptions.Item label="推理强度">{config.reasoning_effort || '-'}</Descriptions.Item>
          <Descriptions.Item label="图片保存目录">{config.image_save_dir || '未配置'}</Descriptions.Item>
        </Descriptions>
      </SettingsSection>


    </>
  )
}

export default SystemPanel
