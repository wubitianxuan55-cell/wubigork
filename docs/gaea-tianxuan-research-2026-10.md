# tianxuan 立项调研：社区与 GitHub（2026-10-17）

> 目标：fork MoE4All → 改名 **tianxuan** → 按本机（Ryzen AI Max/8060S gfx1151，64GB UMA，Windows）重设计架构，打造自有模型引擎。本报告是立项第一步（社区+GitHub 调研）的结论，架构重设计以第 7 节材料为输入。

## 1. 上游关系与仓库健康度

- **MoE4All（Headmaster218）= 事实上的引擎主仓**：2035 commits（最新提交 2026-10-17 凌晨——就在我们调研当天）、299 stars、24 open issues、11 PRs、发版节奏快（v1.0.0 10-14 → v1.0.3 10-16）。MTP/qwen4exp/视觉/hub 等全部演进都发生在这里。
- **kryptic-sh/infr = 起源上游**：MIT、1538 commits、31 stars、自称 "Early WIP"、**无任何发行版**、README 未提 MoE4All。结论：**fork 的有效上游就是 MoE4All 本身**，infr 只需在 NOTICE 里保留署名。
- **巴士因子=2**：mxaddict 1327 提交（65%）+ "John(Desktop)" 497（24%），其余长尾。作者身份混乱（John / John(Desktop) / "Your Name" 多本地身份）印证 AI 辅助开发、git 署名不可靠。两人断更风险真实存在——**反过来说，tianxuan fork 天然就是"延续者"角色的候选**。
- 发版 tag 链完整：`release-0.7.0`…`release-1.0.3`（2026-10-16 打 tag，main 仅领先 2 提交）——**fork 基准点取 `release-1.0.3`**，与我们真机验证的二进制完全对应。

## 2. 社区生态

