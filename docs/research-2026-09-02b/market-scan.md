# 微信助手市场扫描 v2（原始稿，2026-09-02）

> v1（通道格局）结论仍有效：官方 iLink ClawBot 路线是行业最优解，不换道；hook/协议库路线封号风险高。本版按用户拍板的定位补三轴：**对话式改图、文件收发、多微信并行**（用户定位：通过微信与 gaea 对话进行各项工作——聊天、出图、改图、收发文件、多微信并行）。

## 1. 对话式图像编辑（指令改图）市场

产品级指令改图已是成熟品类，四强格局：
- **Qwen-Image-Edit（阿里）**：百炼 API 现货；多图输入+多图输出；改图内文字、增删/移动物体、主体动作、风格迁移、细节增强；Max 版（2026-01）稳定性/角色一致性更好并集成 LoRA。开源 20B 可本地（ModelScope/HF/NVIDIA NIM）。
- **SeedEdit 3.0（字节）**：主打图像保持力+可用率，1K+ 高清，已集成豆包/即梦 API。
- **Nano-Banana（Google Gemini）**：创意灵活性最强。
- **FLUX Kontext**：上下文理解强，Dev 版开源可本地。
- GLM CogView 无图生图端点（gaea `image_glm.go` 已实证并诚实报错）。
- 场景范式：用户发图+一句自然语言指令→模型返回编辑后图片——与微信对话形态天然契合。

## 2. 文件收发

- **iLink/ClawBot 通道**：openclaw-weixin 逆向文档（weixin-bot-api.md）还原完整协议细节，含 file_item 结构——gaea 抓包可与之互证；OpenClaw 官方渠道页标注微信渠道支持「私信和媒体」。
- **企业微信智能机器人（对照通道）**：官方长连接支持**文字/图片/图文混排/音频/视频/文件**全格式收发——文件收发的合规主通道。
- 生态实证场景（OpenClaw 教程）：发文件给机器人→AI 读取分析；AI 生成文件（PDF/Word/Excel/代码）→直接发回用户。这正是 gaea「办公产物回推」的形态。
- gaea 侧：出站文件上传协议未实装（枚举 file=4 已知）、入站 file_item 只出占位提示——均为抓包+实装工作量，无方向性障碍。

## 3. 多微信并行

- **openclaw-weixin 官方支持多个微信号同时登录、上下文完全隔离**，每号扫码一次即加一个通道（cnblogs/腾讯云教程多篇实证）。
- Hermes Gateway：一套网关多微信号并行，无硬性轮询限制。
- 多 Agent 隔离范式（腾讯云「多个相互独立的 Agent」）：每 Agent 独立人格/技能/记忆/工作空间——gaea 的 assistant.PersonalityID 模型已对齐。
- 结论：多号并行是同通道标配能力，gaea 架构（每助手独立 Server/Token/人格，无全局锁）已具备，**差距在管理体验**（前端无管理台）与两个并发正确性 bug（同人格共享 orchestrator 覆盖 AssistantName、Update 不回写 WxBotID/PortraitURL）。

## 来源

改图：https://help.aliyun.com/zh/model-studio/qwen-image-edit-api · https://qwenlm.github.io/zh/blog/qwen-image-edit/ · https://seed.bytedance.com/zh/blog/bytedance-released-image-editing-model-seededit-3-0-enhanced-image-consistency-and-usability-rate · https://wiro.ai/blog/nano-banana-vs-qwen-flux-kontext-pro-seededit/
文件：https://github.com/hao-ji-xing/openclaw-weixin/blob/main/weixin-bot-api.md · https://docs.openclaw.ai/zh-CN/channels/wechat · https://open.work.weixin.qq.com/help2/pc/21657 · https://zhuanlan.zhihu.com/p/2013706717012186762
多号：https://www.cnblogs.com/itech/p/20140797 · https://blog.csdn.net/hiwangwenbing/article/details/162394044 · https://www.tencentcloud.com/techpedia/142821 · https://juejin.cn/post/7620382299998027826
