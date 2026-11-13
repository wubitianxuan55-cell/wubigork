# 灵犀对话窗口 UI 与「文件成果卡」逆向（1.3.36 / webBuild 26b61d0a2d85）

> 取证：`re-lingxi\out\drd\desktop-renderer-dist\assets\*`（渲染层 bundle + css）、`out\x\out\main\index.js`
> 证据等级：`观察`=直接读到代码/CSS；`推断`=结构推导未运行验证

---

## 0. 结论摘要

1. 对话窗口是**一个三层 flex 容器**（`#lingxi` → `.chat__core` / `.drag-line` / `.canvas-container#lingxi-canvas` / `#lingxi-drawer`），消息区、输入区、右侧面板互不嵌套，靠 `canvasWidthPercent` 与拖拽条联动。
2. 「文件成果卡」不是单一组件，而是**四层组合**：
   `PClawPresentFiles`（2 列 grid 容器） → `PClawDeliverResult`（行为层：点击路由/右键菜单/水印/阻断） → `PClawFileCard`（`claw-file-card-view` 定位壳） → `PMessageAIFileCardInline`（视觉层：`file-card-inline` + 按类型渐变底色 + 后缀图标）。
3. 卡片的**全部交互逻辑都在 `PClawDeliverResult`**，共 4 个入口：单击 `ce()`、双击 `Vt()`、右键 `Ut()`、自动打开 `Bt()`；每个入口都先过同一个 `openProducedFile` 拦截钩子与 `artifact blocked` 门闸。
4. 左上角"文件身份"只在 `message.content` 里：`{name, ref, type}`；`ref` 支持 `file://` / 本地盘符 / `sandbox:/...` / `/home/lingxi/...` / `https://` / kdocs 链接 六种形态，靠 `Vn()` / `O()` / `Le()` 三个函数归一化。
5. 卡 → 右侧面板的落点是 `openCanvas({id: link_id, name, type: office_type, url, filePath})`，其中 `link_id/office_type` 来自 `POST /api/aioffice/v1/sandbox/file/preview`；`html` 类走 `openCanvas({type:"html"})`；`OTL/KSHEET/DB` 与无 office_type 的走外部浏览器。

---

## 1. 对话窗口布局骨架（观察）

`localSessionLifecycle-BbrY1I2c.js`（`PChatLayout`，scope `data-v-e758d729`）：

```html
<div id="lingxi" class="lingxichat allow-long-press
        lingxichat--canvas-active lingxichat--canvas-fullscreen">
  <div class="chat__core" style="width:<canvasWidthPercent>px">
      <header>…PChatHeader…</header>                 <!-- 可选，showHeader -->
      <PChatMain ref=chatMainRef>                     <!-- 消息滚动区 U8n -->
          slots: loading / message-tail / tips / welcome / excel-guide
      </PChatMain>
      <PChatFooter>…PChatFooter 输入区 X8n…</PChatFooter>   <!-- 可选，showFooter -->
  </div>
  <div class="drag-line inDrag?"></div>               <!-- 分栏拖拽 -->
  <div class="canvas-container active|inDrag">
      <div id="lingxi-canvas"></div>                  <!-- 右侧面板挂载点 -->
  </div>
  <div id="lingxi-drawer" class="chat__drawer"></div>  <!-- 抽屉层 -->
</div>
```

CSS（`main-CH_U5GfM.css` / `filePreview-ld9_K1oO.css`）：

| 选择器 | 规则 |
|---|---|
| `.canvas-container` | `width:0; padding:20px 20px 20px 5px; margin-right:-25px; transition:all .3s` |
| `.canvas-container.active` | `margin-right:0` |
| `.lingxichat--canvas-active .canvas-container` | `flex:1; min-width:350px` |
| `.lingxichat--canvas-fullscreen .canvas-container.active` | `padding:0` |
| `#lingxi-canvas` | `height:100%` |
| `.chat__drawer` | `height:100%; flex-shrink:0; overflow:hidden` |

消息气泡间距（`data-v-fbe4b4b9` / `data-v-da769498`）：

```css
.message-item > :first-child:not(.claw-present-files){ margin-top:0 }
.message-item--assistant:not(:first-of-type):not(:has(>.claw-present-files)){ margin-top:4px }
.message-item--assistant:not(:first-of-type):has(>.claw-present-files){ margin-top:16px }
```

