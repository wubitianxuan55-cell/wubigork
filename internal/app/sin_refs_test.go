package app

// 原罪 × 角色参考槽（v4.258）：选择规则 / 能力门控 / 文本锚点补强 /
// 参考图失败自动退回纯文本重试。

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/characterlib"
)

func charWith(id, name, appearance string, refs ...string) *characterlib.Character {
	return &characterlib.Character{ID: id, Name: name, Appearance: appearance, ReferenceImages: refs}
}

func TestSinRefPlanCapabilityMatrix(t *testing.T) {
	cases := []struct {
		backend, model string
		wantOK         bool
		wantMode       string
	}{
		{"comfyui", "krea2", true, "img2img"},
		{"comfyui", "krea2-turbo", true, "img2img"}, // 前缀匹配
		{"comfyui", "z-image-turbo", true, "img2img"},
		{"comfyui", "flux-dev", false, ""}, // 该模型没有图生图工作流：如实跳过
		{"herdsman", "whatever", true, "img2img"},
		{"xai", "grok-image", false, ""},
		{"glm", "glm-image", false, ""},
		{"", "", false, ""},
	}
	for _, c := range cases {
		mode, _, ok, reason := sinRefPlan(c.backend, c.model)
		if ok != c.wantOK || mode != c.wantMode {
			t.Errorf("sinRefPlan(%q,%q) = (%q,%v,%q), want (%q,%v)", c.backend, c.model, mode, ok, reason, c.wantMode, c.wantOK)
		}
		if !ok && reason == "" {
			t.Errorf("sinRefPlan(%q,%q) 不可用时必须给原因（诚实跳过）", c.backend, c.model)
		}
	}
}

func TestSinPickRefCharacters(t *testing.T) {
	lin := charWith("c_lin", "林晚", "短发")
	gu := charWith("c_gu", "顾城", "高个")

	// 点名优先（只锚定点到名的那个）
	got := sinPickRefCharacters([]*characterlib.Character{lin, gu}, "近景：林晚回头，雨打在伞上")
	if len(got) != 1 || got[0].Name != "林晚" {
		t.Fatalf("点名选择 = %+v, want 仅林晚", got)
	}
	// 都点名 → 都锚定
	got = sinPickRefCharacters([]*characterlib.Character{lin, gu}, "林晚与顾城在站台对峙")
	if len(got) != 2 {
		t.Fatalf("双点名 = %d 人, want 2", len(got))
	}
	// 未点名 + 单角色 → 用该角色
	got = sinPickRefCharacters([]*characterlib.Character{lin}, "雨夜站台，女人侧身抓拍")
	if len(got) != 1 || got[0].Name != "林晚" {
		t.Fatalf("单角色兜底 = %+v, want 林晚", got)
	}
	// 未点名 + 多角色 → 不锚定（防参考图互串）
	if got = sinPickRefCharacters([]*characterlib.Character{lin, gu}, "雨夜站台全景"); len(got) != 0 {
		t.Fatalf("多角色未点名应不锚定，got %+v", got)
	}
}

func TestSinAugmentPromptWithCast(t *testing.T) {
	lin := charWith("c_lin", "林晚", "短发，右眉骨有一道旧疤")
	out, added := sinAugmentPromptWithCast("雨夜站台，冷蓝主色", []*characterlib.Character{lin})
	if !strings.Contains(out, "人物锚点（林晚）") || !strings.Contains(out, "短发，右眉骨有一道旧疤") {
		t.Fatalf("应追加外观锚点: %q", out)
	}
	if len(added) != 1 || added[0] != "林晚" {
		t.Fatalf("added = %v", added)
	}
	// 提示词已含该外观 → 不重复追加
	dup, added2 := sinAugmentPromptWithCast("近景：" + lin.Appearance, []*characterlib.Character{lin})
	if strings.Contains(dup, "人物锚点") || len(added2) != 0 {
		t.Fatalf("已含外观不应重复追加: %q", dup)
	}
	// 无外观设定 → 不追加、不留空壳
	bare := charWith("c_x", "无名", "")
	out3, added3 := sinAugmentPromptWithCast("空镜", []*characterlib.Character{bare})
	if out3 != "空镜" || len(added3) != 0 {
		t.Fatalf("无外观设定时提示词应原样: %q", out3)
	}
}

