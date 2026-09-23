package app

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/characterlib"
)

// TestBuildConsistencyScorePrompt 一致性评分提示词纯函数：评审维度/JSON 契约/
// 锚点截断/无外观可用。
func TestBuildConsistencyScorePrompt(t *testing.T) {
	p := buildConsistencyScorePrompt(characterlib.Character{Name: "林晚", Appearance: "黑色长发", Figure: "高挑"})
	for _, want := range []string{"面部特征", "发型发色", "体型", "score", "黑色长发", "高挑", "90"} {
		if !strings.Contains(p, want) {
			t.Fatalf("prompt 应含 %q: %s", want, p)
		}
	}
	// 无外观锚点仍可用（只缺文字设定锚点段）
	p2 := buildConsistencyScorePrompt(characterlib.Character{Name: "X"})
	if strings.Contains(p2, "与以下文字设定的一致性") {
		t.Fatalf("无外观不应有文字设定锚点段: %s", p2)
	}
	if !strings.Contains(p2, "score") {
		t.Fatalf("无外观仍应含 JSON 契约: %s", p2)
	}
}

// TestParseConsistencyScoreReply 回复解析：正常/围栏包裹/score 钳位/缺 score/
// issues 清洗。
func TestParseConsistencyScoreReply(t *testing.T) {
	// 正常
	res, err := parseConsistencyScoreReply(`{"score":88,"summary":"基本一致","issues":["发色略深"]}`)
	if err != nil || res.Score != 88 || res.Summary != "基本一致" || len(res.Issues) != 1 {
		t.Fatalf("正常解析: %+v %v", res, err)
	}
	// 围栏包裹
	res, err = parseConsistencyScoreReply("```json\n{\"score\":95,\"summary\":\"高\",\"issues\":[]}\n```")
	if err != nil || res.Score != 95 {
		t.Fatalf("围栏解析: %+v %v", res, err)
	}
	// 钳位：超上界 100、负数 0；四舍五入
	for in, want := range map[string]int{
		`{"score":130}`:  100,
		`{"score":-5}`:   0,
		`{"score":87.6}`: 88,
	} {
		res, err = parseConsistencyScoreReply(in)
		if err != nil || res.Score != want {
			t.Fatalf("钳位 %s: 得 %d %v", in, res.Score, err)
		}
	}
	// 缺 score / 非数值
	if _, err := parseConsistencyScoreReply(`{"summary":"无分"}`); err == nil {
		t.Fatal("缺 score 应报错")
	}
	if _, err := parseConsistencyScoreReply(`{"score":"高"}`); err == nil {
		t.Fatal("score 非数值应报错")
	}
	// 非 JSON
	if _, err := parseConsistencyScoreReply("这张图和设定不一致，我打 60 分"); err == nil {
		t.Fatal("非 JSON 应报错")
	}
	// issues 清洗：非字符串/空串剔除
	res, err = parseConsistencyScoreReply(`{"score":70,"summary":"s","issues":["a","",123,"b"]}`)
	if err != nil || len(res.Issues) != 2 || res.Issues[0] != "a" {
		t.Fatalf("issues 清洗: %+v %v", res, err)
	}
}

