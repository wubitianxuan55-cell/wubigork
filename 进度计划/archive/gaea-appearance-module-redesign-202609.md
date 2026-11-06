# gaea 外观模块梳理与重构（2026-09）

> 范围：设置中心「通用」分类下的 7 个外观面板（主题色系 / 显示模式 / 字体 / 密度 / 动效 / 强调色 / 界面语言），及其背后的主题令牌、持久化与 i18n 链路。本文先梳理现状、再诊断问题、最后给出本次重构设计。行号以 2026-09-15 工作区代码为准（重构已合入：AppearancePanel.tsx 为新实现 421 行；§4 的「重构前」行号系合入前快照，供对照）。

---

## 1. 模块地图

| 文件 | 行数 | 职责 |
|---|---|---|
| `frontend/src/pages/SettingsPage.tsx` | 275 | 设置中心「控制室」三分区工作台。`CATEGORIES[0] = 'general'`（L38-47）串联渲染 7 个外观面板组件（L46）；L45 keywords 供跨分组搜索（主题/暗色/字体/密度/动效/强调色/语言等） |
| `frontend/src/components/settings/AppearancePanel.tsx` | 421（重构后） | **本次重构主体**。导出 1 个默认组件 + 6 个具名面板（签名不变）；内部含 ThemeOrb / AppearancePreview / SegmentedRow 三个私有组件与 THEME_LABEL_KEYS / FONT_LABEL_KEYS 两张 i18n 派生表；ChoiceCards / ThemeCard 已随重构删除 |
| `frontend/src/components/settings/SettingsSection.tsx` | 69 | v3-card 卡容器：霓虹标题条（L22-30，直接消费 `var(--gaea-glow)`）+ 面板级 icon + `instant` 即时生效徽章（L42-54）+ desc。外观 7 面板全部以此为壳 |
| `frontend/src/stores/appStore.ts` | 447 | zustand 全局 store。外观相关：`THEME_PRESETS` 6 主题元数据单一来源（L26-33）、`getThemeTokens`（L129-133）、`FONT_OPTIONS`（L150-156）、7 个外观字段 + setter（localStorage 持久化，L135-147 键常量、L290-334 setters） |
| `frontend/src/App.tsx` | 193 | 令牌消费总入口：订阅 store → `getThemeTokens` → accent 覆盖 → `useEffect` 写 `:root` CSS 变量（L50-154）→ antd ConfigProvider（L161-180）+ density/motion 根类名（L157-160） |
| `frontend/src/gaea/lib/i18n.tsx` | 153 | LocaleProvider + `useT`；zh 静态加载、en/zh-TW 按需 chunk（L26-43）；translate 回退链 当前 locale → zh → en → 裸键（L73-80）；偏好存 `gaea-lang`（L44） |
| `frontend/src/gaea/locales/en.ts` / `zh.ts` / `zh-TW.ts` | 1596 / 1594 / 1598 | 三语字典。en 为 canonical：`export type DictKey = keyof typeof en`（en.ts L1596）；zh（L6）、zh-TW（L5）以 `Record<DictKey, string>` 注解锁键集 |
| `frontend/src/components/settings/AppearancePanel.test.tsx` | 117（随重构新增） | 新设计行为测试：主题下拉选择持久化、悬停预览徽章、三组 Segmented 切换（详见 §6.2） |
| `frontend/src/layouts/MainLayout.tsx` | 799 | 侧边第二个主题入口：启动器主题色点快捷切换（L89-91 模块级 `themeDots`/`themeLabels` ← `THEME_PRESET_COLORS`/`THEME_PRESET_LABELS`；L691-707 渲染色点按钮 + Tooltip + aria-label） |
| `frontend/src/lib/accent.ts` | — | `ensureLightContrast`：亮态下自定义强调色自动深化到 WCAG AA（App.tsx L43-47 消费，v4.274） |
| `frontend/src/stores/appStore.test.ts` | 72 | getThemeTokens 契约测试：12 套（6 色系 × 明暗）令牌完整性、colorDestructive、tertiary 对、onPrimary 配对、共享常量一致（L10-59） |