> 即：**带成果卡的那条消息额外多 12px 上间距**——成果卡是消息组里的"重块"。

---

## 2. 消息数据模型（观察）

```ts
message = {
  id, groupId, role: User|Assistant|Process…, name, state,
  content: {
    text,                                // markdown 源
    files?: [{ id, drive_id?, cc_file_id?, name, size, type, session_id, group_id }],
    items?: [{ name, ref }],             // ← 成果卡数据源（PClawPresentFiles）
    name?, ref?, type?,                  // ← 交付物单体（PClawDeliverResult）
    env?, deprecated_group_id?, …
  },
  extra: { websearches?: [{results:[{title,url,favicon}]}], agentId?, … }
}
```

- 用户消息：`PMessageItemUser`（`.claw-user-message-wrapper[data-groupid]`）→ 图片容器（`Xi`）+ 文件卡（`ri`）+ 收藏卡（`_i`）+ ask-answer 折叠。
- 过程流分组：`groups[]`，类型含 `Process` / `WebSearchEnd` / `DeepSearchReferenceEnd` / `SearchAidocs`，`processStream` 里再分 `websearch_end` 等子类型（`PMessageItemAIRendererMD` 的 `w()`/`C()` 就是在扫这些）。
- 办公页另有独立组件枚举（`PAiofficePage`）：`c_et_tool_call` / `c_et_openapi` / `c_text` / `c_reasoning` / `c_ordered_list` / `c_document_operation`。

---

## 3. 消息渲染栈

`PChatMain` 把每条 message 按 role 分发，AI 侧主要走 Markdown 渲染器：

| 层 | 组件 | 职责 |
|---|---|---|
| Markdown 内核 | `PMessageAIMarkdownPure`（712 KB）+ `PMessageAIMarkdown`（13 KB） | markdown-it + KaTeX + mermaid + 表格包装 `.md-table-wrapper` + 引用角标 `.md__linkx` |
| 特殊块 | `PMessageAIMermaid`、`PMessageAICitationToolTips`、`PMessagePlainCard` | 图/引用/hover 卡片 |
| 操作条 | `PMessageRetry`、`PMessageContextMenu`、`PMessageAIStop`、`PFeedBackPopover` | 重试/右键/停止/点赞点踩 |
| 语音 | `PMessageSpeechToText` | ASR 消息 |
| 卡片族 | `PClawFileOpCard`、`PClawToolCard`、`PClawSearchCard`、`PClawImageSearchCard`、`PClawMemoryCard`、`PClawPermissionCard`、`PMessageAICreationCard`、`PClawPresentFiles` | 工具/搜索/记忆/权限/创作/成果 |

`code-theme.language-lingxi-artifact` 这类代码块会被 `PCoworkClientShareMarkdown` 替换成内联资源卡（`Pt` 组件 + `onOpen: file_path`）——这是 markdown 里"夹带"文件卡的通道。

---

## 4. 文件成果卡：四层拆解（核心）

### 4.1 `PClawPresentFiles` —— 容器（观察）

文件：`PCoworkWorkbenchPanel.vue_…-CXBCXyjF.js`（导出 `a`）

```js
props: { message }
const items = computed(() => (message.content?.items ?? []).map((it, i) => ({
  ...message,
  id: `${message.id}:present-file:${i}`,
  content: { name: it.name, ref: normalize(it.ref), type: inferType(it.ref) }
})))
```

```html
<section class="claw-present-files" aria-label="本次生成的文件">
  <slot name="file" v-for="m in items" :message="m">
      <PClawDeliverResult message="m" presentFilesMode preferDisplayName disableSandboxDirectoryInference />
  </slot>
</section>
```

`normalize(ref)`：Windows 绝对路径 → `file:///E:/a%23b.docx`（`#`→`%23`）。
`inferType(ref)`：`image_url` / `local` / `cloud` / `web` / `""`（按 URL 查询参数 `lingxi_file_name|filename`、`isKdocs`、扩展名判断）。

CSS：

```css
.claw-present-files{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:8px;margin-top:4px}
@media(max-width:640px){.claw-present-files{grid-template-columns:minmax(0,1fr)}}
.claw-present-files .file-card-inline{max-width:none;margin:0}
```

