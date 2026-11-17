# 资产 41 · 首启与信任：步骤状态、取消后的去向、远程公告的边界、启动草稿恢复

> 证据根（只读）：`onboarding/{onboarding_screen,welcome,trust_directory,directory_trust,auth,bedrock}.rs`（1090/222/483/156/… 行）、
> `security_setup.rs`（168）、`permission_discovery.rs`（175）、`startup_{preflight,error,recovery,presentation,draft,orchestration,hooks_review}.rs`、
> `bottom_pane/user_verification.rs`（366）。
> 对照根（只读）：`frontend/src/components/chat/WelcomeScreen.tsx`（6.5KB）、`frontend/src/gaea/components/ApprovalModal.tsx`、
> `frontend/src/pages/SettingsPage.tsx`、`internal/gaea/permission/*`（权限模型）。
>
> 只出机制与落点；不改 gaea 源码。

---

## 一、首启是**步骤列表 + 每步自报状态**

`onboarding/onboarding_screen.rs:69-94`：

| 元素 | 证据 | 说明 |
|---|---|---|
| `enum Step` | `:70-78` | `Welcome(WelcomeWidget)` / `Auth(AuthModeWidget)` / `TrustDirectory(TrustDirectoryWidget)` / `GovCloudGuidance{state, acknowledged}` |
| `enum StepState` | `:85-90` | `Hidden` / `InProgress` / `Complete` |
| `trait StepStateProvider` | `:92-94` | 每步自己回答"我现在什么状态" |

**进度条/侧栏不是外部维护的一份进度表**，而是问每一步要 `get_step_state()`。
`Hidden` 这一档让"不适用于当前环境"的步骤既不计入总数、也不留空位。

`OnboardingScreenArgs`（`:104-113`）决定哪些步出现：`show_trust_screen`、`show_login_screen`、`bedrock_setup_enabled`、
`login_status`、`remote_project_trust`、`app_server_target`。
`OnboardingResult`（`:115-119`）只有两个字段：`directory_trust_persisted`、`should_exit`——**首启只产出这两件事**，
其余（模型、主题）都不在这里顺手写。

`TextEntryContext{active, …}`（`:121-126`）：首启里有文本输入步时，**可打印的退出快捷键要被当作文字吃进去**
（否则用户打 `q` 就退出了）。这是首启最容易被忽略的一条。

## 二、信任目录：文案按"受限 × 已有任务"四分，取消要说明去哪

`trust_directory.rs:30-54`：`TrustDirectoryWidget{restricted, existing_task, cancel, cwd, trust_target, show_windows_create_sandbox_hint, error, selection, highlighted}`；
`TrustDirectorySelection{Trust, Quit}`。

`TrustCancelAction{Quit, AgentsOverview, CurrentTask}`（`:44-48`）——**取消不是"关掉"，而是"回到某个具体地方"**：三条明确出口。

文案按四象限给（`:59-66` 起），例如 `restricted && existing_task` 时直说：

> 「这个已有任务可能保留设置与历史，包括它在受信任期间加载的项目配置或 hooks。要用受限设置请开新任务。**该目录的信任设置不会改变**。」

即：解释"为什么不生效"与"要生效该怎么做"，而不是一句"不允许"。
`error: Option<String>`（`:40`）——信任写入失败时错误就地显示。

`directory_trust.rs:31-` `DirectoryTrustOptions`：信任判定需要哪些输入（哪些文件/目录影响信任）是显式结构。

## 三、远程公告有硬边界（安全面最实用的一条）

`security_setup.rs:21-57`：

```rust
struct Notice { title: String, description: String, action: Action }
struct Action { label: String, url: String }
struct Identity { account: String, user: String }
impl Notice { fn valid(&self) -> bool {
    !title.trim().is_empty() && title.len() <= 256
    && description.len() <= 2048
    && !title.chars().any(char::is_control)
    && !description.chars().any(char::is_control)
} }
```

- 服务端下发的公告**先过本地校验**才展示：标题 ≤256、描述 ≤2048、**禁止控制字符**（防 ANSI/终端注入）；
- `Identity` 由 auth 派生（`account` 必须存在，`user_id` 也要有）——公告按身份取，不按任何本地可伪造标识；
- `permission_discovery.rs:26` `PermissionDiscovery` 是另一个方向：**发现本机实际拥有的权限**，而不是假设。

> 这条纪律对 gaea 直接可用：任何来自模型/服务端/工具输出的"要展示的文案"都必须过长度与控制字符校验，
> 因为 gaea 的终端/日志/Markdown 三条通道都会原样渲染。

## 四、启动草稿恢复有明确阶段

`startup_recovery.rs:14-53`：`tokio::task_local!{ static DRAFT }` + `RecoveryDraft{snapshot, stage}`，
`RecoveryStage = Startup | HandedOff | Submitted | Dispatched(String)`。

- `scope(future)`（`:31-34`）把恢复副本限制在**本次启动**的作用域内（含初始化期间被 poll 的 future）；
- `remember(snapshot)`（`:36-44`）：输入变化时替换唯一副本，阶段回 `Startup`；
- `handed_off(snapshot)`（`:46-53`）：草稿移交进 ChatWidget 后，才把阶段推进到 `HandedOff` 并记录合并后的目标。

即"崩溃/重启后能把没发出去的话找回来"，且**知道它停在哪一步**（还没交出去 / 已交出去 / 已提交 / 已派发）。
`startup_preflight.rs`（78 行）、`startup_presentation.rs`（96 行）、`startup_hooks_review.rs`（18.8KB）、`startup_orchestration.rs`（38.9KB）
把"启动期要做的检查"拆成独立阶段，而不是塞进一个 `main`。

