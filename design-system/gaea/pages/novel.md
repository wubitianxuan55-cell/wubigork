# gaea 3.0 板块蓝图 · novel 小说创作

> 覆盖 MASTER。实施参考 docs/2026-08-15-gaea3-ui-design-system.md。

## 现状（facts）
- 页面：NovelPage + 子页（ChapterPage/CharacterPage/CreatePage/ExportPage/NovelSettingPage）+ 书架/设定/角色/创作/阅读/导出 6 Tab
- 组件：components/novel/*（书架卡、编辑器、角色卡、关系图、AIConsole 等）
- 流式章节创作（create-chapter-stream）+ 停止按钮 + 导出 TXT/MD/EPUB

## 目标态
- **视觉性格：创作工作台**——书架（资产库）+ 编辑器（专注区）+ 角色/设定（知识侧栏）。
- 书架：玻璃面板 + 书籍卡（封面图 + 书名 + 进度条走主色）；hover `--elevation-2` + 位移 2px。
- 编辑器（CreatePage/ChapterPage）：居中专注列（max-width）+ 白色/主题表面；
  - 流式生成中：章节文本光标 `cursor-blink` + 底部发光脉冲（`--gaea-glow`）；
  - 停止按钮 = destructive 次级（危险确认语义）。
- 角色卡：档案卡 + 立绘横幅 + 状态/关系 chips（ROLE_COLORS 走令牌化主色/支持色）；
  - 关系图谱（RelationGraph）：节点色从主题主色/成功/警告派生（不只靠颜色）。
- 设定/大纲：列表 + 卡片式条目，折叠分组。

## 落地清单
- [ ] 书架/角色/编辑器令牌化（清硬编码色值）
- [ ] 流式生成态发光脉冲统一 `--gaea-glow`
- [ ] 关系图谱色板主题派生 + 图例

## 验收
- 12 主题下书架/编辑器/图谱对比度正常；生成中发光无跳动；焦点环可见。
