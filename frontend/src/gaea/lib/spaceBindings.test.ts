// spaceBindings.test.ts — S2.3 bridge 分面分类表（docs/gaea-space-shell-design.md §7）
// 批次记录：v4.323 提示词工坊 +5（PromptTemplateList/Get/Save/Reset/Preview，t6 首刀
// 模板可编辑覆盖层，CreatePage「提示词工坊」面板，play）→ 数量锁 523。前批：7.3-1 任务收件箱 +4（GaeaTaskInboxList/Save/SetStatus/Delete，双空间
// 首页挂点+四入口接线，shared——隔离由 space 参数承担）→ v4.319 技能调用计数 +1（SkillStats，work）→ 数量锁 518。
// 前批：sin 书源 t2 +6（SinBookSourceSearch/Toc/Download/DownloadCancel/BooksList/
// BookDelete）+ t5 +1：SinBookSourceBookExportEpub（EPUB 导出，play）+ ImportNovelBookEx 转正（sin 送小说导入消费，play）→ 数量锁 489。
// 前批：书源取书 +4（NovelBookSourceSearch/Toc/Import/ImportCancel，书源→拆书导入
// t1，NovelB 门面，play 数据面；t3 失败章补下 +1：NovelBookSourceImportChapters，v4.284）→ 481。前批：平台质量评审 +2（NovelReviewPlatforms/
// NovelChapterReview，v4.282 oh-story 蒸馏 T1，NovelB 门面，play 数据面）→ 472。
import { describe, expect, it } from "vitest";
import {
  GAEA_METHOD_FACETS,
  WORK_BINDING_NAMES,
  PLAY_BINDING_NAMES,
  SHARED_BINDING_NAMES,
  bindingSpaceOf,
  isBindingAllowedInSpace,
} from "./spaceBindings";

