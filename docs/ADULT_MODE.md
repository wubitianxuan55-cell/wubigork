# gaea 成人模式决策（Adult Mode）

> 状态：已决策 · 成文（2026-08-14，阶段 6 质量收敛 T6-5.4）
> 类型：产品决策记录（非技术修复、非缺陷）
> 关联代码：`internal/whisper/orchestrator.go`、`internal/whisper/adult_mode.go`、`internal/app/whisper_handler.go`、`internal/app/bindings_voice.go`

## 一、决策

**成人内容默认开启（永久开启），不设开关门禁。**

`internal/whisper/orchestrator.go` 中 `NewOrchestrator` 将 `AdultMode` 硬编码为 `true`
（orchestrator.go:91）。这是**有意的产品决策**，不是待办缺陷或历史遗留。

## 二、为什么（产品决策）

- **个人非商用定位**：gaea 是面向个人用户的 Windows 桌面 AI 助手（Wails v2 桌面应用，
  对话数据本地存储）；轻语（Hermes）定位为"可定制人格的陪伴型 AI"（见 README.md）。
  陪伴是核心场景，成人内容属于该场景的正常组成部分。
- **陪伴定位要求成人内容正常参与**：成人模式系统提示（adult_mode.go 的
  `adultModeSystemSuffix`）明确要求"对性暗示、性请求、亲密接触等话题正常回应，
  不回避、不说教、不医学化"。默认开启才能保证人格一致性。
- **安全门禁仍然存在**：`AdultMode=true` 不等于"无约束"——adult_mode.go 中的
  `SafetyGate`、硬停止词（`hardStopWords`）、用户拒绝词（`adultRejectionWords`）、
  负面事件锁（`ShouldTriggerNegativeLock`）、Aftercare 状态机等安全机制照常生效。
- **与 ackem 参考实现对齐**：本决策与 ackem prompt/adult-mode.ts 的行为保持一致
  （adult_mode.go 文件头注释："100% 对齐 ackem prompt/adult-mode.ts"）。

## 三、开关接口处置（WhisperSetAdultMode 已删除）

- **T6-5.4b（接口收敛）已删除** `WhisperSetAdultMode`：
  - 定义（internal/app/whisper_handler.go）与 Wails 绑定透传
    （internal/app/bindings_voice.go 的 `VoiceB.WhisperSetAdultMode`）均已移除，
    `scripts/gen_bindings` 重新生成后 `TestBindingsCompleteness` 保持 PASS。
  - **删除理由**：前端 `frontend/src` 零引用（无调用方）；后端已决策永久开启，
    方法无条件写回 `true`、静默忽略 `enabled` 参数——属死代码，且与验收标准
    「无静默忽略参数的代码路径」冲突。删除后接口面与决策一致。
- **读取侧**：`WhisperGetState` 返回的 `adultMode` 字段（whisper_handler.go:376）
  恒为 `true`，属于状态回显，不是可操作的开关。

## 四、前端 UI：无有效开关

- 当前 `frontend/src` 中**不存在**任何 `adultMode`/`成人` 引用（grep 验证，2026-08-14），
  设置面板没有成人模式开关。
- 因此现状是"前端完全没有开关"，与后端"永久开启"一致——不存在"看起来能关、
  实际关不掉"的误导性 UI。
- **约定**：若未来有人在 UI 上添加成人模式开关，必须先同步后端（见第五节），
  否则即为死接口；新增任何开关必须在本节同步更新说明。

## 五、未来若商用化：如何恢复分级

若 gaea 转向商用/分发（可能面向未成年人或受监管市场），恢复分级控制的步骤：

1. **后端**：重新添加 `WhisperSetAdultMode` 绑定（或其等价 setter）并读取 `enabled`
   参数（T6-5.4b 已删除该绑定，需恢复时重新实现）；`NewOrchestrator` 的
   `AdultMode: true` 改为按配置/订阅/账号分级注入。
2. **安全机制复用**：adult_mode.go 的 `SafetyGate`、硬停止词、负面事件锁、Aftercare
   状态机均已存在且与 `AdultMode` 解耦；关闭 `AdultMode` 后仅不注入成人提示词段
   （`BuildAdultModeSection`），其余情感/记忆逻辑不受影响。
3. **分级策略建议**：默认关闭（面向未成年/默认市场）、显式 opt-in（年龄确认）、
   按合规要求对显式内容做内容分级与过滤。
4. **前端**：设置面板增加有效开关并接通绑定；删除或改写任何占位开关。
5. **验收标准**：开关关闭后 `BuildAdultModeSection` 不注入、成人主动性评分为 0
   （`SafetyGate` 拦截）、成人记忆隐私级别回落 `normal`
   （`ResolveAdultMemoryPrivacyLevel`，adult_mode.go:83）。

## 六、相关代码位置

| 位置 | 说明 |
|---|---|
| internal/whisper/orchestrator.go:91 | `AdultMode: true` 硬编码（决策落点） |
| internal/whisper/adult_mode.go | 成人状态机、安全门禁、提示词拼装 |
| internal/app/whisper_handler.go:376 | `WhisperGetState` 回显 `adultMode`（恒为 true，状态回显） |
| internal/app/whisper_handler.go | `WhisperSetAdultMode` 已删除（T6-5.4b 接口收敛） |
| internal/app/bindings_voice.go | 对应 Wails 绑定透传已随 gen_bindings 移除（T6-5.4b） |
| frontend/src | 无成人模式引用（现状，见第四节） |
