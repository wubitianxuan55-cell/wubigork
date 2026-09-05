package builtin

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// U1 ②：office 域工具错误统一 `Error [CODE]: message` 形态，模型按 code 路由恢复。

func TestCodedErrorFormat(t *testing.T) {
	err := codedError("FORMAT_TEST", "bad thing: %d", 42)
	if err.Error() != "Error [FORMAT_TEST]: bad thing: 42" {
		t.Fatalf("codedError = %q", err.Error())
	}
	if !strings.HasPrefix(err.Error(), "Error [FORMAT_TEST]: ") {
		t.Fatal("missing stable prefix")
	}
}

func TestFormatConvertErrorCodes(t *testing.T) {
	fc := formatConvert{}

	// 空 path → INVALID_ARGS
	_, err := fc.Execute(context.Background(), json.RawMessage(`{}`))
	if err == nil || !strings.HasPrefix(err.Error(), "Error [FORMAT_INVALID_ARGS]: ") {
		t.Fatalf("empty path error = %v, want Error [FORMAT_INVALID_ARGS] prefix", err)
	}

	// 坏 JSON → INVALID_ARGS
	_, err = fc.Execute(context.Background(), json.RawMessage(`{bad`))
	if err == nil || !strings.HasPrefix(err.Error(), "Error [FORMAT_INVALID_ARGS]: ") {
		t.Fatalf("bad json error = %v, want Error [FORMAT_INVALID_ARGS] prefix", err)
	}

	// 不支持的扩展名 → UNSUPPORTED（docmd 的中文消息原样保留在 code 之后）
	_, err = fc.Execute(context.Background(), json.RawMessage(`{"path":"a/b.txt"}`))
	if err == nil || !strings.HasPrefix(err.Error(), "Error [FORMAT_UNSUPPORTED]: ") {
		t.Fatalf("unsupported ext error = %v, want Error [FORMAT_UNSUPPORTED] prefix", err)
	}

	// 源不存在 → SOURCE_MISSING 或 CONVERT_FAILED（内层库包装方式不定），但必须带稳定 code 前缀
	_, err = fc.Execute(context.Background(), json.RawMessage(`{"path":"definitely/missing/file.docx"}`))
	if err == nil || (!strings.HasPrefix(err.Error(), "Error [FORMAT_SOURCE_MISSING]: ") && !strings.HasPrefix(err.Error(), "Error [FORMAT_CONVERT_FAILED]: ")) {
		t.Fatalf("missing source error = %v, want stable FORMAT_* code prefix", err)
	}
	if err != nil && !strings.HasSuffix(err.Error(), "no such file or directory") && !strings.Contains(err.Error(), "不存在") && !strings.Contains(err.Error(), "cannot") && !strings.Contains(err.Error(), "failed") {
		t.Logf("missing-source message (informational): %v", err)
	}
}
