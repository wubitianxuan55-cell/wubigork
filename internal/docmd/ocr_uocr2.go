package docmd

// ocr_uocr2.go — OCR Provider Seam 的 uocr2 引擎（Unlimited-OCR-2，百度/DeepSeek-OCR
// 血统 3B MoE + SR-SWA，MIT）：常驻 Python HTTP 服务（默认 C:\AI\unlimited-ocr2，
// transformers bf16，模型驻留内存、端口应答即就绪），探测/拉起/健康检查与单页识别
// 编排口径与 ovis（ocr.go）同款：on-demand 拉起、失败短冷却、fail-closed。
//
// 服务协议（server.py）：
//
//	GET  /health → {"status":"ok"}（模型加载完成后才监听）
//	POST /ocr    {"image":<base64 png>,"max_tokens":N} → {"text":…}；
//	             422 = 输出被 max_tokens 截断（同 ovis finish_reason=length 口径，
//	             不把残缺文本当完整结果，按单页失败处理走引擎链回退）
//
// 模型原始输出含 <PAGE>/<BLOCK>…<FORK> 结构标记，uocr2CleanText 在 Go 侧清洗
// （可单测；服务端返回原始输出）。

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

// uocr2ServerBase 返回常驻 Unlimited-OCR-2 服务的 base URL；未安装/无法拉起时返回 ""。
// 优先复用已在跑的实例，否则按需静默拉起一次。
func uocr2ServerBase() string {
	base := strings.TrimRight(os.Getenv("GAEA_UOCR2_URL"), "/")
	if base == "" {
		port := os.Getenv("GAEA_UOCR2_PORT")
		if port == "" {
			port = "8138"
		}
		base = "http://127.0.0.1:" + port
	}
	client := &http.Client{Timeout: 3 * time.Second}
	if uocr2Healthy(client, base) {
		return base
	}
	if startUocr2Server(client, base) {
		return base
	}
	return ""
}

// uocr2ServerHealthy 探测 /health（服务端模型加载完成后才监听，应答即就绪）。
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

