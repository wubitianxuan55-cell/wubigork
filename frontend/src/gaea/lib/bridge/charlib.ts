// charlib.ts — CharLibBindings（AppBindings 分域接口之一，Go CharlibB 门面）：
// 角色库分页查询与角色立绘设置（无类型导入，全 Record）。

export interface CharLibBindings {
  GetCharacters(): Promise<Record<string, unknown>>;
  SetCharacterPortrait(characterId: string, portraitPath: string): Promise<void>;
  // CharacterList 角色库分页查询（chatOnly 过滤可聊天角色，返回 {items,total}）。
  CharacterList(query: string, kind: string, chatOnly: boolean, page: number, pageSize: number): Promise<Record<string, unknown>>;
  // ── 批次三b wailsjsCompat 双轨退役转正（Go CharlibB 门面 bindings_charlib.go，
  // 同名前缀；api/characterlib.ts 消费面=角色库档案/加入项目/抽卡/补全/剧照，
  // 角色库板块 characterlib 归 play 空间）──
  // CharacterSave 保存/新建统一角色（Go 返回 characterlib.Character；Bridge 返回
  // Record 让调用方 cast，api/characterlib.saveCharacter 直接透传）。
  CharacterSave(cJSON: string): Promise<Record<string, unknown>>;
  CharacterGet(id: string): Promise<Record<string, unknown>>;
  CharacterDelete(id: string): Promise<void>;
  // 回写：characters.json → 全局库（新导入/补空缺/确认覆盖 计数回执；
  // overwritesJSON 为预览勾选清单 {"<角色ID>":["appearance",...]}，空串=不覆盖非空字段）
  CharacterImportPreview(): Promise<Record<string, unknown>>;
  CharacterImportProject(overwritesJSON: string): Promise<{ imported: number; filled: number; overwritten: number }>;
  CharacterListByProject(): Promise<Array<Record<string, unknown>>>;
  CharacterAssociate(charID: string, role: string): Promise<void>;
  CharacterAssociateTo(projectDir: string, charID: string, role: string): Promise<void>;
  CharacterSetProjectState(charID: string, role: string, arcState: string, status: string): Promise<void>;
  CharacterDissociate(charID: string): Promise<void>;
  CharacterSyncProject(): Promise<void>;
  CharacterDrawRandom(count: number, gender: string, tags: string, chatOnly: boolean): Promise<Array<Record<string, unknown>>>;
  CharacterGenerateFill(chJSON: string): Promise<string>;
  CharacterGenerateRandom(chJSON: string, fields: string): Promise<string>;
  CharacterFillAll(): Promise<Record<string, unknown>>;
  CharacterGeneratePortrait(chJSON: string, model: string): Promise<string>;
  CharacterGeneratePortraitWithRef(chJSON: string, model: string, refImageDataURL: string): Promise<string>;
}