# 附录 B · 绑定面死面 / 包依赖图 / 资产存活（2026-10-02）

> 脚本：`machine/dead_bindings.py`、`machine/imports.py`、`machine/api_layer.py`。全部数据可复跑。

## B1. 绑定面（743 方法）的前端消费面

口径：取 `go run ./scripts/gen_bindings -names` 的 Go 侧全部导出绑定方法（743 个，与 `bindingNames.ts` 零漂移），
然后在前端源码里统计「生产代码」「测试文件」两类消费者。以下文件视为**机械登记面**不计消费者：
`bindingNames.ts`、`lib/bridge/**`、`lib/mock/**`、`spaceBindings.ts`、`types/wails.d.ts`、`wailsjs/**`、`wailsjsCompat/**`。
**`frontend/src/api/**` 是真实调用层（18~53 个导入方），已计入消费者。**

| 分类 | 方法数 | 占比 | 含义 |
|---|---|---|---|
| 零引用（生产+测试都没有） | 322 | 43.3% | 前端代码里根本没人调用 |
| 仅测试引用 | 31 | 4.2% | 生产不调用，只有测试钉着 |
| 仅 1 处生产引用 | 287 | 38.6% | 调用面极窄 |
| 合计覆盖 | 743 / 743 | | 绑定面总数 743 |

> **口径警告（必须随结论一起引用）**：「零引用」= 前端（含 legacy `api/` 层）没有任何调用点，
> **不等于可以直接删**——其中一部分是给 CLI / 外部脚本 / HTTP 面（`GAEA_HTTP_PORT` 模式）留的导出。
> AGENTS 里已把「零调用者绑定删除」列为待拍板项（宽口径 36，与本处窄口径 322 差一个量级，**两套口径需要对齐**）。

### B1.1 零引用方法按前缀分组（共 322 个）

| 前缀 | 数量 | 样例（最多 12） |
|---|---|---|
| Gaea | 152 | `GaeaAcceptMemorySuggestion`, `GaeaAcceptMergeSuggestion`, `GaeaAcceptSkillSuggestion`, `GaeaAddMCPServer`, `GaeaAddPermissionRule`, `GaeaApplyUpdate`, `GaeaArchiveSession`, `GaeaCallTool`, `GaeaCapabilities`, `GaeaCaptureScreen`, `GaeaChangeFactType`, `GaeaCommands` |
| 其它 | 82 | `AddOutlineNode`, `AnalyzeStyle`, `BuildBacklinkIndex`, `BuildContextBudget`, `BuildRichContext`, `CheckModuleIntegrity`, `CloseProject`, `ContinueOutline`, `CreateSnapshot`, `DeleteCharacter`, `DeleteLorebookEntry`, `ExpandOutlineNode` |
| GaeaCost | 41 | `GaeaCostAttribution`, `GaeaCostCategories`, `GaeaCostCategoryDelete`, `GaeaCostCategorySave`, `GaeaCostCompare`, `GaeaCostComposeApply`, `GaeaCostComposeRecords`, `GaeaCostDelete`, `GaeaCostEstimateItemDelete`, `GaeaCostEstimateItemSave`, `GaeaCostEstimateItems`, `GaeaCostEstimateSediment` |
| GaeaMemory | 12 | `GaeaMemory`, `GaeaMemoryBrief`, `GaeaMemoryDuplicates`, `GaeaMemoryGraph`, `GaeaMemoryHubOverview`, `GaeaMemoryLifecycle`, `GaeaMemoryMerge`, `GaeaMemoryPin`, `GaeaMemorySetRetentionDays`, `GaeaMemorySuggestions`, `GaeaMemoryUnarchive`, `GaeaMemoryUnarchiveBatch` |
| Office | 8 | `OfficeCancelJob`, `OfficeExecute`, `OfficeGetJobState`, `OfficeGetMode`, `OfficeIsTask`, `OfficeListFolder`, `OfficeReadFile`, `OfficeSetMode` |
| Chat | 5 | `ChatCharacter`, `ChatCharacterDetail`, `ChatGeneral`, `ChatOutline`, `ChatOutlineNode` |
| GaeaSkill | 5 | `GaeaSkillDistillDecide`, `GaeaSkillDistillDraft`, `GaeaSkillDraftFromSession`, `GaeaSkillDraftSave`, `GaeaSkills` |
| Brain | 4 | `BrainCrossRefs`, `BrainSearch`, `BrainWrite`, `BrainstormBranches` |
| GaeaTask | 4 | `GaeaTaskCancel`, `GaeaTaskKill`, `GaeaTaskRetry`, `GaeaTaskTemplates` |
| Voice | 4 | `VoiceGetState`, `VoiceRestartService`, `VoiceSetInputChannel`, `VoiceSetMode` |
| GaeaDream | 2 | `GaeaDreamPurge`, `GaeaDreamPurgePreview` |
| Export | 1 | `ExportHTML` |
| Import | 1 | `ImportStyleProfile` |
| Search | 1 | `SearchMemories` |

