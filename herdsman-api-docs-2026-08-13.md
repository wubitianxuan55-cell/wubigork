# 接口文档

查看系统API接口文档

> 生成时间: 2026/08/13 08:54:22

## OpenAI Compatible 接入点

### 接入点信息

OpenAI 兼容 API 服务接入点，无需认证

#### 基础 URL

```
http://localhost:8080/v1
```

#### 接入点示例

```
// AI 模型接口
POST http://localhost:8080/v1/chat/completions

// Anthropic 接口
POST http://localhost:8080/v1/anthropic/messages
```

## 模型性能测试接口

### 测试任务生命周期

创建异步模型性能测试，并通过轮询查询进度、取消或删除任务，以及导出 Markdown/CSV 报告。全局同一时间只允许一个测试任务运行。

> 该接口不校验身份。开启局域网访问后，任何能访问 API 端口的客户端都可以启动、取消、删除测试，并读取本机路径和运行端口。

```http
POST   /api/benchmarks
GET    /api/benchmarks?status=succeeded&limit=50&offset=0
GET    /api/benchmarks/:id
POST   /api/benchmarks/:id/cancel
DELETE /api/benchmarks/:id
GET    /api/benchmarks/:id/export?format=markdown|csv
```

#### 请求示例

```json
{
  "model_name": "qwen3.5-9b",
  "variants": ["standard"],
  "context_sizes": [4096],
  "warmup_count": 0,
  "repeat_count": 1,
  "request": {
    "user_prompt": "Please respond with a short benchmark-friendly answer.",
    "max_tokens": 256,
    "stream": true,
    "timeout_seconds": 1800
  }
}
```

#### 轮询任务状态

创建请求返回 202 和 Location 后，轮询详情接口，直到 status 为 succeeded、failed 或 canceled。

```http
HTTP/1.1 202 Accepted
Location: /api/benchmarks/{id}

GET /api/benchmarks/{id}
```

#### 状态码

| HTTP | Meaning |
| ---: | --- |
| 202 | Benchmark queued |
| 204 | Benchmark deleted |
| 400 | Invalid request |
| 404 | Run not found |
| 409 | Active benchmark conflict |
| 500 | Internal error |

## OpenAI 兼容接口

### 获取模型列表

获取所有可用的模型列表

**请求方法:** GET
**接口地址:** /v1/models

#### 请求示例

```http
GET /v1/models
```

#### 响应示例

```json
{
  "object": "list",
  "data": [
    {
      "id": "llama3-8b",
      "object": "model",
      "created": 1677858242,
      "owned_by": "Herdsman",
      "status": "running"
    }
  ]
}
```

### 对话补全

发送对话请求，获取AI回复

**请求方法:** POST
**接口地址:** /v1/chat/completions

#### 参数

| 参数名 | 类型 | 必填 | 描述 |
|--------|------|------|------|
| model | string | ✓ | 模型名称 |
| messages | array | ✓ | 对话消息列表 |
| temperature | number | ✗ | 采样温度 |
| max_tokens | number | ✗ | 最大生成 tokens 数 |
| top_p | number | ✗ | 核采样概率 |
| stream | boolean | ✗ | 是否使用流式响应 |
| reasoning_effort | string | ✗ | OpenAI Chat Completions 兼容推理级别，可选值：low、medium、high；本地 llama.cpp 会映射到模板参数 |
| thinking_enabled | boolean | ✗ | Herdsman 兼容扩展：为支持的模型启用或关闭思考模式；本地 llama.cpp 会映射为 enable_thinking |
| thinking_tokens | number | ✗ | Herdsman 兼容扩展：思考 token 预算；本地 llama.cpp 会映射为 reasoning_budget |

#### 请求示例

```http
POST /v1/chat/completions
Content-Type: application/json

{
  "model": "llama3-8b",
  "messages": [
    {
      "role": "user",
      "content": "Hello, how are you?"
    }
  ],
  "temperature": 0.7,
  "max_tokens": 1000,
  "reasoning_effort": "high"
}
```

#### 响应示例

```json
{
  "id": "chatcmpl-123",
  "object": "chat.completion",
  "created": 1677858242,
  "model": "llama3-8b",
  "system_fingerprint": "fp_44709d6fcb",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "I'm doing well, thank you! How can I help you today?"
      },
      "logprobs": null,
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 13,
    "completion_tokens": 17,
    "total_tokens": 30
  }
}
```

### 向量化接口

将文本转换为向量表示

**请求方法:** POST
**接口地址:** /v1/embeddings

#### 参数

