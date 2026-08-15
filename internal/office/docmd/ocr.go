package docmd

// ocr.go — 扫描件 PDF 的 OCR 编排实现（docmd.go 拆分）。
// 职责：常驻 OvisOCR2 llama-server 的探测/拉起/健康检查与单页识别
// （ovisServerBase/startOvisServer/ovisPageOCR）、tesseract 回退，以及
// ocrPDFRange 的 pdftoppm 渲染→逐页识别流水线；单图识别入口 OCRImageText。

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gaea/gaea/internal/gaea/proc"
)

// ovisOCRPrompt 是 OvisOCR2 文档解析的固定提示词（与 pdf 技能保持一致）。
const ovisOCRPrompt = "Extract all readable content from the image in natural human reading order and output the result as a single Markdown document. Preserve the original text without translation."

// ovisServerBase 返回常驻 OvisOCR2 llama-server 的 base URL；未安装/无法拉起时返回 ""，
// 调用方退回 tesseract。优先复用已在跑的实例（pdf 技能可能已拉起），否则按需静默拉起一次。
func ovisServerBase() string {
	base := strings.TrimRight(os.Getenv("GAEA_OCR_URL"), "/")
	if base == "" {
		port := os.Getenv("GAEA_OCR_PORT")
		if port == "" {
			port = "8137"
		}
		base = "http://127.0.0.1:" + port
	}
	client := &http.Client{Timeout: 3 * time.Second}
	if ovisServerHealthy(client, base) {
		return base
	}
	if startOvisServer(client, base) {
		return base
	}
	return ""
}

