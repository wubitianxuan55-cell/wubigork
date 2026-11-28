// Package proposal 方案库存储层（2026-11-23 用户拍板立项）：按标题层级 1~4 级
// 存取与检索。录入与索引都在办公板块的方案模式内由模型完成（proposal_save/
// proposal_search/proposal_read 三工具），无独立解析入口。与造价库同模式：
// 存 Hephaestus.db（SchemaV35~V37）。
//
// 一份方案存成三层：标题树（1~4 级标题与父子关系，管定位与骨架参照）、小节
// （某标题下逐字原样的正文，管阅读与出处）、段落（小节正文切块后的语义索引
// 单元，管被检索命中，见 semantic.go）。正文图片另走托管资产：二进制按
// sha256 落用户数据目录，proposal_assets 记归属，正文只留稳定 asset URI。
//
// 录入两形态：A=模型直给各节正文（sections）；B=模型只给标题与层级（outline），
// 正文由源文件按标题行逐字切出——几百页的方案不必让模型重打十几万字。两形态
// 都可分批续录（mode=append）；B 形态的末节正文留到下一批首个标题或收尾时
// 才截取（期间方案状态为「录入中」）。
//
// 检索口径：全文路（FTS5 trigram 短语 ≥3 字 + LIKE 兜短词，只管精确词）+ 语义路
// （段落向量，本地 embedding）→ 融合 → 本地精排；命中回小节并给出命中位置段落。
package proposal

import (
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	// MaxLevels 标题层级深度（用户拍板：做多到四级）。
	MaxLevels = 4
	// MaxBodyChars 单节正文上限（rune）。超过即拒——静默截断会丢正文。
	MaxBodyChars = 200000
	// MaxTitleRunes 标题长度上限。
	MaxTitleRunes = 200
)

// 方案状态。
const (
	StatusActive    = "现行"  // 录入完成
	StatusIngesting = "录入中" // 源文件分批录入未收尾（末节正文待截取）
)

// 检索范围。
const (
	ScopeBody  = ""      // 默认：全文 + 语义
	ScopeTitle = "title" // 只在标题树里找（骨架参照与导航）
)

// ErrStoreUnreadable 库读失败统一文案前缀：检索/读取必须如实报错，不得当「库空」。
const ErrStoreUnreadable = "方案库暂不可读（读取失败），可重试"

func unreadable(err error) error { return fmt.Errorf("%s: %w", ErrStoreUnreadable, err) }

// Proposal 是一份方案成品的元数据（不含小节）。
type Proposal struct {
	ID          int64
	Name        string // 唯一键（标题 slug）；同名即同一份方案
	Title       string
	ProjectType string // 项目类型（如 技术标/土壤修复方案/调查报告）
	Year        string // 年度（如 2026）
	Source      string // 来源说明（如「包3技术标_第三版.docx」）
	Workspace   string // 来源工作区路径
	Status      string // 现行 / 录入中
	Summary     string // 一段话摘要（模型在方案模式内生成）
	SourceFile  string // 形态 B 的源文件路径（续录时核对）
	IngestLine  int    // 形态 B 已录到源文件第几行（末个标题所在行；0=未开始）
	IngestHash  string // 已录前缀的指纹（续录时核对源文件未被改动）
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Section 是一个标题节点（level 1~4）：标题树的一行，带该标题下的正文。
type Section struct {
	ID        int64
	ParentID  int64
	Level     int
	Seq       int // 全方案内顺序
	Title     string
	TitlePath string // 「第4章 安全保障 › 4.3 危险源辨识」
	Body      string // 该标题下、下个标题前的逐字正文（目录面查询时为空）
	SrcLine   int    // 形态 B：标题在源文件里的行号（溯源）
	Chars     int    // 正文字数（rune）
}

// Child 是子节的轻量描述（命中回执与阅读回执里供下钻）。
type Child struct {
	ID    int64  `json:"id"`
	Level int    `json:"level"`
	Title string `json:"title"`
	Chars int    `json:"chars"`
}

// Stats 是一份方案的结构统计。
type Stats struct {
	Sections     int
	Chars        int
	LevelCounts  [MaxLevels + 1]int // 下标=层级（0 不用）
	LongestTitle string
	LongestChars int
}

// ProposalInfo 是清单面的一条：元数据 + 规模 + 语义索引进度。
type ProposalInfo struct {
	Proposal
	Sections   int
	Chars      int
	IndexDone  int // 已向量化的段落数
	IndexTotal int // 段落总数
}

// Store 方案库存储。
type Store struct {
	db        *sql.DB
	assetRoot string
	noWrite   bool // 演练模式（SaveInput.DryRun）：图片文件不落盘，事务最后回滚
}

// Open 打开方案库；gdb 为 nil 时不可用。
func Open(gdb *sql.DB) (*Store, error) {
	if gdb == nil {
		return nil, errors.New("proposal db unavailable")
	}
	return &Store{db: gdb}, nil
}

func (s *Store) available() error {
	if s == nil || s.db == nil {
		return fmt.Errorf("%s: %w", ErrStoreUnreadable, errors.New("db unavailable"))
	}
	return nil
}

const proposalCols = `id,name,title,project_type,year,source,workspace,status,summary,source_file,ingest_line,ingest_hash,created_at,updated_at`

type rowScanner interface{ Scan(...any) error }

func scanProposal(sc rowScanner) (Proposal, error) {
	var p Proposal
	var created, updated string
	err := sc.Scan(&p.ID, &p.Name, &p.Title, &p.ProjectType, &p.Year, &p.Source, &p.Workspace,
		&p.Status, &p.Summary, &p.SourceFile, &p.IngestLine, &p.IngestHash, &created, &updated)
	if err != nil {
		return p, err
	}
	p.CreatedAt, _ = time.Parse(time.RFC3339, created)
	p.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
	return p, nil
}

// Get 按 name 取方案元数据；未找到返回 (nil, nil)。
func (s *Store) Get(name string) (*Proposal, error) {
	if err := s.available(); err != nil {
		return nil, err
	}
	p, err := scanProposal(s.db.QueryRow(`SELECT `+proposalCols+` FROM proposals WHERE name=?`, name))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, unreadable(err)
	}
	return &p, nil
}

