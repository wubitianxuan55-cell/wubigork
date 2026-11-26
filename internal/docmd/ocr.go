package docmd

// ocr.go — 扫描件 PDF 的 OCR 编排实现（docmd.go 拆分）。
// 职责：OCR provider seam（注册表 / 引擎选择 / 错误文案）、单图识别入口 OCRImageText，
// 以及 ocrPDFRange 的 pdftoppm 渲染→逐页/分窗批识别流水线。
// 本地唯一引擎是 Unlimited-OCR-2（ocr_uocr2.go）；OvisOCR2、tesseract 与 uocr2 的 python
// 后端已于 2026-11-26 退役（用户拍板：WPS 优先后本地 OCR 只做兜底，留一层就够）。

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gaea/gaea/internal/gaea/proc"
)

// ── OCR Provider Seam（Step 3c） ──────────────────────────────────────────
//
// seam 三元组（定义/提供者/消费者，范式见 internal/ai/image_backend.go 的
// Register/New/Kinds）：
//   - 定义：OCRProvider 接口（Name + ExtractImage）+ 可选的 PageBatchOCRProvider；
//   - 提供者：uocr2（常驻 Unlimited-OCR-2 llama-server，ocr_uocr2.go 的 init() 自注册）；
//   - 消费者：OCRImageText / ocrPDFRange 只依赖接口，引擎由配置 GAEA_OCR_ENGINE 选择。
//
// 2026-11-26 精简（用户拍板「本地 OCR 只留最好的一个」）：本地引擎只保留 Unlimited-OCR-2
// 的 llama 后端。此前是 uocr2→ovis→tesseract 三层链 + uocr2 的 python 后端逐层回退；
// WPS 优先后本地 OCR 只做兜底，多层兜底失去存在理由（OvisOCR2 单页约慢 5 倍、python
// 后端慢 10~14 倍、tesseract 本机未装）。seam 本身保留：引擎可插拔、测试可注入替身。
//
// ── 口径边界（审计 U54/IN3-09 权威口径，改动前必读） ─────────────────────
//
// 本 seam 的 kind 取值域 = 本地 OCR 引擎（uocr2），消费方三组都是「本地、离线、
// 文件不出机」用例：扫描件 PDF（ocrPDFRange）、成本库报价单图片（app visionOCRImage →
// OCRImageText）、办公提取文字兜底腿（app GaeaOCRText 的本地腿）。远端 herdsman 链
// （internal/ocr 的 PaddleOCR /v1/ocr 与 MinerU /v1/documents/parse，由 app/gaea_ocr.go
// 编排）刻意不注册为本 seam 的 provider kind（取值域钉子：ocr_kind_domain_test.go）：
//   - 接入即改道：herdsman 进链后上述本地用例的流量就被远端接管（结果/延迟/隐私面全变）；
//   - 装不进 provider 形状：其 URL/模型源是引擎配置 BaseURL + 模型中心
//     active_ocr_engine 绑定 + HERDSMAN_* env，本包无参 factory 解析不到；
//     MinerU 是文档解析（mode/dpi/formula），不是单图 OCR。
//   - 探测模型不同构：本 seam 按需拉起进程 + /health 探测 + 冷却；herdsman
//     外部常驻、无探测，HTTP 即错误。
// 两链是分工不是重复，配置名词同名不同物（勿「顺手统一」）：
//   GAEA_OCR_ENGINE — 本 seam 引擎 kind（空 / auto / uocr2），本文件唯一解析点；
//   active_ocr_engine/active_ocr_model — 模型中心「设为 OCR」绑定（引擎/模型
//     ID 域），只影响 app GaeaOCRText 的指定引擎腿，本 seam 不读它；
//   GAEA_UOCR2_LLAMA_URL/_PORT/_DIR/_MODEL/_MMPROJ — llama 后端工件与地址
//     （默认 8139 / C:\AI\gaea-OCR\uocr2-llama）；
//   HERDSMAN_OCR_MODEL/HERDSMAN_PARSE_MODEL — 远端模型 ID。
// 已退役、不再读的配置名（GAEA_OCR_ENGINE 设成 ovis / tesseract 会如实报错，不静默换挡）：
// GAEA_OCR_URL/_PORT/_DIR/_LLAMA/_MODEL/_MMPROJ（OvisOCR2）、GAEA_UOCR2_BACKEND、
// GAEA_UOCR2_URL/_PORT/_DIR/_PY（python 后端）。

