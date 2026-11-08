package builtin

// websearch_core_test.go — 统一检索核测试（众神殿刀1 编排 + 刀2 主源保留窗/
// 短窗合并/覆盖门）。桩引擎经注册表注入（kind 名独立命名空间），其中
// core-stub-primary 经 searchPrimaryKinds 注入为主源类。验证矩阵含 Codex
// 会话桩测五条（retention/超窗降级/URL 去重双源/EMPTY vs ALL_FAILED/注入头
// 四态文案——末条在 app 包 adapter 测试）。

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/config"
)

// coreStubSearch 各桩 kind 的行为函数（测试内按 kind 注入；可带 sleep 模拟慢引擎）。
var coreStubSearch = map[string]func(ctx context.Context, query string, limit int) ([]SearchResult, error){}

// withCoreStubs 注册测试桩引擎并在测试结束清理（注册表与主源类别都是全局，
// 桩必须随测试退出还原，否则污染 TestSearchEngineRegistry_AllKinds 等全量断言）。
// 2026-11-08 补漏：coreStubSearch 桩函数表也是全局——各测试覆写
// core-stub-1 等键后从不还原，测试行为随执行顺序漂移（偶发
// TestRunSearch_CoverageGateJunk_Degraded 拿到上一个测试的报错桩）。现在
// 入场快照、退场还原。
func withCoreStubs(t *testing.T) {
	t.Helper()
	saved := make(map[string]SearchEngineFactory, len(searchEngineRegistry))
	for k, f := range searchEngineRegistry {
		saved[k] = f
	}
	stubKinds := []string{"core-stub-1", "core-stub-2", "core-stub-3", "core-stub-primary"}
	savedStubs := make(map[string]func(ctx context.Context, query string, limit int) ([]SearchResult, error), len(stubKinds))
	for _, kind := range stubKinds {
		savedStubs[kind] = coreStubSearch[kind]
	}
	for _, kind := range stubKinds {
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
		for _, kind := range stubKinds {
			if fn := savedStubs[kind]; fn == nil {
				delete(coreStubSearch, kind)
			} else {
				coreStubSearch[kind] = fn
			}
		}
	})
	// 桩主源类 kind 注入（isPrimaryKind 按 kind 判主源类别）
	if !searchPrimaryKinds["core-stub-primary"] {
		searchPrimaryKinds["core-stub-primary"] = true
		t.Cleanup(func() { delete(searchPrimaryKinds, "core-stub-primary") })
	}
}

type coreStubEngine struct{ kind string }

func (e coreStubEngine) Name() string    { return e.kind }
func (e coreStubEngine) Available() bool { return true }
func (e coreStubEngine) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	if f := coreStubSearch[e.kind]; f != nil {
		return f(ctx, query, limit)
	}
	return nil, nil
}

// TestRunSearch_FirstWin 单 keyless 引擎：结果原样返回、ok 尝试在案、覆盖门过
// → OK（单引擎行为等价现状的回归守卫）。
func TestRunSearch_FirstWin(t *testing.T) {
	withCoreStubs(t)
	saveSearchGlobals(t)
	SetSearchConfig(config.SearchConfig{})
	SetSearchEngineOrder([]string{"core-stub-1"})
	coreStubSearch["core-stub-1"] = func(_ context.Context, q string, limit int) ([]SearchResult, error) {
		if q != "golang 1.24 release notes" {
			t.Errorf("query 原样透传被改写: %q", q) // 办公 query 原样不改写（site:/操作符边界同源验证）
		}
		return []SearchResult{{Title: "Go 1.24 release notes", URL: "https://go.dev/doc/go1.24", Snippet: "notes", Source: "core-stub-1"}}, nil
	}
	outcome, err := RunSearch(context.Background(), "golang 1.24 release notes", 5)
	if err != nil {
		t.Fatalf("RunSearch: %v", err)
	}
	if len(outcome.Results) != 1 || outcome.Results[0].Title != "Go 1.24 release notes" {
		t.Errorf("results = %+v", outcome.Results)
	}
	if len(outcome.Attempts) != 1 || outcome.Attempts[0].Outcome != attemptOK {
		t.Errorf("attempts = %+v", outcome.Attempts)
	}
	if outcome.State != SearchStateOK || outcome.DegradedReason != "" {
		t.Errorf("State/Reason = %q/%q, want ok/\"\"", outcome.State, outcome.DegradedReason)
	}
}