// Resolve 按 name 或标题（精确）取方案；未找到返回 (nil, nil)。
func (s *Store) Resolve(nameOrTitle string) (*Proposal, error) {
	if err := s.available(); err != nil {
		return nil, err
	}
	p, err := scanProposal(s.db.QueryRow(`SELECT `+proposalCols+` FROM proposals WHERE name=? OR title=? ORDER BY id LIMIT 1`,
		nameOrTitle, nameOrTitle))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, unreadable(err)
	}
	return &p, nil
}

// List 返回全部方案（更新时间降序），带规模与语义索引进度。读失败如实报错。
func (s *Store) List() ([]ProposalInfo, error) {
	if err := s.available(); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(`SELECT p.` + strings.ReplaceAll(proposalCols, ",", ",p.") + `,
  (SELECT COUNT(*) FROM proposal_sections x WHERE x.proposal_id=p.id),
  (SELECT COALESCE(SUM(LENGTH(x.body)),0) FROM proposal_sections x WHERE x.proposal_id=p.id),
  (SELECT COUNT(*) FROM proposal_chunks c WHERE c.proposal_id=p.id AND c.vec IS NOT NULL),
  (SELECT COUNT(*) FROM proposal_chunks c WHERE c.proposal_id=p.id)
FROM proposals p ORDER BY p.updated_at DESC, p.id DESC`)
	if err != nil {
		return nil, unreadable(err)
	}
	defer func() { _ = rows.Close() }()
	var out []ProposalInfo
	for rows.Next() {
		var it ProposalInfo
		var created, updated string
		if err := rows.Scan(&it.ID, &it.Name, &it.Title, &it.ProjectType, &it.Year, &it.Source, &it.Workspace,
			&it.Status, &it.Summary, &it.SourceFile, &it.IngestLine, &it.IngestHash, &created, &updated,
			&it.Sections, &it.Chars, &it.IndexDone, &it.IndexTotal); err != nil {
			return nil, unreadable(err)
		}
		it.CreatedAt, _ = time.Parse(time.RFC3339, created)
		it.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
		out = append(out, it)
	}
	return out, rows.Err()
}

// ── 录入 ───────────────────────────────────────────────────────────

// OutlineEntry 是形态 B 的一项：模型认定的一个标题及其层级。
type OutlineEntry struct {
	Level int
	Title string // 在源文件里定位用的字面：源文件怎么写就怎么写（OCR 错字也照写）
	Line  int    // 模型给的 1-based 源文件行号；0=从上一项之后前向搜索
	// DisplayTitle 入库显示名（可选，2026-11-27 AI 反馈 #11）：给了就用它当标题，
	// 修正源文件里的 OCR 错字。源文件行带编号而它没带时，编号会补回去。给了它，
	// title 定位不到就在标题样行里找与它只差一两个字的那一行。
	DisplayTitle string
}

// SaveInput 一次录入调用的全部输入。
type SaveInput struct {
	Meta         Proposal
	Overwrite    bool           // 整册覆盖已存在的同名方案须显式确认
	Append       bool           // mode=append：续录同一方案
	Sections     []Section      // 形态 A：模型直给正文（Level/Title/Body）
	Outline      []OutlineEntry // 形态 B：模型只给标题与层级
	Lines        []string       // 形态 B 的源文件行（调用方读好传入；存储层不做文件 IO）
	SourceFile   string         // 形态 B 的源文件路径（落库留痕，续录核对用）
	AssetBaseDir string         // 正文相对图片的解析根；通常是 Markdown 源文件所在目录
	Finalize     bool           // 形态 B：本批是最后一批，末节正文截至文件尾
	DryRun       bool           // 演练：整批校验、定位、切正文照做，但库与图片文件都不写
}

// ExistsError 同名方案已存在且未确认覆盖。
type ExistsError struct {
	Name     string
	Title    string
	Source   string
	Sections int
}

func (e *ExistsError) Error() string {
	src := e.Source
	if src == "" {
		src = "未注明"
	}
	return fmt.Sprintf("方案库已有同名方案《%s》（来源：%s，%d 节）。整册覆盖须显式 overwrite=true；续录同一份用 mode=append；另起一份请换标题",
		e.Title, src, e.Sections)
}

