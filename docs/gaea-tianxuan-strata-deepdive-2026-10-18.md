# tianxuan × strata 深度调研：从「27B 专用」到「多模型引擎」

> 调研日期：2026-10-18。方法：本机源码/日志实读（`C:\AI\strata\repo`、`C:\AI\tianxuan`）＋
> GitHub 社区证据（Strata / Gufo / PolyStrata / AIMA / ROCmFPX）＋ 本机实测数据（gaea 台账）。
> 标注约定：**[实测]** 本机或引用方明确测过；**[源码]** 逐行读过；**[社区]** GitHub/文档公开报告；
> **[推断]** 由证据推导、待本机验证。

---

## 0. 一页结论

1. **27B dense 已经贴住带宽墙，继续调只能拿到 10–20%。**
   16.46 GB 权重 × 12.48 tok/s ≈ **205 GB/s**，本机 LPDDR5X 理论上限约 256 GB/s（约 80% 效率）；
   即使把内核压到极限，AR 上限也就 ~19 tok/s；现有 22.4 tok/s 已靠 DFlash2（每轮 ~1.8 token）把
   「等效带宽」抬到 360 GB/s 级。**要数量级的提升，只能换模型类别（MoE 小激活），不是继续磨 27B。**

2. **strata 的最大价值不是「另一个引擎」，而是一套已被本机验证的内存分层架构。**
   host 常驻专家 arena（33 GB pinned）＋ VRAM 专家槽（22,042/24,576，命中 98.8%）＋ CPU pool
   并行算 miss ＋ DXGI/WDDM 预算＋「装不下就拒绝」——68 GB 的 Flash-Next 在 64 GB carve 的
   Windows 上稳定跑到 31–52 tok/s。**[实测/源码]** 这正是 tianxuan 35B 线缺的那一层。

3. **本机 strata 0.1.45 二进制已经带 `qwen35moe` 引擎路径。**
   `strata.exe --families` → `qwen4exp qwen35moe`；本机 35B 文件头确认 `qwen35moe` + `nextn`（MTP）。
   也就是说：**不用改一行代码，P0 就能用 strata 的 q36 基准模式在本机验证 35B-A3B**（`--bench-tg/--bench-pp`）。
   这是当前性价比最高的一步，且能同时回答「Windows AMD 能不能跑 35B」与「能跑多快」。

4. **「35B 跑不动」不是模型或硬件问题，是 gufo 装载/解码路径的问题。**
   同芯片社区：gufo PR #299（Linux gfx1151）35B 解码 62/86/102 tok/s（1/2/4 并发）、prefill 1.75–2.5K；
   gufo Windows 文档 35B MTP 76/80/93 tok/s（Q8_K_XL、96 GB carve）；PolyStrata 在 8 GB 笔记本上
   UD-Q4_K_M 82–86 tok/s；AIMA 在 Windows 上用 BF16 35B 跑到 30.55 tok/s。**[社区]**
   tianxuan 的七次重启集中在 gufo A3B 的「整段设备驻留＋区域批量 H2D＋未验证内核段」，与 strata/AIMA
   的「host 驻留＋按需/分页＋成熟内核」是两种形态。

5. **建议路线：双引擎短期 + gufo 内 MoE 运行时中期。**
   27B 质量档保留（TTFT 30 ms 是真正的差异化）；35B-A3B 先用 strata q36 验证并可作为过渡后端；
   同时把 strata 的「专家分层＋预算拒绝超配＋GGUF 原地读＋家族注册」移植进 tianxuan，最终做到
   **一个 tianxuan、多个模型家族**，而不是一个模型一个特化运行时。

---

## 1. 本机基线（同一台 GMKtec EVO-X3：128 GB 物理 / 64 GB carve / gfx1151 / Windows 11）

| 引擎 / 档位 | 模型 | decode | prefill | 上下文 | 稳定性 | 备注 |
|---|---|---|---:|---:|---|---|
| **strata 0.1.45 HIP** | Flash-Next IQ2_XS（68 GB，两卷） | **31–52 tok/s** | 448–493 tok/s | 131K（KV 32K 常驻） | 稳（本机现役） | 专家槽 22,042/24,576，命中 98.8%；arena 33 GB；VRAM 槽 29.6 GB；15 pool workers |
| **tianxuan 27B 质量档** | Qwen3.8-27B UD-Q4_K_M（15.3 GB）+ DFlash2 | AR 12.48 / DFlash2 d4 **22.44**（服务内 19–22） | 589 tok/s @8192 | 32K | 稳 | 重复代码 52.6 tok/s；热 TTFT 27–30 ms；磁盘缓存 179 ms；提交 ~32 GB |
| **tianxuan 35B 极速档（外部）** | Qwen3.6-35B-A3B UD-Q4_K_M + llama.cpp Vulkan MTP | **59.5–62.1 tok/s** | — | 32K | 稳 | 512-token 长答 9.6–11.4 s（27B 为 29–37 s） |
| **gufo 上游 27B（Linux 128 GB，参考）** | Q4_K_XL + DFlash2 | AR 12.37；DFlash2 mixed 26.13；repetitive 63.12 | 656 tok/s | 262K | — | 与本机 27B 数字同量级，说明 Windows 侧不是主要差距 **[社区]** |

