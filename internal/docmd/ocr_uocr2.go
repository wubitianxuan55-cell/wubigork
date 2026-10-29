package docmd

// ocr_uocr2.go — OCR Provider Seam 的 uocr2 引擎（Unlimited-OCR-2，百度/DeepSeek-OCR
// 血统 3B MoE + SR-SWA，MIT）。双后端（2026-10-29 拍板：llama 主、python 兜底）：
//
//   - llama 后端（默认）：常驻 llama-server（b11980+unlimited-ocr2.patch + Q8_0 GGUF，
//     工件 C:\AI\uocr2-llama，见该目录 README），8060S Vulkan 实测单页 6s/15 页
//     106s（python CPU 的 10~14 倍），SR-SWA 跨页续排完整保住。协议=OpenAI chat
//     completions；GAEA_UOCR2_LLAMA_URL/_PORT/_DIR/_MODEL/_MMPROJ 配置（默认
//     127.0.0.1:8139 / C:\AI\uocr2-llama）。
//   - python 后端（兜底）：transformers server.py（scripts/uocr2 为版本控制副本，
//     运行实例 C:\AI\unlimited-ocr2），协议=/ocr+/ocr_batch；GAEA_UOCR2_URL/_PORT/
//     _DIR/_PY 配置（默认 8138）。
//
// GAEA_UOCR2_BACKEND 驱动腿序：空/auto = llama→python 自动回退；显式 llama/python
// 仅该后端；未知值 fail-closed。两后端 /health 应答同为 {"status":"ok"}，共用探活。
//
// llama 后端两条铁律（2026-10-29 实弹实证，详见各自函数注释）：
//   - 铁律一：chat 消息必须以文本块收尾——补丁把 unlimited-ocr2 的 img_end 置空，
//     图片收尾的请求触发 server 空批断言整进程崩溃（uocr2LlamaChatContent 客户端
//     无条件追加收尾文本）；
//   - 铁律二：--ubatch-size 4096——默认 512+flash-attn 下多图请求丢前面的图
//     （buildUocr2LlamaCmd 固化）。
//
// 模型原始输出含 <PAGE>/<BLOCK>…<FORK> 结构标记，uocr2CleanText 在 Go 侧清洗
// （两后端输出同格式，清洗器共用；可单测）。

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

// uocr2BackendKind 常驻 Unlimited-OCR-2 服务的后端形态。
type uocr2BackendKind string

const (
	uocr2BackendLlama  uocr2BackendKind = "llama"
	uocr2BackendPython uocr2BackendKind = "python"
)

// uocr2BackendOrder 解析 GAEA_UOCR2_BACKEND：空/"auto" = llama→python 自动回退
// （2026-10-29 拍板：质量同级下 llama 快 10~14 倍）；显式 "llama"/"python" 仅该
// 后端；其他值 fail-closed 报错（与 GAEA_OCR_ENGINE 口径一致，不静默回退）。
func uocr2BackendOrder() ([]uocr2BackendKind, error) {
	raw := strings.TrimSpace(os.Getenv("GAEA_UOCR2_BACKEND"))
	switch {
	case raw == "", strings.EqualFold(raw, "auto"):
		return []uocr2BackendKind{uocr2BackendLlama, uocr2BackendPython}, nil
	case strings.EqualFold(raw, string(uocr2BackendLlama)):
		return []uocr2BackendKind{uocr2BackendLlama}, nil
	case strings.EqualFold(raw, string(uocr2BackendPython)):
		return []uocr2BackendKind{uocr2BackendPython}, nil
	}
	return nil, fmt.Errorf("docmd: 未知 GAEA_UOCR2_BACKEND %q（可用：auto / llama / python）", raw)
}

// uocr2LlamaBaseURL 解析 llama 后端服务地址（GAEA_UOCR2_LLAMA_URL 优先，其次
// GAEA_UOCR2_LLAMA_PORT，默认 8139——与 python 的 8138 分离，两后端可并存）。
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

// uocr2PythonBaseURL 解析 python 后端服务地址（原单后端口径不变）。
func uocr2PythonBaseURL() string {
	base := strings.TrimRight(os.Getenv("GAEA_UOCR2_URL"), "/")
	if base == "" {
		port := os.Getenv("GAEA_UOCR2_PORT")
		if port == "" {
			port = "8138"
		}
		base = "http://127.0.0.1:" + port
	}
	return base
}

