import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import CharacterLibInspector from './CharacterLibInspector'
import type { LibraryCharacter } from '../../api/characterlib'

const { playCharacterVoiceLine: playCharacterVoiceLineMock } = vi.hoisted(() => ({
  playCharacterVoiceLine: vi.fn(async () => true),
}))

vi.mock('./voicePreview', async (importOriginal) => {
  // 台词解析用真实现（wire 正确性含解析口径），只桩播放面
  const actual = await importOriginal<typeof import('./voicePreview')>()
  return {
    ...actual,
    playCharacterVoiceLine: playCharacterVoiceLineMock,
    stopVoicePreview: vi.fn(),
  }
})

vi.mock('./PortraitImg', () => ({
  PortraitImg: ({ src, alt }: { src?: string; alt?: string }) => (
    <img src={src} alt={alt} />
  ),
}))

function makeCharacter(overrides: Partial<LibraryCharacter> = {}): LibraryCharacter {
  return {
    id: 'c1', name: '苏念', kind: 'custom', gender: 'female', age: '23',
    tags: [], portraitUrl: '', roleType: 'protagonist', status: 'Alive',
    chatEnabled: false, dims: { T: 50, I: 50, S: 50, O: 50, R: 50 },
    createdAt: '', updatedAt: '', hidden: false,
    ...overrides,
  } as unknown as LibraryCharacter
}

function renderInspector(character: LibraryCharacter | null) {
  return render(
    <CharacterLibInspector
      character={character}
      index={0}
      collapsed={false}
      onToggleCollapse={() => {}}
      onEdit={() => {}}
      onSetPersona={() => {}}
      onMemory={() => {}}
      onAssociate={() => {}}
      onDissociate={() => {}}
      onDelete={() => {}}
    />,
  )
}

beforeEach(() => {
  playCharacterVoiceLineMock.mockReset()
  playCharacterVoiceLineMock.mockResolvedValue(true)
})

describe('CharacterLibInspector 语音台词试听', () => {
  it('专属台词行展示台词 + 试听钮，点击整卡透传文件缓存链', async () => {
    renderInspector(makeCharacter({ ttsLine: '夜色温柔，我亦如此。', ttsVoiceId: 'my-clone-9', ttsDesc: '低声细语' }))
    expect(screen.getByText(/语音台词/)).toBeTruthy()
    expect(screen.getByText(/夜色温柔，我亦如此。/)).toBeTruthy()
    expect(screen.getByText(/克隆音色：my-clone-9/)).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: /试听/ }))
    await waitFor(() => {
      expect(playCharacterVoiceLineMock).toHaveBeenCalledTimes(1)
    })
    expect(playCharacterVoiceLineMock).toHaveBeenCalledWith(
      expect.objectContaining({
        id: 'c1', name: '苏念', ttsLine: '夜色温柔，我亦如此。',
        ttsVoiceId: 'my-clone-9', ttsDesc: '低声细语', gender: 'female',
      }),
    )
  })

  it('未填台词：展示回退口径的生效台词（对话样本 > 问候）仍可试听', async () => {
    renderInspector(makeCharacter({ ttsLine: '', dialogueSamples: ['别挡路。'] }))
    // 同句既出现在「语音台词」行也出现在「对话样本」行——按多命中断言
    expect(screen.getAllByText(/别挡路。/).length).toBeGreaterThanOrEqual(1)
    fireEvent.click(screen.getByRole('button', { name: /试听/ }))
    await waitFor(() => {
      expect(playCharacterVoiceLineMock).toHaveBeenCalledWith(
        expect.objectContaining({ dialogueSamples: ['别挡路。'] }),
      )
    })
  })

  it('未填音色：语音音色行不渲染，台词行照常渲染可试听', () => {
    renderInspector(makeCharacter({ ttsVoiceId: '', ttsDesc: '' }))
    expect(screen.queryByText('语音音色')).toBeNull()
    expect(screen.getByRole('button', { name: /试听/ })).toBeTruthy()
  })
})
