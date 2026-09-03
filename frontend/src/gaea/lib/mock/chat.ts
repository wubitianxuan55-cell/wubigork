// mock/chat.ts — 对话/轻语/会话域（T6-10.1 拆分自 lib/mock.ts，方法体零改动）。
import type { AppBindings } from "../bridge";
// chat 板块契约类型（wails 生成物；AppBindings 契约同步 T6-3）。
import type { app as AppModels, chat } from "../../../../wailsjs/go/models";
import { delay, emit, mockScenario } from "./shared";
import type { MakeMockState } from "./state";
import type { HistoryMessage, QuestionAnswer } from "../types";

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

// 季度报告会话（/mock/sessions/a.jsonl，preview「compile quarterly report」）
// 恢复回放：一段真实感办公任务——查素材 → docx 转 Markdown → 查知识库规范
// → 生成对比图 → 架构图 → multi_edit 改报告 → 收尾。五个办公工具卡的输出
// 格式与 lib/tools.ts summarize/subjectOf 的解析口径逐字对齐（信封 JSON
// 字符串、全角括号「（31409 字节，类型: …）」、message 内换行为 JSON \n、
// knowledge_search 两个「### 」行首小节、diagram_gen 裸 JSON 非信封），
// 供浏览器端工具卡渲染联调；契约由 mock-session-replay.test.ts 锁定。
function quarterlyReportReplay(): HistoryMessage[] {
  return [
    { role: "user", content: "把本季度的经营数据汇总一下，整理出季度报告，并配一张分项成本对比图。" },
    { role: "assistant", content: "好的。我先确认工作区素材，再把 docx 底稿转成 Markdown 汇总进报告。" },
    { role: "tool", content: "", toolId: "mock-call-1", toolName: "ls", toolArgs: '{"path":"."}' },
    { role: "tool_result", toolId: "mock-call-1", toolName: "ls", content: "方案.md\n成本测算.md\n表格.xlsx\n素材.docx" },
    { role: "assistant", content: "素材已确认：季度数据在 表格.xlsx，文字底稿在 素材.docx。先把 docx 转成可编辑的 Markdown。" },
    { role: "tool", content: "", toolId: "mock-call-2", toolName: "format_convert", toolArgs: '{"path":"素材.docx","target":"markdown"}' },
    { role: "tool_result", toolId: "mock-call-2", toolName: "format_convert", content: '{"ok":true,"success":true,"code":0,"message":"已转换并保存为 out/季度报告.md（3210 字符）"}' },
    { role: "assistant", content: "底稿已转成 out/季度报告.md。动笔画图前，先查一下知识库里的图表与报告规范。" },
    { role: "tool", content: "", toolId: "mock-call-3", toolName: "knowledge_search", toolArgs: '{"query":"季度报告 图表规范"}' },
    { role: "tool_result", toolId: "mock-call-3", toolName: "knowledge_search", content: '{"ok":true,"success":true,"code":0,"message":"命中 2 条「季度报告图表」相关规范：\\n### 季度图表配色规范\\n对比图统一使用分组柱状图，主系列取品牌蓝、对比系列取中性灰，数值标签保留一位小数。\\n### 报告结构与图表标注规范\\n图表插入对应章节小节之后，标题置于图下方并按「图 N」编号。"}' },
    { role: "assistant", content: "按规范：对比图用分组柱状图、标题置图下并编号。现在基于汇总数据生成分项单价对比图。" },
    { role: "tool", content: "", toolId: "mock-call-4", toolName: "chart_gen", toolArgs: '{"title":"分项单价对比","labels":["人力","物料","物流"],"chart_type":"grouped_bar","series":[{"name":"本季度","data":[120,86,64]},{"name":"上季度","data":[108,92,70]}]}' },
    { role: "tool_result", toolId: "mock-call-4", toolName: "chart_gen", content: '{"ok":true,"success":true,"code":0,"message":"✅ 图表已生成: charts/cost.png（31409 字节，类型: grouped_bar）\\n标题: 分项单价对比\\n系列: 2 · 类别: 3"}' },
    { role: "assistant", content: "对比图 charts/cost.png 已生成。报告还需要一张处理链路架构图，一并补上。" },
    { role: "tool", content: "", toolId: "mock-call-5", toolName: "diagram_gen", toolArgs: '{"title":"季度报告处理链路","nodes":["数据汇总","格式转换","图表生成","报告成稿"],"edges":[["数据汇总","格式转换"],["格式转换","图表生成"],["图表生成","报告成稿"]]}' },
    { role: "tool_result", toolId: "mock-call-5", toolName: "diagram_gen", content: '{"ok":true,"output":"diagrams/架构.png","size_bytes":40960}' },
    { role: "assistant", content: "架构图已生成。最后把数据结论与两张图写回报告正文，共三处修改。" },
    { role: "tool", content: "", toolId: "mock-call-6", toolName: "multi_edit", toolArgs: '{"path":"out/季度报告.md","edits":[{"old_string":"## 一、总体经营情况\\n（待补充）","new_string":"## 一、总体经营情况\\n本季度营收环比增长 12%，三项费用率下降 1.8 个百分点。"},{"old_string":"（图表待插入）","new_string":"![分项单价对比](charts/cost.png)\\n图 1 分项单价对比（本季度 vs 上季度）"},{"old_string":"> 状态：初稿","new_string":"> 状态：定稿"}]}' },
    { role: "tool_result", toolId: "mock-call-6", toolName: "multi_edit", content: "已更新 out/季度报告.md（3 处修改）" },
    { role: "assistant", content: "季度报告已整理完成：正文三处更新，charts/cost.png 与 diagrams/架构.png 两张配图均已就位。如需导出 PDF 或调整图表配色，随时告诉我。" },
  ];
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
      // 与真实内核一致：resume 把目标会话置为当前（侧栏「当前」徽标跟随）。
      sessions.forEach((x) => { x.current = x.path === path; });
      // 季度报告会话（a.jsonl）：真实感办公任务回放，覆盖五类办公工具卡样例
      // （quarterlyReportReplay 注释里有格式对齐说明）；其余路径保持原序列。
      const msgs: HistoryMessage[] = path === "/mock/sessions/a.jsonl"
        ? quarterlyReportReplay()
        : [
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
