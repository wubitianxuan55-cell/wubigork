// memory.ts — MemoryBindings（AppBindings 分域接口之一，Go MemoryB 门面）：
// 办公记忆/知识库/记忆中枢/轻语（Whisper）/微信触点/角色库之外的记忆与画像域。
import type {
  KnowledgeEntry,
  KnowledgeHistoryView,
  KnowledgeImportPreview,
  KnowledgeSaveRequest,
  KnowledgeSummary,
  MemoryArchivedPage,
  MemoryLifecycleView,
  MemoryDuplicateView,
  MemoryGraphView,
  SemanticGraphView,
  MemoryHubOverview,
  MemorySuggestion,
  MemorySuggestionsView,
  MemoryView,
  ProfileFactView,
  SimilarView,
  SkillSuggestion,
  WeixinAssistantStatusRow,
  WeixinAssistantView,
  WeixinReminderConfigView,
  WeixinReminderView,
  WhisperAnchorReplayView,
  WhisperAnchorView,
  WhisperEpisodeReplayView,
  WhisperEpisodeView,
  WhisperMemoryView,
  WhisperProactiveConfigView,
  WhisperProactiveResult,
  WhisperSubgraph,
} from "../types";
// chat 板块契约类型来自 wails 生成物（wails build 自动生成，勿手改生成物本身）。
import type { whisper } from "../../../../wailsjs/go/models";

