# 闲庭 · 原罪板块设计（sin）——对话式图文混杂故事创作

> 状态：已落地（v4.256.0，2026-09-12）· 类型：板块设计与契约基线
> 关联代码：`internal/app/sin_handler.go`、`internal/app/sin_prompt.go`、
> `internal/app/board/builtins.go`、`frontend/src/pages/OriginalSinPage.tsx`、
> `frontend/src/pages/sin/*`、`frontend/src/boards/manifests.ts`
> 关联决策：`docs/ADULT_MODE.md`（成人内容默认开启，个人非商用定位）

## 1. 需求与口径（用户原话）

> 「在闲庭增加一个单独的功能板块，复用办公的组件，主要用于与 AI 对话，
> 可以生成图文混杂的 H 故事小说，板块取名原罪。」

拆成四条可验收口径：

1. **单独板块**：闲庭（play 空间）内的一级板块，独立 id/入口/启动器卡，不寄生在小说或聊天里；
2. **复用办公组件**：输入器、正文渲染、工具按钮、样式层全部用办公（gaea）既有件，不新建聊天组件；
3. **以对话为主**：交互形态是「说一句 → 写一段 → 你接着指方向」，不是表单式流水线；
4. **图文混杂**：故事正文与插图在同一阅读流里交替出现，导出物同样是图文混排的 Markdown。

## 2. 板块形态

| 项 | 值 | 说明 |
|---|---|---|
| id | `sin` | manifest 键；导航白名单/快捷键/启动器均由它派生 |
| label | 原罪 | 菜单/面包屑/启动器显示名（单一来源） |
| space | play | 闲庭；书斋侧不出现入口 |
| layout | full | 全出血（与 chat/gaea 同档：零 padding + 玻璃底 + 隐藏滚动） |
| page | OriginalSinPage | PageRegistry 注册键 |
| featureModel | sin | 模型中心「原罪」功能绑定（可单挂模型） |
| menuOrder | 13 | 闲庭末位（聊天/小说/绘梦/角色库之后） |
| nav | 无 | 单页板块不声明子导航（不造不存在的二级入口） |

版面：左「故事」架（列表/新建/重命名/删除）＋ 中「故事流 + 办公输入器」＋ 右「玩法 · 插图协议 · 边界」三卡；窄窗按 1180 / 880 两级收起右栏与左架（功能不以尺寸为借口消失）。

## 3. 组件复用清单（「复用办公组件」的正面落地）

| 办公件 | 用在原罪哪里 | 继承到的能力 |
|---|---|---|
| `gaea/components/Composer` | 故事指令输入 | 附件/粘贴表格转 Markdown/截图/裁剪/识图/OCR/输入历史/高度记忆 |
| `gaea/components/Markdown` | 故事正文本段渲染 | 本地文件链接 FileChip/代码高亮折叠/mermaid/genui/数学公式/消毒与 URL 白名单 |
| `gaea/components/ToolbarButton`、`EmptyState` | 顶栏动作、空态 | 办公视觉与交互惯例 |
| `gaea/lib/i18n` `LocaleProvider` | 页面外层 | 办公组件依赖的 i18n 上下文 |
| `gaea/styles.css` + `tailwind.css` + `redesign.css` | 全页样式层 | 令牌体系（`--bg/--fg/--border-soft/--accent/--warn/--err`）、明暗联动 |

反向纪律：**不改办公组件本身**——原罪只做消费方；需要新行为时在 `pages/sin/*` 内组合，避免为一块新板块改动办公线（改动面越小越可回滚）。

## 4. 后端契约

### 4.1 会话（复用统一聊天存储）

- 存储 = `internal/chat.Store`（SQLite，与聊天板块**同一张表**），话题 `mode=sin`；
- `SinTopicsList/SinMessages/SinTopicRename/SinTopicDelete/SinTopicClear` 全部先过 `sinTopicGuard`（**话题必须存在且 mode=sin**，fail-closed：前端误传聊天话题 id 时拒绝改写，不跨板块破坏数据）；
- 反向隔离：`ChatTopicsList` 过滤 `mode=sin`——聊天板块的 persona 分支不认识 sin 模式，放进去既污染列表也会走错生成路径。

