// mock/state.ts — makeMockApp 每次调用创建的一次性本地状态（T6-10.1 拆分）。
// 原 makeMockApp 内部的 let/const 状态全部迁入 createMockState；域方法通过
// MakeMockState 读写。仅 capServers/keepWarm/preloadPlan 会被整体重绑
// （经 setter），其余均为引用内原地变更（splice/push/set/属性赋值），
// 解构后仍实时可见，行为与原闭包一致。

import type {
  ProjectGroup,
  Requirement,
  ServerView,
  SessionMeta,
  SettingsView,
  SkillView,
} from "../types";
import { mockScenario } from "./shared";

export interface MakeMockState {
  scenario: "demo" | "fresh" | "running";
  freshMock: boolean;
  runningMock: boolean;
  cwd: string;
  workspaces: string[];
  capServers: ServerView[];
  capSkills: SkillView[];
  sessions: SessionMeta[];
  archivedMock: SessionMeta[];
  requirementsMock: Map<string, Requirement>;
  projectGroupsMock: ProjectGroup[];
  settings: SettingsView;
  keepWarm: boolean;
  preloadPlan: boolean;
  setCapServers(v: ServerView[]): void;
  setKeepWarm(v: boolean): void;
  setPreloadPlan(v: boolean): void;
}

// mockSwitchWorkspace：切换工作区并置顶（原 makeMockApp 内部函数）。
export function switchWorkspace(s: MakeMockState, path: string): string {
  s.cwd = path || "~";
  s.workspaces = [s.cwd, ...s.workspaces.filter((p) => p !== s.cwd)].slice(0, 12);
  return s.cwd;
}

export function createMockState(): MakeMockState {
  const scenario = mockScenario();
  const freshMock = scenario === "fresh";
  const runningMock = scenario === "running";
  const cwd = "~/projects/gaea"; // mutable so PickWorkspace is visible in dev
  const workspaces = freshMock ? [] : ["~/projects/gaea", "~/projects/blade", "~/projects/deepseek-forge", "~/projects/cc-switch-light", "~/projects/SuperRig"];
  const day = 86_400_000;
  const t0 = Date.now();
  // Mutable so MCP add/remove/retry are observable in browser dev.
  const capServers: ServerView[] = [
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
  // Mutable so delete/rename are observable in browser dev.
  // interrupted=true 表示上次运行中断未完成（T5-4）；d.jsonl 刻意标记中断，供浏览器
  // 联调「未完成」徽标与恢复摘要注入。
  const sessions: SessionMeta[] = freshMock ? [] : [
    { path: "/mock/sessions/a.jsonl", preview: "compile quarterly report", turns: 12, modTime: t0 - 3_600_000, current: true, hasRequirement: true, interrupted: false },
    { path: "/mock/sessions/b.jsonl", preview: "convert docx to markdown", turns: 5, modTime: t0 - 6 * 3_600_000, current: false, pinned: true, interrupted: false },
    { path: "/mock/sessions/c.jsonl", preview: "build chart from data", turns: 8, modTime: t0 - day - 3_600_000, current: false, hasRequirement: true, requirementDone: true, interrupted: false },
    { path: "/mock/sessions/d.jsonl", preview: "explain the plugin host design", turns: 3, modTime: t0 - 4 * day, current: false, interrupted: true },
  ];
  // 已归档会话（可恢复；浏览器 mock 内存态）
  const archivedMock: SessionMeta[] = freshMock ? [] : [
    { path: "/mock/sessions/arch1.jsonl", preview: "上季度费用报销整理", turns: 7, modTime: t0 - 20 * day, current: false, archived: true, hasRequirement: true, requirementDone: true, interrupted: false },
  ];
  // 会话任务目标（从需求到验收；浏览器 mock 内存态）
  const requirementsMock = new Map<string, Requirement>();
  if (!freshMock) {
    requirementsMock.set("/mock/sessions/a.jsonl", {
      text: "整理季度经营数据，输出一份带图表的总结报告（docx）",
      done: false,
      updatedAt: t0 - 3_600_000,
    });
  }
  // 侧边栏「项目」分组 mock：当前工作区 + 两个历史项目。
  const projectGroupsMock: ProjectGroup[] = freshMock ? [] : [
    {
      path: cwd,
      name: cwd.split("/").filter(Boolean).pop() ?? cwd,
      current: true,
      sessions,
      archived: archivedMock,
      modTime: t0 - 3_600_000,
    },
    {
      path: "~/projects/annual-report",
      name: "annual-report",
      current: false,
      sessions: [
        { path: "/mock/sessions/annual/r1.jsonl", preview: "整理年度经营数据", turns: 9, modTime: t0 - 2 * day, current: false },
        { path: "/mock/sessions/annual/r2.jsonl", preview: "起草董事会报告框架", turns: 6, modTime: t0 - 9 * day, current: false },
      ],
      archived: [],
      modTime: t0 - 2 * day,
    },
    {
      path: "~/projects/market-research",
      name: "market-research",
      current: false,
      sessions: [
        { path: "/mock/sessions/mkt/m1.jsonl", preview: "竞品价格对比表", turns: 4, modTime: t0 - 12 * day, current: false },
      ],
      archived: [],
      modTime: t0 - 12 * day,
    },
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
  // 本地模型调度 mock（T5-3）：保活/自动预载开关（内存态，浏览器联调可切换）。
  const s: MakeMockState = {
    scenario,
    freshMock,
    runningMock,
    cwd,
    workspaces,
    capServers,
    capSkills,
    sessions,
    archivedMock,
    requirementsMock,
    projectGroupsMock,
    settings,
    keepWarm: true,
    preloadPlan: true,
    setCapServers(v) {
      s.capServers = v;
    },
    setKeepWarm(v) {
      s.keepWarm = v;
    },
    setPreloadPlan(v) {
      s.preloadPlan = v;
    },
  };
  return s;
}
