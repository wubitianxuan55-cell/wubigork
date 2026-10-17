# MoE4All 引擎接入实用性评估（2026-10-17）

> 调研对象：https://github.com/Headmaster218/MoE4All （真机验证用 v1.0.3，2026-10-16 发布）
> 结论先行：**实用性=高，建议立项。阶段 0 真机验证已全部通过（2026-10-17 本机 Strix Halo）**。本质是「同模型家族、更高吞吐、支持并发解码、集成面与 strata 近乎同构」的本地 Vulkan 推理引擎。集成成本 ≈ 一条注册表种子 + 一套 herdsman 式生命周期。

## 0. 阶段 0 真机验证结果（2026-10-17，全部通过）

**环境**：AMD Radeon 8060S（Strix Halo 核显，UMA 63.6GB，Vulkan 报 74.4GiB device-local / 111.65GiB 双堆）+ Windows 11。安装 v1.0.3（zip 17MB，SHA256 校验过）至 `C:\MoE4All`。测试模型复用 herdsman 本地库 **Qwen3.6-35B-A3B-UD-Q4_K_M（21.6GB 标准GGUF，只读共享）**，无需下载。

| 验证项 | 结果 | 实测数据 |
|---|---|---|
| 闸门① 工具调用-非流式 | ✅ | `tool_calls` name+arguments 正确（`{"city":"上海"}`），finish_reason=tool_calls |
| 闸门① 工具调用-流式 | ✅ | delta 分片 name+arguments 到位，finish_reason=tool_calls，1.9s（注意：须关思考或给足预算，思考会先吃 max_tokens） |
| 闸门② 流式 usage | ✅ | 流内自带 usage 块；`stream_options.include_usage` 被接受且有独立 usage 末块；额外附赠 llama.cpp 风格 `timings` 对象（prompt/predicted_per_second 直接可读） |
| 闸门③ /v1/models 形状 | ✅ | 教科书标准：`{"object":"list","data":[{"id":…,"object":"model","owned_by":"local"}]}`，无 flexStatus 类异形 |
| thinking 开关 | ✅ | `chat_template_kwargs.enable_thinking` false→reasoning_content 缺席直接答；true→思考 2084 字；与 strata/modelhub 同通道 |
| reasoning_content 流式 | ✅ | reasoning 增量流式下发，gaea client_stream.go 已解码同字段 |
| 单路 decode | 50.3 tok/s | 服务端 timings 口径（256 token，短上下文，关思考） |
| prefill | 859.6 tok/s | 4029 token prompt（核显+conservative 预填充块 1024 行） |
| **并发 x2** | **双路各 ~39 tok/s，合计 52** | `--parallel 2`，strata 的 35B 完全做不到 |
| 冷加载 | 21.2s | 2 槽 × 262144 ctx；30723 专家块全量入 18.2GiB GPU 缓存（100% 覆盖） |
| 内存占用 | 进程提交 22.5GB | 系统同刻仍余 16.6GB 物理；KV 默认 Q8_0 动态扩展 |
| /health | ✅ 200 | 免鉴权（body 为空），健康探测直用 |
| embeddings | ✅ | serve-embedding 8093：批量输入+usage+1024 维向量正常 |
| 优雅停机 | ✅ | 进程可直接终止；`INFR_SHUTDOWN_FILE` 机制在册未单独验证 |

**生命周期相关实测补充**：`infr devices` 枚举 Vulkan 设备（含 external-memory 能力）；`infr resources` 输出 RAM/VRAM JSON 预检（gaea 启动前安全闸可直接消费）；`--addr` 改端口无冲突（测试用 8092/8093）；`infr multi` 可一服多模型按名路由（暂不需要）；采样缺省自动按 arch 表+generation_config（qwen35moe 自动 presence_penalty=1.5）。

**关键发现——模型库零成本复用**：本机 herdsman 库全是标准 GGUF（Qwen3.6-35B ×3 个变体、Qwen3.8-27B、Qwen3VL-2B、bge-m3/Qwen3-Embedding 等），MoE4All 直接吃路径即可。gaea 接入后模型选择器可扫描 herdsman 模型目录，**用户零下载**。此前评估里「GSQ-RCO 与 GGUF 不通用」的担忧仅限 strata 专有量化包，herdsman 库不受影响。

