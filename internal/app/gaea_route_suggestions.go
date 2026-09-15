package app

// 路由学习建议（阶段七 7.1-2）：按「成本 × 质量」评分差生成功能域改绑建议，
// 建议制不自动改绑——采纳是用户显式动作（GaeaRouteSuggestionApply 内部走
// App.SetFeatureModel 既有链路，gaea/office 域即时重建语义保持）。
// 纯评分逻辑在 internal/routesuggest（零 IO 表驱动可测）；本文件只做数据组装
// 与建议状态持久化（<DataRoot>/route_suggestions.json，原子写）。
import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/modelengine"
	"github.com/gaea/gaea/internal/routesuggest"
)

// routeFeatures 建议覆盖的功能绑定键（whisper 并入 chat——featureModelKeys 同口径）。
var routeFeatures = []string{"chat", "novel", "office", "gaea", "characterlib", "routine", "sin"}

// RouteSuggestionsView 建议视图（Wails 绑定，模型中心「成本归因」tab 消费）。
type RouteSuggestionsView struct {
	GeneratedAt string                    `json:"generated_at"`
	Suggestions []routesuggest.Suggestion `json:"suggestions"`
}

// ── 建议状态持久化（采纳/忽略记录，ID 确定性→重算幂等）──

type routeSuggestionRecord struct {
	Status    string `json:"status"` // "ignored" | "applied"
	DecidedAt string `json:"decided_at"`
}

type routeSuggestionFile struct {
	Version int                              `json:"version"`
	Records map[string]routeSuggestionRecord `json:"records"`
}

func routeSuggestionsPath(dataRoot string) string {
	return filepath.Join(dataRoot, "route_suggestions.json")
}

// loadRouteSuggestionRecords 读建议状态；文件缺失/损坏一律回空表（建议可重算，状态丢得起）。
func loadRouteSuggestionRecords(dataRoot string) map[string]routeSuggestionRecord {
	b, err := os.ReadFile(routeSuggestionsPath(dataRoot))
	if err != nil {
		return map[string]routeSuggestionRecord{}
	}
	var f routeSuggestionFile
	if err := json.Unmarshal(b, &f); err != nil || f.Records == nil {
		return map[string]routeSuggestionRecord{}
	}
	return f.Records
}

// saveRouteSuggestionRecord 原子写一条状态（temp+rename，chapter_art_manifest 先例）。
// ID 在此二次校验（单一落盘咽喉）：非法 ID 不落盘。
func saveRouteSuggestionRecord(dataRoot, id, status string) error {
	if _, _, _, _, _, ok := routesuggest.ParseSuggestionID(id); !ok {
		return &appError{"建议 ID 无法解析: " + id}
	}
	recs := loadRouteSuggestionRecords(dataRoot)
	recs[id] = routeSuggestionRecord{Status: status, DecidedAt: time.Now().Format(time.RFC3339)}
	b, err := json.MarshalIndent(routeSuggestionFile{Version: 1, Records: recs}, "", "  ")
	if err != nil {
		return err
	}
	p := routeSuggestionsPath(dataRoot)
	dir := filepath.Dir(p)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "route-suggest-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, p); err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
}

