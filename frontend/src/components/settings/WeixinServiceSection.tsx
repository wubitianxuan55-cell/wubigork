import React, { useEffect, useState } from 'react'
import { Switch, message } from 'antd'
import { WechatOutlined } from '@ant-design/icons'
import { app } from '../../gaea/lib/bridge'
import SettingsSection from './SettingsSection'

/**
 * WeixinServiceSection — 微信到点回推服务开关（设置中枢批 5 从 WeixinPage
 * 提醒区收编）。与板块页开关同键同源（WeixinReminderConfig/SetConfig），页内
 * 开关保留为会话内高频快捷态；助手级「接收微信消息」逐助手操作留板块页
 * （核心助手不可停等域逻辑不复制）。
 */
const WeixinServiceSection: React.FC = () => {
  const [enabled, setEnabled] = useState<boolean | null>(null)

  useEffect(() => {
    app.WeixinReminderConfig()
      .then((c) => setEnabled(Boolean(c?.remindersEnabled)))
      .catch(() => {})
  }, [])

  return (
    <SettingsSection
      icon={<span style={{ fontSize: 15 }}><WechatOutlined /></span>}
      title="回推服务"
      desc="到点回推微信：提醒服务常驻监听，关闭后暂停全部定时回推（助手级接收开关在微信助手板块页）"
    >
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 12, padding: '8px 0' }}>
        <span style={{ fontSize: 13, color: 'var(--md-sys-color-text)' }}>
          {enabled === null ? '读取中…' : enabled ? '回推服务正在监听' : '回推服务已暂停'}
        </span>
        <Switch
          size="small"
          aria-label="到点回推微信"
          checked={enabled === true}
          disabled={enabled === null}
          onChange={(v) => {
            const prev = enabled
            setEnabled(v)
            app.WeixinReminderSetConfig(JSON.stringify({ remindersEnabled: v }))
              .catch(() => {
                setEnabled(prev)
                message.warning('保存配置失败，已还原')
              })
          }}
        />
      </div>
    </SettingsSection>
  )
}

export default WeixinServiceSection
