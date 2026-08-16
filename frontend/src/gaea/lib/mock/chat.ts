// mock/chat.ts — 对话/轻语/会话域（T6-10.1 拆分自 lib/mock.ts，方法体零改动）。
import type { AppBindings } from "../bridge";
// chat 板块契约类型（wails 生成物；AppBindings 契约同步 T6-3）。
import type { app as AppModels, chat } from "../../../../wailsjs/go/models";
import { delay, emit } from "./shared";
import type { MakeMockState } from "./state";

type ChatMethods = Pick<
  AppBindings,
  | "Submit" | "SubmitDisplay" | "Cancel" | "Approve" | "AnswerQuestion"
  | "GaeaRunning" | "SetAgentMode" | "AgentMode" | "Compact" | "NewSession"
  | "Reload" | "CaptureSkill" | "Checkpoints" | "Rewind" | "Fork"
  | "SummarizeFrom" | "SummarizeUpTo" | "History"
  | "ListSessions" | "ListProjectSessions" | "ArchiveSession" | "UnarchiveSession"
  | "PinSession" | "ResumeSession" | "DeleteSession" | "RenameSession"
  | "Requirement" | "SetRequirement" | "SetRequirementDone"
  | "SessionStats"
  | "ChatTopicsList" | "ChatMessagesList" | "ChatAppendMessages"
>;

export function buildChat(s: MakeMockState): ChatMethods {
  const { runningMock, sessions, archivedMock, requirementsMock, projectGroupsMock } = s;
  let cancelled = false; // Submit/Cancel 共享的中断标记（原 makeMockApp 局部状态）
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
      emit({ kind: "tool_dispatch", tool: { id: "t2", name: "write_file", args: '{"path":"季度总结.md","content":"# 季度总结\\n\\n经营数据平稳增长。"}', readOnly: false } });
      await delay(350);
      emit({ kind: "tool_result", tool: { id: "t2", name: "write_file", output: "已写入 季度总结.md", readOnly: false } });
      emit({ kind: "tool_dispatch", tool: { id: "t3", name: "edit_file", args: '{"path":"方案.md","edits":[{"old":"初稿","new":"终稿"}]}', readOnly: false } });
      await delay(300);
      emit({ kind: "tool_result", tool: { id: "t3", name: "edit_file", output: "已更新 方案.md", readOnly: false } });
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
    async GaeaRunning() { return false; },
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
    async ListProjectSessions() {
      return projectGroupsMock.map((g) => ({
        ...g,
        sessions: g.current ? sessions.map((s) => ({ ...s })) : g.sessions.map((s) => ({ ...s })),
        archived: g.current ? archivedMock.map((s) => ({ ...s })) : g.archived.map((s) => ({ ...s })),
      }));
    },
    async ArchiveSession(path: string) {
      const i = sessions.findIndex((s) => s.path === path);
      if (i >= 0) {
        const [s] = sessions.splice(i, 1);
        archivedMock.push({ ...s, archived: true, pinned: false });
      }
    },
    async UnarchiveSession(path: string) {
      const i = archivedMock.findIndex((s) => s.path === path);
      if (i >= 0) {
        const [s] = archivedMock.splice(i, 1);
        sessions.push({ ...s, archived: false });
        return path;
      }
      return "";
    },
    async PinSession(path: string, pinned: boolean) {
      const s = sessions.find((x) => x.path === path);
      if (s) s.pinned = pinned;
    },
    async ResumeSession(path: string) {
      // T5-4 契约：interrupted 会话恢复时注入「上次会话中断」摘要提示并清除标记。
      const s = sessions.find((x) => x.path === path);
      const wasInterrupted = !!s?.interrupted;
      if (s) s.interrupted = false;
      const msgs = [
        { role: "user", content: `(mock) resumed ${path}` },
        { role: "assistant", content: "让我看看之前改到哪里了。" },
        { role: "tool", content: "", toolId: "mock-call-1", toolName: "edit_file", toolArgs: '{"path":"方案.md","edits":[]}' },
        { role: "tool_result", toolId: "mock-call-1", toolName: "edit_file", content: "已更新 方案.md" },
        { role: "assistant", content: "方案已按上次进度继续完善。" },
      ];
      if (wasInterrupted) {
        msgs.unshift({ role: "assistant", content: "⚠️ 上次会话中断未完成，已从中断点继续（mock 摘要）。" });
      }
      return msgs;
    },
    async DeleteSession(path: string) {
      const i = sessions.findIndex((s) => s.path === path);
      if (i >= 0) sessions.splice(i, 1);
    },
    async SessionStats(path: string) {
      // 派生统计 mock：可用 + 固定示例值（浏览器开发模式回填历史成本用）。
      return {
        available: true,
        stats: {
          promptTokens: 128000,
          completionTokens: 32000,
          totalTokens: 160000,
          cacheHitTokens: 96000,
          cacheMissTokens: 32000,
          reasoningTokens: 0,
          usageCount: 12,
          cost: 0.284,
          currency: "usd",
          mainCost: 0.211,
          subagentCost: 0.073,
        },
      };
    },
    async RenameSession(path: string, title: string) {
      const s = sessions.find((x) => x.path === path);
      if (s) s.title = title.trim() || undefined;
    },
    async Requirement(path: string) {
      return requirementsMock.get(path) ?? { text: "", done: false, updatedAt: 0 };
    },
    async SetRequirement(path: string, text: string) {
      const prev = requirementsMock.get(path) ?? { text: "", done: false, updatedAt: 0 };
      if (!text.trim()) {
        requirementsMock.delete(path);
        return;
      }
      requirementsMock.set(path, { text: text.trim(), done: prev.done, updatedAt: Date.now() });
    },
    async SetRequirementDone(path: string, done: boolean) {
      const r = requirementsMock.get(path);
      if (r && r.text) requirementsMock.set(path, { ...r, done, updatedAt: Date.now() });
    },
    // ── 对话 chat（T6-3 契约同步：ChatTopicsList/ChatMessagesList 返回
    // [数据, 错误] 元组形态；ChatAppendMessages 语音消息持久化 no-op）──
    async ChatTopicsList(): Promise<[chat.Topic[], unknown]> {
      return [[], null];
    },
    async ChatMessagesList(_topicID: string): Promise<[chat.Message[], unknown]> {
      return [[], null];
    },
    async ChatAppendMessages(_topicID: string, _messages: AppModels.ChatMessageInput[]) {
      // mock: 浏览器开发环境不落库（no-op）——无真实 whisper 库，语义与
      // Go 侧「未初始化聊天库时静默丢弃」一致。
    },
  };
}
