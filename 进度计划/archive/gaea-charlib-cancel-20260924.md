# gaea 角色库生成取消（v4.407.0）规格书

> 2026-09-24 立项。来源：图像等待体验线程观察池头名——剧照/设定卡接入全局
> 可取消机制。纯接线，绑定面 717 零变更（复用 CancelImageGeneration）。
> 目标版本：v4.407.0。

## 1. 论点

剧照/设定卡接了进度（v4.406）但没有出口——20 分钟冷载时只能干看着。全局
取消机制（beginImageGen/endImageGen + CancelImageGeneration：ctx 退出+
ComfyUI /interrupt 双达，T6-4.1）已完备，角色两链挂上同槽即可。

## 2. 裁决

| # | 裁决 |
|---|---|
| Q1 | 剧照/设定卡两链改走 `beginImageGen/endImageGen` 对（替换 v4.406 的裸 `defer clearComfyTaskProgress`）——ID 守卫下完成清理由 endImageGen 承担，且修掉 v4.406 的一个潜在缺陷：无条件 clear 会擦掉并发绘梦任务的进度快照 |
| Q2 | 并发语义：begin 覆盖 imageGenCancel——编辑器后提交时 CancelImageGeneration 的取消目标是编辑器这次（最近提交=用户直觉目标）；绘梦队列自身 ctx 由其 deferred endImageGen 持有不丢失。注释在代码 |
| Q3 | `resetComfyCancel()` 镜像队列路径（T6-4.1 本地取消标记复位——取消后可再次提交） |
| Q4 | 前端：进度行内联「取消」钮（仅生成中可见）→ cancelImageGeneration()；`context canceled` 拒绝语义化为「已取消生成」（剧照/设定卡两处 catch），不再报错误样式 |
| Q5 | sin 侧设定卡入口复用同一 Go 链自动获得可取消性（UI 取消钮 sin 面板暂不挂，挂池） |

## 3. 测试

- Go `TestCharacterGenerateSheetCancel`：假 ComfyUI /history 挂起模拟冷载，
  CancelImageGeneration → 生成以 context canceled 退出 + /interrupt 恰一次
  + imageGenRunning 复位。
- 回归：sheet/portrait/ref-slot 全链绿（期间修出 mediaState 嵌入 core 为 nil
  时 resetComfyCancel 触 cfg 解引用 panic——测试字面量补 core）。
- 前端：进度行内取消钮点击调用全局取消；context canceled 拒绝显示「已取消
  生成」而非错误。

## 4. 门禁

全量 ci、版本漂移闸 OK@4.407.0、绑定面 717 零漂移。

## 5. 观察池

sin 面板取消钮；评分 LLM 流式；iGPU 压力预检。
