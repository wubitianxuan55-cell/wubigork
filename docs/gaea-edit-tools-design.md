# gaea S0.6 edit_file 工具层 · 设计（实现权威）

> 依据：只读勘察报告（2026）。背景：edit_file/multi_edit/edit_lines/move_file/grep 被 ~40 处
> 引用但全无实现（模型收到 "unknown tool"）。本文件为实现权威；实现顺序 grep → edit_file →
> multi_edit → edit_lines → move_file（风险递增），全部绿地为新工具（无半实现）。

## 0. 结论速览
- 五个工具全无实现（tool/builtin 无对应文件）；CLI 默认清单已列名（config.go:566-572），注册时打 warning（boot.go:542-548）；桌面端 Enabled=nil 全量注册（gaea_handler.go:59-60）→ **init 自注册后桌面端零配置自动生效**。
- **名字进 4 个名单即自动获得** stale 守卫/冲突串行/循环检测/缓存失效/证据台账：edit_file/multi_edit/delete_range/delete_symbol 已在全部名单；需补 edit_lines/move_file/grep。
- 最大坑：① move_file 缓存失效取不到路径（失效 switch 只解析 `path` 字段）；② compactDesc/compactSchema 无新条目会导出空 schema。

## 1. 五个工具最小契约
统一：路径经 `resolveIn(workDir, path)`；写目标过 `confine(roots,…)`；读写走编码感知 `readFileEncoded`/`Encode`（保 GB18030/UTF-16）；写回 `fileutil.AtomicWrite`（perm 沿用原文件）；错误一律 `return "", fmt.Errorf(...)`（CodeExecError + schema 回显，与 write_file 同水位）。

| 工具 | args（粗体必填） | 语义 | 输出 | ReadOnly | PersistWrite |
|---|---|---|---|---|---|
| edit_file | **path** **old_string**（非空） **new_string**（可空串=删除） replace_all(bool,false) | 读全文→统计：0 次→报错(old_string not found + CRLF 提示 + 建议重读)；≥2 且 !replace_all→报错(扩上下文或 replace_all)；替换一次后写回 | `edited <path>: N occurrence(s) replaced (+A/-B bytes)` | false | 否 |
| multi_edit | **path** **edits**[{**old_string** 非空, new_string, replace_all?}] | 内存串行替换（后一条作用于前一条结果），任一失败整体不落盘；成功单次原子写 | `<path>: applied M/N edits` + 逐行 `#i ok` | false | 否 |
| edit_lines | **path** **start_line**(1-based) **end_line**(含端点) **new_content** | 校验 1≤start≤end≤行数；替换区间；写回 | `replaced lines S-E in <path> (M→K lines)` | false | 否 |
| move_file | **source** **destination** overwrite(bool,false) | 源存在且 regular；两端 confine；目标父 MkdirAll；已存在且 !overwrite 报错；os.Rename 失败（跨卷）回退 copy+remove；成功后失效源/目标两路缓存 | `moved <src> → <dst>` | false | 否 |
| grep | **pattern** path(默认 ".") include(glob) max_results(默认 200) | regexp.Compile（失败→validation 风格错）；目录 WalkDir（跳 noiseDirs + .git + 二进制 NUL）按 include 过滤；输出 `path:line: content`（SmartCompress 自动接 compressGrep）；零命中 `(no matches ...)` | 匹配行流 | **true** | 否 |

取舍：edit_file 删除用 `new_string:""`（boot.go:671 注释宣称的 mode 参数不存在——**同步修正该注释**）；multi_edit 空 old_string **统一拒绝**（precheck 与 Execute 对齐，杜绝"precheck 放行→执行必败"）；edit_lines 用 1-based（与 read_file 显示行号一致）；grep 只读→权限自动 Allow、子代理可用；**不实现 PersistWrite**（写工作区文件非共享存储写）。

## 2. 名单对齐（实现时同步补全）
| 名单 | 位置 | edit_lines | move_file | grep |
|---|---|---|---|---|
| 缓存失效 switch | execute_one.go:184 | **加** | **加（特判双路径 source+destination）** | 不需要（只读） |
| isFileWriter（stale+NotifyEdit） | execute_one.go:471 | **加** | **加** | 不需要 |
| repeatSuccessSignature | execute_one.go:365 | **加** | **加** | 不需要（ReadOnly 不签名） |
| getConflictKey | batch_executor.go:173 | 加 `"file:"+path` | **返回 `"!write"` 串行**（同目标不同源并行有覆盖竞态） | 建议加 `"read:"+path`（可选） |
| turnFilesModified | agent_run.go:300 | 建议加 | 已列 | — |
| evidence isWriterTool | evidence.go:303 | **加** | **加** | 已在 isReaderTool |
| permission subjectKeys | permission.go:151 | — | **加 `"source"`**（否则 move_file(路径*) 规则无法匹配） | pattern 已在 |

