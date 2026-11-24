package proposal

// maintain.go — 方案库维护面（2026-11-24）：元数据修改、小节增量修正、整册导出。
//
// 背景：此前库只有「整册录入 / 整册覆盖 / 删除」三种写法。录错一个标题或一个类型字段，
// 只能整册重录（236 个节点重传），或者绕开产品直连 SQLite 改库（用户 2026-11-24 实弹）。
// 这里补上受控的增量修正；导出让库内方案能回到 Markdown 文件。工具面见
// tool/builtin/proposal_maintain_tools.go。

import (
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// UpdateInput 修改一份方案的元数据；nil 表示该项不改。
type UpdateInput struct {
	Title       *string // 改标题；方案名（唯一键）随之改成新标题
	ProjectType *string
	Year        *string
	Summary     *string
	Workspace   *string
}

// UpdateMeta 修改方案元数据（标题 / 类型 / 年度 / 摘要 / 工作区）。nameOrTitle 为方案名
// 或现标题；方案不存在、或改名撞到已有方案时返回 error。
func (s *Store) UpdateMeta(nameOrTitle string, in UpdateInput) (*Proposal, error) {
	if err := s.available(); err != nil {
		return nil, err
	}
	p, err := s.Resolve(nameOrTitle)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, fmt.Errorf("方案库里没有《%s》", nameOrTitle)
	}
	next := *p
	if in.Title != nil {
		t := truncRunes(strings.TrimSpace(*in.Title), MaxTitleRunes)
		if t == "" {
			return nil, errors.New("新标题不能为空")
		}
		if t != p.Title {
			if other, err := s.Resolve(t); err != nil {
				return nil, err
			} else if other != nil && other.ID != p.ID {
				return nil, fmt.Errorf("方案库里已有《%s》，改名会撞名；换一个标题", t)
			}
		}
		next.Title, next.Name = t, t // 方案名=标题（与录入时的约定一致）
	}
	set := func(dst *string, v *string) {
		if v != nil {
			*dst = strings.TrimSpace(*v)
		}
	}
	set(&next.ProjectType, in.ProjectType)
	set(&next.Year, in.Year)
	set(&next.Summary, in.Summary)
	set(&next.Workspace, in.Workspace)
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := s.db.Exec(`UPDATE proposals SET title=?, name=?, project_type=?, year=?, summary=?, workspace=?, updated_at=? WHERE id=?`,
		next.Title, next.Name, next.ProjectType, next.Year, next.Summary, next.Workspace, now, p.ID); err != nil {
		return nil, fmt.Errorf("修改方案《%s》: %w", p.Title, err)
	}
	next.UpdatedAt, _ = time.Parse(time.RFC3339, now)
	return &next, nil
}

// SectionEdit 小节增量修正；nil 表示该项不改。
type SectionEdit struct {
	Title        *string // 改标题（标题路径随之改，子树各节的路径一并改）
	Body         *string // 整节替换正文（该节旧正文与它挂的托管图片让位，语义向量重算）
	AssetBaseDir string  // 新正文里相对图片引用的解析根
}

