# gaea 角色库生成进度接线（v4.406.0）规格书

> 2026-09-23 立项。来源：用户指令「继续」——今天图像等待体验线程（
> v4.404.1 超时根修/v4.405 载入可见）收尾：绘梦有进度条，**角色库编辑器的
> 剧照/设定卡生成只能盲转**。纯接线，绑定面 717 零变更。目标版本：v4.406.0。

## 1. 论点

设定卡/剧照走 qedit/img2img 同样可能撞上 20 分钟冷载，编辑器按钮 loading
转圈零信息也无处可去。绘梦的进度管线（WS→updateComfyTaskProgress→
GetComfyUITaskProgress 轮询面）已完备——角色库生成链挂上同一管线即可，
零新绑定。

## 2. 裁决

| # | 裁决 |
|---|---|
| Q1 | Go：`attachComfyProgress(req, backend)` 助手——comfyui 后端预置 queued+挂 updateComfyTaskProgress 回调；其他后端不挂（无实时进度语义）。剧照（characterGeneratePortrait）与设定卡（CharacterGenerateSheet）两链同挂，`defer clearComfyTaskProgress()` 防陈旧残留 |
| Q2 | 前端：CharacterLibEditor 在 genPortrait/sheetGen 忙碌期间 1s 轮询 getComfyUITaskProgress（既有 api），hero 卡左上角进度行（不定态语义：当前节点中文名+用时/排队中）；空闲即隐藏 |
| Q3 | 可读化复用 v4.405 的 COMFY_NODE_LABELS（后端 class_type 契约不变） |
| Q4 | sin 侧设定卡入口（handleCastSheet）不接 UI 进度（挂池——面板空间另议）；评分/补全走文本 LLM 不涉 ComfyUI 不接 |
| Q5 | 相对路径写入源清点**关账**（同日调查结论）：`.gaea/attachments` @mention 语法与办公交付卡本就以工作区相对形态为规范存储——相对是设计不是事故，v4.405.1 读侧解析即正解，写入侧绝对化是错方向 |

## 3. 测试

- Go `TestAttachComfyProgress`（comfyui 挂回调+queued 预置/herdsman 不挂/
  清理后快照空）；既有 sheet/portrait/ref-slot 全链回归（newCharacterLib
  TestApp 构造器补 mediaState 初始化——attachComfyProgress 使 nil 嵌入显形）。
- 前端：生成中进度行出现（加载模型+用时）、结束消失（mock 快照+可控 promise）。

## 4. 门禁

全量 ci、版本漂移闸 OK@4.406.0、绑定面 717 零漂移。

## 5. 观察池

角色库生成取消（需接 beginImageGen 可取消机制，与队列互斥语义另议）；sin
侧进度行；评分 LLM 流式反馈。
