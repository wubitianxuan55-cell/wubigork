# 第三批蒸馏计划：前端落地（F 批施工排期 · 只出计划、不改代码）

> **边界**：本文是**计划**。`C:\AI\wubigrok` 仍只读——本轮不写任何 gaea 源码/配置/测试/依赖。
> 落地另开批次（第四批），按本文的波次与门禁执行。
>
> **上游依据**：codex `rust-v0.165.0`（commit `e374836b`），前端二十篇资产：
> 波 1 `asset-27…33`（消息流与渲染）、波 2 `asset-34…37`（卡片/输入/会话/回退）、波 3 `asset-38…46`（九面清扫），
> 全量覆盖见 `frontend-coverage-ledger.md`。
>
> **前两批**：T1–T5（已落地/在改）、T6–T15（已排期）。本批与二者**零代码重叠**，但有 3 处契约需要对齐（§2）。

---

## 1. 批次定位

| 问题 | 回答 |
|---|---|
| 这一批做什么 | 把 F27–F46 二十篇里抽出的**施工卡**收敛成一条可执行排期，按"错得离谱 → 结构性 → 功能"三段推进 |
| 不做什么 | 不动 Go 后端、不动 Wails 绑定面、不做产品新功能设计（只搬机制） |
| 完成判据 | 每条有：落点 `file:line`、纯函数单测或组件测试、性能断言、开关与回退、真机走查记录 |
| 与 T 批关系 | T 批是**上下文装配/执行语义**（后端），本批是**界面契约**（前端）；共享的只有常量命名与单源化纪律 |
| 总量估算 | 34 条（波 A 9 / 波 B 10 / 波 C 15），相对规模 S=0.5 人日、M=1、L=2、XL=3；合计约 **27–31 人日**，可按波独立交付 |

---

## 2. 与前两批的对齐点（三处，必须先定，否则落地时会撞）

| # | 冲突点 | 现状 | 本批要求 |
|---|---|---|---|
| 1 | **常量命名前缀** | T5 用"字节预算"（后端 schema 预算）；F29/F34 也用预算 | 统一后缀语义：`*_CAP`（硬上限）/`*_BUDGET`（可让路预算）/`*_PREVIEW`（折叠展示量），三词不许互借 |
| 2 | **"稳定区"与"上下文碎片"** | T6 碎片契约叫 fragment/section；F27 叫 stable region | 命名空间分开：`ContextSection`（装配）vs `StableRegion`（渲染），禁止同名缩写 |
| 3 | **审批决策枚举** | T 批新增 `internal/gaea/permission/approval_mode.go` | F42-B 的 `__approval` 五值**只能映射**到它，UI 层不得另立枚举 |

---

## 3. 条目清单（34 条）

> 类型：**正**=正确性（行为与提示相反/数据会错）、**构**=结构与单源、**能**=性能与容量、**功**=功能补齐。
> 规模：S/M/L/XL。出处：对应资产篇。

### 波 A · 正确性 hot-fix（9 条，先做，纯前端）

