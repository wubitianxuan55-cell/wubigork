# CosyVoice2 本地 TTS 提速记录（2026-08-09）

> 结论先行：CosyVoice2-0.5B 的 **LLM 环节从 PyTorch CPU 切到 llama.cpp GGUF + Vulkan 核显**，
> 整条合成管线比最初快约 **8–10 倍**（短句 6.5s→~1.5s，长句 24.5s→~2.8s）。
> 默认使用 **f16 GGUF**（音质最接近原始权重），q8 作为更快备选。

## 1. 硬件与运行环境

- CPU：AMD Ryzen AI MAX+ 395（16 核 32 线程，Zen5 带 AVX-512）
- GPU：Radeon 8060S 核显（RDNA 3.5，40 CU，统一内存）
- 系统：Windows（本机无 NVIDIA CUDA；ROCm/vLLM 为 Linux 专用，不可用）
- Python venv：`C:\AI\cosyvoice\venv`（Python 3.11）
- CosyVoice 代码：`C:\AI\cosyvoice\CosyVoice`（git tag **v2.0**）
- 模型：`C:\Users\wubi\CosyVoice2-0.5B`

## 2. 深度调研结论

1. **Tinysoft/Cosyvoice2-0.5B-GGUF**（HuggingFace）：llama.cpp 官方社区方案，
   声称 LLM 步骤 PyTorch→llamacpp 约 **10 倍提速**。README 明确：
   - 只能**直接喂 token id**（tokenizer 不可用）；
   - f16 与 q8 版本带 bias head（q6/q4 不带，音质更差）；
   - 前后处理仍需 CosyVoice 原代码，llama.cpp 只替换 LLM 步。
2. **llama.cpp PR #14711**：允许 Qwen2 架构带可选 bias 张量（CosyVoice2 LLM 需要），
   2025-07 已合入，因此较新的 llama.cpp / llama-cpp-python 都支持该 GGUF。
3. **llama-cpp-python 官方 wheel 索引**（本机没有 MSVC，不能源码编译）：
   `pip install llama-cpp-python --extra-index-url https://abetlen.github.io/llama-cpp-python/whl/vulkan`
   提供 Windows Vulkan 预编译 wheel（v0.3.34）。PyPI 上只有 sdist。

## 3. 最终架构

| 环节 | 之前 | 现在 |
|---|---|---|
| LLM（最慢，曾占 55%） | PyTorch CPU，~7 tok/s | llama.cpp GGUF + **Vulkan 核显**，q8 ~220 tok/s / f16 ~130 tok/s |
| flow 解码器 estimator | torch CPU 10 步 | ONNX(fp32) + **DirectML**，5 步（v2.6.4 已做） |
| flow encoder / hift | torch CPU | 保持 torch CPU（占比小） |

服务端 HTTP：`127.0.0.1:8010`（OpenAI 兼容 `/v1/audio/speech`），gaea 桌面端直接受益。

## 4. GGUF 统一词表布局（实测验证，勿改）

GGUF 把三个 embedding 表拼成一个统一词表，**n_vocab = 158502**（文件里对齐到 158528，尾部 26 个 pad）：

```text
文本 token      0 .. 151935    （Qwen2 tokenizer 原始 id）
sos_eos 位     151936          （原 llm_embedding[0]）
task 位        151937          （原 llm_embedding[1]）
语音 token     151938 + t      （t = 0..6563；6561=EOS，6562/6563=fill）
```

- 语音词表尺寸：`speech_token_size = 6561`（cosyvoice2.yaml），`+3` = 6564 类。
- 验证方法：同一 prompt 下 torch `llm_decoder` logits 与 GGUF logits 切片
  `[151938:151938+6564]` 对比，q8 相关性 **0.9999**、top1 一致、top10 全中。
- SFT 模式 LLM prompt = `[151936] + 文本token + [151937]`（无参考语音）。
- 采样必须复刻 `ras_sampling(top_k=25, top_p=0.8, win_size=10, tau_r=0.1)`，
  见引擎 `_ras_sample()`（numpy 版，含 EOS 重采样上限 100 次）。

## 5. 文件与改动清单

### 新增
- `C:\AI\cosyvoice\CosyVoice\cosyvoice\llm\gguf_engine.py` — GGUF 引擎：
  KV cache 逐 token 解码（`Llama.eval` + `ctx.get_logits()`），锁保护，numpy 采样。
