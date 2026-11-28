package proposal

// index.go — 方案库的同题对照与索引体检 / 重建（2026-11-28 索引优化批）。
//
// 同题对照：几份技术标骨架相同，「污染土壤开挖短驳技术方案」这类标题路径在每份里
// 都有——读到某一节时能一键看到别的方案同一节怎么写的。
//
// 索引体检 / 重建：全文索引行与语义块是靠各写入口同步维护的派生数据，绕开产品直
// 连改库、早期版本没清干净都会留下孤儿行或缺行（实盘见过 3 条孤儿全文索引行）。
// 这里给一个只读的体检快照和一个显式的「修复并重建」。

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// ReindexBudget 重建语义索引的时间预算（整库几千块，本地模型约每秒几十块）。
const ReindexBudget = 10 * time.Minute

// ── 同题对照 ─────────────────────────────────────────────────────────

// Twin 同题小节：别的方案里标题路径与本节「同题」的一个小节。
type Twin struct {
	ProposalName  string `json:"proposalName"`
	ProposalTitle string `json:"proposalTitle"`
	ProjectType   string `json:"projectType"`
	Year          string `json:"year"`
	SectionID     int64  `json:"sectionId"`
	Level         int    `json:"level"`
	Title         string `json:"title"`
	TitlePath     string `json:"titlePath"`
	Chars         int    `json:"chars"`
}

// rePlainNo 首段开头的纯数字章号（「1 项目实施方案」）；只对路径首段认，免得把
// 「3 号地块」这类标题里的数字也当编号削掉。
var rePlainNo = regexp.MustCompile(`^(\d{1,3})[\s　]+(\D.+)$`)

// twinKey 标题路径的「同题」键：各段去掉开头编号后比较（「1.4.7.2 备选修复技术
// 介绍」与「备选修复技术介绍」同题，「第1章 总体」与「1 总体」同题）。
func twinKey(titlePath string) string {
	parts := strings.Split(titlePath, " › ")
	for i, p := range parts {
		core := headingCore(normHeading(p))
		if i == 0 {
			if m := rePlainNo.FindStringSubmatch(core); m != nil {
				core = strings.TrimSpace(m[2])
			}
		}
		parts[i] = strings.Join(strings.Fields(core), " ")
	}
	return strings.Join(parts, " › ")
}

// Twins 找出其他方案里与本节同题的小节（按方案清单序、方案内按标题序）。
func (s *Store) Twins(sectionID int64) ([]Twin, error) {
	if err := s.available(); err != nil {
		return nil, err
	}
	var pid int64
	var path string
	err := s.db.QueryRow(`SELECT proposal_id, title_path FROM proposal_sections WHERE id=?`, sectionID).Scan(&pid, &path)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("小节 %d 不存在（方案可能已被覆盖重录，请刷新标题树后重试）", sectionID)
	}
	if err != nil {
		return nil, unreadable(err)
	}
	key := twinKey(path)
	rows, err := s.db.Query(`SELECT s.id, s.level, s.title, s.title_path, LENGTH(s.body), p.name, p.title, p.project_type, p.year
FROM proposal_sections s JOIN proposals p ON p.id=s.proposal_id
WHERE s.proposal_id<>?
ORDER BY p.updated_at DESC, p.id DESC, s.seq`, pid)
	if err != nil {
		return nil, unreadable(err)
	}
	defer func() { _ = rows.Close() }()
	var out []Twin
	for rows.Next() {
		var t Twin
		if err := rows.Scan(&t.SectionID, &t.Level, &t.Title, &t.TitlePath, &t.Chars,
			&t.ProposalName, &t.ProposalTitle, &t.ProjectType, &t.Year); err != nil {
			return nil, unreadable(err)
		}
		if twinKey(t.TitlePath) == key {
			out = append(out, t)
		}
	}
	return out, rows.Err()
}

// ── 索引体检与重建 ───────────────────────────────────────────────────

