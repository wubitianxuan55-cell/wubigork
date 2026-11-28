package app

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/proposal"
	"github.com/gaea/gaea/internal/gaea/tool/builtin"
)

// 方案库板块（一级板块「方案库」）的绑定面。录入（整册重录、从文件导入）只在办公
// 方案模式里由模型经 proposal_save 完成；板块这里给人看、检索、直接修正入库错误：
// 2026-11-27 加了元数据与小节编辑，2026-11-28 加结构修正（GaeaProposalSectionOp）、
// 分面翻页检索、同题对照、索引体检 / 重建与导出。
// 检索与工具面同一条链（全文 + 本地语义召回 + 本地精排），本地模型依赖取
// builtin.ProposalSemantic——与 proposal_search 同模型，向量索引不会两边互清。

// proposalSearchTimeout 界面检索的时间上限（含检索前补齐向量化与精排）。
const proposalSearchTimeout = 30 * time.Second

// hubProposalStore 打开方案库（与 proposal_save/search/read 三工具同库）。
func (a *App) hubProposalStore() (*proposal.Store, error) {
	return proposal.OpenWithAssets(db.GetDatabase(config.MemoryUserDir()), config.MemoryUserDir())
}

// ProposalSummaryView 方案清单的一行：元数据 + 规模 + 语义索引进度。
type ProposalSummaryView struct {
	Name        string `json:"name"`
	Title       string `json:"title"`
	ProjectType string `json:"projectType"`
	Year        string `json:"year"`
	Status      string `json:"status"`
	Summary     string `json:"summary"`
	Source      string `json:"source"`
	SourceFile  string `json:"sourceFile"`
	UpdatedAt   string `json:"updatedAt"`
	Sections    int    `json:"sections"`
	Chars       int    `json:"chars"`
	IndexDone   int    `json:"indexDone"`
	IndexTotal  int    `json:"indexTotal"`
}

// ProposalSectionView 标题树的一个节点（不含正文）。
type ProposalSectionView struct {
	ID        int64  `json:"id"`
	ParentID  int64  `json:"parentId"`
	Level     int    `json:"level"`
	Seq       int    `json:"seq"`
	Title     string `json:"title"`
	TitlePath string `json:"titlePath"`
	Chars     int    `json:"chars"`
}

// ProposalSectionBodyView 某小节的正文与定位信息。
type ProposalSectionBodyView struct {
	ID            int64            `json:"id"`
	ProposalTitle string           `json:"proposalTitle"`
	Status        string           `json:"status"`
	Title         string           `json:"title"`
	TitlePath     string           `json:"titlePath"`
	Chars         int              `json:"chars"`
	Body          string           `json:"body"`
	Children      []proposal.Child `json:"children"`
}

// ProposalSearchResultView 界面检索结果。Total / ByProposal / ByChapter 让界面能翻页、
// 按方案或章收窄（常见词「监测」全库几百处，只给前 30 条不够用）；Relaxed 注明多词
// 全含无结果、已放宽为任一词；SearchError 是语义路失败的原因（本地模型不可用）。
type ProposalSearchResultView struct {
	Hits        []proposal.Hit   `json:"hits"`
	Mode        string           `json:"mode"`
	Note        string           `json:"note"`
	Total       int              `json:"total"`
	Offset      int              `json:"offset"`
	ByProposal  []proposal.Facet `json:"byProposal"`
	ByChapter   []proposal.Facet `json:"byChapter"`
	Relaxed     bool             `json:"relaxed"`
	SearchError string           `json:"searchError"`
}

// GaeaProposalList 返回方案清单（更新时间降序）。读失败如实报错，不当空库。
func (a *App) GaeaProposalList() ([]ProposalSummaryView, error) {
	st, err := a.hubProposalStore()
	if err != nil {
		return []ProposalSummaryView{}, err
	}
	list, err := st.List()
	if err != nil {
		return []ProposalSummaryView{}, err
	}
	out := make([]ProposalSummaryView, 0, len(list))
	for _, p := range list {
		out = append(out, ProposalSummaryView{
			Name: p.Name, Title: p.Title, ProjectType: p.ProjectType, Year: p.Year,
			Status: p.Status, Summary: p.Summary, Source: p.Source, SourceFile: p.SourceFile,
			UpdatedAt:  p.UpdatedAt.Local().Format("2006-01-02 15:04"),
			Sections:   p.Sections,
			Chars:      p.Chars,
			IndexDone:  p.IndexDone,
			IndexTotal: p.IndexTotal,
		})
	}
	return out, nil
}

