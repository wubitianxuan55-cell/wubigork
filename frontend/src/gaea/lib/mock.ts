// bridge/mock.ts — 浏览器开发模式的 mock 实现。
// T6-10.1 拆分后本文件为聚合入口：各域实现按 makeMockApp 方法归属拆分至
// mock/ 目录（chat/core/cost/memory/model/office/retrieval/settings + shared/state），
// 本文件仅组装 makeMockApp 并 re-export 既有导出——bridge.ts 及全部测试的
// import 路径零改动。
//
// 场景系统：通过 URL 参数切换 mock 行为，无需修改代码。
//   ?mock=fresh       空状态：无工作区、无会话、无 API key
//   ?mock=running     模拟活跃流式输出（工具执行中 / 思考中）
//   ?mock=approval    审批卡流程：tool_dispatch → approval_request 挂起，
//                     Approve 后补发工具结果并收尾（审批卡离线可开发）
//   ?mock=ask         提问卡流程：ask_request 挂起，
//                     AnswerQuestion 后继续收尾（提问卡离线可开发）
//   ?mock=compaction  压缩卡流程：compaction_started → compaction_done → 继续
//   ?mock=demo        默认：完整 mock 数据（等同于不传参数）
//   ?platform=darwin|windows|linux 覆盖平台检测
//
// 缓存安全: 纯前端 mock，不触及 Go 内核。

import type { AppBindings } from "./bridge";
import { buildChat } from "./mock/chat";
import { buildCore } from "./mock/core";
import { buildCost } from "./mock/cost";
import { buildMemory } from "./mock/memory";
import { buildModel } from "./mock/model";
import { buildOffice } from "./mock/office";
import { buildRetrieval } from "./mock/retrieval";
import { buildSettings } from "./mock/settings";
import { createMockState } from "./mock/state";

// 既有导出（含测试辅助）原样 re-export。
export {
  __resetPriceMocksForTest,
  browserPlatformOverride,
  emitMock,
  mockListeners,
  mockScenario,
  mockSubscribe,
  mockTaskListeners,
  mockTaskSubscribe,
  taskView,
  updaterListeners,
} from "./mock/shared";

export function makeMockApp(): AppBindings {
  const state = createMockState();
  return Object.assign(
    {},
    buildCore(state),
    buildChat(state),
    buildMemory(state),
    buildCost(state),
    buildOffice(state),
    buildRetrieval(state),
    buildModel(state),
    buildSettings(state),
  );
}
