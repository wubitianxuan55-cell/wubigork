# gaea 3.0 板块蓝图 · modelcenter 模型中心

> 覆盖 MASTER。实施参考 docs/2026-08-15-gaea3-ui-design-system.md。

## 现状（facts）
- 页面：`frontend/src/pages/ModelCenterPage.tsx` + pages/modelcenter/*（ModelPanel/StatsSection/
  BenchmarkSection/HerdsmanCatalogSection/RetrievalEvalSection/BindSection + hooks/*）
- 引擎状态联动/搜索收藏置顶/资源占用可视化；12 主题双态

## 目标态
- **视觉性格：控制台**——KPI 卡 + 引擎卡 + 统计面板，密度偏高。
- 分类区：KPI 卡（`--color-surface-container` + 主色数字 + 小字说明）：
  - 本地 vs 云端节省、KV 命中率、磁盘 KPI 等走语义色（成功/警告）。
- 引擎卡：实底卡 + 引擎名 + 状态徽标（运行中=success 脉冲 / 停止=secondary / 错误=destructive）
  + 生命周期按钮（启动/停止/下载/卸载，主色/次级/危险）；
  - 模型用途提示 chips（llm/tts/stt/ocr/embedding/rerank/image 用 ClassifyModelKind 单源分类色）。
- 受控测评 / 检索质量区：表格走 antd table（token 化），通过/失败徽标语义色。
- 统计区：图表达标（走令牌色 + 图例 + tooltip；不只靠颜色）。

## 落地清单
- [ ] KPI 卡/引擎卡令牌化（清硬编码）；状态徽标四态统一
- [ ] 图表色板从主题主色/成功/警告/次要派生
- [ ] 生命周期按钮语义（危险操作 = destructive 确认）

## 验收
- 12 主题下 KPI/徽标对比度正常；图表可用颜色+图例区分；焦点环可见。
