// mock/chat.ts — 对话/轻语/会话域（T6-10.1 拆分自 lib/mock.ts，方法体零改动）。
import type { AppBindings } from "../bridge";
// chat 板块契约类型（wails 生成物；AppBindings 契约同步 T6-3）。
import type { app as AppModels, chat } from "../../../../wailsjs/go/models";
import { delay, emit, mockScenario } from "./shared";
import type { MakeMockState } from "./state";
import type { QuestionAnswer } from "../types";

type ChatMethods = Pick<
  AppBindings,
  | "Submit" | "SubmitDisplay" | "Cancel" | "Steer" | "Approve" | "AnswerQuestion"
  | "GaeaRunning" | "Compact" | "NewSession"
  | "Reload" | "CaptureSkill" | "Checkpoints" | "Rewind" | "Fork"
  | "SummarizeFrom" | "SummarizeUpTo" | "History"
  | "ListSessions" | "ListProjectSessions" | "ArchiveSession" | "UnarchiveSession"
  | "PinSession" | "ResumeSession" | "DeleteSession" | "RenameSession"
  | "SessionStats"
  | "ChatTopicsList" | "ChatMessagesList" | "ChatAppendMessages"
>;

// 完整一轮「普通」对话模拟（demo 默认路径）：turn_started → reasoning 逐字
// → 3 个工具（ls/write_file/edit_file）→ 正文逐字 → usage ×2 → turn_done。
// 与真实 Go 事件序列同构，是 UI 全链路（过程卡/工具卡/统计）离线可开发的基础。
async function runDemoTurn(input: string, cancelledRef: { v: boolean }, runningMock: boolean) {
  const cancelled = () => cancelledRef.v;
  emit({ kind: "turn_started" });
  await delay(300);
  if (cancelled()) return;
  if (runningMock) await delay(1500); // simulate existing reasoning in progress
  const isPoetry = /(诗|古诗|词)/.test(input);
  const think = isPoetry ? "用户想写诗，直接创作即可。"
    : `用户说"${input}"，先查看工作区里的资料再回复。`;
  for (const ch of think) { if (cancelled()) break; emit({ kind: "reasoning", reasoning: ch }); await delay(12); }
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
  for (const ch of reply) { if (cancelled()) break; emit({ kind: "text", text: ch }); await delay(10); }
  emit({ kind: "message", text: reply });
  emitUsageTwice();
  emit({ kind: "turn_done" });
}

function emitUsageTwice() {
  emit({ kind: "usage", usage: { promptTokens: 1200, completionTokens: 200, totalTokens: 1400, cacheHitTokens: 800, cacheMissTokens: 400, sessionCacheHitTokens: 800, sessionCacheMissTokens: 400 } });
  emit({ kind: "usage", usage: { promptTokens: 1280, completionTokens: 64, totalTokens: 1344, cacheHitTokens: 1024, cacheMissTokens: 256, sessionCacheHitTokens: 1024, sessionCacheMissTokens: 256 } });
}

// 审批场景（?mock=approval）：turn_started → 工具卡 → approval_request 挂起。
// Approve 后补发 tool_result → 正文 → turn_done（与真实流程一致）。
function runApprovalTurn(input: string) {
  emit({ kind: "turn_started" });
  emit({ kind: "reasoning", reasoning: `用户请求"${input}"，写入工作区文件需要审批确认。` });
  emit({ kind: "tool_dispatch", tool: { id: "appr-t1", name: "write_file", args: '{"path":"审批测试.md","content":"# 待审批内容\\n\\n这是一份需要审批才能落盘的文件。"}', readOnly: false } });
  emit({
    kind: "approval_request",
    approval: { id: "appr-1", tool: "write_file", subject: "写入 审批测试.md（3 行）" },
  });
}