### 4.2 `PClawDeliverResult` —— 行为层（观察，最重）

props：`message / autoOpenHtmlPreview / presentFilesMode / preferDisplayName / disableSandboxDirectoryInference`

依赖注入：`chat`、`canvasManager`、`adapter`、`clawWorkspace`、`openFilePreviewTab`、`requestSandboxFilePreviewTab`、`artifactBlocked({blocked, blockedReason})`、`coworkImageViewerActions`、`lxaiCanvasMode`、`share-artifacts` context。

**渲染分叉**

```
content.type === "image_url"  ||  (local 且扩展名是图片)
   → Gallery 组件 + 右上角「去除水印」按钮(28×28 → hover 展开 86px)
否则
   → PClawFileCard(fileName, filePath, displayName, fileExtension, isDirectory, icon?)
       · .claw-file-card-view
       · .claw-file-card-view--replace-icon 时把内层 .file-card-inline__icon 隐藏，
         用自定义 icon（.claw-deliver-result__file-icon，绝对定位 left:20px）
```

**文件身份归一化（关键函数）**

| 函数 | 作用 |
|---|---|
| `Vn(ref)` | 拆成 `{localPath, sandboxPath, trailingSeparator}`；`sandbox:/…`→sandboxPath，`/home/lingxi/…`→sandboxPath，`file:`→localPath，盘符/`/`→localPath |
| `O(url)` | `file://` → Windows 本地路径（盘符保留、`/`→`\`） |
| `Le(ref)` | `sandbox://` → `/…` |
| `ue(path)` | 统一分隔符（盘符路径转反斜杠） |
| `Hn(content, opts)` | 产出 `{fileName, displayName, displayFileName, cardFileName, cardFileExtension, fileIcon}`，名字优先取 URL 的 `lingxi_file_name`/`filename`，再 pathname 末段 |
| `qn(...)` | 目录推断：先 `wa(path, trailing)` 同步判，再 `isDirectoryPath(path)` 异步校验（异步结果只改一次，防竞态） |

**四个交互入口**

| 入口 | 触发 | 行为 |
|---|---|---|
| 单击 | `ee()` → `scheduleClick(ce)` | 400ms 防抖（区分双击）；`ce()` 路由见下 |
| 双击 | `Vt()` | 取消待执行单击；本地文件 → `isSupportedInFileBrowser` → `openInFileBrowser(path,{pageOpenDetail:"card_double_click"})` |
| 右键 | `Ut(e)` | 按 `kind` 分派菜单（见下表） |
| 自动开 | `onMounted → Bt()` | `autoOpenHtmlPreview && type!=="local"`，若 `cloud` 先 `Ze()` 补文件名/officeType，扩展名为 `html` 则 `await ce()`（打开预览） |

**`ce()` 单击路由（决策树）**

```
if actions.openProducedFile  → await Xe()            // 宿主自定义"打开产物"
if artifactBlocked            → toast(blockedReason) // 不打开
switch ze():
  web      → zt(): presentFilesMode ? (图片→viewer / 否则 Keep(链接) ) : window.open
  cloud    → Zt():
               linkId = xe(ref)                      // 从 kdocs 链接解析
               meta = Se(linkId, ref)                // 取 fileName / officeType
               officeType==="" && ext==="html" → openCanvas({id,name,type:"html"})
               !officeType || OTL|KSHEET|DB    → 外部浏览器
               actions.openRemoteFilePreview   → 交给宿主
               否则                            → openCanvas({id:linkId,name,type:officeType,url})
  其它     → qt(): sandbox 预览
               preview = at({sessionId, projectId}, sandboxPath)
                        // POST /api/aioffice/v1/sandbox/file/preview
               requestSandboxFilePreviewTab?.({filePath,fileName,url,linkId,officeType,projectId,expiredAt})
               openSandboxFilePreviewTab?.({...同上})
               office_type ∈ Ca && link_id → openCanvas({id:link_id,name,type:office_type,url,filePath})
               ext ∈ Ia                    → openCanvas({id:link_id||url,name,type:ext,url,filePath})
               else                        → download(url, fileName)
```

**`Ut()` 右键菜单**

