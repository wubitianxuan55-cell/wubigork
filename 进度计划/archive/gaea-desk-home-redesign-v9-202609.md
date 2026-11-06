# 书斋首页 v9「案头」重设计（2026-09）

> 范围：gaea 双空间首页的书斋（work）变体 `DeskHome`——v9「案头」重设计。闲庭（play）零改动，SpaceSwitch / 共享数据层（`useLauncherData`）零改动。本文与并行的 v9 重写协同：§2 诊断基于落地前的 **v8 快照**（行号以 2026-09-15 工作区 `ModuleLauncher.tsx` 921 行 / `module-launcher.css` 1128 行 / `ModuleLauncher.test.tsx` 84 行为准，v9 合入后行号会漂移，请按类名对照）；§3–§5 为 v9 设计契约，作为重写落地的验收基准；§6 验证由主代理统一执行，结果已回填（§6.3）。
>
> 快照备注：落笔时 `module-launcher.css` L131 的 `.ml-space-chip` 注释已写「书斋 masthead 内」，先于 TSX 结构（chip 实际仍在 `w-deck-head`，TSX L517）——系 v9 重写已在动 CSS 或注释预写，不影响本文契约。

---

## 1. 背景与版本谱系

书斋首页一年内四易其稿：v6 因「等大圆角卡 + 描边、层级靠字号微差、极光斑装饰」被判低级（TSX L3–15 对照检查表）；v7 立「文书台」——杂志刊头 + 仪表条/账页/海报墙三形态 + 四档排印 + 闲庭月洞门，品质达标但刊头占约 25% 视高；v8 压刊头——刊头压缩为 kicker 行、命令台升主角、账页×目录双列并排、右舷晨报+四节主从分层（TSX L509–512 / L592–593 / L652–653），纵向效率换来新问题：**身份记号随刊头一起被砍掉**。v9「案头」纠正矫枉过正：恢复排印锚点但不回 v7 的杂志刊头——开放式 masthead（印章 + 台名 + 一行 lede）+ 更高的命令条 + 案牌网格 + 脉息面板。

| 版本 | 主题 | 关键动作 | 结果与遗留 |
|---|---|---|---|
| v6 | 模板感 | 等大圆角卡+描边铺满、13↔14px 字号微差、aurora blob 装饰 | 用户判「低级」 |
| v7 | 文书台 | 杂志刊头（大标题+三行 lede，~25% 视高）、三形态、四档排印、月洞门 | 品质达标；书斋首屏纵向被刊头吃掉 |
| v8 | 压刊头 | 刊头→kicker 行、命令台升主角、账页×目录双列（7fr/5fr）、右舷主从 | 纵向省了；身份记号丢失，界面滑向清单海（见 §2） |
| **v9** | **案头** | **开放式 masthead（印章+文书台大字）、命令条 60px 巩固主角、案牌网格替索引行、脉息面板升环** | **本文契约** |

「案头」的意象即验收口诀：**抬头见款识**（印章 + 台名），**伸手是笔砚**（命令条），**左手账册**（账页），**右手名牌**（案牌），**侧厢脉案**（脉息面板）。v9 不是回退到 v7——masthead 只占 2–3 行开放式排印（约 8–10% 视高），身份与效率兼得。

## 2. v8 问题诊断

### 2.1 v8 现状结构（快照）

```
DeskHome（.ml.ml-work → .w-dock，主栏 1fr + 右舷 304px，CSS L339-348）
├ ml-strip      SpaceSwitch 顶条（书斋|闲庭 · 日期 · 模型 chip，TSX L398-436）
├ w-main
│  ├ w-deck     命令台（TSX L513-590，CSS L356-379）
│  │  ├ w-deck-head  kicker 行：[w-kicker-mark][ml-space-chip][w-deck-sub][w-kicker-rule][w-kicker-pill]（L514-524）
│  │  ├ w-hero-chat  气泡流（按需，L526-531）
│  │  ├ w-cmd        命令条 54px / 14px（L534-578，CSS L412/L442）
│  │  └ w-voice-status 语音状态行（L580-589）
│  └ w-columns  双列 7fr | 5fr（L594，CSS L531）
│     ├ w-ledger   最近文档账页：表格行 + 等宽序号（L596-623，testid=desk-recent-docs）
│     └ w-cap      能力目录（L626-648）：FeaturedBand 旗舰横带（L697-725）
│                  + w-index 单列 hairline 索引行（IndexItem L728-754）
└ w-rail       右舷 304px（L654）
   ├ MorningBriefCard 晨报卡（L656）
   └ w-rail-panel  四节堆叠（L657-689，CSS L747-754）：内核 / 写作 / 会话 / 记忆（全 hairline 分节）
```

