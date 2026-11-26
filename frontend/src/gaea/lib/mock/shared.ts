// mock/shared.ts — mock 域的共享状态与工具（T6-10.1 拆分自 lib/mock.ts）。
// 承载跨域共享的模块级内存态与事件总线；可变绑定通过 setter 暴露
// （ESM 禁止跨模块重绑定 let），方法体语义与原实现完全一致。

import type {
  TaskView,
  UpdateProgress,
  WireEvent,
} from "../types";

// 对齐 Go app 层 eventChannel（GaeaMeta：internal/app/gaea_ui_meta.go）。
export const EVENT_CHANNEL = "gaea-event";

// 浏览器开发 mock 的固定资料清单（内存态，对应后端 .gaea/pinned.json）。
export let pinnedMock: string[] = [];
export function setPinnedMock(v: string[]) {
  pinnedMock = v;
}

// 浏览器开发 mock 用的最小 docx（含标题/正文/表格），由 docx-preview 渲染。
export const MOCK_DOCX_DATA_URL =
  "data:application/vnd.openxmlformats-officedocument.wordprocessingml.document;base64,UEsDBBQAAAAIAEmjCV3XeYTq8QAAALgBAAATAAAAW0NvbnRlbnRfVHlwZXNdLnhtbH2QzU7DMBCE730Ky9cqccoBIZSkB36OwKE8wMreJFb9J69b2rdn00KREOVozXwz62nXB+/EHjPZGDq5qhspMOhobBg7+b55ru6koALBgIsBO3lEkut+0W6OCUkwHKiTUynpXinSE3qgOiYMrAwxeyj8zKNKoLcworppmlulYygYSlXmDNkvhGgfcYCdK+LpwMr5loyOpHg4e+e6TkJKzmoorKt9ML+Kqq+SmsmThyabaMkGqa6VzOL1jh/0lSfK1qB4g1xewLNRfcRslIl65xmu/0/649o4DFbjhZ/TUo4aiXh77+qL4sGG71+06jR8/wlQSwMEFAAAAAgASaMJXSAbhuqyAAAALgEAAAsAAABfcmVscy8ucmVsc43Puw6CMBQG4J2naM4uBQdjDIXFmLAafICmPZRGeklbL7y9HRzEODie23fyN93TzOSOIWpnGdRlBQStcFJbxeAynDZ7IDFxK/nsLDJYMELXFs0ZZ57yTZy0jyQjNjKYUvIHSqOY0PBYOo82T0YXDE+5DIp6Lq5cId1W1Y6GTwPagpAVS3rJIPSyBjIsHv/h3ThqgUcnbgZt+vHlayPLPChMDB4uSCrf7TKzQHNKuorZvgBQSwMEFAAAAAgASaMJXV5N4XGjAQAAcwMAABEAAAB3b3JkL2RvY3VtZW50LnhtbJVTS0vEMBC++ytCTnrQrA9Elm1FD549KJ7bNO5W26QkcaueFhEUPBRRfKEs6kFvPkBYRPDHuM3qvzB9iQ921ctkpvPNfDNf0srkqu+BOuHCZdSAw0MlCAjFzHFp1YDzczODExAIaVHH8hglBlwjAk6afZWw7DC84hMqge5ARTk0YE3KoIyQwDXiW2KIBYTq3CLjviV1yKsoZNwJOMNECE3ge2ikVBpHvuVSaPYBoLvazFlL3DQIzMTM8uRYwiAs1y3PgFiTEg6RWUEf2cxkvo0SK9YL/Oh4js3z0lTN/ThqtR9P2s9nndPm28Xm69U26PcZXh5IgDKDZwRfp8k7vD4fq6Mb9RDpwvj4On5qxNGuuj6Pn6J2a6ezfwe0PKuDASd1l4RAte5Vc69zsqlOG/FtBBa0DkAdbKnzy5fGRndKaXuZX0T5BrY3rTvoW0sjFhSrJqp6BIJkfQOOwVQJm0nJ/J4QjyzKngDuVmu9EOjrUKgY9tPw2eTY/KZkvH0Ipn4qkH7B3Uumu5eglOx35uH/sY78iTFfPX3NqHjOiVf8LuY7UEsBAhQAFAAAAAgASaMJXdd5hOrxAAAAuAEAABMAAAAAAAAAAAAAAIABAAAAAFtDb250ZW50X1R5cGVzXS54bWxQSwECFAAUAAAACABJowldIBuG6rIAAAAuAQAACwAAAAAAAAAAAAAAgAEiAQAAX3JlbHMvLnJlbHNQSwECFAAUAAAACABJowldXk3hcaMBAABzAwAAEQAAAAAAAAAAAAAAgAH9AQAAd29yZC9kb2N1bWVudC54bWxQSwUGAAAAAAMAAwC5AAAAzwMAAAAA";

