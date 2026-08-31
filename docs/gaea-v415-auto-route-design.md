# gaea-v4.15 设计：聊天路由归位 + 「由谁回答」回显

> 日期：2026-08-31 ｜ 主题：v4.14.0 后主刀（用户拍板收缩版——自动路由 v1 砍掉
> 成本档位机制/开关/UI，只留两块真实价值：①plain 聊天离线裂缝修复（真 bug）
> ②消息级「由谁回答 / 为何 / 花了多少」回显（路线图 §13）。
> 锚点：`.gaea/work/router-v1-explore.md`（探索地图）｜ 原设计
> `docs/gaea-v415-auto-route-design.md` 已按本文件收缩（成本档位/auto_route
> 键/开关卡全部移除）。

## 背景：两个真实问题

1. **离线裂缝（bug）**：`routeModel`（model_router.go:17）每步做离线云过滤，
   但 plain 聊天主链路（chat_service.go:68/:105、chat_handler.go:9）用
   `featureModel("chat")` 原始配置读，**不经 routeModel**——全局离线模式下
   plain 聊天绑了云端引擎仍会发云端（persona 路径却会滤）。「总闸不总」。
2. **透明度缺失**：plain 聊天无 source 标记、无 `model.route` 事件、消息无
   「由谁回答/花了多少」——与 persona 路径不一致，也不满足路线图 §13。

## 方案（最小价值刀，零新增绑定，绑定面 545 不变）

### ① 聊天链路归位（bug 修复）
- `chat_service.go:68`（chatSendPlain）、`:105`（ChatStreamPlain）、
  `chat_handler.go:9` 三处 `eng, model := a.featureModel("chat")` →
  `eng, model, source := a.routeModel("chat")`。
- 语义保证：routeModel 步骤 1 读的就是 `GetFeatureModel("chat")`（与 featureModel
  同源）→ **用户的功能绑定语义逐字节不变**；新增收益 = 离线过滤生效 + 无绑定时
  全局活跃/兜底（与 persona 一致）+ `model.route` 事件。
- `featureModel` 保留（模型中心展示用），不动。

### ② 「由谁回答 / 为何 / 花了多少」回显
- `internal/modelengine/stats.go`：**导出最小费用函数**（不建档位机制）：
  `func EstimateCostCNY(engineID, model string, inTok, outTok int64, usdCny float64) float64`
  ——复用现有 `estimatePrice`（本地 ollama/herdsman/cosyvoice 恒 0；未知模型 0；
  USD 按 usdCny 折算 CNY，CNY 直用）。表驱动测试。
- `chat_service.go`：done 帧（流式路径 ~:166）与 `chatSendPlain` 返回 map
  （:84）各加 `answered_by`：
  ```json
  {"engine":"deepseek","model":"deepseek-v4-flash","source":"feature|global|fallback","cost_cny":0.0123}
  ```
  - engine/model/source 来自 `routeModel("chat")` 返回值；source 即「为何」。
  - cost_cny：用本次实际 token（流式结束帧 usage / Detailed 返回）×
    EstimateCostCNY；usage 在该点不可达时 cost_cny=0（诚实，不虚报）。
  - 顺带在聊天链路补 `emitModelRoute`（现缺失），前端模型中心「当前生效」可展示 chat。
- 前端：
  - `useChatStream.ts` done 分支解析可选字段 `answered_by` → 存入该条消息
    extra（旧事件/旧消息无此字段静默跳过，向后兼容）。
  - 消息行组件底部小字（复用已落盘 `frontend/src/components/chat/AnsweredByLine.tsx`
    ——**修剪**：删 auto-local/auto-cloud 标签，保留 feature/global/fallback +
    未知兜底）：「由 {engine}/{model} 回答 · {sourceLabel} · 约 ¥x.xx」；
    cost_cny=0（本地/未知）时隐藏费用段。
  - 不新增绑定、不动 locales（内容层 zh 硬编码）。

## 明确不做（收缩边界）

- 成本档位机制（CostTier/TierRank/routeAuto）：不做——无官方逐模型缓存/峰谷
  数字，诚实不入表；档位兜底对已绑定用户无感知。
- `auto_route` 配置键、Get/SetAutoRoute 绑定、SchedulingSection 开关卡：不做。
- 按 source 拆分统计、persona 侧 gaea_whisper_causal/retell 的同类裂缝：
  列观察项，不扩面。

## 验证

- Go：EstimateCostCNY 表驱动 + 聊天路由回归（plain + offline + 云端绑定 →
  本地或空；无绑定时全局/兜底）+ 绑定面不变（545，gen_bindings 零改动）+
  全量绿。
- 前端：AnsweredByLine 渲染（有/无费用段/无 answered_by 零渲染）+ useChatStream
  解析；vitest 全量。
- 门禁：go test ./... / vitest / tsc / eslint / drift（545）/ build.bat 冒烟 200。
