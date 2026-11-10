//go:build ignore

// code_atlas_doc — 项目代码图谱导出器（2026-11-10 建图批）。
//
// 从 `go list -json ./...` 与文件盘点单源生成 docs/gaea-code-atlas-2026-11.md：
//   - 分层总览（层数图 + 层间依赖矩阵 + 每层统计）
//   - 分层不变量检查（内核不 import 壳层 / scripts 零被引 / 壳层入边只来自入口）
//   - 全量包清单（层/源码 LOC/测试 LOC/被引/内部出边）
//   - 内部依赖明细（按层分组）
//   - 巨石文件 Top 30
//   - 前端 src 地图（含 boards 细目）
//   - 关键接缝导读（§7 与 §8 为手写导读段，随认知更新改本模板再重导）
//
// 机械段（§1–§6）勿手改文档，改代码后重导；用法：
//
//	go run scripts/code_atlas_doc.go > docs/gaea-code-atlas-2026-11.md
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

const modulePath = "github.com/gaea/gaea"

// 层名（doc 全程使用同一套中文标签）。
var layerNames = map[string]string{
	"entry":      "入口",
	"shell":      "壳层·板块装配",
	"kernel-rt":  "内核·装配执行",
	"kernel-lib": "内核·共享库",
	"office":     "办公域",
	"novel":      "小说域",
	"engine":     "引擎域",
	"pantheon":   "众神殿",
	"domain":     "领域库",
	"devtools":   "scripts 工具带",
}

var layerOrder = []string{"entry", "shell", "kernel-rt", "kernel-lib", "office", "novel", "engine", "pantheon", "domain", "devtools"}

// kernel-rt = 装配与代理运行时（boot/control/tool/agent/evidence/权限/空间/沙箱/插件）。
var kernelRT = map[string]bool{
	"agent": true, "boot": true, "control": true, "tool": true, "evidence": true,
	"spaces": true, "permission": true, "sandbox": true, "plugin": true, "hook": true,
	"command": true, "event": true, "context": true, "contextview": true,
}

// 小说域（创作/书源/人格库）与引擎域（模型/TTS/语音/视觉）的首段集合。
var novelTops = map[string]bool{
	"novelcontext": true, "novelgate": true, "novelreview": true, "novelstyle": true,
	"noveltext": true, "whisper": true, "character": true, "characterlib": true,
	"characterstate": true, "bookimport": true, "booksource": true, "chapter": true,
	"outline": true, "worldview": true, "narrative": true, "maturecraft": true,
	"scene": true, "style": true, "rewrite": true,
}

var engineTops = map[string]bool{
	"herdsman": true, "modelengine": true, "tts": true, "voice": true, "ocr": true, "ai": true,
}

func layerOf(importPath string) string {
	rel := strings.TrimPrefix(importPath, modulePath+"/")
	switch {
	case importPath == modulePath:
		return "entry"
	case strings.HasPrefix(rel, "internal/app"):
		return "shell"
	case strings.HasPrefix(rel, "internal/gaea/"):
		top := strings.SplitN(strings.TrimPrefix(rel, "internal/gaea/"), "/", 2)[0]
		if kernelRT[top] {
			return "kernel-rt"
		}
		return "kernel-lib"
	case strings.HasPrefix(rel, "internal/office"):
		return "office"
	case strings.HasPrefix(rel, "scripts/"):
		return "devtools"
	default:
		top := strings.SplitN(strings.TrimPrefix(rel, "internal/"), "/", 2)[0]
		switch {
		case novelTops[top]:
			return "novel"
		case engineTops[top]:
			return "engine"
		case top == "pantheon":
			return "pantheon"
		default:
			return "domain"
		}
	}
}

type pkg struct {
	ImportPath                string
	Dir                       string
	GoFiles, CgoFiles         []string
	TestGoFiles, XTestGoFiles []string
	Imports                   []string

	layer       string
	srcLOC      int
	testLOC     int
	internalIn  []string // 引用本包的内部包
	internalOut []string // 本包引用的内部包
}

