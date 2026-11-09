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
