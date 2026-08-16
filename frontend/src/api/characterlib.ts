/**
 * 全局角色库 API（统一角色资产：小说 × 聊天）
 */
import * as App from '../../src/wailsjsCompat'
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

/** 书架项目摘要（对齐 internal/app/shelf.go ProjectCard，用于「加入项目」选择目标小说） */
export interface ShelfProject {
  title: string
  genre: string
  style: string
  path: string
  word_count: number
  chapter_count: number
  created_at: string
  last_opened_at: string
}

/** 书架全部小说项目（角色库「加入项目」时供用户选择目标小说） */
export async function listShelfProjects(): Promise<ShelfProject[]> {
  return App.ListProjects() as unknown as ShelfProject[]
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

/** 把库内角色加入指定小说项目（不改变当前打开的项目） */
export async function associateToProjectDir(projectDir: string, charID: string, role: string): Promise<void> {
  return App.CharacterAssociateTo(projectDir, charID, role)
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

/** 一键随机补齐角色空缺内容：只填充空字段、保留已有内容，不依赖小说项目 */
export async function generateFill(c: Partial<LibraryCharacter>): Promise<LibraryCharacter> {
  const json = await App.CharacterGenerateFill(JSON.stringify(c))
  return JSON.parse(json) as LibraryCharacter
}

/**
 * 按字段随机再生成角色设定。
 * fields: 'all' 全部字段重新随机（含性格，姓名不变）；单个/多个字段用逗号分隔（如 'personality,appearance'）
 */
export async function generateRandom(c: Partial<LibraryCharacter>, fields: string): Promise<LibraryCharacter> {
  const json = await App.CharacterGenerateRandom(JSON.stringify(c), fields)
  return JSON.parse(json) as LibraryCharacter
}

export interface FillAllResult {
  total: number
  filled: number
  skipped: number
  failed: number
  failNames: string[]
}

/** 一键补齐全局角色库所有可见角色的空缺字段（只填空缺，保留已有内容） */
export async function fillAllCharacters(): Promise<FillAllResult> {
  const res = await App.CharacterFillAll()
  return res as unknown as FillAllResult
}

/** 生成角色剧照：按角色字段构建智能 prompt，返回图片 data URL / 远程 URL（不自动保存） */
export async function generatePortrait(c: Partial<LibraryCharacter>, model = ''): Promise<string> {
  return App.CharacterGeneratePortrait(JSON.stringify(c), model)
}
