# 资产 02 · 回合上下文装配（world_state sections）

> 源码：`codex-rs/core/src/context/world_state/`（40 文件，其中 18 个是章节实现 + `mod.rs` 装配器 23.5 KB）
> 与 `codex-rs/core/src/session/world_state.rs`（装配入口）。
> 二进制侧对应证据：`hotspot-02` 的字段表（`context_management` / `incremental_tools` / `deferred_tool_world_state`）
> 与 `hotspot-06` 的 tools_budget 算法。

---

## 一、机制：18 个「章节」按稳定 ID 装配，且只渲染差异

`world_state/mod.rs` 定义 `WorldStateSection` trait，每个章节：

```rust
trait WorldStateSection {
    const ID: &'static str;          // 稳定标识，持久化进 rollout，必须永不变
    type Snapshot;                   // 「只存比较所需」的极简快照
    fn should_persist(&self) -> bool;
    fn matches_legacy_fragment(&self, role: &str, text: &str) -> bool;  // 兼容历史片段
    fn matches_retained_fragment(&self, role: &str, text: &str) -> bool;
    fn render_diff(&self, previous: PreviousSectionState<'_, Self::Snapshot>) -> SectionTransition;
}
```

三个设计约束（注释里写死的）：

1. **ID 稳定**：`ID` 会被持久化进 rollout，改了就等于历史会话无法识别自己的章节。
2. **Snapshot 只放比较数据**，且**不得序列化成 null**（merge-patch 语义里 null 表示删除）。
3. **只讲变化**：`render_diff` 拿到上一轮快照，返回的是「这一轮模型需要被告知的部分」，
   而不是整段世界状态重发。

## 二、18 个章节清单（ID 原样）

| 章节 ID | 承载内容 |
|---|---|
| `agents_md` | AGENTS.md 层级内容 |
| `apps_instructions` | Apps（连接器）使用说明 |
| `collaboration_mode` | 协作模式（含 plan 模式） |
| `approved_command_prefixes` | 本轮已批准的命令前缀（供给模型的「别再问了」清单）|
| `context_window_guidance` | 上下文窗口用量指引 |
| `environments` | 环境清单（含子智能体，上限见下）|
| `environments_instructions` | 环境使用说明 |
| `managed_developer_instructions` | 企业托管开发者指令 |
| `model_catalog` | 模型目录摘要 |
| `model` | 当前模型信息 |
| `multi_agent_mode` | 多智能体模式 |
| `multi_agent_usage_hint` | 多智能体用法提示 |
| `permissions` | 权限/审批策略 |
| `persistent_mode` | 常驻模式 |
| `plugins_instructions` | 插件使用说明 |
| `realtime` | 实时会话状态 |
| `tools` | 工具清单（受 tools_budget 预算约束）|
| `top_level_tools` | 顶级工具子集（延迟展开的另一半）|

**每个章节独立成文件、独立测试**（`*_tests.rs` 与实现同目录），说明这是「可插拔上下文」而不是一段拼接字符串。

## 三、预算常量（同族）

- `session/world_state.rs`：`MAX_ENVIRONMENT_SUBAGENTS = 8`、`MAX_ENVIRONMENT_SUBAGENT_BYTES = 1024`；
- `tools_budget.rs`：共享字节预算 + 名字优先 + 描述轮次均分（见 `hotspot-06`）；
- 各章节自己也带截断（例如 `environment.rs` 20 KB 里有渲染与截断逻辑）。

## 四、对 gaea 的映射

gaea 的 `internal/gaea/contextview/`（20 文件）目前是**展示型**：算占用、画上下文构成。
`prompt`（4 文件）负责拼字符串。两者之间没有「章节」这一层。

改造建议（三档，可只做第一档）：

**第一档（最小，1 天）**：把现有注入内容登记成「章节表」——
给每段起稳定 ID（`agents_md` / `world_state` / `memory` / `skills` / `attachment`），
落在 contextview 里能看到「每段多少字节、占比多少」。**不改变注入行为**，只加可观测。

**第二档（2–3 天）**：加 Snapshot 与 diff 渲染——
每轮只把「变化的章节」放进上下文，未变化的章节以「同上」标记，配 `matches_text` 保证能定位旧片段。
收益：长会话里世界状态部分从「每轮重发」变成「变化才发」，直接省 token，且改动可解释。

**第三档（按需）**：每章节独立预算（照 tools_budget 的算法），按配额裁剪而不是全局截断。

## 五、验收（第二档）

- 同一会话连续 3 轮无状态变化 → 上下文里该章节只出现一次（不重复注入）；
- 状态变化时只出现「变化项」，且模型能据此正确回答（准备 3 条人工用例）；
- 历史会话（旧版本产生的片段）能被 `matches_legacy_fragment` 识别，不出现「同一章节两份」。

## 六、为什么值得做

gaea 现在的上下文是「每轮重新拼」，两轮之间没有任何「我已经告诉过模型了」的记忆，
所以长会话里重复注入占比很高——这和 codex 用 18 个章节 + 快照 diff 换来的省法是同一问题的两种解法。
**这是本轮挖到的最省 token 的一处机制**，且不需要模型配合、纯工程。
