package app

// 章节配图 v2（阶段一刀 D 核心片，规格 进度计划/gaea-scene-illustration-v2-20260917.md）
// 测试：风格槽注入/参考附着与 refNote 三态路由/坏 opts 拒绝/空 opts 零变化/
// PortraitURL 本地路径转 dataURL。fake 后端捕获请求形状（fingerprint fixture
// +chapter.Agent 直配）。

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/chapter"
	"github.com/gaea/gaea/internal/types"
)

// newIllustrationApp 配图测试 App：真项目 + fake 图片后端捕获请求 + chapterAgent。
func newIllustrationApp(t *testing.T, backend string) (*App, *fakeImageBackend) {
	t.Helper()
	a := newFingerprintTestApp(t)
	fake := &fakeImageBackend{result: &ai.ImageGenerationResponse{
		Data: []ai.ImageData{{B64JSON: pngDataURLApp("fake-scene"), Kind: "image"}},
	}}
	c := &ai.Client{}
	c.SetImageBackend(fake, backend)
	a.client = c
	a.cfg.ImageBackend = backend
	a.ctx = context.Background() // agent 持 a.ctx 直透后端（nil 会在 fake 的 ctx.Err() 解引用）
	a.chapterAgent = chapter.New(c, a.getPM(), a.cfg, nil)
	mustWriteChapter(t, a, 1, "配图样本正文。")
	return a, fake
}

func mustWriteCharacterWithPortrait(t *testing.T, a *App, id, name, portrait string) {
	t.Helper()
	cf, err := a.getPM().ReadCharacters()
	if err != nil || cf == nil {
		cf = &types.CharacterFile{}
	}
	cf.Characters = append(cf.Characters, types.Character{ID: id, Name: name, PortraitURL: portrait, Status: "Alive"})
	if err := a.getPM().WriteCharacters(cf); err != nil {
		t.Fatalf("写角色: %v", err)
	}
}

func TestSceneIllustrationV2_StyleSlot(t *testing.T) {
	a, fake := newIllustrationApp(t, "comfyui")
	res, err := a.GenerateSceneIllustration(1, `{"style":"水墨国风"}`)
	if err != nil {
		t.Fatalf("生成: %v", err)
	}
	if msg, _ := res["error"].(string); msg != "" {
		t.Fatalf("返回错误: %s", msg)
	}
	if !strings.Contains(fake.lastReq.Prompt, "水墨国风") || strings.Contains(fake.lastReq.Prompt, "数字油画") {
		t.Fatalf("风格槽应替换默认风格: %s", fake.lastReq.Prompt)
	}
	if !strings.Contains(fake.lastReq.Prompt, "16:9构图") {
		t.Fatalf("构图固定保留: %s", fake.lastReq.Prompt)
	}
	if res["refNote"] != "" {
		t.Fatalf("未选角色不应有 refNote: %v", res["refNote"])
	}
}

func TestSceneIllustrationV2_EmptyOptsZeroChange(t *testing.T) {
	a, fake := newIllustrationApp(t, "comfyui")
	if _, err := a.GenerateSceneIllustration(1, ""); err != nil {
		t.Fatalf("空 opts 应零变化通过: %v", err)
	}
	if !strings.Contains(fake.lastReq.Prompt, "数字油画") || fake.lastReq.Mode != "" || len(fake.lastReq.RefImages) != 0 {
		t.Fatalf("空 opts 应旧行为（默认风格+纯文生图）: %+v", fake.lastReq)
	}
}

func TestSceneIllustrationV2_BadOptsRejected(t *testing.T) {
	a, _ := newIllustrationApp(t, "comfyui")
	if _, err := a.GenerateSceneIllustration(1, "not-json"); err == nil || !strings.Contains(err.Error(), "配图选项格式不正确") {
		t.Fatalf("坏 opts 应拒绝，得到: %v", err)
	}
}

func TestSceneIllustrationV2_RefRouting(t *testing.T) {
	t.Run("comfyui 附参考 img2img", func(t *testing.T) {
		a, fake := newIllustrationApp(t, "comfyui")
		mustWriteCharacterWithPortrait(t, a, "c1", "林昭", "data:image/png;base64,AAAA")
		res, err := a.GenerateSceneIllustration(1, `{"characterIds":["c1"]}`)
		if err != nil {
			t.Fatalf("生成: %v", err)
		}
		if fake.lastReq.Mode != "img2img" || len(fake.lastReq.RefImages) != 1 || fake.lastReq.Denoise != chapter.SceneRefDenoise {
			t.Fatalf("参考应附着 img2img: %+v", fake.lastReq)
		}
		if n, _ := res["refNote"].(string); !strings.Contains(n, "已附 1 张") {
			t.Fatalf("refNote 不对: %v", res["refNote"])
		}
	})

	t.Run("非参考后端诚实降级提示", func(t *testing.T) {
		a, fake := newIllustrationApp(t, "glm")
		mustWriteCharacterWithPortrait(t, a, "c1", "林昭", "data:image/png;base64,AAAA")
		res, err := a.GenerateSceneIllustration(1, `{"characterIds":["c1"]}`)
		if err != nil {
			t.Fatalf("生成: %v", err)
		}
		if len(fake.lastReq.RefImages) != 0 || fake.lastReq.Mode != "" {
			t.Fatalf("glm 不应附参考: %+v", fake.lastReq)
		}
		if n, _ := res["refNote"].(string); !strings.Contains(n, "不支持参考图") {
			t.Fatalf("refNote 应提示降级: %v", res["refNote"])
		}
	})

	t.Run("无立绘角色诚实跳过", func(t *testing.T) {
		a, _ := newIllustrationApp(t, "comfyui")
		mustWriteCharacterWithPortrait(t, a, "c1", "林昭", "")
		res, err := a.GenerateSceneIllustration(1, `{"characterIds":["c1","ghost"]}`)
		if err != nil {
			t.Fatalf("生成: %v", err)
		}
		if n, _ := res["refNote"].(string); !strings.Contains(n, "无立绘") || !strings.Contains(n, "不存在") {
			t.Fatalf("refNote 应说明跳过原因: %v", res["refNote"])
		}
	})
}

func TestSceneIllustrationV2_PortraitPathToDataURL(t *testing.T) {
	// PortraitURL 为本地路径 → 读盘转 data URL（参考槽只认 data URL）
	a, fake := newIllustrationApp(t, "comfyui")
	png := []byte{0x89, 0x50, 0x4e, 0x47}
	p := filepath.Join(t.TempDir(), "portrait.png")
	if err := os.WriteFile(p, png, 0o644); err != nil {
		t.Fatalf("写立绘: %v", err)
	}
	mustWriteCharacterWithPortrait(t, a, "c1", "林昭", p)
	if _, err := a.GenerateSceneIllustration(1, `{"characterIds":["c1"]}`); err != nil {
		t.Fatalf("生成: %v", err)
	}
	if len(fake.lastReq.RefImages) != 1 || !strings.HasPrefix(fake.lastReq.RefImages[0], "data:image/png;base64,") {
		t.Fatalf("路径立绘应转 data URL: %+v", fake.lastReq.RefImages)
	}
}