// GaeaProposalOutline 返回某方案（name 或标题）的标题树（不含正文）；方案不
// 存在时返回 error——界面上选中的方案可能已被覆盖重录或删除。
func (a *App) GaeaProposalOutline(name string) ([]ProposalSectionView, error) {
	st, err := a.hubProposalStore()
	if err != nil {
		return []ProposalSectionView{}, err
	}
	p, err := st.Resolve(name)
	if err != nil {
		return []ProposalSectionView{}, err
	}
	if p == nil {
		return []ProposalSectionView{}, fmt.Errorf("方案库里没有《%s》", name)
	}
	secs, err := st.Outline(p.ID)
	if err != nil {
		return []ProposalSectionView{}, err
	}
	out := make([]ProposalSectionView, 0, len(secs))
	for _, s := range secs {
		out = append(out, ProposalSectionView{
			ID: s.ID, ParentID: s.ParentID, Level: s.Level, Seq: s.Seq,
			Title: s.Title, TitlePath: s.TitlePath, Chars: s.Chars,
		})
	}
	return out, nil
}

// GaeaProposalRead 返回某小节的完整正文（ReadAll：界面阅读窗自己滚动，不按页截断）。
func (a *App) GaeaProposalRead(sectionID int64) (*ProposalSectionBodyView, error) {
	st, err := a.hubProposalStore()
	if err != nil {
		return nil, err
	}
	r, err := st.Read(sectionID, proposal.ReadOpt{Limit: proposal.ReadAll})
	if err != nil {
		return nil, err
	}
	return &ProposalSectionBodyView{
		ID:            r.Section.ID,
		ProposalTitle: r.Proposal.Title,
		Status:        r.Proposal.Status,
		Title:         r.Section.Title,
		TitlePath:     r.Section.TitlePath,
		Chars:         r.Total,
		Body:          r.Text,
		Children:      r.Children,
	}, nil
}

// GaeaProposalSearch 界面检索：全文 + 本地语义召回 + 本地精排，命中回到小节
// 并带命中位置摘要。scope="title" 只查标题树（章节骨架参照），空串为全文+语义；
// level/projectType/year/within（限定方案：name 或标题）均可选，空值不过滤。
// under=限定在某个标题之下（含其全部子孙小节）；offset=翻页跳过前几条（按最终排序）；
// sort="doc" 按文档顺序（方案 × 标题序）列，空串按相关度。
// （形参名用 within：叫 proposal 会遮住同名的存储层包名。）
func (a *App) GaeaProposalSearch(query string, scope string, level int, projectType string, year string, within string, under int64, offset int, sort string, limit int) (*ProposalSearchResultView, error) {
	st, err := a.hubProposalStore()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), proposalSearchTimeout)
	defer cancel()
	res, err := st.HybridSearch(ctx, proposal.SearchInput{
		Query: query, Scope: scope, Level: level, ProjectType: projectType, Year: year, Proposal: within,
		Under: under, Offset: offset, Sort: sort, TopN: limit, Facets: true,
	}, builtin.ProposalSemantic())
	if err != nil {
		return nil, err
	}
	return &ProposalSearchResultView{
		Hits: res.Hits, Mode: res.Mode, Note: res.Note,
		Total: res.Total, Offset: res.Offset, ByProposal: res.ByProposal, ByChapter: res.ByChapter,
		Relaxed: res.Relaxed, SearchError: res.SearchError,
	}, nil
}

// GaeaProposalUpdateMeta 修改方案元数据（标题/类型/年度/摘要）。界面编辑表单
// 预填现值、四个字段全量替换落库（走 store.UpdateMeta：全文/语义/小节树不动）。
// 标题即方案名，改名后唯一键随之变——返回新方案名，界面按它重新定位选中行。
func (a *App) GaeaProposalUpdateMeta(name, title, projectType, year, summary string) (string, error) {
	st, err := a.hubProposalStore()
	if err != nil {
		return "", err
	}
	p, err := st.UpdateMeta(name, proposal.UpdateInput{
		Title: &title, ProjectType: &projectType, Year: &year, Summary: &summary,
	})
	if err != nil {
		return "", err
	}
	return p.Name, nil
}

// GaeaProposalUpdateSection 增量修正一个小节（改标题连带子树标题路径；改正文
// 整节替换并置空语义向量待重嵌）。updateTitle/updateBody 都为 false 是调用错误，
// 如实报错；没改的那半边不传，避免白重嵌向量。
func (a *App) GaeaProposalUpdateSection(sectionID int64, title string, updateTitle bool, body string, updateBody bool) error {
	if !updateTitle && !updateBody {
		return fmt.Errorf("没有要改的内容：标题与正文至少勾选一个")
	}
	st, err := a.hubProposalStore()
	if err != nil {
		return err
	}
	in := proposal.SectionEdit{}
	if updateTitle {
		in.Title = &title
	}
	if updateBody {
		in.Body = &body
	}
	_, err = st.UpdateSection(sectionID, in)
	return err
}

// GaeaProposalDelete 删除方案（小节/语义块/全文索引级联清理）。
func (a *App) GaeaProposalDelete(name string) error {
	st, err := a.hubProposalStore()
	if err != nil {
		return err
	}
	return st.Delete(name)
}

// GaeaProposalAssetDataURL 把正文里的 gaea-asset://<sha256> 解析为浏览器可显示
// 的 data URL。资产只从提案托管目录读取，不接受任意磁盘路径。
func (a *App) GaeaProposalAssetDataURL(ref string) (string, error) {
	st, err := a.hubProposalStore()
	if err != nil {
		return "", err
	}
	return st.AssetDataURL(ref)
}

