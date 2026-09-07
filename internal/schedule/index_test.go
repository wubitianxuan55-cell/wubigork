package schedule

// index_test.go — 多工程索引用例（v4.139 差距 #15 刀1+刀2）：
// 缺失/损坏重建、轻扫收编（含 name 回填）、SetCurrent 校验、原子写、SafeSlugName。

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// writeIndexPlan 落一个最小计划文件（轻扫只读 name 字段，无需全文合法）。
func writeIndexPlan(t *testing.T, dir, rel, name string) {
	t.Helper()
	abs := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(map[string]string{"name": name})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

// findIndexEntry 按 rel 找索引条目。
func findIndexEntry(idx ScheduleIndex, rel string) (ScheduleIndexEntry, bool) {
	for _, e := range idx.Projects {
		if e.Rel == rel {
			return e, true
		}
	}
	return ScheduleIndexEntry{}, false
}

func TestLoadScheduleIndexMissingRebuildsFromScan(t *testing.T) {
	dir := t.TempDir()
	writeIndexPlan(t, dir, DefaultRelPath, "当前计划")
	writeIndexPlan(t, dir, "进度计划/办公楼二期.gsched.json", "办公楼二期")

	idx, err := LoadScheduleIndex(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if idx.Version != 1 || idx.Current != DefaultRelPath {
		t.Fatalf("version/current = %d/%q, want 1/%q", idx.Version, idx.Current, DefaultRelPath)
	}
	if len(idx.Projects) != 2 {
		t.Fatalf("projects = %d, want 2（扫描重建）：%+v", len(idx.Projects), idx.Projects)
	}
	if e, ok := findIndexEntry(idx, "进度计划/办公楼二期.gsched.json"); !ok || e.Name != "办公楼二期" || e.Archived {
		t.Fatalf("扫描条目 name 未从文件回填：%+v", e)
	}
	// 重建结果应原子写回磁盘
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(IndexRelPath))); err != nil {
		t.Fatalf("索引未写回：%v", err)
	}
}

func TestLoadScheduleIndexCorruptRebuilds(t *testing.T) {
	dir := t.TempDir()
	writeIndexPlan(t, dir, DefaultRelPath, "当前计划")
	if err := os.MkdirAll(filepath.Join(dir, ".gaea", "schedule"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(IndexRelPath)), []byte("{不是JSON"), 0o644); err != nil {
		t.Fatal(err)
	}
	idx, err := LoadScheduleIndex(dir)
	if err != nil {
		t.Fatalf("损坏索引应重建而非报错：%v", err)
	}
	if idx.Current != DefaultRelPath || len(idx.Projects) != 1 {
		t.Fatalf("重建失败：current=%q projects=%d", idx.Current, len(idx.Projects))
	}
}

