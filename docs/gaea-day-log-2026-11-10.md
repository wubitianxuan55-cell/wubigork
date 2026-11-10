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

## P2 质量矩阵第一波（同日续刀，「继续」=按裁决顺序推进）

- **通道 A 引用级深化（action-5 前半）**：docxedit 新公共原语
  `PendingChangeCount(path, author)`（修订跨度计数，按作者过滤）+
  `DocumentContainsText(path, needle)`（段落拼接口径：w:ins 计入、w:delText
  不计）；gaea_verify.go 新四 case——docx_apply（替换文本在文档+本作者修订
  在册）、docx_accept（无本引擎待处理修订）、pptx_apply（LocateText 读回
  目标页：替换在、目标不在，多处出现残留如实注记不判失败）、office_generate
  （OOXML 容器头引用级+非空）；声明载荷走 opsJSON（docx_apply
  selected/replacement 全文、pptx_apply slide/target/replacement 全文，超限
  不落宁漏勿误同 xlsx 口径），旧卡回退摘要解析、双缺=诚实跳过不冒充核对。
- **abeval 评测面扩（zcode-r3 五缺口销项）**：abEvalRegistry 进
  office_generate/docx_apply/docx_accept/pptx_apply/xlsx_apply（gaeaExtraTools
  装配面，WritePathResolver+AccessPolicy(ws) 同界）+doc_render；
  abFixture.Kind 扩 docx/docx_revs/pptx（纯 Go zip hermetic 造件
  abBuildDocx/abBuildDocxRevs/abBuildPptx，CI 可复算不依赖 python/soffice）；
  abSample.Negative=已知缺陷件标注（漏项为预期，汇总表 ⚠ 单独标记不计正例
  分母）；质量矩阵种子 3 格先行（docx×局部修改/docx_accept/pptx×局部修改）
  +负例 1（clauses-negative-missing-liability）；abRunArm 臂工作区写进 App
  单例（office 工具 gaeaCwd 与 AccessPolicy 同根）；abWriterTools/abArgTarget
  认 office 写面+output_path 键（返工信号口径对齐）。
- **P0 地图谓词续勘**：docx_apply/docx_accept/pptx_apply/office_generate 四行
  验收谓词升级为引用级口径，文档重新导出（守卫绿）。
- **P2 第二波（未动，待点名）**：doc_render 覆盖率字段/交付卡
  （visual_pages_reviewed/skipped_reason）、条件视觉闸门、九格语料真实脱敏件
  采集（需用户授权）。

## P2 质量矩阵第二波（同日三刀：条件视觉首渲+覆盖率透明）

- **Verdict 覆盖率字段**（evidence/journal.go）：visualPagesReviewed /
  visualPagesTotal / visualSkippedReason（omitempty，旧 verdict 读取兼容）——
  备忘 :48「视觉覆盖 3/12」透明口径的代码面落账。
- **条件视觉首渲**（gaea_verify.go）：无基线（首次生成）交付物不再是无视觉
  复核——runFirstRenderReview 复用通道 B 渲染 seam（soffice→PDF→pdftoppm
  逐页 PNG），firstReviewMaxPages=8 封顶抽样（备忘 :42 形状继承；首渲场景
  无编辑定位信息，v1=前 N 页，跳过原因透明）；渲染降级=reviewed 0+原因落账
  （ChannelB 保持 n/a，不冒充审过）；有基线 diff 路径=全页 reviewed 口径统一；
  汇总新增 first-review 态（verified+「逐页产物供人工目检」注记，不冒充
  视觉比对判定）。md 等不可渲染格式=零覆盖字段诚实缺省。
- **前端交付卡**（EvidenceSection）：「首渲覆盖：N/M 页 · 跳过原因」行（与
  diff「视觉复核」行互斥形态）+ 查看产物入口；渲染降级单独行透明上屏；
  VerdictView 类型+三语 i18n（visualCoverage）；mock ev_1002=first-review
  样本（diff 样本 ev_1003/旧形态 ev_1001 不动）。