- `C:\AI\cosyvoice\gguf\cosyvoice_f16.gguf`（1.23GB）、`cosyvoice_q8.gguf`（657MB）
- `C:\AI\cosyvoice\download_gguf.py` — 从 HF 直连下载（hf-mirror 的 LFS 不通，勿改回）
- 基准/验证脚本：`compare_logits.py`、`bench_engine.py`、`bench_vulkan.py`、
  `profile_gguf.py`、`measure_decoder.py`、`measure_steps.py`、`test_gguf.py`、
  `repro_server_path.py`、`debug_fp16*`、`convert/export_estimator_fp16.py`
- 试听样本：`C:\AI\cosyvoice\gguf_out\*.wav`

### 修改
- `C:\AI\cosyvoice\CosyVoice\cosyvoice\llm\llm.py` — `Qwen2LM` 增加 `gguf_engine`、
  `load_gguf()`；`inference()` 提前算 min/max_len 并分支到 GGUF。
- `C:\AI\cosyvoice\server.py` — 启动时按 `COSYVOICE_LLM_GGUF`（**默认 f16**）加载 GGUF，
  加载失败自动回退 torch；启动时做一次合成预热（Vulkan/DML shader）。
- `C:\AI\cosyvoice\CosyVoice\cosyvoice\utils\common.py` — 修复 `mask_to_bias`：
  fp16 下 `-1e10` 溢出为 `-inf` 导致 Softmax NaN，fp16 改用 `-3e4`（可表示且掩码效果等价）。

## 6. 性能数据（服务端 HTTP 实测，均含网络往返）

| 文本 | 最初 torch 全套 | v2.6.4（flow ONNX+5步） | 现在 f16 GGUF |
|---|---|---|---|
| 好的，晚安。 | 6.5s | 4.1s | **~1.4–1.8s** |
| 今天天气真不错，我们一起去公园走走吧。 | 24.5s | — | **~2.8s** |

LLM 单步：torch CPU 128ms/步 → q8 Vulkan ~4.5ms/步（220 tok/s），f16 ~8ms/步。
剩余耗时主要分布在：flow decoder（含 DirectML 按文本长度重编译的波动）、hift 声码器（torch CPU ~0.5s）。

## 7. 尝试过但否决的方案（不要再踩坑）

| 方案 | 结果 |
|---|---|
| LLM 单步 ONNX 导出（KV cache） | 单步正确，但**链式推理发散**（step1 后 max diff≈10-12，音频相关性 -0.05），弃用 |
| LLM bf16 | 1.23s/步 但 **EOS 采样失效**（音频拉满 max_len），弃用 |
| torch.compile | BackendCompilerFailed |
| flow estimator **fp16**（torch 直接导出 ONNX） | DML 上 2 倍速（68→34ms），但**5 步欧拉误差累积**，同一 token 下波形相关性仅 **0.016**，音质不可接受，弃用并删除 fp16 onnx |
| flow 5 步→4 步 | 波形相关性 0.999983 几乎无损，但实测无稳定提速（DML 编译波动），**保持 5 步** |
| vLLM / ROCm | Linux + CUDA 专用，Windows 不可用 |
| onnxconverter-common fp32→fp16 转换 | 模型含 369 个 Cast 节点，转换产物类型错误无法加载 |

## 8. 运维速查

```powershell
# 重启服务（会先杀所有 python 进程，注意别与其他服务共用）
powershell -NoProfile -ExecutionPolicy Bypass -File C:\AI\cosyvoice\restart_server.ps1

# 切回 q8（更快、体积小）：
$env:COSYVOICE_LLM_GGUF='C:\AI\cosyvoice\gguf\cosyvoice_q8.gguf'

# 日志
C:\AI\cosyvoice\server.log        # print 输出 / uvicorn
C:\AI\cosyvoice\server_err.log    # logging 输出（含 GGUF 启用行）
```

确认 GGUF 生效：`server_err.log` 出现
`Qwen2LM switched to llama.cpp GGUF backend: ...cosyvoice_f16.gguf`。

## 9. 已知限制与下一步

- DirectML 会按 mel 帧数（即文本长度）**重编译 shader**，造成单次请求偶发 +0.5~1s 波动。
- 服务端暂为整句合成后返回；若做**流式 TTS**（LLM token 边出边播），首包可缩到 ~0.5s，
  但需要 gaea 前端配合，未做。
- hift 声码器含 `torch.istft`/RNN，未 ONNX 化；如需再提速可考虑卷积部分导出 + istft 留 torch。
- gaea 桌面端无需改代码即受益；已于 2026-08-09 发布 **v2.6.5**：
  `releases/gaea-v2.6.5.exe`（SHA256 `D8F4AA815339F20ADFEEC3C1164B21B51B91EE115C07D0DD8DD2AF52DF81792C`），
  桌面 `C:\Users\wubi\Desktop\gaea.exe` 已同步，git tag `v2.6.5`。
