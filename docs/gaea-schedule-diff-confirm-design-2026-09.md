# gaea 进度计划「对话式调整 diff 确认闭环」设计（diff 确认卡）

> 状态：**✅ 已收官**（v4.146~v4.149 四刀全部落地，2026-09-08）——刀A 对话改计划回滚闭环（v4.146.0，变更 tab 纳入 schedule_apply + 回执卡「回滚本次」）→ 刀B 结构化 diff 确认卡（v4.147.0，diffProjects + ScheduleDiffCard）→ 刀C project 通道引擎强制逐条确认（v4.148.0）→ 刀D ops 通道投影模拟器 + Go/TS 对拍收官（v4.149.0）。发布口径见 releases/v4.146.0.md ~ v4.149.0.md 与 .gaea/AGENTS.md 速览。
> 承接：拍板池最老欠账「对话式调整 diff 确认闭环」「整计划生成三确认」（docs/archive/market-research-2026-09-06.md §三 P1，v4.114.0 欠账清单原文：「对话式调整 diff 确认卡（改动清单+总工期对比预览→确认后 apply——现 apply 直接落盘+证据卡可回滚，确认卡为增强）」）。
> 纪律对齐：AI 产建议、引擎裁决；「成功≠正确」回执核对；零新绑定偏好；TS 块注释禁 `*/`。

---

## 0. 结论速览

- **推荐方案：混合分级确认**——
  - **project 整量通道**（整计划生成/重排/替换）：**前置强制确认**（复用 control 既有 hardAsk 机制做「通道化」：任何权限级别逐条弹卡、不记会话放行），卡体升级为 **schedule diff 确认卡**（改动清单 + 总工期/关键数/总成本 对比）；
  - **ops 增量通道**（对话式小调整）：ask 级别下**现状本就逐次弹审批卡**（读码核实，见 §1.2）——把这张卡的卡体升级为同一张 diff 确认卡即可，保留 allow_session 会话放行（首次确认后顺滑）；auto/yolo 级别不弹；
  - **后置闭环补全**：apply 回执卡上补「回滚本次」入口 + 变更 tab 纳入 schedule_apply（现状读码核实：**schedule_apply 的证据卡不进变更 tab，板块侧没有任何回滚入口**，§1.5）。
- **零绑定可行已核实**：审批挂起期间前端已持有完整 args（ToolDispatch 事件先于闸门发出，§1.3），before 态经既有 GaeaScheduleLoad / 板块 store 可得，TS 引擎镜像（computeCpm/computeCosts/computeBaselineDrift/checkDeadline）全齐——diff 计算可全程在前端纯函数层完成。目标绑定面 **0**（维持 588）。
- **分四刀**（每刀独立可发布可 revert）：A 后置回滚入口 → B diff 确认卡卡体（前端）→ C project 通道引擎强制（Go 小改+技能锚点）→ D ops 通道 simulateOps 镜像预览。

---

## 1. 现状与缺口（读码核实）

### 1.1 schedule_apply 双通道与回执（internal/gaea/tool/builtin/schedule_tools.go）

- **双通道二选一**：project=完整计划 JSON（整量）；ops=增量操作数组，12 种 op（upsert_task / patch_task / remove_task / set_links / set_meta / auto_chain / set_baseline / clear_baseline / upsert_resource / patch_resource / remove_resource / set_assignments，internal/schedule/ops.go，v4.124 刀2 刚扩资源操作）。ApplyOps 单条失败整批拒绝（fail-closed，无部分应用）。
- **写后动作序**：读现状（BeforeSummary：「『X』N 项工作，总工期 M 天」或「（新建计划）」）→ 落盘（Save 校验+CPM fail-closed）→ StageBaseline 整文件快照（.gaea/work/rollback）→ RecordChange 证据卡（Before/After 摘要 + BaselinePath）→ 回执（duration / critical / taskCount / totalCost，ops 通道另带 taskCosts；有基线带 baselineDrift；有目标竣工带 deadlineCheck）。
- **apply 回执强制带总工期/关键数/总成本**（v4.124 口径），工具描述强制模型核对（「成功≠正确」）。