## 2. 数据流与存储

### 2.1 主题令牌链路（选择 → 生效）

```
选择入口（二选一）
  AppearancePanel（设置中心）          MainLayout 启动器色点（L691-707）
        │ setTheme / setMode / …            │ setTheme
        ▼                                    ▼
  useAppStore setter（appStore L290-334）── set state + localStorage.setItem
        │
        ▼ zustand 订阅
  App.tsx L26-33   getThemeTokens(baseTheme, darkMode)   ← appStore L129-133
                   darkFn/lightFn 注册表（L111-112）+ TERTIARY 暖伴侣色（L120-127）
        │
        ▼ L43-47  effTokens = accentColor 覆盖 glow/colorPrimary/accentRgb
                   （亮态先经 ensureLightContrast 深化；无 accent 时保持同一引用防重复写变量）
        │
        ▼ L50-154 useEffect 写 :root CSS 变量
   ① --md-sys-color-* / --md-sys-elevation-* / --md-sys-radius-* / --md-sys-transition-*（M3 全量）
   ② --gaea-glow / --gaea-glass-bg / --gaea-aurora-bg（L111-113，未来感扩展令牌）
   ③ backward-compat shims：--color-* / --bg-* / --radius-* 等（L115-149，v3 层与旧组件消费）
   ④ data-hl="dark|light"（L153，代码高亮明暗挂钩）
        │
        ├─▶ 全站组件直接 var(--gaea-glow) 等（frontend/src 内 212 处引用；
        │    SettingsSection 霓虹标题条即典型消费点 L26-30）
        ├─▶ L157-160 根类名 ui-compact / ui-reduced-motion（density/motion 走 CSS 层而非变量）
        └─▶ L161-180 antd ConfigProvider token（colorPrimary/圆角/字体/字号/controlHeight 均随 density 联动）
```

要点：
- `baseTheme`（色系）与 `mode`（light/dark/system）正交；`darkMode` 是派生态（`resolveDark`，appStore L201-203），system 模式下由 `matchMedia` 监听实时跟进（L438-447）。
- `getThemeTokens(preset, dark)` 返回约 40 个令牌（ThemeTokens 接口 L44-65），明暗两档各有独立工厂函数（暗 L79-94 / 亮 L100-105），共享 elevation/radius/transition 常量（L71-72）。
- 悬停预览（hovered）是**纯组件态**，不进 store、不落盘——只影响 AppearancePreview 渲染哪个主题色，点击才 `setTheme` 持久化。

### 2.2 localStorage 键表

| 键 | 值域 | 默认 | 写入点（appStore） | 备注 |
|---|---|---|---|---|
| `gaea-theme` | 6 个 ThemePreset key | `nightJade` | setTheme（L290-293） | 读时兜底 legacy `wubigork-theme`（L220-226） |
| `gaea-display-mode` | `light`/`dark`/`system` | `dark` | setMode（L301-304） | 读时兼容旧 boolean：`gaea-dark='0'` → light（L209-218） |
| `gaea-dark` | `'1'`/`'0'` | — | toggleDarkMode 双写（L295-299） | 旧版 boolean 键，仅 toggle 路径仍同步写 |
| `gaea-density` | `standard`/`compact` | `standard` | setDensity（L306-309） | |
| `gaea-motion` | `full`/`reduced` | `full` | setMotion（L311-314） | reduced 对齐系统「减弱动态」 |
| `gaea-accent` | 任意 hex | （无键=跟随主题） | setAccentColor（L316-319） | 空串时 `removeItem` 而非写空 |
| `gaea-font-family` | FONT_OPTIONS 5 个 key | `system` | setFontFamily（L321-324） | 读时校验合法 key（L158-164） |
| `gaea-font-size` | 12–20 整数 | `14` | setFontSize（L326-329） | 读时钳制范围（L165-171） |
| `gaea-lang` | `en`/`zh`/`zh-TW` | （无键=auto） | i18n.tsx writePref（L62-69） | auto 时 detectLocale 走 navigator.language |
| ~~`wubigork-theme` / `wubigork-dark`~~ | — | — | — | 只读兜底的迁移前旧键（L136/138） |

