// H0-1 Herdsman 环境探测与兼容契约：gaea 启动时对本机 herdsman 服务
// 依赖的多个非公开契约做一次性探测，覆盖：
//   - config.yaml（存在/可读/非空，提取 api.lan_accessible、api.port）
//   - herdsman.exe CLI（HERDSMAN_EXE 优先，其次 PATH，不实际执行子命令）
//   - HTTP /v1/models（默认 3 秒超时；默认参数下不发真实请求）
//   - 数据契约：launch_records/、model_stats/events.jsonl、
//     skill-operations.json、models/
//
// herdsman 升级可能静默改变上述契约，探测结果供前端启动诊断与告警展示。
package herdsman

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// 探测默认值。
const (
	// defaultBaseURL 是 herdsman OpenAI 兼容 API 的默认地址。
	defaultBaseURL = "http://localhost:8080/v1"
	// apiTimeout 是真实 HTTP 探测的单次超时。
	apiTimeout = 3 * time.Second
)

// Probe 是 Herdsman 环境与数据契约的启动探测结果，字段均带 json tag，
// 可直接序列化给前端展示。未导出字段为探测运行时状态，不参与序列化。
type Probe struct {
	// HomeDir 探测的 herdsman 根目录（默认 %USERPROFILE%\.herdsman）。
	HomeDir string `json:"home_dir"`
	// ConfigPath 完整配置文件路径（HomeDir/config.yaml）。
	ConfigPath string `json:"config_path"`
	// ConfigOK 为 true 表示 config.yaml 存在、可读且非空。
	ConfigOK bool `json:"config_ok"`
	// ConfigError 在 ConfigOK=false 时说明具体原因。
	ConfigError string `json:"config_error,omitempty"`
	// Config 是从 config.yaml 提取的 api 段关键字段。
	Config Config `json:"config"`
	// CLI 是 herdsman.exe 可找到性结果。
	CLI CLIStatus `json:"cli"`
	// APIReachable 表示 HTTP 层能否访问 baseURL/models。
	APIReachable bool `json:"api_reachable"`
	// APIError 在 APIReachable=false 时说明原因（"skipped" 表示跳过真实探测）。
	APIError string `json:"api_error,omitempty"`
	// DataFiles 是各数据契约文件的检查结果，键固定为
	// launch_records、model_stats/events.jsonl、skill-operations.json、models。
	DataFiles map[string]FileStatus `json:"data_files"`
	// Warnings 是探测过程中发现的非致命告警（缺失、暴露、契约漂移等）。
	Warnings []string `json:"warnings,omitempty"`

	// ---- 以下为探测运行时状态（未导出，不序列化） ----
	rootDir      string // 已解析的根目录
	baseURL      string // 已解析的 baseURL
	baseURLIsDft bool   // baseURL 是否为默认值（用于端口漂移告警）
	live         bool   // 是否允许真实 HTTP 探测
}

// Config 是 config.yaml 提取的 api 段关键字段。
type Config struct {
	// LANAccessible 对应 api.lan_accessible（未解析成功时为 null）。
	LANAccessible *bool `json:"lan_accessible"`
	// Port 对应 api.port（未配置时取缺省 8080；未解析成功时为 null）。
	Port *int `json:"port"`
}

// CLIStatus 是 herdsman.exe 可找到性结果。
type CLIStatus struct {
	// Found 为 true 表示找到了可执行的 herdsman.exe。
	Found bool `json:"found"`
	// Path 是找到的可执行文件路径。
	Path string `json:"path,omitempty"`
	// Source 标识来源："env"（HERDSMAN_EXE）或 "path"（PATH）。
	Source string `json:"source,omitempty"`
	// Error 在未找到时说明原因。
	Error string `json:"error,omitempty"`
}

// FileStatus 是单个数据契约文件/目录的检查结果。
type FileStatus struct {
	// Exists 为 true 表示文件或目录存在。
	Exists bool `json:"exists"`
	// Parseable 为 true 表示内容可解析（文件按 JSON 抽查，目录按可读性）。
	Parseable bool `json:"parseable"`
	// Error 在检查失败时说明原因。
	Error string `json:"error,omitempty"`
}