### B1.2 仅测试引用的方法（31 个，按引用数降序）

| 方法 | 仅有的引用位置 |
|---|---|
| `GaeaAnswer` | `frontend/src/gaea/lib/store.t74.test.ts` |
| `GaeaBenchmarkDetail` | `frontend/src/gaea/lib/mock-contract-benchmark.test.ts` |
| `GaeaCancel` | `frontend/src/gaea/lib/store.t74.test.ts` |
| `GaeaCostSearchPage` | `frontend/src/gaea/components/CostLibraryView.test.tsx` |
| `GaeaDagAcceptAll` | `frontend/src/gaea/components/DagPanel.test.tsx` |
| `GaeaDagCancel` | `frontend/src/gaea/components/DagPanel.test.tsx` |
| `GaeaDagNodeAccept` | `frontend/src/gaea/components/DagPanel.test.tsx` |
| `GaeaDagNodeSteer` | `frontend/src/gaea/components/DagPanel.test.tsx` |
| `GaeaDagRun` | `frontend/src/gaea/components/DagPanel.test.tsx` |
| `GaeaDeleteSession` | `frontend/src/gaea/lib/store.t74.test.ts` |
| `GaeaKnowledgeList` | `frontend/src/gaea/components/KnowledgePanel.t74.test.tsx` |
| `GaeaMemoryFeedback` | `frontend/src/gaea/components/Message.test.tsx` |
| `GaeaNewSession` | `frontend/src/gaea/lib/store.t74.test.ts` |
| `GaeaRemember` | `frontend/src/gaea/lib/store.t74.test.ts` |
| `GaeaSaveDoc` | `frontend/src/gaea/lib/store.t74.test.ts` |
| `GaeaScheduleProjectArchive` | `frontend/src/schedule/store.sync.test.ts` |
| `GaeaScheduleProjectCopy` | `frontend/src/schedule/store.sync.test.ts` |
| `GaeaScheduleProjectDelete` | `frontend/src/schedule/store.sync.test.ts` |
| `GaeaSetPermLevel` | `frontend/src/gaea/lib/store.t74.test.ts` |
| `GaeaTrajectory` | `frontend/src/gaea/lib/bridge.variadic.test.ts` |
| `GaeaUnifiedSearch` | `frontend/src/gaea/lib/mock-contract-e5.test.ts` |
| `ToggleOrgMember` | `frontend/src/gaea/lib/spaceBindings.test.ts` |
| `GaeaApprove` | `frontend/src/gaea/components/ApprovalModal.test.tsx`, `frontend/src/gaea/lib/store.t74.test.ts` |
| `GaeaDeliverableRegistry` | `frontend/src/gaea/components/DeliverableCards.test.tsx`, `frontend/src/gaea/lib/store.invalidate.test.ts` |
| `GaeaContext` | `frontend/src/gaea/lib/store.host.test.ts`, `frontend/src/gaea/lib/store.invalidate.test.ts` |
| `GaeaFactBase` | `frontend/src/gaea/lib/store.host.test.ts`, `frontend/src/gaea/lib/store.invalidate.test.ts` |
| `GaeaListSessions` | `frontend/src/gaea/components/DeliverableCards.test.tsx`, `frontend/src/gaea/components/DeliverablesPanel.test.tsx` |
| `GaeaBalance` | `frontend/src/gaea/lib/bridge.test.ts`, `frontend/src/gaea/lib/store.host.test.ts` |
| `GaeaHistory` | `frontend/src/gaea/lib/bridge.test.ts`, `frontend/src/gaea/lib/store.host.test.ts` |
| `GaeaJobs` | `frontend/src/gaea/lib/bridge.test.ts`, `frontend/src/gaea/lib/store.host.test.ts` |
| `GaeaMeta` | `frontend/src/gaea/lib/bridge.test.ts`, `frontend/src/gaea/lib/store.host.test.ts` |

### B1.3 生产调用面最宽的方法 TOP 20（改动风险面）

| 引用点数 | 方法 |
|---|---|
| 23 | `Search` |
| 10 | `GaeaPreview` |
| 9 | `GaeaPickFiles` |
| 8 | `GaeaJournalList` |
| 7 | `HerdsmanModelCatalog` |
| 7 | `GaeaSubagentRuns` |
| 7 | `CharacterList` |
| 6 | `GaeaSpaceActive` |
| 6 | `GaeaAgentNetwork` |
| 6 | `Chat` |
| 5 | `VoiceApplySettings` |
| 5 | `SinIllustrate` |
| 5 | `GaeaScheduleProjects` |
| 5 | `GaeaScheduleLoad` |
| 5 | `GaeaReadFileB64` |
| 5 | `CheckConsistencyDeep` |
| 4 | `GetCharacters` |
| 3 | `WhisperGetPersonalities` |
| 3 | `VoiceGetSettings` |
| 3 | `UpdateProjectMeta` |