// OCRProvider 单图 OCR 提供者（seam 定义）。
type OCRProvider interface {
	// Name 返回提供者 kind（"uocr2"，或测试注册的替身 kind）。
	Name() string
	// ExtractImage 识别单张图片，返回去首尾空白的文本。
	// 引擎不可用（未安装/无法拉起）必须返回错误（fail-closed），不得静默返回空。
	ExtractImage(path string) (string, error)
}

// PageBatchOCRProvider 可选增强：整批多页识别（跨页上下文共享——uocr2 的 SR-SWA
// 多页解析可合并跨页表格/衔接正文，逐页识别做不到）。实现契约：
//   - 返回顺序与输入 paths 严格一致、等长；
//   - 任一页失败/输出对不上必须整体返回错误（fail-closed），调用方对该批回退逐页；
//   - 单次请求页数由调用方控制（uocr2BatchWindow 分窗，防超长请求）。
type PageBatchOCRProvider interface {
	OCRProvider
	ExtractPages(paths []string) ([]string, error)
}

// OCR 引擎 kind 常量（代码与配置只依赖 kind；具体实现由各文件 init() 自注册）。
const (
	// OCRKindUocr2 常驻 Unlimited-OCR-2 llama-server——本地唯一引擎（版式/表格还原
	// 最好；Q8 实测单页 6s、15 页 106s；无 GPU 时 llama-server 自行回退 CPU，约 20s/页）。
	OCRKindUocr2 = "uocr2"
)

// ocrProviderFactory 按实例构造 OCR 提供者（无额外配置，环境变量驱动）。
type ocrProviderFactory func() OCRProvider

// ocrProviderRegistry kind → 工厂注册表（互斥注册，重复即 panic）。
var ocrProviderRegistry = map[string]ocrProviderFactory{}

// RegisterOCRProvider 注册 OCR 引擎 kind（如 "uocr2"；测试可注册替身 kind）。
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

// ocrEngineOrder 由配置解析 OCR 引擎顺序：GAEA_OCR_ENGINE 为空、"auto" 或 "uocr2"
// → [uocr2]（本地唯一引擎，2026-11-26 精简；此前 auto = uocr2→ovis→tesseract）。
// 显式指定已退役的引擎（ovis / tesseract）或未知 kind → 报错，fail-closed 不静默换挡。
func ocrEngineOrder() ([]string, error) {
	raw := strings.TrimSpace(os.Getenv("GAEA_OCR_ENGINE"))
	if raw == "" || strings.EqualFold(raw, "auto") {
		return []string{OCRKindUocr2}, nil
	}
	kind := strings.ToLower(raw)
	if err := ocrRetiredKindError(kind); err != nil {
		return nil, err
	}
	for _, k := range OCRProviderKinds() {
		if k == kind {
			return []string{kind}, nil
		}
	}
	return nil, fmt.Errorf("docmd: 未知 OCR 引擎 %q（可用：auto / %s）", raw, strings.Join(OCRProviderKinds(), " / "))
}

// ocrRetiredKindError kind 是 2026-11-26 退役的引擎时返回指路的错误，否则 nil。
// 旧机器上残留的 GAEA_OCR_ENGINE=ovis 之类要一眼看出该改什么。
func ocrRetiredKindError(kind string) error {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "ovis":
		return fmt.Errorf("docmd: 本地 OCR 引擎 OvisOCR2 已于 2026-11-26 退役（本地只保留 Unlimited-OCR-2）；请把 GAEA_OCR_ENGINE 改成 uocr2 或删掉该配置")
	case "tesseract":
		return fmt.Errorf("docmd: 本地 OCR 引擎 tesseract 已于 2026-11-26 退役（本地只保留 Unlimited-OCR-2）；请把 GAEA_OCR_ENGINE 改成 uocr2 或删掉该配置")
	}
	return nil
}