| 场景 | 菜单项 |
|---|---|
| 本地文件（`platform.supportsLocalFS`） | 打开 / 在独立窗口打开 / 另存为 / 在文件夹中显示（`pageOpenDetail:"card_context_menu"`） |
| 本地目录 | 只有"打开" → `openLocalDirectory`；否则 toast「暂不支持打开文件夹」 |
| `cloud` 且 `office_type ∈ {OTL, KSHEET, DB}` | 只有"下载到本地"，且点击提示"不支持下载该类型文件" |
| `sandbox:/…` | "下载到本地" → `at()` 取 url → `downloadSandboxFile({sessionId,filePath,fileName,url})` |
| 图片 | 走 `ka()`（复制 / 编辑 / 在文件夹显示 / 另存为）或 `_t()`（会话图片菜单） |

**其它约定**

- 埋点：`sendLxaiButtonClick("deliver_card_click")`、`deliver_card_show`（带 `ze()` 得出的来源类型）；`sendLxaiClawCanvas({canvas_id, session_id, type, action, open_way:"manual", mode, button_result})`。
- 阻断样式：`.claw-deliver-result--artifact-blocked{cursor:not-allowed}`。
- 水印：`去除水印` → `messageModal("AI 生成标识设置", …, primaryText:"确认去除")` → `update(false)`（走 `useWatermark` 的 `enabled/transformUrl`）。

### 4.3 `PClawFileCard` —— 定位壳（观察）

```js
props: { fileName, filePath, displayName, fileExtension, isDirectory, icon }
emits: ["click","dblclick"]
```

```html
<div class="claw-file-card-view" :class="{'claw-file-card-view--replace-icon': !!icon && !isDirectory}"
     @dblclick="$emit('dblclick')">
  <PMessageAIFileCardInline ... @click="$emit('click')" />
  <component :is="icon" class="claw-deliver-result__file-icon" :size="24" v-if="icon && !isDirectory" />
</div>
```

### 4.4 `PMessageAIFileCardInline` —— 视觉层（观察）

文件：`localSessionLifecycle-BbrY1I2c.js`（scope `data-v-417b521c`，也是 markdown 内联卡片用的同一个组件）

```js
props: { fileName, filePath, displayName, fileExtension, isDirectory, showFileNameExtension }
emits: ["click"]   // 回调参数 = filePath（若传入）

const IMAGE = new Set(["png","jpg","jpeg","gif","webp","bmp","svg","tif","tiff","heic","ico","psd"])
const MAP = { doc:"word", docx:"word", xls:"excel", xlsx:"excel", csv:"excel",
              ppt:"ppt", pptx:"ppt", pdf:"pdf", html:"html", htm:"html",
              md:"markdown", markdown:"markdown", txt:"txt" }
kind = isDirectory ? "folder" : IMAGE.has(ext) ? "image" : (MAP[ext] || "other")
```

```html
<div class="file-card-inline file-card-inline--{kind}" :title="filePath" @click>
  <div class="file-card-inline__icon">
     folder → FolderIcon(:size=24, fill=var(--kd-color-brand-normal))
     其它   → 后缀文字图标(:text=ext 或 "file.<ext>", :size=24)
  </div>
  <div class="file-card-inline__name">{displayName || showFileNameExtension ? name.ext : name}</div>
</div>
```

样式（`main-CH_U5GfM.css`）：

```css
.file-card-inline{display:flex;align-items:center;gap:12px;width:100%;max-width:440px;
  padding:16px 20px;border-radius:12px;border:1px solid var(--kd-color-line-light);
  cursor:pointer;margin:8px 0;transition:background .2s,border-color .2s;overflow:hidden}
.file-card-inline:hover,.file-card-inline:active{border-color:var(--kd-color-line-regular)}
.file-card-inline__name{font-size:14px;font-weight:600;line-height:22px;overflow:hidden;
  text-overflow:ellipsis;white-space:nowrap;min-width:0}
```

按类型底色（双 linear-gradient 叠出"从左上斜向通透"的玻璃感，alpha 只有 `0f~14`）：

| kind | 主色 |
|---|---|
| word | `#1f69e0` |
| excel | `#418f1f` |
| ppt | `#f5670f` |
| pdf | `#dc3522` |
| image | `#1f69e0`（更淡 `0d`） |
| html | `#e87722` |
| markdown | `#6366f1` |
| txt | `#64748b` |
| other / folder | `var(--kd-color-fill-light)`（无渐变） |

