package snapshot

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/types"
)

// Store 快照存储管理器
// 快照目录: scenes/<sceneID>.snapshots/<snapshotID>.json
type Store struct {
	dir string // chapters/NNN/scenes/
}

// NewStore 创建快照存储
func NewStore(sceneDir string) *Store {
	return &Store{dir: sceneDir}
}

// snapDir 返回特定场景的快照目录
func (s *Store) snapDir(sceneID string) string {
	return filepath.Join(s.dir, sceneID+".snapshots")
}

// ensureSnapDir 确保快照目录存在
func (s *Store) ensureSnapDir(sceneID string) error {
	return os.MkdirAll(s.snapDir(sceneID), 0755)
}

// ── 核心操作 ────────────────────────────────────────────────

// Capture 创建场景当前状态的快照
// content: 当前正文，label: 可选标签，trigger: 触发原因
func (s *Store) Capture(sceneID, content, label, trigger string) (*types.Snapshot, error) {
	if err := s.ensureSnapDir(sceneID); err != nil {
		return nil, err
	}

	// 获取上一个快照用于 diff
	prevSnaps, err := s.List(sceneID)
	if err != nil {
		return nil, err
	}

	var prevContent string
	if len(prevSnaps) > 0 {
		prevContent, err = s.Rebuild(sceneID, prevSnaps)
		if err != nil {
			prevContent = "" // 重建失败则视为无前序内容
		}
	}

	// 计算行级 diff
	diffLines := computeDiff(prevContent, content)

	now := time.Now()
	id := fmt.Sprintf("%d", now.UnixNano())

	snap := types.Snapshot{
		ID:        id,
		SceneID:   sceneID,
		Timestamp: now,
		Label:     label,
		Trigger:   trigger,
		DiffLines: diffLines,
		WordCount: len([]rune(content)),
	}

	// 写入快照文件
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return nil, err
	}
	path := filepath.Join(s.snapDir(sceneID), id+".json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return nil, err
	}

	return &snap, nil
}

// List 列出场景的所有快照（按时间排序）
func (s *Store) List(sceneID string) ([]types.Snapshot, error) {
	dir := s.snapDir(sceneID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []types.Snapshot{}, nil
		}
		return nil, err
	}

	var snaps []types.Snapshot
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var snap types.Snapshot
		if json.Unmarshal(data, &snap) == nil {
			snaps = append(snaps, snap)
		}
	}

	sort.Slice(snaps, func(i, j int) bool {
		return snaps[i].Timestamp.Before(snaps[j].Timestamp)
	})

	return snaps, nil
}

// Restore 恢复到指定快照的内容
func (s *Store) Restore(snapshotID, sceneID string) (string, error) {
	snaps, err := s.List(sceneID)
	if err != nil {
		return "", err
	}

	// 找到目标快照在列表中的位置
	targetIdx := -1
	for i, snap := range snaps {
		if snap.ID == snapshotID {
			targetIdx = i
			break
		}
	}
	if targetIdx < 0 {
		return "", fmt.Errorf("快照 %s 不存在", snapshotID)
	}

	return s.Rebuild(sceneID, snaps[:targetIdx+1])
}

// Rebuild 从快照链重建完整内容
func (s *Store) Rebuild(sceneID string, snaps []types.Snapshot) (string, error) {
	content := ""
	for _, snap := range snaps {
		content = applyDiff(content, snap.DiffLines)
	}
	return content, nil
}

// Diff 比较两个快照之间的差异
func (s *Store) Diff(sceneID, fromID, toID string) ([]types.DiffLine, error) {
	fromContent, err := s.Restore(fromID, sceneID)
	if err != nil {
		return nil, fmt.Errorf("读取 from 快照失败: %w", err)
	}
	toContent, err := s.Restore(toID, sceneID)
	if err != nil {
		return nil, fmt.Errorf("读取 to 快照失败: %w", err)
	}
	return computeDiff(fromContent, toContent), nil
}

// ── diff 算法 ───────────────────────────────────────────────

// computeDiff 计算简单的行级 unified diff
// 返回的行序列可以用于重建内容：same/add 行组成新内容，del 行被排除
func computeDiff(oldContent, newContent string) []types.DiffLine {
	oldLines := splitLines(oldContent)
	newLines := splitLines(newContent)

	var result []types.DiffLine

	oi, ni := 0, 0
	for oi < len(oldLines) && ni < len(newLines) {
		if oldLines[oi] == newLines[ni] {
			result = append(result, types.DiffLine{
				Type: "same", Content: newLines[ni], LineNum: ni + 1,
			})
			oi++
			ni++
		} else {
			// 前向搜索：在新内容中找旧行（匹配窗口 = 3 行）
			found := false
			for look := ni + 1; look < ni+4 && look < len(newLines); look++ {
				if newLines[look] == oldLines[oi] {
					// 中间的行 = 新增
					for ; ni < look; ni++ {
						result = append(result, types.DiffLine{
							Type: "add", Content: newLines[ni], LineNum: ni + 1,
						})
					}
					found = true
					break
				}
			}
			if !found {
				// 旧行被删除
				result = append(result, types.DiffLine{
					Type: "del", Content: oldLines[oi], LineNum: oi + 1,
				})
				oi++
				// 同时把新行当作替换（add）
				if ni < len(newLines) {
					result = append(result, types.DiffLine{
						Type: "add", Content: newLines[ni], LineNum: ni + 1,
					})
					ni++
				}
			}
		}
	}

	// 剩余旧行 → 删除
	for oi < len(oldLines) {
		result = append(result, types.DiffLine{
			Type: "del", Content: oldLines[oi], LineNum: oi + 1,
		})
		oi++
	}

	// 剩余新行 → 新增
	for ni < len(newLines) {
		result = append(result, types.DiffLine{
			Type: "add", Content: newLines[ni], LineNum: ni + 1,
		})
		ni++
	}

	return result
}

// applyDiff 将 diff 应用到旧内容上生成新内容
func applyDiff(oldContent string, diffs []types.DiffLine) string {
	oldLines := splitLines(oldContent)

	var result []string
	oi := 0

	for _, dl := range diffs {
		switch dl.Type {
		case "same":
			if oi < len(oldLines) && oldLines[oi] == dl.Content {
				result = append(result, dl.Content)
				oi++
			} else {
				result = append(result, dl.Content)
			}
		case "add":
			result = append(result, dl.Content)
		case "del":
			if oi < len(oldLines) && oldLines[oi] == dl.Content {
				oi++
			}
		}
	}

	// 追加 diff 未覆盖的旧行
	for oi < len(oldLines) {
		result = append(result, oldLines[oi])
		oi++
	}

	return strings.Join(result, "\n")
}

// splitLines 按行分割，空字符串返回空 slice（不是 [""]）
func splitLines(s string) []string {
	if s == "" {
		return []string{}
	}
	return strings.Split(s, "\n")
}
