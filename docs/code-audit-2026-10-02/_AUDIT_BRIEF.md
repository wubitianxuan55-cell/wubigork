# gaea 全仓代码审计 · 单元审计员作业书（2026-10-02）

你是一次并行代码审计中的**单元审计员**。任务：**只读代码审计，专挖「屎山代码」**——上帝文件/上帝函数、复制粘贴与平行实现、死代码与僵尸资产、吞错与静默失败、层级倒挂与全局状态、并发隐患、多套并行口径、兼容壳堆积、巨型 switch/flag 参数。

## 0. 你的任务参数
派单提示会给出：单元 ID、板块名、文件足迹、本单元重点。足迹=主要对象（只读）；为判断「是否平行实现」可以读邻近文件（只读）。

## 1. 已知基线（不要当新发现报）
- `go vet ./...` 与 `golangci-lint run ./...`（errcheck/govet/ineffassign/staticcheck/unused/misspell 全开）**当前 0 issues**：问题在结构层。别报「err 变量没处理」这类 lint 已覆盖的表面项，除非它导致业务静默失败（要有证据链）。
- 历史审计（重合就写进 prior_audit 字段，标「延续/仍开放」，不要当新发现）：
  - docs/gaea-optimization-direction-2026-09.md（2026-09-26 全仓审计 + P0~P2 优化项 + 14 模块评分）
  - docs/gaea-whisper-deadcode-survey-20260925.md
  - docs/gaea-backend-perf-survey-2026-09.md
- 仓库=gaea：Windows 桌面 AI 助手，Wails v2（Go 1.26 后端 internal/ + React/TS/Vite 前端 frontend/src），私有个人工具，当前 v4.451.0。绑定面=743 个导出方法经 internal/app/bindings_*.go 门面暴露给前端。

## 2. 方法（按序，别跳）
1. 用 glob / pwsh 列出足迹内**非测试**源文件与行数，按 LOC 降序（前端 .ts/.tsx，后端 .go）。
2. **精读 LOC>600 的文件全文**（屎山高发区），中等文件抽样；必须用 read 读真实内容，禁凭文件名猜。
3. 模式扫（用 grep 工具，命中后 read 看上下文）：`TODO|FIXME|HACK|XXX|临时|兜底|兼容|legacy|deprecated|废弃|占位`、`panic(`、`recover()`、`_, _ =`、`nolint`、`time.Sleep`、巨型 `switch`、`interface{}`/`any` 泛滥、同名函数或组件多份。
4. 交叉验证是否已有同功能实现（app handler vs 域包；前端 mock vs 真 bridge；两套 i18n；两个同名视图/工具函数）。
5. 每条发现**必须**带真实 `路径:行号` + 不超过 3 行的逐字源码摘录。行号错=整条作废。

## 3. 交付物 1：证据 JSON（你唯一允许写盘的文件）
路径：`docs/code-audit-2026-10-02/units/<你的单元ID>.json`（相对 C:/AI/wubigrok；UTF-8 无 BOM；严格 JSON，禁注释、禁尾逗号）。

字段名不可改（下面用单引号示意，实际 JSON 用双引号）：

```
{
  'id':'<ID>',
  'module':'<板块名>',
  'footprint':['...'],
  'verdict':'2~4 句中文总评：屎山程度、最痛的是什么',
  'score':0,
  'metrics':{'files_reviewed':0,'loc_reviewed':0,'test_files':0,'test_loc':0},
  'hotspots':[{'path':'','loc':0,'why':'一句话为什么危险'}],
  'findings':[{
    'fid':'<ID>-01',
    'severity':'P1',
    'category':'god-file',
    'title':'不超过30字，可当待办用',
    'file':'',
    'line':'12-58',
    'evidence':'不超过3行逐字源码',
    'why':'结构原因',
    'impact':'可验证后果',
    'fix':'最小可执行修法',
    'effort':'M',
    'prior_audit':''
  }],
  'open_questions':['']
}
```

- score：0~10（10=干净，0=屎山），按实际证据给，别谄媚。
- severity：P0=已实际咬人（有 CHANGELOG/坑记录、数据或崩溃风险）或阻塞发版；P1=上帝文件/平行实现/跨层耦合这类让每次改动成本倍增的结构债；P2=局部重复/可读性/一致性；P3=洁癖项。
- category 枚举：god-file|god-func|duplication|dead-code|error-swallow|coupling|concurrency|consistency|complexity|legacy-residue|test-smell|doc-drift|perf-smell|safety
- findings 12~20 条，按 severity 排序；报不出证据的疑点写进 open_questions。
- hotspots：本板块最大/最烂 8 个文件（path/loc/why）。

## 4. 交付物 2：结构化回报（工具返回值，走 schema）
score / verdict_short（一句话）/ findings_total 与 P0P1P2P3 计数 / 最多 5 条 top（severity,title,file,line,effort）/ hotspots 前 3 / json_path。

## 5. 纪律
- **只读代码**：除写自己的 units/<ID>.json，禁创建/修改/删除任何仓库文件；禁跑 go test / npm / vitest / 构建（主代理统一跑，避免并发自伤与编译竞态）。
- 不进入 node_modules/、clones/、backups/、dist/、build/、releases/、whisper_data/；`.gaea/AGENTS.md` 只读（查历史坑与约定），不要修改。
- 时间预算小于等于 20 分钟：优先大文件，不要试图逐行读完足迹内所有文件。
- 全程中文；禁形容词堆砌；每条发现都要能被别人「打开 file 第 line 行」复核。