// uocr2ServerBase 按后端腿序解析可用服务：健康探测→按需拉起，返回 base 与后端
// 形态。全部不可用返回 ("", "", nil)（调用方报「服务不可用」）；配置错误（未知
// GAEA_UOCR2_BACKEND）返回 err fail-closed 上抛。
func uocr2ServerBase() (string, uocr2BackendKind, error) {
	kinds, err := uocr2BackendOrder()
	if err != nil {
		return "", "", err
	}
	client := &http.Client{Timeout: 3 * time.Second}
	for _, kind := range kinds {
		var base string
		if kind == uocr2BackendLlama {
			base = uocr2LlamaBaseURL()
		} else {
			base = uocr2PythonBaseURL()
		}
		if uocr2Healthy(client, base) {
			return base, kind, nil
		}
		var start func(*http.Client, string) bool
		if kind == uocr2BackendLlama {
			start = startUocr2LlamaServer
		} else {
			start = startUocr2Server
		}
		if start(client, base) {
			return base, kind, nil
		}
	}
	return "", "", nil
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

// startTrackedServer 拉起常驻服务（隐藏窗口，Job Object 跟踪）并等待就绪
// （≤uocr2StartWait，默认 120s——python 首次含 torch 导入+模型加载；llama Q8
// 实测 ~10s）。任何失败路径都在返回 false 前杀整棵进程树，保证不残留孤儿。
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

// startUocr2Server 拉起 python 后端（server.py）。
func startUocr2Server(c *http.Client, base string) bool {
	return startTrackedServer(c, base, func() (*exec.Cmd, *os.File, bool) { return uocr2BuildCmd() })
}

// startUocr2LlamaServer 拉起 llama 后端（llama-server）。
func startUocr2LlamaServer(c *http.Client, base string) bool {
	return startTrackedServer(c, base, func() (*exec.Cmd, *os.File, bool) { return uocr2LlamaBuildCmd() })
}

// uocr2StartWait 是 startUocr2Server 等待服务就绪的时长（默认 120s：CPU 冷启动
// 含 torch 导入+模型加载约 20s，留足裕量）；包级变量便于单测缩短等待。
var uocr2StartWait = 120 * time.Second

// uocr2Healthy 是 startUocr2Server 轮询的就绪探针；包级变量便于单测注入 fake。
var uocr2Healthy = uocr2ServerHealthy

// uocr2BuildCmd 构造 python 服务命令；包级变量便于单测注入假进程。
var uocr2BuildCmd = buildUocr2ServerCmd

// uocr2LlamaBuildCmd 构造 llama 服务命令；包级变量便于单测注入假进程。
var uocr2LlamaBuildCmd = buildUocr2LlamaServerCmd

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

// buildUocr2LlamaServerCmd 按 GAEA_UOCR2_LLAMA_* 配置构造 llama-server 命令与其
// 日志文件。返回 ok=false 表示工件缺失（bin\llama-server.exe / 模型 / mmproj）。
//
// 启动口径逐字来自 2026-10-29 实弹验证（C:\AI\uocr2-llama\README.md）：
//   - --ubatch-size 4096 = 铁律二：默认 512+flash-attn 下多图请求丢前面的图
//     （实测 2 页只转写第 2 页且不报错；-fa 0 亦可修复但更慢），4096 保证图片
//     嵌入块不被 ubatch 切割，勿删；
//   - -c 65536 覆盖 5 页窗（实测 15 页 prompt≈11.5k+生成 8.4k）；-np 1 单序列
//     （unlimited-ocr2 补丁的多序列路径未验证）；-ngl 99 无 GPU 时 llama-server
//     自行忽略并回退 CPU（Q8 CPU 实测 ~20s/页，仍 4 倍于 python）。
func buildUocr2LlamaServerCmd() (*exec.Cmd, *os.File, bool) {
	dir := os.Getenv("GAEA_UOCR2_LLAMA_DIR")
	if dir == "" {
		dir = `C:\AI\uocr2-llama`
	}
	model := os.Getenv("GAEA_UOCR2_LLAMA_MODEL")
	if model == "" {
		model = "uocr2-Q8_0.gguf"
	}
	mmproj := os.Getenv("GAEA_UOCR2_LLAMA_MMPROJ")
	if mmproj == "" {
		mmproj = "mmproj-uocr2-F16.gguf"
	}
	exe := filepath.Join(dir, "bin", "llama-server.exe")
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

// uocr2LlamaChat 调 llama-server 的 /v1/chat/completions 识别一页或多页，
// 返回模型原始输出（含结构标记，单/多页共用；多页由调用方按 <PAGE> 拆分）。
// finish_reason=length 视为截断 fail-closed（对齐 python server 422 口径）；
// 其余非 200 一律报错（fail-closed，由引擎链/编排层回退处理）。
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
	// llama 实测 2~13s/请求（单页~6s、15 页 106s）；沿用批量超时公式（180s+120s/页）
	// 留足裕量，冷路径（无 GPU 回退 CPU ~20s/页）同样覆盖。
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

// uocr2LlamaPageOCR llama 后端单页识别（输出清洗与 python 后端共用 uocr2CleanText）。
func uocr2LlamaPageOCR(base, pngPath string) (string, error) {
	raw, err := uocr2LlamaChat(base, []string{pngPath})
	if err != nil {
		return "", err
	}
	return uocr2CleanText(raw), nil
}

// uocr2LlamaPagesOCR llama 后端多页批识别：一次 chat 请求多图（SR-SWA 跨页上下文
// 在服务端共享），输出按 <PAGE> 拆分。页数对不上=生成缺页 fail-closed（编排层
// 整窗回退逐页，与 python /ocr_batch 的 split_pages 同口径）。
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

// uocr2OCRProvider 常驻 Unlimited-OCR-2 服务 OCR 提供者：首次使用按后端腿序
// （llama→python）探测/拉起并缓存 base URL 与后端形态；探测失败进入短冷却
// （避免逐页重复拉起）。不可用即返回错误。
type uocr2OCRProvider struct {
	mu       sync.Mutex
	base     string           // 已解析的常驻服务 base URL；空 = 未解析/不可用
	backend  uocr2BackendKind // 与 base 一同缓存，决定单/批识别走哪套协议
	lastFail time.Time
}

func (p *uocr2OCRProvider) Name() string { return OCRKindUocr2 }

// Available 预探测提供者是否可用（解析并缓存常驻服务地址；失败进入冷却）。
func (p *uocr2OCRProvider) Available() bool { base, _, _ := p.resolveBase(); return base != "" }

// ExtractImage 识别单张图片；服务不可用/配置错误返回错误（fail-closed）。
func (p *uocr2OCRProvider) ExtractImage(path string) (string, error) {
	base, kind, rerr := p.resolveBase()
	if rerr != nil {
		return "", rerr
	}
	if base == "" {
		return "", fmt.Errorf("Unlimited-OCR-2 服务不可用（llama/python 后端均未安装或无法拉起）")
	}
	var text string
	var err error
	if kind == uocr2BackendLlama {
		text, err = uocr2LlamaPageOCR(base, path)
	} else {
		text, err = uocr2PageOCR(base, path)
	}
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
	base, kind, rerr := p.resolveBase()
	if rerr != nil {
		return nil, rerr
	}
	if base == "" {
		return nil, fmt.Errorf("Unlimited-OCR-2 服务不可用（llama/python 后端均未安装或无法拉起）")
	}
	var texts []string
	var err error
	if kind == uocr2BackendLlama {
		texts, err = uocr2LlamaPagesOCR(base, paths)
	} else {
		texts, err = uocr2PagesOCR(base, paths)
	}
	if err != nil {
		p.dropBaseIfTransportErr(err)
		return nil, err
	}
	return texts, nil
}

// resolveBase 解析常驻服务（base+后端形态）：成功缓存，失败记录冷却时间。
// 配置错误（未知 GAEA_UOCR2_BACKEND）不缓存直接透传——每次调用都 fail-closed 报错。
func (p *uocr2OCRProvider) resolveBase() (string, uocr2BackendKind, error) {
	p.mu.Lock()
	if p.base != "" {
		base, kind := p.base, p.backend
		p.mu.Unlock()
		return base, kind, nil
	}
	if time.Since(p.lastFail) < ovisProbeCooldown {
		p.mu.Unlock()
		return "", "", nil
	}
	p.mu.Unlock()

	base, kind, err := uocr2ServerBase() // 探测/拉起（注入点可控，测试不依赖真实服务）
	p.mu.Lock()
	if err == nil && base != "" {
		p.base, p.backend = base, kind
	} else if err == nil {
		p.lastFail = time.Now()
	}
	p.mu.Unlock()
	return base, kind, err
}

func init() {
	RegisterOCRProvider(OCRKindUocr2, func() OCRProvider { return &uocr2OCRProvider{} })
}
