# gaea 前端 UI 全景调研（只读）— 为 3.0 设计语言统一与全板块 UI 落地提供事实依据

> 域：前端 UI（页面/组件/布局/样式/主题/antd 定制/品牌）。只读调研，未改任何文件。
> 证据格式：`文件路径:行号`。相关设计文档：`docs/2026-08-15-gaea3-ui-design-system.md`（定稿 v1）、
> `design-system/gaea/MASTER.md`、`design-system/gaea/pages/*.md`（8 页）。
> 壳层/页面/办公三份既有档案：`docs/gaea3-review/01-frontend-shell.md`、`02-frontend-pages.md`、`03-office-frontend.md`（本报告与 01/02 无重叠，聚焦视觉层）。

---

## 1. 页面全景（frontend/src/pages/）

| 页面 | 路径 : 行数 | 主要子组件（路径） | 顶部布局结构 |
|---|---|---|---|
| 首页启动器 | `components/ModuleLauncher.tsx` : 290 | `boards/launcher.ts`、`components/VoiceChatOrb.tsx`、`hooks/useVoiceChat.ts` | 顶栏状态条 + 左/中/右三栏（左 3 卡｜正中语音粒子球 + HUD 环 + 均衡条｜右 4 卡），`ml-*` 类 |
| 聊天 | `pages/ChatPage.tsx` : 407 | 左 `components/ChatTopicSidebar.tsx`；主区 `components/chat/{ChatModeBar,ChatPersonaBar,MessageList,WelcomeScreen,ChatComposer}.tsx`、`MarkdownContent`、`FeatureModelBar`（左下浮动） | 左会话栏 + 中对话流（full 布局无 padding），`chat-board.css` |
| 小说（壳） | `pages/NovelPage.tsx` : 86 | 顶部 6 子 tab（书架/设定/角色/创作/阅读/导出）keep-alive 挂载 `HomePage/NovelSettingPage/CharacterPage/CreatePage/ChapterPage/ExportPage` + 右侧 `components/novel/AIConsole.tsx` | 顶部二级导航（`.novel-subnav`）+ 内容 + 右 AI 控制台，`novel-workspace.css` |
| 小说·书架 | `pages/HomePage.tsx` : 200 | `components/WelcomePage.tsx`、`components/novel/CreateNovelModal.tsx`、`components/ProjectCardItem.tsx` | 项目卡网格 |
| 小说·阅读 | `pages/ChapterPage.tsx` : 288 | `components/novel/ChapterEditor.tsx`、`components/novel/OutlinePanel.tsx`、`components/TTSPlayer.tsx` | 左大纲 + 右编辑器 + 状态条 |
| 小说·角色 | `pages/CharacterPage.tsx` : 783 | `components/RelationGraph.tsx`、`components/novel/character/*`（Card/Org/Modal/Lightbox）、`components/characterlib/PortraitImg.tsx` | 顶过滤条 + 角色列（`.char-detail-col`），`character-page.css` |
| 小说·创作 | `pages/CreatePage.tsx` : 288 | `components/novel/create/*`（ChapterTreePanel/EditorPanel/CreateInspector/NewCharactersModal/BranchWizardModal） | 左章节树 + 中编辑器 + 右检查器（`.novel-workspace`） |
| 小说·设定/导出 | `pages/NovelSettingPage.tsx` : 240 / `pages/ExportPage.tsx` : 94 | `components/ChatPanel.tsx`、`MarkdownContent` | 左编辑 + 右 380px 面板（`.novel-panel`） |
| 绘梦 | `pages/ImageGenPage.tsx` : 333 | `components/imagegen/{ControlPanel,ResultStage,GenerationBar,TaskCenter,HistoryRail,TemplatePickerModal,CustomTemplateModal,ui}.tsx`、`components/Lightbox.tsx`、`hooks/useImageGen*` | 左控制台 + 中画布 + 下任务中心，`imagegen.css` |
| 办公 | `pages/GaeaPage.tsx` : 25（薄壳） | 整页 = `gaea/App.tsx` : 920 + `gaea/components/*`（Sidebar 41KB/Transcript 31KB/Composer 19KB 等 65 个） | 自有三栏：左 Sidebar｜中 chat-pane（header+Transcript+Composer+TodoPanel）｜右 WorkspacePanel/可拖预览，`gaea/styles.css`+`tailwind.css`+`redesign.css` |
| 记忆中枢 | `pages/MemoryHubPage.tsx` : 325 | home 三脑总览 + 8 库 tab：`gaea/components/KnowledgePanel.tsx`、`memoryhub/{CostLibraryView,ProfileLibrary,OfficeMemoryLibrary,MaterialsLibrary,WhisperMemoryLibrary,GraphView,DigitalLifeLibrary}.tsx` | 顶标题+检索 + tab 切换，`gaea/styles.css`+`hub.css` |
| 模型中心 | `pages/ModelCenterPage.tsx` : 258 | `pages/modelcenter/` 13 文件：`{Overview,LLM,Image,Voice,Specialty,HerdsmanCatalog,Benchmark,RetrievalEval,Bind,Engine,Scheduling,Stats}Section.tsx`、`charts.tsx`、`ResourceMonitor.tsx`、`ui.tsx`、`utils.tsx` | 顶 `.mc-header` + 10 tab 平铺，`modelcenter.css` |
| 角色库 | `pages/CharacterLibraryPage.tsx` : 371 | `components/characterlib/{CharacterCard,CharacterLibEditor,CharacterMemoryModal}.tsx` + `characterlib/*.css` ×3 | 顶筛选 + 角色卡网格 |
| 设置 | `pages/SettingsPage.tsx` : 178 | 9 分类面板 `components/settings/*`（Appearance/Chat/Workspace/ImageGen/Office/Model/Security/Data/About）+ `SettingField`/`SettingsSection` | 顶 header+搜索 + 磁贴平铺（`settings-tiles`），`settings-page.css` |
| 知识库 | `pages/KnowledgePage.tsx` : 29 | `gaea/components/KnowledgePanel.tsx`（variant=page）+ `FeatureModelBar` | 全屏面板 + 左下浮动模型卡 |

