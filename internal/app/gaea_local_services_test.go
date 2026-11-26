package app

// gaea_local_services_test.go — 本地服务目录卡验收（电脑操控配套刀）：
// 结构不变量 + 被动探活（闭端口→未就绪，绝不拉起服务）+ 本地 OCR 定点的
// 持久化与回报。2026-11-26 精简后本地 OCR 只剩 Unlimited-OCR-2 一张卡
// （卡 ID = local:<docmd kind>），已退役引擎的定点请求如实拒绝。
import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGaeaLocalServiceModels_Structure(t *testing.T) {
	// 闭端口：OCR 卡被动探活必须未就绪（且不得拉起任何服务）。
	t.Setenv("GAEA_UOCR2_LLAMA_PORT", "1")
	a := &App{core: &core{}}
	cards := a.GaeaLocalServiceModels()
	if len(cards) != 3 {
		t.Fatalf("应下发 3 张本地服务卡（OCR / 识图 / STT）: %d", len(cards))
	}
	byID := map[string]LocalServiceModel{}
	for _, c := range cards {
		byID[c.EngineID] = c
		if !strings.HasPrefix(c.EngineID, "local:") {
			t.Fatalf("卡 EngineID 必须 local: 前缀: %s", c.EngineID)
		}
		if c.Detail == "" {
			t.Fatalf("%s detail 不得为空", c.EngineID)
		}
	}
	for _, id := range []string{"local:uocr2", "local:vision", "local:stt"} {
		if _, ok := byID[id]; !ok {
			t.Fatalf("缺本地服务卡 %s", id)
		}
	}
	for _, retired := range []string{"local:ovis", "local:tesseract", "local:uocr2-python", "local:uocr2-llama"} {
		if _, ok := byID[retired]; ok {
			t.Fatalf("已退役引擎不得再下发卡片: %s", retired)
		}
	}
	ocr := byID["local:uocr2"]
	if ocr.Kind != "ocr" {
		t.Fatalf("local:uocr2 kind 应为 ocr: %s", ocr.Kind)
	}
	if ocr.Ready {
		t.Fatal("local:uocr2 闭端口不得就绪（且不得拉起服务）")
	}
	if byID["local:vision"].Kind != "vision" || byID["local:stt"].Kind != "stt" {
		t.Fatalf("vision/stt 卡 kind 错: %+v %+v", byID["local:vision"], byID["local:stt"])
	}
	// vision/stt 就绪态与文件事实自洽（本机有 mmproj/voice 权重则 true，CI 无则 false）。
	if byID["local:vision"].Ready != (managedProfileMmprojPath(llamaProfiles[0]) != "") {
		t.Fatal("vision 就绪态必须与 mmproj 文件事实一致")
	}
	if byID["local:stt"].Ready != voiceV2FileExists(voiceNativeModelPath("model.int8.onnx")) {
		t.Fatal("stt 就绪态必须与本地模型文件事实一致")
	}
}

func TestGaeaLocalServiceModels_HealthUp(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer up.Close()
	t.Setenv("GAEA_UOCR2_LLAMA_URL", up.URL)
	a := &App{core: &core{}}
	for _, c := range a.GaeaLocalServiceModels() {
		if c.EngineID == "local:uocr2" && !c.Ready {
			t.Fatal("健康服务的 uocr2 卡应就绪")
		}
	}
}

func TestSetActiveOCRModel_LocalKind(t *testing.T) {
	// config.Save 写 UserHomeDir 下的 .gaea_config.json——重定向到临时目录
	//（与 feature_model_handler_test 同纪律，不污染真实配置）。
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	c := &core{}
	// 未知本地 kind → 拒绝（不写配置、不落状态）。
	if err := c.SetActiveOCRModel("local:bogus", "x"); err == nil {
		t.Fatal("未知本地 kind 必须拒")
	}
	if c.activeOCRLocal != "" {
		t.Fatal("拒绝路径不得落状态")
	}
	// 已退役引擎 → 拒绝并指路，不静默换成 uocr2。
	for _, retired := range []string{"local:ovis", "local:tesseract"} {
		err := c.SetActiveOCRModel(retired, "x")
		if err == nil || !strings.Contains(err.Error(), "退役") || !strings.Contains(err.Error(), "local:uocr2") {
			t.Fatalf("%s 应报「已退役」并指路: %v", retired, err)
		}
	}
	if c.activeOCRLocal != "" {
		t.Fatal("退役路径不得落状态")
	}
	// 合法本地 kind → 状态+持久化（config.Save 已重定向临时目录）+回报。
	if err := c.SetActiveOCRModel("local:uocr2", "unlimited-ocr2-llama"); err != nil {
		t.Fatalf("local:uocr2 应接受: %v", err)
	}
	if c.activeOCRLocal != "uocr2" {
		t.Fatalf("activeOCRLocal 应为 uocr2: %q", c.activeOCRLocal)
	}
	got := c.GetActiveOCRModel()
	if got["engine"] != "local:uocr2" {
		t.Fatalf("回报 engine 应 local:uocr2: %v", got)
	}
	// 清空绑定回默认（三键全清）。
	if err := c.SetActiveOCRModel("", ""); err != nil {
		t.Fatalf("清空应成功: %v", err)
	}
	if c.activeOCRLocal != "" || c.activeOCREngine != "" {
		t.Fatal("清空后状态必须归零")
	}
	if got := c.GetActiveOCRModel(); got["engine"] != "" {
		t.Fatalf("清空后回报应回 herdsman 腿语义: %v", got)
	}
}

// 启动装载：配置里残留的已退役引擎定点被清掉（不带着坏值去调 docmd）；
// 合法值与空值原样保留。
func TestNormalizeLocalOCRKind(t *testing.T) {
	for in, want := range map[string]string{
		"":          "",
		"uocr2":     "uocr2",
		" Uocr2 ":   "uocr2",
		"ovis":      "",
		"tesseract": "",
		"bogus":     "",
	} {
		if got := normalizeLocalOCRKind(in); got != want {
			t.Errorf("normalizeLocalOCRKind(%q) = %q, want %q", in, got, want)
		}
	}
}
