// spaceBindings.test.ts — S2.3 bridge 分面分类表（docs/gaea-space-shell-design.md §7）
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
    expect(total).toBe(462); // 与 keyof AppBindings 一致（satisfies 编译期钉死；v4.258 原罪取消 +1：SinCancel；v4.257 原罪角色库 +2：SinCastGet/SinCastSet；v4.244 原罪板块 +9：SinTopicsList/SinTopicCreate/SinTopicRename/SinTopicDelete/SinTopicClear/SinMessages/SinStream/SinIllustrate/SinExportMarkdown（闲庭 play 数据面）；v4.28 + PptxOutline/GaeaBrowserObserve；v4.64 + SubagentFollowUp；v4.66 + PromoteSubagent；v4.78 + TaskKill；v4.80 + ContextNodeDetail；v4.86 + GaeaGit*7；v4.94 + SubagentContextView；v4.99 直调转正 + ImageHubAssets/ChapterArtList；v4.102 图像域直调族转正 +17；v4.105 +WarmComfyUI；v4.109 +PptxApplyEdit；v4.113 +ScheduleLoad/Save；v4.134 +ScheduleExportXlsx/ImportXlsx；v4.139 进度计划多工程 +ScheduleProjects/ProjectOpen/ProjectCreate/ProjectArchive/ProjectDelete；v4.140 +ScheduleImportMpp；v4.145 +ScheduleProjectCopy；v4.156 pptx 真编辑刀2 +PptxSlideText；v4.158 组价复核闭环 +CostComposeRecords；v4.162 进度计划导入导出壳内修复 +ReadFileB64/SaveFileAs；v4.163 长期日志机制 +OpenLogsDir；v4.171 wailsjsCompat 双轨退役批次一 +16：GetAppInfo/GetBoardManifests/GetFeatureModel/GetFeatureModelEnabled/SetFeatureModel/SetFeatureModelEnabled/GetActiveModel/StartLocalTTSService/VoiceApplySettings/WhisperGetState/WhisperGetFacts/WhisperGetTraces/WhisperDeleteFact/WhisperUpdateFact/NovelSearch/SaveSettings；批次二 +29：GetConfig/SaveConfig/GetStats/ListSkills/ExportAll/ChatTopicCreate/ChatTopicDelete/ChatTopicRename/ChatTopicSetMode/ChatImportTopic/ChatSend/ChatStreamPlain/ChatTopicClear/ChatTopicExportMarkdown/GetWorldview/SaveWorldview/ChatWorldview/GetWorldviewSections/SaveAllWorldviewSections/CheckConsistency/CheckConsistencyDeep/GetForeshadows/SaveForeshadows/SaveCharactersBatch/NovelReadingAsk/GenerateSceneIllustration/VoiceGetSettings/VoiceChatText/SetImageBackend；批次三a +18：VoiceStart/VoiceStop/VoicePlaybackDone/VoiceCancelTTS/VoicePushAudio/VoiceSetPTTActive/WhisperClearSession/TTSSpeakBase64/TTSSpeakBase64WithParams/GetTTSSpeakers/GetVoicePipelineConfig/DataBackupInfo/DataBackupCreate/DataBackupRestore/DataBackupCancel/DataBackupRollback/DataBackupRestoreResult/ListProjects；批次三b +38：Charlib 16（CharacterSave/Get/Delete/ImportProject/ListByProject/Associate/AssociateTo/SetProjectState/Dissociate/SyncProject/DrawRandom/GenerateFill/GenerateRandom/FillAll/GeneratePortrait/GeneratePortraitWithRef）+ Novel 22（GetChapter/GetChapterBranch/SaveChapterContent/SaveChapterBranchContent/GetNovelState/BuildNovelStatePatch/SettleNovelState/DeSlopChapterAiTaste/RewriteChapterAiTaste/GetEntityRelations/GetChapterScenes/GenerateScene/CreateScene/CancelCreateChapter/GenerateProjectCharacterFill/GenerateCharacterPortrait/MergeCharacters/SaveOrganization/DeleteOrganization/ToggleOrgMember/SaveRelationship/DeleteRelationship；批次三c +3：QuickBrainstormBranches/CreateChapter/DeleteOutlineNode（CreatePage 收尾）；批次四 bridge 双轨退役终局 +4：SetActiveASRModel/SetActiveTTSModel/SetChatVoiceModel/VoiceHealth；v4.193 角色回写欠账刀 +CharacterImportPreview；v4.195 询价库级扫描 +CostInquiryScan；v4.196 语义索引显形 +SemanticIndexStatus/Backfill；v4.199 场景元数据 +SaveSceneMeta；v4.213 三态生命周期 +MemoryPin/MemoryLifecycle；6.3 办公多文件 DAG +7：DagList/DagGet/DagRun/DagNodeRun/DagNodeSteer/DagNodeAccept/DagCancel；v4.220 流水线模板库 +4：DagTemplateList/TemplateSave/TemplateNew/TemplateDelete；v4.241 注入体检 +MemoryEvalRun；v4.242 成品直出 +DagAcceptAll；v4.243 审批分级 +DagNodeApprove；注：基线 435 与旧锁 433 差 2 为入库前既有漂移（本表 satisfies 编译期兜底为准））
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
