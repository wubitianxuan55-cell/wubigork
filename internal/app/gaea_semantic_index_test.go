package app

import (
	"strings"
	"testing"
)

// TestSemanticIndexStatus_ModelUnavailable 模型不可用时索引状态诚实说人话。
// （成本类索引覆盖与显式补齐的用例随旧成本库退役拆除，2026-11-26。）
func TestSemanticIndexStatus_ModelUnavailable(t *testing.T) {
	a, _, _ := countingEmbedEnv(t)
	SetAppEmbedderForTest(nil) // 引擎未配置 → localSearchEmbedder nil

	st := a.GaeaSemanticIndexStatus()
	if st.Available {
		t.Fatalf("无模型应报 Available=false: %+v", st)
	}
	if !strings.Contains(st.Error, "未配置") {
		t.Fatalf("Error 应说人话: %+v", st)
	}
}
