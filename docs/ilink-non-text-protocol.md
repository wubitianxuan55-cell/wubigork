# iLink 非文本消息协议探明度备忘（v4.8）

> 状态：**真机待验证**。本文所有字段名/枚举值按 iLink 惯例留位 + 抓包推断，
> 以服务端真实下发为准。代码侧已按「防御解析」落地——单字段怪异形态降级
> 零值，绝不因类型不符让整条消息（乃至整个长轮询批次）解析失败。
> 测试矩阵：`internal/channels/weixin/clawbot_test.go`（TestUnmarshal_DefensiveMatrix /
> TestUnmarshal_FieldCoercion）。

## 1. 入站消息 `getupdates → msgs[].item_list[]`

代码锚点：`internal/channels/weixin/clawbot.go`（inboundMsg / imageItem / fileItem）。

| 字段 | 类型 | 已探明含义 | 多态/异常风险 | 防御策略（已实现） | 状态 |
|---|---|---|---|---|---|
| `type` | int | 1=文本；3=图片（推断）；4=文件（推断） | 可能出现 2/4/99 等未知枚举 | 按 **payload 存在性** 路由（有 image_item 即按图片处理，不依赖 type 值）；未知 type 且无负载 → 宁漏勿误静默跳过 | 真机待验证 |
| `text_item.text` | string | 文本内容 | 超长 | 入站 4KB 截断（`（消息过长已截断）`） | 真机待验证 |
| `image_item.url` | string | 图片下载地址 | **已见 url=数组/对象 形态的可能** | `coerceString`：string 原样、数字转字符串、数组/对象/null 丢弃为空 | 真机待验证 |
| `image_item.file_id` | string | 文件句柄（疑似备用下载/上传通道） | 可能是数字 | `coerceString`（数字 → 去尾零字符串） | 真机待验证 |
| `image_item.name` | string | 文件名 | 超长（100KB 级）、emoji、中文 | 不报错；最终文本受 4KB 截断兜底 | 真机待验证 |
| `image_item.md5` | string | 内容校验 | 可能缺失/数字 | `coerceString`，缺失不报错 | 真机待验证 |
| `file_item.{file_id,url,name,size}` | string/int64 | 文件消息 | size 可能是数字字符串；url 多态 | `coerceString`/`coerceInt64` 双容忍 | 真机待验证 |
| 未探明 item（voice/video/混合卡片…） | ? | 未知 | 未知字段、未知 type | 静默跳过，不喂给模型 | 真机待验证 |

补充约定：
- `item_list` 多媒体条数上限 **5**：超出的不逐个处理，聚合成一行「…等 N 个文件」。
- 整个 image_item/file_item 本身非 JSON 对象（null/数组/标量）时按空负载降级。

## 2. 上行/上传端点（真机窗口最优先）

- `POST /ilink/bot/sendmessage` 目前仅验证了 `text_item`（`clawbot.go` Send）。
- **上传域最有价值线索**：扫码登录响应 `QRStatusResp.baseurl / redirect_host`
  （`internal/channels/weixin/qrlogin.go:31-39`）。这两个字段在扫码成功后由服务端
  下发，极可能就是媒体上传/下载域——iLink 消息里的 `image_item.url` 大概率落在
  同域或其 CDN 子域。

真机窗口要抓的三样东西：

1. **非文本消息原始 JSON 负载**：别人向 bot 发图片/文件，抓 `getupdates` 里
   `item_list[]` 全字段（type 枚举值、url/file_id/name/md5/size 的真实形态）。
2. **扫码登录响应的 `baseurl` / `redirect_host` 值**：记录并对照消息 URL 的域，
   确认上传域假设。
3. **sendmessage 对文件卡片/图片上传的端点与错误码**：尝试上传一次（哪怕失败），
   抓请求路径、参数、响应 errcode——这决定 SendFileCard seam 的真实现。

## 3. 已实现的离线防线（与协议无关，先立住）

| 防线 | 参数 | 位置 |
|---|---|---|
| 入站限频 | per-peer 滑动窗口 20 条/分钟；超限固定文案「消息太频繁，稍后再说」，不触发 LLM | rate_limit.go + handle |
| 文本截断 | 4096 字节（rune 安全），标记「（消息过长已截断）」 | handle |
| 多媒体条数 | 上限 5，超出「…等 N 个文件」 | handle |
| 图片下载 SSRF | dial-time 拒绝私网/回环/链路本地/CGNAT（比 webfetch 更严：回环也拒） | media_download.go |
| 图片尺寸 | 20MiB（Content-Length 预检 + io.LimitReader 双保险） | media_download.go |
| 图片类型 | Content-Type text/* 直接拒；魔数白名单 png/jpeg/webp/gif 终审 | media_download.go |
| 识别输出 | 识别内容截前 300 rune | handle |

## 4. Seam 约定（上传端点探明后只换实现，调用方不动）

- `Server.SendFileCard(localPath, caption string) error`：当前文本降级
  （「🖼 产物已生成：name（微信暂不支持直接收文件卡片，请在桌面端「书房·绘梦」查看）」
  + caption），经 Push 发往最近活跃会话；端点探明后仅替换本方法实现。
- `whisper_state.go` 的「（产物：路径）」内联拼接**暂不迁移**（避免动 app 层），
  待 SendFileCard 真实现落地时一并收拢。
- `Server.MediaRecognizer func(url string) (string, error)`：图片识别注入点；
  app 层一行接线 `srv.MediaRecognizer = weixin.OCRMediaRecognizer(a.GaeaOCRText)`
  （url → DownloadImage（SSRF/尺寸/魔数防线 + 临时文件）→ OCR → cleanup）。