type fileLOC struct {
	path string
	pkg  string
	loc  int
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "-json" {
		runJSON()
		return
	}
	out, err := exec.Command("go", "list", "-json", "./...").Output()
	if err != nil {
		die("go list: %v", err)
	}
	dec := json.NewDecoder(strings.NewReader(string(out)))
	var pkgs []*pkg
	for dec.More() {
		var p pkg
		if err := dec.Decode(&p); err != nil {
			die("decode: %v", err)
		}
		if strings.Contains(p.ImportPath, "node_modules") {
			continue
		}
		p.layer = layerOf(p.ImportPath)
		pkgs = append(pkgs, &p)
	}

	var godFiles []fileLOC
	for _, p := range pkgs {
		for _, f := range p.GoFiles {
			loc := countLines(filepath.Join(p.Dir, f))
			p.srcLOC += loc
			godFiles = append(godFiles, fileLOC{
				path: strings.TrimPrefix(path.Join(p.ImportPath, f), modulePath+"/"),
				pkg:  shortName(p.ImportPath),
				loc:  loc,
			})
		}
		for _, f := range p.CgoFiles {
			p.srcLOC += countLines(filepath.Join(p.Dir, f))
		}
		for _, f := range p.TestGoFiles {
			p.testLOC += countLines(filepath.Join(p.Dir, f))
		}
		for _, f := range p.XTestGoFiles {
			p.testLOC += countLines(filepath.Join(p.Dir, f))
		}
	}
	// 内部依赖边（滤 node_modules 与集合外目标）。
	index := map[string]*pkg{}
	for _, p := range pkgs {
		index[p.ImportPath] = p
	}
	for _, p := range pkgs {
		for _, imp := range p.Imports {
			if imp == p.ImportPath || !strings.HasPrefix(imp, modulePath+"/") {
				continue
			}
			target, ok := index[imp]
			if !ok || strings.Contains(imp, "node_modules") {
				continue
			}
			p.internalOut = append(p.internalOut, shortName(imp))
			target.internalIn = append(target.internalIn, shortName(p.ImportPath))
		}
		sort.Strings(p.internalOut)
		sort.Strings(p.internalIn)
	}
	sort.Slice(pkgs, func(i, j int) bool {
		if pkgs[i].layer != pkgs[j].layer {
			return layerIndex(pkgs[i].layer) < layerIndex(pkgs[j].layer)
		}
		if pkgs[i].srcLOC != pkgs[j].srcLOC {
			return pkgs[i].srcLOC > pkgs[j].srcLOC
		}
		return pkgs[i].ImportPath < pkgs[j].ImportPath
	})
	sort.Slice(godFiles, func(i, j int) bool { return godFiles[i].loc > godFiles[j].loc })

	printDoc(pkgs, godFiles)
}

func layerIndex(l string) int {
	for i, v := range layerOrder {
		if v == l {
			return i
		}
	}
	return len(layerOrder)
}

func shortName(importPath string) string {
	if importPath == modulePath {
		return "(main)"
	}
	return strings.TrimPrefix(importPath, modulePath+"/")
}

func countLines(path string) int {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	return strings.Count(string(raw), "\n") + 1
}

func die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "code_atlas_doc: "+format+"\n", args...)
	os.Exit(1)
}

