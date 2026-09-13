package app

// 书源取书 → 拆书导入接线（规格 docs/gaea-novel-booksource-import-2026-09.md）。
// internal/booksource 引擎（搜书/目录/下载）作小说导入的取书上游：
// 搜书 → 目录预览 → 范围下载 → createImportedProject 直接入书架。
// 边界：免规则候选（HasRule=false）只到目录为止（引擎同上游边界：正文必填），
// 导入端 fail-closed 拒绝；本文件零 sin 引用（原罪硬隔离不破）。

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/gaea/gaea/internal/bookimport"
	"github.com/gaea/gaea/internal/booksource"
	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/types"
)

// enginesFileName 泛搜索引擎文件与书源规则同目录（EnsureTemplate 单目录落盘），
// 装载书源规则时按名剔除（它是引擎规则不是书源规则）。
const enginesFileName = "websearch-engines.json"

const (
	// booksourceSearchTimeout 聚合 + 泛搜索整体上限（多源并发，串行引擎数少）。
	booksourceSearchTimeout = 60 * time.Second
	// booksourceImportTimeout 整本下载 + 落库宽上限（千章级站点 * 有界并发）。
	booksourceImportTimeout = 30 * time.Minute
	// booksourceProgressStep 进度事件节流：每 N 章一报 + 完成必报（对齐上游 SSE 50 章/次的降频思路）。
	booksourceProgressStep = 20
)

// NovelBookSourceCandidate 搜书候选（书源命中=可导入；泛搜索=参考）。
type NovelBookSourceCandidate struct {
	Kind          string `json:"kind"`   // rule（书源规则命中）| web（泛搜索）
	Source        string `json:"source"` // 书源规则名 / 搜索引擎名
	Title         string `json:"title"`
	Author        string `json:"author,omitempty"`
	URL           string `json:"url"`
	Host          string `json:"host,omitempty"`
	HasRule       bool   `json:"hasRule"` // 同 host 有启用中的书源规则（=可导入）
	LatestChapter string `json:"latestChapter,omitempty"`
}

// NovelBookSourceSearchResult 搜书结果：候选 + 各源失败告警（如实透出，不静默吞）。
type NovelBookSourceSearchResult struct {
	Candidates []NovelBookSourceCandidate `json:"candidates"`
	Warnings   []string                   `json:"warnings,omitempty"`
}

// NovelBookSourceTocPreview 目录预览：总章数 + 首末样例（供范围选择）。
type NovelBookSourceTocPreview struct {
	Total     int                   `json:"total"`
	Sample    []booksource.TocEntry `json:"sample"`
	Truncated bool                  `json:"truncated"` // Total > 样例数时 true
}

// NovelBookSourceImportStart 在线导入起跑回执（进度/终态走事件通道）。
type NovelBookSourceImportStart struct {
	JobID string `json:"jobId"`
}

// ── 共享资产目录（规则是引擎数据资产，同角色库跨板块共享先例；规格 §1.4）──

// booksourceRoot 书源共享资产根（用户配置目录下；原罪线未来同源取用，不各养一份）。
func booksourceRoot() string {
	return filepath.Join(gaeaConfig.MemoryUserDir(), "booksource")
}

// booksourceRulesDir 书源规则目录（EnsureTemplate 幂等落模板：规则模板 disabled
// 不被拾取 + 泛搜索引擎模板）。
func booksourceRulesDir() string {
	return filepath.Join(booksourceRoot(), "rules")
}

// ── 在途导入任务登记（SinCancel 同款先例：句柄登记供精确取消）──

var (
	bookImportMu   sync.Mutex
	bookImportRuns = map[string]context.CancelFunc{}
	bookImportSeq  atomic.Int64
)

// ── 内部实现（目录与网络通道均可注入，测试走假站）──

// loadBookSourceRules 装载启用中的书源规则：剔除引擎文件/坏规则/Disabled；
// 坏文件如实记日志不静默（规则目录里混入坏 JSON 不拖垮搜索）。
func loadBookSourceRules(dir string) []*booksource.Rule {
	var (
		rules []*booksource.Rule
		bad   []string
	)
	for _, l := range booksource.LoadDir(dir) {
		if l.Err != nil {
			if l.File != enginesFileName {
				bad = append(bad, fmt.Sprintf("%s: %v", l.File, l.Err))
			}
			continue
		}
		if l.Rule == nil || l.Rule.Disabled || l.File == enginesFileName {
			continue
		}
		rules = append(rules, l.Rule)
	}
	if len(bad) > 0 {
		slog.Warn("书源规则装载有坏文件（已跳过）", "files", strings.Join(bad, "; "))
	}
	return rules
}

