# gaea 项目记忆

## 项目定位

gaea 是 Windows 桌面端「通用办公」AI 助手（Wails v2：Go 1.26 后端 + React/TypeScript/Vite 前端）。
核心能力：文档撰写、表格处理、格式转换（docx/xlsx/pdf ↔ Markdown）、图表生成、报告拼装、
知识库与记忆中枢、方案编写。品牌定位已从「土壤修复工程办公」全面转为「通用办公」。

## 技术栈与关键约定

- 桌面框架 Wails v2.13（Go + WebView2）；后端事件总线 + 前端 zustand 桥接（bridge.ts → window.go.app.App）
- 单模型架构：一个 executor 完成规划与执行，无独立规划者；任务/技能子代理走 `task` / `run_skill`
- 内置工具精简为 17 个核心工具（v2.4.3 起）：文件/命令、网络、任务、记忆/知识、技能、format_convert、chart_gen
- 文档能力交给 ModelScope 技能：docx / pdf / xlsx（安装在 `~/.codex/skills` 与 `.gaea/skills`），
  转换引擎共用 `internal/office/docmd`（format_convert 工具与预览面板同一实现）
- 内置子代理技能：format-convert / chart-builder / doc-assemble
- 记忆系统：SQLite（`%APPDATA%\gaea\Hephaestus.db` facts 表，按项目 slug 隔离）+ 文档记忆（AGENTS.md 层级）
- 环境依赖：LibreOffice（soffice）、node 全局 docx、Python 3.13（lxml/openpyxl/pypdf/pdfplumber/reportlab/pandas/matplotlib 等）

## 发布流程

1. 更新 CHANGELOG.md / README.md / wails.json（productVersion）
2. `cmd /c build.bat`（wails build，产物 build/bin/gaea.exe，同时复制到桌面）
3. 复制 exe 到 `releases/gaea-v<版本>.exe`，生成 `releases/SHA256SUMS-v<版本>.txt`
4. 写 `releases/v<版本>.md` 发布说明，更新 `releases/README.md` 版本表
5. 更新 `.gaea/progress.md` 进度记忆与本文件

## 版本状态

- 最新发布：v2.6.9（2026-08-09）「移除 VoxCPM2」：实测不达标（耗时长/音色男女混乱），本地 TTS 仅 CosyVoice2
  - v2.6.5/2.6.6：CosyVoice2 LLM 切 GGUF+Vulkan（提速 8–10×）、VoxCPM2 本地引擎接入（ROCm）
- 里程碑：设置/首页/聊天欢迎屏/小说/绘梦五板块统一玻璃 HUD 设计语言；
  绘梦新增图生图（ComfyUI 低 denoise）与文生视频（LTX-Video，GenerateMedia 绑定）；
  模型中心功能绑定实时同步 + 办公/聊天去重合并；角色库新增 LLM 绑定（func_characterlib_*）
  与剧照独立后端/模型（portrait_backend / portrait_model）；vision 技能切 Qwen3.6-35B-A3B
  并修复 PS 5.1 UTF-8 编码（BOM + .NET HttpClient）
- 已知注意：角色库剧照默认跟随绘梦（ImageBackend/ImageModel），可在模型中心单独绑定；
  文生视频依赖本地 ComfyUI 安装 LTX-Video 模型

## 本地 TTS 引擎（重要记忆，勿遗忘；2026-08-09 整理）

> ⚠️ **VoxCPM2 已于 v2.6.9 移除**：实测耗时长（单句 2–3s+）、音色男女混乱、克隆不稳定。
> 下方 VoxCPM/Vulkan 相关方法保留为“已废弃教训”，勿重新安装；当前本地 TTS 仅 CosyVoice2。

本机（Radeon 8060S 核显 / 128GB 统一内存 / Windows）本地 TTS 有两条引擎线，gaea 只连 OpenAI 兼容 8020/8010：

### ~~VoxCPM2~~（已移除 v2.6.9，以下为废弃记录）
- `8030`：**主后端** `C:\AI\llama-omni\build\bin\llama-tts-server.exe`
  （llama.cpp-omni `tools/server/server-voxcpm2.cpp`，C++/ggml + **Vulkan**）
  - 模型：`C:\AI\llama-omni\models\VoxCPM2-BaseLM-Q8_0.gguf`（1.65GB）+ `VoxCPM2-Acoustic-F16.gguf`（1.74GB）
  - 来源：DennisHuang648/VoxCPM2-GGUF（ModelScope 镜像 `modelscope.cn/models/DennisHuang/VoxCPM2-GGUF` 更快）
  - 8060S 识别 `KHR_coopmat + bf16`，全部 29 层 offload Vulkan0，加载约 2s
