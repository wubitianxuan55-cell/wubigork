// mock/sin.ts — 原罪（闲庭·图文故事创作）dev mock。
//
// 口径：会话与消息为内存态（浏览器无 chat.db），列表/读取/新建/重命名/删除/
// 清空全按真实契约返回；SinStream 模拟真实流式——经 window.runtime.EventsEmit
// 下发 `sin-stream:<runID>` 的 delta/done 帧（与真机同事件形态，runtimePolyfill
// 的内存事件总线负责投递）；SinIllustrate 返回占位产物路径（浏览器无图像后端，
// AttachmentDataURL 占位色块让「图文混杂」版面可走查）；导出按内存消息渲染
// Markdown（与 Go SinExportMarkdown 同规则：标记 → 图片/占位）。
import type { AppBindings } from "../bridge";
import type { chat } from "../../../../wailsjs/go/models";
import { SIN_CUE_CLOSE, SIN_CUE_OPEN } from "../../../pages/sin/storyText";

type SinMethods = Pick<
  AppBindings,
  | "SinTopicsList" | "SinTopicCreate" | "SinTopicRename" | "SinTopicDelete"
  | "SinTopicClear" | "SinMessages" | "SinStream" | "SinIllustrate" | "SinExportMarkdown"
  | "SinCastGet" | "SinCastSet" | "SinCancel"
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

export function buildSin(): SinMethods {
  let seq = 0;
  const stories: MockStory[] = [];
  const messages = new Map<string, MockSinMessage[]>();
  // 角色选择内存态（故事 id → 角色库 id）；角色库本身由 mock/weixin.ts 提供。
  const casts = new Map<string, string[]>();
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
    },
    async SinTopicClear(id: string) {
      messages.set(id, []);
    },
    async SinMessages(topicID: string) {
      seed();
      return (messages.get(topicID) ?? []) as unknown as chat.Message[];
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