func printDoc(pkgs []*pkg, godFiles []fileLOC) {
	totalSrc, totalTest := 0, 0
	layerStats := map[string]*[2]int{}
	for _, p := range pkgs {
		st, ok := layerStats[p.layer]
		if !ok {
			st = &[2]int{}
			layerStats[p.layer] = st
		}
		st[0]++
		st[1] += p.srcLOC
		totalSrc += p.srcLOC
		totalTest += p.testLOC
	}

	fmt.Println("# gaea 项目代码图谱（2026-11）")
	fmt.Println()
	fmt.Printf("> 由 `go run scripts/code_atlas_doc.go` 从 `go list -json ./...` + 文件盘点单源生成"+
		"（module `%s`，go 1.26）。§1–§6 为机械段勿手改；§7 接缝导读与 §8 再生成说明随认知"+
		"更新改模板重导。统计口径：Go 源码=GoFiles+CgoFiles 行数（非测试），测试=Test/XTest"+
		"GoFiles；前端=ts/tsx/css 行数。`node_modules` 内杂包已滤。\n", modulePath)
	fmt.Println()
	fmt.Printf("> 全仓 %d 个 Go 包 · 源码 %s 行 · 测试 %s 行（测试/源码 %.2f）。\n",
		len(pkgs), human(totalSrc), human(totalTest), float64(totalTest)/float64(totalSrc))
	fmt.Println()

	// §1 分层总览
	fmt.Println("## 1. 分层总览")
	fmt.Println()
	fmt.Println("```")
	fmt.Println("                ┌─ scripts 工具带（gen_bindings/评测/迁移，零被引）")
	fmt.Println(" 入口 main ──▶ 壳层 internal/app（板块装配/wails bindings）")
	fmt.Println("                ├─▶ 内核·装配执行 internal/gaea/{boot,control,tool,agent,evidence,…}")
	fmt.Println("                │        └─▶ 内核·共享库 internal/gaea/{db,memory,skill,proc,…}")
	fmt.Println("                └─▶ 领域库 internal/{office,novel 系,engine 系,pantheon,schedule,costimport,…}")
	fmt.Println("```")
	fmt.Println()
	fmt.Println("| 层 | 包数 | 源码 LOC | 测试 LOC |")
	fmt.Println("|---|---:|---:|---:|")
	for _, l := range layerOrder {
		if st, ok := layerStats[l]; ok {
			testSum := 0
			for _, p := range pkgs {
				if p.layer == l {
					testSum += p.testLOC
				}
			}
			fmt.Printf("| %s | %d | %s | %s |\n", layerNames[l], st[0], human(st[1]), human(testSum))
		}
	}
	fmt.Printf("| **合计** | **%d** | **%s** | **%s** |\n", len(pkgs), human(totalSrc), human(totalTest))
	fmt.Println()

	// 层间依赖矩阵
	edgeMatrix := map[string]map[string]int{}
	edgeTotal := 0
	for _, p := range pkgs {
		for _, imp := range p.Imports {
			if imp == p.ImportPath || !strings.HasPrefix(imp, modulePath+"/") || strings.Contains(imp, "node_modules") {
				continue
			}
			if t, ok := indexTarget(pkgs, imp); ok {
				if edgeMatrix[p.layer] == nil {
					edgeMatrix[p.layer] = map[string]int{}
				}
				edgeMatrix[p.layer][t.layer]++
				edgeTotal++
			}
		}
	}
	fmt.Println("层间依赖边矩阵（行=引用方 → 列=被引方，共 " + human(edgeTotal) + " 条内部边）：")
	fmt.Println()
	fmt.Println("| ↓从 / 到→ | " + strings.Join(mapNames(layerOrder), " | ") + " |")
	fmt.Println("|---|" + strings.Repeat("---:|", len(layerOrder)))
	for _, from := range layerOrder {
		cells := make([]string, len(layerOrder))
		for i, to := range layerOrder {
			n := edgeMatrix[from][to]
			cells[i] = ""
			if n > 0 {
				cells[i] = human(n)
			}
		}
		fmt.Printf("| %s | %s |\n", layerNames[from], strings.Join(cells, " | "))
	}
	fmt.Println()

	// §2 分层不变量
	fmt.Println("## 2. 分层不变量检查（再生成时复算）")
	fmt.Println()
	var viol []string
	for _, p := range pkgs {
		if strings.HasPrefix(shortName(p.ImportPath), "internal/gaea/") {
			for _, imp := range p.Imports {
				if strings.HasPrefix(imp, modulePath+"/internal/app") {
					viol = append(viol, shortName(p.ImportPath)+" → "+shortName(imp))
				}
			}
		}
	}
	reportCheck(fmt.Sprintf("内核（internal/gaea/**）不 import 壳层（internal/app）——%d 处违例", len(viol)), viol)

	var scriptRefs []string
	for _, p := range pkgs {
		for _, imp := range p.Imports {
			if strings.HasPrefix(imp, modulePath+"/scripts/") {
				scriptRefs = append(scriptRefs, shortName(p.ImportPath)+" → "+shortName(imp))
			}
		}
	}
	reportCheck(fmt.Sprintf("scripts 工具带零被引——%d 处违例", len(scriptRefs)), scriptRefs)

	shellSet := map[string]bool{}
	for _, p := range pkgs {
		if p.layer == "shell" {
			shellSet[shortName(p.ImportPath)] = true
		}
	}
	var shellIn []string
	for _, p := range pkgs {
		if p.layer == "shell" {
			shellIn = append(shellIn, p.internalIn...)
		}
	}
	sort.Strings(shellIn)
	var display, bad []string
	for i, s := range shellIn {
		if i > 0 && s == shellIn[i-1] {
			continue
		}
		display = append(display, s)
		if s != "(main)" && !shellSet[s] {
			bad = append(bad, s)
		}
	}
	reportCheck("壳层入边只来自入口 main（壳层内部互引不计）——实际入边源: "+strings.Join(display, ", "), bad)
	fmt.Println()

	// §3 包清单
	fmt.Println("## 3. 包清单（层内按源码 LOC 降序）")
	fmt.Println()
	fmt.Println("| 包 | 层 | 源码 LOC | 测试 LOC | 被引 | 内部出边 |")
	fmt.Println("|---|---|---:|---:|---:|---:|")
	for _, p := range pkgs {
		fmt.Printf("| `%s` | %s | %s | %s | %d | %d |\n",
			shortName(p.ImportPath), layerNames[p.layer], human(p.srcLOC),
			humanOrDash(p.testLOC), len(p.internalIn), len(p.internalOut))
	}
	fmt.Println()

	// §4 依赖明细
	fmt.Println("## 4. 内部依赖明细（按层分组；→ 后为引用的内部包）")
	fmt.Println()
	for _, l := range layerOrder {
		var inLayer []*pkg
		for _, p := range pkgs {
			if p.layer == l && len(p.internalOut) > 0 {
				inLayer = append(inLayer, p)
			}
		}
		if len(inLayer) == 0 {
			continue
		}
		fmt.Printf("### %s\n\n", layerNames[l])
		for _, p := range inLayer {
			fmt.Printf("- `%s` → %s\n", shortName(p.ImportPath), strings.Join(p.internalOut, ", "))
		}
		fmt.Println()
	}

	// §5 巨石文件
	fmt.Println("## 5. 巨石文件 Top 30（非测试，按 LOC）")
	fmt.Println()
	fmt.Println("| 文件 | 包 | LOC |")
	fmt.Println("|---|---|---:|")
	limit := 30
	if len(godFiles) < limit {
		limit = len(godFiles)
	}
	for _, f := range godFiles[:limit] {
		fmt.Printf("| `%s` | `%s` | %s |\n", f.path, f.pkg, human(f.loc))
	}
	fmt.Println()

	// §6 前端
	fmt.Println("## 6. 前端 src 地图（ts/tsx 计数；.test.* 单列）")
	fmt.Println()
	frontendTable()
	fmt.Println()

	// §7/§8 curated（模板内手写导读）
	printSeams()
}