// TestRunSearch_RetentionWindow_KeepsPrimary Codex 桩测①：keyless 20ms 返回、
// 主源 300ms 返回（窗内）→ 结果必含主源，且 keyless 独有结果保留。
func TestRunSearch_RetentionWindow_KeepsPrimary(t *testing.T) {
	withCoreStubs(t)
	saveSearchGlobals(t)
	SetSearchConfig(config.SearchConfig{})
	SetSearchEngineOrder([]string{"core-stub-primary", "core-stub-1"})
	coreStubSearch["core-stub-primary"] = func(_ context.Context, _ string, _ int) ([]SearchResult, error) {
		time.Sleep(300 * time.Millisecond)
		return []SearchResult{{Title: "主源正题结果", URL: "https://primary.example/a", Snippet: "汇率", Source: "core-stub-primary"}}, nil
	}
	coreStubSearch["core-stub-1"] = func(_ context.Context, _ string, _ int) ([]SearchResult, error) {
		time.Sleep(20 * time.Millisecond)
		return []SearchResult{{Title: "keyless 独有", URL: "https://keyless.example/b", Snippet: "汇率", Source: "core-stub-1"}}, nil
	}
	outcome, err := RunSearch(context.Background(), "人民币 汇率", 5)
	if err != nil {
		t.Fatalf("RunSearch: %v", err)
	}
	if outcome.State != SearchStateOK {
		t.Errorf("State = %q (%s), want ok", outcome.State, outcome.DegradedReason)
	}
	if len(outcome.Results) != 2 {
		t.Fatalf("结果应含主源+keyless 独有共 2 条, got %d: %+v", len(outcome.Results), outcome.Results)
	}
	if outcome.Results[0].Source != "core-stub-primary" {
		t.Errorf("主源应排首位, got %+v", outcome.Results[0])
	}
	if outcome.Results[1].Source != "core-stub-1" {
		t.Errorf("keyless 独有结果应保留, got %+v", outcome.Results[1])
	}
}

// TestRunSearch_PrimaryTimeout_Degraded Codex 桩测②：主源超过保留窗（超窗后
// 才返回也不得采纳）→ DEGRADED(primary-timeout)，禁止静默标 OK。
func TestRunSearch_PrimaryTimeout_Degraded(t *testing.T) {
	withCoreStubs(t)
	saveSearchGlobals(t)
	SetSearchConfig(config.SearchConfig{})
	SetSearchEngineOrder([]string{"core-stub-primary", "core-stub-1"})
	oldWindow := searchPrimaryWindow
	searchPrimaryWindow = 100 * time.Millisecond
	t.Cleanup(func() { searchPrimaryWindow = oldWindow })
	coreStubSearch["core-stub-primary"] = func(_ context.Context, _ string, _ int) ([]SearchResult, error) {
		time.Sleep(400 * time.Millisecond) // 远超 100ms 保留窗
		return []SearchResult{{Title: "迟到主源", URL: "https://primary.example/late", Snippet: "汇率", Source: "core-stub-primary"}}, nil
	}
	coreStubSearch["core-stub-1"] = func(_ context.Context, _ string, _ int) ([]SearchResult, error) {
		time.Sleep(20 * time.Millisecond)
		return []SearchResult{{Title: "汇率兜底", URL: "https://keyless.example/fx", Snippet: "汇率", Source: "core-stub-1"}}, nil
	}
	outcome, err := RunSearch(context.Background(), "人民币 汇率", 5)
	if err != nil {
		t.Fatalf("RunSearch: %v", err)
	}
	if outcome.State != SearchStateDegraded || outcome.DegradedReason != "primary-timeout" {
		t.Errorf("State/Reason = %q/%q, want degraded/primary-timeout", outcome.State, outcome.DegradedReason)
	}
	for _, r := range outcome.Results {
		if strings.Contains(r.Source, "core-stub-primary") {
			t.Errorf("超窗主源结果不得进合并集: %+v", outcome.Results)
		}
	}
	// 诊断可见「配了主源但没生效」（合成超时尝试在案）
	found := false
	for _, a := range outcome.Attempts {
		if a.Name == "core-stub-primary" && strings.Contains(a.Err, "保留窗") {
			found = true
		}
	}
	if !found {
		t.Errorf("超窗主源应有合成诊断尝试: %+v", outcome.Attempts)
	}
}

// TestRunSearch_PrimaryError_Degraded 主源明确失败 → keyless 兜底 + primary-error。
func TestRunSearch_PrimaryError_Degraded(t *testing.T) {
	withCoreStubs(t)
	saveSearchGlobals(t)
	SetSearchConfig(config.SearchConfig{})
	SetSearchEngineOrder([]string{"core-stub-primary", "core-stub-1"})
	coreStubSearch["core-stub-primary"] = func(_ context.Context, _ string, _ int) ([]SearchResult, error) {
		return nil, errors.New("401 unauthorized")
	}
	coreStubSearch["core-stub-1"] = func(_ context.Context, _ string, _ int) ([]SearchResult, error) {
		return []SearchResult{{Title: "汇率兜底", URL: "https://keyless.example/fx", Snippet: "汇率", Source: "core-stub-1"}}, nil
	}
	outcome, err := RunSearch(context.Background(), "人民币 汇率", 5)
	if err != nil {
		t.Fatalf("keyless 兜底成功不应报错: %v", err)
	}
	if outcome.State != SearchStateDegraded || outcome.DegradedReason != "primary-error" {
		t.Errorf("State/Reason = %q/%q, want degraded/primary-error", outcome.State, outcome.DegradedReason)
	}
}

