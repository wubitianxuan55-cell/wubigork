# office 枢纽解耦审计与收刀（2026-09-10，file:line 口径）

> 收敛计划 W3-4「office 枢纽解耦」的读档半刀+收刀记录。输入口径为
> gaea-convergence-plan-2026-09.md §3：「先 trace whisper→office（96 调用）与
> modelengine→office（62 调用）逐条归类…此刀优先」。
> **结论先行：该两条边在 Go import 层面均为 0；「office 272 入/70 出枢纽」
> 的度量口径已过时。真正的内核耦合只剩 docmd 文档转换一族，本刀已随审计
> 收干净（docmd → internal/docmd）。**

## 一、审计修正（与计划前提相悖的证据）

| 计划前提 | 实测（2026-09-10） |
|---|---|
| whisper→office 96 调用 | **import 层 0 处**。internal/whisper 全目录（含 db 子包）无一文件 import office 或其子包；间接边（≤2 跳）亦为 0（其 7 个内部依赖均不触 office）。「96」疑似把 app 接线层 wx_file_handler.go 或 gaea 侧 control/refs.go 误计入 whisper |
| modelengine→office 62 调用 | **0 处**。internal/modelengine 全目录零 "office" 字样（连注释都没有）；其内部依赖仅 fileutil/strutil/netclient 三者，均不触 office |
| office = 事实枢纽（journal/evidence/会话记忆在 office） | **内核能力早已下沉 internal/core**（office/aliases.go 头注释自证：ExecResult/AgentJobState 已是 internal/core 的等价别名，仅为保生成绑定签名而留） |
| 需抽「中立包」承接文件工具+会话记忆两类 | **会话记忆类在内核侧为空集**；文件工具类只剩 docmd 一族 |

全仓 import internal/office 的文件共 20 个：14 个在 internal/app（Wails 绑定层
与功能 handler，属正常消费），6 个在 internal/gaea 内核侧——后者全部只走
`office/docmd` 子包。

## 二、内核侧耦合清单（解耦前，6 文件 9 处，100% 文件工具类）

| 位置 | 符号 | 用途 |
|---|---|---|
| gaea/control/refs.go:260 | docmd.ConvertLimit | @引用办公文档转 Markdown |
| gaea/fileindex/fileindex.go:96 | docmd.ConvertLimit | 文件索引正文抽取（30 页帽） |
| gaea/knowledgeimport/knowledgeimport.go:70 | docmd.ConvertLimit | 知识导入前提取文本 |
| gaea/largefile/summary.go:66,174 | docmd.ConvertLimit + DefaultMaxPDFPages | 大文件摘要 |
| gaea/tool/builtin/format_convert.go:59,73 | docmd.ConvertLimit + DefaultMaxPDFPages | format_convert 工具 |
| gaea/wssearch/wssearch.go:120 | docmd.ConvertLimit | 工作区全文搜索抽取 |

另有 app 接线层 wx_file_handler.go:254 `docmd.Convert`（轻语微信文件转文本，
唯一与 whisper 域功能相关的调用点，也在 app 包）。

归类分布：①文件工具 10/10 · ②会话记忆 0 · ③疑似误用 0。

## 三、收刀（v4.197.0，随审计落地）

`git mv internal/office/docmd internal/docmd`（包名不变，纯 import 路径更新，
20 行零逻辑改动）。docmd 自身仅依赖 gaea/proc，office 内部无消费者，无环。

收刀后：**internal/gaea 内核对 internal/office 的 import 边 = 0**。office 包
剩余消费面全部在 internal/app 绑定层（docx/xlsx/pptx 编辑、预览、crosslink、
standard 公文检查、OCR——即「office 编辑/检查/联动」本体的正常消费者），
不再具备「内核枢纽」性质。W3-4 此刀的抽取目标实质上由更早的 internal/core
下沉完成，本刀只是摘掉最后一根线。

## 四、遗留候选（观察池，不立案）

- `office/aliases.go`：等价别名门面。若未来 gen_bindings 可改指 internal/core
  类型，别名即可删（需动生成签名，另刀评估）。
- `office/archive.go` ExportMemoryArchive：包外无调用者（疑似遗留），先挂
  观察池，删除需拍板（藏≠删）。
- 依赖度度量口径：入度/出度统计应排除 `internal/app` 绑定层（其对各域门面
  的引用是架构使然，不是耦合），否则每次都会高估出「枢纽」。
