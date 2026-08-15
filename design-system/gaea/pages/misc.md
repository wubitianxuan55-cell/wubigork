# gaea 3.0 板块蓝图 · characterlib 角色库 / settings 设置 / knowledge 知识库 / weixin 微信

> 覆盖 MASTER。实施参考 docs/2026-08-15-gaea3-ui-design-system.md。

## characterlib 角色库
- **视觉性格：档案库**——档案卡网格 + 详情抽屉。
- 现状：CharacterLibraryPage + CharacterCard/CharacterLibEditor（立绘横幅/随机补齐/生成剧照/可聊天开关）；
  角色库联动聊天/小说（单向引用）。
- 目标：档案卡（立绘 + 名称 + 类型 chips + 状态），hover `--elevation-2` + 位移 2px；
  详情抽屉（`--radius-lg` 玻璃 + 分区表单）；「可聊天」徽标用主色；删除 = destructive 确认。
- 落地：卡片/抽屉/徽标令牌化；焦点环。

## settings 设置
- **视觉性格：克制工具**——分组导航 + 表单，信息密度低、可扫读。
- 现状：SettingsPage + components/settings/*（通用/聊天/小说/绘梦/办公/模型/安全/数据/关于 9 分组，
  左侧导航 + 全局搜索）。
- 目标：左侧分组导航（激活 = primary-container + 左缘条）；表单走令牌 + 显式标签 + 错误就近；
  数据备份/恢复（危险操作 destructive 确认）；关于 = 品牌卡 + 版本信息。
- 落地：分组导航激活态统一；表单 focus 环；危险操作确认色。

## knowledge 知识库（3.0 D7 板块）
- **视觉性格：文档库**——面板 + 导入/检索/预览。
- 现状：KnowledgePage（KnowledgePanel variant="page"）+ 导入 Modal + 预览；已注册 PageRegistry + manifest。
- 目标：面板玻璃化 + 分类树（激活 primary-container）+ 条目卡（名称/摘要/来源）；导入走确认流程；
  预览抽屉玻璃化。落地：面板/条目令牌化 + 焦点环。

## weixin 微信助手（服务型板块，无页面）
- 无前端页面（manifest page=""）；仅扫码绑定提示。保持现状，无 UI 落地项。
