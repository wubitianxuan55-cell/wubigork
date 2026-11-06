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

- **节点产物**（v4.221 按会话归因定案）：子代理证据落账后（SessionID=run.Ref，Journal 按会话分文件），节点 outputs=SessionID==ref 的证据卡 Target——精确归因，主对话/其他节点同期写盘不再并入；ref 为空（ephemeral）回退窗口增量口径。UI 口径注「本节点子代理会话写入的工作区文件」。
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

- 运行中 GaeaSteer 直穿改向 + 危险操作分级审批（roadmap §12.4 后半，等审批闸分级面）。
- 产物自动登记 DeliverableRegistry 侧（现走证据卡侧通道，够用）。
- 壳内真机走查（含真实三件套样例链）挂观察池。

> 进度（2026-09-11，v4.220.0）：**模板库已清**——FromRun 剥运行痕迹存 `.gaea/work/dag/templates/`（Save 前全量 Validate，坏形状拒入库），GaeaDagTemplateNew 一键重建为**草稿** run（不自动起跑，起跑仍人拍板）；绑定 620→624；DagPanel 模板区（折叠条+新建/删）与 run 卡「存模板」内联输入。余项剩上列四条。
> 进度（2026-09-11，v4.221.0）：**波内并行已放开（§6 首条清掉）**——前提升案：真机归因链核查发现**子代理写盘从不落 Journal**（子代理 Options 无 JournalDir/SessionID，flushJournal 直接跳过；v4.219 的窗口归因在真机上恒为空集，fake-runner 测试靠手工塞卡，真机走查恰挂观察池未跑——结构性缺口）。本刀三合一定案：① **子代理证据落账**=TaskTool 增 journalDir（boot 注入，与主执行器同目录）+ Options.SessionID=run.Ref 经 runSubSession 下发（task 工具/RunNew/RunFollowUp 三路径全覆盖），子代理写盘回合收尾落卡、按会话分文件——顺带收口「task 子代理编辑对版本时间线/回滚不可见」的既有审计缺口（6.1 同向）；② **按会话归因**=节点 outputs=SessionID==ref 的卡（ref 空回退窗口增量），主对话同期写盘不再并入；③ **波内并行**=dagExecute 波内 goroutine 并发（归因已精确，不再依赖窗口不重叠），失败级联/终止级联语义不变。Go +3（agent 落账往返/app 并行互等+会话隔离/生命周期测试改模拟真实落账）。**§6 余项剩三条**：运行中 steer 直穿+分级审批/增量改图/DeliverableRegistry。
> 进度（2026-09-11，v4.222.0）：**增量改图已清**——dag_plan 增 `run_id` 可选参=编辑模式：dag.ApplyEdit 纯函数原地调和（节点按 id 对账，title/prompt/dependsOn 全等→保留状态与产物；有变/新增→回 pending；被移除节点仍被依赖=悬空依赖 fail-closed；运行中拒绝；改已验收节点=撤销验收且 RevokedAcc 计数透出文案）。模型改单节点指令不再整链重规划；**零新绑定零前端改动**（沿用既有面板与轮询）。Go +2（dag 调和四态/app 编辑路径+三守卫）。**§6 余项剩两条**：运行中 steer 直穿+分级审批/DeliverableRegistry。

