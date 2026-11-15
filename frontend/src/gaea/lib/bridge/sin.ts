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
import type { ContextNodeDetailView, ContextTimeline, GaeaResyncResult, HistoryMessage, QuestionAnswer, SessionMeta, Trajectory } from "../types";
import type { NovelBookSourceSearchResult, NovelBookSourceTocPreview } from "./novel";

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
  /**
   * 按窗口读取故事消息（v4.428，千回合老故事不再全量拉取）：beforeSeq<=0 取
   * 最新一页，否则取 seq < beforeSeq 的更早一页；返回升序切片 + has_more。
   */
  SinMessagesPage(topicID: string, beforeSeq: number, limit: number): Promise<{ messages: chat.Message[]; has_more: boolean }>;
  /**
   * 故事轨迹时间线（v4.412）：把落库消息 + extra 工具轨迹折叠成与办公
   * Trajectory 同构的轮次视图，前端 TrajectoryView 原组件复用。
   */
  SinTrajectory(topicID: string): Promise<Trajectory>;
  /**
   * 故事上下文构成快照（v4.412，估算口径）：无 usage 上报 → 请求全部
   * Estimated 标注、窗口未知（window=0，前端水位显示「—」）；前端
   * ContextView 原组件复用。
   */
  SinContextView(topicID: string): Promise<ContextTimeline>;
  /** 上下文浏览器节点「完整调用」详情（user/assistant 全文、tool 参数+输出）。 */
  SinContextNodeDetail(topicID: string, seq: number): Promise<ContextNodeDetailView>;
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
   * 统一插图画风（N10 图文穿插一致性锚）：该故事每张插图（正文标记与
   * sin_illustrate 工具同路）都追加同一条画风层；空串=清空回无锚。
   * 存储同 cast.json（styles 键），未设 = 逐字节旧行为。
   */
  SinStoryStyleGet(topicID: string): Promise<string>;
  SinStoryStyleSet(topicID: string, style: string): Promise<string>;
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
  /** 导出 EPUB 电子书（插图内嵌；每条助手回合一节，落 sin/exports，同名不覆盖）。返回文件路径。 */
  SinExportEpub(topicID: string): Promise<string>;
  /**
   * 读取某故事的便签（设定集）与大纲（右栏面板只读展示）：
   * 写作侧由 sin_notes / sin_outline 工具落 <用户配置目录>/gaea/sin/notes/，
   * 这里只读同源文件；缺失/损坏 = 空清单 + 空大纲（辅助数据不阻断）。
   */
  SinNotesGet(topicID: string): Promise<{ notes: string[]; outline: string }>;
  /**
   * 面板内编辑保存（v4.266，大纲+设定集整包写回）：baseline 是开始编辑时的
   * doc 快照 JSON（{"notes":[…],"outline":"…"}），锁内与当前文件比对，不符
   * 拒绝（错误带「底稿冲突：」前缀）；确认后 force=true 覆盖。
   */
  SinNotesSave(topicID: string, baseline: string, outline: string, notes: string, force: boolean): Promise<{ notes: string[]; outline: string }>;
  // ── sin 书源 t2（规格 docs/gaea-sin-booksource-distill-2026-09.md §5；产物落 sin 数据面）──
  /** 书源聚合搜索 + 泛搜索（与小说侧同源同表；HasRule=可下载，免规则候选仅参考）。 */
  SinBookSourceSearch(keyword: string): Promise<NovelBookSourceSearchResult>;
  /** 目录预览（总数 + 首8末4样例 + truncated 注记），供范围选择。 */
  SinBookSourceToc(source: string, detailURL: string): Promise<NovelBookSourceTocPreview>;
  /**
   * 下载整本成书 TXT → <用户配置目录>/gaea/sin/books/（硬隔离：sin 数据面，
   * 不写工作区/书架）；进度/终态订阅 sin-booksource:<jobId>（progress/done/error，
   * done 附 Failed 清单）。免规则来源同步拒绝（正文必须规则）。
   */
  SinBookSourceDownload(source: string, detailURL: string, start: number, end: number, title: string): Promise<SinBookSourceDownloadStart>;
  /** 取消在途下载（与小说侧共用 job 登记簿；未知 job 返回 false）。 */
  SinBookSourceDownloadCancel(jobId: string): Promise<boolean>;
  /** 成书清单（.txt，按修改时间新→旧；未下载过 = 空清单不报错）。 */
  SinBookSourceBooksList(): Promise<SinBookSourceBook[]>;
  /** 删除成书（fail-closed 路径护栏：限成书目录内 .txt，越界拒绝；同名 .epub 连带清理）。 */
  SinBookSourceBookDelete(path: string): Promise<void>;
  /** 导出 EPUB（t5：同名 .epub 落同目录，重复导出覆盖；返回产物路径）。 */
  SinBookSourceBookExportEpub(path: string): Promise<string>;

  // ── 2026-11-15 升格刀：原罪独立引擎（第二控制器，与办公 Gaea* 平行）──
  // 事件走 gaea-event-sin 专用通道（独立 seq+断号补拉），帧形态与 gaea-event
  // 同构（text/reasoning/tool_dispatch/tool_result/notice/phase/turn_done/…）；
  // 会话真源 = 书目录 .gaea/sessions/play/*.jsonl（引擎 session log）。
  /** 打开/切换书工作区（幂等；书架 containment 校验 fail-closed）。 */
  SinEngineOpenBook(bookPath: string): Promise<void>;
  /** 提交一轮创作（输入富化=角色块+底稿块按书 slug 注入，历史由会话自带）。 */
  SinEngineSend(input: string): Promise<void>;
  /** 运行中插话调整（不打断工具执行）。 */
  SinEngineSteer(input: string): Promise<void>;
  /** 取消当前回合。 */
  SinEngineCancel(): Promise<void>;
  /** 引擎是否在运行。 */
  SinEngineRunning(): Promise<boolean>;
  /** 开新会话（保留书级记忆；运行闸同款）。 */
  SinEngineNewSession(): Promise<void>;
  /** 引擎状态视图（壳启动/书架联动消费）。 */
  SinEngineState(): Promise<SinEngineStateView>;
  /** 审批决策回传（决策语义与办公 ApprovalModal 同族）。 */
  SinEngineApprove(id: string, decision: string): Promise<void>;
  /** ask 结构化提问答复。 */
  SinEngineAnswerQuestion(id: string, answers: QuestionAnswer[]): Promise<void>;
  /** 断号补拉（独立 wire seq；返回会话全量 items）。 */
  SinEngineResyncEvents(afterSeq: number): Promise<GaeaResyncResult>;
  /** 书工作区会话清单（新→旧）。 */
  SinEngineListSessions(): Promise<SessionMeta[]>;
  /** 切换到指定会话继续（返回其消息投影）。 */
  SinEngineResumeSession(path: string): Promise<HistoryMessage[]>;

  // ── 书架工作空间绑定（一书一故事，书下多会话）──
  /** 读取某会话的书绑定（未绑 = bound:false）。 */
  SinWorkspaceGet(topicID: string): Promise<SinWorkspaceInfo>;
  /** 绑定会话到书（校验 project.json+书架 containment；绑定即迁移存量键）。 */
  SinWorkspaceSet(topicID: string, projectPath: string): Promise<SinWorkspaceInfo>;
  /** 解除绑定（书键数据保留）。 */
  SinWorkspaceUnbind(topicID: string): Promise<void>;
  /** 全量绑定清单（书架栏分组用）。 */
  SinWorkspaceBindings(): Promise<SinWorkspaceBindingView[]>;
}

/** 原罪引擎状态视图（Go SinEngineStateView 投影）。 */
export interface SinEngineStateView {
  ready: boolean;
  bookPath: string;
  bookSlug: string;
  running: boolean;
  label: string;
}

/** 书绑定视图（Go SinWorkspaceInfo 投影）。 */
export interface SinWorkspaceInfo {
  bound: boolean;
  path: string;
  title: string;
}

/** 绑定清单条目（Go SinWorkspaceBindingView 投影）。 */
export interface SinWorkspaceBindingView {
  topicId: string;
  path: string;
  title: string;
}

/** 下载起跑回执：进度/终态订阅 sin-booksource:<jobId>。 */
export interface SinBookSourceDownloadStart {
  jobId: string;
}

/** 成书清单条目。 */
export interface SinBookSourceBook {
  title: string;
  path: string;
  sizeBytes: number;
  /** RFC3339。 */
  modifiedAt: string;
}

