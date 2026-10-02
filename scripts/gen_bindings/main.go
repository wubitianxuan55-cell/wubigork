// gen_bindings 生成 gaea 的板块绑定门面（S2-3「App 绑定面拆分」）。
//
// 用途：把 App（及其嵌入的 core/writingState/mediaState/whisperState/
// officeState）的全部导出方法按板块拆到多个 Wails 绑定对象，方法体零改动
// （纯委托 b.a.Method(args)）。生成物：
//   - internal/app/bindings_<板块>.go：每个板块一个门面结构体 + 委托方法
//   - internal/app/bindings_manifest.go：NewBindings(a *App) []any（main.go 用）
//   - internal/app/bindings_completeness_test.go：反射完备性测试（测试兜底）
//
// 用法：go run ./scripts/gen_bindings
// 方法 → 板块映射规则见 mapMethod 函数；未覆盖的方法会报错退出（防遗漏）。
package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// receiverTypes 参与绑定的接收者类型（App 及其嵌入的子状态结构）。
var receiverTypes = map[string]bool{
	"App": true, "core": true, "writingState": true,
	"mediaState": true, "whisperState": true, "officeState": true,
}

// facadeOrder 板块顺序（生成文件与绑定顺序一致，稳定可读）。
var facadeOrder = []string{"core", "office", "memory", "cost", "model", "voice", "chat", "novel", "image", "charlib", "sin"}

type method struct {
	Name      string
	Params    string // "(a1 T1, a2 T2)" 含括号，用于签名与调用
	ArgNames  string // "a1, a2"
	Results   string // "" | "R" | "R, error" 等（不含括号）… 见下
	HasParens bool   // 结果是否有括号（多返回值）
	Receiver  string
	File      string // 声明所在文件（含 internal/app/ 前缀，供遮蔽基线报点）
	Line      int    // 声明行号
}

// shadowPair 一处「内嵌接收者版本的实现被 App 同名声明遮蔽」——该实现不进
// 绑定门面，但仍编译、仍可被包内代码直接调用（绕过 App 版本里的修复）。
type shadowPair struct {
	Name     string // 方法名
	Receiver string // 被遮蔽方的接收者类型（core/writingState/…）
	File     string // 被遮蔽方声明所在文件（含 internal/app/ 前缀）
	Line     int    // 被遮蔽方声明行号
}

// shadowBaseline 在册遮蔽清单（**显式列名字**，不是「条数上限」）。
//
// 为什么不用条数上限（批次十三 round 15，审计 X1-13 精化）：条数口径下「超基线时
// 到底新增了哪些」只能靠 `shadow[max:]` 切片猜，而 shadow 是按方法名字典序排的——
// 新增的名字若排序靠前，切片会把**在册条目**当成新增报出来（总量仍红、不静默，
// 但报点指错条目，接手的人照抄登记反而把基线写坏）。集合差没这个问题：按名字对照
// 在册清单，多出来的才叫新增。
//
// 口径（与 collectMethods 的去重逻辑同源，别处统计的数字不可直接对比）：
//
//	① 只解析 internal/app 顶层、非 _test.go、非 bindings_*.go 的文件；
//	② 接收者必须是绑定面白名单 receiverTypes（App/core/writingState/mediaState/
//	   whisperState/officeState）——**门面类型（CoreB/ModelB/…）不计**，它们在
//	   shadow-diag 里也会被算成一组，那会把数字抬高到 4 倍以上（实测 455）；
//	③ 只数「App 声明了同名方法、内嵌类型的实现因此被去重丢弃」的那些实现。
//
// 判据（集合逐名字对照，与顺序无关）：
//
//	实测有、在册无 → 新增遮蔽 → exit 1，逐条列出**新增的名字**（内嵌实现会绕过
//	                  App 版本里的修复，必须显式处置）；
//	在册有、实测无 → 在册项消失（改名/删除实现）→ exit 1，如实报「消失」并提示
//	                  同步本清单，**不会**被误报成新增（集合差里两者分开算）；
//	实测 == 在册 → exit 0（存量遮蔽打一条 stderr 告警但不拦门）。
//
// 本清单就是「当前允许存在的遮蔽全集」，任何不一致都要显式同步这里并写明理由，
// 不许静默漂移——「消失也红」正是为了让清单不悄悄过期（守卫自身也得守住）。
//
// 基线来源 / 日期：2026-10-02 批次十二 round 14 实测（线 4）= 2 处
// （core 的 SetFeatureModel / SetFeatureModelEnabled 被 App 同名声明遮蔽，
// internal/app/feature_model_handler.go:79 / :147）；批次十三 round 15 把
// 「条数上限 shadowBaselineMax = 2」改成这份显式清单。
//
// 与审计描述的关系：审计 X1-13 提到的「约 281 份被遮蔽实现」不是本口径——它对
// 应的是全仓同名对（含门面重复），本口径只数真正被丢掉的实现；归零仍是余量
// （需逐条改名/加说明，见 docs/code-audit-2026-10-02/ 分册）。
var shadowBaseline = []shadowPair{
	{Name: "SetFeatureModel", Receiver: "core", File: "internal/app/feature_model_handler.go", Line: 79},
	{Name: "SetFeatureModelEnabled", Receiver: "core", File: "internal/app/feature_model_handler.go", Line: 147},
}