**读法**：27B 的 AR 12.4 是本机带宽决定的天花板附近；DFlash2 把「普通长答」抬到 22–26，把「重复内容」
抬到 52–63。strata 的 Flash-Next 用 2–3 bit 量化 + 专家分页，把 68 GB 模型塞进 64 GB carve，代价是
decode 31–52（仍远高于 27B 的等效 token 成本）。**两者不是替代关系，是不同档位。**

---

## 2. strata 源码级拆解：它到底做对了什么

### 2.1 家族注册：只跑「认识的」，并且「认识的」也未必跑

`include/strata/artifact/model_family.hpp`：只有两个家族——
`qwen4exp`（Qwen3.8-Flash-Next 及其微调/裁剪）与 `qwen35moe`（Qwen3.6-35B-A3B，同 tokenizer、
同 GDN+attention 3:1 层模式）。别的架构**明确拒绝**，不会「猜着跑」：

- `model_family_from_arch()` 把 `general.architecture` 映射到枚举；
- `model_family_runs()` 只对 `Qwen4Exp` 返回 true（q36 走自己的 `src/qwen35` 路径）；
- 拒绝时给 `model_family_not_supported` 的可读原因。

**要点**：strata 不是「通用引擎」，而是「**少数家族 + 每个家族一份几何/布局契约**」。
它在 `family_layout.hpp` 里为每个家族定义：从 GGUF 元数据读几何、逐层张量布局表（名字+形状）、
以及 `compiled_geometry_mismatch`（与内核编译期形状逐字段比对）。**布局表完整**的家族（qwen35moe）
连「多一个表外张量」都算错。这是「多模型」能稳的前提：先有契约，再有内核。

### 2.2 qwen36 独立引擎路径：GGUF 原地读 + 混合专家放置

`src/qwen35/q36_engine.cu`（~4,000 行）是 strata 给 35B-A3B 的完整引擎路径，`generate.cpp:2253`
在创建任何引擎状态之前按文件头切换过去（`strata` 直接进 q36；另有 `strata-q36` 独立入口）。

它的装载形态与 gufo 相反：

- **不建 pack、不建 expert profile、不单独 MTP 模型**：GGUF 原地读（映射文件保持打开），
  几何从元数据算，MTP 用文件自带的 `nextn` 块。
- **专家三层放置**（`place_experts`）：
  1. 全部装得下 → 全部进 VRAM（每层 blob，`kMmqTail` 对齐）；
  2. 装不下 → **hybrid**：`cudaHostAlloc` 一块 **host arena 装下每个专家 blob**（pinned），
     VRAM 只放 `S` 个/层 的 slot（`--expert-slots`，最大 256/层；`--expert-vram` 给 GiB 预算）；
  3. 解码时 `k_group_hy` 只对「有 slot」的专家分组走 GPU，miss 的进 CPU pool；
     `k_merge_cpu` 把 CPU 结果并回各行；prompt 批走 staging（`--hybrid-rows` 控制行数）。
- **显式拒绝**：slot 数、staging、上下文、会话放不下时打印「需要多少 / 剩多少」并退出，
  不会先跑起来再在 token 4000 处 OOM。

**这解释了本机为什么 strata 稳、gufo 冻**：gufo A3B 是「每个映射区一次 `hipMalloc` ＋
`hipMemcpyAsync` 整段上传」（`src/models/qwen36_a3b/engine.cpp:74/100/141`），
峰值是 21 GB 级别的区域驻留；strata 的专家流量天然被「slot 命中 + 分页 + CPU pool」节流，
VRAM 常驻只有一个可控的槽数组。**[源码对照]**

### 2.3 专家缓存不是「LRU」，是「按路由频率的静态画像 + 运行时热度」

`include/strata/core/expert_cache.hpp` 顶部把账算得很清楚：CPU pool 是最大单项成本
（663.6 MB/ token × ~40 GB/s ≈ 16.2 ms，占 ~53 ms/token 的 30%；流水只掩盖 1 ms），
所以要把「最常路由的专家」放进 VRAM，让 CPU 只算 miss。

- `expert-profile.bin`：`STRP` 头 + 按路由频率降序的 `(layer, expert)` 对 + 频率表；
  `make_profile.py` 离线生成，`--expert-profile` 加载；`--expert-profile-save` 把本次学到的热度写回。
- 运行时自适应 tier 按 `heat` 换入换出；启动从 profile 预填（本机日志：`pre-filled 22042 of 22042 slots
  from the profile in 3.1 s`）。
- 关键指标 `h`（命中率）被当作一等公民度量，而不是「缓存越大越好」。

### 2.4 内存计划与 Windows/UMA 预算：先算账，再分配

- `device_free_bytes()`：统一内存系统上不数 `MemFree`，而是 `MemAvailable - 6 GiB`
  （`STRATA_UMA_HEADROOM_GIB` 可调）；否则文件页缓存会被误判成「可用」。
- Windows 上另有 **DXGI `QueryVideoMemoryInfo` 取进程的 WDDM shared budget**（`src/platform/memory.cpp`），
  避免 `hipMemGetInfo` 在 carve 之外「每分配 1 GiB 免费量掉 2 GiB」的双计。
