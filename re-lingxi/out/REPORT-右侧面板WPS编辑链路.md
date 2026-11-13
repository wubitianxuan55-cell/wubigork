# 灵犀：右侧面板打开 WPS 文件并编辑 —— 链路逆向（1.3.36 / webBuild 26b61d0a2d85）

> 目标：`%LOCALAPPDATA%\Programs\lingxi-desktop`（WPS 灵犀 1.3.36）
> 取证材料：`out\drd\desktop-renderer-dist\assets\*`（渲染层）、`out\x\out\main\index.js`（主进程）、`out\x\out\preload\*`
> 证据等级：`观察`=直接读到代码/字符串；`推断`=结构推导未运行验证

---

## 0. 一句话结论

右侧面板不是一个原生控件，而是一个 **固定 DOM 挂载点 `#lingxi-canvas`**：`canvasManager` 在运行时用
`createApp(...).mount()` 把一个 Vue 容器组件（WPS 系文件走 `PCanvasContainerWeboffice`）塞进这个挂载点，
容器里再用 **WebOffice JS-SDK v1.1.27** 造一个跨域 `<iframe class="web-office-iframe">` 指向
`https://www.kdocs.cn/office/{w|s|p}/{linkId}?...`（或 `/l/{linkId}?...`，本文件额外带 `#lingxi_local_file_path=`），
最后由 **主进程** 给匹配到规则的 iframe 注入 `window.LingxiBridge` + 插件运行时 `0.3.0`（AI 悬浮按钮、选区上报、Ctrl+S 拦截）。

---

## 1. 五段链路

```
① 触发        文件卡片 / 产物抽屉 / 输入框附件 / AI 产物
                └→ openCanvas(file)   (PCanvasContainerWeboffice-DpRsNnJK.js: oe/w/C)
② 挂载        canvasManager.openCanvas(component, {openingFile, injection})
                └→ document.querySelector("#lingxi-canvas") → createApp(component).mount(div)
③ URL 构造    本地文件:  POST /api/aioffice/v1/sandbox/file/preview → {link_id, office_type, url}
                └→ pF(office_type, link_id, localPath, gate)     ← 本文件路径进 URL hash
               云文档:   ejr({url, linkId, urlQuery, officeType}) ← 复制 query + lingxi_office_type
④ 渲染        WebOfficeSDK.config({url, mount}) → iframe.web-office-iframe
                └→ postMessage 协议: wps.jssdk.api / api.basic / api.callback / api.event …
⑤ 主进程注入   webFrameMain 监听 frame-created / did-frame-finish-load
                └→ GRe(url) 命中规则 → executeJavaScript(插件 IIFE) → window.LingxiBridge
```

---

## 2. ① 触发与 ② 挂载（观察）

### 2.1 面板挂载点是静态 DOM

`localSessionLifecycle-BbrY1I2c.js`（`PChatLayout`，scope `data-v-e758d729`）：

```html
<div id="lingxi" class="lingxichat lingxichat--canvas-active">
  <div class="chat__core" style="width:<canvasWidthPercent>px">…对话区…</div>
  <div class="drag-line"></div>
  <div class="canvas-container active" tabindex="-1">
      <div id="lingxi-canvas"></div>      <!-- ← 右侧面板实体 -->
  </div>
  <div id="lingxi-drawer"></div>
</div>
```

CSS（`main-CH_U5GfM.css` / `filePreview-ld9_K1oO.css`）：

| 选择器 | 规则 |
|---|---|
| `.canvas-container` | `width:0`、`padding:20px 20px 20px 5px`、`margin-right:-25px` |
| `.canvas-container.active` | `margin-right:0` |
| `.lingxichat--canvas-active .canvas-container` | `flex:1; min-width:350px` |
| `.lingxichat--canvas-fullscreen` | 全屏时 padding 归零 |
| `#lingxi-canvas` | `height:100%` |

宽度由 `canvasWidthPercent`（`ChatApp` prop，默认 **50**）+ `.drag-line` 拖拽控制，宽度状态写进 `.chat__core` 的内联 style。

### 2.2 canvasManager 就是一个"运行时挂载器"

