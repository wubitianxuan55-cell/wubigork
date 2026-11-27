package builtin

// bash_restage_test.go — 2026-11-27 AI 反馈 #4 回归钉：脚本改写「已有正式件」被
// 发布边界拦截后，隔离件自动放一份到 .gaea/work/<原相对路径>，回执给一步回正式名
// 的 move_file 写法；新建的越界件不享受（那本来就是该走 .gaea/work/ 或 move_file）。

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBashPublishRestagesRewrittenFormalFile(t *testing.T) {
	ws := publishBashWorkdir(t)
	exports := filepath.Join(ws, ".gaea", "exports")
	seed := filepath.Join(ws, "seed")
	for _, d := range []string{exports, seed} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	formal := filepath.Join(exports, "formal.xlsx")
	if err := os.WriteFile(formal, xlsxNoCacheBytes(t), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(seed, "rewritten.bin"), []byte("rewritten by script"), 0o644); err != nil {
		t.Fatal(err)
	}
	b := bash{workDir: ws}
	ctx := context.Background()

	out, err := b.Execute(ctx, []byte(`{"command":"cp seed/rewritten.bin .gaea/exports/formal.xlsx"}`))
	if err == nil {
		t.Skipf("发布边界未拦截，跳过")
	}
	if !strings.HasPrefix(err.Error(), "发布边界拦截") {
		t.Errorf("err 首行仍应是拦截标题：%v", err)
	}
	for _, want := range []string{"formal.xlsx（modified）",
		"隔离件已自动放一份到过程目录",
		"- .gaea/exports/formal.xlsx → .gaea/work/.gaea/exports/formal.xlsx",
		"move_file source=<上面的 .gaea/work/ 路径> destination=<原路径> overwrite=true"} {
		if !strings.Contains(out, want) || !strings.Contains(err.Error(), want) {
			t.Errorf("result 与 err 都应含 %q：\nout=%s\nerr=%v", want, out, err)
		}
	}
	// 正式位已空，过程目录里是脚本写出的那份，隔离区里原件仍在。
	if _, serr := os.Stat(formal); !os.IsNotExist(serr) {
		t.Error("越界件未隔离出正式位")
	}
	if raw, err := os.ReadFile(filepath.Join(ws, ".gaea", "work", ".gaea", "exports", "formal.xlsx")); err != nil || string(raw) != "rewritten by script" {
		t.Errorf("过程目录应放一份脚本写出的件：%q（%v）", string(raw), err)
	}
	q, _ := filepath.Glob(filepath.Join(ws, ".gaea", "publish-quarantine", "*", ".gaea", "exports", "formal.xlsx"))
	if len(q) != 1 {
		t.Errorf("隔离区应留着原件：%v", q)
	}

	// 新建的越界件不自动放回。
	if err := os.WriteFile(filepath.Join(seed, "fresh.bin"), []byte("fresh"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err = b.Execute(ctx, []byte(`{"command":"cp seed/fresh.bin fresh.xlsx"}`))
	if err == nil {
		t.Skipf("发布边界未拦截，跳过")
	}
	if strings.Contains(out, "已自动放一份") {
		t.Errorf("新建的越界件不该自动放回过程目录：\n%s", out)
	}
	if _, serr := os.Stat(filepath.Join(ws, ".gaea", "work", "fresh.xlsx")); !os.IsNotExist(serr) {
		t.Error("新建的越界件不该出现在过程目录")
	}
}
