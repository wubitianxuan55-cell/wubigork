// mock/charlib.ts — 角色库域 dev mock（v4.174 批次三b wailsjsCompat 双轨退役
// 转正启动；角色库板块 characterlib 归 play 空间）。
// Go 侧对应 CharlibB 门面（internal/app/bindings_charlib.go，除 GetCharacters/
// SetCharacterPortrait/CharacterList 已有实现外，本文件补齐角色库档案/加入项目/
// 抽卡/补全/剧照族）。
// 口径：查询类中性空态（浏览器开发无可检索角色库数据）、动作类 no-op（无角色库
// 可写）、生成类诚实样例 JSON（消费方 JSON.parse 兜底）。
import type { AppBindings } from "../bridge";

type CharlibMethods = Pick<
  AppBindings,
  | "CharacterSave" | "CharacterGet" | "CharacterDelete" | "CharacterImportProject"
  | "CharacterListByProject" | "CharacterAssociate" | "CharacterAssociateTo"
  | "CharacterSetProjectState" | "CharacterDissociate" | "CharacterSyncProject"
  | "CharacterDrawRandom" | "CharacterGenerateFill" | "CharacterGenerateRandom"
  | "CharacterFillAll" | "CharacterGeneratePortrait" | "CharacterGeneratePortraitWithRef"
>;

export function buildCharlib(): CharlibMethods {
  return {
    async CharacterSave(_cJSON: string) {
      // 保存/新建统一角色：返回占位角色（真实实现返回 characterlib.Character）。
      return { id: "c-saved" };
    },
    async CharacterGet(_id: string) {
      // 无角色库数据：空对象（消费方空态兜底）。
      return {};
    },
    async CharacterDelete(_id: string) {
      // mock: no-op（浏览器开发无角色库可写）。
    },
    async CharacterImportProject() {
      return 0;
    },
    async CharacterListByProject() {
      return [];
    },
    async CharacterAssociate(_charID: string, _role: string) {
      // mock: no-op。
    },
    async CharacterAssociateTo(_projectDir: string, _charID: string, _role: string) {
      // mock: no-op。
    },
    async CharacterSetProjectState(_charID: string, _role: string, _arcState: string, _status: string) {
      // mock: no-op。
    },
    async CharacterDissociate(_charID: string) {
      // mock: no-op。
    },
    async CharacterSyncProject() {
      // mock: no-op。
    },
    async CharacterDrawRandom(_count: number, _gender: string, _tags: string, _chatOnly: boolean) {
      return [];
    },
    async CharacterGenerateFill(_chJSON: string) {
      // 诚实样例 JSON（消费方 JSON.parse 兜底）。
      return '{"id":"c1"}';
    },
    async CharacterGenerateRandom(_chJSON: string, _fields: string) {
      return '{"id":"c1"}';
    },
    async CharacterFillAll() {
      return { total: 0, filled: 0, skipped: 0, failed: 0, failNames: [] };
    },
    async CharacterGeneratePortrait(_chJSON: string, _model: string) {
      return "";
    },
    async CharacterGeneratePortraitWithRef(_chJSON: string, _model: string, _refImageDataURL: string) {
      return "";
    },
  };
}