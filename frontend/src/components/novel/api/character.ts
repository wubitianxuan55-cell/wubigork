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
