# 办公多文件 DAG 6.3 设计（2026-09）

> 阶段六「书斋纵深：办公主角」最后一把交椅（v4.219.0 起）。上位文档：`gaea-next-stage-plan-2026-09.md` §3 阶段六。路线图对位：`gaea-nextgen-roadmap-2026.md` 办公#3「多文件任务图（DAG）编排」——「读 12 份报表→透视→嵌图表→生成 Word→统一导 PDF」跨格式任务图，每节点单独 steer/重跑/验收；§12.4 反超策略 4 细化（任务中心可视化、运行中改向、危险操作分级审批、本地全程透明）；§16 终止级联与失败可恢复。**本文件是 6.3 施工的域内基线：后续刀在此文档上续写，不另立新档。**

## 1. 问题与目标

委托式完成（办公#1 范式迁移）至今是「单回合单账本」：一回合对话改一批文件，证据链可溯可回滚——但跨格式长链（读 N 报表→透视→嵌图表→出 Word→导 PDF）仍要模型一回合烧完：中途无法插手、一步失败全链重来、产物验收只能等回合结束。阶段六要链条成为一等公民：**规划出任务图、节点分波执行、每节点独立跑/重跑/改向/验收、产物回流记忆，人退验收位**（依赖 5.1 记忆图谱与 6.2 项目本体已就位）。

| 判据（规划 §3 出口判据/办公#3） | 承担者 | 状态 |
|---|---|---|
| 一条 ≥3 节点链全节点可控（跑/重跑/steer/验收） | dag 包存储+App 执行器（gaea_dag.go）+ TasksWorkbench「文件流水线」区 | ✅ v4.219.0（fake-runner 全生命周期锁 Go 测试；壳内真机走查挂观察池） |
| 跨格式任务图（读报表→透视→嵌图表→出 Word→导 PDF） | 节点 prompt 派发既有 office 工具（xlsxedit Plan/Apply、crosslink、ExportDeliverable/ConvertToPdf） | ✅ 同刀（工具面全在，本刀零新增加工工具） |
| 产物回流记忆 | GaeaDagNodeAccept→hubOfficeStore.Save（**验收=人拍板，拍板即写入确认面**） | ✅ 同刀（save 路径自动落 memory_events→5.1 图谱实体） |
| 运行中改向 GaeaSteer + 危险操作分级审批 | 节点 steer=续跑管道（本刀）；运行中直穿改向与审批分级待下刀 | ⬜ 挂 §6 余项 |

## 2. 架构（五层）

```
模型面    dag_plan（builtin 工具，work 空间，PersistWrite 标记防子代理嵌套）
            │ 规划出 goal+nodes[]+dependsOn → 校验（无环/依赖在场/非空）→ 落 run 档
存储面   <cwd>/.gaea/work/dag/<id>.json           —— run/节点状态机真相（workspace 本地，
            │                                       与 journal/rollback 同域；删档即弃不进用户库）
执行面   App gaea_dag.go 执行 goroutine           —— 拓扑分波（波内顺序首刀）→ TaskTool.RunNew
            │   每节点=一次全新子代理会话（headless 闸/过滤工具集/transcripts 落盘，
            │   自动出现在既有子代理 tab；ref 回写节点）
            │ 节点产物=运行窗口前后 Journal List 按 ID 增量 diff，Target 在工作区者
控制面   GaeaDagList / GaeaDagGet / GaeaDagRun / GaeaDagNodeRun / GaeaDagNodeSteer /
            │   GaeaDagNodeAccept / GaeaDagCancel（绑定 613→620，Office 门面委托）
UI 面    TasksWorkbench 新增「文件流水线」区 —— run 卡+节点卡（状态/产物预览/steer 输入框/
             重跑/验收），运行中轮询自校正；?mock=1 可走查
```

选型红线：

- **节点=子代理会话，执行器=既有 TaskTool**：headless 审批闸、FilterRegistry 工具裁剪、transcripts 持久化、`gaea-subagent-text` 文本通道全部白得；不新造 agent 管道，不碰 agent stream() 主循环。
- **波内顺序（首刀）**：产物归因=Journal ID 窗口增量，要求同 run 节点写盘窗口单调不重叠；波内并行等归因切分方案（按 ref 归因）定案后再开，挂 §6。
- **不塞 tasks.Manager**：那是系统 cron 后台任务表（价格抓取/文件索引），重启续跑语义与「人验收的文件流水线」不同构，硬塞只会污染任务中心既有口径。

