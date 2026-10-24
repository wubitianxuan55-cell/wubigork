# S2 · ZCode 工具层设计蒸馏（文档/文件写入类工具）

- 调研对象：`clones\zcode`（只读检出），CLI 侧 `apps/zcode-cli/packages/{contracts,core,adapters}` + `docs/`
- 目的：为 gaea「办公板块真实改写 docx/pptx/xlsx 且不丢格式」抽取可复用的**工具层工程质量**，并显式标注不适用项
- 纪律：结论均带 `file:line`；gaea 侧模块先 grep 确认存在与真实签名后再引用；本次**未改任何代码**
- 结论强度标注：`[实读]` = 逐行读过实现；`[文档]` = 只读到 spec/文档，未逐行读实现；`[推断]` = 由已读代码外推，需二次验证

---

## 0. gaea 侧基线（先确认「已存在」，避免重复建设）

| 模块 | 真实存在与签名 | 证据 |
| --- | --- | --- |
| 工具接口 | `Tool{Name/Description/Schema/Execute/ReadOnly}`；可选能力 `CompactDescriptor`/`PersistWriteTool`/`ModelBackedTool`/`SpaceTaggedTool`/`ContextualTool` | `internal/gaea/tool/tool.go:18-32,56-72,74-84,102-111` |
| 结果信封 | `ToolEnvelope{ok,success,code,error,message,data}` + 稳定 code 集（ok/timeout/denied/not_found/exec_error/validation_error/blocked…） | `internal/gaea/tool/envelope.go:18-37,40-75` |
| 写文件 | `write_file`：保留原权限位 + 原编码，写前捕获 before/baseline，写后 `RecordChange` | `internal/gaea/tool/builtin/writefile.go:56-101`（权限 `76-82`、编码 `83-89`、证据 `94-100`） |
| 改文件 | `edit_file`：exact `old_string` 唯一匹配 + CRLF 诊断 + 原子写；与 `multi_edit` 共用 `replaceInContent` | `internal/gaea/tool/builtin/editfile.go:120-142,144-153` |
| 批量改 | `multi_edit`：内存内顺序应用、全成功才落盘一次 | `internal/gaea/tool/builtin/multiedit.go:15-29,86-107` |
| 原子写 | `AtomicWrite`（temp→chmod→rename）/`AtomicWriteSync`（+fsync）/`RenameWithRetry`（Windows 权限抖动退避 3 次） | `internal/gaea/fileutil/atomic.go:20-30,32-73,78-93` |
| 编码 | `fileutil/encoding`：BOM 建模为 `Kind`（UTF8BOM/UTF16LE/BE/NoBOM），`Encode` 重新写回 BOM；GB18030 级联探测 | `internal/gaea/fileutil/encoding/encoding.go:16-41,45-70,122-179`（BOM 回写 `159-162`） |
| 泄洪 | 结果 ≥24KB 存会话级 `spill.Store`，内联附 locator；`read_spill` 按字节分页取回；`read_file`/`spill`/`task` 豁免 | `internal/gaea/agent/spill.go:20-40`、`internal/gaea/agent/execute_one.go:347-361`、`internal/gaea/tool/builtin/readspill.go:12-65` |
| 读状态/陈旧守卫 | `fileObs[path]` = `v1:<size>:<mtimeNano>` 指纹；写前 `checkFileStale` 拦「外部已改」；同轮「写过未读」也拦 | `internal/gaea/agent/file_observation.go:26-34,55-79`、`execute_one.go:112-132` |
| 并发分批 | 按冲突键 `file:<path>` / `!全局` 连续扫描分批，同键串行、异键并行（保序） | `internal/gaea/agent/batch_executor.go:173-207,209-249` |
| 沙箱收敛 | `ConfineWriters` 绑定 roots；`confine` 用 symlink-free 解析判定「在工作区内」 | `internal/gaea/tool/builtin/confine.go:21-37,56-72,74-88` |
| 配置保留式落盘 | `RenderTOMLPreserving`：键级合并 + 未知段整块保留（「渲染器没写的键也不丢」） | `internal/gaea/config/render_preserve.go:9-28,64-96,326-337` |
| 预览 | `GaeaPreview` 返回 `kind`(image/docx/xlsx/markdown/pdf/text) + `body/dataUrl/Pages`；xlsx 走 `xlsxpreview.Render` | `internal/app/gaea_preview.go:17-45`、`internal/office/xlsxpreview/xlsxpreview.go` |
| 复核/回滚 | Verifier 双通道（A 结构/引用、B 视觉 diff）+ 基线快照 + 冲突保护回滚；结论落 `verdicts.jsonl` | `internal/app/gaea_verify.go:1-60`、`internal/gaea/evidence/journal.go:35-60,104-149`、`docs/archive/gaea-v41-evidence-chain-design.md:63-89,138-148` |
| 办公编辑本体 | docx/web 修订制 `ApplyTrackedReplace`/`AcceptChanges`/`RejectChanges`；pptx 文本 run 级替换；xlsx Op 集 + LibreOffice 重算 | `internal/office/docxedit/docxedit.go:1-6,24-49,51-81`、`internal/office/pptxedit/pptxedit.go`、`internal/office/xlsxedit/xlsxedit.go:1-4,21-54,110,170-195` |
| OOXML 容器 | `ooxml.ReadZip/WriteAtomic`：**条目序保留 + Deflate + 临时件 rename**，字节级冻结 | `internal/office/ooxml/ooxml.go:1-9,24-60` |
| 知识/校验工具 | `genui_validate`（结构校验）、`format_convert`、`read_spill`、`wait`/`bash_output`/`kill_shell` | `internal/gaea/tool/builtin/genui_validate.go:22-63`、`bgjobs.go:18-65` |

**关键结构事实**：gaea 的办公编辑入口在 Wails 绑定层（`internal/app/gaea_xlsx_edit.go`、`gaea_docx_edit.go`、`gaea_pptx_edit.go`），**不在模型工具注册表里**——`internal/gaea` 下没有任何 `internal/office/*` 的 import（grep 无命中），内置工具表也没有 docx/pptx/xlsx 类工具（`internal/gaea/tool/builtin/*.go` 的 `Name()` 集合共 46 个，无 office 项）。这决定了 ZCode 工具层经验对 gaea 的落点：**要么先把办公改写提升为工具，要么把同样的契约用到绑定层**。

---

## 1. 文件工具族

### 1.1 读/写/改三件套的契约分层

**设计要点**
- 三份 schema 分工：provider 面 `inputSchema`（JSON Schema）、运行时 `runtimeInputSchema`（zod，真正 parse）、`outputSchema/runtimeOutputSchema`（输出也双向校验）。
- 工具元数据显式声明副作用面：`readOnly/destructive/concurrentSafe/sideEffectScope/riskLevel/needsApproval/timeoutMs/maxOutputBytes`，权限系统读声明而不是猜；`Write` 明确 `concurrentSafe:false`（同批不可并行）。
- 每个工具还声明 `timeout{defaultMs,maxMs,allowCallOverride}`、`cancellation{supported,cleanup,userVisibleMessage}`、`trace{required,propagateToAdapters}`——超时/取消/追踪是**工具契约的一部分**，不是调用点临时决定。
- 结果预算也写在工具上：`resultBudget{maxInlineBytes,maxModelBytes,strategy,preview,artifact}`。

**证据**
- `apps/zcode-cli/packages/core/src/tool/handlers/write.ts:211-266`（Web 工具入口全字段）、`edit.ts:254-309`、`read.ts:464-526`
- 输入/输出 schema：`packages/contracts/src/tools/write.ts:15-30,110-123`（`z.object(...).strict()` → `toToolJsonSchema`）
- `ApplyPatch` 同一套：`handlers/apply-patch.ts:325-381`

**对 gaea 的启发**
- gaea 的 `Tool` 只有 `Name/Description/Schema/Execute/ReadOnly`（`tool.go:18-32`），**缺** timeout/取消语义、输出 schema、结果预算、审批风险的声明位；这些现在散在 `execute_one.go` 与审批层。写类工具（尤其 office 的）应把这些补齐到接口上，否则「docx 改写超时 30s 还是 5min」「失败能否重试」无处声明。
- `Write` 把 `concurrentSafe:false` 写进契约这一点值得直接搬：gaea 已有更细的 `file:<path>` 冲突键，但**办公改写必须显式声明「同文件串行」**（现在靠 `isFileWriter` 名单推断，名单漏一个工具就并发出错，见 `execute_one.go:335`）。

**成本** S（接口加可选字段 + 一处装配）
**风险** 低；真正的成本在把「审批/超时/预算」从散点收敛到声明后，需要一次性校对全部 46 个内置工具。

---

### 1.2 read-file-state：read-before-write / stale / partial view

