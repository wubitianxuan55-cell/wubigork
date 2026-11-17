# codex → gaea 前端蒸馏 · 第二批（波 2）清单：资产面之外的四个面

> **边界（本批硬约束）**：`C:\AI\wubigrok` 只读。产物只有本目录下的文档，
> **不改 gaea 源码、配置、测试、依赖、绑定**。所有"施工卡"都是待拍板计划。
>
> 上位清单见 `frontend-distill-plan.md`（波 1：F27–F33，消息流与渲染面）。
> 本篇是波 2：**卡片面、输入面、会话面、回退/提示面**——即"一条消息之外"的四个界面。

---

## 1. 为什么还要一波

波 1 的七篇全部落在**消息流内部**（流式渲染、转录视图、markdown、diff、底部面板、键位、状态/剪贴板）。
把七篇的缺口去重后只剩 6 条（`frontend-distill-plan.md` §2.6），但那 6 条都能归结为"消息流怎么画"。

codex 的 `tui/src` 里还有另一半面积，与消息流正交、且 gaea 都有对应界面：

| codex 模块 | 体量 | gaea 对应 | 波 2 是否覆盖 |
|---|---|---|---|
| `history_cell/*` + `exec_cell/*` | 434 + 85 KB | `gaea/components/ToolCard.tsx`、`ToolGroup.tsx`、`Transcript.tsx` | ✅ 资产 34 |
| `bottom_pane/mentions_v2` + `command_popup` + `slash_input` | ~120 KB | `useComposerMenus.ts`、`SlashMenu.tsx`、`ArgMenu.tsx` | ✅ 资产 35 |
| `resume_picker.rs` + `named_session_lookup` | 264 + 7 KB | `useSessionManager.ts`、`Sidebar.tsx`、`projectGroups.ts` | ✅ 资产 36 |
| `app_backtrack.rs` + `footer_hint` + `shortcut_help` | 44 + 32 KB | `useAppKeyboard.ts`、`ChatRow` 编辑态 | ✅ 资产 37 |
| `config_panel` + `theme_picker` + `onboarding` | 68 + 22 + 191 KB | 设置页族、模型中心、`WelcomeScreen` | ⬜ 波 3 候选 |
| `multi_agents.rs` + `app/agent_center` + `thread_transcript` | 35 + 35 + 12 KB | 众神殿 / `SubagentThread` / `DagPanel` / Agent 网络 | ⬜ 波 3 候选 |
| `analytics/*` + `token_usage` + `status/*` | 429 + 3 KB | `ContextView`、调用统计、成本库 | ⬜ 波 3 候选 |
| `wrapping.rs` + `live_wrap` + `line_truncation` | 73 + 10 KB | 无对应（浏览器换行） | ⬜ 判定不搬（见 §5） |
| `notifications/*` + `terminal_title` + `screen_reader` | 8 + 11 KB | 系统通知 / 窗口标题 | ⬜ 低价值，暂不立项 |

## 2. 本批四篇的定位与一句话结论

| 编号 | 文件 | 主题 | 一句话结论 |
|---|---|---|---|
| 34 | `asset-34-tool-exec-card-family.md` | 工具/执行卡片族 | 一张卡是**七种投影 + 四条旁路**的契约；披露必须**量化**（`OutputLines(hidden)`），输出预算是**三层**（字节→行→单行头尾），且 `call_id` 路由失配应报错而不是把孤儿结果挂到别的卡上 |
| 35 | `asset-35-input-popup-candidates.md` | 输入弹层候选 | 匹配要**两段**（exact→prefix）、Esc 身份要**三元组**（query+token+occurrence）、异步结果要**三元守卫**（pending/display/waiting）、Tab 有**六道**判据（含"有可见项但无选中项不许提交"） |
| 36 | `asset-36-session-surface.md` | 会话面 | 检索要能**边搜边续页**且带 token；预览与打开是**两个状态机**；按名字找会话**同名必报歧义、跨页不确定就拒绝猜** |
| 37 | `asset-37-backtrack-and-hint-single-source.md` | 回退与提示 | Esc 是**阶梯 + 守卫**（草稿非空不回退、两段式防误触）；回退判据落在**回合边界/完整性/在飞状态**；提示纪律是**不切开 + 按宽度换档 + 度量与渲染同源** |