// mapMethod 方法 → 板块。规则按优先级：显式覆盖表 → 前缀规则 → 接收者默认。
func mapMethod(m method) string {
	// 显式覆盖（规则优先）
	if f, ok := explicitOverrides[m.Name]; ok {
		return f
	}
	n := m.Name
	switch {
	case strings.HasPrefix(n, "Gaea"):
		return mapGaea(n)
	case strings.HasPrefix(n, "Herdsman"):
		return "model"
	case strings.HasPrefix(n, "TTS"), strings.HasPrefix(n, "Voice"),
		strings.HasPrefix(n, "Whisper"), strings.HasPrefix(n, "ASR"):
		return "voice"
	case strings.HasPrefix(n, "Chat"), strings.HasPrefix(n, "MainBrainChat"),
		strings.HasPrefix(n, "Brain"), n == "RunModule":
		return "chat"
	case strings.HasPrefix(n, "Sin"):
		// 原罪板块（闲庭·图文故事创作）：Sin* 前缀独占一门面
		return "sin"
	case strings.HasPrefix(n, "Character"):
		return "charlib"
	case strings.HasPrefix(n, "GenerateFreeImage"), strings.HasPrefix(n, "CancelImageGeneration"),
		strings.HasPrefix(n, "GetComfyUI"), strings.HasPrefix(n, "GetImageBackend"),
		strings.HasPrefix(n, "SetImageBackend"), strings.HasPrefix(n, "GetPortraitConfig"),
		strings.HasPrefix(n, "SetPortraitConfig"):
		return "image"
	}
	switch m.Receiver {
	case "writingState":
		return "novel"
	case "mediaState":
		return "image"
	case "whisperState":
		return "voice"
	case "officeState":
		return "office"
	}
	return "core"
}

// mapGaea Gaea 前缀方法按功能细分（办公引擎为默认）。
func mapGaea(n string) string {
	switch {
	case strings.HasPrefix(n, "GaeaCost"), strings.HasPrefix(n, "GaeaPrice"):
		return "cost"
	case strings.HasPrefix(n, "GaeaKnowledge"), strings.HasPrefix(n, "GaeaMemory"),
		strings.HasPrefix(n, "GaeaProfile"), strings.HasPrefix(n, "GaeaSemantic"),
		strings.HasPrefix(n, "GaeaWhisper"):
		return "memory"
	case strings.HasPrefix(n, "GaeaModels"), strings.HasPrefix(n, "GaeaSetModel"),
		strings.HasPrefix(n, "GaeaModel"), strings.HasPrefix(n, "GaeaEngines"),
		strings.HasPrefix(n, "GaeaSetEngine"):
		return "model"
	case strings.HasPrefix(n, "GaeaCharacter"):
		return "charlib"
	}
	return "office"
}