### 1.2 「确认」环节现状：闸门已存在，卡里没有 diff

- **默认权限级别就是 ask**（internal/gaea/control/controller.go，permLevel 初始化 "ask"）。schedule_apply 为非只读工具，ask 级别下**每次调用在执行前就必经 gateApprover → ApprovalRequest 事件 → 前端 ApprovalModal 五决策卡**（deny / allow_once / allow_session / persist_allow / abort；超时=拒绝；internal/gaea/control/controller_approval.go）。
- **但卡上只有 subject=args.path**（permission.Subject 的 subjectKeys 命中 "path"）——用户看到的是「schedule_apply 当前计划」一行字 + 参数原文（折叠区），**没有任何改动内容摘要**。这是缺口的核心：有确认之形、无审阅之实。
- **allow_session / persist_allow 会话记忆**：key=`schedule_apply\x00<path>`，点一次「本会话内允许」后该会话所有对同路径的 apply 全部静默放行——对整计划替换这种毁伤半径最大的操作，现行语义偏松。
- **AlwaysAsk 硬确认机制已存在**（internal/gaea/permission/permission.go Gate.AlwaysAsk + controller hardAskSet，S1.5-A 按空间策略参数化）：无视权限级别/放行规则/会话记忆逐条确认；前端对 hardAsk 工具只渲染 拒绝/允许一次/终止 三钮（ApprovalModal isHardAsk）；非交互（headless）保持自治放行。cost_save/remember/knowledge_add 等在用。**本项目不需要发明新机制，只需要把 schedule_apply 的 project 通道接上去**。
- **拒绝语义已定式**：闸门拒绝回传给模型的固定文案为 "the user declined this tool call — do not retry it; ask how they would like to proceed or choose another approach."（permission.go Gate.Check Ask/AlwaysAsk 分支）。被拒的 apply **未执行、未落盘、无证据卡、无快照**——天然干净。
- **注意**：V10.13 循环守卫（repeatSuccessSignature）只覆盖文件写工具与 bash，**不含 schedule_apply**；风暴熔断（applyStormBreaker）显式排除 blocked 结果——防「拒绝后原样重发」目前只靠拒绝文案与技能纪律，缺口见 §5。

### 1.3 弹卡期间前端已持有完整 args（零绑定可行性的关键）

事件顺序（internal/gaea/agent/batch_executor.go）：**ToolDispatch 事件（带完整 args）先发出** → executeOne → dispatcher.Check → 闸门弹卡阻塞。因此审批卡挂起期间，Transcript 里工具卡（status=running）已携带 schedule_apply 的完整参数（project 整量 JSON 或 ops 数组），前端可即时计算 diff。且 schedule_apply 缺省冲突键为空串=恒独占串行批（partitionToolCalls），弹卡期间不存在并发工具同写一份计划文件。

### 1.4 TS 侧引擎镜像已齐，ops 模拟器是唯一缺口

- 前端 schedule 域已有与 Go 同口径的纯函数镜像：computeCpm（cpm.ts）、computeCosts（cost.ts）、computeBaselineDrift（baseline.ts）、checkDeadline（deadline.ts）、normalizeCalendar（calendar.ts）；数据模型 SchedProject 与 Go schedule.Project 同构（types.ts）。
- **project 通道 diff = diffProjects(文件现状, args.project)**：两份 project JSON 的纯 TS 比较，立即可做，零风险。
- **ops 通道投影 = simulateOps(文件现状, ops)**：Go ApplyOps 的 TS 镜像，目前不存在（唯一缺口）；含指针三态/级联删除/重复校验等语义，有漂移风险（§5 对策）。
- before 态来源：GaeaScheduleLoad（既有绑定，internal/app/gaea_schedule_file.go，只服务缺省路径「进度计划/当前计划.gsched.json」）或已水合的 schedule store；板块 dirty（防抖 800ms 未落盘）时文件与板块内存态可能短暂不一致。

### 1.5 回滚链现状：底盘齐全，入口没通到计划文件