`localSessionLifecycle-BbrY1I2c.js`（`openCanvas:async function(v,E,C=!1,b="#lingxi-canvas")`）：

```js
const T = document.querySelector(k);            // 默认 "#lingxi-canvas"
if (!T) return R({result:{success:false,msg:`Canvas container (${k}) not found`}});
const {injection, openingFile, events, props} = S;
// 同文件且未失败 → 直接复用（幂等）
const J = {showCanvas:ref(U), openingFile:ref(_), fullscreen:ref(..), opening:ref(true), openFailed:ref(false)};
const $ = createApp({ setup: () => { injection && Reflect.ownKeys(injection).forEach(k => provide(k, injection[k]));
                      return () => h(component, {visible:J.showCanvas, openingFile:J.openingFile.value, ...onInited, ref:z}) } });
const oe = document.createElement("div"); oe.style.height="100%"; $.mount(oe); T.appendChild(oe);
```

- 注入的 key（`provide`）：`chat` / `drawerManager` / `canvasManager` / `adapter` / `colorTheme` / `clawWorkspace` …（`PCanvasContainerWeboffice` 用 `E("canvasManager")` 等 `inject` 取回）。
- 关闭：`vP()` → `app.unmount()` + 移除 element（`closeCanvas`/`closeAllCanvas`）。
- 全屏：只切 `J.fullscreen`，不改 iframe。

### 2.3 谁调 openCanvas

`PCanvasContainerWeboffice-DpRsNnJK.js` 内 `oe()` 返回 `{openCanvas: w}`：

```js
function w(o /*file*/, i, r, c) {
  if (route.name === "aiofficeShare" && o.id && /wps|et|wpp/.test(o.type)) { Ct(o.id); return }  // 分享页跳浏览器
  if (o.id && nexusGuard) { nexusGuard({target:{kind:"canvas",identity:String(o.id)}, open:()=>C(...)}) }
  return C(o,i,r,c);
}
function C(o,i,r,c,g){
  const S = !!o.id && typeof clawWorkspace?.openCanvasTab === "function";   // 云端 workspace 多标签
  const F = r || St[o.type];            // et/wps/wpp/otl/htmlppt/md/txt/html/xml/py/json/csv → 容器组件
  S && clawWorkspace.openCanvasTab({fileId, title, fileType});
  return clawWorkspace ? Je().then(D => h.openCanvas(F, {openingFile:o, injection:{chat,drawerManager,canvasManager,adapter,…}}, i, D))
                       : h.openCanvas(F, {openingFile:o, injection:{…}}, i);
}
```

组件映射（`St`）：`et|wps|wpp|otl` → 同一懒加载工厂 `$`（即 `PCanvasContainerWeboffice`）；`htmlppt` → `PHtmlpptContainer`；`md|txt|html|xml|py|json|csv` → `PArtifactFilePreview`。

---

## 3. ③ URL 构造（核心，观察）

`localSessionLifecycle` 内两个构造器 —— 这是整个"打开 WPS 文件编辑"的钥匙。

### 3.1 `pF(officeType, linkId, localPath, gate)` —— **本地/沙箱文件**

```js
function pF(t, e, n, r = true) {
  const base = (typeof window!=="undefined" && window.muse) ? "https://web.wps.cn" : "https://www.kdocs.cn";
  const o = new URL(`/office/${encodeURIComponent(t)}/${encodeURIComponent(e)}`, base);
  o.searchParams.set("lingxi", "true");
  o.searchParams.set("disable_custom_domain", "true");
  o.searchParams.set("_w_appid", "lingclaw");
  o.searchParams.set("enterMode", "edit");      // ← 进入编辑态
  o.searchParams.set("simple", "true");
  if (n?.trim()) o.hash = `lingxi_local_file_path=${encodeURIComponent(n.trim())}`;  // ← 本机绝对路径
  xGe(o, t, r);                                 // sets lingxi_wo_ai_button = y5r(t, r)
  return o.toString();
}
```