## 3. i18n：settings.appear.\* 三语与键集锁定

- **键数量与分布**：`settings.appear.*` 共 **64 键**（zh.ts L901-964；en.ts / zh-TW.ts 同键集，起始行号 901 / 906）。按面板分组：预览 7 键（livePreviewTitle/Desc、previewing、previewCardTitle/Desc、dark、light）、主题 14 键（themeTitle/Desc + 6 主题 × Label/Desc）、显示模式 8 键、字体 12 键、密度 6 键、动效 6 键、强调色 5 键、语言 6 键。
- **双层键集锁**：
  1. 字典层：en.ts L1596 `export type DictKey = keyof typeof en`，zh.ts L6 / zh-TW.ts L5 的 `Record<DictKey, string>` 注解——en 增删键而其他语言未跟上，`tsc` 直接编译失败（en.ts 头注释 L1-4 明言此契约）。
  2. 面板层：AppearancePanel.tsx L15-22 `THEME_LABEL_KEYS: Record<ThemePreset, {label: DictKey; desc: DictKey}>`——appStore 新增主题（`ThemePreset` 联合类型扩员）时 tsc 强制补对应 i18n 键；字体同理 `FONT_LABEL_KEYS`（L243-249）。
- **数据/文案分离**：主题元数据（key/label/desc/color）的**数据单一来源**在 appStore `THEME_PRESETS`（L26-33），其中 label/desc 是 zh 兜底文案；面板渲染时经 `useThemeOptions()`（AppearancePanel L35-41）用 `THEME_LABEL_KEYS` 派生当前语言 label/desc（`t` 变更时重算）。
- **回退与按需加载**：translate 回退链 当前 locale → zh → en → 裸键（i18n.tsx L73-80）；zh 静态恒可用，en/zh-TW 首次切换才动态加载 chunk（L33-43）。
- **本次重构的 i18n 变更面**：不增删任何 key，仅更新 `settings.appear.livePreviewDesc` 一行三语——旧文案「鼠标悬停下方主题卡可即时预览」（对应 ThemeCard 网格 hover）→ 新文案「打开下拉悬停选项可即时预览，点击才生效」（对应 Select 下拉 hover）。**该改动已先行落在工作区**（git diff 确认三文件各恰 1 行，未提交），组件重写现已合入，文案与实现自洽。

## 4. 重构前结构与问题诊断

### 4.1 现状结构（'general' 分类 = 8 张 SettingsSection 卡）

SettingsPage L46 按序串联 `<AppearancePanel /><DarkModePanel /><FontPanel /><DensityPanel /><MotionPanel /><AccentPanel /><LanguagePanel />`，其中 AppearancePanel 组件输出 2 张卡，共 **8 卡**：

