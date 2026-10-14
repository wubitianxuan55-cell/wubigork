package memory

// ── 记忆冲突比对与失效（阶段九 9.3）──────────────────────────
//
// auto-dream 建议入库（作者点接受）时与既有记忆比对，两档保守：
//
//	Confirmed（确定性信号，自动失效）：Title 规范化后与既有条目完全相同且
//	  Name 不同——做梦对同一主题重复提炼的典型实态。自动置 invalid_at（原文
//	  保留可查可恢复，绝不删除），返回消息人可读告知。
//	Suspected（仅提示，不改状态）：BM25 相似 top 命中（排除同名/同题/已失效）
//	  ——错误失效一条好记忆的代价 > 漏掉一条矛盾记忆（宁漏勿误），语义裁决
//	  归作者（面板可见，Restore 可恢复）。
//
// SubjectKey 单值冲突走既有撞键拒绝（Save 内，v4.378）——那是「revision vs
// contradiction」的显式路径，与本文件的语义冲突检测互补。

import (
	"fmt"
	"strings"

	"github.com/gaea/gaea/internal/gaea/bm25"
)

// ConflictReport 冲突比对结果（人可读消息的素材）。
type ConflictReport struct {
	// Confirmed 同题旧记忆（已自动失效，原文保留）。
	Confirmed []string
	// Suspected BM25 相似旧记忆（仅提示，状态未动）。
	Suspected []SearchMatch
}

// DetectConflicts 入库后冲突比对：找出与新条目同题的旧记忆（Confirmed）与
// BM25 相似的旧记忆（Suspected，top≤3）。已失效（InvalidAt 非空）与同名条目
// 不参与（同名=Save 覆盖语义的修订；已失效=已裁决过）。
func (s Store) DetectConflicts(m Memory) ConflictReport {
	rep := ConflictReport{}
	newTitle := normalizeTitle(m.Title)
	name := slug(m.Name)

	var candidates []Memory
	for _, e := range s.List() {
		if e.Name == name || !e.InvalidAt.IsZero() {
			continue
		}
		if normalizeTitle(e.Title) == newTitle && newTitle != "" {
			rep.Confirmed = append(rep.Confirmed, e.Name)
			continue
		}
		candidates = append(candidates, e)
	}
	if len(candidates) == 0 {
		return rep
	}
	// BM25 疑似相似：用新条目全文做查询，top≤3（分数>0 才算命中）。
	idx := (&Store{Dir: s.Dir, GlobalDir: s.GlobalDir}).buildIndexOver(candidates)
	if idx == nil {
		return rep
	}
	for _, hit := range idx.Search(strings.TrimSpace(m.Title + " " + m.Description + " " + m.Body)) {
		if hit.Score <= 0 {
			continue
		}
		if hit.Name == name {
			continue
		}
		rep.Suspected = append(rep.Suspected, hit)
		if len(rep.Suspected) >= 3 {
			break
		}
	}
	return rep
}

// Report 人可读汇总（接受动作的返回消息拼接用）。
func (r ConflictReport) Report() string {
	var b strings.Builder
	if len(r.Confirmed) > 0 {
		fmt.Fprintf(&b, "同题旧记忆已自动标记失效（原文保留，可恢复）：%s", strings.Join(r.Confirmed, "、"))
	}
	for _, sm := range r.Suspected {
		if b.Len() > 0 {
			b.WriteString("；")
		}
		fmt.Fprintf(&b, "疑似相关旧记忆「%s」（%s）——请裁决是否需要归档", sm.Name, sm.Preview)
	}
	return b.String()
}

// normalizeTitle 标题规范化：去首尾空白+折叠内部空白+小写（同题判定的唯一口径）。
func normalizeTitle(t string) string {
	return strings.Join(strings.Fields(strings.ToLower(t)), " ")
}

// buildIndexOver 对给定候选集建 BM25 索引（DetectConflicts 专用——BuildSearchIndex
// 走全库+docs，这里只要候选子集）。
func (s Store) buildIndexOver(candidates []Memory) *SearchIndex {
	if len(candidates) == 0 {
		return nil
	}
	idx := &SearchIndex{
		previews: make(map[string]string, len(candidates)),
		kinds:    make(map[string]Kind, len(candidates)),
	}
	var docs []bm25.Doc
	for _, m := range candidates {
		text := strings.ToLower(m.Title + " " + m.Description + " " + m.Body)
		idx.addDoc(&docs, m.Name, text, m.Title, m.Description, m.Kind)
	}
	idx.ranker = bm25.NewRankerOpts(docs, memTokenizeOpts, bm25.DefaultParams)
	return idx
}