- 校验复用：**不从 GaeaWriteFile 抽共享校验器**（internal/app 依赖方向倒挂；且其语义是用户显式保存）。工具侧用 builtin 内 resolveIn+confine（写根/符号链接穿越已解决）+ fileenc。GaeaWriteFile 白名单/2MB 是 UI 防呆，不是安全边界。
- 装配面：`Workspace.Tools()` 的 `all` 列表（workspace.go:39-46）与 `ConfineWriters`（confine.go:25-30）**必须加新工具绑定实例**——漏加的后果：CLI 可用而桌面端永远 unknown tool（白名单陷阱）。
- boot/config 默认清单**不需要动**（config.go:566-572 已列 4 名；multi_edit 不进 CLI 默认清单是既有 HideUnlessOnly 设计）。

## 3. precheck 对齐
- edit_file/multi_edit 参数名照 precheck 现状设计（path/old_string/new_string/edits[]），precheck 零改动适配；唯一改动：`precheckMultiEdit` 补"空 old_string 拒绝"一行。
- edit_lines/move_file/grep 不进 precheck（无锚点语义）。

## 4. 必踩风险
1. **compactDesc/compactSchema 缺条目**：所有现存工具实现 CompactDescriptor 且从 compact.go map 取值；新工具不补条目 → Schemas() 导出空描述+空 schema（tool.go:392-399 无 fallback）。**每个新工具必须补 2 条 compact 条目**（或不实现 CompactDescriptor）。
2. **move_file 缓存失效盲区**：失效 switch 按 `path` 字段解析，move_file args 无 path → 静默不失效；必须专案分支 source+destination 双 InvalidatePath。
3. **getConflictKey 对 move_file**：extractFilePath 取 source；"同目标不同源"两 move 不同键可并行 → 覆盖竞态；返回 `"!write"` 串行最稳。
4. **boot.go:670-671 注释与实现背离**（mode 参数不存在）→ 修正注释。
5. **编码/行尾**：readFileEncoded/Encode 成对；CRLF 文件中 `\n` old_string 不命中 → precheck 拦截给诊断；Execute 错误消息提 CRLF 提示，不做静默行尾归一。
6. **grep 输出契约**：必须 `path:line: content`（compress.go:250-252 解析格式），否则 compressGrep 退化为 passthrough；噪声目录（compress.go:269-273 noiseDirs）与二进制跳过必做。
7. **大文件**：edit 系读全文（与 write_file 同水位）；grep 必须 max_results 上限。
8. 命名撞车提示：whisper/agent_policy.go:20 与 office/executor.go:114-116 各有独立 `move_file` 动作，与工具层零耦合，勿混淆。

## 5. 测试计划
- builtin 包单测（白盒，参照 websearch_test/format_convert_test 风格 + addbuiltins_test.go:21-58 的 TempDir+Chdir 工作区绑定）：每工具核心路径 + 拒绝路径（old_string 缺失/多处无 replace_all/行号越界/跨卷/confine 外/符号链接穿越/编码保持 GB18030/权限位保持/空 old_string 拒绝/grep 非法正则/二进制跳过/噪声目录/max_results 截断/零命中文案）。
- agent 层集成（参照 tool_precheck_more_test 的 `&AgentRunner{}` 直调 + evidence_flow_test 的 `_ "builtin"` 注册）：precheck 阻断与放行；名单表驱动断言（tool_coherence_test.go:178-204 现成表加行）；move_file 后 tc 双路径失效。
- boot 装配：Workspace.Tools() 含新工具且绑定工作区；Compact=true 时 HideUnlessOnly 隐藏 multi_edit。

## 6. 实现顺序（每步可独立提交/回退）
S1 grep（只读最低风险）→ S2 edit_file（+Workspace/ConfineWriters 绑定 + 修正 boot 注释 + precheckMultiEdit 一行）→ S3 multi_edit（共享替换内核）→ S4 edit_lines（补 execute_one 三名单 + agent_run + evidence）→ S5 move_file（+缓存失效特判 + getConflictKey 串行 + subjectKeys 加 source）。
验证：build/vet/工具包+agent 全量测试；编辑链路端到端（stale 守卫/循环守卫/rewind/证据台账自动生效）；回退点=删除注册行即消失。
