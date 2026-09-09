# WebView2 壳内交互残留面审计（2026-09-08）

> **状态：✅ 审计完成（只读）**——收敛计划 W1 刀 1.4 的验收物。修复另立刀（刀序见 §5），
> 本文档是后续刀的权威底稿。基准=v4.163.0 工作区。
>
> 背景：v4.162 实证 Wails 壳（WebView2）两个坑——`input[type=file].click()` 不弹系统
> 文件对话框、`<a download>` 下载不落盘（浏览器均正常）；当时修了进度计划域五路导出+
> 四路导入，自记欠账「window.print 壳内行为未走查 / 全 app input[file] 与 a[download]
> 残留扫尾」。本审计即该欠账的全量收敛前置。

## 1. 可达性总判定

`boards/manifests.ts` canonicalBoards（13 板块）**无平台门控**，桌面壳与浏览器网页版跑
同一份前端——下表所有点位壳内可达均已确认，差异只在入口层级。

## 2. P0（壳内可达 + 点击完全无反应 + 无替代出路）

| 点位 | 入口 | 壳内行为 | 修法 |
---|---|---|---|
| `frontend/src/schedule/BaselinesPanel.tsx:102` exportLedger 直呼 `downloadBlob` | 进度计划→基线对比弹层→导出签证台账 CSV | `<a download>` 不落盘静默无反应；签证台账无其他导出出口 | **一行**改 `saveExportBlob`（照 SchedulePage.tsx:509 模式；v4.162 修了主链路漏了同域邻居的典型） |
| `frontend/src/components/characterlib/CharacterLibEditor.tsx:582` 参考图 input（handler :331-344） | 角色库→角色详情→添加参考图 | `fileRef.click()` 不弹框；无粘贴/拖拽替代，下游「以参考图生成剧照」被卡死 | 照 `pickImport` 模式：`inShell()`→`GaeaPickFiles`+`GaeaReadFileB64` 还原 File 喂原 FileReader→dataURL 管线 |

## 3. P1（壳内可达，原交互无反应，但有替代出路）

| 点位 | 替代出路 | 修法 |
---|---|---|
| `pages/ImageGenPage.tsx:151` + `hooks/useImageGenHistory.ts:59`（两处下载） | 后端已落盘+「打开生成图片目录」按钮 | dataURL→Blob 后统一 `saveExportBlob`，浏览器保留原路径 |
| `pages/NovelSettingPage.tsx:98` 导入 + `:112-119` 导出 | 页面本体即文本编辑器可复制粘贴 | 导入 PickFiles+ReadFileB64；导出 saveExportBlob |
| `gaea/lib/export.ts:27` downloadMarkdown（App.tsx:1459 md 分支） | docx/pptx/xlsx/pdf 四路均正常 | **未被 v4.162 统一**；最优=md 分支改走 `exportConversation`（签名已支持 "md"，零新代码） |
| `components/imagegen/ControlPanel.tsx:276` img2img 选图 | 拖拽替代已在（WebView2 DnD 待真机确认） | pickImport 模式接 PickFiles+ReadFileB64，hidden input 留浏览器回退 |
| `components/imagegen/VisionTrial.tsx:229` 选图 | 粘贴是既定漏斗（SavePastedImage 落盘） | 选中文件→`fileToDataUrl` 喂同一 handleImageDataUrl 漏斗，零新绑定 |
| `components/SkillModal.tsx:53` 动态 input（**主列表遗漏，本审计补**） | 文案本就引导手动放 skills/ 目录（浏览器里也只展示文件名的半桩功能） | PickFiles+ReadFileB64 取名+内容，或壳内隐藏按钮改提示 |
| `schedule/exportArtifact.ts:136` printSvg（iframe `contentWindow.print()`） | 同弹层 PNG/PDF 已 saveExportBlob 化 | **待真机走查**（见 §6 观察项），勿臆断 |

## 4. P2（不构成残留）

SchedulePage 四个 hidden input（已修，`pickImport` :619-635 分流）· `exportArtifact.ts`
downloadBlob（saveExportBlob 的浏览器回退层本体）· ChatPage/useVoiceChat/TTSPlayer 的
createObjectURL（音频播放非落盘）· 各 `.test.tsx` 同款（测试桩）。

## 5. 修复刀序（后续版本）

- **刀A（P0×2，最小刀先行）**：BaselinesPanel 一行 + CharacterLibEditor 接 PickFiles；
  顺手抽 `pickImageAsDataUrl`/`pickFileAsFile` 共享 util（放中立目录，防 gaea↔schedule
  依赖倒挂）。