### 2.2 五条诊断

1. **无第一印象**。v8 为压缩纵向空间砍掉 v7 杂志刊头后，页面没有任何排印锚点与身份记号：闲庭有月洞门环 + `p-display` clamp(34–54px) 大字（CSS L783–841），书斋无对应物——首屏最大字号是旗舰横带名的 19px（L660），剩下是一堆等权小件。两个空间并置时书斋像闲庭的「无签名版」。
2. **清单海**。账页行 + 索引行 + 右舷四节 ≈ 80% 界面是 hairline 文字行；等宽字消费点在书斋首屏就有约 10 处（`ml-strip-date`/`w-ledger-idx`/`w-ledger-path`/`w-index-num`/`w-kbd`/`ml-sess-meta`/`ml-krow-strong`/`ml-meter-label`/`ml-meter-val`/`ml-ring-num`），字号 10–13px 密布——仪表盘/调试台气质，与「书斋」意象相悖。
3. **主角不突出**。命令条是页面第一任务（发起工作）的载体，但仅 54px 高（CSS L412）、输入 14px（L442），上方还压着一条 kicker 行分走首行视线，左右又被两段列表与右舷稀释——「第一任务」在视觉权重上不成立。
4. **右舷拥挤**。304px 窄栏里晨报卡 + 四节面板全部纵排（CSS L747–754），节奏是「卡-线-线-线-线」一路到底，无图形锚点，也不与左栏任何件形成对位。
5. **双列失衡**。7fr/5fr 两栏都是同构行式列表（账页行 vs 索引行），并排后互相竞争扫视；旗舰横带本是为全宽设计的件（68px 序号水印 L624–635 + 46px 图标章 L642–651），挤进 5fr 窄列后水印与三行文案施展不开。

## 3. v9 设计总览

### 3.1 布局示意（宽高不成比例）

```
├ ml-strip ── [书斋 | 闲庭] · 日期 · … … … … … … … … · 模型 chip ──┤ ← 零改动
├ w-masthead（开放式排印，无卡片盒；底缘 hairline 收束）───────────┤ ← 新增
│ [印]  文书台（clamp 30–42px/650/-0.02em）      chip · 就绪 pill   │
│ 44px↺ 办公、造价、记忆、进度计划同在一张台面上…（home.sub）        │
├ w-deck（命令台：kicker 行已移除）────────────────────────────────┤
│  ●  对 gaea 说点什么…                      [语音] [发送]  ⌘K    │ ← 60px/15px
│  （气泡流 / 语音状态原样收进台内）                                │
├ w-columns（7fr ｜ 5fr）───────────────┬ w-rail（304px）──────────┤
│  最近文档 · 账页（行式不动）           │  晨报卡（不动）           │
│  01 02 03 …（≤6 行）                  │  w-vitals（单卡四节）    │
│                                       │   ◔ 写作 80px 环（首节） │ ← 图形锚点
│  能力目录                              │   — 内核                 │
│  [旗舰横带 · gaea]（保留为目录首件）   │   — 会话                 │
│  ┌案牌┐┌案牌┐ ┌案牌┐┌案牌┐（双列）    │   — 记忆                 │
└───────────────────────────────────────┴──────────────────────────┘
```

### 3.2 分区对照表

| 分区 | v8 | v9 | 关键改动 |
|---|---|---|---|
| ml-strip（SpaceSwitch） | 顶条切换器 | 同 | 零改动 |
| w-masthead | 无（v8 砍刊头） | 印章 + 台名大字 + lede + chip/pill | 新增身份区，2–3 行开放式排印 |
| w-deck（命令台） | kicker 行 + 54px 命令条 + 气泡/语音 | 去 kicker 行 + 60px/15px 命令条 | 信息上移 masthead，主角加高 |
| w-columns 左（w-ledger 账页） | 表格行 + 等宽序号 | 同 | 文档驱动主角，行式保留 |
| w-columns 右（w-cap 目录） | 旗舰横带 + w-index 单列行 | 旗舰横带 + w-plaques 双列案牌 | 行式→案牌，去等宽序号 |
| w-rail | 晨报 + w-rail-panel 四节纵排 | 晨报 + w-vitals 单卡（写作首节 80px 环） | 更名重排，升图形锚点 |

