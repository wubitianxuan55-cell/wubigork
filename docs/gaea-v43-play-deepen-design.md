# gaea v4.3「乐园」娱乐做深 · 会客厅关系记忆+情感语音 + 创作间世界模型+图文联动

> 2026-08-29 定稿。权威路线图 `docs/gaea-nextgen-roadmap-2026.md` §10.4/§14 阶段 3+；
> 本文件为该 Step 的执行契约。前置：4 份只读调研（轻语关系记忆 / 情感语音 TTS /
> 创作间世界模型 / 角色资产库）结论已并入。纪律沿用：每 Step 独立提交可回退、
> 旧数据只读兼容、不做新板块、不堆功能、**与工位零交叉**（play 空间红线）。

## 1. 背景与调研结论（只读调研，全部落库为事实）

**关键判断：后端骨架约 70% 已存在但「断链 + 参数不透传」。v4.3 最小增量以
「接线已有后端 + 参数扩展 + 前端面板」为主，不做从零新建。**

| 领域 | 已具备（证据） | 关键缺口 |
|---|---|---|
| 轻语关系记忆 | 关系状态机 `relationship.go UpdateRelationship/evolveStage`；PAD 情绪 `emotion.go EmotionStep`（4D+9 标签+惯性衰减 0.97+锁区+clamp10+持久化 companion_state）；情绪入记忆（MemoryFact.EmotionalContext）；三元组图谱 `memory_graph.go` KG + `knowledge_triples` 表持久化 + `ExtractTriples`；规则版主动消息合成器 `companion_proactive.go ComposeProactiveMessage`（6 类型+人格感知）；特殊日期 v1 | ① `memory_associations`/`user_habits`/`temporal_anchors` 表**有 schema 无 repos**（ReseedAssociationGraph 从未调用，重启关联全空）；② 子图召回仅一跳；③ 主动关心**无定时器/推送通道/频控未接线**（AttentionManager 仅测试引用）；④ 三信号缺「未完成事件登记」「作息习惯落库」；⑤ `DetectSpecialDatesV2`/`ShouldWriteTemporalAnchor` 等死代码未接线（用户生日未进日期检测）；⑥ 前端无关系图谱/主动消息流 |
| 情感语音 | 语音管道已自动朗读带情绪标签（voice_manager.handleReply → GetVoiceDescriptionWithPersonality → herdsman voicedesign/voxcpm 生效）；`emotion_voice_map.go` 有 EdgeRate/EdgePitch 数据但从未进合成路径 | ① TTS Seam 只暴露 `Synthesize(text)`（无 speed/style/emotion/pause）；② cosyvoice 走 herdsman default 分支 **voiceDescription 被丢弃**；③ edge SSML 硬编码 rate/pitch；④ **无「即时情绪 vs 长期心境」区分**；⑤ 文本聊天朗读无情绪（TTSSpeakBase64 单参） |
| 创作间世界模型 | worldview.json 6 维度文本结构；**伏笔全链路已实现**（analysis.go syncForeshadows：planted/hinted/revealed+stable ID）；`graph.CheckConsistency` 三类规则（角色属性/状态/时间线）；角色状态回写 syncCharacterStates；实体库 | ① 绑定前端**零调用**（GetForeshadows/CheckConsistency/AnalyzeChapter/GenerateSceneIllustration 死绑定）；② Analyze 只手动触发、生成后不自动落盘；③ 章节生成只注入 plot_req/setting/characters 摘要——**伏笔/lorebook/实体不注入**；④ AddRelation 关系边永不填充；⑤ 设定页纯 Markdown 整篇编辑器（结构化绑定零调用） |
| 角色资产库 | 全局 SQLite characterlib.db 跨项目共享已成熟；立绘生成（CharacterGeneratePortrait）；角色字段含 portraitUrl | ① 无参考图/多视角（单字段 portraitUrl）；② 生图无 IP-Adapter/PuLID 参考槽（ImageGenerationRequest 仅 Lora+InitImage）；③ **章节→配图后端已有（GenerateSceneIllustration）但前端零调用**；④ 书封完全缺失；⑤ 多图叙事缺失 |

