# 灵犀：思考卡 / 工具卡 / 过程卡 逆向与完整样式

> 目标：lingxi-desktop 1.3.36（webBuild 26b61d0a2d85）
> 取证：`out\drd\desktop-renderer-dist\assets\*`
> 证据等级：`观察`=读到代码/CSS；`推断`=结构推导未运行验证

---

## 0. 三类卡的关系

```
PClawMessageGroup（消息组，index-BCzbcUJw.js，scope data-v-fbe4b4b9）
├─ .claw-message-group-placeholder          历史消息骨架（IntersectionObserver 懒挂载）
├─ .claw-process-collapse  ◀── 过程卡（折叠条）
│    └─ .claw-process-collapsible > __inner
│         └─ content[] 逐项 → chat.resolveRenderer(item)   ← 渲染器由宿主注入
│              ├─ PClawToolCard        （工具卡基座，PClawExpertInfo-DJhxJFWu.js，data-v-3b788e79）
│              │    ├─ PClawUrlFetchCard / PClawSearchCard / PClawImageSearchCard
│              │    ├─ PClawFileOpCard / PClawOpenOnlineFile / PClawMemoryCard
│              │    ├─ PBashAuthCard / PClawPermissionCard / PClawConnectorToolCard
│              │    └─ PClawToolCard(通用，title=E6r 类别)
│              └─ PMessagePlainCard    （思考卡/能力卡基座，data-v-d1f641c3 / data-v-43d13d74）
│                   ├─ 思考：title="思考已完成"，正文用 .ai-reasoning markdown
│                   └─ 其它："智能体正在运行" / "智能体运行完成"
└─ .cowork-waiting-answer                   等待回答 / bash 授权 药丸
```

关键结论：**三类卡共用一套排版语言**——`28px` 高 header、`8px` 圆角、`16px` 图标位、`13px` 正文、`--kd-color-text-secondary` 文字色、右上/下方各有 `max-height` 上限。思考卡与工具卡只是"基座 + 不同插槽内容"，过程卡是它们的容器。

---

## 1. 思考卡：`PMessagePlainCard`

### 1.1 两个变体（观察）

| 文件 | scope | 差异 |
|---|---|---|
| `PMessagePlainCard-GPsxyt3m.js` | `data-v-d1f641c3` | **无**折叠箭头（由 `PAgentModeSpotlight` 承担标题动画） |
| `PMessagePlainCard-BEaX2-5x.js` | `data-v-43d13d74` | 多 `showArrow` prop + `.header__arrow`（`--folded` 时 `rotate(-90deg)`） |

### 1.2 props 与状态推导

```js
props: { message, title, icon,
         enableExpand: Boolean = true,
         showArrow: Boolean = true,      // 仅 43d13d74
         completed, initFolded }
```

```js
const folded   = ref(props.initFolded ?? false)
const isDone   = computed(() => props.completed === true ? true
                             : props.message?.state ? props.message.state === State.End : false)
const title    = computed(() =>
    props.title || props.message?.content?.title ? (props.title || props.message.content.title)
  : isDone.value ? "智能体运行完成" : "智能体正在运行")

watch(isDone, done => { if (done && enableExpand) { folded.value = true; setTimeout(() => scroller.resetMinHeight(), 300) } },
      { immediate: true })
```

- **默认标题只有两句**：未完成 `智能体正在运行`，完成 `智能体运行完成`；可被 `title` prop 或 `message.content.title` 覆盖（思考卡传 `"思考已完成"`）。
- `initFolded` 默认 `false`（展开）；完成时**自动展开一次**（`folded = true` 表示"已折叠过/默认收起"，注意语义与字面相反：`initFolded=false` → 初始展开）。
- 每次折叠切换后 `setTimeout(...,300)` 调 `scroller.resetMinHeight()`——消息滚动容器需要重新测量高度（配合 `expand-content` 的 0.3s 过渡）。

