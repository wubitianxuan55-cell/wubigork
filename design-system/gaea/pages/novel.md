# gaea 3.0 板块蓝图 · novel 小说创作

> 覆盖 MASTER。实施参考 docs/2026-08-15-gaea3-ui-design-system.md。
> v3.1（2026-08-15）：书架/阅读/控制台/令牌化。
> v4 书房工坊：身份头栏 + 按页显隐分区 + 书封物化书架。

## 现状（facts）
- 页面：NovelPage + 子页（ChapterPage/CharacterPage/CreatePage/NovelSettingPage）+ 书架/设定/角色/创作/阅读 5 模式
- 组件：components/novel/*（书架卡、编辑器、角色卡、关系图、AIConsole 等）
- 流式章节创作（create-chapter-stream）+ 停止按钮 + 导出 TXT/MD/EPUB

## 目标态
- **视觉性格：书房工坊**——书架是画廊，阅读是纸面，创作是三栏工作台。
- 外壳：身份头栏（书名 + 字数）+ 居中模式轨 + 封面动作；不把生成封面塞进主导航。
- 按页显隐：书架/设定/角色全幅；阅读保留目录、收起属性检查器；创作用自带三栏。
- 书架：竖版书封 + 书脊 + 题材色带；「正在编辑」横条；搜索/排序；空态。
- 阅读（ChapterPage）：单条 `.novel-chrome`；居中衬线列；F11 专注；Esc 逐级退出。
- 编辑器：纸面化衬线；流式 `--gaea-glow` 进度条；停止 = destructive 次级。
- 角色卡：档案卡 + 立绘 + 状态 chips；关系图谱走令牌色。
- AI 控制台：玻璃面板，FAB ≥40px，不与轨道条重叠。

## 书房工坊落地
- [x] 身份头栏 `.novel-atelier-bar`（身份 / 模式轨 / 封面）
- [x] 按 `data-novel-tab` 显隐外壳分区（子选择器，不误伤创作页内部分隔条）
- [x] 书架画廊头 + 正在编辑横条 + 竖版书封（v4.205：JSX 接到已有 CSS）
- [x] 目录侧栏不再展示原始路径；已开书不重复身份卡
- [x] 创作页工具条抽成 `.novel-create-rail`，与三栏解耦
- [x] 阅读页属性检查器可折叠（默认收起，章节体检入口可见）
- [ ] 12 主题浏览器截图抽查（书架/阅读/控制台对比度）

## 验收
- 12 主题下书架/编辑器/图谱对比度正常；生成中发光无跳动；焦点环可见；
- reduced-motion 降级；零硬编码色值（novel 板块范围内，阅读主题色除外）。
- 事件契约不变：`novel:goto-tab` / `novel:open-chapter` / `novel:chapter-active` / `novel:focus-mode`。