// explicitOverrides 无法用前缀表达的映射。
var explicitOverrides = map[string]string{
	// v4.291：反推任务化两绑定挂 *App（taskMgr 在 App 级），点名归 novel。
	"NovelOutlineReconstructStart":   "novel",
	"NovelOutlineReconstructTaskGet": "novel",
	// v4.323：t6 提示词工坊五绑定挂 *App（覆盖缓存在 writingState 侧、方法
	// 在 App 级），点名归 novel（规格 进度计划/gaea-prompt-workshop-t6-20260916.md §4.2）。
	"PromptTemplateList":    "novel",
	"PromptTemplateGet":     "novel",
	"PromptTemplateSave":    "novel",
	"PromptTemplateReset":   "novel",
	"PromptTemplatePreview": "novel",
	// v4.324：t6-C2 模板包导入导出两绑定同因挂 *App，点名归 novel
	//（规格 进度计划/gaea-prompt-bundle-t6c2-20260916.md §4）。
	"PromptBundleExport":        "novel",
	"PromptBundleImport":        "novel",
	"GetModelRoute":             "model",
	"GetSensitiveLocal":         "model",
	"GetOfficeLocal":            "model",
	"GetOfflineMode":            "model",
	"SetOfflineMode":            "model",
	"SetSensitiveLocal":         "model",
	"SetOfficeLocal":            "model",
	"GetModelMonitor":           "model",
	"SetFeatureModel":           "model",
	"SetFeatureModelEnabled":    "model",
	"GetEngines":                "model",
	"GetEngineFailover":         "model",
	"SetEngineFailover":         "model",
	"GetActiveEngine":           "model",
	"GetActiveModel":            "model",
	"GetEngineStatus":           "model",
	"GetModelCatalog":           "model",
	"Startup":                   "core",
	"Shutdown":                  "core",
	"SetDistFS":                 "core",
	"SetPromptFS":               "core",
	"Login":                     "core",
	"GetLoginStatus":            "core",
	"Logout":                    "core",
	"SaveToken":                 "core",
	"GenerateCharacterPortrait": "image",
	"SetCharacterPortrait":      "image",
	"CmdKEdit":                  "novel",
	"LocalTranslate":            "office",
	"Search":                    "core",
	"ExportAll":                 "core",
	"GetConfig":                 "core",
	"SaveConfig":                "core",
	"ListSkills":                "core",
	"GetStats":                  "core",
	"CreateProject":             "core",
	"OpenProject":               "core",
	"CloseProject":              "core",
	"GetProjectInfo":            "core",
	"GetNovelsDir":              "core",
	"ListProjects":              "core",
	"DeleteProject":             "core",
	"GetCompileTemplates":       "core",
	"ExportHTML":                "core",
	"GetDashboard":              "core",
	"AnalyzeStyle":              "core",
	"GetStyleProfile":           "core",
	"ImportStyleProfile":        "core",
	"ChatGeneral":               "chat",
	"MainBrainChat":             "chat",
	// 阶段 3（D3）：分流统计/索引状态/受控测评归属模型中心与记忆中枢
	"GaeaUsageOverview":       "model",
	"GaeaGetUsdCnyRate":       "model", // T6-6.2 汇率配置（模型中心）
	"GaeaSetUsdCnyRate":       "model",
	"GaeaSemanticIndexStatus": "memory",
	"GaeaBenchmarkList":       "model",
	"GaeaBenchmarkStart":      "model",
	"GaeaBenchmarkDetail":     "model",
	"GaeaBenchmarkExport":     "model",
	// 3.0 Step 2：板块 manifest 查询挂 CoreB（前缀规则默认 core，显式声明对齐文档）
	"GetBoardManifests": "core",
	// S4 双空间：GaeaSpace* 绑定面挂 CoreB（前缀规则会落 office，设计 §6 显式覆盖）
	"GaeaSpaceList":       "core",
	"GaeaSpaceActive":     "core",
	"GaeaSpaceActivate":   "core",
	"GaeaSpaceProfiles":   "core",
	"GaeaSpaceProfileSet": "core",
	// v4.3：TTS 参数预览归 voice 板块；书封生成归 novel 板块（前缀规则会落 office）
	"GaeaTTSVoiceParams":    "voice",
	"GaeaGenerateBookCover": "novel",
	// v4.113.0 刀4：进度计划文件持久化（前缀规则本就落 office，显式声明对齐登记惯例）
	"GaeaScheduleLoad": "office",
	"GaeaScheduleSave": "office",
	// 阶段七 7.1-2 路由学习：账本/建议绑定面挂 ModelB（前缀规则默认落 office，
	// 模型中心「成本归因」tab 消费，GaeaUsageOverview 同族先例）
	"GaeaRouteLedger":           "model",
	"GaeaRouteSuggestions":      "model",
	"GaeaRouteSuggestionApply":  "model",
	"GaeaRouteSuggestionIgnore": "model",
}

func main() { os.Exit(run(os.Args[1:])) }