### 3.3 分区细则

**masthead（身份区）**——`w-masthead` 开放式排印：无卡片盒（不包边、不铺底色板），仅底缘一条 hairline（`--w-line`）收束，与 `ml-strip` 的 border-bottom 呼应成两条平行细线。

- `w-seal` 印章：44×44 方形、`rotate(-3deg)`、底色**印泥朱 `#d06055`**（全文件唯一 raw hex，豁免口径见 §5.4）、取空间名首字（zh「书斋」→「书」，en/zh-TW 随 `shell.space.*` 标签首字）、`aria-hidden`。章面文字反白走 token（如 `var(--color-surface)`）。
- `w-mast-title`：`t('home.title')`「文书台」，`clamp(30px, 3vw, 42px)` / 650 / `-0.02em`——对齐闲庭 `p-display` 的 display 档做法但幅度收敛（闲庭 34–54px 居中仪式，书斋 30–42px 左置刊头）。
- `w-mast-lede`：`t('home.sub')`（v8 的 `w-deck-sub` 内容原样升格）。
- 右侧：`ml-space-chip`（testid 原样，物理位置从 `w-deck-head` 迁入）+ `w-mast-pill` 就绪徽记（`t('home.pill')`，即 v8 `w-kicker-pill` 更名迁入）。
- 实现备注：masthead 区域 `aria-label` 建议复用 `home.title`（对齐闲庭 p-hero 做法，TSX L780）。

**命令台（w-deck）**——kicker 行整体删除，五件去向明确：

| v8 kicker 件 | 去向 |
|---|---|
| `w-kicker-mark`（22px 色条，CSS L590） | 退役（印章接管记号职能） |
| `ml-space-chip` | 迁 masthead 右侧 |
| `w-deck-sub`（home.sub） | 升格 `w-mast-lede` |
| `w-kicker-rule`（渐隐细线，L391） | 退役（masthead 底缘 hairline 接管） |
| `w-kicker-pill`（home.pill） | 更名 `w-mast-pill` 迁 masthead 右侧 |

命令条 `w-cmd`：高 54→**60px**、输入字号 14→**15px**、`focus-within` 光晕加强（光晕半径/混合比例上调，仍全走 `--w-accent` token）；`w-hero-chat` 气泡与 `w-voice-status` 原样收进台内（aria/行为零改动）。

**能力目录案牌网格（w-plaques）**——`FeaturedBand` 旗舰横带保留为目录首件（横带形态已验证：hover 抬升 -2px + 箭头位移，work 旗舰 = gaea 办公工作台，launcher.ts L28）；其下 `w-index` 单列 hairline 行改为 `w-plaques` 双列网格（`repeat(2, minmax(0,1fr))`）。`w-plaque` 案牌：图标章 32px（圆角方块，尺寸对齐闲庭 `p-poster-seal.is-small` 32px，CSS L963）+ 名称 + 单行描述（ellipsis）+ hover 抬升（仅 transform）+ 箭头 hover 显现（沿用 `w-index-arrow` 模式）；**删除 `w-index-num` 等宽序号**——序号本就是位置冗余信息，删除直接降 mono 密度（对应诊断 2）。`IndexItem` 改写为案牌件；数据源 `deriveLauncherModules` / `LAUNCHER_DESC` / `LAUNCHER_FEATURED`（launcher.ts）零改动，消费同一 `LauncherModule[]` 流。

**右舷脉息面板（w-vitals）**——`w-rail-panel` 更名 `w-vitals`：仍是「一张色调面 + 内部 hairline 分节」的单卡形态，语义升格为状态脉搏。节序调整：**写作进度升首节**，`WritingRing` 的 `ml-ring` 从 72px（CSS L244–245）放大到 **80px** 作为右舷唯一图形锚点（晨报之外右舷全是文字，需要一个环打破行海）；其余三节（内核 / 会话 / 记忆）内容与组件零改动（`TelemetryBody` / `SessionList` / `MemoryPulse` 原样复用）。**注意**：`ml-ring` 为书斋/闲庭共用（闲庭 `p-foot` 亦渲染，TSX L810），80px 放大必须用 `.w-vitals .ml-ring` 作用域选择器实现，否则波及闲庭（违反 §5.2 红线）。