**遗留观察项**：流式 chunk 顶层多出非标 `timings` 字段（gaea 解码器忽略未知键，无害）；首请求 prefill 慢（24 token 用 1s，管线首编译，后续正常）；`INFR_SHUTDOWN_FILE` 与多模型 `multi` 模式待接 gaea 时按需验证。

## 0.5 同模型真机对擂：MoE4All vs Strata（2026-10-17，Qwen3.8-Flash-Next IQ2_XS）

**同一份 GGUF 文件字节级相同**：`C:\AI\strata\gguf\Qwen3.8-Flash-Next-GSQ-RCO-IQ2_XS-*.gguf`（68GB/2 分片）。关键发现：GSQ-RCO 只是量化校准配方，专家张量实为标准 llama.cpp 类型混编（IQ2_S/IQ2_XXS/Q2_0 等），**MoE4All v1.0.3 可直接加载 strata 专有包**（qwen4exp 架构正常解析，47 MoE 层/512 专家 top_k=10）。本机=Strix Halo 8060S 核显（strata 走 HIP/ROCm gfx1151，MoE4All 走 Vulkan）。测量=同一脚本 `scripts/verify_engine_compare.go`（gaea 自家 ai 客户端+聊天页 tok/s 口径，提示词与 verify_strata_speed.go 逐字节一致，各侧预热后采集）。

| 档位（gaea 客户端口径 tok/s） | Strata（HIP+MTP，自带调优） | MoE4All 保守档 | MoE4All 激进档 |
|---|---:|---:|---:|
| 短答·关思考 | **33.5** | 17.0 | 19.4 |
| 长答·关思考 | **31.2** | 17.3 | 20.2 |
| 思考缺省开·短答 | **38.9** | 20.7 | 21.1 |
| 长提示词 2111tok·TTFT | **4.8s** / 28.7 tok/s | 24.0s / 19.9 | 15.5s / 21.3 |
| 短答·复测 | **38.9** | 23.8 | 23.7 |
| 并发 x2 | 单路 **49.7/57.0**（合计 36.6） | 14.4/13.9（16.7） | 14.8/14.1（17.1） |
| 引擎侧 prefill @2111tok | 448 tok/s | ~11 tok/s | ~13 tok/s |
| 进程提交量 | **65.9GB**（吃满箱剩 8GB，先启 MoE4All 会挤爆它） | ~22GB | 38.2GB |
| 冷加载 | ~30s（含 MTP draft head） | **9.6s**（2 槽×262K ctx） | 32.7s |
| 上下文 | 131072（kv int8，32K 常驻） | 262144（kv Q8_0 动态） | 同左 |

**结论：本机+本模型=Strata 全面胜出，decode 约 1.7~2 倍、prefill 约 30 倍、并发反升（MTP 接受率 51/51、60/60）。差距主因排序：**
1. **MTP 投机解码**——strata 默认开（--spec 4 + 本地 draft），MoE4All v1.0.3 的 MTP **PARKED**（infr.example.toml 明示 "currently parked"），无公平开启路径；
2. **HIP/ROCm vs Vulkan 后端**——同卡上 strata 的 gfx1151 专用内核+hipBLASLt 调优在案；
3. SSD 分页**不是**主因——激进档把 73728 个专家块 100% 装进 33GB GPU arena 后速度几乎不变（19~24 vs 17~24），排除 I/O 瓶颈。

**MoE4All 的真实优势面**：轻（提交 38GB vs 66GB，不挤爆同箱其他引擎）、快装（9.6s vs 30s）、上下文翻倍（262K vs 131K）、双槽并发开箱即用、吃标准 GGUF（含 strata 专有包！）。**定位修正：MoE4All 不是 strata 的替代者，而是「标准 GGUF 通用引擎」**——本机 125B Flash-Next 场景 strata 不可替代；MoE4All 的价值在跑 strata 没有打包的模型（herdsman 库的 35B/27B/VL/embedding 全家）。

**对擂过程的运维教训**（接入 gaea 时要处理）：strata 提交量 66GB 意味着**与任何大模型引擎互斥运行**——gaea 侧若两引擎都装，切换时必须先停一个再起另一个（strata 在 MoE4All 残留未释放时启动会 WDDM 预算爆炸反复降槽失败）；反之 MoE4All 残留也会挤 strata。262K ctx 的 KV 动态扩展在 2 槽下实际 32K 起步。