**设计要点（ZCode 最值得抄的一条）**
1. **状态键 = (规范化路径, offset, limit)**，Windows 路径大小写/分隔符归一；不是单纯 path 键。
2. **写前必须「读过」**：`Write` 覆盖已存在文件时，若无 read-state 或该 state 是 partial view → 直接失败（`write_file_not_read`）；`Edit` 同理（`FILE_NOT_READ`）。
3. **stale 判定三级**：revisionId 变化 → mtime 前进或 size 变化 → **严格全文读且内容逐字相等则放行**（关键：避免 linter/formatter 触碰 mtime 后误报）。
4. **partial view 语义**：`offset/limit` 是 range view，不算 partial；只有「被 token 上限截断」才置 `isPartialView=true`，并据此拒绝写。
5. **读缓存命中返回 `file_unchanged` 桩**（"Wasted call — file unchanged since your last Read"），压掉重复读的 token。
6. mtime 统一 `Math.floor` 到整数毫秒再比，规避亚毫秒精度导致的假 stale。
7. **跨工具一致性**：Bash 的 `cat/head/tail/sed -n/grep` 会回填 read-state（stdout 被截断则跳过）；Bash 写类命令（`--write/--fix/prettier/black/gofmt/…`）执行后扫描 read-state，mtime 前进了就在 Bash 结果里追加 `[This command modified N files you've previously read: …]` 提示。

**证据**
- 键构造：`core/src/tool/read-file-state.ts:18-29`；最新态选择与 mtime 归一 `:51-67`
- 写前守卫：`handlers/write.ts:278-304`（not-read / partial / stale）、`handlers/edit.ts:421-437`
- 内容相等放行：`write.ts:294-296`、`edit.ts:433-434`；mtime 取整 `write.ts:315-325`、`edit.ts:444-458`
- partial 语义：`handlers/read.ts:367-379`（注释明确「range view ≠ partial view」）
- 读缓存桩：`read.ts:186-198`、`:55-56`
- Bash 回填：`handlers/bash-read-file-state.ts:95-166`（截断跳过 `:97-99`）、stale 提示 `:59-93,182-190`
- 规约来源：`docs/superpowers/plans/2026-07-06-bash-read-file-state-parity.md:56-79`、`docs/superpowers/plans/2026-06-15-read-file-state-windows-path-key.md:13-30`、`docs/edit-tool-model-visible-result.md:11-18`

**对 gaea 的启发**
- gaea **已有**近似物：`fileObs` 指纹 + `checkFileStale` + 同轮「写过未读」拦截（`file_observation.go:26-34,55-79`、`execute_one.go:112-132`）。差异点正是可抄部分：
  1. **内容相等的逃生门**：gaea 只比 `size+mtimeNano`，Windows 上「AV 扫描/同步盘 touch」会造成写被拦；ZCode 的「严格全文读且内容相同则放行」值得加（代价：一次读全文，Office 文件除外）。
  2. **partial view 概念**：gaea 现在没有「模型只看到片段」的表达。Office 预览天然是片段（前 N 行/前 N 行单元格），**必须**有等价位，否则模型会拿着「预览片段」当全文去改写。
  3. **range view vs partial view 的区分**：这是 Office 场景的核心——`xlsxpreview.Render` 的 `sample` 只是抽样，模型改 `B4` 前并未看到 `B4`；工具的 stale 判定必须基于「声明操作的单元格/段落是否在已观察范围内」，而不是「整文件是否读过」。
  4. **写后刷新观察**：gaea 在自身写成功后会刷新（`file_observation.go:24`），这点比 ZCode 更省；保留。
- 不建议抄：ZCode 的 read-state 是**进程内 Map**，跨会话丢失；Office 场景需要跨进程（前端预览 ↔ 后端写）共享「用户看到了哪一版」，应落到 host 侧持久事实（gaea 的 `fileObs` 也只在内存，需一起考虑）。

**成本** M（接口 + 判定语义 + 测试矩阵）
**风险** 中：Office 全文读代价高（50MB xlsx），不能照搬「读全文才算已读」；必须先定义「投影版本号」（见 §8）。

---

### 1.3 原子写：temp→chmod→fsync→rename，以及它们的失败策略

**设计要点**
- ZCode：`atomic:true` 时写同目录临时文件（`O_WRONLY|O_CREAT|O_EXCL|O_NOFOLLOW`，随机后缀）→ 复制原文件 mode → `handle.sync()` → `rename`；**atomic 失败则 fallback 到 `O_TRUNC` 非原子写**（文档明确承认这会在异常磁盘错误下截断目标）。
- **拒绝写穿符号链接**：先 `lstat` 判 symlink 直接报 `Refusing to write through symlink`；临时文件用 `O_NOFOLLOW`；fallback 分支对 `ELOOP` 同样拒绝。
- gaea：`AtomicWrite` 同样 temp+`Chmod(perm)`+`rename`，且额外有 `RenameWithRetry`（Windows AV/索引器持锁导致 `Access is denied` 的重试退避）与 `AtomicWriteSync`（rename 前 fsync，断电级）。但**没有** symlink 拒绝，也**不做**非原子 fallback。

**证据**
- ZCode：`adapters/src/fs/index.ts:700-755`（symlink 拒绝 `704-710`、`O_NOFOLLOW` `720-723`、chmod 复制 `726-729`、sync `730`、rename `735`、fallback `736-754`）
- 覆盖矩阵自述风险：`docs/testing/file-tools-atomic-write-e2e-coverage-matrix.md:9`（「fallback 在异常磁盘错误下可能截断已有目标；本轮暂接受」）、`:11`（不覆盖磁盘满/rename 失败）
- gaea：`fileutil/atomic.go:32-73`、`78-93`

**对 gaea 的启发**
- gaea 的原子写**整体优于** ZCode（有 fsync 变体与 rename 重试，Office 文件正好落在「Windows + 云同步盘」重灾区），应保持并把它作为办公写盘的唯一入口（现在 `ooxml.WriteAtomic` 也复用它，`ooxml.go:53-56`）。
- 需补两点：
  1. **symlink 拒绝**：`confine` 只保证「解析后在工作区内」（`confine.go:60-63`），但「目标是 symlink 且指向工作区内另一个文件」会静默改写真实目标——对办公文件意味着「用户以为在改 A.docx，实际改了模板」。这是 ZCode 明确防住的场景。
  2. **绝不 fallback 成非原子写**：Office 文件被截断即「文件损坏 + 格式全丢」，比失败严重得多。ZCode 自己的矩阵把这条列为「暂接受」，gaea 不应接受。
- 另外把 `AtomicWriteSync` 定为 Office 唯一写路径（用户手工拖拽+崩溃窗口），代价是一次 fsync（几十 ms 级）。

**成本** S（symlink 检查 + 调用点收口）
**风险** 低；symlink 拒绝可能与「用户故意软链模板」的工作流冲突，需要明确错误文案与放开方式。

---

### 1.4 编码与 BOM

**设计要点**
- 编码是**元数据**而非猜测：`readTextFile` 返回 `encoding` + `lineEndings`（LF/CRLF 主导风格），内容统一归一为 LF 供工具层使用；写回时 `encoding: 原编码, lineEndings: 原行尾`，由 adapter 还原。
- 中文遗留编码是一等公民：`FileSystemTextEncoding = BufferEncoding | "gb2312"|"gbk"|"gb18030"`；探测级联 = BOM → `looksLikeBinary`（NUL/控制字节 >30%）→ 严格 UTF-8（`TextDecoder{fatal:true}`）→ GB 系列 round-trip。
- 写回时代码页**必须自证可逆**：`assertLegacyEncodingRoundTrip`——iconv 编码后再解码，不等就报 `unsupported`，绝不写出一份「看起来成功其实乱码」的文件。
- 行尾：`applyRequestedLineEndings` 在编码前把 LF 还原为 CRLF（只做一次 split/join）。
- **BOM 没有元数据位**：`detectTextEncoding` 把 BOM 前缀映射成普通 `"utf8"`/`"utf16le"`，内容里保留 `\uFEFF` 字符；写回靠 `Buffer.from(content, enc)` 把该字符再编成 BOM 字节。即 BOM 是「内容的一部分」，不是编码属性。

**证据**
- 类型：`contracts/src/interfaces/file-system.port.ts:58-59`；读结果含 `lineEndings` `:93-107`；写请求含 `encoding/lineEndings/atomic/expectedRevision` `:163-174`
- 实现：`adapters/src/fs/index.ts:155-203`（读：探测+归一）、`:291-295`（写：还原行尾+编码）
- 编码器：`adapters/src/fs/text-metadata.ts:30-49`（探测）、`:63-80`（编码 + round-trip 断言）、`:116-130`、`:213-240`
- BOM 当内容：`text-metadata.ts:10-11,31-36`；测试断言 `adapters/tests/fs-text-encoding.test.ts:156-160`（`expect(read.content).toBe("\uFEFF中文\n…")`）
- GB18030 抽样边界：同文件 `:143-160`（跨采样边界可读）

