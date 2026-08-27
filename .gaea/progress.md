# 任务进度

> 最后更新: 2026-08-26 21:00（v3.1.1 已发布）

## v3.1.1（2026-08-26 发布）——全部完成 ✅

- ✅ 测算项目 UI（CostProjectsView）：造价板块新增导航——项目列表（状态/条目数/合计/版本数 +
  新建/级联删除）+ 详情（信息编辑 / 工程量清单行内编辑失焦保存 / 金额自动算 / 引用成本库单价
  搜索下拉 / 保存版本快照+查看+恢复 / 沉淀选中行回成本库并刷新概览）；复用 v3.1.0 绑定零后端改动
- ✅ 造价参考 UI（CostIndicatorsView）：按科目/分类分组 + 分位数表格（P25/中位数/均值/P75），
  实时聚合不落表，空态引导
- ✅ 复盘笔记 UI（CostNotesView）：搜索 + 状态过滤 + 编辑弹窗（结论/边界/风险/证据/可信度/
  复核/分类/类型/有效期）+ 引用计数，删除 Modal.confirm
- ✅ 板块导航同步：board cost manifest Nav 增 测算项目/造价参考/复盘笔记
- ✅ C4 选区转对话（SelectionToComposer，纯前端）：办公板选中正文 → 浮动「转为提问」→ 引用
  插入输入框；忽略输入框/弹窗内选区
- ✅ 仓储卫生：删根目录 `.go`/`.split.go`/旧 `gaea.exe`；releases/README.md 补 v3.0.8 行 +
  修 v3.0.1/v3.0.0 乱码
- ✅ 发布工程：版本四处统一 3.1.1、绑定面 495 零新增漂移 PASS、vitest 643/643（+13）、tsc/eslint
  0 errors、Go build/vet + board 测试绿、wails build + 冒烟 /api/health 200；CHANGELOG /
  releases/v3.1.1.md / README / AGENTS.md / progress.md 同步；git tag v3.1.1

## 下一阶段候选（v3.2.0）

- ⏳ 办公蒸馏收尾：C1 任务实时输出（需后端 GaeaTaskOutput）/ C2 子代理活动行（需后端字段）/
  C5 工作区内联编辑（需 GaeaWriteFile）/ C9 分栏对照（C4 已完成）
- ⏳ 造价数据库体验收口：手册二期（其他专业手册导入）、造价参考应用到报价、
  测算项目批量导入/导出、分类树维护界面
- ⏳ 3.2.0 路线图：记忆统一层（V4）、受控自主（goal gate 深化）、板块声明式装配边界（V8）
- ⏳ 质量收敛遗留：eslint 359 存量 warnings 收敛、flaky 治理（AV 锁/filewatch 抖动）、
  releases/README.md 更深历史乱码修复、前端性能复查
