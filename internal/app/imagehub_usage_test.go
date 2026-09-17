package app

// 画室消耗（阶段一刀 E，规格 进度计划/gaea-studio-usage-20260917.md）测试：
// 当月过滤/分组计数/估算算术/未定价诚实/空台账零报告/记录单价优先于目录回落。

import (
	"path/filepath"
	"testing"
	"time"
)

// usageAt 测试直调聚合实现（cwd 参数化；绕开 gaeaCwd 全局态——与
// ImageHubAssets 同款生产口径由绑定入口负责）。
func usageAt(t *testing.T, cwd string) ImageHubMonthlyUsageReport {
	t.Helper()
	report, err := imageHubMonthlyUsageAt(cwd, "play")
	if err != nil {
		t.Fatalf("聚合: %v", err)
	}
	return report
}

func recordUsageLine(t *testing.T, cwd, model, backend, cost, createdAt string) {
	t.Helper()
	led := newImageHubLedger(cwd)
	if err := led.record("play", imageHubLedgerRecord{
		Meta: imageHubAssetMeta{
			Space: "play", SourceBoard: "imagegen", Capability: string(CapabilityMediaGenerate),
			Backend: backend, Model: model, Cost: cost, CreatedAt: createdAt, AIFlag: true,
		},
		Asset: imageHubAsset{ID: "ih-" + createdAt + model + backend, Kind: ImageHubAssetKindImage, Path: filepath.Join(cwd, "x.png")},
	}); err != nil {
		t.Fatalf("登记: %v", err)
	}
}

func TestImageHubMonthlyUsage_EmptyLedger(t *testing.T) {
	cwd := t.TempDir()
	report := usageAt(t, cwd)
	if report.Total != 0 || report.ByModel == nil {
		t.Fatalf("空报告形状不对: %+v", report)
	}
	if report.Estimated != "本月还没有创作记录" {
		t.Fatalf("空台账提示不对: %s", report.Estimated)
	}
}

func TestImageHubMonthlyUsage_Aggregation(t *testing.T) {
	cwd := t.TempDir()
	now := time.Now()
	thisMonth := now.Format("2006-01")
	lastMonth := now.AddDate(0, -1, 0).Format("2006-01")
	recordUsageLine(t, cwd, "glm-image", "glm", "0.1 CNY/张", thisMonth+"-01T10:00:00+08:00")
	recordUsageLine(t, cwd, "glm-image", "glm", "0.1 CNY/张", thisMonth+"-02T10:00:00+08:00")
	recordUsageLine(t, cwd, "krea2", "comfyui", "0", thisMonth+"-03T10:00:00+08:00")
	recordUsageLine(t, cwd, "grok-imagine-image", "openai", "未定价", thisMonth+"-04T10:00:00+08:00")
	recordUsageLine(t, cwd, "glm-image", "glm", "0.1 CNY/张", lastMonth+"-28T10:00:00+08:00") // 上月剔除

	report := usageAt(t, cwd)
	if report.Month != thisMonth || report.Total != 4 {
		t.Fatalf("总数不对（上月应剔除）: %+v", report)
	}
	if report.Unpriced != 1 || report.FreeCount != 1 {
		t.Fatalf("未定价/免费计数不对: %+v", report)
	}
	// 分组：glm-image 2 张 0.20；krea2 免费；grok 未定价
	byModel := map[string]ImageHubUsageModelRow{}
	for _, r := range report.ByModel {
		byModel[r.Model] = r
	}
	if byModel["glm-image"].Count != 2 || byModel["glm-image"].EstCost != "0.20 CNY" {
		t.Fatalf("glm 聚合不对: %+v", byModel["glm-image"])
	}
	if byModel["krea2"].EstCost != "0 CNY" {
		t.Fatalf("本地免费应 0 CNY: %+v", byModel["krea2"])
	}
	if byModel["grok-imagine-image"].EstCost != "" || byModel["grok-imagine-image"].UnitCost != "未定价" {
		t.Fatalf("未定价行应留空估算: %+v", byModel["grok-imagine-image"])
	}
	if report.Estimated != "0.20 CNY（另有 1 张未定价）" {
		t.Fatalf("合计口径不对: %s", report.Estimated)
	}
}

func TestImageHubMonthlyUsage_RecordCostOverCatalog(t *testing.T) {
	// 记录单价是事实源：记录空价时回落目录表（qwen-image-3.0-pro=0.18）
	cwd := t.TempDir()
	thisMonth := time.Now().Format("2006-01")
	recordUsageLine(t, cwd, "qwen-image-3.0-pro", "openai", "", thisMonth+"-05T09:00:00+08:00")

	report := usageAt(t, cwd)
	if report.Total != 1 {
		t.Fatalf("总数不对: %+v", report)
	}
	if len(report.ByModel) != 1 || report.ByModel[0].UnitCost != "0.18 CNY/张" || report.ByModel[0].EstCost != "0.18 CNY" {
		t.Fatalf("目录回落不对: %+v", report.ByModel)
	}
	if report.Estimated != "0.18 CNY" {
		t.Fatalf("合计不对: %s", report.Estimated)
	}
}

func TestParseUnitCostCNY(t *testing.T) {
	for _, c := range []struct {
		in string
		v  float64
		ok bool
	}{
		{"0.1 CNY/张", 0.1, true},
		{"0.18 CNY/张", 0.18, true},
		{"0 CNY/张", 0, true},
		{"0", 0, true},
		{"未定价", -1, false},
		{"", -1, false},
		{"免费", -1, false},
	} {
		v, ok := parseUnitCostCNY(c.in)
		if ok != c.ok || (ok && v != c.v) {
			t.Fatalf("%q → (%v,%v) want (%v,%v)", c.in, v, ok, c.v, c.ok)
		}
	}
}