**对 gaea 的启发**
- gaea 的 `encoding.Kind` **显式建模 BOM**（`UTF8BOM`/`UTF16LE(BE)`/`NoBOM` 三态，`encoding.go:16-41`）并在 `Encode` 时按 Kind 写回 BOM（`:159-162`）——**gaea 在这点上领先，不要倒退**。ZCode 的 BOM 是「字符串里的 U+FEFF」，任何 `TrimSpace`/前导清理都会静默去掉它；gaea 明确 Kind 更安全。
- 值得抄的是 **round-trip 自证**：gaea 的 `fileenc.Encode` 对 GB18030 只做编码（`encoding.go:159-179`），没有「编码后解码再比对」的断言。CSV/TSV（办公里被 Excel 打开的中间产物）一旦不可逆就是静默乱码，ZCode 的 `assertLegacyEncodingRoundTrip`（`text-metadata.ts:116-130`）是低成本高收益的护栏。
- **行尾元数据**同样值得抄：gaea 的 `replaceInContent` 靠「CRLF 诊断」救场（`editfile.go:122-128`），是事后补救；ZCode 是读时就记 `lineEndings`、写时还原（`index.ts:293-295`）。对 docx/pptx 无意义（zip 二进制），但对同目录的 `.csv/.md/.txt` 附件有意义。
- **Office 文件不能走这条链路**：ZCode 的文本读入遇到二进制直接报 `Unsupported or binary text encoding`（`text-metadata.ts:205-211` + `looksLikeBinary :179-189`）。也就是说 **ZCode 的 Read 根本读不了 .docx/.xlsx**，这正是它不需要关心格式保真的结构性原因。

**成本** S（round-trip 断言）、M（行尾元数据若要贯穿到 office 周边文本）
**风险** 低。注意：gaea 的 `writefile.go:83-89` 只在「旧文件存在且可读」时保留编码，新建文件恒 UTF-8——这是合理默认，但「另存为」场景（从 GB18030 源复制）需要显式传递，别依赖默认。

---

### 1.5 并发与幂等

**设计要点**
- 幂等靠**版本前条件**：`writeTextFile(expectedRevision)`，adapter 在写前 `stat` 比对 `revisionId (= mtime+size)`，不等则 `stale_write`。三层叠加：工具层 read-state 判定（内容级）+ adapter 层 expectedRevision（元数据级）+ 注释里明确的「mtime 只比整数毫秒」。
- 并发靠**声明 + 调度**：`concurrentSafe`（布尔）优先，其次 `readOnly`/`sideEffectScope === "none"`；`destructive` 一律串行；`maxConcurrency=10`；拓扑分层成 parallel groups，环依赖直接报错。
- 工具内部串行化：`Write/Edit/ApplyPatch` 都 `concurrentSafe:false`；`Read/Grep/Glob` 为 `concurrentSafe:true`。
- **`revisionId` 的软肋**：`v1` 指纹只有 `mtimeMs+size`（`index.ts:775-777`），同 ms 同 size 的改写会逃逸；read 结果里其实带了 `hash`（`:197`），但写前条件没用它。

**证据**
- 前条件：`adapters/src/fs/index.ts:298-300,473-483`；调用方 `handlers/write.ts:125-137`、`edit.ts:508-520`、`apply-patch.ts:90-102`
- 调度：`core/src/tool/scheduler.ts:60-103`（判定顺序 `97-102`）、`:233-245`（READ_ONLY_TOOLS 名单）、`:48`（并发上限）
- 工具声明：`write.ts:218`（`concurrentSafe:false`）、`read.ts:472`（`true`）、`grep.ts:147`（`true`）

**对 gaea 的启发**
- gaea 的分批比 ZCode **更细**（资源键 `file:<path>` 而非布尔，`batch_executor.go:173-207,209-249`）——**不要用 ZCode 的布尔模型替换它**。
- 真正该抄的是 **adapter 层前条件**：gaea 只有 agent 层 `fileObs` 指纹（`file_observation.go:26-34`），**写出瞬间**没有「我读到的版本还在不在」的原子检查——`checkFileStale` 与真正 `WriteFile` 之间存在 TOCTOU 窗口（用户此刻在 WPS 里保存一次就丢）。给 `fileutil.AtomicWrite` 加一个 `expectedVersion` 变体（rename 之前再 `stat` 一次），把窗口压到最小。
- 指纹建议升级为 **内容哈希**（Office 文件尤其需要：`rename` 会换 inode 但 mtime 精度在 Windows 上可能只有 100ns 却仍可能相同；且「另存为」会保留 mtime）。gaea 已经有 `fileutil` 与 `evidence` 的 hash 设施可复用。
- **幂等**：ZCode 的 `expectedRevision` 是「乐观锁」，不是「幂等」。gaea 需要的是「同一 (file, ops) 重复执行不产生叠加」——`xlsxedit` 的 Op 集是声明式（`xlsxedit.go:21-40`），天然可按 ops 哈希去重；这是 gaea 可以比 ZCode 做得更好的地方。

**成本** S（expectedVersion） / M（若引入内容哈希与 ops 幂等键）
**风险** 中：Windows 上 `stat` 与 `rename` 之间仍非原子（无跨平台 CAS），只能缩小窗口；文案必须告诉模型「重试要重新读」。

---

### 1.6 Edit 匹配语义（以及它为什么在 Office 上失效）

**设计要点**
- 唯一匹配：`old_string` 必须唯一，否则明确报「找到 N 处，请给更多上下文或 `replace_all:true`」；`old_string === new_string` 直接 `NO_CHANGE` 失败。
- 归一化：内容与 `old_string/new_string` 都先 `normalizeLineEndings`；空 `old_string` = 「创建文件/填空文件」语义（文件已存在且非空则拒绝）。
- 特殊文件类型拒绝：`.ipynb` 直接引导到 `NotebookEdit`。
- **字面量替换**：`content.replace(search, () => replacement)`——用函数形式避免 `$$`/`$&` 被当替换 token（注释明确）。
- 匹配策略层：`findEditMatch` 返回 `strategy/candidateCount/actualString`，并做 `preserveQuoteStyle`（保留原引号风格）、`normalizeReplacementForMatch`（按策略调整替换文本）；unicode escape（`\u003C`）能匹配真实字符。
- 编辑后**不改写 `filePath` 回显**：输出用调用方原始 `inputFilePath`（相对路径保持相对），但 read-state 用解析后的绝对路径。
- 大文件闸：>1GB 直接拒绝。
- 失败形态是 `ToolHandlerFailure{result:false, errorCode, message}`——**业务失败不是异常**，带稳定数字码。

**证据**
- `handlers/edit.ts:52-58`（provider 描述含「strip the Read line prefix」）、`:104-115`、`:124-153`、`:164-166`、`:193-201`、`:203-234`、`:625-635`（`$&` 转义）、`:637-642`、`:311-313`（failure 形态）、`:470-485`（输出用 input 路径）
- 匹配器：`core/src/tool/edit-matchers.ts`（`findEditMatch`/`normalizeReplacementForMatch`/`preserveQuoteStyle`，见 `edit.ts:32-37` import）
- 输出契约：`docs/edit-tool-model-visible-result.md:11-18`

**对 gaea 的启发**
- gaea 的 `replaceInContent`（`editfile.go:120-142`）已有唯一性 + CRLF 诊断 + `replace_all`，**但用 `strings.Replace` 而不是函数式替换**——Go 的 `strings.Replace` 没有 `$&` 问题（不是正则），这点 gaea 反而安全，无需照搬。
- 可抄两点：① `old_string === new_string` 的 `NO_CHANGE` 早失败（省一次全文件写）；② **输出回显用调用方原始路径**（gaea 现在 `fmt.Sprintf("edited %s…", p.Path)` 回显的是 `resolveIn` 后的路径，`editfile.go:107`，相对路径会变成绝对路径，模型下一轮容易用错）。
- **Office 不能有 exact-string 编辑工具**：docx 的文本被拆在多个 `w:r/w:t` 里，Word 的拼写检查/rsid 会让「同一句话」跨 run；pptx 同理（`a:r/a:t`）。gaea 已经用 run 分组手术解决（`docxedit.go:465-471` 指向 `ooxml/runs.go` 共享核心）。结论：**ZCode 的 Edit 契约只能是 gaea 的「纯文本附件」通道，不能作为 Office 的编辑原语**（详见 §8）。

**成本** S（两条小改）
**风险** 低。

---

### 1.7 ApplyPatch：先全量校验、再统一落盘

**设计要点**
- 两阶段：`parseStructuredPatch` → 对每个 section 先 `verify*`（算出 `oldContent/newContent/encoding/lineEndings/revision`，不落盘）→ 全部通过后才循环写。**任一 hunk 对不上就整体不写**，模型拿到 `HUNK_NOT_FOUND` 与上下文提示。
- 写入仍带 `expectedRevision`（update 才有）；`add` 会先 `stat` 确认不存在；`move` = 写新路径 + 删旧路径。
- 输出是「每文件 structuredPatch + additions/deletions + `A/M/D path` 摘要」，供 UI 与模型共用。
- provider 描述里明确禁止「转去写 python/shell 替换脚本」——这是一条**行为约束写进工具描述**的例子。