| 参数名 | 类型 | 必填 | 描述 |
|--------|------|------|------|
| model | string | ✓ | 模型名称 |
| input | string/array | ✓ | 输入文本或文本数组 |
| encoding_format | string | ✗ | 嵌入向量编码格式 |

#### 请求示例

```http
POST /v1/embeddings
Content-Type: application/json

{
  "model": "llama3-8b",
  "input": "Hello world"
}
```

#### 响应示例

```json
{
  "object": "list",
  "data": [
    {
      "object": "embedding",
      "index": 0,
      "embedding": [
        0.0023064255,
        -0.009327664,
        0.015790065,
        // ... 更多嵌入值
      ]
    }
  ],
  "model": "llama3-8b",
  "usage": {
    "prompt_tokens": 2,
    "total_tokens": 2
  }
}
```

### Rerank 接口

对文档进行重新排序

**请求方法:** POST
**接口地址:** /v1/rerank

#### 参数

| 参数名 | 类型 | 必填 | 描述 |
|--------|------|------|------|
| model | string | ✓ | 模型名称 |
| query | string | ✓ | 查询文本 |
| documents | array | ✓ | 文档列表 |
| top_n | number | ✗ | 返回的最大结果数 |

## Anthropic 接口

### Anthropic 对话

使用 Anthropic 模型进行对话

**请求方法:** POST
**接口地址:** /v1/anthropic/messages

#### 参数

| 参数名 | 类型 | 必填 | 描述 |
|--------|------|------|------|
| model | string | ✓ | 模型名称 |
| messages | array | ✓ | 对话消息列表 |
| temperature | number | ✗ | 采样温度 |
| max_tokens | number | ✗ | 最大生成 tokens 数 |

#### 请求示例

```http
POST /v1/anthropic/messages
Content-Type: application/json

{
  "model": "claude-3-opus-20240229",
  "messages": [
    {
      "role": "user",
      "content": "Hello, how are you?"
    }
  ],
  "temperature": 0.7,
  "max_tokens": 1000
}
```

## AI 模型接口

### 图片生成接口

根据提示词生成图片

**请求方法:** POST
**接口地址:** /v1/images/generations

#### 参数

| 参数名 | 类型 | 必填 | 描述 |
|--------|------|------|------|
| prompt | string | ✓ | 图片生成提示词 |
| model | string | ✗ | 模型名称 |
| n | number | ✗ | 生成图片数量 |
| size | string | ✗ | 图片尺寸 |

### 图片编辑接口

编辑现有图片

**请求方法:** POST
**接口地址:** /v1/images/edits

#### 参数

| 参数名 | 类型 | 必填 | 描述 |
|--------|------|------|------|
| image | file | ✓ | 图片文件或图片数据 |
| prompt | string | ✓ | 图片生成提示词 |
| mask | file | ✗ | 掩码图片文件 |
| model | string | ✗ | 模型名称 |
| n | number | ✗ | 生成图片数量 |
| size | string | ✗ | 图片尺寸 |

### 图生图接口

根据现有图片生成新图片

**请求方法:** POST
**接口地址:** /v1/images/img2img

#### 参数

| 参数名 | 类型 | 必填 | 描述 |
|--------|------|------|------|
| image | file | ✓ | 图片文件或图片数据 |
| prompt | string | ✓ | 图片生成提示词 |
| model | string | ✗ | 模型名称 |
| n | number | ✗ | 生成图片数量 |
| size | string | ✗ | 图片尺寸 |

### OCR 识别接口

识别图片中的文本，返回整页文本、逐行结果、置信度和文本框坐标

**请求方法:** POST
**接口地址:** /v1/ocr

#### 支持的模型

| 模型名称 | 描述 |
|---|---|
| paddleocr-ppocrv5-server | PaddleOCR PP-OCRv5 Server 文本检测与识别模型 |

#### 参数

| 参数名 | 类型 | 必填 | 描述 |
|--------|------|------|------|
| model | string | ✓ | OCR 模型名称，目前支持 paddleocr-ppocrv5-server |
| image_base64 | string | ✓ | 图片 base64 数据，支持纯 base64 或 data:image/...;base64 格式 |

#### 请求示例

```http
POST /v1/ocr
Content-Type: application/json

{
  "model": "paddleocr-ppocrv5-server",
  "image_base64": "data:image/png;base64,iVBORw0KGgo..."
}
```

#### 响应示例

```json
{
  "text": "识别出的整页文本",
  "lines": [
    {
      "text": "单行识别文本",
      "score": 0.98,
      "box": [[12, 20], [180, 20], [180, 42], [12, 42]]
    }
  ],
  "image_width": 640,
  "image_height": 360,
  "elapsed_ms": 1327
}
```

