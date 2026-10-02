// sin/SinCastPanel.tsx — 右栏「角色」卡：本故事已带入的角色库角色。

import { useEffect, useRef, useState } from 'react'
import { Button, Tooltip } from 'antd'
import { AimOutlined, CloseOutlined, IdcardOutlined, PlusOutlined, TeamOutlined } from '@ant-design/icons'
import { PortraitImg } from '../../components/characterlib/PortraitImg'
import { COMFY_NODE_LABELS } from '../../components/imagegen/GenerationProgress'
import { cancelImageGeneration } from '../../api/image'
import { useComfyTaskProgress } from '../../components/imagegen/useComfyTaskProgress'
import {
  cancelIllustration, enqueueIllustration, illustrationQueueSnapshot, subscribeIllustrationQueue,
} from './illustrationQueue'
import type { SinCastCharacter } from './useSinCast'

export interface SinCastPanelProps {
  cast: SinCastCharacter[]
  saving: boolean
  onOpenPicker: () => void
  onRemove: (id: string) => void
  /** 生成设定卡（v4.403，sin 侧入口）：取全量角色→qedit 三视图→存回角色库参考图。 */
  onGenerateSheet?: (id: string) => Promise<void>
  /** 一致性评分（v4.410，sin 侧快路径）：评角色最新一张参考图，结果由页面层 Modal 呈现。 */
  onScore?: (id: string) => Promise<void>
}