**证据**
- `handlers/apply-patch.ts:67-110`（两阶段）、`120-216`（per-section verify）、`22-30`（modelInstructions）、`49-66`
- 解析器：`core/src/tool/patch-parser.ts`、`patch-apply.ts`（`derivePatchedContent`）

**对 gaea 的启发**
- gaea 的 `multi_edit` 已是「全成功才写」（`multiedit.go:86-107`），方向一致。缺口在 **Office**：`xlsxedit` 有 `PlanOps`（干跑算 diff，`xlsxedit.go:170-195`）与 `ApplyOps`（`:110`）两个函数，但**调用方是绑定层**（`gaea_xlsx_edit.go:88,141`），模型没有「先 plan 再 apply」的工具面。这正是应该把 ZCode 的 ApplyPatch 形态搬过去的地方：`office_apply(patch/ops)` 先返回 plan（预览 + 影响面 + 冲突），确认后再 apply。
- 另一点：`docs/superpowers` 里的 patch 语义文档化程度很高，gaea 若做 Office patch，应同样把「hunk 对不上时给什么提示」写成契约（`apply-patch.ts:181-203` 的错误信息就是范本）。

**成本** M（Office 版 plan/apply 工具面）
**风险** 中：跨文件事务仍非原子（ZCode 也没有），Office 多文件（docx+附件）需要显式「一次一个文件」或引入 journal 补偿。

---

### 1.8 multi-edit：**ZCode 没有，gaea 有**（不要反向抄）

**事实**
- ZCode 工具集没有 multi-edit：`core/src/tool/handlers/` 下无 multi-edit 实现，provider-visible 名单里也没有（`provider-visible-order.ts:1-33` 全表无 MultiEdit）。批量改靠「多次 Edit / ApplyPatch」。
- gaea 有 `multi_edit` 且语义更强（all-or-nothing + 共用匹配内核 + 逐条证据上报，`multiedit.go:15-29,86-121`）。

**结论**：这一项 gaea 领先，ZCode 无可抄；反过来，若 gaea 要把 Office 批量操作收敛，应复刻自己的 `multi_edit` 形态（op 数组 + 全成功才落盘），而不是 ZCode 的「多次单点编辑」。

---

## 2. 工具 schema 与输出协议

### 2.1 工具定义面（ToolEntry）

**设计要点**
- 一个工具条目 = 能力声明（`capability`）+ 元数据 + handler + 三层 schema + 权限声明（`patternSources`/`denyPriority`/`alwaysAllow`）+ 结果预算 + 超时 + 取消 + trace + 一组可选钩子：
  - `resolveModelContract(ctx)`：按当前模型能力投影**不同的 description/inputSchema**（如 PDF 支持与否）。
  - `validateInput`：在 JSON Schema 通过之后、hook/权限之前做的**语义预检**（例：`TaskOutput` 的空 task_id、Read 的 `pages` 范围）。
  - `resolveInput`：把模型入参归一成「将要执行的事实」，之后 hook、权限、审批、handler 看到同一份字节（注释里称之为「确认与执行同字节」）。
  - `applyInputAdjustments`：用户在确认窗里改参数后回灌，并要求「批准 A 不会跑 B」。
  - `prepareApproval`：权限已决定 ask 之后的最后一句话，只能收窄成 proceed 或补 preview，不能把 allow 变 ask；且必须同步、无 I/O。
- 工具可见性：`shouldDefer/alwaysLoad/providerVisible` + `tool-visibility.ts` / `tool-search`（延迟加载发现）。

**证据**
- `core/src/tool/types.ts:302-401`（ToolEntry 全字段，含 `:306-311` 的 host-issued `modelContentProtection`、「executor 不得按工具名推断」）
- `:323-368`（resolveModelContract/validateInput/resolveInput/applyInputAdjustments）、`:385-395`（prepareApproval 的约束）
- `read.ts:480-485`（resolveModelContract 实例）、`:277-291`（validateInput 实例）
- `provider-visible-order.ts:1-52`（稳定排序：参考工具按名排前、本地工具保持原序 → 前缀稳定，利于 prompt cache）

**对 gaea 的启发**
- gaea 的接口只有 5 个方法，**没有任何「按模型能力投影 schema」的位**。gaea 有本地模型（vision/多引擎）与远程 provider 混合，工具描述写死会浪费小模型上下文。加一个可选 `ModelContract(modelID) (desc, schema)` 是 S 成本。
- `resolveInput` 的「一次解析、全链同字节」对 gaea 直接有用：Office 工具入参里会有 `range:"B2:B10"`/`target:"B4"` 这类需要解析成坐标的字符串，**必须在审批前解析并回显**，否则用户批准的和实际执行的可能不同。
- `provider-visible-order` 的稳定排序思路与 gaea 的 `Schemas()` 按名排序等价（`tool.go:428`），无需改动；但 gaea 应把「工具顺序/描述变更会导致 cache 前缀失效」写进契约（现在只有 `cache_guard.go:9-16` 的事后检测）。

**成本** S/M（`ModelContract` + `ResolveInput` 两个可选接口）
**风险** 低。

---

### 2.2 错误返回形态

**设计要点**
- **三种失败面**，各有消费者：
  1. **schema 预检失败** → provider 面文本 `<tool_use_error>InputValidationError: …</tool_use_error>`，但 error 里保留结构化 issues；humanized 文案（missing/unexpected/type mismatch 三分支）。
  2. **handler 业务失败** → `ToolHandlerFailure{result:false, errorCode:number, message}`，由 executor 统一组装，**不打成异常**。
  3. **基础设施失败** → `createCoreError(type, message, {context:{code, filePath…}, recoverable})`，带 `recoverable` 决定是否重试。
- 模型可见文本与 UI/日志文本**分离**：`<tool_use_error>` 只进模型面，UI 用裸 message（`docs/edit-tool-model-visible-result.md:11-14`）。
- adapter 层错误是**带 code 的类型化错误**（`FileSystemPortError.code`：not_found/permission_denied/stale_write/too_large/invalid_path/unsupported/cancelled/io_error），工具层按 code 分支（如 Read 把 `not_found` 变成「附相似文件名建议」的可恢复错误）。
- 成功也讲「状态事实」：`Write/Edit` 成功文案带 `(file state is current in your context — no need to Read it back)`；`replace_all` 有专门措辞；用户改过参数时追一句 `The user modified your proposed content before accepting it.`

**证据**
- `core/src/tool/input-validation-model-content.ts:12-72`（三分支 humanize）
- `types.ts:241-246`（ToolHandlerFailure）、`handlers/edit.ts:311-313`、`contracts/src/interfaces/file-system.port.ts:7-47`（错误码 + `isFileSystemPortError`）
- `handlers/read.ts:226-249`（not_found/too_large 分支）、`:401-410`（相似名建议）
- 成功文案：`write.ts:42-64`、`edit.ts:65-82`
- 契约：`docs/edit-tool-model-visible-result.md:11-18`

**对 gaea 的启发**
- gaea 的 `ToolEnvelope`（`envelope.go:18-37`）已经有稳定 code，**结构上等价甚至更严**（`ok/success` 双写、`data` 结构化）。真正缺的是：
  1. **错误码分层的细则**：`edit_file`/`write_file` 现在把业务失败当 `error` 返回（`editfile.go:93`、`writefile.go:91`），靠 `executeOne` 统一包成 `WrapError`；模型拿到的 code 只有 `exec_error`，**无法区分「old_string 没找到」与「磁盘满」**。ZCode 的 `EditErrorCode`（`contracts/src/tools/edit.ts`，如 `OLD_STRING_NOT_FOUND`/`AMBIGUOUS_REPLACE`/`STALE_FILE`/`FILE_NOT_READ`）就是**给模型的可编程分支**，应照搬成 gaea 的 code 集。
  2. **成功文案的「状态事实」**：`write_file` 现在返回 `wrote N bytes`（`writefile.go:101`）——这对 Office 没意义（改 docx 的字节数不是信息）。应改成「改了哪些 part/单元格 + 预览版本号 + 是否需要复核」。
  3. **空结果占位**：ZCode 对空输出给 `(Tool completed with no output)`（`result-serialization.ts:58-68`），避免模型误判「工具没跑」。gaea 的 `WrapResult` 会给出 `{"data":{"tool":…,"result":""}}`，模型可能读成空——建议补一层。

**成本** S（code 集） / M（成功文案重构，因为要定义 Office 的「变更事实」）
**风险** 低；但改 code 集会破坏既有前端/日志解析，需同步。

---

### 2.3 大输出：三级预算、artifact 落盘、显式截断后缀