- host arena 的 pin 有上限：`STRATA_ARENA_PIN_GIB=-2`（auto）= **不超过 GPU 的 shared budget**；
  Windows 无 `SeLockMemoryPrivilege` 时退化 4 KB 页（本机日志实锤：`large pages refused ... using 4 KB pages`）。
- `plan/plan.hpp` 是老组件的「算术计划」：KV + 不可驱逐的 GDN 状态 + 专家槽共享一个 VRAM 池，
  不闭合就抛 `DoesNotClose`。理念贯穿全引擎：**拒绝，而不是超配**。

### 2.5 KV 流式 + int8：长上下文不靠「全量常驻」

本机 strata 配置：`--max-context 131072 --kv int8 --kv-resident 32768`；日志显示
`KV streaming: 32768 of 131072 cells per QSA layer in VRAM, the K/V in 1.55 GiB of pinned RAM`，
运行中 `98–99% block reads hit VRAM`。效果：131K 上下文只占 ~1.55 GiB 常驻 + 流式补读，
给专家槽让出空间。**[实测/源码]**

### 2.6 投机解码、批处理、会话缓存

- 投机：MTP draft（最多 3）+ prompt lookup（重复片段，最多 5）+ suffix drafter；
  `--spec 4`、`--spec-min-p 0.5`、`--mtp-window 8192`、`--lookup-chain 3`；verify 窗口
  6/4/3/2/1 token 各自捕获图；接受率与草稿成本共同决定是否发草稿。
- 批处理：`--batch N`（最多 8）把多会话放在同一窗口解码，共享 dense/路由专家；
  QWEN36 文档实测 1/2/4/8 并发 = 83.9/173.5/252.0/325.4 tok/s（对 llama.cpp 1.5–1.6×）。
- 会话：磁盘/内存 conversation cache + 快照 + 前缀复用；`--prompt-cache`、`--cache-disk` 一类。
- 视觉：q36 用 Unsloth `mmproj-F16`，M-RoPE 三维位置，与 llama.cpp 对齐（KL 数量级 1e-3）。

### 2.7 gfx1151 出厂开关与 hipBLASLt 调优表

`docs/STRIX_HALO.md` 与 `src/core/arch_defaults.*`：启动检测到 gfx1151 后默认打开 **18 个**
「同输出、只提速」开关（GDN 四件套、HCD_EXACT、CVEC_FUSE、Q8/Q6 packed、MMVF_ROWS、
ATTN_LANECELL、EXPERT_V2、TSUM、LFUSE、GDN_SPLIT、QFUSE、PLE_BATCH、SH_STREAM），
每一条都在文档里注明「byte-identical」的验证方式。另有**按 arch + hipBLASLt 版本命名的
prompt GEMM 调优表**（`tools/hip/gfx1151-hipblaslt-*.txt`），启动时校验第二行的
`gfx1151 <version>` 才启用，日志会打印 `prefill gemm: hipBLASLt tuning enabled (N rows, gfx1151, version ...)`。

对比：gufo 只有 `docs/WINDOWS.md` 里的一组 Windows decode 开关（copy_kernels、flag_waits、
fast_sampling、hot_first_upload 等，实测全开比全关快 7.6–9.5%），**没有随包发布 hipBLASLt 调优表**。
这是 prefill 侧一个现成的、低风险的移植点。

---

## 3. tianxuan 现状：引擎能力 ≠ 产品暴露

### 3.1 引擎其实有 8 个家族，产品只暴露 1 档

gufo 代码树 `src/models/` 已有 8 个模型运行时：`qwen`（27B dense）、`qwen36_a3b`、
`qwen38_flash_next`、`deepseek_v4_flash`、`minimax_h3`、`qwen3_asr`、`qwen3_tts`、`qwen_image_21`。
35B-A3B 甚至有完整的 Windows 运行时（`engine.cpp` 98 KB）和文档（`docs/models/qwen3.6-35b-a3b/README.md`，
写明 Windows gfx1151 上 AR 53–54、MTP 76/80/93 tok/s、prefill 1.5–1.8K，Q8_K_XL、96 GB carve）。

但 tianxuan 产品层（`internal/app/tianxuan_proc.go`）只有两个硬编码档：
`quality`（27B + DFlash2）与 `fast`（35B-A3B + **llama.cpp Vulkan**，绕开 HIP）。
模型路径、参数、`--served-model-name` 全是常量；启动后只把「实际 `/v1/models` 的第一个 id」
写回引擎默认模型。**「只适用于 27B」首先是产品层的一档化，其次才是引擎层的 Windows 稳定性。**

### 3.2 35B HIP 七次重启的证据链（摘要）

台账（`docs/gaea-tianxuan-research-2026-10.md` §6、§7、§8）与 gufo 仓 `docs/tianxuan/README.md`：

1. 装载期冻结：最初怀疑 22 GB 区域 `hipMalloc`/整段 SDMA；
   阶梯探针 4→22 GiB 单笔分配全过、逐张量上传仍冻 → **单笔分配与拷贝模式都不是根因**。
2. 自检：合成权重段六步全过（MoE 内核无罪）；真实权重段在「上传完成后、任何 MoE 内核之前」
   就冻，冻结点漂移 → 状态/驱动域问题。