---

## 5. 成果卡与右侧面板的接口契约

```ts
openCanvas({ id, name, type, url, filePath })
// id      = link_id（kdocs 预览链接）或 url
// type    = office_type(w/s/p) | "html" | 扩展名
// filePath= sandbox 绝对路径（用于二次校验与 hash 传参）
```

- `office_type ∈ Ca`（w/s/p/f 系）→ 走 `PCanvasContainerWeboffice`（WebOffice iframe，见另一篇报告）。
- `type === "html"` → 走 HTML 预览容器（`PFileHtmlPreview` / `PFileHtmlPreview` 的 iframe 分支）。
- 其它 → 工作台的新 tab（`requestSandboxFilePreviewTab`）或下载。

后端前置：

```
POST /api/aioffice/v1/sandbox/file/preview
body: { key: "/home/lingxi/workspace/.lxenc/<encoded>", session_id | project_id }
resp: { result:"ok", data:{ url, link_id, office_type, expired_at } }
```

---

## 6. 能力位与开关

| 开关 | 位置 | 作用 |
|---|---|---|
| `messagePresentation.allowToolResultInteraction` | 宿主注入 | `false` → 卡片不可点（`canOpenDetails` 直接 false） |
| `messagePresentation.allowWorkbenchNavigation` | 同上 | `false` → 不给 `openCanvas` / `canvasManager` |
| `messagePresentation.allowAutoOpen` | 同上 | `false` → 关掉 html 自动预览 |
| `isReadonly` | 会话 | 只读会话里图片点击/编辑被拦 |
| `platform.supportsLocalFS` | adapter | 决定本地路径右键菜单是否出现 |
| `platform.openInFileBrowser` / `isSupportedInFileBrowser` | adapter | 双击"在独立窗口打开"的能力位 |
| `artifactBlocked.blocked / blockedReason` | inject | 卡片整体禁用（`cursor:not-allowed`） |
| `actions.openProducedFile` | 宿主钩子 | 最高优先级，接管"打开" |
| `presentFilesMode` | prop | 多文件网格模式（改文件名/扩展名推导与图标） |
| `preferDisplayName` | prop | 优先 `content.name` 而不是从 ref 解析 |

---

## 7. 复核方法

1. 打开任一带产物的会话，DevTools 里执行：
   ```js
   document.querySelectorAll(".claw-present-files > *").length
   [...document.querySelectorAll(".file-card-inline")].map(el => ({
     cls: el.className, name: el.querySelector(".file-card-inline__name")?.textContent,
     title: el.getAttribute("title")
   }))
   ```
   `title` 属性就是原始 `ref`（沙箱路径/URL），可直接对照后端 preview 接口。
2. 单击卡片时抓 `POST /api/aioffice/v1/sandbox/file/preview` 的请求/响应，核对 `link_id`/`office_type`。
3. 右键卡片看菜单项集合反推 `platform.*` 能力位与 `office_type` 分支。
4. 埋点验证：Network 里搜 `sendLxaiButtonClick` 对应的上报（`deliver_card_click` / `deliver_card_show`）。

---

## 8. 对 gaea 办公板块的蒸馏点

1. **卡片四层职责分离**很值得抄：容器(grid) / 行为(路由+菜单+门闸) / 定位壳(自定义图标就位) / 视觉(类型渐变+后缀图标)。换皮肤只动第四层。
2. **`ref` 六态归一化**是这类卡片的真正难点：`file://`、盘符、`sandbox:/`、`/home/lingxi/`、`https`、kdocs link —— 集中到 3 个纯函数里，UI 只吃归一化结果。
3. **单击防抖 400ms + 双击优先**是"既想单击打开、又想双击进文件浏览器"的标准解法。
4. **成果卡不是终点，是"预览入口"**：所有点击最终都收敛到 `link_id + office_type` 两个字段，这就是把卡片和编辑器解耦的关键契约。
5. **权限/阻断用注入而不是 prop 传参**：`artifactBlocked` 一个 inject，卡片内所有入口统一早退，避免每个按钮各判一次。
