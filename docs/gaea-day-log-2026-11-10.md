# 当日总账（2026-11-10）：办公板块工具治理第一批（P0+P1a+P1b spike/接线，众神殿裁决落地）

> 承接众神殿 pan_1794234593182「gaea工具优化」（签注 accepted，六阶段计划
> P0→P1a→P1b→P2→P3→P4，D3 止损门）。用户令「开始优化迭代办公板块工具」=
> 开工令。本批按裁决推荐顺序执行 **P0 注册表地图 + P1a 结构化写入口 + P1b
> spike 并行首验**，并顺带完成 P1b 生产接线（spike 有利）。P2–P4 未动（D3
> 四条件全过后才启动）。

## P1b spike：五类写入拦截实证（action-2，最先补）

- 机制=P1b 发布边界本体 `internal/gaea/tool/publishguard.go`（package tool，
  纯 stdlib）：Begin 快照（size+mtime，durable 档=docx/xlsx/pptx，D1 口径）→
  Scan 比对 → Quarantine 移出正式工作区 + fail-loud 回显期望路径集。
- 五类实弹（Windows cmd.exe/解释器真子进程，`publishguard_test.go` spike 腿）：
  直接重定向 / Python 生成 / 改名 / 后台任务 / 脚本文件——**全部可拦可提交
  判定**；快照开销 begin=1.58ms / scan=1.05ms（200 文件）。
- 关键设计前提实证：**调用返回瞬间前台 Scan 恒 clean，后台拦截点必须在 job
  完成回调**（jobs.Manager 后台 bash 同构复刻）。
- **结论=spike 有利：P1b 按 4–6 人日基线推进，P4 不后置**（对照裁决上修场景
  24–31 人日，未触发）。
- spike 实弹坑两条：`cmd /c` 对 `python -c` 内联引号会被吃掉（exit 0 但没写
  文件=python 进 REPL 静默退出）；`cmd /c start /b` 嵌套引号起出交互式 cmd
  永挂管道（30s 假死）。脚本类负载一律落盘为脚本文件或直调解释器。

## P1a 结构化写入口（action-3）

- **`GaeaExportDeliverable` 补 `output_path`**（gaea_export.go）：声明路径分支
  （工作区内校验 + 扩展名与 format 一致闸）优先；空=历史 ExportsDir+时间戳
  默认分支不动（UI 契约不变，让位方式=显式声明优先）。验收=声明路径与实际
  落点逐字节一致（测试钉死）。
- **新 agent 工具族**（internal/app/office_tools.go，装配进 `gaeaExtraTools`
  装配单点——原 handler 内联列表提取为函数，不新开装配源）：
  - `office_generate`：从零生成唯一声明入口，包装 GaeaExportDeliverable（模板
    链 create_docx.py 不重写），实现 `WritePathResolver`（DAG write_paths 声明
    面可用）、不标 `PersistWrite`（FilterRegistry 不摘，接口断言+测试钉死）；
    Journal 证据卡落账（新文件无基线=通道 B 记 n/a 诚实缺省）。
  - 四类写升 agent 工具：`docx_apply` / `docx_accept`（仅 accept=true，拒绝
    路径留 UI）/ `pptx_apply` / `xlsx_apply`——薄包装 App 绑定方法，Journal
    落账单点仍在 appendOfficeEvidence（不建第四本账）。
- **通道 A「未比对回 pass」文案修复**（gaea_verify.go）：非 editLike 工具
  （docx_apply/pptx_apply 等）未比对摘要时改回「目标存在且可读（本工具暂未
  做引用级比对，摘要未核对）」；editLike 三工具语义不变。引用级比对属 P2。
- **三本账名单对齐**：Go `isWriterTool` +四类写、`isProducerTool`
  +office_generate、`ExtractDeliverablePaths` +output_path 键；前端
  officeTurnProjection `OFFICE_WRITE_TOOLS` +office_generate/四类写、
  `extractOfficeWritePaths` +output_path 键（L9 台账销项）。
- agent 包幽灵名守卫（v4.520）扩面：app ExtraTools 装配名显式登记（agent 包
  不能 import app，按守卫纪律登记真实注册名）。

## P1b 发布边界生产接线（action-4，spike 有利顺带完成）

