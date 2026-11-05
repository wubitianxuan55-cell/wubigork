package builtin

// websearch_core_test.go — 统一检索核测试（众神殿刀1）：桩引擎经注册表注入
// （kind 名独立命名空间，不与真实 6 引擎冲突），验证首胜编排、策略滤空拆分、
// 全败诊断与 ClassifySearchState 归类。

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/config"
)

// coreStubSearch 各桩 kind 的行为函数（测试内按 kind 注入）。
var coreStubSearch = map[string]func(ctx context.Context, query string, limit int) ([]SearchResult, error){}

// withCoreStubs 注册测试桩引擎并在测试结束清理（注册表是全局互斥的，桩 kind
// 必须随测试退出还原，否则污染 TestSearchEngineRegistry_AllKinds 的全量断言）。
func withCoreStubs(t *testing.T) {
	t.Helper()
	saved := make(map[string]SearchEngineFactory, len(searchEngineRegistry))
	for k, f := range searchEngineRegistry {
		saved[k] = f
	}
	for _, kind := range []string{"core-stub-1", "core-stub-2", "core-stub-3"} {
		k := kind
		RegisterSearchEngine(k, func(cfg SearchEngineConfig) (SearchEngine, error) {
			return coreStubEngine{kind: k}, nil
		})
	}
	t.Cleanup(func() {
		for k := range searchEngineRegistry {
			if _, keep := saved[k]; !keep {
				delete(searchEngineRegistry, k)
			}
		}
	})
}

type coreStubEngine struct{ kind string }

func (e coreStubEngine) Name() string      { return e.kind }
func (e coreStubEngine) Available() bool   { return true }
func (e coreStubEngine) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	if f := coreStubSearch[e.kind]; f != nil {
		return f(ctx, query, limit)
	}
	return nil, nil
}

// TestRunSearch_FirstWin 首胜编排：单引擎成功 → 结果原样返回、ok 尝试在案、零错误。
func TestRunSearch_FirstWin(t *testing.T) {
	withCoreStubs(t)
	saveSearchGlobals(t)
	SetSearchConfig(config.SearchConfig{})
	SetSearchEngineOrder([]string{"core-stub-1"})
	coreStubSearch["core-stub-1"] = func(_ context.Context, q string, limit int) ([]SearchResult, error) {
		if q != "golang 1.24 release notes" {
			t.Errorf("query 原样透传被改写: %q", q) // 办公 query 原样不改写（site:/操作符边界同源验证）
		}
		return []SearchResult{{Title: "Go 1.24", URL: "https://go.dev/doc/go1.24", Snippet: "notes", Source: "core-stub-1"}}, nil
	}
	results, attempts, err := RunSearch(context.Background(), "golang 1.24 release notes", 5)
	if err != nil {
		t.Fatalf("RunSearch: %v", err)
	}
	if len(results) != 1 || results[0].Title != "Go 1.24" {
		t.Errorf("results = %+v", results)
	}
	if len(attempts) != 1 || attempts[0].Outcome != attemptOK {
		t.Errorf("attempts = %+v", attempts)
	}
	if got := ClassifySearchState(results, attempts); got != SearchStateOK {
		t.Errorf("ClassifySearchState = %q, want ok", got)
	}
}

// TestRunSearch_FilteredEmptySplit 刀1 关键拆分：引擎返回了结果但被域名策略
// 滤空 → filtered-empty（不是引擎失败），全败诊断写明「策略过滤后无可用结果」。
func TestRunSearch_FilteredEmptySplit(t *testing.T) {
	withCoreStubs(t)
	saveSearchGlobals(t)
	SetSearchConfig(config.SearchConfig{DenyDomains: []string{"denied.com"}})
	SetSearchEngineOrder([]string{"core-stub-1"})
	coreStubSearch["core-stub-1"] = func(_ context.Context, _ string, _ int) ([]SearchResult, error) {
		return []SearchResult{{Title: "被拒", URL: "https://denied.com/x", Snippet: "s", Source: "core-stub-1"}}, nil
	}
	results, attempts, err := RunSearch(context.Background(), "测试", 5)
	if err == nil || len(results) != 0 {
		t.Fatalf("滤空应无可选用结果: results=%v err=%v", results, err)
	}
	if len(attempts) != 1 || attempts[0].Outcome != attemptFilteredEmpty {
		t.Fatalf("attempts = %+v, want filtered-empty", attempts)
	}
	if got := ClassifySearchState(results, attempts); got != SearchStateEmpty {
		t.Errorf("ClassifySearchState = %q, want empty（滤空≠ALL_FAILED）", got)
	}
	if diag := buildAllFailedDiag(attempts); !strings.Contains(diag, "策略过滤后无可用结果") {
		t.Errorf("全败诊断应写明策略滤空: %q", diag)
	}
}

