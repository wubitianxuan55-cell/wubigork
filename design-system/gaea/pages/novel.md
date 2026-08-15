# gaea 3.0 板块蓝图 · novel 小说创作

> 覆盖 MASTER。实施参考 docs/2026-08-15-gaea3-ui-design-system.md。
> v3.1（2026-08-15 重构轮）：书架/阅读/控制台/令牌化已落地，见「v3.1 落地记录」。

## 现状（facts）
- 页面：NovelPage + 子页（ChapterPage/CharacterPage/CreatePage/ExportPage/NovelSettingPage）+ 书架/设定/角色/创作/阅读/导出 6 Tab
- 组件：components/novel/*（书架卡、编辑器、角色卡、关系图、AIConsole 等）
- 流式章节创作（create-chapter-stream）+ 停止按钮 + 导出 TXT/MD/EPUB

## 目标态
- **视觉性格：创作工作台**——书架（资产库）+ 编辑器（专注区）+ 角色/设定（知识侧栏）。
- 书架：书籍卡（封面渐变条 + 书名 + 题材标签 + 主色进度条 + 继续阅读主操作）；
  搜索/排序工具条；空态；hover 位移 ≤2px + `--v3-glow-soft`；全令牌化。
- 阅读（ChapterPage）：单条 chrome（收敛原 tab/信息/工具栏三层）；阅读模式 =
  居中限宽 46rem 衬线列（`--v3-edge` 高光容器）；F11 专注模式联动壳层；Esc 逐级退出。
- 编辑器（CreatePage/ChapterPage）：纸面化衬线书写面；
  - 流式生成中：`--gaea-glow` 进度条（`novel-gen-progress-*`）；
  - 停止按钮 = destructive 次级（危险确认语义）。
- 角色卡：档案卡 + 立绘横幅 + 状态/关系 chips（ROLE_COLORS 走令牌化主色/支持色）；
  - 关系图谱（RelationGraph）：节点色挂载时经 getComputedStyle 解析令牌（主题派生）+ 图例。
- AI 控制台：v3 玻璃面板（`ai-console-*` 类），字号 ≥10px，状态色走令牌，FAB 不与轨道条重叠。
- 设定/大纲：列表 + 卡片式条目，折叠分组。

## v3.1 落地记录（2026-08-15）
- [x] 书架重构：HomePage 搜索/排序 + ProjectCardItem 封面渐变条（题材→tone 类）、
      进度条（章节进度 %）、继续阅读主操作（`novel:goto-tab` 联动阅读 tab）、空态/无结果态
- [x] 阅读页重构：ChapterPage 单条 `.novel-chrome` + 阅读模式（居中限宽衬线排版 +
      场景分隔符 + 页脚前后章导航）+ 编辑/阅读分离 + Esc/F11 层级退出
- [x] AIConsole 面板化：`.ai-console-fab/.ai-console-panel/.ai-console-*` 类，
      令牌状态色（novel-tag-tone），条目键盘可达
- [x] 令牌化清理：NovelSettingPage 导入/导出、ChapterEditor 重试条与右键菜单、
      分支色板（usePlotBranch toneColors）、BranchSelector/NextChapter/PlotBranch/
      NewCharacters/OrganizationEdit、NovelInspector 保存状态、EditorPanel 状态与进度条
- [x] 关系图谱令牌派生：RelationGraph 挂载时解析 `--color-*`/`--gaea-glow`（canvas 不解析 var()）
- [x] 生成发光统一：`.novel-gen-progress-track/fill`（`--gaea-glow`）+ 停止按钮语义保留
- [x] 无障碍：新元素进 reduced-motion 降级（书架卡/控制台/进度条）
- [ ] 12 主题浏览器截图抽查（书架/阅读/控制台对比度）

## 验收
- 12 主题下书架/编辑器/图谱对比度正常；生成中发光无跳动；焦点环可见；
- reduced-motion 降级；零硬编码色值（novel 板块范围内）。