// 浏览器开发 mock 用的 xlsx 结构化预览（含公式/样式/合并/多 sheet）。
export const MOCK_XLSX_BODY = JSON.stringify({
  sheets: [
    {
      name: "预算",
      rows: [
        [{ ref: "A1", value: "项目", type: "string", style: { bold: true, fill: "4472C4", fontColor: "FFFFFF", align: "center", border: true } },
         { ref: "B1", value: "金额", type: "string", style: { bold: true, fill: "4472C4", fontColor: "FFFFFF", align: "center", border: true } },
         { ref: "C1", value: "阶段", type: "string", style: { bold: true, fill: "4472C4", fontColor: "FFFFFF", align: "center", border: true } }],
        [{ ref: "A2", value: "设备", type: "string" }, { ref: "B2", value: "120.50", type: "number", style: { numFmt: "0.00%" } }, { ref: "C2", value: "设计", type: "string" }],
        [{ ref: "A3", value: "人工", type: "string" }, { ref: "B3", value: "80", type: "number", style: { numFmt: "0.00%" } }, { ref: "C3", value: "施工", type: "string" }],
        [{ ref: "A4", value: "合计", type: "string", style: { bold: true } }, { ref: "B4", value: "200.50", formula: "SUM(B2:B3)", type: "string", style: { bold: true } }, { ref: "C4", value: "完成", type: "string" }],
        [{ ref: "A5", value: "合并单元格（mock）", type: "string" }],
      ],
      merged: ["A5:B5"],
      colWidths: { A: 16, B: 14, C: 12 },
      // v4.106 条件格式走查样例：金额>100 → 红底红字加粗（CellIs 静态子集）
      condRules: [{ range: "B2:B4", op: "greaterThan", formulas: ["100"], priority: 1, fill: "FFC7CE", fontColor: "9C0006", bold: true }],
    },
    {
      name: "明细",
      rows: [
        [{ ref: "A1", value: "日期", type: "string", style: { bold: true } }, { ref: "B1", value: "备注", type: "string", style: { bold: true } }],
        [{ ref: "A2", value: "2026-08-09", type: "string" }, { ref: "B2", value: "mock 数据", type: "string" }],
      ],
      colWidths: { A: 12, B: 20 },
    },
  ],
});

// ── 事件总线（浏览器 mock 通道；对接 bridge.onEvent 的 mockSubscribe 分支）──
export const mockListeners = new Set<(e: WireEvent) => void>();

export function mockSubscribe(cb: (e: WireEvent) => void): () => void {
  mockListeners.add(cb);
  return () => {
    mockListeners.delete(cb);
  };
}

export function emitMock(e: WireEvent) {
  mockListeners.forEach((l) => l(e));
}


// ── 任务中心 mock：内存任务表（TaskList/TaskCancel/TaskRetry + gaea-task 事件）──
export const taskMock: TaskView[] = [];
let taskSeq = 0;

export function taskView(kind: string, label: string, result: Record<string, unknown> = {}, outputTail?: string): TaskView {
  const t: TaskView = {
    id: "tsk_" + ++taskSeq, kind, label,
    status: "succeeded", progress: 100, message: "完成",
    error: "", retryCount: 0, maxRetries: 2,
    payload: "{}", result: JSON.stringify(result),
    createdAt: Date.now(), startedAt: Date.now(), finishedAt: Date.now(),
  };
  if (outputTail) t.outputTail = outputTail; // C9：事件携带输出尾回放
  taskMock.unshift(t);
  if (taskMock.length > 20) taskMock.pop();
  mockTaskListeners.forEach((l) => l(t));
  return t;
}

export const mockTaskListeners = new Set<(t: TaskView) => void>();

export function mockTaskSubscribe(cb: (t: TaskView) => void): () => void {
  mockTaskListeners.add(cb);
  return () => {
    mockTaskListeners.delete(cb);
  };
}

// ── 更新进度总线（ApplyUpdate 驱动，bridge 订阅 updater:progress 的 mock 分支）──
export const updaterListeners = new Set<(p: UpdateProgress) => void>();

export function emitUpdater(p: UpdateProgress) {
  updaterListeners.forEach((l) => l(p));
}

export function delay(ms: number): Promise<void> {
  return new Promise((r) => setTimeout(r, ms));
}

// ── 场景系统（URL 参数驱动）────────────────────────────────────────────────

// MockScenario 是浏览器开发模式的场景全集：demo/fresh/running 为既有，
// approval/ask/compaction 为评审 03-office-frontend.md 缺陷 8 补全——
// 审批卡/提问卡/压缩卡此前无 mock 场景，浏览器开发无法覆盖这三条事件流。
// calm（v4.123 走查发现）：无 running 子代理/无自动置前触发源的安静场景——
// 办公冷挂载时 demo 恒有 running 子代理会触发「子代理任务自动置前」
// （openTasksAuto→closeFilePreview）顶掉打开中的文件预览，走查预览链路用 calm。
// subagents（2026-11-18 委派卡上浮）：并行派发 task ×3 + turn_done 后子代理
// 事件回投（parentId 子调用/task_result 带 sa_ 引用）——委派卡上浮对话流的
// ocr（2026-11-18 OCR 卡重设计）：并行露出识别中、引擎拉起、缓存命中与失败
// 四种作业态，作为办公 OCR 卡的稳定视觉走查锚点。
export type MockScenario = "demo" | "fresh" | "running" | "approval" | "ask" | "compaction" | "calm" | "subagents" | "ocr";

export function mockScenario(): MockScenario {
  if (typeof window === "undefined") return "demo";
  const value = new URLSearchParams(window.location.search).get("mock")?.trim().toLowerCase();
  if (value === "fresh" || value === "empty" || value === "first-run") return "fresh";
  if (value === "running" || value === "busy" || value === "streaming") return "running";
  if (value === "approval" || value === "approve") return "approval";
  if (value === "ask" || value === "question") return "ask";
  if (value === "compaction" || value === "compress") return "compaction";
  if (value === "calm" || value === "quiet") return "calm";
  if (value === "subagents" || value === "parallel") return "subagents";
  if (value === "ocr" || value === "scan") return "ocr";
  return "demo";
}

export function browserPlatformOverride(): "darwin" | "windows" | "linux" | "" {
  if (typeof window === "undefined" || window.runtime) return "";
  const value = new URLSearchParams(window.location.search).get("platform");
  return value === "darwin" || value === "windows" || value === "linux" ? value : "";
}
