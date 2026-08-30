package app

// v4.9.1 做梦 2.0 第一刀「蒸馏真实合并」的 app 视图层：
//   - distillMergeViews 把 memory.DistillMergeCandidates 的纯检测结果映射为
//     面板卡片（补标题/时间等展示字段，稳定 ID）；
//   - GaeaAcceptMergeSuggestion 分发到 control.DistillMerge（锁内重算校验 +
//     归档较旧条 + Touch 较新条 + 审计）。
// 检测纪律（宁漏勿误）与空间红线见 memory/distill.go 顶部注释。

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/memory"
)

// MergeSuggestionView 是蒸馏合并候选（记忆面板「建议」合并卡片）。
type MergeSuggestionView struct {
	ID               string `json:"id"`
	Keep             string `json:"keep"` // 保留条 name（较新）
	Archive          string `json:"archive"`
	KeepTitle        string `json:"keepTitle,omitempty"`
	ArchiveTitle     string `json:"archiveTitle,omitempty"`
	KeepUpdatedAt    string `json:"keepUpdatedAt,omitempty"`
	ArchiveUpdatedAt string `json:"archiveUpdatedAt,omitempty"`
	Reason           string `json:"reason"`
}

// distillMergeViews 把检测结果映射为视图（name→记忆 索引补展示字段）。
func distillMergeViews(ms []memory.Memory) []MergeSuggestionView {
	byName := make(map[string]memory.Memory, len(ms))
	for _, m := range ms {
		byName[m.Name] = m
	}
	cands := memory.DistillMergeCandidates(ms)
	out := make([]MergeSuggestionView, 0, len(cands))
	for _, c := range cands {
		v := MergeSuggestionView{
			ID:      distillMergeID(c.Keep, c.Archive),
			Keep:    c.Keep,
			Archive: c.Archive,
			Reason:  c.Reason,
		}
		if k, ok := byName[c.Keep]; ok {
			v.KeepTitle = k.Title
			v.KeepUpdatedAt = formatDistillTime(k.UpdatedAt)
		}
		if o, ok := byName[c.Archive]; ok {
			v.ArchiveTitle = o.Title
			v.ArchiveUpdatedAt = formatDistillTime(o.UpdatedAt)
		}
		out = append(out, v)
	}
	return out
}

// distillMergeID 候选稳定 ID（keep+archive 决定，刷新面板不闪烁）。
func distillMergeID(keep, archive string) string {
	sum := sha256.Sum256([]byte(keep + "\x00" + archive))
	return "merge-" + hex.EncodeToString(sum[:6])
}

func formatDistillTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02 15:04")
}

// GaeaAcceptMergeSuggestion 执行一条蒸馏合并：control.DistillMerge 内部锁内
// 重算候选校验配对（不在候选内一律拒绝），归档较旧条（可逆）+ Touch 较新条。
func (a *App) GaeaAcceptMergeSuggestion(keep, archive string) (string, error) {
	keep = strings.TrimSpace(keep)
	archive = strings.TrimSpace(archive)
	if keep == "" || archive == "" {
		return "", fmt.Errorf("keep/archive 不能为空")
	}
	ctrl := gaeaCtrl()
	if ctrl == nil {
		return "", fmt.Errorf("办公引擎未初始化")
	}
	return ctrl.DistillMerge(keep, archive)
}
