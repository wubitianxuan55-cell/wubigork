# gaea 3.0 板块蓝图 · memoryhub 记忆中枢

> 覆盖 MASTER。实施参考 docs/2026-08-15-gaea3-ui-design-system.md。

## 现状（facts）
- 页面：MemoryHubPage + components/memoryhub/*（KnowledgePanel/CostLibraryView/ProfileLibrary/
  OfficeMemoryLibrary/MaterialsLibrary/WhisperMemoryLibrary/GraphView/DigitalLifeLibrary）
- 8 库聚合：知识/成本/画像/办公记忆/项目资料/聊天记忆/记忆图谱/数字生命；分类 Tab 导航
- 图谱：3d-force-graph（GraphView）；检索：统一检索入口（关键词+语义两组）

## 目标态
- **视觉性格：知识/图谱**——数据密集但信息分层；图谱是视觉主角。
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
- [ ] Tab/列表行/KPI 令牌化；图谱节点色板主题派生 + 图例
- [ ] 详情 Modal 玻璃化；检索命中高亮统一

## 验收
- 12 主题下图谱节点/边可区分（色+图例+悬浮标签）；列表焦点环可见。