| # | 卡（SettingsSection） | 来源组件 | 控件形态 | 关键行号 |
|---|---|---|---|---|
| 1 | 外观实时预览（noMargin） | AppearancePanel | AppearancePreview 微缩预览画布（霓虹标题条 + 玻璃卡模拟） | L227-229、预览实现 L157-214 |
| 2 | 主题色系 | AppearancePanel | **6 张 ThemeCard 并排网格**（`repeat(auto-fill, minmax(170px,1fr))`）；hover 卡片即时预览（onMouseEnter → setHovered，L100-107）+ 边框亮起，点击 setTheme | L230-246（网格 L236）、ThemeCard L88-154 |
| 3 | 显示模式 | DarkModePanel | ChoiceCards **三卡**（暗色/亮色/跟随系统，grid `minmax(150px,1fr)` auto-fit） | L252-268、ChoiceCards L36-85 |
| 4 | 字体设置 | FontPanel | **行式 × 2**：字体族 Select（width 180）+ 字号 InputNumber（12-20，带实时字号样张） | L279-330 |
| 5 | 界面密度 | DensityPanel | ChoiceCards **两卡**（标准/紧凑） | L333-348 |
| 6 | 动效强度 | MotionPanel | ChoiceCards **两卡**（完整/减弱） | L351-366 |
| 7 | 强调色 | AccentPanel | **行式**：发光色块取色器（input[type=color]）+「跟随主题」重置按钮 | L369-410 |
| 8 | 界面语言 | LanguagePanel | **行式**：Select（width 260，auto + 三语 autonym） | L415-445 |

### 4.2 问题诊断

1. **主题 6 卡网格纵向占用大**：`minmax(170px,1fr)` auto-fill 在设置页中部内容区宽度下每行 3~4 卡，6 卡占 2 行、加上氛围色渐变条（62px）单卡高约 120px——仅主题选择就吃掉 ~250px 纵向空间，页面被拉长。
2. **预览与选择割裂**：实时预览（卡 1）与主题选择（卡 2）分属两张卡，hover 主题卡时用户视线上下来回扫视才能确认预览效果；预览徽章「预览中」（previewing 键）挂在卡 1 内，距离 hover 目标远。
3. **2~3 选项维度滥用卡片网格**：显示模式（3 选项）、密度（2）、动效（2）都是低基数单选，却各占一张全宽卡 + 卡片网格一行；行式控件（Segmented 一行 ~32px）即可承载，纵向省 ~60%。
4. **控件语言不统一**：卡 2/3/5/6 是卡片网格、卡 4/7/8 是「图标 | 标题描述 | 控件」行式——同一分类内两种交互范式混排，视觉节奏断裂。
5. **两套选中态实现重复**：ChoiceCards 与 ThemeCard 各自维护「选中发光边框 + 右侧发光对勾」逻辑（L73-79 vs L143-150），样式细节漂移需双处同步。

## 5. 重构设计

### 5.1 前后对照总表

| 维度 | 重构前 | 重构后 |
|---|---|---|
| 卡数量（'general'） | 8 张 | **7 张**（实时预览合并进主题卡） |
| 主题选择 | 6 张 ThemeCard 网格（2~3 行） | **antd Select 下拉**（单行） |
| 主题预览 | 独立预览卡 + hover 主题卡触发 | AppearancePreview 内嵌主题卡；**hover 下拉 option 触发** |
| 显示模式 / 密度 / 动效 | ChoiceCards 卡片网格（3+2+2 张卡） | **行式条目 + antd Segmented**（3 行） |
| 控件范式 | 卡片网格与行式混排 | 统一「图标 \| 标题描述 \| 控件」同构行式 |
| 字体 / 强调色 / 语言 | 行式 | **保持不变** |
| 导出面 | default + 6 具名面板 | **不变**（SettingsPage 零改动） |
| i18n | 64 键 | 64 键不变，仅 livePreviewDesc 文案更新（已落地） |
| 主题数据源 | appStore.THEME_PRESETS | 不变；面板仍经 THEME_LABEL_KEYS 派生三语 label |
| 测试 | 无面板级测试 | 新增 AppearancePanel.test.tsx 行为测试（已落地，117 行 6 用例） |

### 5.2 主题下拉交互细节（AppearancePanel）

