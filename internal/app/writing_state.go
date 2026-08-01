package app

import (
	"log/slog"
	"path/filepath"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/analysis"
	"github.com/gaea/gaea/internal/chapter"
	"github.com/gaea/gaea/internal/character"
	"github.com/gaea/gaea/internal/outline"
	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/skill"
	"github.com/gaea/gaea/internal/worldview"
)

// ── 写作域：项目管理与子代理初始化（原 app.go 中的方法）──

// getPM 以读锁获取当前项目（调用方用完即释放引用）
func (w *writingState) getPM() *project.Manager {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.pm
}

// setPM 以写锁设置当前项目
func (w *writingState) setPM(pm *project.Manager) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.pm = pm
}

// closePM 以写锁关闭并清空当前项目
func (w *writingState) closePM() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.pm == nil {
		return nil
	}
	err := w.pm.Close()
	w.pm = nil
	return err
}

// initAgents 初始化全部写作子代理
func (w *writingState) initAgents() {
	w.worldviewAgent = worldview.New(w.client, w.pm, w.cfg, w.eng)
	w.characterAgent = character.New(w.client, w.pm, w.cfg, w.eng)
	w.outlineAgent = outline.New(w.client, w.pm, w.cfg, w.eng)
	w.chapterAgent = chapter.New(w.client, w.pm, w.cfg, w.eng)
	w.analysisAgent = analysis.New(w.client, w.pm, w.cfg, w.eng)
	w.skillLoader = skill.NewLoader(filepath.Join(w.cfg.ResourceDir, "skills"))

	// 恢复上次保存的图像后端配置（写作用于媒体域，但启动时统一执行）
	w.restoreImageBackend()
}

// restoreImageBackend 从配置恢复图像后端（应用重启后自动恢复）
func (w *writingState) restoreImageBackend() {
	if w.client == nil {
		return
	}
	switch w.cfg.ImageBackend {
	case "comfyui":
		if w.cfg.ComfyUIURL != "" {
			w.client.SetImageBackend(ai.NewComfyUIBackend(w.cfg.ComfyUIURL), "comfyui")
			slog.Info("已恢复 ComfyUI 图像后端", "url", w.cfg.ComfyUIURL, "model", w.cfg.ImageModel)
		}
		if w.engineMgr != nil {
			if eng, ok := w.engineMgr.GetEngine("herdsman"); ok && eng.Enabled {
				w.client.SetImageBackend(ai.NewOpenAIImageBackend(eng.BaseURL, eng.APIKey), "herdsman")
				slog.Info("已恢复 Herdsman 图像后端", "url", eng.BaseURL)
			}
		}
	case "ollama":
		if w.engineMgr != nil {
			if eng, ok := w.engineMgr.GetEngine("ollama"); ok && eng.Enabled {
				w.client.SetImageBackend(ai.NewOpenAIImageBackend(eng.BaseURL, eng.APIKey), "ollama")
				slog.Info("已恢复 Ollama 图像后端")
			}
		}
		// xai 不需要恢复（默认就是 xai fallback）
	}
}
