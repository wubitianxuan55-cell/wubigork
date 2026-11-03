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
    `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 240 320">
      <rect width="240" height="320" fill="hsl(${hue},32%,16%)"/>
      <circle cx="120" cy="72" r="70" fill="hsl(${hue},42%,26%)" opacity=".7"/>
      <path d="M38 320c5-76 37-116 82-116s77 40 82 116z" fill="hsl(${hue},30%,24%)"/>
      <path d="M57 196l63 48 63-48 22 124H35z" fill="hsl(${hue},38%,36%)"/>
      <path d="M72 130h96v70H72z" fill="hsl(${hue},45%,72%)"/>
      <ellipse cx="120" cy="103" rx="54" ry="68" fill="hsl(${hue},46%,76%)"/>
      <path d="M62 108c-5-57 23-84 60-84 42 0 67 31 57 92-9-23-27-39-51-43-24 20-44 31-66 35z" fill="hsl(${hue},28%,20%)"/>
      <circle cx="99" cy="105" r="3.5" fill="hsl(${hue},35%,24%)"/>
      <circle cx="141" cy="105" r="3.5" fill="hsl(${hue},35%,24%)"/>
      <path d="M108 128c8 6 16 6 24 0" fill="none" stroke="hsl(${hue},38%,38%)" stroke-width="2.5" stroke-linecap="round"/>
      <path d="M120 175l-31 38h62z" fill="hsl(${hue},46%,80%)" opacity=".28"/>
      <text x="120" y="299" font-size="20" text-anchor="middle" fill="hsl(${hue},52%,86%)" font-family="serif">${initial}</text>
    </svg>`,
  )}`;
const characters = [
  {
    id: "gaea", name: "盖亚", kind: "builtin", gender: "female", age: "28",
    tags: ["秘书", "工作"], portraitUrl: charPortrait("盖", 168),
    roleType: "supporting", personality: "沉稳可靠，温和坦诚，结论先行",
    background: "轻语内置工作人格，专注桌面协作与任务执行。",
    voiceGuide: "专业严谨的办公秘书口吻，条理清晰", chatEnabled: true,
    dims: { T: 85, I: 55, S: 20, O: 80, R: 50 },
  },
  {
    id: "muse", name: "缪斯", kind: "builtin", gender: "female",
    tags: ["创作"], portraitUrl: charPortrait("缪", 286),
    personality: "灵感充沛的写作伙伴，善比喻",
    background: "轻语内置创作人格，擅长把模糊想法变成清晰文本。",
    voiceGuide: "灵感充沛，善用比喻", chatEnabled: true,
    dims: { T: 60, I: 82, S: 45, O: 88, R: 40 },
  },
  {
    id: "fixer", name: "修哥", kind: "builtin", gender: "male",
    tags: ["执行"], portraitUrl: charPortrait("修", 32),
    personality: "干脆利落，直给结论",
    background: "轻语内置执行人格，偏好短链路解决问题。",
    voiceGuide: "干脆利落的执行者，直给结论", chatEnabled: true,
    dims: { T: 72, I: 30, S: 40, O: 55, R: 75 },
  },
  {
    id: "c_lin", name: "林晚", kind: "custom", gender: "female", tags: ["女主角", "清冷"],
    portraitUrl: charPortrait("林", 265), roleType: "女主", personality: "清冷聪慧，外冷内热",
    background: "前朝遗孤，隐姓埋名于市井，身负血仇", chatEnabled: true,
    // dims 与真实后端对齐：Go 侧 Dims 为值类型必出（whisper.PersonalityDims）
    dims: { T: 35, I: 78, S: 82, O: 64, R: 41 },
  },
  {
    id: "c_gucheng", name: "顾城", kind: "custom", gender: "male", tags: ["男主角", "将领"],
    portraitUrl: charPortrait("顾", 200), roleType: "男主", personality: "沉稳寡言，护短",
    background: "边关少将，战功赫赫却厌战", chatEnabled: true,
    dims: { T: 28, I: 55, S: 40, O: 47, R: 80 },
  },
  {
    // T1 画室「创作资产」角色槽扩样例：无立绘可聊天角色（诚实占位路径）。
    id: "c_baimo", name: "白墨", kind: "custom", gender: "male", tags: ["书生", "谋士"],
    portraitUrl: "", roleType: "配角", personality: "温吞细致，擅推演",
    background: "落第书生，以抄书为业，胸有丘壑", chatEnabled: true,
    dims: { T: 52, I: 60, S: 58, O: 72, R: 76 },
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