## 五、需要用户"本人验证"的请求是三态

`bottom_pane/user_verification.rs:33-69`：`UserVerificationRequest{title, description, thread_label, server_name, request_id}`、
`UserVerificationDecision{Verify, Cancel}`、`UserVerificationState{Prompt, Waiting, Complete(ViewCompletion)}`，
选项带 `shortcuts: Vec<KeyBinding>`（每个选项的键来自运行期键位表）。

`Waiting` 这一态很关键：**点了验证之后界面不能假装已经好了**，要等结果回报成 `Complete`。

`onboarding/welcome.rs:28-31`：`MIN_ANIMATION_HEIGHT = 41`、`MIN_ANIMATION_WIDTH = 64`、动画框 60×21——
窗口不够大就不放动画，而不是把它裁一半。

## 六、gaea 现状核对

| gaea | 现状 | 差在哪 |
|---|---|---|
| `components/chat/WelcomeScreen.tsx`（6.5KB） | 欢迎屏（空会话展示 + 建议） | 不是"首启流程"：无模型/登录/目录信任的分步引导，也没有"步骤状态" |
| `internal/gaea/permission/*`（含新 `approval_mode.go`） | 权限模型 + 审批模式 | 后端有语义；**前端没有"目录信任"这一层 UI**，也没有取消去向 |
| `gaea/components/ApprovalModal.tsx` | 审批卡 | 见 F31 两处缺陷（键盘陷阱、`3=终止` 无效）；与"信任"不是一回事 |
| 全仓 | 无启动草稿恢复 | 崩溃/重启后未发送的输入丢失 |
| 全仓 | 无远程公告通道 | 若将来加，必须带长度/控制字符校验 |
| 全仓 | 无"本人验证"三态组件 | 缺（现有 AskCard 是问答，不是验证） |

## 七、三档判定

**档一（照搬语义）**

1. **步骤状态由每步自报**（`StepState{Hidden,InProgress,Complete}`），进度不另存一份。
2. **`Hidden` 不计入总数**——不适用步骤不留空位。
3. **取消给出路**：取消 = 回到某个具体界面（三选一），不是"关掉"。
4. **解释"为什么不变"**：受限/已有任务这类组合要写明后果（信任设置不变、需开新任务）。
5. **远程文案过校验**：长度上限 + 禁控制字符（gaea 的 Markdown/日志/终端三通道都需要）。

**档二（改造搬）**

6. **首启三步**：模型（含本地/云）、目录信任、可选的引导文档。落地前先定 gaea 的"信任"到底是什么（空间策略？工作区？）。
7. **启动草稿恢复**：`RecoveryStage` 四态映射到前端"未发送草稿"持久化（localStorage + 会话 key），至少做到"重启找回"。
8. **本人验证三态**（Prompt/Waiting/Complete）：给"需要用户确认身份"的流程用（如敏感工具首次授权）。

**档三（不搬）**

9. `WelcomeWidget` 的 60×21 动画框与最小尺寸门槛（Web 用 CSS 容器查询实现同类降级）；
10. `GovCloudGuidance`/`bedrock` 与 `auth` 的 OAuth 设备码流程（gaea 用自己的模型/Key 体系）。

## 八、施工卡

| 卡 | 落点 | 改动 | 验收 |
|---|---|---|---|
| **F41-A 步骤状态接口** | 新 `gaea/lib/onboardingSteps.ts` | `Step[]` + `getStepState()` + `visibleSteps()` | 单测：Hidden 步骤不进进度分母；完成态可回退不重算 |
| **F41-B 目录信任 UI** | `SettingsPage`/首启 | 信任态 + 取消三出口 + 四象限文案 | 单测：受限+已有任务 ⇒ 出现"信任设置不变"文案；取消 ⇒ 去 AgentsOverview |
| **F41-C 文案校验** | 新 `gaea/lib/noticeGuard.ts` | `sanitizeNotice({title,description,action})`：长度 + 控制字符 | 单测：含 `\x1b` 的标题被拒；2049 字被拒；正常通过 |
| **F41-D 草稿恢复** | `lib/composerDrafts.ts`（已存在）+ 启动钩子 | 阶段标记 + 重启找回提示 | 单测：写入→模拟重启→草稿回来；已发送的不再恢复 |

**开关**：C 可先行（纯函数）；A/B 依赖产品对"信任"的定义拍板；D 与既有 `composerDrafts` 合并，勿新开存储。

## 九、风险

1. **信任语义**：gaea 已有"空间策略/权限模式/工作区"三层概念，新增"目录信任"若定义不清会变成第四套。建议复用 `permission` 包而非另立。
2. **草稿恢复与分支树**：草稿属于哪个会话/分支必须写清楚（`sessionKey` 已存在），否则恢复后发到错会话。
3. **首启不可阻断**：步骤可跳过；`should_exit` 语义在 Web 侧等价于"关闭引导"。

## 十、证据自查

- 步骤与状态：`onboarding/onboarding_screen.rs:69-126`。
- 信任：`onboarding/trust_directory.rs:30-66`、`onboarding/directory_trust.rs:31+`。
- 公告边界：`security_setup.rs:21-57`；权限发现：`permission_discovery.rs:26`。
- 草稿恢复：`startup_recovery.rs:14-53`；启动分层：`startup_*` 七个文件。
- 验证三态：`bottom_pane/user_verification.rs:33-69`；欢迎动画门槛：`onboarding/welcome.rs:28-31`。