// SaveResult 录入回执数据。
type SaveResult struct {
	ProposalID    int64
	Created       bool     // 本次新建
	Replaced      bool     // 本次整册覆盖了旧内容
	Added         int      // 本次写入的标题节点数
	Clamped       int      // level>4 被压到 4 级的节点数
	SkippedLines  int      // 首个标题前未入库的行数（封面/目录）
	IngestLine    int      // 已录到源文件第几行
	TotalLines    int      // 源文件总行数（形态 A 为 0）
	Done          bool     // 是否录入完成
	PendingTitle  string   // 未收尾时待截取正文的末节标题
	WeakMatches   []string // 定位到「不像标题的行」的标题（模型宜核对行号）
	TitleFixes    []string // 用 display_title 修正过的标题（「源文件写法 → 入库写法」），回执列给用户确认
	Assets        int      // 本次托管、且正文已改写引用的图片数
	MissingAssets []string // 正文里未能托管的图片引用（保留原写法）
	Stats         Stats    // 写入后的结构统计
	DryRun        bool     // 本次是演练：未写库（Stats 为「假如写入」的统计）
}

// Save 录入一批：整批在一个事务里，任何一项出错整批不落库（fail-closed）。
func (s *Store) Save(in SaveInput) (*SaveResult, error) {
	if err := s.available(); err != nil {
		return nil, err
	}
	title := strings.TrimSpace(in.Meta.Title)
	if title == "" {
		return nil, errors.New("title 为必填项")
	}
	formA, formB := len(in.Sections) > 0, len(in.Outline) > 0
	switch {
	case formA && formB:
		return nil, errors.New("sections 与 outline 只能二选一")
	case !formA && !formB && !(in.Append && in.Finalize):
		return nil, errors.New("sections 或 outline 至少给一个（只收尾时可仅 mode=append + finalize=true）")
	case in.Append && in.Overwrite:
		return nil, errors.New("mode=append 不能与 overwrite 同用（overwrite 只属于整册覆盖）")
	case formB && len(in.Lines) == 0:
		return nil, errors.New("outline 形态需要源文件内容（source_file）")
	}
	name := strings.TrimSpace(in.Meta.Name)
	if name == "" {
		name = title // 方案名=标题：唯一键直接用标题，不经 slug 截断（长标题同前缀会撞名）
	}
	if in.DryRun { // 演练：后续所有经 s 的写入都带着「不落图片文件」标记
		dry := *s
		dry.noWrite = true
		s = &dry
	}
	srcFile := strings.TrimSpace(in.SourceFile)
	if srcFile == "" {
		srcFile = strings.TrimSpace(in.Meta.SourceFile)
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, unreadable(err)
	}
	defer func() { _ = tx.Rollback() }()

	existing, err := scanProposal(tx.QueryRow(`SELECT `+proposalCols+` FROM proposals WHERE name=?`, name))
	hasExisting := err == nil
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, unreadable(err)
	}
	var oldSections []Section
	if hasExisting {
		if in.Append && formB && existing.Status == StatusActive && existing.IngestLine > 0 {
			return nil, fmt.Errorf("《%s》已录入完成，不能再追加标题；要改结构请整册重录（mode=replace + overwrite=true）", title)
		}
		oldSections, err = querySectionsTx(tx, existing.ID, false)
		if err != nil {
			return nil, unreadable(err)
		}
	}
	if in.Append && !hasExisting {
		return nil, fmt.Errorf("方案库没有《%s》：追加前先用 mode=replace 新建", title)
	}
	if !in.Append && hasExisting && !in.Overwrite {
		return nil, &ExistsError{Name: existing.Name, Title: existing.Title, Source: existing.Source, Sections: len(oldSections)}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	meta := in.Meta
	meta.Title = title
	if hasExisting && in.Append { // 追加只补非空字段
		meta.Name = existing.Name
		fill := func(v, old string) string {
			if strings.TrimSpace(v) == "" {
				return old
			}
			return v
		}
		meta.ProjectType, meta.Year = fill(meta.ProjectType, existing.ProjectType), fill(meta.Year, existing.Year)
		meta.Source, meta.Summary = fill(meta.Source, existing.Source), fill(meta.Summary, existing.Summary)
		meta.Workspace, meta.SourceFile = fill(meta.Workspace, existing.Workspace), fill(srcFile, existing.SourceFile)
	} else {
		meta.SourceFile = srcFile
	}
	pid := int64(0)
	created, replaced := !hasExisting, false
	var oldAssetRefs []assetDiskRef
	if created {
		err = tx.QueryRow(`INSERT INTO proposals(name,title,project_type,year,source,workspace,status,summary,source_file,created_at,updated_at)
VALUES(?,?,?,?,?,?,?,?,?,?,?) RETURNING id`,
			name, title, meta.ProjectType, meta.Year, meta.Source, meta.Workspace, StatusActive, meta.Summary, meta.SourceFile, now, now).Scan(&pid)
		if err != nil {
			return nil, fmt.Errorf("保存方案 %s: %w", title, err)
		}
	} else {
		pid = existing.ID
		if !in.Append { // 整册覆盖：旧小节、全文索引、语义块全部让位
			replaced = len(oldSections) > 0
			oldAssetRefs, err = assetRefsTx(tx, pid)
			if err != nil {
				return nil, unreadable(err)
			}
			if _, err := tx.Exec(`DELETE FROM proposal_fts WHERE rowid IN (SELECT id FROM proposal_sections WHERE proposal_id=?)`, pid); err != nil {
				return nil, err
			}
			if _, err := tx.Exec(`DELETE FROM proposal_assets WHERE proposal_id=?`, pid); err != nil {
				return nil, err
			}
			if _, err := tx.Exec(`DELETE FROM proposal_sections WHERE proposal_id=?`, pid); err != nil {
				return nil, err
			}
			oldSections = nil
		}
		if _, err := tx.Exec(`UPDATE proposals SET title=?,project_type=?,year=?,source=?,workspace=?,summary=?,source_file=?,updated_at=? WHERE id=?`,
			title, meta.ProjectType, meta.Year, meta.Source, meta.Workspace, meta.Summary, meta.SourceFile, now, pid); err != nil {
			return nil, fmt.Errorf("保存方案 %s: %w", title, err)
		}
	}

	// 续录游标：上一批末个标题所在行；其后到本批首个标题之间的正文属于上一批的末节。
	cursor, pendingID, pendingPath := 0, int64(0), ""
	if in.Append && hasExisting && existing.Status == StatusIngesting && existing.IngestLine > 0 {
		cursor = existing.IngestLine
		if n := len(oldSections); n > 0 {
			pendingID, pendingPath = oldSections[n-1].ID, oldSections[n-1].TitlePath
		}
		if formA {
			return nil, fmt.Errorf("《%s》上一批按源文件录入、末节正文待截取：请用同一 source_file 续录（mode=append + outline），或先 mode=append + finalize=true 收尾", title)
		}
		if len(in.Lines) > 0 {
			if cursor > len(in.Lines) || prefixHash(in.Lines, cursor) != existing.IngestHash {
				return nil, fmt.Errorf("源文件在录入期间被修改（前 %d 行与已录内容不一致），不能续录；请整册重录（mode=replace + overwrite=true）", cursor)
			}
		}
	}

	newSecs := make([]Section, 0, max(len(in.Sections), len(in.Outline)))
	var weak, fixes []string
	if formA {
		for i, sec := range in.Sections {
			t := strings.TrimSpace(sec.Title)
			if t == "" {
				return nil, fmt.Errorf("sections 第 %d 项缺 title", i+1)
			}
			newSecs = append(newSecs, Section{Level: sec.Level, Title: truncRunes(t, MaxTitleRunes), Body: sec.Body})
		}
	} else {
		// 逐项顺序定位：每一项从上一项标题行的下一行开始找——标题在文档里是有序的，
		// 各章重复出现的标题（如「概况」）才不会互相串位。
		searchFrom := cursor
		for i, e := range in.Outline {
			t := strings.TrimSpace(e.Title)
			if t == "" {
				return nil, fmt.Errorf("outline 第 %d 项缺 title", i+1)
			}
			idx, grade, err := findHeading(in.Lines, searchFrom, t, e.Line)
			display := strings.TrimSpace(e.DisplayTitle)
			if err != nil && display != "" {
				// 错字容错（2026-11-27 AI 反馈 #11）：模型给的是修正后的写法，源文件是
				// OCR 错字。指了行号就认那一行（须像同一个标题），否则在标题样行里找
				// 与显示名最接近的一行。
				fixed := -1
				if e.Line > 0 && e.Line-1 >= searchFrom && e.Line-1 < len(in.Lines) {
					if typoDistance(in.Lines[e.Line-1], display) >= 0 {
						fixed = e.Line - 1
					}
				} else if j, _ := findHeadingByDisplay(in.Lines, searchFrom, display); j >= 0 {
					fixed = j
				}
				if fixed >= 0 {
					idx, grade, err = fixed, 2, nil
				}
			}
			if err != nil {
				return nil, err
			}
			src := normHeading(in.Lines[idx])
			switch {
			case display != "":
				// 入库用显示名；源文件行带编号而显示名没带时把编号补回去，标题树里
				// 编号才齐。
				t = display
				if num := leadingNumber(src); num != "" && leadingNumber(normHeading(display)) == "" {
					t = strings.Join(strings.Fields(num+" "+display), " ")
				}
				fixes = append(fixes, fmt.Sprintf("第 %d 项：源文件「%s」→ 入库「%s」（第 %d 行）", i+1, src, t, idx+1))
			case src != "" && grade >= 3:
				// 标题取源文件原行的标题文本（保留文档自带编号）。定位到的只是「行里提到
				// 它」的正文句或长行时，沿用模型给的写法并点名让模型核对行号——短标题行
				// （无 Markdown 标记的纯文本文档很常见）不算弱匹配，免得整批都是提示。
				t = src
			}
			if grade <= 1 {
				weak = append(weak, fmt.Sprintf("第 %d 项 %q → 第 %d 行：%s", i+1, t, idx+1, truncRunes(strings.TrimSpace(in.Lines[idx]), 60)))
			}
			newSecs = append(newSecs, Section{Level: e.Level, Title: truncRunes(t, MaxTitleRunes), SrcLine: idx + 1})
			searchFrom = idx + 1
		}
	}

	prev, clamped := 0, 0
	if n := len(oldSections); n > 0 {
		prev = oldSections[n-1].Level
	}
	for i := range newSecs {
		lv := newSecs[i].Level
		if lv < 1 {
			lv = 1
		}
		if lv > MaxLevels {
			lv = MaxLevels
			clamped++
		}
		if lv > prev+1 {
			return nil, fmt.Errorf("第 %d 节 %q 的 level=%d 相对前一节跳级（前一节 level=%d）：标题须父级在前、逐级展开，且第一项为 1 级",
				len(oldSections)+i+1, newSecs[i].Title, newSecs[i].Level, prev)
		}
		newSecs[i].Level = lv
		prev = lv
	}

	done, skipped, ingestLine := true, 0, 0
	ingestHash, pendingTitle := "", ""
	assetCount := 0
	var missingAssets []string
	if formB {
		h := make([]int, len(newSecs))
		for i := range newSecs {
			h[i] = newSecs[i].SrcLine - 1
		}
		if pendingID != 0 {
			added, missing, err := s.setSectionBodyTx(tx, pendingID, pid, pendingPath, joinLines(in.Lines[cursor:h[0]]), in.AssetBaseDir, now)
			if err != nil {
				return nil, err
			}
			assetCount += added
			missingAssets = append(missingAssets, missing...)
		} else if cursor == 0 && h[0] > 0 {
			skipped = h[0] // 封面、目录等首个标题之前的内容不入库（目录会重复全部标题，进了反而是噪声）
		}
		for i := 0; i < len(newSecs)-1; i++ {
			newSecs[i].Body = joinLines(in.Lines[h[i]+1 : h[i+1]])
		}
		last := len(newSecs) - 1
		rest := in.Lines[h[last]+1:]
		done = in.Finalize || !hasContent(rest)
		if done {
			newSecs[last].Body = joinLines(rest)
		} else {
			pendingTitle = newSecs[last].Title
		}
		ingestLine = h[last] + 1
		ingestHash = prefixHash(in.Lines, ingestLine)
	} else if in.Append && in.Finalize && pendingID != 0 && len(in.Lines) > 0 {
		// 只收尾：待截取的末节正文截至文件尾。
		added, missing, err := s.setSectionBodyTx(tx, pendingID, pid, pendingPath, joinLines(in.Lines[cursor:]), in.AssetBaseDir, now)
		if err != nil {
			return nil, err
		}
		assetCount += added
		missingAssets = append(missingAssets, missing...)
	}
	if !formB {
		// 形态 A 不用源文件游标：整册覆盖后旧游标作废；追加时原样保留。
		ingestLine, ingestHash = 0, ""
		if in.Append && hasExisting {
			ingestLine, ingestHash = existing.IngestLine, existing.IngestHash
		}
	}
	preparedAssets := make([][]proposalAsset, len(newSecs))
	for i := range newSecs {
		prepared, err := s.prepareBodyAssets(newSecs[i].Body, in.AssetBaseDir)
		if err != nil {
			return nil, fmt.Errorf("托管小节 %q 的图片: %w", newSecs[i].Title, err)
		}
		newSecs[i].Body = prepared.Body
		preparedAssets[i] = prepared.Assets
		assetCount += len(prepared.Assets)
		missingAssets = append(missingAssets, prepared.Missing...)
		if n := len([]rune(newSecs[i].Body)); n > MaxBodyChars {
			return nil, fmt.Errorf("小节 %q 正文 %d 字超过单节上限 %d 字：请在该节范围内多列一些下级标题后再录入", newSecs[i].Title, n, MaxBodyChars)
		}
	}
	missingAssets = uniqueStrings(missingAssets)

	// 标题路径与父节点：按 level 维护栈，续录时接续已存末节的栈。
	var stackID [MaxLevels + 1]int64
	var stackTitle [MaxLevels + 1]string
	rebuildStack := func() {
		for _, es := range oldSections {
			stackID[es.Level], stackTitle[es.Level] = es.ID, es.Title
			for lv := es.Level + 1; lv <= MaxLevels; lv++ {
				stackID[lv], stackTitle[lv] = 0, ""
			}
		}
	}
	rebuildStack()
	seq := len(oldSections)
	for i := range newSecs {
		sec := &newSecs[i]
		stackTitle[sec.Level] = sec.Title
		for lv := sec.Level + 1; lv <= MaxLevels; lv++ {
			stackID[lv], stackTitle[lv] = 0, ""
		}
		parts := make([]string, 0, sec.Level)
		for lv := 1; lv <= sec.Level; lv++ {
			parts = append(parts, stackTitle[lv])
		}
		sec.TitlePath, sec.ParentID, sec.Seq = strings.Join(parts, " › "), stackID[sec.Level-1], seq
		res, err := tx.Exec(`INSERT INTO proposal_sections(proposal_id, parent_id, level, seq, title, title_path, body, src_line, created_at, updated_at)
VALUES(?,?,?,?,?,?,?,?,?,?)`,
			pid, sec.ParentID, sec.Level, sec.Seq, sec.Title, sec.TitlePath, sec.Body, sec.SrcLine, now, now)
		if err != nil {
			return nil, fmt.Errorf("写入小节 %s: %w", sec.Title, err)
		}
		sid, _ := res.LastInsertId()
		sec.ID = sid
		stackID[sec.Level] = sid
		if err := s.insertAssetsTx(tx, pid, sid, preparedAssets[i], now); err != nil {
			return nil, fmt.Errorf("写入小节 %s 的图片资产: %w", sec.Title, err)
		}
		if err := insertFTS(tx, sid, sec.Title, sec.TitlePath, sec.Body, pid); err != nil {
			return nil, fmt.Errorf("索引小节 %s: %w", sec.Title, err)
		}
		if err := insertChunks(tx, pid, sid, sec.TitlePath, sec.Body, now); err != nil {
			return nil, fmt.Errorf("切分小节 %s: %w", sec.Title, err)
		}
		seq++
	}

	status := StatusActive
	if !done {
		status = StatusIngesting
	}
	if _, err := tx.Exec(`UPDATE proposals SET status=?, ingest_line=?, ingest_hash=?, updated_at=? WHERE id=?`,
		status, ingestLine, ingestHash, now, pid); err != nil {
		return nil, err
	}
	if in.DryRun {
		// 演练到此为止：校验、定位、切正文、算层级都已走完，库与图片都不落。
		// 统计必须在回滚之前、在同一事务里读——读的才是「假如写入」的结构。
		stats, _ := statsOf(tx, pid)
		_ = tx.Rollback()
		res := &SaveResult{
			ProposalID: pid, Created: created, Replaced: replaced, Added: len(newSecs), Clamped: clamped,
			SkippedLines: skipped, IngestLine: ingestLine, TotalLines: len(in.Lines), Done: done, PendingTitle: pendingTitle,
			WeakMatches: weak, TitleFixes: fixes, Assets: assetCount, MissingAssets: missingAssets, DryRun: true,
		}
		res.Stats = stats
		return res, nil
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	s.gcAssetRefs(oldAssetRefs)
	res := &SaveResult{
		ProposalID: pid, Created: created, Replaced: replaced, Added: len(newSecs), Clamped: clamped,
		SkippedLines: skipped, IngestLine: ingestLine, TotalLines: len(in.Lines), Done: done, PendingTitle: pendingTitle,
		WeakMatches: weak, TitleFixes: fixes, Assets: assetCount, MissingAssets: missingAssets,
	}
	res.Stats, err = s.Stats(pid)
	if err != nil {
		return res, err
	}
	return res, nil
}

// Delete 删除方案（小节/语义块随外键级联；全文索引显式清）。
func (s *Store) Delete(name string) error {
	if err := s.available(); err != nil {
		return err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return unreadable(err)
	}
	defer func() { _ = tx.Rollback() }()
	var pid int64
	if err := tx.QueryRow(`SELECT id FROM proposals WHERE name=?`, name).Scan(&pid); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("方案 %q 不存在", name)
		}
		return unreadable(err)
	}
	oldAssetRefs, err := assetRefsTx(tx, pid)
	if err != nil {
		return unreadable(err)
	}
	if _, err := tx.Exec(`DELETE FROM proposal_fts WHERE rowid IN (SELECT s.id FROM proposal_sections s JOIN proposals p ON p.id=s.proposal_id WHERE p.name=?)`, name); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM proposal_assets WHERE proposal_id=?`, pid); err != nil {
		return err
	}
	res, err := tx.Exec(`DELETE FROM proposals WHERE name=?`, name)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("方案 %q 不存在", name)
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	s.gcAssetRefs(oldAssetRefs)
	return nil
}

// ── 读取面 ─────────────────────────────────────────────────────────

func querySectionsTx(q querier, pid int64, withBody bool) ([]Section, error) {
	cols := `id, parent_id, level, seq, title, title_path, src_line, LENGTH(body)`
	if withBody {
		cols = `id, parent_id, level, seq, title, title_path, src_line, LENGTH(body), body`
	}
	rows, err := q.Query(`SELECT `+cols+` FROM proposal_sections WHERE proposal_id=? ORDER BY seq`, pid)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []Section
	for rows.Next() {
		var sec Section
		dest := []any{&sec.ID, &sec.ParentID, &sec.Level, &sec.Seq, &sec.Title, &sec.TitlePath, &sec.SrcLine, &sec.Chars}
		if withBody {
			dest = append(dest, &sec.Body)
		}
		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}
		out = append(out, sec)
	}
	return out, rows.Err()
}

type querier interface {
	Query(query string, args ...any) (*sql.Rows, error)
	Exec(query string, args ...any) (sql.Result, error)
}

// Outline 返回整册标题树（不带正文；Chars 为各节字数）。
func (s *Store) Outline(pid int64) ([]Section, error) {
	if err := s.available(); err != nil {
		return nil, err
	}
	secs, err := querySectionsTx(s.db, pid, false)
	if err != nil {
		return nil, unreadable(err)
	}
	return secs, nil
}

// Stats 返回一份方案的结构统计。
func (s *Store) Stats(pid int64) (Stats, error) {
	if err := s.available(); err != nil {
		return Stats{}, err
	}
	return statsOf(s.db, pid)
}

// rowQuerier 是 *sql.DB 与 *sql.Tx 的公共面：事务内外同一份统计实现（演练在回滚前读）。
type rowQuerier interface {
	QueryRow(query string, args ...any) *sql.Row
}

func statsOf(q rowQuerier, pid int64) (Stats, error) {
	var st Stats
	err := q.QueryRow(`SELECT COUNT(*), COALESCE(SUM(LENGTH(body)),0),
  COALESCE(SUM(level=1),0), COALESCE(SUM(level=2),0), COALESCE(SUM(level=3),0), COALESCE(SUM(level=4),0)
