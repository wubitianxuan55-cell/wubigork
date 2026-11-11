package app

// whirl_proc.go — 自管 WHIRL 引擎（Windows HIP 推理，8126）进程生命周期。
//
// 2026-11-11 对擂实测：同机 8060S 上 WHIRL 0.2.5.1 跑 Qwen3.6-35B-A3B
// （whirlfix2 张量手术版，仅 blk.40 两个 BF16 router 转 F32，权重位级无损）
// MTP 解码 102.57 tok/s，为 llama Vulkan+MTP 53.75 的 1.91x。API 形态与
// llama-server 同构（OpenAI /v1 免鉴权），消费链零改动复用。
//
// 模型根与 llama 同源（C:\AI\models 单副本原则）：llama 8124 与本引擎服务
// 同一份 35B whirlfix2 文件（2026-11-11 原件已删，whirlfix2 为唯一副本），
// 两引擎并存按需拉起。
// thinking 由请求级 enable_thinking=false 关闭（见 internal/ai client_chat
// whirl 分支；llama 侧为服务级 --reasoning off）。

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/config"
)

const (
	whirlPort         = 8126
	whirlStartWait    = 300 * time.Second
	whirlProbeDial    = 1500 * time.Millisecond
	whirlStopWait     = 5 * time.Second
	whirlStopPollStep = 500 * time.Millisecond
	whirlBinRel       = `whirl-server.exe`
	// whirlfix2：whirl-bakeoff 产出的双引擎兼容版（BF16 router→F32 顺序重排，
	// llama b10837 与 WHIRL 0.2.5.1 双验证，decode hash 与 WHIRL 原生验证版全等）。
	whirlModelFile  = "Qwen3.6-35B-A3B-Uncensored-HauhauCS-Aggressive-Q4_K_P-MTP-r2-whirlfix2.gguf"
	whirlMmprojFile = "mmproj-Qwen3.6-35B-A3B-Uncensored-HauhauCS-Aggressive-f16.gguf"
)

var whirlProfiles = []engineProfileOption{
	{
		ID:        "qwen35_whirl",
		Label:     "qwen3.6-35b-a3b-mtp（WHIRL）",
		ModelID:   "qwen35_whirl",
		ModelDir:  "qwen35-uncensored",
		ModelFile: whirlModelFile,
		MMProj:    whirlMmprojFile,
	},
}

// whirlRootPath 返回 WHIRL 安装根目录（whirl-server.exe 所在目录，DLL 同驻）。
// 配置 whirl_path 为空时回退默认装机位。
func (a *App) whirlRootPath() string {
	if a != nil && a.core != nil && a.cfg != nil {
		if p := strings.TrimSpace(a.cfg.WhirlPath); p != "" {
			return filepath.Clean(p)
		}
	}
	return config.DefaultWhirlPath
}

func whirlBinPath(a *App) string {
	return filepath.Join(a.whirlRootPath(), whirlBinRel)
}

// whirlProfileModelPath 档位 GGUF 路径（统一模型根）。
func whirlProfileModelPath(p engineProfileOption) string {
	if p.ModelDir == "" || p.ModelFile == "" {
		return ""
	}
	return filepath.Join(voiceV2ModelsRoot, p.ModelDir, p.ModelFile)
}

// whirlMmprojPath 档位 vision 编码器路径；文件缺失时返回空串（vision 降级可启动）。
func whirlMmprojPath(p engineProfileOption) string {
	if p.ModelDir == "" || p.MMProj == "" {
		return ""
	}
	path := filepath.Join(voiceV2ModelsRoot, p.ModelDir, p.MMProj)
	if voiceV2FileExists(path) {
		return path
	}
	return ""
}

// whirlProfileFromModels 把 whirl-server 的 served id（--alias 稳定别名，
// 未设 alias 时为 GGUF 文件名）映射回档位 id。
func whirlProfileFromModels(served []string) string {
	for _, id := range served {
		norm := strings.ToLower(strings.TrimSpace(id))
		if norm == "" {
			continue
		}
		for _, p := range whirlProfiles {
			base := strings.TrimSuffix(strings.ToLower(p.ModelFile), ".gguf")
			for _, candidate := range []string{p.ID, p.ModelID, base} {
				if candidate != "" && strings.Contains(norm, candidate) {
					return p.ID
				}
			}
		}
	}
	return ""
}

