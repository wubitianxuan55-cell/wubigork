// mock/sin.ts — 原罪（闲庭·图文故事创作）dev mock。
//
// 口径：会话与消息为内存态（浏览器无 chat.db），列表/读取/新建/重命名/删除/
// 清空全按真实契约返回；SinStream 模拟真实流式——经 window.runtime.EventsEmit
// 下发 `sin-stream:<runID>` 的 delta/done 帧（与真机同事件形态，runtimePolyfill
// 的内存事件总线负责投递）；SinIllustrate 返回占位产物路径（浏览器无图像后端，
// AttachmentDataURL 占位色块让「图文混杂」版面可走查）；导出按内存消息渲染
// Markdown（与 Go SinExportMarkdown 同规则：标记 → 图片/占位）。
import type { AppBindings } from "../bridge";
import type { ContextTimeline, Trajectory } from "../types";
import type { chat } from "../../../../wailsjs/go/models";
import { SIN_CUE_CLOSE, SIN_CUE_OPEN } from "../../../pages/sin/storyText";

type SinMethods = Pick<
  AppBindings,
  | "SinTopicsList" | "SinTopicCreate" | "SinTopicRename" | "SinTopicDelete"
  | "SinTopicClear" | "SinMessages" | "SinStream" | "SinIllustrate" | "SinExportMarkdown"
  | "SinExportEpub"
  | "SinCastGet" | "SinCastSet" | "SinCancel" | "SinNotesGet" | "SinNotesSave"
  | "SinTrajectory" | "SinContextView" | "SinContextNodeDetail"
  | "SinBookSourceSearch" | "SinBookSourceToc" | "SinBookSourceDownload"
  | "SinBookSourceDownloadCancel" | "SinBookSourceBooksList" | "SinBookSourceBookDelete"
  | "SinBookSourceBookExportEpub"
>;

interface MockStory {
  id: string;
  title: string;
  mode: string;
  created_at: string;
  updated_at: string;
  preview: string;
}

interface MockSinMessage {
  id: number;
  topic_id: string;
  role: string;
  content: string;
  extra: string;
  seq: number;
  created_at: string;
}

/** 流式样例正文：一段故事 + 两处插图标记（走查图文混杂与自动出图）。 */
const DEMO_REPLY = [
  "雨把站台的灯拉成一条条竖线。她把相机压在大衣里，数着第三节车厢亮着的窗。",
  "",
  SIN_CUE_OPEN + "雨夜站台，女人侧身抓拍，湿风衣反光，胶片颗粒，冷蓝主色" + SIN_CUE_CLOSE,
  "",
  "车门开合的间隙只有两秒。她侧身挤进去，正撞上那人抬起的手——指节上有旧伤。",
  "「你拍到我了。」他说，声音压得很低，像怕被车厢里的其他人听见。",
  "",
  SIN_CUE_OPEN + "车厢内近景特写，男人抬起的手与指节旧伤，暖色顶灯，浅景深" + SIN_CUE_CLOSE,
  "",
  "她没有否认，只是把相机往身后挪了半寸。列车启动，两个人的影子同时晃了一下。",
].join("\n");

/** 走查用工具轨迹：与后端 sinToolTrace 同形（snake_case，键名逐字对应），
 *  既喂流式过程卡（dispatch → result），也写进消息 extra.tools 供重开还原。 */
const DEMO_TOOLS = [
  {
    id: "call_demo_1", name: "web_search",
    args: JSON.stringify({ query: "1980年代 通勤电车 车厢内景 顶灯" }),
    output: '[{"title":"通勤电车车厢内饰","url":"https://example.com/train","snippet":"暖色顶灯、吊环、雨天玻璃反光"}]',
    error: "", elapsed_ms: 1240, read_only: true,
  },
  {
    id: "call_demo_2", name: "sin_notes",
    args: JSON.stringify({ action: "write", content: "女主：林晚，地方台记者；男主：指节有旧伤" }),
    output: "已记录便签 #0（共 1 条）。",
    error: "", elapsed_ms: 5, read_only: false,
  },
];