// ocrAnyAvailable 预探测提供者链中是否至少一个引擎可用（uocr2 探测并缓存常驻
// 服务地址）。用于 ocrPDFRange 渲染前的提前失败。
func ocrAnyAvailable(providers []OCRProvider) bool {
	for _, p := range providers {
		if checker, ok := p.(interface{ Available() bool }); ok && checker.Available() {
			return true
		}
	}
	return false
}

// ocrUnavailableError 生成引擎不可用错误：本地唯一引擎是 Unlimited-OCR-2（llama 后端）；
// 显式指定的替身 kind（测试）给出针对性文案。
func ocrUnavailableError(kinds []string) error {
	if len(kinds) == 1 && kinds[0] != OCRKindUocr2 {
		return fmt.Errorf("OCR 引擎 %q 不可用（未安装或无法运行）。请安装该引擎或检查 GAEA_OCR_ENGINE 配置。", kinds[0])
	}
	return fmt.Errorf("本地 OCR 不可用（未安装或服务无法拉起）。请安装 Unlimited-OCR-2" +
		"（llama 后端，C:\\AI\\gaea-OCR\\uocr2-llama 或 GAEA_UOCR2_LLAMA_DIR）")
}

// ocrPDFUnavailableError 生成扫描件 PDF 的引擎不可用错误。
func ocrPDFUnavailableError(kinds []string) error {
	if len(kinds) == 1 && kinds[0] != OCRKindUocr2 {
		return fmt.Errorf("扫描件 PDF 需要 OCR 引擎 %q，但该引擎不可用（未安装或无法运行）。"+
			"请检查 GAEA_OCR_ENGINE 配置或安装该引擎，或使用文本 PDF（非扫描件）。", kinds[0])
	}
	return fmt.Errorf("扫描件 PDF 需要 OCR 引擎，但 Unlimited-OCR-2 不可用（未安装或服务无法拉起）。" +
		"请安装 Unlimited-OCR-2（llama 后端，C:\\AI\\gaea-OCR\\uocr2-llama），" +
		"或优先用 WPS 转换（format_convert 默认先走 WPS 在线 OCR）；或者使用文本 PDF（非扫描件）")
}

// OCRImageText 识别单张图片中的文字。引擎由 GAEA_OCR_ENGINE 配置驱动：空 / "auto" /
// "uocr2"（默认）= Unlimited-OCR-2（本地唯一引擎，见 ocrEngineOrder 注释）；显式指定
// 已退役或未知的 kind 直接报错（fail-closed，不静默换挡）。
// 引擎选择与错误行为由测试固化（见 ocr_seam_test.go / ocr_test.go）。
func OCRImageText(path string) (string, error) {
	kinds, err := ocrEngineOrder()
	if err != nil {
		return "", err
	}
	// 进程内缓存（见 ocrcache.go）：同一图片反复 OCR（@引用重发/多消费方）只跑
	// 一次引擎链；键含引擎序，GAEA_OCR_ENGINE 切换不串味。读不到文件时绕过
	// 缓存直接走链——引擎不可用错误文案是契约（含引擎名与安装提示），不能被
	// 文件缺失的 open 错误短路。
	fp, ferr := ocrFileFingerprint(path)
	if ferr != nil {
		return ocrImageTextUncached(path, kinds)
	}
	key := "img:" + fp + ":" + engineOrderKey()
	// 缓存命中即报 cached（大卡秒回态）：同文件重复 OCR（@引用重发/多消费方）
	// 不该再演一遍引擎拉起。
	if text, ok := defaultOCRCache.get(key); ok {
		emitOCRProgress(OCRProgressEvent{Path: path, Stage: OCRStageCached, Total: 1, Chars: utf8.RuneCountInString(text)})
		return text, nil
	}
	return ocrCached(key, func() (string, error) {
		return ocrImageTextUncached(path, kinds)
	})
}