func run(args []string) int {
	fs := flag.NewFlagSet("gen_bindings", flag.ContinueOnError)
	namesOnly := fs.Bool("names", false, "只输出全部导出方法名（一行一个，稳定排序），不写任何生成文件")
	shadowCheck := fs.Bool("shadow-check", false, "只校验遮蔽基线与新增遮蔽（不写任何生成文件）")
	shadowDiag := fs.Bool("shadow-diag", false, "只打印遮蔽基线/实测数/全量遮蔽对（不写任何生成文件）")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	dir := "internal/app"
	methods, shadow, err := collectMethods(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "collect:", err)
		return 1
	}
	if len(methods) == 0 {
		fmt.Fprintln(os.Stderr, "no methods collected")
		return 1
	}

	// 遮蔽基线闸（X1-13）：任何新增遮蔽在写生成物之前就红，且不落盘——生成物
	// 只在闸门通过后才写，避免半截产物污染工作树。
	if *shadowCheck || *shadowDiag {
		printShadowReport(shadow, *shadowDiag)
		rc := checkShadowBaseline(shadow)
		if rc != 0 {
			return rc
		}
		return 0
	}
	if rc := checkShadowBaseline(shadow); rc != 0 {
		return rc
	}

	// -names：仅输出方法名清单（供前端 bindingNames.ts 与 CI 漂移检查对照），
	// 不写任何生成文件。同一方法名按字典序稳定排序。
	if *namesOnly {
		names := make([]string, 0, len(methods))
		for _, m := range methods {
			names = append(names, m.Name)
		}
		sort.Strings(names)
		for _, n := range names {
			fmt.Println(n)
		}
		return 0
	}

	// 按板块分组
	groups := map[string][]method{}
	for _, m := range methods {
		f := mapMethod(m)
		if f == "" {
			fmt.Fprintf(os.Stderr, "方法 %s 未映射到板块\n", m.Name)
			return 1
		}
		groups[f] = append(groups[f], m)
	}

	// 每个板块内按名字排序（稳定）
	for k := range groups {
		sort.Slice(groups[k], func(i, j int) bool { return groups[k][i].Name < groups[k][j].Name })
	}

	// 写文件
	for _, facade := range facadeOrder {
		ms := groups[facade]
		if len(ms) == 0 {
			continue
		}
		if err := writeFacade(facade, ms); err != nil {
			fmt.Fprintln(os.Stderr, "write facade:", err)
			return 1
		}
		fmt.Printf("%-10s %3d 个方法\n", facade, len(ms))
	}
	if err := writeManifest(groups); err != nil {
		fmt.Fprintln(os.Stderr, "write manifest:", err)
		return 1
	}
	if err := writeCompletenessTest(groups); err != nil {
		fmt.Fprintln(os.Stderr, "write test:", err)
		return 1
	}
	fmt.Printf("合计 %d 个导出方法 → %d 个绑定门面\n", len(methods), len(groups))
	return 0
}

// printShadowReport 打印在册清单与本次实测（-shadow-check / -shadow-diag 用）。
func printShadowReport(shadow []shadowPair, full bool) {
	fmt.Printf("遮蔽基线（在册清单 %d 项）：", len(shadowBaseline))
	for i, s := range shadowBaseline {
		if i > 0 {
			fmt.Print("、")
		}
		fmt.Printf("%s（%s %s:%d）", s.Name, s.Receiver, s.File, s.Line)
	}
	fmt.Printf("\n本次实测 = %d 项\n", len(shadow))
	if !full {
		return
	}
	for _, s := range shadow {
		fmt.Printf("  %s ← %s %s:%d\n", s.Name, s.Receiver, s.File, s.Line)
	}
}

// shadowMove 同名遮蔽的声明位置与在册记录不一致（仅提示，不参与判闸——行号会随
// 上方代码变动，不该因此拦门）。
type shadowMove struct{ registered, measured shadowPair }

// shadowDriftReport 归因结果：退出码 + 逐行文案。文案与判闸共用同一份事实，
// 自测可直接断言它，不必重解析 stderr。
type shadowDriftReport struct {
	code  int
	lines []string
}

// shadowDiff 集合差：**按名字对照**，与顺序无关——这正是取代「按字典序切片猜
// 新增」的关键。added = 实测有而在册无（新增遮蔽）；gone = 在册有而实测无
// （改名/删除实现）；moved = 同名但声明位置漂移。
func shadowDiff(shadow, baseline []shadowPair) (added, gone []shadowPair, moved []shadowMove) {
	measured := make(map[string]shadowPair, len(shadow))
	for _, s := range shadow {
		measured[s.Name] = s
	}
	registered := make(map[string]shadowPair, len(baseline))
	for _, s := range baseline {
		registered[s.Name] = s
	}
	for _, s := range shadow {
		r, ok := registered[s.Name]
		if !ok {
			added = append(added, s)
			continue
		}
		if r.File != s.File || r.Line != s.Line {
			moved = append(moved, shadowMove{registered: r, measured: s})
		}
	}
	for _, r := range baseline {
		if _, ok := measured[r.Name]; !ok {
			gone = append(gone, r)
		}
	}
	return added, gone, moved
}

