# gaea 绘梦·阶段一刀 E：画室消耗（月度聚合 + 折叠面板）规格书

> 2026-09-17 立项。来源：用户指令「继续」。规格 docs/gaea-dream-studio-nextgen-
> 2026-09.md §4 刀 E 轻量收尾：「每次生成记录『张数 × 后端单价』，画室侧折叠
> 面板显示『本月画室消耗』（本地免费即显示 0 元），为留存与未来计量铺垫。
> 文案用『画室消耗/创作记录』，不用积分话术」。
> 前置核对：台账已逐条记 Cost（目录单价口径）+CreatedAt（T0 落地）；目录
> UnitCost 表在册（本地=0 / glm-image=0.1 CNY/张 / qwen-image-3.0-pro=0.18
> CNY/张 / grok·cogview=未定价）。**本刀=聚合面+消费面，登记侧零改动**。
> 本刀合入即绘梦阶段一（A/B/C/D/E）全清。
> 目标版本：v4.329.0。

## 1. 论点

生成侧的消耗记录（台账 Cost 字段）已存在但没有任何聚合视图——用户看不到
「本月画了几张、花了多少」；目录单价已标档但只活在模型目录卡片上。

## 2. 裁决

| # | 裁决 |
|---|---|
| Q1 | +1 绑定 `ImageHubMonthlyUsage(space)`（705→706，ImageB 门面——图像域数据面） |
| Q2 | 聚合口径=台账 list 全量 → 本地时区当月（CreatedAt 前缀 `YYYY-MM` 匹配）→ 按 model+backend 分组计数；UnitCost 取目录表（记录里的 Cost 与目录不一致时**以记录为准**——台账是事实源） |
| Q3 | 估算：记录 Cost 可解析 `X CNY/张` → 张数×X（两位小数）；"0"→免费；"未定价"/空→unpriced 计数诚实可见，不猜 |
| Q4 | 合计行：可算部分相加；全免费→「0 CNY（本地免费）」；有未定价→注明「另有 N 张未定价」 |
| Q5 | 面板=ImageGenPage 画室侧折叠面板（antd Collapse 默认收起，标题「本月画室消耗」）；space 固定 play（画室口径；work 空间消耗走成本归因不在此面板） |
| Q6 | 文案「画室消耗/创作记录」，不用积分话术（规格原文）；零后端计费动作（纯只读聚合） |

## 3. 落地

- **Go** `internal/app/imagehub_usage.go`：`ImageHubMonthlyUsage(space string)
  (ImageHubMonthlyUsageReport, error)`——聚合+typed camelCase 报告
  （month/space/total/byModel[]{model,backend,count,unitCost,estCost}/
  estimated/unpriced/freeCount）。空台账=零报告不算错。
- **前端** `StudioUsagePanel.tsx`（Collapse：标题带总数与合计；展开=按模型行
  表+免费/未定价注记+说明脚注）；ImageGenPage 顶栏下方挂点；bridge/image.ts
  +视图类型；mock +1；spaceBindings +1（play）；bindingNames 再生（706）；
  计数锁 528→529。
- 测试：Go（当月过滤跨月剔除/分组计数/est 算术/未定价/空台账）+前端
  （渲染聚合行/空态）+mock 契约。

## 4. 观察池（本刀不做）

按 sourceBoard 细分（绘梦/小说配图/角色库分列）；单价表动态化（用户自定义
目录价）；消耗趋势图；work 空间并入成本归因视图。

## 5. 门禁

go build/vet + 触面包测试、tsc -b、eslint、vitest 全量、drift OK@706、
计数锁 529、ci.ps1、版本三处 4.329.0。
