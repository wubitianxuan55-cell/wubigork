# 办公板块运行模式（标准/造价/方案）设计论证
2026-11-21 · 只论证未动码 · 侦察基准 2e7a8ac3

## 一、需求口径

模仿 DSH 档位制，给办公板块加「运行模式」三档：
- **标准**：通用工具（=现状全量，默认档）
- **造价**：带工程造价专用工具和技能
- **方案**：带编制方案专用工具和技能

**口径歧义（需拍板）**：「造价模式带上造价专用」= 叠加（通用+造价）还是收窄（只有造价相关）？
- **推荐叠加不减**：模式选错时工具不全=死路；DSH 收窄制依赖预设面严格对齐，gaea 工具是共享 builtin 池，收窄会砍掉 bash/read_file 等基础面，代价不成比例。
- 折中强化：专用模式下**技能索引把专用技能置顶+L1 点名模式定位**，让模型聚焦但不失能。
- 将来实测若专用模式被无关工具干扰（schema 预算/选择漂移），再评估收窄白名单（基建已支持，见 §三.2）。

## 二、复用基建（零新造轮子）

| 面 | 现成缝 | 落点 |
|---|---|---|
| 配置持久化+重建 | `gaeaApplyCfg`（mutate→Save→Rebuild→图文跟随） | gaea_ui_extra.go:174-197 |
| 工具过滤 | `boot.Options.ToolFilter`（办公现 nil） | boot.go:143-148；参照 wenxue_tool_policy.go:63-77 |
| 技能门 | `boot.Options.SkillFilter/SkillExtras`（办公现双 nil，五面同源：索引/run_skill/read_skill/slash） | boot.go:149-155,643-661；参照 wenxue_skills.go:23-39 |
| UI 形态 | 「参数」弹出层第三组 chip（权限/思考同款） | ComposerToolbar.tsx:181-229 |
| 状态 hook | `useModeManager` 同款 | useModeManager.ts:9-46 |
| 回合中拒绝 | `SetXxxReported` 报错透传 | gaea_ui_extra.go:439-449 |

## 三、方案要点

1. **配置键**：`[office] run_mode = "standard"|"cost"|"proposal"`（默认 standard）+ 绑定 `GaeaSetOfficeRunMode` → gaeaApplyCfg（自动 Save+重建+wenxueReloadIfInit）。
2. **工具面（ToolFilter，按模式）**：
   - standard：nil（=现状，零风险）
   - cost：放行全量 **+ 强调** cost_* 四件（cost_search/cost_save/cost_compose/cost_indicators——registrymap.go:93-96 已 work 域 builtin，无需新写工具）+schedule_analyze
   - proposal：全量 + 强调 office_generate/docx_apply/docx_accept/pptx_apply/xlsx_apply/format_convert/office_batch
   - 注意：叠加制下 ToolFilter 恒 true 即可，模式差异落在**技能门+提示词**；收窄制才需要真过滤谓词（基建保留）。
3. **技能门（SkillFilter allowlist，wenxue 先例）**：
   - cost：`cost-compose, schedule-dcma, engineering-figures, cad-takeoff, xlsx`
   - proposal：`docx, pptx, pptx-template, pdf, xlsx, doc-assemble, office-gongwen, gongwen-9704, format-convert`
   - SKILL.md frontmatter 无 mode 键（skill.go:339-368 只解析 name/description/allowed-tools/runas/context/agent/model 五键）→ app 层 allowlist 零迁移面；将来要技能自带元数据再扩 frontmatter+Skill 结构。
   - standard：nil（全量）。