**采集原始件**：`temp/cmp_strata.json`、`temp/cmp_moe4all.json`、`temp/cmp_moe4all_aggr.json`；对擂脚本新增 `scripts/verify_engine_compare.go`（未提交，随接入批次入库）。

## 0.6 同模型真机对擂：MoE4All vs herdsman（2026-10-17，Qwen3.6-35B-A3B-UD-Q4_K_M）

**同文件同脚本**（herdsman 库的 21.6GB 标准GGUF，`scripts/verify_engine_compare.go` 同口径）。herdsman 走其正规途径加载（`skill models start`），推理本体=llama.cpp b10837 Vulkan 构建，出厂参数：draft-mtp 投机解码、-np 1 单槽、f16 KV、256144 ctx、mmproj 视觉、temp 0.7/top-k 40/rep 1.1、--no-mmap。MoE4All：--parallel 2、Q8 KV、262144 ctx、temp 1/top-k 20（模型推荐）。

| 档位（gaea 客户端口径 tok/s） | herdsman（llama.cpp+draft-mtp） | MoE4All v1.0.3 |
|---|---:|---:|
| 短答·关思考 | **72.4** | 58.9 |
| 长答·关思考 | **58.3** | 54.0 |
| 思考缺省开·短答 | **68.9** | 52.9 |
| 长提示词 2111tok·TTFT | 7.69s / 63.6 | **2.18s** / 63.9 |
| 短答·复测 | **66.0** | 58.6 |
| 并发 x2 | -np 1 串行排队：单路虚高 82.8/82.6，**聚合仅 14.6**（wall 10.1s） | **双槽真并行 49.0/51.9，聚合 52.0**（wall 2.83s） |
| 内存提交 | 30.8GB（--no-mmap+f16 KV） | **22.5GB**（mmap+Q8 KV） |
| 冷加载 | ~10s | 21.2s 冷 / **5.7s 热** |

**结论：各赢半壁，没有全域王者。**
- **单路 decode：herdsman 赢 15~25%**——又是 MTP 起作用（llama.cpp 的 draft-mtp 已上车，MoE4All 的 MTP 仍 PARKED）。herdsman 的推理栈并不弱，recent llama.cpp + MTP 是硬底子。
- **Prefill/TTFT：MoE4All 赢 3.5 倍**（2111 token：2.2s vs 7.7s）；herdsman 每请求有 ~3-5s 固定开销（29 token 的短问也要 2.9~4.9s TTFT，疑似 per-request MTP/mmproj 初始化成本）。
- **并发：MoE4All 赢 3.6 倍聚合吞吐**（52 vs 14.6）——herdsman 出厂 -np 1 串行排队，双请求直接互相堵；MoE4All 双槽真并行。gaea 若要"对话+后台任务并发"，这是决定性差异。
- **内存：MoE4All 省 8GB**（mmap+Q8 KV vs no-mmap+f16 KV）。
- **herdsman 自身的优化空间被顺带照出**：manifest 自己推荐 q8_0 KV 实配却用 f16（白吃数 GB）；manifest `estimated_tps: 5.66` 与实测 58~72 严重失真；`--reasoning off` 下 chat_template_kwargs.enable_thinking=true 仍能正确开思考并解析 reasoning_content（能力在，只是默认配置保守）。

**对 gaea 的最终定位（三轮对擂汇总）**：Flash-Next 125B 旗舰场景=strata 不可替代（对擂 0.5）；标准 GGUF 单路对话=herdsman 已经够好甚至更快；**MoE4All 的净增量=并发槽+快 prefill+轻内存+统一 262K 上下文**，接入与否取决于 gaea 是否要"多功能真并发"这一场景。若立项，建议作为模型中心的"并发档"与 herdsman（单路档）并列，且其 `--parallel`/`--ctx` 作为 gaea 可配参数暴露。

**采集原始件**：`temp/cmp_herdsman_35b.json`、`temp/cmp_moe4all_35b.json`（+0.5 节的 cmp_strata/cmp_moe4all/cmp_moe4all_aggr）。

## 0.7 自持 fork 可行性（2026-10-17 源码核验）

