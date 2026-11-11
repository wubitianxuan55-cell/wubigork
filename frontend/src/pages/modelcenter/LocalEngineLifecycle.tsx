import { useEffect, useRef, useState } from 'react'
import { Button, message, Popconfirm } from 'antd'
import { StatusChip } from './ui'
import {
  getStrataEngineStatus,
  stopStrataEngine,
  type EngineProfileOption,
  type ManagedEngineStatusResult,
  type ManagedEngineStopResult,
  getLlamaEngineStatus,
  stopLlamaEngine,
  getWhirlEngineStatus,
  stopWhirlEngine,
} from '../../api/engines'

type ManagedEngineID = 'strata' | 'llama' | 'whirl'

interface LifecycleAPI {
  fallbackProfiles: EngineProfileOption[]
  status: () => Promise<ManagedEngineStatusResult>
  stop: () => Promise<ManagedEngineStopResult>
}

// 外部项目的档位契约在 Go 侧是单一来源；这里只做状态接口失败时的展示兜底。
const API: Record<ManagedEngineID, LifecycleAPI> = {
  strata: {
    fallbackProfiles: [
      { id: 'qwen38fn_flash', label: 'qwen3.8-flash-next-uncensored (UD-IQ4_XS)' },
    ],
    status: getStrataEngineStatus,
    stop: stopStrataEngine,
  },
  llama: {
    fallbackProfiles: [
      { id: 'qwen35', label: 'qwen3.6-35b-a3b-mtp（自管 llama）' },
      { id: 'gemma4_26b_a4b', label: 'gemma-4-26B-A4B-it QAT Q4_K_XL + MTP（自管 llama）' },
    ],
    status: getLlamaEngineStatus,
    stop: stopLlamaEngine,
  },
  whirl: {
    fallbackProfiles: [
      { id: 'qwen35_whirl', label: 'qwen3.6-35b-a3b-mtp（WHIRL）' },
    ],
    status: getWhirlEngineStatus,
    stop: stopWhirlEngine,
  },
}

interface LocalEngineLifecycleProps {
  engineId: ManagedEngineID
  name: string
  /** 把探活结果上报给父级，用于隐藏停止态的旧连接错误。 */
  onRunningChange?: (running: boolean) => void
  /** 父级 5s 刷新得到的连接状态；变化时重探活，外部启停也能同步。 */
  externalConnected?: boolean
}

/**
 * 外部本地引擎的卡内生命周期控件：探活状态 + 档位选择 + 启动/停止。
 * Strata / Llama 均不手动捆绑模型，只展示状态和停止；真正的进程由功能绑定
 * 的首次请求按需拉起。模型的手动启动入口在语言模型列表，不在引擎卡。
 */
export function LocalEngineLifecycle({ engineId, name, onRunningChange, externalConnected }: LocalEngineLifecycleProps) {
  const api = API[engineId]
  const [running, setRunning] = useState<boolean | null>(null)
  const [currentProfile, setCurrentProfile] = useState('')
  const [profiles, setProfiles] = useState<EngineProfileOption[]>(api.fallbackProfiles)
  const [busy, setBusy] = useState(false)
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
      })
      .catch(() => {
        if (!alive) return
        setRunningState(false)
        setProfiles(api.fallbackProfiles)
      })
    return () => { alive = false }
  }, [api, externalConnected])

  const profileLabel = profiles.find(p => p.id === currentProfile)?.label || currentProfile

  const handleStop = async () => {
    setBusy(true)
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
      setBusy(false)
    }
  }

  return (
    <>
      <StatusChip tone={running ? 'ok' : 'neutral'} dot={!!running}>
        {running ? (profileLabel ? `引擎运行中 · ${profileLabel}` : '引擎运行中') : '引擎已停止'}
      </StatusChip>
      {!running ? (
        <span className="mc-engine-sub">按需启动 · 由功能绑定首次请求触发</span>
      ) : (
        <Popconfirm
          title={`停止${name}？`}
          description="生成中的请求会被中断（引擎硬杀有 D 状态风险），请确认引擎空闲。"
          onConfirm={handleStop}
        >
          <Button size="small" danger loading={busy}>停止引擎</Button>
        </Popconfirm>
      )}
    </>
  )
}