- **测试**：Go 5 例（抽样 8/12、整页 5/5、soffice 降级、pdftoppm 失败、md
  不可渲染——seam 注入无 soffice/poppler 依赖）+前端 2 例（覆盖率行/双行
  互斥）；deliverables 族 51/51、tsc 0。
- **P2 第二波余项（未动，待点名）**：九格真实脱敏件采集（需用户授权/提供）、
  回修≤1 轮的缺陷闭环记账、触及∪当前∪可疑页定位（需编辑定位元数据，v2）。

## P3 非文档四类验收谓词（同日四刀，action-6 全清）

- **谓词类表**（registrymap.go 单源）：PredicateClasses 四类——读取检索
  （grep/read_file/knowledge_search/memory_search/cost_search）/领域计算
  （schedule_analyze/cost_indicators/cost_compose）/交付写（office 五工具+
  schedule_apply/write_file）/任务编排（dag_plan/task/todo_write）；生成器出
  文档新节，tool 守卫扩断言（类 id+类名进文档）。
- **harness**（builtin/p3_predicates_test.go）：每类金标例+缺陷例——
  读取检索=知识/记忆/造价三库沉淀→检索命中（库实况一致）+参数闸 fail-closed；
  领域计算=进度计划 analyze 关键路径复算（duration 10+critical 链）+环依赖
  fail-closed；交付写=write_file 声明落点逐字节一致+越界拒绝零落盘（办公
  四类写引用级在 app 族已覆盖）；任务编排=todo 编排计数+.gaea/todos.md 落盘
  可恢复+非法状态正确失败。守卫双钉：类缺 harness 红、代表工具缺 RegistryMap
  行红（P0/P3 对齐）。
- **语义勘误**（实弹修正，如实记档）：知识/记忆/造价检索为模糊匹配语义，
  小语料下「无关查询零命中」不成立——诚实缺省改钉参数闸 fail-closed，不冒充
  精确率（README 口径同步）。坑：记忆 Save 的 Name 校验不接受纯 CJK 名
  （ASCII slug+中文描述/正文）；检索命中形态=名称+描述（不含 Body）。
