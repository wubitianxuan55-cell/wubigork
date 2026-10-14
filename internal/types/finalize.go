package types

// ── 章节定稿档（阶段八 8.4 定稿回写）─────────────────────────
//
// 定稿 = 作者宣布本章正文为最终版的显式动作。定稿时基于**当前正文**（而非生成
// 时的瞬时正文）提取「阶段摘要 + 角色状态 + 伏笔状态」三件套草案，作者确认后才
// 落库（建议制红线延伸：AI 产物未经确认不写持久化数据）。定稿档 chapters/
// NNN{,a,b,c}-final.json 是三件套的落库回执与留痕：回写失败不阻塞定稿，失败件
// 如实进 Errors（作者可重开定稿面板重试）。

// ChapterFinalizationVersion 定稿档契约版本。
const ChapterFinalizationVersion = 1

// StageRecapEntry 阶段合账缓存条目（阶段 = 每 stageLength 章一阶段，见
// create_chapter_stage.go）。定稿阶段收官章时合成落盘，阶段开篇章生成时直接
// 读缓存不再重烧 LLM；未命中回落即时合成（现状兜底，缓存只是加速不是依赖）。
type StageRecapEntry struct {
	Start         int    `json:"start"`          // 阶段起始章号（含）
	End           int    `json:"end"`            // 阶段收官章号（含）
	Text          string `json:"text"`           // 合账文本（三段式）
	FinalizedAt   string `json:"finalized_at"`   // ISO8601
	SourceChapter int    `json:"source_chapter"` // 由哪章定稿触发（追溯）
}

// StageRecapCacheFile chapters/stage-recaps.json（key = 阶段号十进制字符串）。
type StageRecapCacheFile struct {
	Version int                      `json:"version"`
	Recaps  map[string]StageRecapEntry `json:"recaps"`
}

// FinalizeForeshadowSync 伏笔件同步回执（analysis.SyncResult 的落档投影——
// types 不依赖 analysis 包，取作者关心的四字段）。
type FinalizeForeshadowSync struct {
	PlantedCount  int      `json:"planted_count"`
	ResolvedCount int      `json:"resolved_count"`
	CreatedCount  int      `json:"created_count"`
	Errors        []string `json:"errors,omitempty"`
}

// FinalizeApplied 主线章三件套落库回执（分支章为 nil——登记表/合账/状态卡都是
// 主线口径，分支只标记+留痕，见 ChapterFinalization.Skipped）。
type FinalizeApplied struct {
	StageRecap  *StageRecapEntry       `json:"stage_recap,omitempty"` // 本轮落盘的合账（非阶段收官章/未应用=空）
	Characters  int                    `json:"characters"`            // 追加的状态演进条数
	Foreshadows *FinalizeForeshadowSync `json:"foreshadows,omitempty"` // 伏笔同步回执（未应用=空）
}

// ChapterFinalization 定稿档。Applied 与 Errors 并存是正常态：三件套逐件尽力
// 落库，失败件不回滚已成功件（事务性=逐件如实回执，不是全有或全无——作者可
// 针对失败件重试，成功件重放有幂等保护不重复追加）。
type ChapterFinalization struct {
	Version     int              `json:"version"`
	ChapterNum  int              `json:"chapter_num"`
	Branch      string           `json:"branch,omitempty"` // ""=主线
	FinalizedAt string           `json:"finalized_at"`     // ISO8601
	Note        string           `json:"note,omitempty"`   // 作者定稿备注
	Applied     *FinalizeApplied `json:"applied,omitempty"`
	Skipped     string           `json:"skipped,omitempty"` // 分支章：三件套跳过原因
	Errors      []string         `json:"errors,omitempty"`  // 留痕：失败件与原因
}
