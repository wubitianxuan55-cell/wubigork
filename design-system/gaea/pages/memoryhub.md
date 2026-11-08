# gaea 3.0 板块蓝图 · memoryhub 记忆中枢

## 2026-11-08 · 真 3D 脑网格（three.js）

> - 触发：用户指出「不像左右脑」「放大点击看不到记忆」，明确要求「换真正的 3D 脑网格」。
> - 网格：NIH 3D `3DPX-021161`「Detailed Human Brain Model」（Johnson J，CC BY 4.0）的 X3D，离线解析 **18,511 顶点 / 26,265 三角面**；保留真实脑回、纵裂、颞叶/小脑，不再用 2D 掩膜或 SDF 椭球。
> - 渲染：`MemoryAtlasStage` 改为 **three.js WebGL**——Fresnel 发光脑壳 + 唯一棱线框 + 18.5k 顶点粒子 + 820 条沿网格边随机游走的脑回纤维；左金右紫按真实 x 分半球；3D 平台、轨道环、双色光晕、星点、光柱同一场景。GPU 渲染，1600×1000 rAF ≈57fps；DPR ≤1.5、页面不可见停画。
> - 视角/交互：拖拽 360° 旋转、滚轮缩放、自动慢转、重置回正视；three Raycaster 命中真实 `GraphNode` 精灵（真实 `GraphLink` 连线），拖动阈值区分旋转与点击；默认正视与放大后点击都能打开 `.hub-detail-title`，悬浮 tooltip 跟随。
> - 懒加载：three + 网格数据经 `React.lazy` 独立 chunk（`MemoryAtlasStage`），首屏页面壳不背 3D；GraphView 的 `3d-force-graph` 仍是独立 chunk。显示设置内署名 NIH 3D / Johnson J / CC BY 4.0。
> - 数据诚实：网格/纤维只是脑形氛围层，只有真实 `GraphNode` 成为可点击大光点，链路来自真实 `GraphLink`；图谱失败时保留氛围态，右栏显式报错。
> - 主要文件：`frontend/src/gaea/components/memoryhub/MemoryAtlasStage.tsx`、`brainMeshData.ts`（离线采样网格 + CC BY 署名）、`atlasConfig.ts`、`atlas.css`、`frontend/src/pages/MemoryHubPage.tsx`。
> - 验证：`tsc -b`、改动文件 ESLint、`vitest` MemoryHubPage 5 例 + memoryhub 164 例、`vite build`；CDP `?mock=1` 实拍正视/重置/放大，默认与放大后点击均打开详情。

> 以下 2026-11-07 条目为当时 Canvas 2D 掩膜版记录；中央脑形已由本条的 three.js 真实 3D 网格取代。

## 2026-11-07 · Living Constellation 星图观测台

> - 触发：用户给出「双脑星云 + 全息平台」参考图，要求记忆中枢达到同款 UI 效果。
> - 入口：总览改为全幅活体星图（左索引 / 中双脑 / 右记忆流 / 底控制），库页收进同一观测台的玻璃抽屉，左索引常驻可切换。
> - 星图：`MemoryAtlasStage` 以 CC0 正视脑图（freesvg #152745）离线采样 200×200 的脑形/脑沟掩膜与距离场厚度，Canvas 2D 实时投影约 3 万粒子：左金右紫、脑沟亮线、体内粒子、真实节点亮星与真实关系弧、轨道环、全息平台与光柱。左右半球上方有 HEMISPHERE HUD（事实/知识/办公 vs 自我/画像/聊天），脑形来自真实正视脑图而非椭球近似。
> - 点击修复：画布事件从 viewport 坐标换算成 canvas 局部坐标（`clientX - rect.left`）再做命中测试；此前漏减 rect，点击/悬浮整体偏移，放大后尤其点不中。现在默认视角和滚轮放大后点星都能打开右侧详情，22px 命中半径。
> - 视角：脑形掩膜需要正视轮廓，自动旋转改为 -0.08~0.26 rad 的缓慢摆动（拖拽 ±0.38 rad），保留前后粒子视差与 3D 平台，又不把脑转成侧片。
> - 数据诚实：粒子/脑沟只是脑形氛围层；只有真实 `GraphNode` 才成为可点击亮星，链路来自真实 `GraphLink`。图谱失败时星图停在氛围态，右栏显式报错并重试，不伪造条目。
> - 索引：标题/统计/三脑检索/九库索引/构成条；窄高屏按断点收起构成与装饰段，保证九库全部可达。
> - 记忆流：类型 tabs + 按关联度排序的记忆条目；点条目在星图上点亮对应星辰并展开 Inspector；检索命中沿用 `.hub-hit` 键盘语义与文件预览/@引用。
> - 控制：暂停流览、旋转视角、重置视角、显示设置（轨道环/星尘/自动旋转）；`prefers-reduced-motion`/设置里的减少动态默认冻结星图，仍允许手动继续。
> - 性能：30fps 绘制上限、DPR ≤1.5、页面不可见/容器折叠停画；1600×1000 实测 rAF ≈42fps（脑形粒子抽稀后）。场景几何一次构建，数据变化只重排节点。
> - 响应式：≤1440 / ≤1240 / ≤1040 / ≤900 宽度与 ≤900 / ≤800 / ≤760 高度断点；1366×768、1280×800 实拍无截断。
> - 验证：`tsc -b`、`vite build`、ESLint（改动文件）、`vitest MemoryHubPage` 5 例、GraphView/mock contract 18 例；浏览器实拍总览/检索/命中详情/库页/图谱/离线态/设置面板，默认与放大后的画布点星均打开详情。
> - 主要文件：`frontend/src/pages/MemoryHubPage.tsx`、`frontend/src/gaea/components/memoryhub/MemoryAtlasStage.tsx`、`frontend/src/gaea/components/memoryhub/brainMask.ts`（CC0 脑图采样数据）、`frontend/src/gaea/components/memoryhub/atlas.css`。

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