// TestMergeSearchResults_DedupDualSource Codex 桩测③：两引擎同一规范 URL
// （www./尾斜杠/utm_ 差异归一）→ 只保留一条且 Source 记录双来源。
func TestMergeSearchResults_DedupDualSource(t *testing.T) {
	primary := []SearchResult{{Title: "正题", URL: "https://www.example.com/a/", Snippet: "s", Source: "tavily"}}
	extras := []taggedResults{{name: "bing", results: []SearchResult{
		{Title: "同页", URL: "https://example.com/a?utm_source=x&utm_medium=y", Snippet: "s", Source: "bing"},
		{Title: "独有", URL: "https://example.com/unique", Snippet: "s", Source: "bing"},
	}}}
	merged := mergeSearchResults(primary, extras, 5)
	if len(merged) != 2 {
		t.Fatalf("去重后应剩 2 条, got %d: %+v", len(merged), merged)
	}
	if merged[0].Source != "tavily+bing" {
		t.Errorf("重复条目应并入双来源, got %q", merged[0].Source)
	}
	if merged[1].URL != "https://example.com/unique" {
		t.Errorf("独有条目应保留: %+v", merged[1])
	}
}

// TestRunSearch_CoverageGateJunk_Degraded 对抗离题回归（基线 E 类机制面）：
// 无主源、keyless 返回非空全离题结果 → DEGRADED(coverage-miss)，不得标 OK。
func TestRunSearch_CoverageGateJunk_Degraded(t *testing.T) {
	withCoreStubs(t)
	saveSearchGlobals(t)
	SetSearchConfig(config.SearchConfig{})
	SetSearchEngineOrder([]string{"core-stub-1"}) // 无主源
	coreStubSearch["core-stub-1"] = func(_ context.Context, _ string, _ int) ([]SearchResult, error) {
		// 复刻 grok-r2 探针：日期查询 → 百科/万年历全离题
		return []SearchResult{
			{Title: "2026年_百度百科", URL: "https://baike.example/2026", Snippet: "2026年是一个平年", Source: "bing"},
			{Title: "万年历2026年日历表", URL: "https://rili.example/2026", Snippet: "2026年日历", Source: "bing"},
		}, nil
	}
	outcome, err := RunSearch(context.Background(), "2026年11月3日 汇率", 5)
	if err != nil {
		t.Fatalf("非空结果不该报错: %v", err)
	}
	if outcome.State != SearchStateDegraded || outcome.DegradedReason != "coverage-miss" {
		t.Errorf("State/Reason = %q/%q, want degraded/coverage-miss（离题结果不得标 OK）", outcome.State, outcome.DegradedReason)
	}
}

// TestRunSearch_CoverageGenericQuery 泛词查询（无可核验实体）→ DEGRADED，
// 结果仍返回但标注（不冒充可核验）。
func TestRunSearch_CoverageGenericQuery(t *testing.T) {
	withCoreStubs(t)
	saveSearchGlobals(t)
	SetSearchConfig(config.SearchConfig{})
	SetSearchEngineOrder([]string{"core-stub-1"})
	coreStubSearch["core-stub-1"] = func(_ context.Context, _ string, _ int) ([]SearchResult, error) {
		return []SearchResult{{Title: "最新消息汇总", URL: "https://news.example/all", Snippet: "今日要点", Source: "bing"}}, nil
	}
	outcome, _ := RunSearch(context.Background(), "最新 消息", 5)
	if outcome.State != SearchStateDegraded || outcome.DegradedReason != "coverage-generic-query" {
		t.Errorf("State/Reason = %q/%q, want degraded/coverage-generic-query", outcome.State, outcome.DegradedReason)
	}
	if len(outcome.Results) != 1 {
		t.Errorf("DEGRADED 应保留结果（标注而非丢弃）: %+v", outcome.Results)
	}
}

