package main

import (
	"go/ast"
	"strings"

	"github.com/MakeNowJust/enumcase"
	"github.com/timakin/bodyclose/passes/bodyclose"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/assign"
	"golang.org/x/tools/go/analysis/passes/atomic"
	"golang.org/x/tools/go/analysis/passes/bools"
	"golang.org/x/tools/go/analysis/passes/copylock"
	"golang.org/x/tools/go/analysis/passes/errorsas"
	"golang.org/x/tools/go/analysis/passes/httpresponse"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/analysis/passes/unreachable"
	"golang.org/x/tools/go/analysis/passes/unusedresult"
	"honnef.co/go/tools/staticcheck"
)

func main() {
	// staticcheck
	extraChecks := map[string]bool{
		"S1000": true,
		"S1001": true,
		"S1002": true,
		"S1005": true,
	}

	var checks []*analysis.Analyzer
	for _, v := range staticcheck.Analyzers {
		if strings.HasPrefix(v.Analyzer.Name, "SA") {
			checks = append(checks, v.Analyzer)
			continue
		}

		if extraChecks[v.Analyzer.Name] {
			checks = append(checks, v.Analyzer)
		}
	}

	// analysis/passes
	checks = append(checks,
		printf.Analyzer,
		shadow.Analyzer,
		structtag.Analyzer,
		assign.Analyzer,
		atomic.Analyzer,
		bools.Analyzer,
		copylock.Analyzer,
		unreachable.Analyzer,
		unusedresult.Analyzer,
		errorsas.Analyzer,
		httpresponse.Analyzer,
	)

	// двух или более любых публичных анализаторов на ваш выбор.
	checks = append(checks,
		bodyclose.Analyzer,
		enumcase.Analyzer,
	)

	checks = append(checks, OsExitAnalyzer)

	multichecker.Main(
		checks...,
	)
}

var OsExitAnalyzer = &analysis.Analyzer{
	Name: "osexitcheck",
	Doc:  "check for os.Exit() in main",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	expr := func(x *ast.ExprStmt) {
		if call, ok := x.X.(*ast.CallExpr); ok {
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
				if xxx, ok := sel.X.(*ast.Ident); ok && xxx.Name == "os" && sel.Sel.Name == "Exit" {
					pass.Reportf(x.Pos(), "don't use os.Exit() in main")
				}
			}
		}
	}

	for _, file := range pass.Files {
		ast.Inspect(file, func(node ast.Node) bool {
			switch x := node.(type) {
			case *ast.FuncDecl: // опрератор присваивания
				if x.Name.Name == "main" {
					for _, stmt := range x.Body.List {
						switch x := stmt.(type) {
						case *ast.ExprStmt: // выражение
							expr(x)
						}
					}
				}
			}
			return true
		})
	}
	return nil, nil
}
