# iLink 非文本消息协议备忘（v4.8.3 图片真机定稿 + v4.9 文件定稿）

> 状态：**真机抓包定稿 + 开源 SDK 双重印证**。入站图片/文件消息字段与上传
> 全链路已由三方交叉验证：① 本机真机抓包（`%LocalAppData%\gaea\wx_capture.jsonl`
> 的 inbound_media 样例 + 密文实测解密；文件样例 2026-09-02 16:58）；②
> hermes-agent（NousResearch）`weixin.py` 生产实现；③ openilink-sdk-go/-python
> 导出符号（ENCRYPT_AES128_ECB / GetUploadURLReq / UploadMediaType）。
> 解析侧仍保留防御式降级（宁漏勿误），但协议形态不再是猜测。

## 1. 入站消息 `getupdates → msgs[].item_list[]`（图片）

代码锚点：`internal/channels/weixin/clawbot.go`（imageItem / cdnMedia / resolveDownload）。

| 字段 | 真机实测形态 | 说明 | 状态 |
|---|---|---|---|
| `type` | **2 = 图片**（1=文本、3=语音、4=文件、5=视频，hermes 枚举） | 早期文档猜 3=图片是错的 | 已定稿 |
| `image_item.aeskey` | string，32 位 hex | 媒体解密密钥（hex → 16 字节 AES-128） | 已定稿 |
| `image_item.media.encrypt_query_param` | string | CDN 下载票据（也出现在 full_url query 里） | 已定稿 |
| `image_item.media.aes_key` | string，**base64(hex 字符串)** | 同 aeskey 的另一编码形态；**不是** base64(原始字节) | 已定稿 |
| `image_item.media.full_url` | string | `https://novac2c.cdn.weixin.qq.com/c2c/download?encrypted_query_param=…&taskid=…` | 已定稿 |
| `image_item.{mid_size,hd_size,thumb_size,thumb_width,thumb_height}` | int | 密文/明文/缩略图尺寸（hd_size=明文长度） | 已定稿 |
| `image_item.{url,file_id,name,md5}` | 真机图片消息**不下发** | 保留为兼容留位字段 | — |
| `msg.context_token / root_id / parent_id` | string | 会话上下文 | 已定稿 |

明文 = `AES-128-ECB 解密(hexdecode(aeskey)) + PKCS7 去填充`；真实 CDN 明文在
图片 EOI 后可能带少量尾随字节（实测 24B），交解码器按 EOI 截断即可。

## 2. 入站文件 `file_item`（v4.9 真机抓包定稿 2026-09-02 16:58）

> 证据：`%LocalAppData%\gaea\wx_capture.jsonl` 最新 inbound_media（seq=28 批次，
> 样例「专家评审打分报告_中科一兵.docx」，433849 字节实测）。**file_item 与
> image_item 完全同构**——同一张 CDN 下载票据 + 同一种 AES 密钥编码，直接复用
> DownloadImageEncrypted 同款解密（`media_crypt.go`）。

代码锚点：`internal/channels/weixin/clawbot.go`（fileItem / fileItemLabel）+
`media_download.go`（resolveInboundFile / resolveFileDownload）。

| 字段 | 真机实测形态 | 说明 | 状态 |
|---|---|---|---|
| `type` | **4 = 文件** | hermes 枚举 1=文本 2=图片 3=语音 4=文件 | 已定稿 |
| `item.is_completed` | bool `true` | 完成标记（暂不消费） | 留观 |
| `file_item.file_name` | string（中文原样） | 完整文件名 | 已定稿 |
| `file_item.md5` | string，32 位 hex | **明文** MD5（下载解密后比对） | 已定稿 |
| `file_item.len` | **string** `"433849"` | ⚠ **字符串形态数字**——解析按数字/字符串双容忍（coerceInt64），出站回发用数字 | 已定稿 |
| `file_item.media.encrypt_query_param` | string | CDN 下载票据 | 已定稿 |
| `file_item.media.aes_key` | string，**base64(hex 字符串)** | 与图片 aes_key 同口径：base64 解出 hex 字符串再 hex 解码得 16 字节 key | 已定稿 |
| `file_item.media.full_url` | string | `https://novac2c.cdn.weixin.qq.com/c2c/download?encrypted_query_param=…&taskid=…` | 已定稿 |
| `file_item.{file_id,url,name,size}` | 真机文件消息**不下发** | 旧留位字段保留兼容（防御解析矩阵继续覆盖） | — |

