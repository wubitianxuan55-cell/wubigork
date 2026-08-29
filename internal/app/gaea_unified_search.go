package app

import (
	"strings"

	"github.com/gaea/gaea/internal/gaea/spaces"
	"github.com/gaea/gaea/internal/gaea/wssearch"
)

// UnifiedSearchView 是统一检索入口（T5-6）的返回视图：一次搜索同时出
// 「关键词全文 + 语义跨库」两组结果，前端一个搜索框两段展示。
// 记忆统一层第一刀扩展：新增 Brain（三脑命中）与 Files（文件语义命中）
// 两组——hub 搜索由「4 绑定前端拼装」收敛为「1 绑定后端聚合」。
type UnifiedSearchView struct {
	// Keyword 是工作区关键词全文命中（轻量 RAG，与 WorkspaceSearchHit 一致）。
	Keyword []WorkspaceSearchHit `json:"keyword"`
	// Semantic 是跨库语义命中（cost/knowledge/office/file）；embedding 不可用
	// 时为空数组而 Keyword 照常（与现有降级行为一致）。
	Semantic []SemanticHitView `json:"semantic"`
	// Brain 是三脑命中（brain.main/left/right）；三脑未装配（a.brain==nil）时
	// 为空数组，不报错。
	Brain []Hit `json:"brain"`
	// Files 是工作区文件语义命中（path/score/snippet）；embedding 不可用时为空数组。
	Files []FileSemanticHit `json:"files"`
}

// workspaceSearchHits 工作区关键词全文检索的共用私有实现：逻辑与
// GaeaWorkspaceSearch 完全一致（抽取自其方法体，统一入口不重复实现）。
func (a *App) workspaceSearchHits(query string, topN int) []WorkspaceSearchHit {
	hits := wssearch.Search(gaeaCwd(), query, topN)
	if len(hits) == 0 {
		return []WorkspaceSearchHit{}
	}
	out := make([]WorkspaceSearchHit, 0, len(hits))
	for _, h := range hits {
		out = append(out, WorkspaceSearchHit{
			Path:      h.Path,
			Name:      h.Name,
			Size:      h.Size,
			ModTime:   h.ModTime,
			Score:     h.Score,
			Snippet:   h.Snippet,
			Truncated: h.Truncated,
			Skipped:   h.Skipped,
		})
	}
	return out
}

// semanticSearchHits 跨库统一语义检索的共用私有实现（统一入口复用，T5-6）：
// 实现收敛到 gaea_semantic_search.go 的按需/缓存版 semanticSearchHitsOnDemand
// （T7-3：避免每查询先 Ensure 全量扫描；逻辑与 GaeaSemanticSearch 完全一致）。
func (a *App) semanticSearchHits(query string) ([]SemanticHitView, error) {
	return a.semanticSearchHitsOnDemand(query)
}

// brainSearchHits 三脑统一检索的共用私有实现：三脑未装配（a.brain==nil）
// 时返回空数组而不报错（hub 搜索降级为 keyword/semantic/files 三组照常）。
// scope 为空 = 全部（旧行为）；"work"/"play" 走 SearchInSpace 空间限定
// （whisper 右脑 play 专属、左脑 facts 按空间谓词、主脑共享不过滤）。
func (a *App) brainSearchHits(query, scope string) []Hit {
	if a.brain == nil {
		return []Hit{}
	}
	hits, err := a.brain.SearchInSpace(query, scope)
	if err != nil || len(hits) == 0 {
		return []Hit{}
	}
	return hits
}

// searchScope 归一统一检索 scope（S1.2 B）：仅 "work"/"play" 是空间过滤态，
// 其余（含 "" 与非法值）一律回退 "" = 全部/旧行为——后端不把 "" 归一成
// work，space.mode=off 平铺形态的旧数据在缺省调用下保持全量可见；scope=""
// =全部仅显式选择时由前端传入（C 步）。
func searchScope(scope string) string {
	if spaces.Valid(scope) {
		return scope
	}
	return ""
}

