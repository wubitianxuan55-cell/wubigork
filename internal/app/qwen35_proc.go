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
	"strings"
	"time"
)

const (
	qwen35Port         = 8124
	qwen35StartWait    = 300 * time.Second
	qwen35ModelFile    = "Qwen3.6-35B-A3B-Uncensored-HauhauCS-Aggressive-Q4_K_P-MTP-r2.gguf"
	qwen35CtxSize      = 32768 // 语音+聊天共用：语音短轮足够，聊天留增长余量
	qwen35HealthScript = `{"model":"qwen35","stream":false,"max_tokens":1,"messages":[{"role":"user","content":"hi"}]}`
)

// qwen35ModelPath 35B gguf：base 根 models 优先（P3 布局），缺失回退历史路径。
func qwen35ModelPath() string {
	if p := filepath.Join(voiceV2BaseRoot(), "models", qwen35ModelFile); voiceV2FileExists(p) {
		return p
	}
	return filepath.Join(voiceV2LegacyBaseRoot, "models", qwen35ModelFile)
}

func qwen35Running() bool {
	return voiceV2TCPProbe(fmt.Sprintf("127.0.0.1:%d", qwen35Port), 800*time.Millisecond)
}

// ── managed 引擎目录层（模型中心，参照 strata/tianxuan 模板） ──

var llamaProfiles = []engineProfileOption{
	{ID: "qwen35", Label: "qwen3.6-35b-a3b-mtp（自管 llama）", ModelID: "qwen35"},
}

// llamaProbe 探活 + /v1/models 实际 served id（llama-server 返回 gguf 文件名）。
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
	return ids, len(ids) > 0
}

func llamaProfileFromModels(served []string) string {
	if len(served) == 0 {
		return ""
	}
	for _, p := range llamaProfiles {
		if strings.Contains(served[0], "35B") || strings.Contains(served[0], "Qwen3.6") {
			return p.ID
		}
	}
	return llamaProfiles[0].ID
}

// qwen35Ensure 幂等拉起自管 35B（语音 Ensure 链与聊天前置共用）。
// 返回 {running:bool, started:bool, error:string}。
func (a *App) qwen35Ensure() map[string]interface{} {
	if qwen35Running() {
		return map[string]interface{}{"running": true, "started": false}
	}
	model := qwen35ModelPath()
	if _, err := os.Stat(model); err != nil {
		return map[string]interface{}{"running": false, "error": fmt.Sprintf("35B gguf 缺失（%s）", model)}
	}
	cmd, err := detachedCmd(voiceV2LLMBinPath(), []string{
		"--host", "127.0.0.1", "--port", fmt.Sprintf("%d", qwen35Port),
		"--jinja", "-m", model, "-ngl", "99",
		"--ctx-size", fmt.Sprintf("%d", qwen35CtxSize),
		"--reasoning", "off", "-np", "1", "--cache-reuse", "256",
	}, "")
	if err != nil {
		return map[string]interface{}{"running": false, "error": fmt.Sprintf("35B 拉起失败: %v", err)}
	}
	if err := voiceV2Spawn("qwen35", cmd); err != nil {
		return map[string]interface{}{"running": false, "error": fmt.Sprintf("35B 拉起失败: %v", err)}
	}
	if !voiceV2WaitPort(qwen35Port, qwen35StartWait) {
		return map[string]interface{}{"running": false, "error": "35B 300s 未就绪"}
	}
	return map[string]interface{}{"running": true, "started": true}
}

// Qwen35EngineStart 模型中心/语音设置入口：幂等拉起自管 35B。
func (a *App) Qwen35EngineStart() map[string]interface{} {
	return a.qwen35Ensure()
}

// Qwen35EngineStop 按端口击杀（聊天/语音都不再依赖时手动停）。
func (a *App) Qwen35EngineStop() map[string]interface{} {
	killed := voiceV2KillPort(qwen35Port)
	return map[string]interface{}{"stopped": killed}
}

// Qwen35EngineStatus 探活 + 关键路径诊断（gguf/bin 是否在位）。
func (a *App) Qwen35EngineStatus() map[string]interface{} {
	running := qwen35Running()
	model := qwen35ModelPath()
	_, modelOK := os.Stat(model)
	bin := voiceV2LLMBinPath()
	_, binOK := os.Stat(bin)
	return map[string]interface{}{
		"running":     running,
		"port":        qwen35Port,
		"modelPath":   model,
		"modelReady":  modelOK == nil,
		"binPath":     bin,
		"binReady":    binOK == nil,
		"healthCheck": qwen35HealthScript,
	}
}
