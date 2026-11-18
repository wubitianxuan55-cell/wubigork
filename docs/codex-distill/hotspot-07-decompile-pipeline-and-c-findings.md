# Hotspot 07 · 反编译流水线打通 + C 级发现（会话记录 / 工具输出序列化器）

> 本轮把「汇编级」推进到「C 级」：Ghidra 12.1.4 headless 在**不做全量分析**的前提下，
> 定点反汇编指定区间并反编译。产物在 `work/decomp/*.c`（12 个函数）与 `work/decomp-summary.txt`。

## 1. 流水线（可复跑）

```
# 1) 一次性：无分析导入（把 codex.exe 装进独立工程，避免与全量分析工程争锁）
analyzeHeadless C:\Tools\ghidra-proj codex-targeted -import codex.exe -noanalysis

# 2) 每次：只对 .pdata 给出的函数区间反汇编 + 反编译
analyzeHeadless C:\Tools\ghidra-proj codex-targeted -process codex.exe -noanalysis ^
  -scriptPath C:\AI\re\codex-0.165.0\scripts ^
  -postScript DumpDecomp2.java C:\AI\re\codex-0.165.0\work\decomp 0x141140580:10169 ...
```

`scripts/DumpDecomp2.java` 的三步：`Disassembler.disassemble(entry,set,true)` →
`AutoAnalysisManager.disassemble(set)` → `createFunction(entry)` → `DecompInterface.decompileFunction`。

**踩过的坑（Ghidra 12.1.4 实测签名）**

| 尝试 | 结果 |
|---|---|
| `GhidraScript.disassemble(addr, setView, boolean)` | 不存在（只有 `disassemble(Address)`，单条指令） |
| `AutoAnalysisManager.startAnalysis(monitor, setView, boolean)` | 不存在（只有 `startAnalysis(TaskMonitor)` / `(TaskMonitor, boolean)`，都是全程序） |
| `Disassembler.disassemble(Address)` | 不存在（需 `(Address, AddressSetView, boolean)`） |
| 已存在的旧函数（body=1 字节） | 不会被 `createFunction` 覆盖 → 必须先 `removeFunction` 再建 |
| 结论 | `AutoAnalysisManager.disassemble(AddressSetView)` + `Disassembler.disassemble(addr,set,true)` 组合有效 |

## 2. 已反编译的函数（12 个）

| 文件 | 函数 VA | body | 判定 |
|---|---|---|---|
| `142b36390` | `0x142b36390` | 1,090 B | **Guardian 证据包 JSON 序列化器**（写 `{`=0x7b，按选项位条件发字段） |
| `140a1e070` | `0x140a1e070` | 4,569 B | **工具输出/终端记录序列化器**（`write_stdin` 命中） |
| `140a1fbe0` | `0x140a1fbe0` | 2,016 B | 同族序列化器（`write_stdin` 命中） |
| `140a1d4d0` | `0x140a1d4d0` | 1,342 B | 同族（`terminal_id` 命中） |
| `147eadaf0` | `0x147eadaf0` | 15,767 B | 工具搜索/路由簇（`tool_search` 命中） |
| `147ea8970` | `0x147ea8970` | 9,057 B | 同上 |
| `147ea5c30` | `0x147ea5c30` | 7,357 B | 同上 |
| `1457a2370` | `0x1457a2370` | 部分 95 B | apply_patch 簇（跳转表未跟进） |
| `145792020` | `0x145792020` | 部分 92 B | apply_patch 簇 |
| `145791210` | `0x145791210` | 604 B | apply_patch 簇（行匹配 helper） |
| `1410ca7b0` | `0x1410ca7b0` | 部分 57 B | ApprovalDecision 反序列化器（跳转表未跟进） |
| `141140580` | `0x141140580` | 部分 126 B | SandboxPolicy 反序列化器（同） |

## 3. C 级发现：会话记录的字段顺序（实测）

把伪代码里的 `&UNK_xxx` 解析回字符串后，`0x140a1d4d0` 与 `0x140a1e070` 引用的
数据区揭示了**序列化输出的字段顺序**（顺序即代码写法）：

