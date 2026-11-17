# 第三批（F 批 · 前端）计划复核（决策视图）

> 对象：`batch3-execution-plan.md`（34 条，27–31 人日）。
> 口径同第四批：**不为蒸馏而蒸馏**——三条闸（用户可感知 / 可度量 / 成本≤收益），过不了就归档或转产品 backlog。
> 本次复核**只读**：`frontend/src` 一行未改。证据均为实测（命令见 §7）。
>
> **落地状态（2026-11-17）**：Wave 1–3（W1–W9）已全部完成并验证。
> 全量前端 Vitest 494 文件/4453 例、tsc+Vite build、全量 ESLint
> 及 Go Policy 相关包门禁均通过。

---

## 0. 结论

| 项 | 原计划 | 复核后 |
|---|---|---|
| 落地条目 | 34 条 | **11 条**（Wave 1 5 / Wave 2 3 / Wave 3 3） |
| 工期 | 27–31 人日 | **约 12 人日**（含单源化前置） |
| 转产品 backlog | — | 12 条（功能新增，需产品决策，不该占蒸馏工期） |
| 归档（已修/收益不可感知） | — | 11 条 |

四条定位纠错（这是本次复核的主要产出）：
1. **双壳不是"结构项"，是工期乘数**——实测 **308 个同名双份文件**，且文学壳的 `ApprovalModal.tsx:41` 同样是 `"3": "allow_session"`。**A1/A2 若只在 `gaea/` 修，文学壳继续错**。原计划把单源化放在波 B，顺序是反的。
2. **A6 已落地**（`markdownCut.ts:22–24` 的 v4.368 修复：未闭合围栏时切点回退到围栏行之前）——计划没跟上游代码同步，34 条里至少有 1 条应删。
3. **波 C 15 条里 12 条是产品功能**（设置索引、主题预览、多选器、统计热力图、首启步骤、schema 表单…），不是"机制蒸馏"带来的提升；它们缺的是产品决策与设计资源，不该混进蒸馏排期。
4. **B8（前端三层输出预算）与第四批 P2（后端截断 Policy）会做成两套口径**——同一个"省略多少行"的问题，前后端各定义一次必然漂移；必须同源（见 §5）。

---

## 1. 复核方法与实测证据

| 事实 | 证据 |
|---|---|
| 双壳都在线上 | `frontend/src/main.tsx:31–33`（`registerPage('WenxuePage', lazy(...))`，注释："办公板块完整拷贝，全共享引擎"） |
| 同名双份 308 个 | `gaea/**` 639 个 ts/tsx vs `wenxue/**` 328 个，同名交集 308（命令见 §7） |
| A1 真 bug 仍在 | `gaea/components/ApprovalModal.tsx:90`：`if (target?.tagName === "INPUT" \|\| "TEXTAREA" \|\| isContentEditable) return;` |
| A2 真 bug 仍在 | 同上 `:43` `"3": "allow_session"` + `:98` hardConfirm 下拒绝 `allow_session/persist_allow` → **按 3 无反应**；文学壳 `wenxue/components/ApprovalModal.tsx:41` 同样问题 |
| A6 已修 | `gaea/lib/markdownCut.ts:6–24`：`findStableCut` 已全量行扫描 fence 状态，"若切点落在未闭合 fence 内，回退切点到该行之前"（v4.368） |
| A5 部分存在 | `gaea/lib/tools.ts:449–478`：`boundedOutput` 有 `hiddenLines`/`totalLines`，但 `preview` 是**只留头**（`head` + 折叠计数），无尾部保留 |
| A7/A8 落点存在 | `gaea/hooks/useComposerMenus.ts`、`gaea/hooks/useSessionManager.ts` 均在 |
| B3/B4 前置 | `gaea/app/useAppKeyboard.ts` 存在；`gaea/lib/keymap.ts`、`escLadder.ts`、`textMetrics.ts`、`backtrack.ts`、`agentGroups.ts` **均不存在**（=未开工） |
| 滚动宿主 | `ChatPage.tsx:211` 用 `useFollowScroll`；`MessageList.tsx:149` 有 `TOP_GROW_PX` 扩载 → 原计划的"250ms setTimeout 还原"已不是当前形态，B3 的痛点描述需按现状重写 |

---

## 2. 逐条判定（34 条）

> 判定：**留**=进落地；**并**=并入别的条目；**延**=Wave 3 择机；**背**=转产品 backlog；**档**=归档（已修/无可测收益）。

### 波 A（9 条）