// whirlProbe 探活并返回 served 模型清单。
func whirlProbe() ([]string, bool) {
	client := &http.Client{Timeout: whirlProbeDial}
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/v1/models", whirlPort))
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
	return ids, true
}

func whirlRunning() bool {
	return voiceV2TCPProbe(fmt.Sprintf("127.0.0.1:%d", whirlPort), 800*time.Millisecond)
}

// whirlServeArgs 组装 whirl-server 启动参数（档位 → 命令行契约，测试钉死）。
func whirlServeArgs(p engineProfileOption, port int) []string {
	args := []string{
		whirlProfileModelPath(p),
		"--port", strconv.Itoa(port),
		"--balance", "-np", "1",
		"--alias", p.ID,
	}
	if mmproj := whirlMmprojPath(p); mmproj != "" {
		args = append(args, "--mmproj", mmproj)
	}
	return args
}

// whirlEnsure 幂等拉起指定 WHIRL 档位（照 llamaEnsure 语义：同端口一次一个
// 档位；其它档位在跑时如实返回，不静默换模型）。
func (a *App) whirlEnsure(profileID string) map[string]interface{} {
	a.whirlEnsureMu.Lock()
	defer a.whirlEnsureMu.Unlock()

	if strings.TrimSpace(profileID) == "" {
		return map[string]interface{}{
			"running":  whirlRunning(),
			"started":  false,
			"reason":   "model-required",
			"profiles": engineProfileJSON(whirlProfiles),
			"error":    "WHIRL 必须由功能绑定指定具体模型档位，不能只启动空引擎",
		}
	}
	p, ok := engineProfileByID(whirlProfiles, profileID)
	if !ok {
		return map[string]interface{}{
			"running":  whirlRunning(),
			"started":  false,
			"reason":   "unknown-model",
			"model":    profileID,
			"profiles": engineProfileJSON(whirlProfiles),
		}
	}
	if served, running := whirlProbe(); running {
		if a.engineMgr != nil {
			a.engineMgr.TouchEngine("whirl")
		}
		active := whirlProfileFromModels(served)
		res := map[string]interface{}{
			"running":           true,
			"started":           false,
			"reason":            "already-running",
			"profile":           active,
			"requested_profile": p.ID,
			"profiles":          engineProfileJSON(whirlProfiles),
			"served_models":     served,
		}
		if active != "" {
			if activeProfile, ok := engineProfileByID(whirlProfiles, active); ok {
				res["model_path"] = whirlProfileModelPath(activeProfile)
			}
		}
		if active != "" && active != p.ID {
			res["error"] = fmt.Sprintf("WHIRL 正在运行其它档位 %s", active)
		}
		return res
	}
	if whirlRunning() {
		return map[string]interface{}{
			"running":           true,
			"started":           false,
			"reason":            "starting",
			"profile":           "",
			"requested_profile": p.ID,
			"profiles":          engineProfileJSON(whirlProfiles),
			"served_models":     []string{},
		}
	}
	model := whirlProfileModelPath(p)
	if _, err := os.Stat(model); err != nil {
		return map[string]interface{}{
			"running":    false,
			"started":    false,
			"reason":     "model-not-found",
			"profile":    p.ID,
			"model_path": model,
			"model":      p.ModelID,
			"profiles":   engineProfileJSON(whirlProfiles),
			"error":      fmt.Sprintf("WHIRL 档位 %s 的 GGUF 缺失（%s）", p.ID, model),
		}
	}
	bin := whirlBinPath(a)
	if _, err := os.Stat(bin); err != nil {
		return map[string]interface{}{
			"running":    false,
			"started":    false,
			"reason":     "bin-not-found",
			"profile":    p.ID,
			"model_path": model,
			"bin_path":   bin,
			"profiles":   engineProfileJSON(whirlProfiles),
			"error":      fmt.Sprintf("whirl-server.exe 缺失（%s），请确认 WHIRL 安装目录", bin),
		}
	}
	args := whirlServeArgs(p, whirlPort)
	cmd, err := detachedCmd(bin, args, a.whirlRootPath())
	if err != nil {
		return map[string]interface{}{
			"running":    false,
			"started":    false,
			"reason":     "spawn-failed",
			"profile":    p.ID,
			"model_path": model,
			"profiles":   engineProfileJSON(whirlProfiles),
			"error":      fmt.Sprintf("%s 拉起失败: %v", p.ID, err),
		}
	}
	if err := voiceV2Spawn("whirl-"+p.ID, cmd); err != nil {
		return map[string]interface{}{
			"running":    false,
			"started":    false,
			"reason":     "spawn-failed",
			"profile":    p.ID,
			"model_path": model,
			"profiles":   engineProfileJSON(whirlProfiles),
			"error":      fmt.Sprintf("%s 拉起失败: %v", p.ID, err),
		}
	}
	if cmd.Process != nil {
		a.whirlProc.setWithJob(cmd, nil)
		go func() {
			_ = cmd.Wait()
			a.whirlProc.clearIfCurrent(cmd)
		}()
	}
	startedAt := time.Now()
	served, ready := waitWhirlModelsReady(whirlStartWait)
	if !ready {
		a.WhirlEngineStop()
		return map[string]interface{}{
			"running":    whirlRunning(),
			"started":    false,
			"reason":     "timeout",
			"profile":    p.ID,
			"model_path": model,
			"profiles":   engineProfileJSON(whirlProfiles),
			"error":      fmt.Sprintf("%s 300s 未就绪（%s）", p.ID, voiceV2LogPath(whirlBinRel+".log")),
		}
	}
	if a.engineMgr != nil {
		a.engineMgr.TouchEngine("whirl")
	}
	active := whirlProfileFromModels(served)
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
		"profiles":      engineProfileJSON(whirlProfiles),
	}
}