**设计要点**
- 每个工具声明 `resultBudget{maxInlineBytes, maxModelBytes, strategy: "truncate"|"artifact", preview{direction}, artifact{enabled,retention}}`；默认 100KB。
- 序列化决策顺序：空内容占位 → 参考投影（`projectToolReferenceContent`，把「已在上下文里的文件引用」折叠成引用）→ 若 strategy=artifact 且超限则**落盘**（按 `retention: session|project|temporary`）→ 否则**截断**，并在尾部追加机器可读元信息：`[Tool output truncated by resultBudget: artifactPath=…, originalBytes=…, maxModelBytes=…, strategy=…]`。
- artifact 写失败**不能**把成功调用变失败：捕获后回退到统一截断路径（注释明确）。
- 媒体内容单独计量：`returnedBytes` 计入真实 base64 栅格字节，否则用量观测少计（`:152-157`）。
- 截断保留方向可配（`head`/`tail`），`TaskOutput` 还专门规定「末尾 8MiB + `[NKB of earlier output omitted]`」（`docs/task-output-contract.md:112-124`）。

**证据**
- `core/src/tool/executor/result-serialization.ts:34-41,47-69,97-113,175-232`（截断后缀 `:214-221`）、`:323-356`（artifact 失败回退）、`:306-316`（媒体计量）
- `contracts/src/interfaces/tool-artifact-store.port.ts:8-53,106-141`
- 工具侧预算实例：`grep.ts:170-182`（20KB + artifact/session）、`read.ts:501-509`
- `docs/task-output-contract.md:112-124,152-154`

**对 gaea 的启发**
- gaea 的泄洪是**全局阈值 24KB + 内存 FIFO**（`agent/spill.go:20-40`），没有 per-tool 预算。对 Office 这会出问题：`xlsxpreview.Render` 的 JSON 可能几 MB，而 `read_spill` 每次只能取 16KB（`readspill.go:27`），模型要翻几十页才能拼出一张表——**这是最具体的「水土不服」之一**。
- 应抄：
  1. **per-tool 预算声明**（`maxModelBytes` 与 `maxInlineBytes` 分离），至少给 `grep`/`ls`/`read_file`/office preview 各配一份；
  2. **截断后缀带机器可读元信息**（gaea 现在 `truncateToolOutput` 只给一句人话，模型不知道「被截了多少、全量在哪」）；
  3. **结构化截断**：截断必须发生在**结构边界**（行/单元格/变更条目），而不是字节中间。gaea 已有正确示范（`xlsxedit.PlanOps` 的 `truncated` 标志 + `gaea_xlsx_edit.go:33`「变更清单截断到上限」），应把它变成通用设施。
- 落盘对比：ZCode 的 artifact 有 `retention` 与 URI/路径双标识；gaea 泄洪是会话内存、重启即失（`spill.go:16-17` 明确这是设计选择）。对「用户要的交付物」应落盘（gaea 走 evidence/journal+baseline），对「工具中间输出」内存即可——但**要给模型一个明确的「全量在哪」的短期引用**，否则模型无法判断该不该继续取。

**成本** M（per-tool 预算 + 结构化截断设施）
**风险** 中：per-tool 预算需要逐个工具标定，标错要么截太狠（模型反复取）要么白占上下文。

---

### 2.4 输入校验的双层 + 预检在链路上的位置

**设计要点**
- 顺序固定：初始 JSON Schema 校验 → 工具 `validateInput`（语义） → `resolveInput`（归一化事实） → `PreToolUse` hook → 权限 → `applyInputAdjustments` → handler。
- `validateInput` 的失败**不运行 hook/权限/PostToolUseFailure**，且 provider 面是 `<tool_use_error>`；这是为了「别为一次格式错误打断用户确认」。
- 自带一个**最小 JSON Schema 校验器**（不依赖 ajv），只覆盖工具契约用到的子集（type/oneOf/required/additionalProperties/数值边界），产出统一的 `ToolInputValidationIssue`——目的是让 provider schema 与 runtime 校验**同源**，避免「JSON Schema 接受、zod 拒绝」的裂缝。
- 工具可以按模型能力替换 schema（`resolveModelContract`），但 handler 始终用 runtime schema parse，保证执行面稳定。

**证据**
- `core/src/tool/json-schema.ts:33-51,53-70`（自研校验器 + oneOf 分支）
- `core/src/tool/input-normalization.ts`、`tool-input-validation-issues.ts:1-9`（issue 类型）
- `read.ts:253-291`（handler 内二次 parse + `validateInput` 只做语义约束）
- `docs/task-output-contract.md:56-73`（两层错误 + 「executor 不识别具体 tool name」）

**对 gaea 的启发**
- gaea 的 schema 是**手写 JSON 字符串**（`editfile.go:34-45`、`multiedit.go:31-44`），参数解析靠 `json.Unmarshal` 到匿名 struct——**没有 required/类型校验层**：`{"edits":"oops"}` 这类输入会走到 `Unmarshal` 报一个 Go 错误文本，模型拿到的是 `invalid args: json: cannot unmarshal…`。ZCode 的 humanized 校验文案（`input-validation-model-content.ts:49-63`）与「required 缺失/多余字段/类型不符」三分支是低成本高收益的补丁。
- 更重要的结构建议：**不要**为 gaea 引入第三方 JSON Schema 库（依赖与体积）；ZCode 的做法是「只实现子集 + 统一 issue 模型」，gaea 可以照此写 80 行校验器，并让 `compact_drift_test.go`（`tool.go:64-68` 提到的漂移测试）扩成「full schema 与 compact schema 字段集一致」的自动校验。
- `resolveInput` 对 gaea 的价值：Office 工具的 `range`/`target`/`sheet` 需要在审批前解析成真实坐标与工作表索引，否则「用户批准的是 A 表 B4，实际写到 C 表 B4」。

**成本** S（子集校验器 + issue 文案）
**风险** 低。

---

## 3. 搜索工具（ripgrep/ugrep 内置化、嵌入式搜索）

### 3.1 ZCode 的做法

**设计要点**
- 三层：① 随包原生产物（`bfs`/`ugrep`/`rg`）由固定 SHA-256 的 release plan 分发；② `runtimeToolResolver` 把绝对路径投影为 `ZCODE_BFS_BINARY`/`ZCODE_UGREP_BINARY`/`ZCODE_RG_BINARY`；③ Bash prelude 把 `find`/`grep` shadow 成 `bfs`/`ugrep`（`rg` 只补缺，不覆盖用户已有）。
- **Grep 工具不走 shell**：直接 `FileSystemPort.searchText` → adapter 用 WASI ripgrep **在 Worker 线程**里跑（注释：不能占住主线程，否则 session/stop 无法被处理），带 30s 超时 + abort 时 `terminate()`；`ripgrep` 运行失败自动降级到纯 JS 实现（`searchTextWithJavaScript`）。
- 结果协议：`output_mode: content|files_with_matches|count`；`head_limit` 默认 250（adapter 常量）；VCS 目录硬排除（`.git/.svn/.hg/.bzr/.jj/.sl`，ripgrep 参数与 JS 回退两条路都排除）；`glob`/`type` 过滤；结果输出相对路径、`path:line:text` 格式。
- 预算：Grep 的 `maxModelBytes=20_000` 且 `strategy:"artifact"`——超限落盘后模型拿引用。

**证据**
- 分发与开关：`docs/runtime-tools/embedded-search-native-build.md:7,11-19,153,224-229`、`docs/runtime-tools/ripgrep-runtime.md:5-11`
- Worker 隔离：`adapters/src/fs/index.ts:1029-1124`（注释 `1056-1058`）、超时/abort `1089-1098`
- 降级：`adapters/src/fs/index.ts:463-470`（`RipgrepRuntimeFailure` → JS）
- 默认值与排除：`adapters/src/fs/index.ts:51-54,943,1721,1824`
- 工具预算：`handlers/grep.ts:23,140-199,201-224`

### 3.2 gaea 的现状与判断

**事实**
- gaea 的 `grep` 是**纯 Go regexp + walk**，无外部二进制（`internal/gaea/tool/builtin/grep.go:1-18,77-80`）；噪声目录清单收敛到 `wspath` 单一真相源（`:73-75`）；输出格式刻意与 `agent/compress.go` 的 `compressGrep` 对齐，**为了回灌时能无损压缩**（`:22-27`）；默认 `max_results=200`、上限 1000（`:30-34`）。

**判断**：**不要抄 ZCode 的原生二进制内置化**。理由是成本/风险完全不对称——
- ZCode 为此付出的代价是：三平台产物构建、GLIBC 2.28 ceiling、PE 静态依赖门禁、SHA-256 清单、SEA 单文件解压、代理与内网镜像、arm64 未验证态……（`embedded-search-native-build.md:57-236` 整整 180 行都在讲分发）。gaea 是 Wails 单机应用，`grep` 只在工作区内跑（`confine`/`wspath` 已有边界），纯 Go 方案零分发成本且天然跨平台。
- ZCode 需要它是因为「用户机器上 `rg` 不一定有、GUI 起 shell 丢 PATH、还要在 SSH/WSL/Docker 远端跑」——gaea 没有这些场景。

