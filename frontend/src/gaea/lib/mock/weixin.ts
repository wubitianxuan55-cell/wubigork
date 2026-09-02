// mock/weixin.ts — 微信助手域（v4.47 拆分自浏览器 mock 缺口：WeixinPage 离线
// 可开发/可视觉走查）。内存态模拟：助手 CRUD + 扫码流相位推进 + 提醒 CRUD。
import type { AppBindings } from "../bridge";
import type {
  WeixinAssistantStatusRow,
  WeixinAssistantView,
  WeixinReminderView,
} from "../types";
import { whisper } from "../../../../wailsjs/go/models";

type WeixinMethods = Pick<
  AppBindings,
  | "WhisperWeixinGetQR" | "WhisperWeixinQRStatus" | "WhisperWeixinQRStatusWithCode"
  | "WhisperWeixinStatus" | "WhisperAssistantList" | "WhisperAssistantSave" | "WhisperAssistantDelete"
  | "WeixinReminderList" | "WeixinReminderAdd" | "WeixinReminderDelete"
  | "WeixinReminderConfig" | "WeixinReminderSetConfig"
  // v4.48 人格选择器：新增助手弹窗拉预设 + 角色库可聊天角色
  | "WhisperGetPersonalities" | "CharacterList"
>;

// ── 内存态（浏览器开发环境，无持久化；模块级 = 同会话多次打开页面状态延续）──
const assistants: WeixinAssistantView[] = [
  { id: "gaea", name: "gaea", personalityId: "gaea", enabled: true, wxToken: "mock-tok-gaea", wxBotId: "mock-bot-gaea", wxUserId: "mock-uid-gaea" },
  { id: "wx_muse", name: "小雨", personalityId: "muse", enabled: true, portraitUrl: "" },
];
// 会话过期演示：gaea 通道运行中；小雨未绑定（扫码流可走通）；阿修已停止。
const extraRows: WeixinAssistantStatusRow[] = [
  { id: "wx_muse", name: "小雨", personalityId: "muse", enabled: true, hasToken: false, wxRunning: false },
  { id: "wx_fix", name: "阿修", personalityId: "fixer", enabled: false, hasToken: true, wxRunning: false },
];

const reminders: WeixinReminderView[] = [
  { id: "r1", text: "交周报", fireAt: "2026-09-03T09:00:00+08:00", assistantId: "gaea", source: "weixin", status: "pending", failCount: 0, createdAt: "2026-09-02T08:00:00+08:00" },
  { id: "r2", text: "下午茶歇", fireAt: "2026-09-02T15:30:00+08:00", assistantId: "gaea", source: "manual", status: "done", failCount: 0, createdAt: "2026-09-01T10:00:00+08:00", sentAt: "2026-09-02T15:30:01+08:00" },
];
let remindersEnabled = true;
let reminderSeq = 3;

// 角色库可聊天角色（人格选择器右栏详情数据源；立绘用确定性 SVG 占位）
const charPortrait = (initial: string, hue: number) =>
  `data:image/svg+xml;utf8,${encodeURIComponent(
    `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 60 60"><rect width="60" height="60" fill="hsl(${hue},45%,22%)"/><text x="30" y="40" font-size="26" text-anchor="middle" fill="hsl(${hue},70%,80%)" font-family="sans-serif">${initial}</text></svg>`,
  )}`;
const characters = [
  {
    id: "c_lin", name: "林晚", kind: "custom", gender: "female", tags: ["女主角", "清冷"],
    portraitUrl: charPortrait("林", 265), roleType: "女主", personality: "清冷聪慧，外冷内热",
    background: "前朝遗孤，隐姓埋名于市井，身负血仇", chatEnabled: true,
  },
  {
    id: "c_gucheng", name: "顾城", kind: "custom", gender: "male", tags: ["男主角", "将领"],
    portraitUrl: charPortrait("顾", 200), roleType: "男主", personality: "沉稳寡言，护短",
    background: "边关少将，战功赫赫却厌战", chatEnabled: true,
  },
];

// 扫码流模拟：每个 token 独立相位（轮询 2 次 → 已扫码；4 次 → confirmed）。
const qrPolls = new Map<string, number>();

/** 伪二维码（SVG dataURL）：定位角 + 确定性伪随机模块，仅作离线占位。 */
function fakeQrDataUrl(seed: string): string {
  let h = 0;
  for (const c of seed) h = (h * 31 + c.charCodeAt(0)) >>> 0;
  const n = 25;
  const cell = (i: number, j: number) => {
    const finder = (i < 7 && j < 7) || (i < 7 && j >= n - 7) || (i >= n - 7 && j < 7);
    if (finder) {
      const li = i < 7 ? i : i - (n - 7);
      const lj = j < 7 ? j : j - (n - 7);
      const ring = Math.max(Math.abs(li - 3), Math.abs(lj - 3));
      return ring !== 2;
    }
    h = (h * 1103515245 + 12345) >>> 0;
    return (h >>> 16) % 100 < 42;
  };
  let rects = "";
  for (let i = 0; i < n; i++) {
    for (let j = 0; j < n; j++) {
      if (cell(i, j)) rects += `<rect x="${j * 4}" y="${i * 4}" width="4" height="4"/>`;
    }
  }
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><rect width="100" height="100" fill="#fff"/><g fill="#000">${rects}</g></svg>`;
  return `data:image/svg+xml;utf8,${encodeURIComponent(svg)}`;
}

