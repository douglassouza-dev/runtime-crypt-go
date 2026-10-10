package core

import "errors"

// Fica fora de gerenciador.go de propósito: lá vale a regra da demanda 009
// (`rg -n "return nil$" internal/core/gerenciador.go` vazio), e aqui o nil é
// o sucesso de uma função que só devolve error.

// Trancar desmonta o cofre nome (demandas 022, 025 e 026). A senha da sessão
// só é apagada quando o cofre não tem mais montagem nenhuma, ou seja, quando
// a tela passa a mostrar "Trancado". Se a desmontagem não terminou, a senha
// fica e o erro traz o motivo.
//
// Cofre que caiu (026) não conta como desmontado: com o rclone morto, se o
// cache da VFS ainda tem arquivo que não subiu, Trancar devolve
// *ErroNaoSubiram e não mexe em nada.
func (g *GerenciadorRClone) Trancar(nome string) error {
	e := g.EstadoDoCofre(nome)
	var err error
	switch {
	case e.Estado == EstadoFalhou && e.Caiu:
		err = g.Montagens.DesmontarQueda(e.Letra)
	default:
		if letra := g.Montagens.ObterLetraPorRemoto(nome); letra != "" {
			if ok, msg := g.Montagens.DesmontarUnidade(letra); !ok {
				err = errors.New(msg)
			}
		}
	}
	if !g.Montagens.temMontagem(nome) {
		g.Senhas.Limpar(nome)
	}
	return err
}