3. 事件日志：5 次 `Bugcheck 0x119 VIDEO_SCHEDULER_INTERNAL_ERROR`，P1=0x10（引擎切换后驱动报告
   意外中断）× 同签名；WHEA 零记录 → 应用层 HIP 代码很难直接触发，社区对应 gufo #599
   （Strix Halo 加载期整机复位，嫌疑 AGESA/固件）。
4. 后期把嫌疑夹到 `hipcub MtpCandidateTileKernel`（35B 专属、上游未在 Windows 验证），
   改 CPU top-64 + 零拷贝专家驻留，首请求有响应后**第 7 次重启**；用户拍板终止 35B 线。

**能下的结论**：不是「35B 太大」——同机 strata 稳跑 68 GB，AIMA 在 Windows 跑 BF16 35B 30.55 tok/s，
gufo 自己在 96 GB carve 的 Windows 上跑 35B MTP 76–93 tok/s。**是「整段设备驻留 + 21 GB 级区域搬运 +
未验证的专属内核段」这条组合路径，在这台机器的固件/驱动上不可靠。** 换架构（host 分层）比继续
修内核更符合证据。

### 3.3 27B 的带宽墙（为什么「不只 27B」是架构问题）

| 量 | 值 | 说明 |
|---|---:|---|
| 27B Q4_K_M 权重 | 16.46 GB | 每 token 至少读一遍 |
| 实测 AR | 12.48 tok/s | 16.46 × 12.48 = **205 GB/s** |
| 本机 LPDDR5X 理论 | ~256 GB/s | 实测有效上限通常 80–90% |
| AR 理论上限 | ~19 tok/s | 即使 100% 带宽、零开销 |
| DFlash2 有效 | 22.4 tok/s（mixed） | 每轮 ~1.8 token，等效 369 GB/s |
| 重复内容 | 52.6 tok/s | lookup 命中时 |

→ 27B 的「提速空间」只剩：更好的草稿（+10–20%）、更低 bit（Q3/IQ3，-20% 字节但质量下降）、
更好的 prefill（对长会话有意义，对 decode 无意义）。**要 2–4 倍，必须换成 3B 激活的 MoE。**

---

## 4. 社区 / GitHub 证据

### 4.1 Gufo 侧