| # | 条目 | 类型 | 落点 | 规模 | 出处 |
|---|---|---|---|---|---|
| **A1** | 审批模态禁止键盘穿透：模态期间除本卡键位外一律吞掉（`INPUT/TEXTAREA/contentEditable` 不再直接 return） | 正 | `gaea/components/ApprovalModal.tsx:78-81` | S | F31 |
| **A2** | 硬确认卡「3 = 终止」生效：`hardConfirm` 下键位派生独立一组（3=abort / 1=allow / 2=deny），修 `KEY_DECISIONS["3"]=allow_session` 与 `:89` 拦截的自相矛盾 | 正 | `ApprovalModal.tsx:40-46,89,172` | S | F31 |
| **A3** | Tab 契约：补全前 resync → 立即派发型命令 → 保尾部补全 → `hasVisible && !selectable` 吞键 | 正 | `gaea/components/Composer.tsx` keydown、`hooks/useComposerMenus.ts:105-119` | M | F35-C |
| **A4** | 回退判据纯函数：回合边界/历史完整/非在飞/非隐藏；失败把 prompt 回填 composer 并报错 | 正 | 新 `gaea/lib/backtrack.ts`；`pages/chat/branches.ts` 接入 | M | F37-B/C |
| **A5** | 披露量化：`expandable` 改由"实际隐藏量"判定；`hiddenLines` 计入宽度裁断条数；hidden=0 不给展开 | 正 | `gaea/lib/tools.ts:449-482`、`components/ToolCard.tsx:186,365-377` | M | F34-A |
| **A6** | 稳定切点不许回退：围栏未闭合时切点停在围栏前而不是退回上一段，保证"稳定段字符串不变" | 正 | `gaea/lib/markdownCut.ts`（`findStableCut`） | M | F27 |
| **A7** | 异步三元守卫：`pendingQuery/displayQuery/waiting` + 过期结果丢弃 + loading 文案（斜杠参数与 @ 搜索两处） | 正 | 新 `gaea/lib/useAsyncQuery.ts`；`useComposerMenus.ts:28-38,63-70` | M | F35-D |
| **A8** | 会话列表稳定键去重：刷新/追加/删除按 `key` 合并，禁重复行 | 正 | `gaea/hooks/useSessionManager.ts:32-85`、`lib/types`（`SessionMeta`） | S | F36-A |
| **A9** | "不许假装成功"三处收口：空态五分支（断线/加载/失败/空/未匹配）、Git 摘要缺字段省略而非报错、卡片可展开必有内容 | 正 | `gaea/components/TaskInboxPanel.tsx`、`GitPanel.tsx`、`ToolCard.tsx` | S | F38-C、F43-D、F34-A |

**波 A 退出判据**：三个与提示相反的 bug（A1/A2/A6）各有回归测试；A3/A4/A5 为纯函数 + 组件测试；
`perf-guards` 无新增告警；真机走查记录"审批卡开着时打字/回车无效、按 3 终止成功"。

### 波 B · 结构、单源与容量（10 条）

| # | 条目 | 类型 | 落点 | 规模 | 出处 |
|---|---|---|---|---|---|
| **B1** | `ToolCard` 单源化：`wenxue/components/ToolCard.tsx`(18.4KB) 改为 re-export `gaea/`(19.6KB) 或反向，差异抽 props | 构 | `wenxue/components/ToolCard.tsx` | M | F34-E |
| **B2** | 其余双份收口：`SubagentThread`(33.4×2)、`CapabilitiesPanel`/`MemoryPanel`/`ToolsSection`、`useComposerMenus` 覆盖 wenxue | 构 | `wenxue/components/*` | L | F38/F42 |
| **B3** | 阅读位置改坐标系：`Anchor{entryKey,index,offset,rowBias}` 取代 `scrollTop` + 250ms `setTimeout` 还原 | 构 | `components/chat/MessageList.tsx`、`pages/ChatPage.tsx:211`、`hooks/useFollowScroll` | XL | F28 |
| **B4** | 键位表单源：`(context, action)` 表 + 上下文位集 + 单键/和弦两层，`CommandRail` 内联键位迁出 `MainLayout.tsx:257-347` | 构 | `gaea/app/useAppKeyboard.ts`、新 `lib/keymap.ts` | L | F32 |
| **B5** | Esc 阶梯表（依赖 B4）：`escLadder.ts` 返回 owner，`useAppKeyboard.ts:34-38` 只做派发 | 构 | 新 `gaea/lib/escLadder.ts` | M | F37-A |
| **B6** | 提示不切开 + 三档塌缩（依赖 B4）：`firstFittingLine` / `wrapHintRows` / 3→2→1 列 | 构 | 新 `components/ui/HintRow.tsx`、帮助浮层 | M | F37-D/E |
| **B7** | 统一文本度量：`lineWidth/truncateToWidth`（显示宽度、`Intl.Segmenter`），替换字符切；`MAX_RENDERED_ROWS` 显式化 | 能 | 新 `lib/textMetrics.ts`；`MessageList.tsx` | M | F44-A/B、F34 §二 |
| **B8** | 工具输出三层预算（依赖 B1）：字节→行→单行头尾 + `total/retained` 分离 + 诚实省略标记 | 能 | `gaea/lib/tools.ts`（`boundedOutput` 重构） | L | F34-B/C |
| **B9** | 代理状态机与选区算法（纯函数）：`groupForStatus`、`statusBadge`、`nextSelectionAfterRemoval`、`reconcileSelection` | 构 | 新 `gaea/lib/agentGroups.ts` | M | F38-A/B |
| **B10** | 工作树 owner 标注（依赖 B9 的分组键）：`ownerFor(worktree, sessions)` + 动作守卫 | 构 | `GitPanel.tsx`、新 `lib/worktreeOwners.ts`、`lib/projectGroups.ts` | M | F43-A/B、F38-D |

