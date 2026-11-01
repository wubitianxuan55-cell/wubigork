# Gaea Voice V2 — M3 中文 EOT 评测报告

> 日期：2026-11-01
> 承接：[M1](./gaea-voice-v2-m1-spike-report-20261101.md) / [M2](./gaea-voice-v2-m2-protocol-report-20261101.md)
> 方案：[V2 总方案](./gaea-voice-v2-rearchitecture-20261101.md) §8/§14 M3

## 1. 结论

**M3 收窄目标全部达成，Smart Turn 赢下中文 A/B。**

| 指标 | 门禁 | 实测 | 判定 |
|---|---|---|---|
| 中文 EOT 行为面准确率 | ≥90% | **10/11 = 91%**（唯一分歧在 2.5s 灰区，见 §3） | ✅ |
| 内部停顿（≥1s）hold-or-reopen | — | **5/5 = 100%**（逗号/思考/改口/多段） | ✅ |
| Smart Turn 推理 p95 | ≤80ms | **24.6ms**（cpu_count=2；单线程曾 112.8ms） | ✅ |
| FireRed 中文文本 EOT（对照） | — | 64%（9/14 边界） | ❌ 结构性盲区 |
| 长间隙切轮机制 | — | 3.5s 间隙两轮切分 PASS | ✅ |

**A/B 定论**：文本 EOT（FireRed）在「文本上完整、韵律上未完」的边界系统性失明
（"明天下午三点开会" p=0.94、"先去一趟图书馆" p=0.96——单看文字就是完整句）；
声学 Smart Turn 恰好补位。两者非替代关系：**声学为主、文本为辅的二次意见**定位
与方案 §8.1 判定树一致，M5/M6 若引入文本意见只做低成本兜底。

## 2. 评测台（复现配方）

```bash
# 语料构建（breeze 合成中文段落 + 精确插入停顿；停顿 >800ms reopen grace 才真触发 soft-end）
go run ./scripts/voice_v2/ws_probe -build-corpus .tmp/voice_v2/corpus_manifest.json -corpus-dir .tmp/voice_v2/corpus
# 跑评测（逐案例新连接；含自愈阶梯：重连→重启 sidecar）
go run ./scripts/voice_v2/ws_probe -scenario corpus -corpus-dir .tmp/voice_v2/corpus -out .tmp/voice_v2/m3_run3
# FireRed 文本 EOT 离线评分（同边界前缀文本）
python -I scripts/voice_v2/firered_eot.py .tmp/voice_v2/corpus_manifest.json .tmp/voice_v2/firered <out.json>
# 汇总：服务端日志按连接切片 → Smart Turn 逐边界判决表 + 指标
python -I scripts/voice_v2/eot_analyze.py <server.log> .tmp/voice_v2/corpus/corpus_built.json
```

- 语料：11 案例（完整短句/疑问/长句/逗号停顿/思考停顿/改口/多段/挂尾/单发语气词/
  2.5s 与 3.5s 间隙新话题），停顿档 1000~1200ms + 2500/3500 两档。
- 判据以**线缆行为面**为准（transcription.completed 的轮数与文本）：对错看用户
  实际被切句与否；服务端逐边界 p 值作辅助（breeze 合成自带句内自然停顿会多出
  soft-end，日志判决与人工边界非一一对应，分析器已注明仅辅助）。
- ASR 字错容忍：轮数对 + Levenshtein 相似度 ≥0.8 记 PASS*（correction 案例的
  「哦→嗯、是→这」是 paraformer 字错，EOT 语义正确）。

## 3. 关键发现

1. **complete_then_new 2.5s 间隙未切两轮（唯一 FAIL）**：3.5s 变体 PASS（2/2）证明
   切轮机制健康；2.5s 落在社区端点参数 max delay 2500~3000ms 窗内，归入
   「按参数口径的合理 hold」而非 bug。要 2.5s 也切轮需压 speculative 重开上限
   （smart_turn_max_wait_ms 2000，代价=误切率上升），挂产品手感拍板。
2. **单发「嗯」被 VAD 吞掉**（soft_ends=0）：min_speech_ms=384ms 门槛所致，符合
   方案 §4.3 min_speech 160~384ms 参数族；四态 backchannel 语义留 M5/M6。
3. **sidecar 间歇性连接饿死 bug（M1 未暴露）**：带音频流量会话断开后，后续连接
   间歇性被认领但零事件上缆（session.created 永不到达；日志无异常，连接面完全
   静默）。裸连断循环不复现（0/15），与「TTS 播报中/后断开」相关。评测台已内置
   自愈阶梯（重连→重启）；**生产客户端=单长连接，风险低但接入前必须上游修或
   包重连逻辑**——「接入 vs 提取」天平又添一码。
4. **smart_turn_cpu_count=2 必配**：单线程时 paraformer 推理与 Smart Turn 抢核
   出现 109~113ms 离群（p95 112.8ms 超标）；2 线程后 p50 19.3ms / p95 24.6ms /
   max 24.6ms。已固化进 gaea-spike.yaml。
5. FireRed 推理本身很快（p50 17.3ms），其 178MB ONNX + tokenizer 走 HF
   `FireRedTeam/FireRedChat-turn-detector` 直下（pip 包名搜不到，权重不在仓库）。

## 4. 工件与参数落位

- `scripts/voice_v2/ws_probe/corpus.go`（新增）：build-corpus / corpus 场景 / 自愈 /
  Levenshtein 容错；`gaea-spike.yaml` vad 块补 `smart_turn_cpu_count: 2`。
- `scripts/voice_v2/firered_eot.py` / `eot_analyze.py`（新增，跑在 sidecar venv）。
- 权重：`.tmp/voice_v2/firered/`（178MB，不入仓；HF Apache-2.0）。
- 产物：`.tmp/voice_v2/m3_run3/corpus_run.json`（11 案例明细）等。

## 5. 下一步

- M4：Gaea 对话大脑接入（快模型路由 `voice_chat_engine/model` + 人格/记忆写回 +
  ChatPage feature flag `voice.backend=v2` 接 VoiceClientV2）。
- 挂拍板：2.5s 灰区手感（压 max_wait 换更快切轮 vs 维持宽容）；backchannel 四态
  是否提前（现单发「嗯」被 min_speech 吞，实际语音对话中影响有限）。
- 挂池：sidecar 饿死 bug 上游报告/修复跟踪；STT 对照（paraformer vs sherpa
  streaming）挂 M4 一起（评测台已可直接复用）。