// GaeaRouteSuggestions 现算建议（合并忽略/已采纳状态后返回）。
// 数据口径：候选=全部启用引擎的 llm 模型（本地云端同池，判据④）；质量=既调用量
// 统计（成功率/均时长/单次成本=窗口估算总额/调用数）；当前绑定=routeModel 解析的
// 生效值（只读调用，路由链零改动）。
func (a *App) GaeaRouteSuggestions() (RouteSuggestionsView, error) {
	stats := a.GetModelCallStats()

	// (engine|model) → 质量聚合。
	type quality struct {
		rate    float64
		samples int64
		avgMs   int64
		costPer float64
	}
	qs := make(map[string]quality, len(stats.PerModel))
	for _, pm := range stats.PerModel {
		if pm.CallCount <= 0 {
			continue
		}
		qs[pm.EngineID+"|"+pm.Model] = quality{
			rate:    float64(pm.SuccessCount) / float64(pm.CallCount),
			samples: pm.CallCount,
			avgMs:   pm.TotalDurationMs / pm.CallCount,
			costPer: pm.EstimatedCost / float64(pm.CallCount),
		}
	}

	// 候选池：启用引擎 × llm 模型（Kind 缺省回退 ClassifyModelKind）。
	var cands []routesuggest.Candidate
	if a.engineMgr != nil {
		for _, e := range a.engineMgr.GetEngines() {
			if !e.Enabled {
				continue
			}
			for _, m := range e.Models {
				kind := m.Kind
				if kind == "" {
					kind = modelengine.ClassifyModelKind(e.Type, m.ID)
				}
				if kind != "llm" {
					continue
				}
				c := routesuggest.Candidate{EngineID: e.ID, Model: m.ID, IsLocal: e.Type.IsLocal()}
				if q, ok := qs[e.ID+"|"+m.ID]; ok {
					rate := q.rate
					c.SuccessRate = &rate
					c.Samples = q.samples
					c.AvgMs = q.avgMs
					c.CostCNY = q.costPer
				}
				cands = append(cands, c)
			}
		}
	}

	// 功能绑定：routeModel 解析生效值（engine/model 空则跳过——无可用路由不评）。
	var fbs []routesuggest.FeatureBinding
	for _, f := range routeFeatures {
		eng, model, _ := a.routeModel(f)
		if eng == "" || model == "" {
			continue
		}
		fbs = append(fbs, routesuggest.FeatureBinding{Feature: f, EngineID: eng, Model: model})
	}

	// 忽略与已采纳都静默（采纳后绑定已变，ID 随之漂移，双过滤兜底重算窗口竞态）。
	recs := loadRouteSuggestionRecords(config.DataRoot())
	silenced := make(map[string]struct{}, len(recs))
	for id, r := range recs {
		if r.Status == "ignored" || r.Status == "applied" {
			silenced[id] = struct{}{}
		}
	}

	out := routesuggest.Suggest(routesuggest.Input{Features: fbs, Candidates: cands}, silenced)
	if out == nil {
		out = []routesuggest.Suggestion{}
	}
	return RouteSuggestionsView{GeneratedAt: time.Now().Format(time.RFC3339), Suggestions: out}, nil
}

// GaeaRouteSuggestionApply 采纳建议：显式用户动作，走 SetFeatureModel 既有链路
// （校验引擎存在/启用/模型在列，gaea 域即时 applyOfficeFeatureModel）。改绑成功后
// 状态落盘失败只告警不回滚——改绑本身已生效。
func (a *App) GaeaRouteSuggestionApply(id string) error {
	feature, _, _, toEng, toModel, ok := routesuggest.ParseSuggestionID(id)
	if !ok {
		return &appError{"建议 ID 无法解析: " + id}
	}
	if _, _, valid := featureModelKeys(feature); !valid {
		return &appError{"未知功能: " + feature}
	}
	if err := a.SetFeatureModel(feature, toEng, toModel); err != nil {
		return fmt.Errorf("采纳失败（改绑被拒）: %w", err)
	}
	if err := saveRouteSuggestionRecord(config.DataRoot(), id, "applied"); err != nil {
		slog.Warn("路由建议采纳状态落盘失败", "error", err)
	}
	slog.Info("路由建议已采纳", "feature", feature, "engine", toEng, "model", toModel)
	return nil
}

// GaeaRouteSuggestionIgnore 忽略建议：被忽略 ID 不再出现（重算幂等）。
func (a *App) GaeaRouteSuggestionIgnore(id string) error {
	if _, _, _, _, _, ok := routesuggest.ParseSuggestionID(id); !ok {
		return &appError{"建议 ID 无法解析: " + id}
	}
	return saveRouteSuggestionRecord(config.DataRoot(), id, "ignored")
}
