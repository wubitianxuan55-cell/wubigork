# 上下文界面 1:1 完整复刻收官批（刀11–15）· 2026-11-12

> 承 `docs/gaea-contextview-upgrade-research-2026-11.md`（对账表+差距四档）与
> Wave1–4（ba4e97bf→fda7f7b3）之后，用户令「完整复刻 dsh-context、全面升级
> 换代」——拍板池清池批。事实源：本机 DSH 安装的 dsh-context@0.82.2 包 +
> 上游 dsh-context@0.83.1 tgz 的 `lib/client.js` 内嵌字典（734 键全提取，
> `docs/dsh-context-locale-zh-upstream.txt`），非 README 截图。

## 本批五刀

| 刀 | 内容 | 上游对照 |
|---|---|---|
| 刀11 | Fleet 成员卡树：`FamilyAgentRows`（Go 源日志直算口径，不吃事件流 400 截断）+ 每成员卡（主控/子 Agent 徽标、状态点 join 实时 runs、Token/费用/委派/消息/回报计数）+ 点击联动事件检查筛选（复用 Wave4 chips 模式） | `fleet.role.lead=主控`/`fleet.role.subagent=子 Agent`/成员卡费用 |
| 刀12 | 趋势步详情标题：`第{turn}轮·第{step}步 (共{total}步)`；轮粒度模式 `第{turn}轮 · 共{total}步`；粒度态上提父层 | `detail.step`/`detail.turn` |
| 刀13 | inject 源名细分：fold 解析引用块结构标签（`<file/dir/image/referenced-session/resource>`）→ `ContextEvent.Form` 五形态+mixed，事件行分类 chip；**顺带根修 `splitInjected` 按首空行切分把第 2/N 个引用块误记成 user 正文的既有缺陷** | inject 事件 `form` 字段（上游 context/instructions；gaea 语义映射为引用块五形态——gaea 的 inject=@引用解析，无 developer message 注入） |
| 刀14 | promptPrefix/promptSuffix：`[agent] prompt_prefix/prompt_suffix` 配置键 + `buildSystemPrompt` 首尾缝合（子代理 L1 从同一 compiler Fork 派生，一处缝合全覆盖）+ SettingsView 顶层字段显式写回（空串=停用）+ 设置中心「系统提示词」组（前缀/后缀 textarea+hint+保存） | `settings.group.prompt`/`promptPrefix=前缀`/`promptSuffix=后缀`/`promptHint` 原文照录；上游经 systemPrompt.section 注册段，gaea 等价缝在装配函数首尾 |
| 刀15 | 趋势耗时曲线开关持久化（`contextPrefs.trendDuration`，Wave4 落开关时是视图局部态） | `settings.durationCurve` |

## 拍板池销项与诚实余项

**销项**：Fleet 卡片树 ✓ / 趋势步详情「(共 N 步)」✓ / inject 源名细分 ✓（形态与
上游对齐，来源模型如实映射）/ promptPrefix/Suffix ✓ / durationCurve 偏好 ✓。

**余项（结构差异，非缺口）**：
- `settings.placement/insightsEntry/fleetTab` 入口可见性——gaea 六 tab 结构固定，
  无「面板入口」概念，产品决策不做。
- `settings.deltaBase`（浏览器对比基线）——gaea 浏览器无 diff 基线视图，不适用。
- 上游 inject 的「技能目录/记忆」来源形态——gaea 的记忆/技能走 L1 系统提示词
  （system 分类），不产生逐轮 inject，语义等价、形态不同。

## 测试债根修（顺带，独立笔）

ContextView.test.tsx 在分支 HEAD 即红（Wave1–3 欠账，被「红灯被 tail 截断掩盖」
坑掩盖）：
- bridge mock 漏补 `FamilyEvents`（Wave3 家族费用行接入 Context tab 时）→ 全文件 34 例连环红；
- Wave2 StatsCard 复合重构（8 tile→4 tile+chips 行）与 TokenCard 缓存命中率移卡头
  （requests 汇总口径→stats.cacheHitPercent）的断言未跟改；
- 新文案使「工具调用/上下文事件/read_file/bash/缓存输入」出现多份 → 单元素
  getByText 撞车，改 findAll 口径；夹具补 `timing` 键（TimingCard 六行数据源）。

## 验证

- Go：config/contextview/trajectory/boot/app 五包全绿（新增 `TestFamilyAgentRows`/
  `TestFoldInjectSourceClassification`/`TestFoldInjectLegacyFormatFallback`）；
  gen_bindings 724=724 无漂移（本批零新增绑定方法，仅结构体扩展）。
- 前端：`tsc -b --force` 0 错（zh/zh-TW `Record<DictKey,string>` 即三语言 parity 闸）；
  全量 vitest 见提交记录（ContextView 35/35、Fleet 5/5、contextPrefs/OfficePanel 绿）。
- `wails build` 时 models.ts 随建再生（SettingsView/FamilyEventStream/ContextEvent
  结构体扩展入库）。

## 合并挂账（未动）

`git merge feat/context-upgrade`（main 已分叉为同内容 cherry-pick 并存，ff-only
不再可行，需常规 merge 自动合流）仍被主树另一条线（办公对话输出重设计）未提交
改动锁住（App.tsx/三语言 locales 相交）。树清后合并，CHANGELOG/day-log 的本批
段落在 main 侧补记（分支不预写，避免冲突）。