`xGe → Q9e → searchParams.set("lingxi_wo_ai_button", String(gate))`，
`y5r(type, gate)`：`gate` 为布尔直接透传，否则 `bne(type, allowedTypes)`，
`b5r` 映射：`w|wps|word|doc|docx → "w"`、`s|et|excel|xls|xlsx → "s"`、`p|wpp|ppt|pptx → "p"`；`C5r = "lingxi_wo_ai_button"`。

### 3.2 `ejr({url, linkId, urlQuery, officeType}, gate)` —— **云文档**

```js
const r = new URL(`/l/${encodeURIComponent(t.linkId)}`, originOf(t.url) || "https://www.kdocs.cn");
new URL(t.url).searchParams.forEach((v,k) => r.searchParams.set(k,v));   // 继承原链接 query
Object.entries(t.urlQuery||{}).forEach(([k,v]) => v!=="" && r.searchParams.set(k,v));
const o = {w:"word", s:"excel", p:"pptx", o:"otl", d:"dbsheet"}[t.officeType];
o && r.searchParams.set("lingxi_office_type", o);   // ← 主进程靠它判定文档类型
r.searchParams.set("lingxiclaw","true");            // 与 lingxi=true 二选一（claw 分支用）
r.searchParams.set("disable_custom_domain","true");
r.searchParams.set("enterMode","edit");
r.searchParams.set("disablePlugins","true");
r.searchParams.set("simple","true");
xGe(r, t.officeType, gate);
```

### 3.3 变体 `S5r(sessionId, mountRef, configGetter)`（老版 hook 等价实现）

同一逻辑的早期版本，差异点：当 `officeType` 存在时走 `/office/{officeType}/{fileId}?_w_appid=lingclaw`，
否则走 `{R}/l/{fileId}`，其中 `R = pq() ? w4e : "https://www.kdocs.cn"`，`pq()=isCustomDomain()`（`window.__CONFIG__.servicesConfig` 私有域），`w4e` 为私有域基址。
`useWebOffice-BxACfzVC.js` 的 `st()` 是更早的裸版：`${custom ? w4e : "https://web.wps.cn"}/l/{fileId}` + `lingxi/enterMode/lingerEmbed…`。

### 3.4 参数清单（实测拼装结果）