### 4.2 文本生成

- 绑定：`SinStream(topicID, message) → runID`，事件 `sin-stream:<runID>`（delta/reasoning/done/error，与 `chat-stream` 同形）；
- 路由：`routeModel("sin")`（功能绑定 → 全局激活 → 首个可用引擎兜底；离线模式过滤云端）；
- 前情装配：最近 12 条历史（单条截断 1500 rune），历史里的插图标记替换为「（已配图）」——标记是渲染指令不是故事内容，重复喂回去会诱导模型乱发标记；
- play 护栏：温度/输出上限经 `clampPlayTemperature/clampPlayMaxTokens` 只降不升（配置未启用 = 零值 = 不钳制）；
- 落库：完成后单事务写「用户消息 + 助手消息」，`extra` 带 reasoning；**正文原样落库**（正文 = 模型输出，可回溯）。

### 4.3 插图

- 协议：模型在需要配图处**另起一行**输出 `@@插图|画面描述@@`（一轮最多 2 张）；
- 生成：`SinIllustrate(topicID, messageID, cue, prompt, size)` → 复用绘梦图像后端 `mediaState.GenerateFreeImage`（play 护栏 `image_safe_mode` 照常生效），未落盘时补一次本地落盘；
- 登记：图像域台账 `space=play / source_board=sin`（画室素材库可按来源筛选；失败只 warn，不拖垮生成）；
- 回写：路径写进该轮助手消息 `extra.illustrations[cue]`（cue = 该条消息内插图出现次序，0 起；前后端同规则）——**重开故事按映射直接渲染，不重新生成**；
- 导出：`SinExportMarkdown` 把标记换成 Markdown 图片（未生成的保留「（插图未生成：…）」占位），前端经 `saveExportBlob` 走壳内「另存为」。

### 4.4 模型绑定

配置三键 `func_sin_engine` / `func_sin_model` / `func_sin_enabled`（默认启用），模型中心「功能绑定」多一项「原罪」：可给故事创作单挂长上下文/无审查本地模型，与聊天/小说互不干扰；未绑定即跟随全局激活模型。

## 5. 内容边界（沿用 ADULT_MODE，写进提示词并用测试锁死）

成人内容口径**零新立**：见 `docs/ADULT_MODE.md`（个人非商用桌面应用、默认开启、不回避不说教不医学化）。原罪只把它落到「故事创作」语境，并把硬边界写进 `sinSystemPrompt`：

1. 性相关角色一律成年人，禁止未成年/年龄模糊/师生监护等可读成未成年的设定；
2. 禁止以真实在世人物为主角写性内容；
3. 保持合意基调（可有张力，不写非自愿性暴力与以伤害为目的的凌辱）；
4. 用户说停/跳过/改设定时立刻照做。

测试 `TestSinSystemPromptHasHardBoundaries` 对这四条与插图协议做存在性断言：改文案时防静默丢边界。

## 6. 已知边界与后续可选刀

- 插图只走绘梦当前配置的图像后端（云端安全策略可能拒绝露骨提示词）→ 需要本地 ComfyUI 才能稳定出露骨画面，属配置事实，UI 不代用户决定；
- 未做「插图重绘变体历史」（只保留最新一张）；画室素材库已能按 `source_board=sin` 看到全部产物，需要变体窗口时再立项；
- 未做多人物一致性锚定（角色参考图槽）；若要长期连载同一人物，应复用角色库工作副本与图域 T2 参考槽，而不是在本板块另造一套；
- 无「章节化/长篇结构」管理：故事即一条对话流。若需要卷/章结构，属于与小说板块的边界问题，需先拍板再动。

## 7. 验收证据（v4.256.0）