### 1.3 DOM

```html
<div class="card card--completed?">
  <div class="header header--can-fold?" @click="toggle">
    <Icon class="header__icon lx-icon--current-color" v-if="icon" />
    <span class="title-text" v-if="isDone">{{ title }}</span>
    <PAgentModeSpotlight class="title-spotlight" :text="title" :spotlight="true" v-else />
    <ArrowRight class="header__arrow header__arrow--folded?" v-if="enableExpand && showArrow"/>
  </div>
  <Transition name="expand-content">
    <main class="main" v-show="!folded">
      <div class="main__list"><slot name="main"/></div>
    </main>
  </Transition>
</div>
```

未完成时标题用 `PAgentModeSpotlight`（文字扫光动画），完成后换成静态 `title-text`——这就是"思考中"文字在闪的效果来源。

### 1.4 思考正文

正文是 markdown，套 `.ai-reasoning`：

```css
.ai-reasoning{
  --lx-md-font-size-text:13px;
  --lx-md-line-height-text:var(--kd-font-line-height-small,20px);
  color:var(--kd-color-text-secondary);
}
.ai-reasoning p,.ai-reasoning ol,.ai-reasoning ul,.ai-reasoning section{font-size:13px;line-height:20px}
.cowork-share-markdown.ai-reasoning--cloud{margin-left:8px;color:var(--kd-color-text-tertiary)}
```

即：思考内容**比正文小一号（13px）、用 secondary 灰**，链接/引用降级到 tertiary。分享页再多 8px 左缩进。

---

## 2. 工具卡：`PClawToolCard`

### 2.1 实现（观察）

```js
props: { message, title, statusText, icon,
         enableExpand: Boolean = false,   // 默认不展开
         isError: Boolean = false,
         canOpen: Boolean = false }       // 能否进工作台
```

```js
const isDone   = computed(() => message?.state === State.End)
const scroller = inject("scroller"); const openInWorkspace = inject("openComponentInWorkspace", () => {})
const highlighted = inject("highlightedComponentId") === message.id

const interactionOk  = messagePresentation?.allowToolResultInteraction !== false
const inlineExpandOk = messagePresentation?.allowInlineToolExpansion ?? interactionOk
const canOpenWs      = interactionOk && canOpen && allowWorkbenchNavigation !== false
const clickable      = canOpenWs || (inlineExpandOk && enableExpand)

function onClick() {
  if (!clickable) return
  if (canOpenWs) { openInWorkspace(message.id); return }        // 优先：进工作台
  if (inlineExpandOk && enableExpand) { expanded = !expanded; setTimeout(() => scroller?.resetMinHeight(), 300) }
}
watch(isDone, d => { if (d && enableExpand) { expanded = true; setTimeout(...,300) } }, { immediate: true })
```

**点击优先级**：`canOpen`（进工作台） > `enableExpand`（就地展开）> 不可点。

### 2.2 DOM

```html
<div class="claw-tool-card">
  <div class="claw-tool-card__header claw-tool-card__header--clickable?
              claw-tool-card--highlighted?" @click>
    <img class="claw-tool-card__icon image__url__icon" v-if="typeof icon==='string'">
    <Icon class="claw-tool-card__icon lx-icon--current-color" :size="16" v-else />
    <!-- 完成且不可展开 → 静态标题；否则 Spotlight 动画标题 -->
    <span class="claw-tool-card__title-text" v-if="isDone && !enableExpand">{{ title }}</span>
    <PAgentModeSpotlight class="claw-tool-card__title" :text="title" :spotlight="!isDone && !isError" v-else />
    <span class="claw-tool-card__status-text" v-if="isDone && statusText">{{ statusText }}</span>
    <slot name="desc"/>
  </div>
  <Transition name="claw-tool-card-expand">
    <div class="claw-tool-card__body" v-if="inlineExpandOk && enableExpand && !canOpenWs && !expanded">
      <slot name="main"/>
    </div>
  </Transition>
</div>
```

