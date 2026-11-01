# Gaea Voice V2 — M2 协议固化与无模型回放报告

> 日期：2026-11-01
> 承接：[M1 Runtime Spike 报告](./gaea-voice-v2-m1-spike-report-20261101.md)
> 方案：[V2 重新架构总方案](./gaea-voice-v2-rearchitecture-20261101.md) §14 M2

## 1. 结论

**M2 退出条件达成：不需要真实模型即可完整回放并断言事件有序性。**

- 前端协议层 + 会话状态机 + 播放队列 + Realtime 客户端全部落地为纯 TS
  （`frontend/src/voice/`），**29/29 单测绿**（无 DOM/无网络/无模型）。
- 浏览器实证 **10/10 断言绿**（CDP Edge + vite dev）：
  - mock 回放（M1 实录事件流灌入）：全句转写渲染、助手文本、
    播放回执 `resp_turn_1 播144B 弃0B ✓回执`——**144B 全部经真实 AudioWorklet 音频栈播出**；
  - 真服务握手：连 `ws://127.0.0.1:8791/v1/realtime`（M1 sidecar）session.created/updated 正常、零协议错误。
- 门禁全绿：tsc 零错、eslint 零告警、knip 零回归、`go vet`/golangci（M1 侧）零 issue。

## 2. 交付物（全部未提交，V1 一行未动）

```text
frontend/src/voice/
  protocol/events.ts            事件类型固化 + decode/encode + 宽容未知事件
  session/voiceSession.ts       纯 reducer 相位机 + orderingViolations 有序性断言面
  session/fixtures.ts           M1 实录三场景事件流（turn/cancel/bargein）+ 两条违序负例
  playback/pcmQueue.ts          代际闸播放队列（open/append/markClosed/flush/回执）
  client/VoiceClientV2.ts       transport 无关客户端（WebSocket/回放桩同构）
  worklets/captureProcessor.js  采集：128 帧量子→320 样本(20ms@16k) Int16 postMessage
  worklets/sinkProcessor.js     播放：环形缓冲(4s)+欠载静音+溢出上报
  dev/VoiceV2Lab.tsx            ?voicev2=1 实验页（MainLayout 懒挂载，独立 chunk）
测试：voiceSession.test.ts(16) / pcmQueue.test.ts(7) / VoiceClientV2.test.ts(6)
走查：.tmp/voice_v2/walk.mjs（CDP 9223 + 127.0.0.1:5173，10 断言）
```

## 3. 固化下来的语义（实测纠偏后的最终口径）

| 语义 | 口径 | 来源 |
|---|---|---|
| 事件有序性 | session.created 居首；speech started/stopped 计数配对；response 帧/done 不得早于 created；**终态后同 id 帧=泄漏（断言恒 0）** | `orderingViolations`，M1 实测序 |
| 相位机 | speech_started→LISTENING；response.created→THINKING；首 audio.delta→SPEAKING；done(completed)→IDLE；done(cancelled/**turn_detected**)→INTERRUPTED（随后 speech_started 接管）；done(client_cancelled)→IDLE | M1 服务端发送序：**旧 response 完整闭合先于插话 speech_started** |
| 助手文本 | delta 累积显示，`transcript.done` **全量权威替换**（非拼接漂移） | 协议 README + 实测 |
| 播放队列 | 代际生命周期=协议驱动：`response.created→open(id)`（旧存量作废）、`audio.delta→append`（非当前代际只记账）、`response.done→markClosed`；**非 completed 终态（打断/取消/失败）未播块立即冲刷**，不等下一代际 | M2 测试期两轮纠偏 |
| 播放回执 | **闭合且排空**一次性触发，携带 response id（含 flush 收尾路径），禁全局布尔 | V2 方案 §10.2 |
| 麦克风 | AudioWorklet 20ms/320 Int16@16k；无 ScriptProcessor=无 IndexSizeError 雷；约束阶梯保留 | V2 方案 §10.1 + 884e9eb3 |

## 4. 坑（本轮新增）

1. **块注释里写路径通配 `*/` 会提前闭合注释**（fixtures.ts `…/voice_v2/*/report…`）——esbuild 报 `Unexpected "."`，定位在注释内不易想到。
2. knip 门禁：懒挂载页面**只准 default export**（命名+default 双导出=Duplicate exports 回归）；worklets 用 `new URL(...,import.meta.url)` 动态加载不进依赖图，暂未被 knip 报（观察项）。
3. CDP 走查点击前必须**轮询等挂载**（vite 首载预打包 >10s，固定 sleep 3.5s 点了个空）。
4. 实验页 mock 回放会占用 client 槽位，真连接前需 dispose 让位（已修）。

## 5. 状态与下一步

- **M3（收窄后）**：Smart Turn 中文 A/B（vs FireRed Chinese EOT）+ 端点参数区间调优；
  语料用 M1 探针造料线（breeze 合成中文测试语音）扩展多场景。
- **M4**：Gaea 对话大脑接入（`/internal/voice/chat/completions` + 人格/记忆写回）+
  **语音专属快模型路由**（`voice_chat_engine/model`，35B 只验管线不作延迟口径）+
  ChatPage/VoiceConsole 接入实验页验证过的 VoiceClientV2（feature flag `voice.backend=v2`）。
- **M5**：推测生成/软结束 reopen/false-interruption 恢复（sidecar 已有雏形，前端协议已留扩展位）。
- 挂用户：真机麦克风全链（实验页开麦→8791 真服务→说话→breeze 回放）——本轮走查用假设备只证了握手与 mock 链。
