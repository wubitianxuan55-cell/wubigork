# gaea Day Log · 2026-11-19 · 办公板块工具 R2 分档诚实恢复契约（众神殿 pan_1795058214142 裁决施工）

> 众神殿 pan_1795058214142（「优化完善办公板块的工具，特别是本地工具」，4 席 3 轮，
> 主席 Codex，用户签注 accepted）裁决定案：办公本地工具的主要矛盾已从数量不足转为
> **COM 直接写入后的恢复边界与可验证性**；唯一 P1=R2 分档诚实恢复契约（影响高、
> 成本低）；R1 持久快照另立需求（须先定义未保存修改语义）；打开中 xlsx 的占用 A/B
> 后置观察。本批施工 action-1（R2 契约）+ action-4（文档回写）。

## 1. 裁决事实基础（各席工具核验）

- 现行 office_edit 描述只承诺 Word 一档「一次 Ctrl+Z 可整体撤销」（Excel/PPT 无说明）；
  office_save 描述只说证据卡「可复核」——**是边界遗漏，不是虚假承诺，R2 是补齐不是删除**。
- `OfficeHelper.cs` 的 undo 命令（:925 非 word 直接 BAD_ARGS）与 UndoRecord
  （:9741 StartCustomRecord/:9801 EndCustomRecord 仅 word 分支）——Excel/PPT 无整批撤销。
- office_save Journal 卡 BaselinePath 恒空+摘要自陈「无基线快照」（诚实缺省已实现）。
- Excel/PPT 应用自身撤销粒度未核验——只能声明助手不提供整批撤销，**不得写成逐操作可撤**
  （grok-r3 推翻 dsh-r3「逐操作」措辞：用新的过度声明替换旧的省略）。

## 2. R2 契约落地（action-1）

- **office_edit 描述分档**（`internal/app/office_com_tools.go`）：「Word 的一批修改…」
  单句扩为分档句——Word=一次 Ctrl+Z 整批+仅限当前文档与当前会话+不能恢复编辑前版本；
  Excel/PPT=助手不提供整批撤销+应用自身撤销能力未核验不要向用户保证可撤销。
- **office_save 描述**：末句「落盘后自动登记证据卡（可复核）。」→「（仅可复核，无基线
  快照——保存后没有跨会话回滚）。」
- **回执侧单源边界行**（`internal/office/comhelper/format.go`）：新导出
  `UndoBoundary(kind AppKind) string`（Word/Excel·PPT 两档文案）；`FormatEdit`
  签名加 kind 参数，成功与失败两条支路末尾都追加边界行（grok-r3 定稿：成功/失败回执
  都必须带分档失效边界）；唯一调用方 office_com_tools.go:842 同步。
- **save 回执恒带恢复边界**（office_com_tools.go text 组装处）：「恢复边界：证据卡仅可
  复核、无基线快照，保存后没有跨会话回滚——改错只能靠应用内撤销（仅 Word 一次 Ctrl+Z
  整批、限当前会话）或重做。」带不带 Final check 段的保存回执都携带。
- **静态契约测试**（裁决验收=成功/失败/保存三类测试均证明分档边界）：
  - `internal/office/comhelper/format_edit_boundary_test.go`：FormatEdit 成功/失败×
    word/excel/ppt 象限（含 prechecked 码 BAD_ARGS/NOT_FOUND 失败腿）边界断言+禁词
    「可回滚/可恢复」+Excel·PPT 不得出现「逐操作撤销」。
  - `internal/app/office_recovery_contract_test.go`：①描述钉住（两描述含分档钉句、
    禁词「可回滚/可恢复」）；②save 回执恢复边界四钉句+Journal 诚实缺省上锁
    （BaselinePath==""+摘要含「无基线快照」）；③源级守卫 TestOfficeHelperUndoGuardPinned
    （repoRoot 先例=wire_shape_guard_test.go）：读 OfficeHelper.cs 钉 undo 守卫+
    StartCustomRecord/EndCustomRecord 三行原文——未来给 Excel/PPT 加整批撤销时红=
    提醒同批更新 R2 文案（dsh-r3 断言④）。
- **明确不动**：office_live_prompt.go（A/B 出现可复现失败才改提示词=裁决
  disagreements #2）；OfficeHelper.cs 行为（R2 纯文案+测试面）。

## 3. registrymap 谓词+灵犀文档回写（action-4）

- `internal/gaea/tool/registrymap.go` xlsx_apply 谓词补「落盘前整文件基线快照+证据卡」
  （对齐 docx_apply/docx_accept/pptx_apply 同族句式，防基线实现被误判缺失）；
  镜像文档 `docs/gaea-tool-registry-map-2026-11.md` 同步。
- `docs/gaea-lingxi-optimize-plan-2026-11.md` 文首回写（grok-r2 裁定）：P1 四项
  2026-11-13 收官（复用系统 Python）+P1-4 dock 同日晚撤项（ec3f4f48）+batch2 五项
  同日收官+权威=todos/batch2、勿重开；L163 阻塞点行标注已解除；L130-133 P1-4 残留
  步骤标「仅存档勿执行」。