> 进度（2026-09-12，v4.242.0）：**成品直出首刀已清（DeliverableRegistry 用户面，市场调研候选3 防御性第一刀）**——§1 差距=MS Agent Mode/WPS 把「一句话→成品」做到大众可用，gaea 流水线的产物散在各节点证据卡里、逐节点验收=N 次盖章。本刀口径：**成品=done/accepted 节点 outputs（工作区相对路径）的 run 级汇总**，验收语义零变更（仍人拍板，一键=一次拍板覆盖清单所列节点；不自动验收不定时验收——「人拍板回流记忆」红线不动）。**落地**：① Go=提取单节点验收核心为 dagAcceptMemoryWrite（记忆负载单一真源，单验收/一键验收同语义）+ 新绑定 `GaeaDagAcceptAll(id)`（锁内一次翻转全部 done 节点+save 一次；锁外逐节点记忆回写，单条失败不阻断其余、汇总如实上报=与单验收「验收已成立不回滚」同哲学；无可验收节点返回明确信息非错误）；**绑定 626→627**（gen_bindings 再生后 bindings_office 手工补行+还原 v4.237 单参化七行，office 门面不整文件重排）。② 前端=DagPanel run 头 meta 行「M 件成品」计数徽标（title=文件清单，点击走既有 openFilePreview 通道看节点产物 chips）+「一键验收 N」两段式按钮（首击武装=文案变「确认验收 N 节点」，再击执行；运行中/无 done 隐藏；轮询重拉后自动解除武装防陈旧误击）；绑定六处接线（bridge/office 接口+mappings+bindingNames+spaceBindings work 域+锁 448→449）。**测试**=Go +3（一键全收+混合态跳过/记忆回写失败续行汇总/空 done 提示；单验收回归不动）+vitest +2（成品徽标+两段式验收）。**§6 余项剩两条**：运行中 steer 直穿+分级审批/壳内真机走查（DeliverableRegistry 用户面已清；「成品导出打包」若真机反馈需要再立）。
> 进度（2026-09-12，v4.243.0）：**运行中 steer 直穿 + 危险操作分级审批已清（§6 最后一条功能余项）**。①**直穿**=主对话 GaeaSteer 的同机制下沉到子代理：agent 包加在跑登记 `subRunners`（ref=SessionID→AgentRunner，runSubAgentInternal 起跑登记/defer 注销）+ `SteerSubagent(ref,text)`；GaeaDagNodeSteer 放行 running 节点——凭 node.Ref 直穿注入 steer 队列（不打断工具执行，下一回合生效，与主对话同语义），查无在跑登记（恰收跑/ephemeral）如实报错不静默转续跑。②**审批分级**=dag_plan 节点参数 `risk`（normal 缺省/high=覆盖/删除既有文件、批量移动、全局性改动）；Node 加 Risk/Approved；**执行器起跑前置闸**——dagExecute 每节点起跑前 Risk==high && !Approved → 置 StatusHold「高风险待审批」，波收尾链停（下游保持 pending 不级联跳过）；`GaeaDagNodeApprove` hold→pending+Approved=true（**只放行不自动跑**，起跑仍人拍板同闸）；已批准重跑不再挂闸；ApplyEdit 把 Risk 计入形状比较——**风险变更=回 pending+审批归零**（新风险要重新批）；FromRun/Instantiate risk 随模板走、approved 剥净。前端：hold 徽标（var(--warn) 待审批）+「批准」按钮+running 节点直穿改向框（占位文案区分）。绑定 627→628（gen_bindings 再生+office 门面手工补行+checkout 还原单参化，diff 恰 1 行）。测试 Go+4（Validate 风险值/ApplyEdit 风险变更归零审批/FromRun-Instantiate 携带/审批闸端到端=hold+链停+approve 放行+已批准重跑不挂闸；SteerSubagent 寻址三态）+vitest +2（hold 徽标+批准按钮/running 直穿占位）。**§6 余项剩一条**：壳内真机走查（含真实三件套样例链）。
> 进度（2026-09-12，v4.243 壳内真机走查，非版本刀）：**§6 最后一条余项清账——§6 余项全清**。CDP 9333 起壳（.tmp/walk-v4243.mjs/.tmp/walk-dag.mjs DOM 断言配方，不点危险操作不打模型）：①注入体检 Drawer 真跑=GaeaMemoryEvalRun 真实 wire+真实库**端到端打通**（通过横幅+晨报预载 2 条·104/600+本体 2 引用·181/600+门控四开关+toast，截图 .tmp/eval-drawer.png）；②办公流水线区 dag-section 在位（空态引导文案正确，用户暂无 run）；③全程 exceptionThrown=0、页面渲染出错=0。**观察池新增**=办公记忆库面板显示「不可用—未配置」而同库体检读出 2 条活跃——面板 available 旗与体检取数路径（hubOfficeStore 直连）口径不同，属既有语义非本线回归，按需另刀。**DAG 6.3 线功能面+走查全收官。**
