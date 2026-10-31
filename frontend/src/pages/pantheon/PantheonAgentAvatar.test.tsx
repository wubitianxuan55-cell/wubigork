import { describe, expect, it } from 'vitest'
import { render } from '@testing-library/react'
import { PantheonAgentAvatar } from './PantheonAgentAvatar'

const AGENT_IDS = ['grok', 'codex', 'zcode', 'dsh'] as const

describe('PantheonAgentAvatar', () => {
  it('为四席加载各自对应的官方品牌 Logo', () => {
    const { container } = render(
      <>
        {AGENT_IDS.map((agentId) => (
          <PantheonAgentAvatar key={agentId} agentId={agentId} />
        ))}
      </>,
    )

    const avatars = AGENT_IDS.map((agentId) => {
      const avatar = container.querySelector<HTMLElement>(`[data-avatar-agent="${agentId}"]`)
      expect(avatar).toBeTruthy()
      expect(avatar?.getAttribute('aria-hidden')).toBe('true')
      const image = avatar?.querySelector<HTMLImageElement>('img')
      expect(image).toBeTruthy()
      expect(image?.getAttribute('data-agent-logo')).toBeTruthy()
      return image?.getAttribute('src')
    })

    expect(new Set(avatars).size).toBe(4)
  })

  it('头像状态点沿用 Agent 运行态语义', () => {
    const { container } = render(
      <PantheonAgentAvatar agentId="dsh" status="running" />,
    )
    const avatar = container.querySelector('[data-avatar-agent="dsh"]')
    expect(avatar?.getAttribute('data-avatar-status')).toBe('running')
    expect(avatar?.querySelector('[data-agent-logo="DeepSeek"]')).toBeTruthy()
    expect(avatar?.querySelector('.pantheon-agent-avatar-status.is-running')).toBeTruthy()
  })
})
