// qwen35_proc.go — 自管 35B llama 引擎（Qwen3.6-35B MTP-r2，8124）进程生命周期。
//
// P2 续拍板（2026-11-05）：35B 聊天模型不走 herdsman——实测 herdsman 全链
// TTFT 3.32s/total 4.61s，自管 llama（b11980+cache-reuse+ctx32K）0.08s/1.22s。
// 语音（voice_chat_llm=qwen35，网关直代）与聊天板块（模型中心自定义引擎指向
// 8124）共用本实例；与 herdsman 实例可共存（双 35B 驻留实测通过）。
package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	qwen35Port            = 8124
	qwen35StartWait       = 300 * time.Second
	qwen35ModelFile       = "Qwen3.6-35B-A3B-Uncensored-HauhauCS-Aggressive-Q4_K_P-MTP-r2.gguf"
	qwen35CtxSize         = 32768 // 语音+聊天共用：语音短轮足够，聊天留增长余量
	qwen35HealthScript    = `{"model":"qwen35","stream":false,"max_tokens":1,"messages":[{"role":"user","content":"hi"}]}`
	gemma4_26BModelDir    = "gemma-4-26b-a4b-it-qat"
	gemma4_26BModelFile   = "gemma-4-26B-A4B-it-qat-UD-Q4_K_XL.gguf"
	gemma4_26BMTPFile     = "mtp-gemma-4-26B-A4B-it.gguf"
	gemma4_26BCtxSize     = 32768
	llamaDefaultProfileID = "qwen35"
)

// qwen35ModelPath 35B gguf：统一模型根 C:\AI\models 优先（单副本原则），缺失
// 回退 base/models（历史布局兼容）。不复制模型——多处引用同一文件。
func qwen35ModelPath() string {
	if p := filepath.Join(voiceV2ModelsRoot, "qwen35-uncensored", qwen35ModelFile); voiceV2FileExists(p) {
		return p
	}
	return filepath.Join(voiceV2LegacyBaseRoot, "models", qwen35ModelFile)
}

func qwen35Running() bool {
	return voiceV2TCPProbe(fmt.Sprintf("127.0.0.1:%d", qwen35Port), 800*time.Millisecond)
}

// ── managed 引擎目录层（模型中心，参照 strata/tianxuan 模板） ──

var llamaProfiles = []engineProfileOption{
	{
		ID:            "qwen35",
		Label:         "qwen3.6-35b-a3b-mtp（自管 llama）",
		ModelID:       "qwen35",
		ModelDir:      "qwen35-uncensored",
		ModelFile:     qwen35ModelFile,
		SpecType:      "draft-mtp",
		SpecDraftNMax: 3,
		CtxSize:       qwen35CtxSize,
	},
	{
		ID:            "gemma4_26b_a4b",
		Label:         "gemma-4-26B-A4B-it QAT Q4_K_XL + MTP（自管 llama）",
		ModelID:       "gemma4_26b_a4b",
		ModelDir:      gemma4_26BModelDir,
		ModelFile:     gemma4_26BModelFile,
		DraftModel:    gemma4_26BMTPFile,
		SpecType:      "draft-mtp",
		SpecDraftNMax: 2,
		CtxSize:       gemma4_26BCtxSize,
	},
}

// llamaProfileModelPath 返回 llama 档位的 GGUF 路径。Qwen35 保留统一模型根
// 优先、历史 base 根回退；新档位只使用统一模型根，避免再次形成多副本。
func llamaProfileModelPath(p engineProfileOption) string {
	if p.ID == "qwen35" {
		return qwen35ModelPath()
	}
	if p.ModelDir == "" || p.ModelFile == "" {
		return ""
	}
	return filepath.Join(voiceV2ModelsRoot, p.ModelDir, p.ModelFile)
}

// llamaProfileDraftModelPath 返回推测解码 draft/MTP 文件路径。
func llamaProfileDraftModelPath(p engineProfileOption) string {
	if p.ModelDir == "" || p.DraftModel == "" {
		return ""
	}
	return filepath.Join(voiceV2ModelsRoot, p.ModelDir, p.DraftModel)
}

// llamaProfileFromModels 把 llama-server 的 served id（稳定别名或 GGUF
// 绝对路径）映射回档位 id。兼容旧实例未设置 --alias 时返回绝对路径。
func llamaProfileFromModels(served []string) string {
	for _, id := range served {
		norm := strings.ToLower(filepath.ToSlash(strings.TrimSpace(id)))
		if norm == "" {
			continue
		}
		for _, p := range llamaProfiles {
			for _, candidate := range []string{p.ID, p.ModelID, filepath.Base(p.ModelFile)} {
				candidate = strings.ToLower(strings.TrimSpace(candidate))
				if candidate != "" && strings.Contains(norm, candidate) {
					return p.ID
				}
			}
		}
	}
	return ""
}

