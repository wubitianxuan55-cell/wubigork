import { useEffect, useState } from 'react'
import { Space, Tag, Typography } from 'antd'
import { DashboardOutlined } from '@ant-design/icons'
import * as App from '../../../wailsjs/go/app/App'
import { C } from '../../utils/theme'
import {
  computeResourceSnapshot,
  fmtGB,
  resourceLevel,
  resourceLevelColor,
  type ResourceMonitorData,
  type ResourceSnapshot,
} from './resource'

// 本地资源占用实时面板：轮询 GetModelMonitor，展示 CPU/内存/GPU(显存) 与本地已启动模型。
export function ResourceMonitor() {
  const [snap, setSnap] = useState<ResourceSnapshot | null>(null)

  useEffect(() => {
    let alive = true
    const load = async () => {
      try {
        const m: ResourceMonitorData = await App.GetModelMonitor()
        if (alive) setSnap(computeResourceSnapshot(m))
      } catch {
        // 后端尚未就绪时静默，等待下一次轮询
      }
    }
    load()
    const t = window.setInterval(load, 4000)
    return () => { alive = false; window.clearInterval(t) }
  }, [])

  if (!snap) return null

  const bars = [
    { label: 'CPU', pct: snap.cpu },
    { label: '内存', pct: snap.memPct },
    { label: 'GPU', pct: snap.gpuPct },
    { label: '显存', pct: snap.vramPct },
  ]

  return (
    <div className="mc-panel" style={{ padding: '12px 14px', marginBottom: 16 }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 10, flexWrap: 'wrap' }}>
        <Space size={8}>
          <DashboardOutlined style={{ color: C('color-text-secondary') }} />
          <Typography.Text strong style={{ color: C('color-text'), fontSize: 13 }}>本地资源占用</Typography.Text>
          <span className="live-dot" style={{ width: 6, height: 6 }} />
        </Space>
        <Space size={6} wrap>
          {snap.localEngines.length > 0
            ? snap.localEngines.map(e => <Tag key={e} color="geekblue" style={{ fontSize: 10, margin: 0 }}>{e}</Tag>)
            : <Tag style={{ fontSize: 10, margin: 0 }}>无本地引擎</Tag>}
          {snap.comfyRunning && <Tag color="orange" style={{ fontSize: 10, margin: 0 }}>ComfyUI 运行中</Tag>}
        </Space>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(140px, 1fr))', gap: 12, marginTop: 12 }}>
        {bars.map(b => {
          const level = resourceLevel(b.pct)
          const color = resourceLevelColor[level]
          return (
            <div key={b.label}>
              <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 11, color: C('color-text-secondary') }}>
                <span>{b.label}</span>
                <span style={{ color }}>{b.pct}%</span>
              </div>
              <div style={{ height: 6, borderRadius: 3, background: 'rgba(148,163,184,0.16)', marginTop: 5, overflow: 'hidden' }}>
                <div style={{ width: `${Math.min(Math.max(b.pct, 0), 100)}%`, height: '100%', background: color, borderRadius: 3, transition: 'width 320ms ease' }} />
              </div>
            </div>
          )
        })}
      </div>

      {snap.gpuName && (
        <div style={{ marginTop: 10, fontSize: 10, color: C('color-text-secondary'), display: 'flex', flexWrap: 'wrap', gap: 8 }}>
          <span>GPU：{snap.gpuName}</span>
          {snap.vramTotal > 0 && <span>显存 {fmtGB(snap.vramUsed)} / {fmtGB(snap.vramTotal)}</span>}
          {snap.memTotal > 0 && <span>内存 {fmtGB(snap.memUsed)} / {fmtGB(snap.memTotal)}</span>}
        </div>
      )}
    </div>
  )
}
