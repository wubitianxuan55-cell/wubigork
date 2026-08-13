/**
 * Wails 运行时类型声明
 * 为 window.go.app.App.* 提供类型安全，消除全站的 @ts-ignore 和 any 类型
 */

/* eslint-disable @typescript-eslint/no-explicit-any */

declare global {
  interface Window {
    go?: {
      app?: {
        App: AppAPI
      }
    }
    runtime?: {
      EventsOn: (event: string, handler: (data: any) => void) => void
      EventsOff?: (event: string) => void
      BrowserOpenURL?: (url: string) => void
    }
  }
}

/** gaea 后端 App 接口 */
interface AppAPI {
  // ── 认证 ──
  Login(): Promise<string>
  GetLoginStatus(): Promise<boolean>

  // ── 项目 ──
  CreateProject(dir: string, title: string, genre: string, style: string): Promise<any>
  BootstrapProject(dir: string, title: string, genre: string, style: string, reference: string): Promise<any>
  OpenProject(dir: string): Promise<any>
  CloseProject(): Promise<void>
  GetProjectInfo(): any
  GetNovelsDir(): string
  ListProjects(): Promise<ProjectCard[]>
  DeleteProject(dir: string): Promise<void>

  // ── 大纲 ──
  GetOutlines(): { nodes: any[]; story_thread?: string }
  SaveOutlineNode(nodeJSON: string): Promise<void>
  AddOutlineNode(nodeJSON: string): Promise<void>
  DeleteOutlineNode(nodeID: string): Promise<void>
  GenerateOutlineNodeDetail(nodeID: string): Promise<any>
  ContinueOutline(count: number): Promise<any>
  ExpandOutlineNode(nodeID: string, subCount: number): Promise<any>
  SaveStoryThread(storyThread: string): Promise<void>
  GetStoryThread(): string
  GenerateStoryThread(userHint: string): Promise<any>
  ChatStoryThread(userMsg: string): Promise<any>
  ChatOutline(userMsg: string): Promise<any>
  ChatOutlineNode(nodeID: string, userMsg: string): Promise<any>
  ImportStoryThreadFile(): Promise<any>
  GenerateOutlineWithDialogue(storyPrompt: string, numChapters: number, maxTurns: number): Promise<any>

  // ── 章节 ──
  GenerateChapter(outlineNodeID: string, skillName: string, targetWords: number): Promise<any>
  GetChapter(num: number): Promise<{ content?: string; summary?: any }>
  GetChapterBranch(num: number, branch: string): Promise<{ content?: string; summary?: any }>
  SaveChapterContent(num: number, content: string): Promise<void>
  SaveChapterBranchContent(num: number, branch: string, content: string): Promise<void>
  ChatChapter(chapterNum: number, userMsg: string): Promise<any>
  ReviewChapter(chapterNum: number): Promise<any>
  GenerateSceneIllustration(chapterNum: number): Promise<any>
  GetChapterScenes(chapterNum: number): Promise<any[]>
  SaveScene(chapterNum: number, sceneID: string, content: string): Promise<void>
  ReorderScenes(chapterNum: number, sceneIDs: string[]): Promise<void>
  CreateSnapshot(sceneID: string, chapterNum: number, label: string): Promise<any>
  ListSnapshots(sceneID: string, chapterNum: number): Promise<any[]>
  RestoreSnapshot(snapshotID: string, sceneID: string, chapterNum: number): Promise<void>
  MigrateProjectToV4(): Promise<void>
  IsProjectV4(): boolean

  // ── 角色 ──
  GetCharacters(): { characters?: any[]; organizations?: any[]; relationships?: any[] }
  GenerateCharacters(count: number): Promise<any>
  SaveCharacter(chJSON: string): Promise<void>
  DeleteCharacter(id: string): Promise<void>
  GenerateSingleCharacter(chJSON: string): Promise<any>
  ChatCharacter(userMsg: string): Promise<any>
  ChatCharacterDetail(charID: string, userMsg: string): Promise<any>
  SaveOrganization(orgJSON: string): Promise<void>
  DeleteOrganization(id: string): Promise<void>
  ToggleOrgMember(charID: string, orgID: string): Promise<void>
  SaveRelationship(relJSON: string): Promise<void>
  DeleteRelationship(fromID: string, toID: string): Promise<void>
  SaveCharacters(cfJSON: string): Promise<void>
  GenerateCharacterPortrait(charID: string, model: string): Promise<string>
  GenerateProjectCharacterFill(chJSON: string): Promise<string>
  MergeCharacters(keepID: string, mergeID: string): Promise<void>
  SetCharacterPortrait(charID: string, imageData: string): Promise<void>