// reportShadowDrift 遮蔽基线闸（X1-13）的归因：实测集合与在册清单逐名字对照，
// 「新增」与「在册消失」分开报，两边都能直接照抄登记。
//
//	新增或消失任一不一致 → exit 1；位置漂移只提示；完全一致 → exit 0。
func reportShadowDrift(shadow, baseline []shadowPair) shadowDriftReport {
	added, gone, moved := shadowDiff(shadow, baseline)
	if len(added) == 0 && len(gone) == 0 {
		var lines []string
		if len(shadow) > 0 {
			lines = append(lines, fmt.Sprintf(
				"gen_bindings: %d 个方法被 App 同名声明遮蔽（在册清单 %d 项逐名字对齐——设计内委托；清单脚本 scripts/gen_bindings -shadow-diag）",
				len(shadow), len(baseline)))
		}
		for _, m := range moved {
			lines = append(lines, fmt.Sprintf(
				"gen_bindings: 在册位置漂移（仅提示，不改判据）：%s 在册 %s:%d / 实测 %s:%d——需要精确报点时同步清单",
				m.registered.Name, m.registered.File, m.registered.Line, m.measured.File, m.measured.Line))
		}
		return shadowDriftReport{code: 0, lines: lines}
	}
	var lines []string
	if len(added) > 0 {
		lines = append(lines, fmt.Sprintf(
			"gen_bindings: 新增遮蔽 %d 处（实测 %d 项 / 在册清单 %d 项）——新增遮蔽会让内嵌实现绕过 App 版本的修复，必须显式处置：",
			len(added), len(shadow), len(baseline)))
		for _, s := range added {
			lines = append(lines, fmt.Sprintf("  - %s（被遮蔽实现：%s，%s:%d）", s.Name, s.Receiver, s.File, s.Line))
		}
	}
	if len(gone) > 0 {
		lines = append(lines, fmt.Sprintf(
			"gen_bindings: 在册遮蔽消失 %d 处（在册有、本次实测无——多半是改名或删除实现），这不是新增：", len(gone)))
		for _, s := range gone {
			lines = append(lines, fmt.Sprintf("  - %s（在册记录：%s，%s:%d）", s.Name, s.Receiver, s.File, s.Line))
		}
		lines = append(lines, "  请同步在册清单（scripts/gen_bindings/main.go 的 shadowBaseline），别让在册项过期。")
	}
	lines = append(lines, fmt.Sprintf(
		"gen_bindings: 处置二选一：①消除遮蔽（改名/删内嵌重复实现）②确有设计理由（或确为永久改名/删除）则把 shadowBaseline 同步为本次实测的 %d 项并写明理由。",
		len(shadow)))
	return shadowDriftReport{code: 1, lines: lines}
}

// checkShadowBaseline 遮蔽基线闸：打印归因、返回退出码（0 = 与在册清单一致）。
// run() 在读盘解析之后、**写任何生成物之前**调用它——判闸失败不落盘，不留半截产物。
func checkShadowBaseline(shadow []shadowPair) int {
	r := reportShadowDrift(shadow, shadowBaseline)
	for _, line := range r.lines {
		fmt.Fprintln(os.Stderr, line)
	}
	return r.code
}