- 证据卡（internal/gaea/evidence/journal.go ChangeRecord）→ Journal JSONL 按会话落盘 → GaeaRollbackRecord（internal/app/gaea_verify.go）按 BaselinePath 恢复，带「已被手工修改」守卫 + 恢复前再快照 + 撤销恢复（v4.32 线A）。
- 前端入口只有「变更」tab（ChangesPanel.tsx：GaeaJournalList → 按 target 匹配 → RollbackRecord 按钮）。**但 changes.ts 的 WRITE_TOOL_NAMES 白名单不含 schedule_apply**，且 buildChangeCalls 只聚合该白名单——schedule_apply 的证据卡在 Journal 里存在、在变更 tab 里不可见，**进度计划侧用户没有任何可点的回滚入口**（只能走办公文件轨道兜底）。这是后置闭环的实际缺口。
- 工具卡呈现（ToolCard.tsx + lib/tools.ts summarize）：apply=「总工期 X 天 · 关键 Y 项 · N 处修改」；板块左栏 ChatPane 与办公主聊天共享同一 store/Transcript/ApprovalModal（ChatPane.tsx），审批挂起时 Composer 自动禁用——确认卡两宿主天然同源。

### 1.6 compact 通道与锚点锁

- compact.go：schedule_get/apply/analyze 三条 compactDesc + compactSchema，工具 Description 改动须同步（v4.124 刀2 先例）。
- schedule-edit 内联技能（internal/gaea/skill/builtins.go）锚点锁 31 条（schedule_skill_test.go，v4.124 口径）——技能 Body 增删分节必须同步更新锚点测试。

### 1.7 上游参照（只取口径）

- 调研四步安全口径（docs/archive/research-2026-09-06/参考-AI改计划的确认与回滚设计模式.md）：preview the diff → explicit confirmation gate → easy revert → familiar metaphors；PM 域是安全空白，gaea 可自定义标准。
- Ingantt 三确认：「as-is / tweak / from scratch」——整体接受 / 微调 / 从零手编，三态并列的生成确认文案。

---

## 2. 设计提案

### 2.1 裁定一：确认时机 = 混合分级（推荐）

| 通道 | 时机 | 机制 | 会话记忆 | 理由 |
|---|---|---|---|---|
| **project 整量**（整计划生成/重排/替换） | **前置强制** | hardAsk 机制通道化（§2.5 刀C）：任何权限级别逐条弹卡 | **禁止**（三钮，无 allow_session） | 毁伤半径最大（整量覆盖任务/搭接/资源/分配）；且发生在「整计划生成」场景，用户预期就是先看后批（Ingantt 三确认）；现有 allow_session 记忆对整量替换语义过松 |
| **ops 增量**（对话式调整） | **前置（ask 级）** | 现状闸门不动的现状卡升级为 diff 确认卡 | 保留 allow_session（首次确认后顺滑） | 拍板池原文即「NL 改计划 → diff 摘要卡片 → 人工确认后 apply」；ask 级别下卡本来就弹（§1.2），delta 只是卡体内容；字段级小改逐次强制会在多轮调整中制造审批疲劳（斑马「关不掉的行内 AI」反面教训的镜像） |
| **后置闭环**（全通道兜底） | 事后 | 回执核对纪律 + 证据卡 + **回滚入口进卡/进变更 tab**（§2.4） | — | 「easy revert」是确认闭环的第三步；后置兜底让 ops 在 auto/yolo 级别下仍有安全网 |

落选方案如实记录：
- **纯前置**（所有 apply 一律强制逐卡）：多轮对话式调整（「压 5 天」「再压 3 天」）每句都弹卡，摩擦最大；且 auto/yolo 用户的明确意愿被覆盖。若拍板要更严，ops 通道加入 hardAsk 集合只是一行参数（空间策略 HardAskTools），保留为拍板项 3。
- **纯后置**（现状证据卡强化了事）：不满足拍板池「人工确认后 apply」的原文口径；「出问题靠 Journal 回滚」正是本欠账要补的缺口而非答案。

### 2.2 裁定二：diff 粒度与渲染 = schedule 专用纯函数 + 专用卡体（不复用 ChangesDiff 本体）