- **触发器（labelRender）**：`Select<ThemePreset, ThemeSelectOption>` 双泛型参数（L131-133，onChange 直接收 ThemePreset、零类型断言）；labelRender = 发光色点 ThemeOrb（16px，取 `THEME_PRESETS.color` 渲染径向渐变 + 双层光晕）+ 当前主题名（L138-146）；替代 6 卡网格成为单行入口。
- **选项行（optionRender）**：ThemeOrb（20px）+ 主题名（13px/600）+ 一行描述（11px，color/desc 经 `option.data` 透传，类型见 ThemeSelectOption L27-32）+ 选中项尾部 CheckOutlined（`--gaea-glow` 色，L147-163）；信息密度对齐原 ThemeCard 的名称区，纵向收敛为下拉浮层内的 6 行。
- **悬停即时预览**：option 容器 onMouseEnter → `setHovered(d.value)`、onMouseLeave → 清空（L151-152）→ 内嵌 AppearancePreview 切换到该主题色渲染（hovered 派生逻辑沿用：`previewT = themeOptions.find(x => x.key === (hovered ?? baseTheme))`，L122）；预览徽章仍用 `settings.appear.previewing`（「预览中」）标识非当前持久值。
- **取消预览**：下拉关闭经 `onOpenChange(false)` 清空 hovered（L137——antd 新 API，未用已废弃的 onDropdownVisibleChange），预览回落到 baseTheme；Esc/点选同理不残留。
- **生效**：点击 option → `setTheme(key)` 持久化（localStorage `gaea-theme`）+ App.tsx 令牌链路全站联动。悬停永不落盘，与重构前语义一致。
- **预览合并**：AppearancePreview 从独立 SettingsSection 移入主题卡内部——卡内自上而下为：主题下拉（width 320）→ hover 说明行（livePreviewDesc）→ 预览小标题（livePreviewTitle）→ 预览画布（L131-174），8 卡 → 7 卡；livePreviewDesc 新文案（「打开下拉悬停选项可即时预览，点击才生效」）与交互精确对应。

### 5.3 Segmented 行式规格（DarkModePanel / DensityPanel / MotionPanel）

三面板统一为同构行式条目，落地为本地泛型组件 `SegmentedRow<T>`（L181-220），规格对齐 FontPanel（L256-299）/ AccentPanel（L348-381）既有行式风格：

```
[ 30×30 图标方块 ] [ 标题(13px/600) + 描述(10.5px/次级色) ] [ antd Segmented ]
```

- **左侧图标方块**：跟随**当前选中项**动态切换（如显示模式：暗色= MoonOutlined / 亮色= SunOutlined / 跟随系统= DesktopOutlined），30×30 圆角 9px、surface-variant 底——与字体/强调色行 icon 方块同规格。
- **中部文案**：标题 = 维度名（displayTitle/densityTitle/motionTitle），描述 = **当前选中项**的 label+desc（如「紧凑 · 信息密集，一屏更多」），让行内即读出当前状态，不必先解析 Segmented 高亮。
- **右侧 Segmented**：选项 label 取各维度 mode\*/density\*/motion\* 键（短词，天然适配 Segmented）；block=false 右对齐，宽度随文案。onChange → setMode/setDensity/setMotion 持久化。
- **卡的其余部分不动**：仍各自一张 SettingsSection（icon/title/desc 壳层不变），仅卡内 ChoiceCards → 行式。
- ChoiceCards / ThemeCard 私有组件随重构移除（无外部引用，仅本文件使用），顺带消除两套选中态实现的重复维护。

### 5.4 不变项（明确边界）

- FontPanel / AccentPanel / LanguagePanel 逐行保持现状（已是目标行式范式）。
- 导出签名不变：`export default AppearancePanel` + `DarkModePanel` / `FontPanel` / `DensityPanel` / `MotionPanel` / `AccentPanel` / `LanguagePanel` 具名导出——SettingsPage L11 import 与 L46 渲染**零改动**。
- MainLayout 启动器色点（第二主题入口）不动：它消费 `THEME_PRESET_COLORS/LABELS`，与面板共享同一数据源，重构天然兼容。
- appStore 全部键名、默认值、legacy 兜底逻辑不动——存量用户 localStorage 无迁移成本。

