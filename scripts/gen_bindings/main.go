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
var facadeOrder = []string{"core", "office", "memory", "cost", "model", "voice", "chat", "novel", "image", "charlib"}

type method struct {
	Name   string
	Params string // "(a1 T1, a2 T2)" 含括号，用于签名与调用
	ArgNames string // "a1, a2"
	Results string // "" | "R" | "R, error" 等（不含括号）… 见下
	HasParens bool // 结果是否有括号（多返回值）
	Receiver string
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
	"GetModelRoute":      "model",
	"GetSensitiveLocal":  "model",
	"GetOfficeLocal":     "model",
	"SetSensitiveLocal":  "model",
	"SetOfficeLocal":     "model",
	"GetModelMonitor":    "model",
	"SetFeatureModel":    "model",
	"SetFeatureModelEnabled": "model",
	"GetEngines":         "model",
	"GetActiveEngine":    "model",
	"GetActiveModel":     "model",
	"GetEngineStatus":    "model",
	"GetModelCatalog":    "model",
	"Startup":            "core",
	"Shutdown":           "core",
	"SetDistFS":          "core",
	"SetPromptFS":        "core",
	"Login":              "core",
	"GetLoginStatus":     "core",
	"Logout":             "core",
	"SaveToken":          "core",
	"GenerateCharacterPortrait": "image",
	"SetCharacterPortrait":      "image",
	"CmdKEdit":           "novel",
	"LocalTranslate":     "office",
	"Search":             "core",
	"ExportAll":          "core",
	"GetConfig":          "core",
	"SaveConfig":         "core",
	"ListSkills":         "core",
	"GetStats":           "core",
	"CreateProject":      "core",
	"OpenProject":        "core",
	"CloseProject":       "core",
	"GetProjectInfo":     "core",
	"GetNovelsDir":       "core",
	"ListProjects":       "core",
	"DeleteProject":      "core",
	"GetCompileTemplates": "core",
	"ExportHTML":         "core",
	"GetDashboard":       "core",
	"AnalyzeStyle":       "core",
	"GetStyleProfile":    "core",
	"ImportStyleProfile": "core",
	"ChatGeneral":        "chat",
	"MainBrainChat":      "chat",
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
}

func main() {
	namesOnly := flag.Bool("names", false, "只输出全部导出方法名（一行一个，稳定排序），不写任何生成文件")
	flag.Parse()

	dir := "internal/app"
	methods, err := collectMethods(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "collect:", err)
		os.Exit(1)
	}
	if len(methods) == 0 {
		fmt.Fprintln(os.Stderr, "no methods collected")
		os.Exit(1)
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
		return
	}

	// 按板块分组
	groups := map[string][]method{}
	for _, m := range methods {
		f := mapMethod(m)
		if f == "" {
			fmt.Fprintf(os.Stderr, "方法 %s 未映射到板块\n", m.Name)
			os.Exit(1)
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
			os.Exit(1)
		}
		fmt.Printf("%-10s %3d 个方法\n", facade, len(ms))
	}
	if err := writeManifest(groups); err != nil {
		fmt.Fprintln(os.Stderr, "write manifest:", err)
		os.Exit(1)
	}
	if err := writeCompletenessTest(groups); err != nil {
		fmt.Fprintln(os.Stderr, "write test:", err)
		os.Exit(1)
	}
	fmt.Printf("合计 %d 个导出方法 → %d 个绑定门面\n", len(methods), len(groups))
}

// collectMethods 解析 internal/app 下所有非测试 .go 文件，收集绑定面方法的签名。
// 同时收集 import 名→路径映射（生成门面文件需要）。
func collectMethods(dir string) ([]method, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
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
			return nil, fmt.Errorf("%s: %w", path, err)
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
			m := method{Name: fd.Name.Name, Receiver: ident.Name}
			// 参数
			var params, args []string
			for _, p := range fd.Type.Params.List {
				typeStr := exprString(p.Type)
				if len(p.Names) == 0 {
					params = append(params, "_ "+typeStr)
					args = append(args, "_")
					continue
				}
				for _, nm := range p.Names {
					params = append(params, nm.Name+" "+typeStr)
					args = append(args, nm.Name)
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
	for _, m := range out {
		if seen[m.Name] {
			continue
		}
		if hasApp[m.Name] && m.Receiver != "App" {
			continue // 被 App 版本 shadow；不标记 seen，App 版本随后占用
		}
		seen[m.Name] = true
		dedup = append(dedup, m)
	}
	globalImports = imports
	return dedup, nil
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
		"charlib": "角色库",
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