// filterSemanticHitsByScope 语义命中后过滤（S1.2 B）。**严禁按空间过滤索引
// 源**：semantic_vectors 是四库共享索引且无空间列，Ensure/Stale 以「向量条数
// == 源文档数」判定就绪——按空间砍源会让另一空间的向量被 Stale(keep) 物理删
// 除（设计 §风险「semantic Stale 陷阱」，最高危），因此只在最终 hits 上按
// kind 映射过滤（勘察锚点 5）：
//   - office（办公 facts）：facts 同库混存两空间——按 ListInSpace(scope) 的
//     名称集映射剔除（唯一键 (project, name) 全局唯一，名称即身份）；
//   - cost / knowledge / file：不过滤（勘察锚点 5）——三库当前只在 work 语义
//     落库、无 play 侧数据可漏（红线「工位搜不到乐园记忆」不受影响），且
//     knowledge 与 brain.main 工程知识库同属共享面（play 可见，见 brain 组）；
//   - 未知 kind：保守丢弃（fail-closed 宁缺毋漏；当前四库封闭不会触达，
//     scope="" 不走本过滤）。whisper 不入 semantic 库（play 专属走
//     brain.right，由 brain 组隔离）。
func (a *App) filterSemanticHitsByScope(hits []SemanticHitView, scope string) []SemanticHitView {
	if scope == "" || len(hits) == 0 {
		return hits
	}
	officeAllowed := map[string]bool{}
	for _, m := range a.hubOfficeStore().ListInSpace(scope) {
		officeAllowed[m.Name] = true
	}
	out := make([]SemanticHitView, 0, len(hits))
	for _, h := range hits {
		switch h.Kind {
		case "office":
			if officeAllowed[h.Name] {
				out = append(out, h)
			}
		case "cost", "knowledge", "file":
			out = append(out, h) // 不过滤：共享源，无 play 侧数据
		}
	}
	return out
}

// GaeaUnifiedSearch 统一检索入口（T5-6）：一个搜索框同时出「关键词全文 +
// 语义跨库」两组结果。记忆统一层第一刀扩展为四组：keyword（工作区全文）+
// semantic（跨库语义）+ brain（三脑命中）+ files（文件语义命中）。
// 内部串行调用现有各域检索的共用私有实现；空 query 返回空视图；
// embedding 不可用时 semantic/files 为空数组而 keyword/brain 照常。
//
// S1.2 B 读端隔离器：scope 为可变参（保持既有两参调用兼容——Wails 反射
// Call 对变参安全，前端 C 步前传 (query, topN) 不破，scope 缺省 ""=全部/
// 旧行为；C 步开始显式传 "work"/"play"）。四组过滤语义：
//   - keyword / files：共享工作区面不过滤（与旧行为一致；.gaea/play/exports
//     已并入 wssearch 噪音规则，play 交付物不进检索面）；
//   - brain：SearchInSpace（work 滤 right、facts 按空间谓词、主脑共享）；
//   - semantic：只对最终 hits 后过滤（严禁动索引源，见
//     filterSemanticHitsByScope 注释）。
func (a *App) GaeaUnifiedSearch(query string, topN int, scope ...string) (UnifiedSearchView, error) {
	sc := ""
	if len(scope) > 0 {
		sc = searchScope(scope[0]) // 只取首个 scope；未传 = "" = 全部/旧行为
	}
	view := UnifiedSearchView{
		Keyword:  []WorkspaceSearchHit{},
		Semantic: []SemanticHitView{},
		Brain:    []Hit{},
		Files:    []FileSemanticHit{},
	}
	if strings.TrimSpace(query) == "" {
		return view, nil
	}
	view.Keyword = a.workspaceSearchHits(query, topN)
	view.Brain = a.brainSearchHits(query, sc)
	if topN <= 0 {
		topN = 10
	}
	files, err := a.fileSemanticHits(query, topN)
	if err != nil {
		return view, err
	}
	if files != nil {
		view.Files = files
	}
	sem, err := a.semanticSearchHits(query)
	if err != nil {
		return view, err
	}
	view.Semantic = a.filterSemanticHitsByScope(sem, sc)
	if view.Semantic == nil {
		view.Semantic = []SemanticHitView{}
	}
	return view, nil
}