- bash.go：`newPublishGuard()`/`publishCheck()`——**前台**启动前 Begin、终段
  （JSON+plain）Scan，命中即 Quarantine+fail-loud（返回 error）；**后台 job**
  启动即 Begin、`cmd.Wait` 返回（=job 完成回调）Scan，命中隔离+job 终态失败。
  早期返回（长运行进程）路径 Scan 已落盘越界写并以 `publish_blocked` 字段/
  附注上屏（进程存活不置错；其后写盘=记档残余窗）。
- 快照剪枝：`.git`/`node_modules`/`__pycache__`/`.venv`/`venv`/`target`（依赖
  目录非交付落点，如实记档的范围裁剪）；`~$` Office 锁件=ephemeral 豁免；
  隔离目录自查剪枝。
- 接线覆盖：桌面办公 agent bash 全部经 `Workspace.Tools`（workDir 已设）→
  全覆盖；`ConfineBash` 路径仅在 dir=="" legacy 分支（非办公面）。零值 bash
  实例（测试/非工作区装配）不守卫。
- durable 越界写被隔离到 `<工作区>/.gaea/publish-quarantine/<时间戳>/<相对
  路径>`，fail-loud 文案回显越界清单与结构化入口指引。

## P0 注册表地图（action-1）

- **`internal/gaea/tool/registrymap.go`**：唯一注册表全部 reg.Add 行的契约
  清单 82 行（49 builtin〔20 workspace+29 领域〕+16 Build 内部+16 app
  ExtraTools+1 mcp__ 动态备注行），每行=来源/空间/副作用/证据档/验收谓词；
  `/mcp__` prompt 旁路仅备注（不进注册表，非第五工具面——Grok-R4 结论继承）。
- **三处防漂移守卫**（裁决验收：新增工具缺契约即守卫失败）：
  `tool/registrymap_test.go`（Builtins ⊆ map + 行格式 + 文档覆盖）、
  `boot/registrymap_census_test.go`（mock provider 全量 boot.Build 运行时
  普查 ⊆ map）、`app/registrymap_extra_test.go`（gaeaExtraTools ⊆ map）。
- 导出器 `scripts/registry_map_doc.go`（//go:build ignore）单源生成
  `docs/gaea-tool-registry-map-2026-11.md`（勿手改）。

## 测试与门禁

- Go 新增：publishguard 单元 6 例+spike 5 类+开销定量；bash 接线 3 例（前台
  拦截/后台 job 失败终态/非 durable 放行）；app office 工具族 7 例（逐字节
  一致/三拒绝/WritePaths+形状/缺文件 fail-loud/通道 A 文案两态）；P0 守卫
  5 例。全绿：tool+builtin 全包、agent 全包（含守卫扩面）、boot、evidence、
  **app 全量 142.9s 零 FAIL**、全树编译（CGO_ENABLED=0）。
- 前端：officeTurnProjection 43/43（+P1a 白名单/路径提取 2 例）、`tsc -b` 零错。
- wails build + `/api/health` 冒烟：见装机段（构建进行中时本行以 tasklist
  输出为准）。

## D3 止损门四条件核对（全过 → P2–P4 具备启动前提）

1. **P0 清单无未知工具且守卫绿** ✅（82 行三守卫+文档同源）。
2. **四类写可被 agent 调用且全进 Journal** ✅（gaeaExtraTools 装配+落账单点
   appendOfficeEvidence+测试钉）。
3. **发布边界拦住 durable 写**（重定向/脚本/改名/后台四路）✅（spike+接线
   实弹；v1 durable=Office 三格式，非 Office durable=D1 范围外风险）。
4. **非 Office 工具无回归** ✅（app 全量+agent/boot/evidence/tool/builtin 绿）。

## 遗留与拍板池（裁决映射）

- P2 质量矩阵（九格+docx_accept+PDF 派生校验接 abeval；通道 A docx/pptx 引用
  级比对深化）——D3 已过，待用户点名。
- P3 非文档四类验收谓词、P4 持久任务中心（缓冲）——待点名。
- 在线 MCP 名单（凭证边界不读）、A1 会话锚点、读侧缓存统一——原台账继承。
- 早期返回（长运行进程）存续期内的写盘=记档残余窗（其每次前台窗口被下轮
  Begin/Scan 覆盖一部分；彻底闭合归 P2/P4）。
