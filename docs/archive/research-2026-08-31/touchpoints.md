# 触点层 · 微信 + 语音 + 指令中枢（v4.11.0 基线复扫 · 2026-08-31）

> 调研方法：cn.bing / GitHub API / 官方文档站实际检索与抓取（2026-08-31）。国内搜索引擎对「微信机器人/封号」话题结果受合规过滤，相关结论已标注未核实。

## 8. 触点层 · 微信 + 语音 + 指令中枢（v4.11.0 基线复扫 · 2026-08-31）

### 市场格局 · 最新动态

**端到端实时语音 API 已成红海，国产可直连且计费透明。** OpenAI gpt-realtime 于 2025-08-28 GA：音频输入 $32/百万 tokens、输出 $64/百万 tokens，32k 上下文，支持 WebRTC/WebSocket/SIP（developers.openai.com/api/docs/models/gpt-realtime）。国内三家可直连：①阿里百炼 Qwen3.5-Omni-Realtime（plus/flash）：WebSocket/WebRTC/AOQ 三协议，支持 semantic_vad 语义打断、对话内语音控制（"语速快一些"）、113 语种识别与声音复刻；官方实测单轮总响应约 5.1 秒；音频 token 换算输入=秒×7、输出=秒×12.5，单价在控制台（help.aliyun.com/zh/model-studio/realtime）。②智谱 GLM-Realtime（flash-9B/air-32B）：wss 接入，server/client VAD、实时打断、Function Calling；按分钟计费，flash 音频 0.18 元/分钟、air 0.3 元/分钟（docs.bigmodel.cn/cn/guide/models/sound-and-video/glm-realtime.md）。③字节豆包实时语音模型 3.0「Seeduplex」原生全双工，wss duplex 协议 + response.cancel 打断，支持 8 种方言与复刻音色，2026-04 发布（volcengine.com/docs/6561/2549778；zhuanlan.zhihu.com/p/2025601158207521490）；API 单价未核实。MiniMax 实时语音 API 细节未核实，仅见 Music 模型 2026-08-20 停服公告（platform.minimaxi.com/subscribe/token-plan）。

**微信个人号 bot：hook 类灰色生态收缩，iLink 生态走强。** hook 类代表 WeChatFerry 已归档停更（6.8k★，最后推送 2026-07，github.com/lich0821/WeChatFerry），wechaty 更新停滞（2025-12 后无推送，github.com/wechaty/wechaty）；而面向 AI Agent 的「微信 iLink Bot SDK」持续活跃（622★，github.com/corespeed-io/wechatbot）。未发现微信官方「个人微信 AI 助理」合规通道（未核实，搜索结果受过滤）。企业微信官方转向拥抱 Agent：wecom-cli 2026-03-29 开源，「让人类和 AI Agent 都能在终端中操作企业微信」，长连接机器人可装 skills，覆盖消息/文档/会议/待办，官方明示幻觉与数据外泄风险（open.work.weixin.qq.com/help2/pc/21676；github.com/WecomTeam/wecom-cli，2.9k★）。

**「任何入口唤起同一个助理」已成默认玩法。** 开源侧 OpenClaw（38.8 万★，"Any OS. Any Platform."）与 hermes-agent（23.9 万★）把多入口个人助理做成基础设施（api.github.com，2026-08-31 检索）；产品侧腾讯元宝推电脑版「随时随地唤起、一键划词」，与手机 App、微信小程序构成多端（yuanbao.tencent.com/evt/promo/23c396e6dc400a72ee75384474bcb04a）；ChatGPT Windows 主打读取应用/文件/屏幕上下文（apps.microsoft.com/detail/9plm9xgg6vks）。

### 范式迁移（上轮调研以来的变化）

上轮基线（2025 末）：Realtime API 多为半双工 + 静音阈值 VAD，ASR+LLM+TTS 拼接管线仍是主流。2026 年四个迁移：①级联架构向端到端统一网络「全面转型」（k.sina.cn/article_7879923802_1d5ae185a06801g1ru.html，2026-08-03）；②全双工 + 语义打断成旗舰标配：Seeduplex（2026-04）、NVIDIA 开源 PersonaPlex 与 VoiceChat 11B（插话即让出话语权，cloud.tencent.com/developer/news/3809407；sohu.com/a/1061284680_122396381）、哈工大 Lychee-FD（2026-07，163.com/dy/article/L1VD13P30511AQHO.html）；③工程难点从「压延迟」转向 Voice Runtime——连续交互的回合状态管理（juejin.cn/post/7660896500881866804，2026-07-12）；④语义 VAD 普及：附和声/背景音不再误触发打断（Qwen realtime 文档）。

