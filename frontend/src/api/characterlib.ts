/**
 * 全局角色库 API（统一角色资产：小说 × 聊天）
 */
import * as App from '../../wailsjs/go/app/App'
import type { characterlib } from '../../wailsjs/go/models'

export type LibraryCharacter = characterlib.Character
export type ProjectCharacter = characterlib.ProjectCharacter

export interface CharacterListResult {
  items: LibraryCharacter[]
  total: number
  error?: string
}

export interface CharacterDetail {
  character: LibraryCharacter
  projects: string[]
}

/** 分页查询全局角色库 */
export async function listCharacters(
  query: string,
  kind: string,
  chatOnly: boolean,
  page: number,
  pageSize: number,
): Promise<CharacterListResult> {
  const res = await App.CharacterList(query, kind, chatOnly, page, pageSize)
  return res as unknown as CharacterListResult
}

/** 读取单个角色（含引用它的项目列表） */
export async function getCharacter(id: string): Promise<CharacterDetail> {
  const res = await App.CharacterGet(id)
  return res as unknown as CharacterDetail
}

/** 保存/新建统一角色 */
export async function saveCharacter(c: Partial<LibraryCharacter>): Promise<LibraryCharacter> {
  return App.CharacterSave(JSON.stringify(c))
}

/** 删除角色（内置软隐藏，其余硬删） */
export async function deleteCharacter(id: string): Promise<void> {
  return App.CharacterDelete(id)
}

/** 把当前小说项目 characters.json 导入全局库并建立引用 */
export async function importProjectCharacters(): Promise<number> {
  return App.CharacterImportProject()
}

/** 当前项目已引用的角色 */
export async function listProjectCharacters(): Promise<ProjectCharacter[]> {
  return App.CharacterListByProject()
}

/** 把库内角色加入当前项目 */
export async function associateToProject(charID: string, role: string): Promise<void> {
  return App.CharacterAssociate(charID, role)
}

/** 更新角色在当前项目的状态（定位/弧线/状态，仅项目内覆盖，不影响全局角色） */
export async function setProjectState(charID: string, role: string, arcState: string, status: string): Promise<void> {
  return App.CharacterSetProjectState(charID, role, arcState, status)
}

/** 把角色从当前项目移除（角色保留在全局库） */
export async function dissociateFromProject(charID: string): Promise<void> {
  return App.CharacterDissociate(charID)
}

/** 把项目引用物化回 characters.json（小说 Agent 消费） */
export async function syncProjectCharacters(): Promise<void> {
  return App.CharacterSyncProject()
}

/** 从角色库随机抽卡（小说角色面板不再自行生成角色） */
export async function drawRandom(count: number, gender: string, tags: string, chatOnly: boolean): Promise<LibraryCharacter[]> {
  return App.CharacterDrawRandom(count, gender, tags, chatOnly)
}