// collectMethods 解析 internal/app 下所有非测试 .go 文件，收集绑定面方法的签名。
// 同时收集 import 名→路径映射（生成门面文件需要），并返回被 App 遮蔽的内嵌
// 实现清单（按方法名排序——排序只影响打印可读性，判闸是名字集合差，见
// shadowBaseline 与 shadowDiff）。
func collectMethods(dir string) ([]method, []shadowPair, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, err
	}
	var out []method
	imports := map[string]string{}
	fset := token.NewFileSet()
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		// 跳过生成物（本生成器输出）
		if strings.HasPrefix(e.Name(), "bindings_") {
			continue
		}
		f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %w", path, err)
		}
		// 收集 import 别名 → 路径
		for _, imp := range f.Imports {
			p := strings.Trim(imp.Path.Value, `"`)
			name := p
			if i := strings.LastIndexByte(p, '/'); i >= 0 {
				name = p[i+1:]
			}
			if imp.Name != nil {
				name = imp.Name.Name
			}
			if _, dup := imports[name]; !dup {
				imports[name] = p
			}
		}
		for _, decl := range f.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || fd.Recv == nil || len(fd.Recv.List) != 1 {
				continue
			}
			recv := fd.Recv.List[0].Type
			star, ok := recv.(*ast.StarExpr)
			if !ok {
				continue
			}
			ident, ok := star.X.(*ast.Ident)
			if !ok || !receiverTypes[ident.Name] {
				continue
			}
			if !fd.Name.IsExported() {
				continue
			}
			m := method{
				Name:     fd.Name.Name,
				Receiver: ident.Name,
				File:     filepath.ToSlash(path), // 报点统一 '/'（跨平台一致，便于照抄登记）
				Line:     fset.Position(fd.Pos()).Line,
			}
			// 参数
			var params, args []string
			usedNames := map[string]bool{}
			slot := 0
			for _, p := range fd.Type.Params.List {
				typeStr := exprString(p.Type)
				names := p.Names
				if len(names) == 0 {
					names = []*ast.Ident{{Name: "_"}}
				}
				for _, nm := range names {
					slot++
					name := nm.Name
					// 空白标识符/撞名参数不能做转发实参（`f(_)` 编译错误）——
					// 合成占位名（handler 侧 `_ string` 第 5 参先例）。
					if name == "_" || usedNames[name] {
						base := fmt.Sprintf("_p%d", slot)
						for usedNames[base] {
							base += "x"
						}
						name = base
					}
					usedNames[name] = true
					arg := name
					// v4.285 根治（v4.237「绑定层禁变参」工具化）：Wails v2.13
					// 变参绑定不可用（wire []string 与 reflect.Call 口径自相
					// 矛盾），门面签名一律把 ...T 收窄成单值 T、按单值透传——
					// 核心层保持变参语义不变。此前靠 E27 守卫拦 + 手工还原，
					// 每次再生都要返工，现由生成器直接产出合规签名。
					if ell, ok := p.Type.(*ast.Ellipsis); ok {
						params = append(params, name+" "+exprString(ell.Elt))
						args = append(args, name)
						continue
					}
					params = append(params, name+" "+typeStr)
					args = append(args, arg)
				}
			}
			m.Params = "(" + strings.Join(params, ", ") + ")"
			m.ArgNames = strings.Join(args, ", ")
			// 结果
			if fd.Type.Results != nil {
				var res []string
				for _, r := range fd.Type.Results.List {
					typeStr := exprString(r.Type)
					if len(r.Names) > 0 {
						res = append(res, strings.Join(goNames(r.Names), ", ")+" "+typeStr)
					} else {
						res = append(res, typeStr)
					}
				}
				if len(res) > 1 {
					m.HasParens = true
				}
				m.Results = strings.Join(res, ", ")
			}
			out = append(out, m)
		}
	}
	// 去重：同名方法保留 App 直接声明的（shadow 嵌入），否则保留首次出现的。
	hasApp := map[string]bool{}
	for _, m := range out {
		if m.Receiver == "App" {
			hasApp[m.Name] = true
		}
	}
	seen := map[string]bool{}
	dedup := make([]method, 0, len(out))
	var shadow []shadowPair // P1-2/X1-13：遮蔽不再静默去重——计数进基线闸，明细可诊断
	for _, m := range out {
		if seen[m.Name] {
			continue
		}
		if hasApp[m.Name] && m.Receiver != "App" {
			shadow = append(shadow, shadowPair{Name: m.Name, Receiver: m.Receiver, File: m.File, Line: m.Line})
			continue // 被 App 版本 shadow；不标记 seen，App 版本随后占用
		}
		seen[m.Name] = true
		dedup = append(dedup, m)
	}
	sort.Slice(shadow, func(i, j int) bool {
		if shadow[i].Name != shadow[j].Name {
			return shadow[i].Name < shadow[j].Name
		}
		return shadow[i].File < shadow[j].File
	})
	globalImports = imports
	return dedup, shadow, nil
}

// globalImports 收集到的 import 名→路径（写门面文件时按需引用）。
var globalImports map[string]string

func goNames(names []*ast.Ident) []string {
	var out []string
	for _, n := range names {
		out = append(out, n.Name)
	}
	return out
}

