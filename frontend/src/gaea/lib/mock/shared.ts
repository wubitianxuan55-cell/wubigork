// mock/shared.ts — mock 域的共享状态与工具（T6-10.1 拆分自 lib/mock.ts）。
// 承载跨域共享的模块级内存态与事件总线；可变绑定通过 setter 暴露
// （ESM 禁止跨模块重绑定 let），方法体语义与原实现完全一致。

import type {
  CostCategory,
  CostSummary,
  PriceFetchRecord,
  PriceSource,
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

// 成本库 mock：与记忆中枢 CostLibrary 同库的单价条目，供办公侧「成本库」
// Tab 浏览/引用与产物「沉淀到成本库」流程联调。
export const costMock: CostSummary[] = [
  {
    name: "hp300", title: "HP300 高频液压振动锤", category: "机械", categoryPath: "机械/桩基机械", unit: "台班",
    price: 3200, spec: "300kW", source: "市场询价", tags: ["振动锤", "桩基"], status: "现行", updatedAt: "",
  },
  {
    name: "cement", title: "P.O 42.5 水泥", category: "水泥", categoryPath: "材料/土建材料/水泥及水泥制品", unit: "吨",
    price: 480, spec: "", source: "定额", tags: [], status: "现行", updatedAt: "",
  },
];

// 多级分类树 mock（与后端默认树一致，供浏览器联调）。
export let costCategoriesMock: CostCategory[] = [
  { id: 1, parentId: 0, name: "人工", sort: 0, count: 0, children: [
    { id: 11, parentId: 1, name: "普工", sort: 0, count: 0 },
    { id: 12, parentId: 1, name: "技工", sort: 0, count: 0 },
  ] },
  { id: 2, parentId: 0, name: "材料", sort: 0, count: 0, children: [
    { id: 21, parentId: 2, name: "土建材料", sort: 0, count: 0, children: [
      { id: 211, parentId: 21, name: "水泥及水泥制品", sort: 0, count: 1 },
      { id: 212, parentId: 21, name: "砂石", sort: 0, count: 0 },
      { id: 213, parentId: 21, name: "钢材", sort: 0, count: 0 },
    ] },
    { id: 22, parentId: 2, name: "安装材料", sort: 0, count: 0, children: [
      { id: 221, parentId: 22, name: "电线电缆", sort: 0, count: 0 },
      { id: 222, parentId: 22, name: "管材管件", sort: 0, count: 0 },
    ] },
    { id: 23, parentId: 2, name: "周转材料", sort: 0, count: 0 },
  ] },
  { id: 3, parentId: 0, name: "机械", sort: 0, count: 0, children: [
    { id: 31, parentId: 3, name: "桩基机械", sort: 0, count: 1 },
  ] },
  { id: 4, parentId: 0, name: "运输", sort: 0, count: 0 },
  { id: 5, parentId: 0, name: "检测", sort: 0, count: 0 },
  { id: 6, parentId: 0, name: "综合单价", sort: 0, count: 0 },
  { id: 7, parentId: 0, name: "其他", sort: 0, count: 0 },
];
export function setCostCategoriesMock(v: CostCategory[]) {
  costCategoriesMock = v;
}

// 价格源 mock：内置重庆/四川两个造价信息源，抓取返回样例候选（更新/新增）。
export const initialPriceSourcesMock: PriceSource[] = [
  {
    id: "src-cq", name: "重庆施工造价信息网", parser: "sc_table", frequencyHours: 24, area: "重庆",
    url: "http://www.cqsgczjxx.org/Pages/CQZJW/priceInformation.aspx",
    enabled: true, lastFetchAt: "", createdAt: "2026-08-10T00:00:00Z",
  },
  {
    id: "src-sc", name: "四川造价信息网（期 758）", parser: "sc_table", frequencyHours: 24, area: "成都市区",
    url: "http://202.61.90.35:8032/pubpages/pricelist.aspx?period=758",
    enabled: true, lastFetchAt: "", createdAt: "2026-08-10T00:00:00Z",
  },
];
export let priceSourcesMock: PriceSource[] = initialPriceSourcesMock.map((s) => ({ ...s }));
export function setPriceSourcesMock(v: PriceSource[]) {
  priceSourcesMock = v;
}

export const initialPriceFetchMock: PriceFetchRecord[] = [
  {
    id: "fetch-1", sourceId: "src-sc", sourceName: "四川造价信息网（期 758）",
    url: "http://202.61.90.35:8032/pubpages/pricelist.aspx?period=758",
    period: "758", fetchedAt: new Date().toISOString(), status: "pending",
    candidates: [
      {
        title: "热轧光圆钢筋", spec: "HPB300 Φ12", unit: "t", price: 3750, tax: "不含税",
        existingName: "rebar", existingPrice: 3000, status: "更新", diff: 750, diffPct: 25,
        anomaly: true, anomalyReason: "单期跳幅 +25.0%（基准 ¥3,000）",
      },
      {
        title: "螺纹钢", spec: "HRB400 Φ20", unit: "t", price: 3420, tax: "不含税",
        existingName: "", existingPrice: 0, status: "新增", diff: 0, diffPct: 0,
        anomaly: false, anomalyReason: "",
      },
    ],
  },
];
export let priceFetchMock: PriceFetchRecord[] = initialPriceFetchMock.map((f) => ({ ...f, fetchedAt: new Date().toISOString() }));
export function setPriceFetchMock(v: PriceFetchRecord[]) {
  priceFetchMock = v;
}

// 测试辅助：重置价格源/抓取记录 mock 状态（避免用例间串扰）。
export function __resetPriceMocksForTest() {
  priceFetchMock = initialPriceFetchMock.map((f) => ({ ...f, fetchedAt: new Date().toISOString() }));
  priceSourcesMock = initialPriceSourcesMock.map((s) => ({ ...s }));
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
         { ref: "B1", value: "金额", type: "string", style: { bold: true, fill: "4472C4", fontColor: "FFFFFF", align: "center", border: true } }],
        [{ ref: "A2", value: "设备", type: "string" }, { ref: "B2", value: "120.50", type: "number", style: { numFmt: "0.00%" } }],
        [{ ref: "A3", value: "人工", type: "string" }, { ref: "B3", value: "80", type: "number", style: { numFmt: "0.00%" } }],
        [{ ref: "A4", value: "合计", type: "string", style: { bold: true } }, { ref: "B4", value: "200.50", formula: "SUM(B2:B3)", type: "string", style: { bold: true } }],
        [{ ref: "A5", value: "合并单元格（mock）", type: "string" }],
      ],
      merged: ["A5:B5"],
      colWidths: { A: 16, B: 14 },
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