**波 B 退出判据**：两壳渲染同一夹具逐像素一致（B1/B2）；`perf-guards.test.ts` 每条新增订阅/rAF/全局监听各有一条断言；
流式行 60fps 录屏改前/改后比对（B3/B7/B8）；DOM 节点数有上限且可测。

### 波 C · 功能补齐（15 条，可分三小批交付）

| # | 条目 | 类型 | 落点 | 规模 | 出处 |
|---|---|---|---|---|---|
| **C1** | 匹配两段化 + Esc 身份三元组（exact→prefix；`{query,token,occurrence}`） | 功 | `useComposerMenus.ts:18,24,103` | M | F35-A/B |
| **C2** | 会话面四件（依赖 A7/A8）：5 字段搜索、预览三态、名字解析不猜、动作三态 | 功 | `Sidebar.tsx`、新 `lib/sessionSearch.ts`/`sessionLookup.ts`、`SessionPreview` | L | F36-B/C/D/E |
| **C3** | 设置索引 + 不可用原因（依赖 C3 自身先行） | 功 | 新 `lib/settingsIndex.ts`、`SettingsPage.tsx`、`CommandPalette` | L | F39-A/B |
| **C4** | 主题实时预览（含 add/del/context 三类样本） | 功 | `design-system/` 注入层 | M | F39-C |
| **C5** | 多选器（必选可配、取消恒可用、可选重排、实时预览） | 功 | 新 `components/ui/MultiSelect.tsx` | M | F39-D |
| **C6** | 状态卡可复制 + 新鲜度标记（依赖 B7 的列宽稳定） | 功 | `AppStatus.tsx`、新 `lib/statusText.ts`、`RuntimePage.tsx` | M | F40-A/B |
| **C7** | 统计：枚举顺序数组 + 热力图缺数据不压缩 | 功 | 新 `lib/usageOrder.ts`、`RuntimePage.tsx` | S | F40-C/D |
| **C8** | 首启步骤状态（`Hidden` 不计分母）+ 取消三出口 | 功 | 新 `lib/onboardingSteps.ts`、`WelcomeScreen.tsx` | M | F41-A/B |
| **C9** | 文案校验（长度 + 禁控制字符）+ 启动草稿恢复 | 功 | 新 `lib/noticeGuard.ts`、`lib/composerDrafts.ts` | S | F41-C/D |
| **C10** | schema 驱动表单 + 审批注入字段（依赖 §2 #3 对齐） | 功 | 新 `components/ui/SchemaForm.tsx`、`lib/schemaForm.ts` | L | F42-A/B |
| **C11** | hooks 浏览 + 记忆双开关 | 功 | `CapabilitiesPanel.tsx`、`MemoryHubPage.tsx` | M | F42-C/D |
| **C12** | 目标落文件（超阈值文本/图片入 `attachments/` + 引用句） | 功 | 新 `lib/goalFiles.ts`、ChatPage 发送钩子 | M | F43-C |
| **C13** | 动效裁决 + 空态白名单 + GenUI 护栏（2MiB/伪域名）+ 提示外置 | 功 | 新 `lib/motionPolicy.ts`、`EmptyState`、`genui/*` 解析层、`assets/tips.json` | M | F45-A/B/C/D |
| **C14** | 复制链与渲染闸门（波 1 余项）：表格→GFM 源表、剪贴板双表示、512KB/10000 行/单行 4KiB 统一闸门 | 功 | `lib/{tableData,clipboard,codeHighlight}.ts`、`ChatMarkdown` | L | F29/F33/F30 |
| **C15** | 探索组（依赖 B1）：`read/list/search` 相邻归组并保留组内推理 | 功 | `ToolGroup.tsx:86-115` | M | F34-D |