- Go：`go build ./...`、`go vet ./...`、`go test ./... -count=1` 全绿；新增用例 7 项（域隔离与守卫/流式落库与消息 id/未知话题拒绝/插图回写与导出/前情装配/提示词边界）+ config 绑定往返 1；
- 绑定面：`scripts/check-bindings-drift.ps1` PASS@637（新门面 `SinB` 9 方法）；`gen_bindings` 再生后 office/cost 门面已按既有纪律还原，diff 仅 manifest + 新文件；
- 前端：`tsc -b` 0、`eslint .` 0 error、`vite build` 成功、`vitest run` 334 文件全绿（新增 27 例）；
- 守卫：`scripts/frontend-e-check.mjs`、`scripts/check-docs.mjs` 绿；版本三处 4.256.0。

## 8. 与办公板块硬隔离（v4.257.0，用户拍板「原罪需要与办公板块硬隔离」）

隔离口径：**组件可复用（渲染层），数据面/记忆面/模型路由面/工作区面一律不通**。
逐条落点与守卫：

| 面 | 原罪归属 | 隔离实现 |
|---|---|---|
| 会话与消息 | `chat.db`（话题 `mode=sin`） | 同表不同域：`SinTopicsList` 只见 sin，`ChatTopicsList` 过滤 sin（chat_service.go）；不进办公会话/日志（`.gaea/work/sessions`、journal） |
| 故事记忆 | **无**：不读也不写办公记忆 | `SinStream` 只带「本次故事的历史消息 + 已选角色」，零记忆注入；办公 facts（Hephaestus.db）与轻语记忆（hermes.db）都不在链路里 |
| 插图产物 | `<用户配置目录>/gaea/sin/art` | 生成链 provenance 参数化（`generateFreeImageProvenanced`）：原罪传自有 saveDir + `source_board=sin`，单条登记进图像域台账；**不写办公工作区 `.gaea/`、不依赖办公 ImageSaveDir 配置**，也不用小说项目目录 |
| 角色选择 | `<用户配置目录>/gaea/sin/cast.json` | `SinCastGet/SinCastSet`（sin_store.go）；角色库是跨板块共享资产层，不是办公数据面 |
| 模型路由 | `routeModel("sin")` | 功能绑定 → 全局激活 → 首个可用引擎；**不经办公（gaea/office）绑定**，办公换模型不影响原罪 |
| 工具与权限 | 无 | 不挂办公 agent 工具链（无 permission 引擎、无工作区、无会话、无审批面）；原罪只调 `SinStream/SinIllustrate/...` |
| 反向（办公看不到乐园） | 工位检索面 | `wssearch.isNoiseRel` 把**整个 `.gaea/play`** 判为噪音（v4.257 前只跳 `play/exports`——画室台账/章节图/原罪台账会漏进工位关键词检索）；`fileindex` 本就整目录跳过 `.gaea` |

**共用但不算越界**（明确列出，防误判为泄漏）：绘梦图像后端与其全局配置（图像后端是应用级能力，
原罪只是调用方，产物归属已按 `source_board=sin` 标注）；角色库（跨板块共享资产）；
办公 UI 组件（Composer/Markdown/…，仅渲染层，不携带办公数据）。

测试锚：`TestSinStorageRootsAreIsolated`（数据根必须在用户配置目录、不得落在 `.gaea` 下）、
`TestSearchSkipsWholePlayPartition`（play 整分区不进工位检索）。

## 9. 角色库接入（v4.257.0，「原罪能够使用角色库内容」）

- **选择**：右栏「角色」卡 → 弹层按角色库列表多选（头像走 `PortraitImg` 三态：远程/data URL/本地路径）；
- **持久化**：`SinCastGet/SinCastSet(topicID, ids)` → `<用户配置目录>/gaea/sin/cast.json`（原子写；损坏按空表继续）；
  后端对 id 做**悬空过滤 + 去重保序 + 单故事封顶 8**，返回生效清单（前端以后端为准回填）；
- **注入**：`SinStream` 装配本故事角色块（`sin_prompt.go::sinCastBlock`）——名字/性别/年龄/定位/外观/身材/性格/背景/动机/状态/口吻/最多 3 条说话样例，值空则整行不写（不编造），并声明「须严格按其设定写、不得改设定或改名」；
- **边界**：只引用不修改——原罪不改角色库档案（增删改仍在角色库板块）；
- **未做（后续可选刀）**：把角色剧照/参考图当**图像参考槽**喂给绘梦后端（人物一致性锚定）——需要 ComfyUI/herdsman 的 img2img 与参考图槽，且与「未配置本地后端时兜底 txt2img」组合，留待拍板；当前一致性靠提示词层（角色块 + 插图提示词自带外观锚点）。

