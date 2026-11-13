package app

// office_macro_tools_test.go — 受控宏逃生舱的契约测试。
// 只测纯逻辑（宏展开、清单、错误提示），不触碰 COM：run 的成功路径需要
// 真机 Office/WPS，放在 office_com_tools_test.go 的同类环境里做。

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestOfficeMacroList(t *testing.T) {
	out, err := (officeMacroTool{}).Execute(context.Background(), json.RawMessage(`{"action":"list"}`))
	if err != nil {
		t.Fatalf("list 失败：%v", err)
	}
	for _, want := range []string{"word_replace_all", "sheet_replace_all", "参数"} {
		if !strings.Contains(out, want) {
			t.Fatalf("list 输出缺少 %q：\n%s", want, out)
		}
	}
	if !strings.Contains(out, "不支持任意脚本") {
		t.Error("list 应显式声明「非任意脚本」边界")
	}
}

func TestOfficeMacroRunUnknownMacro(t *testing.T) {
	_, err := (officeMacroTool{}).Execute(context.Background(), json.RawMessage(`{"action":"run","macro":"nope"}`))
	if err == nil {
		t.Fatal("未知宏应报错")
	}
	msg := err.Error()
	if !strings.Contains(msg, "未知宏") || !strings.Contains(msg, "word_replace_all") {
		t.Fatalf("报错应点名未知宏并列出可用宏，实际：%v", msg)
	}
}

func TestOfficeMacroRunMissingAction(t *testing.T) {
	if _, err := (officeMacroTool{}).Execute(context.Background(), json.RawMessage(`{}`)); err == nil ||
		!strings.Contains(err.Error(), "缺少参数 action") {
		t.Fatalf("缺 action 应给出精确提示，实际：%v", err)
	}
	if _, err := (officeMacroTool{}).Execute(context.Background(), json.RawMessage(`{"action":"bogus"}`)); err == nil ||
		!strings.Contains(err.Error(), "合法值") {
		t.Fatalf("非法 action 应给出合法值，实际：%v", err)
	}
}

func TestOfficeMacroWordReplaceBuild(t *testing.T) {
	spec := officeMacros["word_replace_all"]
	ops, err := spec.Build(map[string]any{"find": "甲方", "replace": "乙方"})
	if err != nil {
		t.Fatalf("展开失败：%v", err)
	}
	if len(ops) != 1 || ops[0]["op"] != "replace_text" || ops[0]["find"] != "甲方" || ops[0]["replace"] != "乙方" {
		t.Fatalf("展开结果不符合 replace_text 契约：%#v", ops)
	}
	// 删除式替换：replace 传空串必须允许（与「没传」区分开）。
	ops, err = spec.Build(map[string]any{"find": "作废", "replace": ""})
	if err != nil {
		t.Fatalf("空串替换应允许：%v", err)
	}
	if ops[0]["replace"] != "" {
		t.Fatalf("空串替换被吞：%#v", ops[0])
	}
}

func TestOfficeMacroWordReplaceNeedsFind(t *testing.T) {
	_, err := officeMacros["word_replace_all"].Build(map[string]any{"replace": "x"})
	if err == nil || !strings.Contains(err.Error(), "find") {
		t.Fatalf("缺 find 应报字段名，实际：%v", err)
	}
	_, err = officeMacros["word_replace_all"].Build(map[string]any{"find": "a"})
	if err == nil || !strings.Contains(err.Error(), "replace") {
		t.Fatalf("缺 replace 应报字段名，实际：%v", err)
	}
	// 非字符串 replace 要明确拒绝，不静默转字符串。
	_, err = officeMacros["word_replace_all"].Build(map[string]any{"find": "a", "replace": 1})
	if err == nil || !strings.Contains(err.Error(), "必须是字符串") {
		t.Fatalf("类型错误应拒绝，实际：%v", err)
	}
}

func TestOfficeMacroSheetReplaceNeedsRange(t *testing.T) {
	_, err := officeMacros["sheet_replace_all"].Build(map[string]any{"find": "a", "replace": "b"})
	if err == nil || !strings.Contains(err.Error(), "range") {
		t.Fatalf("缺 range 应报错（防误伤整表），实际：%v", err)
	}
	ops, err := officeMacros["sheet_replace_all"].Build(map[string]any{"range": "A1:B9", "find": "旧", "replace": "新"})
	if err != nil {
		t.Fatalf("展开失败：%v", err)
	}
	if ops[0]["op"] != "replace" || ops[0]["range"] != "A1:B9" {
		t.Fatalf("展开结果不符合 replace 契约：%#v", ops)
	}
}

func TestOfficeMacroAppGuard(t *testing.T) {
	// word 宏传给 excel 时必须拒绝，避免在错误的文档上执行。
	_, err := (officeMacroTool{}).Execute(context.Background(),
		json.RawMessage(`{"action":"run","macro":"word_replace_all","app":"excel","args":{"find":"a","replace":"b"}}`))
	if err == nil || !strings.Contains(err.Error(), "只适用于 word") {
		t.Fatalf("应用不匹配应拒绝，实际：%v", err)
	}
}

func TestOfficeMacroSchemaValid(t *testing.T) {
	if !json.Valid((officeMacroTool{}).Schema()) {
		t.Fatal("schema 非 JSON")
	}
	if (officeMacroTool{}).Name() != "office_macro" {
		t.Fatal("工具名不符合契约")
	}
	if (officeMacroTool{}).ReadOnly() {
		t.Fatal("office_macro 会写文档，ReadOnly 必须为 false")
	}
}