// exprString 把 AST 类型表达式还原为源码文本（含 import 别名前的包名）。
func exprString(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + exprString(t.X)
	case *ast.SelectorExpr:
		return exprString(t.X) + "." + t.Sel.Name
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + exprString(t.Elt)
		}
		return "[" + exprString(t.Len) + "]" + exprString(t.Elt)
	case *ast.MapType:
		return "map[" + exprString(t.Key) + "]" + exprString(t.Value)
	case *ast.ChanType:
		switch t.Dir {
		case ast.SEND:
			return "chan<- " + exprString(t.Value)
		case ast.RECV:
			return "<-chan " + exprString(t.Value)
		}
		return "chan " + exprString(t.Value)
	case *ast.FuncType:
		return "func" + exprStringFieldList(t.Params) + resultsString(t.Results)
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.StructType:
		return "struct{}"
	case *ast.Ellipsis:
		return "..." + exprString(t.Elt)
	case *ast.BasicLit:
		return t.Value
	case *ast.ParenExpr:
		return "(" + exprString(t.X) + ")"
	case *ast.IndexExpr:
		return exprString(t.X) + "[" + exprString(t.Index) + "]"
	case *ast.UnaryExpr:
		return t.Op.String() + exprString(t.X)
	}
	return "any"
}

func exprStringFieldList(fl *ast.FieldList) string {
	if fl == nil {
		return ""
	}
	var parts []string
	for _, p := range fl.List {
		parts = append(parts, exprString(p.Type))
	}
	return "(" + strings.Join(parts, ", ") + ")"
}

func resultsString(fl *ast.FieldList) string {
	if fl == nil {
		return ""
	}
	var parts []string
	for _, p := range fl.List {
		parts = append(parts, exprString(p.Type))
	}
	if len(parts) == 1 {
		return " " + parts[0]
	}
	return " (" + strings.Join(parts, ", ") + ")"
}

// facadeTitle 板块展示名（生成注释用）。
func facadeTitle(f string) string {
	titles := map[string]string{
		"core": "核心（认证/项目/设置/杂项）", "office": "办公引擎与工作区",
		"memory": "记忆中枢与知识库", "cost": "成本库与价格源",
		"model": "模型中心与 Herdsman 底座", "voice": "语音（TTS/ASR/轻语）",
		"chat": "聊天与主脑", "novel": "小说写作", "image": "绘梦与媒体",
		"charlib": "角色库", "sin": "原罪（闲庭·图文故事创作）",
	}
	if t, ok := titles[f]; ok {
		return t
	}
	return f
}

// facadeType 门面类型名。
func facadeType(f string) string {
	return strings.ToUpper(f[:1]) + f[1:] + "B"
}

func writeFacade(facade string, ms []method) error {
	var b strings.Builder
	fmt.Fprintf(&b, "// Code generated by scripts/gen_bindings; DO NOT EDIT.\n\n")
	fmt.Fprintf(&b, "package app\n\n")
	// 收集签名中用到的外部包（按 import 别名引用）
	needed := map[string]bool{}
	re := regexp.MustCompile(`\b([A-Za-z_][A-Za-z0-9_]*)\.`)
	scan := func(s string) {
		for _, m := range re.FindAllStringSubmatch(s, -1) {
			if _, ok := globalImports[m[1]]; ok {
				needed[m[1]] = true
			}
		}
	}
	for _, m := range ms {
		scan(m.Params)
		scan(m.Results)
	}
	if len(needed) > 0 {
		names := make([]string, 0, len(needed))
		for n := range needed {
			names = append(names, n)
		}
		sort.Strings(names)
		b.WriteString("import (\n")
		for _, n := range names {
			p := globalImports[n]
			if base := p[strings.LastIndexByte(p, '/')+1:]; base != n {
				fmt.Fprintf(&b, "\t%s %q\n", n, p)
			} else {
				fmt.Fprintf(&b, "\t%q\n", p)
			}
		}
		b.WriteString(")\n\n")
	}
	fmt.Fprintf(&b, "// %s %s绑定门面（S2-3「App 绑定面拆分」）：仅暴露%s的方法，\n",
		facadeType(facade), facadeTitle(facade), facadeTitle(facade))
	fmt.Fprintf(&b, "// 方法体零改动——纯委托给 App 实例（b.a.<Method>）。\n")
	fmt.Fprintf(&b, "type %s struct{ a *App }\n\n", facadeType(facade))
	for _, m := range ms {
		// 结果：单值直接返回；多值含 error 也直接返回；无返回值直接调用。
		ret := ""
		if m.Results != "" {
			if m.HasParens {
				ret = " (" + m.Results + ") { return b.a." + m.Name + "(" + m.ArgNames + ") }"
			} else {
				ret = " " + m.Results + " { return b.a." + m.Name + "(" + m.ArgNames + ") }"
			}
		} else {
			ret = " { b.a." + m.Name + "(" + m.ArgNames + ") }"
		}
		fmt.Fprintf(&b, "func (b *%s) %s%s%s\n", facadeType(facade), m.Name, m.Params, ret)
	}
	path := filepath.Join("internal/app", "bindings_"+facade+".go")
	return os.WriteFile(path, []byte(b.String()), 0644)
}

