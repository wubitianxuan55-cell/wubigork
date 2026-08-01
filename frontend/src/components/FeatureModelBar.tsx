import React, { useState, useEffect, useCallback } from 'react'
import { Tag, Tooltip, Button, message } from 'antd'
import { PoweroffOutlined } from '@ant-design/icons'
import * as App from '../../wailsjs/go/app/App'
import { getEngines, saveEngine } from '../api/engines'
import { useFeatureModel } from '../hooks/useFeatureModel'

/**
 * 功能模型状态条 — 显示某功能的绑定模型 + 引擎运行状态 + 一键启停按钮。
 * 状态明确区分：运行中（绿）/ 已停用（灰）/ 未绑定（描边虚线）；
 * 启停操作友好降级，绝不弹错。
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
  const running = !!boundEngine?.enabled
  const bound = !!m.engine && !!m.model && !!boundEngine

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
    setBusy(true)
    try {
      await saveEngine({ ...boundEngine, enabled: !running })
      message.success(`「${label}」${running ? '已停用' : '已启动'}：${m.model}`)
      load()
    } catch (err: any) {
      // 友好提示，不弹错误
      console.warn('[FeatureModelBar] 启停失败:', err)
      message.warning(`「${label}」操作未生效：${err?.message || '请稍后再试'}`)
    } finally {
      setBusy(false)
    }
  }

  const statusColor = bound ? (running ? '#22c55e' : '#64748b') : '#a855f7'
  const statusText = !bound ? '未绑定' : running ? '运行中' : '已停用'

  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 11 }}>
      <Tag color="default" style={{ fontSize: 10, margin: 0 }}>{label}</Tag>
      <span style={{
        width: 7, height: 7, borderRadius: '50%',
        background: statusColor,
        boxShadow: running ? '0 0 6px #22c55e' : 'none',
        display: 'inline-block',
      }} />
      <span style={{
        color: running ? 'var(--md-sys-color-text)' : 'var(--md-sys-color-text-secondary)',
        fontWeight: running ? 600 : 400,
        maxWidth: 160, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
      }}>
        {m.model || (m.engine ? `${m.engine} 默认` : '未绑定')}
      </span>
      <span style={{
        fontSize: 10, padding: '0 6px', borderRadius: 8, lineHeight: '16px',
        color: bound ? (running ? '#22c55e' : '#94a3b8') : '#a855f7',
        background: bound ? (running ? 'rgba(34,197,94,0.1)' : 'rgba(100,116,139,0.08)') : 'rgba(168,85,247,0.08)',
        border: bound ? 'none' : '1px dashed rgba(168,85,247,0.4)',
      }}>
        {statusText}
      </span>
      <Tooltip title={!bound ? '先到模型中心绑定' : (running ? `停用「${label}」模型` : `启动「${label}」模型`)}>
        <Button type="text" size="small" loading={busy} onClick={toggle}
          icon={<PoweroffOutlined style={{ color: running ? '#22c55e' : 'var(--md-sys-color-text-secondary)' }} />}
          style={{ width: 22, height: 22, minWidth: 22, padding: 0, fontSize: 11 }} />
      </Tooltip>
    </div>
  )
}

export default FeatureModelBar