| 参数 | 值 | 作用 |
|---|---|---|
| `lingxi` / `lingxiclaw` | `true` | 灵犀专用皮肤与接口开关（claw 分支） |
| `disable_custom_domain` | `true` | 忽略私有域配置，走 kdocs 公共域 |
| `_w_appid` | `lingclaw` | 接入方 appId（仅 `/office/` 形态） |
| `enterMode` | `edit` | **以编辑态打开**（非 preview） |
| `simple` | `true` | 精简 UI（隐藏协作/工具页签） |
| `hidecmb` | 视配置 | 隐藏命令栏 |
| `include_plugin_names` | 白名单串 | 只放行 KDOCS 系插件（见下） |
| `disablePlugins` | 视配置 | 全禁插件 |
| `lingxiEmbed` | `et`（仅表格） | 表格嵌入模式 |
| `disableSSR` | `true`（表格） | 关服务端渲染 |
| `lingxi_office_type` | `word/excel/pptx/otl/dbsheet` | **主进程识别文档类型的唯一依据** |
| `lingxi_wo_ai_button` | `true/false` | **是否注入 AI 插件**（`false` → 插件直接 disabled） |
| `#lingxi_local_file_path` | URL 编码的绝对路径 | 本地文件身份（`/^\//、`C:\`、`\\` 才认） |

`PCanvasContainerWeboffice` 里按类型的编辑态裁剪（`ce` 常量）：

- `et`：`commonOptions{isHistoryPreviewUseIFrame:true, isShowPluginCenterEntry:false, changePcHeader:[remove ToolsTab, remove CooperationTab], isEnableChangeDocumentTitle:false}` + 一大票 `etOptions` 关闭（AI 修公式/数据切片/DB 视图/共享表/KS 升级/Dashboard/批注/保护/导入/智能填充/护眼/`wps_ai_entry`），`commandBars` 隐藏 `HeaderLeft/HeaderRight/CooperationTab/ToolsTab/FloatQuickHelp`。
- `wps`：`simple:true`、`wpsOptions{isShowDocMap:false, disableComment:true, disableRevisionComment:true, isShowBottomStatusBar:false, officialMode:false, wps_ai_entry:false}`、`cooperUserAttribute{isCooperUsersAvatarVisible:false}`、只放行 `KDOCS|FloatQuickHelp`。
- `otl`：`simple:true` + `otlOptions{enableAI:false, enableLUIV2:false, isHidePlayMode:true…}`。
- `wpp`：`simple:true`。
- 白名单 `Kt` = `KDOCS|Schedule, KDOCS|KDOCS, KDOCS|AT, KDOCS|LoginGuide, KDOCS|Report, KDOCS|ReportComment, KDOCS|FeedBackReport, Map|Map, CommonUserCard|CommonUserCard, Publib|Publib, CoEditProcesson|CoEditProcesson, KDOCS|VipPayCounter, KDOCS_PUBLIC_PLUGIN|*, KDOCS|FileSelector, KDOCS|LinkPerm, KDOCS|KUpload, CloudDocInfoV2|CloudDocInfoV2, InsertGroupCard|InsertGroupCard, DrawingBoard|DrawingBoard`。

---

## 4. 本地文件如何变成一个可编辑的云链接（观察）

```
本地路径 e:\a.docx
  ├ Xj(path)                                    → stat（size / mtime，大小上限 ySe）
  ├ fn.readFileAsBlob(path)                     → Blob（IPC: file:*）
  ├ G2e(sessionId, path, File)                  → POST /api/aioffice/v1/cowork/sessions/{sid}/files
  │                                              返回 data.encoded_path / data.preview_path
  ├ $6r(blob)                                   → 内容 hash（缓存键）
  ├ sandboxPath = lre(encoded_path) = /home/lingxi/workspace/.lxenc/{encoded_path}
  └ od(sessionId, sandboxPath)                  → POST /api/aioffice/v1/sandbox/file/preview {key, session_id}
                                                 返回 { link_id, office_type, url }
        └ canvasUrl = pF(office_type, link_id, 原始本地路径, gate)
```

- 缓存：`t5`（按 `sessionId\0path\0gate` 记忆 canvasUrl 结果）、`yW`（按 `sessionId:path` 记忆上传结果）。
- 失效：`YI(linkId)` = `GET /api/v5/links/{id}?from=preview&check_collaboration=false`；`kfe(resp)` 命中 404 / `SessionNotExisted` / `*not found*` → 视为链接失效，丢弃缓存重新预览（`lci`）。
- 关键：**"编辑"不是打开本地文件本身**，而是打开一个服务端预览链接（kdocs 侧 light link），真实内容在服务端；对本地磁盘的回写靠插件与 agent 的保存通道，不是 WPS 客户端的文件锁。

---

## 5. ④ WebOffice SDK 渲染层（观察）

`useWebOffice-BxACfzVC.js` = 打包进前端的 `WebOfficeSDK JS-SDK V1.1.27`（`console.log("WebOfficeSDK JS-SDK V1.1.27")`）。

- `config(options)` → 创建唯一 iframe：`id="office-iframe"`、`class="web-office-iframe"`、`allow="clipboard-read; clipboard-write; …"`、`ResizeObserver` 跟随容器尺寸；`destroy()` 移除。
- 消息总线：`window.addEventListener("message")` + `postMessage(JSON.stringify(msg), origin)`，`_e("origin", …)` 记录目标 origin 并做同源校验（`te()` 忽略 `www.` 前缀差异）。
- 事件：`ready` / `open.result` / `api.ready` / `api.event` / `api.callback` / `fileStatus` / `error` / `stage` / `setConfig` / `setToken` / `basic.event` / `fullscreenchange`。
- `fileStatus`（`file.saved`）→ 容器内 `on("fileStatus", onSaved)` 触发自动保存回调。
- 办公对象：`app.Application`，按 officeType 映射 `WordApplication/ExcelApplication/PPTApplication/PDFApplication`；`ApiEvent.AddApiEventListener`（如表格的 `OnHistoryVisibleChanged`、`HISTORY_PREVIEWER_RESTORE` → "历史版本"）。
- `useWebOffice` 尾部还做了 `PageFit=2`（页面适配窗口）——`disablePageFit` 为真时跳过（`wps` 类型默认跳过）。

容器组件 `PCanvasContainerWeboffice` 的头部动作（`PCanvasHeader`）：

| 动作 | 实现 |
|---|---|
| 下载 | 本地/沙箱文件走 `file:*` 读取；`otl` 类型改为"导出为文字"（`Qe(id)`→fileinfo→`tt(id,"docx",fver)` 拿 url→`De(url,name)`） |
| 归档（历史版本） | 仅 `et` 且有 `etOpenAPI` 时显示，`etOpenAPI.toggleHistoryPanel()` |
| 全屏 / 退出 | `canvasManager.enterFullscreen()` / `.exitFullscreen()` |
| 关闭 | `canvasManager.closeCanvas()` |
| 打开超时 | `Promise.race([reInit(), 30s timeout])` → `openFailed` → "文件加载失败 / 重试" |

---

## 6. ⑤ 主进程注入：`LingxiBridge` + 插件 0.3.0（观察，最有价值的部分）

代码在 `out/main/index.js` 的 `oK(webContents, opts)`，日志 tag `lingxi-wo-init` / `[LingxiAI]`。

### 6.1 帧匹配规则

```js
const sK="lingxi_office_type", WRe="lingxi_local_file_path";
const jRe=new Set(["word","excel","otl","pptx","dbsheet"]);
const zRe={w:"word", s:"excel", o:"otl", p:"pptx", d:"dbsheet", word:"word", …};

