# 资产 40 · 用量与统计面：状态卡的字段纪律、限额条的"陈旧"标记、图表的数据新鲜度

> 证据根（只读）：`status/{card,rate_limits,thread_usage,helpers,remote_connection,format}.rs`（978/550/337/350/… 行）、
> `token_usage.rs`（89）、`analytics/`（66 文件 429KB，本篇聚焦 `dashboard/report_data/chart/activity_chart/summary_panel/navigation/hints/controls/data/normalize`）。
> 对照根（只读）：`frontend/src/pages/RuntimePage.tsx`（57.6KB）、`frontend/src/gaea/components/ContextView.tsx`（88.7KB）、
> `frontend/src/pages/CostLibraryPage.tsx`（28.5KB）、`frontend/src/gaea/components/{AppStatus,ContextPill}.tsx`。
>
> 只出机制与落点；不改 gaea 源码。

---

## 一、`/status` 卡是"提交进历史的一条 cell"，不是浮层

`status/card.rs`：`StatusHistoryCell`（`:126-`）装着 `model_name / model_details / directory / …`，
由 `StatusHistoryHandle`（`:76-123`）暴露三个动作：

| 动作 | 证据 | 说明 |
|---|---|---|
| `copy_text()` | `:81-91` | 用 `content_lines(u16::MAX)` 拼纯文本——**屏幕上的卡可原样复制**（宽高无关） |
| `reserve_thread_usage_label_width()` | `:93-95` | 先占位标签宽度，估算到达后**数字变长也不抖列** |
| `finish_rate_limit_refresh(&snapshots, now)` | `:97-115` | 刷新完成后重建限额数据 |
| `set_thread_usage(Option<ThreadUsage>)` | `:117-122` | 异步估算落进 `Arc<RwLock<Option<_>>>` |

数据结构（`:55-68`）：`StatusTokenUsageData{total, input, output, context_window?}`，其中
`StatusContextWindowData{percent_remaining, tokens_in_context, window}`——**百分比按"剩余"存**（不是已用），
与 F33 记的"限额按剩余口径"一致。另有一个 `CHATGPT_USAGE_URL`（`:53`）：本地算不清的部分给外链，而不是自己编。

## 二、限额条：20 段、15 分钟陈旧阈值

`status/rate_limits.rs`：

| 常量/结构 | 证据 | 含义 |
|---|---|---|
| `STATUS_LIMIT_BAR_SEGMENTS = 20` | `:24` | 条固定 20 格 |
| `STATUS_LIMIT_BAR_FILLED/EMPTY = "█"/"░"` | `:25-26` | 填充/空格字形 |
| `RATE_LIMIT_STALE_THRESHOLD_MINUTES = 15` | `:66` | **超过 15 分钟的数据要标"可能过期"** |
| `RateLimitWindowDisplay / RateLimitSnapshotDisplay` | `:70-105` | 单窗口 / 快照（多窗口） |
| `CreditsSnapshotDisplay / SpendControlLimitSnapshotDisplay` | `:120-131` | 余额 / 消费控制限额，各自独立结构 |

15 分钟那条是关键：限额数据不是实时真值，**界面必须能说"这是多久以前的数"**。

## 三、用量分档是**固定顺序数组**，不是散落映射

`status/thread_usage.rs:19-33`：

```
REASONING_ORDER: [&str; 8] = [None, Minimal, Light, Medium, High, Extra High, Max, Ultra]
SPEED_ORDER:     [&str; 3] = [Fast mode, Ultrafast, Standard]
MODELS_LABEL / REASONING_LABEL / SPEED_LABEL / BILLED_TOKENS_LABEL  // 均以两个空格开头
```

- 顺序由数组定义（渲染按数组走，不按键名排序）；
- 标签常量自带两个空格缩进 → **缩进属于标签文本**，不会因为拼接位置不同而错位；
- `reserve_label_width`（`:37-45`，`Arc<AtomicBool>`）解决"异步估算到达时列宽跳变"。

`token_usage.rs:9` `BASELINE_TOKENS = 12000`：基线 token 是常量，估算不凭空。

## 四、统计面：数据形状 → 归一化 → 渲染，三段分离

`analytics/report_data.rs:1-40` 文件头就把边界写死：

> 「把 endpoint 专属后端模型投影成展示归一化的输入。**这个表示永远不用于反序列化 HTTP 响应**。
> 可选字段选出每份报告用到的输入；后端缺的值保持缺失。」

- `AnalyticsData{data, units?, data_freshness_ts?, series?}`（`:11-16`）；
- `AnalyticsRecord` 的每个维度都是 `Option`（`:27-40`）：`values` / `product_surface_usage_values` / `premium_usage_values` /
  `attribution` / `credit_amount` / `totals` / `models` / `groups` / `clients` / `plugin_usage_overviews` / `skill_usage_overviews`；
- 值容器用 `BTreeMap<String, f64>`：**键有序**，图表不用二次排序；
- `units` 与 `data_freshness_ts` 与数据同框：单位与新鲜度是数据的一部分，不是渲染时的猜测。

