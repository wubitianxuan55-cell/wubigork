// sin.ts — SinBindings（AppBindings 分域接口之一，Go SinB 门面）：
// 原罪（闲庭 · 图文故事创作）板块契约。
//
// 会话复用统一聊天存储（话题 mode=sin，与聊天板块同表不同域）：
// SinTopicsList/SinMessages 只读写 sin 话题；SinStream 走功能级路由 sin
// （模型中心「原罪」绑定，未绑定回退全局），事件流 sin-stream:<runID>
// （delta/reasoning/done/error）与聊天板块 chat-stream 同形。
// 插图：SinIllustrate 复用绘梦图像后端生成，产物落 imagehub 台账
// （space=play / source_board=sin），并把路径回写到该轮消息 extra.illustrations。
import type { chat } from "../../../../wailsjs/go/models";

export interface SinBindings {
  /** 故事列表（仅 mode=sin；聊天板块话题不混入）。 */
  SinTopicsList(): Promise<chat.Topic[]>;
  /** 新建故事（title 空 = 「新故事」）。 */
  SinTopicCreate(title: string): Promise<{ id: string; title: string; mode: string; [k: string]: unknown }>;
  SinTopicRename(id: string, title: string): Promise<void>;
  SinTopicDelete(id: string): Promise<void>;
  SinTopicClear(id: string): Promise<void>;
  /** 故事全部消息（按 seq 升序）。 */
  SinMessages(topicID: string): Promise<chat.Message[]>;
  /** 故事续写流式入口：返回 runID，前端订阅 sin-stream:<runID>。 */
  SinStream(topicID: string, message: string): Promise<string>;
  /**
   * 故事角色（角色库 id 列表，按选择顺序）。
   * 硬隔离说明：选择结果存 <用户配置目录>/gaea/sin/cast.json（原罪自有存储，
   * 不写办公工作区）；角色库本身是跨板块共享资产层，不是办公数据面。
   */
  SinCastGet(topicID: string): Promise<string[]>;
  /** 保存故事角色选择，返回生效清单（悬空 id 丢弃、去重保序、单故事封顶）。 */
  SinCastSet(topicID: string, ids: string[]): Promise<string[]>;
  /**
   * 取消某故事当前在途的生成（Composer「停止」）：
   * 后端中止底层流式请求，已生成的部分照常落库（extra.cancelled=true）。
   * 无在途流时 reject（前端如实提示「没有正在生成的故事」）。
   */
  SinCancel(topicID: string): Promise<void>;
  /**
   * 为某处插图标记生成画面并回写消息 extra：
   * messageID <= 0 时只生成不落库；cue = 该条消息内插图出现次序（0 起，与
   * Go 侧 sinCueKey / 前端 sinCueKey 同规则）；返回 { path, cue, message_id, ... }。
   */
  SinIllustrate(topicID: string, messageID: number, cue: string, prompt: string, size: string): Promise<Record<string, unknown>>;
  /** 导出图文 Markdown（插图标记 → Markdown 图片；未生成的保留占位）。 */
  SinExportMarkdown(topicID: string): Promise<string>;
  /**
   * 读取某故事的便签（设定集）与大纲（右栏面板只读展示）：
   * 写作侧由 sin_notes / sin_outline 工具落 <用户配置目录>/gaea/sin/notes/，
   * 这里只读同源文件；缺失/损坏 = 空清单 + 空大纲（辅助数据不阻断）。
   */
  SinNotesGet(topicID: string): Promise<{ notes: string[]; outline: string }>;
}
