# VoxCPM2 本地语音引擎接入记录（2026-08-09，已废弃）
> ⚠️ **已废弃 / 已移除（v2.6.9，2026-08-09）**：实测 VoxCPM2 不达标——
> 合成耗时长、音色男女混乱、克隆不稳定，已从 gaea 彻底移除，本地文件已删除。
> 本文保留为技术过程记录；当前本地 TTS 仅 CosyVoice2（8010）。

> 结论先行：VoxCPM2（2B 扩散式多语种 TTS）已按 CosyVoice2 同款「本地 OpenAI 兼容 TTS 服务 + 模型中心引擎」模式接入 gaea。
> 通过 AMD ROCm（Windows 原生）驱动 Radeon 8060S 核显推理，配合 PyTorch TunableOp 对固定形状 GEMM 调优，
> 实测中文中长句（约 7.7s 音频）合成约 16s，RTF ≈ 2.0；短句（约 5.4s 音频）RTF ≈ 1.35。

## 一、环境与部署

- 服务目录：`C:\AI\voxcpm`
- Python venv：`C:\AI\voxcpm\venv`（Python 3.12.13，cp312）
- PyTorch：`torch==2.9.1+rocm7.2.1` / `torchaudio==2.9.1+rocm7.2.1`
  （AMD ROCm Windows 轮子，源：`https://repo.radeon.com/rocm/windows/rocm-rel-7.2.1/`）
- VoxCPM 包：`voxcpm==2.0.3`（`--no-deps` 安装后手动补齐依赖，避免覆盖 ROCm torch）
- 模型：`C:\AI\voxcpm\models\VoxCPM2`（4.96GB，ModelScope 官方镜像下载：
  `modelscope download --model OpenBMB/VoxCPM2 --local_dir C:\AI\voxcpm\models\VoxCPM2`；
  HuggingFace 直连 LFS CDN 与 hf-mirror 均不通，勿改回）
- 服务端口：`127.0.0.1:8020`（OpenAI 兼容 `/v1/audio/speech`）
- 启动：`C:\AI\voxcpm\start_voxcpm.bat`；重启/查日志：`C:\AI\voxcpm\restart_server.ps1`

## 二、gaea 接入（与 CosyVoice2 同构）

- 引擎注册：`internal/modelengine/engine.go` 新增 `voxcpm` 类型与内置引擎
  （BaseURL `http://127.0.0.1:8020/v1`，默认模型 `VoxCPM2`）
- 启动补齐：`internal/app/app.go` 启动时 `EnsureModel("voxcpm", "VoxCPM2")`
- TTS 路由：`internal/app/tts_handler.go` 自动扫描列表加入 `voxcpm`，模型识别加入 `vox`
- 音色路由：`internal/app/voice_model_handler.go` / `internal/tts/herdsman.go`
  （`GetTTSSpeakers` 走 `voxcpm` 引擎、默认音色 `中文女`）
- 前端：`engines.ts` 类型、`ModelCenterPage` 图标/颜色/标签/兜底音色、
  `VoiceSettingsPanel` / `ChatPanel` 音色选择器（VoxCPM2 与 CosyVoice2 共用 7 个内置音色名）
- 复用 `HerdsmanTTS` 客户端（OpenAI `/v1/audio/speech` + data URI 解码），无需新协议

## 三、服务端要点（`C:\AI\voxcpm\server.py`）

- 设备：`VOXCPM_DEVICE=auto`，ROCm 下 `torch.cuda.is_available()=True` 自动走核显；`optimize=False`
  （ROCm/Windows 下 torch.compile 不可靠）
- **TunableOp（提速关键）**：`PYTORCH_TUNABLEOP_ENABLED=1` + `TUNING=1`，结果缓存到
  `C:\AI\voxcpm\tunable_results.csv`（实测对固定形状 GEMM 提速约 4~5 倍）；
  启动预热后调用 `torch.cuda.tunable.write_file()` 持久化
- **CFG 取值（稳定性关键）**：CFG=2.0 在中文长文本上会触发「生成跑飞 → retry_badcase 整段重试」
  （实测 RTF 4.8~7.8）；默认 `VOXCPM_CFG=1.5` 后稳定且 RTF ≈ 2.0
- 推理步数 `VOXCPM_STEPS=10`：步数对总耗时影响有限（瓶颈在逐 token LM 前向），保留 10 步保音质
- 音色：内置 7 个与 CosyVoice2 同源参考音频（`refs16k/*.wav` → `voices/`），
  显示名一致（中文女/中文男/英文女/英文男/日语男/粤语女/韩语女）；
  `POST /v1/voices` 上传参考音频注册克隆音色（librosa 加载 → 16kHz 落盘，持久化）
- 声音设计：请求带 `voice_description` 时走「(描述)文本」模式，无需参考音频
- 推理串行化（单实例 + 锁），启动预热短句，`/v1/status` 暴露 device/load_s/warmup_rtf

## 四、实测数据（Radeon 8060S，64GB 统一内存）

| 场景 | 音频时长 | 合成耗时 | RTF |
|---|---|---|---|
| 短句（TUNING=0 缓存命中） | 5.44s | 7.35s | 1.35 |
| 中长句稳态（CFG=1.5，连续 3 次） | 7.5~8.0s | 16.0~16.4s | 2.06~2.13 |
| 中长句 CFG=2.0（跑飞+重试） | 6.88s | 33.0s | 4.79 |
| 声音设计模式（首次含调优） | 4.16s | 38.5s | 9.26 |
| 克隆音色（首次含调优） | 3.84s | 34.1s | 8.88 |

