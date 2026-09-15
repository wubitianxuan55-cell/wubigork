package app

// 7.2-2 判据② 技能调用计数：持久化往返/损坏回空/绑定视图排序。
// GAEA_DATA_ROOT 隔离数据根（gaea_data_backup_test 先例）。
import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSkillStatsRecordRoundTrip(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GAEA_DATA_ROOT", root)

	recordSkillUse("cost-compose", true)
	recordSkillUse("cost-compose", true)
	recordSkillUse("cost-compose", false)

	f := loadSkillStats(root)
	st := f.Skills["cost-compose"]
	if st.Calls != 3 || st.Ok != 2 || st.LastAt == 0 {
		t.Fatalf("累计不对: %+v", st)
	}
	// 落盘形状：camelCase + version 1。
	b, err := os.ReadFile(filepath.Join(root, "skill_stats.json"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, want := range []string{`"version": 1`, `"calls": 3`, `"ok": 2`} {
		if !strings.Contains(s, want) {
			t.Fatalf("落盘缺 %q: %s", want, s)
		}
	}
}

func TestSkillStatsCorruptFileReturnsEmpty(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GAEA_DATA_ROOT", root)
	if err := os.WriteFile(filepath.Join(root, "skill_stats.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if f := loadSkillStats(root); len(f.Skills) != 0 {
		t.Fatalf("损坏文件应回空表: %+v", f)
	}
	// version≠1 同样回空（未知未来版本不盲读）。
	if err := os.WriteFile(filepath.Join(root, "skill_stats.json"), []byte(`{"version":2,"skills":{"x":{"calls":1}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if f := loadSkillStats(root); len(f.Skills) != 0 {
		t.Fatalf("version 2 应回空表: %+v", f)
	}
}

func TestSkillStatsBindingViewSorted(t *testing.T) {
	root := t.TempDir()
	t.Setenv("GAEA_DATA_ROOT", root)

	recordSkillUse("alpha", true)
	recordSkillUse("beta", true)
	recordSkillUse("beta", true)

	var a App
	v := a.GaeaSkillStats()
	if len(v) != 2 {
		t.Fatalf("应有 2 条: %+v", v)
	}
	if v[0].Name != "beta" || v[0].Calls != 2 {
		t.Fatalf("calls 降序不对: %+v", v)
	}
	// 绑定视图 JSON 形状（Wails 线上 camelCase）。
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"name":"beta"`) || !strings.Contains(string(b), `"lastAt"`) {
		t.Fatalf("视图 JSON 形状不对: %s", b)
	}
}
