package core

import "fmt"

// Arquivos escreve a quantidade com singular e plural: "1 arquivo",
// "3 arquivos". Toda frase com contagem de arquivos usa a mesma regra.
func Arquivos(n int) string {
	if n == 1 {
		return "1 arquivo"
	}
	return fmt.Sprintf("%d arquivos", n)
}