- 至此六阶段：P0/P1a/P1b/P2 两波/**P3 全清**；余=P2 九格真实脱敏件（需
  用户授权）+P4 持久任务（缓冲，待点名）。

## P4 持久任务中心·办公批处理（同日五刀，action-7 全清——六阶段收官）

- **普查改判**：持久任务调度器（internal/gaea/tasks，Hephaestus.db+断窗续跑
  Startup 重排+两段式取消+Retry=re-enqueue+空间分账+事件推送）与前端任务中心
  （GaeaTaskList/Cancel/Retry）早已全备——P4 真缺口=办公批处理没接进来。
- **KindOfficeBatch**（批量文档转 PDF）：handler 逐文件转换+进度逐文件上报
  （p.Report）+ctx 可取消+单文件失败不中断批次（Result 逐文件 ok/fail 报告，
  全败才整任务 failed 诚实口径）；声明纪律与 P1a 同口径：out_dir 必填
  （WritePathResolver 声明写路径=产物目录）；office_batch agent 工具提交即
  返回任务 id（不占回合、不依赖单回合存活）；单批上限 200 防误提交全工作区；
  优先级 15（用户显式提交＞后台索引 10，＜价格抓取 20）。
- **坑**：任务结果须走 `p.Result()`（Progress API 落库）——直接赋 `t.Result`
  不持久化（首个成功用例 Result 空实锤后改）；转换 seam 带 ctx（取消观测），
  生产包装忽略 ctx（取消由 handler 逐文件间隙判定）。
- **测试**：成功（3 文件逐文件报告+落点全在声明目录内）/全败诚实 failed
  （含重试同轨：换 seam→Retry→succeeded）/取消两段式 cancelled/提交端四拒绝
  （空 files/缺 out_dir/超批/越界 out_dir）。tasks 包回归绿。
- **六阶段收官**：P0/P1a/P1b/P2 两波/P3/P4 全落。余池=P2 九格真实脱敏件
  （需用户授权）+装机实弹+push（四批本地提交）。

## 装机（2026-11-10 01:48，五批产物上正机）

- gaea_install.ps1 -Launch：gaea.exe（61,735,424B，SHA256 前缀 F9BA8DBEDA41…，
  与 build/bin 同哈希）+三语音 DLL 装 C:\AI\gaea-app；桌面快捷方式刷新；
  新 PID 51408 启动 01:48:25 > 构建 01:47:06（INSTALL OK/PID AFTER BUILD OK）。
- 健康核验口径澄清（如实记档）：/api/health 是 opt-in 调试桥接（main.go:43
  设 GAEA_HTTP_PORT 才起，smoke.ps1 即用此法验的同字节 exe=200）；装机的
  桌面壳不开 HTTP 桥——活体证据=诊断口 127.0.0.1:18123 LISTENING（新 PID）
  + 壳未崩退。本机 8080 为 herdsman 常驻，与壳无涉。
- 实弹清单（交用户，五面）：①办公 agent 让它 bash 写一个 .docx 交付 → 应被
  发布边界拦截并回显 office_generate 指引；②让它 office_generate 生成声明
  路径文档 → 交付面板看「首渲覆盖 N/M 页」行+证据卡；③对生成卡点复核 →
  通道 A 应显示引用级结论（OOXML 容器头）；④多选几个文件让它 office_batch
  批量转 PDF → 任务中心看进度/取消/重试；⑤docx_apply 改一处文字后复核 →
  引用级「替换文本在+修订在册」。

## P2 质量矩阵实弹（同日六刀：办公种子 A/B 对真实引擎全绿）

- **配置**：GAEA_AB_EVAL=1+PROVIDER=gaea（wubigrok 桥路由办公板块绑定引擎）+
  样本 JSON 四办公种子（.tmp 不入库）+OUT=.tmp/abeval-office-20261110；
  go test -c 出二进制包目录直跑（CGO 配方），总墙钟 ~50 分钟，全程 PASS。
- **结果**（8/8 臂全出，正例清单全命中）：

| 样本 | A 墙钟/审批 | B 墙钟/审批 | 清单 A/B | 证据卡 A/B |
|---|---|---|---|---|
| docx-term-edit | 36s/8 | 600s⁺/118 | 3/3 · 3/3 | 2 · 2 |
| docx-accept-revs | 36s/8 | 502s/74 | 2/2 · 2/2 | 2 · 2 |
| pptx-budget-edit | 266s/58 | 461s/83 | 2/2 · 2/2 | 1 · 1 |
| clauses-negative ⚠ | 32s/9 | 77s/17 | 5/5 · 5/5 | 1 · 1 |

  ⁺B 臂撞全局信号量槽位超时（bridge: 等待 API 槽位 context deadline exceeded，
  11-03 众神殿已知病）但超时前产物已落、清单仍 3/3。
- **发现四条**：①办公三编辑种子（docx_apply/docx_accept/pptx_apply）被真实
  agent 经两臂调用全链通过——九格矩阵「局部修改」三格 live 达成（裁决 P2
  验收「可跑」从基建半边补成全边）；②**A 臂碾压 B 臂**（确定性小编辑任务：
  墙钟 3-15×、token 4-13×、审批 2-15×）——11-08 裁决「探索型 readonly 子代理
  对确定性任务是弱臂」实锤加深，readonly 常开门槛更远；③pptx 两臂各 1 返工
  信号=回修 1 轮内（交付纪律达标）；④**负例 substring 局限实锤**：模型诚实
  标注「违约责任：原文未载明」时清单子串照样命中→5/5——负例语义漏项须
  Verifier 谓词承载（裁决预判印证，与 P3 挂账同源）。
- 实弹记录（record.json 逐臂轨迹+用量）留 .tmp/abeval-office-20261110 供
  复核；评测基础设施经此轮实弹验证（hermetic 造件/臂注册表/负例标注/报告
  链全走通）。

## 沙箱机制实弹（同日七刀：走查壳 GaeaCallTool 绑定面驱动，3.5/5 面全绿）

- **口径**：GAEA_WALKTHROUGH=1 沙箱壳（workspace 强制 %TEMP%gaea-walkthrough，
  与用户实盘壳并行）+ GAEA_WEBVIEW_DEBUG=1 → WebView2 CDP；9333 双栈坑
  （kidsabc 占 IPv4/本壳落 [::1]:9333——cdp-walk 需 IPv6 变体 .tmp/cdp-walk6.mjs）；
  agent LLM 腿被「bridge: 登录已过期」全阻（走查实例无法刷新 token——LLM 纪律
  面只能用户验），机制面改走 wails 绑定（window.go.app.OfficeB，与按钮同路径）。
- **面2 ✓**：GaeaExportDeliverable+output_path 声明路径 → 合同/验收合同.docx
  （37,587B，python 技能链在走查态也解析到全局技能库）；xlsx 台账/验收同验。
- **面3 ✓（最深形态）**：xlsx_apply 通道 A「公式重算零错误；引用级复核 1 项
  声明与实况一致」；docx_apply 通道 A「替换文本在文档中且 2 处待处理修订在册
  （引用级）」——裁决谓词逐字命中；**通道 B 真视觉 diff 活体**（soffice+poppler
  真渲染，抓到 1→2 页版式变化→诚实 fail 整卡）+覆盖率字段 live 回填
  （reviewed=2/total=2）。
- **面5 ✓**：docx_apply 修订制写入（预览 56KB）+Journal 卡 pending_verify。
- **面4 半程 ✓**：GaeaTaskList('work')=37 任务台账（调度器活体）；office_batch
  提交腿需 agent 登录（用户验）。
- **面1 ✗（如实）**：PublishGuard 拦截实例是会话注册表-scoped（零值全局 bash
  无守卫）——agent 会话腿被登录阻，真壳验证只能用户做；机制本身已由
  publishguard 测试族+接线测试钉死。
- **坑**：GaeaJournalList 收 int limit；wails 命名空间 window.go.app（非
  go.main.App）；CDP Runtime.evaluate 同上下文声明残留——eval 一律 IIFE。
- 清场：走查壳 9340 已杀、%TEMP%gaea-walkthrough 已删；用户壳 51408 未动。

## dsh-ops 蒸馏候选池第一波（同日八刀：A3 revision CAS+B1 凭据围栏+A1' grep 续页游标）

- **普查改判（诚实记档）**：A1 结果预算的大头**已在库**——v4.379 spill 泄洪库
  本就是从 dsh spill-policy 蒸馏的（≥24KB→库+read_spill 分页+会话级+内存 FIFO
  预算+豁免纪律），磁盘 ref:// 版的唯一收益（重启幸存）被当时自评否决（无主
  痕迹文件卫生面）——本轮不重复立项。A1 波内真增量=grep 续页游标（命中截断
  后模型无法续读）。
- **A3 revision CAS**：read_file 文本读尾缀 `[revision sha256:<hex>]`（CAS 值
  来源）；写族四件（write_file/edit_file/multi_edit/edit_lines）增可选
  `expected_revision`——""=零行为变化、"absent"=create-only、64hex=内容哈希
  比对；不匹配→`REVISION_CONFLICT`（带当前修订值，重读即自愈）。Full+Compact
  双 schema 同步（compact 才是模型看的）。防多执行体同树 stale-Edit；与
  BaselinePath 互补（回滚原料 vs 前置冲突闸）；TOCTOU 窗口如实记档。
- **B1 凭据围栏**：credential_fence.go 目录段+名字双轨（.ssh/.aws/.kube/
  .gnupg/.docker/.config/gcloud/.azure 段；.env 家族/.netrc/.git-credentials/
  id_rsa 族/​.pem/.key/.pfx/.p12/.ppk/.keystore 后缀），大小写不敏感+反斜杠
  归一；read_file 命中→`CREDENTIAL_DENIED`（错误即值+自愈指引）；grep 遍历
  跳过+头部记账「credential-fence skipped N」；ls 不围（文件名非内容）、bash
  不围（审批/PublishGuard 域，如实记档）；普通办公文件永不命中（保守清单）。
- **A1' grep 续页游标**：`offset` 参数跳过前 N 条命中（**单遍 skip+emit——
  skip 在文件中间耗尽时同文件剩余命中接着发射，页边界不错切**，首版两段式
  丢文件尾命中当场抓住改掉）；头部行 `[grep page] offset=N shown=M — more
  matches may remain; re-run with offset=…`（头部行截断生存性同 spillNotice
  结论；total 在预算帽提前止步时是「至少」语义，不给伪造总数）。
- **测试**：revision_cas_test.go 6 例（CAS 全腿+尾标闭环+围栏规则表+read 拒
  绝+游标/围栏混合）全绿；tool 包全绿（一次负载期偶发 websearch 假红，三连
  复跑绿挂账观察）。

## 空终答 ×3 翻车根修（同日插刀，用户实弹报障 3acea975 已推）

- **取证**：用户 workspace 会话 gaea-log——三次 `empty final answer blocked` 各距
  1s、三连 2 秒内打光；同请求 12 秒后自愈成功；prefixBytes=65KB 远离 256K 窗口
  （排除规模问题）。**定性=上游瞬时空响应窗口内的无退避连发**——重试是对同一
  坏时刻的浪费。
- **根修**：第 N 次重试前等 N×emptyFinalRetryBackoff（缺省 2s；ctx 感知取消即退；
  Options.EmptyFinalRetryBackoff 测试注入）；mockProvider 增 streamFn per-call
  序列支持（瞬态恢复测试）。真零内容冻结重放语义/reasoning-only 豁免/上限 3 不变。
- **测试**：TestEmptyFinalRetryBackoffAllowsRecovery（两次零→第三次恢复，计退避
  发生）+既有两测全绿；agent 17.9s+app 全量 133.2s 零 FAIL。
- **取证配方入册**：用户 workspace（config.toml workspace 键）下
  `.gaea/sessions/work/*.gaea-log.jsonl`——request_header 的 prefixBytes/window
  排除规模、时间戳间距定性瞬态。

## 桌面端构建装机（2026-11-10 09:37，含空终答退避修复上机）

- npm 生产构建 23.9s+wails build 过；exe 61,749,760B，SHA256 前缀
  `4D280DE90C09D124`（build/bin 与 C:\AI\gaea-app 同哈希）。
- 冒烟 /api/health 200；gaea_install.ps1 -Launch：三 DLL 同装、桌面快捷方式
  刷新、新 PID 30776 启动 09:37:36 > 构建完成 09:37:23（PID AFTER BUILD OK）。
- 本壳含今日全部源码：办公治理六阶段（P0–P4）+dsh-ops 候选池第一波（revision
  CAS/凭据围栏/grep 游标）+空终答退避根修。用户可在原工作区重试此前失败的
  任务验证退避生效。

## B2 失败即值落地（同日，dsh-ops 候选池第二波，用户点名）

- **刀型**：工具错误从「Go error→统一 exec_error 首行文本」升级为结构化值
  `{code,retryable,effect_state,next_action}`，核心=副作用工具的诚实失败语义
  （effect_state=unknown 时 next_action 明示先核实再重试）。四刀分提交：
- **刀1（tool 包）**：`tool.Failure` 实现 error（文案原样透传，REVISION_CONFLICT/
  CREDENTIAL_DENIED 前缀=兼容层）；信封纯加法扩三字段+新码 conflict+
  MarshalEnvelope 导出；bash 超时/取消/非零退出（effect 恒 unknown+退出码结构化，
  取消不可重试）、CAS 三冲突=conflict+none（原子拒绝=写未发生）、围栏=denied
  分类接线；DefaultRetryable/DefaultNextAction 按码默认表。
- **刀2（executeOne 收口）**：AsFailure 命中按字段渲染（conflict/timeout 不再
  退化 exec_error）；未结构化错误按副作用账本默认分类（IsDeliverableTool+bash
  →unknown+先核实，只读→none）；**内层失败信封塌缩**——工具回整串失败信封
  （bash json 非零退出/browser validation/图表族）不再包 ok:true 外壳，仅整串
  解析+code 非空双条件命中（任意 JSON 可 Unmarshal 成零值信封的坑在测出后
  以 code 守卫钉死）。原计划 recoverable 接线经核实取消：error 通道恒 true
  是「模型下轮可修复」的既有正确语义，与 retryable 不同维，强接会把
  validation 类可修复错误误标不可恢复。
- **刀3（盲计根修，盘查意外收获）**：普通工具失败自 V8.9 起就是 WrapError
  信封 JSON，但四处消费方仍按裸 "error:" 前缀数失败——shouldMidTurnSteer
  全批失败纠偏从未因真实工具失败触发（steer_test 用旧格式假绿）、
  complete_step 失败照推 canonical todo、turnToolErrors 回合摘要漏失败、
  hasToolErrorPrefix（模型工具终态）同病。修=toolOutputClass 双轨判定
  （裸前缀兼容+整串失败信封；code=blocked 归被拦不归失败）+
  compactToolError 压一行入账。
- **刀4（纪律提示）**：系统提示原则节一行（effect_state=unknown 先核实现状
  再决定重试）+compact 描述 bash/四写各一短句；distill 报告 B2 销项。
- **测试**：envelope roundtrip+纯加法/Failure 透传与包装链/bash 真工具级退出
  码与取消/收口形状（写族 unknown·只读 none·Failure 不退化·塌缩不误伤）/
  steer 信封形态（旧前缀用例全保留防回退）；tool+agent+config 包全绿。
- **不抄**：phase 字段（gaea 无相位概念）；blocked/suppressed/panic 裸前缀
  路径 envelope 化（前缀消费方在册，留待点名）；office 族逐错误分类（默认
  分类已给诚实 effect_state）。

## dsh-ops 第二轮深审（同日晚，用户令重审，纯调研）

- 三路并行：lib/ 未挖子系统逐个读（发现 scoped-ownership 等 6 文件=2 行转发
  shim，真身在 publication//execution/）+native/crates 算法级深读+docs/118 md
  全目录扫描（新读 36 篇+快扫 12 篇，5 条过时段警示入册）。
- 产出候选池第二波 D1-D7（docs/gaea-dsh-ops-distill-2026-11-10.md §8）：
  D1 输出预算三件套（推荐首位）/D2 自校验续读游标（A1' 补强）/D3 错误码子层
  +applied（B2 补强）/D4 跨文件 replace+写锁/D5 三态跳过+完备性三元组/D6
  清理债务账本（publish-quarantine 只进不出待实测）/D7 作用域权威三件
  （双执行体同树第二刀）。观察池 8 项+同构不立项 8 项。
- gaea 侧对位断言核实：spill=内存库（budget+FIFO+重启清）无磁盘长草——
  spill 台账腿如实改判剔除；publish-quarantine 未见清理面命中（挂 D6 待实测）。

## dsh-ops 候选池第二波落地（同日晚，「开始」拍板，D1+D2+D3+D5+D6 六刀）

- **刀0 前置根修（两个生产 bug，盘查实证）**：①A1' 游标不可见——executeOne
  恒过 compressGrep，parseGrepLines 只认 path:line: 形状，控制行全被洗掉、
  零命中 grep 生产路径变空串（测试直调工具假绿）；②grep 扫进 .gaea/sessions
  （工作区根搜索第一页常被会话 JSONL 占满）。修=控制行摘出前置回放+
  wspath.IsGaeaHostNoiseRel 相对路径谓词+头行真前置（旧写在匹配行后与注释
  相悖）+toolcache 键补 limit（同 path+offset 撞不同 limit 页）。
- **刀1（D5）**：excluded（选择不看）vs gap（没看到：不可读/超长行 scanner
  截断——静默丢同文件余下命中是真 bug）分开记账；scanner.Err 检查；取消不再
  伪装完整 no matches；缺口不发续读指引。如实缩小：完备性二元不抄三元组、
  read 侧不做 FILE_CHANGED（单次 ReadFile 即快照）。
- **刀2（D2）**：grep 自校验续读游标 g1.<bind>.<snap>.<next>.<mac>（. 分隔防
  :digits: 误判）；双快照（prefix 边界校验/walk 全程发射）；树变=conflict
  重跑自愈、参数变=拒、缺口不发。如实缩小：进程级 HMAC 不加 SessionID。
- **刀3（D1）**：EscapeJSONLen（与 encoding/json 逐例相等钉死）+
  TruncateToEscapedBudget 迭代收敛+executeOne payload 序列化前预算截断——
  信封恒完整合法 JSON（旧口径序列化后截断必劈坏 JSON 根修）；B2 三字段挪
  Data 前；分页读压缩保头保尾标记。textutils 首建测试（此前零测试）。
- **刀4（D3）**：信封 subcode+EffectApplied（applied 真实落点仅 2 处：move_file
  删源失败/xlsx_apply 预览失败降级——不虚构）；原子写族默认 effect=none
  精确化；bash 守卫 validation+none+SANDBOX_UNAVAILABLE/START_FAILED 等子码；
  bash json 模式失败改走错误通道（B2 塌缩注释不实修正）。
- **刀5（D6）**：publish-quarantine 先记账后放句柄（ledger.jsonl pending→
  终态，账本失败不阻断隔离但 stamp 无主永不删）；fail-loud 带隔离目的地；
  boot.Build 异步清扫（账本背书+30 天+账本外名字整目录跳过+自底向上+失败
  留账重试）；存量无账本 stamp 不动；wssearch 噪声补齐。
- **坑（测试当场逮住三例）**：①retention<=0 兜底默认把负保留期（即刻过期
  测试语义）吞了——改仅零值兜底；②同文案 payload 被 Normalize 重复行折叠
  压小（预算测试要内容多变）；③grep 头行回显误用遍历中递减的 skip 游标
  （应回显本页起始 pageOffset）。
- **验证**：gaea 树 71 包全绿（CGO_ENABLED=1+ucrt64+DLL 进 PATH）+app 全量
  +gofmt 净+六刀分提交。**余池=D4 跨文件 replace+写锁、D7 作用域权威三件
  待点名**；观察池不动。B2/本波均未装机（要上桌需构建装机，说一声即办）。

## 桌面端构建装机（2026-11-10 13:08，含 B2 失败即值+第二波 D 系六刀上桌）

- 前端生产构建 21.3s + `wails build -s -ldflags "-s -w" -trimpath` 过
  （ucrt64 gcc 前置，系统默认 TMP/TEMP，无孤儿进程）；exe 61,808,640B，
  SHA256 前缀 `E87E08D6D99B0C57`（build/bin 与 C:\AI\gaea-app 同哈希）。
- 冒烟 `/api/health` 200（HTTP 桥接 18999，进程已回收）；gaea_install.ps1
  -Launch：四文件同装、桌面快捷方式刷新、新 PID 37472 启动 13:08:18 >
  构建完成 13:07:52（PID AFTER BUILD OK）。
- 本壳含今日全部源码：B2 失败即值四刀 + 第二波 D 系六刀（刀0 两生产 bug
  根修/D5 三态跳过/D2 自校验游标/D1 输出预算/D3 subcode+applied/D6 隔离
  账本+启动清扫）。可实弹面：grep 翻页（cursor token）/大输出截断（信封
  恒完整 JSON）/bash 失败信封 subcode/publish-quarantine 账本与 30 天清扫。

## dsh-ops D4 落地（同日晚，点名，replace+写锁三刀）

- **刀1 写锁原语**：O_EXCL 哨兵+进程 token+bind 重核（absent→出现/存在→删除/
  替换三态 conflict）+release token 防误删+stale 年龄回收（>10s 畸形 token）；
  LOCK_CONTENTION 可重试零效应。gaea 零 OS 锁先例下选 O_EXCL；
  gaea_task_inbox.go:95「文件锁属另一刀」销案。
- **刀2 锁接线**：edit 族+write_file「读→CAS→写」整段包 WithFileLock（CAS 防
  陈旧读+锁防跨进程 TOCTOU 互补）；write_file 原地写半写债收口
  （atomicWriteEncoded 化+effect 挪 none 组）；坑=锁创建先于 MkdirAll（新目录
  炸，acquireFileLock 先建父目录修，P3 谓词/E11 回归测试当场逮住）。
- **刀3 replace 工具**：两阶段（全计划 32MiB 预检零变更/逐文件原子非事务=
  已发布不回滚+PARTIAL_PUBLISH+effect unknown）；changed 计替换次数+预算帽
  文件内部分替换+pending；发布前锁内计划哈希重核（publishItem 抽方法直测
  竞态窗口）；凭据围栏写向跳过；四表全做（RegistryMap Spaces 填 work=实际
  口径，兄弟行 shared 漂移另开刀）+evidence/executeOne×4/batch 冲突键/
  coherence/contextview/前端 WRITE_TOOL_NAMES+WRITE_ONLY 名单全量+钉名测试。
- **如实缩小记档**：字面量 pattern/单串 include/年龄阈值回收/move_file 不挂锁/
  不带 caller revision。
- 验证：tool+builtin+agent+evidence+boot+前端 changes.test 全绿；全树+app
  全量见段末（后台跑完补记）。

## 桌面端构建装机（2026-11-10 14:33，D4 三刀上桌）

- 前端 21.7s + wails build 过（ucrt64，无孤儿；用户旧壳 37472 由装机脚本
  杀壳流程处理）；exe 61,847,040B，SHA256 前缀 `DE44AB5A400158A2`
  （build/bin 与 C:\AI\gaea-app 同哈希）；冒烟 /api/health 200；
  新 PID 37680 启动 14:33:39 > 构建 14:32:54（PID AFTER BUILD OK）。
- 本壳增量=D4 三刀：replace 跨文件批量替换工具+写锁 sidecar+write_file
  半写债收口。可实弹面：replace 批量替换（dry_run 预览/PARTIAL_PUBLISH 回执）、
  双开 gaea 同树写同一文件（LOCK_CONTENTION/REVISION_CONFLICT 竞争互斥）。
- 今日累计五批装机中第三次：晨间（空终答退避）→13:08（B2+D 系六刀）→
  本次（D4）。

## dsh-ops A2+D7 全清收官（同日晚，A2 两刀+D7 审计一刀两项不立项）

- **A2 动态工具目录两刀**：①机制（Options.ResidentTools/activated 会话集
  〔照 fileObs 挂法跨轮存活〕+currentSchemas 过滤+未激活软拒 TOOL_NOT_
  ACTIVATED 引导 discover+discover_tools 元工具〔词法匹配名字+3/描述+1/
  汉字二元组容措辞，每次≤5，下一模型步生效，可见性≠授权〕+in-history 播种
  〔历史调用过的工具幂等激活，重启/续会话不破〕）；②装配 `[agent]
  dynamic_tools` 三态——**auto 缺省（仅 MCP schema 超 32KB 时 mcp__* 转可
  发现：标准会话零变化——前缀缓存经济账：工具面 10-15K 稳定命中 vs 激活
  一次=整段前缀失效，默认全开长会话可能净亏；REA 式 60K 重载面才是无争议
  收益）**/on（激进两级 A/B 用）/off；discover 仅开启时注册（关闭零 schema
  增量）；子代理同面（父 CurrentSchemas 快照）；CaptureShape 指纹改取
  currentSchemas（旧口径探测盲区）。坑=sed 批量替换翻反三处布尔检查
  （测试当场逮住，教训：布尔条件替换禁用 sed）。
- **D7 对位审计（宁不立项不虚构）**：A 作用域仲裁=同构不立项（注册面静态
  单源、跨源同名结构性不可达、Add 即最后写者赢且从不并发）；C 变更归属
  传播=同构不立项（配置面全量重建、MCP 热插已天然 O(affected)）；B 超代
  对账主语义（新轮取消旧轮）在回合串行模型下不存在，但实锤真实窄缺口=
  **bridge 取消时干净关流被当正常收尾→已收到的工具调用仍被派发执行（含
  写盘，auto/yolo 完全放行）**——修=取消执行双闸（agent_stream 出口带
  判中断+executeBatch 入口带 NOT_EXECUTED 零效应信封，已完成预执行照常
  消费）。次要缺口记观察池（pendingSends 取消后仍发/Close 不取消在跑回合/
  move_file 不在锁族）。
- 验证：agent 全包+boot+tool+config 绿；全树+app 全量后台跑（见下段补记）。
  **dsh-ops 候选池至此全清收官**（A1'/A2/A3/B1/B2/D1-D7 全处置：
  落地或审计不立项，零悬置）。
