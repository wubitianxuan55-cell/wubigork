package schedule

// xlsx_test.go — Excel 导入导出用例（v4.134.0 刀D3）。
//
// 口径：往返一致（任务名/层级/工期/里程碑/搭接/开工日）；表头别名容错；
// 前置引用「2、2FS、2SS+3」按序号回链；工期文本「5 个工作日」提取；
// WBS 深度与空工期行的分组判定；CPM 循环/未知表头 fail-closed。

import (
	"strconv"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func xlsxSampleProject() Project {
	return Project{
		Name:      "样例工程",
		StartDate: "2026-09-07",
		Tasks: []Task{
			{ID: "g1", Name: "前期准备", Level: 0},
			{ID: "a", Name: "场地三通一平", Duration: 10, Level: 1},
			{ID: "b", Name: "施工许可办理", Duration: 15, Level: 1},
			{ID: "m", Name: "开工令", Level: 1, IsMilestone: true},
			{ID: "g2", Name: "基础工程", Level: 0},
			{ID: "c", Name: "土方开挖", Duration: 12, Level: 1},
		},
		Links: []Link{
			{From: "a", To: "b", Type: FS, Lag: 3},
			{From: "b", To: "m", Type: FS, Lag: 0},
			{From: "m", To: "c", Type: SS, Lag: 5},
		},
	}
}

func TestExportXlsxRoundTrip(t *testing.T) {
	p := xlsxSampleProject()
	data, err := ExportXlsx(p)
	if err != nil {
		t.Fatalf("导出失败：%v", err)
	}
	got, err := ImportXlsx(data)
	if err != nil {
		t.Fatalf("导入失败：%v", err)
	}
	if got.Name != xlsxSheetName {
		t.Errorf("工程名=工作表名：%q", got.Name)
	}
	if got.StartDate != p.StartDate {
		t.Errorf("开工日：%q，希望 %q", got.StartDate, p.StartDate)
	}
	if len(got.Tasks) != len(p.Tasks) {
		t.Fatalf("任务数 %d，希望 %d", len(got.Tasks), len(p.Tasks))
	}
	for i, want := range p.Tasks {
		have := got.Tasks[i]
		if have.Name != want.Name {
			t.Errorf("行 %d 名称：%q，希望 %q", i+1, have.Name, want.Name)
		}
		if have.Level != want.Level {
			t.Errorf("行 %d（%s）层级 %d，希望 %d", i+1, want.Name, have.Level, want.Level)
		}
		if have.IsMilestone != want.IsMilestone {
			t.Errorf("行 %d（%s）里程碑标记不一致", i+1, want.Name)
		}
		wantDur := want.Duration
		if want.Level == 0 || want.IsMilestone {
			wantDur = 0
		}
		if have.Duration != wantDur {
			t.Errorf("行 %d（%s）工期 %d，希望 %d", i+1, want.Name, have.Duration, wantDur)
		}
	}
	if len(got.Links) != len(p.Links) {
		t.Fatalf("搭接数 %d，希望 %d", len(got.Links), len(p.Links))
	}
	// 回链按名称对照：导出前置引用=原任务行号，导入 t<N> id 与行号一一对应
	idName := func(pr Project, id string) string {
		n, _ := strconv.Atoi(strings.TrimPrefix(id, "t"))
		return pr.Tasks[n-1].Name
	}
	for i, want := range p.Links {
		have := got.Links[i]
		wantFrom := p.Tasks[rowNoOf(p.Tasks, want.From)-1].Name
		wantTo := p.Tasks[rowNoOf(p.Tasks, want.To)-1].Name
		if idName(got, have.From) != wantFrom || idName(got, have.To) != wantTo {
			t.Errorf("搭接 %d 回链：%s→%s，希望 %s→%s", i+1, idName(got, have.From), idName(got, have.To), wantFrom, wantTo)
		}
		if have.Type != want.Type || have.Lag != want.Lag {
			t.Errorf("搭接 %d 类型/时距：%s+%d，希望 %s+%d", i+1, have.Type, have.Lag, want.Type, want.Lag)
		}
	}
}

func TestImportXlsxAliases(t *testing.T) {
	f := excelize.NewFile()
	rows := [][]any{
		{"工作名称", "持续时间", "开工日期", "紧前工作"},
		{"场地平整", "5 个工作日", "2026/9/7", ""},
		{"管线铺设", "8", "2026/9/10", "1"},
		{"竣工验收", "0", "2026/9/20", "2FS+3"},
	}
	for r, row := range rows {
		for c, v := range row {
			cell, _ := excelize.CoordinatesToCellName(c+1, r+1)
			if err := f.SetCellValue("Sheet1", cell, v); err != nil {
				t.Fatal(err)
			}
		}
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	p, err := ImportXlsx(buf.Bytes())
	if err != nil {
		t.Fatalf("导入失败：%v", err)
	}
	if len(p.Tasks) != 3 {
		t.Fatalf("任务数 %d，希望 3", len(p.Tasks))
	}
	if p.Tasks[0].Duration != 5 {
		t.Errorf("工期文本「5 个工作日」应提取 5，得 %d", p.Tasks[0].Duration)
	}
	if p.StartDate != "2026-09-07" {
		t.Errorf("开工日取最早开始：%q", p.StartDate)
	}
	if !p.Tasks[2].IsMilestone {
		t.Errorf("0 工期叶任务应为里程碑")
	}
	if len(p.Links) != 2 {
		t.Fatalf("搭接数 %d，希望 2", len(p.Links))
	}
	if p.Links[0].Type != FS || p.Links[0].Lag != 0 {
		t.Errorf("「1」应解析为 FS+0：%s+%d", p.Links[0].Type, p.Links[0].Lag)
	}
	if p.Links[1].Type != FS || p.Links[1].Lag != 3 {
		t.Errorf("「2FS+3」应解析为 FS+3：%s+%d", p.Links[1].Type, p.Links[1].Lag)
	}
}

func TestImportXlsxRefsVariants(t *testing.T) {
	f := excelize.NewFile()
	rows := [][]any{
		{"序号", "任务名称", "工期", "前置"},
		{"1", "A", "3", ""},
		{"2", "B", "2", "1, 3SS-1"},
		{"3", "C", "4", "1fs"},
		{"4", "D", "1", "999FS"}, // 未知序号：跳过
	}
	for r, row := range rows {
		for c, v := range row {
			cell, _ := excelize.CoordinatesToCellName(c+1, r+1)
			_ = f.SetCellValue("Sheet1", cell, v)
		}
	}
	buf, _ := f.WriteToBuffer()
	p, err := ImportXlsx(buf.Bytes())
	if err != nil {
		t.Fatalf("导入失败：%v", err)
	}
	// A→B(FS0)、C→B(SS-1)、B→C(FS0，小写 fs)；999FS 跳过
	if len(p.Links) != 3 {
		t.Fatalf("搭接数 %d，希望 3（%v）", len(p.Links), p.Links)
	}
	if p.Links[0].From != "t1" || p.Links[0].To != "t2" {
		t.Errorf("「1」回链 A→B：%v", p.Links[0])
	}
	if p.Links[1].Type != SS || p.Links[1].Lag != -1 {
		t.Errorf("「3SS-1」：%s+%d", p.Links[1].Type, p.Links[1].Lag)
	}
	if p.Links[2].Type != FS || p.Links[2].From != "t1" || p.Links[2].To != "t3" {
		t.Errorf("「1fs」小写类型：%v", p.Links[2])
	}
}

func TestImportXlsxLevelsByWbs(t *testing.T) {
	f := excelize.NewFile()
	rows := [][]any{
		{"WBS", "任务名称", "工期"},
		{"1", "前期准备", ""},
		{"1.1", "场地平整", "5"},
		{"1.2", "管线铺设", "8"},
		{"2", "验收", "0"},
	}
	for r, row := range rows {
		for c, v := range row {
			cell, _ := excelize.CoordinatesToCellName(c+1, r+1)
			_ = f.SetCellValue("Sheet1", cell, v)
		}
	}
	buf, _ := f.WriteToBuffer()
	p, err := ImportXlsx(buf.Bytes())
	if err != nil {
		t.Fatalf("导入失败：%v", err)
	}
	wantLevels := []int{0, 1, 1, 1}
	for i, lv := range wantLevels {
		if p.Tasks[i].Level != lv {
			t.Errorf("行 %d 层级 %d，希望 %d", i+1, p.Tasks[i].Level, lv)
		}
	}
}

func TestImportXlsxLevelsWithoutWbs(t *testing.T) {
	f := excelize.NewFile()
	rows := [][]any{
		{"任务名称", "工期(天)"},
		{"土建工程", ""}, // 空工期行=分组
		{"土方开挖", "12"},
		{"垫层浇筑", "3"},
	}
	for r, row := range rows {
		for c, v := range row {
			cell, _ := excelize.CoordinatesToCellName(c+1, r+1)
			_ = f.SetCellValue("Sheet1", cell, v)
		}
	}
	buf, _ := f.WriteToBuffer()
	p, err := ImportXlsx(buf.Bytes())
	if err != nil {
		t.Fatalf("导入失败：%v", err)
	}
	if p.Tasks[0].Level != 0 || p.Tasks[1].Level != 1 || p.Tasks[2].Level != 1 {
		t.Errorf("层级判定（无 WBS 列）：%v", p.Tasks)
	}
}

func TestExportXlsxFailClosedOnCycle(t *testing.T) {
	p := Project{
		StartDate: "2026-09-07",
		Tasks:     []Task{{ID: "a", Name: "A", Duration: 1, Level: 1}, {ID: "b", Name: "B", Duration: 1, Level: 1}},
		Links:     []Link{{From: "a", To: "b", Type: FS}, {From: "b", To: "a", Type: FS}},
	}
	if _, err := ExportXlsx(p); err == nil {
		t.Fatal("循环依赖导出应 fail-closed")
	}
}

func TestImportXlsxUnknownHeader(t *testing.T) {
	f := excelize.NewFile()
	rows := [][]any{{"甲", "乙"}, {"1", "2"}}
	for r, row := range rows {
		for c, v := range row {
			cell, _ := excelize.CoordinatesToCellName(c+1, r+1)
			_ = f.SetCellValue("Sheet1", cell, v)
		}
	}
	buf, _ := f.WriteToBuffer()
	_, err := ImportXlsx(buf.Bytes())
	if err == nil || !strings.Contains(err.Error(), "未识别表头") {
		t.Fatalf("未知表头应报可读错误：%v", err)
	}
}