### 2.3 工具名 → 类别 / 中文名（观察）

`E6r`：工具名 → header 上显示的一级分类标签

| 工具 | 类别 | 工具 | 类别 |
|---|---|---|---|
| `browser_action` | Browser | `read_file` / `write_file` / `edit_file` / `list_files` | File |
| `execute_code` | Python | `speech_to_text` | Audio |
| `bash` | Bash | `get_memory` / `edit_memory` / `write_memory` / `wiki_memory` | Memory |
| `install_package` | Shell | `image_container` / `view_image` | Image |
| `websearch` / `image_search` / `academic_search` / `macro_search` | Search | `open_online_file` | Doc |
| `deliver_result` | — | `generate_ppt_block` | PPT |
| `connector_tool_result` / `connector_manage_result` | Connector | | |

`Tve[name]`：工具名 → 中文（如 `代码执行`、`失败` 后缀 `${Tve[n]} - 失败`）。
`k6r[name]`：工具名 → 详情面板标题（`代码详情 / 文件详情 / 搜索详情 / 记忆详情 / 文档详情 / 浏览器详情 / 图片详情`）。

### 2.4 工具结果归一化 `kb(item)`

```js
// 出错：content 是 {err,...}
{ contentType:"error", title: content.brief || `${Tve[name]} - 失败`,
  displayContent: content.err, codeContent: 可执行类取 code/command, codeLanguage: bash|python }

// 正常：按工具名分支
execute_code → { contentType:"image"|"text", title: content.brief || "代码执行 - 图表", displayContent, codeContent }
bash / install_package → {...}
websearch / *_search → 命中列表（renderer 里 `I6r` 取首条 title 或 "N 条结果"）
read_file / write_file / edit_file / list_files → 文件行
```

`Hai(item)` 最终产出工作台步骤数据：`{ category, description, code, language, isBash, resultLine }`。
`ch(text, 72)`：取第一行非空、长度上限 72 的摘要。

### 2.5 子类卡（同一个 header，换插槽）

| 卡 | title | desc | main |
|---|---|---|---|
| `PClawUrlFetchCard` | `content.brief \|\| "获取网页"` | `N 个网页` | 列表：域名 + `compass`(成功) / `validation_circle`(失败) |
| `PClawSearchCard` / `PClawImageSearchCard` | 搜索摘要 | 结果数 | 命中列表 |
| `PClawFileOpCard` | 文件操作摘要 | — | 文件行 |
| `PClawMemoryCard` | 记忆摘要 | — | 记忆条目 |
| `PBashAuthCard` / `PClawPermissionCard` | 授权文案 | — | 允许/拒绝按钮 |

---

## 3. 过程卡：折叠条 + 步骤容器

### 3.1 结构（观察，`PClawMessageGroup`）

```html
<button class="claw-process-collapse" @click="toggle">
  <ArrowRight class="claw-process-collapse__arrow is-expanded?" />
  <span class="claw-process-collapse__text">{{ 文案 }}</span>
</button>
<div class="claw-process-collapsible is-expanded?" @transitionend="onEnd">
  <div class="claw-process-collapsible__inner">
    <!-- 每个 content 项一套 -->
    <div data-tool="<toolName>">…resolveRenderer(item)…</div>
  </div>
</div>
```

折叠动画用 `grid-template-rows: 0fr → 1fr`（可以动画到内容高度，比 `max-height` 精确）：

```css
.claw-process-collapsible{display:grid;grid-template-rows:0fr;transition:grid-template-rows .35s cubic-bezier(.4,0,.2,1)}
.claw-process-collapsible.is-expanded{grid-template-rows:1fr}
.claw-process-collapsible__inner{overflow:hidden;min-height:0;opacity:0;transition:opacity .2s ease}
.claw-process-collapsible.is-expanded .claw-process-collapsible__inner{opacity:1;transition:opacity .2s ease .1s}
```

