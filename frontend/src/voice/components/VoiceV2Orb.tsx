import React from 'react'
import { AudioOutlined, LoadingOutlined, SoundOutlined } from '@ant-design/icons'
import type { VoiceSessionState } from '../session/voiceSession'

export interface VoiceV2OrbProps {
  phase: VoiceSessionState['phase']
  active: boolean
  connected: boolean
}

/** Shared visual core for the product console and the V2 protocol lab. */
export const VoiceV2Orb: React.FC<VoiceV2OrbProps> = ({ phase, active, connected }) => {
  const PhaseIcon = phase === 'thinking'
    ? LoadingOutlined
    : phase === 'speaking'
      ? SoundOutlined
      : AudioOutlined

  return (
    <div
      className={[
        'vc2-orb',
        `is-${phase}`,
        active ? 'is-active' : '',
        connected ? 'is-connected' : 'is-disconnected',
      ].filter(Boolean).join(' ')}
      aria-hidden="true"
    >
      <span className="vc2-orb-orbit is-outer" />
      <span className="vc2-orb-orbit is-inner" />
      <span className="vc2-orb-glow" />
      <span className="vc2-orb-core">
        <PhaseIcon spin={phase === 'thinking'} />
      </span>
      <span className="vc2-orb-wave">
        {[0.42, 0.7, 1, 0.58, 0.82, 0.5, 0.92].map((height, index) => (
          <i key={index} style={{ '--vc2-bar': height } as React.CSSProperties} />
        ))}
      </span>
    </div>
  )
}
