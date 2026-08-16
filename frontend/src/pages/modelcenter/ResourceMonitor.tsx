import { useCallback, useEffect, useState } from 'react'
import { Button, Tooltip } from 'antd'
import { DashboardOutlined, ReloadOutlined } from '@ant-design/icons'
import * as App from '../../../src/wailsjsCompat'
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
  const [error, setError] = useState<string | null>(null)
  const [lastUpdated, setLastUpdated] = useState('')

  const load = useCallback(async () => {
    try {
      const m: ResourceMonitorData = await App.GetModelMonitor()
      setSnap(computeResourceSnapshot(m))
      setError(null)
      setLastUpdated(new Date().toLocaleTimeString())
    } catch (err: unknown) {
      // 后端尚未就绪：保留已有快照，仅提示刷新失败，不隐藏整个资源块
      setError(err instanceof Error ? err.message : '读取资源信息失败')
    }
  }, [])

  useEffect(() => {
    load()
    const t = window.setInterval(load, 4000)
    return () => window.clearInterval(t)
  }, [load])

  const refreshBtn = (
    <Tooltip title="刷新资源">
      <Button
        size="small"
        type="text"
        icon={<ReloadOutlined />}
        aria-label="刷新资源"
        onClick={() => void load()}
      />
    </Tooltip>
  )

  if (!snap) {
    return (
      <div className="mc-live">
        <div className="mc-live-title">
          <DashboardOutlined /> 本地资源
          <span style={{ flex: 1 }} />
          {refreshBtn}
        </div>
        {error ? (
          <div className="mc-inspector-error">
            <span>资源加载失败：{error}</span>
            <Button size="small" type="primary" ghost onClick={() => void load()}>重试</Button>
          </div>
        ) : (
          <div className="mc-live-loading">正在读取系统资源…</div>
        )}
      </div>
    )
  }

  const bars = [
    { label: 'CPU', pct: snap.cpu },
    { label: '内存', pct: snap.memPct },
    { label: 'GPU', pct: snap.gpuPct },
    { label: '显存', pct: snap.vramPct },
  ]

  return (
    <div className="mc-live">
      <div className="mc-live-title">
        <DashboardOutlined /> 本地资源
        <span style={{ flex: 1 }} />
        {refreshBtn}
        {lastUpdated && <span className="mc-live-updated">{lastUpdated}</span>}
      </div>
      {error && <div className="mc-live-error">刷新失败：{error}</div>}
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
          <StatusChip title={snap.vramUsed > 0 ? `已用 ${fmtGB(snap.vramUsed)} / 共 ${fmtGB(snap.vramTotal)}` : '当前无可用占用数据（AMD 需 ComfyUI 运行）'}>
            {snap.vramUsed > 0
              ? `显存 ${fmtGB(snap.vramUsed)} / ${fmtGB(snap.vramTotal)}`
              : `显存 ${fmtGB(snap.vramTotal)}`}
          </StatusChip>
        )}
        {snap.memTotal > 0 && (
          <StatusChip>内存 {fmtGB(snap.memUsed)} / {fmtGB(snap.memTotal)}</StatusChip>
        )}
      </div>
    </div>
  )
}
