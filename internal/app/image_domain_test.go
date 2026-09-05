package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── 图像能力域 T0 契约单测（设计 docs/gaea-image-domain-t0-contract-design-2026-09.md）──

func TestImageDomainRegistry(t *testing.T) {
	for capName, wantAsset := range map[ImageCapability]bool{
		CapabilityVisionRead:       false,
		CapabilityVisionUnderstand: false,
		CapabilityMediaGenerate:    true,
		CapabilityMediaEdit:        true,
		CapabilityMediaDiagram:     true,
	} {
		e, err := imageDomainEntry(capName)
		if err != nil {
			t.Fatalf("imageDomainEntry(%s): %v", capName, err)
		}
		if e.ProducesAsset != wantAsset {
			t.Errorf("%s ProducesAsset = %v, want %v", capName, e.ProducesAsset, wantAsset)
		}
	}
	// T0 改图只立契约位，实现留 T3 → Available=false。
	e, _ := imageDomainEntry(CapabilityMediaEdit)
	if e.Available {
		t.Fatalf("media.edit 在 T0 不应可用")
	}
	// 未知原语 fail-closed。
	if _, err := imageDomainEntry(ImageCapability("no.such.cap")); err == nil {
		t.Fatalf("未知原语应报错")
	}
}

func TestImageHubLedgerPathSpaces(t *testing.T) {
	p := imageHubLedgerPath("C:/ws", "play")
	if !strings.Contains(filepath.ToSlash(p), ".gaea/play/imagehub/assets.jsonl") {
		t.Fatalf("play 路径 = %q", p)
	}
	for _, sp := range []string{"work", ""} {
		p := imageHubLedgerPath("C:/ws", sp)
		if strings.Contains(p, "play") || !strings.HasSuffix(filepath.ToSlash(p), ".gaea/imagehub/assets.jsonl") {
			t.Fatalf("work/空空间路径 = %q", p)
		}
	}
}

func TestImageHubLedgerRecordList(t *testing.T) {
	cwd := t.TempDir()
	led := newImageHubLedger(cwd)
	first := imageHubLedgerRecord{
		Meta:  imageHubAssetMeta{Space: "play", SourceBoard: "novel", Capability: string(CapabilityMediaGenerate), AIFlag: true},
		Asset: imageHubAsset{ID: "ih-1", Kind: ImageHubAssetKindImage, Path: filepath.Join(cwd, "a.png")},
	}
	second := imageHubLedgerRecord{
		Meta:  imageHubAssetMeta{Space: "play", SourceBoard: "imagegen", Capability: string(CapabilityMediaGenerate), AIFlag: true},
		Asset: imageHubAsset{ID: "ih-2", Kind: ImageHubAssetKindVideo, Path: filepath.Join(cwd, "b.mp4")},
	}
	if err := led.record("play", first); err != nil {
		t.Fatalf("record first: %v", err)
	}
	if err := led.record("play", second); err != nil {
		t.Fatalf("record second: %v", err)
	}
	all := led.list("play", 0)
	if len(all) != 2 || all[1].Asset.ID != "ih-2" {
		t.Fatalf("list all = %+v", all)
	}
	last := led.list("play", 1)
	if len(last) != 1 || last[0].Asset.ID != "ih-2" {
		t.Fatalf("list limit = %+v", last)
	}
	// 缺 ID/路径拒绝。
	if err := led.record("work", imageHubLedgerRecord{Asset: imageHubAsset{ID: "x"}}); err == nil {
		t.Fatalf("缺路径应拒绝")
	}
}