4. **提示词（L1，缓存友好）**：buildSystemPrompt（sysprompt.go:51）在冻结点（:194）前按模式拼一句：「当前运行模式：造价——聚焦工程造价编制，优先使用造价工具与技能……」。模式切换走重建，L1 前缀在模式恒定期间逐字节稳定（与今日缓存线结论同向：模式=低频档位，不做会话内高频切换）。
5. **UI**：「参数」弹出层第三组 chip（标准/造价/方案）+ 触发钮摘要带当前模式；绑定/mappings/useModeManager 三件套同权限档位先例。
6. **回合中切换**：**推荐拒绝**（SetOfficeRunModeReported 回合中报错透传）——重建即换 controller、进行中会话不续接；免重建实时通道（/plan 式 AppendUserMessage 注入）工具面要等下次重建才生效，半态更混乱。文案指路：等回合结束再切。

## 四、代价与风险

- **重建=会话断+缓存冷**：每次切模式一次冷启动。模式定位为低频档位（跨任务换档，不是会话内开关）——可接受，且频率天然低。
- **allowlist 漂移风险**：技能/工具清单变动时白名单漏配 → 需守卫测试钉死（模仿 spacetags/registrymap 三守卫纪律：模式×技能allowlist×实际索引一致性断言）。
- **wenxue 不受模式影响**：run_mode 只进办公引擎；wenxueReloadIfInit 跟随重载是既有通道行为，图文引擎的 SkillFilter/ToolFilter 不动。
- **ai_feedback/md_validate 等新工具**：shared/read 类不进 allowlist 也不受影响（叠加制下恒在）。

## 五、验收面（实施时的测试钉）

1. 三模式×工具装配：cost/proposal 下 cost_*/office 交付族在 schema（.boot 装配断言）。
2. 三模式×技能索引：allowlist 技能进索引、其余不被 run_skill/slash 触达（五面同源收口断言）。
3. L1 快照：三模式各冻一版，模式说明句在场、前缀稳定（复用 sysprompt 冻结锚测试形态）。
4. 回合中切换拒绝+文案。
5. 前端：chip 三态切换持久化+重建通知；ComposerToolbar 摘要。

## 六、工作量

后端（配置键/绑定/SkillFilter/ToolFilter/L1 句/守卫测试）≈ 中等一批；
前端（绑定三件套+chip 组+hook）≈ 小批；合计约与今日「实测九刀」单批相当。
基建零新增——全部落在既有四条缝上。

## 七、与工具面板线的关系

「工具下拉按板块区分」（侦察已毕：office=CAD/进度计划现状；wenxue 共用 gaea ComposerToolbar、工具钮已渲染但 wenxue App 未接 onOpenTool，接上即亮）是**纯 UI 线**，与本模式线正交；若模式落地，工具下拉可再按模式派生（造价模式置顶 cost 面板项），属可选增量。

## 八、DSH 上游对照（2026-11-21 补：实查 dsh-web-latest 快照，修正初稿）

**DSH 实际形态**（dsh-client-ui-agent-preset + dsh-agent-preset + dsh-agent-preset-registry 三包实锤）：

1. **模式=Agent preset**：Cordis YAML 每档声明一份子插件组合（`config.id`+`plugins:[]`——工具/提示词/能力的完整清单），内置 standard/ptc/minimal/cordis（创造模式）+自定义；registry 登记默认值（`default: standard`）。
2. **应用点=会话边界**：「选择变更只影响之后的任务」「已有会话保留所选模式；新会话使用当前默认值」——切换=**暂存给下一个 Session**，从不断当前会话；KV Cache 语义：同 Agent 组合稳定。
3. **选择面**：设置页卡片分组（内置+自定义，默认值高亮）+ 新任务选择器 + 每卡「查看配置」（只读 YAML 展示声明的插件清单）；Host 拒绝切换 Toast 原因；已有会话的模式不在菜单仍诚实显示标签。
4. **创造模式**：专用档位（选它=空白 Session 绑 cordis preset+随包技能 agent-experience 等），专门用于创建/修改 preset（产出 bundle 声明新 preset 或按行 id 覆盖内置）。
5. **门面联动**：代码工作工具开关改变可用档位集合（关=standard/创造/自定义，开=+PTC/极简）；关闭时已保存的 PTC/极简默认值回退 standard，选择保持可恢复。

