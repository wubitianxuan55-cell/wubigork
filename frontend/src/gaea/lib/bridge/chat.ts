// chat.ts — ChatBindings（AppBindings 分域接口之一，Go ChatB 门面）：
// 对话板块契约（T6-3）：Topics/Messages/语音消息持久化。
import type { app as AppModels, chat } from "../../../../wailsjs/go/models";

export interface ChatBindings {
  // ── 对话 chat（T6-3 契约同步）────────────────────────────
  // ChatTopicsList/ChatMessagesList Go 侧签名为 ([]T, error)：Wails 绑定成功
  // 返回 T[]、失败为 rejected promise；历史曾有 [T[], unknown] 元组误标，
  // 已修正为数组（调用点用 try/catch 防失败，不能 || [] 吞错）。
  ChatTopicsList(): Promise<chat.Topic[]>;
  ChatMessagesList(topicID: string): Promise<chat.Message[]>;
  // ChatMessagesPage 游标分页（v4.370）：beforeSeq<=0 从最新向前取 limit 条；
  // 返回升序消息+hasMore（是否还有更早历史）+本批最早 seq（下次游标）。
  ChatMessagesPage(topicID: string, limit: number, beforeSeq: number): Promise<{ messages: chat.Message[]; hasMore: boolean; oldestSeq: number }>;
  // ChatAppendMessages 语音消息持久化（T6-3.3）：单事务批量追加，
  // role 仅接受 user/assistant（其余后端跳过）。
  ChatAppendMessages(topicID: string, messages: AppModels.ChatMessageInput[]): Promise<void>;
  // ── 批次二 wailsjsCompat 双轨退役转正（Go ChatB 门面，同名前缀）──
  // 对话板块话题管理 + 发送：ChatTopicCreate 新建话题（title/mode，mode 如
  // "chat"/"novel" 等）；Delete/Rename/SetMode/Clear 话题生命周期操作；
  // ChatImportTopic 批量导入消息建话题（返回 chat.Topic 形状 Record）；
  // ChatSend 发送消息（mode + 搜索/思考/强搜开关，返回结构化结果）；
  // ChatStreamPlain 无流式纯文本通道（返回最终文本）；
  // ChatTopicExportMarkdown 导出话题为 Markdown 文本。
  ChatTopicCreate(title: string, mode: string): Promise<{ id: string; title: string; mode: string; [k: string]: unknown }>;
  ChatTopicDelete(id: string): Promise<void>;
  ChatTopicRename(id: string, title: string): Promise<void>;
  ChatTopicSetMode(id: string, mode: string): Promise<void>;
  ChatImportTopic(title: string, mode: string, messages: Array<Record<string, unknown>>): Promise<Record<string, unknown>>;
  ChatSend(topicID: string, message: string, mode: string, searchEnabled: boolean, thinking: boolean, forceSearch: boolean): Promise<Record<string, unknown>>;
  ChatStreamPlain(topicID: string, message: string, searchEnabled: boolean, thinking: boolean, forceSearch: boolean): Promise<string>;
  ChatTopicClear(id: string): Promise<void>;
  ChatTopicExportMarkdown(topicID: string): Promise<string>;
}