**许可结论：三家两家可克隆。** MoE4All=Apache-2.0+上游 infr MIT（源码完整 17 crates，已克隆 `C:\AI\MoE4All`，25MB）；**strata 意外发现也是 MIT 开源**（Niko1221 and contributors，本机 `C:\AI\strata\repo` 自带完整 src/CMakeLists）；herdsman 唯一闭源（starwave 产品，仅 CLI/命名管道面）。

**MoE4All fork 的第一刀目标已实锤——MTP 解锁是「有图纸的修复」而非逆向**：
- MTP 代码在树且完成度高：`crates/infr-llama/src/mtp/`（qwen35 1 层 MTP 头加载+前向图+`catch_up`/`draft` 驱动原语，移植自 llama.cpp `draft_mtp`；qwen4.rs=Qwen3.8 专用 detached head 运行时）；qwen35 与 qwen35moe 都在 nextn 支持名单里。
- 被禁原因不是没写完，是已知正确性 bug：`mtp_enabled()` 硬编码 `false`，注释明说 "known-broken — 在 int8 解码内核下投机输出不再匹配贪心输出"。
- **上游在注释里写明了修复方向**：`mtp_spec_matches_target_only_greedy` 测试重新通过即可，"真正的修法是精度缓解（如 top-2 logit margin 低于阈值时用 f32 复核 draft），不是更快的内核"。修好→单路追平 herdsman/strata 的主差距（三轮对擂证明 MTP 即差距来源）；修不好零风险退回（普通 decode 路径不受影响）。
- 附带发现：herdsman 的 35B GGUF 自带 `qwen35moe.nextn_predict_layers=1`（MoE4All 可直接用）；strata 的 Flash-Next GSQ-RCO **无** nextn 键（其 MTP draft 是独立文件 `mtp-q2_0.gguf`，MoE4All 现行实现只吃 in-GGUF 头，Flash-Next 场景解锁不了——但那场景本就归 strata）。

**构建链现状与 fork 模型**：本机无 Rust 工具链（仓库钉 1.97.1 + 需 Vulkan SDK）；main 分支头在 1.0.0-maintenance，**fork 必须对准 release tag**（实测二进制=v1.0.3）。维护模型建议「发行跟踪型 fork」：对准 tag + 小 patch 集（MTP 修复 + gfx1151 出厂预设〔auto_profile/ubatch/submit dispatches/host tier，对擂数据在案〕+ 去向导 headless-first）+ 定期 rebase 上游摘樱桃。不建议深度内核自研（Vulkan MoE 内核=专家级工作量，上游 2035 commits 高速迭代追不动）。真实成本=团队需补 Rust+Vulkan 计算能力。

**strata fork 裁决**：合法但暂不启动——构建链重（Windows HIP/ROCm+hipBLASLt gfx1151），且其已是本机调优最优（对擂全胜），真实风险是上游单人项目断供；处置=源码留档（本机 repo 即存档）+跟随发行版，断供风险兑现时再启用自建（MIT 允许）。herdsman 维持闭源外部运行时定位，其三处自相矛盾（KV 推荐不采用/tps 估计失真/每请求固定开销）只能上游修。

**若立项自持引擎，批次拆分**：①装工具链+v1.0.3 tag 构建复现（半天，验证可建性）②MTP 闸修复+35B 同模型复测（1~3 天，图纸在注释里）③gfx1151 预设+headless 化+gaea 接入（1~2 天，复用 0.6 节集成清单）。一周内见分晓，失败兜底=继续用官方二进制+strata/herdsman 双引擎现状。

## 1. 项目是什么