// TestRunSearch_RawEmpty 真空结果（引擎成功返回空）→ empty 尝试、EMPTY 状态。
func TestRunSearch_RawEmpty(t *testing.T) {
	withCoreStubs(t)
	saveSearchGlobals(t)
	SetSearchConfig(config.SearchConfig{})
	SetSearchEngineOrder([]string{"core-stub-1"})
	coreStubSearch["core-stub-1"] = func(_ context.Context, _ string, _ int) ([]SearchResult, error) {
		return nil, nil
	}
	results, attempts, err := RunSearch(context.Background(), "测试", 5)
	if err == nil || len(results) != 0 {
		t.Fatalf("空结果应无可选用结果: err=%v", err)
	}
	if len(attempts) != 1 || attempts[0].Outcome != attemptEmpty {
		t.Fatalf("attempts = %+v, want empty", attempts)
	}
	if got := ClassifySearchState(results, attempts); got != SearchStateEmpty {
		t.Errorf("ClassifySearchState = %q, want empty", got)
	}
}

// TestRunSearch_AllEngineError 全部传输/解析失败 → error 尝试、ALL_FAILED、
// 诊断含引擎名与原因。
func TestRunSearch_AllEngineError(t *testing.T) {
	withCoreStubs(t)
	saveSearchGlobals(t)
	SetSearchConfig(config.SearchConfig{})
	SetSearchEngineOrder([]string{"core-stub-1", "core-stub-2"})
	coreStubSearch["core-stub-1"] = func(_ context.Context, _ string, _ int) ([]SearchResult, error) {
		return nil, errors.New("connection refused")
	}
	coreStubSearch["core-stub-2"] = func(_ context.Context, _ string, _ int) ([]SearchResult, error) {
		return nil, errors.New("tls handshake timeout")
	}
	results, attempts, err := RunSearch(context.Background(), "测试", 5)
	if err == nil || len(results) != 0 {
		t.Fatalf("全败应报错: err=%v", err)
	}
	if len(attempts) != 2 {
		t.Fatalf("attempts = %+v, want 2", attempts)
	}
	for _, a := range attempts {
		if a.Outcome != attemptError {
			t.Errorf("attempt %s outcome = %q, want error", a.Name, a.Outcome)
		}
	}
	if got := ClassifySearchState(results, attempts); got != SearchStateAllFailed {
		t.Errorf("ClassifySearchState = %q, want all_failed", got)
	}
	diag := buildAllFailedDiag(attempts)
	if !strings.Contains(diag, "core-stub-1") || !strings.Contains(diag, "connection refused") {
		t.Errorf("诊断应含引擎名与原因: %q", diag)
	}
	if !strings.Contains(diag, "TAVILY_API_KEY") {
		t.Errorf("零配置全败应附主源指引（文案现状保留）: %q", diag)
	}
}

// TestClassifySearchState 归类表：有结果=OK；有引擎完成检索（含滤空）=EMPTY；
// 全错或零引擎=ALL_FAILED。
func TestClassifySearchState(t *testing.T) {
	cases := []struct {
		name     string
		results  []SearchResult
		attempts []SearchAttempt
		want     SearchState
	}{
		{"有结果", []SearchResult{{Title: "t"}}, []SearchAttempt{{Outcome: attemptOK}}, SearchStateOK},
		{"真空结果", nil, []SearchAttempt{{Outcome: attemptEmpty}}, SearchStateEmpty},
		{"滤空", nil, []SearchAttempt{{Outcome: attemptFilteredEmpty}}, SearchStateEmpty},
		{"混合错误与真空", nil, []SearchAttempt{{Outcome: attemptError}, {Outcome: attemptEmpty}}, SearchStateEmpty},
		{"全错", nil, []SearchAttempt{{Outcome: attemptError}}, SearchStateAllFailed},
		{"超时伪引擎", nil, []SearchAttempt{{Name: "timeout", Outcome: attemptError}}, SearchStateAllFailed},
		{"零引擎", nil, nil, SearchStateAllFailed},
	}
	for _, c := range cases {
		if got := ClassifySearchState(c.results, c.attempts); got != c.want {
			t.Errorf("%s: ClassifySearchState = %q, want %q", c.name, got, c.want)
		}
	}
}
