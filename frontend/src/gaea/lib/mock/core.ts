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
  | "ContextUsage" | "TCCAReport" | "Jobs"
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
      throw new Error("浏览器开发模式不支持启动 DeepSeek Harness（请运行桌面版）");
    },
    async StopProgrammingWeb() {
      throw new Error("浏览器开发模式无 gaea 自启进程");
    },
  };
}