- **中文社区活跃**：CSDN 部署教程（[moe4all AMD消费显卡跑MOE模型](https://blog.csdn.net/huanglin6/article/details/164195130)）、issue #83 建了微信群。中文是项目第一语言（README/文档/commit 皆中文）——对 tianxuan 团队是天然优势。
- **Strix Halo 生态现成**：[kyuz0/amd-strix-halo-toolboxes](https://github.com/kyuz0/amd-strix-halo-toolboxes)（gfx1151 上 llama.cpp Vulkan/ROCm 基准容器社区）、[AMD 官方 Ryzen AI Halo 开发者页](https://www.amd.com/en/products/processors/desktops/ryzen/ai-halo.html)、[LocalLLaMA MoE-on-iGPU 讨论](https://www.reddit.com/r/LocalLLaMA/comments/1na96gx/)。tianxuan 若做好 gfx1151，这就是现成的早期用户与反馈池。
- **竞品参照系**：llama.cpp Vulkan 在 gfx1151 有 [MoE prompt processing -39% 回归 issue](https://github.com/ggml-org/llama.cpp/issues/22375)；issue #30 报告里同卡 llama.cpp ROCm tg128 63-85 vs infr 51.4（35B UD-Q5）——**llama.cpp 仍是本芯片 decode 标杆**，tianxuan 的对标比值口径（optimization-playbook 定义）可直接沿用。

## 3. 直接命中本机的已知问题（= fork 的机会清单）

| Issue | 内容 | 对 tianxuan 的意义 |
|---|---|---|
| **#28** | **APU/UMA 预算双计**：device.ram_budget 只管 host 层，GPU 侧独立规划，UMA 上两层是同一份物理内存 → 超预算 swap 抖动整机假死（D 状态） | 维护者 10-15 回复"新版应已修"但 **issue 未验证关闭**。tianxuan 第一贡献点：UMA 单预算规划器+启动期硬校验（Windows WDDM 同样适用，我们真机就撞过 strata/MoE4All 互斥假死） |
| **#30** | **Strix Halo Phase-1 端到端报告**（128GB Linux/RADV）：构建直接过；submit splitter split/16→4096 = **3.6× decode**；35B 全驻留 51.4 tg128；Flash-Next 74GB 全驻留 19.1；144GB DeepSeek 分页可跑 | 同芯片已验证可行；**split 因子是本机第一调优旋钮**（Windows 侧没人报过——我们的对擂数据可成首发 Windows gfx1151 报告） |
| #36/#70 | MTP 复活追踪 + 资源预留提案（含 AMD 反例） | MTP 修复有社区协同，不是我们独自战斗 |
| #31 | 分页 decode 单线程 ~2GB/s、NVMe 闲置 | 若跑 68GB+ 模型（Flash-Next），多线程分页是性能大头 |
| #71 | KV cache 优化提案（预估 50.9→62.6 tok/s） | decode 提速候选 |
| #5 | OpenAI usage 只报未缓存 token，harness 记账错 | **直接影响 gaea 计量**，接入前必须修或对账 |
| #52/#51 | Windows device-lost（NVIDIA）/ qwen4exp vision panic | Windows 稳定性面参考 |

## 4. 架构资料完备度（fork 可行性最大加分项）

`documentation/` 是罕见的专业级文档树，重设计不用从零考古：

- **architecture/**：system-overview、codebase-map、runtime/parallel-scheduler、**runtime/qwen38-mtp（MTP 专项设计）**、memory/runtime-resource-lifecycle、services/server-and-session-cache、vision-and-embedding
- **decisions/**：ADR 决策记录
- **development/**：building-from-source（Rust 1.97.1 钉版 / MSVC x64 crt-static / **glslc shaderc≥2025 构建期编译是唯一硬门槛** / `cargo build --release --locked -p infr-cli` / package-windows.ps1）、**optimization-playbook（性能方法论）**
- **evidence/**：acceptance、audits（含 48h soak）、benchmarks（**qwen38-mtp-150k 基准 = MTP 曾实测通过才被停用的证据**、flashnext expert cache capacity、community-reports）
- **optimization-playbook 核心资产**：6 类瓶颈分类法（错误内核门控→网格未填满→占用率→派发开销→CPU 受限→内核微架构，按修复性价比排序）；**已实测死路清单（fp8/int8/bf16 coopmat 全输 f16——省我们重蹈）**；全套 profiling 工具链（INFR_PROF_OPS 逐算子、INFR_PROFILE=1 构建期插桩、samply）；**已知缺口正好是我们的模型：qwen35 引擎（DeltaNet occupancy、narrow-n split-K）、warptile 36TF vs llama.cpp 45-50（宽形状）**
- 工作纪律成熟：设备互斥串行、CPU 后端为黄金基准、逐位一致设计原则、集成 GPU 提交中途 kill 会留 D 状态（解释了我们真机撞过的引擎互斥假死）

## 5. 改名 tianxuan 与合规清单

1. 保留上游 `LICENSE`（Apache-2.0）+ `LICENSE-MIT` + `NOTICE`（infr 署名），一字不改；
2. README 顶部声明："tianxuan is a fork of MoE4All (Apache-2.0), engine based on kryptic-sh/infr (MIT)"——改名无商标约束（两许可都无商标条款），"MoE4All"/"infr" 名字留给上游；
3. fork 结构：`upstream` remote = Headmaster218/MoE4All，从 `release-1.0.3` tag 拉 `main`；patch 集走 `gfx1151` / `mtp-revival` 等特性分支，定期 `git fetch upstream` 摘樱桃；
4. 品牌物（名称/图标/文案）全新，不复用上游 wizard 文案与 Logo。

## 6. 风险登记

- **两人上游**：断更/转向风险高（我们 fork 后就是延续者，可控）；PR 上游的合并节奏不可控（#28 十天未验证关闭）。
- **Rust+Vulkan 计算能力线**：团队主栈 Go/TS，需补能力；但 optimization-playbook 的方法论+分类法+工具链大幅降低门槛（照着 6 类分类法打，第一类"内核门控"修复是一行改动级别）。
- **Windows 是二线平台**：上游主力验证 Linux/AMD（RADV），Windows AMD 闭源驱动路径社区报告少——我们真机数据（8092/8093 对擂全套）恰好是稀缺样本，tianxuan 的 Windows gfx1151 支持本身就是差异化贡献。
- AI 辅助代码占比高：靠 evidence/ 的黄金值+一致性测试套件兜底（CPU 后端为基准），fork 后每刀先跑 `cargo test -p infr-llama --release --test cpu_backend`。

## 7. 重设计的输入材料（下一步架构设计的弹药）

1. **本机画像**：8060S 40CU RDNA3.5 / 64GB LPDDR5X UMA / Windows 11 / 单用户桌面 + gaea 多功能并发需求；模型甜点=35B-A3B 级（专家可 100% 入 GPU），上限=68GB Flash-Next（需分页）。
2. **实测基线**（我们的对擂数据）：MoE4All 35B 53~64 tok/s 单路 / 52 聚合并发 / prefill 860 tok/s@4K / 22.5GB 提交；对照 herdsman 72 / strata 31~39(66GB)。
3. **调优旋钮清单**（来自 playbook+issue #30）：submit split 因子（4096 级收益 3.6×）、ubatch、auto_profile 两档、paging.cache、KV dtype（q8_0）、--parallel、--ctx、MTP 闸。
4. **重设计方向候选**（按价值排序）：①UMA 单预算规划器（修 #28 类，Windows 假死免疫）②MTP 复活（图纸在 mtp_enabled 注释+issue #36/#70 协同）③gfx1151/Windows 出厂预设+启动自调优（resources 预检→自动选档）④gaea 深度集成（生命周期/模型扫描/usage 记账对齐 #5）⑤分页多线程（68GB+ 场景）⑥qwen35 decode 内核（playbook 认领的缺口，风险最高放最后）。

**结论：调研支持立项。** 上游健康但两人巴士因子、社区活跃且中文主导、架构文档完备到超出预期、已知问题清单正好是 fork 的路线图，改名与许可无障碍。下一步=批次 1（工具链+release-1.0.3 构建复现）。

## 8. 重点参照：halogen（peonist-ai）——本芯片"手工内核"路线的 proof（2026-10-17 补）

> 用户指定重点参照。halogen 是专为 Strix Halo（gfx1151）手工写内核的推理引擎家族，**它证明了"单芯片单模型极端定制"路线在本机硬件上的天花板**——正是 tianxuan 想走的路。

### 8.1 两个产品

| | [halogen-flash-server](https://github.com/peonist-ai/halogen-flash-server) | [halogen-server](https://github.com/peonist-ai/halogen-server) |
|---|---|---|
| 模型 | Qwen3.8-Flash-Next（179.55B MoE） | Qwen3.8-27B |
| 热度 | **1.0k stars**/73 forks/144 commits，活跃 | 131 stars/11 commits，早期 |
| 内核 | HIP/ROCm 手写，"every kernel for this one GPU" | 同左，构建硬性拒绝其他 GPU |
| 本机内存适配 | **默认 checkpoint ~93GB（4.16bpw），超出我们 64GB** | 35.9GB（~6.3bpw）✅ 装得下 |
| 基准（128GB 机） | decode 54~94 tok/s（按文本类型）、prefill 1675~1821 tok/s、**131K 上下文只衰减 4%**、代理轮 4.2s/TTFT 1.6s@30-96K | decode 31.7 tok/s（DFlash2）/串行 10.6（已贴带宽墙：249 vs 实测 240 GB/s） |

对照我们实测：strata 跑同款 Flash-Next（3.075bpw）31~39 tok/s、prefill 448——**halogen 用更高精度（4.16bpw）还快 2~3 倍、prefill 快 4 倍**。通用引擎（MoE4All Vulkan 17~24）与手工内核的差距在这个芯片上是量级性的。

### 8.2 许可判定（决定性）

**source-available，不是开源**：自用修改允许，**向第三方分发修改版禁止**（"Redistribute a modified Software image to third parties" 明文禁止），商用/原样再分发/发布基准免费，"halogen"/"Peonist" 商标受保护。
→ **tianxuan 不能基于 halogen 代码 fork**（代码基保持 MoE4All，Apache-2.0/MIT 全权可改）；可以：参考架构思想（不受版权约束）、原样运行 halogen（外部引擎，同 strata/herdsman 模式）、引用其公开文档与基准方法。

### 8.3 tianxuan 要借鉴的系统设计（思想移植，非代码移植）

1. **字节级一致的 draft-then-verify 投机解码**：输出与串行贪心逐位相同、每次发布在三个 drafter 上验证——这正是 MoE4All MTP 闸（"不再匹配贪心输出"）的行业标准解法，tianxuan 修 MTP 时照此纪律做（dual-draft：draft head + prompt lookup，后者对 agent 场景的工具 JSON 重复结构近乎免费）。
2. **磁盘持久化 prompt cache**：重启后会话恢复；26K 提示 TTFT 提升 20×。gaea 办公/小说代理的长会话直接受益，属 tianxuan 服务层必抄设计。
3. **统一 KV 池 + 会话 fork/join**：一个 KV 池承载多会话——gaea 多功能并发的正确形态（对应我们实测的并发档诉求）。
4. **代理轮经济学**：按"每轮重发完整历史的总耗时"优化（4.2s/轮），prefill 深度衰减小是前提——tianxuan 的 prefill 目标应设为"131K 保持 32K 速率 96%"级别。
5. **GDN O(1) 状态**：Qwen3.6/3.8 的 GDN 层 48/64 无需 KV——内存与速度的结构性杠杆，tianxuan 的 KV 规划应显式利用。
6. **带宽墙诚实核算**：decode tok/s × bytes/token vs 实测带宽上限（本机 ~240-256GB/s）作为天花板指标，量化选型（bpw）据此定——halogen 用 4.16bpw 比低精度还快，说明内核效率可补偿带宽差。
7. **面向 agent 的产品面**：Responses API（Codex）/Messages API（Claude Code）/llms.txt/AGENTS.md/token 预算/分类器端点——与 gaea 的 agent 主航道完全同构。
8. **host-setup 教训**：BIOS UMA carve-out 设最小、全部走系统 RAM。

### 8.4 平台差异与 tianxuan 的路径修正

- halogen 只发 **Linux ROCm 容器**（podman + /dev/kfd）；本机是 Windows 11。strata 证明了 Windows HIP 在 gfx1151 可行（ROCm Windows + hipblaslt），但 halogen 不发 Windows 包；WSL2 ROCm 对 gfx1151 的支持未经验证。
- **修正后的 tianxuan 三层路径**：①近期=fork MoE4All（Vulkan/Rust 可改合法）+ 修 MTP（照 halogen 的 verify 纪律）+ gfx1151 预设 + 系统层借鉴（prompt cache/KV 池/双 drafter）；②中期=评估 halogen(-server) 经 WSL2 或原样安装作为 gaea 的外部引擎（同 strata 模式，27B 档可行、Flash-Next 档受 64GB 限制）；③远期=若 tianxuan 成熟，自写 gfx1151 HIP 内核（halogen+strata 双证明该路线天花板最高）。

## 9. 结论（更新）

调研（含 halogen 重点参照）**强烈支持立项**：halogen 用同芯片证明了"为一块 GPU 手写内核"能比通用引擎快 2~4 倍——tianxuan 的差异化路线（gfx1151+Windows 深度定制）有天花板依据；代码基走 MoE4All fork（许可全权可改），halogen 以思想参照与外部引擎两种方式参与。批次 1 不变：工具链 + release-1.0.3 构建复现。

## 10. Gufo 深挖与基座决策修订（2026-10-17 补，用户点名）

### 10.1 [gufo-org/gufo](https://github.com/gufo-org/gufo) 是什么

"The Open Source Strix Halo AI engine"——**就是我们这颗 8060S（gfx1151/Ryzen AI Max+ 395）的垂直引擎**：C++20 原生 HIP 内核（hipBLAS/Lt+rocWMMA），GGUF 原生加载，783 stars/112 forks/601 commits 活跃，**MIT**（可全权 fork 修改再发布）。参考谱系：llama.cpp（GGUF/正确性）、hipEngine（torch-free HIP）、vLLM（continuous batching 思想）、ds4、audio.cpp。设计哲学：不跨模型复用 kernel，只做该尺寸下最优的模型。

**本芯片级实测（README 峰值口径）**：

| 模型 | prefill | 单路 decode | 8 并发聚合 |
|---|---:|---:|---:|
| Flash-Next Q4_K_XL（MTP） | **1722 tok/s** | **73.44** | **194.29** |
| Qwen3.8-27B（DFlash2） | 656 | 70.56 | 123.00 |
| DeepSeek V4 Flash（DSpark IQ2XXS） | 485 | 26.62 | 54.74 |

投机解码按模型配 drafter（DFlash2/MTP/DSpark），**MTP 是活的**（MoE4All 的修复难题在 Gufo 是招牌功能）；conversation caching 一等公民；OpenAI Chat+Responses API；**多模态全家桶**（图像输入/ASR 15.27×实时/TTS 语音克隆 2.54×/Qwen-Image-2.1 图像生成进行中/视频进行中）——与 gaea 的工作室/语音/图像线天然重合。

### 10.2 平台：Windows 障碍已有人趟平

上游官方仅 Linux（GCC 15.3+ROCm 7.2.3 / Nix / podman）。但 **[pixmaate/gufo](https://github.com/pixmaate/gufo) 分支把 Windows 11 x64 做成一等功能**：原生构建走 AMD **TheRock ROCm**（AMD 开源 ROCm 发行线，strata 已证明 Windows HIP+gfx1151 工具链可行）、预构建 zip 发布、start.cmd/compat 层，459 commits。待验证项：该 fork 的维护活跃度与实测稳定性。

### 10.3 基座决策修订（重要）

**建议 tianxuan 主基座从 MoE4All 切换为 Gufo。** 理由全部指向"适合本机配置"：
1. **性能 3 倍**：同芯片同模型级 73/194 vs MoE4All 17~24/52——垂直 HIP 内核+可用的 MTP 是现成的，不用我们修；
2. **MIT 全权可改**，与 MoE4All 同级干净（halogen 做不了基座）；
3. **功能面重合 gaea 全栈**：LLM+ASR+TTS+图像生成+会话缓存+Responses API，fork 一次全家受益；
4. **Windows 缺位=tianxuan 的差异化正身**：Gufo 上游 Linux-only、halogen Linux-only、strata/herdsman 闭源——"第一个一等支持 Windows gfx1151 的开源 Strix Halo 引擎"这个生态位还空着；
5. MoE4All 降级为参照系：工程纪律/documentation 树/Vulkan 兜底层（若 HIP 链路受挫，它仍是可用的 fallback 基座）。

**修订后批次**：批次 1 = ①跑通 pixmaate fork 的 Windows 预构建包（半天实测）+ 读 fork 的 fork 仓库的 `WINDOWS.md` 摸 TheRock 工具链 ②上游 Gufo Linux 容器冒烟（对照组）→ **基座定夺点**；批次 2 = Windows 链路打磨 + gaea 接入（8094 端口/lifecycle/模型扫描照旧清单）；批次 3 = 本机深度调优（prefill/并发/prompt cache）+ MTP/DFlash2 预设 + 品牌化 tianxuan 发行。

### 10.4 调研过的引擎全景（截至本日）

| 引擎 | 栈 | 本机可用性 | 同芯片性能锚点 | 许可 | 对 tianxuan 的角色 |
|---|---|---|---|---|---|
| **Gufo** | C++20 原生 HIP | Linux 官方 + **Windows 分支现成** | **73.4 / 194 聚合 / prefill 1722** | **MIT** | **主基座（建议）** |
| MoE4All/infr | Rust+Vulkan | Windows✅（实测） | 35B 17~24 | Apache+MIT | 参照系+fallback 基座 |
| strata | HIP 闭源安装包 | Windows✅ 本机现役 | Flash-Next 31~39/prefill 448 | MIT（源码在本机） | Flash-Next 现役/内核参考 |
| herdsman | llama.cpp Vulkan 壳 | Windows✅ 现役 | 35B 58~72（MTP）、并发串行 | 闭源 | 现役通用/闭源依赖待替代 |
| halogen-flash-server | 手写 HIP Linux 容器 | ❌（93GB+Linux） | Flash-Next 54~94/prefill 1821 | source-available | 思想参照（8.3 节清单） |
| halogen-server | 手写 HIP Linux 容器 | 35.9GB 可装但 Linux-only | 27B 31.7 | source-available | 同上 |
| llama.cpp | C++ Vulkan/ROCm | Windows✅ | 35B tg128 63~85（ROCm，#30） | MIT | 基准对照组 |
| hipEngine / vulkanforge / airframe / KyaniteLabs / q38rocm / ds4 | HIP 或 Rust/WebGPU 各异 | 未深查 | — | 各异 | Gufo 的参考谱系成员，随基座顺手覆盖 |

## 11. 最终结论（本轮修订）

**"打造适合本机配置的引擎"的最短路径 = fork Gufo、把 Windows gfx1151 做成一等功能、品牌化为 tianxuan。** 性能上限（73/194）已被 MIT 许可的开源项目在本芯片实测证明，Windows 生态位空缺正是差异化价值，MoE4All 保留为 fallback 与纪律参照。下一步=修订后批次 1（pixmaate Windows 包实测 + TheRock 工具链盘点）。

## 12. 批次 1 实测结果（2026-10-17，基座定夺=Gufo 成立）

预构建包：`gufo-windows-2026-10-03.zip`（138MB，tag windows-2026-10-03，commit 2eb0778），装 `C:\AI\gufo-win`。**zip 自带 ROCm/hipBLASLt/VC++ 运行时，TheRock 仅自构建才需要**——运行门槛只有"Strix Halo 机器+AMD 驱动"。

- **HIP 通路验证**：35B 权重 21.1GiB 1.7 秒上传（12.46 GiB/s）。
- **坑（已定位）**：本地 35B UD-Q4_K_M 不兼容——`output.weight` 是 Q6_K，此构建只认 F32/Q8_0/BF16（作者实测 35B 用 UD-Q8_K_XL）。换本地 Qwen3.8-27B-UD-Q4_K_M 一次成功。
- **同口径实测**（8094 端口，`-p` 避开 herdsman 的 8080；gaea 客户端脚本四档+并发）：

| 配置 | 单路 decode | 并发 x2 | 备注 |
|---|---:|---:|---|
| 27B 裸解码 | 12.2~12.5 tok/s | 各 12.5/聚合 11.6 | 贴带宽墙（halogen 同模型串行 10.6 佐证）；prefill 620 t/s@2111tok 与官方 656 一致；TTFT 439ms |
| 27B + DFlash2（draft 1.1GB，hf-mirror 下载） | 13.7~21.2 tok/s | **双路各 44 / 聚合 34.9**（wall 4.2s） | README 所说"针对 Windows 编译器调优的 verify 内核"在并发下兑现 |

- 内存提交 32.3GB（模型+KV+draft）；`/v1/models` 标准形状；thinking 的 `chat_template_kwargs` 接受；API 支持 `--cache-disk`（重启安全 prompt cache）、`--sessions`、Responses API。
- 工具链盘点（未来自构建）：TheRock ROCm 10.0.0（gfx1151 tarball，仅构建机需要）+ vcpkg（icu/curl/openssl/libpng/libjpeg-turbo）+ VS Build Tools + CMake/Ninja；`tools/windows/check.ps1` 一键预检。
- **35B 旗舰数字（作者报 76~93 tok/s MTP）待兼容量化验证**：UD-Q8_K_XL（~40GB）或试 unsloth UD-Q4_K_XL——批次 2 项。
- **⚠️ 命名冲突待用户裁决**：`C:\AI\tianxuan` 已存在用户自己的项目「天璇 · Tianxuan」（DeepSeek Go 编程助手，wubitianxuan55-cell）——引擎命名需换名（如 tianxuan-engine / 天璇引擎归属确认）或用户拍板共用品牌系。

**定夺：Gufo 基座成立。** Windows 链路半天跑通且数据真实，批次 2 = 35B 兼容量化旗舰复测 + fork 仓库结构（upstream=gufo-org/gufo，Windows 分支基线=pixmaate）+ gaea 接入。

## 13. 批次 2 执行记录（2026-10-17 续）

### 13.1 fork 仓库结构（完成）

- **代码基**：`clones/gufo`（原地 git clone，不另复制）。分支布局＝
  `main`（`upstream/main` 只读镜像，只 fast-forward）／`windows`（`pixmaate/windows-port` @ `2eb0778`，
  即预构建 tag `windows-2026-10-03`）／`tianxuan`（集成分支，首个提交 `db86aabc`）。
- **remote**：`upstream` = gufo-org/gufo，`pixmaate` = pixmaate/gufo；`origin` 待 tianxuan 自己的
  GitHub 仓库建好后补（命令与同步流程写在台账里）。
- **合规面**：`LICENSE`（MIT）/`NOTICE`/`THIRD_PARTY_NOTICES.md` 一字未动；README 顶部加 fork
  声明（Gufo 名号与 Logo 留给原作者）；台账 = `clones/gufo` 的 tianxuan 台账（`tianxuan/README.md`）
  （定位 / 分支与同步 / 许可清单 / Windows 构建与发布 / 批次 / 已知坑）。

### 13.2 gaea 接入补漏（完成，`fa94b269`）

- **思考预算守护漏斗补 tianxuan**（`applyThinkingBudgetGuard`）：接入时只写了
  `prepareStreamRequest` 的装配分支，手装 ChatRequest 的短预算调用点（copilot 续写 256 /
  章节护栏 / wx_agent / provider bridge）在 27B 上会复现「推理吃光预算、正文只剩几个字」——
  与 strata 接入时同一个坑。现在两个漏斗入口（`ChatStream` / `chatOnce`）都守护。
- **小说防重复采样**（`samplerAppliesToEngine`）：gufo 的 DRY/XTC 通道未验，按未知引擎口径
  不注入非标键（与 strata 同处理），测试钉住「enabled=false 且请求零脏键」。
- **启动器** `scripts/tianxuan-serve.ps1`：27b / 35b 两档预设 → 8094，参数固化
  （`-j 2` / `-c 32768` / `--speculative dflash2|mtp` / `--cache-disk`），stdout/stderr 分落
  `.tmp/tianxuan/`；PS 5.1 把引擎 stderr 当 `NativeCommandError` 终止脚本的坑写进注释
  （实测踩过：引擎随包装脚本一起死）。
- **端口口径**：8094（8080 herdsman / 8091 strata / 8092 MoE4All / 8888 modelhub），免鉴权。

### 13.3 35B：不下载模型，直接用本机已有的那份（阻塞=代码，不是带宽）

**用户指令（2026-10-17 续）：本机已有很多模型，不重新下载；要做的是基于本设备开发自己的引擎。**
据此把「下载 UD-Q8_K_XL」整条撤掉（已删掉 1.56 GiB 前缀与下载器），改成本机已有模型直接验证。

- 本机 35B 在 herdsman 库：`Qwen3.6-35B-A3B-UD-Q4_K_M`（22.6 GB，arch `qwen35moe`，
  含 `ffn_gate_inh` 无、有 `ffn_gate_exps` / `ffn_gate_inp` / `ffn_gate_shexp` 共享专家布局）。
- **两只入口不一样（关键）**：`gufo bench -m <35B>` 走通用 Qwen 加载器，报
  `Missing required Qwen tensor: blk.0.ffn_gate.weight`（误导）；`gufo serve llm -m <35B>`
  才走 A3B 运行时，真报错是
  `Failed to create Qwen3.6-35B-A3B model: unsupported type Q6_K for output.weight (this build handles F32/Q8_0/BF16)`
  （证据：`.tmp/tianxuan/35b-serve-test.err.log`）。A3B 权重上传本身没问题：
  `a3b: weight upload 21.1 GiB in 10.6 s (1.99 GiB/s)`。
- **量化分布核实**（gguf 头逐张量）：该文件除 `output.weight`（Q6_K，2048×248320）外，
  所有非专家张量都是 F32/Q8_0/BF16 —— 也就是**只差一个张量的类型支持**就能跑，
  不需要下载任何东西。
- **第一批引擎代码改动（已编译通过，装载破关，解码待修）**：`fn::Q6KRowsToQ8_0` 公开化 +
  `Engine::UploadQ6KAsQ8_0`（output.weight Q6_K 与 3 层 Q6_K 专家栈装载期重编码）+
  `GetExpert` 直收 Q4_K/Q5_K/Q5_1 专家栈（MMQ 内核白名单 `MmqExpertType`，prefill WideMoe 同步
  分派）。实测链：21:15 撞 Q4_K 专家栈 → 加 `GetExpert`；21:19 撞 blk.34 Q6_K down → 专家栈重编码；
  21:20 **装载通过**（21.1GiB 1.7s 上传，MTP draft_limit=6）。
- **当前阻塞 = 解码异步非法访存**：首个 forward 即 `A3B decode: stream sync: unspecified launch
  failure`（AR/MTP 同炸；27B 同二进制正常）。静态排查全部界内（MMVQ 标量点积/fragment/分组分发/
  launch_bounds），唯一无本机实证的是 Q4_K/Q5_K 向量 MoE 内核本体（pixmaate 只真机验过 Q8_0/
  IQ4_XS）。已落 `A3B_MOE_SELFTEST=1` 自检钩子（逐内核发射+逐 sync+逐行 stderr 报告+完整 eager
  forward 收尾，结束拒服务），一次 GPU 运行即可点名炸点。
- **⚠️ 整机风险实录 → 根因已定（2026-10-17 深夜，重启×4 换来的结论）**：两段式自检实锤
  **MoE 内核无罪**（合成权重段 ~1.3GB 六步全过，含冻结同款 Q4_K gated n=1）；`=2` 真实段在上传
  完成后、任何 MoE 内核之前冻结且两轮冻结点不同 = 状态问题。体检实锤根因：**本机 VGM 缺省，
  专用显存区块仅 4GB**（Win32_VideoController.AdapterRAM=4293918720；注册表无
  DedicatedSegmentSize）——27B/35B 权重全躺 WDDM 共享页内存，35B 21GB+herdsman 15GB 共存触发
  工作集修剪→TDR→整机冻结；上传速度 13.8→2.0 GiB/s 为共享页旁证。**修复=AMD Software→
  Performance→Tuning→Variable Graphics Memory 调固定大值（≥32GB）后重启，再测 35B；
  调大前禁止任何 35B 整模型运行。** herdsman 的 27B 同样受益（同池）。

### 13.4 27B 调优实测（本轮主收获）

配置：Qwen3.8-27B-UD-Q4_K_M（15.33 GiB）+ DFlash2 draft（1.1 GB），Windows gfx1151，64 GB UMA。
两个口径：`gufo bench`（一次装载、合成提示词、r=2）与 `scripts/verify_engine_compare.go`（gaea 客户端 + 聊天页口径）。

| 指标 | 数值 | 备注 |
|---|---:|---|
| AR tg128 | 12.48 tok/s | 基线（无投机） |
| DFlash2 tg128（adaptive, d7） | **20.41** | 接受率 41.9%，**+63% vs AR** |
| DFlash2（fixed, d7） | 17.02 | 接受率 25.5%，固定块长明显更差，不要用 |
| DFlash2（adaptive, d4） | **22.44** | 接受率 47.6%，本轮最优 |
| DFlash2（adaptive, d10） | 21.30 | 接受率 41.9%，不涨 |
| pp512 / pp2048 / pp8192 | 354 / 568 / **589** | prefill 随长度走平（官方 656 的 90%） |
| tg @ d4096 / d16384 | 21.07 / 20.77 | 接受率 42.6% / 45.3%，**深上下文几乎不衰减（-1.4%）** |
| 服务端四档（client 口径） | 13.1~21.2 | 与 bench 同量级；并发 x2 聚合 **37.4**（双路 24.6 / 26.9，batch_width=2） |
| 磁盘续跑缓存 | TTFT 4244ms → **179ms** | 同提示词第二次：cached_tokens=29、restore 177ms |

结论（可直接执行）：

1. **投机解码是唯一的大杠杆**：DFlash2 把 decode 从 12.5 拉到 20+（+63%）；固定 draft 块长是负收益。
2. **draft 深度不是越深越好**：d4 反超 d7（bench +10%，服务端四档在噪声内），27B 默认建议 `-d 4`，
   按场景扫（列表/代码等重复结构接受率高，收益更大）。
3. **深上下文不衰减**（4K→16K 只掉 1.4%），gaea 长会话成本主要在 prefill；重复前缀靠磁盘缓存把
   TTFT 压到 179ms —— 长会话必须开 `--cache-disk`。
4. prefill 已接近官方（589 vs 656，90%），Windows 侧无需再调；真正待补的是**上游 2 周的性能提交**——
   我们的二进制停在 2026-10-03，上游 main 已到 0.17.0（含
   `perf(qwen-flash): cut prefill and decode attention cost at depth` 等）。

### 13.5 批次 3 队列（2026-10-17 晚状态）

1. **自构建（✅ 完成）**：TheRock ROCm 10.0.0 + vcpkg + 已有 MSVC/SDK/CMake/Ninja，`ninja` 出
   `build/release/gufo-server.exe`；每一处改动可本机验证。
2. **第一批 C++ 改动（装载✅ 解码❌）**：Q6_K 重编码 + `GetExpert` Q4_K/Q5_K 直收已编译实测——
   35B 装载通过，解码首个 forward 异步非法访存（详见 §13.3 末尾）；`A3B_MOE_SELFTEST=1`
   自检钩子已落，下一步=一次自检运行点名炸点→修内核→四档实测。
3. **出厂预设（✅ 写入启动器；35b 参数待实测校准）**：默认二进制改指自构建（预构建兜底、
   `-Binary` 覆盖）；27b=dflash2+`-d 4`+mmproj+2 sessions+`--cache-disk`；35b=mtp+survival+
   `--prompt-lookup`+1 session+mmproj-F16（视觉）+`--cache-disk`。
4. **下一批 C++（不变）**：统一 KV 池/会话 fork-join、UMA 单预算启动校验（#28 类，Windows WDDM
   预算）、gfx1151 启动自调优（按可用显存自动选 ctx/sessions）。
5. **判定（新增）**：Flash-Next 本机出局（68GB+ > 64GB UMA、无本地文件）；上游 184 未移植提交
   主体是 Flash-Next 内核与 cache 打磨，对 27B 无增量——windows 层移植暂不做。