## 3. 跨篇收敛：波 2 真正的缺口（去重后 5 条）

| # | 缺口 | 出处 | 性质 | 与波 1 的关系 |
|---|---|---|---|---|
| 1 | **披露/截断是"按能力"不是"按事实"**：卡片可展开但点开是空块；折叠计数不含宽度裁掉的条数 | 34 §六/§八 | 正确性（撒谎式 UI） | 与 F27 的"表格 holdback"同族，但作用在卡片层 |
| 2 | **异步结果无身份守卫**：`live` 布尔只防卸载，不防乱序覆盖；无 loading 态 | 35 §四、36 §二 | 正确性（慢盘/大列表必现） | 与 F28 的"搜索快照"同族 |
| 3 | **回退不是事务**：没有回合边界判据，也没有失败回填 | 37 §三 | 数据安全 | 全新，波 1 未涉及 |
| 4 | **提示零散**：无 hint bar、无换档、无同源派生；Esc 阶梯隐含在 3 行 if 里 | 37 §四/§五 | 一致性/可维护 | F32（键位表）的下游，须先有键位表 |
| 5 | **两壳双份拷贝开始漂移**：`ToolCard` 19.6KB vs 18.4KB、`useComposerMenus` 只在 gaea 侧 | 34 §八、35 §十一 | 结构性风险 | 波 1 §6 已点名，本批给出单源化路径（F34-E） |

**建议优先级**（按"错得离谱"排，不看改动量）：
① F34-A 披露量化 → ② F35-C Tab 契约（含"不许提交"）→ ③ F37-B/C 回退判据与失败回填 →
④ F35-D/F36-A 异步守卫与去重 → ⑤ F34-E 单源化 → ⑥ F37-D/E 提示同源 → ⑦ 其余。

## 4. 与既有批次的去重

| 已有 | 相邻项 | 处理 |
|---|---|---|
| 波 1 F27 流式双区 | 34 的 live 输出三层预算 | 不同层：F27 管"已提交正文的稳定切点"，34 管"工具输出的保留量"；常量命名要分开（`stable region` vs `retained output`） |
| 波 1 F28 转录视图（行级、搜索、选区） | 36 的会话列表检索 | 一个在"会话内"，一个在"会话间"；`seen_key` 与 F28 的 `EntryKey` 不同命名空间 |
| 波 1 F31 底部面板/视图栈 | 37 的 Esc 阶梯 | **F31 是编写方向（谁能收键），37 是入口守卫（草稿非空、两段式）**；实现上 37 的阶梯必须调用 F31 的同一份裁决表，不许各写一套 |
| 波 1 F32 键位表 | 37 的提示同源、36 的键位占用检查 | 37-D/E 依赖 F32；F32 未落地前 37-D 只能做"提示不切开"，做不了"从键位表派生" |
| 波 1 F33 剪贴板/状态 | 34 的 `copy_source` | 34 提供"复制源"这一层，F33 提供"双表示写入"；两者合起来才是完整复制链 |
| 第二批 T6/T7/T9（上下文碎片/章节/压缩） | 34–37 全部 | 零重叠：那边是**上下文装配**，本批是**界面契约** |
| T13 执行语义（pending vs transcript 双缓冲） | 34 的 live/aggregated 双份 | 概念同构、实现独立，别互引代码 |

## 5. 明确判定"不搬"的清单（免得下批重复讨论）

1. **终端渲染层**：`custom_terminal.rs`、`insert_history.rs`、`tui/*`、`Clear` 重绘、`HyperlinkLine` 元数据。
2. **终端度量**：`wrapping.rs`（grapheme 级换行 + sentinel 字节）、`live_wrap`、`display_width`、`column_width` 配置。
   浏览器换行由 Blink 负责；可迁移的只有**不变量**（"软换行不改源文本""光标是源偏移"），
   而 gaea 的输入面是 DOM textarea，这条不变量天然成立。
