# gaea 角色资产 v2·首刀：设定卡生成（三视图，qedit 通道）规格书

> 2026-09-23 立项。来源：用户指令「继续」（T2 余项按序：角色资产 v2 首刀）。
> 上游：长期规划 T2「角色三视图/多姿态设定卡生成；跨图一致性评分；角色画廊
> 化管理」；复用 v4.398 qedit 通道+characterlib 既有参考图列表（画廊即参考图，
> 零 schema）。目标版本：v4.400.0。

## 1. 论点

设定卡（character sheet：三视图并排）是「确定性人设复现」的基建——一张高质量
设定卡本身就是最佳参考图。qedit 通道（参考图+指令→编辑引擎）正是生成它的正确
工具：人物形象与现有参考保持一致、构图按指令展开（三视图并排白背景）。产出
直接追加进 ReferenceImages（角色库画廊=参考图列表，既有管理/删除/落盘全复用）。

## 2. 裁决

| # | 裁决 |
|---|---|
| Q1 | 新绑定 `CharacterGenerateSheet(chJSON) → data URL`（CharlibB，715→716，play）：取角色第一张参考图（参考图优先其次剧照，同 sinCharacterRefPath 口径）→ qedit（txt2img+RefImages[1]+RefMethod=qedit）→ 返回图片 data URL 不自动保存 |
| Q2 | 无参考图报错（「设定卡生成需要至少一张参考图或剧照——qedit 以参考锚定人物」）；后端门：生效后端（PortraitBackend 级联全局）非 comfyui 报「设定卡（Qwen 参考编辑）当前仅 ComfyUI 本地档」 |
| Q3 | prompt 模板纯函数 `buildCharacterSheetPrompt`：三视图并排（正面/侧面/背面全身）+白背景+外观锚点（Appearance/Figure 截断 160 rune）+character sheet 术语；Negative 沿用剧照款 |
| Q4 | 前端：CharacterLibEditor 剧照区旁加「生成设定卡」按钮 → 成功后**追加进 referenceImages**（patch，保存时后端本地化落盘——与手动上传同管线）+ message 提示 |
| Q5 | 一致性评分挂观察池（v1 用 vision LLM 文本锚点评分的方案待拍板）；多姿态扩展（站/坐/动作）挂观察池（同通道改模板即可） |

## 3. 测试

- Go：prompt 纯函数（模板三要素+锚点截断+无外观仍可用）；绑定（fake 捕获 req：
  Mode=txt2img+RefMethod=qedit+refs=1+prompt 含「三视图」；无参考报错；非
  comfyui 报错）。
- 前端：按钮渲染+调用+追加 referenceImages（mock api）。

## 4. 门禁

定向+全量 ci、drift PASS@716、spaceBindings 539。

## 5. 观察池

一致性评分（vision 文本锚点法）；多姿势模板；设定卡批量（三视图分张而非并排）；
sin 侧设定卡入口；评分驱动的「低分提示补参考」。
