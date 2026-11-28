package proposal

// structure.go — 方案库结构修正（2026-11-28 用户令「优化完善方案库的索引和编辑能力」）。
//
// 此前标题树只能改标题、整节替换正文：模型定错的层级、该拆的长节、多录的小节，
// 只能整册重录。这里补上受控的结构操作，整批在一个事务里，任何一步出错整批不落：
//   promote / demote      升级 / 降级。升级后紧跟在原父节之后（原父节的其余小节
//                          不动）；降级后归到上一同级小节名下，位置不变。对降过级的
//                          小节再升级会回到原位。
//   moveUp / moveDown     与上一 / 下一同级小节换位（整棵子树一起动）。
//   split                 光标所在行及之后的正文切成新子节。
//   mergeParent           并入父节（只许第一个子节：阅读顺序才不变）；子节升一级。
//   mergePrev            并入上一同级小节：正文接在它后面，子节原层级归到它名下。
//   addChild / addSibling 新增子节 / 新增同级小节。
//   remove                删除小节（连同全部子孙）。
// 标题里的编号（「1.4.7.2」）是原文照录的身份文字，结构操作不改它（用户 2026-11-28
// 拍板：界面只提示「编号与层级不符」）。全文索引与语义块随标题路径同步改写，
// 路径变了的小节向量置空待重嵌。

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// 结构操作名。
const (
	OpPromote     = "promote"
	OpDemote      = "demote"
	OpMoveUp      = "moveUp"
	OpMoveDown    = "moveDown"
	OpSplit       = "split"
	OpMergeParent = "mergeParent"
	OpMergePrev   = "mergePrev"
	OpAddChild    = "addChild"
	OpAddSibling  = "addSibling"
	OpRemove      = "remove"
)

// StructureInput 一次结构操作的输入。
type StructureInput struct {
	Op     string
	Offset int    // split：正文里的 rune 偏移；光标所在行及之后切出
	Title  string // split / addChild / addSibling：新小节标题
	Body   string // addChild / addSibling：新小节正文（可空）
}

// StructureResult 结构操作回执。
type StructureResult struct {
	Op            string
	FocusID       int64 // 操作完成后应落到的小节：新增=新节；并入 / 删除=承接它的小节；其余=本节；0=整册已空
	Affected      int   // 路径 / 层级 / 顺序 / 正文被改动的小节数（不含新建的）
	Created       int   // 新建的小节数
	Removed       int   // 删除的小节数（含子孙）
	RemovedChars  int   // 删除的正文总字数
	MissingAssets []string
}

// stNode 结构操作的工作节点：标题树的一行，带旧值用来算最小改动面。
type stNode struct {
	id      int64
	level   int
	title   string
	path    string // 新标题路径
	parent  int64  // 新父节点（1 级为 0）
	srcLine int

	oldPath   string
	oldLevel  int
	oldParent int64
	oldSeq    int

	body     string
	bodySet  bool // 正文已载入或已定
	bodyEdit bool // 正文要改写（body 为新正文）
	newNode  bool
}

func loadTreeTx(q querier, pid int64) ([]stNode, error) {
	rows, err := q.Query(`SELECT id, parent_id, level, seq, title, title_path, src_line FROM proposal_sections WHERE proposal_id=? ORDER BY seq`, pid)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []stNode
	for rows.Next() {
		var n stNode
		if err := rows.Scan(&n.id, &n.parent, &n.level, &n.oldSeq, &n.title, &n.path, &n.srcLine); err != nil {
			return nil, err
		}
		n.oldPath, n.oldLevel, n.oldParent = n.path, n.level, n.parent
		out = append(out, n)
	}
	return out, rows.Err()
}

func loadBodyTx(q rowQuerier, id int64) (string, error) {
	var b string
	err := q.QueryRow(`SELECT body FROM proposal_sections WHERE id=?`, id).Scan(&b)
	if err != nil {
		return "", err
	}
	return b, nil
}

// subtreeEnd 返回 nodes[i] 的子树结束下标（不含）：其后第一个层级不深于它的节点。
func subtreeEnd(nodes []stNode, i int) int {
	for j := i + 1; j < len(nodes); j++ {
		if nodes[j].level <= nodes[i].level {
			return j
		}
	}
	return len(nodes)
}

