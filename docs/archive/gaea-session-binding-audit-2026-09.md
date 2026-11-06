# 无参绑定会话语义审计（2026-09-19，v4.338.0 收官）

> 背景：todos 挂账「GaeaHistory/ContextView 等无参绑定内核会话语义逐个审计」。
> v4.181.0 修 GaeaTrajectory/GaeaAgentNetwork（无参绑定恒读内核 `ga.ctrl`
> 单例=最近活跃会话，UI 切历史会话后旁路看板仍显示内核会话内容）。本审计
> 枚举 `internal/app` 全部 `Gaea*` 导出绑定（App 层+门面），逐个核对 `ga.ctrl`
> 会话态读点与前端消费方，判定是否需要同款参数化。

## 结论：需修 0 项，家族关闭

该应用的会话模型是**「看历史=切内核」**：前端 `currentSessionPath` 的
`current` 标记本身来自 `GaeaListSessions` 读 `c.SessionPath()`；浏览历史会话
唯一入口是 `GaeaResumeSession(path)`（先切内核再拉数据）。因此：

1. **对话主流水线**（Transcript/事件流/仪表）无参读内核会话是**构造性自洽**——
   UI 上不存在「旁路查看另一个会话」的目标。
2. v4.181 修的一族是**带显式 sessionPath 的旁路看板**（轨迹/网络/上下文/子代理
   runs）——面板显示的目标会话可以≠内核会话，必须参数化。普查后**无同构残余**。
3. 提交类动作（Submit/Steer/Approve/Answer/Rewind/Fork/NewSession）天然作用于
   活跃会话，豁免。

## 清单（读内核会话态的无参绑定，全部「保持」）

| 族 | 成员 | 读的会话态 | 判定理由 |
|---|---|---|---|
| 对话主流水线 | GaeaHistory / GaeaResyncEvents | c.History() / ReadEntriesFor(c.SessionPath()) | 恒跟随内核会话（resume 先切内核），无旁路目标 |
| 仪表 | GaeaContext / GaeaJobs / GaeaBalance / GaeaMeta | 活跃会话快照 / 控制器级任务表 | 「正在对话的会话」语义 |
| 会话标识 | GaeaListSessions / GaeaListProjectSessions | c.SessionPath()（仅 Current 标记） | 正是 currentSessionPath 的定义点 |
| 事实底座 | GaeaFactBase / Clear / Promote | PathFor(c.SessionPath()) | 随 live 会话刷新，面板无独立会话目标 |
| 长期记忆 | GaeaMemory / Suggestions / Remember / Forget / UpdateFact / ChangeFactType / SaveDoc / Accept* | c.Memory()（跨会话库） | 长期记忆显式跨会话 |
| 引擎级 | GaeaSkills / Commands / Tools / Capabilities / PermLevel(+Set) / Settings | 引擎清单/权限 | 会话无关 |
| 动作类 | GaeaSubagentFollowUp / GaeaDag* / GaeaCaptureSkill / SkillDistill* / office 编辑绑定 | running 守卫 / 空间注入 / SessionPath 仅 journal 写侧归因 | 动作作用于活跃会话或工作区文件，归因语义正确 |

其余约 150 个无参绑定（模型/空间/用量/路由/任务/收件箱/记忆中枢/语音/成本/
知识/进度计划/Git/备份/搜索等）`ga.ctrl` 零命中，无成员资格。

## 顺手清账（随 v4.338.0 落地）

- **GaeaCheckpoints 死绑定删除**：全仓唯一引用=bridge 类型声明+mock，UI 回退
  菜单实际走 GaeaRewind(turn 号来自实时流)。连带删 Go `CheckpointMeta` 类型、
  门面转发、bridge/mappings/spaceBindings/mock/wire 类型与 wailsjs 生成物
  （`wails generate module` 再生）。绑定面 707→705。
- **GaeaTCCAReport 孤儿链删除**：返回值仅入库 `state.tcca` 且**零渲染消费方**
  （V3.0 缓存指标无处展示）。连带删 controller 两处拉取+reducer/initial/action
  类型、`TCCAReport` 前端接口、mock、三处测试 mock 键与 JSON.parse 错误注入用例
  （该链路的 LogFrontendError 行为随链路一并消失）。绑定面同上并入 705。

## 拍板池（不自动做）

- **GaeaSkillDraftFromSession 参数化**：唯一从内核会话读全量内容做 LLM 加工的
  无参读点（技能草稿蒸馏）。当前由 Composer/MemoryPanel 在 live 会话显式触发，
  内核语义成立；若要「从任意历史会话蒸馏技能」需加可选 sessionPath+入口
  （会话右键「沉淀为技能」），属功能增量，候拍板。

## 复审口径（下次再加绑定时的自检）

新绑定若满足以下全部条件才允许无参读内核会话：①消费方是实时事件流的跟随视图
（无独立会话选择）；②或提交/动作类作用于活跃会话；③或引擎级/工作区级元数据。
带会话切换能力的看板（历史轨迹/回放/对比）一律显式 `sessionPath ...string`
（显式优先、空串回落内核，v4.181/GaeaTaskList 先例）。
