// useProjectCharacterEdit — 角色详情抽屉的「本书局部设定」域：定位/弧线/状态、
// 职业体系（即时保存）、AI 补齐/剧照/合并、移出本书。跨域依赖（全量数据写入器
// 与刷新）经 data 切片显式注入。2026-10-04 自 CharacterPage.tsx hook 分域
// 首刀逐字搬迁。
import { useCallback, useState } from 'react'
import type { Dispatch, SetStateAction } from 'react'
import { message } from 'antd'
import {
  getCharacters, setCharacterCareer, removeCharacterCareer,
  generateCharacterFill, generateCharacterPortrait, mergeCharacters,
} from '../../components/novel/api/character'
import { setProjectState, syncProjectCharacters, dissociateFromProject } from '../../api/characterlib'
import { errText } from '../../utils/errText'
import { normalizeCharacterStatus } from '../../utils/characterStatus'
import type { CharacterData, OrganizationData, RelationshipData } from '../../types'

/** 数据面切片：编辑域写回全量三表并触发刷新（由 useCharacterData 提供）。 */
export interface CharacterDataSlice {
  setCharacters: Dispatch<SetStateAction<CharacterData[]>>
  setOrganizations: Dispatch<SetStateAction<OrganizationData[]>>
  setRelationships: Dispatch<SetStateAction<RelationshipData[]>>
  loadData: () => Promise<void>
  refreshAll: () => Promise<void>
}

