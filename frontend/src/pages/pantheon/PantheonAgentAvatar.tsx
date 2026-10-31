import React from 'react'
import deepseekLogo from '../../assets/agents/deepseek-dsh.png'
import openaiLogo from '../../assets/agents/openai-codex.png'
import xaiLogo from '../../assets/agents/xai-grok.png'
import zaiLogo from '../../assets/agents/zai-zcode.svg'
import { AGENT_ACCENTS } from './visuals'

interface Props {
  agentId: string
  className?: string
  status?: 'online' | 'offline' | 'running' | 'waiting' | 'finished' | 'skipped' | 'failed'
}

const AGENT_LOGOS: Record<string, { src: string; label: string }> = {
  grok: { src: xaiLogo, label: 'xAI' },
  codex: { src: openaiLogo, label: 'OpenAI' },
  zcode: { src: zaiLogo, label: 'Z.ai' },
  dsh: { src: deepseekLogo, label: 'DeepSeek' },
}

export const PantheonAgentAvatar: React.FC<Props> = ({
  agentId,
  className = '',
  status,
}) => {
  const logo = AGENT_LOGOS[agentId]
  return (
    <span
      className={`pantheon-agent-avatar${className ? ` ${className}` : ''}`}
      data-avatar-agent={agentId}
      data-avatar-status={status}
      data-avatar-brand={logo?.label}
      style={{ '--agent-accent': AGENT_ACCENTS[agentId] || 'var(--pantheon-accent)' } as React.CSSProperties}
      aria-hidden="true"
    >
      {logo && (
        <img
          src={logo.src}
          alt=""
          draggable={false}
          decoding="async"
          data-agent-logo={logo.label}
        />
      )}
      {status && <i className={`pantheon-agent-avatar-status is-${status}`} />}
    </span>
  )
}
