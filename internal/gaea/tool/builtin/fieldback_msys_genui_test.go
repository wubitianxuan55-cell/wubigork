package builtin

// fieldback_msys_genui_test.go — 2026-11-21 造价门实测复盘钉：resolveIn 的
// MSYS 盘符翻译（2.2 镜像目录树根修）与 genui_validate 双形态（4.3）。

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestResolveInMSYSDrive(t *testing.T) {
	got := resolveIn(`D:\ws`, "/d/新工作/沿江/x.md")
	if got != `D:\新工作\沿江\x.md` {
		t.Fatalf("/d/... 应翻成 D:\\...，got %q", got)
	}
	if got := resolveIn(`D:\ws`, "a/b.md"); got != `D:\ws\a\b.md` {
		t.Fatalf("相对路径应 Join：got %q", got)
	}
	if got := resolveIn(`D:\ws`, `E:\x\y.md`); got != `E:\x\y.md` {
		t.Fatalf("Windows 绝对路径应原样：got %q", got)
	}
	// 2026-11-21 沿江反馈 P2-11：正斜杠盘符绝对路径归一为反斜杠——两种斜杠
	// 都认，不再逼试错。
	if got := resolveIn(`D:\ws`, `D:/新工作/南岸/清单.md`); got != `D:\新工作\南岸\清单.md` {
		t.Fatalf("正斜杠盘符绝对路径应归一为反斜杠，got %q", got)
	}
}

func TestGenuiValidateDualForm(t *testing.T) {
	run := func(args string) (string, error) {
		return genuiValidate{}.Execute(context.Background(), json.RawMessage(args))
	}
	// 字符串形态（旧契约）不受影响。
	strForm := `{"spec":"{\"type\":\"code\",\"code\":\"x\"}"}`
	out, err := run(strForm)
	if err != nil || !strings.Contains(out, "OK") {
		t.Fatalf("字符串形态应通过：out=%q err=%v", out, err)
	}
	// 对象形态（旧实现 unmarshal 炸「of type string」）。
	out2, err := run(`{"spec":{"type":"code","code":"x"}}`)
	if err != nil || !strings.Contains(out2, "OK") {
		t.Fatalf("对象形态应通过：out=%q err=%v", out2, err)
	}
	// 非法 spec 报 ❌ 而非参数错误。
	out3, err := run(`{"spec":{"type":"nope"}}`)
	if err != nil {
		t.Fatalf("非法 spec 应报校验失败而非工具错误: %v", err)
	}
	if !strings.Contains(out3, "❌") {
		t.Fatalf("非法 spec 应回 ❌：out=%q", out3)
	}
}