function iK(url){ // 域名白名单
  const h=new URL(url).hostname.toLowerCase();
  return h==="kdocs.cn"||h.endsWith(".kdocs.cn")||h==="wps.cn"||h.endsWith(".wps.cn");
}

function GRe(url){
  if(!iK(url)) return null;
  if (/^\/l\/([^\/]+)\/?$/.test(path))                    // 云文档形态
      return {documentId, documentType: jRe.has(q.get("lingxi_office_type")) ? … : "word",
              documentUrl:"", source:"cloud"};
  if (m = /^\/office\/([^\/]+)\/([^\/]+)/.exec(path))     // 编辑形态（含 w|s|p|o|d）
      return {documentId, documentType: zRe[m[1]],
              documentUrl: VRe(url) /* #lingxi_local_file_path=… */ || `artifact-office://${documentId}`,
              source:"local"};
  return null;
}
```

### 6.2 注入时机

```js
wc.on("frame-created", (_, d) => d.frame && hook(d.frame));
wc.on("did-frame-finish-load", (_, isMain, frameProcessId, frameRoutingId) =>
      hook(isMain ? wc.mainFrame : webFrameMain.fromId(frameProcessId, frameRoutingId)));
hook(wc.mainFrame);                                  // 主框架也试
for (const f of wc.mainFrame.framesInSubtree) hook(f) // 现有子树
// hook(frame): frame.on("dom-ready", tryInject) 且立即 tryInject 一次
// tryInject: GRe(frame.url) 命中 → NRe({documentType, source, session, documentUrl, visualTheme})
//            → frame.executeJavaScript(`(${$Re})(${JSON.stringify(cfg)}, ${rK})`)
```

### 6.3 注入内容

`XRe(payload)` 先挂标记 `data-lingxi-wo-plugin-installed` 和桥：

```js
const bridge = { emit(event, data) {
  window.top.postMessage({source:'lingxi-ai-poc', event, data, origin:window.location.origin}, targetOrigin);
  console.log('[LingxiAI]', {event, summary});
}};
window.LingxiBridge = bridge;
($Re)(cfg, rK)   // 插件运行时，version "0.3.0"
```

`$Re(cfg, rK)` 做的事（约 49 KB）：

1. **门闸**：`URLSearchParams(location.search).get("lingxi_wo_ai_button") !== "false"`，否则返回 `{status:"disabled"}`。
2. 注入 `<style>`（含 `@keyframes lingxi-wo-ai-panel-shake`）+ 悬浮按钮：`[data-lingxi-wo-ai-button]`、`.lingxi-wo-ai-ppt-button`、`.lingxi-wo-ai-excel-button`、`[data-lingxi-wo-ai-panel]`、`#lingxi-wo-ai-panel-tooltip`。
3. 遍历同源 iframe（`contentDocument` 递归）与 MutationObserver，把按钮挂到每页。
4. 选区捕获：`selectionchange` / `mouseup` / `mousedown`；按类型注册办公事件
   `word: WindowSelectionChange`、`excel: Worksheet_SelectionChange`、`otl: SelectionChange`、`pptx: SlideSelectionChanged`、`dbsheet: SelectionChange`。
