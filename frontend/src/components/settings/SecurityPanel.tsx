import React, { useCallback, useEffect, useState } from 'react'
import { Button, Space, Switch, Tag, Typography, message } from 'antd'
import { ReloadOutlined, SafetyCertificateOutlined, LockOutlined, BugOutlined } from '@ant-design/icons'
import SettingsSection from './SettingsSection'
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
      message.success(v ? '敏感域 AI 已改为本地优先（数据不出本机）' : '敏感域 AI 已改为常规路由（可回云端）')
    } catch (err: unknown) {
      setSensitiveLocal(prev)
      message.error(err instanceof Error ? err.message : '保存失败')
    }
  }

  const handleToggleOffice = async (v: boolean) => {
    const prev = officeLocal
    setOfficeLocal(v)
    try {
      await go?.SetOfficeLocal?.(v)
      message.success(v ? '办公板块 AI 已改为本地优先（数据不出本机、省 token）' : '办公板块 AI 已改为常规路由（可回云端）')
    } catch (err: unknown) {
      setOfficeLocal(prev)
      message.error(err instanceof Error ? err.message : '保存失败')
    }
  }

  const handleToggleOffline = async (v: boolean) => {
    const prev = offlineMode
    setOfflineMode(v)
    try {
      await go?.SetOfflineMode?.(v)
      message.success(v ? '全局离线模式已开启：所有 AI 只走本地引擎' : '全局离线模式已关闭：可回云端')
    } catch (err: unknown) {
      setOfflineMode(prev)
      message.error(err instanceof Error ? err.message : '保存失败')
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
        title="敏感域本地化"
        desc="成本/报价等商务敏感数据的 AI 处理默认路由本地 Herdsman（数据不出本机）；关闭后可按常规路由回云端。"
        instant
      >
        <Space size={12}>
          <Switch
            checked={sensitiveLocal}
            loading={sensitiveLoading}
            onChange={handleToggleSensitive}
          />
          <Typography.Text style={{ fontSize: 13, color: 'var(--md-sys-color-text)' }}>
            {sensitiveLocal ? '本地优先（推荐）' : '常规路由'}
          </Typography.Text>
        </Space>
        <div style={{ fontSize: 11, color: 'var(--md-sys-color-text-secondary)', marginTop: 6, opacity: 0.8 }}>
          当前生效范围：报价单/测算表 AI 归一化解析（成本库导入）。Herdsman 引擎停用或不可用时自动回退常规路由。
        </div>
      </SettingsSection>

      <SettingsSection
        icon={<LockOutlined />}
        title="办公本地优先"
        desc="办公板块的功能级 AI 调用（Word/Excel 编辑、资料摘要、知识导入、记忆整理）默认路由本地 Herdsman：数据不出本机、不烧 token；关闭后按常规路由可回云端。"
        instant
      >
        <Space size={12}>
          <Switch
            checked={officeLocal}
            loading={officeLoading}
            onChange={handleToggleOffice}
          />
          <Typography.Text style={{ fontSize: 13, color: 'var(--md-sys-color-text)' }}>
            {officeLocal ? '本地优先（推荐）' : '常规路由'}
          </Typography.Text>
        </Space>
        <div style={{ fontSize: 11, color: 'var(--md-sys-color-text-secondary)', marginTop: 6, opacity: 0.8 }}>
          聊天主 agent（统筹规划）不受此开关影响，仍按模型中心绑定走。Herdsman 引擎停用或不可用时自动回退常规路由。
        </div>
      </SettingsSection>

      <SettingsSection
        icon={<LockOutlined />}
        title="全局离线模式"
        desc="数据不出本机的总闸：开启后所有 AI 路由只允许本地引擎（Herdsman/Ollama/CosyVoice），云端引擎（Grok/DeepSeek/OpenCode）一律跳过；无本地可用时按「模型不可用」如实降级。"
        instant
      >
        <Space size={12}>
          <Switch
            checked={offlineMode}
            loading={offlineLoading}
            onChange={handleToggleOffline}
          />
          <Typography.Text style={{ fontSize: 13, color: 'var(--md-sys-color-text)' }}>
            {offlineMode ? '仅本地（离线）' : '关闭（默认）'}
          </Typography.Text>
        </Space>
        <div style={{ fontSize: 11, color: 'var(--md-sys-color-text-secondary)', marginTop: 6, opacity: 0.8 }}>
          与「敏感域/办公本地优先」叠加生效：那两个开关只约束各自功能域，本开关约束全部 AI 调用（含意图 LLM 兜底）。
        </div>
      </SettingsSection>

      <SettingsSection
        icon={<SafetyCertificateOutlined />}
        title="Herdsman 局域网暴露"
        desc="检测 herdsman config.yaml 的 api.lan_accessible。暴露时局域网内任意设备可调用本机大模型，建议关闭。"
      >
        <Space size={8} wrap>
          {exposure?.exposed ? (
            <Tag color="error">⚠ 已暴露（端口 {exposure.port}）</Tag>
          ) : exposure ? (
            <Tag color="success">✓ 未暴露</Tag>
          ) : (
            <Tag>未检测到 herdsman 配置</Tag>
          )}
          <Button size="small" icon={<ReloadOutlined spin={checking} />} loading={checking} onClick={() => void check()}>
            重新检测
          </Button>
        </Space>
        {exposure?.exposed && (
          <div style={{
            marginTop: 8, padding: '10px 12px', borderRadius: 'var(--md-sys-radius-md)',
            background: 'var(--md-sys-color-surface-container)',
            border: '1px solid var(--md-sys-color-outline-variant)',
            fontSize: 12, lineHeight: 1.7, color: 'var(--md-sys-color-text)',
          }}>
            {exposure.guidance || `请检查 ${exposure.config_path} 的 api 段 lan_accessible 配置。`}
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
        title="WebView2 远程调试"
        desc="默认关闭（安全收敛 S2-2）。需要排查渲染问题时，以 GAEA_WEBVIEW_DEBUG=1 环境变量启动 gaea，远程调试端口 127.0.0.1:9333 才会开启。"
      >
        <Typography.Text style={{ fontSize: 12, color: 'var(--md-sys-color-text-secondary)' }}>
          状态：默认关闭。调试用开关，勿在生产环境常开。
        </Typography.Text>
      </SettingsSection>
    </>
  )
}

export default SecurityPanel