`analytics/activity_chart.rs:18-25`：`WEEK_COUNT = 52`、`DAY_COUNT = 7`、`CELL_COUNT = WEEK_COUNT * DAY_COUNT`（364）、
`CHART_LEFT_WIDTH = 4`、`TokenActivityView` 枚举——热力图的格子数是**编译期常量**，缺数据留空而不是压缩网格。

面板是分工的：`dashboard` / `summary_panel` / `plan_panel` / `chat_panel` / `task_panel` 各一块（`analytics/panels.rs` 组装），
`navigation` + `hints` + `controls` 负责"怎么在这些面板之间走"，`normalize` 负责口径归一。

## 五、gaea 现状核对

| gaea | 现状 | 差在哪 |
|---|---|---|
| `pages/RuntimePage.tsx`（57.6KB） | 调用统计主工作区（模型筛选、范围 KPI、错误/耗时分析） | 有面板分工；缺"**数据新鲜度 / 单位**"与"数据同框"的显式表达 |
| `gaea/components/ContextView.tsx`（88.7KB） | 上下文占用、treemap、趋势、事件/命令/文件下钻 | 已经是"多面板 + 下钻"结构（比 codex 更细）；缺 `reserve_label_width` 式的列宽稳定 |
| `pages/CostLibraryPage.tsx`（28.5KB） | 成本库（单价、费率、组价） | 有"来源"概念；缺限额条的"陈旧"标记 |
| `gaea/components/AppStatus.tsx` / `ContextPill` | 状态单一真相（模型/目录/权限/token） | 无"可复制的状态卡文本"（`copy_text` 的等价物） |
| 全仓 | 无 20 段限额条、无 15 分钟陈旧阈值 | 限额/配额类数字目前是"直接显示数字" |

## 六、三档判定

**档一（照搬语义）**

1. **状态卡可复制**：`copy_text()` → 前端"复制状态"按钮，输出与屏幕一致的纯文本（宽高无关）。
2. **数据新鲜度必显示**：任何统计/限额都带 `as of HH:mm`，超过阈值（15 分钟为参考值）标"可能过期"。
3. **单位与数据同框**：`units` 跟着数据走，不在渲染层猜。
4. **标签缩进属于文本**：避免拼接错位（gaea 的状态行是拼字符串，容易出这类错）。

**档二（改造搬）**

5. **列宽预留**：`reserve_label_width` 的等价物——异步数字到达前先按最大位数占位，消除抖动。
6. **固定顺序数组**：推理档/速度档这类枚举顺序用数组声明一次，渲染与筛选同源。
7. **热力图格子数固定**：`52×7` 常量，缺数据留空（gaea 的趋势图已有可见窗口逻辑，需对齐"缺数据不压缩"）。

**档三（不搬）**

8. `CHATGPT_USAGE_URL` 类外链、`STATUS_LIMIT_BAR_*` 字形（Web 用 CSS 进度条）；
9. `remote_connection.rs`（远程连接状态）与 `helpers.rs` 的终端版式。

## 七、施工卡

| 卡 | 落点 | 改动 | 验收 |
|---|---|---|---|
| **F40-A 状态卡可复制** | `gaea/components/AppStatus.tsx` + 新 `lib/statusText.ts` | `statusToText(state)` 纯函数 + 复制按钮（走 `clipboard.ts` 的诚实写入） | 单测：文本与 UI 字段一一对应；写失败不翻转成功态 |
| **F40-B 新鲜度标记** | `RuntimePage`/限额组件 | `asOf` + `staleAfterMs`（默认 15min） | 单测：now-asOf > 阈值 ⇒ 标 stale |
| **F40-C 列宽预留** | `ContextView` 数字列 | `reserveWidth(maxDigits)` | 手测：流式更新时列不抖 |
| **F40-D 枚举顺序表** | `lib/usageOrder.ts` | `REASONING_ORDER`/`SPEED_ORDER` 数组 | 单测：渲染顺序 = 数组顺序 |

**开关**：全部为展示层；F40-A 复用既有 clip 通道（`gaea/lib/clipboard.ts:1-9` 的诚实写入纪律）。

## 八、风险

1. **口径混用**：gaea 已有成本库与调用统计两套金额/用量口径，F40-B 的"新鲜度"要分别挂，不能共用一个 `asOf`。
2. **`ContextView` 是重组件**（88.7KB，含 lazy 子图）：改列宽策略前先确认 `content-visibility` 与 treemap 测量不受影响。
3. **不要照抄 20 段**：进度条格数是终端字符限制，Web 用连续进度条 + 百分比文本。

## 九、证据自查

- 状态卡结构/复制/预留/刷新：`status/card.rs:53-123,126+`。
- 限额条与陈旧阈值：`status/rate_limits.rs:24-26,66,70-131`。
- 分档顺序与标签：`status/thread_usage.rs:19-45`；基线：`token_usage.rs:9`。
- 统计三段与数据形状：`analytics/report_data.rs:1-40`；热力图常量：`analytics/activity_chart.rs:18-25`。
