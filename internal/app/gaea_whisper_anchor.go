package app

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gaea/gaea/internal/whisper"
	whisperdb "github.com/gaea/gaea/internal/whisper/db/repos"
)

// WhisperAnchorView 轻语时间锚点只读视图（play 空间「纪念日」）。
type WhisperAnchorView struct {
	ID                 string   `json:"id"`
	AnchorDate         string   `json:"anchorDate"`
	AnchorType         string   `json:"anchorType"`
	RecurrenceRule     string   `json:"recurrenceRule"`
	Domain             string   `json:"domain"`
	Summary            string   `json:"summary"`
	EmotionalValence   float64  `json:"emotionalValence"`
	EmotionalIntensity float64  `json:"emotionalIntensity"`
	LinkedFactIDs      []string `json:"linkedFactIds"`
}

// WhisperAnchorReplayView 时间锚点回放视图：锚点信息 + 关联事实摘要 +
// 命中的情节原始对话回放（未命中情节时 Replayable=false）。
type WhisperAnchorReplayView struct {
	AnchorID            string                    `json:"anchorId"`
	AnchorDate          string                    `json:"anchorDate"`
	AnchorType          string                    `json:"anchorType"`
	Domain              string                    `json:"domain"`
	Summary             string                    `json:"summary"`
	EmotionalValence    float64                   `json:"emotionalValence"`
	EmotionalIntensity  float64                   `json:"emotionalIntensity"`
	LinkedFactSummaries []string                  `json:"linkedFactSummaries"`
	EpisodeReplay       *WhisperEpisodeReplayView `json:"episodeReplay,omitempty"`
	Replayable          bool                      `json:"replayable"`
}

// GaeaWhisperAnchors 返回轻语（hermes.db）时间锚点只读列表（play 空间纪念日）。
// 按 AnchorDate 降序（YYYY-MM-DD 字典序即时间序）；数据随 companion_state
// 三表落库（v4.3a 持久化闭环）。
func (a *App) GaeaWhisperAnchors() []WhisperAnchorView {
	anchors, err := loadWhisperAnchors(a.whisperDataRoot)
	if err != nil {
		return []WhisperAnchorView{}
	}
	out := make([]WhisperAnchorView, 0, len(anchors))
	for _, an := range anchors {
		out = append(out, WhisperAnchorView{
			ID: an.ID, AnchorDate: an.AnchorDate, AnchorType: string(an.AnchorType),
			RecurrenceRule: an.RecurrenceRule, Domain: an.Domain, Summary: an.Summary,
			EmotionalValence: an.EmotionalValence, EmotionalIntensity: an.EmotionalIntensity,
			LinkedFactIDs: nonNilStrings(an.LinkedFactIDs),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].AnchorDate > out[j].AnchorDate })
	return out
}

// GaeaWhisperAnchorReplay 按时间锚点回放「重访那一天」：锚点 → linked 事实的
// (session, turn) → 覆盖该轮次的情节 → 复用 buildEpisodeReplay 重建原始对话。
// 未命中情节时 Replayable=false，仅回退锚点摘要 + 关联事实摘要。只读、无副作用。
func (a *App) GaeaWhisperAnchorReplay(anchorID string) (WhisperAnchorReplayView, error) {
	if anchorID == "" {
		return WhisperAnchorReplayView{}, fmt.Errorf("anchorID 为空")
	}
	anchors, err := loadWhisperAnchors(a.whisperDataRoot)
	if err != nil {
		return WhisperAnchorReplayView{}, fmt.Errorf("读取时间锚点失败: %w", err)
	}
	var anchor *whisper.TemporalAnchor
	for i := range anchors {
		if anchors[i].ID == anchorID {
			anchor = &anchors[i]
			break
		}
	}
	if anchor == nil {
		return WhisperAnchorReplayView{}, fmt.Errorf("时间锚点不存在: %s", anchorID)
	}

	view := WhisperAnchorReplayView{
		AnchorID:            anchor.ID,
		AnchorDate:          anchor.AnchorDate,
		AnchorType:          string(anchor.AnchorType),
		Domain:              anchor.Domain,
		Summary:             anchor.Summary,
		EmotionalValence:    anchor.EmotionalValence,
		EmotionalIntensity:  anchor.EmotionalIntensity,
		LinkedFactSummaries: []string{},
	}

	factByID := map[string]whisper.MemoryFact{}
	for _, f := range whisperdb.LoadFactsFromDB(a.whisperDataRoot) {
		factByID[f.ID] = f
	}
	for _, fid := range anchor.LinkedFactIDs {
		if f, ok := factByID[fid]; ok && strings.TrimSpace(f.Summary) != "" {
			view.LinkedFactSummaries = append(view.LinkedFactSummaries, f.Summary)
		}
	}

	episodes, err := whisperdb.LoadEpisodesFromDB(a.whisperDataRoot)
	if err != nil {
		return WhisperAnchorReplayView{}, fmt.Errorf("读取情节库失败: %w", err)
	}
	best, bestHits := pickBestEpisodeForAnchor(episodes, anchor, factByID)
	if best != nil {
		replay, err := a.buildEpisodeReplay(best)
		if err != nil {
			return WhisperAnchorReplayView{}, err
		}
		view.EpisodeReplay = &replay
		view.Replayable = replay.Replayable
		_ = bestHits
	}
	return view, nil
}

// pickBestEpisodeForAnchor 在情节中挑选覆盖最多锚点关联事实轮次的一条；
// 平局取情绪强度更高者，再平局取更早创建（原始记忆优先）。无命中返回 nil。
func pickBestEpisodeForAnchor(episodes []whisper.Episode, anchor *whisper.TemporalAnchor, factByID map[string]whisper.MemoryFact) (*whisper.Episode, int) {
	var best *whisper.Episode
	bestHits := 0
	for i := range episodes {
		ep := &episodes[i]
		hits := 0
		for _, fid := range anchor.LinkedFactIDs {
			f, ok := factByID[fid]
			if !ok || f.SourceSessionID != ep.SourceSessionID {
				continue
			}
			if f.SourceTurnIndex >= ep.StartTurn && f.SourceTurnIndex <= ep.EndTurn {
				hits++
			}
		}
		if hits <= 0 {
			continue
		}
		if best == nil ||
			hits > bestHits ||
			(hits == bestHits && ep.EmotionalIntensity > best.EmotionalIntensity) ||
			(hits == bestHits && ep.EmotionalIntensity == best.EmotionalIntensity && ep.CreatedAt.Before(best.CreatedAt)) {
			best = ep
			bestHits = hits
		}
	}
	return best, bestHits
}

// loadWhisperAnchors 打开 hermes.db 锚点仓库并列出全部锚点。
func loadWhisperAnchors(dataRoot string) ([]whisper.TemporalAnchor, error) {
	repo, err := whisperdb.OpenTemporalAnchorsRepo(dataRoot)
	if err != nil {
		return nil, err
	}
	return repo.List()
}