## 2. 组件资产（frontend/src/components/）

- **顶层**：AppBar（**死代码**，无任何 import）、ChatMarkdown、ChatPanel、ChatTopicSidebar、CompanionAvatar、EmotionStarMap、ErrorBoundary、FeatureModelBar、Lightbox、MarkdownContent、MobileSheet（**死代码**）、ModuleLauncher、ParticleFlow、PersonaPicker、ProjectCardItem、RelationGraph、SearchModal、SecurityBanner、SettingField、SkillModal、SoundWaveOverlay、TisorRadar、TTSPlayer、VoiceChatOrb、VoiceSettingsPanel、WelcomePage、WhisperDesire/Emotion/MemoryList/MemoryModal/TracePanel（5 个轻语组件）。
- **子目录**：`chat/`（6）、`characterlib/`（4+3css）、`imagegen/`（7+1css）、`novel/`（AIConsole/ChapterEditor/OutlinePanel/CreateNovelModal/NextChapterModal/PlotBranchModal/BranchSelectorPanel + `novel/create/`×7 + `novel/editor/`×3 + `novel/character/`×6）、`office/`（2）、`settings/`（11）。
- **gaea/components/**（65 个，办公自成一系）：Markdown、Transcript、Toast（自定义非 antd）、FilePreview、FilePreviewModal、XlsxPreview、DocxPreview、Sidebar、Composer（+composer/×7）、Message、Welcome、Skeleton、ToolCard/ToolGroup/tool_icons、ProcessCard、TaskCenter、StatsPanel、HistoryPanel、MemoryPanel、KnowledgePanel、CapabilitiesPanel、CommandPalette、TodoPanel、ChangesPanel 等。
- **通用可复用（跨板块候选）**：MarkdownContent、Lightbox、FeatureModelBar（唯一跨板块共享浮件）、SearchModal、ErrorBoundary、Toast（gaea）、ProcessCard（gaea）、FilePreview（gaea）、Transcript（gaea）。
- **重复实现**：Markdown 渲染 ×4（`components/MarkdownContent.tsx`、`components/ChatMarkdown.tsx`、`gaea/components/Markdown.tsx`、`gaea/components/MemoMarkdown.tsx`）；SuggestionCard ×2（`chat/SuggestionCard.tsx` 与 `gaea/components/SuggestionCard.tsx`）；TaskCenter ×2（`imagegen/TaskCenter.tsx` 与 `gaea/components/TaskCenter.tsx`）；欢迎屏 ×3（`chat/WelcomeScreen.tsx`、`WelcomePage.tsx`、`gaea/components/Welcome.tsx`）。

## 3. 布局与壳层

- **`layouts/MainLayout.tsx`** : 480。antd Layout 纵向：`.gaea-bg` 氛围层（cyber-grid/star-dots/aurora-orb×2/space-dust，index.css:429-526）→ Header 48px（favicon+gaea 品牌字、横向 Menu、6 主题色点、明暗切换、搜索、设置、模型/登录 Tag）→ 面包屑（projectOpen 且非锚点页，anchor=novel）→ Content（按 `manifest.layout`：chat/gaea=full 零 padding，其余 padded）→ Footer 32px StatusBar（引擎监控 3s 轮询 + CPU/GPU/内存 + 全书进度，MainLayout.tsx:100-203）。快捷键 Ctrl+1~4 + Ctrl+N（MainLayout.tsx:272-291）。页面 keep-alive：visitedPages 保留已访问页 DOM（:449-459）。导航完全由 `boards/manifests.ts`（10 板块 manifest）派生。
- **首页启动器 `ModuleLauncher.tsx`**：ml-top 状态条（GAEA CORE + 模型 chip）→ ml-main 三栏：左卡片列（前 3 模块）｜正中语音晶核（VoiceChatOrb + 3 雷达环 + 9 均衡条 + HUD 遥测 + 气泡 + 启停按钮）｜右卡片列（后 4 模块）；8 模块清单由 `boards/launcher.ts` 派生。
- **办公工作台 `gaea/App.tsx`**：独立壳层——Sidebar（会话树/项目分组/记忆入口，`gaea/components/Sidebar.tsx` : 41KB）｜chat-pane（顶栏 ModelSwitcher+上下文条+导出+专注模式 / Transcript 主区 / Composer+TodoPanel 底）｜右侧可拖宽 WorkspacePanel/FilePreview（Codex 式，redesign.css:1-17）。自带 CommandPalette/JumpBar/ResizableDrawer/Toast。
- 死代码：`components/AppBar.tsx`、`components/MobileSheet.tsx` 无引用；`App.css` 为 91B 空注释且无人 import。

## 4. 样式体系

**4.1 index.css（909 行 / 31.6KB）— 关键事实：自身不定义任何 CSS 变量**，全部 `--md-sys-*`/`--gaea-*`/legacy shim 由 `App.tsx:44-130` 在运行时用 JS 写入 `:root`（`getThemeTokens` → 30+ 变量）。消费的变量命名空间（全 css 汇总）：
- `--md-sys-*`（M3 本体，约 30 个：color-primary/on-primary/primary-container/surface/surface-container(-high/-highest)/surface-dim/variant/outline(-variant)/elevation-1..5/radius-sm..xl/transition-fast..slow/error）
- `--gaea-glow` / `--gaea-glass-bg` / `--gaea-aurora-bg`（未来感扩展）
- legacy shims（App.tsx 兼容层）：`--color-*`（primary/text/text-secondary/border/success/warning/bg-container/bg-layout）、`--bg-*`（glass/elevated/deep/base）、`--border-subtle`、`--shadow-*`、`--radius-*`、`--transition-*`、`--accent-rgb`
- 组件私有：`--mc-*`（modelcenter，125 处）、`--ds-*`（gaea，27 处但**仅 2 处被定义**，见问题清单）、`--sidebar-*`、`--hl-*`（语法高亮）、`--whisper-*`、`--eq-*`、`--ccard-i/--ml-i/--char-side/--engine-color/--hub-color/--preview-width/--workspace-width` 等

主要 utility 类：`.md-surface(-container[-high|-highest])`、`.md-elevation-1..5`、`.md-card`、`.md-ripple`、`.md-glass`/`.md-glass-strong`、`.neon-card`/`.neon-glow-text`/`.neon-glow-icon`、`.void-card`、`.gaea-bg`（背景层）、`.launcher-card`、`.typing-dots`、`.btn-primary-glow`、`.chat-*`、`.img-card`、`.page-transition-enter`、`.ui-compact`、`.ui-reduced-motion`。另有大量 WebView2 rAF 降级规则（`html.gaea-raf-degraded`）与 antd 弹层动画永久禁用（index.css:736-848）。没有任务描述提到的 `.glass/.card/.btn` 通用类（对应物为 md-glass/md-card/btn-primary-glow）。

**4.2 CSS 文件分布（16 个）与 import 映射**：
| css（路径） | 字节 | 由谁 import |
|---|---|---|
| `src/index.css` | 31597 | `main.tsx:3`（全局唯一） |
| `src/gaea/styles.css` | 41312 | `pages/GaeaPage/KnowledgePage/MemoryHubPage` |
| `src/gaea/tailwind.css`（真 Tailwind v4 `@import "tailwindcss"` + `@theme`） | 9755 | 同上 |
| `src/gaea/redesign.css`（Claude/ChatGPT/Linear 风格整层重设计） | 4895 | `pages/GaeaPage` |
| `src/chat-board.css` | 19366 | `pages/ChatPage` |
| `src/novel-workspace.css` | 12207 | `pages/NovelPage` |
| `src/pages/character-page.css` | 13866 | `pages/CharacterPage` |
| `src/pages/settings-page.css` | 4990 | `pages/SettingsPage` |
| `src/pages/modelcenter/modelcenter.css` | 22551 | `pages/ModelCenterPage` |
| `src/components/module-launcher.css` | 18061 | `ModuleLauncher` |
| `src/components/imagegen/imagegen.css` | 14463 | `ImageGenPage` |
| `src/components/characterlib/{character-card,character-detail,character-library}.css` | 8.6/10.2/3.9KB | CharacterCard / CharacterLibEditor / CharacterLibraryPage |
| `src/gaea/components/memoryhub/hub.css` | 4564 | `MemoryHubPage` |
| `src/App.css` | 91 | 无人 import（空壳） |

各 css 的令牌方言（grep 计数，见 §7 证据）：chat/novel/imagegen/module-launcher/settings/characterlib/character-page 均以 `--md-sys-*`+`--gaea-*` 为主；**modelcenter.css 完全自成一系**（`--mc-*` 125 处，半径 18/13/9px，映射 legacy shims）；hub.css 走 gaea 短名（`--bg/--fg/--border`）。

**4.3 主题工具与状态（真实位置与任务描述不同）**：
- `utils/theme.ts` : 34 —— 只有 `C()` 快捷引用 + `STATUS_COLORS/ROLE_COLORS`（**内含硬编码 `#60a5fa`/`#f87171`**）。ThemeTokens 不在本文件。
- `stores/appStore.ts` : 375 —— ThemeTokens 接口（32 字段：colorPrimary/onPrimary/primaryContainer/onPrimaryContainer、surface 系 6 个、outline 系 2、colorBgContainer/Layout、colorText/Secondary、colorBorder、colorSuccess/Warning、elevation1-5、radiusSm-Xl、transitionFast-Slow、accentRgb、glow、glassBg、auroraBg，appStore.ts:14-31）。
  - 6 暗色主题（:45-60）：nightJade 主色 `#2dd4bf`、nightViolet `#a78bfa`、nightRose `#fb7185`、nightAmber `#f59e0b`、nightMoss `#84cc16`、nightSlate `#94a3b8`；各含完整 surface 阶梯 + `auroraBg` 渐变串。
  - 6 浅色主题（:66-71）：同 6 色系，主色转为 `#0d9488/#7c3aed/#e11d48/#d97706/#4d7c0f/#475569`。
  - 共享 `dS/lS`（:37-38）：elevation/radius（8/12/16/28px）/transition（200/300/400ms）；`getThemeTokens(base,darkMode)`（:80）。
  - 状态：`baseTheme: ThemePreset` + `mode: DisplayMode(light/dark/system)` + 派生 `darkMode` + `density/motion/accentColor/fontFamily/fontSize`；`setTheme/setMode/toggleDarkMode/setDensity/setMotion/setAccentColor/setFontFamily/setFontSize`（:223-262）；localStorage 持久化 key `gaea-theme/gaea-dark/...` + 旧 key 兼容（:82-91）；`FONT_OPTIONS` 5 套字体（:94-100）；matchMedia 监听系统明暗（:366-375）。
- **双 token 层的桥**：`gaea/styles.css:14-79` 定义第二套短名令牌（`--bg/--bg-soft/--bg-elev/--bg-elev-2/--sidebar-*/--border/--fg/--fg-dim/--fg-faint/--accent/--accent-soft/--ok/--warn/--err/--info/--hl-*`/`--radius:9px`/`--mono/--sans`/motion），全部 `var(--md-sys-*, 暗色兜底)` —— 即 gaea 层以**硬编码暗色 hex 为兜底**。`tailwind.css:22-76` 的 `@theme` 再把这些映射成 `--color-*` 供工具类使用。

## 5. Ant Design 定制

- **唯一 ConfigProvider**：`App.tsx:137-155` —— `algorithm: darkMode ? theme.darkAlgorithm : theme.defaultAlgorithm`；token：colorPrimary/colorBgContainer/colorBgLayout/colorText/colorTextSecondary/colorBorder、borderRadius 16（compact 12）、borderRadiusLG 20/14、borderRadiusSM 12/8、fontFamily（FONT_OPTIONS）、fontSize（12-20 可调）、controlHeight 36/32、lineHeight 1.5。
- **index.css 全局覆盖（大量 `!important`）**：Modal/Drawer/Dropdown/Popover/Select/Tooltip/Picker 弹层玻璃化（`:root` 变量）、Menu pill 形、Tabs 霓虹选中、`.ant-modal*` 动画永久禁用（WebView2 冻结 rAF 兼容，index.css:780-848）。
- gaea 工作台几乎不用 antd（仅 `Layout`），全套自绘组件 + Tailwind。

## 6. 品牌元素

- `frontend/public/favicon.svg`（48px）：翡翠球体 + 破土嫩芽 + 星芒，绿系渐变 `#6ee7b7→#34d399→#047857`。
- `gaea/assets/logo.svg` / `logo-light.svg`（512px）：同图 + `#0f172a` 深色圆角底板（logo-light 为浅底版）。
- `index.html:7` 标题「gaea · 多功能 AI 助手」；favicon 链接 `index.html:5`。
- 品牌色：**翡翠绿系**（jade/emerald）为品牌色，默认主题 nightJade 主色 `#2dd4bf`；MainLayout 头部品牌字用 `--md-sys-color-primary` + `--gaea-glow` 霓虹（MainLayout.tsx:331-335）。注意：favicon 的品牌绿（#34d399 系）与 nightJade 主色 #2dd4bf 并非同一色值。

