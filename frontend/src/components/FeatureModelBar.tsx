import React, { useState, useEffect, useCallback } from 'react'
import { Button, Tooltip, message } from 'antd'
import { PoweroffOutlined } from '@ant-design/icons'
import * as App from '../../src/wailsjsCompat'
import { getEngines } from '../api/engines'
import { useFeatureModel } from '../hooks/useFeatureModel'

/**
 * 功能模型状态卡 — 显示某功能的绑定模型 + 引擎运行状态 + 一键启停按钮。
 * 卡片形态，统一放置在各功能页面左下角（由页面用 absolute 定位包裹）。
 * 状态明确区分：运行中（绿）/ 已停用（灰）/ 未绑定（虚线紫）；
 * 「启停」是功能级开关（SetFeatureModelEnabled），只影响该功能的路由，
 * 绝不改动引擎整体启用状态（避免误关全局引擎）。
 */
const FeatureModelBar: React.FC<{ feature: string; label: string }> = ({ feature, label }) => {
  const m = useFeatureModel(feature)
  const [engines, setEngines] = useState<any[]>([])
  const [busy, setBusy] = useState(false)

  const load = useCallback(async () => {
    try { setEngines(await getEngines()) } catch (_) {}
  }, [])

  useEffect(() => { load() }, [load])
  useEffect(() => {
    let unsub: any
    try {
      unsub = (window as any).runtime?.EventsOn?.('feature-model-changed', load) || unsub
    } catch (_) {}
    const t = setInterval(load, 8000)
    return () => { try { clearInterval(t); if (typeof unsub === 'function') unsub() } catch (_) {} }
  }, [load])

  const boundEngine = m.engine ? engines.find((e: any) => e.id === m.engine) : undefined
  const bound = !!m.engine && !!m.model && !!boundEngine
  // 运行中 = 功能启用 + 绑定引擎启用
  const running = bound && m.enabled && !!boundEngine?.enabled

  const toggle = async () => {
    if (busy) return
    if (!m.engine || !m.model) {
      message.info(`「${label}」尚未绑定引擎和模型，请到「模型中心 → 功能模型绑定」设置`)
      return
    }
    if (!boundEngine) {
      message.info(`「${label}」绑定的引擎 ${m.engine} 未找到，请到模型中心检查`)
      return
    }
    if (m.enabled && !boundEngine?.enabled) {
      message.warning(`「${label}」绑定的引擎已停用，请到「模型中心 → 引擎管理」启用 ${m.engine}`)
      return
    }
    setBusy(true)
    try {
      await App.SetFeatureModelEnabled(feature, !m.enabled)
      message.success(`「${label}」${m.enabled ? '已停用' : '已启动'}：${m.model}`)
      load()
    } catch (err: any) {
      // 友好提示，不弹错误
      console.warn('[FeatureModelBar] 启停失败:', err)
      message.warning(`「${label}」操作未生效：${err?.message || '请稍后再试'}`)
    } finally {
      setBusy(false)
    }
  }

  const runningColor = '#22c55e'
  const idleColor = '#64748b'
  const unboundColor = '#a855f7'
  const engineOff = bound && m.enabled && !boundEngine?.enabled
  const statusColor = !bound ? unboundColor : (running ? runningColor : idleColor)
  const statusText = !bound ? '未绑定' : running ? '运行中' : engineOff ? '引擎已停用' : '已停用'

  return (
    <div style={{
      width: 200,
      borderRadius: 14,
      padding: '10px 12px',
      background: 'var(--gaea-glass-bg, var(--md-sys-color-surface-container))',
      WebkitBackdropFilter: 'blur(16px) saturate(140%)',
      backdropFilter: 'blur(16px) saturate(140%)',
      border: bound ? (running ? '1px solid rgba(34,197,94,0.4)' : '1px solid var(--md-sys-color-outline-variant)') : '1px dashed rgba(168,85,247,0.45)',
      boxShadow: running ? '0 8px 24px rgba(34,197,94,0.15)' : '0 8px 24px rgba(0,0,0,0.18)',
      display: 'flex', flexDirection: 'column', gap: 8, fontSize: 11,
    }}>
      {/* 头部：功能名 + 状态徽标 */}
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <span style={{ fontWeight: 600, color: 'var(--md-sys-color-text)', fontSize: 12 }}>{label}</span>
        <span style={{
          display: 'inline-flex', alignItems: 'center', gap: 4,
          fontSize: 10, padding: '1px 8px', borderRadius: 8, lineHeight: '16px',
          color: bound ? (running ? runningColor : '#94a3b8') : unboundColor,
          background: bound ? (running ? 'rgba(34,197,94,0.12)' : 'rgba(100,116,139,0.08)') : 'rgba(168,85,247,0.08)',
        }}>
          <span style={{ width: 6, height: 6, borderRadius: '50%', background: statusColor, boxShadow: running ? `0 0 6px ${runningColor}` : 'none', display: 'inline-block' }} />
          {statusText}
        </span>
      </div>
      {/* 模型名 */}
      <div style={{
        color: running ? 'var(--md-sys-color-text)' : 'var(--md-sys-color-text-secondary)',
        fontWeight: running ? 600 : 400,
        overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
      }} title={m.model || ''}>
        {m.model || (m.engine ? `${m.engine} 默认` : '未绑定模型')}
      </div>
      {/* 启停按钮（功能级开关，不碰引擎） */}
      <Tooltip title={!bound ? '先到模型中心绑定' : (engineOff ? '绑定引擎已停用，请到模型中心启用' : (running ? `停用「${label}」功能模型` : `启用「${label}」功能模型`))}>
        <Button size="small" block loading={busy} onClick={toggle}
          icon={<PoweroffOutlined />}
          style={{
            fontSize: 11, height: 24,
            color: running ? runningColor : 'var(--md-sys-color-text-secondary)',
            borderColor: running ? 'rgba(34,197,94,0.5)' : 'var(--md-sys-color-outline-variant)',
            background: running ? 'rgba(34,197,94,0.06)' : 'transparent',
          }}>
          {running ? '停用' : '启动'}
        </Button>
      </Tooltip>
    </div>
  )
}

export default FeatureModelBar