| # | 判定 | 理由（一句） |
|---|---|---|
| A1 审批模态吞键 | **留（Wave 1）** | 真 bug：模态开着时打字/回车穿透到 composer，用户会误发消息 |
| A2 hardConfirm「3=终止」 | **留（Wave 1）** | 真 bug：紧急终止键按了没反应，属安全相关 |
| A3 Tab 契约 | 并 | 并入 A7 的"异步三元守卫"同批做（同一处 keydown 分支，分开做要改两遍） |
| A4 回退判据纯函数 | **留（Wave 1）** | 可度量：回退失败的"数据损坏/半回退"风险；纯函数易测 |
| A5 披露量化 | **留（Wave 2）** | 需与 B8 同源（见 §5），单做会二次改 |
| A6 稳定切点 | **档** | 已修（`markdownCut.ts` v4.368），计划过时 |
| A7 异步三元守卫 | **留（Wave 1）** | 可度量：过期结果乱序覆盖（用户能看到列表跳变） |
| A8 列表稳定键去重 | **留（Wave 1）** | 可度量：重复行（刷新后同一会话出现两次） |
| A9 "不许假装成功"三处 | 并 | 拆进 A1/A4/A5 的测试用例（空态/缺字段断言），不单列条目 |

### 波 B（10 条）

| # | 判定 | 理由 |
|---|---|---|
| B1 ToolCard 单源 | **留（Wave 1，前置）** | 308 双份里被 A 批直接触达的一个；不改则每次修复要做两遍 |
| B2 其余双份收口 | 并 | 并入 B1 同一机制，但**只做被 A 批/A9 触碰到的文件**（ApprovalModal/Composer/tools.ts），不搞全量 |
| B3 阅读位置坐标系（XL） | 延 | 收益真实但成本 XL 且动滚动宿主（计划自评高风险）；先用"最小修"验证痛点是否还在（见 §4） |
| B4 键位表单源 | 延 | 依赖 G2 拍板（两壳键位合并）；单做会与 B5/B6 反复 |
| B5 Esc 阶梯表 | **留（Wave 2）** | 小而明确：Esc 该退到哪一层是用户每天碰到的行为 |
| B6 提示不切开 + 三档塌缩 | 并 | 并入 B5（同一派生源） |
| B7 统一按宽截断 | **留（Wave 2）** | 可度量：窄宽 880 下截断位置错（字符切 vs 显示宽度） |
| B8 工具输出三层预算 | **留（Wave 2，与 P2 同源）** | 可度量：长输出只留头 → 错误行在尾部时看不到 |
| B9 代理状态机/选区算法 | 延 | 需要多智能体面真实使用反馈；纯函数易做但收益未证 |
| B10 工作树 owner 标注 | 延 | 与 B9 同源，B9 未做前不单独上 |

### 波 C（15 条）

| # | 判定 | 理由 |
|---|---|---|
| C1 匹配两段化 + Esc 三元组 | 并 | 并入 A7（同一弹层状态机） |
| C2 会话面四件 | 背 | 功能新增（搜索/预览/解析/动作三态），需产品定交互 |
| C3 设置索引 + 不可用原因 | 背 | 同上 |
| C4 主题实时预览 | 背 | 设计资源 |
| C5 多选器 | 背 | 组件库新增 |
| C6 状态卡可复制 + 新鲜度 | 背 | 功能新增 |
| C7 统计顺序/热力图 | 背 | 展示口径，需数据侧配合 |
| C8 首启步骤状态 | 背 | 需 G1（信任语义）拍板 |
| C9 文案校验 + 草稿恢复 | 并 | 文案校验（禁控制字符）并入 A9 的"诚实性"用例；草稿恢复转 backlog |
| C10 schema 驱动表单 | 背 | 依赖 T10 审批枚举 + MCP 集成面，非本批 |
| C11 hooks 浏览 + 记忆双开关 | 背 | 设置面功能 |
| C12 目标落文件 | 背 | 属后端能力（引用句），前端只是展示 |
| C13 GenUI 护栏 | **留（Wave 3）** | gaea 强项的风险面：2MiB 上限 + 伪域名占位，**只加上限不做渲染模型改动** |
| C14 复制链与渲染闸门 | **留（Wave 3）** | 正确性 + 性能：512KB/10000 行/单行 4KiB 闸门缺失会直接卡 UI |
| C15 探索组归组 | 并 | 并入 B8（同一工具卡渲染链） |

---

## 3. 重排后的三波

```
Wave 1（约 5 人日）—— 一次修，两个壳都好
  W1 单源化前置（B1+精简 B2）：ApprovalModal / Composer / lib/tools.ts 三个文件先合并
  W2 A1 模态吞键 + A2 hardConfirm「3=终止」+ A8 列表去重
  W3 A7「异步三元守卫」（含 A3 Tab 契约、C1 匹配两段化并入）
  W4 A4 回退判据纯函数（含 A9 的空态/缺字段断言）

Wave 2（约 4.5 人日）—— 展示正确性
  W5 A5 披露量化 + B8 三层预算（与第四批 P2 共用同一 Policy 口径）
  W6 B5 Esc 阶梯（含 B6 提示三档）
  W7 B7 统一按宽截断（textMetrics）

Wave 3（约 2.5 人日）—— 风险面收口
  W8 C14 渲染闸门（512KB/10000 行/单行 4KiB）+ 复制链
  W9 C13 GenUI 护栏（只加上限与占位）

合计 11 条 / ≈12 人日（原 34 条 / 27–31 人日）
```

