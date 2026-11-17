# 资产 33：状态面 / 动效无障碍 / 剪贴板双表示（前端第三梯队，判定为主）

> 证据：codex 0.165.0 上游 `codex-rs/tui/src`。gaea 对照：`frontend/src`。
> 本篇结论偏"判定"——这批机制里只有 3 项值得动，其余明确不搬，写清楚是为了**以后不再重开**。

---

## 1. 状态卡：一张卡就是一份字段表

codex `/status` 的卡片不是"拼字符串"，而是**先收集标签、再统一列宽、再逐行渲染**：

| 机制 | 证据 |
|---|---|
| 标签先入集合去重（`push_label(labels, seen, ...)`），用于计算整卡标签列宽 | `status/card.rs:587-598`、`status/format.rs:85` |
| 字段顺序固定：Model → Model provider → Directory → Permissions → Agents.md → Account → Thread name → Collaboration mode → Session → Forked from → 空行 → Token usage → Context window | `status/card.rs:827-861` |
| 限额与告警单独成段（"Limits" / "Warning"） | `status/card.rs:587-598` |
| 取值渲染器带 `line()` / `continuation()`：首行带标签，续行按标签宽度缩进对齐 | `status/format.rs:38-53` |
| 目录按剩余宽度截断（`format_directory_display(dir, max_width)`） | `status/helpers.rs:147` |
| token 紧凑格式化（`format_tokens_compact`） | `status/helpers.rs:106` |
| 重置时间戳本地化 | `status/helpers.rs:170` |

**为什么值得抄**：这套结构解决的是"值长短不一导致列抖动"。gaea 现在的状态展示分散在 `AppStatus.tsx`（速度计）与输入卡底栏 `ContextPill`（上下文占用），各自算宽度，没有统一标签列——一旦加字段就会横向跳动。

gaea 已有正确的**单一真相**原则（`gaea/components/AppStatus.tsx:56-57` 注释：上下文占用只由 `ContextPill` 承载，状态行不重复显示）。这条比 codex 更严，保留。

### 可迁移判定
- **照搬**：字段表 + 标签去重 + 续行缩进三条结构（落到一个 `statusFields.ts` 纯函数，输出 `{label, value, kind}[]`）。
- **改造**：终端按字符宽度算列宽（等宽字体），Web 用 CSS Grid `grid-template-columns: max-content 1fr` 直接让浏览器算，不需要预扫描标签。**只抄语义，不抄宽度算法**。
- **不搬**：`FieldFormatter` 的逐 span 拼装（ratatui 专有）。

---

## 2. 限额与用量：条形图是"剩余"不是"已用"

| 事实 | 证据 |
|---|---|
| 限额进度条固定 20 段，填充 `█`、空白 `░` | `status/rate_limits.rs:24-26` |
| 渲染入口 `render_status_limit_progress_bar(percent_remaining)` / `render_limit_progress_bar(p, segments)` | `status/rate_limits.rs:369-374` |
| 摘要文案 `format_status_limit_summary(percent_remaining)` | `status/rate_limits.rs:387` |
| 线程用量分三组：Models / Reasoning（8 档有序）/ Speed（`Fast mode` `Ultrafast` `Standard`）/ Billed tokens | `status/thread_usage.rs:19-33` |

要点是**口径**：进度条与百分比都按"剩余"算，且 reasoning 档位有固定顺序表（`REASONING_ORDER: [&str; 8]`），不是按出现顺序。gaea 的成本/模型中心如果按"已用"画，就会和用户直觉相反。

### 可迁移判定
- **照搬**：`percent_remaining` 单一入参 + 固定档位顺序表（`status/thread_usage.rs:19`）→ gaea `ModelCenterPage` / 成本库的档位排序。
- **不搬**：`█/░` 字符条（Web 用 `<progress>` 或 div 宽度）。

---

## 3. 动效与无障碍：偏好是**启动时读一次**，且 JS 必须显式降级

