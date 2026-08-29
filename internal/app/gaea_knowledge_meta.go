package app

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/knowledge"
	"github.com/gaea/gaea/internal/gaea/knowledgeimport"
	"github.com/gaea/gaea/internal/gaea/spaces"
)

// knowledgeHistoryDBOverride 测试注入隔离的历史库（避免触碰真实 Hephaestus.db）。
var knowledgeHistoryDBOverride *sql.DB

// SetKnowledgeHistoryDBForTest 注入隔离的知识版本历史库（测试用）。
func SetKnowledgeHistoryDBForTest(gdb *sql.DB) { knowledgeHistoryDBOverride = gdb }

func knowledgeHistoryDB() *sql.DB {
	if knowledgeHistoryDBOverride != nil {
		return knowledgeHistoryDBOverride
	}
	return db.GetDatabase(config.MemoryUserDir())
}

// KnowledgeHistoryView 是一条知识版本历史快照。
type KnowledgeHistoryView struct {
	Name       string   `json:"name"`
	Title      string   `json:"title"`
	Version    int      `json:"version"`
	Category   string   `json:"category"`
	Phase      string   `json:"phase"`
	Discipline string   `json:"discipline"`
	Tags       []string `json:"tags"`
	Status     string   `json:"status"`
	Author     string   `json:"author"`
	Reviewer   string   `json:"reviewer"`
	Source     string   `json:"source"`
	Body       string   `json:"body"`
	ChangedAt  string   `json:"changedAt"`
	Note       string   `json:"note"`
}

// SimilarView 是查重命中的相似条目。
type SimilarView struct {
	Name  string  `json:"name"`
	Title string  `json:"title"`
	Score float64 `json:"score"`
}

// saveKnowledgeVersioned 保存知识条目并维护版本历史：内容变化时把旧快照写入
// knowledge_history（版本号 +1），供版本回溯与审核。
func saveKnowledgeVersioned(store *knowledge.Store, e knowledge.Entry) error {
	existing, err := store.Get(e.Name)
	changed := true
	if err == nil && existing != nil {
		if strings.TrimSpace(existing.Body) == strings.TrimSpace(e.Body) &&
			existing.Title == e.Title && existing.Category == e.Category {
			changed = false
		}
		if e.Version <= existing.Version {
			e.Version = existing.Version + 1
		}
		if changed {
			_ = addKnowledgeHistory(existing, "内容更新")
		}
	} else if e.Version <= 0 {
		e.Version = 1
	}
	if e.CreatedAt.IsZero() && existing != nil {
		e.CreatedAt = existing.CreatedAt
	}
	return store.Save(e)
}