- **新纯函数层 `frontend/src/schedule/applyDiff.ts`**（零绑定依赖，与 gschedSummary.ts 同层同范式）：
  - `diffProjects(before: SchedProject, after: SchedProject): ScheduleApplyDiff`——project 通道用，纯比较无模拟：
    - 任务行级：added / removed / modified（按 id 键控；modified 展开字段级 from→to：name / duration / progress / level / isMilestone / mode / manualStart / fixedCost）；
    - 搭接级：按 to 分组比对该任务入边集合（added / removed / type·lag 变化——与 set_links「整体替换入边」语义对齐）；
    - 资源与分配：资源 upsert/删除（标注级联分配数）、按任务的分配集增删改；
    - meta：工程名 / 开工日期 / 日历 / 目标竣工；
    - 汇总行（卡头醒目）：总工期 X→Y 天（computeCpm 双方）、关键 N→M 项、总成本 A→B 元（computeCosts）；有基线附漂移预览（computeBaselineDrift）、有 deadline 附倒排校核预览（checkDeadline）。CPM 循环等校验失败如实标注「引擎将拒绝（CPM 不过）」——预览层复刻 fail-closed 口径，提示模型这单会被拒。
  - `simulateOps(before: SchedProject, ops: Op[]): { after?: SchedProject; error?: string }`——刀D 交付的 Go ApplyOps TS 镜像（12 种 op 逐条对齐，单测对拍，§5）；模拟失败诚实降级为「只列 ops 意图清单」。
  - 输出结构直接面向渲染（分组行/字段中文名/单位标注 元/工日 等），不强行塞进 ChangesDiff 的 hunks/DiffRow。
- **渲染：新 `ScheduleDiffCard`（schedule 域组件），作为 ApprovalModal 的 schedule_apply 变体卡体**（`approval.tool === "schedule_apply"` 时替换通用卡体；通道/决策/快捷键/超时/两宿主挂载全部复用 ApprovalModal 既有件）。视觉复用 changesdiff-tok 的 --add/--del 令牌与「行数上限截断」口径（建议 60 行上限 + 折叠），但**不复用 ChangesDiff 本体**：那是文本行 LCS 范式，表达不了 id 键控的字段 from→to；planDiff.ts 对 write_file 的先例是「诚实降级为内容预览」，本设计同理——能算结构化 diff 就结构化，不能就降级为 ops/project 意图清单并说明原因，绝不伪造 before 值。
- **数据来源与降级**：before = GaeaScheduleLoad 读文件实况（与 apply 的实际作用对象一致）；args.path 非缺省路径时 loadScheduleFile 拿不到（绑定只服务缺省路径）→ 降级为意图清单（ops 本身携带完整意图：patch_task 带目标值），卡上如实标注「非当前计划，无现状对比」。

### 2.3 裁定三：与既有件的关系 = 证据卡保留、回滚入口进卡

- **证据卡保留，不合并不替代**：确认卡=事前预览（投影），证据卡=事后凭据（实况+快照）。职责不同且互补；「成功≠正确」核对**永远以回执为准**——预览数字是前端投影，落盘后回执的 duration/totalCost 才是实况（§5 漂移风险的对冲条款）。
- **Journal 回滚入口进卡（刀A）**：
  - 变更 tab 纳入 schedule_apply（changes.ts 白名单扩一个并行集合或直接进 WRITE_TOOL_NAMES；extractChangedPaths 已能从 args.path 提取路径，无结构改动）；
  - apply 回执工具卡上补「回滚本次」按钮（按该次调用的证据卡 id 调既有 RollbackRecord 绑定，ChangesPanel doRollback 同款语义）。这一刀与确认卡无关、独立可发布，先把「easy revert」补齐。
- **ApprovalModal 复用通道而非卡体**：事件（ApprovalRequest）、Approve(ID, decision) 应答、超时=拒绝、快捷键、hardAsk 三钮形态、ChatPane/办公双宿主挂载全部照旧；新增的只有 schedule_apply 分支的卡体渲染与 i18n 键（schedDiff.*，en/zh 双份）。