## 2. 三支柱与验收红线

1. **会客厅 · 关系记忆 + 主动关心 + 情感语音**：关系图谱持久化闭环（重启关联仍在）+
   子图召回 + 前端图谱面板；主动关心定时推送（频控+作息尊重+上次情绪三信号）；
   TTS 参数扩展（speed/style/emotion）+ 情绪→参数映射 + 长期心境维。
2. **创作间 · 世界模型接线**：设定页维度化编辑器 + 生成后自动 Analyze 落盘 +
   伏笔登记表面板 + 一致性检查面板 + 章节生成结构化注入。
3. **创作间 · 角色资产与图文联动**：角色参考图字段 + 章节配图按钮（打通已有
   GenerateSceneIllustration）+ 书封生成。

**红线**：① 全部 play 空间（轻语/小说/绘梦/角色库），**与工位零交叉**（记忆分区
互不检索、绑定 spaceBindings=play、事件订阅 play 过滤）；② 主动关心带频控与
作息尊重（合规友好，关怀而非诱导）；③ 语音全本地（CosyVoice2/edge/herdsman，
情绪参数不依赖云端）；④ 前端「沉浸优先」——不出现任务/进度/审计词汇。

## 3. Step 拆分与发布节奏（v4.3.0 一次发布，每 Step 独立提交）

- **v4.3a 会客厅·记忆持久化闭环**（后端）：补 `memory_associations`/`user_habits`/
  `temporal_anchors` repos + restore 时 `ReseedAssociationGraph` + 习惯/锚点回填。
- **v4.3b 会客厅·子图召回**（后端纯函数 + 绑定）：`KG.QuerySubgraph(entity, hops)` +
  `GaeaWhisperGraphSubgraph` 绑定（play 域）+ 前端图谱面板（react-flow 或 SVG 邻接图）。
- **v4.3c 会客厅·主动关心定时推送**（后端接线 + 前端消息流）：app 层 ticker
  （30–60min）评估 `ComposeProactiveMessage`（现成）+ `AttentionManager` 频控
  （≤3 条/小时）+ `MatchHabits` 作息尊重 + 上次情绪 → events 通道推前端；
  顺带接线 `ShouldWriteTemporalAnchor`（未完成事件到期锚点）+ `DetectSpecialDatesV2`
  （含用户生日，灌入 AgeMeta.BirthdayMMDD）。
- **v4.3d 会客厅·情感语音**（后端参数扩展 + 心境维 + 前端按钮）：
  `TTSProvider.SynthesizeWithParams(text, TTSParams{Speed,Pitch,Style,Emotion})`
  （旧方法保留兼容）；herdsman buildBody 加 speed/emotion/style、cosyvoice 工厂
  不再丢弃 voiceDescription、edge SSML 参数化 rate/pitch；`EmotionVoiceMap` 扩为
  结构化参数并进合成路径；`EmotionState`/`FullState` 增 `Mood` 4D 长期心境
  （慢速 EWMA 0.01，持久化）；语音参数 = 即时情绪为主 + 心境偏移；文本聊天朗读
  接入最近一轮情绪（`TTSSpeakBase64WithParams` 或等价）。
- **v4.3e 创作间·世界模型接线**（后端自动 Analyze + 前端设定页）：章节生成后自动
  `Analyze` 落盘（foreshadows.json + 角色状态 + 实体）；设定页维度化编辑器
  （复用已有 GetWorldviewSections/SaveAllWorldviewSections 绑定，6 维度分卡片编辑）。
- **v4.3f 创作间·伏笔登记表 + 一致性面板**（前端为主 + 小后端）：伏笔登记表面板
  （GetForeshadows + 回收率统计 planted→hinted→revealed）；`CheckConsistency`
  检查面板（三类规则告警）；章节生成注入伏笔/lorebook（CreateChapter 结构化注入）。
