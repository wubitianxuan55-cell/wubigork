# 全仓审计第 18 批 · FE3-03 死绑定检测 + GA6 族收尾（主代理直做）· 2026-10-03

> 接续 [round-19（批次十七）](round-19-p1-batch17.md) §4.1 的两条候选：FE3-03（双全集数据源已备齐）+ 组价两面接 `cost.Enhance`（一行事）。
> 本批**不抬版本**、不动 CHANGELOG（发版仪式归用户）；交付物 = 本文件 + `.gaea/progress.md` 一条。
> 快照：开工时 HEAD = `c0b30b87`（批次十七）；工作树干净。两件量级小，**主代理直做**（批 15 线 1 先例：不值得一个子代理波次）。

---

## 〇、主代理预核（2026-10-03，读码实测）

| # | 审计项 | 预核结论 | 关键证据 |
|---|---|---|---|
| 1 | **FE3-03** 漂移检查只认名字被认领，死绑定无法发现 | **成立，且现场比审计的「7 个」大一个量级**：批 16 把 LegacySurfaceNames 生成化（184 名）后，「被认领」仍不等于「有人调用」。审计 7 死名实测 6 死 1 活（`GaeaUsageOverview` 已复活）；真实调用形态=legacy 名以**全名**文本出现在调用点（`NovelGhostSuggest` 在 GhostText.tsx 命中）⇒ 全名文本匹配即正确检测器 | 首轮全量实测：**184 名中 109 名零命中（59%）**——v1 小说域/office daemon/TTS·语音/whisper/记忆·大脑各族成片遗留 |
| 2 | **GA6 族残留**：组价两面直调钩子 + 内联 `<3/10` 阈值 | 成立（批 16 线 1 足迹外残留）：`gaea_cost_compose.go:89-99` 与 `cost_compose.go:74-84` 各持一份 | `cost.Enhance`（批 16 产）签名完全匹配两面钩子 |

## 一、落地（主代理直做）

### 1.1 组价两面接 cost.Enhance（GA6-04 收尾）

- `internal/app/gaea_cost_compose.go`：`<3 补召回 + 精排` 内联块 → `cost.Enhance(desc, similar, cost.SearchHooks{Recall: a.semanticCostRecall, Rerank: a.rerankCostSearch}, 12)`——与绑定面（memory_hub，limit 20）/工具面（cost_tools，limit=用户 limit）共用同一编排出处；组价面 rerankLimit=12 归调用面（Enhance 既有语义）。
- `internal/gaea/tool/builtin/cost_compose.go`：同改（钩子经闭包注入 ctx/store）；空态判断后移到 Enhance 之后——可观测等价（精排实现 ≤8 条短路，空列表零副作用；批 16 已证同款等价性）。
- 验证：`go build`/`go vet`/gofmt 干净；builtin 全包 12.9s 绿 + app compose 族测试绿（records round-trip 等 4 例）。行为不变由 Enhance 既有 7 子用例 + 两面包测试背书；compose 检索路径无独立测试（**留池小项**：补一条 `<3 触发补召回` 的 compose 级用例）。

### 1.2 FE3-03 死绑定检测守卫（`check-bindings-drift.ps1` §4）

**实现**（审计 fix 落地，按批 13「前端驼峰名反查」教训校准口径）：
- legacy 面（`legacyBindings.ts` 生成物，184 名）逐名在 frontend/src 做**全名 `\b` 词边界文本命中**——实测确认 legacy 名以全名出现在调用点，无需驼峰变换（批 13 的反查纪律用于「名字怎么写」，此处实测「名字就是全名」）。
- 排除集：生成物三件（bindingNames/legacyBindings/bindingSignatures——全名清单混入=全员假活）+ 守卫自身（drift/spaceBindings）+ `types/wails.d.ts`（类型声明非调用）+ `mock/**`（模拟实现非真实路径）。命中口径=文本命中（注释/测试也算——审计原口径，偏保守只抓真正不可达；测试命中算活的双刃性在 §3.2 登记）。
- **在册死名单基线 = 首轮普查 109 名**（删除=绑定面 744→635 候选，按 v4.429 在册口径「零调用者绑定删除需拍板」**整批留拍板**，本守卫只拦普查外**新增**死绑定=「新导出即失联」的接线漏；在册条目复活提示清册）。

