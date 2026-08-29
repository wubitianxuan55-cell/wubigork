# gaea 双空间 S1.3+S1.4+S1.5 按空间装配 · 设计（实现权威）

> 依据：只读勘察报告（2026）。S1.1/S1.2 已完成。本文件为实现权威。
> 实现顺序：**装配族**（S1.3-A 模型 profile + S1.3-B 工具标签 + S1.5-A 权限策略装配）→ **S1.4 任务分账**（可与装配族并行）→ **S1.5-B play 护栏**（依赖装配族配置键，后派）。

## 0. 关键勘察事实
1. **双路由平面**：Plane A = Seam 注册表（provider/provider.go:326-361 Register/New、llm.go:115-127 NewLLM、boot.go:515-530 NewProvider、gaea.toml providers[].kind）；Plane B = 模型中心功能绑定（app/model_router.go:14-49 routeModel + :75-93 routeSensitiveLocal/routeOfficeLocal、config GetFeatureModel、feature_model_handler.go:17-59 功能域 chat/whisper/novel/office/gaea/characterlib/routine）。**play 域（轻语/小说/角色库/绘梦）全走 Plane B 直连绑定**（whisper_handler.go:136、create_chapter_handler.go:277），不经过工具面与审批引擎。
2. **Registry 是 per-Build 实例**（boot.go:152）；控制器按空间构建；GaeaSpaceActivate 不重建运行中引擎（gaea_spaces.go:114-118）。→ **工具空间过滤在装配期物理过滤**（addBuiltins/ExtraTools/startPlugins spec 层），不改 executeOne；task.go:356,366 / boot.go:284 三处 ActiveSchemas=父 reg.Schemas() 的子代理缓存对齐自动一致。运行时过滤会破坏该对齐，**已否决**。
3. **tasks Manager 单队列 FIFO**（GetFirstQueued ORDER BY created_at，tasks.go:738-748），全局 sem MaxConcurrent=1（gaea_tasks.go:45-47 + tasks.go:118），HasActive 去重是 kind 全局（tasks.go:438-445）；现有 kind 全 work（price_fetch/price_fetch_all/file_index，gaea_tasks.go:48-50）；生图不在任务体系（mediaState 单飞）；GPU 串行由 herdsman local_concurrency=1 保证。
4. **权限引擎纯函数**（permission.go:90-128 Decide）+ 单 Policy per controller（boot.go:182）+ hardAskTools 包级 map（controller_approval.go:20-27）+ approval_timeout/persist_allow 已是 Options 注入（controller.go:165-172、boot.go:425-438）。→ per-space 策略=「配置合并 + boot 取值点替换 + hardAsk 参数化」；**play 默认 mode=allow + hard_ask=[] 即天然不弹审批卡**。

## 1. S1.3-A 模型 profile 按空间
- 配置段：**`[space_profiles.<space>]`**（拒 per-provider space 字段：provider 是实例声明、空间是路由选择）；字段 = 现有模型选择键的引用（走 `ResolveModel` 既有链），如 `[space_profiles.work] chat="…" / office="…"`、`[space_profiles.play] chat="…" whisper="…" novel="…"`（对应 feature_model_handler 功能域）。
- 桌面端 play 模型：走 `bridge_feature` 键复用既有功能绑定（**零新增绑定**）；BalanceKind/usage 分账随所选 entry + 按空间分区的会话日志自动成立。
- 实现点：config.go 加段解析 + `cfg.SpaceProfile(space)`；boot 装配时按控制器空间取 profile 注入。

## 2. S1.3-B 工具空间标签 + 装配期过滤
- 标签=数据：新建 `internal/gaea/tool/builtin/spacetags.go` 分类表（工具名→work/play/shared，缺省 shared）+ 可选 `SpaceTaggedTool` 接口（仿 PersistWriteTool 模式，tool.go:65-81）。
- 归类（38 工具）：work 28（含 S0.6 edit 系/grep/bash/write_file/read_file/memory_search 等）、play 1（image_gen）、shared 9（ask/complete_step/todo_write 等）；MCP 默认 shared。
- 过滤在**装配期**：`addBuiltins(reg, …, space)`（boot.go:536-569）按空间跳过非 shared/本空间工具；ExtraTools/startPlugins（MCP）同 spec 层过滤（**MCP 热插拔绕过构建期过滤 → 必须在 spec 层滤**）；桌面端全量注册（gaea_handler.go:59-60）改为按当前空间过滤（重建时生效，与 GaeaSpaceActivate 语义一致）。
- HideUnlessOnly 与过滤后注册表交互 = 安全 no-op（tool.go:194-210）。

## 3. S1.4 任务/资源按空间分账
- `tasks.Options` 加 `PerSpace map[string]int`（每空间并发）与 `Priority map[string]int`（按 kind，默认 0）；`pickNext` 改为在「空间 sem 有余量的 queued」中按 优先级+最早 选择（防饥饿=完成时 re-signal 既有 runNext :691-693）；`HasActiveInSpace(kind, space)`。
- 重启续跑零改动（space_id 在行上保留）；现有 kind（price_fetch/file_index）显式 work；cron 后台提交点**无会话空间必须显式 work**（gaea_price_sources.go:57-113）。
- `GaeaTaskList` 改 variadic space（对齐 GaeaUnifiedSearch 先例：bindings_office.go:136 + gen_bindings main.go:304 特例），绑定面方法数不变。

## 4. S1.5-A 权限策略按空间装配
- `[space_profiles.<space>.permissions]`（mode/hard_ask/approval_timeout_secs/allow…）合并进 `cfg.PermissionsForSpace(space)`；boot.go:182 换取值点（按控制器空间）。
- `hardAskTools` 参数化为 `Options.HardAskTools`（4 个挂点 controller.go:638,698,706,713）；`persist_allow` 按空间分段回写（AddPermissionRuleForSpace）。
- play 默认 mode=allow + hard_ask=[]（不弹审批卡）——**产品确认**：play 清空 hardAsk 后 remember 等不再确认（记忆写 play 域无闸，与 S1.2 play 记忆隔离配套，可接受）。

## 5. S1.5-B play 内容护栏（依赖装配族配置键，最后做）
- `[space_profiles.play.guardrails]`（temperature_max/max_output_tokens/image_safe_mode/persona_lock）；落点=**5 个直连生成点参数钳制**（whisper 生成/章节/支线/角色卡/生图 handler）；**护栏不走 permission 引擎**（这些点本来没有闸）。

## 6. Step 拆解与回退
- S1.3-A（模型 profile）｜S1.3-B（工具标签+装配）｜S1.4（任务分账，可并行）｜S1.5-A（策略装配）｜S1.5-B（play 护栏，最后）
- 回退机制：**配置缺省=现状逐字节回退**（space_profiles 缺省时空值→旧行为；标签表缺省 shared→全注册；PerSpace 缺省→全局 sem）。

## 7. 风险
双平面错配（桌面 play 必须接 bridge feature 而非 Seam 直配）；MCP 热插拔绕过构建期过滤（spec 层滤）；cron 无会话空间必须显式 work；HasActive kind 全局去重挡跨空间（改 HasActiveInSpace）；space.mode=off 三处恒等现状；play 清空 hardAsk 后 remember 不再确认（产品确认）。
