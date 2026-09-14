package app

// 原罪·书源接线（sin 书源线 t2，规格 docs/gaea-sin-booksource-distill-2026-09.md §5；
// 2026-09-14 用户拍板：该线自 DSH 并行线移交本线执行）。
//
// 与小说侧（novel_booksource_handler.go）的分工：搜索/目录/规则装载/取消登记簿
// 全部同源复用（同一引擎、同一共享规则层、同一套 fail-closed 边界），差异只在
// **产物归属**——小说侧下载即落库成书架项目，原罪侧成书 TXT 落 sin 数据面
// （sinRoot()/books），零工作区/记忆面接触（硬隔离口径不变）。
//
// 规则目录合流注记：本规格原定 sinRoot()/rules，小说侧 t1 先落地共享层
// %配置目录%/gaea/booksource/rules——按「以先落地为准」收敛到共享层，不各养一份。

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/booksource"
)

// sinBooksDir 原罪成书目录（书源下载的整本 TXT；sin 数据面，与办公工作区隔离）。
func sinBooksDir() string {
	return filepath.Join(sinRoot(), "books")
}

// SinBookSourceBook 成书清单条目（sinRoot()/books 下的整本 TXT）。
type SinBookSourceBook struct {
	Title      string `json:"title"`
	Path       string `json:"path"`
	SizeBytes  int64  `json:"sizeBytes"`
	ModifiedAt string `json:"modifiedAt"` // RFC3339
}

// SinBookSourceDownloadStart 下载起跑回执（进度/终态走 sin-booksource:<jobId> 事件）。
type SinBookSourceDownloadStart struct {
	JobID string `json:"jobId"`
}

// SinBookSourceDownloadResult 成书结果（done 事件载荷；Failed 如实带出不静默）。
type SinBookSourceDownloadResult struct {
	Title    string                     `json:"title"`
	Path     string                     `json:"path"`
	Chapters int                        `json:"chapters"`
	Words    int                        `json:"words"`
	Failed   []booksource.FailedChapter `json:"failed,omitempty"`
}

// ── 内部实现（目录可注入，测试走假站）──

// sinUniqueBookPath 成书文件不冲突命名（同名追加序号，uniqueProjectDir 同思路）。
func sinUniqueBookPath(dir, title string) string {
	base := sanitizeDirName(title)
	path := filepath.Join(dir, base+".txt")
	for i := 2; ; i++ {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return path
		}
		path = filepath.Join(dir, fmt.Sprintf("%s (%d).txt", base, i))
	}
}

// sinDownloadBook 下载整本 → 组装 TXT → 落 sin 成书目录（同步内核；绑定层包
// goroutine + 取消）。失败语义沿引擎：单章失败不中断整本，Failed 随结果带出。
func sinDownloadBook(ctx context.Context, rulesDir, booksDir string, p bookImportParams, opt booksource.Options, onProgress func(done, total int)) (SinBookSourceDownloadResult, error) {
	rule := findRuleByName(loadBookSourceRules(rulesDir), p.Source)
	if rule == nil {
		return SinBookSourceDownloadResult{}, fmt.Errorf("书源规则不存在或未启用：%s（免规则来源正文不可解析）", p.Source)
	}
	if strings.TrimSpace(p.URL) == "" {
		return SinBookSourceDownloadResult{}, fmt.Errorf("详情地址为空")
	}
	engine := booksource.New(rule, opt)
	report, err := engine.Download(ctx, p.URL, booksource.DownloadOptions{Start: p.Start, End: p.End, OnProgress: onProgress})
	if err != nil {
		return SinBookSourceDownloadResult{}, err
	}
	if len(report.Chapters) == 0 {
		return SinBookSourceDownloadResult{}, fmt.Errorf("整本下载失败：%d 章全部未取到（源站可能反爬或规则失配）", len(report.Failed))
	}
	// 书名：显式指定 > 详情页 > 兜底（与小说侧导入同优先级）。
	title := strings.TrimSpace(p.Title)
	info, _ := engine.BookDetail(ctx, p.URL) // best-effort：失败不阻断成书
	if title == "" {
		if strings.TrimSpace(info.Name) != "" {
			title = strings.TrimSpace(info.Name)
		} else {
			title = "书源成书"
		}
	}
	if strings.TrimSpace(info.Name) == "" {
		info.Name = title // 成书头部书名与文件名同源（详情页解析不到时不留空）
	}
	if err := os.MkdirAll(booksDir, 0o755); err != nil {
		return SinBookSourceDownloadResult{}, fmt.Errorf("创建成书目录失败: %w", err)
	}
	// 范围下载序号在引擎 Toc 已重编为 1..N（规格 §3.5 同口径），组装直接用。
	raw, err := booksource.AssembleTXT(info, report.Chapters, "utf-8")
	if err != nil {
		return SinBookSourceDownloadResult{}, err
	}
	path := sinUniqueBookPath(booksDir, title)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return SinBookSourceDownloadResult{}, fmt.Errorf("写成书文件失败: %w", err)
	}
	words := 0
	for _, ch := range report.Chapters {
		for _, para := range ch.Paragraphs {
			words += len([]rune(para))
		}
	}
	return SinBookSourceDownloadResult{
		Title: title, Path: path, Chapters: len(report.Chapters), Words: words, Failed: report.Failed,
	}, nil
}

