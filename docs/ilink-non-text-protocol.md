# iLink 非文本消息协议备忘（v4.8.3 真机定稿）

> 状态：**真机抓包定稿 + 开源 SDK 双重印证**。入站图片消息字段与上传全链路
> 已由三方交叉验证：① 本机真机抓包（`%LocalAppData%\gaea\wx_capture.jsonl`
> 的 inbound_media 样例 + 密文实测解密）；② hermes-agent（NousResearch）
> `weixin.py` 生产实现；③ openilink-sdk-go/-python 导出符号
> （ENCRYPT_AES128_ECB / GetUploadURLReq / UploadMediaType）。
> 解析侧仍保留防御式降级（宁漏勿误），但协议形态不再是猜测。

## 1. 入站消息 `getupdates → msgs[].item_list[]`

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

## 2. 上传全链路（SendFileCard 真实现）

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

## 3. 已实现的离线防线

| 防线 | 参数 | 位置 |
|---|---|---|
| 入站限频 | per-peer 滑动窗口 20 条/分钟；超限固定文案「消息太频繁，稍后再说」，不触发 LLM | rate_limit.go + handle |
| 文本截断 | 4096 字节（rune 安全），标记「（消息过长已截断）」 | handle |
| 多媒体条数 | 上限 5，超出「…等 N 个文件」 | handle |
| 图片下载 SSRF | dial-time 拒绝私网/回环/链路本地/CGNAT（比 webfetch 更严：回环也拒） | media_download.go |
| 图片尺寸 | 20MiB（Content-Length 预检 + io.LimitReader 双保险） | media_download.go |
| 图片类型 | Content-Type text/* 直接拒；魔数白名单 png/jpeg/webp/gif 终审 | media_download.go |
| 解密媒体 | 密文无魔数——先全量下载（SSRF/尺寸防线不变）→ AES-128-ECB 解密 → 魔数终审 → 落盘；解密失败不落盘 | media_download.go DownloadImageEncrypted |
| file:// 本地流 | 仅限 os.TempDir() 前缀 + 魔数终审 + no-op cleanup；只服务本包解密产物流转，消息内 URL（恒 http(s)）进不来 | media_download.go |
| 识别输出 | 识别内容截前 300 rune | handle |

## 4. Seam 约定（v4.8.3 已落地，调用方零改动）

- `Server.SendFileCard(localPath, caption string) error`：真协议实现
  （getuploadurl → CDN 密文上传 → image_item 卡片 → caption 独立补发）；
  app 层调用签名未动。
- `whisper_state.go` 的「（产物：路径）」内联拼接**维持不迁移**（不动 app 层）。
- `Server.MediaRecognizer func(url string) (string, error)`：识别注入点不变；
  v4.8.3 起「发图即识别」对真机加密图片生效——weixin 包先下载解密落盘，再以
  `file://` 交识别器（DownloadImage 支持 file:// 读本包解密产物），
  app 层现有注入 `srv.MediaRecognizer = weixin.OCRMediaRecognizer(a.GaeaOCRText)`
  零改动即走通（url → DownloadImage（http(s)/file:// 双形态 + 防线）→ OCR → cleanup）。
- 测试矩阵：`capture_test.go`（真协议端到端/各失败节点降级）、
  `media_upload_test.go`（AES 原语/aes_key 双形态/真机形态解析/解密下载/
  入站加密图片全链路识别）、`media_download_test.go`（SSRF/魔数/file://）。
