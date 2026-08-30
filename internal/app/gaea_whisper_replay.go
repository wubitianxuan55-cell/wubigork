package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/whisper"
	whisperdb "github.com/gaea/gaea/internal/whisper/db/repos"
)

// WhisperReplayLine 记忆回放中的一行原始对话。
type WhisperReplayLine struct {
	TurnIndex int    `json:"turnIndex"`
	Role      string `json:"role"` // user / assistant
	Text      string `json:"text"`
}

// WhisperEpisodeReplayView 情节记忆回放视图：情节元数据 + 原始对话回放。
type WhisperEpisodeReplayView struct {
	ID                 string              `json:"id"`
	Summary            string              `json:"summary"`
	DominantEmotion    string              `json:"dominantEmotion"`
	EmotionalIntensity float64             `json:"emotionalIntensity"`
	Keywords           []string            `json:"keywords"`
	CreatedAt          time.Time           `json:"createdAt"`
	SourceSessionID    string              `json:"sourceSessionId"`
	StartTurn          int                 `json:"startTurn"`
	EndTurn            int                 `json:"endTurn"`
	Dialogue           []WhisperReplayLine `json:"dialogue"`
	Replayable         bool                `json:"replayable"`
}

// GaeaWhisperEpisodeReplay 返回一条情节记忆的原始对话回放（play 空间记忆回放，
// 审计 §C 乐园做深欠账收口）。从 hermes.db 读取情节，再按 SourceSessionID +
// [StartTurn, EndTurn] 从 chat_history 重建原对话——确定性实现，不调用 LLM。
// chat_history 裁剪至最近 2000 行，过旧情节可能无原始对话（Replayable=false，
// 前端回退为仅展示记忆摘要）。只读查询，无副作用。
func (a *App) GaeaWhisperEpisodeReplay(episodeID string) (WhisperEpisodeReplayView, error) {
	if episodeID == "" {
		return WhisperEpisodeReplayView{}, fmt.Errorf("episodeID 为空")
	}
	eps, err := whisperdb.LoadEpisodesFromDB(a.whisperDataRoot)
	if err != nil {
		return WhisperEpisodeReplayView{}, fmt.Errorf("读取情节库失败: %w", err)
	}
	for i := range eps {
		if eps[i].ID == episodeID {
			return a.buildEpisodeReplay(&eps[i])
		}
	}
	return WhisperEpisodeReplayView{}, fmt.Errorf("情节不存在: %s", episodeID)
}

// buildEpisodeReplay 按情节重建原始对话（情节回放与锚点回放共用）。
func (a *App) buildEpisodeReplay(ep *whisper.Episode) (WhisperEpisodeReplayView, error) {
	view := WhisperEpisodeReplayView{
		ID:                 ep.ID,
		Summary:            ep.Summary,
		DominantEmotion:    ep.DominantEmotion,
		EmotionalIntensity: ep.EmotionalIntensity,
		Keywords:           nonNilStrings(ep.Keywords),
		CreatedAt:          ep.CreatedAt,
		SourceSessionID:    ep.SourceSessionID,
		StartTurn:          ep.StartTurn,
		EndTurn:            ep.EndTurn,
		Dialogue:           []WhisperReplayLine{},
	}

	rows, err := whisperdb.LoadChatHistoryFromDB(a.whisperDataRoot, ep.SourceSessionID)
	if err != nil {
		return WhisperEpisodeReplayView{}, fmt.Errorf("读取会话历史失败: %w", err)
	}
	for _, r := range rows {
		ti, ok := r["turnIndex"].(float64)
		if !ok {
			continue
		}
		turn := int(ti)
		if turn < ep.StartTurn || turn > ep.EndTurn {
			continue
		}
		if ut, _ := r["userText"].(string); strings.TrimSpace(ut) != "" {
			view.Dialogue = append(view.Dialogue, WhisperReplayLine{TurnIndex: turn, Role: "user", Text: ut})
		}
		if at, _ := r["assistantText"].(string); strings.TrimSpace(at) != "" {
			view.Dialogue = append(view.Dialogue, WhisperReplayLine{TurnIndex: turn, Role: "assistant", Text: at})
		}
	}
	view.Replayable = len(view.Dialogue) > 0
	return view, nil
}
