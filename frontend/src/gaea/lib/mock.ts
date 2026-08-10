// bridge/mock.ts — 浏览器开发模式的 mock 实现。
// 仅在 Wails 环境不可用时加载（pnpm dev 模式），
// 模拟 gaea 后端的响应，让整个 UI 可独立开发调试。
//
// 场景系统：通过 URL 参数切换 mock 行为，无需修改代码。
//   ?mock=fresh     空状态：无工作区、无会话、无 API key
//   ?mock=running   模拟活跃流式输出（工具执行中 / 思考中）
//   ?mock=demo      默认：完整 mock 数据（等同于不传参数）
//   ?platform=darwin|windows|linux 覆盖平台检测
//
// 缓存安全: 纯前端 mock，不触及 Go 内核。

import type {
  CostSummary,
  PriceFetchRecord,
  PriceSource,
  KnowledgeEntry,
  KnowledgeSaveRequest,
  KnowledgeSummary,
  MCPServerInput,
  MemorySuggestion,
  Meta,
  ProviderView,
  ServerView,
  SessionMeta,
  SettingsView,
  SkillSuggestion,
  SkillView,
  UpdateProgress,
  WireEvent,
} from "./types";
import type { AppBindings } from "./bridge";

const EVENT_CHANNEL = "agent:event";

// 浏览器开发 mock 的固定资料清单（内存态，对应后端 .gaea/pinned.json）。
let pinnedMock: string[] = [];

// 成本库 mock：与记忆中枢 CostLibrary 同库的单价条目，供办公侧「成本库」
// Tab 浏览/引用与产物「沉淀到成本库」流程联调。
let costMock: CostSummary[] = [
  {
    name: "hp300", title: "HP300 高频液压振动锤", category: "机械", unit: "台班",
    price: 3200, spec: "300kW", source: "市场询价", tags: ["振动锤", "桩基"], status: "现行", updatedAt: "",
  },
  {
    name: "cement", title: "P.O 42.5 水泥", category: "材料", unit: "吨",
    price: 480, spec: "", source: "定额", tags: [], status: "现行", updatedAt: "",
  },
];