### 2.4 裁定四：失败路径（用户拒绝后的行为语义）

| 决策 | 引擎行为（全部既有语义，零改动） | agent 收到什么 |
|---|---|---|
| deny（拒绝） | apply 未执行、未落盘、无证据卡、无快照；板块不刷新（notifyScheduleFileChanged 仅 done 触发） | 工具结果="the user declined this tool call — do not retry it; ask how they would like to proceed or choose another approach."（blocked，回合继续） |
| abort（终止） | 取消当前回合（controller.Cancel） | 回合终止 |
| timeout（超时无人响应） | 按拒绝处理 + Notice 告知哪些步骤被超时拒绝 | 同 deny；**无人值守的整计划生成会被拒**（发布说明须提示） |
| 批准（allow_once） | 闸门放行 → apply 正常执行 → 落盘+快照+证据卡+回执 | 正常回执，照常核对「成功≠正确」 |

- 技能纪律补条款（刀C 随 schedule-edit Body 更新）：确认被拒 ≠ 失败终局——按用户在对话里给出的意见**修改 ops/project 参数后重发是新一次确认**（合法）；原样重发同参数调用既无意义也会被拒绝文案点名的 retry 禁令约束；连续被拒两次应停止下发、向用户要明确口径。
- 「三确认」文案映射（拍板项 5）：批准写入（as-is）/ 拒绝并说明改法（tweak）/ 终止·我来手编（from-scratch 映射为拒绝页提示文案，**不加第四颗按钮**——审批决策枚举是跨工具公共件，不为单工具扩枚举）。

### 2.5 裁定五：compact 通道、锚点锁与绑定面预估

- **绑定面：目标 0，预估 0。** 全部改动落在：前端纯函数/组件/i18n（applyDiff.ts、ScheduleDiffCard、ApprovalModal 变体、changes.ts 集合、tools.ts summarize 可选加总成本）+ Go 侧 control 包闸门参数化（approvalSubjectFor 增 schedule_apply 分支产出可读 subject，如「整计划替换：N 项工作 → M 项，总工期 X→Y」+ 通道化 alwaysPrompt 判定）+ 技能与 compact 文案。不新增 App 导出方法，不动 event.Approval 结构，check-bindings-drift 预期 PASS@588 不变。
- **compact 同步**：schedule_apply 的 Description 增确认机制说明（「project 通道须经用户在确认卡批准后才落盘；被拒即未写入」）→ compact.go compactDesc["schedule_apply"] 同步改写（≤一行口径）。
- **锚点锁**：schedule-edit Body 增「确认与回滚」分节（三确认口径/被拒处置/回滚入口/预览以回执为准）→ schedule_skill_test.go 锚点锁 31→N 随刀更新（v4.124 刀2「23→31」同款动作）。

---

## 3. 分刀方案（每刀独立可发布、可 revert）

| 刀 | 内容 | 改动面 | 绑定面 | 测试面 |
|---|---|---|---|---|
| **刀A** 后置闭环补全 | 变更 tab 纳入 schedule_apply（路径聚合+diff 缺省降级说明）+ apply 回执工具卡「回滚本次」按钮 + summarize 增总成本（可选） | 纯前端：changes.ts / ChangesPanel.tsx / ToolCard 一侧 tools.ts | **0**（RollbackRecord/GaeaJournalList 绑定已有） | vitest：changes/ChangesPanel 扩展用例；变更 tab 对 .gsched.json 路径的聚合断言 |
| **刀B** diff 确认卡（project 通道） | applyDiff.ts 的 diffProjects + ScheduleDiffCard + ApprovalModal schedule_apply 变体（含 hardAsk 形态三钮预置）+ i18n schedDiff.* | 纯前端 + 零 Go | **0** | vitest：diffProjects 纯函数全分支（增/删/改/搭接/资源/meta/CPM 循环降级）+ 卡体渲染 + 快捷键；tsc/eslint 全量 |
| **刀C** project 通道引擎强制 + 纪律同步 | Go：controller_approval.go approvalSubjectFor 增 schedule_apply project 分支（alwaysPrompt + 可读 subject）；Description/compactDesc 同步；schedule-edit Body 增「确认与回滚」分节 | Go（control 包内，无新绑定）+ 技能文案 | **0** | Go：controller 审批分支用例（project 弹/ops 不弹/各权限级别/超时）；schedule_skill_test 锚点锁 31→N 更新；drift PASS@588 |
| **刀D** ops 通道投影（拍板后） | simulateOps TS 镜像（12 op 对齐 Go ApplyOps）+ ops 卡在 ask 级别下升级为带投影 diff 卡；模拟失败降级意图清单 | 纯前端 | **0** | vitest：**Go/TS 对拍用例**（同 ops 序列双跑比对 after——口径钉死，防漂移主阵地）+ 降级路径 |