## 4. 身份记号：印章与月洞门的对仗

| 维度 | 闲庭 · 月洞门（p-gate） | 书斋 · 印章（w-seal） |
|---|---|---|
| 形 | 圆：外环 + 内环 + 顶部渐隐地平线（CSS L783–816） | 方：44px 方形，定格旋转 -3° |
| 位 | 居中仪式入口（p-hero 中轴） | 左置落款（masthead 行首） |
| 色 | 强调色 token 派生（color-mix `--w-accent`，L788/L798） | 实色印泥朱 `#d06055`，不随主题 |
| 尺度 | 96px 大环（居中 hero 的仪式感） | 44px 小章（刊头的落款比例） |
| 意象 | 门 = 可穿行的环境入口，游园之序 | 印 = 落在纸面的所有权记号，案头之款 |
| 可访问 | aria-hidden 装饰 | aria-hidden 装饰（信息由 chip/title 承担） |
| 动 | 静态 | 静态（transform 定格，不做动画） |

设计语言层面两空间互为对仗：闲庭以「圆 / 居中 / 环境色」表**游**，书斋以「方 / 左置 / 实色」表**作**。月洞门用 token 派生色融入主题随主题换色；印章 deliberately 不随主题——印泥盖下去是什么色就是什么色，这是「品牌识别物」而非「UI chrome 配色」的物理隐喻，也正是它成为本文件唯一 hex 豁免的理由（eslint 配置注释对「品牌识别色」的豁免口径，eslint.config.js L35–36）。

## 5. 契约与兼容

### 5.1 testid / 类名契约（5 钩子全部保持）

| 钩子 | v8 位置 | v9 位置 | 依赖方 |
|---|---|---|---|
| `ml-space-switch` | SpaceSwitch（TSX L406） | 不动 | TSX 契约注释 L17–22 |
| `ml-space-work` / `ml-space-play` | L413 派生 | 不动 | test L46–47 / L52 / L63 / L77 |
| `ml-space-chip` | `w-deck-head`（L517） | **迁 w-masthead，testid 不变** | test L64（title 提示）/ L70 / L82 |
| `desk-recent-docs` | `w-ledger`（L596） | 不动 | test L69 / L81 |
| `.garden-banner` | GardenBanner（L851） | 不动（闲庭） | test L80 |

配套不变项：`VOICE_LAUNCH_FLAG`（L52）、`useLauncherData` 数据层单源、`MorningBriefCard`、共享排版原语（`SectionLabel` / `WritingRing` / `TelemetryBody` / `SessionList` / `MemoryPulse` / `ChatBubble`）签名均零改动。

### 5.2 闲庭零改动

`GardenHome`（TSX L759–841）与 `.p-*` / `.garden-*` 全部 CSS 规则逐行不动；test L75–83 用例必须原样全绿。唯一渗点风险是共享件的样式覆盖（如 §3.3 的 `ml-ring` 放大），一律用书斋侧作用域选择器（`.w-vitals …`）隔离。

### 5.3 i18n 零新增

v9 所需键全部已存在（zh.ts）：

| 键 | zh 值 | v9 用途 |
|---|---|---|
| `home.title`（L650） | 文书台 | w-mast-title |
| `home.sub`（L649） | 办公、造价、记忆、进度计划同在一张台面上… | w-mast-lede |
| `home.pill`（L642） | GAEA 已就绪 · 本地 AI 创作中枢 | w-mast-pill |
| `home.capTitle`（L658）/ `home.featured`（L659） | 能力矩阵 / 旗舰工作台 | 目录节标 / 横带 badge |
| `shell.space.work`（L535） | 书斋 | chip 文案 + 印章首字来源 |

en / zh-TW 走既有回退链与按需 chunk，不触键集锁（`Record<DictKey, string>`）。

### 5.4 令牌纪律与印泥朱豁免

