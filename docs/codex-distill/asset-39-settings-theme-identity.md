# 资产 39 · 设置面：分区、可搜索配置项、不可用原因、实时预览

> 证据根（只读）：`config_panel/{mod,render,state}.rs`（472/520 行）、`theme_picker.rs`（631）、
> `terminal_palette.rs`（556）、`config_update.rs`（463）、`local_settings.rs`（278）、
> `bottom_pane/{status_line_setup,title_setup,multi_select_picker,experimental_features_view,oss_selection}.rs`、
> `cwd_prompt.rs`（587）、`external_editor.rs`（316）。
> 对照根（只读）：`frontend/src/pages/SettingsPage.tsx`（11.9KB）、`frontend/src/components/VoiceSettingsPanel.tsx`（40.9KB）、
> `frontend/src/gaea/components/{CapabilitiesPanel.tsx,KnowledgePanel.tsx,MemoryPanel.tsx}`、`frontend/src/pages/modelcenter/*`、
> `frontend/src/gaea/lib/theme*` / `design-system/`。
>
> 只出机制与落点；不改 gaea 源码。

---

## 一、设置是五段分区，分区表只有一份

`config_panel/mod.rs:191-218`：

| 段 | 标题 |
|---|---|
| `Appearance` | Appearance |
| `Notifications` | Notifications |
| `Input` | Input |
| `Sessions` | Sessions |
| `TextSelection` | Text selection |

`ConfigSection::ALL: [Self; 5]`（`:201-207`）是渲染顺序的唯一来源，标题由 `title()` 派生（`:209-217`）——
**没有第二处再列一遍分区**。设置为空的段也不会凭空出现（列表由 `settings()` 生成，逐项带 `section`）。

## 二、配置项是五元组，`keywords` 是它能被搜到的原因

`ConfigSetting{section, label, description, keywords, value}`（`mod.rs:220-226`）——
`keywords` 与 label/description 并列存储，供搜索索引（设置页的过滤不是"在渲染文本里 contains"）。

`ConfigSettingValue` 五形态（`:228-247`）+ 派生标签（`:249-266`）：

| 形态 | 字段 | 显示的字符串 |
|---|---|---|
| `Toggle` | `setting, enabled` | `[x] On` / `[ ] Off` |
| `Editor` | `label, editor` | `{label} ›`（子页面入口） |
| `DependentToggle` | `dependency_met, unavailable_reason` | 满足走 On/Off；**不满足则 `Unavailable`** |
| `NotificationTest` | — | `Send in 5s`（可当场验证通知） |
| `Unavailable` | `reason` | `Unavailable` |

两条纪律藏在这里：
1. **依赖未满足时不隐藏，显示 `Unavailable`**——用户仍能看见"有个设置存在，但当前不可用"，不会以为功能不存在；
2. **`NotificationTest` 是一个可执行项**：通知类设置必须能当场试一次，而不是让用户等下一个任务结束。

## 三、主题是**实时预览**，且预览样本有下限

`theme_picker.rs`：
- `NARROW_PREVIEW_ROWS: [PreviewRow; 4]`（`:59-82`），注释写明「紧凑回退预览，**始终保证一删一增可见**」；
- `WIDE_PREVIEW_ROWS: [PreviewRow; 8]`（`:84-`），"刻意混入 context / added / removed"；
- `WIDE_PREVIEW_MIN_WIDTH = 44`（`:130`）、`WIDE_PREVIEW_LEFT_INSET = 2`（`:133`）、`PREVIEW_FRAME_PADDING = 1`（`:136`）；
- `PREVIEW_FALLBACK_SUBTITLE = "Move up/down to live preview themes"`（`:138`）——**上下移动即换主题、即时上色**，不要求确认。

`terminal_palette.rs`：`StdoutColorLevel`（`:7`）+ `DefaultColors`（`:92`）+ `XTERM_COLORS: [(u8,u8,u8);256]`（`:235`），
即"我用的是哪个色号体系"也要探测，256 色表是唯一事实来源。

## 四、可访问性偏好**不来自设置页**，来自系统

`local_settings.rs:24-79`：`LocalSettings` 聚了 11 组 tui 设置（notifications/animations/effects/rendering/tooltips/auto_recap/
paste_burst/vim/question_esc_back/raw_output/fullscreen/mouse_scroll/copy_on_select/alternate_screen/status_line/terminal_title…），
但构造入口是 `with_accessibility_preferences(config, system_motion::mode(), screen_reader::animation_default())`。

即：**动效强度由系统（prefers-reduced-motion / 读屏器）决定，配置文件只提供默认值**。
这条与 F33 的"动效偏好启动播种 + JS 显式降级"是同一条纪律的终端版。

## 五、多选器：取消恒可用、必选可控、重排可选

`bottom_pane/multi_select_picker.rs:1-23` 的文件头把契约写完了：

- 模糊搜索过滤；空格切换；**可选"必须非空才能确认"，但取消永远可用**；
- 可选左右键重排（`orderable`）；可选实时预览回调；change/confirm/cancel 三回调；
- **checkbox 与聚焦行独立**（"full-width focus with overflow hints in spacer rows; checkboxes remain independent of the focused row"）；
- 名称截断 `ITEM_NAME_TRUNCATE_LEN = 21`、搜索占位 `Type to search`、分节行 `SECTION_BREAK_ROW`。

## 六、动作文案由动作派生