## 10. 修复记录：插图读图报「AttachmentDataURL is not a function」（v4.257.1）

现象：真机点出图后插图位显示「插图失败：e(...).AttachmentDataURL is not a function」。

根因：S2-3 兼容代理 `window.go.app.App`（`bridge/http.ts::ensureLegacyAppProxy`）
只按字面名在各板块门面里查方法，**不做 `gaeaToGaea` 短名映射**；`api/image.ts`
的图像域 helper 直连这个代理并写短名 `AttachmentDataURL`，而 OfficeB 上的真名是
`GaeaAttachmentDataURL` → 查找落空 → 调用抛 not a function。走 bridge 代理的调用点
（PortraitImg、消息内联图）不受影响，所以只有一部分功能中招。

同一根因波及的 4 个方法：`AttachmentDataURL`（读图，原罪插图/章节配图历史）、
`SavePastedImage`（粘贴落盘）、`RecognizeImage`（识图）、`OCRText`（OCR）。

修复：兼容代理补映射回退（字面名优先 → 映射名 → 仍未命中才 undefined）；
`readFileAsDataURL` 改走规范 bridge 路径（映射 + mock 兜底），仅当 bridge 未认领时
退回旧直连；HTTP 模式门面补齐表加 `SinB`。回归测试
`frontend/src/gaea/lib/bridge/legacyAppProxy.test.ts` 用真机形态的假门面锁住这条链路。

## 11. 生成进度：进度动画 + 串行队列 + 可停止（v4.258.0）

用户口径：「增加图片生成动画知道完成进度情况」。

### 11.1 进度来源（诚实优先）

| 后端 | 进度来源 | UI |
|---|---|---|
| ComfyUI | `GetComfyUITaskProgress`（逐节点上报） | 确定态进度条「全部: N%」+「当前节点: 采样中」+「已用时 Ns」 |
| 其余后端（xai/glm/herdsman…） | 后端不提供百分比 | 不定态光带「生成中…」+ 本地秒表「已用时 Ns」——**不编造百分比** |

进度条组件 = `components/imagegen/GenerationProgress.tsx`（**从绘梦 GenerationBar 抽出共用**，
类名 `.ig-progress*` 与阶段名表 `COMFY_NODE_LABELS` 单点维护，防两处视觉/文案漂移）；
轮询 = `components/imagegen/useComfyTaskProgress.ts`（只在 `active` 时轮询、读失败保留上一帧、
非 ComfyUI 后端如实给 `percent = -1`）。

### 11.2 串行队列（进度可信的前提）

`pages/sin/illustrationQueue.ts`：同一时刻只跑一张插图——图像后端（尤其 ComfyUI）
一次只执行一个任务、进度也是单任务维度，并发提交会让两条进度条抢同一个百分比。
排队中的插图显示「排队中 · 前面还有 N 张」，轮到时自动切换为实时进度。

### 11.3 取消（真停止，不是假装）

- 前端：插图卡「取消」→ `CancelImageGeneration`；故事流「停止」→ `SinCancel(topicID)`；
- 后端 `SinCancel`：中止底层流式请求（ctx），**已生成的部分照常落库**
  （`extra.cancelled = true`），用户消息不丢；无在途流时如实报错；
- 关闭入口：`singRuns` 进程内登记表按话题单飞（一次一轮）。

### 11.4 顺带修掉两个自伤缺陷

1. **`sending` 卡死**：`useSinStory` 用 `deadRef` 跳过收尾，React StrictMode 双挂载
   （dev / 空间剪枝重挂）后永久为 true → 输入框被「请先完成或停止当前回合」锁死。
   现在无论是否卸载/取消都必收尾（`setSending(false)` + 刷新故事列表）。