5. 键盘：**Ctrl/Cmd+S 拦截** → `preventDefault + stopImmediatePropagation` → `emit("artifact-office-save-request")`。
6. 本地/`artifact-office://` 文档额外：`beforeinput|input|change|paste|cut|drop` → `emit("artifact-office-interaction")`（脏标记）。
7. `DocumentSaveStatus` → `emit("artifact-office-save-status")`。
8. 暴露 `__officeEvalResult(id, result, error)` → `emit("eval-result")` —— 即 **渲染层可通过桥给文档内下发 JS 求值并拿结果**。
9. 就绪：`emit("wps-ready", {docType})`，返回 `{status:"ready", version:"0.3.0", entries:{snapshot}, dispose}`。

### 6.4 渲染层如何接

`assets/lingxiAiPocBridge-BsCAZpDV.js`：

- 常量：`G="lingxi-ai-poc"`、`K="ai-input-submit"`、`Q="ai-panel-open"`。
- 校验：`https` + `*.kdocs.cn|*.wps.cn`；`data.source===G`。
- 类型映射：`{word:"wps", excel:"et", pptx:"wpp"}`；选区 kind 白名单 `text/table/image/element/elements/cell-range/chart/mixed`。
- `Pt(msg)` → `{action:"selection_edit"|"selection_edit_send"|"selection_edit_chat", canvasId, type:"word_kdocs"|"excel_kdocs"}`（走 `onCanvasReport` 埋点）。
- `At/Et(msg)` → 把 `prompt/selectedText/selection/wps_docs_context/wps_docs_selection` 填进主输入框（`handleSetInputContentWithSelection` / `setLingxiSelectionContent`），或 `autoSend` 直发；文本前缀 `[选区1]`。
- 快捷意图：`解释选中的内容→explain`、`翻译选中的内容→translate`、`快速润色选中的内容→polish`、`根据选中的内容续写→continue`。
- 路径判定：`/^(?:\/|[A-Za-z]:[\\/]|\\\\)/` → 本地；否则 `https://www.kdocs.cn/l/{linkId}`。

---

## 7. 开关与准入（谁能编辑、谁有 AI 按钮）

| 开关 | 位置 | 语义 |
|---|---|---|
| `lingxi_wo_ai_button` | URL 参数，`pF`/`ejr` 的 `gate` | `false` → 插件 `{status:"disabled"}`，文档里没有 AI 悬浮按钮 |
| `lingxi_wo_ai_edit_enabled_v2` | `GET https://www.kdocs.cn/kdg/api/v1/params/MusaConfig`（`credentials:"include"`）→ `{visible, allow_type:["w","s","p"]}` | 远控总闸；`allow_type` 去重后只留 `w/s/p` |
| store `lingxiWoAiEditAllowedTypes` / `lingxiWoAiEditEnabled` | `localSessionLifecycle` | 本地覆盖（`[...allowed]` 优先，否则布尔） |
| `MusaConfig.lingxi_wo_ai_edit_enabled_v2.visible` | 同上 | 为 `true` 且 `allow_type` 非空才启用 |
| `link` 有效性 | `GET /api/v5/links/{id}?from=preview` | 404/`SessionNotExisted` → 重传/重新预览 |
| `window.muse` | 渲染层全局 | 存在 → `web.wps.cn`（WPS 客户端 WebView 宿主）；Electron 里不存在 → `www.kdocs.cn` |