// startUocr2Server 拉起常驻 python 服务（隐藏窗口，Job Object 跟踪）。等待就绪
// （≤uocr2StartWait，默认 120s——首次含 torch 导入+模型加载）。任何失败路径都在
// 返回 false 前杀整棵进程树，保证不残留孤儿 python。
func startUocr2Server(c *http.Client, base string) bool {
	cmd, logf, ok := uocr2BuildCmd()
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

// uocr2StartWait 是 startUocr2Server 等待服务就绪的时长（默认 120s：CPU 冷启动
// 含 torch 导入+模型加载约 20s，留足裕量）；包级变量便于单测缩短等待。
var uocr2StartWait = 120 * time.Second

// uocr2Healthy 是 startUocr2Server 轮询的就绪探针；包级变量便于单测注入 fake。
var uocr2Healthy = uocr2ServerHealthy

// uocr2BuildCmd 构造服务命令；包级变量便于单测注入假进程。
var uocr2BuildCmd = buildUocr2ServerCmd

// buildUocr2ServerCmd 按 GAEA_UOCR2_* 配置构造 python 服务命令与其日志文件。
// 返回 ok=false 表示配置缺失（python/server.py 不存在）。
func buildUocr2ServerCmd() (*exec.Cmd, *os.File, bool) {
	dir := os.Getenv("GAEA_UOCR2_DIR")
	if dir == "" {
		dir = `C:\AI\unlimited-ocr2`
	}
	py := os.Getenv("GAEA_UOCR2_PY")
	if py == "" {
		py = filepath.Join(dir, "venv", "Scripts", "python.exe")
	}
	server := filepath.Join(dir, "server.py")
	for _, p := range []string{py, server} {
		if _, err := os.Stat(p); err != nil {
			return nil, nil, false
		}
	}
	port := os.Getenv("GAEA_UOCR2_PORT")
	if port == "" {
		port = "8138"
	}
	var logf *os.File
	if f, err := os.OpenFile(filepath.Join(dir, "server.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err == nil {
		logf = f
	}
	cmd := exec.Command(py, server, "--host", "127.0.0.1", "--port", port)
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

// uocr2BatchTimeout 单批请求超时：基座 180s + 每页 120s 裕量（CPU 实测约 60-80s/页，
// 窗口越大线性增长；超时防呆不等价于服务端截断判定）。
func uocr2BatchTimeout(pages int) time.Duration {
	return 180*time.Second + time.Duration(pages)*120*time.Second
}

// uocr2PagesOCR 把一批 PNG（一窗，≤ocrBatchWindow 页）发给常驻服务的 /ocr_batch，
// 返回与输入等长同序、清洗后的识别文本。422（截断/页数对不上）与其余非 200 一律
// 报错（fail-closed，由编排层整窗回退逐页）。
func uocr2PagesOCR(base string, pngPaths []string) ([]string, error) {
	images := make([]string, 0, len(pngPaths))
	for _, p := range pngPaths {
		data, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		images = append(images, base64.StdEncoding.EncodeToString(data))
	}
	payload := map[string]any{
		"images":              images,
		"max_tokens_per_page": uocr2PageMaxTokens,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: uocr2BatchTimeout(len(pngPaths))}
	resp, err := client.Post(base+"/ocr_batch", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var e struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&e)
		return nil, fmt.Errorf("Unlimited-OCR-2 批识别返回 %d: %s", resp.StatusCode, e.Error)
	}
	var out struct {
		Texts []string `json:"texts"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<20)).Decode(&out); err != nil {
		return nil, err
	}
	if len(out.Texts) != len(pngPaths) {
		return nil, fmt.Errorf("Unlimited-OCR-2 批识别页数不齐（%d/%d）", len(out.Texts), len(pngPaths))
	}
	texts := make([]string, len(out.Texts))
	for i, raw := range out.Texts {
		texts[i] = uocr2CleanText(raw)
	}
	return texts, nil
}

// uocr2PageOCR 把一页 PNG 发给常驻 Unlimited-OCR-2 服务，返回清洗后的识别文本。
// 422（截断）与其余非 200 一律报错（fail-closed，由引擎链回退处理）。
func uocr2PageOCR(base, pngPath string) (string, error) {
	data, err := os.ReadFile(pngPath)
	if err != nil {
		return "", err
	}
	payload := map[string]any{
		"image":      base64.StdEncoding.EncodeToString(data),
		"max_tokens": uocr2PageMaxTokens,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	// CPU 推理约 50-60s/页，超时给足 10 分钟（远大于单页正常耗时，防呆不等价于截断）。
	client := &http.Client{Timeout: 600 * time.Second}
	resp, err := client.Post(base+"/ocr", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var e struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&e)
		return "", fmt.Errorf("Unlimited-OCR-2 服务返回 %d: %s", resp.StatusCode, e.Error)
	}
	var out struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 32<<20)).Decode(&out); err != nil {
		return "", err
	}
	return uocr2CleanText(out.Text), nil
}

// uocr2OCRProvider 常驻 Unlimited-OCR-2 服务 OCR 提供者：首次使用探测/拉起并缓存
// base URL；探测失败进入短冷却（避免逐页重复拉起）。不可用即返回错误。
type uocr2OCRProvider struct {
	mu       sync.Mutex
	base     string // 已解析的常驻服务 base URL；空 = 未解析/不可用
	lastFail time.Time
}

func (p *uocr2OCRProvider) Name() string { return OCRKindUocr2 }

// Available 预探测提供者是否可用（解析并缓存常驻服务地址；失败进入冷却）。
func (p *uocr2OCRProvider) Available() bool { return p.resolveBase() != "" }

// ExtractImage 识别单张图片；服务不可用返回错误（fail-closed）。
func (p *uocr2OCRProvider) ExtractImage(path string) (string, error) {
	base := p.resolveBase()
	if base == "" {
		return "", fmt.Errorf("Unlimited-OCR-2 服务不可用（未安装或无法拉起）")
	}
	text, err := uocr2PageOCR(base, path)
	if err != nil {
		p.dropBaseIfTransportErr(err) // 传输层失败=服务大概率已死（重启/被杀），下页重新探测/拉起
		return "", err
	}
	return strings.TrimSpace(text), nil
}

// dropBaseIfTransportErr 传输层错误（连接拒绝/重置等 *url.Error）时清除缓存的
// base URL：长驻 gaea 进程里服务可能中途消亡（重启/被杀），不清会导致本次
// 会话内 OCR 永远打死地址、永远走不到重拉。HTTP 状态错误（422/500）不算——
// 服务活着，地址仍有效。
func (p *uocr2OCRProvider) dropBaseIfTransportErr(err error) {
	var urlErr *url.Error
	if !errors.As(err, &urlErr) {
		return
	}
	p.mu.Lock()
	p.base = ""
	p.mu.Unlock()
}

// ExtractPages 实现批量接口：一批 PNG（一窗，≤ocrBatchWindow 页）一次请求识别
// （跨页上下文共享）。顺序与输入严格一致；任一页失败整体报错（编排层整窗回退逐页）。
func (p *uocr2OCRProvider) ExtractPages(paths []string) ([]string, error) {
	if len(paths) == 0 {
		return nil, fmt.Errorf("ExtractPages：空页列表")
	}
	base := p.resolveBase()
	if base == "" {
		return nil, fmt.Errorf("Unlimited-OCR-2 服务不可用（未安装或无法拉起）")
	}
	texts, err := uocr2PagesOCR(base, paths)
	if err != nil {
		p.dropBaseIfTransportErr(err)
		return nil, err
	}
	return texts, nil
}

// resolveBase 解析常驻服务 base URL：成功缓存，失败记录冷却时间。
func (p *uocr2OCRProvider) resolveBase() string {
	p.mu.Lock()
	if p.base != "" {
		base := p.base
		p.mu.Unlock()
		return base
	}
	if time.Since(p.lastFail) < ovisProbeCooldown {
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
