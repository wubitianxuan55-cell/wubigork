// sin/SinCastPanel.tsx — 右栏「角色」卡：本故事已带入的角色库角色。

import { useEffect, useState } from 'react'
import { Button, Tooltip } from 'antd'
import { CloseOutlined, IdcardOutlined, PlusOutlined, TeamOutlined } from '@ant-design/icons'
import { PortraitImg } from '../../components/characterlib/PortraitImg'
import { COMFY_NODE_LABELS } from '../../components/imagegen/GenerationProgress'
import { cancelImageGeneration, getComfyUITaskProgress } from '../../api/image'
import type { SinCastCharacter } from './useSinCast'

export interface SinCastPanelProps {
  cast: SinCastCharacter[]
  saving: boolean
  onOpenPicker: () => void
  onRemove: (id: string) => void
  /** 生成设定卡（v4.403，sin 侧入口）：取全量角色→qedit 三视图→存回角色库参考图。 */
  onGenerateSheet?: (id: string) => Promise<void>
}

export function SinCastPanel({ cast, saving, onOpenPicker, onRemove, onGenerateSheet }: SinCastPanelProps) {
  // 进行中的生成（单飞：同时只允许一张，本地 ComfyUI 串行）
  const [genId, setGenId] = useState<string | null>(null)
  // ComfyUI 生成进度（v4.408）：生成期间 1s 轮询同源快照，载入/排队可见（与角色库编辑器同款）
  const [comfyProgress, setComfyProgress] = useState<{ status: string; elapsed: number; node: string } | null>(null)
  const runSheet = async (id: string) => {
    if (!onGenerateSheet || genId) return
    setGenId(id)
    try {
      await onGenerateSheet(id)
    } catch {
      // 错误提示由页面层实现负责（handleCastSheet 内 message.error），面板只管 busy 复位
    } finally {
      setGenId(null)
    }
  }
  useEffect(() => {
    if (!genId) {
      setComfyProgress(null)
      return
    }
    let live = true
    const tick = () => {
      getComfyUITaskProgress()
        .then(p => {
          if (live) setComfyProgress({ status: p.status || '', elapsed: p.elapsed || 0, node: p.node || '' })
        })
        .catch(() => { /* 读不到按无进度处理，不阻断生成 */ })
    }
    tick()
    const t = setInterval(tick, 1000)
    return () => { live = false; clearInterval(t) }
  }, [genId])
  // 取消生成（v4.408）：全局 CancelImageGeneration——设定卡链自 v4.407 起走
  // beginImageGen/endImageGen（ctx+/interrupt 双达），此处取消即中止当次提交。
  const handleCancel = async () => {
    try {
      await cancelImageGeneration()
    } catch { /* 取消失败静默——生成自身会结束或超时 */ }
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
                    disabled={saving || (genId !== null && genId !== c.id)}
                    onClick={() => void runSheet(c.id)}
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
      {genId && comfyProgress && (comfyProgress.status === 'running' || comfyProgress.status === 'queued') && (
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
