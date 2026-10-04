// useOrgRelation — 组织/关系域（项目内数据，保留原能力）：组织 CRUD、关系
// 建立与删除、两弹窗受控状态、getCharName 反查。跨域依赖（全量数据与刷新）
// 显式注入。2026-10-04 自 CharacterPage.tsx hook 分域首刀逐字搬迁。
import { useState } from 'react'
import { message } from 'antd'
import {
  saveOrganization, deleteOrganization, saveRelationship, deleteRelationship,
} from '../../components/novel/api/character'
import type { CharacterData, OrganizationData, RelationshipData } from '../../types'

export interface OrgRelationDeps {
  characters: CharacterData[]
  organizations: OrganizationData[]
  loadData: () => Promise<void>
}

export function useOrgRelation(deps: OrgRelationDeps) {
  const [modalOrg, setModalOrg] = useState<OrganizationData | null>(null)
  const [editOrg, setEditOrg] = useState<OrganizationData | null>(null)
  const [relTargetId, setRelTargetId] = useState<string>('')
  const [relFromId, setRelFromId] = useState<string>('')
  const [relType, setRelType] = useState<string>('friend')
  const [relModalOpen, setRelModalOpen] = useState(false)

  const getCharName = (id: string) =>
    deps.characters.find(c => c.id === id)?.name || deps.organizations.find(o => o.id === id)?.name || id

  const handleNewOrg = () => {
    const blank: OrganizationData = { id: 'org_' + Date.now(), name: '新组织', type: '', description: '', power_level: '' }
    setModalOrg(blank); setEditOrg({ ...blank })
  }

  const handleSaveOrg = async () => {
    if (!editOrg) return
    try {
      await saveOrganization(editOrg)
      await deps.loadData()
      message.success('组织已保存')
      setModalOrg(null)
    } catch { message.error('保存失败') }
  }
  const handleDeleteOrg = async (id: string) => {
    try {
      await deleteOrganization(id)
      await deps.loadData()
      message.success('组织已删除')
    } catch { message.error('删除失败') }
  }
  const handleAddRel = async () => {
    if (!relFromId || !relTargetId) return
    const rel: RelationshipData = { from_id: relFromId, to_id: relTargetId, relation_type: relType, description: '', intimacy: 0 }
    try {
      await saveRelationship(rel)
      await deps.loadData()
      message.success('关系已建立')
      setRelModalOpen(false)
    } catch { message.error('建立关系失败') }
  }
  const handleDeleteRel = async (rel: RelationshipData) => {
    try {
      await deleteRelationship(rel.from_id, rel.to_id)
      await deps.loadData()
    } catch { message.error('删除关系失败') }
  }

  return {
    modalOrg, setModalOrg, editOrg, setEditOrg,
    relTargetId, setRelTargetId, relFromId, setRelFromId,
    relType, setRelType, relModalOpen, setRelModalOpen,
    getCharName, handleNewOrg, handleSaveOrg, handleDeleteOrg,
    handleAddRel, handleDeleteRel,
  }
}