describe("spaceBindings 分类表（S2.3 bridge 分面）", () => {
  it("AppBindings 全部方法被显式分类（satisfies 编译期已兜底；此处锁数量）", () => {
    const total = Object.keys(GAEA_METHOD_FACETS).length;
    expect(total).toBe(540); // v4.404 CharacterScoreConsistency +1：一致性评分（文字锚点 v1，编辑器参考图评分钮，play） // v4.400 CharacterGenerateSheet +1：角色设定卡三视图生成（CharacterLibEditor 参考图区按钮，qedit 通道，play） // v4.397 ImageCutout +1：抠图/透明底导出（ResultStage「抠图」→CutoutModal 涂选导出，play）；前值 537=v4.388 原罪插图生图绑定 +2 // v4.388 原罪插图生图绑定 +2：GetSinImageConfig/SetSinImageConfig（模型中心功能绑定卡·原罪插图独立后端/模型，play）；前批 v4.386 造价库分页 +1：CostSearchPage（成本库列表/表格分页+EntryPicker top8，work）；前批 v4.377 建议制 +4：SetDreamMode/DismissMemorySuggestion/DreamPurgePreview/DreamPurge（记忆面板自动做梦开关+建议忽略+清污，work）； 与 keyof AppBindings 一致（satisfies 编译期钉死；v4.348 故事 EPUB 导出 +1：SinExportEpub（原罪顶栏导出下拉，插图内嵌，play）；v4.338 死链清理 -2：GaeaCheckpoints（UI 回退实走 GaeaRewind，唯一引用=bridge 声明+mock）/GaeaTCCAReport（state.tcca 零渲染消费），无参绑定会话审计收官，work）；v4.330 POV 视图 +1：NovelSceneBibleView（刀7续场景圣经消费面，ChapterEditor「视角」抽屉，play）；v4.329 画室消耗 +1：ImageHubMonthlyUsage（刀 E 月度聚合，ImageGenPage 画室消耗面板，play）；v4.326 全书体检 +1：RunBookHealthCheck（GenerationGate 闭环收口，CreatePage「全书体检」面板，play）；v4.325 分析接线 +2：AnalyzeChapter（出 Legacy 转正）+NovelChapterAnalysisV2（t7 分析 V2 面板，CreatePage「章节分析」，play）；v4.324 模板包导入导出 +2：PromptBundleExport/PromptBundleImport（t6-C2 content_hash 三态，工坊面板「导出/导入模板包」，play）；v4.323 提示词工坊 +5：PromptTemplateList/Get/Save/Reset/Preview（t6 首刀模板可编辑覆盖层，CreatePage「提示词工坊」面板，play）；7.3-1 任务收件箱 +4：GaeaTaskInboxList/Save/SetStatus/Delete（阶段七 7.3-1 多入口统一任务收件箱，双空间首页挂点+四入口接线，shared——隔离由 space 参数承担）；7.2-2 journal 历史蒸馏 +3：SkillDistillCandidates/SkillDistillDraft/SkillDistillDecide（阶段七 7.2-2 journal 审批链→重复模式→结晶技能，记忆面板建议 tab「流程蒸馏」分区，work）；7.1-2 路由学习 +4：GaeaRouteLedger/GaeaRouteSuggestions/GaeaRouteSuggestionApply/GaeaRouteSuggestionIgnore（阶段七 7.1-2 路由账本+建议制改绑，模型中心「成本归因」tab，shared）；v4.313 会话录制技能 +2：SkillDraftFromSession/SkillDraftSave（阶段七 7.2-1 演示式录制，办公 Composer 工具栏入口，work）；v4.310 职业绑定 +2：SetCharacterCareer/RemoveCharacterCareer（t5 §7.6 角色-职业绑定 UI，写 characters.json 与差分器同库，play）；v4.306 标注层 +1：NovelChapterAnnotations（t4-C4 keyword→正文偏移，play）；v4.305 建议读取 +1：NovelChapterSuggestions（t4-C3 重写建议驱动 UI 数据源，play）；v4.304 驱动式重写批次 +6：NovelChapterRewrite/ListRewriteVersions/GetRewriteVersion/ApplyRewriteVersion/DiscardRewriteVersion/RestoreRewriteVersion（t4-C3 whole 模式+版本库+恢复，play）；v4.299 同步结果批次 +1：GetLastForeshadowSync（t1-P4 SyncResult 上绑定面，跳过原因可见不静默 D3，play）；v4.298 伏笔清理统计批次 +4：CleanChapterAnalysisForeshadows/ClearProjectForeshadowsForReset/DeleteChapterForeshadows/GetForeshadowStats（t1-P3 生命周期清理入口+统计，spec §8.1/§5.1，play）；sin 书源 t2 +6：SinBookSourceSearch/Toc/Download/DownloadCancel/BooksList/BookDelete（原罪右栏书源卡数据面，产物落 sin 数据面，play）；书源取书 +4：NovelBookSourceSearch/NovelBookSourceToc/NovelBookSourceImport/NovelBookSourceImportCancel（书源→拆书导入 t1，书架「在线搜书」数据面，play；t3 补下 +1 NovelBookSourceImportChapters v4.284）；平台质量评审批次 +2：NovelReviewPlatforms/NovelChapterReview（v4.282 oh-story 蒸馏 T1，CreatePage「平台评审」面板，play 数据面）；拆书反推批次 +2：NovelOutlineReconstruct/NovelOutlineReconstructApply；伏笔一致性体检 +1：LintForeshadows；文风指纹批次 +3：NovelFingerprintStatus/NovelFingerprintBuild/NovelFingerprintScore（CreatePage 文风指纹面板，play 数据面）；v4.266 面板编辑保存 +1：SinNotesSave；v4.263 原罪右栏面板 +1：SinNotesGet；v4.258 原罪取消 +1：SinCancel；v4.257 原罪角色库 +2：SinCastGet/SinCastSet；v4.244 原罪板块 +9：SinTopicsList/SinTopicCreate/SinTopicRename/SinTopicDelete/SinTopicClear/SinMessages/SinStream/SinIllustrate/SinExportMarkdown（闲庭 play 数据面）；v4.28 + PptxOutline/GaeaBrowserObserve；v4.64 + SubagentFollowUp；v4.66 + PromoteSubagent；v4.78 + TaskKill；v4.80 + ContextNodeDetail；v4.86 + GaeaGit*7；v4.94 + SubagentContextView；v4.99 直调转正 + ImageHubAssets/ChapterArtList；v4.102 图像域直调族转正 +17；v4.105 +WarmComfyUI；v4.109 +PptxApplyEdit；v4.113 +ScheduleLoad/Save；v4.134 +ScheduleExportXlsx/ImportXlsx；v4.139 进度计划多工程 +ScheduleProjects/ProjectOpen/ProjectCreate/ProjectArchive/ProjectDelete；v4.140 +ScheduleImportMpp；v4.145 +ScheduleProjectCopy；v4.156 pptx 真编辑刀2 +PptxSlideText；v4.158 组价复核闭环 +CostComposeRecords；v4.162 进度计划导入导出壳内修复 +ReadFileB64/SaveFileAs；v4.163 长期日志机制 +OpenLogsDir；v4.171 wailsjsCompat 双轨退役批次一 +16：GetAppInfo/GetBoardManifests/GetFeatureModel/GetFeatureModelEnabled/SetFeatureModel/SetFeatureModelEnabled/GetActiveModel/StartLocalTTSService/VoiceApplySettings/WhisperGetState/WhisperGetFacts/WhisperGetTraces/WhisperDeleteFact/WhisperUpdateFact/NovelSearch/SaveSettings；批次二 +29：GetConfig/SaveConfig/GetStats/ListSkills/ExportAll/ChatTopicCreate/ChatTopicDelete/ChatTopicRename/ChatTopicSetMode/ChatImportTopic/ChatSend/ChatStreamPlain/ChatTopicClear/ChatTopicExportMarkdown/GetWorldview/SaveWorldview/ChatWorldview/GetWorldviewSections/SaveAllWorldviewSections/CheckConsistency/CheckConsistencyDeep/GetForeshadows/SaveForeshadows/SaveCharactersBatch/NovelReadingAsk/GenerateSceneIllustration/VoiceGetSettings/VoiceChatText/SetImageBackend；批次三a +18：VoiceStart/VoiceStop/VoicePlaybackDone/VoiceCancelTTS/VoicePushAudio/VoiceSetPTTActive/WhisperClearSession/TTSSpeakBase64/TTSSpeakBase64WithParams/GetTTSSpeakers/GetVoicePipelineConfig/DataBackupInfo/DataBackupCreate/DataBackupRestore/DataBackupCancel/DataBackupRollback/DataBackupRestoreResult/ListProjects；批次三b +38：Charlib 16（CharacterSave/Get/Delete/ImportProject/ListByProject/Associate/AssociateTo/SetProjectState/Dissociate/SyncProject/DrawRandom/GenerateFill/GenerateRandom/FillAll/GeneratePortrait/GeneratePortraitWithRef）+ Novel 22（GetChapter/GetChapterBranch/SaveChapterContent/SaveChapterBranchContent/GetNovelState/BuildNovelStatePatch/SettleNovelState/DeSlopChapterAiTaste/RewriteChapterAiTaste/GetEntityRelations/GetChapterScenes/GenerateScene/CreateScene/CancelCreateChapter/GenerateProjectCharacterFill/GenerateCharacterPortrait/MergeCharacters/SaveOrganization/DeleteOrganization/ToggleOrgMember/SaveRelationship/DeleteRelationship；批次三c +3：QuickBrainstormBranches/CreateChapter/DeleteOutlineNode（CreatePage 收尾）；批次四 bridge 双轨退役终局 +4：SetActiveASRModel/SetActiveTTSModel/SetChatVoiceModel/VoiceHealth；v4.193 角色回写欠账刀 +CharacterImportPreview；v4.195 询价库级扫描 +CostInquiryScan；v4.196 语义索引显形 +SemanticIndexStatus/Backfill；v4.199 场景元数据 +SaveSceneMeta；v4.213 三态生命周期 +MemoryPin/MemoryLifecycle；6.3 办公多文件 DAG +7：DagList/DagGet/DagRun/DagNodeRun/DagNodeSteer/DagNodeAccept/DagCancel；v4.220 流水线模板库 +4：DagTemplateList/TemplateSave/TemplateNew/TemplateDelete；v4.241 注入体检 +MemoryEvalRun；v4.242 成品直出 +DagAcceptAll；v4.243 审批分级 +DagNodeApprove；注：基线 435 与旧锁 433 差 2 为入库前既有漂移（本表 satisfies 编译期兜底为准））
  });

  it("work/play/shared 三数组两两无交集且之和 + independent = 总数", () => {
    const work = new Set<string>(WORK_BINDING_NAMES);
    const play = new Set<string>(PLAY_BINDING_NAMES);
    const shared = new Set<string>(SHARED_BINDING_NAMES);
    for (const n of work) {
      expect(play.has(n)).toBe(false);
      expect(shared.has(n)).toBe(false);
    }
    for (const n of play) {
      expect(shared.has(n)).toBe(false);
    }
    const independent = Object.keys(GAEA_METHOD_FACETS).filter(
      (k) => GAEA_METHOD_FACETS[k as keyof typeof GAEA_METHOD_FACETS] === "independent",
    );
    expect(work.size + play.size + shared.size + independent.length).toBe(
      Object.keys(GAEA_METHOD_FACETS).length,
    );
  });

  it("SPACE_BINDINGS 解析与抽查：办公/任务→work，轻语→play，空间/模型→shared", () => {
    expect(bindingSpaceOf("XlsxPlanEdit")).toBe("work");
    expect(bindingSpaceOf("TaskList")).toBe("work");
    expect(bindingSpaceOf("WhisperMemories")).toBe("play");
    expect(bindingSpaceOf("GaeaSpaceActivate")).toBe("shared");
    expect(bindingSpaceOf("GaeaSpaceList")).toBe("shared");
    expect(bindingSpaceOf("UnifiedSearch")).toBe("shared"); // scope 参数隔离（S1.2-C）
    expect(bindingSpaceOf("StartProgrammingWeb")).toBe("independent");
  });

  it("isBindingAllowedInSpace：shared/independent 两空间可达，work/play 仅所属空间", () => {
    expect(isBindingAllowedInSpace("XlsxPlanEdit", "work")).toBe(true);
    expect(isBindingAllowedInSpace("XlsxPlanEdit", "play")).toBe(false);
    expect(isBindingAllowedInSpace("WhisperMemories", "play")).toBe(true);
    expect(isBindingAllowedInSpace("WhisperMemories", "work")).toBe(false);
    expect(isBindingAllowedInSpace("GaeaSpaceList", "work")).toBe(true);
    expect(isBindingAllowedInSpace("GaeaSpaceList", "play")).toBe(true);
    expect(isBindingAllowedInSpace("StartProgrammingWeb", "work")).toBe(true);
    expect(isBindingAllowedInSpace("StartProgrammingWeb", "play")).toBe(true);
    // 未知名方法兜底 work（合法调用面已由编译期覆盖）
    expect(isBindingAllowedInSpace("NoSuchMethod", "play")).toBe(false);
  });
});