- 输出恒为 48kHz / float32 / 无削波（RMS 0.04~0.06，峰值 0.27~0.47）
- AOTriton 实验性注意力（`TORCH_ROCM_AOTRITON_ENABLE_EXPERIMENTAL=1`）反而慢约 8 倍，已弃用
- torch.compile 因 Windows ROCm 无 triton 不可用（`optimize=True` 会自动回退并告警）

## 五、已知限制

- iGPU 算力有限：速度远低于 RTX 4090（官方 RTF 0.3），长文建议按句切分（gaea 流式朗读已按句调用）
- 极长文本偶发不稳定（官方已知限制），服务端默认 `retry_badcase=True` 兜底
- 首次请求若出现未见过的形状会触发一次性 kernel 调优（数百毫秒/形状），随后命中缓存

## 六、更新（同日）：llama.cpp-omni Vulkan GGUF 深度加速

> 目标：让 VoxCPM2 在 AMD Radeon 8060S 核显上接近 NVIDIA 的响应速度。
> 结论：新增 **llama.cpp-omni（C++/ggml + Vulkan）** 推理后端，短句克隆音色 RTF 从 ROCm 的 ~1.1–1.4
> 降到 **0.65–0.84**，语音设计模式 RTF **0.60**；中文/英文 4 音色已替换为 Speech-AI-Forge-spks
> 火山引擎录音室级样本，并做自动音量归一。

### 架构（三层，gaea 无需改动，仍指向 127.0.0.1:8020/v1）
- `8030`：`llama-tts-server.exe`（llama.cpp-omni `tools/server/server-voxcpm2.cpp`，Vulkan，Q8_0 BaseLM + F16 Acoustic）
- `8021`：原 ROCm PyTorch 服务（`server.py` + `VOXCPM_PORT=8021`）作为备胎
- `8020`：`adapter.py`（FastAPI 适配器）：按 OpenAI 契约转发，内置音色/声音设计走 Vulkan，
  后端不可用时自动回退 ROCm；默认 `inference_timesteps=6`、`cfg=1.5`、`max_steps=100`，峰值 <0.85 时自动增益归一

### 构建与关键修复
- 工具链：MSYS2 UCRT64（gcc 16 / cmake / ninja）+ `GGML_VULKAN=ON`；shaders 走 glslc（shaderc），
  8060S 识别为 `KHR_coopmat + bf16`，全部 29 层 offload 到 Vulkan0
- 模型文件：`VoxCPM2-BaseLM-Q8_0.gguf`（1.65GB）+ `VoxCPM2-Acoustic-F16.gguf`（1.74GB），
  来源 DennisHuang648/VoxCPM2-GGUF（ModelScope 镜像更快）
- 修复 1（绑定失败）：构建检出 OpenSSL 后 `server-voxcpm2` 会创建 `SSLServer`，空证书导致
  `is_valid_=false`、任何端口都绑不上；本机回环不需要 TLS，改为普通 `httplib::Server`
- 修复 2（参考特征布局）：`encode_reference_audio` 原用 `cont(transpose(latent))` 得到 dim-major，
  与 Python `[patches, patch_size, feat_dim]` 不一致，改为 frame-major（`cont(latent)`），
  使克隆条件特征与 Python 对齐（absmean 1.063 vs 1.065）
- 启动：`C:\AI\voxcpm\start_voxcpm_stack.ps1`（顺序启动 8030 → 8021 → 8020，按端口安全清理）

### 实测（Radeon 8060S，Vulkan，6 步 / CFG 1.5）
| 场景 | 音频时长 | 生成耗时 | RTF |
|---|---|---|---|
| 中文女克隆（短句） | 3.52s | 2.6–2.9s | 0.74–0.83 |
| 中文男克隆（短句） | 2.56s | 2.1–2.2s | 0.83 |
| 英文女克隆（短句） | 2.56s | 2.1–2.2s | 0.81–0.84 |
| 英文男克隆（短句） | 2.88s | 2.2–2.3s | 0.77–0.78 |
| 语音设计（英文） | 7.52s | 4.49s | 0.60 |
| 语音设计（中文） | 4.64s | 2.65s | 0.57 |

对比：ROCm PyTorch 5 步短句 RTF ≈1.06–1.12、10 步 ≈1.3+；Vulkan 后端整体快 **1.5–1.8×**。
同 seed 输出确定（seed=42），可复现。

### 音色（与 CosyVoice 同步替换，共 4 个）
- 中文女 `zh_female.wav`（volc 知性温婉，f0≈221Hz）
- 中文男 `zh_male.wav`（volc 儒雅青年，f0≈133Hz）
- 英文女 `en_female.wav`（volc Sarah，f0≈191Hz）
- 英文男 `en_male.wav`（volc Daniel，f0≈109Hz）
- 参考音频已裁剪到 ≤7s 并 16kHz 落盘（`voices/_meta.json` 保留转写），适配器对偏轻的中文克隆自动增益

### 已知问题
- llama.cpp-omni 的 CLI `-r` 克隆路径在部分调用下偏静音，HTTP server 路径正常；生产走 server
- C++ 克隆对中文长文本仍建议按句切分（gaea 流式朗读已按句调用）
- `voxcpm2-cli`/server 依赖 `C:\msys64\ucrt64\bin`（启动脚本已处理 PATH）

### 模型中心自动拉起（v2.6.8）
- gaea 启动时后台 ensure cosyvoice/voxcpm（幂等，已就绪零开销）；模型中心 TTS 模型卡片
  「启动」按钮调用 `App.StartLocalTTSService(engineId)`；「测试连接」与合成前兜底同样自动拉起
- 实现在 `internal/app/tts_service.go`：探测 8010 `/v1/models`、8020 `/v1/status`；
  CosyVoice 直接 `python server.py`，VoxCPM 执行 `start_voxcpm_stack.ps1`（隐藏窗口，
  CREATE_NO_WINDOW）；异步轮询就绪并 emit `tts-service-status`