| 事实 | 证据 |
|---|---|
| 动效原语集中在一个模块，要求调用方**显式选择** reduced-motion 回退，而不是各调各的 | `motion.rs:1-3` |
| 启动时读一次系统无障碍偏好；显式请求"减少动效"会压制 `tui.animations` 下所有效果；改 OS 偏好需重启生效 | `system_motion.rs:1-5` |
| 屏幕阅读器一次性探测 → 播种**持久**动画默认值，且不覆盖显式配置 | `screen_reader.rs:1-4` |
| Windows 特例：Narrator 不设置其他辅助工具用的屏读标志，单独判定 | `screen_reader_windows.rs:3` |
| 空态动画：固定 50ms 帧间隔、舞台 21 行（最小 14 行）、终态用静止姿态（`SETTLED_BLOSSOM = 0.5`） | `empty_state_animation.rs:32,34,200-201,231` |
| 全局帧率上限：`MIN_FRAME_INTERVAL = 8_333_334ns`（≈120fps），提交动画 tick 对齐 `TARGET_FRAME_INTERVAL` | `tui/frame_rate_limiter.rs:13`、`app.rs:460` |

gaea 现状：CSS 层已全面尊重 `prefers-reduced-motion`（`chat-board.css:1308`、`chat-signal-redesign.css:1822/2454/3405/3630`、`chat-workspace.css:622/1572/2751`、`index.css:378`），JS 层也有单源判定 `prefersReducedMotion()`（`gaea/lib/gsapAnimations.ts:15-18`）并被三个 hook 使用（`useEntranceAnimation.ts:55`、`useGSAPCollapse.ts:61`、`useMountTransition.ts:34`）。

**真实缺口只有两处**：
1. 判定是**每次调用时**读 `matchMedia().matches`，没有"启动播种 + 变更订阅"——用户中途改系统偏好，已挂载的动画不响应（codex 的做法是启动读一次并明确"重启生效"，比"静默不响应"诚实）。
2. JS 动画没有统一帧率上限：`requestAnimationFrame` 在 144Hz 屏上会跑 144fps，做逐帧补间时浪费；codex 把上限定死在 ≈120fps（`frame_rate_limiter.rs:13`），且测试用 `MIN_FRAME_INTERVAL` 断言"不许早于"。

### 可迁移判定
- **照搬**：单一 `motion` 出口 + 显式回退参数（禁止调用方裸调 gsap）；帧率上限常量。
- **改造**：`prefers-reduced-motion` 改为 `addEventListener('change')` 订阅 + 首帧播种（浏览器支持，比重启生效体验好）。
- **不搬**：屏幕阅读器探测（Web 有 ARIA/`aria-live`，探测无意义）、sixel/像素舞台。

---

## 4. 剪贴板：一次写入两种表示

| 事实 | 证据 |
|---|---|
| 剪贴板句柄用 `ClipboardLease` 包住（Windows 上句柄必须保活） | `clipboard_copy.rs:12` |
| 空选择**不修改**剪贴板（避免清空用户剪贴板） | `clipboard_copy.rs:16` |
| 复制 Markdown 时**同时**向原生剪贴板提供 HTML 表示 | `clipboard_copy.rs:17` |
| 终端/WSL 的回退写入"与已确认的原生写入不可区分"，调用方必须自己区分 | `clipboard_copy.rs:19` |
| 语义超链接与可见文本分离携带（可见文本可被折行/截断，链接不受影响） | `terminal_hyperlinks.rs:1` |

gaea 现状：`gaea/lib/clipboard.ts:11-33` 是单出口，`navigator.clipboard.writeText` 优先、`execCommand` 兜底、返回布尔值，注释（:1-9）记录了 v4.513 那轮"假已复制"修复——**痛点与 codex 的 `:19` 完全同构**。

**真实缺口**：只有 `writeText`，没有 `ClipboardItem` 多 MIME。后果：从聊天里复制一段带标题/列表/表格的内容到 Word、飞书、Notion，格式全丢，只剩纯文本。