// findRuleByName 按规则名精确找书源（搜书结果回传 source → 下载/目录定位规则）。
func findRuleByName(rules []*booksource.Rule, name string) *booksource.Rule {
	for _, r := range rules {
		if r.Name == name {
			return r
		}
	}
	return nil
}

// ruleHost 书源站点宿主（与引擎 hostOf 同口径：小写 Host 含端口），供 HasRule 打标。
func ruleHost(r *booksource.Rule) string {
	raw := r.BaseURI
	if raw == "" && r.Search != nil {
		raw = r.Search.URL
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Host)
}

// searchBookSources 聚合搜索：书源 Aggregate（可导入候选）+ 泛搜索 DiscoverySearch
// （参考候选，带 HasRule 打标）；按 URL 去重（书源命中优先），各源失败进 warnings。
func searchBookSources(ctx context.Context, dir, keyword string, opt booksource.Options) (NovelBookSourceSearchResult, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return NovelBookSourceSearchResult{}, fmt.Errorf("搜索关键字为空")
	}
	rules := loadBookSourceRules(dir)
	hostRule := map[string]string{}
	for _, r := range rules {
		if h := ruleHost(r); h != "" {
			hostRule[h] = r.Name
		}
	}

	res := NovelBookSourceSearchResult{}
	seen := map[string]bool{}
	add := func(c NovelBookSourceCandidate) {
		key := strings.TrimRight(c.URL, "/")
		if key == "" || seen[key] {
			return
		}
		seen[key] = true
		res.Candidates = append(res.Candidates, c)
	}

	hits, srcErrs := booksource.Aggregate(ctx, opt, keyword, rules...)
	for _, e := range srcErrs {
		res.Warnings = append(res.Warnings, fmt.Sprintf("书源「%s」搜索失败: %v", e.Source, e.Err))
	}
	for _, h := range hits {
		add(NovelBookSourceCandidate{
			Kind: "rule", Source: h.Source, Title: h.BookName, Author: h.Author,
			URL: h.URL, Host: hostOfCompat(h.URL), HasRule: true, LatestChapter: h.LatestChapter,
		})
	}

	engines, err := booksource.LoadEnginesFile(filepath.Join(dir, enginesFileName))
	if err != nil {
		res.Warnings = append(res.Warnings, "泛搜索引擎规则未就绪: "+err.Error())
	} else {
		ws := booksource.NewWebSearcher(booksource.WebOptions{Options: opt})
		for _, eng := range engines {
			if eng.Disabled {
				continue
			}
			webc, err := ws.DiscoverySearch(ctx, eng, keyword, booksource.WebSearchOptions{
				KnownRules: func(host string) bool { _, ok := hostRule[host]; return ok },
			})
			if err != nil {
				res.Warnings = append(res.Warnings, fmt.Sprintf("搜索引擎「%s」失败: %v", eng.Name, err))
				continue
			}
			for _, c := range webc {
				add(NovelBookSourceCandidate{
					Kind: "web", Source: eng.Name, Title: c.Title,
					URL: c.URL, Host: c.Host, HasRule: c.HasRule,
				})
			}
		}
	}
	return res, nil
}

// hostOfCompat 与引擎 hostOf 同口径（小写 Host 含端口）；规则侧候选补齐 Host 字段。
func hostOfCompat(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Host)
}

// tocBookSource 目录预览（首末样例截取，总数如实）。
func tocBookSource(ctx context.Context, dir, source, detailURL string, opt booksource.Options) (NovelBookSourceTocPreview, error) {
	rule := findRuleByName(loadBookSourceRules(dir), source)
	if rule == nil {
		return NovelBookSourceTocPreview{}, fmt.Errorf("书源规则不存在或未启用：%s（免规则来源正文不可解析，不提供目录预览）", source)
	}
	if strings.TrimSpace(detailURL) == "" {
		return NovelBookSourceTocPreview{}, fmt.Errorf("详情地址为空")
	}
	toc, err := booksource.New(rule, opt).Toc(ctx, detailURL, 0, 0)
	if err != nil {
		return NovelBookSourceTocPreview{}, err
	}
	preview := NovelBookSourceTocPreview{Total: len(toc)}
	const head, tail = 8, 4
	if len(toc) > head+tail {
		preview.Truncated = true
		preview.Sample = append(append([]booksource.TocEntry{}, toc[:head]...), toc[len(toc)-tail:]...)
		return preview, nil
	}
	preview.Sample = toc
	return preview, nil
}