// ocrImageTextUncached OCRImageText 的引擎链本体（缓存未命中时执行）。
func ocrImageTextUncached(path string, kinds []string) (string, error) {
	for _, kind := range kinds {
		p, perr := NewOCRProvider(kind)
		if perr != nil {
			continue // kind 已校验，防御性跳过
		}
		// 进度事件：引擎拉起（uocr2 冷启动可达分钟级，进度价值最大的阶段）。
		emitOCRProgress(OCRProgressEvent{Path: path, Engine: kind, Stage: OCRStageEngineStart, Total: 1})
		if text, err := p.ExtractImage(path); err == nil && strings.TrimSpace(text) != "" {
			emitOCRDone(path, kind, utf8.RuneCountInString(text))
			return text, nil
		}
	}
	unavailable := ocrUnavailableError(kinds)
	emitOCRProgress(OCRProgressEvent{Path: path, Stage: OCRStageFailed, Total: 1, Err: unavailable.Error()})
	return "", unavailable
}

// OCRPDFText 强制走扫描件 OCR 管线（忽略文本层）：供「文本层提取成功但内容
// 不可用」的兜底消费方调用（成本导入扫描件兜底等）——这种场景
// pdfToMarkdownLimit 的自动 OCR 已跑过或被稀疏文本层顶掉，直接渲染+OCR 拿
// 全文。maxPages<=0 取 DefaultMaxPDFPages。走进程内缓存，键与
// pdfToMarkdownLimit 的 OCR 分支同形（同文件同页规格命中互认，effSpec 由同一
// capPageSpec 收敛得出）。
func OCRPDFText(path string, maxPages int) (string, error) {
	if maxPages <= 0 {
		maxPages = DefaultMaxPDFPages
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	raw := string(data)
	totalPages := pdfTotalPages(raw, raw) // 两视图同串=计数必等，不触发口径告警
	effSpec, _, err := capPageSpec("", maxPages, totalPages)
	if err != nil {
		return "", err
	}
	first, last := pageBounds(effSpec, totalPages)
	sum := sha256.Sum256(data)
	key := "pdf:" + hex.EncodeToString(sum[:16]) + ":" + effSpec + ":" + engineOrderKey()
	return ocrCached(key, func() (string, error) {
		return ocrPDFRange(path, effSpec, first, last, totalPages, nil)
	})
}

// pdftoppmLookup 渲染前置查找 seam（包级变量以便测试堵住渲染路径；生产仍走
// findPdftoppm 三级查找：GAEA_PDFTOPM → PATH .exe → codex-runtimes poppler）。
var pdftoppmLookup = findPdftoppm

// ocrMaxPages 本地 OCR 单次识别的页数上限。本地 OCR 是 WPS 之后的兜底，且整条链不看
// 取消信号（渲染与引擎请求都是整段同步；GPU 约 7s/页、CPU 约 20s/页起）：超限直接
// 拒绝并指路，不闷头跑（2026-11-26 实弹：174 页扫描件同步 OCR 堵死 40 分钟直到重启）。
const ocrMaxPages = 30

// ocrPDFRange 处理扫描件 PDF：引擎由 GAEA_OCR_ENGINE 配置驱动（默认 Unlimited-OCR-2，
// 2026-11-26 精简后是本地唯一引擎，见 ocrEngineOrder 注释），pdftoppm 300dpi 全范围一次
// 渲染 → 逐页/分窗批识别（uocr2 实现 PageBatchOCRProvider）。构造提供者实例后预探测
// 一次：引擎可用才继续（uocr2 探测并缓存常驻服务地址，失败整轮跳过、不逐页重试拉起），
// 与历史提前失败行为一致。
func ocrPDFRange(path, pages string, first, last, total int, progress func(done, totalN int)) (text string, err error) {
	// 进度事件（OCR 大卡）：页粒度 running + 终态 done/failed。终态走 defer
	// 统一收口（错误路径多，逐处 emit 会漏）；engine 取链首可用引擎 kind。
	pagesInRange := last - first + 1
	if pagesInRange < 1 {
		pagesInRange = 1
	}
	ocrEngine := ""
	defer func() {
		if err != nil {
			emitOCRProgress(OCRProgressEvent{Path: path, Engine: ocrEngine, Stage: OCRStageFailed, Total: pagesInRange, Err: err.Error()})
			return
		}
		emitOCRProgress(OCRProgressEvent{Path: path, Engine: ocrEngine, Stage: OCRStageDone, Done: pagesInRange, Total: pagesInRange, Chars: utf8.RuneCountInString(text)})
	}()
	if pagesInRange > ocrMaxPages {
		return "", fmt.Errorf("本地 OCR 一次最多识别 %d 页，这份要识别 %d 页；请用 pages 参数分段识别",
			ocrMaxPages, pagesInRange)
	}
	pdftoppmPath := pdftoppmLookup()
	if pdftoppmPath == "" {
		return "", fmt.Errorf("扫描件 PDF 需要 poppler 渲染（pdftoppm），但未找到。" +
			"请安装 poppler：https://poppler.freedesktop.org\n\n或者使用文本 PDF（非扫描件）")
	}
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
	if len(providers) > 0 {
		ocrEngine = providers[0].Name()
	}
	emitOCRProgress(OCRProgressEvent{Path: path, Engine: ocrEngine, Stage: OCRStageEngineStart, Total: pagesInRange})

	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "gaea-ocr-*")
	if err != nil {
		return "", fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

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

	// 收集生成的 PNG 文件：页码唯一来源 = pdftoppm **实际产物**文件名后缀
	// （<prefix>-<n>.png，与 render.go 视觉 diff 渲染同一解析实现，IN3-11），
	// 不再用「first+i」假设每页都渲染成功、也不依赖目录读取顺序。
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
	renderedJobs, fromArtifacts := renderedPageJobs(pngFiles, first)
	// 不可解析 → 回退 first+i：告警由 ocrPageLoop 在同一判定点统一发出（避免重复日志）。
	if fromArtifacts {
		nums := make([]int, 0, len(renderedJobs))
		for _, j := range renderedJobs {
			nums = append(nums, j.num)
		}
		// 产物侧对账（IN3-11）：渲染页码不连续（pdftoppm 跳页/损坏页）或末页
		// 超出声明总页数时，文本路径页码与 OCR 路径页码会错位 → 诚实告警。
		if nums[0] != first || nums[len(nums)-1] > total || len(nums) != nums[len(nums)-1]-nums[0]+1 {
			slog.Warn("docmd: OCR 渲染产物页码与声明页数/连续性不一致（页码可能错位）",
				"path", path, "firstRendered", nums[0], "lastRendered", nums[len(nums)-1],
				"renderedPages", len(nums), "declaredTotal", total, "requestedFirst", first)
		}
	}

	// 逐页 OCR：页码取上一步的产物页码（绝对页码，与文本路径 pageInRange 同口径）。
	// 单页失败跳过继续（失败页计入 failed 列表，在结果尾部注明），全部失败才报错。
	// 引擎实现 PageBatchOCRProvider 时启用分窗批识别（跨页上下文，见
	// ocrPageLoopBatch）；批失败整窗回退逐页，容错语义不变。
	pageOCR := func(pageNum int, pngPath string) (string, error) {
		// 逐引擎尝试（引擎顺序见 ocrEngineOrder；默认只有 uocr2）；
		// 单页全部引擎失败由 ocrPageLoop 跳过继续（逐页容错，测试固化）。
		for _, p := range providers {
			if t, err := p.ExtractImage(pngPath); err == nil && strings.TrimSpace(t) != "" {
				return t, nil
			}
		}
		return "", fmt.Errorf("无可用 OCR 引擎")
	}
	var batchOCR func([]ocrPageJob) ([]string, error)
	if len(providers) > 0 {
		if bp, ok := providers[0].(PageBatchOCRProvider); ok {
			batchOCR = func(jobs []ocrPageJob) ([]string, error) {
				paths := make([]string, len(jobs))
				for i, j := range jobs {
					paths[i] = j.png
				}
				return bp.ExtractPages(paths)
			}
		}
	}
	pageTexts, _, failed, err := ocrPageLoopBatch(pngFiles, first, pages, pageOCR, batchOCR, progress)
	if err != nil {
		return "", err
	}
	if len(failed) > 0 {
		pageTexts = append(pageTexts, fmt.Sprintf("（第 %v 页 OCR 失败已跳过，共 %d 页）", failed, len(failed)))
	}
	result := strings.Join(pageTexts, "\n\n---\n\n")
	text = fmt.Sprintf("（以下内容由 OCR 识别，可能存在误差）\n\n%s", result)
	return text, nil
}

// ocrPageJob 一页待识别的渲染产物：num = 绝对页码（来自实际产物文件名）。
type ocrPageJob struct {
	num int
	png string
}

// renderedPageJobs 把 pdftoppm 产物路径列表转成按绝对页码升序的待识别页列表。
//
// IN3-11：页码唯一来源 = **实际产物**文件名后缀 <prefix>-<n>.png 的 n（与
// render.go 的 RenderPDFPages 共用 parseRenderedPageNumber）。全部文件名可解析时
// 使用产物页码并按数值排序 —— pdftoppm 跳过损坏页（例如只生成 -1/-2/-4）时页号
// 仍然正确，且 page-10.png 不会因字典序排在 page-2.png 之前。
//
// 仅当存在不可解析的文件名时整体回退「first+i 顺序编号」（历史口径；真实
// pdftoppm 产物不会出现这种名字，测试注入的合成名会走到这里），返回值
// fromArtifacts=false 供调用方诚实告警。
func renderedPageJobs(pngFiles []string, first int) (jobs []ocrPageJob, fromArtifacts bool) {
	if len(pngFiles) == 0 {
		return nil, false
	}
	type cand struct {
		num int
		png string
	}
	cands := make([]cand, 0, len(pngFiles))
	for _, p := range pngFiles {
		n, ok := parseRenderedPageNumber(filepath.Base(p))
		if !ok {
			fromArtifacts = false
			break
		}
		cands = append(cands, cand{num: n, png: p})
		fromArtifacts = true
	}
	if !fromArtifacts {
		out := make([]ocrPageJob, 0, len(pngFiles))
		for i, p := range pngFiles {
			out = append(out, ocrPageJob{num: first + i, png: p})
		}
		return out, false
	}
	sort.Slice(cands, func(i, j int) bool { return cands[i].num < cands[j].num })
	out := make([]ocrPageJob, 0, len(cands))
	for _, c := range cands {
		out = append(out, ocrPageJob{num: c.num, png: c.png})
	}
	return out, true
}

// ocrBatchWindow 批识别窗口（页/请求）：跨页上下文收益与单请求时长/失败重试
// 成本的折中（uocr2 CPU 约 1 分钟/页，5 页 ≈ 6-8 分钟/请求；官方 SGLang 默认
// 窗口 10 页，取其半控最坏重试成本）。仅编排层使用。
const ocrBatchWindow = 5

// ocrBatchMinPageChars 批结果单页近空阈值（rune 数）：批生成退化实测形态=仅页眉
// 无正文（~28 字符，<PAGE> 数可能对上）。低于阈值整窗回退逐页（逐页 gundam 保真
// 最高，重试无害；真空白/封面级稀疏页逐页路径同样出短文本，重复识别一次成本可
// 接受，空文本仍按既有「空文本=失败」口径记账）。
const ocrBatchMinPageChars = 48

// ocrPageLoop 逐页调用 pageOCR 识别（单页失败跳过继续，全部失败才报错）。
func ocrPageLoop(pngFiles []string, first int, pages string,
	pageOCR func(pageNum int, pngPath string) (string, error),
	progress func(done, total int)) ([]string, int, []int, error) {
	return ocrPageLoopBatch(pngFiles, first, pages, pageOCR, nil, progress)
}

// ocrPageLoopBatch 逐页/分窗批识别编排：页码来自 renderedPageJobs（IN3-11：实际
// 产物页码优先，不可解析时回退 first+i 并告警），页范围过滤用绝对页码（与文本
// 路径 pageInRange 同口径）。batchOCR 非 nil 且待识别 ≥2 页时按 ocrBatchWindow
// 分窗走批识别（跨页上下文，见 PageBatchOCRProvider）；任一窗批失败/结果不齐/
// 含空文本，整窗回退逐页——容错语义与纯逐页完全一致。progress 回调 (done, total)，
// total 为实际 OCR 页数而非渲染页数，保证进度条反映真实工作量（批路径按窗跳进）。
func ocrPageLoopBatch(pngFiles []string, first int, pages string,
	pageOCR func(pageNum int, pngPath string) (string, error),
	batchOCR func(jobs []ocrPageJob) ([]string, error),
	progress func(done, total int)) ([]string, int, []int, error) {

	jobs, fromArtifacts := renderedPageJobs(pngFiles, first)
	if !fromArtifacts && len(pngFiles) > 0 {
		slog.Warn("docmd: OCR 产物文件名不可解析页码，回退 first+i 顺序编号", "first", first, "files", len(pngFiles))
	}
	if pages != "" {
		kept := jobs[:0]
		for _, j := range jobs {
			if pageInRange(j.num, pages) {
				kept = append(kept, j)
			}
		}
		jobs = kept
	}
	total := len(jobs)
	if total == 0 {
		return nil, 0, nil, fmt.Errorf("没有需要 OCR 的页面")
	}
	if progress != nil {
		progress(0, total) // 起步事件：批模式下首窗完成前（可达 5-8 分钟）先给即时反馈
	}

	batching := batchOCR != nil && len(jobs) >= 2
	textsByJob := make([]string, len(jobs))
	failedByJob := make([]bool, len(jobs))
	done := 0
	for start := 0; start < len(jobs); {
		win := 1 // 非批路径：逐页（与历史语义逐字一致，含 progress 步进）
		if batching {
			win = len(jobs) - start
			if win > ocrBatchWindow {
				win = ocrBatchWindow
			}
		}
		winJobs := jobs[start : start+win]
		handled := false
		if batching {
			texts, err := batchOCR(winJobs)
			if err == nil && len(texts) == win {
				complete := true
				for _, tx := range texts {
					if n := utf8.RuneCountInString(strings.TrimSpace(tx)); n < ocrBatchMinPageChars {
						complete = false // 近空页：批生成退化，整窗回退逐页
						break
					}
				}
				if complete {
					for i, tx := range texts {
						textsByJob[start+i] = strings.TrimSpace(tx)
					}
					handled = true
				}
			}
		}
		if !handled { // 批失败/不可用：整窗回退逐页（容错语义不变）
			for i, j := range winJobs {
				text, err := pageOCR(j.num, j.png)
				if err != nil || strings.TrimSpace(text) == "" {
					failedByJob[start+i] = true
				} else {
					textsByJob[start+i] = strings.TrimSpace(text)
				}
			}
		}
		start += win
		done += win
		if progress != nil {
			progress(done, total)
		}
	}

	var texts []string
	var failed []int
	for i, j := range jobs {
		if failedByJob[i] || strings.TrimSpace(textsByJob[i]) == "" {
			failed = append(failed, j.num)
		} else {
			texts = append(texts, textsByJob[i])
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
