# sin 设定卡 chip 进度行+取消钮（图像等待体验线程 sin 侧收尾）

> 2026-09-25 立项。来源：v4.407 观察池头名「sin 面板取消钮」+v4.406 观察池
> 「sin 进度行」。纯前端，绑定面 717 零变更。目标版本：v4.408.0。

## 1. 背景

- v4.403 给 sin 右栏「角色」卡 chip 加了设定卡一键生成（triptych 快路径，
  自动存回角色库）；busy 只有 chip spinner，无进度、无取消。
- v4.407 让设定卡 Go 链走 beginImageGen/endImageGen（ctx+/interrupt 双达）
  ——sin 侧复用同一 Go 链**自动获得可取消性**（规格 Q5 在案），只缺 UI 出口。
- 角色库编辑器侧 v4.406/v4.407 已落同款（进度行+行尾取消钮）。

## 2. 落地

1. **SinCastPanel**：genId 忙碌期 1s 轮询 `getComfyUITaskProgress`（与编辑器
   同源快照），chips 下方进度行——「当前节点中文名 · 已用时 Xs」/「排队中
   （前有任务）」/「生成中 · 已用时 Xs」，复用 COMFY_NODE_LABELS（后端
   class_type 契约不变）；行尾「取消」钮 → `cancelImageGeneration()`。
   读失败按无进度处理不阻断；空闲即隐藏。
2. **OriginalSinPage.handleCastSheet**：`context canceled` 拒绝语义化为
   「已取消生成」info 提示（对齐编辑器 v4.407 先例），不走错误样式。
3. CSS：`.sin-cast-progress`（chips 与「调整角色」钮之间的弱化文本行）。

## 3. 测试

- SinCastPanel.test：+3 例——进度行出现（节点中文+用时）+取消钮调用全局
  取消+结束后消失；排队中文案+结束消失；进度读取失败不渲染不阻断。
- 既有 3 例回归（mock 补默认空快照防瞬时轮询拿 undefined）。
- 全量 ci 绿 + tsc 0 + eslint 0。

## 4. 门禁

全量 `scripts/ci.ps1` 绿 EXIT=0；版本漂移闸 OK@4.408.0；绑定面 717 零变更。

## 5. 观察池

评分 LLM 流式；iGPU 压力预检；sin 评分入口（评分 v1 只在编辑器参考图上）。
