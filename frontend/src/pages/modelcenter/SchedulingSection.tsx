import { useCallback, useEffect, useState } from 'react'
import { Switch, message } from 'antd'
import { AppstoreOutlined, CloudSyncOutlined, FireOutlined, RocketOutlined } from '@ant-design/icons'
import { app } from '../../gaea/lib/bridge'
import { getHerdsmanCatalog } from '../../api/engines'
import { SectionHead, StatusChip } from './ui'

// 模型中心「本地调度」设置区（T5-3）：保活开关 + 启动自动预载开关 + 当前运行中的本地模型数。
// 绑定契约见 frontend/src/gaea/lib/bridge.ts（KeepWarmGet/Set → GaeaKeepWarmGet/Set，
// PreloadPlanGet/Set → GaeaPreloadPlanGet/Set，由父代理实现的后端 Gaea* 绑定持久化于
// ~/.gaea_config.json）。
export function SchedulingSection() {
  // null = 尚未读取/读取失败；undefined 区分「未知」与「已关闭」
  const [keepWarm, setKeepWarm] = useState<boolean | null>(null)
  const [preloadPlan, setPreloadPlan] = useState<boolean | null>(null)
  const [saving, setSaving] = useState<'' | 'keepwarm' | 'preload'>('')
  // 当前保活状态：Herdsman 运行中的模型数（HerdsmanModelCatalog 聚合）
  const [running, setRunning] = useState<number | null>(null)
  const [runningErr, setRunningErr] = useState<string | null>(null)

  const load = useCallback(async () => {
    try { setKeepWarm(await app.KeepWarmGet()) } catch { setKeepWarm(null) }
    try { setPreloadPlan(await app.PreloadPlanGet()) } catch { setPreloadPlan(null) }
    try {
      const c = await getHerdsmanCatalog()
      setRunning(typeof c.running === 'number' ? c.running : (c.models?.filter(m => m.running).length ?? null))
      setRunningErr(c.error || null)
    } catch (e: unknown) {
      setRunning(null)
      setRunningErr(e instanceof Error ? e.message : String(e))
    }
  }, [])

  useEffect(() => { void load() }, [load])

  const toggle = async (kind: 'keepwarm' | 'preload', next: boolean) => {
    setSaving(kind)
    try {
      if (kind === 'keepwarm') await app.KeepWarmSet(next)
      else await app.PreloadPlanSet(next)
      message.success(kind === 'keepwarm' ? (next ? '已开启保活' : '已关闭保活') : (next ? '已开启自动预载' : '已关闭自动预载'))
      if (kind === 'keepwarm') setKeepWarm(next)
      else setPreloadPlan(next)
    } catch (e: unknown) {
      message.error(e instanceof Error ? e.message : '保存失败')
    } finally {
      setSaving('')
    }
  }

  return (
    <section className="mc-section">
      <SectionHead
        icon={<CloudSyncOutlined />}
        title="本地调度"
        desc="本地模型运行策略：保活防卸载、启动自动预载、换模等待预估（T5-3）"
      />

      <div className="mc-grid two-col">
        {/* 保活开关 */}
        <div className="mc-bind-card">
          <div className="mc-bind-head">
            <span className="mc-bind-title"><FireOutlined /> 保活</span>
            <StatusChip tone={keepWarm === null ? 'neutral' : keepWarm ? 'ok' : 'warn'} dot>
              {keepWarm === null ? '未知' : keepWarm ? '已开启' : '已关闭'}
            </StatusChip>
          </div>
          <div className="mc-bind-desc">空闲时定期轻量探测，防止本地模型被卸载（推荐开启）</div>
          <div className="mc-bind-meta">设置持久化于 ~/.gaea_config.json，重启后仍生效</div>
          <div className="mc-bind-switch">
            <span>启用保活</span>
            <Switch
              size="small"
              checked={keepWarm === true}
              disabled={keepWarm === null}
              loading={saving === 'keepwarm'}
              onChange={(v: boolean) => void toggle('keepwarm', v)}
            />
          </div>
        </div>

        {/* 自动预载开关 */}
        <div className="mc-bind-card">
          <div className="mc-bind-head">
            <span className="mc-bind-title"><RocketOutlined /> 自动预载</span>
            <StatusChip tone={preloadPlan === null ? 'neutral' : preloadPlan ? 'ok' : 'warn'} dot>
              {preloadPlan === null ? '未知' : preloadPlan ? '已开启' : '已关闭'}
            </StatusChip>
          </div>
          <div className="mc-bind-desc">启动时后台预载常用本地模型，减少首次对话等待</div>
          <div className="mc-bind-meta">仅影响启动阶段；运行中手动切换仍按换模预估提示</div>
          <div className="mc-bind-switch">
            <span>启用自动预载</span>
            <Switch
              size="small"
              checked={preloadPlan === true}
              disabled={preloadPlan === null}
              loading={saving === 'preload'}
              onChange={(v: boolean) => void toggle('preload', v)}
            />
          </div>
        </div>
      </div>

      {/* 当前保活状态：运行中的本地模型数（来自 Herdsman 模型目录） */}
      <div className="mc-bind-row" style={{ gap: 8, marginTop: 14 }}>
        <span className="mc-bind-title" style={{ fontSize: 12 }}><AppstoreOutlined /> 当前运行中的本地模型</span>
        <StatusChip tone={running === null ? 'neutral' : running > 0 ? 'ok' : 'warn'} dot>
          {running === null ? (runningErr ? '不可用' : '探测中…') : `${running} 个`}
        </StatusChip>
        {runningErr && <span style={{ color: 'var(--mc-muted)', fontSize: 11 }}>（{runningErr}）</span>}
      </div>
    </section>
  )
}