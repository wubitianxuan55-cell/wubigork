# Plan mode 审批门（dsh plan-mode 蒸馏）——规格

> 2026-09-26 立项。DSH 留池⑦（①子代理控制面等上游稳定，本项为实质头名）。
> 上游=`/c/AI/deepseek-harness/packages/plan/plan-mode`（index.ts 471 行+types+invariant）。
> 蒸馏=取道不取器。目标版本：v4.414.0。

## 上游机制五要点（取道）

1. **状态=日志事件折叠**（`plan/mode {active}`），resume/fork 恢复；最后一条生效。
2. **active 时每请求带政策段**（`plan:policy`，部署方所有）：研究/只读纪律+产出计划。
3. **exit_plan_mode 工具恒注册**（目录跨过渡稳定，进退只变提示段不变工具清单）；
   调用→计划 markdown（`#` 标题开头强制）→审批问询（intent=plan-review）：
   Approve=退出并执行 / Keep planning=带反馈继续 / 关闭=留在计划模式等用户说话。
4. **模式切换叙事**：switch 以 user 消息注入告知模型；mid-turn 选择挂起到下一
   accepted pre-step，turn 间立即提交。
5. **沙箱/审批政策独立**——计划模式只动提示+审批流，不做硬工具门（上游明示
   "do not read or write plan state"）。

## gaea 落地设计（不取器）

### 约束→两个关键偏离

- **系统前缀冻结**（V10.36 缓存纪律，MergeRuntimePrompt 合并后消息前缀永不改变）：
  mid-session 改 system 段必破缓存前缀（技能目录 digest 热重载同判已辨伪销号）。
  → **政策走 user 消息注入**（上游 switch 叙事同通道，但携带完整政策文本）：
  `/plan on` 注入「[plan mode] 研究只读纪律+exit_plan_mode 用法」user 消息；
  off 注入退出叙事。前缀零触碰；压缩卷走标记无碍（状态权威在内存，不折叠历史）。
- **事件日志面零改动**：session/log.go 的恢复折叠是最敏感面（golden/rewind/
  checkpoint 守卫族），且 legacy 格式（旧会话缺省）无事件日志可落。
  → **v1 状态=runner 内存态**（`AgentRunner.planMode bool` + 互斥），应用重启丢失
  模式（用户重发 /plan 即可，单会话场景可接受）；事件日志 kind=plan_mode+恢复折叠
  留二刀（观察池在册）。

### 刀面（绑定面 720 零变更，前端零改动）

1. **`internal/gaea/agent/plan_mode.go`**（新）：
   - `ExitPlanModeTool`：恒注册（boot.go reg.Add，与 request_permission 同段——
     会话级元工具 shared 语义，work/play 两空间可见，不经 spacetasks 装配过滤）。
   - Schema：`plan`（string 必填，须 `#` 标题开头）。
   - Execute：非 plan 态→错误「仅计划模式可用」；plan 态→`CallContext(ctx)` 取
     Asker（与 ask 同通道）→单问 header「计划审批」/「批准，退出计划模式并执行」
     /「继续计划（反馈带回）」Multi=false：
     - 批准→runner 状态 off+注入叙事 user 消息（「用户已批准计划：退出计划模式，
       从下一步开始执行」）→返回 `{approved:true}` 文本。
     - 继续计划→返回错误「用户选择继续计划；反馈：…」（同上游语义：反馈在
       tool result 里回模型）。
     - 无 Asker（headless）→诚实降级错误「无交互用户可审批；保持计划模式并停止，
       等待用户」（不仿 ask 的自选语义——执行闸必须人来开）。
   - Runner 侧：`SetPlanMode(bool)`+`PlanMode()`+`injectUserNotice(text)`（追加
     user 角色消息到 session.Messages，mu 内；下一 stream() 自然拾取，前缀不动）。
2. **`internal/gaea/control/controller_submit.go`**：斜杠动词 `/plan`：
   - `/plan on|off|status`（裸 /plan=status）；running 中拒绝（同 /compact 模式）；
     Notice 反馈（「计划模式已开启：模型将只研究并产出计划，exit_plan_mode 提交
     审批」/「已退出计划模式」/「当前：开/关」）。
   - on/off 调 executor.SetPlanMode+注入叙事；状态翻转即注入政策/退出消息。
3. **政策文本**（包级常量，中文，对齐办公系统提示风格）：研究只读+先出计划+
   exit_plan_mode 提交+批准前不动任何写工具。
4. **前端**：v1 零改动——审批 ride ask 泛型选项卡；工具调用卡泛型渲染计划文本
  （上游 presentCall 也是 generic 卡）。intent 专用卡留观察池。
5. **测试**：`plan_mode_test.go`——工具四态（inactive 错误/批准翻转+叙事注入/
   继续计划反馈/headless 降级）+`/plan` 动词三态+注入消息断言（role/前缀不动）
   +既有回归（注册表目录含新工具）。

### 明确不做（本刀）

- 事件日志持久化+恢复折叠（二刀，观察池）。
- mid-turn 挂起语义（gaea 斜杠在 turn 间执行，无 pre-step 折叠面）。
- plan-review 专用 intent 卡（ask 泛型卡够用，真实使用后再议）。
- 硬工具门（上游亦无；现有 permission 闸已独立把关写操作）。

## 风险

- 工具目录+1：boot 装配时恒注册→所有会话目录一致，无 mid-session 目录漂移；
  新目录=新会话前缀，无缓存语义破坏。
- 注入 user 消息在 turn 间执行，不与插话（steer）通道竞争。
- 子代理：FilterRegistry 不影响（exit_plan_mode 加入 SubagentMetaTools 排除面
  与否——**子代理不应见到审批工具**，注册后加入 SubagentMetaTools() 排除清单，
  防子代理代替父会话退出计划模式）。