export interface MemoryBindings {
  // Memory panel: read the loaded REASONIX.md hierarchy + saved auto-memories,
  // quick-add a note to a scope's REASONIX.md (≡ "#<note>"), and overwrite a doc
  // from the in-place editor.
  Memory(): Promise<MemoryView>;
  Remember(scope: string, note: string): Promise<string>;
  Forget(name: string): Promise<void>;
  SaveDoc(path: string, body: string): Promise<string>;
  UpdateFact(name: string, body: string): Promise<string>;
  ChangeFactType(name: string, type: string): Promise<string>;
  // SetMemoryEnabled 记忆开关（记忆可控性）：关闭后不再注入画像/规则/事实，
  // 持久化并重建办公引擎立即生效。
  SetMemoryEnabled(enabled: boolean): Promise<void>;
  // MemoryPin 固化/解除固化一条办公记忆（5.3 三态生命周期）。
  MemoryPin(name: string, pinned: boolean): Promise<void>;
  // MemoryLifecycle 三态生命周期总览（固化/衰减/归档 + 衰减评分）。
  MemoryLifecycle(): Promise<MemoryLifecycleView>;
  // MorningPreload 读/写晨报预载开关（~/.gaea_config.json，默认开）：work
  // 空间新会话装配时预装配高频工作记忆；写后重建引擎即时生效。
  MorningPreload(): Promise<boolean>;
  SetMorningPreload(enabled: boolean): Promise<void>;
  MemorySuggestions(): Promise<MemorySuggestionsView>;
  // Knowledge base panel.
  KnowledgeList(): Promise<KnowledgeSummary[]>;
  // KnowledgeSearch 全文检索（标题/分类/标签/正文），空 query 等价于 List。
  KnowledgeSearch(query: string, category: string, phase: string, status: string): Promise<KnowledgeSummary[]>;
  // ── 记忆中枢 ──
  MemoryHubOverview(): Promise<MemoryHubOverview>;
  ProfileList(): Promise<ProfileFactView[]>;
  ProfileSave(f: ProfileFactView): Promise<void>;
  ProfileDelete(name: string): Promise<void>;
  ProfileConflicts(): Promise<string[]>;
  WhisperMemories(): Promise<WhisperMemoryView[]>;
  // ── v4.3 会客厅：关系图谱 + 主动关心 ──
  // WhisperGraphSubgraph 返回指定人格关系图谱中以 entity 为中心、hops 跳内子图。
  WhisperGraphSubgraph(personalityId: string, entity: string, hops: number): Promise<WhisperSubgraph>;
  // WhisperProactiveNow 手动触发主动关心评估（「轻语先开口」按钮/定时器共用）。
  WhisperProactiveNow(personalityId: string): Promise<WhisperProactiveResult>;
  // WhisperProactiveConfig 返回主动关心定时推送配置（频控上限/间隔/时窗/开关）。
  WhisperProactiveConfig(): Promise<WhisperProactiveConfigView>;
  // WhisperProactiveSetConfig 部分更新主动关心定时推送配置（JSON 字符串，校验失败报错）。
  WhisperProactiveSetConfig(cfgJSON: string): Promise<void>;
  // ── v4.4 微信触点（书房·离线代办；WhisperWeixin* 自 LegacySurface 转正）──
  // WhisperWeixinGetQR 获取微信扫码登录二维码（dataURL 图片 + 会话 token）。
  WhisperWeixinGetQR(): Promise<{ qrcode: string; imageUrl: string }>;
  // WhisperWeixinQRStatus 轮询扫码状态（waiting/scanned/confirmed；confirmed 携带 botToken/botId）。
  WhisperWeixinQRStatus(qrcode: string): Promise<Record<string, unknown>>;
  // WhisperWeixinQRStatusWithCode 带手机配对码轮询（need_verifycode 状态时使用）。
  WhisperWeixinQRStatusWithCode(qrcode: string, verifyCode: string): Promise<Record<string, unknown>>;
  // WhisperWeixinStatus 全部助手的微信通道状态（运行/过期/未配置）。
  WhisperWeixinStatus(): Promise<WeixinAssistantStatusRow[]>;
  // v4.48 青鸟人格选择器：WeixinPage 经 bridge 消费（原 legacy wailsjsCompat
  // 直调转正，浏览器 dev mock 可达）；从 LegacySurfaceNames 摘除。
  // WhisperGetPersonalities 轻语预设人格清单。
  WhisperGetPersonalities(): Promise<whisper.PersonalityPreset[]>;
  // WhisperAssistantList 全部虚拟助手（微信绑定/人格/启停）。
  WhisperAssistantList(): Promise<WeixinAssistantView[]>;
  // WhisperAssistantSave 新建/更新助手（含 WxToken/WxBotID 微信绑定，保存后自动重拉通道）。
  WhisperAssistantSave(ast: Partial<WeixinAssistantView>): Promise<void>;
  // WhisperAssistantDelete 删除助手并停其微信通道。
  WhisperAssistantDelete(id: string): Promise<void>;
  // WeixinReminderList 全量微信提醒（待触发/已完成/失败，触发时间升序）。
  WeixinReminderList(): Promise<WeixinReminderView[]>;
  // WeixinReminderAdd 前端手动建提醒（fireAtRFC3339 为 RFC3339 时间串，必须在未来）。
  WeixinReminderAdd(text: string, fireAtRFC3339: string): Promise<{ id: string; fireAt: string; status: string }>;
  // WeixinReminderDelete 删除提醒（任意状态可删）。
  WeixinReminderDelete(id: string): Promise<void>;
  // WeixinReminderConfig 微信任务化配置（当前仅提醒开关）。
  WeixinReminderConfig(): Promise<WeixinReminderConfigView>;
  // WeixinReminderSetConfig 部分更新微信任务化配置（JSON 字符串）。
  WeixinReminderSetConfig(cfgJSON: string): Promise<void>;
  // WhisperEpisodes 聊天情节记忆（hermes.db，时间倒序）。
  WhisperEpisodes(): Promise<WhisperEpisodeView[]>;
  // WhisperEpisodeReplay 情节记忆回放（hermes.db，只读）：按情节 ID 重建原始对话。
  WhisperEpisodeReplay(episodeId: string): Promise<WhisperEpisodeReplayView>;
  // WhisperAnchors 轻语时间锚点列表（hermes.db，play 空间纪念日）。
  WhisperAnchors(): Promise<WhisperAnchorView[]>;
  // WhisperAnchorReplay 按时间锚点回放「重访那一天」：锚点 → 关联情节 → 原始对话。
  WhisperAnchorReplay(anchorId: string): Promise<WhisperAnchorReplayView>;
  // WhisperMemoryRetell 让 gaea 以当前人格口吻把一段记忆重述成故事（LLM 叙事）。
  WhisperMemoryRetell(kind: "episode" | "anchor", id: string, personalityId: string): Promise<string>;
  // WhisperCausalExplain 跨事实因果推断：基于图谱「导致」边 + event_chain 关联
  // 解释「为什么<entity>」，无证据时返回诚实回退文案。
  WhisperCausalExplain(entity: string, personalityId: string): Promise<string>;
  // WhisperExportArchive 导出聊天记忆归档（hermes.db → Markdown 分目录），返回文件数。
  WhisperExportArchive(dir: string): Promise<number>;
  // PickDirectory 系统目录选择对话框，返回所选目录（取消返回空串）。
  PickDirectory(): Promise<string>;
  MemoryGraph(): Promise<MemoryGraphView>;
  // SemanticGraph 记忆语义图谱：事件日志投影（entity/event/source 三向边），
  // 与 MemoryGraph（标签/分类关联图）同一渲染面、不同事实源。
  SemanticGraph(): Promise<SemanticGraphView>;
  // ── 做梦 2.0 晨报（纯本地主动预取）──
  // MemoryMorningBrief 返回「今日晨报」JSON 串（前端 JSON.parse 后渲染）：
  // work 空间记忆 top5 + 常驻规则 + 近 24h dream 沉淀计数。零 LLM、只读。
  MemoryMorningBrief(): Promise<string>;
  // ── 办公记忆查重/合并 ──
  MemoryDuplicates(min: number): Promise<MemoryDuplicateView[]>;
  MemoryMerge(targetName: string, sourceNames: string[]): Promise<string>;
  // ── 办公记忆归档生命周期（T6-8.2 / 记忆统一层）──
  // MemoryArchivedList 分页列出归档（含超过 90 天的硬删除候选），
  // 响应含 retentionDays（归档保留期天数，前端展示「归档保留 N 天」）；
  // MemoryCleanupArchived 硬删除归档超过保留期的事实，返回删除条数（无超期返回 0）；
  // MemoryUnarchive 恢复一条已归档记忆回活跃列表（误归档可在保留期内一键恢复）；
  // MemoryUnarchiveBatch 批量恢复（逐条恢复、失败跳过并聚合错误，返回成功数）；
  // MemorySetRetentionDays 设置归档保留期（天，钳制 [1,730]，持久化生效）。
  MemoryArchivedList(limit: number, offset: number): Promise<MemoryArchivedPage>;
  MemoryCleanupArchived(): Promise<number>;
  MemoryUnarchive(name: string): Promise<void>;
  MemoryUnarchiveBatch(names: string[]): Promise<number>;
  MemorySetRetentionDays(days: number): Promise<void>;
  // Herdsman 深挖：数字生命记忆总览（只读）与最近异步操作。
  HerdsmanDigitalLife(): Promise<unknown>;
  HerdsmanOperations(): Promise<unknown>;
  // 画像冲突裁决
  ProfileResolveConflict(name: string, prefer: string): Promise<void>;
  // ── 知识库导入 ──
  KnowledgeImportPreview(path: string): Promise<KnowledgeImportPreview>;
  KnowledgeImportAIParse(path: string): Promise<KnowledgeImportPreview>;
  KnowledgeImportApply(rows: KnowledgeEntry[]): Promise<number>;
  // KnowledgeHistory 返回某条目的版本历史（新→旧）。
  KnowledgeHistory(name: string): Promise<KnowledgeHistoryView[]>;
  // KnowledgeFindSimilar 查重：标题模糊相似的既有条目。
  KnowledgeFindSimilar(title: string): Promise<SimilarView[]>;
  // KnowledgeExport 批量导出知识库为 Markdown，返回条数。
  KnowledgeExport(dir: string): Promise<number>;
  // KnowledgeReview 审核流：approve=true 置为现行并记录审核人，false 驳回。
  KnowledgeReview(name: string, approve: boolean, reviewer: string): Promise<void>;
  // KnowledgeMerge 把 sourceNames 合并进 targetName（标签并集/来源合并），返回目标名。
  KnowledgeMerge(targetName: string, sourceNames: string[]): Promise<string>;
  KnowledgeGet(name: string): Promise<KnowledgeEntry | null>;
  KnowledgeSave(entry: KnowledgeSaveRequest): Promise<void>;
  KnowledgeDelete(name: string): Promise<void>;
  AcceptMemorySuggestion(candidate: MemorySuggestion): Promise<string>;
  AcceptMergeSuggestion(keep: string, archive: string): Promise<string>;
  AcceptSkillSuggestion(candidate: SkillSuggestion): Promise<string>;
}