**值得抄的四条（与二进制无关）**：
1. **Worker/goroutine 隔离 + 超时 + 可中断**：gaea 的 `grep` 在工作区大目录上没有显式超时（`grep.go:77+` 未见 deadline），模型一个 `pattern: "."` 就能卡住一轮。抄「独立执行单元 + 30s 上限 + abort」。
2. **三态输出模式**（content / files_with_matches / count）与 `head_limit`：gaea 只有 `max_results`（行数），没有「只给文件列表」模式；Office 场景「哪些 docx 含这段文字」是最常见查询。
3. **`glob`/`type` 过滤 + VCS 排除**：gaea 有 `include` 与噪声目录，够用。
4. **超限 artifact/spill + 明确 locator**：gaea 已有 spill，但 grep 的输出被 `compressGrep` 压缩后可能反而丢信息——两者顺序需确认（`execute_one.go:356` 的 `SmartCompress` 在 spill 之后，OK）。

**成本** S（超时 + 三态模式）
**风险** 低。

---

## 4. 校验与自检

### 4.1 ZCode 的「自检」是什么：**没有 verify 工具，靠状态机 + 工具语义 + 提示**

**设计要点**
- 没有 `verify`/`check` 类工具。自检通过四条机制实现：
  1. **读状态机**：Write/Edit 成功文案直接告诉模型「file state 是最新的，不用回读」（`write.ts:42-43`、`edit.ts:65`）；重复读命中缓存给 `file_unchanged` 桩（`read.ts:55-56`）。这是**用状态替代「写完再读来验证」**，省 token 且避免「读到旧缓存自我安慰」。
  2. **工具即校验**：Edit 的「唯一匹配 + stale 检查」让「改错了」在写入前就失败，而不是写完再验证。
  3. **ApplyPatch 的先验后写**（§1.7）。
  4. **Hook**：PreToolUse/PostToolUse 生态（`executor/hook-flow.ts`），项目可通过 hook 在写后跑 formatter/lint，并**把 Bash 写类命令导致的 read-state 失效提示回灌**（`bash-read-file-state.ts:59-93`）。
- 还有一条成人向的取舍：ZCode 明确**不**要求「写完必须跑 typecheck」——那是 AGENTS.md/项目规则的事，工具层不硬编码（对比 gaea 的 `stop_gate.go:111-115` 直接把「跑测试」写进 nudge）。

**证据**
- `write.ts:42-47`、`edit.ts:65,75`、`read.ts:55-56`、`edit.ts:203-234`
- `execute_one` 等价物：`executor/hook-flow.ts`、`docs/edit-tool-model-visible-result.md:11-18`
- 模型可见失败契约：`input-validation-model-content.ts:12-22`

### 4.2 gaea 的现状：**更强，不要降级**

- gaea 有 stop-gate 的 verify nudge（`agent/stop_gate.go:109-127`）、子代理 completion 的 acceptance_criteria 回执语义（`agent/complete_subtask.go:88`）、evidence ledger（`evidence/journal.go:35-60`）、Verifier 双通道（A 结构/引用、B 像素 diff）、带冲突保护的回滚（`app/gaea_verify.go:1-6`、`docs/archive/gaea-v41-evidence-chain-design.md:63-89`）。**这是 ZCode 完全没有的能力**。

**可抄的两点**
1. **「成功即声明新鲜度」的文案契约**：gaea 的写工具现在只报字节数（`writefile.go:101`、`editfile.go:107`）。应改成「改了什么 + 哪一版 + 是否需要重看预览」，并让模型知道**不必**回读（Office 场景应改成「必须重看预览」，因为模型从没看过字节）。
2. **跨工具读状态一致性（Bash 回填 / stale hint）**：gaea 的 `bash` 是独立工具，它的写不进 `fileObs`（`execute_one.go:322-341` 只认 `read_file`/`isFileWriter` 名单）。若模型用 `bash` + `python-docx` 改了 docx，`fileObs` 不刷新，下一轮 `edit_file` 却被 `checkFileStale` 拦住（因为它自己没写，观察也没变？实际是磁盘变了而观察没变 → 拦）。这个「工具间状态不一致」ZCode 用 `applyBashReadFileStateEffects` 系统性解决了，gaea 应对齐（或干脆禁止 bash 改办公文件）。

**成本** S（文案与 code） / M（bash 写/读状态对齐）
**风险** 中：给 bash 回填状态需要保守判定（ZCode 只认 `cat/head/tail/sed -n/grep` 且截断时不回填，`bash-read-file-state.ts:95-106`），做错会造出「模型以为读过了其实没读」的假新鲜。

---

## 5. 生成物交付（gen-ui file summary / artifact / output-directory）

### 5.1 ZCode 的契约

**设计要点**
- **输出目录不是 cwd**：生成文件落在 Host 提供的固定输出根（应用数据目录下 `visualizations/`），按 `workspaceIdentity/session` 分子目录；Agent **不回退到当前工作目录**。「固定目录」是位置约定，不是权限边界（`gen-ui-output-directory.md:5-19`）。
- 读取授权与目录分组**解耦**：会话子目录只是组织，不是读取边界；fork 继承父会话的 HTML 路径可直接加载，不需复制（`:13-15`）。同时仍拒绝根外路径、根下符号链接、非普通文件、超尺寸（`:17-19`）。
- **file summary 与 fileChanges 分离**：轮尾变更摘要（文件数/增删行数/展开列表 + 撤销）来自 CLI 的 `fileChanges`，而「可视化源文件」在摘要里被**过滤**（不计入统计、不显示 HTML 卡片），且过滤规则只认「本轮正文里完整且通过 schema 校验的绝对路径」（`gen-ui-file-summary.md:5-13`）。
- 摘要只含聚合值 → 折叠态也要异步读详情，读完前**不先显示**（避免统计先出现再消失）；请求失败保留原摘要 + 重试入口（`:24-27`）。
- 撤销边界不变：混合修改仍是「整轮撤销」，预检弹窗展示完整范围（`:29-30`）。

**证据**
- `docs/specs/gen-ui-output-directory.md:5-19,23-39,54-67`
- `docs/specs/gen-ui-file-summary.md:5-13,15-30,36-48`

### 5.2 对 gaea 的启发

- gaea 已有 `genui_validate`（结构校验，`genui_validate.go:22-63`）与 `internal/gaea/genui`（`ValidateSpec` + 200 节点上限 + 与前端 `limits.ts` 的单一真相源测试，`limits.go:16-18`、`limits_sync_test.go`）。**缺的是「交付物目录」与「文件摘要」两层**：
  1. gaea 没有「Host 提供的固定输出根」概念，生成物（图表 PNG、转换后的 PDF、导出的 docx）目前跟工作区混在一起。ZCode 的「输出根目录 + workspace/session 子目录」契约值得直接照搬——但它对 gaea 的取舍要反过来：gaea 是本地应用，用户**希望**产物就在工作区里；所以应是「工作区内固定子目录（如 `.gaea/deliverables/`）+ 可在 UI 打开」，而不是应用数据目录。
  2. 「摘要不计可视化源文件」这类细节对 gaea 的价值在于**同一原则**：模型产出的中间物（预览 JSON、baseline 快照、spill 文件）**绝不能**混进「用户交付物清单」，否则用户分不清哪个是成品。gaea 的 evidence journal 与 DeliverablesPanel 已有雏形，缺的是「什么算交付物」的显式白名单。
- **artifact 读回必须是字节**：ZCode 自己的注释写得很痛——`tool-artifact-store.port.ts:55-70` 明确「`.xlsx/.docx/.pptx` 落到默认 `application/json` 被当 utf8 解码——**办公文件读回即损坏且不可恢复**」。这是 gaea 必须一开始就做对的：Office 产物只走 `[]byte`/文件路径，任何「文本化」中间层都是雷。

**成本** M（输出目录约定 + 交付物白名单）
**风险** 低；风险主要在破坏既有产物路径（用户习惯），需要迁移提示。

---

## 6. 长任务（background task / 通知 / 进度 / 中间产物）

### 6.1 ZCode 的机制

**设计要点**
- **生命周期提供者**抽象：按工具名查一次表拿到 `{getSnapshot?, waitForTerminal?, cancellable?}`，五处分派（起轮询、等待终态、取消、事件 payload 的 cancellable、清理）退化成读字段——注释明确批评「按工具名散在五处 if」的漏改风险（`background-tasks.ts:68-84`）。
- **双通道终态**：1s 轮询（`getSnapshot`）+ 直接等待（`waitForTerminal`，更及时，且轮询源缺席时是唯一来源）。
- **任务输出契约**（`TaskOutput`）写得极细：
  - 状态机：running/pending + `block=false` → `not_ready`；`block=true` 每 100ms 轮询；终态 → success；超时 → timeout + 最新快照；removed → timeout + `task:null`（`task-output-contract.md:75-95`）。
  - 去重：先在输出投影成功后再原子写 `notified:true`；通知入队前同样检查——**TaskOutput 与 completion 竞争最多产生一次 `<task-notification>`**（`:94-95`）。
  - 输出来源分级：bash 优先取活动 shell 的 stdout/stderr 合并；否则读 canonical 输出文件；运行中只读头 30000 bytes（不加省略提示），终态读末尾 8MiB 并加 `[NKB of earlier output omitted]`（`:97-114`）。
  - 截断形状固定：`[Truncated. Full output: <path>]\n\n<保留尾部>`，且「前缀+尾部 = 限长」（`:116-124`）。
  - provider 可见结果用 XML 标签（`<retrieval_status>/<task_id>/<output>…`），**禁止**输出 description/prompt/result 等内部字段（`:126-152`）。
  - UI 展示是**独立投影**（`task_output` display，最多 2000 字符），不改写 provider 文本（`:156-170`）。