**过程事故两起（主代理自己的，如实登记）**：
1. **排除模式正反斜杠不匹配**：`$_.FullName` 是反斜杠、排除模式写正斜杠 ⇒ 三个生成物清单根本没被排除，全员假活（首轮「6 条复活」假象）。
2. **正则不容行尾注释**：生成器给 7 个名挂了 `// 版本注解` 尾巴，`^\s*"...",?\s*$` 漏解析（177/184）⇒ 漏检的 6 个已知死名被误报「复活」。探测配方=`.tmp` 探针 .ps1（内联 PS 撞转义，在册坑第 N 次）。

**反向证据（主代理亲手）**：把活名 `NovelGhostSuggest` 的全部引用改名（组件+测试两处——单改组件不够，测试 mock 也算命中）→ 守卫红：`发现清单外死绑定 1 个（legacy 面共 184 名，零源码命中）：- NovelGhostSuggest`，**真 exit 1**（管道后 `$?` 是 tail 的——在册坑再踩，取重定向日志的真码）→ 还原 exit 0。

## 二、本批结论

| 项 | 变化 |
|---|---|
| FE3-03 | 死绑定从「编译器与 CI 双双沉默」（审计原话）→ **守卫机器可检**（新增死绑定 exit 1）；普查 109 名入册，删除走拍板 |
| GA6-04 | 组价两面内联阈值收敛 `cost.Enhance`——三消费面（绑定/工具/组价）编排与阈值唯一出处，GA6 族全清 |
| 守卫 | `check-bindings-drift.ps1` 从「bindingNames↔Go 一致性」扩为「一致性 + 死绑定检测」双职责（744 一致 + 184 legacy 逐名文本命中） |

### 门禁（收线跑，终值见台账底部）

（ci.ps1 前台单次结果收线时回填）

## 三、留池与拍板

**新增拍板（本批最重要的产出）**：**109 条零调用绑定删除**（候选名单=`check-bindings-drift.ps1` §4 knownDead 普查清册，v1 小说域 ~30/whisper·语音 ~20/office daemon ~9/记忆·大脑 ~10/TTS ~8/杂项）——删除=绑定面 744→635、Go 侧下导出+再生清单+门面委托同步，收益=绑定向真实面收缩 + 后续漂移守卫信噪比提升；风险=若有外部调用方（headless 脚本/未来前端）需逐名复核。**等用户裁决后单开一刀**。

**留池**：compose 检索路径无独立测试（补 `<3 触发补召回` 用例）；测试命中算活的双刃性（只被死代码+测试引用的名字守卫看不见——文本命中口径的已知边界）。

## 四、审计原文纠错

| # | 审计原文 | 实测 | 处置 |
|---|---|---|---|
| 1 | FE3-03「743 个 Go 名中有 7 个零引用」 | 6 死 1 活（GaeaUsageOverview 已复活）；**首轮全量普查实为 109 名零命中**——审计数字是当时口径，且只查了 Gaea 前缀名（legacy 短名族没进它的视野） | 检测器 + 普查清册；删除留拍板 |
| 2 | FE3-03 fix「check-bindings-drift.ps1 增一遍文本命中检查」 | 照做成立；补充：排除集是口径成立的前提（生成物/mock/声明不排除则全员假活——本批两起事故实证） | §1.2 排除集六类 |

**收线终值**：ci.ps1 前台 CI OK / exit 0（golangci 0 issues · go test 全量 · 前端 build · vitest 428 文件 3723 例 · E 系列 · 卫生四查）；守卫：primitives --strict 无新增 / test-ctors 364 零新增 / contract-drift 无新增 / **bindings-drift OK@744 + 死绑定检测 109 在册零新增**。