func TestSinRefDataURL(t *testing.T) {
	dir := t.TempDir()
	png := filepath.Join(dir, "ref.png")
	if err := os.WriteFile(png, []byte{0x89, 'P', 'N', 'G'}, 0o644); err != nil {
		t.Fatal(err)
	}
	got, ok := sinRefDataURL(png)
	if !ok || !strings.HasPrefix(got, "data:image/png;base64,") {
		t.Fatalf("本地文件应转 data URL: %q ok=%v", got, ok)
	}
	// data URL 原样透传
	if got, ok := sinRefDataURL("data:image/jpeg;base64,AAA"); !ok || got != "data:image/jpeg;base64,AAA" {
		t.Fatalf("data URL 应原样: %q ok=%v", got, ok)
	}
	// 远端 URL / 缺失文件 → 跳过（不扩展网络面、不报错）
	if _, ok := sinRefDataURL("https://example.com/a.png"); ok {
		t.Error("远端 URL 应跳过")
	}
	if _, ok := sinRefDataURL(filepath.Join(dir, "missing.png")); ok {
		t.Error("缺失文件应跳过")
	}
}

// sinRefBackend 记录每次请求；failWithRefs=true 时对带参考图的请求失败，
// 用来验证「参考图失败 → 退回纯文本重试」。
type sinRefBackend struct {
	requests     []ai.ImageGenerationRequest
	failWithRefs bool
}

func (f *sinRefBackend) GenerateImage(_ context.Context, req *ai.ImageGenerationRequest) (*ai.ImageGenerationResponse, error) {
	f.requests = append(f.requests, *req)
	if f.failWithRefs && len(req.RefImages) > 0 {
		return nil, errFakeRef("参考图不可用（fake）")
	}
	return &ai.ImageGenerationResponse{
		Data: []ai.ImageData{{B64JSON: "data:image/png;base64,AAAA"}},
	}, nil
}

type errFakeRef string

func (e errFakeRef) Error() string { return string(e) }

func newSinRefTestApp(t *testing.T, backendKind, model string, fake *sinRefBackend) *App {
	t.Helper()
	a, lib := newSinCastTestApp(t)
	seedCharacter(t, lib, "c_lin", "林晚") // Appearance: 短发，右眉骨有一道旧疤
	// 图片生成挂在 mediaState 上（与真机同构）：core 复用同一个 cfg/client
	a.mediaState = &mediaState{core: a.core, app: a}
	a.mediaState.cfg.ImageBackend = backendKind
	a.mediaState.cfg.ImageModel = model
	if fake != nil {
		a.client.SetImageBackend(fake, backendKind)
	}
	return a
}

// TestSinIllustratePassesCharacterRef 参考槽命中：角色有参考图 + 后端支持 →
// 请求里带 mode=img2img 与参考图，返回值如实标注使用了谁。
func TestSinIllustratePassesCharacterRef(t *testing.T) {
	dir := t.TempDir()
	ref := filepath.Join(dir, "lin.png")
	if err := os.WriteFile(ref, []byte{0x89, 'P', 'N', 'G'}, 0o644); err != nil {
		t.Fatal(err)
	}
	fake := &sinRefBackend{}
	a := newSinRefTestApp(t, "comfyui", "krea2", fake)
	c, err := a.charLib.Get("c_lin")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	c.ReferenceImages = []string{ref}
	if err := a.charLib.Upsert(c); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	story, err := a.SinTopicCreate("雨夜")
	if err != nil {
		t.Fatalf("SinTopicCreate: %v", err)
	}
	if _, err := a.SinCastSet(story.ID, []string{"c_lin"}); err != nil {
		t.Fatalf("SinCastSet: %v", err)
	}

	res, err := a.SinIllustrate(story.ID, 0, "0", "雨夜站台，林晚侧身抓拍", "")
	if err != nil {
		t.Fatalf("SinIllustrate: %v", err)
	}
	if len(fake.requests) != 1 {
		t.Fatalf("应只调用一次生成: %d", len(fake.requests))
	}
	req := fake.requests[0]
	if req.Mode != "img2img" || len(req.RefImages) != 1 || req.RefMethod != "img2img" {
		t.Fatalf("参考槽未透传: mode=%q refs=%d method=%q", req.Mode, len(req.RefImages), req.RefMethod)
	}
	if !strings.Contains(req.Prompt, "人物锚点（林晚）") {
		t.Errorf("提示词应带外观锚点: %q", req.Prompt)
	}
	if used, _ := res["ref_used"].(bool); !used {
		t.Errorf("ref_used 应为 true: %+v", res)
	}
	if names, _ := res["ref_characters"].([]string); len(names) != 1 || names[0] != "林晚" {
		t.Errorf("ref_characters = %v", res["ref_characters"])
	}
}