export function SinCastPanel({ cast, saving, onOpenPicker, onRemove, onGenerateSheet, onScore }: SinCastPanelProps) {
  // 进行中的生成（单飞：同时只允许一张，本地 ComfyUI 串行）
  const [genId, setGenId] = useState<string | null>(null)
  // 进行中的评分（单飞；与生成互斥——本地视觉模型与 ComfyUI 同 GPU）
  const [scoreId, setScoreId] = useState<string | null>(null)
  // v4.427：设定卡与流内插图/画廊重生成走同一条串行队列（illustrationQueue）——
  // 此前三条链各自直发 ComfyUI，进度互相串台（两条进度条跳同一个百分比）、
  // 任一侧取消误杀另一侧的任务。入队后一次只跑一个，进度与取消都有归属。
  const sheetTokenRef = useRef(0)
  const [queuedAhead, setQueuedAhead] = useState(0)
  const runSheet = async (id: string) => {
    if (!onGenerateSheet || genId) return
    setGenId(id)
    const { promise, token } = enqueueIllustration(() => onGenerateSheet(id))
    sheetTokenRef.current = token
    try {
      await promise
    } catch {
      // 错误提示由页面层实现负责（handleCastSheet 内 message.error）；
      // 排队中取消的哨兵拒绝也走这里（busy 复位即可，取消不是失败）
    } finally {
      setGenId(null)
      sheetTokenRef.current = 0
    }
  }
  // 队列位次：排队中显示「前面还有 N 张」而不是错拿别人的 ComfyUI 进度
  useEffect(() => {
    if (!genId) return
    const sync = () => {
      const pos = illustrationQueueSnapshot().positionOf(sheetTokenRef.current)
      setQueuedAhead(pos > 1 ? pos - 1 : 0)
    }
    sync()
    return subscribeIllustrationQueue(sync)
  }, [genId])
  const sheetQueued = genId !== null && illustrationQueueSnapshot().positionOf(sheetTokenRef.current) > 0
  // ComfyUI 生成进度（v4.408，FE6-06 收敛）：轮到自己在跑才轮询同源快照（排队中
  // 不轮询、不拿别人的进度）；取消链路（摘队位/中断 ComfyUI）不经过轮询，语义不变。
  const comfyProgress = useComfyTaskProgress(genId !== null && !sheetQueued)
  // 取消生成（v4.408）：排队中只摘自己的队位（不碰别人在跑的任务）；轮到自己
  // 在跑才中断 ComfyUI 当前任务（ctx+/interrupt 双达，与流内插图取消同款制导）。
  const handleCancel = async () => {
    const mode = cancelIllustration(sheetTokenRef.current)
    if (mode === 'active') {
      try {
        await cancelImageGeneration()
      } catch { /* 取消失败静默——生成自身会结束或超时 */ }
    }
  }
  // 一致性评分（v4.410）：面板只管 busy 复位，取图/调链/Modal 由页面层负责
  const runScore = async (id: string) => {
    if (!onScore || scoreId || genId) return
    setScoreId(id)
    try {
      await onScore(id)
    } catch {
      // 错误提示由页面层实现负责，面板只管复位
    } finally {
      setScoreId(null)
    }
  }

  return (
    <section className="sin-card sin-cast-card">
      <div className="sin-card-title">
        <TeamOutlined /> 角色
        <span className="sin-cast-count">{cast.length > 0 ? `${cast.length} 人` : ''}</span>
      </div>
      {cast.length === 0 ? (
        <p className="sin-card-text">
          从角色库挑人：他们的外观、性格、背景会随故事一起交给 AI，写出来不走形。
        </p>
      ) : (
        <div className="sin-cast-chips">
          {cast.map((c) => (
            <span className="sin-cast-chip" key={c.id} title={`${c.name}${c.personality ? ' · ' + c.personality : ''}`}>
              <span className="sin-cast-chip-art">
                <PortraitImg src={c.portraitUrl} alt={c.name} />
              </span>
              <span className="sin-cast-chip-name">{c.name}</span>
              {onGenerateSheet && (
                <Tooltip title="生成设定卡（三视图，自动存入角色库参考图）">
                  <Button
                    size="small"
                    type="text"
                    icon={<IdcardOutlined />}
                    aria-label={`生成 ${c.name} 的设定卡`}
                    loading={genId === c.id}
                    disabled={saving || scoreId !== null || (genId !== null && genId !== c.id)}
                    onClick={() => void runSheet(c.id)}
                  />
                </Tooltip>
              )}
              {onScore && (
                <Tooltip title="一致性评分（对照角色文字设定评估最新参考图）">
                  <Button
                    size="small"
                    type="text"
                    icon={<AimOutlined />}
                    aria-label={`评分 ${c.name} 的参考图`}
                    loading={scoreId === c.id}
                    disabled={saving || genId !== null || (scoreId !== null && scoreId !== c.id)}
                    onClick={() => void runScore(c.id)}
                  />
                </Tooltip>
              )}
              <Tooltip title="从故事移除（不改角色库）">
                <Button
                  size="small"
                  type="text"
                  icon={<CloseOutlined />}
                  aria-label={`移除 ${c.name}`}
                  disabled={saving}
                  onClick={() => onRemove(c.id)}
                />
              </Tooltip>
            </span>
          ))}
        </div>
      )}
      {genId && sheetQueued && (
        <div className="sin-cast-progress" data-testid="sin-cast-progress" aria-live="polite">
          排队中{queuedAhead > 0 ? `（前面还有 ${queuedAhead} 张）` : '（即将开始）'}
          <Button
            size="small"
            type="text"
            className="sin-cast-progress-cancel"
            data-testid="sin-cast-cancel"
            aria-label="取消生成"
            title="取消排队（不影响正在生成的任务）"
            onClick={() => void handleCancel()}
          >
            取消
          </Button>
        </div>
      )}
      {genId && !sheetQueued && (comfyProgress.status === 'running' || comfyProgress.status === 'queued') && (
        <div className="sin-cast-progress" data-testid="sin-cast-progress" aria-live="polite">
          {comfyProgress.status === 'queued'
            ? '排队中（前有任务）'
            : comfyProgress.node
              ? `${COMFY_NODE_LABELS[comfyProgress.node] || comfyProgress.node} · 已用时 ${comfyProgress.elapsed}s`
              : `生成中 · 已用时 ${comfyProgress.elapsed}s`}
          <Button
            size="small"
            type="text"
            className="sin-cast-progress-cancel"
            data-testid="sin-cast-cancel"
            aria-label="取消生成"
            title="取消生成（中断 ComfyUI 当前任务）"
            onClick={() => void handleCancel()}
          >
            取消
          </Button>
        </div>
      )}
      <Button size="small" icon={<PlusOutlined />} onClick={onOpenPicker} disabled={saving} block>
        {cast.length === 0 ? '选择角色' : '调整角色'}
      </Button>
    </section>
  )
}
