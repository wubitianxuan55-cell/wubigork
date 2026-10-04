// useCharDraw — 抽卡域：从角色库随机抽取并加入本书（筛选参数、结果集、
// 逐个/全部加入）。跨域依赖（本书引用集合、全量刷新）显式注入。
// 2026-10-04 自 CharacterPage.tsx hook 分域首刀逐字搬迁。
import { useState } from 'react'
import { message } from 'antd'
import { associateToProject, syncProjectCharacters, drawRandom, type LibraryCharacter } from '../../api/characterlib'
import { errText } from '../../utils/errText'

export interface CharDrawDeps {
  projectRefs: Set<string>
  refreshAll: () => Promise<void>
}

export function useCharDraw(deps: CharDrawDeps) {
  const [drawOpen, setDrawOpen] = useState(false)
  const [drawCount, setDrawCount] = useState(5)
  const [drawGender, setDrawGender] = useState('')
  const [drawTags, setDrawTags] = useState('')
  const [drawChatOnly, setDrawChatOnly] = useState(false)
  const [drawResult, setDrawResult] = useState<LibraryCharacter[]>([])
  const [drawLoading, setDrawLoading] = useState(false)

  // ── 抽卡：从角色库随机抽取，加入当前项目 ──
  const handleDraw = async () => {
    setDrawLoading(true)
    try {
      const items = await drawRandom(drawCount, drawGender, drawTags.trim(), drawChatOnly)
      setDrawResult(items || [])
      if (!items?.length) message.info('没有抽到符合条件的角色，换个条件试试')
    } catch (err: unknown) {
      message.error(errText(err, '抽卡失败'))
    } finally {
      setDrawLoading(false)
    }
  }

  const handleAddDrawn = async (c: LibraryCharacter) => {
    try {
      await associateToProject(c.id, c.roleType || 'supporting')
      await syncProjectCharacters()
      await deps.refreshAll()
      message.success(`「${c.name}」已加入本书`)
    } catch (err: unknown) {
      message.error(errText(err, '加入失败'))
    }
  }

  const handleAddAllDrawn = async () => {
    const pending = drawResult.filter(c => !deps.projectRefs.has(c.id))
    if (!pending.length) return
    try {
      for (const c of pending) await associateToProject(c.id, c.roleType || 'supporting')
      await syncProjectCharacters()
      await deps.refreshAll()
      message.success(`已加入 ${pending.length} 个角色`)
      setDrawResult([])
    } catch (err: unknown) {
      message.error(errText(err, '加入失败'))
    }
  }

  return {
    drawOpen, setDrawOpen,
    drawCount, setDrawCount,
    drawGender, setDrawGender,
    drawTags, setDrawTags,
    drawChatOnly, setDrawChatOnly,
    drawResult, drawLoading,
    handleDraw, handleAddDrawn, handleAddAllDrawn,
  }
}