`cwd_prompt.rs:37-60`：`CwdPromptAction{Resume, Fork}` 带 `verb()`（`resume`/`fork`）与 `past_participle()`（`resumed`/`forked`），
用于"已 resume 到 X"/"已 fork 到 X"这类**结果句**；`CwdSelection{Current, ...}` 是选择项。

`config_update.rs:44-50,425`：`ProjectTrustHost` / `RemoteProjectTrust` / `EffectiveConfig`——
**"最终生效的配置"是一个显式结构**，不是"读一堆字段自己拼"。

## 七、gaea 现状核对

| gaea | 现状 | 差在哪 |
|---|---|---|
| `pages/SettingsPage.tsx`（11.9KB） | 设置页壳，实际内容在子面板（`VoiceSettingsPanel` 40.9KB、`CapabilitiesPanel` 11.6KB、模型中心等） | 无"分区 + 可搜索配置项"的统一列表；设置项散在各面板，用户只能按导航找 |
| `design-system/` + 明暗主题 | 主题切换存在 | 无**实时预览样本**（点一下才生效）；无"一删一增"最小预览 |
| 无 | 配置项 `keywords` 搜索 | 缺；`CommandPalette` 顶层搜索也搜不到设置项 |
| 无 | `DependentToggle` 的 `Unavailable` 表达 | 依赖不满足时多为隐藏或不渲染 → 用户以为没这个功能 |
| 无 | 实验特性开关页 | 缺（codex 有 `experimental_features_view`） |
| 无 | 状态行 / 窗口标题模板定制 | 缺（gaea 的状态栏是固定组件） |
| 无 | 多选器（必选/可选、重排、实时预览） | 缺通用件；现有选择多为单选 + 复选框直排 |

## 八、三档判定

**档一（照搬语义，成本低收益高）**

1. **配置项 keywords 搜索**：每个设置项带 `keywords`，命令面板与设置页共用同一个索引（`CommandPalette` 已存在）。
2. **不可用要说明原因**：`Unavailable{reason}` 形态，不隐藏。
3. **动作派生文案**：结果句由动作枚举派生（`resumed`/`forked` 类），避免每个调用点手写。
4. **多选器契约**：取消恒可用、必选可配、checkbox 与聚焦行解耦。

**档二（改造搬）**

5. **五段分区 + `ALL` 单一来源**：把现有设置面板重排成"分区 → 配置项"两层，渲染顺序只有一份。
6. **主题实时预览**：选中即上色（CSS 变量注入），并保留 `Move up/down` 式的操作提示；预览样本固定含新增/删除/上下文三类。
7. **`EffectiveConfig` 投影**：设置页显示"最终生效值 + 来源"（默认/项目/全局/会话），而不是只显示当前值。

**档三（不搬）**

8. `XTERM_COLORS` 256 色表、`StdoutColorLevel`、终端标题模板（`title_setup`）、状态行模板（`status_line_setup`）——
   终端概念；Web 侧的等价物是 CSS 令牌与 `document.title`，语义不同，不硬套。

## 九、施工卡

| 卡 | 落点 | 改动 | 验收 |
|---|---|---|---|
| **F39-A 设置索引** | 新 `gaea/lib/settingsIndex.ts`（`{id, section, label, keywords[], value, unavailableReason?}`） | 设置项集中声明；`CommandPalette` 增加"设置"分组 | 单测：搜 `关键词` 命中；未命中返回空；不可用项的 reason 出现在结果副标题 |
| **F39-B 不可用表达** | `SettingsPage` 渲染层 | 依赖不满足 → 灰态 + 原因 tooltip（不隐藏） | 单测：父开关关 ⇒ 子项仍在列表且标记不可用+原因 |
| **F39-C 主题实时预览** | `design-system` 注入层 | 悬停/聚焦即预览；`Move up/down` 提示；预览含 add/del/context 三类高亮 | 手测：切 10 次主题无闪烁；预览样本三类色齐 |
| **F39-D 多选器** | 新 `components/ui/MultiSelect.tsx` | 模糊搜索 + 空格切换 + `requireNonEmpty` + 可选重排 + 实时预览回调 | 单测：空选不可确认但可取消；重排后顺序回调；21 字符截断 |

**开关**：F39-A/D 为新增件，`FEATURE_SETTINGS_INDEX` 单闸；F39-C 只改注入时机（预览不落盘）。

## 十、风险

1. **设置写路径**：gaea 的设置既落 Go 配置又落前端 store（`useModelManager` 等），F39-A 只做**读取索引**，写仍走既有绑定，避免双写。
2. **`perf-guards`**：设置索引若在命令面板每次打开时重建，必须 memo 到"设置版本"。
3. **不可用原因要真**：填 `reason` 前必须确认依赖关系在代码里真实存在，不许写"暂不可用"糊过去（诚实纪律）。

## 十一、证据自查

- 分区与 ALL：`config_panel/mod.rs:191-218`；五元组与 keywords：`:220-226`；五形态与派生标签：`:228-267`。
- 主题预览：`theme_picker.rs:59-140`；调色板：`terminal_palette.rs:7,92,235`。
- 可访问性注入：`local_settings.rs:24-79`。
- 多选器契约：`bottom_pane/multi_select_picker.rs:1-23,87-92`。
- 动作派生文案：`cwd_prompt.rs:37-60`；生效配置：`config_update.rs:44-50,425`。
