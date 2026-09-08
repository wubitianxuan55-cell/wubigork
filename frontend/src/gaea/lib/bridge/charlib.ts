// charlib.ts — CharLibBindings（AppBindings 分域接口之一，Go CharlibB 门面）：
// 角色库分页查询与角色立绘设置（无类型导入，全 Record）。

export interface CharLibBindings {
  GetCharacters(): Promise<Record<string, unknown>>;
  SetCharacterPortrait(characterId: string, portraitPath: string): Promise<void>;
  // CharacterList 角色库分页查询（chatOnly 过滤可聊天角色，返回 {items,total}）。
  CharacterList(query: string, kind: string, chatOnly: boolean, page: number, pageSize: number): Promise<Record<string, unknown>>;
}