刀序建议 A→B→C→D：A 最小先把「可回滚」补齐（即使确认卡不做也独立成立）；B 交付可见价值（ask 级别立即有 diff 卡）；C 把「整计划三确认」做成引擎强制；D 视拍板与 A/B 走查反馈决定。每刀独立 revert：A/B/D 纯前端 revert 无残留；C 的 Go 闸门分支收敛在一个函数内，revert 后回到现状（ask 级普卡）语义。

---

## 4. 拍板问题清单（每点带推荐）

| # | 问题 | 推荐 | 备选 |
|---|---|---|---|
| 1 | 确认时机总方案 | **混合分级**（§2.1：project 前置强制 / ops ask 级前置卡 / 后置回滚兜底） | 全前置强制；纯后置强化 |
| 2 | project 通道是否禁止会话记忆 | **禁止**（hardAsk 三钮，任何级别逐条确认） | 保留 allow_session（首次放行后整量替换静默——不建议） |
| 3 | ops 通道 auto/yolo 级别是否豁免弹卡 | **豁免**（尊重用户自选的权限级别；后置回执+回滚兜底） | ops 也进 hardAsk 集合（空间策略 HardAskTools 参数，一行改动） |
| 4 | before 态口径 | **文件实况**（GaeaScheduleLoad，与 apply 实际作用对象一致）；板块 dirty 时卡上提示「板块有未保存修改，写入将以文件与本次参数为准」 | 板块内存态为准（会与实况漂移，不建议） |
| 5 | 三确认文案映射 | **批准写入 / 拒绝并说明改法 / 终止**；from-scratch 作拒绝页提示文案不加按钮 | 新增第四决策枚举（动公共件，不建议） |
| 6 | 预览与回执不一致时的口径 | **以回执为准**并向用户点破差异（「成功≠正确」既有纪律延伸；卡上预览数字标注「预览」） | 以预览为准（预览层无落盘权威，不成立） |
| 7 | simulateOps 漂移控制 | **Go/TS 对拍单测钉死 + 诚实降级 + 落盘权威永远在 Go**；不做 Go 侧算 diff 下发事件（改 event.Approval 结构，复杂度高） | Go 渲染 diff 随 ApprovalRequest 下发（动公共事件件，不建议） |
| 8 | 回滚入口粒度 | **按证据卡逐次回滚**（复用 GaeaRollbackRecord 与「已被手工修改」守卫，ChangesPanel 同语义）；不做「回滚到任意历史版本」选择器 | 任意快照选择器（Journal 无版本树 UI，另案） |

---

## 5. 风险与欠账

**风险与对策**

