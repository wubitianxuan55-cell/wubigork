// ── v4.4 微信触点（书房·离线代办）──────────────────────────────
// WeixinAssistantView 虚拟助手（assistant.Assistant 的前端投影，含微信绑定字段）。
export interface WeixinAssistantView {
  id: string;
  name: string;
  personalityId: string;
  wxToken?: string;
  wxBotId?: string;
  wxUserId?: string;
  enabled: boolean;
  portraitUrl?: string;
}
// WeixinAssistantStatusRow 单个微信助手的通道状态（WhisperWeixinStatus 行）。
export interface WeixinAssistantStatusRow {
  id: string;
  name: string;
  personalityId: string;
  enabled: boolean;
  hasToken: boolean;
  wxRunning: boolean;
  wxSessionExpired?: boolean;
}
// WeixinReminderView 微信「离线代办」提醒条目（WeixinReminderList 行）。
export interface WeixinReminderView {
  id: string;
  text: string; // 提醒事项（已剥离触发词/时间表达）
  fireAt: string; // RFC3339 触发时间
  assistantId: string;
  source: 'weixin' | 'manual' | string;
  status: 'pending' | 'done' | 'failed' | string;
  failCount: number;
  createdAt: string;
  sentAt?: string | null;
}
// WeixinReminderConfigView 微信任务化配置（当前仅提醒开关）。
export interface WeixinReminderConfigView {
  remindersEnabled: boolean;
}