### 对初稿的修正（四点）

- **应用语义改 DSH 同构（最大修正）**：初稿「切换→gaeaApplyCfg 立即重建」会断当前会话，与 DSH「已有会话保留所选模式」相悖。改：**模式切换=默认值登记+暂存，新会话（或下次引擎重建后的新语境）生效；当前会话保持现有模式不变**。gaea 落点：cfg 键照旧，但 UI 文案明示「新会话生效」；controller/会话层记录本会话所用模式（诚实标签）。
- **模式声明收敛为组合表（对标 plugins:[]）**：每模式一份声明 `{工具强调面, 技能 allowlist, L1 句, 名称/描述}` 单表定义（不散在四处 if）——DSH 的 preset 本质就是「组合声明」，gaea 用 Go 表+守卫测试等价实现；将来扩展自定义模式（投标/审计档）只加表行。
- **创造模式对应物留位**：v1 只内置三档；组合表结构天然容纳「用户自建档」（未来=技能+提示词组合包，对应 DSH bundle/创造模式），不做即留扩展位。
- **「查看配置」对应物**：模式卡片附只读视图（该模式的工具面+技能清单渲染），DSH 同款诚实面；Host 拒绝切换→gaea 的回合中拒绝 Toast 同款。

### 维持不变的判断

- 技能门 allowlist（wenxue 先例）≈ DSH preset 的 plugins 声明——机制同构，只是 DSH 以插件为单位、gaea 以技能/工具缝为单位。
- L1 提示词位、叠加不减口径、守卫测试纪律——与 DSH KV Cache 语义（组合稳定）一致，维持。

## 九、缓存约束下的应用语义分叉（2026-11-21 补，用户点题：DSH 这样设计就是为了会话内缓存命中）

DSH 的会话边界应用不是 UX 偏好，是**缓存设计**：会话内模式恒定=前缀逐字节稳定=全程命中；
README 明写「KV Cache effect：同一 Agent 的组合保持稳定；选择变更只影响之后的任务」。

gaea 适配的难点：**工具面/技能门/L1 是引擎级，会话级只有一条现成缝**（锚点已核）：
- L2 会话合并：`MergeRuntimePrompt→MergeIntoSystem` 每会话首轮一次性并入首条 system 消息
  （controller.go:699→session.go:99）——**模式句走这里=天然会话级，零缓存损失**。
- 工具面：executor 构造于 boot（boot.go:894），activeSchemas 每回合恢复构造期基线——引擎级。
- 技能门：SkillFilter 是 boot 期闭包——引擎级；gaea 单活跃会话，闭包可动态读 controller
  当前会话模式（单值，非多会话竞态）。

**三条路**：
- **路 A（初稿残留）**：切换→立即 gaeaApplyCfg 重建。语义=DSH 反面（断当前会话语境+缓存冷）。
  弃。
- **路 B'**：切换→重建+ResumeFromDisk 续历史。会话续了但前缀全变=该会话缓存冷一次+模式中途
  漂移（DSH 明确避免）。仅当用户接受「切模式=当前会话缓存作废」时可用。
- **路 C（推荐，DSH 全同构）**：模式=会话属性，引擎恒模式中立——
  - C1（小）：模式句+模式技能索引走 L2 会话合并缝（现成）；技能门闭包动态读当前会话模式。
    切换=登记默认值，**零重建零缓存损失**；当前会话保持旧模式，新会话自动新模式。
  - C2（中，视口径）：工具 schema 会话基线挂钩（activeSchemas 按会话模式设定）。
    **叠加制下不需要**（工具面恒全量=字节稳定，模式差异全在提示词/技能门）；收窄制必需。

**结论强化**：叠加制 + 路 C1 = DSH 全同构（会话内恒定/边界切换/零缓存损失）且改造面最小
——「叠加 vs 收窄」的拍板现在直接决定要不要做 C2。缓存视角下叠加制优势进一步拉大。