- **v4.3g 创作间·角色资产与图文联动**（后端 + 前端）：Character 增
  ReferenceImages/GalleryImages（SchemaV2 迁移，characterlib.db）；章节配图按钮
  （ChapterPage 打通 GenerateSceneIllustration，前端已死绑定复活）；
  `GaeaGenerateBookCover` 新绑定（3:4 封面，复用 GenerateSceneIllustration 管线）。

> v4.3a/b/c/d（会客厅）与 e/f/g（创作间）两组内部可并行；绑定面/前端集成由父代理收口。

## 4. 数据模型（增量，父代理收编正式迁移）

```sql
-- 轻语记忆持久化闭环（whisper hermes.db，schema 已含表结构，补 repos）
--   memory_associations / user_habits / temporal_anchors —— 只补访问层，表不动。
-- 角色参考图（characterlib.db 增列，v4.3g）
--   ALTER TABLE characters ADD COLUMN reference_images TEXT NOT NULL DEFAULT '[]'; -- JSON []string
--   ALTER TABLE characters ADD COLUMN gallery_images  TEXT NOT NULL DEFAULT '[]'; -- JSON []string
```

不做新表（轻语三表已有 schema）；角色两列走 characterlib 自身迁移机制
（调研确认 characterlib 有独立 schema 版本，沿用其迁移）。

## 5. 后端包契约（v4.3a/b/d/g 核心，子代理实现）

### 5.1 `internal/whisper/db/repos/`（v4.3a）— 补三个 repo

参照既有 repos（companion_state.go 风格）：
- `AssociationsRepo`：`List()/SaveAll([]Association)/Clear()`（表 memory_associations）
- `HabitsRepo`：`List()/SaveAll([]Habit)/Upsert`（表 user_habits）
- `TemporalAnchorsRepo`：`List()/SaveAll/Delete(expired)`（表 temporal_anchors）
- 持久化装配：`state_persistence.go` 保存/恢复路径接入三 repo；`Restore` 后
  `ReseedAssociationGraph`（现有函数，打通调用）。

### 5.2 `internal/whisper/memory_graph.go`（v4.3b）— 子图召回

```go
// QuerySubgraph 返回以 entity 为中心、hops 跳内的邻接子图（节点+边，含类型/权重）。
// 纯函数读取 KG 索引；hop<=0 视为 1。测试覆盖：一跳/两跳/孤立实体/空图。
func (kg *KnowledgeGraph) QuerySubgraph(entity string, hops int) Subgraph
```

`Subgraph{Nodes []GraphNode, Edges []GraphEdge}`（GraphNode: ID/Name/Type/Weight；
GraphEdge: From/To/Type/Weight）。绑定 `GaeaWhisperGraphSubgraph(entity, hops)`。

### 5.3 `internal/tts/provider.go` + 各 kind（v4.3d）— 参数扩展

```go
type TTSParams struct {
    Speed   float64 // 倍速 0.5-2.0（0=不指定，edge 映射为 rate、herdsman 传 speed）
    Pitch   float64 // 音高偏移半音 -12..12（0=不指定，edge 映射 pitch、herdsman 传 pitch）
    Style   string  // 风格标签（cosyvoice/herdsman 透传 style）
    Emotion string  // 情绪标签（ANGRY/CALM/... 透传 emotion）
}
type TTSProvider interface {
    Name() string
    Synthesize(text string) ([]byte, error)                    // 保留兼容
    SynthesizeWithMime(text string) ([]byte, string, error)    // 保留兼容
    SynthesizeWithParams(text string, p TTSParams) ([]byte, string, error) // 新增
}
```

- herdsman buildBody：加 `speed`（Speed>0 时）、`emotion`、`style`；**cosyvoice 模型
  分支保留并传递 voiceDescription（不再落 default 丢弃）**。
- edge：SSML 由参数生成 `rate='+X%' pitch='+YHz'`（Speed→rate、Pitch→Hz）。
- xai/sapi：忽略参数（能力外），不报错。
- `EmotionVoiceMap` 扩为结构化（每情绪标签 → TTSParams 默认值：语速百分比/音高/
  风格/情绪），`GetEmotionVoiceParams(label)` 导出；voice_manager 合成路径传参。
