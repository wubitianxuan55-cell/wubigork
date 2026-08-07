package ai

import (
	"encoding/json"
	"testing"
)

// E01：ComfyUI 工作流参数顺序稳定（flaky 回归）——节点连线与关键参数必须固定。
func TestRegressionE01ZImageWorkflowStable(t *testing.T) {
	b := &ComfyUIBackend{}
	build := func(steps int) map[string]interface{} {
		return b.buildZImageWorkflow("提示词", "负面", 1024, 1024, 42, steps, "unet.safetensors", nil)
	}
	wf := build(0)
	node := func(id string) map[string]interface{} { return wf[id].(map[string]interface{}) }
	ks := node("10")["inputs"].(map[string]interface{})
	if ks["seed"] != 42 {
		t.Fatalf("seed = %v", ks["seed"])
	}
	if ks["steps"] != 8 {
		t.Fatalf("steps 默认应 8: %v", ks["steps"])
	}
	if ks["positive"].([]interface{})[0] != "7" || ks["negative"].([]interface{})[0] != "13" {
		t.Fatalf("KSampler 连线 = %v", ks)
	}
	if node("7")["inputs"].(map[string]interface{})["text"] != "提示词" {
		t.Fatalf("prompt 节点 = %+v", node("7"))
	}
	b1, _ := json.Marshal(wf)
	b2, _ := json.Marshal(build(0))
	if string(b1) != string(b2) {
		t.Fatal("workflow 序列化不稳定")
	}
	if got := build(25)["10"].(map[string]interface{})["inputs"].(map[string]interface{})["steps"]; got != 20 {
		t.Fatalf("steps 未钳制到 20: %v", got)
	}
}