// 价格源 mock：内置重庆/四川两个造价信息源，抓取返回样例候选（更新/新增）。
const initialPriceSourcesMock: PriceSource[] = [
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
let priceSourcesMock: PriceSource[] = initialPriceSourcesMock.map((s) => ({ ...s }));
const initialPriceFetchMock: PriceFetchRecord[] = [
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
let priceFetchMock: PriceFetchRecord[] = initialPriceFetchMock.map((f) => ({ ...f, fetchedAt: new Date().toISOString() }));

// 测试辅助：重置价格源/抓取记录 mock 状态（避免用例间串扰）。
export function __resetPriceMocksForTest() {
  priceFetchMock = initialPriceFetchMock.map((f) => ({ ...f, fetchedAt: new Date().toISOString() }));
  priceSourcesMock = initialPriceSourcesMock.map((s) => ({ ...s }));
}

// 浏览器开发 mock 用的最小 docx（含标题/正文/表格），由 docx-preview 渲染。
const MOCK_DOCX_DATA_URL =
  "data:application/vnd.openxmlformats-officedocument.wordprocessingml.document;base64,UEsDBBQAAAAIAEmjCV3XeYTq8QAAALgBAAATAAAAW0NvbnRlbnRfVHlwZXNdLnhtbH2QzU7DMBCE730Ky9cqccoBIZSkB36OwKE8wMreJFb9J69b2rdn00KREOVozXwz62nXB+/EHjPZGDq5qhspMOhobBg7+b55ru6koALBgIsBO3lEkut+0W6OCUkwHKiTUynpXinSE3qgOiYMrAwxeyj8zKNKoLcworppmlulYygYSlXmDNkvhGgfcYCdK+LpwMr5loyOpHg4e+e6TkJKzmoorKt9ML+Kqq+SmsmThyabaMkGqa6VzOL1jh/0lSfK1qB4g1xewLNRfcRslIl65xmu/0/649o4DFbjhZ/TUo4aiXh77+qL4sGG71+06jR8/wlQSwMEFAAAAAgASaMJXSAbhuqyAAAALgEAAAsAAABfcmVscy8ucmVsc43Puw6CMBQG4J2naM4uBQdjDIXFmLAafICmPZRGeklbL7y9HRzEODie23fyN93TzOSOIWpnGdRlBQStcFJbxeAynDZ7IDFxK/nsLDJYMELXFs0ZZ57yTZy0jyQjNjKYUvIHSqOY0PBYOo82T0YXDE+5DIp6Lq5cId1W1Y6GTwPagpAVS3rJIPSyBjIsHv/h3ThqgUcnbgZt+vHlayPLPChMDB4uSCrf7TKzQHNKuorZvgBQSwMEFAAAAAgASaMJXV5N4XGjAQAAcwMAABEAAAB3b3JkL2RvY3VtZW50LnhtbJVTS0vEMBC++ytCTnrQrA9Elm1FD549KJ7bNO5W26QkcaueFhEUPBRRfKEs6kFvPkBYRPDHuM3qvzB9iQ921ctkpvPNfDNf0srkqu+BOuHCZdSAw0MlCAjFzHFp1YDzczODExAIaVHH8hglBlwjAk6afZWw7DC84hMqge5ARTk0YE3KoIyQwDXiW2KIBYTq3CLjviV1yKsoZNwJOMNECE3ge2ikVBpHvuVSaPYBoLvazFlL3DQIzMTM8uRYwiAs1y3PgFiTEg6RWUEf2cxkvo0SK9YL/Oh4js3z0lTN/ThqtR9P2s9nndPm28Xm69U26PcZXh5IgDKDZwRfp8k7vD4fq6Mb9RDpwvj4On5qxNGuuj6Pn6J2a6ezfwe0PKuDASd1l4RAte5Vc69zsqlOG/FtBBa0DkAdbKnzy5fGRndKaXuZX0T5BrY3rTvoW0sjFhSrJqp6BIJkfQOOwVQJm0nJ/J4QjyzKngDuVmu9EOjrUKgY9tPw2eTY/KZkvH0Ipn4qkH7B3Uumu5eglOx35uH/sY78iTFfPX3NqHjOiVf8LuY7UEsBAhQAFAAAAAgASaMJXdd5hOrxAAAAuAEAABMAAAAAAAAAAAAAAIABAAAAAFtDb250ZW50X1R5cGVzXS54bWxQSwECFAAUAAAACABJowldIBuG6rIAAAAuAQAACwAAAAAAAAAAAAAAgAEiAQAAX3JlbHMvLnJlbHNQSwECFAAUAAAACABJowldXk3hcaMBAABzAwAAEQAAAAAAAAAAAAAAgAH9AQAAd29yZC9kb2N1bWVudC54bWxQSwUGAAAAAAMAAwC5AAAAzwMAAAAA";

// 浏览器开发 mock 用的 xlsx 结构化预览（含公式/样式/合并/多 sheet）。
const MOCK_XLSX_BODY = JSON.stringify({
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

// 内部别名 — makeMockApp 内部用 emit() 调用
const emit = emitMock;

export const updaterListeners = new Set<(p: UpdateProgress) => void>();

function emitUpdater(p: UpdateProgress) {
  updaterListeners.forEach((l) => l(p));
}

function delay(ms: number): Promise<void> {
  return new Promise((r) => setTimeout(r, ms));
}

// ── 场景系统（URL 参数驱动）────────────────────────────────────────────────

export function mockScenario(): "demo" | "fresh" | "running" {
  if (typeof window === "undefined") return "demo";
  const value = new URLSearchParams(window.location.search).get("mock")?.trim().toLowerCase();
  if (value === "fresh" || value === "empty" || value === "first-run") return "fresh";
  if (value === "running" || value === "busy" || value === "streaming") return "running";
  return "demo";
}

export function browserPlatformOverride(): "darwin" | "windows" | "linux" | "" {
  if (typeof window === "undefined" || window.runtime) return "";
  const value = new URLSearchParams(window.location.search).get("platform");
  return value === "darwin" || value === "windows" || value === "linux" ? value : "";
}

export function makeMockApp(): AppBindings {
  const scenario = mockScenario();
  const freshMock = scenario === "fresh";
  const runningMock = scenario === "running";
  let cancelled = false;
  let cwd = "~/projects/gaea"; // mutable so PickWorkspace is visible in dev
  let workspaces = freshMock ? [] : ["~/projects/gaea", "~/projects/blade", "~/projects/deepseek-forge", "~/projects/cc-switch-light", "~/projects/SuperRig"];
  const day = 86_400_000;
  const t0 = Date.now();
  // Mutable so MCP add/remove/retry are observable in browser dev.
  let capServers: ServerView[] = [
    {
      name: "documents",
      transport: "stdio",
      status: "connected",
      tools: 4,
      prompts: 0,
      resources: 1,
      toolList: [
        { name: "read", description: "Read a file from the allowed directory." },
        { name: "write", description: "Write content to a file." },
        { name: "list", description: "List files and directories." },
        { name: "search", description: "Search for files by name pattern." },
      ],
    },
    { name: "github", transport: "stdio", status: "connected", tools: 12, prompts: 2, resources: 0 },
    { name: "linear", transport: "http", status: "connected", tools: 8, prompts: 0, resources: 0 },
    { name: "figma", transport: "http", status: "failed", tools: 0, prompts: 0, resources: 0, error: "connect: 401 unauthorized" },
  ];
  const capSkills: SkillView[] = [
    { name: "research", description: "Research a question with web search and analysis", scope: "builtin", runAs: "subagent" },
    { name: "format-convert", description: "Convert docx/xlsx/pdf to editable Markdown", scope: "builtin", runAs: "subagent" },
    { name: "chart-builder", description: "Generate statistics charts from data", scope: "builtin", runAs: "subagent" },
    { name: "doc-assemble", description: "Assemble multiple documents into a complete report", scope: "builtin", runAs: "subagent" },
  ];
  const mockSwitchWorkspace = async (path: string) => {
    cwd = path || "~";
    workspaces = [cwd, ...workspaces.filter((p) => p !== cwd)].slice(0, 12);
    return cwd;
  };
  // Mutable so delete/rename are observable in browser dev.
  const sessions: SessionMeta[] = freshMock ? [] : [
    { path: "/mock/sessions/a.jsonl", preview: "compile quarterly report", turns: 12, modTime: t0 - 3_600_000, current: true },
    { path: "/mock/sessions/b.jsonl", preview: "convert docx to markdown", turns: 5, modTime: t0 - 6 * 3_600_000, current: false },
    { path: "/mock/sessions/c.jsonl", preview: "build chart from data", turns: 8, modTime: t0 - day - 3_600_000, current: false },
    { path: "/mock/sessions/d.jsonl", preview: "explain the plugin host design", turns: 3, modTime: t0 - 4 * day, current: false },
  ];
  // Mutable settings so the Settings panel's edits are observable in browser dev.
  const settings: SettingsView = {
    defaultModel: "deepseek-flash",
    providers: [
      { name: "deepseek-flash", kind: "openai", baseUrl: "https://api.deepseek.com", models: ["deepseek-v4-flash"], default: "deepseek-v4-flash", apiKeyEnv: "DEEPSEEK_API_KEY", keySet: !freshMock, balanceUrl: "https://api.deepseek.com/user/balance", contextWindow: 1_000_000, oauthKind: "", oauthReady: false },
      { name: "mimo-pro", kind: "openai", baseUrl: "https://api.xiaomimimo.com/v1", models: ["mimo-v2.5-pro"], default: "mimo-v2.5-pro", apiKeyEnv: "MIMO_API_KEY", keySet: false, balanceUrl: "", contextWindow: 1_000_000, oauthKind: "", oauthReady: false },
      { name: "xai-oauth", kind: "xai", baseUrl: "https://api.x.ai/v1", models: ["grok-4.3"], default: "grok-4.3", apiKeyEnv: "", keySet: false, balanceUrl: "", contextWindow: 1_000_000, oauthKind: "xai", oauthReady: false },
    ],
    permissions: { mode: "ask", allow: ["ls", "read_file"], ask: [], deny: ["bash(rm *)"] },
    sandbox: { bash: "enforce", network: true, workspaceRoot: "", allowWrite: [] },
    agent: { temperature: 0.2, maxSteps: 0, systemPrompt: "你是 gaea（盖亚）——用户的通用办公 AI 助手，负责文档撰写与编辑、表格与数据处理、方案与报告编写等办公工作。沉稳、清晰、可靠，温和而不说教。所有思考和输出必须使用中文。", subagentTemperature: 0, effort: "", subagentEffort: "" },
    subagentModel: "",
    subagentModels: {},
    subagentSkills: ["explore", "research", "review", "security-review"],
    configPath: freshMock ? "~/.gaea/config.toml" : "~/projects/gaea/gaea.toml",
    providerKinds: ["openai", "xai"],
    bypass: false,
    permLevel: "ask",
  };
  return {
    async Submit(input) {
      cancelled = false;
      emit({ kind: "turn_started" });
      await delay(300);
      if (cancelled) return;
      if (runningMock) await delay(1500); // simulate existing reasoning in progress
      const isPoetry = /(诗|古诗|词)/.test(input);
      const think = isPoetry ? "用户想写诗，直接创作即可。"
        : `用户说"${input}"，先查看工作区里的资料再回复。`;
      for (const ch of think) { if (cancelled) break; emit({ kind: "reasoning", reasoning: ch }); await delay(12); }
      await delay(200);
      emit({ kind: "tool_dispatch", tool: { id: "t1", name: "ls", args: '{"path":"."}', readOnly: true } });
      await delay(400);
      emit({ kind: "tool_result", tool: { id: "t1", name: "ls", output: "方案.md\n成本测算.md\n表格.xlsx", readOnly: true } });
      await delay(200);
      let reply: string;
      if (isPoetry) reply = "**《山居秋暝》**\n\n> 空山新雨后，天气晚来秋。\n> 明月松间照，清泉石上流。";
      else reply = `收到！**${input}**\n\n我先查看当前办公目录中的资料（方案、成本测算、表格等），整理好后给你。`;
      for (const ch of reply) { if (cancelled) break; emit({ kind: "text", text: ch }); await delay(10); }
      emit({ kind: "message", text: reply });
      emit({ kind: "usage", usage: { promptTokens: 1200, completionTokens: 200, totalTokens: 1400, cacheHitTokens: 800, cacheMissTokens: 400, sessionCacheHitTokens: 800, sessionCacheMissTokens: 400 } });
      emit({
        kind: "usage",
        usage: {
          promptTokens: 1280,
          completionTokens: 64,
          totalTokens: 1344,
          cacheHitTokens: 1024,
          cacheMissTokens: 256,
          sessionCacheHitTokens: 1024,
          sessionCacheMissTokens: 256,
        },
      });
      emit({ kind: "turn_done" });
    },
    async SubmitDisplay(_display, input) {
      await this.Submit(input);
    },
    async Cancel() {
      cancelled = true;
      emit({ kind: "turn_done" });
    },
    async Approve() {},
    async AnswerQuestion() {},
    async SetAgentMode(_mode: string) {},
    async AgentMode() { return "develop"; },
    async Compact() {},
    async NewSession() {},
    async Reload() {
      // mock: 无真实内核，返回空结果
      return { tools: 0, skills: 0 };
    },
    async CaptureSkill(_input) {
      // mock: 浏览器开发环境不写磁盘，返回假结果
      return { name: _input.name || "mock-skill", description: _input.description, path: "", reloaded: false, tools: 0, skills: 0 };
    },
    async Checkpoints() {
      return [];
    },
    async Rewind() {},
    async Fork() {},
    async SummarizeFrom() {},
    async SummarizeUpTo() {},
    async History() {
      return [];
    },
    async ListSessions() {
      return sessions.map((s) => ({ ...s }));
    },
    async ResumeSession(path: string) {
      return [
        { role: "user", content: `(mock) resumed ${path}` },
        { role: "assistant", content: "This is a mock resumed transcript — the real one comes from the kernel." },
      ];
    },
    async DeleteSession(path: string) {
      const i = sessions.findIndex((s) => s.path === path);
      if (i >= 0) sessions.splice(i, 1);
    },
    async RenameSession(path: string, title: string) {
      const s = sessions.find((x) => x.path === path);
      if (s) s.title = title.trim() || undefined;
    },
    async ListWorkspaces() {
      return workspaces.map((path) => ({
        path,
        name: path.split("/").filter(Boolean).pop() ?? path,
        current: path === cwd,
      }));
    },
    async PickWorkspace() {
      // Browser dev has no native dialog; simulate picking a folder and re-root so
      // the topbar folder chip visibly changes.
      return mockSwitchWorkspace(cwd.endsWith("another-project") ? "~/projects/gaea" : "~/projects/another-project");
    },
    async SwitchWorkspace(path: string) {
      return mockSwitchWorkspace(path);
    },
    async ContextUsage() {
      return { used: 1280, window: 1_000_000 };
    },
    async TCCAReport() {
      return JSON.stringify({
        l1Size: 12400,
        l2Size: 1200,
        l3Version: 2,
        l4Messages: 18,
        savedByCompact: 82000,
        savedByFork: 100300,
        forkCount: 23,
        savedUsd: 0.24,
        savedLatencyMs: 4500,
        compactionCount: 3,
      });
    },
    async Balance() {
      // Mirror the active mock provider: deepseek-flash carries a balance_url.
      const p = settings.providers.find((x) => x.name === settings.defaultModel);
      if (!p?.balanceUrl) return { available: false, display: "" };
      return { available: true, display: "¥128.50" };
    },
    async Jobs() {
      return []; // browser dev mock has no background jobs
    },
    async FactBase() {
      return { facts: [], markdown: "", count: 0, path: "" };
    },
    async FactBaseClear() {
      // browser dev mock: nothing to clear
    },
    async FactBasePromote() {
      return 0; // browser dev mock: no persistent memory
    },
    async Meta(): Promise<Meta> {
      return {
        label: "mock model · browser dev",
        ready: true,
        eventChannel: EVENT_CHANNEL,
        cwd,
        bypass: settings.permLevel !== "ask",
        agentMode: "develop",
      };
    },
    async Commands() {
      return [
        { name: "new", description: "Start a new session", kind: "builtin" as const },
        { name: "compact", description: "Summarize older history to free up context", kind: "builtin" as const },
        { name: "model", description: "Switch model", kind: "builtin" as const },
        { name: "skill", description: "List skills", kind: "builtin" as const },
        { name: "explore", description: "Investigate the codebase in an isolated subagent", kind: "skill" as const },
        { name: "review", description: "Review the staged diff", hint: "[focus]", kind: "custom" as const },
      ];
    },
    async Capabilities() {
      return { servers: capServers.map((s) => ({ ...s })), skills: capSkills.map((s) => ({ ...s })) };
    },
    async AddMCPServer(input: MCPServerInput) {
      const tools = input.transport === "stdio" ? 3 : 5;
      capServers.push({
        name: input.name,
        transport: input.transport,
        status: "connected",
        tools,
        prompts: 0,
        resources: 0,
        toolList: Array.from({ length: tools }, (_, i) => ({
          name: `${input.name}_tool_${i + 1}`,
          description: `Mock tool ${i + 1} exposed by ${input.name}.`,
        })),
      });
      return tools;
    },
    async RemoveMCPServer(name: string) {
      capServers = capServers.filter((s) => s.name !== name);
    },
    async RetryMCPServer(name: string) {
      capServers = capServers.map((s) =>
        s.name === name ? { ...s, status: "connected", tools: s.tools || 4, error: undefined } : s,
      );
    },
    async SetMCPServerEnabled(name: string, enabled: boolean) {
      capServers = capServers.map((s) =>
        s.name === name
          ? { ...s, status: enabled ? "connected" : "disabled", tools: enabled ? s.tools || 4 : 0, error: undefined }
          : s,
      );
    },
    async SlashArgs(input: string) {
      // Mirror a slice of the real arg hints so the menu is exercisable in browser dev.
      const from = input.lastIndexOf(" ") + 1;
      const cur = input.slice(from);
      const cmd = input.slice(0, input.indexOf(" ") < 0 ? input.length : input.indexOf(" "));
      const subs: Record<string, { label: string; insert: string; hint: string; descend?: boolean }[]> = {
        "/skill": [
          { label: "list", insert: "list", hint: "list skills" },
          { label: "show", insert: "show ", hint: "show a skill's body", descend: true },
          { label: "new", insert: "new ", hint: "scaffold a new skill" },
          { label: "paths", insert: "paths", hint: "show discovery paths" },
        ],
        "/hooks": [
          { label: "list", insert: "list", hint: "list active hooks" },
          { label: "trust", insert: "trust", hint: "trust this project's hooks" },
        ],
        "/model": [
          { label: "deepseek/deepseek-v4-flash", insert: "deepseek/deepseek-v4-flash", hint: "current" },
          { label: "deepseek/deepseek-v4-pro", insert: "deepseek/deepseek-v4-pro", hint: "" },
        ],
      };
      const items = (subs[cmd] ?? [])
        .filter((it) => it.label.toLowerCase().startsWith(cur.toLowerCase()))
        .map((it) => ({ label: it.label, insert: it.insert, hint: it.hint, descend: it.descend ?? false }));
      return { items, from };
    },
    async ListDir(rel: string) {
      // A tiny fake tree so the @ menu is navigable in browser dev.
      if (rel === "" || rel === "./") {
        return [
          { name: "internal", isDir: true },
          { name: "desktop", isDir: true },
          { name: "README.md", isDir: false },
          { name: "go.mod", isDir: false },
        ];
      }
      if (rel === "internal/") {
        return [
          { name: "control", isDir: true },
          { name: "boot", isDir: true },
          { name: "event.go", isDir: false },
        ];
      }
      return [{ name: "file.go", isDir: false }];
    },
    async FileSearch(query: string, limit = 30) {
      const tree = [
        { path: "README.md", name: "README.md", isDir: false, size: 18 },
        { path: "desktop/file.go", name: "file.go", isDir: false, size: 42 },
        { path: "docs/成本测算.xlsx", name: "成本测算.xlsx", isDir: false, size: 120 },
        { path: "docs/方案.docx", name: "方案.docx", isDir: false, size: 80 },
        { path: "internal/control", name: "control", isDir: true },
      ];
      const q = query.toLowerCase();
      return tree.filter((f) => f.name.toLowerCase().includes(q)).slice(0, limit);
    },
    async Materials(limit = 100) {
      const now = Date.now();
      return [
        { path: "docs/成本测算.xlsx", name: "成本测算.xlsx", isDir: false, size: 120, modTime: now },
        { path: "docs/方案.docx", name: "方案.docx", isDir: false, size: 80, modTime: now - 1000 },
        { path: "docs/说明.md", name: "说明.md", isDir: false, size: 40, modTime: now - 2000 },
      ].slice(0, limit);
    },
    async WorkspaceSearch(query: string, limit = 20) {
      const q = query.toLowerCase();
      const corpus = [
        { path: "docs/成本测算.xlsx", name: "成本测算.xlsx", size: 120, body: "成本测算总金额 100 万元，材料费 60 万、人工费 40 万。" },
        { path: "docs/方案.docx", name: "方案.docx", size: 80, body: "市政道路改造方案：背景、目标、实施计划与预算。" },
        { path: "docs/说明.md", name: "说明.md", size: 40, body: "这是固定的项目说明，包含本周进展与下周计划。" },
        { path: "README.md", name: "README.md", size: 18, body: "gaea 办公助手使用说明。" },
      ];
      const now = Date.now();
      return corpus
        .filter((f) => f.name.toLowerCase().includes(q) || f.body.toLowerCase().includes(q))
        .slice(0, limit)
        .map((f, i) => ({
          path: f.path,
          name: f.name,
          size: f.size,
          modTime: now - i * 1000,
          score: 0.9 - i * 0.1,
          snippet: f.body.length > 40 ? `…${f.body.slice(0, 40)}…` : f.body,
        }));
    },
    async PinnedMaterials() {
      return pinnedMock.map((path) => ({
        path,
        name: path.split("/").pop() ?? path,
        isDir: false,
        size: 40,
        modTime: Date.now(),
      }));
    },
    async PinMaterial(rel: string) {
      if (!pinnedMock.includes(rel)) pinnedMock.push(rel);
      return this.PinnedMaterials();
    },
    async UnpinMaterial(rel: string) {
      pinnedMock = pinnedMock.filter((p) => p !== rel);
      return this.PinnedMaterials();
    },
    async SummarizeFile(rel: string, focus?: string) {
      const name = rel.split("/").pop() ?? rel;
      return {
        path: rel,
        totalPages: 0,
        chars: 120,
        chunks: 1,
        summary: `${name} 的分块摘要（mock）：主题、要点与关键数据概览${focus ? `，侧重「${focus}」` : ""}。`,
      };
    },
    async TaskTemplates() {
      return [
        { name: "weekly-report", title: "周报", description: "结构化周报：进展 / 数据 / 问题 / 下周计划", prompt: "帮我生成一份本周工作周报：按「本周进展 / 关键数据 / 遇到的问题 / 下周计划」四部分撰写，输出 Markdown 并保存到 .gaea/exports/。" },
        { name: "meeting-minutes", title: "会议纪要", description: "纪要模板：议题 / 结论 / 行动项", prompt: "帮我整理一份会议纪要：按「议题与讨论 / 结论 / 行动项」组织，行动项包含负责人和截止时间。" },
        { name: "cost-estimate", title: "成本测算", description: "生成 xlsx 成本测算表：公式 + 图表", prompt: "帮我制作一份成本测算表（.xlsx）：\n1. 先与我对齐测算范围和科目（人工/材料/机械/管理费/税费等）；\n2. 测算前先用 cost_search 查询成本库中的历史单价作为定价依据：命中的科目直接引用并在正文注明依据的条目名称，缺失科目与用户确认或给出合理估价并说明假设；\n3. 用 xlsx 能力创建表格：科目、单位、数量、单价、金额，金额用公式计算（数量×单价），并提供汇总行；\n4. 为费用构成生成原生图表（柱状/饼图）；\n5. 测算完成后用 cost_save 把本次采用的单价沉淀为成本条目（来源标注本次项目/文件，同名覆盖），并在正文汇报新增/更新条数；\n6. 保存到 .gaea/exports/ 并在正文给出可点击的 [文件名](路径)。" },
        { name: "proposal-outline", title: "方案大纲", description: "背景 / 目标 / 方案对比 / 实施 / 预算 / 风险", prompt: "帮我撰写一份方案大纲：按「背景与目标 / 现状分析 / 方案设计 / 实施计划 / 预算 / 风险」组织。" },
        { name: "data-analysis", title: "数据分析", description: "清洗 → 透视 → 图表 → 结论", prompt: "帮我做一份数据分析：清洗数据 → 分类汇总 → 生成图表 → 输出结论。" },
        { name: "document-convert", title: "文档转换", description: "docx / xlsx / pdf 与 Markdown 互转", prompt: "帮我转换这份文档：用 format_convert 转为 Markdown 并保留标题层级与表格。" },
        { name: "report-assemble", title: "报告拼装", description: "多素材合并为完整报告", prompt: "帮我拼装一份完整报告：封面 / 目录 / 正文 / 附录，保留来源标注。" },
        { name: "ppt-deck", title: "演示文稿", description: "大纲 → PPT 成稿（.pptx）", prompt: "帮我生成一份演示文稿（.pptx）：先列 8-12 页大纲再成稿。" },
      ];
    },
    async ReadFile(rel: string) {
      const samples: Record<string, string> = {
        "README.md": "# gaea\n\nBrowser-dev workspace preview.\n\n- Chat in the center\n- Browse files on the right\n- Keep sessions on the left\n",
        "go.mod": "module gaea\n\ngo 1.23\n",
        "desktop/file.go": "package desktop\n\nfunc main() {\n\tprintln(\"workspace preview\")\n}\n",
        "internal/event.go": "package internal\n\n// mock file used by the browser dev seam\n",
      };
      return {
        path: rel,
        body: samples[rel] ?? `// ${rel}\n\nMock file body from browser dev.`,
        size: samples[rel]?.length ?? 42,
        truncated: false,
        binary: false,
      };
    },
    async Preview(rel: string) {
      const samples: Record<string, string> = {
        "README.md": "# gaea\n\nBrowser-dev workspace preview.\n\n- Chat in the center\n- Browse files on the right\n- Keep sessions on the left\n",
        "go.mod": "module gaea\n\ngo 1.23\n",
      };
      const ext = rel.split(".").pop()?.toLowerCase() ?? "";
      if (["png", "jpg", "jpeg", "gif", "webp", "svg"].includes(ext)) {
        return {
          path: rel, name: rel.split("/").pop() ?? rel, ext: `.${ext}`,
          size: 1024, kind: "image" as const,
          body: "", dataUrl: "data:image/png;base64,iVBORw0KGgo=", error: "",
        };
      }
      if (ext === "docx") {
        // 最小 docx（mock），由 docx-preview 渲染成版式预览。
        return {
          path: rel, name: rel.split("/").pop() ?? rel, ext: ".docx",
          size: 1728, kind: "docx" as const,
          body: "", dataUrl: MOCK_DOCX_DATA_URL, error: "",
        };
      }
      if (ext === "xlsx") {
        // 结构化单元格预览（mock），由 XlsxPreview 渲染。
        return {
          path: rel, name: rel.split("/").pop() ?? rel, ext: ".xlsx",
          size: 2048, kind: "xlsx" as const,
          body: MOCK_XLSX_BODY, dataUrl: "", error: "",
        };
      }
      if (ext === "md") {
        return {
          path: rel, name: rel.split("/").pop() ?? rel, ext: ".md",
          size: samples[rel]?.length ?? 0, kind: "markdown" as const,
          body: samples[rel] ?? "# Mock\n\n预览内容来自浏览器 mock。", dataUrl: "", error: "",
        };
      }
      if (ext === "pdf") {
        if (rel === "scan.pdf") {
          // 扫描件 PDF：模拟 OCR 逐页进度事件，随后返回识别结果。
          emit({ kind: "preview_progress", path: rel, progress: { path: rel, done: 1, total: 3 } });
          await delay(80);
          emit({ kind: "preview_progress", path: rel, progress: { path: rel, done: 2, total: 3 } });
          await delay(80);
          emit({ kind: "preview_progress", path: rel, progress: { path: rel, done: 3, total: 3 } });
          return {
            path: rel, name: rel.split("/").pop() ?? rel, ext: ".pdf",
            size: 2048, kind: "markdown" as const,
            body: "（以下内容由 OCR 识别）\n\n扫描页内容。", dataUrl: "", error: "",
          };
        }
        // 大 PDF 预览截断样例：truncated/totalPages 由后端 GaeaPreview 填充。
        const truncated = rel === "big.pdf";
        return {
          path: rel, name: rel.split("/").pop() ?? rel, ext: ".pdf",
          size: truncated ? 2_400_000 : 1024, kind: "markdown" as const,
          body: truncated ? "第 1 页内容\n\n> ⚠️ 预览已截断：PDF 共 1200 页，仅显示前 500 页。" : "# PDF mock",
          dataUrl: "", error: "",
          truncated: truncated || undefined,
          totalPages: truncated ? 1200 : undefined,
        };
      }
      return {
        path: rel, name: rel.split("/").pop() ?? rel, ext: `.${ext}`,
        size: samples[rel]?.length ?? 0, kind: "text" as const,
        body: samples[rel] ?? `// ${rel}\n\nMock file body from browser dev.`, dataUrl: "", error: "",
      };
    },
    async OpenWorkspacePath(rel: string) {
      console.info("mock OpenWorkspacePath", rel);
    },
    async OfficeEditText(selectedText: string, instruction: string) {
      return { edited: `${selectedText}（mock 编辑：${instruction}）` };
    },
    async DocxApplyEdit(rel: string) {
      return {
        path: rel, name: rel.split("/").pop() ?? rel, ext: ".docx",
        size: 1728, kind: "docx" as const,
        body: "", dataUrl: MOCK_DOCX_DATA_URL, error: "",
      };
    },
    async DocxAcceptChanges(rel: string, accept: boolean) {
      return {
        path: rel, name: rel.split("/").pop() ?? rel, ext: ".docx",
        size: 1728, kind: "docx" as const,
        body: "", dataUrl: MOCK_DOCX_DATA_URL, error: "",
      };
    },
    async XlsxEdit(_rel: string, sheet: string, instruction: string, selection: string) {
      return {
        preview: MOCK_XLSX_BODY,
        summary: `（mock）已在 ${sheet} 应用操作：${instruction}（选区 ${selection}）`,
        applied: 1,
      };
    },
    async XlsxSetCell(_rel: string, sheet: string, ref: string, value: string) {
      return {
        preview: MOCK_XLSX_BODY,
        summary: `（mock）已更新 ${sheet}!${ref} = ${value}`,
        applied: 1,
      };
    },
    async XlsxRecalc(_rel: string) {
      return {
        preview: MOCK_XLSX_BODY,
        summary: "（mock）已重算公式",
        applied: 1,
      };
    },
    async XlsxRowOps(_rel: string, sheet: string, action: string, ref: string) {
      return {
        preview: MOCK_XLSX_BODY,
        summary: `（mock）已在 ${sheet} 执行行操作 ${action}@${ref}`,
        applied: 1,
      };
    },
    async XlsxColOps(_rel: string, sheet: string, action: string, ref: string) {
      return {
        preview: MOCK_XLSX_BODY,
        summary: `（mock）已在 ${sheet} 执行列操作 ${action}@${ref}`,
        applied: 1,
      };
    },
    async ExportDeliverable(input: { markdown: string; format: string; title?: string }) {
      const format = input.format.replace(".", "");
      return {
        path: `.gaea/exports/${input.title || "deliverable"}-mock.${format}`,
        name: `${input.title || "deliverable"}-mock.${format}`,
        format,
        size: input.markdown.length,
      };
    },
    async CrossEmbed(input: { xlsxRel: string; into: string; title?: string }) {
      const name = `${input.title || "chart"}-mock.${input.into}`;
      return {
        path: `.gaea/exports/${name}`,
        name,
        size: 4096,
        chartPath: `.gaea/exports/${input.title || "chart"}-chart-mock.png`,
      };
    },
    async WorkspaceChanges() { return []; },
    async RevealWorkspacePath(rel: string) {
      console.info("mock RevealWorkspacePath", rel);
    },
    async SavePastedImage(_dataUrl: string) {
      return ".gaea/attachments/mock.png";
    },
    async SaveAttachmentFile(_fileName: string, _base64Data: string) {
      return ".gaea/attachments/mock-file.bin";
    },
    async AttachmentDataURL(_path: string) {
      return "data:image/png;base64,iVBORw0KGgo=";
    },
    async CaptureScreen() {
      // 1x1 红色 PNG，占位截图
      return "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==";
    },
    async RecognizeImage(_imagePath: string, _prompt: string) {
      return "（开发预览）这是一张模拟识图结果：截图内容为一份通用办公任务清单。";
    },
    async OCRText(_imagePath: string) {
      return "（开发预览）模拟文字提取：项目周报 / 营收 120 万元 / 同比增长 18%。";
    },
    async Models() {
      return [
        { ref: "deepseek/deepseek-v4-flash", provider: "deepseek", model: "deepseek-v4-flash", current: true },
        { ref: "deepseek/deepseek-v4-pro", provider: "deepseek", model: "deepseek-v4-pro", current: false },
      ];
    },
    async SetModel() {},
    async Memory() {
      return {
        available: true,
        storeDir: "~/.config/gaea/projects/-mock/memory",
        docs: [
          {
            path: "REASONIX.md",
            scope: "project",
            body: "# gaea project memory\n\nMock doc shown in the browser dev seam.\n\n## Notes\n\n- prefers concise replies",
          },
          {
            path: "~/.config/gaea/REASONIX.md",
            scope: "user",
            body: "# User memory\n\nAlways respond in 中文.",
          },
        ],
        facts: [
          {
            name: "prefers-tabs",
            description: "User prefers tabs",
            type: "user",
            body: "Indent with tabs.",
            lastUsedAt: new Date(Date.now() - 3 * 86400000).toISOString(),
            sourceSession: "session-mock-demo.jsonl",
            sourceMessage: "turn 2",
          },
        ],
        enabled: true,
        scopes: [
          { scope: "user", path: "~/.config/gaea/REASONIX.md" },
          { scope: "project", path: "REASONIX.md" },
          { scope: "local", path: "REASONIX.local.md" },
        ],
      };
    },
    async Remember(scope: string, note: string) {
      emit({ kind: "notice", level: "info", text: `remembered → ${scope}` });
      return `${scope} REASONIX.md (mock): ${note}`;
    },
    async Forget(name: string) {
      emit({ kind: "notice", level: "info", text: `forgot → ${name}` });
    },
    async SaveDoc(path: string, _body: string) {
      emit({ kind: "notice", level: "info", text: `saved → ${path}` });
      return path;
    },
    async UpdateFact(name: string, _body: string) {
      emit({ kind: "notice", level: "info", text: `updated → ${name}` });
      return name;
    },
    async ChangeFactType(name: string, typ: string) {
      emit({ kind: "notice", level: "info", text: `type changed → ${name} (${typ})` });
      return name;
    },
    async SetMemoryEnabled(enabled: boolean) {
      emit({ kind: "notice", level: "info", text: `memory ${enabled ? "enabled" : "disabled"}` });
    },
    async MemorySuggestions() {
      return { memories: [], skills: [], generatedAt: new Date().toISOString(), available: false, source: "mock" };
    },
    async AcceptMemorySuggestion(_candidate: MemorySuggestion) {
      return "mock-memory-path";
    },
    async AcceptSkillSuggestion(_candidate: SkillSuggestion) {
      return "mock-skill-path";
    },
    async SelectTab(_tabID: string) {},
    async TabMeta() {
      return [{ id: "mock-tab", scope: "project", workspaceRoot: "", title: "Mock", ready: true }] as any;
    },
    async Settings() {
      return JSON.parse(JSON.stringify(settings)) as SettingsView;
    },
    async SetDefaultModel(ref: string) {
      settings.defaultModel = ref;
    },
    async SaveProvider(p: ProviderView) {
      const i = settings.providers.findIndex((x) => x.name === p.name);
      if (i >= 0) settings.providers[i] = p;
      else settings.providers.push(p);
    },
    async DeleteProvider(name: string) {
      settings.providers = settings.providers.filter((p) => p.name !== name);
    },
    async SetProviderKey(apiKeyEnv: string) {
      settings.providers.forEach((p) => {
        if (p.apiKeyEnv === apiKeyEnv) p.keySet = true;
      });
    },
    async LoginProvider(name: string) {
      const p = settings.providers.find((x) => x.name === name);
      if (p) p.oauthReady = true;
    },
    async LogoutProvider(name: string) {
      const p = settings.providers.find((x) => x.name === name);
      if (p) p.oauthReady = false;
    },
    async SetPermissionMode(mode: string) {
      settings.permissions.mode = mode;
    },
    async AddPermissionRule(list: string, rule: string) {
      const k = list as "allow" | "ask" | "deny";
      if (settings.permissions[k] && !settings.permissions[k].includes(rule)) settings.permissions[k].push(rule);
    },
    async RemovePermissionRule(list: string, rule: string) {
      const k = list as "allow" | "ask" | "deny";
      settings.permissions[k] = settings.permissions[k].filter((r) => r !== rule);
    },
    async SetSandbox(bash: string, network: boolean, workspaceRoot: string, allowWrite: string[]) {
      settings.sandbox = { bash, network, workspaceRoot, allowWrite };
    },
    async SetAgentParams(temperature: number, maxSteps: number, systemPrompt: string) {
      settings.agent = { ...settings.agent, temperature, maxSteps, systemPrompt };
    },
    async SetSubagentModel(ref: string) {
      settings.subagentModel = ref;
    },
    async SetSubagentModelForSkill(_skill: string, ref: string) {
      if (!settings.subagentModels) settings.subagentModels = {};
      settings.subagentModels[_skill] = ref;
    },
    async SetSubagentTemperature(temp: number) {
      settings.agent.subagentTemperature = temp;
    },
    async SetEffort(effort: string) {
      settings.agent.effort = effort;
    },
    async SetSubagentEffort(effort: string) {
      settings.agent.subagentEffort = effort;
    },
    async SetPermLevel(level: string) {
      settings.permLevel = level;
    },
    async Version() {
      return "v1.0.0 (browser dev)";
    },
    async CheckUpdate() {
      // Dev mock advertises an update so the banner and apply flow are exercisable
      // in the browser without a real release behind it.
      return {
        available: true,
        current: "v1.0.0",
        latest: "v1.1.0",
        notes: "- Mock release notes\n- The **Update now** button streams a fake download here.",
        canSelfUpdate: true,
        downloadUrl: "https://github.com/wubitianxuan55-cell/gaea/releases/latest",
        assetSize: 12_345_678,
      };
    },
    async ApplyUpdate() {
      const total = 12_345_678;
      for (let r = 0; r <= total; r += 1_800_000) {
        emitUpdater({ phase: "downloading", received: Math.min(r, total), total });
        await delay(120);
      }
      emitUpdater({ phase: "verifying", received: total, total });
      await delay(500);
      emitUpdater({ phase: "applying", received: total, total });
      await delay(500);
      emitUpdater({ phase: "done", received: total, total });
      // The real shell relaunches here; the mock just stops.
    },
    async OpenDownloadPage() {
      if (typeof window !== "undefined") {
        window.open("https://github.com/wubitianxuan55-cell/gaea/releases/latest", "_blank", "noopener");
      }
    },
    async SaveWindowState(_state: {width:number;height:number;x:number;y:number;maximised:boolean}) {
      // no-op in browser dev
    },
    async KnowledgeList(): Promise<KnowledgeSummary[]> {
      return [
        { name: "gb50300-2024", title: "建筑工程施工质量验收统一标准 GB 50300-2024", category: "规范标准", tags: ["施工", "质量", "验收"], status: "现行", updatedAt: "2025-01-15T00:00:00.000Z" },
        { name: "case-bio-remediation", title: "某焦化厂生物修复工程案例", category: "工程案例", tags: ["焦化厂", "生物修复", "PAHs"], status: "已归档", updatedAt: "2024-11-20T00:00:00.000Z" },
        { name: "soil-sampling-guide", title: "污染场地土壤采样技术要点", category: "经验总结", tags: ["采样", "布点", "质量控制"], status: "常用", updatedAt: "2025-02-10T00:00:00.000Z" },
        { name: "hdp-liner-spec", title: "HDPE 土工膜施工技术规范", category: "材料工艺", tags: ["HDPE", "土工膜", "防渗"], status: "现行", updatedAt: "2024-09-05T00:00:00.000Z" },
      ];
    },
    async KnowledgeSearch(query: string, category: string, phase: string, status: string): Promise<KnowledgeSummary[]> {
      let list = await this.KnowledgeList();
      if (category && category !== "all") list = list.filter((e) => e.category === category);
      if (status && status !== "all") list = list.filter((e) => e.status === status);
      if (query) {
        const q = query.trim().toLowerCase();
        list = list.filter((e) => [e.title, e.name, e.category, ...e.tags].join(" ").toLowerCase().includes(q));
      }
      return list;
    },
    async KnowledgeGet(name: string): Promise<KnowledgeEntry | null> {
      const entries: Record<string, KnowledgeEntry> = {
        "gb50300-2024": {
          name: "gb50300-2024", title: "建筑工程施工质量验收统一标准 GB 50300-2024", category: "规范标准", tags: ["施工", "质量", "验收"], status: "现行", updatedAt: "2025-01-15T00:00:00.000Z",
          body: "## 适用范围\n\n本标准适用于建筑工程施工质量的验收，包括地基与基础、主体结构、建筑装饰装修、建筑屋面、建筑给排水及供暖、通风与空调、建筑电气、智能建筑、建筑节能、电梯等分部工程。\n\n## 基本规定\n\n1. 施工现场质量管理应有相应的技术标准。\n2. 建筑工程施工质量应按下列要求进行验收。\n3. 建筑工程施工质量验收应划分为单位工程、分部工程、分项工程和检验批。",
          phase: "施工验收", discipline: "土木工程", source: "住房和城乡建设部", version: 2, author: "住建部标准定额司", reviewer: "", createdAt: "2024-06-01T00:00:00.000Z",
        },
        "case-bio-remediation": {
          name: "case-bio-remediation", title: "某焦化厂生物修复工程案例", category: "工程案例", tags: ["焦化厂", "生物修复", "PAHs"], status: "已归档", updatedAt: "2024-11-20T00:00:00.000Z",
          body: "## 项目概况\n\n某焦化厂退役地块，占地面积约 120 亩。主要污染物为多环芳烃（PAHs）、苯系物（BTEX）和氰化物。\n\n## 修复方案\n\n采用原位生物通风+化学氧化联合修复工艺。\n- 生物通风：注入空气和营养盐，促进土著微生物降解\n- 化学氧化：注射过硫酸钠氧化高浓度区域\n\n## 修复效果\n\n经过 18 个月的修复运行，目标污染物去除率达到 85% 以上，达到修复目标值。",
          phase: "施工", discipline: "环境工程", source: "内部案例库", version: 1, author: "张三", reviewer: "", createdAt: "2024-06-15T00:00:00.000Z",
        },
        "soil-sampling-guide": {
          name: "soil-sampling-guide", title: "污染场地土壤采样技术要点", category: "经验总结", tags: ["采样", "布点", "质量控制"], status: "常用", updatedAt: "2025-02-10T00:00:00.000Z",
          body: "## 采样前准备\n\n1. 收集场地历史资料，了解潜在污染物类型\n2. 制定采样方案，明确布点方法和数量\n3. 准备采样设备、样品容器和现场记录表\n\n## 布点方法\n\n- 系统布点法：适用于污染物分布均匀的场地\n- 分层布点法：适用于污染来源明确的场地\n- 判断布点法：适用于历史污染区域\n\n## 质量控制\n\n- 现场平行样：每 10 个样品至少 1 个\n- 运输空白样：每批次至少 1 个\n- 设备清洗样：每个采样点之间采集",
          phase: "调查", discipline: "环境工程", source: "项目经验总结", version: 3, author: "李四", reviewer: "", createdAt: "2024-08-01T00:00:00.000Z",
        },
        "hdp-liner-spec": {
          name: "hdp-liner-spec", title: "HDPE 土工膜施工技术规范", category: "材料工艺", tags: ["HDPE", "土工膜", "防渗"], status: "现行", updatedAt: "2024-09-05T00:00:00.000Z",
          body: "## 材料要求\n\nHDPE 土工膜厚度不应小于 1.5mm，密度不低于 0.94g/cm³。\n\n## 施工要点\n\n1. 基底应平整压实，无尖锐物\n2. 膜与膜之间采用热熔焊接\n3. 焊缝强度不低于母材强度\n4. 铺设时应预留 5%-8% 的伸缩余量\n\n## 质量检验\n\n- 目测检查：膜面有无破损、褶皱\n- 气密性试验：焊缝处进行气压测试\n- 厚度检测：每 500m² 测一点",
          phase: "施工", discipline: "岩土工程", source: "施工技术手册", version: 2, author: "王五", reviewer: "", createdAt: "2024-07-01T00:00:00.000Z",
        },
      };
      return entries[name] || null;
    },
    // ── 知识库 CRUD Mock ────────────────────────────────────
    async KnowledgeSave(_entry: KnowledgeSaveRequest) {
      // mock: no-op
    },
    async KnowledgeDelete(_name: string) {
      // mock: no-op
    },
    // ── 记忆中枢 Mock ────────────────────────────────────────
    async MemoryHubOverview() {
      return { knowledgeCount: 4, profileCount: 0, officeCount: 0, costCount: 2, whisperCount: 0, pinnedCount: pinnedMock.length, latestUpdated: "" };
    },
    async ProfileList() {
      return [];
    },
    async ProfileSave() {
      // mock: no-op
    },
    async ProfileDelete() {
      // mock: no-op
    },
    async ProfileConflicts() {
      return [];
    },
    async ProfileResolveConflict() {
      // mock: no-op
    },
    async WhisperMemories() {
      return [];
    },
    async WhisperEpisodes() {
      return [];
    },
    async MemoryGraph() {
      return { nodes: [], links: [] };
    },
    async CostList() {
      return costMock;
    },
    async CostSearch(query: string, category: string, status: string) {
      const q = (query ?? "").toLowerCase();
      return costMock.filter((e) => {
        if (category && category !== "all" && e.category !== category) return false;
        if (status && status !== "all" && e.status !== status) return false;
        if (!q) return true;
        return [e.name, e.title, e.spec, e.source].some((s) => (s ?? "").toLowerCase().includes(q));
      });
    },
    async CostGet(name: string) {
      const e = costMock.find((c) => c.name === name);
      return e ? { ...e, body: "", createdAt: "" } : null;
    },
    async CostSave() {
      // mock: no-op
    },
    async CostDelete() {
      // mock: no-op
    },
    // ── 成本库导入（mock：对已知文件返回样例候选）──
    async CostImportPreview(path: string) {
      return {
        path,
        fileName: path.split(/[\\/]/).pop() ?? path,
        columns: ["材料名称", "规格型号", "单位", "单价(元)", "供应商"],
        unmapped: ["备注"],
        rows: [
          {
            name: "hp300", title: "HP300 高频液压振动锤", category: "机械", unit: "台班",
            price: 3200, spec: "300kW", source: "XX租赁", status: "现行",
            existingName: "hp300", existingPrice: 3000, matchNote: "将覆盖更新（现价 ¥3,000）",
            raw: "HP300 高频液压振动锤 | 300kW | 台班 | 3200 | XX租赁", skip: false, skipReason: "",
          },
          {
            name: "", title: "P.O 42.5 水泥", category: "材料", unit: "吨",
            price: 480, spec: "", source: "海螺", status: "现行",
            existingName: "", existingPrice: 0, matchNote: "新增",
            raw: "P.O 42.5 水泥 | | 吨 | 480 | 海螺", skip: false, skipReason: "",
          },
        ],
        message: "",
        aiUsed: false,
      };
    },
    async CostImportAIParse(path: string) {
      const pv = await this.CostImportPreview(path);
      pv.aiUsed = true;
      pv.message = "AI 智能解析完成，请核对后确认导入。";
      return pv;
    },
    async CostImportApply() {
      return 0;
    },
    // ── 价格源（mock）──
    async PriceSources() {
      return priceSourcesMock;
    },
    async PriceSourceSave(src: PriceSource) {
      const i = priceSourcesMock.findIndex((s) => s.id === src.id);
      if (i >= 0) priceSourcesMock[i] = src;
      else priceSourcesMock.push(src);
    },
    async PriceSourceDelete(id: string) {
      priceSourcesMock = priceSourcesMock.filter((s) => s.id !== id);
    },
    async PriceFetch(id: string) {
      const src = priceSourcesMock.find((s) => s.id === id);
      const rec: PriceFetchRecord = {
        id: "fetch-" + Date.now(), sourceId: id, sourceName: src?.name ?? id,
        url: src?.url ?? "", period: "758", fetchedAt: new Date().toISOString(), status: "pending",
        candidates: priceFetchMock[0]?.candidates ?? [],
      };
      priceFetchMock = [rec, ...priceFetchMock.filter((f) => f.id !== rec.id)];
      if (src) src.lastFetchAt = rec.fetchedAt;
      return rec;
    },
    async PriceFetches() {
      return priceFetchMock;
    },
    async PriceFetchApply(fetchId: string, titles: string[]) {
      const rec = priceFetchMock.find((f) => f.id === fetchId);
      if (rec) rec.status = "applied";
      return titles.length;
    },
    async PriceFetchIgnore(fetchId: string) {
      const rec = priceFetchMock.find((f) => f.id === fetchId);
      if (rec) rec.status = "ignored";
    },
    async PriceHistory(name: string) {
      return [
        {
          name, title: "热轧光圆钢筋", unit: "t", price: 3181, source: "四川造价信息网",
          period: "758", fetchedAt: new Date().toISOString(), note: "价格源更新",
        },
        {
          name, title: "热轧光圆钢筋", unit: "t", price: 3000, source: "手动录入",
          period: "", fetchedAt: "", note: "",
        },
      ];
    },
    async SemanticSearch(query: string) {
      if (!query.trim()) return [];
      return [
        {
          kind: "cost", name: "hp300", score: 0.86,
          text: "HP300 高频液压振动锤（300kW） 单位台班 单价3200元 分类机械 来源市场询价",
        },
        {
          kind: "knowledge", name: "桩基-施工要点", score: 0.71,
          text: "桩基施工要点 工程案例 振动锤选型需匹配地质条件…",
        },
      ];
    },
    // ── 知识库导入（mock）──
    async KnowledgeImportPreview(path: string) {
      return {
        path,
        fileName: path.split(/[\\/]/).pop() ?? path,
        columns: ["标题", "分类", "正文"],
        unmapped: [],
        rows: [
          {
            name: "gb36600", title: "GB 36600 风险管控", category: "规范标准", phase: "", discipline: "",
            tags: [], status: "现行", source: path.split(/[\\/]/).pop() ?? path,
            body: "建设用地土壤污染风险管控标准要点…",
            existingName: "", matchNote: "新增", similarName: "", similarNote: "",
            raw: "", skip: false, skipReason: "",
          },
          {
            name: "", title: "桩基施工要点", category: "工程案例", phase: "施工", discipline: "岩土工程",
            tags: ["振动锤", "桩基"], status: "现行", source: path.split(/[\\/]/).pop() ?? path,
            body: "振动锤选型需匹配地质条件…",
            existingName: "pile", matchNote: "将覆盖更新", similarName: "", similarNote: "",
            raw: "", skip: false, skipReason: "",
          },
        ],
        message: "",
        aiUsed: false,
      };
    },
    async KnowledgeImportAIParse(path: string) {
      const pv = await this.KnowledgeImportPreview(path);
      pv.aiUsed = true;
      pv.message = "AI 智能解析完成，请核对后确认导入。";
      return pv;
    },
    async KnowledgeImportApply() {
      return 0;
    },
    async KnowledgeHistory(name: string) {
      return [
        {
          name, title: "桩基施工要点", version: 2, category: "工程案例", phase: "施工",
          discipline: "岩土工程", tags: ["振动锤", "桩基"], status: "现行",
          author: "", reviewer: "", source: "导入文件",
          body: "旧版正文：振动锤选型需匹配地质条件…",
          changedAt: new Date().toISOString(), note: "内容更新",
        },
      ];
    },
    async KnowledgeFindSimilar(title: string) {
      if (!title.trim()) return [];
      return [{ name: "pile", title: "桩基施工要点", score: 0.87 }];
    },
    async KnowledgeExport() {
      return 4;
    },
    async KnowledgeReview() {
      // mock: no-op
    },
    async KnowledgeMerge(_target: string, _sources: string[]) {
      return _target;
    },
    async MemoryDuplicates() {
      return [
        {
          keep: "pile", keepTitle: "桩基施工要点",
          dup: "pile-v2", dupTitle: "桩基施工 要点（修订）",
          score: 0.87,
        },
      ];
    },
    async MemoryMerge(target: string) {
      return target;
    },
    async FileIndexRebuild() {
      return { total: 3, skipped: 0, error: "" };
    },
    async FileSemanticSearch(query: string) {
      if (!query.trim()) return [];
      return [
        {
          path: "docs/桩基施工方案.md", score: 0.82,
          snippet: "振动锤选型需匹配地质条件，HP300 高频液压振动锤…",
        },
        {
          path: "docs/成本测算.xlsx", score: 0.63,
          snippet: "机械台班单价明细：挖掘机、振动锤…",
        },
      ];
    },
    async PickFiles(): Promise<import("./types").FilePickResult[]> {
      // In dev mode there is no native dialog -- return empty.
      return [];
    },
    async WhisperExportArchive(_dir: string): Promise<number> {
      // mock: no-op
      return 0;
    },
    async PickDirectory(): Promise<string> {
      // mock: no native dialog
      return "";
    },
  };
}
