//go:build windows

package core

import "errors"

// pontoAtivo diz se a letra da unidade existe (WinFsp).
func pontoAtivo(caminho string) bool { return caminhoExiste(caminho) }

// desmontarPontoFuse não se aplica ao Windows: lá a unidade some com o
// processo.
func desmontarPontoFuse(string) error {
	return errors.New("desmontar o ponto à mão não se aplica ao Windows")
}