// bookImportParams 在线导入参数（绑定层入参收拢）。
type bookImportParams struct {
	Source string
	URL    string
	Title  string
	Genre  string
	Style  string
	Start  int // 1 起闭区间；0 = 全本
	End    int // 0 = 到末章
}

// importBookSource 下载整本 → 落库成书架项目（同步内核；绑定层包 goroutine + 取消）。
// 失败语义沿引擎：单章失败不中断整本，Failed 清单如实带出。范围导入项目章号从 1
// 重编号（规格 §3.5：项目章号连续自洽，不沿用源站序号）。
func importBookSource(ctx context.Context, dir, novelsDir string, p bookImportParams, opt booksource.Options, onProgress func(done, total int)) (NovelImportResult, []booksource.FailedChapter, error) {
	rule := findRuleByName(loadBookSourceRules(dir), p.Source)
	if rule == nil {
		return NovelImportResult{}, nil, fmt.Errorf("书源规则不存在或未启用：%s（免规则来源正文不可解析）", p.Source)
	}
	if strings.TrimSpace(p.URL) == "" {
		return NovelImportResult{}, nil, fmt.Errorf("详情地址为空")
	}
	engine := booksource.New(rule, opt)
	report, err := engine.Download(ctx, p.URL, booksource.DownloadOptions{Start: p.Start, End: p.End, OnProgress: onProgress})
	if err != nil {
		return NovelImportResult{}, nil, err
	}
	if len(report.Chapters) == 0 {
		return NovelImportResult{}, report.Failed, fmt.Errorf("整本下载失败：%d 章全部未取到（源站可能反爬或规则失配）", len(report.Failed))
	}
	title := strings.TrimSpace(p.Title)
	if title == "" { // 详情页书名兜底（best-effort，失败回退固定名）
		if info, err := engine.BookDetail(ctx, p.URL); err == nil && strings.TrimSpace(info.Name) != "" {
			title = strings.TrimSpace(info.Name)
		} else {
			title = "书源导入"
		}
	}
	chapters := make([]importChapter, 0, len(report.Chapters))
	for _, ch := range report.Chapters {
		chapters = append(chapters, importChapter{
			Title:   ch.Title,
			Content: strings.Join(ch.Paragraphs, "\n"),
		})
	}
	res, err := createImportedProject(novelsDir, title, p.Genre, p.Style, chapters, bookimport.Report{
		SplitStrategy:    "booksource",
		TotalChapters:    len(report.Chapters) + len(report.Failed),
		SelectedChapters: len(report.Chapters),
	})
	if err != nil {
		return NovelImportResult{}, report.Failed, err
	}
	return res, report.Failed, nil
}

// ── 绑定（NovelB 门面经 gen_bindings 收录；真目录 + 真网络）──

