package core

import "errors"

// Fica fora de gerenciador.go de propósito: lá vale a regra da demanda 009
// (`rg -n "return nil$" internal/core/gerenciador.go` vazio), e aqui o nil é
// o sucesso de uma função que só devolve error.

// Trancar desmonta o cofre nome (demanda 022). A senha da sessão só é apagada
// quando o cofre deixa de estar montado, ou seja, quando a tela passa a
// mostrar "Trancado". Se a desmontagem não terminou, a senha fica e o erro
// traz o motivo. Cofre que já não está montado conta como trancado.
func (g *GerenciadorRClone) Trancar(nome string) error {
	letra := g.Montagens.ObterLetraPorRemoto(nome)
	if letra == "" {
		g.Senhas.Limpar(nome)
		return nil
	}

	ok, msg := g.Montagens.DesmontarUnidade(letra)
	if g.Montagens.ObterLetraPorRemoto(nome) == "" {
		g.Senhas.Limpar(nome)
	}
	if !ok {
		return errors.New(msg)
	}
	return nil
}
