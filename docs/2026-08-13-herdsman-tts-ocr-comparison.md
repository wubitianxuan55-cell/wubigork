# Herdsman 语音合成 / OCR 模型接入与对照测试

> 日期：2026-08-13
> 参考：`herdsman-api-docs-2026-08-13.md`
> 本机 Herdsman：`http://localhost:8080/v1`

## 一、本次补齐的调用

| 能力 | 端点 | 模型 | 本次处理 |
|---|---|---|---|
| 语音合成 | `POST /v1/audio/speech` | `qwen3-tts-customvoice` | 已有，继续保留 |
| 语音合成 | `POST /v1/audio/speech` | `qwen3-tts-voicedesign` | 已有，继续保留 |
| 语音合成 | `POST /v1/audio/speech` | `qwen3-tts-voiceclone` | 新增 `ref_audio` / `ref_text` 支持 |
| 语音合成 | `POST /v1/audio/speech` | `edge-tts` | 已纳入模型路由，模型中心可识别为 TTS |
| OCR | `POST /v1/ocr` | `paddleocr-ppocrv5-server` | 新增 Herdsman OCR 客户端，并接入「提取文字」回退链 |
| 文档解析 | `POST /v1/documents/parse` | `minerU` | 新增客户端；图片 OCR 时作为第二回退 |
| 模型分类 | `/v1/models` | `embedding` / `rerank` / `ocr` | 补齐 `bge-m3`、`bge-reranker-v2-m3`、`paddleocr`、`minerU` 分类 |

## 二、代码变更

- `internal/tts/herdsman.go`
  - 新增 `NewHerdsmanTTSWithClone`，构造 `voiceclone` 请求体。
  - `buildBody` 统一处理 voiceclone / voicedesign / customvoice 三种参数。
  - `resolveVoice`、`SupportedSpeakers` 对 voiceclone 不查询音色列表。
- `internal/app/voice_handler.go`
  - `tryEngineTTS` 对 `voiceclone` 模型读取 `HERDSMAN_TTS_REF_AUDIO` / `HERDSMAN_TTS_REF_TEXT`。
- `internal/app/gaea_ocr.go`
  - `GaeaOCRText` 现在按顺序回退：Herdsman PaddleOCR → Herdsman MinerU → OvisOCR2。
- `internal/ocr/`
  - 新增 Herdsman OCR/文档解析客户端及单元测试。
- `internal/modelengine/engine.go` 与前端模型中心
  - 模型分类补充 `funasr`、`voxcpm2`、`ocr`、`rerank`、`embedding`。

## 三、验证

### 自动化测试

```powershell
go test ./internal/tts ./internal/ocr ./internal/modelengine ./internal/app
go test ./...
cd frontend; npm.cmd run build
```

结果：全部通过，前端 TypeScript/Vite 生产构建成功。

### 本机真实服务测试

`/v1/models` 当前暴露：

```text
Qwen3.6-35B-A3B-DSV4Pro-SFT-GPT56Sol-RL-Agent-LynnStyle-Q5-imatrix
Qwen3.6-35B-A3B-Uncensored-HauhauCS-Aggressive-Q4_K_P-2
bge-m3
bge-reranker-v2-m3
edge-tts
minerU
paddleocr-ppocrv5-server
sherpa-onnx-streaming-zipformer-zh-14m
voxcpm2
zimage-turbo
```

#### OCR：PaddleOCR vs 现有 OvisOCR2

测试图片文字：`这是一段用于 Herdsman OCR 对比测试的文字`

| 模型 | 结果 | 耗时（热） | 说明 |
|---|---:|---|---|
| OvisOCR2（旧回退） | `这是一段用于 Herdsman OCR 对比测试的文字` | 503 ms | 文字完全正确 |
| Herdsman PaddleOCR | `这是一段用十HerdsmanOcR对比测试的文字` | 89 ms | 更快，但中文错字/大小写错误较多 |

结论：当前字体/中英文混合场景下，OvisOCR2 质量更稳；PaddleOCR 速度占优。因此
`GaeaOCRText` 采用 PaddleOCR 优先、OvisOCR2 兜底，失败自动切换，兼顾速度与质量。

#### 文档解析 MinerU

已启动并验证通过：

```text
POST /v1/documents/parse
{"model":"minerU","path":"...herdsman_ocr_test.png","format":"json","mode":"pipeline"}
```

响应：

```text
"## 这 是 一 段 用 于 Herdsman OCR 对 比 测 试 的 文 字\n\n"
elapsed_ms=207（热调用）
```

结论：MinerU Pipeline 图片解析可用，但中文输出按字间加了空格，且保留 Markdown 标题符号；
因此当前图片 OCR 仍以 PaddleOCR/OvisOCR2 为主，MinerU 作为第二回退。

#### TTS

| 模型 | 本机结果 |
|---|---|
| `qwen3-tts-customvoice` | `model qwen3-tts-customvoice is not installed` |
| `qwen3-tts-voicedesign` | `model qwen3-tts-voicedesign is not installed` |
| `qwen3-tts-voiceclone` | `model qwen3-tts-voiceclone is not installed` |
| `edge-tts` | `edge tts returned empty audio`（服务端合成失败，疑似上游网络/语音服务问题） |
| `voxcpm2` | 可用：`audio/wav`，约 261 KB；首次冷合成约 50s |

`voxcpm2` 的请求约束实测：

- `voice_description` 或不传 `voice`：成功。
- 传 `voice` / `speaker`：服务端返回
  `VoxCPM2 C++ session requires speaker reference audio, not a cached voice id`。

因此代码已将 `voxcpm2` 归入“无预设音色”的 TTS 路由，并在本地上传模型上放宽 HTTP 超时到
180 秒，避免冷启动被 30 秒超时误杀。

结论：三个 Qwen3-TTS 模型本机仍缺失；`edge-tts` 服务端仍返回空音频；`voxcpm2` 已可端到端合成。

## 四、后续建议

1. 在 Herdsman 侧安装 `qwen3-tts-customvoice` / `qwen3-tts-voicedesign` /
   `qwen3-tts-voiceclone` 后跑真实 TTS 对比。
2. 修复或配置 `edge-tts` 上游，确认空音频问题。
3. 继续补齐 Qwen3-TTS 三个模型后做真实音质对照。
4. 若需在 UI 暴露 OCR/文档解析模型选择，可在模型中心新增专用分类页。
