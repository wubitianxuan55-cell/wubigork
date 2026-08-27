# 任务进度

> 最后更新: 2026-08-27 22:45（v3.4.0 已发布）

## v3.4.0（2026-08-27 发布）——全部完成 ✅

- ✅ **统一检索后端收口**：`GaeaUnifiedSearch` 视图扩展四组——keyword（工作区全文）+
  semantic（跨库语义）+ **brain（三脑命中，新增；a.brain==nil 时空数组不报错）** +
  **files（文件语义，新增；复用 GaeaFileSemanticSearch 抽出的私有实现）**；hub 搜索
  （MemoryHubPage.runSearch）由「4 绑定 Promise.all 前端拼装」（BrainSearch +
  WorkspaceSearch + SemanticSearch + FileSemanticSearch）收敛为「单次 app.UnifiedSearch」，
  四组映射回原 HubSearchHit 渲染（徽标/预览/@ 引用零变化）；WorkspaceSearchPanel 跨库
  模式零改动，kindChip 补 file kind（后端本就返回，前端类型漏声明）。
- ✅ **归档 tab 永远空白（缺陷修复）**：前端归档 tab 读 `view.archives`，但后端
  `GaeaMemory()` 的 MemoryView 结构体没有 archives 字段 → 列表永远空白；改
  `GaeaMemoryArchivedList` 分页加载（每页 50 + 加载更多 + total 展示）。
- ✅ **恢复能力补齐（Unarchive）**：memory 包补双后端 `Store.Unarchive`（sqlite 置
  archived=0 + updated_at；file 从 `.archive/<ts>-<name>.md` 移回主目录 + reindex；
  未归档/已硬删报错）+ 新绑定 `GaeaMemoryUnarchive`（绑定面 497→**498**）+ 归档 tab
  「恢复」按钮（Rollback 图标/恢复中态/成功后刷新提示）。
- ✅ **保留期下发展示**：`MemoryArchivedPage` 增 `RetentionDays`（= ArchivedRetention
  90 天），归档 tab 顶部「归档保留 N 天，超期可清理」，清理确认弹窗文案跟随真实保留期。
- ✅ **修复漂移脚本单条差异静默放行 bug**：`check-bindings-drift.ps1` 判 `$diff.Count -gt 0`
  但 PS 5.1 下单条差异 `$diff` 是单个 PSCustomObject（无 .Count）→ `$null -gt 0` 为 False
  静默放行（实测复现：新增 GaeaMemoryUnarchive 后脚本仍报 OK）；`@()` 强制数组化修复 +
  负向验证（单条漂移现在 exit 1）+ 脚本恢复 UTF-8 带 BOM（AGENTS.md 编码规范）。
- ✅ **验证**：Go 全量 **112/112 包**（+6 测试）、eslint **0/0**、tsc 0 errors、vitest
  **654/654（124 文件，+2）**、绑定面 **498 方法**漂移 PASS（含负向验证）、版本四处统一
  3.4.0、wails build + 冒烟 /api/health 200；v3.3.0 资产归档 releases/archive/；
  CHANGELOG / releases/v3.4.0.md / README / AGENTS.md / progress.md 同步；git tag v3.4.0。

## 下一阶段候选（v3.2.0 里程碑剩余 + 记忆统一层后续）

- ⏳ 记忆统一层后续：生命周期产品化收尾（归档保留期可配置/批量恢复）、统一检索
  持续深化（三脑写入路径/跨库事实底座）
- ⏳ 受控自主（goal gate 深化）：目标验收自动追踪产品化、审批流收敛
- ⏳ C9 分栏对照（候选清单建议验证真实需求后再启动）
- ⏳ 前端性能：XlsxPreview Excel 网格虚拟滚动（待真实卡顿反馈）
- ⏳ 造价数据库体验收口：手册二期、测算项目批量导入/导出、分类树维护界面
