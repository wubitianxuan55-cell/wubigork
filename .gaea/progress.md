# 任务进度

> 最后更新: 2026-08-26 22:00（v3.2.1 已发布）

## v3.2.1（2026-08-26 发布）——全部完成 ✅

- ✅ C5 工作区内联编辑（蒸馏候选清单 9 项全部收官）：GaeaWriteFile（绑定面 497，+1）——
  相对路径（拒绝对/..穿越）+ 写根（WriteRoots）+ 文本扩展名白名单 30 种 + ≤2MB +
  仅已存在文件；原子写（临时文件 + fsync + rename，失败保留原文件）；用户显式保存不走审批
- ✅ FilePreview 编辑模式（markdown/text 且未截断才可编辑）：编辑切换 + 脏标记 + Ctrl+S +
  保存状态机（失败可重试）+ 保存后自动重读预览 + 脏退出内联确认条
- ✅ 发布工程：版本四处统一 3.2.1、绑定面 497 漂移 PASS、go 全量测试绿（TestGaeaWriteFile
  五类拒绝 + 拒绝不改动原文件）、vitest 652/652（+5）、tsc/eslint 0 errors、wails build +
  冒烟 /api/health 200；CHANGELOG / releases/v3.2.1.md / README / AGENTS.md / progress.md
  同步；git tag v3.2.1

## 下一阶段候选（v3.2.0 里程碑剩余）

- ⏳ 记忆统一层（路线图 V4）：三脑/多库记忆归并视图 + 统一检索 + 生命周期产品化
- ⏳ 受控自主（goal gate 深化）：目标验收自动追踪产品化、审批流收敛
- ⏳ C9 分栏对照（候选清单建议验证真实需求后再启动）
- ⏳ 质量收敛：eslint 366 存量 warnings 收敛、flaky 治理（filewatch/AV 锁）、
  前端性能复查、releases/README.md 更深历史乱码修复
- ⏳ 造价数据库体验收口：手册二期、测算项目批量导入/导出、分类树维护界面
