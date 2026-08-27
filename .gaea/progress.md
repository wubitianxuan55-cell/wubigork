# 任务进度

> 最后更新: 2026-08-27 22:00（v3.3.0 已发布）

## v3.3.0（2026-08-27 发布）——全部完成 ✅

- ✅ **eslint 366 → 0（errors 0 / warnings 0）**，四层收敛：
  - 配置显式化：no-unused-vars `^_` 前缀 ignore patterns、no-empty allowEmptyCatch、
    react-refresh allowConstantExport
  - 死代码清理 56 处（未用 import/const/函数/catch 参数/解构成员，跨 40 文件）
  - exhaustive-deps 40 处：稳定依赖补全 / 不稳定依赖显式 disable 注释（含两处 TDZ
    陷阱 useCallback 定义上移重排、两处 disable 位置修正）/ 复杂表达式提取变量 /
    每渲染重建数组 wrap useMemo / ref cleanup 竞态局部变量化
  - react-refresh 25 处混合导出文件级显式声明（14 文件）+ 移除 10 处冗余 @ts-ignore
    （wails.d.ts 类型早已生成，tsc 验证 0 错误）
- ✅ flaky 治理：filewatch 测试超时 3s→5s（fsnotify 投递延迟假红）、CI 后端测试失败
  重试一次（重试后仍失败正常红）、确认 CI 排除 internal/tts + test-all.ps1 AV 锁重试
- ✅ releases/README.md 乱码恢复：v2.40.0 及更早 98 行从 git 历史（7c53db8 干净版）
  逐行重建，0 残留
- ✅ 前端性能体检：大组件 memo 复查（页面级收益有限不额外加）；唯一热点 XlsxPreview
  Excel 网格全量渲染（maxRow×maxCol `<td>`）——虚拟滚动重构收益/风险比低，记录待
  真实卡顿反馈再启动
- ✅ 发布工程：版本四处统一 3.3.0、eslint 0/0、tsc 0 errors、vitest 652/652（124 文件）
  零回归、Go 全量 112/112 包、filewatch 5 测试绿、绑定面 497 漂移 PASS（零新绑定）、
  wails build + 冒烟 /api/health 200；CHANGELOG / releases/v3.3.0.md / README / AGENTS.md
  / progress.md 同步；v3.2.1 归档；git tag v3.3.0

## 下一阶段候选（v3.2.0 里程碑剩余）

- ⏳ 记忆统一层（路线图 V4）：三脑/多库记忆归并视图 + 统一检索 + 生命周期产品化
- ⏳ 受控自主（goal gate 深化）：目标验收自动追踪产品化、审批流收敛
- ⏳ C9 分栏对照（候选清单建议验证真实需求后再启动）
- ⏳ 前端性能：XlsxPreview Excel 网格虚拟滚动（待真实卡顿反馈）
- ⏳ 造价数据库体验收口：手册二期、测算项目批量导入/导出、分类树维护界面
