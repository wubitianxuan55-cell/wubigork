import { useEffect, useRef, useState } from 'react'
import { Button, message, Popconfirm, Select } from 'antd'
import { StatusChip } from './ui'
import {
  getStrataEngineStatus,
  getTianxuanEngineStatus,
  startStrataEngine,
  startTianxuanEngine,
  stopStrataEngine,
  stopTianxuanEngine,
  type EngineProfileOption,
  type ManagedEngineStartResult,
  type ManagedEngineStatusResult,
  type ManagedEngineStopResult,
} from '../../api/engines'

type ManagedEngineID = 'strata' | 'tianxuan'

interface LifecycleAPI {
  defaultProfile: string
  fallbackProfiles: EngineProfileOption[]
  status: () => Promise<ManagedEngineStatusResult>
  start: (profile: string) => Promise<ManagedEngineStartResult>
  stop: () => Promise<ManagedEngineStopResult>
}

// 外部项目的档位契约在 Go 侧是单一来源；这里只做状态接口失败时的展示兜底。
const API: Record<ManagedEngineID, LifecycleAPI> = {
  strata: {
    defaultProfile: 'qwen36_uncensored',
    fallbackProfiles: [
      { id: 'qwen36_uncensored', label: 'qwen3.6-35b-a3b-uncensored-q4_k_p' },
    ],
    status: getStrataEngineStatus,
    start: startStrataEngine,
    stop: stopStrataEngine,
  },
  tianxuan: {
    defaultProfile: 'uncensored',
    fallbackProfiles: [
      { id: 'uncensored', label: 'qwen3.6-35b-a3b-uncensored' },
      { id: 'flash', label: 'qwen3.8-flash-next-uncensored' },
    ],
    status: getTianxuanEngineStatus,
    start: startTianxuanEngine,
    stop: stopTianxuanEngine,
  },
}

interface LocalEngineLifecycleProps {
  engineId: ManagedEngineID
  name: string
  /** 启动/已运行后刷新模型列表与连接状态（由模型中心提供）。 */
  onReady?: () => void | Promise<void>
  /** 把探活结果上报给父级，用于隐藏停止态的旧连接错误。 */
  onRunningChange?: (running: boolean) => void
  /** 父级 5s 刷新得到的连接状态；变化时重探活，外部启停也能同步。 */
  externalConnected?: boolean
}

/**
 * 外部本地引擎的卡内生命周期控件：探活状态 + 档位选择 + 启动/停止。
 * 天璇与 Strata 共用同一交互，档位由后端状态接口下发。
 */
export function LocalEngineLifecycle({ engineId, name, onReady, onRunningChange, externalConnected }: LocalEngineLifecycleProps) {
  const api = API[engineId]
  const [running, setRunning] = useState<boolean | null>(null)
  const [currentProfile, setCurrentProfile] = useState('')
  const [profiles, setProfiles] = useState<EngineProfileOption[]>(api.fallbackProfiles)
  const [selected, setSelected] = useState(api.defaultProfile)
  const [busy, setBusy] = useState<'start' | 'stop' | null>(null)
  const onRunningChangeRef = useRef(onRunningChange)
  useEffect(() => { onRunningChangeRef.current = onRunningChange }, [onRunningChange])

  const setRunningState = (v: boolean) => {
    setRunning(v)
    onRunningChangeRef.current?.(v)
  }

  useEffect(() => {
    let alive = true
    api.status()
      .then((s) => {
        if (!alive) return
        const opts = s.profiles && s.profiles.length > 0 ? s.profiles : api.fallbackProfiles
        setRunningState(!!s.running)
        setCurrentProfile(s.profile ?? '')
        setProfiles(opts)
        setSelected(s.profile || s.default_profile || opts[0]?.id || api.defaultProfile)
      })
      .catch(() => {
        if (!alive) return
        setRunningState(false)
        setProfiles(api.fallbackProfiles)
        setSelected(api.defaultProfile)
      })
    return () => { alive = false }
  }, [api, externalConnected])

  const profileLabel = profiles.find(p => p.id === currentProfile)?.label || currentProfile

  const handleStart = async () => {
    setBusy('start')
    try {
      const r = await api.start(selected || api.defaultProfile)
      if (r.started) {
        setRunningState(true)
        setCurrentProfile(r.profile ?? selected)
        if (r.profiles && r.profiles.length > 0) setProfiles(r.profiles)
        await onReady?.()
        message.success(`${name}已就绪（${Math.round((r.waited_ms ?? 0) / 1000)}s）`)
      } else if (r.reason === 'already-running') {
        setRunningState(true)
        setCurrentProfile(r.profile ?? '')
        await onReady?.()
      } else if (r.reason === 'other-engine-running') {
        message.warning(`另一个本地引擎（${r.other ?? 'strata/tianxuan'}）正在运行，请先停止它再启动${name}`)
      } else {
        const tail = r.error ? (r.error.length > 240 ? r.error.slice(-240) : r.error) : ''
        const detail = tail ? `：${tail}` : ''
        message.warning(`${name}启动失败：${r.reason ?? '未知原因'}${detail}${r.log ? `（日志 ${r.log}）` : ''}`)
      }
    } catch {
      message.error(`${name}启动异常`)
    } finally {
      setBusy(null)
    }
  }

  const handleStop = async () => {
    setBusy('stop')
    try {
      const r = await api.stop()
      if (r.stopped) {
        setRunningState(false)
        setCurrentProfile('')
        message.success(`${name}已停止`)
      } else if (r.reason === 'not-running') {
        setRunningState(false)
        setCurrentProfile('')
      } else {
        message.warning(`${name}停止失败：${r.reason ?? '未知原因'}`)
      }
    } catch {
      message.error(`${name}停止异常`)
    } finally {
      setBusy(null)
    }
  }

  return (
    <>
      <StatusChip tone={running ? 'ok' : 'neutral'} dot={!!running}>
        {running ? (profileLabel ? `引擎运行中 · ${profileLabel}` : '引擎运行中') : '引擎已停止'}
      </StatusChip>
      {!running ? (
        <>
          {profiles.length > 1 && (
            <Select
              size="small"
              aria-label={`${name}档位`}
              value={selected}
              onChange={setSelected}
              style={{ minWidth: 170 }}
              options={profiles.map(p => ({ value: p.id, label: p.label }))}
            />
          )}
          <Button size="small" type="primary" loading={busy === 'start'} onClick={handleStart}>
            启动档位
          </Button>
        </>
      ) : (
        <Popconfirm
          title={`停止${name}？`}
          description="生成中的请求会被中断（引擎硬杀有 D 状态风险），请确认引擎空闲。"
          onConfirm={handleStop}
        >
          <Button size="small" danger loading={busy === 'stop'}>停止引擎</Button>
        </Popconfirm>
      )}
    </>
  )
}