// IndexStatus 索引健康快照。
type IndexStatus struct {
	VecDone       int    `json:"vecDone"` // 已向量化的语义块数
	VecTotal      int    `json:"vecTotal"`
	OrphanFts     int    `json:"orphanFts"`     // 全文索引行对应的小节已不存在
	MissingFts    int    `json:"missingFts"`    // 有小节却没有全文索引行
	StaleFts      int    `json:"staleFts"`      // 全文索引内容与当前的标题 / 路径 / 正文对不上
	MissingChunks int    `json:"missingChunks"` // 有正文却没有语义块的小节
	StaleChunks   int    `json:"staleChunks"`   // 块前缀还是旧标题路径的语义块
	Model         string `json:"model"`
}

// Broken 有没有结构类的索引问题（待向量化的块不算：检索时会自动补齐）。
func (st IndexStatus) Broken() bool {
	return st.OrphanFts > 0 || st.MissingFts > 0 || st.StaleFts > 0 || st.MissingChunks > 0 || st.StaleChunks > 0
}

// staleFtsSQL 全文索引内容过期的判定：索引行的 title 列=标题、body 列=标题路径+空格+
// 正文（与 insertFTS 同一口径），对不上就是改过而没跟上。
const staleFtsSQL = ` FROM proposal_fts f JOIN proposal_sections s ON s.id=f.rowid
WHERE f.title<>s.title OR f.body<>s.title_path||' '||s.body`

// IndexStatus 读一份索引健康快照（只读，不改库）。
func (s *Store) IndexStatus() (IndexStatus, error) {
	st := IndexStatus{}
	if err := s.available(); err != nil {
		return st, err
	}
	pending, total := s.PendingCount(0)
	st.VecDone, st.VecTotal = total-pending, total
	count := func(dst *int, q string) error { return s.db.QueryRow(q).Scan(dst) }
	if err := count(&st.OrphanFts, `SELECT COUNT(*) FROM proposal_fts f LEFT JOIN proposal_sections s ON s.id=f.rowid WHERE s.id IS NULL`); err != nil {
		return st, unreadable(err)
	}
	if err := count(&st.MissingFts, `SELECT COUNT(*) FROM proposal_sections s LEFT JOIN proposal_fts f ON f.rowid=s.id WHERE f.rowid IS NULL`); err != nil {
		return st, unreadable(err)
	}
	if err := count(&st.StaleFts, `SELECT COUNT(*)`+staleFtsSQL); err != nil {
		return st, unreadable(err)
	}
	if err := count(&st.MissingChunks, `SELECT COUNT(*) FROM proposal_sections s WHERE LENGTH(TRIM(COALESCE(s.body,'')))>0
AND NOT EXISTS(SELECT 1 FROM proposal_chunks c WHERE c.section_id=s.id)`); err != nil {
		return st, unreadable(err)
	}
	if err := count(&st.StaleChunks, `SELECT COUNT(*) FROM proposal_chunks c JOIN proposal_sections s ON s.id=c.section_id
WHERE substr(c.text,1,length(s.title_path))<>s.title_path`); err != nil {
		return st, unreadable(err)
	}
	model, err := s.EmbedModel()
	if err != nil {
		return st, err
	}
	st.Model = model
	return st, nil
}

// secRow 重建索引时读出的一个小节。
type secRow struct {
	id, pid           int64
	title, path, body string
}

