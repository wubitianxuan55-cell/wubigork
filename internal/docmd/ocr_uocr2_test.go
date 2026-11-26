package docmd

// ocr_uocr2_test.go — uocr2 引擎（Unlimited-OCR-2，llama 后端）单测：结构标记清洗、
// 批识别编排（分窗 / 回退逐页）、provider 的地址缓存纪律。
// 全部为纯单测，不依赖真实服务：HTTP 面用 ocr_uocr2_llama_test.go 的 httptest 假服务。
// python 后端（/ocr、/ocr_batch）相关用例随 2026-11-26 退役一并删除；进程拉起的
// 进程纪律用例见 ocr_test.go。

import (
	"errors"
	"net/http"
	"strconv"
	"testing"
)

func TestUocr2CleanText(t *testing.T) {
	raw := "<PAGE><BLOCK>header[42, 16, 143, 27]<FORK>2026/9/20 21:35\n" +
		"<BLOCK>header[490, 16, 654, 29]<FORK>福建省环境监测行业指导价\n" +
		"<BLOCK>table[133, 36, 868, 874]<FORK><table><tr><td>序号</td></tr></table>\n" +
		"<BLOCK>footer[42, 971, 958, 982]<FORK>https://mp.weixin.qq.com/s?src=11 2/15"
	want := "2026/9/20 21:35\n福建省环境监测行业指导价\n<table><tr><td>序号</td></tr></table>\n" +
		"https://mp.weixin.qq.com/s?src=11 2/15"
	if got := uocr2CleanText(raw); got != want {
		t.Errorf("uocr2CleanText = %q, want %q", got, want)
	}
	if got := uocr2CleanText("  纯文本无标记  "); got != "纯文本无标记" {
		t.Errorf("无标记输入应原样去空白: %q", got)
	}
}

// TestUocr2CleanTextDropsMalformedBlock 坏标记行（<FORK> 缺失、坐标与正文粘连，
// GPU bf16 数字噪声实测）整行剥掉，好行保留——宁缺勿把坐标碎片漏进正文。
func TestUocr2CleanTextDropsMalformedBlock(t *testing.T) {
	raw := "<PAGE><BLOCK>header[42, 16, 143, 27]<FORK>好标题\n" +
		"<BLOCK>header[490, 16、坏行坐标粘连无FORK\n" +
		"<BLOCK>table[1, 2, 3, 4]<FORK><table><tr><td>正文</td></tr></table>"
	want := "好标题\n<table><tr><td>正文</td></tr></table>"
	if got := uocr2CleanText(raw); got != want {
		t.Errorf("uocr2CleanText = %q, want %q", got, want)
	}
}

// TestUocr2CleanTextNestedAngle 15 页全量实弹泄漏实证：BLOCK 与 FORK 之间出现
// 任何 `<`（嵌套标签/模型回显）时 [^<]* 形态漏清洗——非贪婪 .*? 行内匹配收口。
func TestUocr2CleanTextNestedAngle(t *testing.T) {
	for _, raw := range []string{
		"<BLOCK>table[1, <x]<FORK>内容",
		"<BLOCK>t[a]<b>提示</b><FORK>内容",
		"前缀<BLOCK>a[1]<i>x</i><FORK>后缀",
	} {
		if got := uocr2CleanText(raw); got != "内容" && got != "前缀后缀" {
			t.Errorf("uocr2CleanText(%q) = %q, 标记应被剥净", raw, got)
		}
	}
}

// mustTempPNG 写一个内容任意的临时「PNG」（客户端只读字节转发，服务端才解码）。
func mustTempPNG(t *testing.T) string {
	t.Helper()
	return writeFakeImage(t)
}

// TestUocr2ProviderDropsCacheOnTransportErr 传输层错误（服务中途消亡）必须清缓存
// base，让下一页重新探测/拉起；HTTP 状态错误（服务活着）不清。
func TestUocr2ProviderDropsCacheOnTransportErr(t *testing.T) {
	// 1) 服务活着但返回 500：错误透传，缓存保留。
	srv, _ := fakeLlamaServer(t, "", "stop", http.StatusInternalServerError)
	t.Setenv("GAEA_UOCR2_LLAMA_URL", srv.URL) // resolveBase 探测假服务 /health 即成功并缓存
	p := &uocr2OCRProvider{}
	if _, err := p.ExtractImage(mustTempPNG(t)); err == nil {
		t.Fatal("500 应报错")
	}
	if p.base != srv.URL {
		t.Fatalf("HTTP 状态错误不应清缓存（服务活着），base = %q", p.base)
	}

	// 2) 服务消亡（关掉监听）：传输层错误 → 缓存必须被清。
	srv.Close()
	if _, err := p.ExtractImage(mustTempPNG(t)); err == nil {
		t.Fatal("传输层失败应报错")
	}
	if p.base != "" {
		t.Fatalf("传输层错误后缓存应被清空重新探测, got %q", p.base)
	}
}

// TestUocr2ProviderExtractPagesDropsCacheOnTransportErr 批接口同款纪律。
func TestUocr2ProviderExtractPagesDropsCacheOnTransportErr(t *testing.T) {
	srv, _ := fakeLlamaServer(t, "<PAGE><BLOCK>x[1]<FORK>内容", "stop", http.StatusOK)
	t.Setenv("GAEA_UOCR2_LLAMA_URL", srv.URL)
	p := &uocr2OCRProvider{}
	got, err := p.ExtractPages([]string{mustTempPNG(t)})
	if err != nil || len(got) != 1 || got[0] != "内容" {
		t.Fatalf("ExtractPages 应成功: %v %+v", err, got)
	}
	srv.Close()
	if _, err := p.ExtractPages([]string{mustTempPNG(t)}); err == nil {
		t.Fatal("传输层失败应报错")
	}
	if p.base != "" {
		t.Fatalf("传输层错误后缓存应被清空, got %q", p.base)
	}
}

