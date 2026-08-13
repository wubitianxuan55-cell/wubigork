import React, { useCallback, useEffect, useState } from 'react'
import { Alert, Button, Space, Typography } from 'antd'
import { ReloadOutlined, SafetyCertificateOutlined } from '@ant-design/icons'

/**
 * SecurityBanner — Herdsman LAN 暴露安全告警（S2-1）
 *
 * 启动时调用后端 App.HerdsmanSecurityCheck()（H0-4）检测 herdsman
 * config.yaml 的 api.lan_accessible；为 true 时在全局顶部展示醒目告警
 * 与中文处置指引（gaea 仅提示，不代改 herdsman 配置）。
 *
 * 行为：
 *  - 挂载即检测，检测失败（如 herdsman 未安装/配置缺失）静默不打扰；
 *  - 暴露时渲染 warning 横幅，可「本次会话忽略」；
 *  - 提供「重新检测」按钮（用户改完配置后手动复核）。
 */

interface LanExposure {
  config_path: string
  exposed: boolean
  port: number
  config_missing: boolean
  parse_error?: string
  guidance?: string
}

const DISMISS_KEY = 'gaea-lan-alert-dismissed'

export const SecurityBanner: React.FC = () => {
  const [exposure, setExposure] = useState<LanExposure | null>(null)
  const [checking, setChecking] = useState(false)
  const [dismissed, setDismissed] = useState<boolean>(() => {
    try {
      return window.localStorage.getItem(DISMISS_KEY) === '1'
    } catch {
      return false
    }
  })

  const check = useCallback(async () => {
    setChecking(true)
    try {
      // @ts-ignore — wails.d.ts 已声明 HerdsmanSecurityCheck；mock/HTTP 环境无此方法
      const res: LanExposure | undefined = await (window as any).go?.app?.App?.HerdsmanSecurityCheck?.()
      if (!res) return
      setExposure(res)
    } catch {
      setExposure(null)
    } finally {
      setChecking(false)
    }
  }, [])

  useEffect(() => {
    void check()
  }, [check])

  const dismiss = () => {
    setDismissed(true)
    try {
      window.localStorage.setItem(DISMISS_KEY, '1')
    } catch {
      /* ignore */
    }
  }

  if (dismissed) return null
  // 未暴露 / 无法检测 / 检测失败：不打扰。
  if (!exposure?.exposed) return null

  return (
    <Alert
      type="warning"
      showIcon
      icon={<SafetyCertificateOutlined />}
      banner
      message={
        <Space size={8} wrap>
          <Typography.Text strong style={{ color: 'inherit' }}>
            ⚠ Herdsman API 已开启局域网访问（端口 {exposure.port}）
          </Typography.Text>
          <Button
            size="small"
            type="text"
            icon={<ReloadOutlined spin={checking} />}
            loading={checking}
            onClick={() => void check()}
            style={{ color: 'inherit' }}
          >
            重新检测
          </Button>
          <Button size="small" type="text" onClick={dismiss} style={{ color: 'inherit' }}>
            本次忽略
          </Button>
        </Space>
      }
      description={
        <div style={{ fontSize: 12, lineHeight: 1.7 }}>
          {exposure.guidance || `请检查 ${exposure.config_path} 的 api 段 lan_accessible 配置。`}
        </div>
      }
      closable={false}
      style={{ borderRadius: 0, borderInlineStartWidth: 4 }}
    />
  )
}

export default SecurityBanner
