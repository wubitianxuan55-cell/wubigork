# 全仓审计第 23 批 · god-func 大拆第二刀（IN3-04 applyOne 表驱动化）· 2026-10-03

> 接续 [round-24（批次二十二）](round-24-p1-batch22.md)。**单线主代理直做**：IN3-04 `internal/schedule/ops.go` `applyOne` 430 行巨型 switch → 按 op 拆独立函数 + map 注册表分派——批 22 AP2-05 验证过的「对照测试先行的表驱动化」配方第二刀。
> 本批**不抬版本**、不动 CHANGELOG；交付物 = 本文件 + `.gaea/progress.md` 一条。
> 快照：开工时 HEAD = `e48c1c56`（源码包归档提交）；工作树干净。

---

## 〇、预核

- 审计指认复核成立：`applyOne` 106–553 行单函数 switch，12 种 op 类型全部内联（最长 patch_task 占 ~115 行）；每个 case 自带校验+变更+回执文案；`ApplyOps` 全链路是唯一测试入口。
- **既有基建盘点**：`ops_golden_test.go` + `frontend/src/schedule/ops_golden.fixture.json` 已是 Go/TS 双端对拍权威源（约 80 用例），冻结了 **错误文案 + after 状态** 两面。**缺口识别**：goldenCase 未冻结 **回执文案（summary）**——`ApplyOps` 的第三个输出面（供 Journal 与工具回执），也是拆分重构唯一可能静默漂移的面（ops_test.go 只零散断言少量子串）。
- 消费方清点：`internal/gaea/tool/builtin/schedule_tools.go:269` 唯一生产调用方（读 summaries）；xlsxedit 的 `applyOne` 是另一包同名函数不受影响；`Op` 结构体与 `ApplyOps` 签名不动 → 绑定面零变更。

## 一、落地（三步配方）

### 1. 契约先行（旧 switch 在位）

- `goldenCase` 增 `Summary []string` 字段；`TestApplyOpsGoldenFixture` 捕获并逐案断言（`reflect.DeepEqual`）。
- `-update-golden` 重生成 fixture（+summary 字段，121 insertions/29 deletions——删除行全为闭合括号 diff 配对，内容无损）；普通复跑绿 = **回执文案对旧 switch 冻结完成**。
- fixture 为共享文件：TS 侧 `opsSim.golden.test.ts` 只消费 `error/after`，加法字段对 vitest 惰性（实测 71 例全绿）。

### 2. 表驱动拆分（逐字搬运）

- `applyOne` 收窄为分派器：`opHandlers` 注册表命中→调用；未命中→`不支持的操作类型：%s`（与拆分前 default 分支同文案）。
- 12 case → 12 个 `applyXxx(p *Project, op Op) (string, error)`：`applyUpsertTask/applyPatchTask/applyRemoveTask/applySetLinks/applySetMeta/applyAutoChain/applySetBaseline/applyClearBaseline/applyUpsertResource/applyPatchResource/applyRemoveResource/applySetAssignments`。**函数体逐字搬运（含全部注释），仅 dedent 一层缩进**；注册表紧贴分派器，键序与原 case 序一致。
- 新增 `ops_dispatch_test.go` 守卫（新文件进显式 add 清单）：①`TestOpHandlersComplete` 注册表键集 ↔ `Op.Type` 文档枚举**双向锁**（漏注册=新增 op 永远拒、拼错键=对应 op 全拒，编译期抓不到只能守卫锁）；②`TestApplyOneUnknownTypeFailsClosed` 未知类型 fail-closed 文案逐字图钉。

### 3. 零编辑复跑（等价证明）

- `TestApplyOpsGoldenFixture` **测试零编辑**复跑全绿——Err/After/**Summary** 三输出面与拆分前冻结 fixture 逐字节一致；`go test ./internal/schedule/` 全包绿。

## 二、门禁

- 前台 `ci.ps1`：version drift / .tmp 卫生 / go build / go vet / golangci-lint（0 issues）/ `go test ./... -count=1` **137 包全绿**。
- vitest 全量：**3760/3762 绿**，2 红在 `src/pages/OriginalSinPage.tabs.test.tsx`（原罪页轨迹页签）——与本批零文件交集（本批前端只动 fixture JSON），单文件复跑 4/4 绿，定性环境瞬时（在册 flaky 先例：sin 族并发）。
- 前端 build（ci 因 vitest 抛错未走到）：`npm run build` exit 0 补跑收口。
- 提交后复验「树干净 + HEAD 可编译」（批 21 教训）。

## 三、大拆配方进在册（第二刀结论）

- AP2-05（63 键 setter 表）+ IN3-04（12 op 处理器表）两刀实证：**「契约先行→表驱动→零编辑复跑」对 switch 型 god-func 是零行为差异形态**；golden/对照测试缺的面（本批=回执文案）必须先补冻结再动手。
- god-func 存量下一刀候选：AP2-01 `runSinStream`（265 行串六职责）——非 switch 型，语义拆分，不适用本配方，需独立预核。

## 四、留池/下一批

- 下一批候选：AP2-01 runSinStream 大拆第三刀（先预核）+ god-file 大拆 ~26 + coupling ~20（拍板）+ 零散死码（等 109 绑定删除拍板）。
