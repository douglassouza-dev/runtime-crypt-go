package core

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// Palavras que, sem acento, denunciam um texto em português escrito sem
// acento ("nao", "invalido", "disponivel"...). "esta" sozinho fica de
// fora: "esta janela" e "esta pasta" estão certos. Vale para todo literal de
// string do app (internal/ e main.go), fora dos testes: quase todos chegam
// à tela ou ao log que o usuário manda quando algo dá errado.
var semAcento = regexp.MustCompile(`\b(nao|Nao|ja|Ja|esta corrompido|sera|possivel|invalido|invalida|numeros|espaco|comecar|hifen|[Nn]ecessario|disponivel|visivel|ilegivel|padrao|saida|codigo|executavel|usuario|[Cc]opia|gravacao|configuracao|[Rr]egiao|sessao|conexao|autorizacao|versao|diretorio|tambem|apos|Apos|mao)\b`)

// Literais que não são texto: nomes de arquivo e afins.
var literaisPermitidos = map[string]bool{
	".teste-gravacao-*": true,
}

func TestTextosDoAppTemAcento(t *testing.T) {
	raiz := filepath.Join("..", "..")
	var arquivos []string
	filepath.WalkDir(filepath.Join(raiz, "internal"), func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(p, ".go") && !strings.HasSuffix(p, "_test.go") {
			arquivos = append(arquivos, p)
		}
		return nil
	})
	arquivos = append(arquivos, filepath.Join(raiz, "main.go"))

	fset := token.NewFileSet()
	for _, arq := range arquivos {
		f, err := parser.ParseFile(fset, arq, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			s, err := strconv.Unquote(lit.Value)
			if err != nil || literaisPermitidos[s] {
				return true
			}
			if m := semAcento.FindString(s); m != "" {
				t.Errorf("%s: %q tem %q sem acento", fset.Position(lit.Pos()), s, m)
			}
			return true
		})
	}
}
