# 微信助手板块现状侦察（原始稿，2026-09-02，基线 v4.38.0，绑定面 557）

## 总评

不是壳，是一条真实打通的个人微信通道。接入方式为**直连微信官方 iLink AI Bot API（ClawBot，`https://ilinkai.weixin.qq.com`）的 HTTP 长轮询**，字段对齐腾讯官方开源 `Tencent/openclaw-weixin`——没有用 wechaty / wcferry / 任何非官方协议库。定位是「beta、个人小号实验功能」（README、CHANGELOG P4-1 均明示）。

## 1. 前端入口与 UI

- 板块注册：`internal/app/board/builtins.go:140-146` — `ID: "weixin", Label: "微信助手", Page: "WeixinPage", MenuOrder: 11`（书房触点）；`manifest.go:92` 有 weixin 特判（旧文案残留）。
- 页面本体：`frontend/src/pages/WeixinPage.tsx`（323 行，三卡布局）：
  - 连接卡：助手列表 + 通道状态徽标（未绑定/已停止/运行中/会话过期）+ 扫码绑定。
  - 扫码绑定流（L47-127）：`waiting → scanned → needVerify（配对码）→ confirmed`，2s 轮询，confirmed 后 `WhisperAssistantSave` 落 token 并自动重启通道。
  - 离线代办卡：手动新建/删除、全局回推开关、状态徽标（待触发/已推送/推送失败/重试 n/5）。
- 前端类型/桥：`types.ts:1721-1756`、`bridge.ts:401-425`（12 方法）、`spaceBindings.ts:97-105`（全归 work 空间）。

## 2. 后端实现

自研 Go 直连 iLink 协议（HTTP + JSON，长轮询 getupdates/sendmessage/get_bot_qrcode），协议证据三方印证（真机抓包 + hermes-agent + openilink-sdk-go），见 `docs/ilink-non-text-protocol.md`。

核心包 `internal/channels/weixin/`（测试 2332 行，全真实实现）：
- `clawbot.go`（898 行）：Config/Server/Start-Stop 幂等生命周期/长轮询 pollLoop（sync_buf 续传、errcode=-14 会话过期自愈）/handle（限频→文本/图片/文件分发→OCR 注入→4KB 截断→chatFn→回复）/图片识别管线/主动推送 Push/SendFileCard 产物图片卡片（失败逐级降级文本卡）。
- `qrlogin.go`：扫码登录 + 配对码二次查询。
- `rate_limit.go`：per-peer 滑动窗口 20 条/分钟。
- `media_crypt.go`：手写 AES-128-ECB + PKCS7；`media_download.go`：入站下载（SSRF 防线/20MiB/魔数白名单）；`media_upload.go`：出站图片真协议（getuploadurl → AES → novac2c CDN → image_item）。
- `capture.go`：真机抓包 JSONL 基建。

app 层：
- `internal/app/whisper_state.go`：initWeixin 启动拉起；startAssistantWx 消息回调链 = 提醒路由 → 统一意图路由（navigate/生图/状态/提醒/读屏）→ 轻语聊天 `WhisperChatWithSearch`。
- `internal/app/whisper_handler.go`：12 个绑定面方法。
- `internal/app/weixin_reminder.go`（530 行）：中文时间解析 → JSON 持久化 → 20s ticker 回推 → 重试 ≤5 次。
- wxToken DPAPI 加密落盘（`internal/assistant/manager.go`）。

## 3. 已实现 vs 未实现

真实实现（真机验证过）：扫码绑定全流程、文本收发+AI 回复（意图路由+轻语）、入站图片识别、出站图片卡片、离线代办提醒、主动推送 Push、防线四件套（限频/截断/多媒体上限/SSRF）、会话过期自愈、多助手架构。

占位/未实现：
- **语音（type=3）/视频（type=5）/文件（type=4）消息静默跳过**（clawbot.go:493-496「宁漏勿误」；roadmap §8「语音条→文字」未落地）。
- 无群聊、无好友/联系人管理、无朋友圈、无企业微信/公众号通道（roadmap 列「企业微信合规通道」为下个方向，未开工）。
- **Push 单会话限制**：只回推最近活跃会话（clawbot.go:76-79 注释自认），多联系人靠多助手分 Server。
- 出站文件仅图片白名单（png/jpeg/webp/gif，20MiB），其他产物降级路径文本卡。
- 死代码遗迹：SQLite 曾有 weixin_* 4 张表，schema_v13 已 DROP，现全走 JSON 文件。

## 4. 绑定面 API（12 个）

`WhisperWeixinGetQR / WhisperWeixinQRStatus / WhisperWeixinQRStatusWithCode / WhisperWeixinStatus / WhisperAssistantList / WhisperAssistantSave / WhisperAssistantDelete / WeixinReminderList / WeixinReminderAdd / WeixinReminderDelete / WeixinReminderConfig / WeixinReminderSetConfig`

iLink 端点：getupdates / sendmessage / notifystart|notifystop / get_bot_qrcode / get_qrcode_status / getuploadurl。

## 5. 耦合点

轻语（聊天+人格+记忆+联网搜索）、统一意图路由（微信是三入口之一，产物经 CardPath 回推）、模型中心 vision/OCR（识图三级链）、绘梦（生图产物）、board 板块系统、gaea/secure（DPAPI）、netclient。

## 6. 文档挂账

- roadmap `docs/gaea-nextgen-roadmap-2026.md` §8（L245-264）三跃升：①远程任务入口（语音条→文字——未落地）②定时提醒/主动推送（已落地）③合规降险：企业微信/公众号为主通道、协议层 Seam 可切换（未开工）。
- `docs/ilink-non-text-protocol.md`：入站图片/上传链路「已定稿」，含防线矩阵与 seam 约定。**语音/文件协议无定稿，需真机抓包**。
- T2 排期（L411）「微信任务入口 + 端到端实时语音」。