- `8021`：**备胎** ROCm PyTorch（`C:\AI\voxcpm\server.py` + `VOXCPM_PORT=8021`，
  torch 2.9.1+rocm7.2.1 + TunableOp，CFG=1.5）
- `8020`：**适配器** `C:\AI\voxcpm\adapter.py`（FastAPI，gaea 的入口，契约不变）：
  内置音色/声音设计优先走 Vulkan，后端不可用自动回退 ROCm；默认 6 步 / CFG 1.5 / max_steps=100；
  峰值 <0.85 自动增益归一（中文克隆偏轻）
- 一键启动：`C:\AI\voxcpm\start_voxcpm_stack.ps1`（8030→8021→8020，按端口安全清理；
  C++ 后端依赖 `C:\msys64\ucrt64\bin`，脚本已处理 PATH）

### CosyVoice2（端口 8010）
- `C:\AI\cosyvoice\server.py`：LLM 用 GGUF + Vulkan（`gguf\cosyvoice_f16.gguf`），flow 用 ONNX + DirectML（5 步）
- 启动：`C:\AI\cosyvoice\start_cosyvoice.bat`；约 14s 加载+预热，短句 ~1.5s

### 音色（两引擎统一 4 个，火山引擎 Speech-AI-Forge-spks 录音室样本）
- 中文女 `zh_female.wav`（volc 知性温婉，f0≈221Hz）、中文男 `zh_male.wav`（volc 儒雅青年，f0≈133Hz）
- 英文女 `en_female.wav`（volc Sarah，f0≈191Hz）、英文男 `en_male.wav`（volc Daniel，f0≈109Hz）
- 参考音频 ≤7s / 16kHz；转写在 `C:\AI\voxcpm\voices\_meta.json`

### 本次优化方法（AMD 核显提速，勿重蹈覆辙）
1. **不要再用纯 ROCm PyTorch 追速度**：iGPU 共享内存架构下 ROCm 与 CPU 基本同速
   （5 步 RTF ≈1.06–1.12）；Vulkan + ggml 的 GEMM/coopmat 才是突破口（克隆 RTF 0.65–0.84）
2. **构建**：MSYS2 UCRT64（`C:\msys64\msys2_shell.cmd -defterm -no-start -ucrt64 -c`），
   装 `mingw-w64-ucrt-x86_64-{toolchain,cmake,ninja,vulkan-headers,vulkan-loader,shaderc,spirv-headers}`，
   `cmake -B build -DGGML_VULKAN=ON -DGGML_NATIVE=ON`，目标 `llama-tts-server voxcpm2-cli`
3. **坑 1（端口绑不上）**：检出 OpenSSL 后 server-voxcpm2 会构建 `SSLServer`，空证书导致
   `is_valid_=false`、任何端口 bind 都失败；本地回环不需要 TLS，改普通 `httplib::Server`
4. **坑 2（克隆近静音）**：AudioVAE 参考特征必须 frame-major（`ggml_cont(latent)`），
   不能 `cont(transpose(latent))`（dim-major），否则与 Python `[patches, patch_size, feat_dim]` 不一致
5. **坑 3**：llama.cpp-omni 的 CLI `-r` 克隆偶发偏静音，HTTP server 路径正常；生产走 server
6. **坑 4**：VoxCPM Python 长文本 CFG 2.0 会「跑飞+整段重试」（RTF 4.8–7.8），CFG 1.5 稳定；
   C++ server 端用 max_steps 限制解码上限
7. **网络**：HuggingFace LFS 直连/hf-mirror 都不通，ModelScope 直链快（8.6MB/s）
8. 实测：短句克隆 RTF 0.65–0.84（6 步）、语音设计 0.57–0.60；同 seed 输出确定

### 详细记录
- `docs/2026-08-09-voxcpm2-integration.md`（VoxCPM2 全部历程：接入、ROCm、Vulkan 加速、音色替换）
- `docs/2026-08-09-cosyvoice2-llm-gguf-speed-optimization.md`（CosyVoice GGUF 提速）

### 自动启动（当前仅 CosyVoice2）
- gaea 启动时后台 ensure cosyvoice；模型中心 TTS 模型卡片「启动」按钮 →
  `App.StartLocalTTSService(engineId)`；引擎「测试连接」先 ensure（等 ≤8s）；TTS 合成前兜底 ensure
- 实现在 `internal/app/tts_service.go`（`core.ensureLocalTTSService` 幂等 + 异步轮询，
  emit `tts-service-status`；CosyVoice 直接 python server.py，隐藏窗口 CREATE_NO_WINDOW）
- 端口探测：CosyVoice2 `8010/v1/models`

