# 当日总账（2026-11-05）：语音 V2 唯一路径化大日——卡死根修 + 网关/V1 双退役 + 协议四修 + 交互重设计 + bundle v2

> 批末=桌面 exe build/bin/gaea.exe（C:\AI\gaea-app 装机同哈希），桌面入口收敛为
> 单个 gaea.lnk 快捷方式（散件 exe/3 DLL/gaea-go.bat 出清）。今日一笔主提交
> （语音生产代码全量）+ bundle v2 一笔 + 文档一笔 + release 一笔，均随本版列车。

## 提交索引（4 笔）

| 提交 | 线 | 一句话 |
|---|---|---|
| （C1）feat(voice)! | 语音全栈 | V2 唯一路径化：卡死根修（三环连环）+ 8792 网关退役 + V1 whisper 链退役 + 协议四修（409 退避/兜底转写/terminal 发布/task schema）+ 交互重设计 + 真实电平/哑麦检测 |
| （C2）feat(voice) | 分发 | bundle v2：可重定位净布局（exe+三 DLL+models/voice+装机脚本）+ gaea_install.ps1 一键装机 |
| （C3）docs(voice) | 台账 | 审计台账/主计划/本日总账 |
| （C4）chore(release) | 发版 | v4.532.0（wails.json+versioninfo.rc 对齐） |

## 卡死根修（用户报「语音系统启动连接卡死」三连重启同死局）

三环连环，CDP 沙箱实锤（9 次 WS/8.1s 全被弹）：
1. **并发 connect 竞态**：ensure 就绪瞬间事件/轮询/按钮三路同 tick 进入
   connect，引用 guard 在 ensureSink await 后才置位 → 三个 WebSocket 同建。
2. **孤儿会话锁死服务端**：第一个 WS 抢到会话槽（池=1）后被引用覆盖成
   孤儿（无人 close，浏览器 pong 保活 → 60s 读超时杀不掉），后续连接全被
   session_limit_reached 秒弹且服务端零日志。
3. **onOpen 清零重连计数**：每次被弹都重数 1/5 → 无限 1s 重连循环。

修：connect 并发去重（connectingRef 首个 await 前置位）+ 重连计数稳定
10s 窗复位（RECONNECT_STABLE_MS，翻转「open 即清零」固化测试）+ 服务端
60s 零客户端帧空闲接管（lastInbound 原子戳；接管等旧 epilogue 防配置抹除）
+ 拒绝留痕日志（incumbent_idle）。

## 网关退役（刀6 核心）

voice_v2_gateway.go -444 行；voiceNativeLLMLeg 进程内腿吸收注入逻辑
（prepare 台账/【本轮上下文】独立消息保 prefix cache/plain 长期记忆/失败
+取消双回滚逐字同位）；路由恒走聊天功能绑定；voice_chat_llm radio 退役；
status 缩 {realtime,breeze}。用户 voice_chat_llm=chat 实况=行为零变化。

## V1 whisper 链退役（用户拍板）

前端 V1 组件/hook/设置 V1 卡/模型中心 ASR 半边全删；后端 voice_handler
17 函数/voice_manager(1681 行)/internal/asr/云 realtime/config 三键出清；
朗读链与麦克风设备工具保留（micDevices util 迁出）；绑定 710→709。

## 协议四修（ws_probe 实弹驱动）

1. TTS 腿 breeze 409（共享引擎抢占）退避重试 400/800/1600ms。
2. F1 兜底语进转写面 + 终态前冲刷停靠 delta（原 done 转写空白）。
3. F2 hasResponse 响应期 user terminal 即时发布（speech_stopped 上线）。
4. TaskTool 全 Schema 补 engine/model（compact_drift 逮住的既有漂移）。

R3 零输出判定野外生效：TTS 全灭回合诚实判 cancelled/empty_output。

## 交互重设计 + 真实电平（ui-ux-pro-max 检索落地）

- bar 相位电平条（listening/speaking 起伏·thinking 呼吸·error/interrupted
  语义色）+ preflight「装配中」态 + 错误重试标签 + 过时文案修正。
- captureProcessor RMS 240ms 上报 → micLevel/micSilent（开麦 3s 无信号=
  哑麦警示）——**实弹当场抓真问题：本机默认录音设备=零信号虚拟设备（ToDesk）**。

## 时序坑（记档防复发）

- bargein 插话语料必须全长（短语料 384ms 阈值丢弃，sidecar 同律）。
- TTS 秒发音频：done 与插话确认只差 27ms——长首句造料恢复原 PASS 口径。
- 沙箱 LLM「登录已过期」=环境噪声非管线病（兜底路径正好被实弹验证）。
- tsc 必须 -b 口径；vitest 必须 frontend/ cwd 净跑（仓库根扫 clones/.tmp 假红）；
  管道 rc 谎报（`cmd | head` 的 rc 是 head 的）。

## 遗留

- 真机一句话验收（用户侧）；M5 §7 四项拍板；误打断率语料门（venv 出局后
  A/B 基线改 Go 自证口径待定）。