// querySecs 把一条「小节行」查询读成切片（先读完再写，免得边读边写）。
func querySecs(q querier, sqlText string) ([]secRow, error) {
	rows, err := q.Query(sqlText)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []secRow
	for rows.Next() {
		var r secRow
		if err := rows.Scan(&r.id, &r.pid, &r.title, &r.path, &r.body); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// Reindex 修复并重建索引：清掉孤儿全文索引行、补上缺的全文索引与语义块、把前缀
// 过期的语义块换成新路径，最后把未向量化的块补齐（不受检索时那几秒预算限制）。
// 当前 embedding 模型与索引记录不一致时整库置空重嵌；模型不可用就只做结构类修复，
// 余下的向量检索时继续补。
func (s *Store) Reindex(ctx context.Context, e Embedder, model string) (IndexStatus, error) {
	if err := s.available(); err != nil {
		return IndexStatus{}, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return IndexStatus{}, unreadable(err)
	}
	defer func() { _ = tx.Rollback() }()
	now := time.Now().UTC().Format(time.RFC3339)

	// ① 孤儿全文索引行
	if _, err := tx.Exec(`DELETE FROM proposal_fts WHERE rowid NOT IN (SELECT id FROM proposal_sections)`); err != nil {
		return IndexStatus{}, err
	}
	// ② 有小节却没有全文索引行
	lost, err := querySecs(tx, `SELECT s.id, s.proposal_id, s.title, s.title_path, s.body FROM proposal_sections s
LEFT JOIN proposal_fts f ON f.rowid=s.id WHERE f.rowid IS NULL`)
	if err != nil {
		return IndexStatus{}, err
	}
	for _, r := range lost {
		if err := insertFTS(tx, r.id, r.title, r.path, r.body, r.pid); err != nil {
			return IndexStatus{}, err
		}
	}
	// ③ 有正文却没有语义块
	bare, err := querySecs(tx, `SELECT s.id, s.proposal_id, '', s.title_path, s.body FROM proposal_sections s
WHERE LENGTH(TRIM(COALESCE(s.body,'')))>0 AND NOT EXISTS(SELECT 1 FROM proposal_chunks c WHERE c.section_id=s.id)`)
	if err != nil {
		return IndexStatus{}, err
	}
	for _, r := range bare {
		if err := insertChunks(tx, r.pid, r.id, r.path, r.body, now); err != nil {
			return IndexStatus{}, err
		}
	}
	// ④ 块前缀还是旧标题路径（块文本=标题路径+换行+段落，换掉首行即可，向量重算）
	staleRows, err := tx.Query(`SELECT c.id, c.text, s.title_path FROM proposal_chunks c JOIN proposal_sections s ON s.id=c.section_id
WHERE substr(c.text,1,length(s.title_path))<>s.title_path`)
	if err != nil {
		return IndexStatus{}, err
	}
	type chunkFix struct {
		id         int64
		text, path string
	}
	var fixes []chunkFix
	for staleRows.Next() {
		var c chunkFix
		if err := staleRows.Scan(&c.id, &c.text, &c.path); err != nil {
			_ = staleRows.Close()
			return IndexStatus{}, err
		}
		fixes = append(fixes, c)
	}
	_ = staleRows.Close()
	if err := staleRows.Err(); err != nil {
		return IndexStatus{}, err
	}
	for _, c := range fixes {
		i := strings.Index(c.text, "\n")
		if i < 0 {
			i = 0
		}
		if _, err := tx.Exec(`UPDATE proposal_chunks SET text=?, vec=NULL, updated_at=? WHERE id=?`, c.path+c.text[i:], now, c.id); err != nil {
			return IndexStatus{}, err
		}
	}
	// ⑤ 全文索引内容过期（标题 / 路径 / 正文改过而索引行没跟上）：整行重写
	oldFts, err := querySecs(tx, `SELECT s.id, s.proposal_id, s.title, s.title_path, s.body`+staleFtsSQL)
	if err != nil {
		return IndexStatus{}, err
	}
	for _, r := range oldFts {
		if _, err := tx.Exec(`DELETE FROM proposal_fts WHERE rowid=?`, r.id); err != nil {
			return IndexStatus{}, err
		}
		if err := insertFTS(tx, r.id, r.title, r.path, r.body, r.pid); err != nil {
			return IndexStatus{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return IndexStatus{}, err
	}

	if e == nil || model == "" {
		return s.IndexStatus() // 本地模型不可用：结构类问题已修，向量留给检索时补
	}
	if cur, err := s.EmbedModel(); err != nil {
		return IndexStatus{}, err
	} else if cur != "" && cur != model {
		if err := s.ResetEmbedding(model); err != nil {
			return IndexStatus{}, err
		}
	}
	if _, _, err := s.EmbedPending(ctx, e, model, 0, ReindexBudget); err != nil {
		st, _ := s.IndexStatus()
		return st, fmt.Errorf("索引的结构问题已修复，但向量化中断（%v）；余下的块检索时会继续补齐", err)
	}
	return s.IndexStatus()
}