// addKnowledgeHistory 把一条旧快照写入版本历史。
func addKnowledgeHistory(e *knowledge.Entry, note string) error {
	gdb := knowledgeHistoryDB()
	if gdb == nil {
		return nil
	}
	tags, _ := json.Marshal(e.Tags)
	_, err := gdb.Exec(`
INSERT INTO knowledge_history(name,title,version,category,phase,discipline,tags,status,author,reviewer,source,body,changed_at,note)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		e.Name, e.Title, e.Version, e.Category, e.Phase, e.Discipline, string(tags),
		e.Status, e.Author, e.Reviewer, e.Source, e.Body,
		time.Now().UTC().Format(time.RFC3339), note)
	return err
}

// GaeaKnowledgeHistory 返回某条目的版本历史（新→旧）。
func (a *App) GaeaKnowledgeHistory(name string) []KnowledgeHistoryView {
	gdb := knowledgeHistoryDB()
	if gdb == nil {
		return nil
	}
	rows, err := gdb.Query(`SELECT name,title,version,category,phase,discipline,tags,status,author,reviewer,source,body,changed_at,note
FROM knowledge_history WHERE name=? ORDER BY changed_at DESC, id DESC LIMIT 30`, name)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []KnowledgeHistoryView
	for rows.Next() {
		var v KnowledgeHistoryView
		var tags string
		if err := rows.Scan(&v.Name, &v.Title, &v.Version, &v.Category, &v.Phase, &v.Discipline,
			&tags, &v.Status, &v.Author, &v.Reviewer, &v.Source, &v.Body, &v.ChangedAt, &v.Note); err != nil {
			continue
		}
		_ = json.Unmarshal([]byte(tags), &v.Tags)
		out = append(out, v)
	}
	return out
}

// GaeaKnowledgeFindSimilar 查重：返回与 title 模糊相似（≥0.65）的既有条目。
func (a *App) GaeaKnowledgeFindSimilar(title string) []SimilarView {
	store, err := a.hubKnowledgeStore()
	if err != nil {
		return nil
	}
	hits := knowledgeimport.FindSimilar(store, title, 0.65)
	out := make([]SimilarView, 0, len(hits))
	for _, h := range hits {
		out = append(out, SimilarView{Name: h.Name, Title: h.Title, Score: h.Score})
	}
	return out
}

// GaeaKnowledgeExport 批量导出知识库为 Markdown 文件（frontmatter + 正文），
// 返回导出条数。dir 为空时导出到 .gaea/exports/knowledge-<日期>。
func (a *App) GaeaKnowledgeExport(dir string) (int, error) {
	store, err := a.hubKnowledgeStore()
	if err != nil {
		return 0, err
	}
	if strings.TrimSpace(dir) == "" {
		// S4：knowledge 域仅 work 侧读写（设计 §5），导出目录固定 work 现状
		// 路径，不随会话空间分区。
		dir = filepath.Join(spaces.ExportsDir(gaeaCwd(), spaces.SpaceWork),
			"knowledge-"+time.Now().Format("20060102"))
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return 0, err
	}
	entries := store.ReadAll()
	n := 0
	for _, e := range entries {
		content := knowledge.RenderFrontmatter(e) + e.Body
		if err := os.WriteFile(filepath.Join(dir, knowledge.FileName(e)), []byte(content), 0o644); err != nil {
			return n, fmt.Errorf("导出 %s 失败: %w", e.Name, err)
		}
		n++
	}
	return n, nil
}

// GaeaKnowledgeReview 审核流：approve=true 把条目置为「现行」并记录审核人，
// false 驳回（保留原状态）；动作留档到版本历史。单用户本地工具：审核即
// 草稿 → 现行 的确认发布动作，reviewer 字段可编辑。
func (a *App) GaeaKnowledgeReview(name string, approve bool, reviewer string) error {
	store, err := a.hubKnowledgeStore()
	if err != nil {
		return err
	}
	e, err := store.Get(name)
	if err != nil {
		return err
	}
	note := "审核驳回"
	if approve {
		e.Status = "现行"
		note = "审核通过"
		if strings.TrimSpace(reviewer) != "" {
			e.Reviewer = reviewer
		}
	}
	_ = addKnowledgeHistory(e, note)
	return saveKnowledgeVersioned(store, *e)
}

// GaeaKnowledgeMerge 把 sourceNames 合并进 targetName：标签取并集、来源合并、
// 各来源与目标各留一条历史（合并至/合并自）后删除来源条目；返回目标条目名。
func (a *App) GaeaKnowledgeMerge(targetName string, sourceNames []string) (string, error) {
	store, err := a.hubKnowledgeStore()
	if err != nil {
		return "", err
	}
	target, err := store.Get(targetName)
	if err != nil {
		return "", err
	}
	tagSet := make(map[string]bool, len(target.Tags))
	for _, t := range target.Tags {
		tagSet[t] = true
	}
	var srcTitles []string
	for _, sn := range sourceNames {
		if sn == "" || sn == targetName {
			continue
		}
		src, err := store.Get(sn)
		if err != nil {
			return "", err
		}
		for _, t := range src.Tags {
			tagSet[t] = true
		}
		if strings.TrimSpace(src.Source) != "" && !strings.Contains(target.Source, src.Source) {
			if strings.TrimSpace(target.Source) != "" {
				target.Source += "；"
			}
			target.Source += src.Source
		}
		srcTitles = append(srcTitles, src.Title)
		_ = addKnowledgeHistory(src, "合并至 "+target.Title)
		if err := store.Delete(sn); err != nil {
			return "", err
		}
	}
	if len(srcTitles) == 0 {
		return target.Name, nil
	}
	target.Tags = sortedTagSet(tagSet)
	_ = addKnowledgeHistory(target, "合并自 "+strings.Join(srcTitles, "、"))
	if err := saveKnowledgeVersioned(store, *target); err != nil {
		return "", err
	}
	return target.Name, nil
}

func sortedTagSet(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for t := range set {
		if strings.TrimSpace(t) != "" {
			out = append(out, t)
		}
	}
	sort.Strings(out)
	return out
}