// NovelBookSourceSearch 书源聚合搜索 + 泛搜索（HasRule 标注可导入性）。
func (w *writingState) NovelBookSourceSearch(keyword string) (NovelBookSourceSearchResult, error) {
	dir := booksourceRulesDir()
	if err := booksource.EnsureTemplate(dir); err != nil {
		return NovelBookSourceSearchResult{}, fmt.Errorf("书源模板初始化失败: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), booksourceSearchTimeout)
	defer cancel()
	return searchBookSources(ctx, dir, keyword, booksource.Options{})
}

// NovelBookSourceToc 目录预览（范围选择的依据）。
func (w *writingState) NovelBookSourceToc(source, detailURL string) (NovelBookSourceTocPreview, error) {
	ctx, cancel := context.WithTimeout(context.Background(), booksourceSearchTimeout)
	defer cancel()
	return tocBookSource(ctx, booksourceRulesDir(), source, detailURL, booksource.Options{})
}

// NovelBookSourceImport 在线导入起跑：后台下载整本 → 落库书架；进度/终态走
// novel-import-progress:<jobID> 事件（progress/done/error；done 附 Failed 清单）。
func (w *writingState) NovelBookSourceImport(source, detailURL string, start, end int, title, genre, style string) (NovelBookSourceImportStart, error) {
	// fail-closed 先查规则（无规则正文不可解析，不入队再失败）
	if findRuleByName(loadBookSourceRules(booksourceRulesDir()), source) == nil {
		return NovelBookSourceImportStart{}, fmt.Errorf("书源规则不存在或未启用：%s（免规则来源正文不可解析）", source)
	}
	ctx, cancel := context.WithTimeout(context.Background(), booksourceImportTimeout)
	jobID := fmt.Sprintf("bsi_%d_%d", time.Now().UnixMilli(), bookImportSeq.Add(1))
	bookImportMu.Lock()
	bookImportRuns[jobID] = cancel
	bookImportMu.Unlock()

	params := bookImportParams{Source: source, URL: detailURL, Title: title, Genre: genre, Style: style, Start: start, End: end}
	go func() {
		defer func() {
			bookImportMu.Lock()
			delete(bookImportRuns, jobID)
			bookImportMu.Unlock()
			cancel()
		}()
		onProgress := func(done, total int) {
			if done%booksourceProgressStep == 0 || done == total {
				w.emit("novel-import-progress:"+jobID, map[string]interface{}{"type": "progress", "done": done, "total": total})
			}
		}
		res, failed, err := importBookSource(ctx, booksourceRulesDir(), w.cfg.NovelsDir, params, booksource.Options{}, onProgress)
		if err != nil {
			w.emit("novel-import-progress:"+jobID, map[string]interface{}{"type": "error", "error": err.Error(), "failed": failedCount(failed)})
			return
		}
		w.emit("novel-import-progress:"+jobID, map[string]interface{}{"type": "done", "result": res, "failed": failed})
	}()
	return NovelBookSourceImportStart{JobID: jobID}, nil
}

// failedCount 终态 error 事件只带失败计数（清单归 done/error 文案，避免载荷膨胀）。
func failedCount(failed []booksource.FailedChapter) int { return len(failed) }

// NovelBookSourceImportCancel 取消在途导入（精确到 job；未知 job 如实返回 false）。
func (w *writingState) NovelBookSourceImportCancel(jobID string) bool {
	bookImportMu.Lock()
	cancel, ok := bookImportRuns[jobID]
	if ok {
		delete(bookImportRuns, jobID)
	}
	bookImportMu.Unlock()
	if !ok {
		return false
	}
	cancel()
	return true
}

// ── 失败章补下（t3 重试；规格 docs/gaea-novel-booksource-import-2026-09.md §5 t3）──

// NovelBookSourceAppendResult 补下结果：追加章数与追加后总量（追加语义词，不与
// 整本导入的 chapter_count 混用）。
type NovelBookSourceAppendResult struct {
	Path          string                     `json:"path"`
	Title         string                     `json:"title"`
	Appended      int                        `json:"appended"`
	TotalChapters int                        `json:"totalChapters"`
	AddedWords    int                        `json:"addedWords"`
	Failed        []booksource.FailedChapter `json:"failed,omitempty"`
}

// NovelBookSourceImportChapters 失败章补下：对既有书架项目按显式清单抓章追加
// （清单 = 整本导入 done 事件的 Failed 原样回传）。章号从项目现有最大章号续编，
// 大纲节点同序追加（imp-NNN 与既有编号规则一致且不撞号）；进度/终态复用
// novel-import-progress:<jobID> 通道（append-done 为补下终态），取消复用同一登记簿。
func (w *writingState) NovelBookSourceImportChapters(source, projectPath, chaptersJSON string) (NovelBookSourceImportStart, error) {
	rule := findRuleByName(loadBookSourceRules(booksourceRulesDir()), source)
	if rule == nil {
		return NovelBookSourceImportStart{}, fmt.Errorf("书源规则不存在或未启用：%s", source)
	}
	var items []booksource.FailedChapter
	if err := json.Unmarshal([]byte(chaptersJSON), &items); err != nil || len(items) == 0 {
		return NovelBookSourceImportStart{}, fmt.Errorf("待补下章节清单为空或格式错误")
	}
	for i := range items {
		if strings.TrimSpace(items[i].URL) == "" {
			return NovelBookSourceImportStart{}, fmt.Errorf("清单第 %d 项缺少 URL", i+1)
		}
	}
	if _, err := os.Stat(filepath.Join(projectPath, "project.json")); err != nil {
		return NovelBookSourceImportStart{}, fmt.Errorf("项目不存在或无效：%s", projectPath)
	}

	ctx, cancel := context.WithTimeout(context.Background(), booksourceImportTimeout)
	jobID := fmt.Sprintf("bsa_%d_%d", time.Now().UnixMilli(), bookImportSeq.Add(1))
	bookImportMu.Lock()
	bookImportRuns[jobID] = cancel
	bookImportMu.Unlock()

	go func() {
		defer func() {
			bookImportMu.Lock()
			delete(bookImportRuns, jobID)
			bookImportMu.Unlock()
			cancel()
		}()
		toc := make([]booksource.TocEntry, len(items))
		for i := range items {
			toc[i] = booksource.TocEntry{Title: items[i].Title, URL: items[i].URL, Order: i + 1}
		}
		engine := booksource.New(rule, booksource.Options{})
		onProgress := func(done, total int) {
			if done%booksourceProgressStep == 0 || done == total {
				w.emit("novel-import-progress:"+jobID, map[string]interface{}{"type": "progress", "done": done, "total": total})
			}
		}
		report, err := engine.DownloadChapters(ctx, toc, booksource.DownloadOptions{OnProgress: onProgress})
		if err != nil {
			w.emit("novel-import-progress:"+jobID, map[string]interface{}{"type": "error", "error": err.Error(), "failed": failedCount(report.Failed)})
			return
		}
		res, err := appendProjectChapters(projectPath, report)
		if err != nil {
			w.emit("novel-import-progress:"+jobID, map[string]interface{}{"type": "error", "error": err.Error(), "failed": failedCount(report.Failed)})
			return
		}
		w.emit("novel-import-progress:"+jobID, map[string]interface{}{"type": "append-done", "result": res})
	}()
	return NovelBookSourceImportStart{JobID: jobID}, nil
}

// appendProjectChapters 把补下章节追加进既有项目：章号从现有最大 OrderIndex 续编，
// 大纲节点（imp-NNN/done）同序追加、其余节点原样保留（outline.Agent 写路径每次
// 调用均重读 outline.json，无缓存副本冲突面）。
func appendProjectChapters(projectPath string, report booksource.DownloadReport) (NovelBookSourceAppendResult, error) {
	pm, err := project.Open(projectPath)
	if err != nil {
		return NovelBookSourceAppendResult{}, fmt.Errorf("项目打开失败: %w", err)
	}
	defer pm.Close() //nolint:errcheck // 元信息时间戳尽力而为

	of, err := pm.ReadOutlines()
	if err != nil || of == nil {
		return NovelBookSourceAppendResult{}, fmt.Errorf("读取项目大纲失败: %w", err)
	}
	maxNum := 0
	for _, n := range of.Nodes {
		if n.OrderIndex > maxNum {
			maxNum = n.OrderIndex
		}
	}
	appended, words := 0, 0
	for _, ch := range report.Chapters {
		num := maxNum + appended + 1
		content := strings.Join(ch.Paragraphs, "\n")
		if strings.TrimSpace(content) == "" {
			content = "（本章暂无内容）"
		}
		if err := pm.WriteChapter(num, content); err != nil {
			return NovelBookSourceAppendResult{}, fmt.Errorf("写章节 %d 失败: %w", num, err)
		}
		words += utf8.RuneCountInString(content)
		of.Nodes = append(of.Nodes, types.OutlineNode{
			ID:          fmt.Sprintf("imp-%03d", num),
			Title:       strings.TrimSpace(ch.Title),
			OrderIndex:  num,
			ChapterFile: fmt.Sprintf("%03d.md", num),
			Status:      types.OutlineDone,
		})
		appended++
	}
	if appended > 0 {
		if err := pm.WriteOutlines(of); err != nil {
			return NovelBookSourceAppendResult{}, fmt.Errorf("写回大纲失败: %w", err)
		}
	}
	title := ""
	if pm.Meta != nil {
		title = pm.Meta.Title
	}
	return NovelBookSourceAppendResult{
		Path: projectPath, Title: title, Appended: appended,
		TotalChapters: maxNum + appended, AddedWords: words, Failed: report.Failed,
	}, nil
}