func indexTarget(pkgs []*pkg, imp string) (*pkg, bool) {
	rel := strings.TrimPrefix(imp, modulePath+"/")
	for _, p := range pkgs {
		if shortName(p.ImportPath) == rel {
			return p, true
		}
	}
	return nil, false
}

func mapNames(ls []string) []string {
	out := make([]string, len(ls))
	for i, l := range ls {
		out[i] = layerNames[l]
	}
	return out
}

func reportCheck(title string, violations []string) {
	if len(violations) == 0 {
		fmt.Printf("- ✅ %s\n", title)
		return
	}
	fmt.Printf("- ❌ %s\n", title)
	for _, v := range violations {
		fmt.Printf("  - %s\n", v)
	}
}

func dedupe(in []string) []string {
	sort.Strings(in)
	out := in[:0]
	for i, s := range in {
		if i == 0 || s != in[i-1] {
			out = append(out, s)
		}
	}
	return out
}

func human(n int) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	parts = append([]string{s}, parts...)
	return strings.Join(parts, ",")
}

func humanOrDash(n int) string {
	if n == 0 {
		return "—"
	}
	return human(n)
}

func frontendTable() {
	root := filepath.Join("frontend", "src")
	type stat struct{ files, loc, testFiles, testLOC int }
	groups := map[string]*stat{}
	boards := map[string]*stat{}
	filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		segs := strings.Split(rel, string(filepath.Separator))
		ext := filepath.Ext(rel)
		if ext != ".ts" && ext != ".tsx" && ext != ".css" && ext != ".js" && ext != ".jsx" {
			return nil
		}
		top := segs[0]
		if len(segs) == 1 {
			top = "(顶层)"
		}
		st := groups[top]
		if st == nil {
			st = &stat{}
			groups[top] = st
		}
		isTest := strings.Contains(filepath.Base(rel), ".test.")
		loc := countLines(path)
		if isTest && (ext == ".ts" || ext == ".tsx") {
			st.testFiles++
			st.testLOC += loc
			return nil
		}
		st.files++
		st.loc += loc
		if len(segs) > 1 && segs[0] == "boards" {
			b := boards[segs[1]]
			if b == nil {
				b = &stat{}
				boards[segs[1]] = b
			}
			b.files++
			b.loc += loc
		}
		return nil
	})
	fmt.Println("| 目录 | 源文件 | 源码 LOC | 测试文件 | 测试 LOC |")
	fmt.Println("|---|---:|---:|---:|---:|")
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return groups[keys[i]].loc > groups[keys[j]].loc
	})
	for _, k := range keys {
		st := groups[k]
		fmt.Printf("| `%s` | %d | %s | %s | %s |\n", k, st.files, human(st.loc), humanOrDash(st.testFiles), humanOrDash(st.testLOC))
	}
	bk := make([]string, 0, len(boards))
	for k := range boards {
		bk = append(bk, k)
	}
	sort.Slice(bk, func(i, j int) bool { return boards[bk[i]].loc > boards[bk[j]].loc })
	fmt.Println()
	fmt.Println("boards/ 细目：")
	fmt.Println()
	for _, k := range bk {
		fmt.Printf("- `%s`：%d 文件 / %s 行\n", k, boards[k].files, human(boards[k].loc))
	}
}

