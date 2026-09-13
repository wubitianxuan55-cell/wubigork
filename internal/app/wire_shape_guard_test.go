package app

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// wireShapeGuardTest 所需的轻量扫描器：绑定面（App 导出方法）签名类型的
// 序列化可达闭包内，不允许出现缺 json 标签的 struct——Wails 线上序列化按
// encoding/json 走，缺标签即输出 PascalCase，前端 camelCase 契约读空
// （v4.269 costproject/coststage、v4.276 priceband/costref 两轮同款线上断链）。
//
// 豁免清单：确属「前端永不消费」的可达类型，加名字+理由；宁可补标签不要加豁免
// （输入方向补标签亦无害——Unmarshal 本就大小写不敏）。
var wireShapeExemptions = map[string]string{}

type guardStruct struct {
	name    string
	missing []string
}

type guardSrcFile struct {
	path string
	pkg  string
	file *ast.File
}

func guardRepoRoot(t *testing.T) string {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	dir := filepath.Dir(thisFile) // internal/app
	for i := 0; i < 4; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	t.Fatal("go.mod not found")
	return ""
}

func TestWireShapeGuard(t *testing.T) {
	root := guardRepoRoot(t)
	fset := token.NewFileSet()
	var files []guardSrcFile
	structs := map[string]*guardStruct{}  // pkg.Name -> def
	structFields := map[string][]string{} // pkg.Name -> 字段类型表达式

	skipDir := func(name string) bool {
		switch name {
		case "node_modules", "clones", "testdata", "wailsjs":
			return true
		}
		return false
	}
	_ = filepath.WalkDir(filepath.Join(root, "internal"), func(path string, d os.DirEntry, err error) error {
		if err != nil || d == nil {
			return nil
		}
		if d.IsDir() {
			if skipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		return loadGuardFile(fset, path, &files, structs, structFields)
	})
	_ = filepath.WalkDir(filepath.Join(root, "shared"), func(path string, d os.DirEntry, err error) error {
		if err != nil || d == nil {
			return nil
		}
		if d.IsDir() {
			if skipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		return loadGuardFile(fset, path, &files, structs, structFields)
	})

	// 绑定面：internal/app 中 App 的导出方法签名
	reachable := map[string]bool{}
	queue := []string{}
	for _, f := range files {
		if !strings.HasSuffix(f.path, filepath.ToSlash(filepath.Join("internal", "app"))+string(filepath.Separator)) &&
			!strings.Contains(filepath.ToSlash(f.path), "/internal/app/") {
			continue
		}
		for _, decl := range f.file.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || fd.Recv == nil || !fd.Name.IsExported() {
				continue
			}
			if st, ok := fd.Recv.List[0].Type.(*ast.StarExpr); ok {
				if ident, ok := st.X.(*ast.Ident); !ok || ident.Name != "App" {
					continue
				}
			} else {
				continue
			}
			collect := func(e ast.Expr) {
				for _, ty := range guardSigTypes(e, f.pkg) {
					if !reachable[ty] {
						reachable[ty] = true
						queue = append(queue, ty)
					}
				}
			}
			if fd.Type.Params != nil {
				for _, p := range fd.Type.Params.List {
					collect(p.Type)
				}
			}
			if fd.Type.Results != nil {
				for _, r := range fd.Type.Results.List {
					collect(r.Type)
				}
			}
		}
	}

	seen := map[string]bool{}
	var violations []string
	for len(queue) > 0 {
		key := queue[0]
		queue = queue[1:]
		if seen[key] {
			continue
		}
		seen[key] = true
		gs := structs[key]
		if gs != nil {
			if len(gs.missing) > 0 {
				if _, exempt := wireShapeExemptions[gs.name]; !exempt {
					violations = append(violations, gs.name+" missing tags: "+strings.Join(gs.missing, ","))
				}
			}
			for _, fldType := range structFields[key] {
				ty := strings.TrimPrefix(fldType, "map:")
				if ty != "" && !reachable[ty] {
					reachable[ty] = true
					queue = append(queue, ty)
				}
			}
		}
	}

	if len(violations) > 0 {
		t.Errorf("绑定面可达类型存在缺 json 标签的 struct（Wails 线上输出 PascalCase，前端 camelCase 读空）:\n  %s\n补齐 json 标签（对齐前端类型定义），或在前述豁免清单登记+写明前端不消费的理由",
			strings.Join(violations, "\n  "))
	}
}

func loadGuardFile(fset *token.FileSet, path string, files *[]guardSrcFile, structs map[string]*guardStruct, fields map[string][]string) error {
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil // 解析失败的文件交给编译器报错
	}
	pkg := f.Name.Name
	*files = append(*files, guardSrcFile{path: filepath.ToSlash(path), pkg: pkg, file: f})
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			continue
		}
		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				continue
			}
			gs := &guardStruct{name: ts.Name.Name}
			for _, fld := range st.Fields.List {
				if len(fld.Names) == 0 || !fld.Names[0].IsExported() {
					continue
				}
				if fld.Tag == nil || !strings.Contains(fld.Tag.Value, "json:") {
					gs.missing = append(gs.missing, fld.Names[0].Name)
				}
				fields[pkg+"."+ts.Name.Name] = append(fields[pkg+"."+ts.Name.Name], guardTypeExpr(fld.Type, pkg))
			}
			structs[pkg+"."+ts.Name.Name] = gs
		}
	}
	return nil
}

func guardTypeExpr(e ast.Expr, curPkg string) string {
	switch t := e.(type) {
	case *ast.Ident:
		return curPkg + "." + t.Name
	case *ast.StarExpr:
		return guardTypeExpr(t.X, curPkg)
	case *ast.ArrayType:
		return guardTypeExpr(t.Elt, curPkg)
	case *ast.SelectorExpr:
		if x, ok := t.X.(*ast.Ident); ok {
			return x.Name + "." + t.Sel.Name
		}
	case *ast.MapType:
		return "map:" + guardTypeExpr(t.Value, curPkg)
	}
	return ""
}

func guardSigTypes(e ast.Expr, curPkg string) []string {
	switch t := e.(type) {
	case *ast.Ident:
		switch t.Name {
		case "error", "string", "int", "int64", "bool", "float64", "byte", "rune", "any":
			return nil
		}
		return []string{curPkg + "." + t.Name}
	case *ast.StarExpr:
		return guardSigTypes(t.X, curPkg)
	case *ast.ArrayType:
		return guardSigTypes(t.Elt, curPkg)
	case *ast.MapType:
		return guardSigTypes(t.Value, curPkg)
	case *ast.SelectorExpr:
		if x, ok := t.X.(*ast.Ident); ok {
			return []string{x.Name + "." + t.Sel.Name}
		}
	}
	return nil
}