// ── 2026-11-28 索引与编辑优化批 ──────────────────────────────────────

// ProposalStructureView 结构操作回执。
type ProposalStructureView struct {
	Op            string   `json:"op"`
	FocusID       int64    `json:"focusId"`
	Affected      int      `json:"affected"`
	Created       int      `json:"created"`
	Removed       int      `json:"removed"`
	RemovedChars  int      `json:"removedChars"`
	MissingAssets []string `json:"missingAssets"`
}

// GaeaProposalSectionOp 对一个小节做结构修正：op=promote 升级 / demote 降级 /
// moveUp 上移 / moveDown 下移 / split 拆分（offset=正文 rune 偏移，光标所在行及
// 之后切成标题为 title 的新子节）/ mergeParent 并入父节 / mergePrev 并入上一同级
// 小节 / addChild 新增子节（title）/ addSibling 新增同级小节（title）/ remove 删除
// （连同子孙）。整批一个事务；标题里的编号不改。回执的 focusId 是操作完成后界面
// 应落到的小节（0=整册已空）。
func (a *App) GaeaProposalSectionOp(sectionID int64, op string, offset int, title string) (*ProposalStructureView, error) {
	st, err := a.hubProposalStore()
	if err != nil {
		return nil, err
	}
	res, err := st.Restructure(sectionID, proposal.StructureInput{Op: op, Offset: offset, Title: title})
	if err != nil {
		return nil, err
	}
	return &ProposalStructureView{
		Op: res.Op, FocusID: res.FocusID, Affected: res.Affected, Created: res.Created,
		Removed: res.Removed, RemovedChars: res.RemovedChars, MissingAssets: res.MissingAssets,
	}, nil
}

// GaeaProposalTwins 其他方案里与本节同题的小节（标题路径去掉编号后相同）。
func (a *App) GaeaProposalTwins(sectionID int64) ([]proposal.Twin, error) {
	st, err := a.hubProposalStore()
	if err != nil {
		return []proposal.Twin{}, err
	}
	return st.Twins(sectionID)
}

// GaeaProposalIndexStatus 索引体检（只读）：向量进度、孤儿 / 缺失 / 过期的全文索引
// 行、缺语义块的小节。
func (a *App) GaeaProposalIndexStatus() (*proposal.IndexStatus, error) {
	st, err := a.hubProposalStore()
	if err != nil {
		return nil, err
	}
	status, err := st.IndexStatus()
	if err != nil {
		return nil, err
	}
	return &status, nil
}

// proposalReindexBusy 防止重复触发重建（整库重嵌要几十秒到几分钟）。
var proposalReindexBusy atomic.Bool

// GaeaProposalReindex 修复并重建索引：清孤儿行、补缺失的全文索引与语义块、换掉
// 过期块前缀，再把未向量化的块补齐（本地模型不可用时只做结构类修复）。
func (a *App) GaeaProposalReindex() (*proposal.IndexStatus, error) {
	if !proposalReindexBusy.CompareAndSwap(false, true) {
		return nil, fmt.Errorf("语义索引正在重建，请稍候再试")
	}
	defer proposalReindexBusy.Store(false)
	st, err := a.hubProposalStore()
	if err != nil {
		return nil, err
	}
	var (
		embed proposal.Embedder
		model string
	)
	if sem := builtin.ProposalSemantic(); sem != nil && sem.Embed != nil {
		embed, model = sem.Embed, sem.Model
	}
	ctx, cancel := context.WithTimeout(context.Background(), proposal.ReindexBudget+time.Minute)
	defer cancel()
	status, err := st.Reindex(ctx, embed, model)
	if err != nil {
		return &status, err
	}
	return &status, nil
}

// GaeaProposalExport 整册导出 Markdown 到 outputPath（须是绝对路径：由另存对话框
// 选定）。托管图片落到同名的 -assets 目录。
func (a *App) GaeaProposalExport(name string, outputPath string) (*ProposalExportView, error) {
	outputPath = filepath.Clean(filepath.FromSlash(strings.TrimSpace(outputPath)))
	if !filepath.IsAbs(outputPath) {
		return nil, fmt.Errorf("导出路径须是绝对路径（用「另存为」对话框选定的位置）")
	}
	st, err := a.hubProposalStore()
	if err != nil {
		return nil, err
	}
	res, err := st.Export(name, outputPath)
	if err != nil {
		return nil, err
	}
	return &ProposalExportView{
		Path: res.Path, AssetsDir: res.AssetsDir, Sections: res.Sections,
		Chars: res.Chars, Images: res.Images, Missing: res.Missing,
	}, nil
}

// ProposalExportView 导出回执。
type ProposalExportView struct {
	Path      string `json:"path"`
	AssetsDir string `json:"assetsDir"`
	Sections  int    `json:"sections"`
	Chars     int    `json:"chars"`
	Images    int    `json:"images"`
	Missing   int    `json:"missing"`
}