func printSeams() {
	fmt.Println("## 7. 关键接缝导读（手写段，动结构时同步更新）")
	fmt.Println()
	fmt.Println("1. **工具面（唯一注册表）**：`boot.Build` 装配单一 `tool.Registry`，四个入口——")
	fmt.Println("   builtin（`RegisterBuiltin` 全集）、ExtraTools（app 层 `gaeaExtraTools` 单点）、")
	fmt.Println("   插件/MCP 热插（`plugins.go`/`controller_mcp.go`，动态 `mcp__<server>__<tool>`）、")
	fmt.Println("   Build 内部直加（skill/memory/task/slash）。契约=`tool.RegistryMap`（P0 五行契约，")
	fmt.Println("   tool/boot/app 三面守卫+文档单源导出）；空间装配期物理过滤=`builtin.AllowsSpace`")
	fmt.Println("   （S1.3-B；2026-11-10 漂移刀后与契约行双守卫钉死）。`/mcp__` 前缀是 prompt 旁路不进注册表。")
	fmt.Println("2. **执行面**：`internal/gaea/agent`（会话/预算/渲染/工具守卫/子代理）、`control`")
	fmt.Println("   （控制器输入流与 MCP prompt 旁路）、`permission`（Approver 审批面，子代理审批放行点）、")
	fmt.Println("   `evidence`（交付证据五档：agent-write/agent-produce/office-journal/publish-guard/none）。")
	fmt.Println("3. **壳层（板块装配）**：`internal/app` 是最大层（600+ 文件），wails bindings 双向")
	fmt.Println("   （`scripts/gen_bindings` 生成，drift 锁）；板块=board manifest + 各 `gaea_*.go` 文件；")
	fmt.Println("   编程板块走隔离 DSH（`C:\\AI\\dsh`:3091，preflight 查红队预设 `id: rea-mcp`）。")
	fmt.Println("4. **引擎域**：herdsman（OpenAI 兼容 API，本机 8080）、modelengine、tts、voice")
	fmt.Println("   （native=cgo sherpa-onnx，三 DLL 与 exe 同目录铁律，缺 ucrt64 gcc 构建会静默降级空壳）、")
	fmt.Println("   ocr、ComfyUI（图片后端）。外部引擎均有 idle-exit/JobObject 收割线。")
	fmt.Println("5. **数据面**：配置 `.gaea_config.json`（旧品牌 `.wubigork_config.json` 回退）；")
	fmt.Println("   `internal/gaea/db`=modernc.org/sqlite（纯 Go，WAL）；memory/knowledge/factbase/cost")
	fmt.Println("   各自持久化；会话存档 jsonl+gaea-log 双形态；众神殿 sessions 在 `$APPDATA/gaea/pantheon`。")
	fmt.Println("6. **前端**：`frontend/src`（React+TS），boards=板块页、stores=状态、api=wails 绑定包装；")
	fmt.Println("   HTTP 桥接 opt-in（`GAEA_HTTP_PORT`，冒烟/走查用）。")
	fmt.Println()
	fmt.Println("## 8. 再生成方式")
	fmt.Println()
	fmt.Println("```bash")
	fmt.Println("go run scripts/code_atlas_doc.go > docs/gaea-code-atlas-2026-11.md")
	fmt.Println("```")
	fmt.Println()
	fmt.Println("机械段（§1–§6）随代码变化重导即可；接缝导读（§7）与分层口径（脚本顶部")
	fmt.Println("kernelRT/novelTops/engineTops 集合）是人工认知，调整后重导。新增顶层包通常只须")
	fmt.Println("确认它落进的层名是否符合预期（domain 是兜底层）。")
}

