package builtin

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

// cad_ask 守卫面单测：元数据/读目标解析/Execute 参数与输入校验错误路径。
// 渲染+视觉问答的 happy path 由 cad 包单测（解析/渲图）与工具面板端到端覆盖，
// 本处不重复起真模型。
func TestCadAskGuards(t *testing.T) {
	v := cadAsk{workDir: t.TempDir()}
	if v.Name() != "cad_ask" {
		t.Fatalf("name = %q", v.Name())
	}
	if !v.ReadOnly() {
		t.Fatal("cad_ask 必须只读（图纸零写入红线）")
	}
	if !v.ModelBacked() {
		t.Fatal("cad_ask 调视觉模型，必须 ModelBacked（mt_ 运行记录面）")
	}
	// ReadTarget：相对路径落 workDir
	rt, ok := v.ReadTarget(json.RawMessage(`{"path":"drawings/a.dwg"}`))
	if !ok || rt != filepath.Join(v.workDir, "drawings", "a.dwg") {
		t.Fatalf("ReadTarget = %q, %v", rt, ok)
	}
	if _, ok := v.ReadTarget(json.RawMessage(`{"path":"  "}`)); ok {
		t.Fatal("空 path 不应产出读目标")
	}
	ctx := context.Background()
	// 空 path
	if _, err := v.Execute(ctx, json.RawMessage(`{"question":"x"}`)); err == nil || !strings.Contains(err.Error(), "path") {
		t.Fatalf("空 path 应报错: %v", err)
	}
	// 空 question
	if _, err := v.Execute(ctx, json.RawMessage(`{"path":"a.dwg"}`)); err == nil || !strings.Contains(err.Error(), "question") {
		t.Fatalf("空 question 应报错: %v", err)
	}
	// 非图纸扩展名
	if _, err := v.Execute(ctx, json.RawMessage(`{"path":"a.txt","question":"x"}`)); err == nil || !strings.Contains(err.Error(), "dwg") {
		t.Fatalf("非图纸扩展名应报错: %v", err)
	}
	// 文件不存在
	if _, err := v.Execute(ctx, json.RawMessage(`{"path":"nope.dwg","question":"x"}`)); err == nil {
		t.Fatal("不存在的图纸应报错")
	}
	// compact 两表与分类表必须在册（CompactDescriptor 无条件替换模型面描述；
	// spaceTags 分类面 fail-closed——漏表守卫 TestSpaceTagsCoversAllBuiltins 同源）
	if compactDesc["cad_ask"] == "" {
		t.Fatal("compactDesc 缺 cad_ask")
	}
	if compactSchema["cad_ask"] == nil {
		t.Fatal("compactSchema 缺 cad_ask")
	}
	if _, ok := spaceTags["cad_ask"]; !ok {
		t.Fatal("spaceTags 缺 cad_ask")
	}
}
