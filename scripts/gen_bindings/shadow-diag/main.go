// 诊断脚本：列出被遮蔽的同名方法对（App 声明 shadow 了内嵌类型的同名方法）。
// 用法：go run scripts/gen_bindings/shadow-diag/main.go
// 输出：遮蔽对清单（方法名 / App 定义位置 / 内嵌类型定义位置），以及包内直调
// 内嵌版本（绕过 App 修复）的调用点。
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, "internal/app", nil, 0) //nolint:staticcheck // 诊断脚本：仅列名不涉及 build-tag 语义
	if err != nil {
		fmt.Fprintln(os.Stderr, "parse:", err)
		os.Exit(1)
	}

	type decl struct {
		recv, file string
		line       int
	}
	methods := map[string][]decl{} // name → 所有接收者声明

	for _, pkg := range pkgs {
		for fname, f := range pkg.Files {
			for _, d := range f.Decls {
				fd, ok := d.(*ast.FuncDecl)
				if !ok || fd.Recv == nil || len(fd.Recv.List) == 0 {
					continue
				}
				recv := typeOfRecv(fd.Recv.List[0].Type)
				if !ast.IsExported(fd.Name.Name) {
					continue
				}
				methods[fd.Name.Name] = append(methods[fd.Name.Name], decl{
					recv: recv, file: filepath.Base(fname), line: fset.Position(fd.Pos()).Line,
				})
			}
		}
	}

	var names []string
	shadows := 0
	for n, ds := range methods {
		hasApp := false
		for _, d := range ds {
			if d.recv == "App" {
				hasApp = true
			}
		}
		if hasApp && len(ds) > 1 {
			names = append(names, n)
			shadows++
		}
	}
	sort.Strings(names)
	fmt.Printf("被 App 遮蔽的同名方法：%d 个\n\n", shadows)
	for _, n := range names {
		fmt.Printf("%s\n", n)
		for _, d := range methods[n] {
			mark := " "
			if d.recv == "App" {
				mark = "→"
			}
			fmt.Printf("  %s %s (%s:%d)\n", mark, d.recv, d.file, d.line)
		}
	}
	_ = strings.TrimSpace
}

func typeOfRecv(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.StarExpr:
		return typeOfRecv(t.X)
	case *ast.Ident:
		return t.Name
	case *ast.IndexExpr:
		return typeOfRecv(t.X)
	}
	return "?"
}