func TestLoadScheduleIndexAdoptsStrayAndDropsMissing(t *testing.T) {
	dir := t.TempDir()
	writeIndexPlan(t, dir, DefaultRelPath, "当前计划")
	base := ScheduleIndex{Version: 1, Current: DefaultRelPath, Projects: []ScheduleIndexEntry{
		{Rel: DefaultRelPath, Name: "当前计划", UpdatedAt: time.Now()},
	}}
	if err := SaveScheduleIndex(dir, base); err != nil {
		t.Fatal(err)
	}

	// 游离文件（未登记）→ 轻扫收编 + name 回填
	writeIndexPlan(t, dir, "进度计划/办公楼二期.gsched.json", "办公楼二期")
	idx, err := LoadScheduleIndex(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(idx.Projects) != 2 {
		t.Fatalf("游离文件应收编：%+v", idx.Projects)
	}
	if e, ok := findIndexEntry(idx, "进度计划/办公楼二期.gsched.json"); !ok || e.Name != "办公楼二期" {
		t.Fatalf("收编条目 name 应回填：%+v", e)
	}
	if idx.Current != DefaultRelPath {
		t.Fatalf("收编不应动指针：%q", idx.Current)
	}

	// 用户手删文件 → 条目摘除（漂移兜底），指针不动
	if err := os.Remove(filepath.Join(dir, "进度计划", "办公楼二期.gsched.json")); err != nil {
		t.Fatal(err)
	}
	idx, err = LoadScheduleIndex(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(idx.Projects) != 1 || idx.Current != DefaultRelPath {
		t.Fatalf("消失文件应摘除：%+v current=%q", idx.Projects, idx.Current)
	}
}

func TestLoadScheduleIndexCurrentFallsToFirstWhenDefaultMissing(t *testing.T) {
	dir := t.TempDir()
	writeIndexPlan(t, dir, "进度计划/办公楼二期.gsched.json", "办公楼二期")
	idx, err := LoadScheduleIndex(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if idx.Current != "进度计划/办公楼二期.gsched.json" {
		t.Fatalf("current = %q, want 第一个条目", idx.Current)
	}
}

func TestSetCurrentScheduleValidation(t *testing.T) {
	dir := t.TempDir()
	writeIndexPlan(t, dir, DefaultRelPath, "当前计划")
	writeIndexPlan(t, dir, "进度计划/未登记.gsched.json", "未登记")
	if _, err := LoadScheduleIndex(dir); err != nil {
		t.Fatal(err)
	}

	// 非法 rel：空 / 后缀错 / 不存在 / .. 逃逸
	for _, bad := range []string{
		"",
		"进度计划/文档.txt",
		"进度计划/不存在.gsched.json",
		"../逃逸.gsched.json",
	} {
		if err := SetCurrentSchedule(dir, bad); err == nil {
			t.Fatalf("非法 rel 应拒绝：%q", bad)
		}
	}

	// 合法：未登记但在盘且后缀合法 → 顺带登记 + 切指针
	good := "进度计划/未登记.gsched.json"
	if err := SetCurrentSchedule(dir, good); err != nil {
		t.Fatalf("SetCurrent(未登记但在盘): %v", err)
	}
	idx, err := LoadScheduleIndex(dir)
	if err != nil {
		t.Fatal(err)
	}
	if idx.Current != good {
		t.Fatalf("current = %q, want %q", idx.Current, good)
	}
	e, ok := findIndexEntry(idx, good)
	if !ok || e.Name != "未登记" {
		t.Fatalf("SetCurrent 应登记条目并回填 name：%+v", e)
	}
}

func TestSaveScheduleIndexAtomicRoundTrip(t *testing.T) {
	dir := t.TempDir()
	// 收编 diff 以目录为权威：条目对应文件必须在盘才能存活过 Load
	writeIndexPlan(t, dir, DefaultRelPath, "当前计划")
	at := time.Date(2026, 9, 7, 12, 0, 0, 0, time.FixedZone("CST", 8*3600))
	in := ScheduleIndex{Version: 1, Current: DefaultRelPath, Projects: []ScheduleIndexEntry{
		{Rel: DefaultRelPath, Name: "当前计划", Archived: true, UpdatedAt: at},
	}}
	if err := SaveScheduleIndex(dir, in); err != nil {
		t.Fatal(err)
	}
	// 原子写：不留临时文件
	entries, err := os.ReadDir(filepath.Join(dir, ".gaea", "schedule"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Fatalf("残留临时文件：%s", e.Name())
		}
	}
	out, err := LoadScheduleIndex(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Projects) != 1 || !out.Projects[0].Archived || !out.Projects[0].UpdatedAt.Equal(at) {
		t.Fatalf("往返不一致：%+v", out.Projects)
	}
	// 零版本兜底为 1
	in.Version = 0
	if err := SaveScheduleIndex(dir, in); err != nil {
		t.Fatal(err)
	}
	out, err = LoadScheduleIndex(dir)
	if err != nil || out.Version != 1 {
		t.Fatalf("零版本应兜底为 1：v=%d err=%v", out.Version, err)
	}
}

func TestSafeSlugNameCases(t *testing.T) {
	// 中文保留
	if got := SafeSlugName("办公楼二期", map[string]bool{}); got != "办公楼二期" {
		t.Fatalf("中文名应保留：%q", got)
	}
	// 非法路径字符替换为 -
	if got := SafeSlugName(`a/b\c:d*e?f"g<h>i|j`, nil); got != "a-b-c-d-e-f-g-h-i-j" {
		t.Fatalf("非法字符应替换：%q", got)
	}
	if got := SafeSlugName("x\x01y", nil); got != "x-y" {
		t.Fatalf("控制符应替换：%q", got)
	}
	// Windows 尾部点/空格去除；空白名回落
	if got := SafeSlugName("计划. ", nil); got != "计划" {
		t.Fatalf("尾部点/空格应去除：%q", got)
	}
	if got := SafeSlugName("  ???  ", nil); got != "未命名" {
		t.Fatalf("清洗后为空应回落：%q", got)
	}
	// 冲突加序号，且选中结果登记回 existing
	existing := map[string]bool{"计划": true}
	if got := SafeSlugName("计划", existing); got != "计划-2" {
		t.Fatalf("冲突应加序号：%q", got)
	}
	if !existing["计划-2"] {
		t.Fatal("选中结果应登记回 existing")
	}
	if got := SafeSlugName("计划", existing); got != "计划-3" {
		t.Fatalf("连续取号应递增：%q", got)
	}
}