- 既有纪律：module-launcher.css 零硬编码色值，全走 `--color-*` / `--md-sys-*` / `--gaea-*` / `--radius-*` / `--shadow-*` / `--transition-*`（文件头 L13–15）；ESLint `local/no-raw-hex` 为 error 级（eslint.config.js L37–63，覆盖 `**/*.{ts,tsx}`）。
- **唯一豁免 = 印泥朱 `#d06055`**，口径对齐配置注释「品牌识别色……行内 `// hex-exempt`」（L35–36）。落点建议在 TSX（w-seal 内联样式或局部 CSS 变量）配行内豁免注释——CSS 文件虽不在 lint 范围，但该文件自我承诺零 hex（L13–14），落 CSS 须同步修订头注释把印泥朱列为唯一例外。验收：`npx eslint .` 零错误，全仓 raw hex 净增恰 1 处且带豁免注释。

### 5.5 降级与响应式

- **降级补挂新类**：`prefers-reduced-motion` 块（CSS L1072–1080）与 `.ui-reduced-motion` 规则（L1081–1084）需补 `.w-plaque:hover { transform: none }`；`gaea-raf-degraded` 通配（L1060–1064）已覆盖一切携带 `.v3-rise` 的新件，seal 的 -3° 为静态 transform 无需降级，若给 seal 加入场动画则必须走 `.v3-rise`。
- **响应式三档**沿用 v8 断点位：≤1180 主栏塌单列 + `w-vitals` 转 2×2（继承 L1099–1101 对 `w-rail-panel` 的处理）、≤900 账页收列 / 案牌降单列（继承 v8 `w-index` 单列语义，L1110）、≤640 命令条换行 + vitals 单列（L1118–1128）。**移动端仅藏 `w-mast-pill`**（就绪徽记信息价值最低），`ml-space-chip` 必须保留（testid 契约 + title 提示语义）；@1400 档只涉闲庭，不动。

## 6. 验证

### 6.1 既有用例（基线，4 例必须原样全绿）

| # | 用例 | 行号 | v9 敏感点 |
|---|---|---|---|
| 1 | 空间切换器 aria-pressed + 回调 | L43–57 | 无（SpaceSwitch 不动） |
| 2 | 切换钮/chip 的 title「不影响办公引擎空间」 | L59–65 | chip 迁 masthead 后 testid 仍可查 |
| 3 | 书斋：desk-recent-docs + chip 文案「书斋」 | L67–73 | 账页不动；chip 文案来源不变 |
| 4 | 闲庭：.garden-banner 在、书斋件不在 | L75–83 | 闲庭零改动的直接断言 |

### 6.2 新增 v9 结构用例（计划，4 例）

| # | 断言 |
|---|---|
| 5 | 书斋渲染 `w-masthead`：`.w-seal` 存在且 `aria-hidden`；标题文案 = home.title「文书台」；`ml-space-chip` 位于 masthead 内 |
| 6 | 能力目录案牌：`.w-plaques` 内 ≥1 个 `.w-plaque`；`.w-index-num` 等宽序号不再渲染（queryBy 为 null） |
| 7 | 右舷脉息：`.w-vitals` 存在且含 4 节；首节为写作（首节内含 `.ml-ring`） |
| 8 | 闲庭不受影响：play 空间下 `.w-masthead` / `.w-plaques` / `.w-vitals` 均 queryBy null；`.garden-banner` 仍在 |

### 6.3 工具链（主代理已于 2026-09-15 统一执行完毕）

| 命令（frontend 目录） | 验证点 | 结果 |
|---|---|---|
| `tsc -b --pretty false` | IndexItem→案牌件改写无死引用、新类名/组件签名、i18n 键引用 | ✅ 零错误 |
| `vitest run src/components src/stores` | ModuleLauncher.test.tsx 8 用例（4 既有 + 4 新增）全绿；既有套件不受影响 | ✅ 55 文件 / 344 用例全过（含 ModuleLauncher 8/8、AppearancePanel 6/6） |
| `eslint ModuleLauncher.tsx ModuleLauncher.test.tsx` | 零新告警（no-raw-hex 只覆盖 ts/tsx；印泥朱 #d06055 落在 CSS，逐处带 hex-exempt 注释） | ✅ 零错误零警告 |

手工冒烟建议：暗夜青/亮色两档下 masthead hairline 与印章对比度；reduced-motion 开关下案牌 hover 无位移、rAF 降级下无元素停在 opacity:0；1180/900/640 三档宽度塌缩与移动端 chip 可见；闲庭页逐目对照零变化。