// prevSibling 返回 nodes[i] 上一个同级小节的下标；它已是本层第一个时返回 -1。
func prevSibling(nodes []stNode, i int) int {
	for k := i - 1; k >= 0; k-- {
		if nodes[k].level == nodes[i].level {
			return k
		}
		if nodes[k].level < nodes[i].level {
			break
		}
	}
	return -1
}

// concatNodes 把几段节点拼成一个新切片（原切片不再被引用，后面的改动不会串）。
func concatNodes(parts ...[]stNode) []stNode {
	n := 0
	for _, p := range parts {
		n += len(p)
	}
	out := make([]stNode, 0, n)
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

// rebuildTree 按各级标题栈重算每个节点的父节点与标题路径（与录入同一套规则：父=
// 前面最近的更高级标题，路径=各级标题自上而下用「 › 」连接，1 级的父为 0）。
func rebuildTree(nodes []stNode) {
	var ids [MaxLevels + 1]int64
	var titles [MaxLevels + 1]string
	for i := range nodes {
		n := &nodes[i]
		titles[n.level], ids[n.level] = n.title, n.id
		for k := n.level + 1; k <= MaxLevels; k++ {
			titles[k], ids[k] = "", 0
		}
		n.path = strings.Join(titles[1:n.level+1], " › ")
		n.parent = ids[n.level-1]
	}
}

// assertTree 结构不变量：首节 1 级、其后逐级展开不跳级、层级 1~4。任一不满足即
// 拒绝落库（fail-closed）。
func assertTree(nodes []stNode) error {
	prev := 0
	for i, n := range nodes {
		if n.level < 1 || n.level > MaxLevels || n.level > prev+1 {
			return fmt.Errorf("结构校验未通过（第 %d 节「%s」level=%d，前一节 level=%d），本次操作未落库", i+1, n.title, n.level, prev)
		}
		prev = n.level
	}
	return nil
}

// cleanTitle 新小节标题：折成一行、收拢空白、截到上限；空标题报错。
func cleanTitle(raw string) (string, error) {
	t := truncRunes(strings.Join(strings.Fields(raw), " "), MaxTitleRunes)
	if t == "" {
		return "", errors.New("新小节的标题不能为空")
	}
	return t, nil
}

// joinBodies 并入时拼接两段正文：各自去首尾空白，中间空一行；一边为空就是另一边。
func joinBodies(a, b string) string {
	a, b = strings.TrimSpace(a), strings.TrimSpace(b)
	switch {
	case a == "":
		return b
	case b == "":
		return a
	}
	return a + "\n\n" + b
}

// Restructure 对一个小节做一次结构操作。回执里的 FocusID 是操作完成后界面应落到
// 的小节。还在分批录入、没收尾的方案不动（续录游标按末节算）。
func (s *Store) Restructure(sectionID int64, in StructureInput) (*StructureResult, error) {
	if err := s.available(); err != nil {
		return nil, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, unreadable(err)
	}
	defer func() { _ = tx.Rollback() }()

	var pid int64
	var status string
	err = tx.QueryRow(`SELECT s.proposal_id, p.status FROM proposal_sections s JOIN proposals p ON p.id=s.proposal_id WHERE s.id=?`, sectionID).
		Scan(&pid, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("小节 %d 不存在（方案可能已被覆盖重录，请刷新标题树后重试）", sectionID)
	}
	if err != nil {
		return nil, unreadable(err)
	}
	if status == StatusIngesting {
		return nil, errors.New("这份方案还在分批录入、尚未收尾，收尾后才能调整结构")
	}
	nodes, err := loadTreeTx(tx, pid)
	if err != nil {
		return nil, unreadable(err)
	}
	i := -1
	for k := range nodes {
		if nodes[k].id == sectionID {
			i = k
			break
		}
	}
	if i < 0 {
		return nil, fmt.Errorf("小节 %d 不在所属方案的标题树里，请刷新标题树后重试", sectionID)
	}
	L := nodes[i].level
	end := subtreeEnd(nodes, i)
	prev := prevSibling(nodes, i)
	next := -1
	if end < len(nodes) && nodes[end].level == L {
		next = end
	}
	bodyOf := func(k int) (string, error) {
		if !nodes[k].bodySet {
			b, err := loadBodyTx(tx, nodes[k].id)
			if err != nil {
				return "", unreadable(err)
			}
			nodes[k].body, nodes[k].bodySet = b, true
		}
		return nodes[k].body, nil
	}
	joinCheck := func(a, b string) (string, error) {
		j := joinBodies(a, b)
		if n := len([]rune(j)); n > MaxBodyChars {
			return "", fmt.Errorf("并入后正文 %d 字超过单节上限 %d 字，未并入", n, MaxBodyChars)
		}
		return j, nil
	}
	var removed []stNode
	focus, newAt := sectionID, -1

	switch in.Op {
	case OpPromote:
		if L <= 1 {
			return nil, errors.New("它已是 1 级标题，不能再用「升级」")
		}
		sub := append([]stNode(nil), nodes[i:end]...)
		for k := range sub {
			sub[k].level--
		}
		rest := make([]stNode, 0, len(nodes)-(end-i))
		rest = append(rest, nodes[:i]...)
		rest = append(rest, nodes[end:]...)
		// 落点：原父节的子树末尾。原父节=去掉子树后仍排在 i 之前、层级恰低一级的那个。
		p := -1
		for k := i - 1; k >= 0; k-- {
			if rest[k].level == L-1 {
				p = k
				break
			}
		}
		if p < 0 {
			return nil, fmt.Errorf("结构校验未通过：找不到「%s」的上一级标题，本次操作未落库", nodes[i].title)
		}
		at := len(rest)
		for k := p + 1; k < len(rest); k++ {
			if rest[k].level <= L-1 {
				at = k
				break
			}
		}
		nodes = append(rest[:at:at], append(sub, rest[at:]...)...)
	case OpDemote:
		if L >= MaxLevels {
			return nil, errors.New("它已是 4 级标题，不能再降级")
		}
		if prev < 0 {
			return nil, errors.New("它前面没有同级小节，降级后没有可挂靠的上一节；先把它上移 / 下移到合适的位置再降级")
		}
		for k := i; k < end; k++ {
			nodes[k].level++
		}
	case OpMoveUp:
		if prev < 0 {
			return nil, errors.New("它已经是上一级标题下的第一个小节，不能再上移")
		}
		nodes = concatNodes(nodes[:prev], nodes[i:end], nodes[prev:i], nodes[end:])
	case OpMoveDown:
		if next < 0 {
			return nil, errors.New("它已经是上一级标题下的最后一个小节，不能再下移")
		}
		nextEnd := subtreeEnd(nodes, next)
		nodes = concatNodes(nodes[:i], nodes[next:nextEnd], nodes[i:end], nodes[nextEnd:])
	case OpSplit:
		if L >= MaxLevels {
			return nil, errors.New("它已是 4 级标题，下面不能再拆出子节；可以先给它升级")
		}
		title, err := cleanTitle(in.Title)
		if err != nil {
			return nil, err
		}
		body, err := bodyOf(i)
		if err != nil {
			return nil, err
		}
		rs := []rune(body)
		off := in.Offset
		if off < 0 || off > len(rs) {
			off = len(rs)
		}
		cut := off // 光标所在行的行首
		for cut > 0 && rs[cut-1] != '\n' {
			cut--
		}
		if cut <= 0 {
			return nil, errors.New("光标在本节第一行，拆分后本节会没有正文；把光标移到要拆出去的那一段再拆")
		}
		head := strings.TrimRight(string(rs[:cut]), "\n \t\r")
		tail := strings.TrimLeft(string(rs[cut:]), "\n")
		if strings.TrimSpace(tail) == "" {
			return nil, errors.New("光标之后没有正文可拆出去")
		}
		nodes[i].body, nodes[i].bodySet, nodes[i].bodyEdit = head, true, true
		nodes = concatNodes(nodes[:i+1], []stNode{{level: L + 1, title: title, body: tail, bodySet: true, newNode: true}}, nodes[i+1:])
		newAt = i + 1
	case OpMergeParent:
		if L <= 1 {
			return nil, errors.New("它是 1 级标题，上面没有父节可并入")
		}
		if prev >= 0 {
			return nil, errors.New("它不是父节下的第一个小节，并入后正文会排到前面几节之前；要保住阅读顺序请用「并入上一同级小节」")
		}
		if i == 0 || nodes[i-1].level != L-1 {
			return nil, fmt.Errorf("结构校验未通过：「%s」前一个节点不是它的父节，本次操作未落库", nodes[i].title)
		}
		pb, err := bodyOf(i - 1)
		if err != nil {
			return nil, err
		}
		tb, err := bodyOf(i)
		if err != nil {
			return nil, err
		}
		joined, err := joinCheck(pb, tb)
		if err != nil {
			return nil, err
		}
		removed = append(removed, nodes[i]) // 正文已搬走：本节连同索引清除
		nodes[i-1].body, nodes[i-1].bodySet, nodes[i-1].bodyEdit = joined, true, true
		for k := i + 1; k < end; k++ { // 子节升一级，顶上来归到父节名下
			nodes[k].level--
		}
		nodes = concatNodes(nodes[:i], nodes[i+1:])
		focus = nodes[i-1].id
	case OpMergePrev:
		if prev < 0 {
			return nil, errors.New("它前面没有同级小节可并入")
		}
		pb, err := bodyOf(prev)
		if err != nil {
			return nil, err
		}
		tb, err := bodyOf(i)
		if err != nil {
			return nil, err
		}
		joined, err := joinCheck(pb, tb)
		if err != nil {
			return nil, err
		}
		removed = append(removed, nodes[i]) // 正文已搬走：本节连同索引清除
		nodes[prev].body, nodes[prev].bodySet, nodes[prev].bodyEdit = joined, true, true
		// 子节层级不变，原位归到上一同级小节名下（两者同级，子节本来就挂在这一层下面）。
		nodes = concatNodes(nodes[:i], nodes[i+1:])
		focus = nodes[prev].id
	case OpRemove:
		removed = append([]stNode(nil), nodes[i:end]...)
		switch {
		case prev >= 0:
			focus = nodes[prev].id
		case i > 0:
			focus = nodes[i-1].id // 它是父节的第一个（或唯一的）子节：落到父节
		case end < len(nodes):
			focus = nodes[end].id
		default:
			focus = 0
		}
		nodes = concatNodes(nodes[:i], nodes[end:])
	case OpAddChild, OpAddSibling:
		level := L
		if in.Op == OpAddChild {
			if L >= MaxLevels {
				return nil, errors.New("它已是 4 级标题，下面不能再加子节；可以改用「新增同级小节」")
			}
			level = L + 1
		}
		title, err := cleanTitle(in.Title)
		if err != nil {
			return nil, err
		}
		body := strings.ReplaceAll(strings.ReplaceAll(strings.TrimPrefix(in.Body, "\uFEFF"), "\r\n", "\n"), "\r", "\n")
		if n := len([]rune(body)); n > MaxBodyChars {
			return nil, fmt.Errorf("新小节正文 %d 字超过单节上限 %d 字", n, MaxBodyChars)
		}
		nodes = concatNodes(nodes[:end], []stNode{{level: level, title: title, body: body, bodySet: true, newNode: true}}, nodes[end:])
		newAt = end
	default:
		return nil, fmt.Errorf("未知的结构操作 %q", in.Op)
	}

	rebuildTree(nodes)
	if err := assertTree(nodes); err != nil {
		return nil, err
	}
	var oldRefs []assetDiskRef
	removedChars := 0
	if len(removed) > 0 { // 删除与并入都会清掉小节行；并入的正文只是搬了家，不计入删除字数
		if in.Op == OpRemove {
			ph := make([]string, len(removed))
			args := make([]any, len(removed))
			for k, n := range removed {
				ph[k], args[k] = "?", n.id
			}
			if err := tx.QueryRow(`SELECT COALESCE(SUM(LENGTH(body)),0) FROM proposal_sections WHERE id IN (`+strings.Join(ph, ",")+`)`, args...).Scan(&removedChars); err != nil {
				return nil, unreadable(err)
			}
		}
		if oldRefs, err = assetRefsTx(tx, pid); err != nil {
			return nil, unreadable(err)
		}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	res, missing, err := s.persistTree(tx, pid, nodes, removed, now)
	if err != nil {
		return nil, err
	}
	if newAt >= 0 {
		focus = nodes[newAt].id
	}
	if _, err := tx.Exec(`UPDATE proposals SET updated_at=? WHERE id=?`, now, pid); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	s.gcAssetRefs(oldRefs)
	res.Op, res.FocusID = in.Op, focus
	res.RemovedChars, res.MissingAssets = removedChars, missing
	return res, nil
}

// persistTree 把算好的新树落库：先删后插；路径变了的小节改写全文索引与语义块前缀，
// 正文变了的小节整节重建索引；只有父节点 / 层级 / 顺序变了的只改行。
func (s *Store) persistTree(tx *sql.Tx, pid int64, nodes, removed []stNode, now string) (*StructureResult, []string, error) {
	res := &StructureResult{}
	for _, n := range removed {
		if _, err := tx.Exec(`DELETE FROM proposal_fts WHERE rowid=?`, n.id); err != nil {
			return nil, nil, err
		}
		if _, err := tx.Exec(`DELETE FROM proposal_assets WHERE section_id=?`, n.id); err != nil {
			return nil, nil, err
		}
		if _, err := tx.Exec(`DELETE FROM proposal_chunks WHERE section_id=?`, n.id); err != nil {
			return nil, nil, err
		}
		if _, err := tx.Exec(`DELETE FROM proposal_sections WHERE id=?`, n.id); err != nil {
			return nil, nil, err
		}
	}
	res.Removed = len(removed)
	var missingAll []string
	for i := range nodes {
		n := &nodes[i]
		if n.newNode {
			prepared, err := s.prepareBodyAssets(n.body, "")
			if err != nil {
				return nil, nil, fmt.Errorf("托管小节 %q 的图片: %w", n.title, err)
			}
			if c := len([]rune(prepared.Body)); c > MaxBodyChars {
				return nil, nil, fmt.Errorf("新小节正文 %d 字超过单节上限 %d 字", c, MaxBodyChars)
			}
			out, err := tx.Exec(`INSERT INTO proposal_sections(proposal_id, parent_id, level, seq, title, title_path, body, src_line, created_at, updated_at)
VALUES(?,?,?,?,?,?,?,?,?,?)`, pid, n.parent, n.level, i, n.title, n.path, prepared.Body, 0, now, now)
			if err != nil {
				return nil, nil, fmt.Errorf("写入小节 %s: %w", n.title, err)
			}
			n.id, _ = out.LastInsertId()
			if err := s.insertAssetsTx(tx, pid, n.id, prepared.Assets, now); err != nil {
				return nil, nil, fmt.Errorf("写入小节 %s 的图片资产: %w", n.title, err)
			}
			if err := insertFTS(tx, n.id, n.title, n.path, prepared.Body, pid); err != nil {
				return nil, nil, fmt.Errorf("索引小节 %s: %w", n.title, err)
			}
			if err := insertChunks(tx, pid, n.id, n.path, prepared.Body, now); err != nil {
				return nil, nil, fmt.Errorf("切分小节 %s: %w", n.title, err)
			}
			res.Created++
			continue
		}
		switch {
		case n.bodyEdit:
			if _, missing, err := s.setSectionBodyTx(tx, n.id, pid, n.path, n.body, "", now); err != nil {
				return nil, nil, fmt.Errorf("改写小节《%s》正文: %w", n.title, err)
			} else {
				missingAll = append(missingAll, missing...)
			}
		case n.path != n.oldPath:
			if err := retitleIndexTx(tx, n.id, pid, n.title, n.oldPath, n.path); err != nil {
				return nil, nil, fmt.Errorf("改写小节 %s: %w", n.oldPath, err)
			}
		}
		if n.path == n.oldPath && n.level == n.oldLevel && n.parent == n.oldParent && i == n.oldSeq {
			continue
		}
		if _, err := tx.Exec(`UPDATE proposal_sections SET parent_id=?, level=?, seq=?, title_path=?, updated_at=? WHERE id=?`,
			n.parent, n.level, i, n.path, now, n.id); err != nil {
			return nil, nil, err
		}
		res.Affected++
	}
	return res, missingAll, nil
}