**波 C 退出判据**：每条带纯函数单测 + 组件测试；所有新能力**默认关闭**，关闸时与现状逐字节等价（见 §6）；
每批交付后跑一次真机走查（Wails 壳，含窄宽 880/1100 两档）。

---

## 4. 顺序约束（10 条，违反就会返工）

1. **A1/A2 先于一切**：模态会吞掉/放行按键，其它验收在它修好前都不可信。
2. **B4（键位表）先于 B5（Esc 阶梯）与 B6（提示）**：阶梯要挂上下文位集，提示要从表派生。
3. **B3（坐标系）先于任何滚动相关项**（C2 预览、C6 列表刷新、B7 上限）。
4. **B1（ToolCard 单源）先于 B8 与 C15**：否则改一份、另一份静默退化。
5. **A5 披露量化先于 B8 三层预算**：先让"该不该展开"正确，再谈"保留多少行"。
6. **B9 状态机先于 B10 与 C2**：owner 与分组键复用同一份 git 身份函数。
7. **B7 统一按宽截断先于 C14**：复制链与表格对齐依赖同一度量。
8. **A7/A8 先于 C2**：预览会新增请求与列表行，守卫与去重必须先就位。
9. **C3（设置索引）先于 C8（首启信任 UI）**：信任项的"不可用原因"表达复用同一个形态。
10. **任何"新增订阅/rAF/全局监听"同批加 `perf-guards` 断言**——不另立批。

---

## 5. 验收口径（每类给最小证据，不许口头通过）

| 类型 | 必备证据 |
|---|---|
| 纯函数（B9/B5/A4/B7/C1） | vitest 单测：正常 + 边界 + 空输入；错误输入返回确定值（不抛） |
| 组件行为（A1/A2/A3/A5/C5） | RTL 用例：键盘事件序列 → 断言动作与可见文案；含"不该发生的事"（如打字不应穿透） |
| 流式/性能（B3/B7/B8） | 改前/改后 60fps 录屏；DOM 节点数上限；`perf-guards.test.ts` 断言 |
| 两壳一致（B1/B2） | 同一夹具在 `gaea/` 与 `wenxue/` 渲染逐像素比对（或 DOM 快照 diff） |
| 诚实性（A9/C6/C9） | 失败路径有可见文案且不翻转成功态；缺字段省略而非报错 |
| 真机（每波收尾） | Wails 壳走查记录：1440 / 1100 / 880 三档 + 明暗两主题 |

---

## 6. 开关与回退矩阵

| 条目组 | 开关 | 默认 | 关闸等价性 | 回退方式 |
|---|---|---|---|---|
| A1/A2/A3/A4/A5/A6 | 无开关（缺陷修复） | — | — | 单 commit revert |
| A7/A8 | 无开关（内部重构） | — | 行为等价（仅修乱序/重复） | revert |
| B1/B2 | `FEATURE_SINGLE_SOURCE_TOOLCARD` | 关 | 是（旧路径保留） | 关闸即回旧组件 |
| B3 | `FEATURE_ANCHOR_SCROLL` | 关 | 是（像素路径保留） | 关闸回 `scrollTop` |
| B4/B5/B6 | `FEATURE_KEYMAP_V2` | 关 | 是（旧 if 链保留） | 关闸回旧键盘实现 |
| B7/B8 | `FEATURE_TEXT_METRICS` / `FEATURE_OUTPUT_BUDGET` | 关 | 是 | 关闸回字符切/旧预算 |
| B9/B10 | `FEATURE_AGENT_GROUPS` | 关 | 是（不渲染新标注） | 关闸隐藏 |
| C1–C15 | 各自 `FEATURE_*`，**默认关** | 关 | 是 | 逐项关闸 |

**硬约束**：`FEATURE_*` 关闭时，生产路径不得出现任何新增分支的副作用（订阅/定时器/rAF 一律不注册）。

---

## 7. 门禁（每波收尾必跑）

```powershell
cd frontend
npm.cmd run build      # tsc -b + vite build
npm.cmd run lint       # eslint .
npm.cmd test           # vitest run（含 perf-guards.test.ts）
```

附加门禁：
- `git diff --stat` 必须只含本批声称的文件（防顺手改动）；
- 真机走查：`GAEA_WEBVIEW_DEBUG` CDP 起壳，1440/1100/880 × 明暗，零前端报错；
- 构建产物核对按 `docs/gaea-build-guide.md`（本批若需出包才执行，默认不出包）。

