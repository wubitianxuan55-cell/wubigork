# 成人编制链上游对齐：maturecraft 包抽取 + 大纲链/角色链注入（v4.440.0）

> 用户指令「继续」。刀2（承接 v4.439.0 成人小说编制优化）：v4.439 覆盖了
> 正文/重写/收敛/计划/场景，但编制链还有两段上游没接成人向口径——**大纲生成**
> （章级事件在哪排亲密戏）与**角色生成**（欲望线/关系张力的种子）。本刀补齐后，
> 成人向口径贯通 大纲→计划→正文→重写→收敛→场景 全链。落地明细见
> releases/v4.440.0.md。

## 设计

- **包抽取**：工艺文本自 internal/app 抽为 `internal/maturecraft` 单一源
  （正文向 Craft / 计划向 Plan / 大纲向 Outline / 角色向 Character 四组纯函数），
  app / outline / character 三个 agent 共用；app 七处调用点换装，行为零变化
  （文本逐字节未动）。
- **大纲链五路注入**（新增 `OutlineSection`：欲望线一等叙事线/每章关系位移显式
  写进要点/亲密章节带节拍功能/欲望障碍对手性/整章让位须同时推主线）：
  Continue（outline-continue）、ExpandNode（outline-expand）、Chat·ChatNode
  （outline-chat·node，对话式大纲助手同口径）、GenerateOutlineWithDialogue
  （writerSystem 追加）。
- **角色链两路注入**（新增 `CharacterSection`：主要角色带欲望线与亲密张力来源
  供材/关系条目标注张力类型与边界/硬线：不生成涉未成年的欲望设定、不生成美化
  胁迫的关系）：GenerateCharacters（batch 模板）、GenerateSingleCharacter
  （single 模板）。**全局角色库级调用无书级档位，槽天然零渲染**——同模板双
  消费方无串扰。
- 注入纪律不变：空档位零渲染，非成人向项目逐字节零变化。

## 落地情况

> 已发版 v4.440.0（2026-10-02）。Go 3 包改（+1 新包 2 文件）、prompts 6 模板加槽；
> **绑定面 740 零变更**（纯内部注入，无新绑定）。测试：maturecraft 矩阵 5 例
> （含大纲向/角色向新矩阵）+ outline Continue / character batch 注入断言各 1 例
> （含非成人向零出现反向）；三包测试绿；全量 ci CI OK。

## 留池

- plot-branch-browser（分支构思）未注入——分支轴是分歧结构生成，成人向张力
  已由计划/正文层兜底；待真机反馈分支方向过于保守再接。
- worldview / 叙事状态（BuildNovelStatePatch）链未注入——状态机是结构化数据，
  与尺度无关；如亲密状态入状态机再议。
- 大纲向纪律对非虚构/纪实类项目的适配（现按成人向小说口径写死）。