// runJSON 图数据导出（交互视图/后续工具用）：
//
//	go run scripts/code_atlas_doc.go -json > atlas.json
//
// 设置面板「代码图谱」标签页的仓内快照资产同源再生成：
//
//	go run scripts/code_atlas_doc.go -json > frontend/src/data/codeAtlas.json
//
// 节点=包（id/层/源码与测试 LOC），边=内部依赖（from 层标记，便于视图折叠），
// meta=生成日期与底数（前端页脚如实展示快照口径）。
func runJSON() {
	out, err := exec.Command("go", "list", "-json", "./...").Output()
	if err != nil {
		die("go list: %v", err)
	}
	dec := json.NewDecoder(strings.NewReader(string(out)))
	var pkgs []*pkg
	for dec.More() {
		var p pkg
		if err := dec.Decode(&p); err != nil {
			die("decode: %v", err)
		}
		if strings.Contains(p.ImportPath, "node_modules") {
			continue
		}
		p.layer = layerOf(p.ImportPath)
		for _, f := range p.GoFiles {
			p.srcLOC += countLines(filepath.Join(p.Dir, f))
		}
		for _, f := range p.CgoFiles {
			p.srcLOC += countLines(filepath.Join(p.Dir, f))
		}
		for _, f := range p.TestGoFiles {
			p.testLOC += countLines(filepath.Join(p.Dir, f))
		}
		for _, f := range p.XTestGoFiles {
			p.testLOC += countLines(filepath.Join(p.Dir, f))
		}
		pkgs = append(pkgs, &p)
	}
	index := map[string]*pkg{}
	for _, p := range pkgs {
		index[p.ImportPath] = p
	}
	type node struct {
		ID      string `json:"id"`
		Layer   string `json:"layer"`
		SrcLOC  int    `json:"srcLoc"`
		TestLOC int    `json:"testLoc"`
	}
	type edge struct {
		From  string `json:"from"`
		To    string `json:"to"`
		Layer string `json:"layer"`
	}
	var nodes []node
	var edges []edge
	for _, p := range pkgs {
		nodes = append(nodes, node{shortName(p.ImportPath), p.layer, p.srcLOC, p.testLOC})
		for _, imp := range p.Imports {
			if _, ok := index[imp]; !ok || imp == p.ImportPath || strings.Contains(imp, "node_modules") {
				continue
			}
			edges = append(edges, edge{shortName(p.ImportPath), shortName(imp), p.layer})
		}
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", " ")
	totalSrc, totalTest := 0, 0
	for _, p := range pkgs {
		totalSrc += p.srcLOC
		totalTest += p.testLOC
	}
	_ = enc.Encode(map[string]any{
		"meta": map[string]any{
			"generated": "2026-11-10",
			"packages":  len(nodes),
			"edges":     len(edges),
			"srcLoc":    totalSrc,
			"testLoc":   totalTest,
		},
		"nodes": nodes,
		"edges": edges,
	})
}
