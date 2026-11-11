// mock/core.ts — 平台/工作区/能力/更新域（T6-10.1 拆分自 lib/mock.ts，方法体零改动）。
// 说明：capServers 会被整体重绑（Remove/Retry/SetEnabled），必须经 s.capServers
// 读写（经 setCapServers 重绑），不能解构快照，否则 Capabilities 读到旧数组。
import type { AppBindings } from "../bridge";
import type {
  MCPServerInput,
  ProgrammingWebLogTail,
  ProgrammingWebPreflight,
  ProgrammingWebStatus,
  SpaceProfileView,
} from "../types";
import { delay, emitUpdater } from "./shared";
import { switchWorkspace } from "./state";
import type { MakeMockState } from "./state";
import type { ContextInsightsView } from "../types/context";

// mock 空间 profile（浏览器演示态）：模块级状态支撑 GaeaSpaceProfileSet 编辑
// 闭环；真机的 ResolveModel 校验在 Go 侧，演示态非空即收。
const mockSpaceProfilesInit: SpaceProfileView[] = [
  { space: "work", gaea: "", gaeaResolved: "", gaeaOk: false, models: {}, permMode: "ask", permHardAskCount: 0, permHardAskBySpace: false, guardrailsOn: false, modeOn: true },
  { space: "play", gaea: "glm/glm-5.3", gaeaResolved: "glm · glm-5.3", gaeaOk: true, models: { novel: "glm/glm-5.3-air" }, permMode: "allow", permHardAskCount: 0, permHardAskBySpace: true, guardrailsOn: true, modeOn: true },
];
let mockSpaceProfiles: SpaceProfileView[] = mockSpaceProfilesInit.map((p) => ({ ...p, models: { ...p.models } }));

type CoreMethods = Pick<
  AppBindings,
  | "ListWorkspaces" | "PickWorkspace" | "SwitchWorkspace"
  | "GaeaSpaceList" | "GaeaSpaceActive" | "GaeaSpaceActivate" | "GaeaSpaceProfiles" | "GaeaSpaceProfileSet"
  | "ContextUsage" | "ContextView" | "ContextNodeDetail" | "Trajectory" | "AgentNetwork" | "Jobs" | "ContextInsights"
  | "GaeaBrowserObserve" | "GaeaSubagentContextView"
  | "Meta" | "Commands" | "Capabilities"
  | "AddMCPServer" | "RemoveMCPServer" | "RetryMCPServer" | "SetMCPServerEnabled"
  | "SlashArgs"
  | "Version" | "CheckUpdate" | "ApplyUpdate" | "OpenDownloadPage" | "SaveWindowState"
  | "LogFrontendError"
  | "GetProgrammingWebPreflight" | "GetProgrammingWebStatus"
  | "StartProgrammingWeb" | "StopProgrammingWeb" | "ProgrammingWebLogTail"
  // v4.171 批次一 legacy 直调转正（CoreB 门面）：应用信息/板块清单/功能级模型读。
  | "GetAppInfo" | "GetBoardManifests" | "GetFeatureModel" | "GetFeatureModelEnabled"
  // 批次二 legacy 直调转正（CoreB 门面）：配置读写/统计/技能/全量导出。
  | "GetConfig" | "SaveConfig" | "GetStats" | "ListSkills" | "ExportAll"
  // 批次三a legacy 直调转正（CoreB 门面）：书架工程卡片列表。
  | "ListProjects"
>;