// NewProbe 构造 Herdsman 环境探测器。
//
// rootDir 为空时使用默认目录 %USERPROFILE%\.herdsman（即 os.UserHomeDir()/.herdsman），
// baseURL 为空时使用默认 http://localhost:8080/v1。
//
// 真实 HTTP 探测的开关：当 rootDir 与 baseURL 均为默认值（空）且环境变量
// HERDSMAN_PROBE_LIVE != "1" 时，探测不发送真实 HTTP 请求（APIReachable=false、
// APIError="skipped"），保证单测与无服务环境不产生网络依赖；任一参数显式传入
// 或设置 HERDSMAN_PROBE_LIVE=1 时才真实探测。
func NewProbe(rootDir, baseURL string) *Probe {
	home := strings.TrimSpace(rootDir)
	if home == "" {
		home = defaultHomeDir()
	}
	url := strings.TrimSpace(baseURL)
	if url == "" {
		url = defaultBaseURL
	}
	// 任一输入非默认，或显式开启实时探测，则允许真实 HTTP 请求。
	live := strings.TrimSpace(rootDir) != "" ||
		strings.TrimSpace(baseURL) != "" ||
		os.Getenv("HERDSMAN_PROBE_LIVE") == "1"
	return &Probe{
		HomeDir:      home,
		rootDir:      home,
		baseURL:      url,
		baseURLIsDft: strings.TrimSpace(baseURL) == "",
		live:         live,
		DataFiles:    map[string]FileStatus{},
	}
}

// defaultHomeDir 计算默认 herdsman 数据目录：HERDSMAN_DATA_DIR 环境变量优先
// （与 internal/app 的 herdsmanDataDir 口径一致），回退 %USERPROFILE%\.herdsman。
func defaultHomeDir() string {
	if p := strings.TrimSpace(os.Getenv("HERDSMAN_DATA_DIR")); p != "" {
		return p
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".herdsman")
	}
	return ".herdsman"
}

// Run 执行完整探测并返回结构化结果（结果即探测器自身拷贝）。
func (p *Probe) Run() Probe {
	out := *p
	out.DataFiles = map[string]FileStatus{}
	out.probeConfig()
	out.probeCLI()
	out.probeDataFiles()
	out.probeAPI()
	return out
}

// probeConfig 检查 config.yaml：可读且非空即 ConfigOK=true；
// api 段字段提取复用 H0-4（lancheck.go）的逐行 YAML section 解析。
func (p *Probe) probeConfig() {
	p.ConfigPath = filepath.Join(p.rootDir, "config.yaml")
	data, err := os.ReadFile(p.ConfigPath)
	if err != nil {
		p.ConfigError = err.Error()
		p.Warnings = append(p.Warnings, fmt.Sprintf("config.yaml 缺失或不可读（%s）", p.ConfigPath))
		return
	}
	if len(bytes.TrimSpace(data)) == 0 {
		p.ConfigError = "config.yaml 为空"
		p.Warnings = append(p.Warnings, "config.yaml 存在但为空")
		return
	}
	p.ConfigOK = true

	// 复用 H0-4 的逐行 section 解析提取 api.lan_accessible 与 api.port，
	// 保持 YAML 契约解析逻辑单一来源（不引入 YAML 依赖）。
	expo := CheckLanExposure(p.ConfigPath)
	if expo.ParseError != "" {
		// 文件可读非空但字段格式异常：不影响 ConfigOK，仅告警。
		p.Warnings = append(p.Warnings, fmt.Sprintf("config.yaml api 段解析异常：%s", expo.ParseError))
		return
	}
	lan := expo.Exposed
	port := expo.Port
	p.Config.LANAccessible = &lan
	p.Config.Port = &port

	if lan {
		p.Warnings = append(p.Warnings, fmt.Sprintf(
			"config.yaml 中 api.lan_accessible=true：Herdsman API 暴露于局域网（端口 %d），请注意安全", port))
	}
	// 端口漂移告警：baseURL 用默认值且配置端口非 8080 时，默认连接可能失效。
	if p.baseURLIsDft && port != defaultPort {
		p.Warnings = append(p.Warnings, fmt.Sprintf(
			"config.yaml 中 api.port=%d 与默认端口 %d 不一致，gaea 默认连接的 %s 可能不可达", port, defaultPort, defaultBaseURL))
	}
}

// probeCLI 探测 herdsman.exe：HERDSMAN_EXE 环境变量优先（须真实存在），
// 其次用 exec.LookPath 在 PATH 中查找。只报告可找到与否，不执行任何子命令。
func (p *Probe) probeCLI() {
	if exe := strings.TrimSpace(os.Getenv("HERDSMAN_EXE")); exe != "" {
		if fi, err := os.Stat(exe); err == nil && !fi.IsDir() {
			p.CLI = CLIStatus{Found: true, Path: exe, Source: "env"}
			return
		}
		p.CLI = CLIStatus{Found: false, Path: exe, Source: "env", Error: "HERDSMAN_EXE 指向的文件不存在"}
		p.Warnings = append(p.Warnings, fmt.Sprintf("未找到 herdsman.exe：%s", p.CLI.Error))
		return
	}
	if exe, err := exec.LookPath("herdsman.exe"); err == nil {
		p.CLI = CLIStatus{Found: true, Path: exe, Source: "path"}
		return
	}
	p.CLI = CLIStatus{Found: false, Source: "path", Error: "PATH 中未找到 herdsman.exe"}
	p.Warnings = append(p.Warnings, fmt.Sprintf("未找到 herdsman.exe（%s）：模型目录/启动等 CLI 契约功能不可用", p.CLI.Error))
}