// 内部别名 — 各域方法内用 emit() 调用
export const emit = emitMock;

// ── 任务中心 mock：内存任务表（TaskList/TaskCancel/TaskRetry + gaea-task 事件）──
export const taskMock: TaskView[] = [];
let taskSeq = 0;

export function taskView(kind: string, label: string, result: Record<string, unknown> = {}): TaskView {
  const t: TaskView = {
    id: "tsk_" + ++taskSeq, kind, label,
    status: "succeeded", progress: 100, message: "完成",
    error: "", retryCount: 0, maxRetries: 2,
    payload: "{}", result: JSON.stringify(result),
    createdAt: Date.now(), startedAt: Date.now(), finishedAt: Date.now(),
  };
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
export type MockScenario = "demo" | "fresh" | "running" | "approval" | "ask" | "compaction";

export function mockScenario(): MockScenario {
  if (typeof window === "undefined") return "demo";
  const value = new URLSearchParams(window.location.search).get("mock")?.trim().toLowerCase();
  if (value === "fresh" || value === "empty" || value === "first-run") return "fresh";
  if (value === "running" || value === "busy" || value === "streaming") return "running";
  if (value === "approval" || value === "approve") return "approval";
  if (value === "ask" || value === "question") return "ask";
  if (value === "compaction" || value === "compress") return "compaction";
  return "demo";
}

export function browserPlatformOverride(): "darwin" | "windows" | "linux" | "" {
  if (typeof window === "undefined" || window.runtime) return "";
  const value = new URLSearchParams(window.location.search).get("platform");
  return value === "darwin" || value === "windows" || value === "linux" ? value : "";
}
