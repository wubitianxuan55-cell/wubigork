# 微信助手能力底座侦察（原始稿二，2026-09-02）

目标：为「微信对话里出图、改图、收发文件、多微信并行」摸清内部能力。分类：已可复用 / 有 seam 缺接线 / 完全缺失。

## 一、改图能力

**已可复用**
- 多模式生成内核：`image_handler.go:279-428` `GenerateMedia`（Mode=txt2img|img2img|t2v，InitImage/Denoise/Frames）；`GenerateFreeImage`（158-276）为微信生图现用入口。
- 后端注册表 `internal/ai/image_backend.go:38-77`（kind→factory，init 自注册）：GLM CogView 仅文生图（`image_glm.go:70-73` 显式拒绝 img2img——官方端点无参数，诚实报错）；ComfyUI img2img 仅 krea2/z-image-turbo（`image_comfyui.go:251-258`）；Herdsman 走 `/v1/images/img2img`（`image_openai.go:125-146`）；xAI grok-imagine。
- 参考图改图现成 seam：`characterlib_gen_handler.go:503-560` `CharacterGeneratePortraitWithRef`（参考图+提示词→img2img denoise=0.55）——「给一张图+一句话→改图」最近似实现。
- 产物落盘 CardPath→微信图片卡（真机定稿）：`image_handler.go:431-522` → `intent_router.go:184-200` → `whisper_state.go:77-90` `SendFileCard`。

**有 seam 缺接线**
- 微信入站图片字节已有解密落盘路径（`clawbot.go:330-343` resolveDownload），但只进 OCR 识别，没接生图链路——差一步转成 `GenerateMedia.InitImage`。
- 意图层只有 `ActionGenerateImage`（`intent/intent.go:21`），无「改图」意图。
- 多模态识图（`gaea_vision.go:22-34`）只做理解不做编辑，可复用为改图指令理解。
- roadmap:210,214 已规划「Qwen-Image-Edit / Flux Kontext、蒙版 inpaint、IP-Adapter 参考槽」零实现。

**完全缺失**
- inpaint/蒙版局部重绘（全仓零命中）；云端 image edit 端点（无 qwen-image-edit/flux-kontext 接入）。

## 二、文件收发

**已可复用**
- 入站 type=4 防御留位：`clawbot.go:247-252` fileItem{FileID,URL,Name,Size} + 347-365 防御解析；handle 486-492 只出占位提示行不下载。协议文档 item 枚举已定稿（1文本/2图片/3语音/4文件/5视频）；抓包窗口对 file_item 同样触发。
- 图片全链路可照抄：加密媒体结构/resolveDownload/DownloadImageEncrypted（SSRF+20MiB+魔数，`media_download.go:96-179`）。
- 出站：`media_upload.go:66-193` 图片上传真协议；注释注明枚举 image/video/file/voice=1/2/3/4 但常量仅 image=1；`readUploadableFile` 白名单锁死图片（`clawbot.go:738-751`）。
- 产物中心化：`.gaea/exports`（`spaces.go:67-73`）+ 交付物登记表（`gaea_deliverable_registry.go:25-40` + `trajectory/deliverable.go`，上限 200 条）——微信推送可直接读表取路径。绘梦产物不走登记表（CardPath 直传）。

**缺接线**：file_item 入站下载/类型识别/内容提取（桌面侧 document_import 已有，缺微信侧搬运）；出站 file 卡片构造。
**完全缺失**：出站文件/视频上传协议实装；入站语音处理。

## 三、多微信并行

**已可复用（架构天然支持）**
- 每助手独立凭据+人格（`assistant/manager.go:23-38`，Token DPAPI）；启动遍历全部 enabled 拉起独立 Server（`whisper_state.go:45-47`，`weixinServers map[assistantID]`）；每 Server 独立长轮询 goroutine，`running.Swap` 幂等，**无全局单例锁**；保存即重启对应通道；per-assistant 状态查询/徽标已有。
- 提醒按 assistantID 路由到对应 Server（`weixin_reminder.go:377-386`）。

**真约束/bug**
- 同人格多助手共享 orchestrator（按 `whisper_"+personalityID` 缓存），`AssistantName` 直接写无锁互相覆盖（`whisper_state.go:93-95`）——多号需每号独立 personalityID，或修共享冲突。
- `manager.Update` 只回写 Name/PersonalityID/WxToken/WxUserID/Enabled，WxBotID/PortraitURL/VoiceGuide/Dims 传了也被忽略（`manager.go:196-205`）。
- `FindByWxUser` 联系人路由映射 app 层零调用（死代码 seam）。

**管理体验缺口（后端 CRUD 已绑定、前端没做）**
- WeixinPage 助手列表是只读状态列表：无启停开关/新增删除/逐助手扫码（`confirmBinding` 硬编码 `id:'gaea'`，WeixinPage.tsx:130-147）/portrait 展示；WhisperAssistantList/Save/Delete 前端无管理 UI 调用。

## 四、对话与出图（确认）

链路完整真机已验证：微信文本→提醒路由→意图路由（生图 execGenerateImage→GenerateFreeImage→CardPath→SendFileCard，失败降级）→未命中走 WhisperChatWithSearch（ast.PersonalityID + AssistantName 注入，`whisper_handler.go:503-553`、`whisper_state.go:66-105`）。