export function buildCore(s: MakeMockState): CoreMethods {
  return {
    async ListWorkspaces() {
      return s.workspaces.map((path) => ({
        path,
        name: path.split("/").filter(Boolean).pop() ?? path,
        current: path === s.cwd,
      }));
    },
    async PickWorkspace() {
      // Browser dev has no native dialog; simulate picking a folder and re-root so
      // the topbar folder chip visibly changes.
      return switchWorkspace(s, s.cwd.endsWith("another-project") ? "~/projects/gaea" : "~/projects/another-project");
    },
    async SwitchWorkspace(path: string) {
      return switchWorkspace(s, path);
    },
    async ContextUsage() {
      return { used: 1280, window: 1_000_000 };
    },
    async ContextView(_sessionPath?: string) {
      // 浏览器开发 mock：固定一份与效果图形态一致的快照。
      return {
        ok: true,
        window: 1_000_000,
        current: { system: 2100, tools: 10400, user: 20, skill: 0, inject: 21900, assistant: 93400, tool: 114000 },
        stats: { turns: 2, steps: 60, injects: 6, compacts: 0, prunes: 0, toolCalls: 279, images: 0, cacheHitPercent: 99.57, costEstimate: 3.83 },
        requests: [
          {
            seq: 8, ts: 1750000000, turn: 1, step: 1,
            category: { system: 2100, tools: 10400, user: 20, skill: 0, inject: 21000, assistant: 1200, tool: 8000 },
            briefUser: "grep C:\\AI\\wubigrok\\internal\\gaea\\config",
            briefResp: "read C:\\AI\\wubigrok\\internal\\gaea\\config\\config.go",
            // 2.5d 跳转锚点走查样例：指向下方新增的 user(seq 5)/assistant(seq 6) 节点
            briefUserSeq: 5,
            briefRespSeq: 6,
            promptTokens: 350600, outputTokens: 93, cacheHitTokens: 350200, cacheMissTokens: 400,
            // v4.79 对比上一步：请求详情 delta 条走查样例（跨压缩近似 + 逐类增量）
            delta: {
              items: 3, tokens: 1500, approx: true,
              byCat: [
                { cat: "user", items: 1, tokens: 800 },
                { cat: "assistant", items: 1, tokens: 500 },
                { cat: "tool", items: 1, tokens: 200 },
              ],
            },
          },
        ],
        events: [
          { kind: "inject", seq: 2, delta: 10500, source: "指令注入 · .gaea\\AGENTS.md", turn: 1, step: 4, ts: 1750000000 },
          { kind: "compact", seq: 30, delta: -535500, source: "ratio", turn: 2, step: 20, ts: 1750000100 },
          { kind: "prune", seq: 31, delta: 0, source: "bash", turn: 2, step: 21, ts: 1750000101 },
        ],
        nodes: [
          { seq: 1, cat: "system", tokens: 2100 },
          { seq: 2, cat: "tools", tokens: 10400 },
          { seq: 3, cat: "inject", tokens: 21000, text: "Referenced context: …" },
          { seq: 4, cat: "tool", tokens: 8000, text: "package main …" },
          { seq: 5, cat: "user", tokens: 20, text: "帮我梳理 config 装载链路（mock 跳转锚点样例）" },
          { seq: 6, cat: "assistant", tokens: 1200, text: "好的，config 装载从 Load() 入口…（mock 跳转锚点样例）" },
          { seq: 8, cat: "tool", tokens: 500, text: "图中是一张季度报表截图（mock 样例：图片缩略卡走查）", tool: "vision" },
        ],
        archive: [],
        // 2.5e 成本 hover 走查样例：最近一次用量上报的单价（每 1M tokens）
        rate: { inputPer1M: 2, outputPer1M: 8, cacheHitPer1M: 0.5, currency: "CNY" },
        commands: [
          { seq: 9, resultSeq: 10, time: 1750000009, tool: "bash", cmd: "go build ./...", turn: 1, step: 2, start: 1750000009, end: 1750000013, status: "completed", exit: 0, outChars: 1240, derived: true },
          { seq: 14, resultSeq: 15, time: 1750000014, tool: "bash", cmd: "npm run dev", turn: 1, step: 3, start: 1750000014, end: 1750000015, status: "completed", background: true, outChars: 88, derived: true },
          { seq: 18, resultSeq: 19, time: 1750000018, tool: "bash", cmd: "go vet ./internal/gaea/contextview/", turn: 2, step: 4, start: 1750000018, end: 1750000021, status: "failed", err: true, exit: 1, outChars: 640, derived: true },
        ],
        commandStats: { runs: 3, failed: 1, ms: 8000, bg: 1 },
        files: [
          { seq: 5, ts: 1750000005, turn: 1, step: 1, tool: "read_file", action: "read", path: "internal/gaea/config/config.go" },
          { seq: 9, ts: 1750000009, turn: 1, step: 2, tool: "grep", action: "read", path: "internal/gaea/config", hits: 4 },
          { seq: 12, ts: 1750000012, turn: 1, step: 3, tool: "write_file", action: "write", path: "docs/调研结论.md", added: 42 },
          { seq: 20, ts: 1750000020, turn: 1, step: 4, tool: "write_file", action: "write", path: "reports/季度报告.html" },
        ],
      };
    },
    async ContextNodeDetail(seq: number, _sessionPath?: string) {
      // mock：按 seq 返回样例详情（真实实现 = 按 seq 回读当前会话日志）。
      // 与上方 ContextView 场景的节点对齐：seq 4 = tool 节点（read_file）。
      if (seq === 4) {
        return {
          seq,
          kind: "tool_result" as const,
          ts: 1750000004,
          tool: "read_file",
          args: '{"path":"internal/gaea/config/config.go"}',
          output:
            "package main\n\n// config.go — gaea 配置装载\n// （mock 样例正文：完整调用输出，Raw/渲染切换走查用）\nfunc Load() (*Config, error) {\n\treturn loadFromDisk()\n}\n",
          lines: 7,
        };
      }
      if (seq === 8) {
        // 2.5b 后半缩略卡走查样例：识图工具结果（参数带 image_path + 输出
        // 引用产物图）。tokens 按 ⌈w/28⌉×⌈h/28⌉ 官方口径（1000×1000→1296）。
        return {
          seq,
          kind: "tool_result" as const,
          ts: 1750000008,
          tool: "vision",
          args: '{"image_path":"C:/demo/报表截图.png","prompt":"描述这张图"}',
          output: "图中是一张季度报表截图（mock 样例：图片缩略卡走查）",
          lines: 1,
          imageRefs: ["C:/demo/报表截图.png", "C:/demo/缺失图.png"],
          images: [
            {
              ref: "C:/demo/报表截图.png",
              path: "C:/demo/报表截图.png",
              exists: true,
              width: 1000,
              height: 1000,
              scaledW: 1000,
              scaledH: 1000,
              stdTokens: 1296,
              highTokens: 1296,
            },
            { ref: "C:/demo/缺失图.png", path: "C:/demo/缺失图.png", exists: false },
          ],
        };
      }
      if (seq === 3) {
        return {
          seq,
          kind: "user_message" as const,
          ts: 1750000003,
          text: "Referenced context: …（mock 样例正文）",
          lines: 1,
        };
      }
      if (seq === 5) {
        // 2.5d 趋势 brief「输入」行跳转样例（user 节点）
        return {
          seq,
          kind: "user_message" as const,
          ts: 1750000005,
          text: "帮我梳理 config 装载链路（mock 样例正文：完整用户消息）",
          lines: 1,
        };
      }
      if (seq === 6) {
        // 2.5d 趋势 brief「回复」行跳转样例（assistant 节点）
        return {
          seq,
          kind: "assistant_message" as const,
          ts: 1750000006,
          text: "好的，config 装载从 Load() 入口开始…（mock 样例正文：完整助手消息）",
          lines: 1,
        };
      }
      if (seq === 12) {
        // 文件活动操作行跳转样例（与 ContextView 场景 files 数组的 write_file 对齐）
        return {
          seq,
          kind: "tool_result" as const,
          ts: 1750000012,
          tool: "write_file",
          args: '{"path":"docs/调研结论.md"}',
          output: "# 调研结论\n\n（mock 样例正文：完整写入内容，Raw/渲染切换走查用）\n",
          lines: 3,
        };
      }
      throw new Error("mock: 未找到 seq=" + seq + " 的可展开节点");
    },
    async GaeaBrowserObserve() {
      // 浏览器观察窗单帧（v4.28 A2；批 45 NOT_MOCKED 续刀三）：浏览器未运行是
      // 观察窗的正常态（Go 不返回 error，前端按空态渲染）——mock 恒空态。
      return { available: false, url: "", title: "", image: "", width: 0, height: 0, updatedAt: 0 };
    },
    async ContextInsights(rangeKey: string, _includeArchived: boolean) {
      // 浏览器开发 mock：固定两份会话样卡 + 近 7 天柱图/8 周热力样点
      // （形状与 Go BuildInsights 输出一致；洞察页走查用）。
      const today = new Date();
      const d = (offset: number): string => {
        const x = new Date(today);
        x.setDate(x.getDate() - offset);
        return `${x.getFullYear()}-${`${x.getMonth() + 1}`.padStart(2, "0")}-${`${x.getDate()}`.padStart(2, "0")}`;
      };
      const days = Array.from({ length: 56 }, (_, i) => ({
        date: d(55 - i),
        tokens: i % 7 === 5 ? 0 : Math.round(20000 + 4000 * ((i * 37) % 11)),
        requests: i % 7 === 5 ? 0 : 6 + ((i * 3) % 9),
        turns: 2 + (i % 3),
        toolCalls: i % 7 === 5 ? 0 : 8 + ((i * 5) % 13),
        cost: i % 7 === 5 ? 0 : 0.4 + ((i * 11) % 17) / 40,
      }));
      const comp = { system: 2100, tools: 10400, user: 5200, skill: 3200, inject: 21900, assistant: 93400, tool: 114000 };
      return {
        ok: true,
        range: rangeKey as ContextInsightsView["range"],
        generatedAt: Math.floor(Date.now() / 1000),
        kpi: {
          sessions: 2, tokens: 248600, requests: 128, turns: 34,
          toolCalls: 279, skills: 5, cost: 3.83, cacheHitPercent: 99.57,
          toolsMs: 812000, wallMs: 1600000,
        },
        days,
        skills: [
          { name: "gbt-9704-docx", count: 3, last: Math.floor(Date.now() / 1000) - 3600 },
          { name: "cad-takeoff", count: 2, last: Math.floor(Date.now() / 1000) - 86400 },
        ],
        tools: [
          { name: "read_file", count: 96, last: Math.floor(Date.now() / 1000) - 600 },
          { name: "bash", count: 41, last: Math.floor(Date.now() / 1000) - 1200 },
          { name: "grep", count: 58, last: Math.floor(Date.now() / 1000) - 900 },
        ],
        programs: [
          { name: "go", count: 22, last: Math.floor(Date.now() / 1000) - 1200 },
          { name: "npm.cmd", count: 9, last: Math.floor(Date.now() / 1000) - 5400 },
        ],
        sessions: [
          {
            path: "C:\demo\.gaea\sessions\work\demo-a.jsonl",
            title: "上下文界面升级换代", preview: "根据调研结果，拟定计划并开始",
            spaceId: "work", modTime: Math.floor(Date.now() / 1000) - 600,
            lastActive: Math.floor(Date.now() / 1000) - 600,
            tokens: 248600, requests: 96, turns: 21, cost: 3.1, cacheHitPercent: 99.6,
            composition: comp,
          },
          {
            path: "C:\demo\.gaea\sessions\play\demo-b.jsonl",
            title: "旧日志估算样例（无 usage）", preview: "迁移会话",
            spaceId: "play", modTime: Math.floor(Date.now() / 1000) - 86400 * 3,
            lastActive: Math.floor(Date.now() / 1000) - 86400 * 3,
            tokens: 9000, requests: 8, turns: 4,
            composition: { system: 900, tools: 0, user: 800, skill: 0, inject: 300, assistant: 7000, tool: 0 },
            estimated: true,
          },
        ],
        scanned: 2,
      };
    },
    async GaeaSubagentContextView(_sessionPath: string, _ref: string) {
      // 子代理上下文时间线（批 45 NOT_MOCKED 续刀三）：mock 无子代理持久档，
      // 恒空时间线（ok=true 空集 + 零值类目，面板空态渲染路径走查用；行形状
      // 与 ContextView 同源）。
      return {
        ok: true,
        window: 1_000_000,
        current: { system: 0, tools: 0, user: 0, skill: 0, inject: 0, assistant: 0, tool: 0 },
        stats: { turns: 0, steps: 0, injects: 0, compacts: 0, prunes: 0, toolCalls: 0, images: 0, cacheHitPercent: 0, costEstimate: 0 },
        requests: [],
        events: [],
        nodes: [],
        archive: [],
        files: [],
        commands: [],
      };
    },
    async Trajectory(_sessionPath?: string) {
      // 浏览器开发 mock：固定一份与后端折叠形态一致的轨迹快照。
      // sessionPath（Go 变参数组）在 mock 演示数据中不区分会话，签名对齐真绑定。
      return {
        ok: true,
        turns: [
          {
            turn: 1,
            startedAt: 1750000000,
            end: { seq: 30, ts: 1750000100 },
            records: [
              { seq: 2, kind: "user", ts: 1750000001, user: { text: "调研 internal/gaea/config 的配置项" } },
              {
                seq: 3, kind: "header", ts: 1750000002, step: 1,
                header: { system: "系统提示词预览…", toolCount: 50, tokens: 12500, change: "initial" },
              },
              { seq: 4, kind: "assistant", ts: 1750000003, step: 1, assistant: { reasoning: "先搜索配置相关代码", text: "配置集中在 config.go", usage: { promptTokens: 350600, completionTokens: 93, cacheHitTokens: 350200, cacheMissTokens: 400 } } },
              {
                seq: 8, kind: "tool", ts: 1750000004, step: 1, durationMs: 3200,
                tool: { id: "t1", name: "pwsh", args: "{\"command\":\"git status --short\"}", output: "M frontend/src/gaea/App.tsx …", status: "ok" },
              },
              {
                seq: 12, kind: "tool", ts: 1750000008, step: 1, durationMs: 1500,
                tool: { id: "t2", name: "bash", args: "rm -rf /", output: "", err: "denied by policy", status: "error" },
              },
              { seq: 15, kind: "ask", ts: 1750000010, step: 1, ask: { question: "如何协调并行改动？" } },
            ],
          },
          {
            turn: 2,
            startedAt: 1750000101,
            end: { seq: 40, ts: 1750000200 },
            records: [
              { seq: 32, kind: "user", ts: 1750000102, user: { text: "总结" } },
              { seq: 33, kind: "header", ts: 1750000103, step: 1, header: { system: "…", toolCount: 50, tokens: 5000, change: "system" } },
              { seq: 34, kind: "assistant", ts: 1750000104, step: 1, assistant: { text: "总结完毕", usage: { promptTokens: 8000, completionTokens: 120 } } },
            ],
          },
        ],
        betweenTurns: [
          { seq: 31, kind: "compact", ts: 1750000100, compact: { trigger: "manual", summary: "轮间压缩：旧内容已归档" } },
        ],
      };
    },
    async AgentNetwork(_sessionPath?: string) {
      // 浏览器开发 mock：主 agent + 两个子代理（一完成一运行），运行中的
      // 子代理带嵌套孙节点（AgentTree 深层展开/收起用）。节点 id 与
      // SubagentRuns 的 ref 对齐（ref 直等匹配，面板富化运行预览）。
      // sessionPath（Go 变参数组）mock 不区分会话，签名对齐真绑定。
      return {
        ok: true,
        window: 1_000_000,
        root: {
          id: "root",
          name: "主 agent",
          kind: "root",
          status: "running",
          toolCalls: 3,
          errors: 0,
          tokens: 420000,
          firstTs: 1750000000,
          lastTs: 1750000200,
          children: [
            {
              id: "sa_20260817_100000_0000000001_a1a1a1a1",
              name: "task",
              kind: "subagent",
              status: "completed",
              task: "收集 2026 年办公 Agent 竞品更新信息",
              model: "deepseek-v4-flash",
              toolCalls: 3,
              errors: 0,
              tokens: 180000,
              firstTs: 1750000100,
              lastTs: 1750000150,
            },
            {
              id: "sa_20260817_110000_0000000002_b2b2b2b2",
              name: "task",
              kind: "subagent",
              status: "running",
              task: "调研竞品表格 Agent 能力并总结可蒸馏点",
              toolCalls: 1,
              errors: 0,
              tokens: 65000,
              firstTs: 1750000160,
              children: [
                {
                  id: "sa_20260817_113000_0000000003_c3c3c3c3",
                  name: "task",
                  kind: "subagent",
                  status: "running",
                  task: "子任务：对比三家竞品表格交互",
                  toolCalls: 2,
                  errors: 0,
                  tokens: 22000,
                  firstTs: 1750000170,
                },
              ],
            },
          ],
        },
      };
    },
    async Jobs() {
      return []; // browser dev mock has no background jobs
    },
    // Meta 契约对齐 Go GaeaMeta（T6-10.4）：label=Hephaestus、eventChannel=gaea-event、
    // permLevel 同源 settings.permLevel（internal/app/gaea_ui_meta.go）。
    async Meta() {
      return {
        label: "Hephaestus",
        ready: true,
        eventChannel: "gaea-event",
        cwd: s.cwd,
        bypass: s.settings.permLevel !== "ask",
        permLevel: s.settings.permLevel,
        agentMode: "develop",
      };
    },
    async Commands() {
      return [
        { name: "new", description: "Start a new session", kind: "builtin" as const },
        { name: "compact", description: "Summarize older history to free up context", kind: "builtin" as const },
        { name: "model", description: "Switch model", kind: "builtin" as const },
        { name: "context", description: "打开上下文看板", kind: "builtin" as const },
        { name: "skill", description: "List skills", kind: "builtin" as const },
        { name: "explore", description: "Investigate the codebase in an isolated subagent", kind: "skill" as const },
        { name: "review", description: "Review the staged diff", hint: "[focus]", kind: "custom" as const },
      ];
    },
    async Capabilities() {
      return { servers: s.capServers.map((x) => ({ ...x })), skills: s.capSkills.map((x) => ({ ...x })) };
    },
    async AddMCPServer(input: MCPServerInput) {
      const tools = input.transport === "stdio" ? 3 : 5;
      s.capServers.push({
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
      s.setCapServers(s.capServers.filter((x) => x.name !== name));
    },
    async RetryMCPServer(name: string) {
      s.setCapServers(s.capServers.map((x) =>
        x.name === name ? { ...x, status: "connected", tools: x.tools || 4, error: undefined } : x,
      ));
    },
    async SetMCPServerEnabled(name: string, enabled: boolean) {
      s.setCapServers(s.capServers.map((x) =>
        x.name === name
          ? { ...x, status: enabled ? "connected" : "disabled", tools: enabled ? x.tools || 4 : 0, error: undefined }
          : x,
      ));
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
          { label: "deepseek/deepseek-v4-pro", insert: "deepseek/deepseek-v4-pro", hint: "current" },
          { label: "xai/grok-4.20", insert: "xai/grok-4.20", hint: "" },
        ],
      };
      const items = (subs[cmd] ?? [])
        .filter((it) => it.label.toLowerCase().startsWith(cur.toLowerCase()))
        .map((it) => ({ label: it.label, insert: it.insert, hint: it.hint, descend: it.descend ?? false }));
      return { items, from };
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
      // no-op：浏览器开发环境无 shell 窗口可持久化（真实实现写入窗口管理器配置）。
    },
    async LogFrontendError(message: string) {
      console.error("[frontend]", message);
    },
    async GetProgrammingWebStatus(): Promise<ProgrammingWebStatus> {
      return {
        running: false,
        owned: false,
        starting: false,
        pid: 0,
        url: "http://dsh.wails.localhost:3091/",
        root: String.raw`C:\AI\dsh`,
        home: String.raw`C:\AI\dsh\home`,
        version: "0.2.2-rc.1",
        log: String.raw`C:\AI\dsh\home\dsh-web.log`,
        uptime_s: 0,
      };
    },
    async GetProgrammingWebPreflight(): Promise<ProgrammingWebPreflight> {
      return {
        node_found: true,
        node_path: "node",
        install_ready: true,
        profile_ready: false,
        plugin_ready: false,
        rea_ready: false,
        isolated: true,
        version_ok: true,
        port_free: true,
        all_ready: false,
        root: String.raw`C:\AI\dsh`,
        home: String.raw`C:\AI\dsh\home`,
        agents_home: String.raw`C:\AI\dsh\home\agents`,
        base: String.raw`C:\AI\dsh\node_modules\@deepseek-ai`,
        version: "0.2.2-rc.1",
        expected: "0.2.2-rc.1",
      };
    },
    async StartProgrammingWeb() {
      throw new Error("浏览器开发 mock 不启动本机 DSH 进程");
    },
    async StopProgrammingWeb() {
      // no-op
    },
    async ProgrammingWebLogTail(_n: number): Promise<ProgrammingWebLogTail> {
      return { exists: false, path: "", lines: [], error: "浏览器开发 mock 无日志" };
    },
    // ── v4.171 批次一 legacy 直调转正（Go CoreB，同名前缀）────────────────
    async GetAppInfo() {
      // 契约对齐 Go GetAppInfo（internal/app/app_info.go）：name/version/tagline/
      // releases（版本串与 Go AppVersion 同源，updater 面板走查可读）。
      return {
        name: "gaea",
        version: "4.171.0",
        tagline: "多功能 AI 助手 · 本机单用户",
        releases: [],
      };
    },
    async GetBoardManifests() {
      // mock 空清单：调用方 loadBoardManifests 已 fail-closed 兜底静态
      // canonicalBoards（boards/manifests.ts），空数组即可。
      return [];
    },
    async GetFeatureModel(_feature: string) {
      // 未配置任何功能级绑定（对齐 Go 未绑定返回空 map；useFeatureModel 的
      // r?.engine || '' 兜底为空串，不编造模型）。
      return {};
    },
    async GetFeatureModelEnabled(_feature: string) {
      // 功能级启停默认 false（对齐 Go 未配置返回 false；消费方 catch 兜底 true）。
      return false;
    },
    // ── 批次二 legacy 直调转正（Go CoreB，同名前缀）────────────────
    async GetConfig() {
      // 浏览器开发无可持久化配置：空 map（消费方 ?? 兜底，不编造键值）。
      return {};
    },
    async SaveConfig(_key: string, _value: string) {
      // mock：浏览器开发环境不落盘（no-op；真实实现写入 ~/.gaea_config.json）。
    },
    async GetStats() {
      // 最小样例：会话运行统计（NovelPage 消费；字段形状按 Go map 直通）。
      return { turns: 0, messages: 0, startedAt: 0 };
    },
    async ListSkills() {
      // 技能清单样例（CreateInspector 消费；浏览器无真实技能注册 → 空数组
      // 为诚实口径，给出单条样例便于面板走查）。
      return [
        { id: "research", name: "调研", description: "联网检索与资料整理" },
        { id: "review", name: "复核", description: "结构/引用/视觉双通道复核" },
      ];
    },
    async ExportAll(onlyMainline: boolean) {
      // mock：无可导出内容 → 空映射（ExportPanel 对空结果渲染空态）。
      void onlyMainline;
      return {};
    },
    // 双空间（S4）：mock 会话内可切换（真实后端 Activate 只写配置，重启后生效）。
    async GaeaSpaceList() {
      await delay(80);
      return [
        { id: "work", title: "办公空间", desc: "交付物落 .gaea/exports（现状路径）" },
        { id: "play", title: "娱乐空间", desc: "交付物落 .gaea/play/exports（分区）" },
      ];
    },
    async GaeaSpaceActive() {
      await delay(80);
      return { space: "work", modeOn: true, exportsDir: ".gaea/exports", workDir: ".gaea/work" };
    },
    async GaeaSpaceProfiles(): Promise<SpaceProfileView[]> {
      await delay(90);
      return mockSpaceProfiles.map((p) => ({ ...p, models: { ...p.models } }));
    },
    async GaeaSpaceProfileSet(space: string, key: string, ref: string): Promise<SpaceProfileView[]> {
      await delay(150);
      if (space !== "work" && space !== "play") throw new Error(`非法空间 ${space}（仅 work|play）`);
      if (!["gaea", "chat", "whisper", "novel", "office", "characterlib", "routine"].includes(key)) {
        throw new Error(`未知 profile 键 ${key}`);
      }
      const v = ref.trim();
      const idx = mockSpaceProfiles.findIndex((p) => p.space === space);
      if (idx >= 0) {
        const next = { ...mockSpaceProfiles[idx], models: { ...mockSpaceProfiles[idx].models } };
        if (key === "gaea") {
          next.gaea = v;
          next.gaeaOk = v !== "";
          next.gaeaResolved = v ? `demo · ${v.split("/").pop() ?? v}` : "";
        } else if (v) {
          next.models[key] = v;
        } else {
          delete next.models[key];
        }
        mockSpaceProfiles = mockSpaceProfiles.map((p, i) => (i === idx ? next : p));
      }
      return mockSpaceProfiles.map((p) => ({ ...p, models: { ...p.models } }));
    },
    async GaeaSpaceActivate(space: string) {
      await delay(120);
      if (space !== "work" && space !== "play") {
        throw new Error(`非法空间 ${space}（仅 work|play）`);
      }
      return space === "play"
        ? { space, modeOn: true, exportsDir: ".gaea/play/exports", workDir: ".gaea/play/work" }
        : { space, modeOn: true, exportsDir: ".gaea/exports", workDir: ".gaea/work" };
    },
    // ── 批次三a legacy 直调转正（Go CoreB，同名前缀）────────────────
    async ListProjects() {
      // 书架工程卡片：mock 无真实工程 → 空清单（listShelfProjects 空态，不编造）。
      return [];
    },
  };
}
