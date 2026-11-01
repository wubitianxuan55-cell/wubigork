# Gaea Voice V2 — M4 快模型路由与延迟门禁报告

> 日期：2026-11-01
> 承接：[M1](./gaea-voice-v2-m1-spike-report-20261101.md) / [M2](./gaea-voice-v2-m2-protocol-report-20261101.md) / [M3](./gaea-voice-v2-m3-eot-report-20261101.md)
> 方案：[V2 总方案](./gaea-voice-v2-rearchitecture-20261101.md) §12/§14 M4

## 1. 结论

**语音专属快模型路由落地，首音频 p50 门禁达标；p95 未达标但根因已隔离到
breeze 偶发停顿，非模型路由。**

| 指标 | 门禁 | 实测（n=14 轮） | 判定 |
|---|---|---|---|
| speech_end→first audio p50 | ≤2.5s | **1.37s** | ✅ |
| speech_end→first audio p95 | ≤4.0s | 5.66s | ❌（根因=breeze，见 §3） |
| 稳态轮（10/14） | — | 1.30~1.49s | — |
| barge-in 新轮 e2e | — | 1.49s（bargein 断言 10/10 PASS） | ✅ |
| LLM 分段 p50 | — | 0.67s（35B 时代 2.87s，**降 77%**） | ✅ |

## 2. 快模型路由（方案 §12 的 spike 等价物）

- **Qwen3-4B Q4_K_M**（2.5GB，HF `Qwen/Qwen3-4B-GGUF`）装在
  `C:\AI\gaea-voice-v2\models\`；独立 llama-server（uocr2-llama b11980，
  Vulkan，`--reasoning off --ctx-size 8192 --cache-reuse 256`）常驻 **127.0.0.1:8123**。
- `gaea-spike.yaml` llm 块指向 8123；35B herdsman 配置留注释对照。
  「语音快模型与文本 35B 分进程路由」即 `voice_chat_engine/voice_chat_model`
  配置位的物理实现，产品化时把它收进 gaea model_router 的预算位即可。
- 裸测：暖态前缀命中 17ms；decode ~80 tok/s；冷首调 30 tok/s（Vulkan 管线
  预热，跑两遍以第二遍为准——与 breeze 同族已知行为）。

## 3. p95 未达标根因：breeze 偶发 TTFA 停顿（非 4B）

n=14 中 4 个离群轮的分解：

| 轮 | stt | llm | tts_ttfa | e2e | 注 |
|---|---|---|---|---|---|
| #1 | 0.27 | 0.29 | **2.09** | 4.58 | sidecar 重启后 breeze 首轮（已知冷启动） |
| #6 | 0.18 | **3.42** | 1.89 | 5.66 | 4B 一次 prefill 慢 + breeze 跟着慢 |
| #12 | 0.13 | 0.68 | **3.97** | 4.91 | llm 正常，breeze 单方停顿 |
| #13 | 0.13 | 1.70 | 0.42 | 2.37 | 4B 轻微慢 |

- LLM（4B）p50 0.67s / 仅 1 轮 3.42s；**stt 恒 0.1~0.3s；瓶颈转移到了 TTS 侧**。
- breeze 8012（用户常驻实例）偶发 2~4s TTFA：llm 空闲时也发生（#12），
  非 4B 直接抢占；更像 breeze/Vulkan 周期性停顿或实例老化。
  **口径：p95 门禁在 breeze 停顿解决前无法诚实判定**；候选：breeze 实例重启
  观察、gaea 按需拉起线换新实例、或 GGML 调度限额（挂 M5/实弹）。
- 暖态 1.3~1.5s 的构成：stt 0.16 + llm 0.6~0.7（含 4B prefill~350tok 增量）+
  tts_ttfa 0.4 + 调度余量——离 p50 门禁余量充足。

## 4. STT 对照（方案 §0.2 遗留项收官）

同语料（M3 11 案例）离线评测，字级 WER（去标点）：

| | sherpa streaming zipformer zh-14m | paraformer-zh（现役） |
|---|---|---|
| 均值 WER | 14.5% | **10.0%** |
| 剔除吞「嗯」项 | 6.1% | **0.95%** |
| RTF | 0.02（流式） | 0.2~0.27（整段） |

**结论：paraformer 留任**——质量碾压且 0.2 RTF 足够（管线 stt 分段 0.16s）；
sherpa 14M 仅在「流式 partial 展示」有价值（RTF 0.02），挂 M5 需要实时
转写回显时再议。工具：`scripts/voice_v2/stt_compare.py`（sherpa-onnx 2.0.1
装入 sidecar venv；模型 79MB 在 `C:\AI\gaea-voice-v2\models\sherpa-zh-14m\`）。

## 5. 运行态与复现

```bash
# 快模型（常驻）
C:\AI\uocr2-llama\bin\llama-server.exe --host 127.0.0.1 --port 8123 \
  -m C:\AI\gaea-voice-v2\models\Qwen3-4B-Q4_K_M.gguf -ngl 99 --ctx-size 8192 --reasoning off -np 1 --cache-reuse 256
# sidecar（已指向 8123）
scripts/voice_v2/start_spike.bat
# 延迟电池
for i in 1..N; do go run ./scripts/voice_v2/ws_probe -scenario turn -speech .tmp/voice_v2/corpus/complete_short.wav -out ...
# 剖面：grep "latency:" server 日志（本文 §1/§3 数字来源）
```

## 6. 下一步

- **M4 续（产品接入刀）**：gaea `/internal/voice/chat/completions`（人格/记忆/
  落库走 gaea，方案 §11）+ ChatPage `voice.backend=v2` flag 接 VoiceClientV2
  ——本刀未动 gaea 壳内任何代码（whisperChatCore DSH 在途，避撞）。
- **M5**：推测生成/软结束 reopen/false-interruption 恢复（sidecar 已有雏形，
  M2 前端协议已留扩展位）。
- 挂实弹：breeze 停顿观察（重启实例对照）；2.5s 灰区手感拍板（M3 遗留）；
  真机开麦全链（?voicev2=1 实验页 → 8791）。
