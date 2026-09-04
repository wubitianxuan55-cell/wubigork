// mock/core.ts — 平台/工作区/能力/更新域（T6-10.1 拆分自 lib/mock.ts，方法体零改动）。
// 说明：capServers 会被整体重绑（Remove/Retry/SetEnabled），必须经 s.capServers
// 读写（经 setCapServers 重绑），不能解构快照，否则 Capabilities 读到旧数组。
import type { AppBindings } from "../bridge";
import type { MCPServerInput } from "../types";
import { delay, emitUpdater } from "./shared";
import { switchWorkspace } from "./state";
import type { MakeMockState } from "./state";

type CoreMethods = Pick<
  AppBindings,
  | "ListWorkspaces" | "PickWorkspace" | "SwitchWorkspace"
  | "GaeaSpaceList" | "GaeaSpaceActive" | "GaeaSpaceActivate"
  | "ContextUsage" | "ContextView" | "ContextNodeDetail" | "Trajectory" | "AgentNetwork" | "TCCAReport" | "Jobs"
  | "Meta" | "Commands" | "Capabilities"
  | "AddMCPServer" | "RemoveMCPServer" | "RetryMCPServer" | "SetMCPServerEnabled"
  | "SlashArgs"
  | "Version" | "CheckUpdate" | "ApplyUpdate" | "OpenDownloadPage" | "SaveWindowState"
  | "LogFrontendError"
  // 编程板块：DeepSeek Harness Web 进程管理（浏览器 mock 恒为未运行）。
  | "GetProgrammingWebStatus" | "StartProgrammingWeb" | "StopProgrammingWeb"
  | "GetProgrammingWebPreflight" | "ProgrammingWebLogTail"
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
    async ContextView() {
      // 浏览器开发 mock：固定一份与效果图形态一致的快照。
      return {
        ok: true,
        window: 1_000_000,
        current: { system: 2100, tools: 10400, user: 20, inject: 21900, assistant: 93400, tool: 114000 },
        stats: { turns: 2, steps: 60, injects: 6, compacts: 0, prunes: 0, toolCalls: 279, images: 0, cacheHitPercent: 99.57, costEstimate: 3.83 },
        requests: [
          {
            seq: 8, ts: 1750000000, turn: 1, step: 1,
            category: { system: 2100, tools: 10400, user: 20, inject: 21000, assistant: 1200, tool: 8000 },
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
        files: [
          { seq: 5, ts: 1750000005, turn: 1, step: 1, tool: "read_file", action: "read", path: "internal/gaea/config/config.go" },
          { seq: 9, ts: 1750000009, turn: 1, step: 2, tool: "grep", action: "read", path: "internal/gaea/config", hits: 4 },
          { seq: 12, ts: 1750000012, turn: 1, step: 3, tool: "write_file", action: "write", path: "docs/调研结论.md", added: 42 },
        ],
      };
    },
    async ContextNodeDetail(seq: number) {
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
    async Trajectory() {
      // 浏览器开发 mock：固定一份与后端折叠形态一致的轨迹快照。
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
    async AgentNetwork() {
      // 浏览器开发 mock：主 agent + 两个子代理（一完成一运行），运行中的
      // 子代理带嵌套孙节点（AgentTree 深层展开/收起用）。节点 id 与
      // SubagentRuns 的 ref 对齐（ref 直等匹配，面板富化运行预览）。
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
    async GetProgrammingWebStatus() {
      // 浏览器开发模式无 Go 进程，恒返回未运行（空状态引导正常展示）。
      return {
        running: false,
        owned: false,
        pid: 0,
        url: "http://127.0.0.1:3080",
        root: "C:\\AI\\deepseek-harness",
        log: "gaea-dsh-web.log",
        uptime_s: 0,
      };
    },
    async GetProgrammingWebPreflight() {
      // 浏览器开发模式：假定本机 harness 已就绪（未运行 → 端口空闲），
      // 启动引导视图可完整演示。
      return {
        harness_valid: true,
        pnpm_found: true,
        deps_ready: true,
        build_ready: true,
        port_free: true,
        all_ready: true,
        root: "C:\\AI\\deepseek-harness",
      };
    },
    async ProgrammingWebLogTail() {
      // 浏览器开发模式无真实日志：返回未生成提示（面板空态展示）。
      return {
        exists: false,
        path: "gaea-dsh-web.log",
        lines: [],
        error: "日志文件尚未生成（第一次启动后出现）",
      };
    },
    async StartProgrammingWeb() {
      // 延迟 3s 再报错：浏览器开发模式无法真实启动，但保留一段「启动动画」
      // 演示窗口，让启动中视图可被看到（真实桌面版由 Go 侧驱动）。
      await delay(3000);
      throw new Error("浏览器开发模式不支持启动 DeepSeek Harness（请运行桌面版）");
    },
    async StopProgrammingWeb() {
      throw new Error("浏览器开发模式无 gaea 自启进程");
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
    async GaeaSpaceActivate(space: string) {
      await delay(120);
      if (space !== "work" && space !== "play") {
        throw new Error(`非法空间 ${space}（仅 work|play）`);
      }
      return space === "play"
        ? { space, modeOn: true, exportsDir: ".gaea/play/exports", workDir: ".gaea/play/work" }
        : { space, modeOn: true, exportsDir: ".gaea/exports", workDir: ".gaea/work" };
    },
  };
}
