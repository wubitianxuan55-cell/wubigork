# gaea 角色域现状盘点（2026-09-10，file:line 口径）

> 定位：长期规划**阶段四「闲庭同一人设」开工前读档**（规划铁律：先读完两边
> 落盘格式再写出口，未读完不断言「三套人」）。本文回答：角色到底存几处、
> 谁消费哪处、是不是一套人、真缺口在哪。

## 结论先行

**不是三套人。是一套资产 + 一个项目工作副本。**

- 全局统一角色库（SQLite `characters` 表）= 唯一资产：小说侧字段、聊天侧
  字段、绘梦参考图长在**同一份 Character 上**（`internal/characterlib/model.go:27`，
  包注释明示「角色是独立资产：不属于任何一本小说，也不专属于聊天」）。
- 小说项目目录下的 `characters.json` = **项目工作副本**（有意设计：项目内
  本地编辑、组织/关系、项目内弧线状态隔离在副本），不是待合并的重复存储。
- 两面之间有三座桥（见下），但**编辑不双向流动**——这是「章节里的人和出图
  里的人可能长得不一样」的唯一真实根源。

## 落盘格式（读过才写）

| 面 | 落盘 | 消费方 |
|---|---|---|
| 全局库 | SQLite `characters` + `project_characters` 关联表（`internal/characterlib/db.go:123/152`）；Character 统一模型=小说侧（RoleType/Personality/Background/Appearance/Figure/Motivation/Arc/Status/DialogueSamples）+聊天侧（ChatEnabled/Dims/VoiceGuide/BehaviorRules/EmotionLogic/HiddenPersona）+绘梦侧（PortraitURL/ReferenceImages/GalleryImages，v4.3g 本地化落盘）（model.go:27-77） | 角色库页、聊天人格（ListChatEnabled）、绘梦参考图 |
| 项目副本 | `characters.json`（`internal/project/project.go:199` ReadCharacters；建项时创建空文件 project.go:54） | 创作主线 prompt 注入、小说面板 |
| 关联表 | `project_characters(project_id, character_id, role_in_project, arc_state, status)`（store.go:347 Associate） | ProjectCharactersForNovel 投影为 types.Character（store.go:506） |

## 三条消费主路径

- **章节生成（书斋/闲庭的小说主线）**：`buildCharacterSummary` 读**项目副本**
  characters.json → 角色摘要注入生成 prompt（`internal/app/create_chapter_handler.go:720`）。
- **绘梦（闲庭出图）**：`applyRefCharacter` → `getCharacter(id).referenceImages`
  （**全局库**参考图，v0 最多 4 张、首张作图生图种子、denoise 0.65）
  （`frontend/src/hooks/useImageGenConfig.ts:214`）。
- **聊天（轻语人格）**：同一 Character 的 ChatEnabled/Dims/HiddenPersona；
  whisper presets 经 EnsureBuiltins 种子化进库（store.go:52）——人格也是库内角色。

## 三座桥（现状同步语义）

1. **副本→库（导入）**：`CharacterImportProject` 幂等导入 characters.json 进库
   并建关联（`internal/app/characterlib_handler.go:159-171`）。
2. **库→副本（合入）**：`CharacterAssociateTo` 关联后调 `mergeLibraryRefsIntoProject`
   按 ID 幂等合入 characters.json——**缺失追加、已存在跳过（保留项目内既有
   角色/组织/关系与本地编辑）**（characterlib_handler.go:242-270 起）；注释
   明说「角色库关联表与小说面板读的工作副本是两个数据面」。
3. **章节后对账**：`extractCharactersAfterChapter` AI 提取 characters_appeared →
   先对照副本去重、再对照库 FindByName——**库内已有同名走关联而非新建**，
   未知名字通知前端处置（create_chapter_handler.go:800-833）。

## 真缺口（不是合并存储，是演化漂移）

- **副本→库无回写通道**：项目内改了 Personality/Appearance（两面对象都有这
  些字段），库不知道；反向，关联后库内编辑也不传播到已关联项目（合入只在
  Associate 时发生一次）。
- **外观锚点分裂**：绘梦用**库**参考图，章节正文用**副本**设定——同一角色
  两处分别演化后，「章节里的人」和「出图里的人」会不再是同一个人。这正是
  阶段四方向句「角色在章节和出图里是同一个人」要防的事。

## 出口判据草案（供阶段四落刀，不做自动双向同步）

1. **外观锚点单一来源**：绘梦参考槽在库内参考图为空时，回退读库内
   Appearance 字段文案（提示「无参考图，按文字设定出图」），不再静默空槽。
2. **显式回写**：副本→库加一个用户点按钮的「把项目内设定回写角色库」
   （按字段合并：空字段填入、非空字段用户逐项确认），不做后台自动同步。
3. **关联即快照语义写进 UI**：关联卡片注明「项目内弧线/状态以项目为准」，
   消除「改了库就该变」的误解。

## 边界（维持不做）

不做后台自动双向同步（漂移是本地编辑隔离的代价，显式桥足够）；不做批量
迁移工具；轻语主动开口不在本阶段（空房间不加家具，主动开口极稀）。