// TestOCRPageLoopBatchWinsAndFallsBack 批路径：成功窗整批命中；失败窗整窗回退
// 逐页；顺序/失败页语义与纯逐页一致，progress 按窗跳进。6 页 + 窗口 5 → 两窗。
func TestOCRPageLoopBatchWinsAndFallsBack(t *testing.T) {
	files := []string{"p1.png", "p2.png", "p3.png", "p4.png", "p5.png", "p6.png"}
	var gotProgress []int
	batchCalls := 0
	batchOCR := func(jobs []ocrPageJob) ([]string, error) {
		batchCalls++
		if jobs[0].num == 1 { // 第一窗（5 页）成功（文本须 ≥48 rune 过近空页守卫）
			long := func(tag string) string {
				return tag + "：检测项目水温 pH 值流量透明度浊度总硬度悬浮物色度电导率溶解氧均正确识别，计费单位点次与指导价金额逐行核对无误"
			}
			return []string{long("批-1"), long("批-2"), long("批-3"), long("批-4"), long("批-5")}, nil
		}
		return nil, errors.New("批失败") // 第二窗（1 页）失败 → 回退逐页
	}
	texts, total, failed, err := ocrPageLoopBatch(files, 1, "", func(pageNum int, pngPath string) (string, error) {
		return "逐页-" + strconv.Itoa(pageNum), nil
	}, batchOCR, func(done, totalN int) {
		gotProgress = append(gotProgress, done)
	})
	if err != nil {
		t.Fatalf("批失败窗回退逐页不应整体报错: %v", err)
	}
	if batchCalls != 2 {
		t.Fatalf("6 页应分 2 窗（窗口=%d）: %d", ocrBatchWindow, batchCalls)
	}
	want := []string{
		"批-1：检测项目水温 pH 值流量透明度浊度总硬度悬浮物色度电导率溶解氧均正确识别，计费单位点次与指导价金额逐行核对无误",
		"批-2：检测项目水温 pH 值流量透明度浊度总硬度悬浮物色度电导率溶解氧均正确识别，计费单位点次与指导价金额逐行核对无误",
		"批-3：检测项目水温 pH 值流量透明度浊度总硬度悬浮物色度电导率溶解氧均正确识别，计费单位点次与指导价金额逐行核对无误",
		"批-4：检测项目水温 pH 值流量透明度浊度总硬度悬浮物色度电导率溶解氧均正确识别，计费单位点次与指导价金额逐行核对无误",
		"批-5：检测项目水温 pH 值流量透明度浊度总硬度悬浮物色度电导率溶解氧均正确识别，计费单位点次与指导价金额逐行核对无误",
		"逐页-6",
	}
	if len(texts) != 6 {
		t.Fatalf("应 6 页全出: %+v", texts)
	}
	for i := range want {
		if texts[i] != want[i] {
			t.Fatalf("texts[%d] = %q, want %q", i, texts[i], want[i])
		}
	}
	if total != 6 || len(failed) != 0 {
		t.Fatalf("total=6 无失败页: total=%d failed=%v", total, failed)
	}
	if len(gotProgress) != 3 || gotProgress[0] != 0 || gotProgress[1] != 5 || gotProgress[2] != 6 {
		t.Fatalf("progress 应含起步事件(0)+按窗跳进 5/6: %+v", gotProgress)
	}
}

// TestOCRPageLoopBatchNearEmptyPageFallsBack 批结果含近空页（生成退化：仅页眉
// 无正文）→ 整窗回退逐页，近空页不进结果。
func TestOCRPageLoopBatchNearEmptyPageFallsBack(t *testing.T) {
	files := []string{"p1.png", "p2.png"}
	batchOCR := func([]ocrPageJob) ([]string, error) {
		// 页数对得上但第二页近空（3 字符 < 20 阈值）→ 不接受
		return []string{"第一页完整正文内容足够长", "页眉"}, nil
	}
	texts, _, failed, err := ocrPageLoopBatch(files, 1, "", func(pageNum int, pngPath string) (string, error) {
		return "逐页完整结果-" + strconv.Itoa(pageNum), nil
	}, batchOCR, nil)
	if err != nil {
		t.Fatalf("近空页回退逐页不应报错: %v", err)
	}
	if len(texts) != 2 || texts[0] != "逐页完整结果-1" || texts[1] != "逐页完整结果-2" {
		t.Fatalf("整窗应回退逐页: %+v", texts)
	}
	if len(failed) != 0 {
		t.Fatalf("不应有失败页: %v", failed)
	}
}

// TestOCRPageLoopBatchSkippedForSinglePage 单页不走批（gundam 逐页切片更优）。
func TestOCRPageLoopBatchSkippedForSinglePage(t *testing.T) {
	batchOCR := func([]ocrPageJob) ([]string, error) {
		t.Error("单页不应调用批识别")
		return nil, errors.New("不应到达")
	}
	texts, _, _, err := ocrPageLoopBatch([]string{"p1.png"}, 1, "",
		func(int, string) (string, error) { return "单页", nil }, batchOCR, nil)
	if err != nil || len(texts) != 1 || texts[0] != "单页" {
		t.Fatalf("单页应走逐页路径: %v %+v", err, texts)
	}
}