- **构成**：MoE4All = 上游推理引擎 [infr](https://github.com/kryptic-sh/infr)（Rust、Vulkan-first、MIT）的发行包装：启动向导 + 模型下载中心 + 文档（Apache-2.0）。双许可均宽松，无传染性问题，集成只需保留 NOTICE。
- **原理**：吃标准 GGUF，利用 MoE 每 token 只激活少量专家的特性做三级存储流动：SSD 全量 → RAM cache → 弹性 GPU expert cache → Vulkan 计算。无 CUDA/ROCm 依赖，AMD 主力验证，NVIDIA/Intel 社区实测可用，Windows 一线支持（约 17 MiB 免安装包）。
- **服务形态**：`infr.exe serve --addr <ip:port> --parallel N --dev Vulkan0 <model.gguf>` 完全可无向导直启；OpenAI 兼容 `/v1/chat/completions`、`/v1/embeddings`、`/v1/models`、`/v1/responses`（部分）、`/health`（**永不鉴权**）。
- **成熟度**：299 stars / 2035 commits / 高速迭代（v1.0.0→v1.0.2 三天两修版），README 自认大量 AI 辅助开发。年轻、单人主导、快速补丁期——这是主要风险面。

## 2. 性能：与 strata 同机对照（RX 7900 XTX 24G + 64G RAM，ctx=163840）

| 模型 | 输入 | | Prefill | Decode x1 | x2 | x3 |
|---|---|---|---:|---:|---:|---:|
| Flash-Next 4.27bpw | 30K | MoE4All | 1415 | 50.4 | 66.2 | 71.2 |
| | | Strata v0.1.44.1 | 576 | 30.2 | 48.1 | 不支持 |
| Flash-Next | 150K | MoE4All | 1446 | 44.2 | 49.5 | 45.5 |
| | | Strata | 650 | 36.5 | — | — |
| Qwen3.6 35B | 30K | MoE4All | 2749 | 80.5 | 118 | 111 |
| | | Strata | 1325 | 19.4 | **不支持** | **不支持** |
| Qwen3.6 35B | 150K | MoE4All | 1169 | 41.9 | 50.9 | 49.3 |
| | | Strata | 437 | 11.0 | 不支持 | 不支持 |

- 单路 decode 1.7~4.2 倍；**并发 decode 越批越快**（微批合层），35B 三路 111~201 tok/s——strata 的 35B 完全不支持并发。这对 gaea 的多功能并发场景（对话+后台任务）是质变。
- README 自评「方向性对照」：对照时为兼容 strata 做了张量精度转换（Flash-Next 363 投影张量转 BF16），不能全归因 kernel。
- 内存口径：MoE4All 激进档=总 RAM−14GiB、保守档=可用 RAM−3GiB，可调；strata 实测提交量 ~95GiB（Host arena 47GiB + GPU expert cache 13GiB）。同量级，未见更省。
- 社区低配锚点：RX 7700 XT 12GB + 64G RAM 跑 35B-A3B @131K ctx 得 41.5 tok/s；Arc A770 16GB 得 17.4 tok/s。

## 3. 与 gaea 集成面逐项对照

gaea 引擎架构是数据驱动的：OpenAI 兼容 `/v1` 即自动继承调用、SSE 流式、故障转移、健康巡检豁免、IsLocal 离线门控、本地免费计价（`internal/modelengine/engine.go` 注册表 + `internal/ai/client*.go` 单一调用面）。

| 集成点 | MoE4All 现状 | gaea 对应 | 判定 |
|---|---|---|---|
| chat completions + SSE | stream=true、增量、[DONE]、SSE 错误事件 | `client_stream.go` 通用解析 | ✅ 即插即用 |
| thinking 开关 | `chat_template_kwargs.enable_thinking`（仅此通道+preserve_thinking+reasoning_effort 三键，余键 400） | strata/modelhub 已实证同通道（`client_chat.go:376`） | ✅ 复用既有分支 |
| 思考内容返回 | `reasoning_content` 字段（`reasoning` 兜底），流式下发 | `client_stream.go:489` 已解码累计 | ✅ 零改动 |
| 思考强度 | `reasoning_effort` low/medium/high/xhigh/max，**按模型模板收窄**（Qwen3.8 无 high，默认 xhigh） | `ReasoningEffort` 管道已有 | ⚠️ 需按模型映射，不翻译 high→xhigh |
| max_tokens | reasoning 与正文共享预算；另有 `serve.max_tokens_cap` | `ClampThinkingMaxTokens` 4096 守护同病同方 | ✅ 机制对齐 |
| 流式 usage 记账 | 文档未提 include_usage | 末块 usage 记账（`RecordCall`），已有 include_usage 降级 | ❓ **真机验证项 1** |
| 工具调用 | 文档确认存在（tool_calls 回放轮），无参数表/流式 delta 说明 | agent 功能重度依赖 | ❓ **真机验证项 2（最关键）** |
| /v1/models | 列已加载模型；鉴权可选 Bearer | flexStatus 教训在案 | ❓ **真机验证项 3** |
| 端口 | `--addr` CLI 参数，向导默认 8080 只是 wizard-state 非硬编码 | 8080=herdsman、8091=strata、8888=modelhub | ✅ gaea 自管进程指 **8092** 即零冲突 |
| 生命周期 | `INFR_SHUTDOWN_FILE` 优雅排空（含加载期）、`/health` 免鉴权、`infr devices` 枚举设备 | herdsman_lifecycle.go 模板（进程树击杀） | ✅ 比 herdsman 更干净：文件信号+健康端点 |
| 模型下载 | 内置 hub（HF 端点可换 **hf-mirror.com**、断点续传） | gaea 不下载 GGUF（strata 先例：用户自管） | ✅ 一期沿用 strata 口径，二期可收编 |
| 视觉/Embedding | Flash-Next 视觉（--mmproj + data URI）；/v1/embeddings（embedding_runner 可托管 llama-server） | 潜在增益，未评估 | 📌 另立观察项 |
| 多模型 | 单 serve 进程单 GGUF | strata 同口径（常驻单模型） | ✅ 口径一致 |
| License | Apache-2.0 + 上游 MIT | — | ✅ 宽松 |

## 4. 风险与不兼容点

1. **模型资产不通用**：strata 的 GSQ-RCO 量化包是专有格式，MoE4All 吃标准 GGUF——切引擎需经其 hub 重新下载（支持 hf-mirror，国内可行）。两引擎并存期磁盘双份。
2. **RAM 门槛**：Flash-Next 125B 级仍需 64GB 内存档机器；低配用户走 Qwen3.6 35B-A3B 或更低 bpw 量化。与 strata 同档，非新增门槛但需在模型卡标注。
3. **项目年轻**：v1.0.x 补丁连发（v1.0.1 修 UMA 显存重复、35B 并发显存异常），升级须逐版验证；建议 gaea 侧锚定发行版号+验收脚本（tok/s 四档实过先例）。
4. **上游关系**：引擎本体在 kryptic-sh/infr，MoE4All 是发行层；跟踪上游方向，防发行层停更时引擎断供（MIT 允许 gaea 自持 fork 兜底）。
5. **AI 辅助开发比例高**：文档与行为以 `crates/infr-server/src/lib.rs` 为准的口吻本身提示文档可能滞后——这正是三项真机验证存在的原因。

## 5. 集成方案草图（放行后）

**改动清单**（对齐 `internal/modelengine` 注册表惯例）：
1. `engine.go`：`EngineTypeMoe4All` 常量 + `IsLocal()` + `BuiltinEngineTypeOf` + NewManager 种子（BaseURL `http://127.0.0.1:8092/v1`，Label/Color/Icon，免鉴权对齐 strata）+ 对账测试三件同步。
2. `internal/app/moe4all_lifecycle.go`：仿 herdsman_lifecycle——起 `infr.exe serve --addr 127.0.0.1:8092 --parallel N --dev <dev> --set device.auto_profile=... <gguf>`；停机走 `INFR_SHUTDOWN_FILE`（比进程树击杀干净）；健康用 `/health`。设备枚举 `infr devices` 脚本化。
3. `client_chat.go`/`bridge.go`：strata 的 chat_template_kwargs 分支把 moe4all 并入同 case；reasoning_effort 按模型模板收窄映射。
4. 前端：`api/engines.ts` type 联合 + EngineSection/ModelCenterPage 卡片 + 模型选择。
5. 绑定面零新增（功能级 model key 是自由字符串）。

**工作量**：真机评估半日；接入约 1~2 天一个批次（含测试同步）。规模显著小于 strata 接入（当时含 flexStatus 解码等未知面）。

## 6. 立项建议：三阶段

- **阶段 0·真机评估（已完成 2026-10-17，全绿，见第 0 节）**：三项闸门全过，模型库零成本复用成立。~~本机装 v1.0.x…~~
- **阶段 1·注册表接入（1~2 天）**：按第 5 节清单落地，模型中心与 strata 并列，用户自选，定位互补非替换（strata 的 GSQ-RCO 量化继续服务存量）。
- **阶段 2·收编（另立批次）**：infr hub 下载接入（hf-mirror 默认）、keep-warm、vision/embedding 评估。

**与既有决策关系**：不在排除清单（GoalCard 系/MCP/平台化均无关）；符合「引擎可插拔、OpenAI 兼容即注册」既有架构方向。