### 对 gaea 的机会与威胁

机会：gaea 的 Realtime 代码（openai provider、16k→24k 重采样、TurnControl）与主流事件协议高度同构（commit / cancel / speech_started），换引擎边际成本低；GLM 按分钟计费使个人成本可预算（0.18 元/分钟≈月重度使用十元级）；微信 iLink 通道与 OpenClaw 生态同构，且图片链路真机定稿是稀缺资产；wecom-cli 提供合规的「第二 IM 入口」。威胁：①Whisper+LLM+TTS 拼接管线相对端到端已可感落后一代，打断自然度差距最直观；②微信通道政策风险未除——hook 类项目归档说明通道可能随时失效，封号 2026 新政未核实，须有降级预案；③OpenClaw/hermes 在「多入口同一助理」上生态先发；④元宝把「桌面唤起 + 划词」做成免费国民级功能，抬高了桌面入口心智门槛。

### 优先级建议（现在 0-3 月 / 下个 3-6 月 / 愿景 6-12 月）

- **0-3 月**：真机验证 Realtime 全链路（先跑通 openai provider 与打断）；指令内核先落 Ctrl+K 命令面板；继续微信语音/视频消息抓包取样；成本口径入库（GLM 按分钟、OpenAI/Qwen 按 audio token）。
- **3-6 月**：接入第二个国产 provider——优先评估 Qwen3.5-Omni-Realtime（semantic_vad/语义打断与 TurnControl 对齐）或 GLM-Realtime（按分钟计费可预算）；打断体验升级到语义级；试点 wecom-cli 作为办公/团队入口；为 iLink 通道加风控（白名单、节流、人格分句播报的拟人化节流）与封号降级预案。
- **6-12 月**：全双工/免唤醒连续交互实验；「任何入口、任何模态唤起同一个 gaea」对标 OpenClaw 多平台与元宝多端完成体验闭环。

### 参考来源

1. OpenAI gpt-realtime 模型文档：https://developers.openai.com/api/docs/models/gpt-realtime
2. 阿里云百炼 Qwen-Omni 实时模型文档：https://help.aliyun.com/zh/model-studio/realtime
3. 智谱 GLM-Realtime 指南（含价格）：https://docs.bigmodel.cn/cn/guide/models/sound-and-video/glm-realtime.md
4. 智谱 GLM-Realtime AsyncAPI（wss 协议）：https://docs.bigmodel.cn/cn/asyncapi/realtime.md
5. 火山引擎豆包端到端实时语音（全双工版本）：https://www.volcengine.com/docs/6561/2549778
6. Seeduplex 官方页：https://seed.bytedance.com/zh/seeduplex ；发布文：https://zhuanlan.zhihu.com/p/2025601158207521490
7. MiniMax Token Plan（Music 停服公告）：https://platform.minimaxi.com/subscribe/token-plan
8. WeChatFerry（已归档）：https://github.com/lich0821/WeChatFerry ；wechaty：https://github.com/wechaty/wechaty
9. 微信 iLink Bot SDK for OpenClaw：https://github.com/corespeed-io/wechatbot
10. 企微 wecom-cli 帮助文档：https://open.work.weixin.qq.com/help2/pc/21676 ；仓库：https://github.com/WecomTeam/wecom-cli
11. 腾讯元宝电脑版推广页：https://yuanbao.tencent.com/evt/promo/23c396e6dc400a72ee75384474bcb04a
12. ChatGPT Windows（微软商店）：https://apps.microsoft.com/detail/9plm9xgg6vks
13. 级联→端到端转型报道：https://k.sina.cn/article_7879923802_1d5ae185a06801g1ru.html
14. GPT-Live 全双工语音 Agent 拆解：https://juejin.cn/post/7660896500881866804
15. NVIDIA PersonaPlex：https://cloud.tencent.com/developer/news/3809407 ；VoiceChat 11B：https://www.sohu.com/a/1061284680_122396381
16. Lychee-FD 全双工开源：https://www.163.com/dy/article/L1VD13P30511AQHO.html
17. 豆包/字节 API 价格汇总（2026-08-30 更新）：https://apirank.vip/zh/providers/bytedance/