- **通知内容带产物清单**：`<reports count/preview/shown>`、`<artifacts count/preview/shown>`，`count` 是真实总数、`shown` 是预览条数，不等即「局部预览，全量经 run id 可取」（`runtime-task/notification.ts:20-45`）。

**证据**
- `core/src/tool/executor/background-tasks.ts:68-84,86-120`、`background-task-registry.ts`、`background-task-output.ts`
- `core/src/runtime-task/notification.ts:20-45`、`runtime-task/registry.ts`
- 契约全文：`docs/task-output-contract.md:1-181`

### 6.2 对 gaea 的启发

- gaea 有 `jobs` 管理器 + `bash_output`（非阻塞、可按 `filter` 正则、`tail_lines` 上限 500）+ `wait` + `kill_shell`（`bgjobs.go:18-80`），**基础齐**。缺三样：
  1. **单次通知去重**：gaea 没有「任务完成只通知一次」的原子 claim。`bash_output` 被模型反复调用是常态，若未来加「自动通知回灌」，必须有 `notified` 级别的幂等（ZCode 的 `claimRuntimeBackgroundTaskNotification`）。
  2. **终态读取规则**（运行中读头、终态读尾、带省略提示）：gaea 的 `bash_output` 是**增量**语义（只给新输出），一旦某次读取超过上限就会永久丢失中段，而且模型不知道自己丢了。ZCode 的「按任务状态切换读法 + 明确省略提示」是更诚实的协议。
  3. **产物清单进通知**：Office 长任务（批量改写 30 个 docx）的完成通知应带「改了 N 个文件、其中 M 个有警告、全量见 journal id」，而不是一句「done」。ZCode 的 `count/shown` 双计数模式可直接借。
- **不要抄**：ZCode 的 TaskOutput 已自标 DEPRECATED（`task-output-contract.md:28-31`：改让模型直接 Read 输出文件）。gaea 的 `bash_output`+spill 组合更经济。

**成本** S/M
**风险** 低。

---

## 7. 办公场景特有而 ZCode 没有的（以及水土不服清单）

ZCode 是代码 agent：它的世界是「UTF-8 文本 + 行 + 精确字符串 + diff + git」。Office 是「zip 容器 + XML 树 + 二进制 part + 关系图 + 版式引擎」。以下逐条对照。

### 7.1 gaea 必须自己解决、ZCode 完全无对应的六件事

| # | 能力 | ZCode 的真实状态（证据） | gaea 已有 | 缺口 |
| --- | --- | --- | --- | --- |
| O1 | **二进制容器读写** | Read 遇二进制直接报错（`text-metadata.ts:179-189,205-211`）；Write 只写文本（`file-system.port.ts:163-174`）。**它读不了 .docx** | `ooxml.ReadZip/WriteAtomic` 保条目序 + Deflate + 临时件替换（`ooxml.go:24-60`） | 无（gaea 领先） |
| O2 | **格式保真（不重排 XML）** | 无对应；它的「保真」= 不丢换行/编码 | docx/pptx 只对命中段落做字节级手术，「不解析/重建整份 OOXML」（`docxedit.go:1-6`） | 无（gaea 领先） |
| O3 | **样式继承 / 页面布局** | 无 | 部分：`xlsxedit.Style` 指针语义「不改的字段保持」（`xlsxedit.go:42-54`）；`standard/redhead.go` 做 GB/T 9704 版式 lint（设计见 `docs/archive/gaea-v41-evidence-chain-design.md:91-98`） | 版式回归检测依赖通道 B 像素 diff（`gaea_verify.go:50-57`），阈值/误报需标定 |
| O4 | **修订留痕（w:del/w:ins）** | 无（代码世界用 git diff） | `ApplyTrackedReplace`/`AcceptChanges`/`RejectChanges`（`docxedit.go:24-81`） | pptx 无修订制（`pptxedit.go:1-13` 直改 run），语义需对齐 |
| O5 | **公式重算** | 无 | `xlsxedit.Recalc(path, workspaceRoot) (RecalcReport, error)`（LibreOffice，`xlsxedit.go:477-487`）；apply 后重算 `gaea_xlsx_edit.go:149`，预览前按需重算 `gaea_preview.go:228-231` | 重算失败时的降级语义（写缓存值还是报错）需明文 |
| O6 | **二进制产物的字节级取回** | **自己踩过坑**：artifact 读回按文件名推 contentType，`.xlsx/.docx/.pptx` 落到 `application/json` 被 utf8 解码 →「办公文件读回即损坏且不可恢复」（`tool-artifact-store.port.ts:55-70`） | 预览走 `[]byte`→dataUrl / 结构化 JSON 两条路（`gaea_preview.go:17-45`） | 无（gaea 领先）；**但引以为戒** |

### 7.2 会在 gaea 水土不服的 ZCode 模式（逐条给替代方案）

**F1 · 「Read 后 Write」模型对二进制无效**
ZCode 整套 read-before-write 建立在「模型上下文里有文件全文」之上（`write.ts:294-305` 甚至用「内容逐字相等」做 stale 逃生门）。Office 里模型从没拥有过字节，只有**投影**（xlsxpreview 的抽样单元格、docx-preview 的渲染图、pptx 的 PDF 缩略）。
→ **替代**：把 stale 的语义从「内容是否相同」改成「**投影版本是否相同**」——每次写盘后递增 `projectionVersion`，模型看到的预览带版本号，写前声明 `based_on_version`；不等就拒绝并返回新预览。这比 ZCode 的内容比对更适合 Office，也顺带解决「模型以为自己看到了整张表」。

**F2 · `old_string` 唯一匹配不能作为编辑原语**
docx 的一句话会被拆进多个 `w:r`（拼写检查、格式、rsid 都会切分），pptx 同理。ZCode 的 `Edit` 假设「文本连续且唯一」（`edit.ts:203-234`），在 Office 上会大面积 `OLD_STRING_NOT_FOUND`。
→ **替代**：gaea 已经走对——段落/run 组级定位 + 不跨 run 断字（`docxedit.go:465-471`、`ooxml/runs.go`）。编辑原语应是**结构化 op**（`xlsxedit.Op` 的形态：`{"type":"set_value","sheet":..,"target":"B4"}`，`xlsxedit.go:21-40`），而不是字符串替换。ZCode 的 `ApplyPatch` 比 `Edit` 更接近这个形态，可借其「先验后写」的两阶段。

**F3 · 结果预算的「字节截断」会制造无效 JSON**
ZCode 的截断是字节/字符级 `truncate`（`result-serialization.ts:214-221`），对文本无所谓；但 gaea 的 `xlsxpreview.Render` 输出是 **JSON**，截在中间会让模型拿到不可解析的碎片，比"没有结果"更糟。
→ **替代**：Office 结果必须**结构化截断**（行/单元格/变更条目为单位 + `truncated:true` + 全量引用）。gaea 已有正确示范（`xlsxedit.PlanOps` 的 `truncated` 与 `gaea_xlsx_edit.go:33`），把它提升为通用设施；并把 `read_spill` 的 16KB 分页对 JSON 场景改成「按条目分页」。

**F4 · `concurrentSafe` 布尔不足以表达 Office 写入冲突**
ZCode 用布尔 + `sideEffectScope`（`scheduler.ts:85-103`）。Office 的冲突是**文件级**的（同一个 docx 的两个 part），且 LibreOffice 重算会派生临时目录/锁文件。
→ **替代**：保留 gaea 的资源键模型（`batch_executor.go:209-249`），并给 Office 工具声明 `file:<path>` + `office:<path>#recalc` 双资源；**不要**退回 ZCode 的布尔。

**F5 · 「不回读」的新鲜度承诺对 Office 是假话**
ZCode 的成功文案断言「file state is current in your context — no need to Read it back」（`write.ts:42-43`、`edit.ts:65`）。Office 里模型必须**重新看预览**才知道版式/样式有没有被破坏。
→ **替代**：文案改成「已写入，预览已更新到 vN，请核对版式/公式」，并把预览刷新做成写后自动动作（gaea 的 `gaea_xlsx_edit.go:156` 已经在 apply 后 Render，把它前移到工具返回值里）。