其他 `MusaConfig` 灰度键（同域）：`hero-badge`、`wiki`、`client-agent`、`memory-organize`、`wiki-purify`、`local-preview`、`desktop_local_image_thumbnail_v2`、`project`、`docspace-v2`、`official-office-plugin{office_plugin_entry, office_plugin_website}`、`remoter`、`with-echo`、`browser-plugin-download`、`about-policy-links`。

---

## 8. 与"网页编辑（IAB）"面板的区别（别混）

| | WPS 文档面板 | 网页编辑面板 |
|---|---|---|
| 载体 | 渲染进程内 `<iframe class="web-office-iframe">`（WebOffice SDK） | IAB 内置浏览器（主进程 HTTP `127.0.0.1:56305`，`iab:*` 52 条 IPC） |
| 挂载点 | `#lingxi-canvas` | IAB page / `iab-webpage-edit` |
| 桥 | `LingxiBridge`（`lingxi-ai-poc`，主进程注入 JS） | `preload/iabWebpageEditBridge.js` → `ipcRenderer.sendToHost("webpageInteractionEvent", {registrationId:"iab-webpage-edit", eventType: hover/selection/refresh/exit-edit})`，仅 `file://*.html?` 页面转发 |
| 编辑对象 | kdocs 文档（`enterMode=edit`） | 本地 HTML（`.htmlppt`、网页产物） |

---

## 9. 复核/复现方法（不用改程序）

1. **看 URL**：面板打开后 `Alt+Ctrl+I`（或 F12）→ Console →
   `document.querySelector(".web-office-iframe")?.src`，应看到 `https://www.kdocs.cn/office/w/{linkId}?lingxi=true&…&lingxi_wo_ai_button=true#lingxi_local_file_path=…`。
2. **看桥**：在该 iframe 的 Context 里执行 `window.LingxiBridge` / `document.documentElement.dataset.lingxiWoPluginInstalled` / `location.search`。
3. **看主进程日志**：`%APPDATA%\WPS 灵犀\logs` 里 grep `[LingxiAI] matched WebOffice frame`、`plugin injected into WebOffice frame`、`local plugin loader enabled`。
4. **手动造链接**：把任一 linkId 代入
   `https://www.kdocs.cn/office/w/{linkId}?lingxi=true&disable_custom_domain=true&_w_appid=lingclaw&enterMode=edit&simple=true&lingxi_wo_ai_button=true`
   可在普通 Chrome 里复现同一编辑器（缺插件，因为注入在 Electron 主进程侧）。
5. **验证本地路径映射**：URL hash 里 `lingxi_local_file_path` 必须是绝对路径，主进程 `HRe` 才认；相对路径会被丢弃 → `documentUrl` 退化为 `artifact-office://{id}`。

---

## 10. 对我们（gaea 办公板块）的可蒸馏点

1. **右侧面板 = 固定挂载点 + 运行时 createApp**：比"每个文件开一个窗口"省内存、可复用注入上下文；`showCanvas/opening/openingFile/openFailed/fullscreen` 五个 ref 就够了。
2. **文件身份三件套**：`link_id`（服务端预览链接）+ `office_type`（w/s/p/o/d）+ 本地路径（hash 传递）。前两者来自后端 preview 接口，第三方不需要自己解析 OOXML 就能直接调起 WPS 编辑器。
3. **编辑器裁剪靠 URL 参数 + commonOptions/commandBars**：`enterMode=edit`、`simple`、`hidecmb`、`include_plugin_names` + `changePcHeader` 移除页签 —— 不用 fork 编辑器就能做"AI 专用办公壳"。
4. **AI 与文档的耦合点是 iframe 内的 JS 注入 + postMessage 桥**（不是 OOXML、不是 JSAPI 直连）：选区/脏标记/保存/求值四条事件即可撑起交互。
5. **`__officeEvalResult` + `eval-result`** 是隐藏的高价值通道：渲染层能给文档内下发任意 JS 求值并同步拿结果 —— 做"AI 改文档"时比自己算 DOM 偏移可靠。
6. **两份实现并存**：`useWebOffice`（老）与 `localSessionLifecycle.S5r`（含 claw/私有域分支），说明这套 URL 约定在他们内部已稳定两代，值得按同一套参数对齐。
