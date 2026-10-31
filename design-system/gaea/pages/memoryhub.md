# gaea 3.0 板块蓝图 · memoryhub 记忆中枢

## 2026-10-31 · Memory Atlas 重设计

> - 触发：用户要求“使用技能重新设计记忆中枢 UI”，随后确认“书房/庭院是过时的东西了”。
> - 流程：按 `redesign-existing-projects` 完成 Scan → Diagnose → Fix；保留既有 React / Ant Design / CSS 令牌栈与全部数据绑定。
> - 入口：品牌化头部合并本地索引状态、三脑检索、工位/乐园/全部范围与最近更新；旧 40px 工具条退役。
> - 索引：左轨按“记忆集合 / 探索与存档”分组，领域色只承担知识类型识别，主体仍使用单一主题强调色。
> - 主区：总览与完整图谱共享三栏关系舞台；空记忆、加载、连接失败、重试均有独立状态；完整图谱筛选随事实源收敛，窄窗口使用两行工具条。
> - Inspector：未选中条目时展示索引总数、五库分布、检索范围和最近更新，库行可直接进入对应视图；选中节点/命中后切换为详情。
> - 主题：3D WebGL 画布改为透明，背景统一由 `hub.css` 使用主题令牌绘制，避免亮暗切换残影；领域节点色仍由图谱语义单一来源提供。
> - 术语：检索范围与全局空间入口统一为“工位 / 乐园”；“书房 / 庭院 / 书斋 / 闲庭”在这些入口退役。
> - 响应式：1092px 三栏、≤980px Inspector 浮层、≤820px 纵向布局；键盘焦点、按下反馈、reduced-motion 全覆盖。
> - 验证：`npm run build`、ESLint、8 个测试文件 30 例通过；本地浏览器检查亮暗主题、检索失败、空态和库页切换。Vite 预览未连接 Go 内核时，图谱按连接失败态诚实展示。
> - 主要文件：`frontend/src/pages/MemoryHubPage.tsx`、`frontend/src/gaea/components/memoryhub/GraphView.tsx`、`frontend/src/gaea/components/memoryhub/hub.css`、`frontend/src/gaea/lib/useSpaceScope.ts`。

> 覆盖 MASTER。实施参考 docs/2026-08-15-gaea3-ui-design-system.md。

## 现状（facts）
- 页面：MemoryHubPage + components/memoryhub/*（KnowledgePanel/CostLibraryView/ProfileLibrary/
  OfficeMemoryLibrary/MaterialsLibrary/WhisperMemoryLibrary/GraphView/DigitalLifeLibrary）
- 8 库聚合：知识/成本/画像/办公记忆/项目资料/聊天记忆/记忆图谱/数字生命；分类 Tab 导航
- 图谱：3d-force-graph（GraphView）；检索：统一检索入口（关键词+语义两组）

## 目标态
- **视觉性格：Memory Atlas**——数据密集但信息分层；图谱是视觉主角，入口、索引与 Inspector 共同解释“记忆如何组织”。
- 分类 Tab：antd tabs 主色 ink-bar + 玻璃容器；
  - 各库列表：实底卡行（名称 + 摘要 + 时间戳小字），hover `--color-surface-container-high`；
  - 库级 KPI 条（条数/分类）走主色数字。
- 记忆图谱（GraphView）：
  - 节点色按类型（成本/知识/办公/画像）从主题色板派生 + 图例；
  - 节点 hover 高亮 + 详情 Modal（`--radius-xl` 玻璃）；
  - 三元组/关系线用 `--color-border` + 主色强调。
- 检索：单框两组（关键词/语义）切换，结果行命中高亮 `--color-primary-container`。
- 数字生命：只读库，卡片式（角色/关系/时间线），克制呈现。

## 落地清单
- [x] Tab/列表行/KPI 令牌化；图谱节点色板主题派生 + 图例
- [x] 详情 Modal 玻璃化；检索命中高亮统一
- [x] 总览升级为图谱主导的 Memory Atlas；补齐加载/空/失败/重试状态
- [x] 3D 画布透明化，亮暗主题背景单源
- [x] Inspector 空态升级为可操作库分布
- [x] 空间术语统一为工位/乐园

## 验收
- 12 主题下图谱节点/边可区分（色+图例+悬浮标签）；列表焦点环可见。