// 提问场景（?mock=ask）：turn_started → reasoning → ask_request 挂起。
// AnswerQuestion 后补发正文 → turn_done。
function runAskTurn(input: string) {
  emit({ kind: "turn_started" });
  emit({ kind: "reasoning", reasoning: `用户请求"${input}"，先与用户确认任务范围与产出物。` });
  emit({
    kind: "ask_request",
    ask: {
      id: "ask-1",
      questions: [
        {
          id: "q1",
          header: "交付格式确认",
          prompt: "请确认这份季度经营报告的交付格式：",
          options: [
            { label: "Word 文档", description: "便于二次编辑与评审" },
            { label: "PDF", description: "定稿版本，适合直接分发" },
          ],
        },
      ],
    },
  });
}

// 压缩场景（?mock=compaction）：turn_started → 正文 → compaction_started →
// compaction_done → 继续正文 → turn_done。
async function runCompactionTurn(input: string, cancelledRef: { v: boolean }) {
  const cancelled = () => cancelledRef.v;
  emit({ kind: "turn_started" });
  await delay(200);
  const think = `用户说"${input}"，上下文已较长，需要压缩历史。`;
  for (const ch of think) { if (cancelled()) break; emit({ kind: "reasoning", reasoning: ch }); await delay(12); }
  emit({ kind: "compaction_started", compaction: { trigger: "auto", messages: 28 } });
  await delay(500);
  if (cancelled()) return;
  emit({
    kind: "compaction_done",
    compaction: {
      trigger: "auto",
      messages: 28,
      summary: "会话早期内容：撰写季度总结、成本测算表编制流程与审批记录。",
      archive: "sessions/archive/2026-08-compacted.jsonl",
    },
  });
  const reply = `已压缩 28 条早期消息，保留关键上下文。继续处理：**${input}**`;
  for (const ch of reply) { if (cancelled()) break; emit({ kind: "text", text: ch }); await delay(10); }
  emit({ kind: "message", text: reply });
  emitUsageTwice();
  emit({ kind: "turn_done" });
}

export function buildChat(s: MakeMockState): ChatMethods {
  const { runningMock, sessions, archivedMock, projectGroupsMock } = s;
  const cancelledRef = { v: false }; // 供异步场景读取最新值
  let pendingFlow: "approval" | "ask" | null = null; // 挂起中的审批/提问场景
  return {
    async Submit(input) {
      cancelledRef.v = false;
      pendingFlow = null;
      const scenario = mockScenario();
      if (scenario === "approval") { pendingFlow = "approval"; runApprovalTurn(input); return; }
      if (scenario === "ask") { pendingFlow = "ask"; runAskTurn(input); return; }
      if (scenario === "compaction") { await runCompactionTurn(input, cancelledRef); return; }
      await runDemoTurn(input, cancelledRef, runningMock);
    },
    async SubmitDisplay(_display, input) {
      await this.Submit(input);
    },
    async Cancel() {
      cancelledRef.v = true;
      pendingFlow = null;
      emit({ kind: "turn_done" });
    },
    async Steer(text: string) {
      // 运行中插话：模拟 agent 收到 guidance 后回显 notice 并继续。
      emit({ kind: "notice", level: "info", text: `已插话：${text.slice(0, 40)}` });
    },
    async Approve(_id: string, _decision: string) {
      if (pendingFlow !== "approval") return;
      pendingFlow = null;
      // 审批通过后补发工具结果与收尾（与真实 gaeaApprove → 工具继续一致）
      emit({ kind: "tool_result", tool: { id: "appr-t1", name: "write_file", output: "已写入 审批测试.md", readOnly: false } });
      const reply = "已获批准，文件已写入工作区。";
      for (const ch of reply) { emit({ kind: "text", text: ch }); }
      emit({ kind: "message", text: reply });
      emitUsageTwice();
      emit({ kind: "turn_done" });
    },
    async AnswerQuestion(_id: string, _answers: QuestionAnswer[]) {
      if (pendingFlow !== "ask") return;
      pendingFlow = null;
      // 回答后继续执行（与真实 gaeaAnswer → 回合继续一致）
      const reply = "已收到你的选择，继续执行。";
      for (const ch of reply) { emit({ kind: "text", text: ch }); }
      emit({ kind: "message", text: reply });
      emitUsageTwice();
      emit({ kind: "turn_done" });
    },
    async GaeaRunning() { return false; },
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
    async SessionStats(_path: string) {
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
