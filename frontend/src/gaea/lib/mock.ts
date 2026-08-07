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
    agent: { temperature: 0.2, maxSteps: 0, systemPrompt: "You are gaea, a coding agent.", subagentTemperature: 0, effort: "", subagentEffort: "" },
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
      const isCodeReq = !isPoetry && /(写|创建|程序|代码|函数|排序)/.test(input);
      const think = isPoetry ? "用户想写诗，直接创作即可。"
        : isCodeReq ? `用户说"${input}"，这是编程任务，先检查项目结构。`
        : `用户说"${input}"，让我看看上下文再回复。`;
      for (const ch of think) { if (cancelled) break; emit({ kind: "reasoning", reasoning: ch }); await delay(12); }
      await delay(200);
      emit({ kind: "tool_dispatch", tool: { id: "t1", name: "glob", args: '{"pattern":"**/*.go"}', readOnly: true } });
      await delay(400);
      emit({ kind: "tool_result", tool: { id: "t1", name: "glob", output: "cmd/gaea/main.go\ninternal/agent/agent.go", readOnly: true } });
      await delay(200);
      let reply: string;
      if (isPoetry) reply = "**《山居秋暝》**\n\n> 空山新雨后，天气晚来秋。\n> 明月松间照，清泉石上流。";
      else if (isCodeReq) reply = `好的，"${input}"：项目是 Go，入口在 cmd/gaea/main.go。要具体实现什么？`;
      else reply = `收到！**${input}**\n\nGo 项目 gaea，核心在 internal/。需要做什么？`;
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
      if (ext === "md") {
        return {
          path: rel, name: rel.split("/").pop() ?? rel, ext: ".md",
          size: samples[rel]?.length ?? 0, kind: "markdown" as const,
          body: samples[rel] ?? "# Mock\n\n预览内容来自浏览器 mock。", dataUrl: "", error: "",
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
          },
        ],
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
      return { knowledgeCount: 4, profileCount: 0, officeCount: 0, whisperCount: 0, latestUpdated: "" };
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
      return [];
    },
    async CostSearch() {
      return [];
    },
    async CostGet() {
      return null;
    },
    async CostSave() {
      // mock: no-op
    },
    async CostDelete() {
      // mock: no-op
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