- **刀B（下载类批量收口）**：ImageGen×2 + NovelSetting 导入导出 + md 分支（优先改走
  exportConversation 零新代码）。
- **刀C（上传类批量收口）**：ControlPanel / VisionTrial / SkillModal 接刀A util，
  hidden input 全保留（测试口径不变）。
- **刀D（观察项，先取证不写码）**：printSvg 壳内真机走查（复用 v4.162 诊断配方：
  `WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS=--remote-debugging-port=9333`+CDP 受信任点击+
  EnumWindows）；ControlPanel 拖拽 / VisionTrial 粘贴真机确认；GaeaPickFiles 无 Filters
  （gaea_ui_extra.go:711）——各接入点需扩展名后置校验或给绑定加可选 filter 参数
  （新绑定必须加门面委托+壳内实测，v4.163 规约）。

## 6. 附带观察（非缺陷）

- `inShell()` 用 `'go' in window`，移动端 HTTP 模式 initBridge 也会挂 window.go
  （bridge.ts:1382-1386）→ 移动端同样走 GaeaSaveFileAs，落盘在桌面主机（语义=存服务端，
  可接受但应知悉）；精确区分可暴露 `isWailsNative()`（bridge.ts:1291-1295）。
- **print 复核**：全库无 `window.print` 直呼；唯一真实打印路径=printSvg 的 iframe
  `contentWindow?.print()`（SchedulePage.tsx:263 触发）。WebView2 iframe print 未实证，
  维持待实测。
- **FileSystem API 第四类**：showOpen/SaveFilePicker / showDirectoryPicker /
  FileSystemHandle / msSaveBlob 全库零命中，无此类残留；无 blob: 导航下载变体；
  现存 window.open 均走 BrowserOpenURL 优先，符合 Wails 正确姿势。

## 6b. 刀D 取证进展（2026-09-09，v4.177.0 真机首走）

**配方实证可用**：`WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS=--remote-debugging-port=9333`
启动发布版壳 + CDP（`Input.dispatchMouseEvent` 受信任 hover/click、`Runtime.evaluate`
DOM 断言、`Page.captureScreenshot`）。工具沉淀 `scripts/cdp-walk.mjs`（node 25 原生
WebSocket，无依赖）。

本轮实锤（与 gaea-slim-p2-dualspace §6 真机清单合并走查）：

| 项 | 结论 |
|---|---|
| rail 自动隐藏 dock | ✅ 默认 `translateX(-112%)+opacity:0` 滑出屏外；左缘 10px 热区受信任 hover 滑出（280ms），`translateX(0)` 实测 x:-43.5→8；悬浮覆盖内容符合 CSS 设计（foundation.css:41-69） |
| rail 切换器/分域导航 | ✅ 工位/乐园两态切换器（work 默认激活、高 57px）+ 主体 8 项分域（首页/办公/造价数据库/记忆中枢/模型中心/青鸟/编程+主题）+ foot 单列；图标渲染零破图；受信任点击「办公」导航成功（aria-current 实锚） |
| knowledge 孤儿页残留 | ✅ rail 8 项与全屏可见交互元素均无「知识库/knowledge」入口（导航侧过滤生效） |
| home「最近文档」面板 | ✅ 空态渲染正常（「还没有打开过文档」）；localStorage 写路径需经原生对话框打开文档，待人工配合复验 |
| .gsched 摘要卡 chips / printSvg print / ControlPanel 拖拽 / VisionTrial 粘贴 | ⏸ 未取证——壳实例两次被手动关闭（桌面使用中，弹窗走查打扰），改挂「需用户协作或闲置时间窗」；配方与脚本已就绪，随时可续 |

**旁证实锤**：办公板块会话视图壳内渲染正常（真实项目会话含 schedule_* 工具调用轨迹与
`.gsched` read_file 记录）=进度计划 agent 通道真机可用的在案旁证。


## 7. 扫描方法（ripgrep PCRE2，排除 *.test.*）

`type=["']file` · `\.click\(`（穷举编程式点击，覆盖动态 createElement input）·
`createElement\(["']a` · `download=` · `GaeaReadFileB64|GaeaSaveFileAs|GaeaPickFiles|
saveExportBlob|downloadBlob` · `print` 全量不分大小写 · FileSystem API 全家 ·
`createObjectURL` · `window\.open\(|location\.href|location\.assign` · `file-saver|saveAs`；
+ 逐点上下文核读触发链 + manifests 可达性判定 + v4.162/163 历史交叉验证。
已知局限：壳内行为为基于 v4.162 实证坑的外推预判；拖拽/粘贴/iframe print 三条待真机。