### 文档解析接口

使用 MinerU 将 PDF、图片和 Office 文档转换为 Markdown 与结构化文档解析结果；PDF/图片可通过 mode 在 Pipeline 与 Hybrid 模式间切换，Office 文档由 Pipeline Office 转换器处理。

**请求方法:** POST
**接口地址:** /v1/documents/parse

#### 支持的模型

| 模型名称 | 描述 |
|---|---|
| minerU | MinerU Pipeline + Hybrid 文档解析模型 |

#### 支持的输入

Pipeline 支持 PDF；图片：png、jpg、jpeg、bmp、gif、tiff、webp；Office：docx、pptx、xlsx。Hybrid 仅支持 PDF 和图片输入；Office 文档请求 mode=hybrid 会返回 400，不会回退。Rust 运行时当前不支持 JP2。

#### 参数

| 参数名 | 类型 | 必填 | 描述 |
|--------|------|------|------|
| model | string | ✓ | 文档解析模型名称，目前支持 minerU |
| file | file | Multipart 二选一 | 通过 multipart/form-data 上传的文档文件，支持 PDF、png、jpg、jpeg、bmp、gif、tiff、webp、docx、pptx、xlsx；file 与 path 二选一 |
| path | string | JSON 必填 / Multipart 二选一 | Herdsman 服务可访问的本地文档路径，支持与 file 上传相同的文件类型；JSON 请求必填，multipart 请求可用它替代 file |
| mode | string | ✗ | 解析模式：pipeline 或 hybrid，默认 pipeline；hybrid 需要模型目录内存在 Qwen3-VL 文件，且仅接受 PDF/图片输入；Office 输入使用 pipeline 的 Office 转换器 |
| format | string | ✗ | 响应格式：json、text、markdown 或 md，默认 json；text/markdown/md 返回 Markdown 文本 |
| dpi | number | ✗ | PDF 渲染 DPI，默认 200 |
| formula | boolean | ✗ | Pipeline 模式是否启用公式识别，默认 true |
| effort | string | ✗ | Hybrid 模式推理强度：low、medium 或 high，默认 medium |
| max_tokens | number | ✗ | Hybrid 模式最大生成 token 数，默认 2048 |

#### 请求示例

```http
POST /v1/documents/parse
Content-Type: application/json

{
  "model": "minerU",
  "path": "C:\\Users\\me\\Documents\\sample.docx",
  "format": "json",
  "mode": "pipeline"
}

POST /v1/documents/parse
Content-Type: multipart/form-data

model=minerU&file=@sample.png&format=text&mode=hybrid
```

#### 响应示例

```json
{
  "model": "minerU",
  "text": "提取出的完整文档文本",
  "markdown": "提取出的完整文档文本",
  "pages": [
    {
      "page_number": 1,
      "text": "单页提取文本",
      "text_items": []
    }
  ],
  "metadata": {
    "page_count": 1,
    "elapsed_ms": 1520,
    "ocr_enabled": true,
    "runtime": "mineru",
    "input_format": "docx",
    "parser": "office",
    "ocr_image_count": 0,
    "unsupported_features": []
  }
}
```

### 图片缓存接口

获取缓存的图片文件

**请求方法:** GET
**接口地址:** /v1/images/cache/:filename

### 语音识别接口

将语音转换为文本

**请求方法:** POST
**接口地址:** /v1/audio/transcriptions

#### 参数

| 参数名 | 类型 | 必填 | 描述 |
|--------|------|------|------|
| model | string | ✓ | 模型名称 |
| audio | string | JSON 必填 | 音频输入，支持本地路径、URL 或 data:audio/...;base64 数据 |
| file | file | Multipart 必填 | multipart/form-data 上传的音频文件 |
| language | string | ✗ | 语言 |

#### 请求示例

```http
POST /v1/audio/transcriptions
Content-Type: multipart/form-data

model=whisper-base&file=@audio.wav&language=zh

POST /v1/audio/transcriptions
Content-Type: application/json

{
  "model": "whisper-base",
  "audio": "data:audio/wav;base64,UklGRi...",
  "language": "auto"
}
```

#### 响应示例

```json
{
  "text": "这是一段语音识别的结果",
  "language": "zh",
  "duration": 3.42
}
```

### 流式语音识别接口

通过 WebSocket 进行实时语音识别

**请求方法:** GET
**接口地址:** /v1/audio/transcriptions/stream?model={model}

#### 参数

