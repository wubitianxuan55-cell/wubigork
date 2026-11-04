# 当日总账（2026-11-04）：三引擎退役 + Strata flash 档线

> 批末=a481e357；桌面 exe=build/bin/gaea.exe 57,996,288B SHA256 前缀
> CF356B95（桌面副本同哈希，壳 PID 23024）。今日两笔：**49876774**
> （三引擎退役批，118 文件）、**a481e357**（Strata flash 线三段拍板，
> 4 文件）。均未 push，随下版列车。

## 提交索引（2 笔）

| 提交 | 线 | 一句话 |
|---|---|---|
| 49876774 | 模型中心 | CosyVoice/Ollama/OpenCode Zen 三引擎退役：类型/种子/order 11→8、cosyvoice TTS 服务腿+Ollama 切换预估+Zen key 链全删、catalog 收窄 deepseek/xai、绑定三清单对齐 wailsjs 717（tianxuan/llama/P2 三批欠账一次清）、knownDead 清册；36 测试文件同步 |
| a481e357 | Strata | 三段拍板：①8091×8124 同驻守卫删除（分板块并行）②flash-next 档接入（AMD_HIP.md 官方配方 wrapper+真机四档实测 PASS）③35B 档删除收窄单档（gguf 留盘归 llama） |

## 三引擎退役要点

- 删：EngineCosyVoice/EngineOllama/EngineOpencodeZen、cosyvoice 本地 TTS
  服务腿（ttsCmdFactory/ttsURL，ttsReady() 无参化）、estimateOllamaModelSwitch、
  ImageBackendTypeOllama、Set/GetOpencodeZenKey+config 持久化。
- 留：tts/herdsman.go 与前端 isCosyvoice 系=模型名键（herdsman 供给
  CosyVoice3 模型），非引擎键；旧持久化值 fail-closed 回退（image_backend=ollama→xai）。
- 绑定面：bindingNames/bindingSignatures/legacyBindings 三清单对齐 wailsjs
  717——tianxuan 批漏清的 Tianxuan*5、llama 批漏再生的 Qwen35*3、P2 台账
  VoiceV2*3 签名（Qwen35 mock 恒未运行诚实空态）；X1-11 五锁全绿。

## Strata flash 线要点

- wrapper=repo/strata-qwen38fn-flash.json（配方照抄 strata 官方
  docs/AMD_HIP.md：--pack pack-ud-iq4xs + --native shard1 + --ple-gguf
  shard2 + --expert-cache auto + --spec 4 --mtp mtp-rt + --kv int8）。
- 真机：~85s 就绪（expert cache 24576 专家 55.43GiB）；四档实测
  短答 32.8 / 长答 31.4 / 思考开 38.6 / 长提示词(2261tok) TTFT 6.9s
  38.3 tok/s——177B 稀疏 MoE 与 35B 同档速度。
- 当日壳演化五版：v1 退役批（F402D6DB）、v2 同驻（8E81DA50）、v3 flash
  双档（C7D38035）、v4 校验壳（41492287）、v5 单档终版（CF356B95，PID 23024）。

## 验证矩阵

- 退役批：go build/vet 全仓 + app 215s + 七包绿；tsc + vitest 4214
  （唯一红=NovelSettingPage known-flaky 在册 worker 饥饿族，-t 单测绿）；
  check-bindings-drift 过（bindingNames↔Go 717 + 死名单清册）。
- flash：/v1/models status.value=loaded + verify_strata_speed 四档 14 断言全 PASS。

## 当日拍板台账

1. CosyVoice/Ollama/OpenCode Zen 删除退役（代码面全删）。
2. Strata×llama 同驻（互斥守卫删；中途「自动停 llama」弯路即撤）。
3. Strata 补 flash 档并实测速度。
4. Strata 收窄单档=flash-next（35B 档删；gguf 留盘归 llama 语音线）。
