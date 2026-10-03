# 全仓审计第 27 批 · god-file 续拆（IN3-03 mpp.go 三分：入口容器/版本布局表/低层块解析）· 2026-10-03

> 接续 [round-28（批次二十六）](round-28-p1-batch26.md)。god-file 同配方续拆单线，**同包纯文件拆分**（原位搬移零逻辑改动），主代理直做。
> 本批**不抬版本**、不动 CHANGELOG；交付物 = 本文件 + `.gaea/progress.md` 一条。
> 快照：开工时 HEAD = `022f1405`（批次二十六）；工作树干净。

---

## 〇、预核与切口

- IN3-03 审计指认复核成立（行号漂移：924→1035 行，4 套版本字节布局同文件）。切口=文件自带五段分节注释：入口/版本布局表(AP4-11)/中间结构与模型装配/基础块解析/版本探测+CFB 容器/字节原语。
- 三文件方案：`mpp.go`（入口与容器：ParseMpp/版本探测/CFB 流装配/中间结构与模型装配，保留全文件域头注）+ `mpp_layout.go`（版本布局表：mppVersion/mppAsgMode/mppLayout/MPP9·12·14+2013+ 变体偏移表/mppLayoutFor，**零 import**）+ `mpp_rows.go`（低层：Props/VarMeta+Var2Data/FixedRows 行读取 + 字节原语八件——块读取与字节原语同属低层，合并一文件）。IN1-01 project.go（动核心 Manager）留给厚预核批次。

## 一、落地

- sed 行号字节精确抽取 + 手写文件头拼接；行数对账闭合（1000 内容行 + 35 旧头 = 1035）。
- import 按段重算：layout 零 import；entry=bytes/fmt/sort/strings/time+mscfb；rows=binary/fmt/math/sort/time/utf16。首版 build/vet 一次过。

## 二、等价证明（双证 + 一次真回归的教训）

- **符号清单**：非测试文件 before/after diff 空（首轮误把 `mpp*.go` glob 含测试文件混入对比——口径要对齐非测试源文件）。
- **go doc -all 逐字节一致**——期间抓到并根修一处真回归：拆分头注把**包级文档注释**错放到 `package` 之后（go doc 的包文档丢失）；复位到紧贴 package 子句上方后 doc 恢复一致。**教训入册：god-file 拆分时，原文件的包级 doc 注释必须逐字节跟随 `package` 子句留在其中一文件（通常留入口文件），且不得在 doc 块与 package 之间插入任何新行；拆分说明写进新文件头或普通位置，勿污染包文档。**
- 包测试：schedule 0.6s 绿（含 mpp_test/mpp_layout_test 真实样本门控与四版本合成用例，直接覆盖搬移代码）。
- 前台 ci **exit 0**：golangci 0 issues / go test 全绿 / vitest 435 文件 3762 例全绿 / 卫生守卫绿。
- 新文件两枚进显式 add 清单；提交后复验「树干净 + HEAD 可编译」。

## 三、god-file 余量与下一批

- 已销：GA2-01、IN2-12、GA5-08、GA5-04（批 25/26）+ IN3-03（本批）。god-file 余 ~16（App.tsx 1041/project.go 1011/controller.go 963/agent.go 975/CostLibraryView 1132/memory_hub.go 等）。
- 下一批候选：IN1-01 project.go 八类关注点（动核心 Manager，需更厚预核：消费方极广、字段耦合深，可能要「接口分域+文件分家」两步走）；前端组件族（FE1-05/FE2-02 CostLibraryView 1132 行等）走「组件拆分+状态上提」配方，与 Go 文件分家不同型。coupling ~20 与零散死码继续等拍板。