func TestImageHubLedgerBadLineSkipped(t *testing.T) {
	cwd := t.TempDir()
	led := newImageHubLedger(cwd)
	good := imageHubLedgerRecord{
		Meta:  imageHubAssetMeta{Space: "work", SourceBoard: "imagegen", Capability: string(CapabilityMediaGenerate), AIFlag: true},
		Asset: imageHubAsset{ID: "ih-ok", Kind: ImageHubAssetKindImage, Path: filepath.Join(cwd, "ok.png")},
	}
	if err := led.record("work", good); err != nil {
		t.Fatalf("record: %v", err)
	}
	p := imageHubLedgerPath(cwd, "work")
	bad, err := os.OpenFile(p, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("open ledger: %v", err)
	}
	if _, err := bad.WriteString("not-json\n"); err != nil {
		_ = bad.Close()
		t.Fatalf("write bad line: %v", err)
	}
	_ = bad.Close()
	got := led.list("work", 0)
	if len(got) != 1 || got[0].Asset.ID != "ih-ok" {
		t.Fatalf("坏行应跳过，got %+v", got)
	}
}

func TestImageHubLedgerPruneKeepsLatest(t *testing.T) {
	orig := imageHubLedgerMaxLines
	imageHubLedgerMaxLines = 3
	defer func() { imageHubLedgerMaxLines = orig }()

	cwd := t.TempDir()
	led := newImageHubLedger(cwd)
	for i := 1; i <= 5; i++ {
		id := "ih-" + string(rune('0'+i))
		if err := led.record("play", imageHubLedgerRecord{
			Meta:  imageHubAssetMeta{Space: "play", SourceBoard: "imagegen", Capability: string(CapabilityMediaGenerate), AIFlag: true},
			Asset: imageHubAsset{ID: id, Kind: ImageHubAssetKindImage, Path: filepath.Join(cwd, id+".png")},
		}); err != nil {
			t.Fatalf("record %d: %v", i, err)
		}
	}
	got := led.list("play", 0)
	if len(got) != 3 {
		t.Fatalf("折叠后应剩 3 条，got %d", len(got))
	}
	if got[0].Asset.ID != "ih-3" || got[2].Asset.ID != "ih-5" {
		t.Fatalf("折叠应保留最近 3 条: %+v", got)
	}
}

func TestImagePathWithinAny(t *testing.T) {
	root := t.TempDir()
	inside := filepath.Join(root, "exports", "a.png")
	if !imagePathWithinAny(inside, []string{filepath.Join(root, "exports")}) {
		t.Fatalf("根内路径应放行")
	}
	outside := filepath.Join(root, "other", "a.png")
	if imagePathWithinAny(outside, []string{filepath.Join(root, "exports")}) {
		t.Fatalf("根外路径应拒绝")
	}
	// 根为空时 helper 一律 false；「跳过校验」由 record 层在 len(roots)==0 时承担。
	if imagePathWithinAny(inside, nil) {
		t.Fatalf("空根列表 helper 不应放行")
	}
}

func TestRecordImageHubGeneratedAssetRootAndGate(t *testing.T) {
	origGate := imageHubLedgerRuntimeCheck
	imageHubLedgerRuntimeCheck = func() bool { return true }
	defer func() { imageHubLedgerRuntimeCheck = origGate }()

	cwd := t.TempDir()
	exports := filepath.Join(cwd, ".gaea", "play", "exports")
	if err := os.MkdirAll(exports, 0o755); err != nil {
		t.Fatalf("mkdir exports: %v", err)
	}
	img := filepath.Join(exports, "cover-x.png")
	if err := os.WriteFile(img, []byte("png"), 0o644); err != nil {
		t.Fatalf("write img: %v", err)
	}
	err := recordImageHubGeneratedAsset(cwd, "play", "novel", "xai", "grok-imagine-image-quality",
		"封面 prompt", map[string]interface{}{"size": "768x1024"},
		imageHubAsset{Path: img}, []string{exports})
	if err != nil {
		t.Fatalf("record: %v", err)
	}
	got := newImageHubLedger(cwd).list("play", 0)
	if len(got) != 1 {
		t.Fatalf("应登记 1 条，got %d", len(got))
	}
	rec := got[0]
	if rec.Meta.SourceBoard != "novel" || rec.Meta.Model != "grok-imagine-image-quality" {
		t.Fatalf("meta 不完整: %+v", rec.Meta)
	}
	if rec.Meta.Cost != "未定价" || !rec.Meta.AIFlag {
		t.Fatalf("目录/溯源字段缺失: %+v", rec.Meta)
	}
	if rec.Asset.Path != img || rec.Asset.Kind != ImageHubAssetKindImage {
		t.Fatalf("asset 登记错误: %+v", rec.Asset)
	}
	// 根外路径拒绝。
	outside := filepath.Join(cwd, "outside.png")
	if err := os.WriteFile(outside, []byte("png"), 0o644); err != nil {
		t.Fatalf("write outside: %v", err)
	}
	if err := recordImageHubGeneratedAsset(cwd, "play", "novel", "", "krea2", "p", nil,
		imageHubAsset{Path: outside}, []string{exports}); err == nil {
		t.Fatalf("根外产物应拒绝登记")
	}
}