2. **「停止」按钮是空实现**：上一刀把 Composer 的 `onCancel` 传成 `() => undefined`，
   点停止什么都不发生。现在接到 `useSinStory.cancel`（后端取消 + 本地立即解封）。

测试锚：`illustrationQueue.test.ts`（串行/位次/失败不阻断）、`useComfyTaskProgress.test.ts`
（轮询/停轮询/未知进度/读失败保帧）、`SinIllustration.progress.test.tsx`（确定态/不定态/已有产物不重跑）、
`sin_cancel_test.go`（取消保留部分 + 跨板块守卫）、`useSinStory.test.tsx`（cancel 复位）。

## 12. 角色一致性：文本锚点 + 图像参考槽（v4.259.0）

接着 §9 的落点继续：角色选定后，让**插图里的人物**也跟上设定。

### 12.1 两段式（互为补偿）

| 手段 | 生效面 | 说明 |
|---|---|---|
| **文本锚点** | 所有图像后端 | 提示词没写到角色外观时，追加「人物锚点（名字）：外观；身材」（≤120 rune）；已写过就不重复 |
| **图像参考槽** | 仅 ComfyUI / Herdsman 图生图 | 取角色参考图（`referenceImages` 优先，其次 `portraitUrl`）走 T2 参考槽：`mode=img2img` + `refImages` + `refMethod=img2img`，denoise 0.65 |

### 12.2 锚定谁（防参考图互串）

`sinPickRefCharacters`：提示词**点名**的角色优先（名字出现在画面描述里）；都没点名时
**仅当故事只选了一个角色**才锚定；多角色且都没点名 → 不锚定（不猜）。

### 12.3 能力门控与诚实跳过（`sinRefPlan` 纯函数）

| 后端 × 模型 | 结果 |
|---|---|
| comfyui + krea2* / z-image-turbo | ✅ img2img 参考槽（与 `ai/image_comfyui.go` 的图生图工作流支持面一致） |
| comfyui + 其他模型 | ⏭ 跳过，原因「当前模型不支持图生图参考」 |
| herdsman | ✅ img2img 参考槽（其文生图端点明确拒绝参考图） |
| xai / glm / 其他 | ⏭ 跳过，原因「当前图像后端不支持参考图」 |

参考图数据形态：`data:` 原样、本地路径读文件转 data URL（comfyui 上传与 herdsman
img2img 都只认 data URL）、远端 URL 跳过（不扩展网络面）、缺失文件跳过并记 warn。

### 12.4 回退（插图不因参考图而失败）

参考槽那条路失败 → **自动退回纯文本重试一次**，返回值标 `ref_fallback=true`，
UI caption 显示「参考图不可用 · 已按纯文本生成」；生成成功则显示
「角色参考：<名字>」与「已补外观锚点」徽标（用了谁、补了什么，一眼可见）。

测试锚：`sin_refs_test.go` 7 例（能力矩阵 / 点名与单角色兜底 / 文本锚点去重与空值 /
data URL 转换与远端跳过 / 参考槽透传 / 不支持后端跳过且文本锚点仍在 / 失败回退重试），
前端 `SinIllustration.progress.test.tsx` 增加参考徽标与回退文案两例。

### 12.5 回归修复：拿不到参考图 ≠ 整单失败（v4.261.1）

v4.259.0 的门控只看「后端×模型能不能用参考槽」，**没看这一轮是否真取到图**：没选角色 /
角色只有远端剧照（不喂后端）/ 本地文件缺失时 `refImages` 为空却仍以 `mode=img2img` 发请求，
Herdsman 图生图端点整单拒绝（真机报错「marshal image request: 图生图需要提供参考图」）。
现收口为「**没有可用参考图 → mode/refMethod 一并清空、走纯文本**」：

| 情形 | 行为 | 用户可见 |
|---|---|---|
| 故事没选角色 / 多角色未点名 | 纯文本 | 无徽标（不弹噪声） |
| 选了角色但没有可用参考图（远端或缺失） | 纯文本 | 「参考图不可用 · 已按纯文本生成」 |
| 解析到可用参考图但生成失败 | 纯文本重试一次 | 同上 + `ref_fallback` |

