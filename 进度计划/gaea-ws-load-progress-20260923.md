# gaea WS 载入阶段进度可读化（v4.405.0）规格书

> 2026-09-23 立项。来源：v4.404.1 分诊后续——用户拍板「可以」（20 分钟冷载
> 期间界面零反馈，做「载入模型 xx%」可读进度）。纯 Go 1 文件+前端 1 标签，
> 绑定面 717 零变更。目标版本：v4.405.0。

## 1. 论点

v4.404.1 分诊坐实：krea2 冷载 20.5 分钟期间，界面无任何「在干活」的证据
（WS 只处理带 value/max 的 progress 事件——载入阶段 value/max 恒 0，被
`Max>0` 过滤整段丢弃）。用户看到的是静止的进度条，无法区分「慢慢载入」
和「卡死」。

## 2. 裁决

| # | 裁决 |
|---|---|
| Q1 | WS 订阅补三类事件（此前只有 progress/progress_state）：**executing**（节点开始执行，无百分比）→ node 名经 nodeClasses 映射透出（percent=-1，前端既有 COMFY_NODE_LABELS 显示「加载模型」+不定态进度条+用时增长）；node=null（整单结束）忽略 |
| Q2 | **status**（queue_remaining）→ 前有任务（>1）时回调 queued+node="queue"（排队可见）；前端标签「排队中（前有任务）」 |
| Q3 | **progress_state max=0 兜底**：优先报带进度的运行节点；全部无 max 时退而报第一个 running 节点名（不再整段丢弃） |
| Q4 | 后端不发中文字符串（维持 class_type 契约），可读化由前端既有标签表承担——零重复维护 |
| Q5 | 已知小瑕疵不动：loader 无百分比时 percent 保留上一帧（如采样后 VAE 阶段显示 100%+不定态条+节点名+用时，真值可读）；percent=-1 语义=「未知保留上次」是既有契约 |

## 3. 测试

- Go `TestPollComfyProgressReadable`（gorilla 假 WS 服务脚本序列）：排队
  回调 queued/queue/-1；executing 透出 UNETLoader 无百分比；progress 透传
  KSampler 37%；progress_state max=0 透出 VAELoader；外来 prompt 与
  node=null 零回调。
- imagegen 前端 21 文件 128 例回归绿。

## 4. 门禁

全量 ci、版本漂移闸 OK@4.405.0、绑定面 717 零漂移。

## 5. 观察池

ComfyUI 载入百分比的真实可得性（新版 progress_state 载入条）；iGPU 内存
压力预检（提交前提示冷载风险）；取消按钮显性化。