---

## 8. 风险登记（前 5 条为高）

| # | 风险 | 触发条件 | 缓解 |
|---|---|---|---|
| 1 | `MessageList` 是历史踩坑区（`content-visibility` + 尾部窗口 + 吸底跟随） | 动 B3/B7 时 | 先加录屏基线；`MessageList.tsx:1-30` 的注释视为约束而非背景 |
| 2 | 两壳漂移 | B1/B2 只改一边 | 单源化先行；CI 加"同名文件内容一致"守卫 |
| 3 | Wails 序列化旧账（Go struct 缺 json tag → 前端读空） | C2/C10 扩 `SessionMeta`/表单模型 | 扩字段必给 camelCase tag + mock 同步 + 一条端到端用例 |
| 4 | `perf-guards` 漏断言 | 任何新增订阅 | 同批加断言，评审时逐条核 |
| 5 | 开关关不干净（关闸仍有副作用） | 新增 hook 生命周期 | 关闸用例：断言零订阅/零定时器 |
| 6 | 审批决策枚举双份 | C10 实现时 | §2 #3：只能映射 `approval_mode` |
| 7 | GenUI 护栏误伤（gaea 强项） | C13 | 只加"上限 + 占位 + 不引外链"三条，不改渲染模型 |

---

## 9. 需要拍板的 5 个门（不拍板就卡住）

| 门 | 内容 | 影响条目 | 建议 |
|---|---|---|---|
| G1 | **信任语义**：目录信任 vs 空间策略/权限模式/工作区，谁是第四层？ | C8 | 复用 `permission` 包，不另立概念 |
| G2 | **键位表单一来源**：`useAppKeyboard`（gaea）与 `wenxue/app/useAppKeyboard`（wenxue）先合并还是各自演进？ | B4/B5/B6 | 先合并到 `lib/keymap.ts`，壳只接线 |
| G3 | **滚动坐标系实现选型**：自研 anchor vs 引入库 | B3 | 自研（已有 `useFollowScroll` 基础，避免新依赖） |
| G4 | **GenUI 护栏边界**：2MiB 是否合适；伪域名占位是否影响现有预览 | C13 | 先按 2MiB 落地，实测后调 |
| G5 | **单源方向**：以 `gaea/` 为准还是 `wenxue/`（后者多个组件更大） | B1/B2 | 以 `gaea/` 为准（功能全集在 gaea），wenxue 只保留差异 props |

---

## 10. 切分建议（3 个可独立交付的 PR 组）

| PR 组 | 内容 | 预估 | 前置 |
|---|---|---|---|
| **PR-1 正确性** | A1–A9 | 5–6 人日 | 无（可立即开工） |
| **PR-2 结构** | B1–B10（B3 可单独一个 PR，因为它动滚动宿主） | 12–14 人日 | PR-1 的 A5/A7/A8 |
| **PR-3 功能** | C1–C15（C3+C5 一批、C2+C6+C7 一批、C8–C13 一批、C14 单独一批） | 9–10 人日 | PR-2 的 B1/B4/B7/B9 |

---

## 11. 汇报格式（落地批次用，比第二批多两行）

```
F<n> <条目号> <名称>
基线：<测试文件> N 例 / 既有失败 M 例
改动：<file:line> 一句话（列全，不多不少）
门禁：build / lint / test 结果
开关：默认 <关/开>；关闸等价：是/否（附关闸用例名）
性能：<是否新增订阅/监听/rAF> + 60fps 录屏或 DOM 节点数
两壳：<是否双份改动/是否单源>
风险：<是否触碰 ChatPage 滚动宿主 / 是否扩 Wails 契约>
```

---

## 12. 交付物

1. 本文：`docs/codex-distill/batch3-execution-plan.md`
2. 依据：`asset-27…46` 二十篇 + `frontend-coverage-ledger.md`（覆盖可核对）
3. 落地后需新增：每条的测试文件路径与开关名，回填到本文对应行（计划 → 台账）

> 本文不含任何"实现代码"——落地批次开工时，按 §10 的 PR 组领条目，逐条按 §5 取证、按 §6 留开关、按 §11 回报。