| 参数名 | 类型 | 必填 | 描述 |
|--------|------|------|------|
| model | query string | ✓ | 支持实时 ASR 的模型名称 |

#### 请求示例

```http
GET /v1/audio/transcriptions/stream?model=sherpa-onnx-streaming-zipformer-zh-14m
Upgrade: websocket

// client -> server: PCM16/16k mono binary frames
// server -> client: {"text":"实时识别结果","is_final":false}

GET /v1/audio/transcriptions/stream?model=funasr
Upgrade: websocket

// FunASR 使用原生 WebSocket 音频协议，适合实时中文识别
```


### 语音合成接口

将文本转换为语音

**请求方法:** POST
**接口地址:** /v1/audio/speech

#### 支持的模型

| 模型名称 | 描述 |
|---|---|
| qwen3-tts-customvoice | 预置说话人模式，使用 voice 或 speaker 选择声音 |
| qwen3-tts-voicedesign | 声音设计模式，使用 voice_description 描述声音风格 |
| qwen3-tts-voiceclone | 声音克隆模式，使用 ref_audio 和可选 ref_text 提供参考音频 |

#### 参数

| 参数名 | 类型 | 必填 | 描述 |
|--------|------|------|------|
| model | string | ✓ | 模型名称 |
| input | string | ✓ | 输入文本 |
| voice | string | ✗ | 语音类型 |
| speaker | string | ✗ | 说话人 ID；未设置时使用 voice |
| voice_description | string | ✗ | VoiceDesign 模式的自然语言声音描述 |
| ref_audio | string | ✗ | VoiceClone 模式的参考音频，支持路径、URL 或 base64 |
| ref_text | string | ✗ | VoiceClone 模式的参考音频文本 |
| language | string | ✗ | 语言 |
| speed | number | ✗ | 语速 |
| stream | boolean | ✗ | 为 true 时返回 stream_url，而不是一次性合成结果 |
| frames | number | ✗ | Qwen-TTS 可选最大音频帧数 |

#### 请求示例

```http
POST /v1/audio/speech
Content-Type: application/json

{
  "model": "qwen3-tts-customvoice",
  "input": "这是一段文字转语音的测试",
  "voice": "Cherry",
  "language": "Chinese",
  "speed": 1.0
}

{
  "model": "qwen3-tts-voicedesign",
  "input": "这是一段文字转语音的测试",
  "voice_description": "温暖、自然、语速适中，适合中文播客旁白",
  "language": "Chinese"
}

{
  "model": "qwen3-tts-voiceclone",
  "input": "这是一段文字转语音的测试",
  "ref_audio": "data:audio/wav;base64,UklGRi...",
  "ref_text": "参考音频对应的文本",
  "language": "Chinese"
}
```

#### 响应示例

```json
{
  "audio_url": "/audio/20260516_abc123.wav",
  "sample_rate": 24000,
  "duration": 2.38
}
```

### 流式语音合成接口

先通过语音合成接口创建流式任务，再使用返回的 stream_url 拉取音频流

**请求方法:** GET
**接口地址:** /v1/audio/speech/stream/:token

#### 请求示例

```http
POST /v1/audio/speech
Content-Type: application/json

{
  "model": "edge-tts",
  "input": "这是一段文字转语音的测试",
  "voice": "zh-CN-YunxiNeural",
  "stream": true
}

// 响应示例
{
  "stream_url": "/v1/audio/speech/stream/550e8400-e29b-41d4-a716-446655440000"
}

GET /v1/audio/speech/stream/550e8400-e29b-41d4-a716-446655440000
```

#### 响应示例

```http
// 二进制音频流响应
// Content-Type: audio/mpeg | audio/wav | application/octet-stream
// Transfer-Encoding: chunked
```

### 音频服务信息

按模型查询音频服务能力信息

**请求方法:** GET
**接口地址:** /v1/audio/info?model={model}

#### 参数

| 参数名 | 类型 | 必填 | 描述 |
|--------|------|------|------|
| model | query string | ✓ | 音频模型名称，例如 qwen3-tts-customvoice、whisper-base、edge-tts |

#### 请求示例

```http
GET /v1/audio/info?model=qwen3-tts-customvoice

GET /v1/audio/info?model=whisper-base

GET /v1/audio/info?model=funasr
```

#### 响应示例

```json
{
  "tts_supported_languages": [
    "Chinese",
    "English"
  ],
  "supported_speakers": [
    "Cherry",
    "Ethan"
  ]
}

{
  "asr_supported_languages": [
    "zh",
    "en",
    "ja"
  ]
}

{
  "asr_supported_languages": [
    "zh",
    "zh-CN"
  ]
}
```