export function buildWeixin(): WeixinMethods {
  return {
    async WhisperWeixinGetQR() {
      const token = `qr-${Date.now().toString(36)}`;
      qrPolls.set(token, 0);
      return { qrcode: token, imageUrl: fakeQrDataUrl(token) };
    },

    async WhisperWeixinQRStatus(qrcode: string) {
      const polls = (qrPolls.get(qrcode) ?? 0) + 1;
      qrPolls.set(qrcode, polls);
      if (polls >= 4) return { status: "confirmed", botToken: `tok-${qrcode}`, botId: "mock-bot", userId: "mock-uid" };
      if (polls >= 2) return { status: "scanned" };
      return { status: "wait_scan" };
    },

    async WhisperWeixinQRStatusWithCode(qrcode: string, verifyCode: string) {
      if (!verifyCode) return { status: "need_verifycode" };
      return { status: "confirmed", botToken: `tok-${qrcode}`, botId: "mock-bot", userId: "mock-uid" };
    },

    async WhisperWeixinStatus() {
      return [
        {
          id: "gaea", name: "gaea", personalityId: "gaea", enabled: true,
          hasToken: Boolean(assistants.find((a) => a.id === "gaea")?.wxToken), wxRunning: true,
        },
        // Status 独有行（后端兜底自建未落库）原样保留——页面按 id merge 会兜底展示
        ...extraRows,
      ];
    },

    async WhisperAssistantList() {
      return assistants.map((a) => ({ ...a }));
    },

    async WhisperAssistantSave(ast: Partial<WeixinAssistantView>) {
      if (!ast.id) return;
      const idx = assistants.findIndex((a) => a.id === ast.id);
      if (idx >= 0) assistants[idx] = { ...assistants[idx], ...ast } as WeixinAssistantView;
      else assistants.push({ ...(ast as WeixinAssistantView) });
      // 保存绑定后通道重启：状态行同步 hasToken
      const row = extraRows.find((s) => s.id === ast.id);
      if (row) row.hasToken = Boolean(ast.wxToken) || row.hasToken;
    },

    async WhisperAssistantDelete(id: string) {
      const idx = assistants.findIndex((a) => a.id === id);
      if (idx >= 0) assistants.splice(idx, 1);
      const row = extraRows.findIndex((s) => s.id === id);
      if (row >= 0) extraRows.splice(row, 1);
    },

    async WeixinReminderList() {
      return reminders.map((r) => ({ ...r }));
    },

    async WeixinReminderAdd(text: string, fireAtRFC3339: string) {
      const row: WeixinReminderView = {
        id: `r${reminderSeq++}`, text, fireAt: fireAtRFC3339, assistantId: "gaea",
        source: "manual", status: "pending", failCount: 0, createdAt: new Date().toISOString(),
      };
      reminders.unshift(row);
      return { id: row.id, fireAt: row.fireAt, status: row.status };
    },

    async WeixinReminderDelete(id: string) {
      const idx = reminders.findIndex((r) => r.id === id);
      if (idx >= 0) reminders.splice(idx, 1);
    },

    async WeixinReminderConfig() {
      return { remindersEnabled };
    },

    async WeixinReminderSetConfig(cfgJSON: string) {
      try {
        const cfg = JSON.parse(cfgJSON) as { remindersEnabled?: boolean };
        if (typeof cfg.remindersEnabled === "boolean") remindersEnabled = cfg.remindersEnabled;
      } catch { /* 非法 JSON 忽略 */ }
    },

    async WhisperGetPersonalities() {
      // 生成类 PersonalityPreset 需要实例形态（dims/convertValues），这里用 createFrom 构造
      return [
        whisper.PersonalityPreset.createFrom({ id: "gaea", label: "盖亚", gender: "female", tags: ["秘书", "工作"], voiceGuide: "专业严谨的办公秘书口吻，条理清晰" }),
        whisper.PersonalityPreset.createFrom({ id: "muse", label: "缪斯", gender: "female", tags: ["创作"], voiceGuide: "灵感充沛的写作伙伴，善比喻" }),
        whisper.PersonalityPreset.createFrom({ id: "fixer", label: "修哥", gender: "male", tags: ["执行"], voiceGuide: "干脆利落的执行者，直给结论" }),
      ];
    },

    async CharacterList(_query: string, _kind: string, chatOnly: boolean, _page: number, _pageSize: number) {
      void _query; void _kind; void _page; void _pageSize;
      const items = chatOnly ? characters.filter((c) => c.chatEnabled) : characters;
      return { items: items.map((c) => ({ ...c })), total: items.length };
    },
  };
}