### 3.2 展开/收起的定时器编排（观察）

```js
function toggle(key) {
  clearTimeout(timers.get(key))
  state[key] ??= { expanded:false, mounted:false }
  if (!state[key].expanded) {
    state[key].mounted = true
    const panel = e.currentTarget.nextElementSibling
    nextTick(() => { state[key].expanded = true
      setTimeout(() => panel?.querySelector("[data-tool]")?.scrollIntoView({behavior:"smooth",block:"nearest"}), 380) })
    return
  }
  state[key].expanded = false
  timers.set(key, setTimeout(() => unmount(key), 400))   // 收起后 400ms 才卸 DOM
}
function onTransitionEnd(e){ // 只为 grid-template-rows 触发，且未展开时卸载
  if (e.target !== e.currentTarget || e.propertyName !== "grid-template-rows" || state[key]?.expanded) return
  unmount(key)
}
```

- 展开：先 `mounted = true` → 下一帧置 `expanded` → **380ms 后**把首个 `[data-tool]` 平滑滚进视野。
- 收起：`expanded=false` → **400ms**（过渡 350ms + 余量）后才 `mounted=false` 卸载，避免动画中途丢节点。

### 3.3 可折叠判定与文案（观察到的字面量）

- 组级开关：`message.extra._hasCollapsibleProcess`；单项开关：`vt.has(name)`（白名单）或 `Is(content)`。
- 文案（取自产物中真实字符串）：

| 场景 | 文案 |
|---|---|
| 运行时 | `正在思考中` |
| 完成（通用） | `智能体任务已完成` / `智能体运行完成` |
| 研究类 | `已完成研究` / `研究已完成` |
| 思考块标题 | `思考已完成` |
| 工具栏 | `实时更新中，点击跳到最新步骤` / `点击查看最新步骤` |

### 3.4 消息组虚拟化参数（观察）

```js
const MAX_ENTRIES = 300                       // 高度缓存 LRU 上限
const OFFSET_HINT = 10
placeholderHeight = clamp(min(1200, max(120, content.length * 96)), …)   // 估算占位高度
IntersectionObserver root: closest(".chat__main"), rootMargin: "1200px 0px"
```
只有**最后 3 组**保持实时渲染（`index >= list.length - 3`），更早的组按测量高度撑出占位，滚到附近 1200px 才真挂载。

---

## 4. 工作台步骤卡：`PWorkbenchStepCard`（data-v-a388dd25）

过程卡点进工作台后的"大卡"，定高 96px、内部裁切可滚：

```css
.workbench-step-card{box-sizing:border-box;width:100%;height:96px;max-height:96px;
  background:var(--lx-background-group);border:1px solid var(--lx-color-line-light);
  border-radius:12px;padding:12px 16px;overflow:hidden}
.workbench-step-card__clip{max-height:72px;overflow-y:auto;contain:layout paint;scrollbar-width:none}
.workbench-step-card__code{font-family:ui-monospace,SFMono-Regular,Menlo,Monaco,Consolas,monospace;
  font-size:12px;line-height:18px;white-space:pre-wrap;word-break:break-word;color:var(--lx-color-text-secondary)}
```

- 三种内容形态：`code`（高亮 / 流式）、`diff`（`--del` 红 / `--add` 绿，左侧 2px 竖条 + `−/+` 标签）、`skeleton`（两行 shimmer，1.4s 循环，`--long 92%` / 默认 78% / `--short 58%`）。
- 流式：`.workbench-step-card__stream-code{transform-origin:left top;will-change:transform}`，配合 `translate3d` 只滚动增量；`__stream-line{min-height:18px}` 保证行高稳定不抖。

---

## 5. 三类卡统一视觉规范（可直接抄）

