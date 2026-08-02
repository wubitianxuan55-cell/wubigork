/**
 * 角色数据 API
 * 封装所有后端角色调用，消除 @ts-ignore
 */

import type { CharacterData, OrganizationData, RelationshipData } from '../../../types'
import * as App from '../../../../wailsjs/go/app/App'

export interface CharacterPageData {
  characters: CharacterData[]
  organizations: OrganizationData[]
  relationships: RelationshipData[]
}

/** 加载角色/组织/关系全量数据 */
export async function getCharacters(): Promise<CharacterPageData> {
  const data = await App.GetCharacters()
  return data as unknown as CharacterPageData
}

/** 保存单个角色 */
export async function saveCharacter(char: CharacterData): Promise<void> {
  await App.SaveCharacter(JSON.stringify(char))
}

/** 删除角色 */
export async function deleteCharacter(id: string): Promise<void> {
  await App.DeleteCharacter(id)
}

/** 批量生成角色 */
export async function generateCharacters(count: number): Promise<CharacterPageData> {
  const result = await App.GenerateCharacters(count)
  return result as unknown as CharacterPageData
}

/** 单角色 AI 补全 */
export async function generateSingleCharacter(char: CharacterData): Promise<{ character?: string; reply?: string }> {
  const result = await App.GenerateSingleCharacter(JSON.stringify({
    id: char.id, name: char.name, role_type: char.role_type,
  }))
  return result as unknown as { character?: string; reply?: string }
}

/** 角色 Agent 对话 */
export async function chatCharacter(userMsg: string): Promise<CharacterPageData & { reply?: string }> {
  const result = await App.ChatCharacter(userMsg)
  return result as unknown as CharacterPageData & { reply?: string }
}

/** 生成角色剧照 */
export async function generateCharacterPortrait(charID: string, model?: string): Promise<string> {
  return await App.GenerateCharacterPortrait(charID, model || '')
}

// ── 组织 ──

/** 保存组织 */
export async function saveOrganization(org: OrganizationData): Promise<void> {
  await App.SaveOrganization(JSON.stringify(org))
}

/** 删除组织 */
export async function deleteOrganization(id: string): Promise<void> {
  await App.DeleteOrganization(id)
}

/** 切换组织成员 */
export async function toggleOrgMember(charID: string, orgID: string): Promise<void> {
  await App.ToggleOrgMember(charID, orgID)
}

// ── 关系 ──

/** 保存关系 */
export async function saveRelationship(rel: RelationshipData): Promise<void> {
  await App.SaveRelationship(JSON.stringify(rel))
}

/** 删除关系 */
export async function deleteRelationship(fromID: string, toID: string): Promise<void> {
  await App.DeleteRelationship(fromID, toID)
}
