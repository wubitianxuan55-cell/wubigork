# 任务进度

> 最后更新: 2026-08-26 21:30（v3.2.0 已发布）

## v3.2.0（2026-08-26 发布）——全部完成 ✅

- ✅ C1 任务实时输出：tasks 包 `Progress.Output(line)` 输出环形缓冲（200 行/64KB 上限 +
  截断标注 + 只回放不消费游标）；三个消费者（价格抓取/批量/语义索引）逐源逐批时间戳输出；
  新绑定 GaeaTaskOutput(taskID) → {tail, truncated}（绑定面 496，+1）；任务中心选中任务 →
  底部输出 dock（2s 轮询 + 尾随滚动 + 截断标注 + 可关闭，终态可复核）
- ✅ C1 结束态细分 stopping：取消 running 先条件置 stopping（WHERE status='running' 防覆盖
  终态竞态）再传播取消，handler 退出终态 cancelled；前端「停止中」琥珀色徽标；重启续跑把
  stopping 一并恢复 queued
- ✅ C2 子代理活动行：summarizeSubagentTranscript 派生 lastText（160 字）/lastTool（工具名+
  结果首行 80 字）随 SubagentRunView 下发；分工面板运行中卡片显示「正在：…」+「⚙ 工具」
  活动线；父子拓扑暂不做（meta 无父子记录）
- ✅ 修复数据备份隔离缺陷：dataBackupPlan 混用 DataRoot 与 MemoryUserDir 系路径（GAEA_DATA_ROOT
  测试隔离时 zip 混入真实用户目录 + 相对路径穿越）；统一从 DataRoot 派生全部条目（生产不变）
- ✅ 发布工程：版本四处统一 3.2.0、绑定面 496 漂移 PASS、go 全量测试绿（tasks +2 / app +1 /
  备份回归）、vitest 647/647（+4）、tsc/eslint 0 errors、wails build + 冒烟 /api/health 200；
  CHANGELOG / releases/v3.2.0.md / README / AGENTS.md / progress.md 同步；git tag v3.2.0

## 下一阶段候选（v3.2.0 后续刀）

- ⏳ 蒸馏收尾：C5 工作区内联编辑（需 GaeaWriteFile 授权面）/ C9 分栏对照（候选清单建议
  验证真实需求后再启动）；C1/C2/C3/C4/C6/C7/C8 已完成
- ⏳ 记忆统一层（路线图 V4）+ 受控自主（goal gate 深化）——v3.2.0 里程碑剩余项
- ⏳ 质量收敛：eslint 360 存量 warnings 收敛、flaky 治理（filewatch 抖动/AV 锁）、
  releases/README.md 更深历史乱码修复、前端性能复查
- ⏳ 造价数据库体验收口：手册二期、测算项目批量导入/导出、分类树维护界面