- 长期心境：`EmotionState` 增 `Mood [4]float64`（-100..100，EWMA α=0.01 向即时
  情绪靠拢，随 FullState 持久化）；语音参数最终值 = 即时情绪为主 + 心境偏移
  （如 Mood 低落 → 语速 -5%）。

### 5.4 `internal/app/gaea_book_cover.go`（v4.3g）— 书封生成

复用 GenerateSceneIllustration 管线：`GaeaGenerateBookCover(projectID, promptHint)` →
3:4 封面图（落 .gaea/play/exports/cover-<project>.png，play 空间）+ 回填项目封面
字段（project 无 cover 字段时仅返回路径，前端书架卡显示）。不落 work 目录（play 红线）。

## 6. 绑定面新增（v4.3，父代理集成；绑定面 517 → 约 526）

- `GaeaWhisperGraphSubgraph(entity string, hops int) (whisper.Subgraph, error)`（play）
- `GaeaWhisperProactiveNow() (string, error)`（主动关心手动触发/调试，play）
- `GaeaWhisperProactiveConfig() / GaeaWhisperSetProactiveConfig(...)`（频控/时窗配置，play）
- `GaeaTTSVoiceParams(emotion string) tts.TTSParams`（前端预览/调试，shared）
- `TTSSpeakBase64WithParams(text string, p tts.TTSParams) (string, error)`（语音绑定面）
- `GaeaGenerateBookCover(projectID, promptHint string) (string, error)`（play）
- （e/f 复用既有绑定：GetForeshadows/CheckConsistency/GenerateSceneIllustration/
  GetWorldviewSections 等已有，只接线前端）

spaceBindings 分类：Whisper*/BookCover/Foreshadows 系 = play；TTS 系 = shared。
绑定新增后 `go run ./scripts/gen_bindings` 重生成 + bindingNames/spaceBindings/bridge 同步。

## 7. 前端面板（v4.3b/c/e/f/g，父代理集成）

- **图谱面板**（会客厅）：实体输入/点击 → 邻接子图（SVG 或 react-flow 现有组件），
  节点按类型着色（人物/约定/纪念日/心结），边标关系类型。
- **主动消息流**（会客厅对话流）：ticker 推来的主动消息以「轻语先开口」气泡插入，
  带类型徽标（check_in/纪念日/约定到期），用户可关闭频控。
- **设定页维度化编辑器**（创作间）：6 维度卡片（era/geography/factions/rules/
  culture/history）分卡片编辑保存（复用已有绑定，零新增）。
- **伏笔登记表面板**（创作间）：列表 + 状态流转（登记→埋设→回收）+ 回收率统计。
- **一致性检查面板**（创作间）：三类规则告警列表 + 「重新检查」。
- **章节配图/书封**（创作间）：ChapterPage「生成配图」按钮（复活死绑定）+
  项目书架「生成封面」按钮。
- 角色库编辑器：参考图/画廊图管理（上传/移除，字段已加）。

## 8. 验收清单

1. 轻语：重启后关联/习惯/锚点仍在（持久化闭环）；查实体返回邻接子图且 play 隔离；
   离线 12h+ 收到 check_in 且 ≤3 条/小时、DND 不发；生日当天首条消息带祝福。
2. 情感语音：同一句在 ANGRY/CALM 下 CosyVoice2/edge 请求参数可观测不同；
   连续低落多轮后中性轮仍带低落偏移（WhisperGetState 可观测 Mood）。
3. 创作间：章节生成后自动落盘 foreshadows+角色状态；伏笔登记表/一致性面板可用；
   章节配图/书封按钮产出图（play exports）。
4. 绑定面漂移 PASS；spaceBindings 新方法分类齐全（play/shared）。
5. 前端 tsc/eslint 0 + vitest 全绿；Go 全量绿。
6. 回退：v4.3a–g 各自独立提交可 revert；旧数据只读兼容（轻语表结构已存在，
   角色表新增列默认 '[]' 兼容旧行）。