func TestImageModelCatalog(t *testing.T) {
	meta, ok := imageModelCatalogMetaFor("KREA2")
	if !ok || meta.Tier != imageTierLocalFree || meta.UnitCost != "0" {
		t.Fatalf("krea2 目录错误: %+v ok=%v", meta, ok)
	}
	meta, ok = imageModelCatalogMetaFor("qwen-image-3.0-pro")
	if !ok || meta.UnitCost != "0.18 CNY/张" {
		t.Fatalf("qwen 目录错误: %+v ok=%v", meta, ok)
	}
	if _, ok := imageModelCatalogMetaFor("some-future-model"); ok {
		t.Fatalf("未知模型应 ok=false")
	}
	cost, lic := imageHubCostAndLicense("no-such-model")
	if cost != "" || lic != "" {
		t.Fatalf("未知模型成本应诚实留空: cost=%q lic=%q", cost, lic)
	}
}

func TestImageHubAssetSummariesFilterAndOrder(t *testing.T) {
	cwd := t.TempDir()
	led := newImageHubLedger(cwd)
	mk := func(id, board string) imageHubLedgerRecord {
		return imageHubLedgerRecord{
			Meta: imageHubAssetMeta{
				Space: "play", SourceBoard: board, Capability: string(CapabilityMediaGenerate),
				Model: "krea2", CreatedAt: "2026-09-05T00:00:00Z", AIFlag: true,
			},
			Asset: imageHubAsset{ID: id, Kind: ImageHubAssetKindImage, Path: filepath.Join(cwd, id+".png")},
		}
	}
	for _, rec := range []imageHubLedgerRecord{
		mk("ih-1", "imagegen"),
		mk("ih-2", "novel"),
		mk("ih-3", "imagegen"),
	} {
		if err := led.record("play", rec); err != nil {
			t.Fatalf("record: %v", err)
		}
	}
	novelOnly := imageHubAssetSummaries(cwd, "play", "novel", "", 0)
	if len(novelOnly) != 1 || novelOnly[0].ID != "ih-2" {
		t.Fatalf("novel 过滤错误: %+v", novelOnly)
	}
	imagegenLimit2 := imageHubAssetSummaries(cwd, "play", "imagegen", "", 2)
	if len(imagegenLimit2) != 2 || imagegenLimit2[0].ID != "ih-3" || imagegenLimit2[1].ID != "ih-1" {
		t.Fatalf("倒序/limit 错误: %+v", imagegenLimit2)
	}
	// 未知来源 → 空。
	if got := imageHubAssetSummaries(cwd, "play", "no-such-board", "", 0); len(got) != 0 {
		t.Fatalf("未知来源应空: %+v", got)
	}
}

// TestImageHubLedgerGateDisarmedInTests 回归钉：登记运行态闸默认未武装
// （只有真实 App Startup 置位）。曾因闸只看 gaeaCfgSnapshot()!=nil 而 app 包
// 测试初始化了全局配置，导致 TestGenerateFreeImage_SafeMode 把登记写进源码树
// internal/app/.gaea/——本钉保证测试进程（不调 Startup）恒不落盘。
func TestImageHubLedgerGateDisarmedInTests(t *testing.T) {
	if imageHubLedgerRuntimeCheck() {
		t.Fatal("测试进程登记闸应为未武装（imageHubRuntimeArmed 只在 App.Startup 置位）")
	}
}
