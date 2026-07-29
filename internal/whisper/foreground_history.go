// Package whisper — foreground_history.go
// 100% 对齐 ackem memory/foregroundHistory.ts
// 前台检测历史：记录窗口场景 + 扫描生成候选长时习惯（简化版，无DB）

package whisper

import (
	"sync"
	"time"
)

// ForegroundRecord 前台检测记录
type ForegroundRecord struct {
	Title      string          `json:"title"`
	Scene      ForegroundScene `json:"scene"`
	DetectedAt int64           `json:"detectedAt"` // unix ms
}

// ForegroundHistory 前台历史存储（内存版）
type ForegroundHistory struct {
	mu      sync.RWMutex
	records []ForegroundRecord
}

// NewForegroundHistory 创建前台历史
func NewForegroundHistory() *ForegroundHistory {
	return &ForegroundHistory{}
}

// RecordForegroundDetection 记录一条前台检测
func (fh *ForegroundHistory) RecordForegroundDetection(title string, scene ForegroundScene) {
	if scene == SceneOther {
		return
	}
	fh.mu.Lock()
	defer fh.mu.Unlock()
	fh.records = append(fh.records, ForegroundRecord{
		Title: title, Scene: scene, DetectedAt: time.Now().UnixMilli(),
	})
	// 清理 28 天前记录
	cutoff := time.Now().Add(-28 * 24 * time.Hour).UnixMilli()
	var kept []ForegroundRecord
	for _, r := range fh.records {
		if r.DetectedAt >= cutoff {
			kept = append(kept, r)
		}
	}
	fh.records = kept
}

// CountForegroundRecords 统计前台记录数
func (fh *ForegroundHistory) CountForegroundRecords() int {
	fh.mu.RLock()
	defer fh.mu.RUnlock()
	return len(fh.records)
}

// GetRecentScenes 获取最近的前台场景（最近N条）
func (fh *ForegroundHistory) GetRecentScenes(n int) []ForegroundRecord {
	fh.mu.RLock()
	defer fh.mu.RUnlock()
	if n <= 0 || n > len(fh.records) {
		n = len(fh.records)
	}
	start := len(fh.records) - n
	result := make([]ForegroundRecord, n)
	copy(result, fh.records[start:])
	return result
}