- 顺手修 HEAD 既有假红：9e366084（子代理线）给 registrymap 加了 send_message 行但
  镜像文档漏导出——TestRegistryMapDocCoversMap 红，补文档行即绿。

## 4. 验证与装机

- go test：comhelper 9.4s 绿+tool 1.9s 绿+tool/builtin 19.6s 绿+app 全包 119.9s 绿；
  gofmt（本批文件）清；CGO_ENABLED=0 全树编译过；golangci 本批文件 0 issues
  （capability.go 等 9 处 gofmt+gaea_verify.go rollbackRecord unused=各线既有非本批）。
- 构建：frontend npm build 29s→wails build 59.9s（ucrt64/CGO_ENABLED=1）→
  smoke.ps1 /api/health 200。
- 装机：gaea_install.ps1 INSTALL OK（exe sha256 9B3761998EBB…），旧壳已杀净
  （CIM probe：no gaea.exe running）。

## 5. 余项（裁决后置，勿擅自提前）

- ~~**action-2 打开中 xlsx 观察 A/B**~~ **同日已跑，见 §6**（工具层 9/9 至；模型两步
  内动作一腿留用户实弹）。
- **action-3 R1 保存前快照**：另立需求，用户先定义「未保存修改」语义（有改动/无改动/
  大文件三例）再评估实现。
- Excel/PPT 应用自身撤销粒度真机演练（临时文件单批修改+逐次 Ctrl+Z 比对，约 1 小时/应用）。

## 6. action-2 占用 A/B 实测（工具层）

载体=新 env 门控 live 测试 `internal/app/xlsx_occupy_ab_test.go`（`GAEA_XLSX_OCCUPY_AB=1`，
同 PANTHEON_LIVE_SMOKE/GAEA_AB_EVAL 纪律；夹具全用可丢弃临时目录，COM 实例由专用
powershell 进程 marker 握手持有、超时只杀自己的树不碰用户 Office）。臂=closed/excel
（Excel.Application）/wps（Ket.Application）各 3 次，走 `officeXlsxApplyTool` 全链。

**结果（9/9 干净）**：

| 臂 | 结果 | 回执/错误原文 |
|---|---|---|
| closed ×3 | success | `xlsx 已应用 1 个操作：…（写入值 E2=AB合计；证据卡已落账，可复核/回滚。）` |
| excel ×3 | **clean-fail** | `保存 xlsx 失败: open <path>: The process cannot access the file because it is being used by another process.` |
| wps ×3 | **clean-fail** | 同上（WPS ET 锁同形） |

判定谓词：**P-a 占用归因=true**（错误文案自带 used by another process）；**P-b 指路=false**
（不提 office_open/关闭后重试）；**半写金标未命中**（全部 sha 未动+excelize 可重开，
不升级 P1 补丁）；基线快照在失败臂也有落（ReadFile 先于锁写）。

**结论（按裁决定策树）**：占用=干净失败且归因明确，无产品损失——**不改提示词**
（裁决 disagreements #2 维持：无「模型高频误走且不自愈」证据不动 office_live_prompt）。
P-c（模型两步内动作）留用户下次实弹观察：若模型见此错误后不自愈，再议一行占用
回退句。证据档=`.tmp/xlsx-occupy-ab-1795067110/`（不入库）。