// TestRunSearch_FilteredEmptySplit 刀1 拆分：策略过滤后空是 EMPTY（检索发生
// 了），不是引擎失败，更不是 ALL_FAILED。
func TestRunSearch_FilteredEmptySplit(t *testing.T) {
	withCoreStubs(t)
	saveSearchGlobals(t)
	SetSearchConfig(config.SearchConfig{DenyDomains: []string{"denied.com"}})
	SetSearchEngineOrder([]string{"core-stub-1"})
	coreStubSearch["core-stub-1"] = func(_ context.Context, _ string, _ int) ([]SearchResult, error) {
		return []SearchResult{{Title: "被拒", URL: "https://denied.com/x", Snippet: "s", Source: "core-stub-1"}}, nil
	}
	outcome, err := RunSearch(context.Background(), "测试", 5)
	if err == nil || len(outcome.Results) != 0 {
		t.Fatalf("滤空应无可选用结果: results=%v err=%v", outcome, err)
	}
	if outcome.State != SearchStateEmpty {
		t.Errorf("State = %q, want empty（滤空≠ALL_FAILED）", outcome.State)
	}
	if len(outcome.Attempts) != 1 || outcome.Attempts[0].Outcome != attemptFilteredEmpty {
		t.Fatalf("attempts = %+v, want filtered-empty", outcome.Attempts)
	}
	if diag := buildAllFailedDiag(outcome.Attempts); !strings.Contains(diag, "策略过滤后无可用结果") {
		t.Errorf("全败诊断应写明策略滤空: %q", diag)
	}
}

// TestRunSearch_RawEmpty Codex 桩测④前半：引擎成功返回空 → EMPTY 尝试、状态 EMPTY。
func TestRunSearch_RawEmpty(t *testing.T) {
	withCoreStubs(t)
	saveSearchGlobals(t)
	SetSearchConfig(config.SearchConfig{})
	SetSearchEngineOrder([]string{"core-stub-1"})
	coreStubSearch["core-stub-1"] = func(_ context.Context, _ string, _ int) ([]SearchResult, error) {
		return nil, nil
	}
	outcome, err := RunSearch(context.Background(), "测试", 5)
	if err == nil || len(outcome.Results) != 0 {
		t.Fatalf("空结果应无可选用结果: err=%v", err)
	}
	if outcome.State != SearchStateEmpty {
		t.Errorf("State = %q, want empty", outcome.State)
	}
	if len(outcome.Attempts) != 1 || outcome.Attempts[0].Outcome != attemptEmpty {
		t.Fatalf("attempts = %+v, want empty", outcome.Attempts)
	}
}

// TestRunSearch_AllEngineError Codex 桩测④后半：全部传输/解析失败 → ALL_FAILED，
// 诊断含引擎名与原因 + 零配置指引（文案现状保留）。
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
	outcome, err := RunSearch(context.Background(), "测试", 5)
	if err == nil || len(outcome.Results) != 0 {
		t.Fatalf("全败应报错: err=%v", err)
	}
	if outcome.State != SearchStateAllFailed {
		t.Errorf("State = %q, want all_failed", outcome.State)
	}
	if len(outcome.Attempts) != 2 {
		t.Fatalf("attempts = %+v, want 2", outcome.Attempts)
	}
	if !strings.Contains(err.Error(), "core-stub-1") || !strings.Contains(err.Error(), "connection refused") {
		t.Errorf("err 消息应含引擎名与原因: %q", err.Error())
	}
	if !strings.Contains(err.Error(), "TAVILY_API_KEY") {
		t.Errorf("零配置全败应附主源指引（文案现状保留）: %q", err.Error())
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

// TestCoverageGate 覆盖门 v0 单元表：泛词、词面不相干、正题命中、snippet 命中。
func TestCoverageGate(t *testing.T) {
	cases := []struct {
		name       string
		query      string
		results    []SearchResult
		wantPass   bool
		wantReason string
	}{
		{"泛词无实体", "最新 消息", []SearchResult{{Title: "最新消息汇总"}}, false, "coverage-generic-query"},
		{"词面不相干（复刻 grok-r2 探针）", "2026年11月3日 汇率", []SearchResult{
			{Title: "2026年_百度百科", Snippet: "2026年是一个平年"},
			{Title: "万年历2026年日历表", Snippet: "2026年日历"},
		}, false, "coverage-miss"},
		{"正题命中", "人民币 汇率", []SearchResult{{Title: "人民币兑美元中间价", Snippet: "11月3日报 6.7150"}}, true, ""},
		{"snippet 命中也算", "golang 1.24", []SearchResult{{Title: "Go 发行说明", Snippet: "golang 1.24 新特性"}}, true, ""},
	}
	for _, c := range cases {
		pass, reason := coverageGate(c.query, c.results)
		if pass != c.wantPass || reason != c.wantReason {
			t.Errorf("%s: coverageGate = (%v,%q), want (%v,%q)", c.name, pass, reason, c.wantPass, c.wantReason)
		}
	}
}
