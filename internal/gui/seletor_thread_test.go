package gui

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// Demanda 023: nenhuma goroutine de seletor_pasta.go mexe em widget direto.
// Dentro de um `go func`, chamadas que mudam a tela (Refresh, RemoveAll, Add,
// SetText, Show, Hide...) e atribuições a .Objects ou .Text só podem aparecer
// dentro de uma função passada a fyne.Do ou fyne.DoAndWait. É o mesmo critério
// de `rg -n "go func" internal/gui/seletor_pasta.go` lido à mão, só que
// conferido pelo compilador de Go.
func TestSeletorGoroutineNaoMexeNaTela(t *testing.T) {
	fset := token.NewFileSet()
	arq, err := parser.ParseFile(fset, "seletor_pasta.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}

	mexeNaTela := map[string]bool{
		"Refresh": true, "RemoveAll": true, "Add": true, "Remove": true,
		"SetText": true, "SetMinSize": true, "Show": true, "Hide": true,
		"Resize": true, "Enable": true, "Disable": true,
	}
	camposDaTela := map[string]bool{"Objects": true, "Text": true}

	goroutines := 0
	ast.Inspect(arq, func(n ast.Node) bool {
		g, ok := n.(*ast.GoStmt)
		if !ok {
			return true
		}
		goroutines++
		ast.Inspect(g.Call, func(m ast.Node) bool {
			switch v := m.(type) {
			case *ast.CallExpr:
				if sel, ok := v.Fun.(*ast.SelectorExpr); ok {
					if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Name == "fyne" && (sel.Sel.Name == "Do" || sel.Sel.Name == "DoAndWait") {
						return false // dentro de fyne.Do pode
					}
					if mexeNaTela[sel.Sel.Name] {
						t.Errorf("%s: goroutine chama %s direto", fset.Position(v.Pos()), sel.Sel.Name)
					}
				}
			case *ast.AssignStmt:
				for _, lhs := range v.Lhs {
					if sel, ok := lhs.(*ast.SelectorExpr); ok && camposDaTela[sel.Sel.Name] {
						t.Errorf("%s: goroutine atribui .%s direto", fset.Position(v.Pos()), sel.Sel.Name)
					}
				}
			}
			return true
		})
		return false
	})
	if goroutines == 0 {
		t.Fatal("nenhum `go func` achado: o teste não está olhando o arquivo certo")
	}
}
