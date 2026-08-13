package app

// Herdsman 最近操作（P5 长任务可见性）：读取 skill-operations.json，
// 展示 Herdsman 异步任务（生图/TTS/下载等）的状态、进度与产物。

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// HerdsmanOperation 单条异步操作。
type HerdsmanOperation struct {
	ID          string `json:"id"`
	Kind        string `json:"kind"`
	Model       string `json:"model"`
	Status      string `json:"status"`
	Stage       string `json:"stage,omitempty"`
	Progress    int    `json:"progress"`
	Artifacts   int    `json:"artifacts"`
	CreatedAt   string `json:"created_at"`
	CompletedAt string `json:"completed_at,omitempty"`
}

// HerdsmanOperations 最近操作列表。
type HerdsmanOperations struct {
	Total  int                 `json:"total"`
	Items  []HerdsmanOperation `json:"items"`
	Source string              `json:"source"`
	Error  string              `json:"error,omitempty"`
}

type herdsmanRawOperation struct {
	ID          string `json:"id"`
	Kind        string `json:"kind"`
	Model       string `json:"model"`
	Status      string `json:"status"`
	Stage       string `json:"stage"`
	Progress    int    `json:"progress"`
	Artifacts   []any  `json:"artifacts"`
	CreatedAt   string `json:"created_at"`
	CompletedAt string `json:"completed_at"`
}

// parseHerdsmanOperations 解析 skill-operations.json（数组），按创建时间倒序。
func parseHerdsmanOperations(data []byte) ([]HerdsmanOperation, error) {
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	var raw []herdsmanRawOperation
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("解析 herdsman 操作记录失败: %w", err)
	}
	out := make([]HerdsmanOperation, 0, len(raw))
	for _, r := range raw {
		out = append(out, HerdsmanOperation{
			ID:          r.ID,
			Kind:        r.Kind,
			Model:       r.Model,
			Status:      r.Status,
			Stage:       r.Stage,
			Progress:    r.Progress,
			Artifacts:   len(r.Artifacts),
			CreatedAt:   r.CreatedAt,
			CompletedAt: r.CompletedAt,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt > out[j].CreatedAt })
	return out, nil
}

// HerdsmanOperations 返回 Herdsman 最近异步操作（最多 20 条）。
func (a *App) HerdsmanOperations() (HerdsmanOperations, error) {
	dir := herdsmanDataDir()
	if dir == "" {
		return HerdsmanOperations{Source: "herdsman-operations", Error: "无法定位 Herdsman 数据目录"}, fmt.Errorf("无法定位 Herdsman 数据目录")
	}
	data, err := os.ReadFile(filepath.Join(dir, "skill-operations.json"))
	if err != nil {
		return HerdsmanOperations{Source: "herdsman-operations", Error: err.Error()}, err
	}
	items, err := parseHerdsmanOperations(data)
	if err != nil {
		return HerdsmanOperations{Source: "herdsman-operations", Error: err.Error()}, err
	}
	if len(items) > 20 {
		items = items[:20]
	}
	return HerdsmanOperations{Total: len(items), Items: items, Source: "herdsman-operations"}, nil
}

// herdOpsKindLabel 供前端展示的 kind 中文化（保留英文原值亦可）。
func herdOpsKindLabel(kind string) string {
	switch strings.ToLower(kind) {
	case "image_generate":
		return "生图"
	case "model_start":
		return "模型启动"
	case "model_download":
		return "模型下载"
	case "speech":
		return "语音合成"
	case "tts":
		return "TTS"
	case "asr":
		return "语音识别"
	case "music":
		return "音乐生成"
	default:
		if kind == "" {
			return "任务"
		}
		return kind
	}
}