解析收进纯函数 `sinResolveRefImages`（返回可用图 / 角色名 / **台账归属 id**）；归属不再取
`picked[0]`，而是「真正带参考图」的那个角色。测试：`sin_refs_test.go` +3（红→绿取证 +
纯函数矩阵 + 端到端 3 子例：没选角色 / 单角色只有远端剧照 / 多角色未点名）。

## 13. 原罪工具集 v1 + 过程卡（v4.262.0）

### 13.1 为什么

v4.256~v4.261 原罪只有「固定管线 + 前端触发」，模型**零工具**（`SinStream` →
`ai.ChatSimpleOptions` + `ChatStreamChunks`，请求体没有 tools 字段；前端也不处理
`tool_dispatch/tool_result`）。用户口径：**要过程卡，要看得见工具调用**，并点名加
**网络搜索工具**。

### 13.2 工具集 v1（原罪域内，不碰办公数据面）

| 工具 | 作用 | 归属/落点 |
|---|---|---|
| `web_search` | 联网搜索公开网页（真人资料：地名/年代/器物/专业细节） | 复用办公 builtin（`tool.LookupBuiltin("web_search")`，[search] 引擎扇出/SSRF 策略/代理同源；**schema 直接转发 builtin 的**，零漂移） |
| `web_fetch` | 抓取指定 URL 正文（搜索摘要不够时补细节） | 复用办公 builtin `web_fetch` |
| `sin_cast` | 查本故事已选角色卡（只读，不改设定/改名） | 角色库（跨板块共享资产层） |
| `sin_notes` | 故事便签/设定集：`action = list/read/write/set/delete`（按序号读写，≤200 条、单条 ≤2000 rune） | 原罪自有：`<用户配置目录>/gaea/sin/notes/<topicID>.json` |
| `sin_outline` | 故事大纲：`action = read/write`（整体替换，≤4000 rune） | 同上（同文件另一键） |

**刻意不做**：`sin_illustrate`（把插图从标记协议升级成工具）。插图已有自己的进度卡 +
串行队列 + 取消，标记协议 `@@插图\|…@@` 是「正文逐字节可回溯」的一部分；工具化只会多出
第二条出图路径，收益仅剩卡片形态，风险大于收益——留待另刀拍板。

**注（2026-09-12 本刀执行记录）**：并行实现期间有代理越过契约实现了
`sin_illustrate`/`sin_export` 两件（委托既有 `SinIllustrate`/`SinExportMarkdown`，方向不坏），
但其**产物回写链路**（工具产物 → 过程卡缩略图 / 重开还原）两个代理各以为对方会补、实际没人补，
属半成品。本刀已把这三份文件挪出树外存于 `.tmp/sin-next-knife/`（不删），下刀接完产物链路
再入册；本刀工具面按上表 5 个交付。

### 13.3 事件与落库契约（`sin-stream:<runID>` 通道，与既有 delta/reasoning/done 平级）

| type | 载荷字段 | 何时 |
|---|---|---|
| `tool_dispatch` | `id` / `name` / `args`（JSON 字符串）/ `read_only` | 模型请求工具、开始执行前 |
| `tool_result` | `id` / `name` / `output` / `error` / `elapsed_ms` | 工具执行完（成功或失败都发） |
| `notice` | `message` | 降级提示（如「当前模型不支持工具调用，已按纯写作继续」） |
| `done` | 既有字段 + **`tools`**：本次工具轨迹数组（见下） | 收尾 |

工具轨迹形态（`done.tools` 与消息 `extra.tools` **同一形态**，前端单点解析）：

```json
{"id":"call_1","name":"web_search","args":"{\"query\":\"…\"}","output":"…",
 "error":"","elapsed_ms":1234,"read_only":true}
```

### 13.4 工具循环（后端）