func ovisServerHealthy(c *http.Client, base string) bool {
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

// startOvisServer 按 GAEA_OCR_* 环境变量或默认 C:\AI\gaea-ocr 拉起 llama-server
// （隐藏窗口，Vulkan，Job Object 跟踪）。等待就绪（≤ovisStartWait，默认 60s）。
// 启动成功返回 true 且进程保留存活（常驻服务继续使用）；任何失败路径（配置缺失、
// 启动失败、超时未就绪）都会在返回 false 前用 proc.KillTracked 杀掉整棵进程树
// （含子进程），保证不残留孤儿 llama-server。
func startOvisServer(c *http.Client, base string) bool {
	cmd, logf, ok := ovisBuildCmd()
	if !ok {
		return false
	}
	proc.HideWindow(cmd)
	handle, err := proc.StartTracked(cmd)
	if err != nil {
		if logf != nil {
			logf.Close()
		}
		return false
	}
	if logf != nil {
		logf.Close() // 子进程已持有文件句柄，父进程可立即释放
	}
	deadline := time.Now().Add(ovisStartWait)
	for time.Now().Before(deadline) {
		time.Sleep(time.Second)
		if ovisHealthy(c, base) {
			go func() { _ = cmd.Wait() }() // 保留进程存活，异步回收
			return true
		}
	}
	proc.KillTracked(cmd, handle) // 超时：杀整棵进程树，防孤儿残留
	_ = cmd.Wait()                // 同步回收，返回 false 前确保进程已终止
	return false
}

// ovisStartWait 是 startOvisServer 等待 llama-server 就绪的时长（默认 60s）；
// 包级变量便于单测缩短等待。
var ovisStartWait = 60 * time.Second

// ovisHealthy 是 startOvisServer 轮询的就绪探针（默认 ovisServerHealthy）；
// 包级变量便于单测注入「永不健康/立即健康」的 fake。
var ovisHealthy = ovisServerHealthy

// ovisBuildCmd 构造 llama-server 命令（默认 buildOvisServerCmd）；包级变量便于
// 单测注入假进程。
var ovisBuildCmd = buildOvisServerCmd

// buildOvisServerCmd 按 GAEA_OCR_* 配置构造 llama-server 命令与其日志文件（打不开时
// 为 nil，调用方在子进程启动后负责 Close）。返回 ok=false 表示配置缺失（exe/模型/
// 投影文件不存在）。
func buildOvisServerCmd() (*exec.Cmd, *os.File, bool) {
	dir := os.Getenv("GAEA_OCR_DIR")
	if dir == "" {
		dir = `C:\AI\gaea-ocr`
	}
	exe := os.Getenv("GAEA_OCR_LLAMA")
	if exe == "" {
		exe = filepath.Join(dir, "llama", "llama-server.exe")
	}
	model := os.Getenv("GAEA_OCR_MODEL")
	if model == "" {
		model = filepath.Join(dir, "models", "OvisOCR2-Q5_K_M.gguf")
	}
	mmproj := os.Getenv("GAEA_OCR_MMPROJ")
	if mmproj == "" {
		mmproj = filepath.Join(dir, "models", "mmproj-F16.gguf")
	}
	for _, p := range []string{exe, model, mmproj} {
		if _, err := os.Stat(p); err != nil {
			return nil, nil, false
		}
	}
	port := os.Getenv("GAEA_OCR_PORT")
	if port == "" {
		port = "8137"
	}
	var logf *os.File
	if f, err := os.OpenFile(filepath.Join(dir, "llama-server.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err == nil {
		logf = f
	}
	cmd := exec.Command(exe, "-m", model, "--mmproj", mmproj, "--port", port,
		"-c", "8192", "-ngl", "99", "--jinja", "--host", "127.0.0.1")
	if logf != nil {
		cmd.Stdout = logf
		cmd.Stderr = logf
	}
	return cmd, logf, true
}

// ovisOCRPage 是单页 OvisOCR2 识别函数（包级变量便于单测注入 fake；默认 ovisPageOCR）。
var ovisOCRPage = ovisPageOCR

// ovisPageOCR 把一页 PNG 发给常驻 OvisOCR2 服务，返回识别文本。
func ovisPageOCR(base, pngPath string) (string, error) {
	data, err := os.ReadFile(pngPath)
	if err != nil {
		return "", err
	}
	payload := map[string]any{
		"model": "ovis",
		"messages": []map[string]any{{
			"role": "user",
			"content": []map[string]any{
				{"type": "text", "text": ovisOCRPrompt},
				{"type": "image_url", "image_url": map[string]string{
					"url": "data:image/png;base64," + base64.StdEncoding.EncodeToString(data),
				}},
			},
		}},
		"temperature": 0.0,
		"max_tokens":  4096,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	client := &http.Client{Timeout: 180 * time.Second}
	resp, err := client.Post(base+"/v1/chat/completions", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OvisOCR2 服务返回 %d", resp.StatusCode)
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&out); err != nil {
		return "", err
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("OvisOCR2 无返回")
	}
	// finish_reason=length 说明输出被 max_tokens 截断：不把残缺文本当完整结果，
	// 返回错误由上层按单页失败处理（跳过该页，避免静默丢内容）。
	if out.Choices[0].FinishReason == "length" {
		return "", fmt.Errorf("OvisOCR2 输出被 max_tokens 截断（finish_reason=length）")
	}
	return strings.TrimSpace(out.Choices[0].Message.Content), nil
}

// tesseractImagePath 用 tesseract 识别单张图片：tesseract <img> stdout
// -l chi_sim+eng --psm 3（隐藏窗口，与 ocrPDFRange 流水线同一参数），
// 返回去首尾空白的识别文本。
func tesseractImagePath(tesseractPath, imagePath string) (string, error) {
	cmd := exec.Command(tesseractPath, imagePath, "stdout", "-l", "chi_sim+eng", "--psm", "3")
	proc.HideWindow(cmd)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// tesseractLookPath 定位 tesseract 可执行文件；tesseractImage 执行单图识别。
// 包级变量以便单测注入假路径/假结果，不依赖真实 tesseract 安装。
var (
	tesseractLookPath = exec.LookPath
	tesseractImage    = tesseractImagePath
)

// ── OCR Provider Seam（Step 3c） ──────────────────────────────────────────
//
// seam 三元组（定义/提供者/消费者，范式见 internal/ai/image_backend.go 的
// Register/New/Kinds）：
//   - 定义：OCRProvider 接口（Name + ExtractImage）
//   - 提供者：ovis（常驻 OvisOCR2 llama-server）、tesseract（命令行），各自
//     init() 自注册，互斥注册（重复即 panic）
//   - 消费者：OCRImageText / ocrPDFRange 只依赖接口，引擎顺序由配置
//     GAEA_OCR_ENGINE 驱动（"auto"=ovis→tesseract 自动回退，保持现状；
//     显式指定 = 仅该引擎，不可用即 fail-closed 报错，不静默降级）
//
// 验收：切换 OCR 引擎只改配置项（GAEA_OCR_ENGINE），代码零改动。

// OCRProvider 单图 OCR 提供者（seam 定义）。
type OCRProvider interface {
	// Name 返回提供者 kind（"ovis" / "tesseract"）。
	Name() string
	// ExtractImage 识别单张图片，返回去首尾空白的文本。
	// 引擎不可用（未安装/无法拉起）必须返回错误（fail-closed），不得静默返回空。
	ExtractImage(path string) (string, error)
}

// OCR 引擎 kind 常量（代码与配置只依赖 kind；具体实现由各文件 init() 自注册）。
const (
	// OCRKindOvis 常驻 OvisOCR2 llama-server（本地推荐，见 pdf 技能）。
	OCRKindOvis = "ovis"
	// OCRKindTesseract tesseract 命令行 OCR（chi_sim+eng）。
	OCRKindTesseract = "tesseract"
)

// ocrProviderFactory 按实例构造 OCR 提供者（无额外配置，环境变量驱动）。
type ocrProviderFactory func() OCRProvider

// ocrProviderRegistry kind → 工厂注册表（互斥注册，重复即 panic）。
var ocrProviderRegistry = map[string]ocrProviderFactory{}

// RegisterOCRProvider 注册 OCR 引擎 kind（如 "ovis" / "tesseract"）。
// 供各引擎 init() 自注册；kind 为空或重复注册直接 panic（编译期接线错误）。
func RegisterOCRProvider(kind string, factory ocrProviderFactory) {
	if kind == "" {
		panic("docmd: ocr provider kind must not be empty")
	}
	if _, dup := ocrProviderRegistry[kind]; dup {
		panic("docmd: duplicate ocr provider kind " + kind)
	}
	ocrProviderRegistry[kind] = factory
}

// NewOCRProvider 经注册表构造 OCR 提供者；未知 kind 返回错误
// （附已注册 kind 列表，fail-closed 不静默降级）。
func NewOCRProvider(kind string) (OCRProvider, error) {
	factory, ok := ocrProviderRegistry[kind]
	if !ok {
		return nil, fmt.Errorf("docmd: unknown ocr provider kind %q (registered: %v)", kind, OCRProviderKinds())
	}
	p := factory()
	if p == nil {
		return nil, fmt.Errorf("docmd: ocr provider factory %q returned nil", kind)
	}
	return p, nil
}

// OCRProviderKinds 返回已注册 OCR 引擎 kind 列表（排序，供诊断/校验）。
func OCRProviderKinds() []string {
	out := make([]string, 0, len(ocrProviderRegistry))
	for k := range ocrProviderRegistry {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// ocrEngineOrder 由配置解析 OCR 引擎顺序：
// GAEA_OCR_ENGINE 为空或 "auto" → [ovis, tesseract]（自动回退链，保持现状）；
// 显式指定 → 单引擎（未知 kind 报错，fail-closed 不静默回退到 auto）。
func ocrEngineOrder() ([]string, error) {
	raw := strings.TrimSpace(os.Getenv("GAEA_OCR_ENGINE"))
	if raw == "" || strings.EqualFold(raw, "auto") {
		return []string{OCRKindOvis, OCRKindTesseract}, nil
	}
	kind := strings.ToLower(raw)
	for _, k := range OCRProviderKinds() {
		if k == kind {
			return []string{kind}, nil
		}
	}
	return nil, fmt.Errorf("docmd: 未知 OCR 引擎 %q（可用：auto / %s）", raw, strings.Join(OCRProviderKinds(), " / "))
}

// ovisOCRProvider 常驻 OvisOCR2 llama-server OCR 提供者：首次使用探测/拉起并
// 缓存 base URL；探测失败进入短冷却（避免逐页重复拉起）。不可用即返回错误。
type ovisOCRProvider struct {
	mu       sync.Mutex
	base     string // 已解析的常驻服务 base URL；空 = 未解析/不可用
	lastFail time.Time
}

// ovisProbeCooldown 探测失败后的冷却时长：冷却期内不再重复探测/拉起服务。
const ovisProbeCooldown = 5 * time.Second

func (p *ovisOCRProvider) Name() string { return OCRKindOvis }

// Available 预探测提供者是否可用（解析并缓存常驻服务地址；失败进入冷却）。
func (p *ovisOCRProvider) Available() bool { return p.resolveBase() != "" }

// ExtractImage 识别单张图片；服务不可用返回错误（fail-closed）。
func (p *ovisOCRProvider) ExtractImage(path string) (string, error) {
	base := p.resolveBase()
	if base == "" {
		return "", fmt.Errorf("OvisOCR2 服务不可用（未安装或无法拉起）")
	}
	text, err := ovisOCRPage(base, path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(text), nil
}

// resolveBase 解析常驻服务 base URL：成功缓存，失败记录冷却时间。
func (p *ovisOCRProvider) resolveBase() string {
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

	base := ovisServerBase() // 探测/拉起（注入点可控，测试不依赖真实服务）
	p.mu.Lock()
	if base != "" {
		p.base = base
	} else {
		p.lastFail = time.Now()
	}
	p.mu.Unlock()
	return base
}

// tesseractOCRProvider tesseract 命令行 OCR 提供者（未安装即 fail-closed 报错）。
type tesseractOCRProvider struct{}

func (p *tesseractOCRProvider) Name() string { return OCRKindTesseract }

// Available 预探测 tesseract 可执行文件是否存在。
func (p *tesseractOCRProvider) Available() bool {
	_, err := tesseractLookPath("tesseract")
	return err == nil
}

// ExtractImage 用 tesseract 识别单张图片（chi_sim+eng，--psm 3）。
func (p *tesseractOCRProvider) ExtractImage(path string) (string, error) {
	tess, err := tesseractLookPath("tesseract")
	if err != nil {
		return "", fmt.Errorf("tesseract 未安装（%v）", err)
	}
	text, err := tesseractImage(tess, path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(text), nil
}

func init() {
	RegisterOCRProvider(OCRKindOvis, func() OCRProvider { return &ovisOCRProvider{} })
	RegisterOCRProvider(OCRKindTesseract, func() OCRProvider { return &tesseractOCRProvider{} })
}

// ocrAnyAvailable 预探测提供者链中是否至少一个引擎可用（ovis 解析并缓存常驻
// 服务；tesseract 检查可执行文件）。用于 ocrPDFRange 渲染前的提前失败。
func ocrAnyAvailable(providers []OCRProvider) bool {
	for _, p := range providers {
		if checker, ok := p.(interface{ Available() bool }); ok && checker.Available() {
			return true
		}
	}
	return false
}

// ocrUnavailableError 生成引擎链不可用错误：auto 链保留原文案（含安装提示），
// 显式单引擎给出针对性文案。
func ocrUnavailableError(kinds []string) error {
	if len(kinds) == 1 {
		return fmt.Errorf("OCR 引擎 %q 不可用（未安装或无法运行）。请安装该引擎或检查 GAEA_OCR_ENGINE 配置。", kinds[0])
	}
	return fmt.Errorf("OvisOCR2 本地 OCR 不可用（未安装或服务无法拉起），且未找到 tesseract。" +
		"请安装其一：\n  - OvisOCR2（本地推荐，见 pdf 技能：C:\\AI\\gaea-ocr 或 GAEA_OCR_DIR）\n" +
		"  - tesseract: https://github.com/tesseract-ocr/tesseract")
}

// ocrPDFUnavailableError 生成扫描件 PDF 的引擎不可用错误：auto 链保留原文案
// （含安装提示），显式单引擎给出针对性文案。
func ocrPDFUnavailableError(kinds []string) error {
	if len(kinds) == 1 {
		return fmt.Errorf("扫描件 PDF 需要 OCR 引擎 %q，但该引擎不可用（未安装或无法运行）。"+
			"请检查 GAEA_OCR_ENGINE 配置或安装该引擎，或使用文本 PDF（非扫描件）。", kinds[0])
	}
	return fmt.Errorf("扫描件 PDF 需要 OCR 引擎，但未找到 OvisOCR2 或 tesseract。" +
		"请安装其一：\n  - OvisOCR2（本地推荐，见 pdf 技能：C:\\AI\\gaea-ocr）\n" +
		"  - tesseract: https://github.com/tesseract-ocr/tesseract\n\n" +
		"或者使用文本 PDF（非扫描件）")
}

// OCRImageText 识别单张图片中的文字。引擎顺序由 GAEA_OCR_ENGINE 配置驱动：
// "auto"（默认）= 常驻 OvisOCR2 服务优先，不可用时降级 tesseract（保持现状）；
// 显式指定（ovis/tesseract）= 仅该引擎，不可用即报错（fail-closed，不静默降级）。
// 引擎顺序与降级行为由测试固化（见 ocr_seam_test.go / ocr_test.go）。
func OCRImageText(path string) (string, error) {
	kinds, err := ocrEngineOrder()
	if err != nil {
		return "", err
	}
	for _, kind := range kinds {
		p, perr := NewOCRProvider(kind)
		if perr != nil {
			continue // kind 已校验，防御性跳过
		}
		if text, err := p.ExtractImage(path); err == nil && strings.TrimSpace(text) != "" {
			return text, nil
		}
	}
	return "", ocrUnavailableError(kinds)
}

// ocrPDF 处理扫描件 PDF：本地 OvisOCR2（常驻 llama-server）优先，
// 未安装/不可用时退回 tesseract（pdftoppm → tesseract 流水线）。
func ocrPDFRange(path, pages string, first, last, total int, progress func(done, totalN int)) (string, error) {
	pdftoppmPath := findPdftoppm()
	if pdftoppmPath == "" {
		return "", fmt.Errorf("扫描件 PDF 需要 poppler 渲染（pdftoppm），但未找到。" +
			"请安装 poppler：https://poppler.freedesktop.org\n\n或者使用文本 PDF（非扫描件）")
	}
	// 引擎链由 GAEA_OCR_ENGINE 配置驱动（"auto" = ovis → tesseract，保持现状；
	// 显式指定 = 仅该引擎）。构造提供者实例后预探测一次：至少一个引擎可用才继续
	// （ovis 探测并缓存常驻服务地址，失败整轮跳过、不逐页重试拉起；tesseract
	// 检查可执行文件），与历史提前失败行为一致。
	kinds, err := ocrEngineOrder()
	if err != nil {
		return "", err
	}
	providers := make([]OCRProvider, 0, len(kinds))
	for _, kind := range kinds {
		if p, perr := NewOCRProvider(kind); perr == nil {
			providers = append(providers, p)
		}
	}
	if !ocrAnyAvailable(providers) {
		return "", ocrPDFUnavailableError(kinds)
	}

	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "gaea-ocr-*")
	if err != nil {
		return "", fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// pdftoppm: PDF → PNG（逐页）
	pngPrefix := filepath.Join(tmpDir, "page")
	args := []string{"-png", "-r", "300"}
	if first != 1 {
		args = append(args, "-f", strconv.Itoa(first))
	}
	if last < total {
		args = append(args, "-l", strconv.Itoa(last))
	}
	args = append(args, path, pngPrefix)
	cmd := exec.Command(pdftoppmPath, args...)
	proc.HideWindow(cmd) // Windows: 防止弹出 cmd 黑框
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("pdftoppm 执行失败: %w\n输出: %s", err, string(out))
	}

	// 收集生成的 PNG 文件并按页码排序
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		return "", fmt.Errorf("读取临时目录失败: %w", err)
	}
	var pngFiles []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".png") {
			pngFiles = append(pngFiles, filepath.Join(tmpDir, e.Name()))
		}
	}
	if len(pngFiles) == 0 {
		return "", fmt.Errorf("pdftoppm 未生成页面图片")
	}

	// 逐页 OCR：pdftoppm 从 first 页起渲染，PNG 按序对应绝对页码 first+i。
	// 页范围过滤用绝对页码（与文本路径一致），避免 pdftoppm 偏移后的错位。
	// 单页失败跳过继续（失败页计入 failed 列表，在结果尾部注明），全部失败才报错。
	pageOCR := func(pageNum int, pngPath string) (string, error) {
		// 逐引擎尝试（auto 链 = OvisOCR2 常驻服务优先，失败/截断退回 tesseract）；
		// 单页全部引擎失败由 ocrPageLoop 跳过继续（逐页容错，测试固化）。
		for _, p := range providers {
			if t, err := p.ExtractImage(pngPath); err == nil && strings.TrimSpace(t) != "" {
				return t, nil
			}
		}
		return "", fmt.Errorf("无可用 OCR 引擎")
	}
	pageTexts, _, failed, err := ocrPageLoop(pngFiles, first, pages, pageOCR, progress)
	if err != nil {
		return "", err
	}
	if len(failed) > 0 {
		pageTexts = append(pageTexts, fmt.Sprintf("（第 %v 页 OCR 失败已跳过，共 %d 页）", failed, len(failed)))
	}
	result := strings.Join(pageTexts, "\n\n---\n\n")
	return fmt.Sprintf("（以下内容由 OCR 识别，可能存在误差）\n\n%s", result), nil
}

// ocrPageLoop 逐页调用 pageOCR 识别（单页失败跳过继续，全部失败才报错）。
// 返回识别文本列表、实际 OCR 页数（按页范围过滤后，即 progress 的 total）、
// 失败页码列表与错误（仅全部失败时非 nil）。progress 回调 (done, total)，
// total 为实际 OCR 页数而非渲染页数，保证进度条反映真实工作量。
func ocrPageLoop(pngFiles []string, first int, pages string,
	pageOCR func(pageNum int, pngPath string) (string, error),
	progress func(done, total int)) ([]string, int, []int, error) {

	type pageJob struct {
		num int
		png string
	}
	var jobs []pageJob
	for i, pngPath := range pngFiles {
		pageNum := first + i
		if pages != "" && !pageInRange(pageNum, pages) {
			continue
		}
		jobs = append(jobs, pageJob{num: pageNum, png: pngPath})
	}
	total := len(jobs)
	if total == 0 {
		return nil, 0, nil, fmt.Errorf("没有需要 OCR 的页面")
	}
	var texts []string
	var failed []int
	done := 0
	for _, j := range jobs {
		text, err := pageOCR(j.num, j.png)
		if err != nil || strings.TrimSpace(text) == "" {
			failed = append(failed, j.num)
		} else {
			texts = append(texts, text)
		}
		done++
		if progress != nil {
			progress(done, total)
		}
	}
	if len(texts) == 0 {
		if len(failed) > 0 {
			return nil, total, failed, fmt.Errorf("OCR 全部 %d 页失败（第 %v 页），未能提取到任何文本",
				total, failed)
		}
		return nil, total, nil, fmt.Errorf("OCR 未能提取到任何文本")
	}
	return texts, total, failed, nil
}

// findPdftoppm 探测可用的 pdftoppm：优先 GAEA_PDFTOPM 显式路径，其次 PATH 里的
// pdftoppm.exe，再回退到 codex 运行时自带的 poppler（本机 PATH 里的 .cmd 包装器
// 指向不存在的路径，直接执行会失败）。
func findPdftoppm() string {
	if p := os.Getenv("GAEA_PDFTOPM"); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	if p, err := exec.LookPath("pdftoppm"); err == nil && strings.EqualFold(filepath.Ext(p), ".exe") {
		return p
	}
	base := filepath.Join(os.Getenv("USERPROFILE"), ".cache", "codex-runtimes")
	matches, _ := filepath.Glob(filepath.Join(base, "*", "dependencies", "native", "poppler", "Library", "bin", "pdftoppm.exe"))
	if len(matches) > 0 {
		return matches[0]
	}
	return ""
}