// llamaProbe 探活并返回稳定的模型 id（不是可能漂移的 GGUF 绝对路径）。
func llamaProbe() ([]string, bool) {
	client := &http.Client{Timeout: 1500 * time.Millisecond}
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/v1/models", qwen35Port))
	if err != nil {
		return nil, false
	}
	defer func() { _ = resp.Body.Close() }()
	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if json.NewDecoder(resp.Body).Decode(&payload) != nil {
		return nil, false
	}
	ids := make([]string, 0, len(payload.Data))
	for _, m := range payload.Data {
		if m.ID != "" {
			ids = append(ids, m.ID)
		}
	}
	if len(ids) == 0 {
		return nil, false
	}
	profileID := llamaProfileFromModels(ids)
	if profileID != "" {
		if p, ok := engineProfileByID(llamaProfiles, profileID); ok {
			return []string{p.ModelID}, true
		}
	}
	return ids, true
}

// llamaEnsure 幂等拉起指定 llama 档位。同一端口一次只驻留一个模型；若其它
// 档位已在运行，如实返回 active profile，不静默换模型。
// 返回 {running:bool, started:bool, error:string}。
func (a *App) llamaEnsure(profileID string) map[string]interface{} {
	a.llamaEnsureMu.Lock()
	defer a.llamaEnsureMu.Unlock()

	if strings.TrimSpace(profileID) == "" {
		return map[string]interface{}{
			"running":  qwen35Running(),
			"started":  false,
			"reason":   "model-required",
			"profiles": engineProfileJSON(llamaProfiles),
			"error":    "llama 必须由功能绑定指定具体模型档位，不能只启动空引擎",
		}
	}
	p, ok := engineProfileByID(llamaProfiles, profileID)
	if !ok {
		return map[string]interface{}{
			"running":  qwen35Running(),
			"started":  false,
			"reason":   "unknown-model",
			"model":    profileID,
			"profiles": engineProfileJSON(llamaProfiles),
		}
	}
	if served, running := llamaProbe(); running {
		if a.engineMgr != nil {
			a.engineMgr.TouchEngine("llama")
		}
		active := llamaProfileFromModels(served)
		res := map[string]interface{}{
			"running":           true,
			"started":           false,
			"reason":            "already-running",
			"profile":           active,
			"requested_profile": p.ID,
			"profiles":          engineProfileJSON(llamaProfiles),
			"served_models":     served,
		}
		if active != "" {
			if activeProfile, ok := engineProfileByID(llamaProfiles, active); ok {
				res["model_path"] = llamaProfileModelPath(activeProfile)
			}
		}
		if active != "" && active != p.ID {
			res["error"] = fmt.Sprintf("llama 正在运行其它档位 %s", active)
		}
		return res
	}
	if qwen35Running() {
		return map[string]interface{}{
			"running":           true,
			"started":           false,
			"reason":            "starting",
			"profile":           "",
			"requested_profile": p.ID,
			"profiles":          engineProfileJSON(llamaProfiles),
			"served_models":     []string{},
		}
	}
	model := llamaProfileModelPath(p)
	if _, err := os.Stat(model); err != nil {
		return map[string]interface{}{
			"running":    false,
			"started":    false,
			"reason":     "model-not-found",
			"profile":    p.ID,
			"model_path": model,
			"model":      p.ModelID,
			"profiles":   engineProfileJSON(llamaProfiles),
			"error":      fmt.Sprintf("llama 档位 %s 的 GGUF 缺失（%s）", p.ID, model),
		}
	}
	draftModel := llamaProfileDraftModelPath(p)
	if draftModel != "" {
		if _, err := os.Stat(draftModel); err != nil {
			return map[string]interface{}{
				"running":    false,
				"started":    false,
				"reason":     "draft-model-not-found",
				"profile":    p.ID,
				"model_path": model,
				"draft_path": draftModel,
				"model":      p.ModelID,
				"profiles":   engineProfileJSON(llamaProfiles),
				"error":      fmt.Sprintf("llama 档位 %s 的 MTP 文件缺失（%s）", p.ID, draftModel),
			}
		}
	}
	ctxSize := p.CtxSize
	if ctxSize <= 0 {
		ctxSize = qwen35CtxSize
	}
	args := []string{
		"--host", "127.0.0.1", "--port", fmt.Sprintf("%d", qwen35Port),
		"--jinja", "-m", model, "-ngl", "99",
		"--ctx-size", strconv.Itoa(ctxSize),
		"--reasoning", "off", "-np", "1", "--cache-reuse", "256",
		"--alias", p.ID,
	}
	if p.ChatTemplate != "" && voiceV2FileExists(p.ChatTemplate) {
		args = append(args, "--chat-template-file", p.ChatTemplate)
	}
	if p.SpecType != "" {
		args = append(args, "--spec-type", p.SpecType)
		if draftModel != "" {
			args = append(args, "-md", draftModel, "--spec-draft-ngl", "99")
		}
		if p.SpecDraftNMax > 0 {
			args = append(args, "--spec-draft-n-max", strconv.Itoa(p.SpecDraftNMax))
		}
	}
	cmd, err := detachedCmd(voiceV2LLMBinPath(), args, "")
	if err != nil {
		return map[string]interface{}{
			"running":    false,
			"started":    false,
			"reason":     "spawn-failed",
			"profile":    p.ID,
			"model_path": model,
			"profiles":   engineProfileJSON(llamaProfiles),
			"error":      fmt.Sprintf("%s 拉起失败: %v", p.ID, err),
		}
	}
	if err := voiceV2Spawn("llama-"+p.ID, cmd); err != nil {
		return map[string]interface{}{
			"running":    false,
			"started":    false,
			"reason":     "spawn-failed",
			"profile":    p.ID,
			"model_path": model,
			"profiles":   engineProfileJSON(llamaProfiles),
			"error":      fmt.Sprintf("%s 拉起失败: %v", p.ID, err),
		}
	}
	if cmd.Process != nil {
		a.llamaProc.setWithJob(cmd, nil)
		go func() {
			_ = cmd.Wait()
			a.llamaProc.clearIfCurrent(cmd)
		}()
	}
	startedAt := time.Now()
	served, ready := waitLlamaModelsReady(qwen35StartWait)
	if !ready {
		a.Qwen35EngineStop()
		return map[string]interface{}{
			"running":    qwen35Running(),
			"started":    false,
			"reason":     "timeout",
			"profile":    p.ID,
			"model_path": model,
			"profiles":   engineProfileJSON(llamaProfiles),
			"error":      fmt.Sprintf("%s 300s 未就绪（%s）", p.ID, voiceV2LogPath("llama-server.exe.log")),
		}
	}
	if a.engineMgr != nil {
		a.engineMgr.TouchEngine("llama")
	}
	active := llamaProfileFromModels(served)
	if active == "" {
		active = p.ID
	}
	return map[string]interface{}{
		"running":       true,
		"started":       true,
		"waited_ms":     time.Since(startedAt).Milliseconds(),
		"profile":       active,
		"model":         p.ModelID,
		"model_path":    model,
		"served_models": served,
		"profiles":      engineProfileJSON(llamaProfiles),
	}
}