// sinBooksList 成书清单（.txt，按修改时间新→旧）。
func sinBooksList(dir string) ([]SinBookSourceBook, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []SinBookSourceBook{}, nil // 未下载过任何成书=空清单，不报错
		}
		return nil, err
	}
	books := make([]SinBookSourceBook, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".txt") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		st, err := e.Info()
		if err != nil {
			continue // 竞态删除的条目跳过
		}
		books = append(books, SinBookSourceBook{
			Title:      strings.TrimSuffix(e.Name(), filepath.Ext(e.Name())),
			Path:       path,
			SizeBytes:  st.Size(),
			ModifiedAt: st.ModTime().UTC().Format(time.RFC3339),
		})
	}
	sort.SliceStable(books, func(i, j int) bool { return books[i].ModifiedAt > books[j].ModifiedAt })
	return books, nil
}

// sinBookDeleteAt 删除成书（fail-closed 路径护栏：必须在成书目录内且是 .txt）。
func sinBookDeleteAt(dir, path string) error {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if absPath != absDir && !strings.HasPrefix(absPath, absDir+string(filepath.Separator)) {
		return fmt.Errorf("路径不在成书目录内，拒绝删除")
	}
	if !strings.EqualFold(filepath.Ext(absPath), ".txt") {
		return fmt.Errorf("只允许删除 .txt 成书文件")
	}
	if err := os.Remove(absPath); err != nil {
		return fmt.Errorf("删除失败: %w", err)
	}
	return nil
}

// ── 绑定（SinB 门面经 gen_bindings 收录；真目录 + 真网络）──

// SinBookSourceSearch 书源聚合搜索 + 泛搜索（与小说侧同源同表，HasRule=可下载）。
func (a *App) SinBookSourceSearch(keyword string) (NovelBookSourceSearchResult, error) {
	dir := booksourceRulesDir()
	if err := booksource.EnsureTemplate(dir); err != nil {
		return NovelBookSourceSearchResult{}, fmt.Errorf("书源模板初始化失败: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), booksourceSearchTimeout)
	defer cancel()
	return searchBookSources(ctx, dir, keyword, booksource.Options{})
}

// SinBookSourceToc 目录预览（总数 + 首8末4样例，供范围选择）。
func (a *App) SinBookSourceToc(source, detailURL string) (NovelBookSourceTocPreview, error) {
	ctx, cancel := context.WithTimeout(context.Background(), booksourceSearchTimeout)
	defer cancel()
	return tocBookSource(ctx, booksourceRulesDir(), source, detailURL, booksource.Options{})
}

// SinBookSourceDownload 下载整本成书 TXT → sinRoot()/books；进度/终态走
// sin-booksource:<jobId> 事件（progress/done/error；done 附 Failed 清单）。
func (a *App) SinBookSourceDownload(source, detailURL string, start, end int, title string) (SinBookSourceDownloadStart, error) {
	// fail-closed 先查规则（无规则正文不可解析，不入队再失败）
	if findRuleByName(loadBookSourceRules(booksourceRulesDir()), source) == nil {
		return SinBookSourceDownloadStart{}, fmt.Errorf("书源规则不存在或未启用：%s（免规则来源正文不可解析）", source)
	}
	ctx, cancel := context.WithTimeout(context.Background(), booksourceImportTimeout)
	jobID := fmt.Sprintf("bss_%d_%d", time.Now().UnixMilli(), bookImportSeq.Add(1))
	bookImportMu.Lock()
	bookImportRuns[jobID] = cancel
	bookImportMu.Unlock()

	params := bookImportParams{Source: source, URL: detailURL, Title: title, Start: start, End: end}
	go func() {
		defer func() {
			bookImportMu.Lock()
			delete(bookImportRuns, jobID)
			bookImportMu.Unlock()
			cancel()
		}()
		onProgress := func(done, total int) {
			if done%booksourceProgressStep == 0 || done == total {
				a.emit("sin-booksource:"+jobID, map[string]interface{}{"type": "progress", "done": done, "total": total})
			}
		}
		res, err := sinDownloadBook(ctx, booksourceRulesDir(), sinBooksDir(), params, booksource.Options{}, onProgress)
		if err != nil {
			a.emit("sin-booksource:"+jobID, map[string]interface{}{"type": "error", "error": err.Error()})
			return
		}
		a.emit("sin-booksource:"+jobID, map[string]interface{}{"type": "done", "result": res})
	}()
	return SinBookSourceDownloadStart{JobID: jobID}, nil
}

// SinBookSourceDownloadCancel 取消在途下载（与小说侧共用 job 登记簿；未知 job 如实 false）。
func (a *App) SinBookSourceDownloadCancel(jobID string) bool {
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

// SinBookSourceBooksList 成书清单（空目录=空清单不报错）。
func (a *App) SinBookSourceBooksList() ([]SinBookSourceBook, error) {
	return sinBooksList(sinBooksDir())
}

// SinBookSourceBookDelete 删除成书（fail-closed 路径护栏：限成书目录内 .txt）。
func (a *App) SinBookSourceBookDelete(path string) error {
	return sinBookDeleteAt(sinBooksDir(), path)
}