FROM proposal_sections WHERE proposal_id=?`, pid).Scan(&st.Sections, &st.Chars,
		&st.LevelCounts[1], &st.LevelCounts[2], &st.LevelCounts[3], &st.LevelCounts[4])
	if err != nil {
		return st, unreadable(err)
	}
	_ = q.QueryRow(`SELECT title, LENGTH(body) FROM proposal_sections WHERE proposal_id=? ORDER BY LENGTH(body) DESC, seq LIMIT 1`, pid).
		Scan(&st.LongestTitle, &st.LongestChars)
	return st, nil
}

// Longest 返回字数最多的 n 个小节（录入回执用：过长的节提示模型补下级标题）。
func (s *Store) Longest(pid int64, n int) ([]Section, error) {
	if err := s.available(); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(`SELECT id, level, title, title_path, LENGTH(body) FROM proposal_sections WHERE proposal_id=? ORDER BY LENGTH(body) DESC, seq LIMIT ?`, pid, n)
	if err != nil {
		return nil, unreadable(err)
	}
	defer func() { _ = rows.Close() }()
	var out []Section
	for rows.Next() {
		var sec Section
		if err := rows.Scan(&sec.ID, &sec.Level, &sec.Title, &sec.TitlePath, &sec.Chars); err != nil {
			return nil, unreadable(err)
		}
		out = append(out, sec)
	}
	return out, rows.Err()
}

// ReadOpt 阅读选项。
type ReadOpt struct {
	Scope       string // ""/self（默认，只读本节）| subtree（连全部子孙节）
	Offset      int    // rune 偏移（无 Query 时分页用）
	Limit       int    // rune 上限（无 Query 时；0=DefaultReadLimit，ReadAll=整节不截断）
	Query       string // 关键词：返回命中段落而非顺序正文
	MaxPassages int    // 最多几个命中段落
}

const (
	DefaultReadLimit = 12000
	MaxReadLimit     = 30000
	// ReadAll 作为 ReadOpt.Limit：不截断，一次返回整节（界面阅读窗用——人读的
	// 页面不该悄悄砍掉超长小节的后半截）。
	ReadAll         = -1
	DefaultPassages = 6
	MaxPassages     = 12
)

// ReadPart 是 subtree 拼接文本里的一节：Start 为该节正文在拼接文本中的 rune 起点。
type ReadPart struct {
	SectionID int64  `json:"sectionId"`
	Level     int    `json:"level"`
	Title     string `json:"title"`
	Start     int    `json:"start"`
	Chars     int    `json:"chars"`
}

// ReadPassage 是一段命中（带【】标记），Start 为它在拼接文本中的 rune 起点。
type ReadPassage struct {
	SectionID int64    `json:"sectionId"`
	Title     string   `json:"title"`
	Start     int      `json:"start"`
	Text      string   `json:"text"`
	Terms     []string `json:"terms"`
}

// ReadResult 阅读回执数据。
type ReadResult struct {
	Proposal Proposal
	Section  Section    // 目标节（Body 为正文）
	Parts    []ReadPart // self + 子孙节（scope=subtree 时多于一节）
	Text     string     // 分页正文（无 Query 时）
	Total    int        // 拼接文本总字数
	Offset   int
	HasMore  bool
	Passages []ReadPassage
	Children []Child
}

// Read 读一个小节：按关键词取命中段落，或顺序分页；可连子孙节整读。
func (s *Store) Read(sectionID int64, opt ReadOpt) (*ReadResult, error) {
	if err := s.available(); err != nil {
		return nil, err
	}
	var sec Section
	var pid int64
	err := s.db.QueryRow(`SELECT id, proposal_id, parent_id, level, seq, title, title_path, body, src_line, LENGTH(body)