// TestSinIllustrateSkipsRefOnUnsupportedBackend 后端不支持 → 不硬塞参考图
// （硬塞会整单报错），但文本锚点照旧，并如实给出原因。
func TestSinIllustrateSkipsRefOnUnsupportedBackend(t *testing.T) {
	fake := &sinRefBackend{}
	a := newSinRefTestApp(t, "xai", "grok-image", fake)
	story, err := a.SinTopicCreate("雨夜")
	if err != nil {
		t.Fatalf("SinTopicCreate: %v", err)
	}
	if _, err := a.SinCastSet(story.ID, []string{"c_lin"}); err != nil {
		t.Fatalf("SinCastSet: %v", err)
	}
	res, err := a.SinIllustrate(story.ID, 0, "0", "林晚在雨里", "")
	if err != nil {
		t.Fatalf("SinIllustrate: %v", err)
	}
	if len(fake.requests) != 1 || len(fake.requests[0].RefImages) != 0 || fake.requests[0].Mode != "" {
		t.Fatalf("不支持的后端不应带参考图: %+v", fake.requests)
	}
	if used, _ := res["ref_used"].(bool); used {
		t.Errorf("ref_used 应为 false: %+v", res)
	}
	if reason, _ := res["ref_reason"].(string); !strings.Contains(reason, "不支持参考图") {
		t.Errorf("应给出跳过原因: %q", res["ref_reason"])
	}
	if !strings.Contains(fake.requests[0].Prompt, "人物锚点") {
		t.Errorf("文本锚点应对所有后端生效: %q", fake.requests[0].Prompt)
	}
}

// TestSinIllustrateFallsBackWhenRefFails 参考图导致失败 → 自动退回纯文本重试一次，
// 插图不因参考图而失败，并如实标记 fallback。
func TestSinIllustrateFallsBackWhenRefFails(t *testing.T) {
	dir := t.TempDir()
	ref := filepath.Join(dir, "lin.png")
	if err := os.WriteFile(ref, []byte{0x89, 'P', 'N', 'G'}, 0o644); err != nil {
		t.Fatal(err)
	}
	fake := &sinRefBackend{failWithRefs: true}
	a := newSinRefTestApp(t, "comfyui", "krea2", fake)
	c, _ := a.charLib.Get("c_lin")
	c.ReferenceImages = []string{ref}
	if err := a.charLib.Upsert(c); err != nil {
		t.Fatal(err)
	}
	story, err := a.SinTopicCreate("雨夜")
	if err != nil {
		t.Fatalf("SinTopicCreate: %v", err)
	}
	if _, err := a.SinCastSet(story.ID, []string{"c_lin"}); err != nil {
		t.Fatalf("SinCastSet: %v", err)
	}
	res, err := a.SinIllustrate(story.ID, 0, "0", "林晚在雨里", "")
	if err != nil {
		t.Fatalf("参考图失败不应让插图失败: %v", err)
	}
	if len(fake.requests) != 2 {
		t.Fatalf("应重试一次（共 2 次调用），got %d", len(fake.requests))
	}
	if len(fake.requests[0].RefImages) == 0 {
		t.Error("首次应带参考图")
	}
	if len(fake.requests[1].RefImages) != 0 || fake.requests[1].Mode != "" {
		t.Errorf("重试应退回纯文本: mode=%q refs=%d", fake.requests[1].Mode, len(fake.requests[1].RefImages))
	}
	if fb, _ := res["ref_fallback"].(bool); !fb {
		t.Errorf("应标记 ref_fallback: %+v", res)
	}
	if used, _ := res["ref_used"].(bool); used {
		t.Errorf("回退后 ref_used 应为 false: %+v", res)
	}
	if p, _ := res["path"].(string); p == "" {
		t.Errorf("回退后应有产物路径: %+v", res)
	}
}
