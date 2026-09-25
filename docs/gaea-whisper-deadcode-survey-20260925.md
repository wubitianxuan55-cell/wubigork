# whisper 域死代码调研档（死代码刀4 立项材料，拍板池）

> 2026-09-25。方法=deadcode -test 基线（刀2/3 同款）+ git 近期活动核对 +
> ackem 蒸馏标记清点。本档只调研不动刀——整域大刀需拍板。

## 1. 基线数据

- `go run golang.org/x/tools/cmd/deadcode@latest -test ./internal/whisper/...`
  → **590 个不可达 func**，分布于 **147 个文件**（这些文件合计 ~21,095 行；
  不可达 func 本体估 **6,000–8,000 行**，其余为同文件在用代码）。
- 域头 267 处 `ackem` 蒸馏标记（对齐 ackem desktop-agent 上游，蒸馏=取道
  不取器纪律下落地为 in-tree 代码的批次）。
- 域活性：**生产活域**——轻语/语音/记忆是核心功能；最近一次生产修复
  v4.359（2026-09-20，whisper db 迁移原子化）距本档仅 5 天。不可达≠域死。

## 2. 分簇（按子系统）

| 簇 | 代表文件（不可达 func 数） | 性质判断 |
|----|----|----|
| A 机器地图族 | machine_map_store(11)/machine_map_indexer(11)/machine_map_collector/desktop_machine_map/desktop_parsers/desktop_routing | 纯 ackem 上游蒸馏，**从未接线**（NewMachineMap 不可达=入口都没有） |
| B agent 自治族 | agent_task_plan(10)/agent_policy(8)/agent_loop(8)/agent_job_manager(7)/agent_investigation/investigation_runner/chat_turn/guard/verify_task_plan(6)/task_plan_* 五件/parse_task_plan/turn_plan_prompt(7)/action_executor(7)/agent_tool_batch/agent_tool_round | ackem agent 回路蒸馏——现行轻语回合走的是 whisper_handler 主链，自治回路整族未挂 |
| C 记忆管道族 | memory_light_extract(10)/memory_write_job(7)/memory_audit(7)/fact_embeddings(9)/vector_search(8)/association_cold_start/auto_mirror_check/auto_mirror_policy/auto_consolidation_policy/contradiction_sampler/creator_memory×2/semantic_reranker/fact_embedding_cache | 记忆增强管道（向量检索/审计/反事实采样）——现行记忆走 facts/episodes 基础链 |
| D 会话呈现族 | turn_bubble_queue(10)/delivery_coordinator(11)/desktop_agent_delivery/desktop_synthesize/wave_chat(6)/wave_endpoint/wave_messages/capability_help_reply(7)/confirm_service/desktop_confirm_bypass | 桌面代理投递/气泡队列/wave 端点——呈现层未接线 |
| E repos 零散方法 | db/repos: memory_facts(11)/episodes(8)/chat_history/companion_state/diary/knowledge_triples/kv/openforu/procedural_habits/turn_traces | repos 整体活着（v4.358 刚修过 fts），死的是个别未调方法 |

## 3. 风险（为何进拍板池而非直接动刀）

1. **DSH 并发会话**：本树有另一执行体（DSH headless 流水线）同树作业的
   在案事实；whisper 是其潜在作业面，590 func 大删撞车代价高。
2. **预置资产口径之争**：267 处 ackem 标记=蒸馏批次资产。若口径是
   「蒸馏=预置能力池」，刀4 删的是未来功能的预制件；若口径是「蒸馏=取道
   不取器，落地未接线即死码」，则刀4 名正言顺。**口径需拍板**。
3. **体量**：估 6-8k 行、147 文件，是前三刀总和（~4,500 行）的 1.5 倍，
   一把梭的 review 面与回归面都大。

## 4. 建议（三选一）

- **方案 a：整域刀4**——冻结 DSH 会话窗口内，按簇 A→B→C→D→E 四个
  commit 分批删，deadcode 基线逐批复扫。收益 ~6-8k 行。
- **方案 b：只删 A 簇**——机器地图族（~40 funcs，machine_map×5 文件
  整文件删，估 ~1,500 行）零争议：纯上游蒸馏、入口都没有、无部分接线
  风险。B/C/D/E 留池待口径拍板。
- **方案 c：维持保留**——口径定为「蒸馏=预置资产」，本档归档为调研
  结论，todos 注明不再重查。

## 5. 复现

```
go run golang.org/x/tools/cmd/deadcode@latest -test ./internal/whisper/...
```
（-test 必带：不带会把测试在用的 seam 误判为死——刀2 的 8 处回滚教训。）

## 6. 执行记录（2026-09-25 追记）

- **方案 b 已执行**（cleanup 非版本刀）：A 簇 machine_map 4 文件（738 行）+ 编译强制级联剥壳investigation_runner/investigation_chat_turn 2 件（全死验证后删，-test 基线全 func 不可达+零外部消费），合计 **6 文件 1,178 行**，基线 590→549。
- **级联发现**：A 簇并非严格自包含——investigation_runner（B 簇死件）引用 desktop_machine_map 的 IsGameDirectory，删 A 后编译强制连剥两件；「零接线」判断需含「死件消费方」维度，刀4 排期时按依赖闭包分批。
- 余 B/C/D/E 簇 549 项维持待口径拍板（方案 a/c 之间）。
