package builtin

import (
	"context"
	"strings"
	"testing"
)

// readabilityFixture 带导航/侧栏/页脚噪声的文章页（正文 5 段、每段 20+ rune，
// 噪声全为无 <p> 的裸元素——真实站点常态）。
const readabilityFixture = `<!doctype html><html><head><title>测试文章标题</title></head><body>
<nav class="main-nav"><a href="/a">导航链接一</a><a href="/b">导航链接二</a><a href="/c">导航链接三</a></nav>
<aside class="sidebar">侧栏推广 侧栏推广 侧栏推广</aside>
<article class="post-content"><h1>文章主标题</h1>
<p>这是正文第一段，讲述主题的背景与动机，交代研究对象的现状与此前方案的不足，长度明显超过二十个字符以通过验收门槛。</p>
<p>这是正文第二段，展开论证的第一个论点并给出证据，引用了三份独立来源的实测数据，长度明显超过二十个字符以通过验收门槛。</p>
<p>这是正文第三段，展开论证的第二个论点并给出数据，对比了两种实现路径的吞吐与延迟差异，长度明显超过二十个字符以通过验收门槛。</p>
<p>这是正文第四段，讨论反驳意见并逐条回应，指出对方样本选择偏差与统计口径不一致的问题，长度明显超过二十个字符以通过验收门槛。</p>
<p>这是正文第五段，总结全文并给出行动建议，列出下一步的三项实验与各自的判定标准，长度明显超过二十个字符以通过验收门槛。</p>
</article>
<footer>页脚版权信息 页脚版权信息</footer>
</body></html>`

// TestReadabilityExtractsArticle 文章页：auto 提主内容（nav/侧栏/页脚出局），
// title 保留为一级标题。
func TestReadabilityExtractsArticle(t *testing.T) {
	frag, ok := readabilityExtract(readabilityFixture)
	if !ok {
		t.Fatal("文章页应提取成功")
	}
	if !strings.Contains(frag, "正文第一段") || !strings.Contains(frag, "文章主标题") {
		t.Fatalf("片段缺正文: %.200s", frag)
	}
	if strings.Contains(frag, "导航链接一") || strings.Contains(frag, "页脚版权") || strings.Contains(frag, "侧栏推广") {
		t.Fatalf("片段混入噪声: %.300s", frag)
	}
	out := htmlToReadableText(readabilityFixture, readabilityModeAuto)
	if !strings.HasPrefix(out, "# 测试文章标题") {
		t.Fatalf("输出应保留 title 一级标题, got %.40q", out)
	}
	if !strings.Contains(out, "正文第三段") {
		t.Fatalf("输出缺正文: %.200s", out)
	}
	for _, noise := range []string{"导航链接一", "页脚版权", "侧栏推广"} {
		if strings.Contains(out, noise) {
			t.Fatalf("输出混入噪声 %q: %.300s", noise, out)
		}
	}
}

// TestReadabilityStripsInsideBest 主容器内部的 script/nav 也被摘除。
func TestReadabilityStripsInsideBest(t *testing.T) {
	page := strings.Replace(readabilityFixture,
		`<article class="post-content"><h1>文章主标题</h1>`,
		`<article class="post-content"><script>var tracking=1;</script><nav>容器内导航噪声</nav><h1>文章主标题</h1>`, 1)
	out := htmlToReadableText(page, readabilityModeAuto)
	if strings.Contains(out, "tracking") || strings.Contains(out, "容器内导航噪声") {
		t.Fatalf("主容器内噪声未摘除: %.300s", out)
	}
	if !strings.Contains(out, "正文第一段") {
		t.Fatalf("摘除误伤正文: %.300s", out)
	}
}

// TestReadabilityFallbackShort 短页/无主容器：判据不足回退整页旧路径（含噪声）。
func TestReadabilityFallbackShort(t *testing.T) {
	page := `<html><head><title>短页</title></head><body><nav>导航噪声</nav>
<div><p>只有两段</p><p>每段都短</p></div></body></html>`
	if _, ok := readabilityExtract(page); ok {
		t.Fatal("短页应判据不足回退")
	}
	if got := htmlToReadableText(page, readabilityModeAuto); got != htmlToText(page) {
		t.Fatalf("回退应与旧行为逐字节一致:\nauto=%.200q\nold =%.200q", got, htmlToText(page))
	}
	if !strings.Contains(htmlToReadableText(page, readabilityModeAuto), "导航噪声") {
		t.Fatal("回退路径应保留整页文本（含噪声）")
	}
}

// TestReadabilityFullMode mode=full 显式旧行为：即使文章页也整页铺开。
func TestReadabilityFullMode(t *testing.T) {
	out := htmlToReadableText(readabilityFixture, readabilityModeFull)
	for _, want := range []string{"导航链接一", "页脚版权", "正文第一段"} {
		if !strings.Contains(out, want) {
			t.Fatalf("full 模式缺 %q", want)
		}
	}
}

// TestWebFetchModeValidation 非法 mode 在抓取之前被拒（Go error 通道，同既有
// web_fetch 参数校验风格）。
func TestWebFetchModeValidation(t *testing.T) {
	if _, err := (webFetch{}).Execute(context.Background(), []byte(`{"url":"http://example.local/x","mode":"bogus"}`)); err == nil {
		t.Fatal("非法 mode 应报错")
	} else if !strings.Contains(err.Error(), "auto") {
		t.Fatalf("错误应提示合法值, got %v", err)
	}
}