```
terminal_sessions / terminal_operations / compactions / compaction_requests /
interaction_edges / raw_payloads                       ← 回合记录的顶层集合
thread_id / codex_turn_id / first_seen_at_unix_ms / channel / kind / body / produced_by
produced_by 取值: system | developer | analysis | final | compaction_marker
terminal_id / created_by_operation_id / operation_ids / request / model_observations /
raw_payload_ids / display_command / yield_time_ms / max_output_tokens / stdin /
exit_code / stdout / stderr / formatted_output / original_token_count / chunk_id /
call_item_ids / direct_tool_call / code_cell_output / target /
carried_item_ids / carried_raw_payload_ids / agent_result
```

**结论**：`0x140a1…` 这一族是**手写 JSON 序列化器**（不是 serde derive）：代码里直接写
`0x7b`(`{`)、`0x22`(`"`)、`0x5c`(`\`)、`0x100`（转义表相关），并按 `Option` 位决定字段是否输出。
这解释了为什么这些字段名能被 `lea` 直接引用（见 hotspot-03 的定位结果）。

## 4. 常量观察（未定论）

`0x140a1e070` 中反复出现 `4768`(0x12A0)、`4762`、`8250`(0x203A)、`596`、`256`；
`0x140a1d4d0` 中出现 `1952`、`1946`。这些更像**结构体偏移/UTF-8 长度表**，不是时间或 token 阈值；
不下结论，待与调用者（`FUN_140a34900` / `FUN_140a2c7e0`）对照后再判。

## 5. 未完成（下一轮）

- **枚举反序列化器被跳转表卡住**（`0x141140580` / `0x1410ca7b0`）：需要在同一区间再跑一次
  「Switch / JumpTable」分析（`AutoAnalysisManager` 的 `codeDefined` + 重新建函数），或改用
  全量分析工程（`codex-backend`，仍在跑）产出的跳转表。
- `0x147ce6880`（compaction_trigger 簇）反编译 **timeout**，需拆分或提高 `decompileFunction` 超时。
- 12 个热点里已落 C 级的只有 6 类；`exec 策略`、`线程持久化`、`MCP 连接`、`网络策略` 四个还停在 schema/字面量级。

---

## 附录 · 全量分析工程的最终结果（2026-11-17，3 小时上限）

后台跑的 `codex-backend`（全量分析）已结束，结论是**超时未完成**：

```
Decompiler Switch Analysis      75,464 s（累计）
Disassemble Entry Points         1,508 s
Non-Returning Functions            238 s
Shared Return Calls                 79 s
Subroutine References               49 s
------------------------------------------------
Total Time                      77,353 s
ERROR: Analysis timed out at 10800 seconds. Processing not completed.
Saved: /codex.exe (codex-backend)  ← 部分结果已入库
```

**三个遗留目标仍未闭合**（用分析工程重跑，结果与之前一致）：

| 函数 | 之前 body | 用分析工程重跑 | 结论 |
|---|---|---|---|
| `0x141140580` SandboxPolicy 反序列化 | 126 B | 126 B | 跳转表未在本轮分析窗口内解出 |
| `0x1410ca7b0` ApprovalDecision 反序列化 | 57 B | 57 B | 同上 |
| `0x147ce6880` 压缩触发族 | 93 B（失败） | 93 B（失败） | 同一条 pcode 错误 + 超时 |

**判断**：全量分析在 263 MB `.text` 上无法在 3 小时内收敛，且这两个枚举反序列化器的跳转表
恰好落在未完成区间。三条替代路径（都未执行）：

1. **手工读跳转表**：`0x1410ca7b0` 的反汇编里有 `jmp *0x...(,%rax,4)`，表在 `0x10782894`（RVA），
   用 4 字节相对偏移可直接算出各分支地址，再逐段 objdump——不依赖 Ghidra；
2. 提高 `-analysisTimeoutPerFile` 到 8–12 小时重跑（成本高，收益仅剩两处字段类型）；
3. 接受现状：两处枚举的变体集合已由**字面量**证明、语义由**源码**（`protocol.rs`）确认，
   缺的只是「字段类型与判别式编码」这类实现细节。

**工程体积**：`codex-backend.rep` 5.27 GB、`codex-targeted.rep` 1.77 GB（都在 `C:\Tools\ghidra-proj`，
不在 gaea 仓库也不在 RE 资料目录内，可按需删除）。
