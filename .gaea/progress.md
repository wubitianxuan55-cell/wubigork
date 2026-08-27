# 任务进度

> 最后更新: 2026-08-26 20:30（v3.1.0 已发布）

## v3.1.0（2026-08-26 发布）——全部完成 ✅

- ✅ 造价数据库一级板块（board cost，CostLibraryPage）：成本库从记忆中枢二级分类提升为一级导航；
  记忆中枢成本二级入口移除（图谱节点保留琥珀色）
- ✅ 综合单价架构（用户定调「数据库就是数据库」）：综合单价=一级、人材机=二级组成；
  SchemaV12/V13（人工/材料/机械合计 + 管理/利润/垫资/税率仅展示 + cost_entry_components 组成行表）
- ✅ 价格三要素与溯源 SchemaV9（region/price_date/price_type/valid_until/source_row + history 同步）
- ✅ 默认分类树重构：综合单价 → 专业（道路/交通/绿化/电力/给水/暖气/雨污/照明/其他）→ 分部
- ✅ 《市政成本测算手册》整本导入：综合单价分析表头专有解析 + 人材机组成行文本解析；实测 8 表 234 条全命中
- ✅ 测算项目与沉淀闭环（costproject 包 + SchemaV10 三表）：容器/明细/不可变版本 + 沉淀 UPSERT 回成本库
- ✅ 造价参考与复盘笔记（costref 包）：分位数指标实时聚合（不落表）+ 复盘笔记 + cost_indicators 办公工具
- ✅ 成本库数据自愈（cost/repair.go）：201 条非法 category_path 映射回合法路径 + 地区/期数回填，Store.Open 自动执行
- ✅ 造价数据库面板重设计：库规模 hero + 人材机构成占比 + 数据健康 + 空库引导 + 骨架屏
- ✅ 办公蒸馏两轮（2026-08-20/26）：C3 会话级右侧面板持久化 / C6 运行域活动角标 / C7 预览队列 chip 化；
  FileTree → 资源管理器（@引用/右键菜单/cwd 持久化/树内搜索）；删除完成轮大过程卡；GoalCard/TodoCard 紧凑化；
  Tailwind v4 max-w-(--maxw) 括号语法修复
- ✅ 成本库入口接线（用户决策 2026-08-26）：办公右侧「文件」组新增「成本库」子 Tab（CostLibraryPanel），4 主 Tab 收敛不变
- ✅ 修复办公板块初始化死锁（GaeaInit 非重入锁二次加锁；syncGoalForSession 显式接收控制器 + 回归测试）
- ✅ V4.0 dsh化 验证失败正式废弃（删除工作空间内 V4.0 文档；C:\AI\gaea-v4 与 ~/.dsh* 不动）；继续 V3 迭代
- ✅ 发布工程：版本四处统一 3.1.0、绑定面 495 方法漂移 PASS、go 全量测试绿、vitest 630/630、wails build + 冒烟 /api/health 200；
  CHANGELOG / releases/v3.1.0.md / README / AGENTS.md / progress.md 同步；git tag v3.1.0

## 下一阶段候选（v3.1.1 / v3.2.0）

- ⏳ 办公蒸馏后续轮次：C1 任务实时输出（需后端 GaeaTaskOutput）/ C2 子代理活动行（需后端字段）/
  C4 选区转对话 / C5 工作区内联编辑（需 GaeaWriteFile）/ C9 分栏对照（见蒸馏候选清单）
- ⏳ 造价数据库体验收口：测算项目页 UI 打磨（版本对比/恢复交互）、造价参考应用到报价、
  手册二期（其他专业手册导入）、分类树维护界面
- ⏳ 3.1.0 板块生态·记忆起步：记忆统一层（3.2.0 规划 V4）、受控自主（3.2.0）、板块插件化边界（V8 声明式装配）
- ⏳ 质量收敛遗留：eslint 存量 warnings 收敛、flaky 测试基线治理（AV 锁/filewatch 抖动）、
  前端性能（Sidebar 虚拟滚动已做，其余大组件 memo 复查）
