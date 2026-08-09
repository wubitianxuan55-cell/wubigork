package builtin

import "testing"

func TestParseBingResults(t *testing.T) {
	html := `<html><body><ol id="b_results">
<li class="b_algo"><h2><a href="https://example.com/a">第一 <strong>结果</strong></a></h2>
<div class="b_caption"><p class="b_lineclamp2">这是摘要内容。</p></div></li>
<li class="b_algo"><h2><a href="javascript:void(0)">跳过链接</a></h2></li>
<li class="b_algo"><h2><a href="https://example.com/b">第二个结果</a></h2>
<div class="b_caption"><p>另一段摘要。</p></div></li>
</ol></body></html>`
	results, err := parseBingResults([]byte(html), 5)
	if err != nil {
		t.Fatalf("parseBingResults: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("want 2 results, got %d: %+v", len(results), results)
	}
	if results[0].Title != "第一 结果" || results[0].URL != "https://example.com/a" {
		t.Errorf("result[0] = %+v", results[0])
	}
	if results[0].Snippet != "这是摘要内容。" || results[0].Source != "bing" {
		t.Errorf("result[0] snippet/source = %q / %q", results[0].Snippet, results[0].Source)
	}
	if results[1].Title != "第二个结果" || results[1].Snippet != "另一段摘要。" {
		t.Errorf("result[1] = %+v", results[1])
	}
}

func TestParseBingResultsNoResults(t *testing.T) {
	if _, err := parseBingResults([]byte("<html><body>no results</body></html>"), 5); err == nil {
		t.Fatal("want error for empty SERP")
	}
}

func TestParseDDGLiteResults(t *testing.T) {
	html := `<html><body><table>
<tr><td><a rel="nofollow" class="result-link" href="https://example.com/1">标题一</a></td></tr>
<tr><td class="result-snippet">摘要一</td></tr>
<tr><td><a rel="nofollow" class="result-link" href="https://example.com/2">标题二</a></td></tr>
<tr><td class="result-snippet">摘要二</td></tr>
</table></body></html>`
	results, err := parseDDGLiteResults([]byte(html), 5)
	if err != nil {
		t.Fatalf("parseDDGLiteResults: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("want 2 results, got %d: %+v", len(results), results)
	}
	if results[0].Title != "标题一" || results[0].URL != "https://example.com/1" || results[0].Snippet != "摘要一" {
		t.Errorf("result[0] = %+v", results[0])
	}
	if results[1].Snippet != "摘要二" || results[1].Source != "duckduckgo" {
		t.Errorf("result[1] = %+v", results[1])
	}
}

func TestParseDDGLiteResultsAnomaly(t *testing.T) {
	if _, err := parseDDGLiteResults([]byte("<html><body>anomaly challenge</body></html>"), 5); err == nil {
		t.Fatal("want error for anomaly page")
	}
}