  // ── 世界观 ──
  GetWorldviewSections(): Promise<any>
  ChatWorldview(userMsg: string): Promise<any>
  ChatWorldviewSection(sectionID: string, userMsg: string): Promise<any>
  SaveWorldviewSection(sectionID: string, content: string): Promise<void>
  SaveAllWorldviewSections(sectionsJSON: string): Promise<void>
  CheckWorldviewConsistency(): Promise<any>
  GenerateWorldviewSections(): Promise<any>
  SaveWorldview(content: string): Promise<void>
  GetWorldview(): string

  // ── AI 协写 ──
  GhostComplete(currentText: string, styleProfile: string): Promise<any>
  CancelGhost(): void
  CmdKEdit(selectedText: string, instruction: string, styleProfile: string): Promise<any>
  GenerateBeats(outlineNodeID: string): Promise<any[]>
  GenerateProseFromBeat(beatID: string, allBeatJSON: string, chapterNum: number): Promise<any>

  // ── 图片生成 ──
  GenerateFreeImage(prompt: string, negative: string, size: string, style: string, model: string, seed: number, n: number): Promise<any>
  CancelImageGeneration(): Promise<boolean>
  GetComfyUITaskProgress(): Promise<{ status?: string; elapsed?: number }>
  GetImageBackend(): string
  GetImageBackendInfo(): { backend?: string; model?: string }
  SetImageBackend(backend: string, comfyUIURL: string, imageModel: string): Promise<void>

  // ── 分析/审稿 ──
  AnalyzeChapter(chapterNum: number): Promise<any>
  GetForeshadows(): any
  ReviewBook(): Promise<any>
  GetBookData(): any

  // ── 统计 ──
  GetStats(): any
  GetConfig(): Record<string, string>
  SaveConfig(key: string, value: string): Promise<void>
  ListSkills(): any[]
  GetDashboard(dailyGoal: number): Promise<any>

  // ── TTS ──
  StartTTSServer(modelPath: string, port: number, backend: string): Promise<void>
  StopTTSServer(): Promise<void>
  TTSSpeakStreaming(text: string): Promise<void>
  GetTTSStatus(): any
  GetTTSConfig(): any
  SaveTTSConfig(modelPath: string, serverPath: string, port: number, backend: string, speed: number): Promise<void>

  // ── 导出 ──
  ExportAll(): Promise<Record<string, string>>

  // ── 知识图谱 ──
  BuildBacklinkIndex(): Promise<any>
  GetBacklinks(entityName: string): Promise<any[]>
  GetAllEntityNames(): Promise<any[]>
  BuildContextBudget(systemPrompt: string, currentScene: string, previousScene: string, characterInfo: string, memoryInfo: string, modelCapacity: number): Promise<any>
  CheckConsistency(): Promise<any>
  GetEntityRelations(): Promise<any>

  // ── 可视化 ──
  ExtractTimeline(): Promise<any>
  ExtractEmotionCurve(): Promise<any[]>
  ExtractCharacterHeatmap(): Promise<any>
  GenerateDefaultCanvas(): Promise<any>

  // ── 搜索 ──
  Search(query: string): Promise<any>

  // ── 脑暴 ──
  BrainstormIdeas(genre: string): Promise<any>
  BrainstormBranches(nodeID: string): Promise<any>
  ApplyBranch(nodeID: string, branchIndex: number, userInput: string): Promise<any>

  // ── Lorebook ──
  GetLorebookEntries(): any
  SaveLorebookEntry(entryJSON: string): Promise<void>
  DeleteLorebookEntry(key: string): Promise<void>

}

interface ProjectCard {
  path: string
  title: string
  chapter_count: number
  word_count: number
  last_modified: string
}

export {}
