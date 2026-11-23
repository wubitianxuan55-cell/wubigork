package builtin

import (
	"encoding/json"
	"testing"
)

// toJSON 测试助手：map → JSON 参数。原定义在 knowledge_test.go，2026-11-23
// 知识库退役删除该文件后迁到此处（造价工具测试仍在用）。
func toJSON(t *testing.T, m map[string]interface{}) []byte {
	t.Helper()
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	return b
}