## 3. 关键口径

- **节点产物**：节点起跑前快照 Journal 最大 ID，收跑后取新增 ChangeRecord、Target 过滤出工作区相对路径=该节点 outputs。主对话同期写盘会并入——UI 口径注明示「运行窗口内新增证据卡」，**不造精确归因**。
- **状态机**：节点 `pending→running→done|failed→（steer 续跑→done|failed）→accepted`；重跑把 accepted 退回 done（重跑即产物可能变，验收失效诚实降级）。run 态=派生：全 accepted=已验收 / 有 running=运行中 / 有 failed=有失败 / 其余=待验收/草稿，不落盘不双写。
- **steer=续跑管道**：GaeaDagNodeSteer 走 RunFollowUp（PrepareContinue 拒 running/mt_/跨空间；followUpClaims 每 ref 单飞防双击）；running 中的节点不可 steer——先 Cancel 再 NodeRun。
- **验收=人拍板**：GaeaDagNodeAccept 落一条办公记忆（name=节点标题归一、description=goal+节点+产物路径清单、tags 含 dag），save 路径自动落 memory_events 成图谱实体——**不在节点完成时静默入记忆**，6.2 项目本体与晨报自然吃到。
- **终止级联**（§16）：GaeaDagCancel 取消 run ctx→在跑节点子代理 ctx 随之取消→未起跑节点置 skipped；run 态=cancelled。
- **fail-closed**：runner 未接线（引擎未构建完）→ GaeaDagRun/NodeRun 显式报错不静默；上游 failed→下游 skipped 不空跑；dag_plan 校验拒绝环/悬空依赖/空链，拒绝即报错不截断。
- **可见性**：节点文本增量走 `gaea-subagent-text` 专用通道（与追问同路），权威状态以 run 档轮询为准，不依赖事件流。

## 4. 消费面

- 绑定 613→620（+7）：GaeaDagList / GaeaDagGet / GaeaDagRun / GaeaDagNodeRun / GaeaDagNodeSteer / GaeaDagNodeAccept / GaeaDagCancel；Office 门面委托同步，bridge mappings+mock 同步。
- 模型工具：`dag_plan`（work 空间；PersistWrite 标记→子代理注册表自动剔除，防嵌套派生）。
- 前端：TasksWorkbench「文件流水线」区（testid 前缀 `dag-`），节点产物点击走 openFilePreview 复用预览面。
- Go API 已备未出绑定：`TaskTool.RunNew`（导出的全新子代理入口，headless 场景复用）。

## 5. 测试与门禁

- Go：dag 包（校验四态：环/悬空依赖/空链/正常；拓扑分波确定性；run 态派生）+ app 层（fake runner 3 节点链全生命周期：起跑→产物归因→steer 续跑→重跑降验收→验收落记忆；Cancel 级联 skipped；上游失败下游 skipped；runner 未接线 fail-closed）。
- 前端：DagPanel（空态/渲染/steer/重跑/验收交互）+ bindings mock 同步既有用例零变更。
- 门禁：Go 全量 0 FAIL / drift PASS@620 / tsc 0 / vitest 全绿 / 版本三处 sync 4.219.0。

## 6. 余项与后续刀

- 波内并行（按 ref 归因切分证据卡窗口后放开；波间依赖分波已就绪）。
- 运行中 GaeaSteer 直穿改向 + 危险操作分级审批（roadmap §12.4 后半，等审批闸分级面）。
- dag_plan 的增量改图（首刀只整链重建；改单节点指令须整链重新规划）。
- 产物自动登记 DeliverableRegistry 侧（现走证据卡侧通道，够用）。
- 壳内真机走查（含真实三件套样例链）挂观察池。

> 进度（2026-09-11，v4.220.0）：**模板库已清**——FromRun 剥运行痕迹存 `.gaea/work/dag/templates/`（Save 前全量 Validate，坏形状拒入库），GaeaDagTemplateNew 一键重建为**草稿** run（不自动起跑，起跑仍人拍板）；绑定 620→624；DagPanel 模板区（折叠条+新建/删）与 run 卡「存模板」内联输入。余项剩上列四条。