---

## 7. 现状问题清单

### 7.1 各板块视觉风格不统一的证据（4 种并行「方言」）
| 方言 | 板块 | 证据 |
|---|---|---|
| A. M3 玻璃霓虹（md-sys + gaea-glow + blur） | 聊天/小说/绘梦/设置/角色库/首页 | chat-board.css、novel-workspace.css、imagegen.css、settings-page.css、characterlib/*.css、module-launcher.css 均 `--md-sys-*`+`--gaea-*`（各 10-59 处） |
| B. 办公扁平 Tailwind（短名令牌、刻意无 blur） | 办公/记忆中枢/知识库 | `gaea/styles.css:5-12`「office-focused, terminal-flavored」；`tailwind.css:10`「No backdrop-filter / blur (slow on WebKitGTK)」——与方言 A 的玻璃拟态正面冲突；`redesign.css` 又叠第三层（Claude 风格极简、去光晕） |
| C. 模型中心私有命名空间 | 模型中心 | `modelcenter.css` 125 处 `--mc-*`、0 处 md-sys/gaea，半径体系 18/13/9px 与全局 8/12/16/28 不同，`--mc-danger:#fb7185` 硬编码不随主题 |
| D. 记忆中枢科幻风 | 记忆中枢 home | `MemoryHubPage.tsx:151-153` 自绘 `hub-bg/hub-grid/hub-scanline` + tailwind 工具类混用；`hub.css` 走短名变量 |

另：`CharacterPage.tsx` 用独立 `character-page.css`（`.char-detail-col` 布局与小说其它子页的 `.novel-panel` 风格不同）；Settings 用 BEM（`settings-page__*`）。

### 7.2 硬编码色值（不走 token 的具体点）
- `utils/theme.ts:9,11,21-22` STATUS/ROLE 色 `#60a5fa`/`#f87171`（不随 12 主题）。
- `components/VoiceSettingsPanel.tsx` zinc 硬编码：`#18181b/#27272a/#d4d4d8/#71717a/#a1a1aa/#52525b/#4ade80/#f87171`（换主题即违和）。
- `components/EmotionStarMap.tsx` 10 个情绪 hex（#f472b6/#fb7185/#f59e0b/...）；`WhisperTracePanel.tsx` 26 处、`WhisperEmotionPanel.tsx` 22 处、`settings/AppearancePanel.tsx` 21 处（主题色块列表属合理，但含其它）。
- `layouts/MainLayout.tsx:55` themeDots 6 个 hex 与 `appStore.ts` 6 色系、`AppearancePanel.tsx:11-16` 三处重复维护同一色值表。
- `pages/ChapterPage.tsx:248` 内联 `#4ade80/#f59e0b`；`pages/chat/constants.ts` 9 处；`gaea/components/StatsPanel.tsx` 9 处（图表色）。
- 全仓 tsx 硬编码 hex 计数：appStore.ts 276（主题表，合理）、RelationGraph 30、VoiceSettingsPanel 29、Whisper* 系 5 文件 60+。

### 7.3 通用组件缺失 / 重复
- Markdown ×4、SuggestionCard ×2、TaskCenter ×2、欢迎屏 ×3（§2）。
- 无统一 Button/Card/Input 规范：antd 组件 + 各板块自绘 `<button>`（如 `.novel-subnav-item`、`.mc-tab`、`.settings-tiles`、gaea ToolbarButton）并存，样式各自为政。
- `--ds-*` 声明为 canonical 设计令牌（`gaea/styles.css:10`「`--ds-*` prefix is canonical (per DESIGN.md)」）但**14 个令牌只有 2 个被定义**（redesign.css:15-16），27 处消费（styles.css:751-1377 及 10 个组件 inline style）实际落到未定义变量 → 圆角/阴影/语义色在运行时失效。
- 死代码：AppBar、MobileSheet、App.css。

### 7.4 主题系统与样式耦合点
- **设计文档与运行时代码令牌名不一致**：MASTER.md 写的 `--glass-bg/--glow/--aurora-bg/--focus-ring/--color-destructive` 在代码中不存在（实际为 `--gaea-glass-bg/--gaea-glow/--gaea-aurora-bg`，`--focus-ring` 全仓 0 命中，destructive 无令牌）；落地时需二选一并迁移。
- 四代 shim 叠加：`--md-sys-*`（运行时）→ `--color-*/--bg-*` legacy shim（运行时）→ gaea 短名（styles.css，硬编码暗色兜底）→ tailwind `--color-*`（@theme）。tailwind `--color-border/--color-fg` 与 legacy shim `--color-border` 同名冲突（App.tsx:109 也写 `--color-border`），依赖加载顺序。
- `gaea/styles.css:17-25` 兜底值全是暗色 hex（#0F172A 等），浅色主题下若主应用令牌未就绪会闪暗色。
- 12 主题色值在 3 处重复（appStore 表 / MainLayout themeDots / AppearancePanel 列表），无单一数据源。
- 字号：tsx 中硬编码 11px 级 321 处、12px 级 267 处（inline `fontSize` + tailwind `text-[11/12px]`），低于 MASTER 正文 14px 底线；`VoiceSettingsPanel.tsx` 甚至 `fontSize: 10`。

### 7.5 可访问性 / 一致性隐患（抽查）
- **Emoji 当图标**：30 个文件 130+ 处（MainLayout StatusBar `🧠💻🎮⚠` 无 aria、WhisperMemoryLibrary 30、AppearancePanel 9、ChatPanel 7、modelcenter/utils 5），违反 MASTER.md「icons only、no emoji-as-icon」。
- 自定义 `<button>`/`<span role="button">` 焦点环不统一：ModuleLauncher 的 LauncherCard 有 `:focus-visible`（index.css:188-191），gaea 工作台靠 `redesign.css:171-174` 的 `.gaea-app-layout :focus-visible` 兜底，但 chat/novel/modelcenter/settings 大量自绘按钮无 focus 样式；antd 组件焦点环被多处覆盖。
- 对比度：多处 `border: rgba(255,255,255,0.06)`、`text-[10px]`、`--mc-muted` 基于 `--color-text-secondary`（#94a3b8 on #0f1a20 ≈ 3.5:1 边缘），浅色主题 `#5c4c30`（amberL 次要文字）在 `#fffdf5` 上需核验 ≥4.5:1。
- 动效：`prefers-reduced-motion` + `.ui-reduced-motion` 双通道已做（index.css:416-422, 727-734，正向）；但 WebView2 兼容把 antd Modal/Drawer/Popover 动画**永久禁用**（index.css:780-848）——桌面端所有弹层无过渡，交互感知突兀，且属于「为单平台 hack 牺牲全平台动效」的耦合点。
- 状态传达：多处仅色区分（如 `.live-dot` 颜色、引擎状态点、Whisper 情绪色），未带图标/文字三重传达。

---
*调研方法：glob/grep/read + 只读 pwsh 统计；未运行测试/构建，未修改任何文件（本报告为唯一写入）。*
