# gaea 3.0 模块协议（预留）

> 状态：接口已实现（2.0 P1–P4），3.0 单窗口编排 UI 待做。

## 1. ModuleRegistry

- 注册：`Module{ID, Name, Intents, Handle}`；`Register` 拒绝重复 ID。
- 派发：`Dispatch(moduleID, intent, input map[string]any) (map[string]any, error)`；未知模块/意图显式报错。
- 绑定：`App.RunModule(moduleID, intent, inputJSON string) (string, error)`。

## 2. 已注册模块（2.0）

| ID | 意图 | 底层方法 |
|----|------|----------|
| gaea | chat | ChatGeneral |
| whisper | chat | WhisperChat |
| novel | create_chapter | CreateChapter |
| office | create | ProposalCreate |
| imagegen | generate | GenerateFreeImage |

## 3. 主脑编排

- `App.MainBrainChat(message)`：意图识别 → BrainSearch 取两脑材料 → Dispatch → 汇总。
- 定位：可选编排入口，不经由任何模块的直接路径。

## 4. 三脑

- 命名空间：`brain.main` / `brain.left` / `brain.right`。
- 访问：`BrainStore.Read/Write/Search/Link/CrossRefs`；绑定 `BrainWrite/BrainSearch/BrainCrossRefs`。
- 跨脑注入：方案生成自动携带 `buildBrainMaterials` 结果（【跨脑记忆·右脑】…）。

## 5. 3.0 窗口编排约定（待实现）

- 单窗口以 ModuleRegistry 为调度表，intent 为任务协议；模块事件经统一事件总线呈现。
- 主脑 UI 形态与"多模块协作任务流"在 3.0 设计。
