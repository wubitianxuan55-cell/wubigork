package app

import (
	"os"
	"time"

	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/knowledge"
	"github.com/gaea/gaea/internal/gaea/memory"
	whisperdb "github.com/gaea/gaea/internal/whisper/db/repos"
)

// ── 记忆中枢绑定（主脑前端入口的统一管理接口）───────────────────────
//
// 记忆中枢 = 三脑架构的前端呈现：左脑办公记忆（Hephaestus.db facts）、
// 主脑全局画像（profile）+ 知识库（knowledge）、右脑轻语记忆（hermes.db 只读）。
// 成本库等未来库沿用同一"库管理"模式扩展。

// ProfileFactView 主脑全局画像事实视图（与 memory.Memory 对应）。
type ProfileFactView struct {
	Name        string   `json:"name"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Type        string   `json:"type"`
	Kind        string   `json:"kind"`
	Tags        []string `json:"tags"`
	Body        string   `json:"body"`
}

// WhisperMemoryView 轻语记忆事实只读视图。
type WhisperMemoryView struct {
	ID          string    `json:"id"`
	Domain      string    `json:"domain"`
	Subcategory string    `json:"subcategory"`
	Subject     string    `json:"subject"`
	Summary     string    `json:"summary"`
	Weight      float64   `json:"weight"`
	Confidence  float64   `json:"confidence"`
	Tier        string    `json:"tier"`
	Status      string    `json:"status"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// MemoryHubOverview 记忆中枢聚合总览（各库统计 + 最近条目）。
type MemoryHubOverview struct {
	KnowledgeCount int    `json:"knowledgeCount"`
	ProfileCount   int    `json:"profileCount"`
	OfficeCount    int    `json:"officeCount"`
	WhisperCount   int    `json:"whisperCount"`
	LatestUpdated  string `json:"latestUpdated"`
}

// hubProfileStore 构造主脑画像存储（nil 表示不可用）。
func (a *App) hubProfileStore() *memory.ProfileStore {
	userDir := config.MemoryUserDir()
	if userDir == "" {
		return memory.NewProfileStore(nil)
	}
	return memory.NewProfileStore(db.GetDatabase(userDir))
}

// hubOfficeStore 构造左脑办公记忆存储（SQLite 后端）。
func (a *App) hubOfficeStore() memory.Store {
	userDir := config.MemoryUserDir()
	cwd, _ := os.Getwd()
	return memory.SQLiteStoreFor(db.GetDatabase(userDir), userDir, cwd)
}

// GaeaProfileList 返回主脑全局画像事实列表。
func (a *App) GaeaProfileList() []ProfileFactView {
	ps := a.hubProfileStore()
	all := ps.All()
	out := make([]ProfileFactView, 0, len(all))
	for _, m := range all {
		out = append(out, ProfileFactView{
			Name: m.Name, Title: m.Title, Description: m.Description,
			Type: string(m.Type), Kind: string(m.Kind), Tags: m.Tags, Body: m.Body,
		})
	}
	return out
}

// GaeaProfileSave 保存一条主脑画像事实（同名覆盖）。
func (a *App) GaeaProfileSave(f ProfileFactView) error {
	return a.hubProfileStore().Save(memory.Memory{
		Name: f.Name, Title: f.Title, Description: f.Description,
		Type: memory.NormalizeType(f.Type), Kind: memory.NormalizeKind(f.Kind),
		Tags: f.Tags, Body: f.Body,
	})
}

// GaeaProfileDelete 删除一条主脑画像事实。
func (a *App) GaeaProfileDelete(name string) error {
	return a.hubProfileStore().Delete(name)
}

// GaeaProfileConflicts 返回画像与办公 facts 中同名且描述不一致的冲突项
// （主脑画像 vs 左脑遗留 facts 对同一事实说法不同）。
func (a *App) GaeaProfileConflicts() []string {
	ps := a.hubProfileStore()
	store := a.hubOfficeStore()
	return ps.DetectConflicts(store.List())
}

// GaeaWhisperMemories 返回轻语（hermes.db）记忆事实只读列表。
// 轻语记忆由 Hermes 自己管理，记忆中枢只读浏览，不提供写入。
func (a *App) GaeaWhisperMemories() []WhisperMemoryView {
	facts := whisperdb.LoadFactsFromDB(a.whisperDataRoot)
	out := make([]WhisperMemoryView, 0, len(facts))
	for _, f := range facts {
		out = append(out, WhisperMemoryView{
			ID: f.ID, Domain: f.Domain, Subcategory: f.Subcategory,
			Subject: f.Subject, Summary: f.Summary, Weight: f.Weight,
			Confidence: f.Confidence, Tier: f.Tier, Status: f.Status,
			UpdatedAt: f.UpdatedAt,
		})
	}
	return out
}

// GaeaMemoryHubOverview 返回记忆中枢聚合总览：各库条目数 + 最近更新时间。
func (a *App) GaeaMemoryHubOverview() MemoryHubOverview {
	ov := MemoryHubOverview{}

	if store, err := knowledge.Global().Store(); err == nil {
		ov.KnowledgeCount = len(store.List())
	}
	ov.ProfileCount = len(a.hubProfileStore().All())
	ov.OfficeCount = len(a.hubOfficeStore().List())
	ov.WhisperCount = len(whisperdb.LoadFactsFromDB(a.whisperDataRoot))

	// 最近更新时间：知识库条目带 UpdatedAt；办公 facts/画像的时间由各自
	// 后端维护（SQLite updated_at），前端按条目展示。
	var latest time.Time
	if store, err := knowledge.Global().Store(); err == nil {
		for _, s := range store.List() {
			if s.UpdatedAt.After(latest) {
				latest = s.UpdatedAt
			}
		}
	}
	if !latest.IsZero() {
		ov.LatestUpdated = latest.Format("2006-01-02 15:04")
	}
	return ov
}
