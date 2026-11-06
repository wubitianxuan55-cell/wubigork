# gaea sin 侧设定卡入口（v4.403.0）规格书

> 2026-09-23 立项。来源：用户指令「继续」（观察池在册项：sin 侧设定卡入口）。
> 复用 v4.400~402 设定卡通道（qedit 三视图）。纯前端编排，绑定面 716 零变更、
> spaceBindings 539。目标版本：v4.403.0。

## 1. 论点

sin（原罪）写作流里角色一致性靠插图参考槽（v4.399 qedit 三图槽）——设定卡
是最佳参考图，但 sin 侧要生成得绕去角色库编辑器。在右栏「角色」卡的角色
chip 上给一键入口：点一下生成三视图设定卡并自动存回角色库参考图，写作流
不中断。

## 2. 裁决

| # | 裁决 |
|---|---|
| Q1 | 挂点=SinCastPanel 角色 chip（IdcardOutlined 图标钮，与移除钮并列）；picker 是纯选择面不掺生成动作 |
| Q2 | 单击=固定 triptych 三视图并排（全模板矩阵留在角色库编辑器——sin 入口做快路径不做全功能复刻）；产物自动存回：getCharacter(id) 取全量（参考图列表在库内）→generateCharacterSheet(full,'triptych')→saveCharacter(追加 referenceImages)→reloadLibrary 刷新 picker 数据；落盘走保存管线与角色库同路 |
| Q3 | busy=chip 级单飞（同时只允许一张，本地 ComfyUI 串行）；面板 runSheet 吞错只管复位（错误提示由页面层 handleCastSheet message.error 负责——契约注释在面板） |
| Q4 | SinSidePanel/SinCastPanel 的 onGenerateSheet 均可选 prop（未传不渲染入口=向后兼容；SinSidePanel 既有测试夹具零改动面最大） |
| Q5 | 一致性评分仍挂观察池待拍板；分张并发档挂池 |

## 3. 测试

- SinCastPanel.test 新建 3 例：点击按 id 调用+复位；单飞（进行中禁他钮/
  失败复位）；未传 prop 不渲染（兼容）。
- SinSidePanel.test 夹具补 prop（1 处）。
- Go 零变更（通道 v4.400~402 既有覆盖）。

## 4. 门禁

全量 ci、版本漂移闸 OK@4.403.0、绑定面 716 零漂移、spaceBindings 539。

## 5. 观察池

一致性评分（vision 文本锚点法待拍板）；chip 长按/右键出模板菜单（需要再
评估 chip 空间）；分张并发档；评分驱动补参考。