| 风险 | 等级 | 对策 |
|---|---|---|
| **TS 镜像与 Go ApplyOps 语义漂移 → 「批的是 A，落的是 B」**（12 种 op、指针三态、级联删除、重复/悬空校验） | 高（本设计最大风险；也是刀D 排最后的理由） | Go/TS 对拍单测；模拟失败诚实降级；卡上数字标注「预览」、落盘后以回执核对点破；project 通道 diffProjects 纯比较无此风险（刀B 先行） |
| 审批疲劳→盲批（确认卡沦为「无脑点 2」） | 中 | 卡头汇总行醒目（总工期/成本变化超阈值红色/绿色高亮）；改动行数上限+折叠；ops 通道保留会话放行出口 |
| 审批超时=拒绝 → 无人值守整计划生成中断 | 中 | 既有 C4 语义，不为本项目改；发布说明与技能条款明示；headless 非交互保持自治放行（AlwaysAsk 语义） |
| 审批挂起期间用户手改板块→防抖落盘改变 before | 低 | v1 不做 mtime 复核（记欠账）；批准后 apply 以写盘时刻实况为准，回执核对兜底 |
| 拒绝后原样重发（循环守卫/风暴熔断均不覆盖 schedule_apply 的 blocked，§1.2） | 低 | 拒绝文案已含 do-not-retry 指令 + 刀C 技能条款「连续被拒两次停止下发」；如实测仍出现，再评估把 schedule_apply 纳入 repeatSuccessSignature（另刀） |
| i18n 键遗漏 | 低 | schedDiff.* en/zh 双份随刀B 交付，ToolCard.i18n 测试范式复用 |

**欠账（本设计不解决、登记在案）**

- 甘特图上的 diff 高亮 + 情景对比（before/after 双排程叠影）——调研稿指出的「gaea 自定义标准」候选，属板块可视化另案。
- 非缺省路径计划文件的 before 态读取（需新绑定或通用文件读取，违背零绑定目标，观察需求再说）。
- 回执工具卡内嵌事后 diff（done 后用同纯函数回放 before/after——依赖快照路径进前端，另案）。
- mtime 复核（审批挂起期间文件变更检测）。
- 拍板池其余项（双工期口径/多工程管理/推荐逻辑关系/报告模板化）不在本文范围。

---

## 6. 明确不做

- **不新增任何 Wails 绑定方法**（目标 0；不引入 GetScheduleDiff / PreviewScheduleApply 一类 Go 绑定——前端自算已可行）。
- **不改 event.Approval / ApprovalRequest 事件结构**（diff 载荷不进事件；审批通道是跨工具公共件）。
- **不做逐 op / 逐字段的部分批准**——确认以一次 apply 调用为单位；要逐项裁决就让模型拆多次 ops 下发（模型侧纪律）。
- **不做卡内编辑/改参数交互**——拒绝后由模型按用户意见重出新参数，卡只读。
- **不做 ops 通道默认 AlwaysAsk**（拍板项 3 保留参数位，默认不加）。
- **不引入第三方 diff 库**；复用既有视觉令牌与自研口径。
- **不让确认卡替代证据卡/回执**——预览层永远不是落盘权威。

---

## 7. 参考

- 读码：internal/gaea/tool/builtin/schedule_tools.go、internal/schedule/ops.go、internal/gaea/evidence/{evidence,journal}.go、internal/gaea/control/controller_approval.go（+controller.go hardAskSet/SetPermLevel）、internal/gaea/permission/permission.go（Gate.AlwaysAsk/Check/Subject）、internal/gaea/agent/{batch_executor,execute_one,tool_dispatch}.go、internal/gaea/tool/builtin/compact.go、internal/gaea/skill/builtins.go（schedule-edit）、internal/app/gaea_schedule_file.go、internal/app/gaea_verify.go（GaeaRollbackRecord）
- 前端：frontend/src/gaea/components/{ApprovalModal,ChangesDiff,ToolCard,ChangesPanel,ScheduleFileCard}.tsx、frontend/src/gaea/lib/{planDiff,changes,tools,store,types}.ts、frontend/src/schedule/{ChatPane,store,types,api,cpm,cost,baseline,deadline}.ts
- 发布口径：releases/v4.114.0.md（刀5 欠账原文/工具卡摘要）、releases/v4.124.0.md（回执与总成本口径/锚点锁 31/绑定面 588）
- 调研：docs/archive/market-research-2026-09-06.md（拍板池 P1 原文）、docs/archive/research-2026-09-06/参考-AI改计划的确认与回滚设计模式.md（四步安全口径）、docs/archive/research-2026-09-06/Ingantt.md（as-is/tweak/from scratch）
- 格式基准：docs/gaea-office-mindmap-base-design-2026-09.md
