package builtin

// md_validate_refs_test.go — 2026-11-28 黄甲反馈回归钉：引用扫描的两类假阳性，以及
// 只传 paths 的全册模式不再被路径权限闸拦。

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 标准条文里的表图号、正文复合词都不是本文档引用（实测全册并集模式把
// 「GB 36600-2018 表 1」「…表 2」判成孤立引用硬伤 2 条，模型得自己写脚本复核）；
// 真引用照旧检出，对账本身不受影响。
func TestMDValidate_StandardRefAndCompoundWordsNotRefs(t *testing.T) {
	out, err := mdvRunMulti(t, map[string]string{
		"ch00.md": "# 摘要\n\n修复目标执行 GB 36600-2018 表 1 第一类用地筛选值；废水执行 GB 36600-2018 表 2。\n监测频次按 HJ 25.1 表 2 与 DB51/2362 表 4.2 布点。\n",
		"ch01.md": "# 概况\n\n本项目共代表 12 家单位进场施工，现场布置按施工总平面布置图 2 张执行。\n工艺见流程示意图 3 与剖面图 2，图表另行归档。\n",
	}, `,"checks":["orphans"]`)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "孤立引用") {
		t.Fatalf("标准条文表图号与正文复合词不得判孤立引用：%s", out)
	}

	// 真引用（含「附录A 表 3」——前缀不含数字，不在标准号排除之列）照旧检出。
	out, err = mdvRunMulti(t, map[string]string{
		"ch00.md": "修复目标见表1-1。\n",
		"ch01.md": "# 概况\n\n详见 图2-1 与附录A 表 3。\n",
	}, `,"checks":["orphans"]`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "孤立引用") {
		t.Fatalf("真孤立引用应照旧检出：%s", out)
	}
}

// 只传 paths 的全册模式：读目标须能解析（此前 ReadTarget 只认 path，路径权限闸
// 直接判 cannot resolve read target——实测主线程被拦 4 次）；path/paths/source/
// baseline 全部进读目标清单。
func TestMDValidate_ReadTargetsWithPathsOnly(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"a.md", "b.md", "base.md", "src.pdf"} {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	v := mdValidate{workDir: dir}

	target, ok := v.ReadTarget(json.RawMessage(`{"paths":["a.md","b.md"]}`))
	if !ok || filepath.Base(target) != "a.md" {
		t.Fatalf("只传 paths 时读目标应解析为首个文件：ok=%v target=%q", ok, target)
	}
	paths, err := v.ReadPaths(context.Background(), json.RawMessage(
		`{"path":"a.md","paths":["b.md"],"source":"src.pdf","baseline":"base.md"}`))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, p := range paths {
		names = append(names, filepath.Base(p))
	}
	if got := strings.Join(names, ","); got != "a.md,src.pdf,base.md,b.md" {
		t.Fatalf("读目标清单应覆盖 path/source/baseline/paths：%s", got)
	}
	if _, ok := v.ReadTarget(json.RawMessage(`{}`)); ok {
		t.Fatal("一个路径都没给时不应解析出读目标")
	}
}

// 工具层：只传 paths 即可跑全册对账（mdvRunMulti 本身就是这种调用形态）。
func TestMDValidate_PathAloneStillWorks(t *testing.T) {
	out, err := mdvRun(t, "见 表1-1。\n\n表1-1 甲表\n", `,"checks":["orphans"]`)
	if err != nil || !strings.HasPrefix(out, "OK:") {
		t.Fatalf("只传 path 应照常工作：out=%s err=%v", out, err)
	}
}