// TestCharacterScoreConsistency 绑定路径：seam 注入假视觉模型——全链返回
// 规范 JSON；data URL 走临时文件；边界报错（无名/空图/远端 URL/视觉失败/坏回复）。
func TestCharacterScoreConsistency(t *testing.T) {
	orig := characterScoreVision
	defer func() { characterScoreVision = orig }()

	var gotPrompt string
	var gotImage string
	characterScoreVision = func(_ context.Context, image, prompt string) (string, error) {
		gotImage = image
		gotPrompt = prompt
		return `{"score":82,"summary":"基本一致","issues":["鼻梁略宽"]}`, nil
	}

	a := &App{core: &core{ctx: context.Background()}}
	ch := `{"name":"林晚","appearance":"黑色长发"}`
	out, err := a.CharacterScoreConsistency(ch, "data:image/png;base64,AAAA")
	if err != nil {
		t.Fatalf("CharacterScoreConsistency: %v", err)
	}
	if !strings.Contains(out, `"score":82`) || !strings.Contains(out, "鼻梁略宽") {
		t.Fatalf("应返回规范 JSON: %s", out)
	}
	if !strings.Contains(gotPrompt, "黑色长发") || !strings.Contains(gotPrompt, "score") {
		t.Fatalf("prompt 应含锚点+契约: %s", gotPrompt)
	}
	// seam 收到的是规整前的输入（data URL 原样，转换在 seam 生产实现内部）
	if gotImage != "data:image/png;base64,AAAA" {
		t.Fatalf("data URL 应原样传给视觉 seam: %s", gotImage)
	}
	// 本地路径原样透传
	if _, err := a.CharacterScoreConsistency(ch, "/tmp/ref.png"); err != nil {
		t.Fatalf("本地路径: %v", err)
	}
	if gotImage != "/tmp/ref.png" {
		t.Fatalf("本地路径应透传: %s", gotImage)
	}

	// data URL→临时文件助手：落盘存在、内容一致、cleanup 后删除
	path, cleanup, err := scoreImageToLocalPath("data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte{0x89, 'P', 'N', 'G'}))
	if err != nil {
		t.Fatalf("scoreImageToLocalPath: %v", err)
	}
	if !strings.Contains(path, "gaea-score-") {
		t.Fatalf("应落临时文件: %s", path)
	}
	if b, err := os.ReadFile(path); err != nil || string(b) != "\x89PNG" {
		t.Fatalf("临时文件内容应一致: %v", err)
	}
	cleanup()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("cleanup 后应删除: %v", err)
	}
	// 非法 data URL（缺逗号）报错
	if _, _, err := scoreImageToLocalPath("data:image/png;base64无逗号"); err == nil || !strings.Contains(err.Error(), "缺逗号") {
		t.Fatalf("缺逗号应报错: %v", err)
	}

	// 边界：无名/空图/远端 URL
	if _, err := a.CharacterScoreConsistency(`{"appearance":"x"}`, "data:image/png;base64,AA"); err == nil || !strings.Contains(err.Error(), "名称") {
		t.Fatalf("无名应报错: %v", err)
	}
	if _, err := a.CharacterScoreConsistency(ch, ""); err == nil || !strings.Contains(err.Error(), "缺少待评分图片") {
		t.Fatalf("空图应报错: %v", err)
	}
	if _, err := a.CharacterScoreConsistency(ch, "https://example.com/a.png"); err == nil || !strings.Contains(err.Error(), "远端 URL") {
		t.Fatalf("远端 URL 应报错: %v", err)
	}

	// 视觉失败透传；坏回复如实报错带原始返回
	characterScoreVision = func(_ context.Context, _, _ string) (string, error) {
		return "", errors.New("视觉模型未启用")
	}
	if _, err := a.CharacterScoreConsistency(ch, "data:image/png;base64,AA"); err == nil || !strings.Contains(err.Error(), "视觉模型未启用") {
		t.Fatalf("视觉失败应透传: %v", err)
	}
	characterScoreVision = func(_ context.Context, _, _ string) (string, error) {
		return "看起来差不多", nil
	}
	if _, err := a.CharacterScoreConsistency(ch, "data:image/png;base64,AA"); err == nil || !strings.Contains(err.Error(), "无法解析") {
		t.Fatalf("坏回复应报错: %v", err)
	}
}

// TestAttachComfyProgress 角色库生成进度接线（v4.406）：ComfyUI 后端预置
// queued 并挂回调；其他后端不挂；清理后快照为空。
func TestAttachComfyProgress(t *testing.T) {
	a := &App{core: &core{ctx: context.Background()}, mediaState: &mediaState{}}

	req := &ai.ImageGenerationRequest{}
	a.attachComfyProgress(req, "comfyui")
	if req.ProgressCallback == nil {
		t.Fatal("comfyui 后端应挂进度回调")
	}
	if snap := a.GetComfyUITaskProgress(); snap["status"] != "queued" {
		t.Fatalf("应预置 queued: %v", snap)
	}

	req2 := &ai.ImageGenerationRequest{}
	a.attachComfyProgress(req2, "herdsman")
	if req2.ProgressCallback != nil {
		t.Fatal("非 comfyui 后端不应挂回调")
	}

	a.clearComfyTaskProgress()
	if snap := a.GetComfyUITaskProgress(); snap["status"] != "" {
		t.Fatalf("清理后应为空: %v", snap)
	}
}