// probeDataFiles 检查四个数据契约。文件尝试 JSON 解析；JSONL 只抽查
// 第一行非空行；目录检查存在性与可读性。
func (p *Probe) probeDataFiles() {
	p.DataFiles["launch_records"] = p.checkDir("launch_records")
	p.DataFiles["models"] = p.checkDir("models")
	p.DataFiles["model_stats/events.jsonl"] = p.checkJSONL(filepath.Join(p.rootDir, "model_stats", "events.jsonl"))
	p.DataFiles["skill-operations.json"] = p.checkJSON(filepath.Join(p.rootDir, "skill-operations.json"))

	// 目录缺失告警：打印被检查的真实目录路径（checkDir 用的
	// rootDir/name），而非配置文件路径 ConfigPath。
	for _, name := range []string{"launch_records", "models"} {
		if st := p.DataFiles[name]; !st.Exists {
			p.Warnings = append(p.Warnings, fmt.Sprintf("数据目录缺失：%s（%s）", name, filepath.Join(p.rootDir, name)))
		}
	}
	// 文件缺失/不可解析告警。
	for _, name := range []string{"model_stats/events.jsonl", "skill-operations.json"} {
		st := p.DataFiles[name]
		switch {
		case !st.Exists:
			p.Warnings = append(p.Warnings, fmt.Sprintf("数据文件缺失：%s", name))
		case !st.Parseable:
			p.Warnings = append(p.Warnings, fmt.Sprintf("数据文件不可解析：%s（%s）", name, st.Error))
		}
	}
}

// checkDir 检查目录：存在且可列出即视为可解析（空目录属正常状态）。
func (p *Probe) checkDir(name string) FileStatus {
	st := FileStatus{}
	if _, err := os.ReadDir(filepath.Join(p.rootDir, name)); err != nil {
		st.Error = err.Error()
		return st
	}
	st.Exists = true
	st.Parseable = true
	return st
}

// checkJSON 检查单个 JSON 文件：整体 unmarshal 到 interface{} 成功即可解析。
func (p *Probe) checkJSON(path string) FileStatus {
	st := FileStatus{}
	data, err := os.ReadFile(path)
	if err != nil {
		st.Error = err.Error()
		return st
	}
	st.Exists = true
	// 兼容 UTF-8 BOM（与仓库既有 herdsman 解析一致）。
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		st.Error = err.Error()
		return st
	}
	st.Parseable = true
	return st
}

// checkJSONL 检查 JSONL 文件：只抽查第一行非空行能否 unmarshal；
// 空文件视为可解析（零条记录合法）。
func (p *Probe) checkJSONL(path string) FileStatus {
	st := FileStatus{}
	data, err := os.ReadFile(path)
	if err != nil {
		st.Error = err.Error()
		return st
	}
	st.Exists = true
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		var v interface{}
		if err := json.Unmarshal([]byte(line), &v); err != nil {
			st.Error = fmt.Sprintf("首行非空记录解析失败：%v", err)
			return st
		}
		st.Parseable = true
		return st
	}
	// 无非空行：空 JSONL 视为可解析。
	st.Parseable = true
	return st
}

// probeAPI 探测 baseURL/models 可达性。live=false 时跳过真实请求。
func (p *Probe) probeAPI() {
	if !p.live {
		p.APIReachable = false
		p.APIError = "skipped"
		return
	}
	url := strings.TrimRight(p.baseURL, "/") + "/models"
	client := &http.Client{Timeout: apiTimeout}
	resp, err := client.Get(url)
	if err != nil {
		p.APIReachable = false
		p.APIError = err.Error()
		p.Warnings = append(p.Warnings, fmt.Sprintf("Herdsman API 不可达：%s", err))
		return
	}
	defer resp.Body.Close()

	// 收到任何 HTTP 响应即视为可达（传输层连通）；
	// 非 2xx/3xx 说明端点契约可能已变更（herdsman 升级场景），给出告警。
	p.APIReachable = true
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		p.Warnings = append(p.Warnings, fmt.Sprintf(
			"GET %s 返回 HTTP %d：OpenAI 兼容契约可能已变更", url, resp.StatusCode))
	}
}
