import React, { useState, useEffect, useCallback } from 'react'
import { Tag, Tooltip, Button, message } from 'antd'
import { PoweroffOutlined } from '@ant-design/icons'
import * as App from '../../wailsjs/go/app/App'
import { getEngines, saveEngine } from '../api/engines'
import { useFeatureModel } from '../hooks/useFeatureModel'

/**
 * 功能模型状态条 — 显示某功能的绑定模型 + 引擎启停状态 + 一键启停按钮。
 * 供各功能板块顶部使用（聊天/轻语/小说/方案编写/办公）。
 */
const FeatureModelBar: React.FC<{ feature: string; label: string }> = ({ feature, label }) => {
  const m = useFeatureModel(feature)
  const [engines, setEngines] = useState<any[]>([])
  const [busy, setBusy] = useState(false)

  const load = useCallback(async () => {
    try { setEngines(await getEngines()) } catch (_) {}
  }, [])

  useEffect(() => { load() }, [load])
  // 引擎状态变化（启停/模型绑定）时刷新
  useEffect(() => {
    let unsub: any
    try {
      unsub = (window as any).runtime?.EventsOn?.('feature-model-changed', load) || unsub
    } catch (_) {}
    const t = setInterval(load, 8000)
    return () => { try { clearInterval(t); if (typeof unsub === 'function') unsub() } catch (_) {} }
  }, [load])

  const boundEngine = engines.find((e: any) => e.id === m.engine && m.engine)
  const enabled = !!boundEngine?.enabled
  const modelLabel = m.model || (m.engine ? `引擎默认（${m.engine}）` : '')

  const toggle = async () => {
    if (!m.engine) { message.info('尚未绑定引擎，请到模型中心设置'); return }
    setBusy(true)
    try {
      await saveEngine({ ...boundEngine, enabled: !enabled })
      message.success(`「${label}」模型${enabled ? '已停用' : '已启用'}`)
      load()
    } catch (err: any) { message.error(err?.message || '操作失败') }
    finally { setBusy(false) }
  }

  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 11 }}>
      <Tag color="default" style={{ fontSize: 10, margin: 0 }}>{label}</Tag>
      <span style={{
        width: 7, height: 7, borderRadius: '50%',
        background: enabled ? '#22c55e' : '#64748b',
        boxShadow: enabled ? '0 0 6px #22c55e' : 'none',
        display: 'inline-block',
      }} />
      <span style={{ color: enabled ? 'var(--md-sys-color-text)' : 'var(--md-sys-color-text-secondary)', fontWeight: enabled ? 600 : 400 }}>
        {modelLabel || '未绑定'}
      </span>
      <Tooltip title={enabled ? `停用「${label}」绑定的模型` : `启用「${label}」绑定的模型`}>
        <Button type="text" size="small" loading={busy} onClick={toggle}
          icon={<PoweroffOutlined style={{ color: enabled ? '#22c55e' : 'var(--md-sys-color-text-secondary)' }} />}
          style={{ width: 22, height: 22, minWidth: 22, padding: 0, fontSize: 11 }} />
      </Tooltip>
    </div>
  )
}

export default FeatureModelBar