// UpdateSection 增量修正一个小节，返回受影响的各节（本节在前，子树按文档顺序）。
// 改标题会连带子树：全文索引与语义块的标题路径前缀同步改写，语义向量置空待重嵌
// （检索时自动补齐）。整批在一个事务里，任何一步出错整批不落。
func (s *Store) UpdateSection(sectionID int64, in SectionEdit) ([]Section, error) {
	if err := s.available(); err != nil {
		return nil, err
	}
	if in.Title == nil && in.Body == nil {
		return nil, errors.New("没有要改的内容：title 与 body 至少给一个")
	}
	body := ""
	if in.Body != nil {
		body = strings.ReplaceAll(strings.ReplaceAll(strings.TrimPrefix(*in.Body, "\uFEFF"), "\r\n", "\n"), "\r", "\n")
		if n := len([]rune(body)); n > MaxBodyChars {
			return nil, fmt.Errorf("新正文 %d 字超过单节上限 %d 字", n, MaxBodyChars)
		}
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, unreadable(err)
	}
	defer func() { _ = tx.Rollback() }()

	var sec Section
	var pid int64
	err = tx.QueryRow(`SELECT id, proposal_id, level, seq, title, title_path FROM proposal_sections WHERE id=?`, sectionID).
		Scan(&sec.ID, &pid, &sec.Level, &sec.Seq, &sec.Title, &sec.TitlePath)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("小节 %d 不存在（方案可能已被覆盖重录，请重新 proposal_search 取新的 section_id）", sectionID)
	}
	if err != nil {
		return nil, unreadable(err)
	}

	// 子树：文档顺序上排在本节之后、层级更深的连续各节。
	affected := []Section{sec}
	rows, err := tx.Query(`SELECT id, level, title, title_path FROM proposal_sections WHERE proposal_id=? AND seq>? ORDER BY seq`, pid, sec.Seq)
	if err != nil {
		return nil, unreadable(err)
	}
	for rows.Next() {
		var a Section
		if err := rows.Scan(&a.ID, &a.Level, &a.Title, &a.TitlePath); err != nil {
			_ = rows.Close()
			return nil, unreadable(err)
		}
		if a.Level <= sec.Level {
			break
		}
		affected = append(affected, a)
	}
	_ = rows.Close()
	if err := rows.Err(); err != nil {
		return nil, unreadable(err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if in.Title != nil {
		nt := truncRunes(strings.TrimSpace(*in.Title), MaxTitleRunes)
		if nt == "" {
			return nil, errors.New("新标题不能为空")
		}
		parent := ""
		if i := strings.LastIndex(sec.TitlePath, " › "); i >= 0 {
			parent = sec.TitlePath[:i]
		}
		newPath := nt
		if parent != "" {
			newPath = parent + " › " + nt
		}
		for i := range affected {
			a := &affected[i]
			oldPath := a.TitlePath
			if i == 0 {
				a.Title, a.TitlePath = nt, newPath
			} else {
				a.TitlePath = newPath + strings.TrimPrefix(oldPath, sec.TitlePath)
			}
			if _, err := tx.Exec(`UPDATE proposal_sections SET title=?, title_path=?, updated_at=? WHERE id=?`,
				a.Title, a.TitlePath, now, a.ID); err != nil {
				return nil, fmt.Errorf("改写小节 %s: %w", oldPath, err)
			}
			if err := retitleIndexTx(tx, a.ID, pid, a.Title, oldPath, a.TitlePath); err != nil {
				return nil, err
			}
		}
	}
	if in.Body != nil {
		if _, _, err := s.setSectionBodyTx(tx, sec.ID, pid, affected[0].TitlePath, body, in.AssetBaseDir, now); err != nil {
			return nil, fmt.Errorf("改写小节《%s》正文: %w", affected[0].Title, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return affected, nil
}

// retitleIndexTx 小节改了标题或标题路径后，同步全文索引与语义块（块文本=标题路径+段落，
// 换前缀即可；文字变了，向量置空待重嵌）。
func retitleIndexTx(tx *sql.Tx, sid, pid int64, title, oldPath, newPath string) error {
	var body string
	if err := tx.QueryRow(`SELECT body FROM proposal_sections WHERE id=?`, sid).Scan(&body); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM proposal_fts WHERE rowid=?`, sid); err != nil {
		return err
	}
	if err := insertFTS(tx, sid, title, newPath, body, pid); err != nil {
		return err
	}
	rows, err := tx.Query(`SELECT id, text FROM proposal_chunks WHERE section_id=?`, sid)
	if err != nil {
		return err
	}
	type chunk struct {
		id   int64
		text string
	}
	var chunks []chunk
	for rows.Next() {
		var c chunk
		if err := rows.Scan(&c.id, &c.text); err != nil {
			_ = rows.Close()
			return err
		}
		chunks = append(chunks, c)
	}
	_ = rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, c := range chunks {
		text := newPath + "\n" + strings.TrimPrefix(c.text, oldPath+"\n")
		if _, err := tx.Exec(`UPDATE proposal_chunks SET text=?, vec=NULL WHERE id=?`, text, c.id); err != nil {
			return err
		}
	}
	return nil
}

// FullText 返回一份方案的全部小节（按文档顺序，带正文），供导出。
func (s *Store) FullText(nameOrTitle string) (*Proposal, []Section, error) {
	if err := s.available(); err != nil {
		return nil, nil, err
	}
	p, err := s.Resolve(nameOrTitle)
	if err != nil {
		return nil, nil, err
	}
	if p == nil {
		return nil, nil, fmt.Errorf("方案库里没有《%s》", nameOrTitle)
	}
	secs, err := querySectionsTx(s.db, p.ID, true)
	if err != nil {
		return nil, nil, unreadable(err)
	}
	return p, secs, nil
}

// AssetPath 托管图片的磁盘路径（ref 为 gaea-asset://<sha256> 或裸哈希）。
func (s *Store) AssetPath(ref string) (string, error) {
	hash, err := assetHash(ref)
	if err != nil {
		return "", err
	}
	if s.assetRoot == "" {
		return "", errors.New("方案图片目录不可用")
	}
	matches, err := filepath.Glob(filepath.Join(s.assetRoot, "objects", hash[:2], hash+".*"))
	if err != nil || len(matches) == 0 {
		return "", fmt.Errorf("托管图片 %s… 的文件不存在", hash[:12])
	}
	return matches[0], nil
}
