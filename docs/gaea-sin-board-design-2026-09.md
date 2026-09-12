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
