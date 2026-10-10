package core

import (
	"os"
	"regexp"
	"strings"
)

// Como o ponto de montagem aparece em mensagens (demanda 013): no Windows a
// unidade (`V:\`); fora dele a pasta, com o home abreviado para `~`.

var reUnidade = regexp.MustCompile(`^[A-Za-z]:\\?$`)

// EhUnidade diz se o ponto é uma letra de unidade (`V:` ou `V:\`).
func EhUnidade(ponto string) bool { return reUnidade.MatchString(ponto) }

// diretorioHome é o home do usuário; os testes trocam.
var diretorioHome = func() string {
	h, _ := os.UserHomeDir()
	return h
}

// TextoPonto é o ponto como a tela escreve: `V:\`, ou a pasta com `~` no
// lugar do home (ex.: `~/RuntimeCrypto/fotos`).
func TextoPonto(ponto string) string {
	if ponto == "" || EhUnidade(ponto) {
		return ponto
	}
	h := strings.TrimRight(diretorioHome(), `/\`)
	if h == "" {
		return ponto
	}
	if ponto == h {
		return "~"
	}
	for _, sep := range []string{"/", `\`} {
		if strings.HasPrefix(ponto, h+sep) {
			return "~" + ponto[len(h):]
		}
	}
	return ponto
}

// RotuloPonto é "a unidade V:\" ou "a pasta ~/RuntimeCrypto/fotos".
func RotuloPonto(ponto string) string {
	if EhUnidade(ponto) {
		return "a unidade " + ponto
	}
	return "a pasta " + TextoPonto(ponto)
}

// maiuscula põe a primeira letra em maiúscula.
func maiuscula(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