3. **终端专属常量**：`MAX_POPUP_ROWS=8`、`TOOL_CALL_MAX_LINES=5`、`DETAIL_PREVIEW_LINES=3`、`LIVE_*_BYTES=10280`、`FOOTER_COMPACT_BREAKPOINT=120`。
   数值全部按 Web 实测重定，只保留"上限/不变量"的语义。
4. **终端会话/窗口集成**：`terminal_title`、`iterm_session_status`、`screen_reader_*`、`AltScreenGuard`。

## 6. 门禁（拍板落地时才跑；本批不跑）

```powershell
cd frontend
npm.cmd run build      # tsc -b + vite build
npm.cmd run lint       # eslint .
npm.cmd test           # vitest run
```

- `frontend/src/perf-guards.test.ts`：任何"新增订阅 / 新增 rAF / 新增全局监听"必须加一条断言。
  本批四篇里 34（卡片预算）、35（菜单重算）、36（预览请求）、37（Esc 阶梯）**都可能触发**，逐卡点明。
- 两壳改动纪律：`gaea/` 与 `wenxue/` 同名组件必须先决定单源方向（F34-E），再动实现。

## 7. 全批风险

1. **两套壳**：同机制两处消费时，纯函数进 `frontend/src/gaea/lib/` 单源，壳只做接线（波 1 §6 已列先例）。
2. **Wails 序列化旧账**：`SessionMeta`/`ToolItem` 扩字段必须给 camelCase json tag + mock 同步
   （历史事故：Go struct 缺 tag → 前端读空，progress v4.269）。
3. **流式路径最敏感**：34 的预算改动会改卡片高度，牵动 `useFollowScroll` 与 `content-visibility`；
   验收必须包含"折叠/展开第 50 张卡后上下滚动无跳变"。
4. **诚实反馈纪律**：34 的 `hiddenLines`、36 的 `reached_scan_cap`、37 的失败回填，三处都是"不许假装成功"，
   与 `gaea/lib/clipboard.ts:1-9` 记录的 v4.513"假已复制"事故同一纪律。

## 8. 报道格式（落地批次用）

```
F<n> <任务名>
基线：<测试文件> N 例 / 既有失败 M 例
改动：<file:line> 一句话
门禁：build / lint / test 结果
开关：默认 <关/开>；关闭时与现状逐字节等价：是/否
性能：<是否新增订阅/监听> + 实测数字
风险：<是否触碰 ChatPage 滚动宿主 / 两壳是否同步>
```

## 9. 波 3（已交付，同一日）——原文登记表保留，供追溯

> 本节原先登记"待立项"，同一批次内已全部交付为 `asset-38…46`，并出 `frontend-coverage-ledger.md` 做全量核对。
> 下表保留当时的判断（价值评级），实际篇号见右列。

| 面 | 值得先看什么 | 当时预估价值 | 交付篇 |
|---|---|---|---|
| 多智能体界面 | `multi_agents.rs`（35KB）与 `app/agents_overview*`（52.9+30.5KB）：子代理进度、角色卡、汇总视图 | 高 | **38** |
| 设置与主题 | `config_panel`（68KB）+ `theme_picker`（22KB）：配置项可发现性与即时预览 | 中高 | **39** |
| 用量分析 | `analytics/*`（66 文件 429KB，含 plot/activity_chart）+ `status/*` | 中 | **40** |
| 引导与安全 | `onboarding/*`（9 文件 ~191KB）、`security_setup.rs`、`permission_discovery.rs` | 中 | **41** |
| 集成面 | MCP elicitation（2884 行）、hooks 浏览、技能/记忆/应用链接、反馈 | 中高 | **42** |
| 工作树与环境 | `worktree_browser`、`managed_worktree_tools`、`goal_files`、`branch_summary` | 中 | **43** |
| 浮层与文本度量 | `pager_overlay`、`resize_reflow_cap`、`line_truncation` | 中高 | **44** |
| 动效与视觉件 | `motion`/`system_motion`/`empty_state_animation`/`inline_visualization`/`tooltips` | 中 | **45** |
| 更新与运行 | `update_*`、`notifications`、`daemon_*` | 低 | **46** |