`runSinStream` 由单轮流式升级为**有界多轮**：每轮走 `ai.Client.ChatStreamMessages`
（ai 包新入口，与既有 `ChatStreamChunks` 共用同一份 `prepareStreamRequest` 装配——模型名
兜底、default 采样、modelhub 让位、Qwen3 思考预算守护都不漂移）→ 消费 `SSEChunk`
（content/reasoning 照旧发 delta；`Done.ToolCalls` 非空则进入工具轮）→ 执行工具 → 追加
`assistant(tool_calls)` + `tool` 消息 → 继续；**轮次上限 4**，最后一轮不带 tools 强制收尾。
轮内正文按轮序拼接落库（正文仍=模型输出，不改写）。整轮多请求的 token 用量按轮累加
（工具循环 = 同一轮里发了多次请求，成本必须如实合计）。

**降级与边界**：①首轮带 tools 直接失败（模型/端点不支持）→ 去掉 tools 重试一次并 emit
`notice`，写作不中断；②已有正文时工具轮中断 → 用已写出来的内容收尾（不吞故事）并给
`notice`；③未知工具 → 如实回错并列可用清单，不猜不静默吞；④工具结果 6000 rune 封顶，
截断留可见标记（模型与过程卡看同一份）；⑤最后一轮仍冒出工具调用 → 不执行（没有下一轮
消费结果），用已有内容收尾；⑥全程只吐工具调用、没有任何正文 → 不落空消息，报错收尾。
工具执行失败只写进轨迹（`error` 字段）并如实回给模型，不炸整轮。

### 13.5 前端过程卡（复用办公视觉语言，不搬办公 store 耦合）

- `SinProcessCard`（新，`pages/sin/`）：「思考过程 · N 字」折叠行（`reasoning`）+
  工具行（图标 + 中文标签：联网搜索/抓取网页/查角色卡/故事便签/故事大纲 + 目标摘要 +
  状态 + 用时），点开展开参数与输出（截断）；运行中显示动态态（**不编造百分比**——运行
  中只给不定态脉冲，耗时只显示后端回传的 `elapsed_ms`）。
- 位置：assistant 卡片内、正文之上（过程先于结果）；历史消息从 `extra.tools` 还原
  （折叠态，与办公过程卡收敛形态一致）；无思考无工具 → 整卡不渲染（纯正文回复不多空行）。
- 文件分工：展示映射（工具名→标签/图标/摘要，动作名中文化）在 `pages/sin/sinToolMeta.ts`
  （纯函数，可单测），组件文件只导出组件（react-refresh 的 only-export-components 纪律）。
- 复用：`gaea/lib/tools` 的 `subjectOf`（联网两件套的 query/url 摘要）与 `boundedOutput`
  （输出 60 行折叠 + 可见「已折叠 N 行」）+ `gaea/icons`；不引 `ToolCard`（它绑办公 store
  的 `Item`/任务卡活动注入）。

### 13.6 测试与门禁

Go +14：工具注册表（顺序/schema 合法/只读标记如实）/联网委托（schema 与 builtin 逐字节
同源）/角色卡渲染矩阵（未选/列示/精确/模糊/未命中/空字段不占位）/便签与大纲往返（含上限、
序号越界、路径守卫 fail-closed、损坏自愈）/线格式键集合逐键锁定/工具循环五例（多轮执行+
轨迹落 extra+次轮消息数组接得上、端点不支持 tools 的降级重试、轮次封顶+未知工具回错+
不落空消息、**调用预算拦同名狂奔**、**收尾轮仍要工具时的兜底正文**）。
vitest +13：过程卡 8（无内容不渲染/历史折叠与展开/运行态/失败态/参数美化与输出折叠/
标签与摘要纯函数）+ 轨迹解析 3（snake_case 解析/容错跳过/失败态推导）+ 故事流 2（过程帧
dispatch→result 合并 + 丢帧兜底 + notice；重开从 extra.tools 还原；**帧到达重置静默计时**）。

### 13.7 真机走查补记（2026-09-12，v4.262.0 exe）

单测覆盖不到、走查才暴露的三个问题（已修 + 已补回归锁）：

1. **搜索狂欢**：模型一轮连搜 12 次，轮次封顶后正文只剩 62 字 ⇒ 13.4 的调用预算
   （同名 ≤3 / 整轮 ≤8，超限不执行、如实回绝并写进轨迹）；