下载防线（文件不能魔数白名单，改为）：
- **声明大小预检**：len/size 超 **50MiB** 直接拒绝，不发起下载；下载再经
  Content-Length 预检 + `io.LimitReader` 双保险（`fetchMediaBytesLimit`）；
- **SSRF**：与图片同一条 dial-time 防线（私网/回环/链路本地/CGNAT 恒拒）；
- **MD5 校验策略**：解密后明文 MD5 与 `file_item.md5` 比对——对上则可信；
  **对不上仅 Warn 保留，不拒收**（微信 CDN 偶发差异以实测为准）；
- **临时文件策略**：落 `os.TempDir()/gaea-wxfile-*.tmp`；`FileHandler` 返回后
  即删（处理器如需留存字节自行复制，同 OnInboundImage 语义）；
- **零流量开关**：`Server.FileHandler == nil` 时不下载，直接占位提示行。

## 3. 上传全链路（SendFileCard 真实现）

代码锚点：`internal/channels/weixin/media_upload.go` + `media_crypt.go`。
上传域与扫码登录的 `baseurl/redirect_host` **无关**（v4.8.2 的媒体域假设作废）。

```
① filekey = 随机 32 hex；aeskey = 随机 16B；rawfilemd5 = md5(明文)；
   filesize = PKCS7 对齐后大小
② POST {api}/ilink/bot/getuploadurl          （鉴权头同 apiPost）
   body: {filekey, media_type:1(图片), to_user_id, rawsize, rawfilemd5,
          filesize, no_need_thumb:true, aeskey:hex, base_info}
   → resp {upload_param? , upload_full_url?}
③ upload_url = upload_full_url（优先）
            || {cdn}/upload?encrypted_query_param={urlencode(upload_param)}&filekey={filekey}
   （cdn = https://novac2c.cdn.weixin.qq.com/c2c；hermes 实测 upload_full_url
    的旧 PUT 会 404——统一用 POST）
④ 明文 → AES-128-ECB(PKCS7) 加密 → POST upload_url
   body = 密文原样，Content-Type: application/octet-stream
   → 200 + 响应头 x-encrypted-param（即下载票据）
⑤ sendmessage（msg 信封同 Send）：
   item_list[0] = {type:2, image_item:{
       media:{encrypt_query_param:票据, aes_key:base64(hex字符串), encrypt_type:1},
       mid_size: len(密文)}}
   ⚠ aes_key 必须 base64(hex 字符串)；base64(原始字节) 接收端解出灰框
   （hermes 生产注释实测教训）
⑥ caption 为独立文本消息（图先文后，图片失败时整体降级单条文本卡片，避免
   图文重复推送）
```

任何失败（无活跃会话/文件白名单外/getuploadurl/CDN/sendmessage）逐节点抓包
（`upload_probe`：stage=upload / send_image_card / delivered，或 skipped=no_peer）
并降级现有文本卡片——产物回推绝不崩主流程。

### 3.1 文件卡（v4.9 探针制）

代码锚点：`media_upload.go`（uploadFileToCDN / uploadMediaBytes / sendFileCardViaUpload）。
`SendFileCard` 按扩展名分流：图片白名单（png/jpeg/webp/gif）走 §3 图片链
（media_type=1，逐字节不变）；**非图片**（docx/xlsx/pptx/pdf/zip/txt/md 等，
无扩展名白名单、上限 50MiB）走文件卡链，与图片五步完全同构，差异仅：

- getuploadurl 请求 **media_type=3**（按 image/video/file/voice=1/2/3/4 枚举；
  ⚠ **探针制假设，待真机验证**，upload_probe 已埋点，真机一跑便知）；
- sendmessage 发 `type=4 file_item`（每次尝试完整记录请求/响应/errcode）：

```json
{"type":4, "file_item":{
  "file_name":"评审报告.docx",
  "len":433849,                     // ⚠ 数字（明文字节数；入站真机为字符串，出站发数字）
  "md5":"<明文 MD5 32hex>",
  "media":{"encrypt_query_param":"<x-encrypted-param 票据>",
           "aes_key":"<base64(hex字符串)>",   // 与图片口径一致，勿发 base64(原始字节)
           "encrypt_type":1}}
```

- 上传内核 `uploadMediaBytes`（probeFile 非空即文件链）逐节点抓
  `upload_probe`：`file_getuploadurl`（请求字段 rawsize/rawfilemd5/filesize/
  filekey/aeskey + 响应原文或 err）→ `file_cdn_upload`（ticket_len 或 err）→
  `file_send_file_card`（含 errcode 的响应原文或 err）→ `delivered`（card=file）；