**顺带发现（新刀挂账）**：`internal/gaea/agent/execute_one.go:424` 注释称
「docx_apply/docx_accept/pptx_apply/**xlsx_apply** 走 ooxml.WriteAtomic（rename 收尾）
→ none」——实查 xlsxedit.ApplyOps 用 excelize `SaveAs`＝原地 `O_WRONLY|O_TRUNC`
直写（docx/pptx 才走 WriteAtomic）。本次占用至在 open 阶段被锁挡住=无半写，但
**中途写失败（盘满/崩溃）理论上可截断文件**（基线快照可救）。~~处置拍板挂用户~~
**→ 用户拍板 A 案，同日落地，见 §7。**

## 7. A 案落地：xlsxedit 原子写（用户拍板）

- `internal/office/xlsxedit/xlsxedit.go` 新增 `saveAtomic`（同目录临时文件
  `.gaea-xlsxedit-*.xlsx` + `fileutil.RenameWithRetry` 原子替换，与
  ooxml.WriteAtomic 同语义）接入 ApplyOps 落盘——中途写失败不再截断原文件，
  失败即目标未动。
- **归因不丢**：rename 被挡（Access is denied/sharing violation/used by
  another process）时错误补占用归因「目标正被其他程序占用，如 Excel/WPS
  打开中——关闭该文件后重试」——首轮实跑发现原子化后 P-a 从 true 退化成
  false（只剩含糊 Access is denied），此句修回；占用 A/B 复验 excel/wps
  各 ×3 clean-fail+P-a=true+零半写+原文件未动。
- 回归测试 `save_atomic_test.go`：成功腿（目标更新+临时零残留）+Windows
  只读目标失败腿（原文件 sha 逐字节未动+零残留；linux skip——rename 可替换
  只读目标是 POSIX 语义）。
- `execute_one.go` 原子性注释按实况改写（xlsx_apply 走 saveAtomic 非
  WriteAtomic，同为 rename 收尾 → EffectNone 语义不变）。
- 观察记档：`Recalc`（recalc.py/LibreOffice）写回是独立管线，不经 saveAtomic，
  中途失败暴露面仍在（其失败只降级为回执备注不拦交付，风险低）——不在 A 案面。

## 8. 收官包（用户令 git 打包）

`releases/gaea-office-r2-recovery-20261119-316860cf/`——打包提交 `316860cf`
（A 案），含本日三笔 `7cbf20f4`/`4958fbd0`/`316860cf`。干净工作树独立构建
（`.tmp/gaea-office-r2-316860cf` detached；node_modules 走主树 junction 规避
无 lockfile 全新 install 的 esbuild 坑；用后 cmd rmdir 摘链+worktree remove
回收净）。产物=payload（exe+三 DLL，DLL 与装机位同源同哈希）+payload ZIP+
源码 ZIP+SHA256SUMS+PACKAGE-INFO（`releases/…/PACKAGE-INFO.md`）。

- payload exe `84ffb2ca7f70b3c8`（与当日装机位 `eb5c57ec` 同提交内容异构
  哈希=构建时间戳；各自冒烟 200——payload 正式路径跑 `smoke.ps1` 过）。
- ZIP `b1be43db39d29f02`；源码 zip `e1ab3ad513c5ee00`。
- 测试底账与余项全录 PACKAGE-INFO；三笔本地未 push。

## 9. 文学板块 P0：回写警告回执 + 事件账本 + 漏斗（众神殿 pan_1795084662473）

用户令「按论证实施」。裁决（主持 Codex，签注 accepted）：P0=本轮唯一施工项
——chapter_write 与 candidate_promote 两条正文入口（施工前只读复核
`wenxue_tools.go:560/:731` 成立）共用警告型回执 + 最小事件账本 + 共同漏斗，
硬拒/长章分页/章节上下文包/通用写板级围栏均为条件项不进本刀。final_write
边界只读复核：只写定稿目录（`:1555` 闸），不进正文分母；quality_check 全书
scope 含 info 组（M4 只称频次）。

施工四件：

- `internal/app/wenxue_funnel.go`（新）：`wenxueWritebackReceipt` 共享回执
  （前一章 characters/timeline/unobserved 三态 + 仅 stale/dirty 的 hint，首章
  省略整块）+ 事件账本（`.gaea/stats/wenxue-funnel.jsonl`，一行一事件仅元数
  据；开关=书内 `.gaea/stats/enable` 标记文件，**默认关闭**、仅本地、随书可
  删——办公记忆写入确认红线的授权形态）+ `wenxueFunnelCompute` 指标纯函数
  （M1 每书每周首写章数/M3 下一章入口回写待办率/M4 质检频次/M5 提升使用率
  零候选 n/a/M6 旁路检出率零判定 n/a；M2 30 天续写率留 P1）。
- `wenxue_tools.go` 五处：chapter_write 删无条件静态 hint、回执补 revision
  +writeback 块+记账（kind=create/rewrite 防虚增）；promote 同构接入；candidate
  _write/quality_check/final_write 三点记账。
- unobserved 语义（ZCode r3 规格）：写第 N 章时前一章当前正文 sha256 vs 账本
  最后一条 body_write.revision——无账本 unknown/一致 clean/不一致 dirty
  （通用 write_file/edit_file 旁路低价检出，不改共享写面；旁路仅标 dirty
  不当成零）。
- RegistryMap 镜像谓词两行同步（两入口共用回执面）。

验证：5 项夹具（`TestWenxueBodyWriteEvents|TestWenxueWritebackReceipt|
TestWenxuePromoteReceiptParity|TestWenxueFunnelMetrics|TestWenxueSurfaceProbe`）
绿——含裁决验收表四场景（首章无块/漏回写 stale+hint 指前一章/补回写 updated
无 hint/缺失目录 none）+ 端到端旁路（直写第1章→写第2章=dirty）+ 事件行无正文
红线断言；wenxue 全族 9.9s 绿；app 全包 172s 绿；全树编译+vet+gofmt 清
（bindings_*.go 生成件不 gofmt 先例）。构建/装机未做：主树有另一并行线在途
未提交批次（BootSplash/Pantheon/OfficeB 等约 30 文件），按打包不混入纪律
（AGENTS.md §7）与双会话合流纪律不抢构建，挂并行线收官后补。

P1 挂用户授权：真实长跑（≥20 有效写章+30 天观察）+ 账本开启拍板（现在默认
关闭；开启=在书内 `.gaea/stats/` 建 enable 标记文件）。