### 可迁移判定
- **照搬**：空内容不写、返回值诚实、失败可见（gaea 已做到）。
- **改造（推荐）**：`copyToClipboard(text, { html })` → `navigator.clipboard.write([new ClipboardItem({'text/plain': ..., 'text/html': ...})])`，HTML 侧复用现有 markdown 渲染结果（`MarkdownContent` 的 DOM 序列化）而非二次解析；`execCommand` 兜底路径保持纯文本并**在返回类型上区分**（`'rich' | 'plain' | false`），调用方据此决定是否提示"已复制为纯文本"。
- **不搬**：`ClipboardLease` 句柄保活（浏览器无此问题）。

---

## 5. 视图切换的按键守卫（小而值钱）

终端里换视图时，用户按 Enter 的那一下可能被**新视图**接收，造成误触发。codex 的处理：

- 增强键盘终端：区分 `Press` 与 `Repeat`（按住产生的重复事件不算激活）。
- 老终端：开一个 1 秒的有界窗口（`INITIAL_GUARD`/`LEGACY_REPEAT_GUARD` 均 1s），按住期间窗口顺延。
- 证据：`activation_key_guard.rs:1-12`。

Web 同款问题：命令面板打开动画期间按住的键会连续触发列表项；`Enter` 打开弹窗后立刻冒泡到弹窗的首个按钮。

### 可迁移判定
- **改造**：`keydown` 用 `event.repeat` 直接过滤（浏览器原生支持，不需要 1s 窗口），仅对"打开弹层/切换视图"这类动作生效，不做全局拦截。

---

## 6. 施工卡（若这批要落地）

| 项 | 落点（计划） | 验收断言 | 开关 |
|---|---|---|---|
| 状态字段表 | `frontend/src/gaea/lib/statusFields.ts`（纯函数）+ 状态组件消费 | 同一状态对象渲染两次输出逐字段相同；字段顺序与表一致；标签列宽由 CSS 决定（无 JS 测宽） | 无（行为等价重构） |
| 档位顺序 | `ModelCenterPage` / 成本库排序 | 8 档 reasoning 按固定表排序，缺档不塌缩 | 无 |
| 帧率上限 | `gaea/lib/frameLimit.ts`（`MIN_FRAME_INTERVAL = 8.33ms`） | 连续 60 次 `schedule()` 在一帧内只执行一次；间隔 < 8.33ms 的请求被合并 | `motion.frame_limit.enabled`（默认开） |
| 动效偏好订阅 | `gsapAnimations.ts` 加 `subscribeReducedMotion` | 运行时切换 OS 偏好，已挂载组件在 1 次渲染内切换（不刷新页面） | 无 |
| 富文本复制 | `gaea/lib/clipboard.ts` 扩签名 | 复制含表格的消息后，`ClipboardItem` 含 `text/html`；HTML 中表格保留 `<table>`；纯文本路径返回值 `'plain'` | `clipboard.rich.enabled`（默认关，先灰度） |
| 按键守卫 | 弹层/视图切换入口 | 打开弹层后 200ms 内 `event.repeat=true` 的 Enter 不触发弹层首项 | 无 |

**回退**：五项均为"新增纯函数 + 调用点接入"，删除接入即回到现状；`clipboard.rich.enabled` 关时走原 `writeText` 分支（逐字节等价）。

---

## 7. 风险

1. **富文本复制带出未清洗 HTML**：复制的 HTML 必须来自已 sanitize 的渲染结果（现有 `gaea/lib/sanitize.ts` 白名单），不能把 markdown 原文直接转 HTML 再塞剪贴板——那是绕过清洗的新通道。
2. **动效偏好订阅造成重渲染风暴**：订阅回调必须只更新一个 boolean 的 store 切片，不能让每个动画组件各自 `setState`。
3. **帧率上限与现有 GSAP 时间线冲突**：GSAP 自己有时钟，硬限帧会让补间抖动；只对"自己写的 rAF 循环"限帧，不碰 gsap ticker。
4. **状态字段表变成第二真相**：字段表只负责"顺序与标签"，值必须仍来自 store 单一来源（对齐 `AppStatus.tsx:56-57` 的既有纪律）。

---

证据自查：`status/card.rs:827-861`（字段行）、`status/rate_limits.rs:24-26`（20 段条）、`clipboard_copy.rs:17`（markdown 复制同时提供 HTML）。
