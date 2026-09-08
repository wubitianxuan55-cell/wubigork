// chat.ts — ChatBindings（AppBindings 分域接口之一，Go ChatB 门面）：
// 对话板块契约（T6-3）：Topics/Messages/语音消息持久化。
import type { app as AppModels, chat } from "../../../../wailsjs/go/models";

export interface ChatBindings {
  // ── 对话 chat（T6-3 契约同步）────────────────────────────
  // ChatTopicsList/ChatMessagesList Go 侧签名变为 ([]T, error)（T6-3.2 读错
  // 返回 error），Wails 绑定后失败为 rejected promise；这里以 [T[], unknown]
  // 元组形态标注「成功数据 + 失败错误」，调用点必须 try/catch，不能再 || [] 吞错。
  ChatTopicsList(): Promise<[chat.Topic[], unknown]>;
  ChatMessagesList(topicID: string): Promise<[chat.Message[], unknown]>;
  // ChatAppendMessages 语音消息持久化（T6-3.3）：单事务批量追加，
  // role 仅接受 user/assistant（其余后端跳过）。
  ChatAppendMessages(topicID: string, messages: AppModels.ChatMessageInput[]): Promise<void>;
}
