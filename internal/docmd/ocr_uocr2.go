package docmd

// ocr_uocr2.go — OCR Provider Seam 的唯一引擎：Unlimited-OCR-2（百度/DeepSeek-OCR
// 血统 3B MoE + SR-SWA，MIT）。常驻 llama-server（b11980+unlimited-ocr2.patch + Q8_0
// GGUF，工件 C:\AI\gaea-OCR\uocr2-llama，见该目录 README），8060S Vulkan 实测单页 6s /
// 15 页 106s，SR-SWA 跨页续排完整保住。协议 = OpenAI chat completions；
// GAEA_UOCR2_LLAMA_URL / _PORT / _DIR / _MODEL / _MMPROJ 配置（默认 127.0.0.1:8139 /
// C:\AI\gaea-OCR\uocr2-llama）。无 GPU 时 llama-server 自行忽略 -ngl 回退 CPU（Q8 实测
// 约 20s/页）。
//
// 2026-11-26 精简（用户拍板「本地 OCR 只留最好的一个」）：python transformers 后端
// （transformers server.py，/ocr + /ocr_batch，端口 8138，CPU 约 1 分钟/页）退役——
// 同一模型下 llama 后端快 10~14 倍，CPU 路径上 llama 也更快，python 后端没有独占价值。
// GAEA_UOCR2_BACKEND / GAEA_UOCR2_URL / _PORT / _DIR / _PY 不再读取。
//
// llama 后端两条铁律（2026-10-29 实弹实证，详见各自函数注释）：
//   - 铁律一：chat 消息必须以文本块收尾——补丁把 unlimited-ocr2 的 img_end 置空，
//     图片收尾的请求触发 server 空批断言整进程崩溃（uocr2LlamaChatContent 客户端
//     无条件追加收尾文本）；
//   - 铁律二：--ubatch-size 4096——默认 512+flash-attn 下多图请求丢前面的图
//     （buildUocr2LlamaCmd 固化）。
//
// 模型原始输出含 <PAGE>/<BLOCK>…<FORK> 结构标记，uocr2CleanText 在 Go 侧清洗。

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gaea/gaea/internal/gaea/proc"
)

// uocr2PageMaxTokens 单页输出 token 上限（官方口径 tokens_per_page=8000）。
const uocr2PageMaxTokens = 8000

// uocr2LlamaBaseURL 解析常驻 llama-server 的服务地址（GAEA_UOCR2_LLAMA_URL 优先，其次
// GAEA_UOCR2_LLAMA_PORT，默认 8139）。
func uocr2LlamaBaseURL() string {
	if u := strings.TrimRight(os.Getenv("GAEA_UOCR2_LLAMA_URL"), "/"); u != "" {
		return u
	}
	port := os.Getenv("GAEA_UOCR2_LLAMA_PORT")
	if port == "" {
		port = "8139"
	}
	return "http://127.0.0.1:" + port
}

// uocr2ServerBase 解析可用服务：健康探测 → 按需拉起，返回 base。不可用返回 ""
// （调用方报「服务不可用」）。
func uocr2ServerBase() string {
	client := &http.Client{Timeout: 3 * time.Second}
	base := uocr2LlamaBaseURL()
	if uocr2Healthy(client, base) {
		return base
	}
	if startUocr2LlamaServer(client, base) {
		return base
	}
	return ""
}