## B2. 前端 legacy `api/` 层的存活面

| 文件 | 生产导入方数量 | 样例导入方 |
|---|---|---|
| `api/characterlib.ts` | 18 | `boards/manifests.ts`, `components/characterlib/CharacterCard.tsx`, `components/characterlib/CharacterLibEditor.tsx` |
| `api/engines.ts` | 20 | `components/ModuleLauncher.tsx`, `components/settings/ImageGenPanel.tsx`, `components/settings/ModelPanel.tsx` |
| `api/httpToken.ts` | 2 | `api/runtimePolyfill.ts`, `gaea/lib/bridge/http.ts` |
| `api/image.ts` | 53 | `components/characterlib/CharacterLibEditor.tsx`, `components/imagegen/AssetLibrary.tsx`, `components/imagegen/AssetStudio.tsx` |
| `api/runtimePolyfill.ts` | 1 | `App.tsx` |
| `api/settings.ts` | 18 | `boards/launcher.ts`, `boards/manifests.ts`, `components/ModuleLauncher.tsx` |

> 结论：`api/` 不是僵尸层（characterlib 18 个导入方、image 53 个、engines 20 个），
> 但它与 `gaea/lib/bridge/**` 构成**两套并存的前端调用面**——同一批 Go 绑定方法两条路径可达。

## B3. Go 包依赖图（internal/ 共 139 个包）

- **上帝装配包**：`internal/app` 直接 import 110 个包（第二名的 `internal/gaea/boot` 27 个）——所有域的装配都压在 app。
- **层级倒挂检查**：非 app 包反向 import `internal/app` 的边 = **0 条**（该维度干净）。
- **同名双包 7 组**（同一概念两套实现的头号嫌疑）：

| 基名 | 包路径 | 非测试文件数 |
|---|---|---|
| cache | `internal/gaea/agent/cache` / `internal/gaea/cache` | |
| config | `internal/config` / `internal/gaea/config` | |
| context | `internal/context` / `internal/gaea/context` | |
| db | `internal/gaea/db` / `internal/whisper/db` | |
| memory | `internal/gaea/memory` / `internal/memory` | |
| search | `internal/gaea/search` / `internal/search` | |
| skill | `internal/gaea/skill` / `internal/skill` | |

- **fan-in TOP 20**（改动波及面最大的包）：

| 被依赖包数 | 包 |
|---|---|
| 19 | `internal/types` |
| 17 | `internal/netclient` |
| 17 | `internal/gaea/provider` |
| 16 | `internal/gaea/fileutil` |
| 15 | `internal/project` |
| 13 | `internal/util` |
| 12 | `internal/gaea/tool` |
| 8 | `internal/gaea/config` |
| 8 | `internal/config` |
| 8 | `internal/ai` |
| 8 | `internal/gaea/proc` |
| 8 | `internal/gaea/cost` |
| 7 | `internal/prompt` |
| 7 | `internal/docmd` |
| 7 | `internal/gaea/event` |
| 7 | `internal/gaea/spaces` |
| 7 | `internal/gaea/strutil` |
| 7 | `internal/gaea/agent/session` |
| 6 | `internal/gaea/genui` |
| 5 | `internal/gaea/memory` |

- **单包 import 最多（最耦合的消费方）TOP 15**：

| import 包数 | 包 |
|---|---|
| 110 | `internal/app` |
| 27 | `internal/gaea/boot` |
| 24 | `internal/gaea/tool/builtin` |
| 23 | `internal/gaea/control` |
| 19 | `internal/gaea/agent` |
| 8 | `internal/character` |
| 7 | `internal/outline` |
| 7 | `internal/analysis` |
| 6 | `internal/chapter` |
| 6 | `internal/ai` |
| 5 | `internal/worldview` |
| 5 | `internal/whisper` |
| 5 | `internal/novelcontext` |
| 5 | `internal/gaea/config` |
| 4 | `internal/project` |

## B4. 提示词资产存活

- `prompts/*.json` 共 23 个模板，**零引用 0 个**（每个模板名都能在 Go/TS 源码里找到引用点）。
- 模板清单：`analysis-chapter`, `book-import-outline`, `book-import-project`, `book-review`, `chapter-plan`, `chapter-review`, `chapter-summary`, `character`, `character-detail`, `character-generate-batch`, `character-generate-single`, `create-chapter`, `create-chapter-first`, `outline-chat`, `outline-chat-node`, `outline-continue`, `outline-expand`, `plot-branch-browser`, `rewrite-chapter`, `skill-from-journal`, `skill-from-session`, `stage-recap`, `worldview`