| 来源 | 事实 | 对 tianxuan 的意义 |
|---|---|---|
| [PR #299](https://github.com/gufo-org/gufo/pull/299)（slimsami，draft） | 给 Qwen 路径加 `qwen35moe`：16q/2kv head_dim 256 编译期特化（否则 ≥16K 退化/崩）、专家复用 Flash-Next routed F16 WMMA GEMM（Q8_0/Q6_K + paired gate/up）、BF16 分组 GEMM、DFlash2 最多 16 taps；gfx1151 实测 prefill 1752/2357/2521/2386（0.5K/1.6K/6.4K/16K），decode 62/86/102（1/2/4）；**已知问题：35B 投机 greedy 与 AR 非逐位一致；MTP 未接 serve** | 35B 在 gfx1151 上「能跑且不慢」的公开证据；其「按格式分派、去掉 prefill MMQ」是移植参考 |
| [Issue #399](https://github.com/gufo-org/gufo/issues/399) + [NinjaPear fork](https://github.com/NinjaPear/gufo-Qwen3.6-35B-A3B-Q6dense) | Q6dense（把专家重新打包成稠密张量）在 Strix Halo 上 **pp 3,095 / tg 190.67 / 8 并发 334** | 另一条「绕开 MoE 内核」的模型侧路线；代价是权重变大、失去 MoE 稀疏性 |
| [Issue #254](https://github.com/gufo-org/gufo/issues/254) | 上游没有 Windows 机器；社区自移植可用，性能方差 10–20%，direct IO 替代 mmap 省 RAM | Windows 是社区补的，不是上游一等目标；tianxuan 的生态位仍在 |
| [Issue #634](https://github.com/gufo-org/gufo/issues/634) | 前缀复用会清空投机接受率历史（DFlash 与 MTP 都有） | tianxuan 已修（批次 4），说明社区问题可直接变产品优势 |
| [Issue #599](https://github.com/gufo-org/gufo/issues/599) | Strix Halo 加载期整机复位（Linux dmesg `data fabric sync flood`），AGESA/固件嫌疑 | 与本机 0x119 同一类「平台级」风险，不能只靠改内核 |
| `docs/WINDOWS.md` / `docs/models/qwen3.6-35b-a3b/README.md` | Windows 35B：AR 53–54，MTP 76/80/93，prefill 1.5–1.8K（Q8_K_XL、96 GB carve）；64 GB carve 跑 Flash-Next 需 ~2 min 装载且 decode 变慢 | 证明 gufo 的 35B 代码本身可用；本机 64 GB carve 是变量之一 |
| [web2bruno/gufo](https://github.com/web2bruno/gufo)（`mtp-priming-qwen36-35b`） | 把 MTP 接进 serve、批量 prime；真实散文深度：4K 57.5 / 16K 50.9 / 65K 35.2 / 131K 26.3 / 256K 16.1 tok/s，接受率 0.74–0.86；`-d 1` 最优 | 35B 的生产形态（长上下文）已被社区跑通；也说明「草稿深度 1」比 2/6 更稳 |

### 4.2 Strata 侧

| 来源 | 事实 | 意义 |
|---|---|---|
| [Strata repo](https://github.com/Niko1221/Strata) / 本地 `C:\AI\strata\repo` | MIT；0.1.45；416 open issues，日更 | 可合法参考/移植；上游极活跃 |
| `docs/STRIX_HALO.md` | gfx1151 专项：UMA 预算、gfx11 WMMA、hipBLASLt 表、18 个默认开关；Windows zip 带 gfx1151 代码「官方未测过 Windows Strix Halo」 | 本机就是它的空白实测场；我们的数据是稀缺样本 |
| `docs/QWEN36.md` | 35B-A3B 独立路径：hybrid 专家、8 并发、MTP、视觉、chunk 512→4096；R9700 32GB UD-Q4_K_XL 优化后 **decode 117.8 / pp4096 6220**；8 并发 386.9 | 证明「qwen35moe + 专家分层」在 RDNA 上能到 100+ tok/s |
| [Issue #2186](https://github.com/Niko1221/Strata/issues/2186) | 作者维护 PolyStrata：保留 Strata 不变，另加第二个引擎跑 Qwen3.6/Ornith/Gemma4/Tiel/GLM | 社区对「strata 只跑一个模型」的正式回应 |
| [Issue #2894](https://github.com/Niko1221/Strata/issues/2894) | Windows 预构建（CUDA sm_120）拒绝 35B，提示 `--build` | 预构建能力因后端而异；本机 HIP 二进制不同（见 §5） |
| Issues [#2947](https://github.com/Niko1221/Strata/issues/2947) / [#2960](https://github.com/Niko1221/Strata/issues/2960) / [#2871](https://github.com/Niko1221/Strata/issues/2871) | Windows HIP：hybrid CPU pool 0xC0000005、KV streaming 偶发错答、prefill cap | **警告**：strata 的 Windows HIP 也不是无 bug；移植/借用必须带 parity gate |
| [#2840](https://github.com/Niko1221/Strata/issues/2840) / [#2693](https://github.com/Niko1221/Strata/issues/2693) | q36 路径还没有会话缓存 / KV 量化 | tianxuan 若移植，这两块要自己做（gufo 已有 disk cache） |

### 4.3 同芯片其他引擎

| 项目 | 关键数字 | 借鉴点 |
|---|---|---|
| [PolyStrata](https://github.com/VecSzn/PolyStrata)（MIT） | 9 个模型、2 个引擎：`strata`（Flash-Next 原样）+ `strata-poly`（GGUF 原地读，4 个架构）；8 GB 笔记本 35B UD-Q4_K_M **82–86 tok/s / pp 1560**；32 GB 5090 **395 tok/s**；模型切换 8–16 s | **最完整的「多模型引擎」蓝图**：`Engine` 基类 + 架构 hook（load_geometry/load_layer/route/moe_shared/block_attend/saves…）；进程级切换（停子进程→起新进程）释放 VRAM/ pinned/映射 |
| [AIMA AMD395 35B Windows](https://github.com/Approaching-AI/AIMA-AMD395-Qwen36-35B-Windows-Engine)（Apache-2.0） | Windows 11 + gfx1151 + **BF16 35B**：8K prefill 2,126 / decode 30.55；131K 冷 TTFT 108.6 s，前缀复用 8.59 s；batch 1、常驻 provider、COW 前缀、Rust 服务层、429/503 | Windows 原生可行性的硬证据；服务层（队列/过载/生命周期）设计可直接照抄 |
| [ROCmFPX Windows](https://github.com/daimonionnn/amd-rocmfpx-for-win) | llama.cpp 的 Windows ROCm 构建；27B 上量化与内核的细账（`t/s × GiB` 常数、IOMMU 关闭在 32K/128K prefill 提升 10–40%） | 说明「同一 HIP 栈」也能稳；IOMMU/BIOS 是免费的 prefill 杠杆 |
| gufo PR #299 / PolyStrata 的对比 | 都证明「35B-A3B 在 8–32 GB 卡上 60–100 tok/s」是常态，不是奇迹 | tianxuan 目标应设为「Windows HIP ≥40 tok/s」而不是「能跑」 |

---

## 5. 差距矩阵：strata vs tianxuan（gufo）

| 能力 | strata | tianxuan 现状 | 结论 |
|---|---|---|---|
| 模型家族 | 2 个家族 + 布局/几何契约；未知拒绝 | 8 个模型运行时，各自特化；无统一家族契约 | **需要家族注册层**（先契约后内核） |
| 装载形态 | GGUF 原地读（q36）；pack 仅 Flash-Next | 每个映射区一次 `hipMalloc` + 整段 `hipMemcpyAsync` | **改为 host 常驻/按张量，避免 21 GB 区域驻留** |
| 专家分层 | VRAM 槽 + host blob + CPU pool + staging | 无（全有或全无） | **35B/大 MoE 的核心缺口** |
| 内存计划 | UMA `MemAvailable-6GiB` / DXGI WDDM budget / 拒绝超配 | DXGI budget 有；无专家槽计划、无拒绝 | 补「预算→槽数→上下文」算术 |
| KV | int8/q4 + 流式（resident window） | f16 KV，全量常驻 | 长上下文/释放 VRAM 的主要杠杆 |
| 投机 | MTP + lookup + suffix + verify 窗口 1–6 | DFlash2/MTP + lookup；已修续写策略 | 已接近；可补 suffix/窗口自适应 |
| 批处理 | 最多 8 slot，窗口共享 | `-j` 会话（2） | 并发吞吐 1.3–1.6× 的空间 |
| 会话缓存 | 磁盘/内存 + 快照 + 前缀复用 | `--cache-disk` + 前缀复用 | 基本对齐 |
| Windows 调优 | 18 个 gfx1151 默认开关 + hipBLASLt 表 | Windows decode 开关一组；无 hipBLASLt 表 | **补表是低风险 prefill 收益** |
| 模型切换 | 服务层 registry + `/api/models/load|unload` + 进程级切换 | 硬编码两档 + 重启引擎 | 产品层能力缺口 |
| 质量门槛 | bit-exact / KL / profile 命中率 | speculative_verification_test / dflash GPU test / 输出哈希 | 已有基础，移植时继续用 |
| 许可 | MIT | MIT（gufo fork） | 可互借，保留 NOTICE |

---

## 6. 「不只 27B」的模型矩阵与容量算术（64 GB carve / 128 GB RAM）

| 档位 | 模型/量化 | 文件 | 每 token 读量（约） | 本机可行性 | 预期 decode | 角色 |
|---|---|---|---:|---|---:|---|
| 质量 | Qwen3.8-27B UD-Q4_K_M | 15.3 GB | 16.5 GB | 已验证 | 12.5 AR / 22 DFlash | 日常质量档（现役） |
| 质量+ | Qwen3.8-27B UD-Q8_K_XL | 29.3 GB | 31 GB | 可装（≤64 carve） | 7.2 AR / 16 DFlash | 高精度，慢 |
| **极速** | **Qwen3.6-35B-A3B UD-Q4_K_M** | **21.1 GB** | **~2 GB（3B 激活）** | **P0 待验证**（strata q36） | **目标 40–90** | **首选「不止 27B」档** |
| 极速+ | 35B-A3B UD-Q6/Q8 | 32–38 GB | ~4–7 GB | 可装 | 50–70（社区） | 质量更好、更慢 |
| 长文/旗舰 | Flash-Next IQ2_XS | 68 GB（含 29 GB PLE 表） | 专家分页 | strata 已验证；gufo 需改装载 | 31–52 | 131K 长上下文档 |
| 超纲 | DeepSeek V4 Flash | 91 GB | — | 64 GB carve 不现实（strata/Poly 需 ~120 GB） | — | 排除 |

**结论**：本机真正缺的不是「更大的模型」，而是 **21–38 GB 的 35B-A3B 这一档**——它比 27B 快 3–4 倍、
质量在同级、文件已在盘上。Flash-Next 68 GB 是「长文旗舰」但不是日常档（装载/分页成本高）。

---

## 7. 可移植清单与路线图

### P0（0.5–1 天，零代码）：用 strata q36 验证本机 35B-A3B

本机 strata 0.1.45 已含 qwen35moe 路径（`--families` 实测），本机 35B 文件头为 `qwen35moe` + `nextn`。
用它的内置基准模式，**强制 hybrid（`--expert-slots 64` 先只放 25% 专家进 VRAM）**，把风险控制在
「host 常驻 + 小 VRAM 槽」这一已被本机 68 GB 跑通的形态：

```powershell
$env:PATH = "C:\AI\strata\repo\engine\rocm\bin;" + $env:PATH
$M = "C:\Users\wubi\.herdsman\models\Qwen3.6-35B-A3B-MTP\windows\amd64\amd\Qwen3.6-35B-A3B-UD-Q4_K_M.gguf"

# 1) 先看它是否接受这个文件（几何/类型/布局），不生成
C:\AI\strata\repo\engine\strata.exe --model $M --max-context 16384 --expert-slots 64 --bench-pp 512

# 2) prompt / decode 基准（hybrid，64/层）
C:\AI\strata\repo\engine\strata.exe --model $M --max-context 32768 --expert-slots 64 --cpu-threads 16 --bench-pp 4096
C:\AI\strata\repo\engine\strata.exe --model $M --max-context 32768 --expert-slots 64 --cpu-threads 16 --bench-tg 128

# 3) 通过后再加槽：128/层 → 256/层（全驻留），观察是否复现冻结
```

**通过标准**：装载不冻；`--bench-tg` ≥ 40 tok/s（hybrid 64/层）、≥ 60（128/层）；
`--bench-pp 4096` ≥ 800 tok/s；连续 3 次启停无异常。
**失败处置**：停在 64/层 的 hybrid；若仍冻，说明本机固件对「35B 级 HIP 负载」本身有问题，
35B 继续走 Vulkan（已有 60 tok/s），tianxuan 只做产品层集成。
**风险控制**：先保存工作、关掉其它引擎（设备互斥）、一次只跑一个、观察 5 分钟；
出现整机异常按台账纪律不再重试，直接走 Vulkan 兜底。

> 如果 P0 通过：tianxuan 立刻多一个「35B-A3B · HIP/strata」档（外部引擎，和 herdsman 同模式），
> 同时拿到「strata q36 在本机 Windows 的真实数字」，为 P2 移植定基线。

### P1（2–4 天）：产品层多模型，不碰内核

- `tianxuan_proc.go` 从「两个常量档」改成 **profile 表**：`quality-27b` / `fast-35b-vulkan` /
  `fast-35b-hip`（P0 通过后）/ `long-flashnext`（P2 后）。
- 每个 profile 声明：二进制、模型/草稿/mmproj、arch（从 GGUF 头读）、上下文、KV 类型、
  投机模式、内存预估、能力（vision/thinking/tools）。
- 启动前预检：文件存在、arch 是否被该二进制支持（`--families` 或 `gufo diagnose`）、
  预计 VRAM/RAM（DXGI budget + `MemAvailable`），不通过就**拒绝启动**并给出数字。
- 启动后把实际 `/v1/models` 的 id 写回引擎默认模型（已有逻辑，扩展为多模型列表）。
- gaea 模型中心显示「档位 + 模型 + 上下文 + 预估内存」，停止/切换仍走 8094 生命周期。

### P2（2–4 周）：把 strata 的内存分层搬进 tianxuan（真正「不只 27B」）

目标：在 tianxuan 内新增一个 **MoE 家族运行时**（先 `qwen35moe`），不重写 27B/Flash-Next 路径。

1. **家族注册**：`ModelFamily`（arch 字符串 → 几何 + 布局表 + 支持的量化类型 + 引擎入口），
   未知 arch 明确拒绝；`qwen35moe` 复用 strata 的 `family_layout` 表（MIT，保留版权头）。
2. **装载**：GGUF 原地读；dense/attention/router 进 VRAM；**专家 blob 进 pinned host arena**；
   VRAM 只放 `S` 个/层 slot（profile 排序，运行期 heat 自适应）。
3. **解码**：slot 命中走 GPU 分组内核（可复用 gufo 的 routed WMMA/MMQ），miss 走 CPU pool
   （AVX2/AVX512，i-quant 用 ggml，K-quant 用自写）；`k_merge` 合并；prompt 批走 staging。
4. **内存计划**：DXGI budget → 扣 dense/KV/state/staging → 剩余给槽；算不闭合就拒绝
   （把 strata 的 `DoesNotClose` 语义搬过来）。
5. **KV**：先 f16（gufo 现状），P3 再加 int8/流式。
6. **质量门**：greedy 与 AR 逐 token 一致（或 KL 门限）；同 seed 复现；与 llama.cpp 对照。

**复用策略**：优先「vendor strata q36 的引擎层」（`src/qwen35/q36_engine.cu` + 其依赖的
kernels/MMQ/CPU pool）作为 tianxuan 的 `moe` 后端，而不是逐行重写；gufo 的 27B/Flash-Next
保持原样。这样风险最低，也符合 MIT。

**替代方案（如果 P2 超预算）**：长期把 strata（或 PolyStrata 的 strata-poly）作为 tianxuan 的
「MoE 子引擎」子进程，tianxuan 只做 registry/生命周期/协议适配——即 PolyStrata 的「两引擎」形态。
代价：多一个运行时依赖；收益：立刻获得多家族 + 专家分层 + 成熟的 Windows HIP 路径。

### P3（各 1–2 周）：把 strata 的「免费提速」逐项搬过来

- **hipBLASLt 调优表**：按本机 hipBLASLt 版本生成/移植 gfx1151 表（prefill GEMM 命中率提升，
  strata 文档显示其对 prompt 路径是明确收益；gufo 目前没有）。
- **KV int8 + 流式**：27B 的 32K→131K、35B 的长上下文；释放 VRAM 给专家槽/草稿。
- **批处理 slots**：`-j` 从「多会话」升级为「窗口共享专家」；并发吞吐 1.3–1.6×。
- **suffix drafter / 窗口自适应**：对重复结构（agent 工具 JSON、代码编辑）补足 DFlash2 之外的草稿源。
- **pinned 大页 / pin-by-need**：Windows 无 `SeLockMemoryPrivilege` 时退化 4 KB 页；
  可按需 pin（strata `STRATA_PIN_BY_NEED` 类似思路）减少常驻。

---

## 8. 验收指标（建议写进 issue/看板）

| 档位 | decode（512-token mixed） | prefill | 上下文 | 稳定性 | 备注 |
|---|---:|---:|---|---|---|
| 27B 质量 | ≥22 tok/s（保持） | ≥550 @8K | 32K | 24h soak | 热 TTFT <50 ms，磁盘缓存 <300 ms |
| 35B-A3B HIP | **≥40**（hybrid） / **≥60**（全驻留） | ≥800 @4K | 32K | 24h + 10 次启停 | 与 Vulkan 档输出同质量；不冻结 |
| Flash-Next 长文 | ≥25 | ≥350 | 131K | 8h | 提交 ≤60 GB；KV 流式命中 ≥95% |
| 产品层 | — | — | — | 切换 ≤16 s | 启停/切换/预检全部有明确原因码 |

---

## 9. 许可与合规

- **Strata：MIT**（`Copyright (c) 2026 Niko1221 and the Strata contributors`）。
  可复制、修改、再分发，只需在副本/实质部分保留版权与许可声明。移植 q36/专家槽/预算代码时，
  在 tianxuan 的 `NOTICE` / `THIRD_PARTY_NOTICES.md` 增加「Portions derived from Strata (MIT)」。
- **PolyStrata：MIT**（作者声明）；其 `strata-poly` 架构思想可参考；代码若借用同样加声明。
- **Gufo：MIT**；tianxuan 已是 fork，保留 `LICENSE`/`NOTICE`。
- **AIMA：Apache-2.0**；思想/服务层设计可参考，代码借用需保留 NOTICE。
- **halogen：source-available，禁止再分发修改版**——只可参考思想，不复制代码（既有台账结论不变）。
- 模型权重各自许可（Qwen Apache-2.0 等），不随引擎分发。

---

## 10. 待用户拍板

1. **是否执行 P0**（strata q36 在本机跑 35B，hybrid 64/层起步）。这是唯一能把「35B HIP 可行/不可行」
   从推断变成实测的动作；有整机风险，需你同意后我再跑。
2. **主路线**：A) 继续 gufo 单栈，把 strata 的内存分层移植进 tianxuan（推荐，最终一个引擎多家族）；
   B) 双引擎长期化（27B 用 gufo，35B/大 MoE 用 strata/strata-poly 子进程，PolyStrata 形态）；
   C) 以 strata 为主基座重做（不推荐：27B dense/TTFT 优势要重做，且 strata 不支持 27B）。
3. **「不只 27B」的首批范围**：只做 35B-A3B（最快、文件已在盘上），还是同时把 Flash-Next 68 GB
   作为长文档档（需要 P2/P3 的 KV 流式 + 装载改造）。

---

## 11. 附录

### 11.1 本机证据

- strata 源码：`C:\AI\strata\repo`（0.1.45，`engine/BUILD.json`）。
- strata 运行日志：`C:\AI\strata\repo\strata-iq2_xs.log`（专家槽 22,042、命中 98.8%、arena 33 GB、KV streaming 32K/131K）。
- strata 家族探测：`strata.exe --families` → `qwen4exp qwen35moe`（本机 2026-10-18 实测）。
- 本机 35B 文件：`C:\Users\wubi\.herdsman\models\Qwen3.6-35B-A3B-MTP\windows\amd64\amd\Qwen3.6-35B-A3B-UD-Q4_K_M.gguf`
  （21.11 GB，头含 `qwen35moe` / `nextn`）。
- tianxuan 独立工作区：`C:\AI\tianxuan`（`tianxuan` 分支，35B A3B 补丁封存于 973a1063…9e9b8e38）。
- gaea 台账：`docs/gaea-tianxuan-research-2026-10.md` §6–§14；产品层：`internal/app/tianxuan_proc.go`。

### 11.2 关键源码位置（strata）

| 主题 | 文件 |
|---|---|
| 家族枚举/拒绝 | `include/strata/artifact/model_family.hpp` |
| 几何/布局/编译期形状检查 | `include/strata/core/family_layout.hpp` |
| q36 引擎（hybrid、专家、serve） | `src/qwen35/q36_engine.cu` |
| 家族切换 | `src/program/generate.cpp`（~2253） |
| 专家槽/画像/自适应 | `include/strata/core/expert_cache.hpp` |
| UMA/DXGI 预算 | `src/platform/memory.cpp` |
| pinned arena / pin-by-need | `include/strata/core/pinned.hpp`、`src/core/pinned.cu` |
| gfx1151 默认开关 | `src/core/arch_defaults.*`、`docs/STRIX_HALO.md` |
| 内存计划（算术/拒绝） | `include/strata/plan/plan.hpp` |

### 11.3 链接

- Strata：https://github.com/Niko1221/Strata ｜ 0.1.45：https://github.com/Niko1221/Strata/releases/tag/v0.1.45
- Strata 35B 文档：`C:\AI\strata\repo\docs\QWEN36.md` ｜ Strix Halo：`docs/STRIX_HALO.md`
- PolyStrata：https://github.com/VecSzn/PolyStrata ｜ 后端/注册表：`docs/BACKENDS.md` ｜ FAQ：`docs/FAQ.md`
- Gufo PR #299：https://github.com/gufo-org/gufo/pull/299
- Gufo #399：https://github.com/gufo-org/gufo/issues/399 ｜ #254：https://github.com/gufo-org/gufo/issues/254
- Gufo #634：https://github.com/gufo-org/gufo/issues/634 ｜ #599：https://github.com/gufo-org/gufo/issues/599
- NinjaPear Q6dense：https://github.com/NinjaPear/gufo-Qwen3.6-35B-A3B-Q6dense
- web2bruno MTP 分支：https://github.com/web2bruno/gufo（`mtp-priming-qwen36-35b`）
- AIMA Windows 35B：https://github.com/Approaching-AI/AIMA-AMD395-Qwen36-35B-Windows-Engine
- ROCmFPX Windows：https://github.com/daimonionnn/amd-rocmfpx-for-win
- Strata issues：#2186 / #2894 / #2947 / #2960 / #2871 / #2840 / #2693

---

## 12. 一句话给决策者

**27B 已经把这块芯片的 dense 带宽吃满；tianxuan 要「不只 27B」，最短路径不是再调 27B，
而是把 strata 已经在本机跑通的「host 专家 + VRAM 槽 + CPU pool + 预算拒绝」搬进 tianxuan，
先让 35B-A3B 在 HIP 上到 40–60 tok/s（Vulkan 已有 60，但会丢掉 TTFT/缓存优势），
再把 Flash-Next 68 GB 作为长文档档。P0 用现成的 strata q36 基准就能验证，零代码。**