// uocr2Healthy 探测 /health（服务端模型加载完成后才监听，应答即就绪）。
func uocr2ServerHealthy(c *http.Client, base string) bool {
	resp, err := c.Get(base + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
	return strings.Contains(string(body), "ok")
}

// startTrackedServer 拉起常驻服务（隐藏窗口，Job Object 跟踪）并等待就绪
// （≤uocr2StartWait，默认 120s；llama Q8 实测约 10s，留足冷启动余量）。任何失败
// 路径都在返回 false 前杀整棵进程树，保证不残留孤儿。
func startTrackedServer(c *http.Client, base string, build func() (*exec.Cmd, *os.File, bool)) bool {
	cmd, logf, ok := build()
	if !ok {
		return false
	}
	proc.HideWindow(cmd)
	handle, err := proc.StartTracked(cmd)
	if err != nil {
		if logf != nil {
			_ = logf.Close()
		}
		return false
	}
	if logf != nil {
		_ = logf.Close() // 子进程已持有文件句柄，父进程可立即释放
	}
	deadline := time.Now().Add(uocr2StartWait)
	for time.Now().Before(deadline) {
		time.Sleep(time.Second)
		if uocr2Healthy(c, base) {
			go func() { _ = cmd.Wait() }() // 保留进程存活，异步回收
			return true
		}
	}
	proc.KillTracked(cmd, handle) // 超时：杀整棵进程树，防孤儿残留
	_ = cmd.Wait()                // 同步回收，返回 false 前确保进程已终止
	return false
}

// startUocr2LlamaServer 拉起 llama-server。
func startUocr2LlamaServer(c *http.Client, base string) bool {
	return startTrackedServer(c, base, func() (*exec.Cmd, *os.File, bool) { return uocr2LlamaBuildCmd() })
}

// uocr2StartWait 是 startTrackedServer 等待服务就绪的时长；包级变量便于单测缩短等待。
var uocr2StartWait = 120 * time.Second

// uocr2Healthy 是 startTrackedServer 轮询的就绪探针；包级变量便于单测注入 fake。
var uocr2Healthy = uocr2ServerHealthy

// uocr2LlamaBuildCmd 构造 llama-server 命令；包级变量便于单测注入假进程。
var uocr2LlamaBuildCmd = buildUocr2LlamaServerCmd

// buildUocr2LlamaServerCmd 按 GAEA_UOCR2_LLAMA_* 配置构造 llama-server 命令与其
// 日志文件。返回 ok=false 表示工件缺失（llama-server.exe / 模型 / mmproj）。
//
// 启动口径逐字来自 2026-10-29 实弹验证（C:\AI\gaea-OCR\uocr2-llama\README.md）：
//   - --ubatch-size 4096 = 铁律二：默认 512+flash-attn 下多图请求丢前面的图
//     （实测 2 页只转写第 2 页且不报错；-fa 0 亦可修复但更慢），4096 保证图片
//     嵌入块不被 ubatch 切割，勿删；
//   - -c 65536 覆盖 5 页窗（实测 15 页 prompt≈11.5k+生成 8.4k）；-np 1 单序列
//     （unlimited-ocr2 补丁的多序列路径未验证）；-ngl 99 无 GPU 时 llama-server
//     自行忽略并回退 CPU（Q8 CPU 实测约 20s/页）。
func buildUocr2LlamaServerCmd() (*exec.Cmd, *os.File, bool) {
	dir := os.Getenv("GAEA_UOCR2_LLAMA_DIR")
	if dir == "" {
		dir = `C:\AI\gaea-OCR\uocr2-llama`
	}
	model := os.Getenv("GAEA_UOCR2_LLAMA_MODEL")
	if model == "" {
		model = "uocr2-Q8_0.gguf"
	}
	mmproj := os.Getenv("GAEA_UOCR2_LLAMA_MMPROJ")
	if mmproj == "" {
		mmproj = "mmproj-uocr2-F16.gguf"
	}
	// OCR 专用构建钉点：unlimited-ocr2 是自定义架构，mainline llama.cpp 不认识
	// （2026-11-11 bin/ 被 llama.cpp 更新机制换成 mainline 即崩，日志留档）。
	// bin/ 归 qwen35 自管引擎（更新追踪），OCR 恒用 bin-ocr-0.8.0-dev。
	exe := filepath.Join(dir, "bin-ocr-0.8.0-dev", "llama-server.exe")
	modelPath := filepath.Join(dir, model)
	mmprojPath := filepath.Join(dir, mmproj)
	for _, p := range []string{exe, modelPath, mmprojPath} {
		if _, err := os.Stat(p); err != nil {
			return nil, nil, false
		}
	}
	port := "8139"
	if u, err := url.Parse(uocr2LlamaBaseURL()); err == nil && u.Port() != "" {
		port = u.Port()
	}
	var logf *os.File
	if f, err := os.OpenFile(filepath.Join(dir, "server.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err == nil {
		logf = f
	}
	cmd := exec.Command(exe,
		"-m", modelPath, "--mmproj", mmprojPath,
		"--host", "127.0.0.1", "--port", port,
		"-c", "65536", "-np", "1", "-ngl", "99",
		"--temp", "0", "--top-k", "1",
		"--ubatch-size", "4096")
	if logf != nil {
		cmd.Stdout = logf
		cmd.Stderr = logf
	}
	return cmd, logf, true
}

// uocr2CleanText 清洗模型原始输出的结构标记为纯文本：
// <BLOCK>header[42, 16, 143, 27]<FORK>正文 → 正文；<PAGE> 移除。
// 防御：偶发生成坏标记（GPU bf16 数字噪声实测出现过 <FORK> 缺失、坐标与正文
// 粘连）时，含 <BLOCK> 而整行无 <FORK> 的行视为结构垃圾整行剥掉——良 form 行
// 必有 <FORK>，坏行内容本就不可恢复，宁缺勿把坐标碎片漏进正文。
func uocr2CleanText(raw string) string {
	s := blockMarkerRe.ReplaceAllString(raw, "")
	lines := strings.Split(s, "\n")
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.Contains(line, "<BLOCK>") && !strings.Contains(line, "<FORK>") {
			continue // 坏标记行：整行丢弃
		}
		kept = append(kept, line)
	}
	s = strings.Join(kept, "\n")
	s = strings.ReplaceAll(s, "<PAGE>", "")
	return strings.TrimSpace(s)
}

// blockMarkerRe 非贪婪行内匹配（. 不跨行）：坐标/内容段里出现任何 `<`（嵌套
// 标签、模型回显）都会让 [^<]* 形态漏清洗——15 页全量实弹实证泄漏，故用 .*?。
var blockMarkerRe = regexp.MustCompile(`<BLOCK>.*?<FORK>`)

// llamaChatPiece chat completions 的 content 分片：文本块或图片块。
type llamaChatPiece struct {
	Type     string          `json:"type"`
	Text     string          `json:"text,omitempty"`
	ImageURL *llamaChatImage `json:"image_url,omitempty"`
}

type llamaChatImage struct {
	URL string `json:"url"`
}

// uocr2LlamaMultiPagePrompt 是收尾指令文本，也是铁律一的载体。
const uocr2LlamaMultiPagePrompt = "Multi page parsing."

// uocr2LlamaChatContent 构造 chat content：每页 [<PAGE> 文本块, 图片块]，末尾
// 无条件追加收尾文本块。铁律一（2026-10-29 实弹实证）：unlimited-ocr2 补丁把
// img_end 置空（页界靠精确 <PAGE><image> token 流），**图片收尾的请求会触发
// llama-server 空批断言 GGML_ASSERT(batch.size()>0) 整进程崩溃**——收尾文本块
// 只在此处构造，不信任任何调用方，勿改为可选。
func uocr2LlamaChatContent(imagesB64 []string) []llamaChatPiece {
	content := make([]llamaChatPiece, 0, len(imagesB64)*2+1)
	for _, b64 := range imagesB64 {
		content = append(content,
			llamaChatPiece{Type: "text", Text: "<PAGE>"},
			llamaChatPiece{Type: "image_url", ImageURL: &llamaChatImage{
				URL: "data:image/png;base64," + b64,
			}},
		)
	}
	content = append(content, llamaChatPiece{Type: "text", Text: uocr2LlamaMultiPagePrompt})
	return content
}

// uocr2BatchTimeout 单批请求超时：基座 180s + 每页 120s 裕量（CPU 实测约 60-80s/页，
// 窗口越大线性增长；超时防呆不等价于服务端截断判定）。
func uocr2BatchTimeout(pages int) time.Duration {
	return 180*time.Second + time.Duration(pages)*120*time.Second
}

// uocr2LlamaChat 调 llama-server 的 /v1/chat/completions 识别一页或多页，
// 返回模型原始输出（含结构标记，单/多页共用；多页由调用方按 <PAGE> 拆分）。
// finish_reason=length 视为截断 fail-closed（对齐 python server 422 口径）；
// 其余非 200 一律报错（fail-closed，由编排层回退处理）。
func uocr2LlamaChat(base string, pngPaths []string) (string, error) {
	b64s := make([]string, len(pngPaths))
	for i, p := range pngPaths {
		data, err := os.ReadFile(p)
		if err != nil {
			return "", err
		}
		b64s[i] = base64.StdEncoding.EncodeToString(data)
	}
	payload := struct {
		Messages []struct {
			Role    string           `json:"role"`
			Content []llamaChatPiece `json:"content"`
		} `json:"messages"`
		MaxTokens   int     `json:"max_tokens"`
		Temperature float64 `json:"temperature"`
	}{
		Messages: []struct {
			Role    string           `json:"role"`
			Content []llamaChatPiece `json:"content"`
		}{{Role: "user", Content: uocr2LlamaChatContent(b64s)}},
		MaxTokens:   uocr2PageMaxTokens * len(pngPaths),
		Temperature: 0,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	// llama 实测 2~13s/请求（单页约 6s、15 页 106s）；批量超时公式（180s+120s/页）
	// 留足裕量，冷路径（无 GPU 回退 CPU 约 20s/页）同样覆盖。
	client := &http.Client{Timeout: uocr2BatchTimeout(len(pngPaths))}
	resp, err := client.Post(base+"/v1/chat/completions", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
		return "", fmt.Errorf("Unlimited-OCR-2 llama 服务返回 %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var out struct {
		Choices []struct {
			FinishReason string `json:"finish_reason"`
			Message      struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<20)).Decode(&out); err != nil {
		return "", err
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("Unlimited-OCR-2 llama 应答无 choices")
	}
	if out.Choices[0].FinishReason == "length" {
		return "", fmt.Errorf("Unlimited-OCR-2 llama 输出被 max_tokens 截断（finish_reason=length），不把残缺文本当完整结果")
	}
	content := out.Choices[0].Message.Content
	if strings.TrimSpace(content) == "" {
		return "", fmt.Errorf("Unlimited-OCR-2 llama 空输出")
	}
	return content, nil
}

// uocr2LlamaPageOCR 单页识别（输出清洗见 uocr2CleanText）。
func uocr2LlamaPageOCR(base, pngPath string) (string, error) {
	raw, err := uocr2LlamaChat(base, []string{pngPath})
	if err != nil {
		return "", err
	}
	return uocr2CleanText(raw), nil
}

// uocr2LlamaPagesOCR 多页批识别：一次 chat 请求多图（SR-SWA 跨页上下文在服务端
// 共享），输出按 <PAGE> 拆分。页数对不上 = 生成缺页 fail-closed（编排层整窗回退
// 逐页）。
func uocr2LlamaPagesOCR(base string, pngPaths []string) ([]string, error) {
	raw, err := uocr2LlamaChat(base, pngPaths)
	if err != nil {
		return nil, err
	}
	parts := strings.Split(raw, "<PAGE>")
	if strings.TrimSpace(parts[0]) != "" {
		return nil, fmt.Errorf("Unlimited-OCR-2 llama 批输出首段非空（结构异常）")
	}
	pages := parts[1:]
	if len(pages) != len(pngPaths) {
		return nil, fmt.Errorf("Unlimited-OCR-2 llama 批识别页数不齐（%d/%d）", len(pages), len(pngPaths))
	}
	texts := make([]string, len(pages))
	for i, p := range pages {
		texts[i] = uocr2CleanText(p)
	}
	return texts, nil
}

// uocr2OCRProvider 常驻 Unlimited-OCR-2 服务的 OCR 提供者：首次使用探测/拉起并
// 缓存 base URL；探测失败进入短冷却（避免逐页重复拉起）。不可用即返回错误。
type uocr2OCRProvider struct {
	mu       sync.Mutex
	base     string // 已解析的常驻服务 base URL；空 = 未解析/不可用
	lastFail time.Time
}

// uocr2ProbeCooldown 探测失败后的冷却时长：冷却期内不再重复探测/拉起服务。
const uocr2ProbeCooldown = 5 * time.Second

func (p *uocr2OCRProvider) Name() string { return OCRKindUocr2 }

// Available 预探测提供者是否可用（解析并缓存常驻服务地址；失败进入冷却）。
func (p *uocr2OCRProvider) Available() bool { return p.resolveBase() != "" }

// ExtractImage 识别单张图片；服务不可用返回错误（fail-closed）。
func (p *uocr2OCRProvider) ExtractImage(path string) (string, error) {
	base := p.resolveBase()
	if base == "" {
		return "", fmt.Errorf("Unlimited-OCR-2 服务不可用（未安装或无法拉起）")
	}
	text, err := uocr2LlamaPageOCR(base, path)
	if err != nil {
		p.dropBaseIfTransportErr(err) // 传输层失败=服务大概率已死（重启/被杀），下页重新探测/拉起
		return "", err
	}
	return strings.TrimSpace(text), nil
}

// dropBaseIfTransportErr 传输层错误（连接拒绝/重置等 *url.Error）时清除缓存的
// base URL：长驻 gaea 进程里服务可能中途消亡（重启/被杀），不清会导致本次
// 会话内 OCR 永远打死地址、永远走不到重拉。HTTP 状态错误不算——服务活着。
func (p *uocr2OCRProvider) dropBaseIfTransportErr(err error) {
	var urlErr *url.Error
	if !errors.As(err, &urlErr) {
		return
	}
	p.mu.Lock()
	p.base = ""
	p.mu.Unlock()
}

// ExtractPages 批量接口：一批 PNG（一窗，≤ocrBatchWindow 页）一次请求识别（跨页
// 上下文共享）。顺序与输入严格一致；任一页失败整体报错（编排层整窗回退逐页）。
func (p *uocr2OCRProvider) ExtractPages(paths []string) ([]string, error) {
	if len(paths) == 0 {
		return nil, fmt.Errorf("ExtractPages：空页列表")
	}
	base := p.resolveBase()
	if base == "" {
		return nil, fmt.Errorf("Unlimited-OCR-2 服务不可用（未安装或无法拉起）")
	}
	texts, err := uocr2LlamaPagesOCR(base, paths)
	if err != nil {
		p.dropBaseIfTransportErr(err)
		return nil, err
	}
	return texts, nil
}

// resolveBase 解析常驻服务地址：成功缓存，失败记录冷却时间。
func (p *uocr2OCRProvider) resolveBase() string {
	p.mu.Lock()
	if p.base != "" {
		base := p.base
		p.mu.Unlock()
		return base
	}
	if time.Since(p.lastFail) < uocr2ProbeCooldown {
		p.mu.Unlock()
		return ""
	}
	p.mu.Unlock()

	base := uocr2ServerBase() // 探测/拉起（注入点可控，测试不依赖真实服务）
	p.mu.Lock()
	if base != "" {
		p.base = base
	} else {
		p.lastFail = time.Now()
	}
	p.mu.Unlock()
	return base
}

func init() {
	RegisterOCRProvider(OCRKindUocr2, func() OCRProvider { return &uocr2OCRProvider{} })
}