FROM proposal_sections WHERE id=?`, sectionID).Scan(&sec.ID, &pid, &sec.ParentID, &sec.Level, &sec.Seq, &sec.Title,
		&sec.TitlePath, &sec.Body, &sec.SrcLine, &sec.Chars)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("小节 %d 不存在（方案可能已被覆盖重录，请重新 proposal_search 取新的 section_id）", sectionID)
	}
	if err != nil {
		return nil, unreadable(err)
	}
	p, err := scanProposal(s.db.QueryRow(`SELECT `+proposalCols+` FROM proposals WHERE id=?`, pid))
	if err != nil {
		return nil, unreadable(err)
	}
	outline, err := s.Outline(pid)
	if err != nil {
		return nil, err
	}
	res := &ReadResult{Proposal: p, Section: sec}
	idx := -1
	for i := range outline {
		if outline[i].ID == sectionID {
			idx = i
		}
	}
	res.Children = childrenOf(outline, idx)

	// 拼接文本：目标节 +（subtree 时）全部子孙节，各自记录 rune 起点。
	parts := []Section{sec}
	if opt.Scope == "subtree" && idx >= 0 {
		for j := idx + 1; j < len(outline) && outline[j].Level > sec.Level; j++ {
			parts = append(parts, outline[j])
		}
		if len(parts) > 1 {
			ids := make([]string, 0, len(parts)-1)
			args := make([]any, 0, len(parts)-1)
			for _, ps := range parts[1:] {
				ids = append(ids, "?")
				args = append(args, ps.ID)
			}
			rows, err := s.db.Query(`SELECT id, body FROM proposal_sections WHERE id IN (`+strings.Join(ids, ",")+`)`, args...)
			if err != nil {
				return nil, unreadable(err)
			}
			bodies := map[int64]string{}
			for rows.Next() {
				var id int64
				var body string
				if err := rows.Scan(&id, &body); err != nil {
					_ = rows.Close()
					return nil, unreadable(err)
				}
				bodies[id] = body
			}
			_ = rows.Close()
			if err := rows.Err(); err != nil {
				return nil, unreadable(err)
			}
			for i := 1; i < len(parts); i++ {
				parts[i].Body = bodies[parts[i].ID]
				parts[i].Chars = len([]rune(parts[i].Body))
			}
		}
	}
	var sb strings.Builder
	for i, ps := range parts {
		if i > 0 {
			sb.WriteString("\n\n")
		}
		res.Parts = append(res.Parts, ReadPart{SectionID: ps.ID, Level: ps.Level, Title: ps.Title, Start: len([]rune(sb.String())), Chars: ps.Chars})
		sb.WriteString(ps.Body)
	}
	text := sb.String()
	rs := []rune(text)
	res.Total = len(rs)

	terms := SplitTerms(opt.Query)
	if len(terms) > 0 {
		maxN := opt.MaxPassages
		if maxN <= 0 {
			maxN = DefaultPassages
		}
		if maxN > MaxPassages {
			maxN = MaxPassages
		}
		for _, ps := range FindPassages(text, terms, 220, 320, maxN) {
			owner := parts[0]
			for _, rp := range res.Parts {
				if rp.Start <= ps.Start {
					owner = Section{ID: rp.SectionID, Title: rp.Title}
				}
			}
			res.Passages = append(res.Passages, ReadPassage{
				SectionID: owner.ID, Title: owner.Title, Start: ps.Start,
				Text: RenderPassage(rs, ps, terms), Terms: ps.Terms,
			})
		}
		return res, nil
	}
	if opt.Offset < 0 {
		opt.Offset = 0
	}
	limit := opt.Limit
	if limit == 0 {
		limit = DefaultReadLimit
	} else if limit < 0 {
		limit = len(rs) // ReadAll：整节不截断
	}
	if limit > MaxReadLimit && opt.Limit >= 0 {
		limit = MaxReadLimit
	}
	if opt.Offset > len(rs) {
		opt.Offset = len(rs)
	}
	end := opt.Offset + limit
	if end > len(rs) {
		end = len(rs)
	}
	res.Offset, res.Text, res.HasMore = opt.Offset, string(rs[opt.Offset:end]), end < len(rs)
	return res, nil
}

// childrenOf 取 outline[idx] 的直接子节（下一个同级或更高级标题之前）。
func childrenOf(outline []Section, idx int) []Child {
	if idx < 0 || idx >= len(outline) {
		return nil
	}
	var out []Child
	for j := idx + 1; j < len(outline) && outline[j].Level > outline[idx].Level; j++ {
		if outline[j].Level == outline[idx].Level+1 {
			out = append(out, Child{ID: outline[j].ID, Level: outline[j].Level, Title: outline[j].Title, Chars: outline[j].Chars})
		}
	}
	return out
}

// 检索（SearchInput / Hit / SearchResult / Search 与分面 / 翻页）在 search.go。

// ── 内部写入辅助 ───────────────────────────────────────────────────

func insertFTS(tx *sql.Tx, sid int64, title, titlePath, body string, pid int64) error {
	_, err := tx.Exec(`INSERT INTO proposal_fts(rowid, title, body, proposal_id, section_id) VALUES(?,?,?,?,?)`,
		sid, title, titlePath+" "+body, pid, sid)
	return err
}

// setSectionBodyTx 改写一个已存小节的正文（续录时截取上一批末节的正文），同步
// 全文索引与语义块（该节的向量需重算）。
func (s *Store) setSectionBodyTx(tx *sql.Tx, sid, pid int64, titlePath, body, assetBaseDir, now string) (int, []string, error) {
	prepared, err := s.prepareBodyAssets(body, assetBaseDir)
	if err != nil {
		return 0, nil, err
	}
	body = prepared.Body
	if _, err := tx.Exec(`UPDATE proposal_sections SET body=?, updated_at=? WHERE id=?`, body, now, sid); err != nil {
		return 0, nil, err
	}
	if _, err := tx.Exec(`DELETE FROM proposal_assets WHERE section_id=?`, sid); err != nil {
		return 0, nil, err
	}
	if err := s.insertAssetsTx(tx, pid, sid, prepared.Assets, now); err != nil {
		return 0, nil, err
	}
	if _, err := tx.Exec(`DELETE FROM proposal_fts WHERE rowid=?`, sid); err != nil {
		return 0, nil, err
	}
	var title string
	if err := tx.QueryRow(`SELECT title FROM proposal_sections WHERE id=?`, sid).Scan(&title); err != nil {
		return 0, nil, err
	}
	if err := insertFTS(tx, sid, title, titlePath, body, pid); err != nil {
		return 0, nil, err
	}
	if _, err := tx.Exec(`DELETE FROM proposal_chunks WHERE section_id=?`, sid); err != nil {
		return 0, nil, err
	}
	if err := insertChunks(tx, pid, sid, titlePath, body, now); err != nil {
		return 0, nil, err
	}
	return len(prepared.Assets), prepared.Missing, nil
}

func joinLines(lines []string) string { return strings.TrimSpace(strings.Join(lines, "\n")) }

// hasContent 判定这些行里是否还有非空内容（全空=末节正文已到文件尾，可自动收尾）。
func hasContent(lines []string) bool {
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			return true
		}
	}
	return false
}

func uniqueStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}