- **任何失败逐级降级**：file 卡 → 文本卡（§3 现状链），绝不影响主流程。

## 4. 已实现的离线防线

| 防线 | 参数 | 位置 |
|---|---|---|
| 入站限频 | per-peer 滑动窗口 20 条/分钟；超限固定文案「消息太频繁，稍后再说」，不触发 LLM | rate_limit.go + handle |
| 文本截断 | 4096 字节（rune 安全），标记「（消息过长已截断）」 | handle |
| 多媒体条数 | 上限 5，超出「…等 N 个文件」 | handle |
| 图片下载 SSRF | dial-time 拒绝私网/回环/链路本地/CGNAT（比 webfetch 更严：回环也拒） | media_download.go |
| 图片尺寸 | 20MiB（Content-Length 预检 + io.LimitReader 双保险） | media_download.go |
| 图片类型 | Content-Type text/* 直接拒；魔数白名单 png/jpeg/webp/gif 终审 | media_download.go |
| 解密媒体 | 密文无魔数——先全量下载（SSRF/尺寸防线不变）→ AES-128-ECB 解密 → 魔数终审 → 落盘；解密失败不落盘 | media_download.go DownloadImageEncrypted |
| 文件下载 SSRF | 与图片同一条 dial-time 防线（私网/回环/链路本地/CGNAT 恒拒，重定向每跳复检） | media_download.go resolveInboundFile |
| 文件尺寸 | **50MiB**：声明 len/size 预检（超限不下载）+ Content-Length/LimitReader 双保险 + 解密后长度复核 | media_download.go |
| 文件完整性 | 明文 MD5 与 file_item.md5 比对：对上可信；**对不上 Warn 保留不拒收**（微信 CDN 偶发差异以实测为准） | resolveInboundFile |
| 文件落盘 | os.TempDir()/gaea-wxfile-*.tmp；FileHandler 返回后即删；aes_key 非法显式报错（绝不把密文当明文落盘） | saveInboundFile / processInboundFile |
| 文件触发面 | FileHandler nil 时不下载（零流量）直接占位；处理器 panic/空串 recover 后回退占位行（宁漏勿误） | clawbot.go processInboundFile |
| file:// 本地流 | 仅限 os.TempDir() 前缀 + 魔数终审 + no-op cleanup；只服务本包解密产物流转，消息内 URL（恒 http(s)）进不来 | media_download.go |
| 识别输出 | 识别内容截前 300 rune | handle |

## 5. Seam 约定（v4.8.3/v4.9 已落地，调用方零改动）

- `Server.SendFileCard(localPath, caption string) error`：签名不变；内部按
  扩展名分流——图片白名单走真机定稿图片卡链，非图片走 §3.1 文件卡链（探针制），
  任何失败降级现有文本卡片。
- `InboundFileHandler func(localPath, fileName string, sizeBytes int64, md5sum string) string`
  + `Server.FileHandler`（v4.9 跨线契约，app 线注入）：入站 file_item 下载解密
  落盘后同步调用，返回值作为注入文本行原样拼入喂模型的消息（建议自带括号包装，
  对齐图片识别提示行风格）；临时文件在处理器返回后即删——留存字节请自行复制；
  nil/panic/空串一律回退 `fileItemLabel` 占位提示行。注入与文本提取由 app 线
  负责，weixin 包只定义与调用。
- `whisper_state.go` 的「（产物：路径）」内联拼接**维持不迁移**（不动 app 层）。
- `Server.MediaRecognizer func(url string) (string, error)`：识别注入点不变；
  v4.8.3 起「发图即识别」对真机加密图片生效——weixin 包先下载解密落盘，再以
  `file://` 交识别器（DownloadImage 支持 file:// 读本包解密产物），
  app 层现有注入 `srv.MediaRecognizer = weixin.OCRMediaRecognizer(a.GaeaOCRText)`
  零改动即走通（url → DownloadImage（http(s)/file:// 双形态 + 防线）→ OCR → cleanup）。
- 测试矩阵：`capture_test.go`（真协议端到端/各失败节点降级/文件卡链探针端到端）、
  `media_upload_test.go`（AES 原语/aes_key 双形态/真机形态解析/解密下载/
  入站加密图片全链路识别）、`media_file_test.go`（file_item 真机形态防御解析/
  resolveInboundFile 防线矩阵/FileHandler 契约）、`media_download_test.go`
  （SSRF/魔数/file://）。