export function useProjectCharacterEdit(data: CharacterDataSlice) {
  const [projectEdit, setProjectEdit] = useState<CharacterData | null>(null)
  const [filling, setFilling] = useState(false)
  const [genPortrait, setGenPortrait] = useState(false)
  const [mergeOpen, setMergeOpen] = useState(false)
  const [mergeTargetId, setMergeTargetId] = useState('')
  const [peRole, setPeRole] = useState('')
  const [peArc, setPeArc] = useState('')
  const [peStatus, setPeStatus] = useState('')
  // 职业编辑（t5 §7.6：即时保存，不随「保存本书状态」）
  const [peCareerMain, setPeCareerMain] = useState('')
  const [peCareerMainStage, setPeCareerMainStage] = useState(1)
  const [peNewSub, setPeNewSub] = useState('')
  const [peNewSubStage, setPeNewSubStage] = useState(1)

  // 职业操作后刷新全量并同步 Drawer 内快照（即时保存语义）
  const reloadProjectEdit = useCallback(async (charID: string) => {
    const fresh = await getCharacters()
    data.setCharacters(fresh.characters || [])
    data.setOrganizations(fresh.organizations || [])
    data.setRelationships(fresh.relationships || [])
    const found = (fresh.characters || []).find(c => c.id === charID)
    if (found) setProjectEdit(found)
  }, [data])

  const handleSetMainCareer = async () => {
    if (!projectEdit) return
    const name = peCareerMain.trim()
    if (!name) { message.warning('请填写职业名称（名称即 ID）'); return }
    try {
      await setCharacterCareer(projectEdit.id, { is_main: true, career_name: name, stage: peCareerMainStage })
      message.success(`主职业已设为「${name}·${peCareerMainStage}阶」`)
      await reloadProjectEdit(projectEdit.id)
    } catch (err) { message.error(errText(err, '设置主职业失败')) }
  }

  const handleRemoveMainCareer = async () => {
    if (!projectEdit) return
    try {
      await removeCharacterCareer(projectEdit.id, { is_main: true })
      message.success('主职业已移除')
      await reloadProjectEdit(projectEdit.id)
    } catch (err) { message.error(errText(err, '移除主职业失败')) }
  }

  const handleAddSubCareer = async () => {
    if (!projectEdit) return
    const name = peNewSub.trim()
    if (!name) { message.warning('请填写副职业名称'); return }
    try {
      await setCharacterCareer(projectEdit.id, { is_main: false, career_name: name, stage: peNewSubStage })
      message.success(`副职业已添加「${name}·${peNewSubStage}阶」`)
      setPeNewSub('')
      await reloadProjectEdit(projectEdit.id)
    } catch (err) { message.error(errText(err, '添加副职业失败')) }
  }

  const handleRemoveSubCareer = async (name: string) => {
    if (!projectEdit) return
    try {
      await removeCharacterCareer(projectEdit.id, { is_main: false, career_name: name })
      message.success(`副职业「${name}」已移除`)
      await reloadProjectEdit(projectEdit.id)
    } catch (err) { message.error(errText(err, '移除副职业失败')) }
  }

  const openProjectEdit = (ch: CharacterData) => {
    setProjectEdit(ch)
    setPeRole(ch.role_type || 'supporting')
    setPeArc(ch.arc || '')
    setPeStatus(normalizeCharacterStatus(ch.status))
    setPeCareerMain(ch.main_career_id || '')
    setPeCareerMainStage(ch.main_career_stage || 1)
    setPeNewSub('')
    setPeNewSubStage(1)
  }

  const handleSaveProjectState = async () => {
    if (!projectEdit) return
    try {
      await setProjectState(projectEdit.id, peRole, peArc, peStatus)
      await syncProjectCharacters()
      await data.loadData()
      message.success(`已更新「${projectEdit.name}」在本书的状态（全局角色未动）`)
      setProjectEdit(null)
    } catch (err: unknown) {
      message.error(errText(err, '保存失败'))
    }
  }

  const handleRemoveFromProject = async (ch: CharacterData) => {
    try {
      await dissociateFromProject(ch.id)
      await syncProjectCharacters()
      await data.refreshAll()
      message.success(`「${ch.name}」已从本书移除（角色保留在角色库）`)
      setProjectEdit(null)
    } catch (err: unknown) {
      message.error(errText(err, '移除失败'))
    }
  }

  // ── 章节捕获角色的补齐 / 剧照 / 合并（未入库时可用，只写本书） ──
  const handleProjectFill = async (ch: CharacterData) => {
    if (filling) return
    setFilling(true)
    try {
      const updated = await generateCharacterFill(ch)
      setProjectEdit(updated)
      await data.loadData()
      message.success('已补齐空缺字段（只写本书，未动角色库）')
    } catch (err: unknown) {
      message.error(errText(err, '补齐失败'))
    } finally {
      setFilling(false)
    }
  }

  const handleProjectPortrait = async (ch: CharacterData) => {
    if (genPortrait) return
    setGenPortrait(true)
    try {
      await generateCharacterPortrait(ch.id)
      await data.loadData()
      message.success('剧照已生成')
    } catch (err: unknown) {
      message.error(errText(err, '剧照生成失败'))
    } finally {
      setGenPortrait(false)
    }
  }

  const handleMergeConfirm = async () => {
    if (!projectEdit || !mergeTargetId) return
    try {
      // 当前角色（A）并入目标角色（B），保留 B
      await mergeCharacters(mergeTargetId, projectEdit.id)
      setMergeOpen(false)
      setMergeTargetId('')
      setProjectEdit(null)
      await data.refreshAll()
      message.success('已合并：空缺信息已补充，关系与组织引用已重定向')
    } catch (err: unknown) {
      message.error(errText(err, '合并失败'))
    }
  }

  return {
    projectEdit, setProjectEdit, openProjectEdit, reloadProjectEdit,
    peRole, setPeRole, peArc, setPeArc, peStatus, setPeStatus,
    peCareerMain, setPeCareerMain, peCareerMainStage, setPeCareerMainStage,
    peNewSub, setPeNewSub, peNewSubStage, setPeNewSubStage,
    filling, genPortrait, mergeOpen, setMergeOpen, mergeTargetId, setMergeTargetId,
    handleSetMainCareer, handleRemoveMainCareer, handleAddSubCareer, handleRemoveSubCareer,
    handleSaveProjectState, handleRemoveFromProject,
    handleProjectFill, handleProjectPortrait, handleMergeConfirm,
  }
}