// qwen35Ensure 保留现有聊天/语音内部调用语义，默认拉起 Qwen35。
func (a *App) qwen35Ensure() map[string]interface{} {
	return a.llamaEnsure(llamaDefaultProfileID)
}

// Qwen35EngineStart 保留旧 Wails 绑定名，但不再提供默认模型：必须由调用方
// 显式给出功能绑定对应的档位；空 profile 返回 model-required，不拉起进程。
func (a *App) Qwen35EngineStart(profile string) map[string]interface{} {
	return a.llamaEnsure(profile)
}

// Qwen35EngineStop 按端口击杀（聊天/语音都不再依赖时手动停）。
func (a *App) Qwen35EngineStop() map[string]interface{} {
	hadProcess := a.llamaProc.stop()
	killed := voiceV2KillPort(qwen35Port)
	a.markManagedCatalogStopped("llama", llamaProfiles)
	return map[string]interface{}{"stopped": hadProcess || killed}
}

// Qwen35EngineStatus 探活 + 档位目录 + 关键路径诊断（gguf/bin 是否在位）。
func (a *App) Qwen35EngineStatus() map[string]interface{} {
	served, ready := llamaProbe()
	running := qwen35Running()
	profile := llamaProfileFromModels(served)
	if profile == "" && !running {
		profile = llamaDefaultProfileID
	}
	p, _ := engineProfileByID(llamaProfiles, profile)
	model := llamaProfileModelPath(p)
	_, modelOK := os.Stat(model)
	bin := voiceV2LLMBinPath()
	_, binOK := os.Stat(bin)
	return map[string]interface{}{
		"running":         running,
		"ready":           ready,
		"port":            qwen35Port,
		"profile":         profile,
		"default_profile": llamaDefaultProfileID,
		"profiles":        engineProfileJSON(llamaProfiles),
		"served_models":   served,
		"modelPath":       model,
		"modelReady":      modelOK == nil,
		"binPath":         bin,
		"binReady":        binOK == nil,
		"healthCheck":     qwen35HealthScript,
	}
}