**F6 · artifact「retention/session/project」与「用户交付物」混淆**
ZCode 的 artifact 是**模型可见内容的落盘副本**（`result-serialization.ts:97-113`），不是交付物。gaea 若照搬，会把「预览 JSON / spill / baseline」和「用户要的 docx」混在一个体系里。
→ **替代**：三条独立通道——① 交付物（用户可见，有目录契约，§5）；② 证据/基线（audit 用，`evidence/journal.go`）；③ 工具结果泄洪（会话内，`spill`）。三者生命周期与可见性不同，**不能共用一个 store 抽象**。

**F7 · 「跑测试验证」的 verify nudge 在 Office 无对应物**
ZCode 不硬编码验证（把 typecheck/lint 交给项目规则）；gaea 的 `stop_gate.go:111-115` 却写死「run the project's test suite (go test ./... or equivalent)」——在办公场景模型根本没有测试可跑。
→ **替代**：verify nudge 应**按空间/域切换**：work 空间给「复核证据卡 / 看预览 / 检查版式」，play/code 给「跑测试」。现在这句 nudge 对办公任务是噪音，且会诱导模型去 `bash` 跑无意义的命令。

### 7.3 额外风险：两个「看起来能抄、实则不能」的项

- **`ApplyPatch` 的 unified diff 形态**：对 docx 无从表达（改的是 XML 树里的一个 run 文本）。可借的只有「两阶段」与「错误形态」，不是 patch 语法。
- **`Grep/Glob` 的路径模型**：ZCode 假设「工作区 = 代码目录」，会硬排除 VCS 目录并对 `node_modules` 之类噪声敏感；gaea 的工作区里可能真有名为 `build/`、`dist/` 的**办公资料目录**，噪声清单必须可配置（gaea 已收敛到 `wspath.GrepExtra`，`grep.go:73-75`，需确认它不含 `build/dist` 这类业务目录名）。

---

## 8. 落地建议（按 ROI 排序）

| 优先级 | 动作 | 落点（已确认存在） | 成本 | 风险 |
| --- | --- | --- | --- | --- |
| P0 | **写前 CAS**：`fileutil.AtomicWrite` 增加 `expectedVersion`（rename 前再 stat 比对 size+mtime/内容哈希） | `internal/gaea/fileutil/atomic.go:32-73`、`agent/file_observation.go:26-34` | S | 中（Windows 仍非真 CAS） |
| P0 | **禁止原子写 fallback 成非原子**；`symlink` 目标拒绝 | `fileutil/atomic.go`、`tool/builtin/confine.go:56-72` | S | 低 |
| P0 | **Office 结果结构化截断**（行/单元格/变更条目 + `truncated` + 全量引用），替代字节截断 | `agent/execute_one.go:356-364`、`office/xlsxedit/xlsxedit.go:170-195` 的 `truncated` 形态 | M | 中 |
| P1 | **投影版本（projectionVersion）契约**：写后递增、预览带版本、写前声明 `based_on_version` | `app/gaea_preview.go:17-45`、`app/gaea_xlsx_edit.go:23-33,156` | M | 中（需前端配合） |
| P1 | **工具级 code 集**（照搬 EditErrorCode 思路）：`old_string_not_found`/`ambiguous`/`stale_version`/`office_part_missing`/`recalc_failed`… | `tool/envelope.go:28-37`、`builtin/editfile.go:120-142` | S | 低（破坏既有解析） |
| P1 | **把 Office 改写提升为工具**（`docx_apply`/`pptx_apply`/`xlsx_apply`），走两阶段 plan→apply，复用 `PlanOps`/`ApplyOps` | 绑定层 `app/gaea_{xlsx,docx,pptx}_edit.go`、`office/xlsxedit:110,170` | L | 高（权限/审批/回滚都要接） |
| P2 | per-tool 结果预算声明 + 机器可读截断后缀 | `tool/tool.go:18-32`（加可选接口）、`agent/spill.go:20-40` | M | 中 |
| P2 | gaea 版 `resolveInput`（Office 坐标系/范围在审批前解析并回显） | `tool/tool.go`、审批层 | S | 低 |
| P2 | Bash 写/读与 read-state 对齐（或禁止 bash 改办公文件） | `agent/execute_one.go:322-341`、`file_observation.go` | M | 中 |
| P3 | 「交付物目录 + 交付物白名单」（区别于证据/泄洪） | `app/gaea_preview.go`、`internal/office/*` | M | 低 |
| P3 | 写后自动刷新预览并把版本号写进工具结果（替代「不用回读」文案） | `app/gaea_xlsx_edit.go:156`、`writefile.go:101`、`editfile.go:107` | S | 低 |
| — | **不抄**：ripgrep/bfs/ugrep 内置化；multi_edit 反向；`concurrentSafe` 布尔；`ApplyPatch` 的 diff 语法；artifact 单 store | — | — | — |

---

## 9. 证据索引

**ZCode（只读检出）**
- `apps/zcode-cli/packages/core/src/tool/`：`types.ts:241-246,302-401`、`read-file-state.ts:18-67`、`scheduler.ts:48,60-103,233-245`、`json-schema.ts:33-70`、`provider-visible-order.ts:1-52`、`input-validation-model-content.ts:12-72`、`handlers/{write,edit,read,read-text,apply-patch,grep,bash-read-file-state}.ts`（行号见正文）、`executor/result-serialization.ts:34-41,47-69,97-113,175-232,306-316,323-356`、`executor/background-tasks.ts:68-84,86-120`
- `apps/zcode-cli/packages/contracts/src/`：`interfaces/file-system.port.ts:7-47,58-59,93-107,163-174,233-273`、`interfaces/tool-artifact-store.port.ts:8-53,55-70,106-141`、`tools/write.ts:15-30,110-123,148-156`
- `apps/zcode-cli/packages/adapters/src/fs/`：`index.ts:51-54,115-203,291-328,463-483,700-755,775-777,1029-1124`、`text-metadata.ts:10-11,30-49,63-80,116-130,179-211,213-240`
- `apps/zcode-cli/packages/core/src/runtime-task/`：`notification.ts:20-45`、`registry.ts`
- `apps/zcode-cli/packages/core/src/embedded-search/`：`capability.ts`、`shell.ts`
- `docs/runtime-tools/{embedded-search-native-build,ripgrep-runtime}.md`、`docs/testing/file-tools-atomic-write-e2e-coverage-matrix.md`、`docs/specs/gen-ui-{output-directory,file-summary}.md`、`docs/task-output-contract.md`、`docs/edit-tool-model-visible-result.md`、`docs/superpowers/plans/{2026-07-06-bash-read-file-state-parity,2026-06-15-read-file-state-windows-path-key}.md`

**gaea（本工作区）**
- `internal/gaea/tool/{tool.go,envelope.go}`、`internal/gaea/tool/builtin/{writefile,editfile,multiedit,readfile,readspill,grep,confine,bgjobs,genui_validate}.go`
- `internal/gaea/{fileutil/atomic.go,fileutil/encoding/encoding.go,evidence/journal.go,spill/*,agent/{execute_one,file_observation,stop_gate,batch_executor,spill,agent}.go}`
- `internal/gaea/config/render_preserve.go`、`internal/gaea/genui/{validate,limits}.go`
- `internal/app/{gaea_preview.go,gaea_verify.go,gaea_verify_channel_a.go,gaea_xlsx_edit.go,gaea_docx_edit.go,gaea_pptx_edit.go}`
- `internal/office/{ooxml/ooxml.go,docxedit/docxedit.go,pptxedit/pptxedit.go,xlsxedit/xlsxedit.go,xlsxpreview/xlsxpreview.go}`
- `docs/archive/gaea-v41-evidence-chain-design.md`

---

## 10. 未验证 / 局限

1. **未运行任何代码**：ZCode 侧为只读源码与文档阅读，未执行其测试；行号以当前检出为准，若有未提交改动可能偏移。
2. **`[文档]` 标记项**（`docs/task-output-contract.md` 的状态机细节、`gen-ui-*` 的 UI 行为、`embedded-search` 的打包链路）只核到文档与关键实现片段，未逐行验证实现与文档一致。
3. **Office 专属结论（§7）基于「阅读 ZCode 的文件工具边界」+「阅读 gaea 的 office 模块」**，未做「用 ZCode 工具跑一遍 docx」的实证；但 `text-metadata.ts:205-211`（二进制拒绝）与 `tool-artifact-store.port.ts:55-70`（自家注释承认 office 读回损坏）是直接证据，足以支撑「ZCode 不覆盖 Office」的判断。
4. **未评估**：ZCode 的 permission/broker 全链路、MCP 工具接入、subagent 工具继承、compaction 与工具结果交互——超出本次「文档/文件写入类工具工程质量」的范围。
5. **gaea 侧未评估**：`internal/app` 绑定层的并发与 Wails 生命周期、前端预览的版本管理现状（只确认了 `PreviewResult` 形状与 apply 后 Render 的调用点），这两块决定 §8 的 P1「投影版本」成本，需要 gaea 负责人复核。