func writeManifest(groups map[string][]method) error {
	var b strings.Builder
	b.WriteString("// Code generated by scripts/gen_bindings; DO NOT EDIT.\n\n")
	b.WriteString("package app\n\n")
	b.WriteString("// NewBindings 返回全部板块绑定门面（S2-3）：main.go 把这些对象传给\n")
	b.WriteString("// wails Bind，替代原来的单一 App 对象。方法按板块拆分，逻辑零改动。\n")
	b.WriteString("func NewBindings(a *App) []interface{} {\n")
	b.WriteString("\tif a == nil {\n\t\treturn nil\n\t}\n\treturn []interface{}{\n")
	for _, facade := range facadeOrder {
		if _, ok := groups[facade]; !ok {
			continue
		}
		fmt.Fprintf(&b, "\t\t&%s{a: a},\n", facadeType(facade))
	}
	b.WriteString("\t}\n}\n")
	path := filepath.Join("internal/app", "bindings_manifest.go")
	return os.WriteFile(path, []byte(b.String()), 0644)
}

func writeCompletenessTest(groups map[string][]method) error {
	var b strings.Builder
	b.WriteString("// Code generated by scripts/gen_bindings; DO NOT EDIT.\n\n")
	b.WriteString("package app\n\n")
	b.WriteString("import (\n\t\"reflect\"\n\t\"sort\"\n\t\"testing\"\n)\n\n")
	b.WriteString("// TestBindingsCompleteness S2-3 测试兜底：绑定门面集合的方法集必须与\n")
	b.WriteString("// App（含嵌入子状态提升）的导出方法集完全一致——不多、不少、无遗漏。\n")
	b.WriteString("func TestBindingsCompleteness(t *testing.T) {\n")
	b.WriteString("\tapp := &App{}\n")
	b.WriteString("\tvar want, got []string\n")
	b.WriteString("\t// *App 的完整方法集（含 core/writingState/mediaState/whisperState/officeState 提升）\n")
	b.WriteString("\tcollectExported(reflect.TypeOf(app), &want)\n")
	b.WriteString("\tfor _, bnd := range NewBindings(app) {\n")
	b.WriteString("\t\tcollectExported(reflect.TypeOf(bnd), &got)\n")
	b.WriteString("\t}\n")
	b.WriteString("\tsort.Strings(want)\n\tsort.Strings(got)\n")
	b.WriteString("\tif len(want) != len(got) {\n")
	b.WriteString("\t\tt.Fatalf(\"绑定面不一致：App 导出 %d 个方法，门面共 %d 个\", len(want), len(got))\n")
	b.WriteString("\t}\n")
	b.WriteString("\tfor i := range want {\n")
	b.WriteString("\t\tif want[i] != got[i] {\n")
	b.WriteString("\t\t\tt.Fatalf(\"绑定面不一致：App[%d]=%s vs 门面[%d]=%s\", i, want[i], i, got[i])\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t}\n")
	b.WriteString("}\n\n")
	b.WriteString("// collectExported 收集类型的导出方法名（指针类型方法集已含嵌入提升，勿 deref）。\n")
	b.WriteString("func collectExported(t reflect.Type, out *[]string) {\n")
	b.WriteString("\tfor i := 0; i < t.NumMethod(); i++ {\n")
	b.WriteString("\t\tm := t.Method(i)\n")
	b.WriteString("\t\tif m.PkgPath == \"\" {\n")
	b.WriteString("\t\t\t*out = append(*out, m.Name)\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t}\n")
	b.WriteString("}\n")
	path := filepath.Join("internal/app", "bindings_completeness_test.go")
	return os.WriteFile(path, []byte(b.String()), 0644)
}