## 6. 兼容性与验证

### 6.1 兼容性清单

| 项 | 结论 |
|---|---|
| SettingsPage | 零改动（导出签名不变，import 列表 L11 原样） |
| localStorage 存量 | 零迁移（键名/值域/默认值全不变） |
| i18n 键集 | 64 键不变；`Record<DictKey, string>` 双层锁不触发；仅 livePreviewDesc 文案更新（工作区已落地） |
| MainLayout 色点 | 不动，共享 THEME_PRESETS 数据源自兼容 |
| appStore / App.tsx | 不动（重构纯组件层） |
| 主题搜索 keywords | SettingsPage L45 keywords 覆盖「主题/颜色」等词，不依赖控件形态，搜索不受影响 |

### 6.2 验证项（frontend 目录）

| 命令 | 验证点 |
|---|---|
| `npx tsc -b`（= build 前半） | 导出签名兼容、THEME_LABEL_KEYS/DICTS 键集锁、无死引用（ChoiceCards/ThemeCard 移除后） |
| `npx vitest run` | ① 新增 AppearancePanel.test.tsx（已落地，117 行 6 用例）：渲染主题下拉显示当前主题（nightJade→「暗夜青」）；下拉选择「暗夜金」→ store.baseTheme + `gaea-theme` 持久化；hover「暗夜紫」option → 「预览中」徽章出现；三组 Segmented（亮色模式/紧凑/减弱动态）→ setMode/setDensity/setMotion + `gaea-display-mode`/`gaea-density`/`gaea-motion` 落盘。② 既有 appStore.test.ts 令牌契约（12 套 × 完整性）不受影响全绿 |
| `npx eslint .` | no-raw-hex 规则：下拉色点/预览取色一律来自 THEME_PRESETS.color 数据，不新硬编码 hex |
| 手工冒烟 | 主题下拉 hover 预览→Esc 取消回落→点选生效全站联动；亮色档重复一遍（effTokens 亮态 accent 深化路径）；语言切 en/zh-TW 后下拉与 Segmented 文案随动（t 变更重算） |

**实测结果（2026-09-15，重构合入后）**：`npx tsc -b` 零错误；`npx vitest run` AppearancePanel.test.tsx **6 用例全部通过**（渲染当前主题 / 下拉选择持久化 `gaea-theme` / 悬停「预览中」徽章 / 三组 Segmented 持久化断言）；`npx eslint .` 零告警。

### 6.3 风险与备注

- **落地终态（2026-09-15 合入）**：重构已合入并通过验证（见 §6.2 实测），此前「文案先行、组件未跟上」的中间态已消解。终版实现与 §5 设计契约一致：`Select<ThemePreset, ThemeSelectOption>` 泛型零断言（L131）；labelRender = ThemeOrb 16px + 主题名（L138-146）；optionRender = ThemeOrb 20px + 名称 + 描述 + 选中 CheckOutlined（L147-163）；option onMouseEnter/onMouseLeave 驱动 hover 预览（L151-152）；下拉关闭经 **onOpenChange**（antd 新 API，L137）清空 hovered，未用已废弃的 onDropdownVisibleChange；模式/密度/动效三面板统一走 `SegmentedRow<T>` 行式组件（L181-220）；ChoiceCards / ThemeCard 已删除，新增本地 ThemeOrb 组件（L44-52）；文件 447 → 421 行。
- Segmented 选项 label 过长会挤压行宽：三语下最长的 modeSystemDesc 等描述文案放**左侧描述区**而非 Segmented 选项内，Segmented 只放短 label，规避 i18n 长词溢出。
- antd Select 自定义 labelRender/optionRender 需要 antd ≥ 5.x 的 API；仓库 antd 版本满足（Select options/popupMatchSelectWidth 等既有用法同源）。
