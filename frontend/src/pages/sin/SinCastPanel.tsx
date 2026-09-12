// sin/SinCastPanel.tsx — 右栏「角色」卡：本故事已带入的角色库角色。

import { Button, Tooltip } from 'antd'
import { CloseOutlined, PlusOutlined, TeamOutlined } from '@ant-design/icons'
import { PortraitImg } from '../../components/characterlib/PortraitImg'
import type { SinCastCharacter } from './useSinCast'

export interface SinCastPanelProps {
  cast: SinCastCharacter[]
  saving: boolean
  onOpenPicker: () => void
  onRemove: (id: string) => void
}

export function SinCastPanel({ cast, saving, onOpenPicker, onRemove }: SinCastPanelProps) {
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