> 排序原则与第四批一致：**先修"用户会看到错"的**，再修"用户会看不懂"的，最后才补功能。

---

## 4. 三条"先验证再动手"（防止为旧痛点施工）

1. **B3（滚动坐标系）先量再改**：当前不是"250ms setTimeout 还原"（实测 `ChatPage.tsx:211` 已用 `useFollowScroll`，
   `MessageList.tsx:149` 用 `TOP_GROW_PX` 扩载）。做法：录一段 1200 条消息的滚动/切会话，量"位置跳变量"；
   跳变 <1 行就不用做 XL 重写。
2. **B4（键位表）先查冲突**：把两壳 `useAppKeyboard` 的按键集合导出做差集；若冲突 <5 个，先合并成一张表（半天），
   不做完整上下文位集。
3. **A5/B8 先对齐口径**：把"头尾各保留多少"定成前端展示常量 + 后端传输 Policy 两级（见 §5），否则先后端上线后前端再改一次。

---

## 5. 与第四批的合并点（必须同源，否则做成两套）

| 主题 | 第四批（后端） | 第三批（前端） | 同源方案 |
|---|---|---|---|
| 输出截断/预算 | P2 `strutil.Policy`（头尾 50/50、1.2 余量） | B8 三层预算（字节→行→单行头尾） | **后端负责"传输前截断"，前端负责"展示折叠"**；两者共用同一组常量名（`*_CAP` 硬上限 / `*_BUDGET` 可让路 / `*_PREVIEW` 折叠量——沿用 batch3 §2 #1 的三词约定），数值由同一份配置导出 |
| 披露量化 | — | A5 `expandable` 由隐藏量决定 | 判定函数放前端纯函数库，后端只在事件里给 `total/retained`，不做"该不该展开"的判断 |
| 审批决策 | P5 step 快照 + 审批一致性 | C10 `__approval` 五值 | 已由 batch3 §2 #3 规定"只能映射 `approval_mode`"，保留该约束 |

---

## 6. 门禁与拍板门（从 5 个减到 3 个）

**门禁**（每波收尾，三条硬门禁 + 每条一个断言）：
```powershell
cd frontend
npm.cmd run build      # tsc -b + vite build
npm.cmd run lint
npm.cmd test           # vitest run（含 perf-guards）
```
- 证据从"每项 60fps 录屏"降为：**一次基线录屏**（改前）+ 之后用 **DOM 节点数 / 首帧耗时**断言回归；
  不再要求每条都录（成本高、判定主观）。
- `git diff --stat` 只含声明文件；`FEATURE_*` 关闸时断言零订阅/零定时器（保留原 §6 硬约束）。

**拍板门（3 个）**：
| 门 | 内容 | 建议 |
|---|---|---|
| G1 双壳策略 | ① 单源化（gaea 为准，wenxue 只留差异 props）② 或明确 wenxue 冻结（只修真 bug 不做功能） | **①**，但只做被本批触碰的 3 个文件，不做全量 308 |
| G2 滚动坐标系 | 自研 anchor vs 现有 `useFollowScroll` 最小修 | 先做 §4-1 的量化，跳变 >1 行再选自研 |
| G3 GenUI 上限 | 2MiB 是否合适、伪域名占位是否影响预览 | 按 2MiB 落地，配置化可调 |

---

## 7. 复现命令（供自查）

```powershell
cd C:\AI\wubigrok\frontend\src
# 1) 双壳是否都在线上
Select-String -Path main.tsx -Pattern "WenxuePage|wenxue"
# 2) 同名双份数量
$g=Get-ChildItem gaea -Recurse -File -Include *.ts,*.tsx | % Name | Sort-Object -Unique
$w=Get-ChildItem wenxue -Recurse -File -Include *.ts,*.tsx | % Name | Sort-Object -Unique
(Compare-Object $g $w -IncludeEqual -ExcludeDifferent).Count
# 3) A1/A2 是否仍存在
Select-String -Path gaea\components\ApprovalModal.tsx -Pattern '"3"|INPUT|contentEditable'
Select-String -Path wenxue\components\ApprovalModal.tsx -Pattern '"3"'
# 4) A6 是否已修
Select-String -Path gaea\lib\markdownCut.ts -Pattern "fence|回退"
# 5) 滚动宿主现状
Select-String -Path pages\ChatPage.tsx -Pattern "useFollowScroll|setTimeout"
```

---

## 8. 一句话总结

第三批真正能给用户带来提升的只有 **5 件**：审批模态吞键、hardConfirm 终止键、回退判据、异步乱序、列表去重——
而它们**都必须在单源化之后做**，否则文学壳继续带着同样的 bug。其余 23 条里，12 条是产品功能（转 backlog）、
6 条并入他人、5 条归档（含 1 条已修）。原计划的 27–31 人日，可以压到 **12 人日**且每一条都能被用户直接感知。