| 项 | 值 |
|---|---|
| header 高 | `28px`（`padding:4px 8px`，`border-radius:8px`，`gap:8px`） |
| 图标位 | `16×16`，色 `--kd-color-icon-tertiary`(#757575) |
| 标题字号 | `var(--kd-font-size-sub-base)`（13px） |
| 标题色 | `--kd-color-text-secondary` |
| 状态文字 | `12px`，`--kd-color-text-tertiary` |
| 正文色 | `--kd-color-text-secondary`，13px/20px |
| 折叠容器上限 | 思考卡 `main max-height:160px`；工具卡 body `min(400px,45vh)`；卡片总高 `min(480px,50vh)` |
| 过渡 | `all .3s cubic-bezier(.4,0,.2,1)`；透明度 `opacity .2s ease` |
| 悬停底 | `--lx-color-state-hover`（深色下 `#ffffff14`） |
| 高亮底 | `--lx-color-state-pressed` / `claw-tool-card--highlighted` |

---

## 6. 完整 CSS

见同目录 [`cards.css`](cards.css)（思考卡 / 工具卡 / 过程卡 / 步骤卡 四套，已剥离 scoped 属性，含深色主题覆盖）。

关键补充变量（原产物里由设计系统提供，复刻时给 fallback）：

```css
--lx-background-group:#fafafa;      /* 深色 #171717 */
--lx-color-line-light:rgba(var(--kd-color-gray-10),.06);
--lx-color-fill-regular:rgba(var(--kd-color-gray-10),.06);
--lx-color-text-secondary:rgba(var(--kd-color-gray-7),1);
--lx-color-text-tertiary:rgba(var(--kd-color-gray-10),.46);
--lx-color-state-hover:rgba(var(--kd-color-gray-10),.06);
--lx-color-state-pressed:rgba(var(--kd-color-gray-10),.1);
--kd-color-error-normal:rgba(220,60,60,.8);
--kd-color-error-light:rgba(227,77,89,.08);
--kd-color-success-normal:rgba(40,160,40,.8);
--kd-color-success-light:rgba(0,168,112,.08);
--kd-font-size-sub-base:13px;
--kd-font-line-height-small:20px;
```

---

## 7. 状态机（三卡共用）

| message.state | 思考卡 | 工具卡 | 过程卡 |
|---|---|---|---|
| `Start` / `Content` | 标题 spotlight 扫光，`智能体正在运行` | spotlight 标题，可展开但**不自动** | 折叠条显示"正在思考中" |
| `End` | 标题静态，`智能体运行完成`（或 `思考已完成`） | 静态标题 + `statusText`；`enableExpand` 时**自动展开一次** | 折叠条显示已完成文案，可收起 |
| 出错 | `isError` 传下去（工具卡 spotlight 关闭） | `kb()` 返回 `contentType:"error"`，标题 `${中文名} - 失败` | 错误项在列表内红色图标 |

---

## 8. 复刻建议

1. **别为三种卡写三套组件**：一个 `CollapsibleHeaderCard`（header+折叠）就能同时撑起思考卡与工具卡，差异只在插槽内容与默认 `enableExpand`。
2. **`canOpen` 与 `enableExpand` 必须互斥**：原版 `canOpenWs` 为真时点击直接进工作台、**不再就地展开**（`v-if` 里同时判了 `!canOpenWs`），否则一次点击两种行为。
3. **展开/收起要配 `resetMinHeight()` 与 380/400ms 定时器**：虚拟滚动 + 高度过渡的组合拳，少任何一环都会出现"展开后跳位/收起后残留"。
4. **折叠用 `grid-template-rows: 0fr→1fr`**，不要用 `max-height`：步骤项高度不定，固定 `max-height` 会导致大段留白或裁切。
5. **文案走"运行中/已完成"两态**：`智能体正在运行` → `智能体运行完成`，思考块单独用 `思考已完成`；这是这套 UI 最容易被用户感知的一致性细节。