// WhirlEngineStart 由功能绑定/模型中心显式启动档位。
func (a *App) WhirlEngineStart(profile string) map[string]interface{} {
	return a.whirlEnsure(profile)
}

// WhirlEngineStop 登记进程树 → 8126 监听 PID 兜底，外部启动的实例也能停。
func (a *App) WhirlEngineStop() map[string]interface{} {
	res := stopManagedEngine(&a.whirlProc, whirlPort, whirlStopWait, whirlStopPollStep, func() bool {
		_, ok := whirlProbe()
		return ok
	})
	if stopped, _ := res["stopped"].(bool); stopped {
		a.markManagedCatalogStopped("whirl", whirlProfiles)
	}
	return res
}

// WhirlEngineStatus 探活 + 档位目录 + 关键路径诊断（gguf/bin 是否在位）。
func (a *App) WhirlEngineStatus() map[string]interface{} {
	served, ready := whirlProbe()
	running := whirlRunning()
	profile := whirlProfileFromModels(served)
	a.syncManagedCatalog("whirl", whirlProfiles, served, profile)
	if a.core != nil && a.engineMgr != nil {
		errMsg := ""
		if !ready {
			errMsg = "引擎未运行"
		}
		a.engineMgr.SetStatus("whirl", managedEngineStatus("whirl", ready, len(served), 0, errMsg))
	}
	if profile == "" {
		profile = "qwen35_whirl"
	}
	p, _ := engineProfileByID(whirlProfiles, profile)
	model := whirlProfileModelPath(p)
	_, modelOK := os.Stat(model)
	bin := whirlBinPath(a)
	_, binOK := os.Stat(bin)
	return map[string]interface{}{
		"running":       running,
		"ready":         ready,
		"port":          whirlPort,
		"profile":       profile,
		"profiles":      engineProfileJSON(whirlProfiles),
		"served_models": served,
		"modelPath":     model,
		"modelReady":    modelOK == nil,
		"binPath":       bin,
		"binReady":      binOK == nil,
	}
}

func waitWhirlModelsReady(timeout time.Duration) ([]string, bool) {
	deadline := time.Now().Add(timeout)
	for {
		if served, ok := whirlProbe(); ok {
			return served, true
		}
		if time.Now().After(deadline) {
			return nil, false
		}
		time.Sleep(500 * time.Millisecond)
	}
}
