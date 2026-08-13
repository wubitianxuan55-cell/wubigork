import { useEffect, useState } from 'react'
import { DashboardOutlined } from '@ant-design/icons'
import * as App from '../../../wailsjs/go/app/App'
import { StatusChip } from './ui'
import {
  computeResourceSnapshot,
  fmtGB,
  resourceLevel,
  resourceLevelColor,
  type ResourceMonitorData,
  type ResourceSnapshot,
} from './resource'

// 本地资源占用实时条：轮询 GetModelMonitor，展示 CPU/内存/GPU(显存) 与本地已启动模型。
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
    <div className="mc-live">
      <span className="mc-live-title"><DashboardOutlined /> 本地资源</span>
      <div className="mc-live-bars">
        {bars.map(b => {
          const level = resourceLevel(b.pct)
          const color = resourceLevelColor[level]
          return (
            <div key={b.label} className="mc-live-bar">
              <div className="mc-live-bar-head">
                <span>{b.label}</span>
                <span style={{ color }}>{b.pct}%</span>
              </div>
              <div className="mc-live-bar-track">
                <div
                  className="mc-live-bar-fill"
                  style={{
                    width: `${Math.min(Math.max(b.pct, 0), 100)}%`,
                    background: color,
                  }}
                />
              </div>
            </div>
          )
        })}
      </div>
      <div className="mc-live-side">
        {snap.localEngines.length > 0
          ? snap.localEngines.map(e => <StatusChip key={e} tone="accent" dot>{e}</StatusChip>)
          : <StatusChip>无本地引擎</StatusChip>}
        {snap.comfyRunning && <StatusChip tone="warn" dot>ComfyUI 运行中</StatusChip>}
        {snap.gpuName && (
          <StatusChip title={snap.gpuName}>{snap.gpuName}</StatusChip>
        )}
        {snap.vramTotal > 0 && (
          <StatusChip>显存 {fmtGB(snap.vramUsed)} / {fmtGB(snap.vramTotal)}</StatusChip>
        )}
        {snap.memTotal > 0 && (
          <StatusChip>内存 {fmtGB(snap.memUsed)} / {fmtGB(snap.memTotal)}</StatusChip>
        )}
      </div>
    </div>
  )
}
