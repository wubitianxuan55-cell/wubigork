# 本地模型引擎/运行时产品化调研（原始稿，2026-09-02）

调研子代理产出，来源均为官方文档/GitHub（2026-09 现状）。合成结论见 `docs/market-research-2026-09-02.md`。

## Ollama
- 发现/下载：模型来自 [registry.ollama.ai](https://docs.ollama.com/faq)，CLI `ollama pull/run`，用 tag 选参数量/量化（如 `llama3.1:8b-instruct-q4_K_M`）；代理走 `HTTPS_PROXY` 环境变量；官方无镜像源配置（可用 Docker-Registry 兼容反代，[issue #2388](https://github.com/ollama/ollama/issues/2388)）；断点续传官方文档未见明确说明，社区反馈中断常从头重下（[issue #13167](https://github.com/ollama/ollama/issues/13167)、[#14254](https://github.com/ollama/ollama/issues/14254)）
- 文件管理：默认路径 macOS `~/.ollama/models`、Linux `/usr/share/ollama/.ollama/models`、Windows `C:\Users\<user>\.ollama\models`，`OLLAMA_MODELS` 改路径；`ollama list` 显示 SIZE、`ollama rm` 删除、`ollama cp` 复制、`ollama show` 元数据；多 tag/量化可共存，blob 层按内容寻址去重（[FAQ](https://docs.ollama.com/faq)）
- 运行时配置：context 默认 4096（`num_ctx` 参数 / `OLLAMA_CONTEXT_LENGTH`）；GPU offload 全自动、无手动层数参数，`ollama ps` 显示 CPU/GPU 分配比例；flash attention 自动开启，KV cache 可选 f16/q8_0/q4_0（FAQ）
- 生命周期：默认 keep_alive 5 分钟空闲后自动卸载，`ollama stop` 立即停；`OLLAMA_KEEP_ALIVE`、`OLLAMA_MAX_LOADED_MODELS`（默认 3×GPU 数）、`OLLAMA_NUM_PARALLEL`（默认 1）、`OLLAMA_MAX_QUEUE`=512；多 GPU 自动切分加载（FAQ）
- 对外暴露：默认端口 11434，自有 `/api/generate|chat|tags|ps` + OpenAI 兼容 `/v1/chat/completions` 等；`OLLAMA_HOST` 改绑定、`OLLAMA_ORIGINS` 配 CORS（FAQ / [api 文档](https://docs.ollama.com/api)）
- 硬件提示：下载前显存评估「未见」（加载时才自动决定 offload 层数）
- 独有：[Modelfile](https://docs.ollama.com/modelfile)（FROM/PARAMETER/SYSTEM/TEMPLATE/ADAPTER）创建衍生模型变体；2025 起含云模型路由，可在 `~/.ollama/server.json` 禁用（FAQ）

## LM Studio
- 发现/下载：内置 Discover 页搜索任意 HF 模型（关键词/用户名/模型名/完整 HF URL），每个模型列出可选量化档并显示文件大小，官方建议 ≥4bit（[download-model](https://lmstudio.ai/docs/app/basics/download-model)）；应用内下载时对量化档与本机内存的适配提示存在（社区确认，官方文档页「未见」专门说明）
- 文件管理：模型目录可在 My Models 页更改（现默认 `~/.lmstudio/models`，旧版 `~/.cache/lm-studio/models`），强制 `发布者/模型名` 两级目录；外部 GGUF 按目录结构放入即识别（[import-model](https://lmstudio.ai/docs/app/advanced/import-model)）；My Models 列表 0.4.0 起有 Capabilities/Format 等列（[blog 0.4.0](https://lmstudio.ai/blog/0.4.0)）；删除走 3 点菜单（曾有删后残留目录的 bug 报告 [issue #199](https://github.com/lmstudio-ai/lmstudio-bug-tracker/issues/199)）
- 运行时配置：每模型可设 context 长度、GPU offload 层、CPU 线程等加载配置；macOS 另有 MLX 引擎（[docs](https://lmstudio.ai/docs/app/basics/download-model)）
- 生命周期：JIT 按需加载 + 「Auto Unload Unused JIT Models」+ 可配 Max idle TTL 自动卸载；支持 headless 后台服务（`lms server start`）（[headless](https://lmstudio.ai/docs/developer/core/headless)、[server settings](https://lmstudio.ai/docs/developer/core/server/settings)）；0.4.0 支持并行请求+continuous batching（blog）
- 对外暴露：本地服务器默认端口 1234，OpenAI 兼容 /v1（chat/completions/embeddings）+ 自有 REST /api/v0（返回模型 loaded/unloaded 状态、max context、quantization、TTFT 等富元数据）（[openai](https://lmstudio.ai/docs/app/api/endpoints/openai)、[rest](https://lmstudio.ai/docs/app/api/endpoints/rest)）
- 独有：TypeScript/Python SDK（可编程加载/卸载/推理）、`lms` CLI（`lms get` 搜索下载、`lms ls/load/unload`）（[developer docs](https://lmstudio.ai/docs/developer)、[cli](https://lmstudio.ai/docs/cli)）

## Jan
- 发现/下载：Hub 直连 Hugging Face 浏览下载，模型详情页展示参数/硬件需求，2025-08 起 Hub 有「模型兼容性检查器」（下载前显示硬件要求）（[manage-models](https://www.jan.ai/docs/desktop/manage-models)、[changelog 2025-08-28](https://jan.ai/changelog/2025-08-28-image-support)）；支持粘贴 HF 模型 ID 导入并选量化档、配 HF Access Token；下载有顶栏进度条+取消
- 文件管理：数据全部存本地 JSON 数据文件夹（Settings > General 可打开），模型默认在 `~/jan/models`（[data-folder](https://www.jan.ai/docs/desktop/data-folder)、[cli](https://www.jan.ai/docs/desktop/cli)）；`jan models list` CLI 管理；Settings > Llama.cpp > Models 支持「Delete All」批量删除并显示可释放空间；任意 .gguf 文件可不限扩展名导入（changelog 2025-08）
- 运行时配置：llama.cpp 引擎按硬件自动选 backend（CUDA 各版本/Vulkan/AVX 变体）；引擎级设置+每模型覆盖：Context Size、GPU Layers、Batch Size、Disable KV Offload、KV cache f16/q8_0/q4_0、threads、Flash Attention、MLock、Mirostat、grammar/JSON schema、RoPE（[llama-cpp](https://www.jan.ai/docs/desktop/local-engine/llama-cpp)、[changelog 2025-02-18](https://jan.ai/changelog/2025-02-18-advanced-llama.cpp-settings)）；硬件面板可启用/停用各 GPU
- 生命周期：模型路由器可设 Max Concurrent Models，加载新模型自动卸载旧模型；独立 keep_alive/TTL 机制「未见」
- 对外暴露：内置 OpenAI 兼容 Local API Server，默认 `http://localhost:1337`（/v1/chat/completions），llama.cpp 驱动，可作云 API 平替（[api-server](https://jan.ai/docs/desktop/api-server)）；远程 provider（OpenAI/Claude 等）同一路由统一管理
- 独有：JS 扩展架构（官方桌面文档本次未见专门页面，仅在 [GitHub 组织](https://github.com/janhq/jan)体现）；v0.6.6 起移除 Cortex、全面转 llama.cpp（[reddit](https://www.reddit.com/r/LocalLLaMA/comments/1mdy1at/jan_now_runs_fully_on_llamacpp_autoupdates_the/)）

## llama.cpp（llama-server）
- 发现/下载：2025 后新增 router 模式：`/models` 网页/API 可直接下载 HF GGUF 模型（`/models/download` POST、SSE 下载进度事件、`/models/delete-model` 删除）；下载入 `LLAMA_CACHE` 目录（HF 仓库布局），另可 `--models-dir` 扫描本地已有模型（[server README](https://github.com/ggml-org/llama.cpp/blob/master/tools/server/README.md)）；GUI 断点续传「未见」
- 文件管理：本体无模型库 UI（传统用法=自己放 GGUF 文件），router 网页可看已下载模型与大小；CLI `llama-gguf` 可查看 GGUF 元数据
- 运行时配置：全量 CLI 参数：`-ngl` GPU 层数、`-t/-tb` 线程、`--ctx-size`（支持 n_ctx=auto）、`--flash-attn`（on/off/auto，另有 `--auto-fa` 自动测速决定）、`--cache-type-k/v` KV 量化、`-b/-ub` batch、`--mlock/--no-mmap`、`--device/--list-devices`、`--split-mode`；`-fit` 自动按显存计算最小可行 offload 层数；`--preset` 文件保存/复用参数（README）
- 生命周期：`--models-max`（>1 时 router 并行驻留多模型）、`--timeout` 空闲自动卸载、`/models/unload` API、`--sleep --idle` 睡眠模式（CUDA 显存释放）（README）
- 对外暴露：OpenAI 兼容 /v1（含 /v1/models 元数据：参数量 n_params、模型体积、ctx）+ Anthropic 兼容 /v1/messages + `/health` 健康检查 + `/metrics` Prometheus + SSE 流式 + `--api-key` 鉴权 + 内置 WebUI（README）
- 硬件提示：无 GUI 级评估；`-ngl auto`/`-fit` 即自动适配机制；显存不足警告「未见」

## vLLM（简）
- `vllm serve <model>` 启动 OpenAI 兼容 REST（默认端口 8000，/v1/chat/completions 等 + /docs OpenAPI）；模型为启动参数而非可管理列表——无内置模型库/下载管理 UI，依赖 HF 缓存；`--api-key` 鉴权，并发连续批处理，extra_parameters 扩展采样参数（[docs](https://docs.vllm.ai/en/latest/serving/openai_compatible_server.html)）
- 健康检查/显存评估 UI「未见」（面向服务器，无桌面产品化层）

## GPT4All（简）
- 现状：2025-02（v2.7.x）后无更新、不支持新模型，社区普遍认为已停滞，无官方公告（[issue #3605](https://github.com/nomic-ai/gpt4all/issues/3605)、[#3558](https://github.com/nomic-ai/gpt4all/issues/3558)）
- 产品化要点（参考价值）：应用内模型目录下载（含体积/描述）、LocalDocs 本地 RAG、内置 OpenAI 兼容本地 API Server（默认端口 4891）（[wiki](https://github.com/nomic-ai/gpt4all/wiki/Local-API-Server)、[docs](https://docs.gpt4all.io/index.html)）

## 横向小结（供「模型中心」设计参考）
- 共性基线：HF 生态为源 + 量化档位选择 + GGUF 为通用格式 + OpenAI 兼容 API + 本地 HTTP 端口；模型目录均可自定义
- 差异点：Ollama 强在「极简 CLI + Modelfile 变体 + 自动卸载」；LM Studio 强在「GUI 下载器 + JIT/TTL + SDK/REST 富元数据」；Jan 强在「Hub 硬件兼容性预检 + 批量删除释放空间 + 多 provider 路由」；llama.cpp 强在「/fit 自动适配 + 健康检查/metrics + 单二进制多模型 router」
- 显存不足警告/推荐量化档位：仅 Jan（兼容性检查器）和 LM Studio（应用内指示）做了下载前评估，是明确的产品机会点
