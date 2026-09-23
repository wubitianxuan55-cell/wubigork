// sin/SinCastPanel.tsx — 右栏「角色」卡：本故事已带入的角色库角色。

import { useState } from 'react'
import { Button, Tooltip } from 'antd'
import { CloseOutlined, IdcardOutlined, PlusOutlined, TeamOutlined } from '@ant-design/icons'
import { PortraitImg } from '../../components/characterlib/PortraitImg'
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
      <Button size="small" icon={<PlusOutlined />} onClick={onOpenPicker} disabled={saving} block>
        {cast.length === 0 ? '选择角色' : '调整角色'}
      </Button>
    </section>
  )
}