2. **收尾轮空转**：收尾轮（不带 tools）模型仍回工具调用 ⇒ 该轮没有正文、整单报错且
   **什么都没落库** ⇒ 收尾轮加「工具阶段结束」收束令 + 允许一次兜底收尾轮；
3. **前端误判超时**：静默超时是一次性 90s 计时器（不随帧重置）⇒ 工具循环的长回合必踩
   （后端已落库 827 字、前端却报超时）⇒ 改每帧重置的真沉默计时。

## 14. 右栏创作面板：角色 / 大纲 / 设定 / 插图（v4.263.0）

### 14.1 为什么

v4.262.0 给了模型 `sin_notes`（设定集）与 `sin_outline`（大纲）两个写作工具，底稿落
`%APPDATA%\gaea\sin\notes\<故事id>.json`——但前端没有任何读取通道：AI 写了什么用户
看不见。用户口径「增加类似办公的右侧面板，可以看角色、大纲、插图、设定」：把故事的
**工作底稿 + 产物**聚合成办公右侧面板同款的多页签面板。

### 14.2 数据面（新绑定 SinNotesGet，640→641，play 空间）

- `sin_store.go::SinNotesGet(topicID) → SinNotesView{notes, outline}`：只读同源便签
  文件（`sinNotesPath`+`loadSinNotes`），与工具侧共用 `sinNotesMu`；`sinTopicGuard`
  守卫；缺失/损坏=空清单空大纲（与工具侧同口径，辅助数据不阻断）。
- **只读**：写路径仍只在工具侧（AI 写作）——用户直接改底稿的冲突合并留待下刀。
- 实时性：页面在每回合结束（sending true→false 沿）重读一次，切故事即重读。

### 14.3 前端形态（SinSidePanel，对标 WorkspacePane）

| 页签 | 内容 | 数据源 |
|---|---|---|
| 角色 | SinCastPanel 原样入页签 | SinCastGet（不变） |
| 大纲 | sin_outline 大纲，pre-wrap 只读 | SinNotesGet().outline |
| 设定 | 设定集逐条卡片（#序号） | SinNotesGet().notes |
| 插图 | 全故事画廊：缩略图 → Modal 大图 | extra.illustrations + 正文反解 caption |

- 插图画廊收集=`collectIllustrations`（storyText.ts 纯函数）；路径→data URL 走
  AttachmentDataURL 通道（与流内插图同口径）；空态 V3Empty compact。
- 玩法/插图协议/边界三段静态说明收进面板头问号气泡（内容逐字保留）。
- 页签**纯文字**（不带图标）：真机走查实证 268px 宽度下「图标+两字」会折行成竖排——
  去图标+nowrap 后四页签单行（宽度预算 244<252）。
- 开合=顶栏 PanelRight 钮；记忆键 `gaea.sin.panelOpen`/`gaea.sin.panelTab`
  （原罪自有命名空间，不借办公 workbench 键）；宽窗默认开/窄窗默认收，
  替代原 1180px 强制隐藏媒体查询。

### 14.4 测试与走查

Go +1（绑定三态）；vitest +14（面板 8 + collectIllustrations 2 + useSinNotes 4）。
空间分类计数锁 462→463。真机走查见 releases/v4.263.0.md 产物行。

### 14.5 面板宽度可拖拽（v4.264.0）

用户口径「右侧面板宽度应该是可以自由拉伸的」。左缘拖拽手柄（`.sin-side-resizer`，
骑边框 8px 命中带、悬停/拖拽 accent 显色），指针拖拽范式与办公 useWorkspaceLayout
同源：window 级 pointermove 实时跟手、拖拽中 body 锁 col-resize+禁选中、pointerup
持久化、pointercancel 兜底。宽度记忆 `gaea.sin.panelWidth`；
`clampSinPanelWidth` 纯函数钳制 240~640 + 视口收敛（innerWidth-520）；双击复位 268；
画廊列数改 auto-fill minmax(104px,1fr) 随宽自适应（拖宽有实际收益）。
