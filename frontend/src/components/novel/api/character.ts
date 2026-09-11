/**
 * 角色数据 API
 * 封装所有后端角色调用，消除 @ts-ignore
 */

import type { CharacterData, OrganizationData, RelationshipData } from '../../../types'
import { app } from '../../../gaea/lib/bridge'

export interface CharacterPageData {
  characters: CharacterData[]
  organizations: OrganizationData[]
  relationships: RelationshipData[]
}

/** 加载角色/组织/关系全量数据 */
export async function getCharacters(): Promise<CharacterPageData> {
  const data = await app.GetCharacters()
  return data as unknown as CharacterPageData
}

/** AI 补齐小说项目角色的空缺字段（只写项目 characters.json，不进全局角色库） */
export async function generateCharacterFill(ch: CharacterData): Promise<CharacterData> {
  const json = await app.GenerateProjectCharacterFill(JSON.stringify(ch))
  return JSON.parse(json) as CharacterData
}

/** 生成小说项目角色剧照（后端自动补齐关键描述并保存到项目） */
export async function generateCharacterPortrait(charID: string, model = ''): Promise<string> {
  return app.GenerateCharacterPortrait(charID, model)
}

/** 合并两个实为同一人的项目角色（mergeID 并入 keepID，保留 keepID） */
export async function mergeCharacters(keepID: string, mergeID: string): Promise<void> {
  await app.MergeCharacters(keepID, mergeID)
}

// ── 组织 ──

/** 保存组织 */
export async function saveOrganization(org: OrganizationData): Promise<void> {
  await app.SaveOrganization(JSON.stringify(org))
}

/** 删除组织 */
export async function deleteOrganization(id: string): Promise<void> {
  await app.DeleteOrganization(id)
}

// ── 关系 ──

/** 保存关系 */
export async function saveRelationship(rel: RelationshipData): Promise<void> {
  await app.SaveRelationship(JSON.stringify(rel))
}

/** 删除关系 */
export async function deleteRelationship(fromID: string, toID: string): Promise<void> {
  await app.DeleteRelationship(fromID, toID)
}
