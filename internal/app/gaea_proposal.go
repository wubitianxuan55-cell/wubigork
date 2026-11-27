package app

import (
	"context"
	"fmt"
	"time"

	"github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/proposal"
	"github.com/gaea/gaea/internal/gaea/tool/builtin"
)

// 方案库板块（一级板块「方案库」）的绑定面。录入与索引只在办公方案模式里
// 由模型经 proposal_save 完成；这里只提供给人看的清单/标题树/阅读/检索/删除。
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

// ProposalSearchResultView 界面检索结果。
type ProposalSearchResultView struct {
	Hits []proposal.Hit `json:"hits"`
	Mode string         `json:"mode"`
	Note string         `json:"note"`
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
// （形参名用 within：叫 proposal 会遮住同名的存储层包名。）
func (a *App) GaeaProposalSearch(query string, scope string, level int, projectType string, year string, within string, limit int) (*ProposalSearchResultView, error) {
	st, err := a.hubProposalStore()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), proposalSearchTimeout)
	defer cancel()
	res, err := st.HybridSearch(ctx, proposal.SearchInput{
		Query: query, Scope: scope, Level: level, ProjectType: projectType, Year: year, Proposal: within, TopN: limit,
	}, builtin.ProposalSemantic())
	if err != nil {
		return nil, err
	}
	return &ProposalSearchResultView{Hits: res.Hits, Mode: res.Mode, Note: res.Note}, nil
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