/** mock 消息 extra 读取投影（与 Go sinMsgExtra / 前端 storyText 同键名）。 */
interface MockSinToolTrace {
  id: string; name: string; args: string; output: string;
  error?: string; elapsed_ms?: number; read_only?: boolean;
}

function mockExtra(raw: string): {
  reasoning?: string;
  tools: MockSinToolTrace[];
  illustrations?: Record<string, string>;
} {
  try {
    const p = JSON.parse(raw || "{}") as Record<string, unknown>;
    return {
      reasoning: typeof p.reasoning === "string" ? p.reasoning : undefined,
      tools: Array.isArray(p.tools) ? (p.tools as MockSinToolTrace[]) : [],
      illustrations: (p.illustrations as Record<string, string>) ?? {},
    };
  } catch {
    return { tools: [], illustrations: {} };
  }
}

/** 与 Go sinNodeSeq 同编址：消息 ID×10 + 槽位（0=user、1..8=tool、9=assistant）。 */
function mockSeq(msgId: number, slot: number): number {
  return msgId * 10 + slot;
}

export function buildSin(): SinMethods {
  let seq = 0;
  const stories: MockStory[] = [];
  const messages = new Map<string, MockSinMessage[]>();
  // 角色选择内存态（故事 id → 角色库 id）；角色库本身由 mock/weixin.ts 提供。
  const casts = new Map<string, string[]>();
  // 便签（设定集）与大纲内存态（故事 id → { notes, outline }）：
  // 真机由 sin_notes/sin_outline 工具落 notes/<故事 id>.json，这里给右栏面板
  // 同样的可走查数据（种子与 DEMO_TOOLS 里写的那条便签对齐）。
  const notesDocs = new Map<string, { notes: string[]; outline: string }>();
  // 在途流（runID → 是否已被取消）：SinCancel 用它中止模拟流式
  const runningStreams = new Map<string, { cancelled: boolean; topicID: string }>();
  const now = () => new Date().toISOString();

  const seed = () => {
    if (stories.length > 0) return;
    const id = "sin_mock_1";
    const ts = now();
    stories.push({
      id, title: "雨夜电车", mode: "sin",
      created_at: ts, updated_at: ts,
      preview: "写一个雨夜电车上的开场，主角是女记者和一个陌生男人",
    });
    messages.set(id, [
      {
        id: ++seq, topic_id: id, role: "user",
        content: "写一个雨夜电车上的开场，主角是女记者和一个陌生男人",
        extra: "", seq: 1, created_at: ts,
      },
      {
        id: ++seq, topic_id: id, role: "assistant",
        content: DEMO_REPLY,
        extra: JSON.stringify({ illustrations: { "0": ".gaea/uploads/sin-mock-1.png" }, tools: DEMO_TOOLS }),
        seq: 2, created_at: ts,
      },
    ]);
    notesDocs.set(id, {
      notes: ["女主：林晚，地方台记者；男主：指节有旧伤，认识她"],
      outline: "第一章：雨夜站台相遇\n第二章：车厢对峙与旧伤来历\n第三章：相机里的秘密",
    });
  };

  const emitFrame = (runID: string, payload: Record<string, unknown>) => {
    if (typeof window === "undefined") return;
    const rt = (window as unknown as { runtime?: { EventsEmit?: (e: string, ...a: unknown[]) => void } }).runtime;
    rt?.EventsEmit?.(`sin-stream:${runID}`, payload);
  };

  return {
    async SinTopicsList() {
      seed();
      return stories.slice() as unknown as chat.Topic[];
    },
    async SinTopicCreate(title: string) {
      seed();
      const id = `sin_mock_${stories.length + 1}`;
      const ts = now();
      const story: MockStory = {
        id, title: title?.trim() || "新故事", mode: "sin",
        created_at: ts, updated_at: ts, preview: "",
      };
      stories.push(story);
      messages.set(id, []);
      return story as unknown as { id: string; title: string; mode: string };
    },
    async SinTopicRename(id: string, title: string) {
      const s = stories.find((x) => x.id === id);
      if (s) { s.title = title.trim() || s.title; s.updated_at = now(); }
    },
    async SinTopicDelete(id: string) {
      const i = stories.findIndex((x) => x.id === id);
      if (i >= 0) stories.splice(i, 1);
      messages.delete(id);
      casts.delete(id);
      notesDocs.delete(id);
    },
    async SinTopicClear(id: string) {
      messages.set(id, []);
    },
    async SinMessages(topicID: string) {
      seed();
      return (messages.get(topicID) ?? []) as unknown as chat.Message[];
    },
    // ── v4.412 看板复用：与 Go sin_insight 同规则的内存折叠（估算口径） ──
    async SinTrajectory(topicID: string) {
      seed();
      const turns: Trajectory["turns"] = [];
      let cur: Trajectory["turns"][number] | null = null;
      for (const m of messages.get(topicID) ?? []) {
        const ts = Math.floor(new Date(m.created_at + "Z").getTime() / 1000) || 0;
        const ex = mockExtra(m.extra);
        if (m.role === "user") {
          if (cur) turns.push(cur);
          cur = {
            turn: turns.length + 1, startedAt: ts, records: [{
              seq: mockSeq(m.id, 0), kind: "user", ts, user: { text: m.content },
            }],
          };
          continue;
        }
        const recs: Trajectory["turns"][number]["records"] = ex.tools.map((t, i) => ({
          seq: mockSeq(m.id, Math.min(i + 1, 8)), kind: "tool", ts,
          durationMs: t.elapsed_ms || undefined,
          tool: {
            id: t.id, name: t.name, args: t.args, output: t.output, err: t.error || undefined,
            readOnly: t.read_only || undefined, status: t.error ? "error" : "ok",
          },
        }));
        recs.push({
          seq: mockSeq(m.id, 9), kind: "assistant", ts,
          assistant: { text: m.content, reasoning: ex.reasoning || undefined },
        });
        if (!cur) continue; // 孤儿助手消息：mock 走查态直接丢弃（真机归轮间）
        cur.records.push(...recs);
        const end = recs[recs.length - 1];
        cur.end = { seq: end.seq, ts: end.ts };
        if (end.ts > (cur.startedAt ?? 0)) cur.durationMs = (end.ts - (cur.startedAt ?? 0)) * 1000;
      }
      if (cur) turns.push(cur);
      return { ok: true, turns };
    },
    async SinContextView(topicID: string) {
      seed();
      const list = messages.get(topicID) ?? [];
      const est = (s: string) => Math.round(s.length * 0.25);
      const catTotal = (c: ContextTimeline["current"]) => c.system + c.tools + c.user + c.inject + c.assistant + c.tool;
      const nodes: ContextTimeline["nodes"] = [];
      const requests: ContextTimeline["requests"] = [];
      let cumUser = 0, cumAsst = 0, cumTool = 0, turn = 0, toolCalls = 0, images = 0;
      let curUser = "";
      let curUserId = 0;
      let prevNodes = nodes.length;
      const mockSystem = "你是原罪板块的图文故事合作写作者。";
      nodes.push({ seq: 1, cat: "system", tokens: est(mockSystem), text: mockSystem });
      for (const m of list) {
        const ex = mockExtra(m.extra);
        images += Object.keys(ex.illustrations ?? {}).length;
        if (m.role === "user") {
          turn += 1;
          curUser = m.content;
          curUserId = m.id;
          cumUser += est(m.content);
          nodes.push({ seq: mockSeq(m.id, 0), cat: "user", tokens: est(m.content), text: m.content.slice(0, 120) });
          continue;
        }
        for (const [i, t] of ex.tools.entries()) {
          cumTool += est(t.output);
          toolCalls += 1;
          nodes.push({
            seq: mockSeq(m.id, Math.min(i + 1, 8)), cat: "tool", tokens: est(t.output),
            text: t.output.slice(0, 120), tool: t.name, err: !!t.error,
          });
        }
        cumAsst += est(m.content);
        nodes.push({ seq: mockSeq(m.id, 9), cat: "assistant", tokens: est(m.content), text: m.content.slice(0, 120) });
        if (turn === 0) continue;
        const category = { system: est(mockSystem), tools: 0, user: cumUser, inject: 0, assistant: cumAsst, tool: cumTool };
        const prev = requests[requests.length - 1]?.category;
        requests.push({
          seq: requests.length + 1, ts: Math.floor(new Date(m.created_at + "Z").getTime() / 1000) || 0,
          turn, step: 1, category, estimated: true,
          briefUser: curUser.split("\n")[0].slice(0, 48), briefUserSeq: mockSeq(curUserId, 0),
          briefResp: m.content.split("\n")[0].slice(0, 48), briefRespSeq: mockSeq(m.id, 9),
          delta: {
            items: nodes.length - prevNodes, first: requests.length === 0,
            tokens: requests.length === 0 ? catTotal(category) : catTotal(category) - catTotal(prev!),
          },
        });
        prevNodes = nodes.length;
      }
      const timing = toolCalls > 0 ? {
        toolsMs: (list.flatMap((m) => mockExtra(m.extra).tools)).reduce((a, t) => a + (t.elapsed_ms || 0), 0),
        toolCalls,
        tools: Object.entries(
          (list.flatMap((m) => mockExtra(m.extra).tools)).reduce<Record<string, { calls: number; ms: number }>>((acc, t) => {
            acc[t.name] = { calls: (acc[t.name]?.calls ?? 0) + 1, ms: (acc[t.name]?.ms ?? 0) + (t.elapsed_ms || 0) };
            return acc;
          }, {}),
        ).map(([name, v]) => ({ name, ...v })).sort((a, b) => b.ms - a.ms),
      } : undefined;
      return {
        ok: true, window: 0,
        current: requests[requests.length - 1]?.category ?? { system: 0, tools: 0, user: 0, inject: 0, assistant: 0, tool: 0 },
        stats: { turns: turn, steps: requests.length, injects: 0, compacts: 0, prunes: 0, toolCalls, images },
        requests, events: [], nodes, archive: [], files: [], timing,
      };
    },
    async SinContextNodeDetail(topicID: string, seq: number) {
      seed();
      for (const m of messages.get(topicID) ?? []) {
        if (m.role === "user" && mockSeq(m.id, 0) === seq) {
          return { seq, kind: "user_message", text: m.content, lines: m.content.split("\n").length };
        }
        if (m.role !== "assistant") continue;
        if (mockSeq(m.id, 9) === seq) {
          return { seq, kind: "assistant_message", text: m.content, lines: m.content.split("\n").length };
        }
        const tools = mockExtra(m.extra).tools;
        for (const [i, t] of tools.entries()) {
          if (mockSeq(m.id, Math.min(i + 1, 8)) !== seq) continue;
          return {
            seq, kind: "tool_result", tool: t.name, args: t.args, output: t.output,
            err: t.error || undefined, lines: (t.output || "").split("\n").length,
          };
        }
      }
      throw new Error("未找到可展开的节点");
    },
    async SinStream(topicID: string, message: string) {
      seed();
      const runID = `ss_mock_${Date.now()}`;
      const list = messages.get(topicID) ?? [];
      const base = list.length;
      const story = stories.find((x) => x.id === topicID);
      if (story && (!story.title || story.title === "新故事")) story.title = message.slice(0, 16) || "新故事";
      // 用户消息先落内存（与真机一致：落库在完成后，这里为走查便利先行）
      list.push({
        id: ++seq, topic_id: topicID, role: "user", content: message,
        extra: "", seq: base + 1, created_at: now(),
      });
      messages.set(topicID, list);

      const handle = { cancelled: false, topicID };
      runningStreams.set(runID, handle);
      void (async () => {
        // 过程帧与真机同序同形：reasoning → tool_dispatch → tool_result → delta → done。
        // 运行态可走查（dispatch 与 result 之间有停顿），且不伪造百分比。
        emitFrame(runID, { type: "reasoning", content: "（mock）先把现实细节查准，再落笔。" });
        for (const tool of DEMO_TOOLS) {
          if (handle?.cancelled) break;
          emitFrame(runID, {
            type: "tool_dispatch", id: tool.id, name: tool.name,
            args: tool.args, read_only: tool.read_only,
          });
          await new Promise((r) => setTimeout(r, 220));
          emitFrame(runID, {
            type: "tool_result", id: tool.id, name: tool.name,
            output: tool.output, error: tool.error, elapsed_ms: tool.elapsed_ms,
          });
          await new Promise((r) => setTimeout(r, 80));
        }
        const step = 24;
        let sent = "";
        for (let i = 0; i < DEMO_REPLY.length; i += step) {
          if (handle?.cancelled) break;
          sent = DEMO_REPLY.slice(0, i + step);
          emitFrame(runID, { type: "delta", content: DEMO_REPLY.slice(i, i + step) });
          await new Promise((r) => setTimeout(r, 18));
        }
        const cancelled = !!handle?.cancelled;
        runningStreams.delete(runID);
        const assistant: MockSinMessage = {
          id: ++seq, topic_id: topicID, role: "assistant",
          content: cancelled ? sent : DEMO_REPLY,
          extra: JSON.stringify(cancelled
            ? { reasoning: "（mock）用户停止", cancelled: true, tools: DEMO_TOOLS }
            : { reasoning: "（mock）按雨夜站台→车厢近景推进", tools: DEMO_TOOLS }),
          seq: base + 2, created_at: now(),
        };
        list.push(assistant);
        emitFrame(runID, {
          type: "done", reply: assistant.content, reasoning: "", cancelled,
          topicID, message_id: assistant.id,
          tools: DEMO_TOOLS,
          answered_by: { engine: "mock", model: "mock-story", source: "feature", cost_cny: 0 },
        });
      })();
      return runID;
    },
    async SinCancel(topicID: string) {
      // mock：标记该话题在途流为已取消（真机还会中止底层请求并保留已生成部分）
      for (const h of runningStreams.values()) {
        if (h.topicID === topicID) h.cancelled = true;
      }
    },
    async SinCastGet(topicID: string) {
      seed();
      return (casts.get(topicID) ?? []).slice();
    },
    async SinNotesGet(topicID: string) {
      seed();
      const doc = notesDocs.get(topicID);
      return doc ? { notes: doc.notes.slice(), outline: doc.outline } : { notes: [], outline: "" };
    },
    async SinNotesSave(topicID: string, baseline: string, outline: string, notes: string, force: boolean) {
      seed();
      // 与真机同口径：锁内基线比对（mock 无锁但保留冲突语义供走查）
      const base = JSON.parse(baseline || "{}") as { notes?: string[]; outline?: string };
      const cur = notesDocs.get(topicID) ?? { notes: [], outline: "" };
      if (!force && (cur.outline !== (base.outline ?? "") || JSON.stringify(cur.notes) !== JSON.stringify(base.notes ?? []))) {
        throw new Error("底稿冲突：便签/大纲已被其他端更新，请确认后再保存");
      }
      const next = { notes: (JSON.parse(notes || "[]") as string[]).slice(), outline };
      notesDocs.set(topicID, next);
      return { notes: next.notes.slice(), outline: next.outline };
    },
    // ── sin 书源 t2：浏览器无书源规则目录与真网络，走查只验版面——诚实空态/拒绝 ──
    async SinBookSourceSearch() {
      return {
        candidates: [],
        warnings: ["dev mock：书源搜索需真实网络与规则目录，浏览器走查仅验版面"],
      };
    },
    async SinBookSourceToc() {
      return { total: 0, sample: [], truncated: false };
    },
    async SinBookSourceDownload() {
      throw new Error("dev mock：下载成书需真实网络与书源规则");
    },
    async SinBookSourceDownloadCancel() {
      return false;
    },
    async SinBookSourceBooksList() {
      return [];
    },
    async SinBookSourceBookDelete() {
      /* mock 无成书可删：如实 no-op */
    },
    async SinBookSourceBookExportEpub() {
      throw new Error("dev mock：EPUB 导出需桌面端文件系统");
    },
    async SinExportEpub() {
      throw new Error("dev mock：EPUB 导出需桌面端文件系统");
    },
    async SinCastSet(topicID: string, ids: string[]) {
      seed();
      // 去重保序（真实实现还会按角色库过滤悬空 id 并封顶 8）
      const out: string[] = [];
      for (const id of ids ?? []) {
        const v = String(id ?? "").trim();
        if (!v || out.includes(v)) continue;
        out.push(v);
        if (out.length >= 8) break;
      }
      if (out.length === 0) casts.delete(topicID);
      else casts.set(topicID, out);
      return out;
    },
    async SinIllustrate(topicID: string, messageID: number, cue: string, _prompt: string, _size: string) {
      // 浏览器无图像后端：返回占位产物路径（AttachmentDataURL 出占位色块），
      // 并把路径回写内存消息 extra，重开故事时按映射渲染。
      // v4.258：故意拖 ~1.8s 模拟真实出图耗时——进度动画（不定态光带 + 已用时 +
      // 排队位次）在 ?mock=1 下可走查；不伪造百分比（mock 的
      // GetComfyUITaskProgress 为空，UI 如实走不定态）。
      await new Promise((r) => setTimeout(r, 1800))
      const path = `.gaea/uploads/sin-mock-${cue || "0"}.png`;
      const list = messages.get(topicID) ?? [];
      const m = list.find((x) => x.id === messageID);
      // 角色一致性信息：dev mock 无图像后端，按「本故事是否选了角色」模拟参考槽
      // 命中（仅供走查徽标；真机的结果由后端按后端能力如实返回）。id → 名字用
      // mock 角色库的样例映射（与 mock/weixin.ts 的样例角色同源），未知回退 id。
      const MOCK_CAST_NAMES: Record<string, string> = {
        c_lin: "林晚", c_gucheng: "顾城", c_baimo: "白墨",
      };
      const castIds = casts.get(topicID) ?? [];
      const castNames = castIds.map((id) => MOCK_CAST_NAMES[id] ?? id);
      if (m) {
        let extra: Record<string, unknown> = {};
        try { extra = JSON.parse(m.extra || "{}") as Record<string, unknown>; } catch { extra = {}; }
        const arts = (extra.illustrations as Record<string, string>) ?? {};
        arts[cue] = path;
        extra.illustrations = arts;
        m.extra = JSON.stringify(extra);
      }
      return {
        path, cue, message_id: messageID, model: "mock-image", seed: 0, persisted: !!m,
        ref_used: castIds.length > 0,
        ref_characters: castNames,
        // 未选角色不算「不可用」（与真机一致：无角色可锚时不提示）
        ref_reason: "",
        ref_fallback: false,
        anchor_added: castNames,
      };
    },
    async SinExportMarkdown(topicID: string) {
      seed();
      const story = stories.find((x) => x.id === topicID);
      const list = messages.get(topicID) ?? [];
      const out: string[] = [`# ${story?.title ?? "原罪故事"}`, ""];
      for (const m of list) {
        if (m.role === "user") {
          out.push(`**我：**`, "", m.content, "");
          continue;
        }
        let arts: Record<string, string> = {};
        try {
          const parsed = JSON.parse(m.extra || "{}") as { illustrations?: Record<string, string> };
          arts = parsed.illustrations ?? {};
        } catch { arts = {}; }
        let rest = m.content;
        let idx = 0;
        let body = "";
        for (;;) {
          const start = rest.indexOf(SIN_CUE_OPEN);
          if (start < 0) { body += rest; break; }
          const end = rest.indexOf(SIN_CUE_CLOSE, start + SIN_CUE_OPEN.length);
          if (end < 0) { body += rest; break; }
          const prompt = rest.slice(start + SIN_CUE_OPEN.length, end).trim();
          body += rest.slice(0, start);
          body += arts[String(idx)]
            ? `![${prompt}](${arts[String(idx)]})`
            : `_（插图未生成：${prompt}）_`;
          rest = rest.slice(end + SIN_CUE_CLOSE.length);
          idx += 1;
        }
        out.push(body.trim(), "", "---", "");
      }
      return out.join("\n");
    },
  };
}
