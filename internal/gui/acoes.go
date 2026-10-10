package gui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/eufrauzino/runtime-crypt-go/internal/core"
	"github.com/eufrauzino/runtime-crypt-go/internal/plataforma"
)

// Acoes liga os botões da janela e da bandeja aos casos de uso do core
// (demanda 017). Aqui só se coleta entrada, chama o core e mostra o resultado.
type Acoes struct {
	g  *core.GerenciadorRClone
	jp *JanelaPrincipal

	// progresso é o diálogo de passos da criação/conexão em andamento
	// (demanda 018); nil fora delas.
	progresso *progressoWizard
}

// NovasAcoes cria as ações da janela jp.
func NovasAcoes(g *core.GerenciadorRClone, jp *JanelaPrincipal) *Acoes {
	return &Acoes{g: g, jp: jp}
}

// AbrirAutorizacao implementa core.Interacao: abre o navegador na URL do
// OAuth; se não abrir, mostra a URL para abrir à mão (demanda 009).
func (a *Acoes) AbrirAutorizacao(url string) {
	if err := core.AbrirNavegador(url); err != nil {
		DialogoMensagem(a.jp.Janela(), "Autorização",
			fmt.Sprintf("Não deu para abrir o navegador: %v\n\nAbra este endereço no navegador:\n%s\n\nApós concluir, volte para esta janela.", err, url), MsgAviso)
		return
	}
	DialogoMensagem(a.jp.Janela(), "Autorização",
		"O navegador foi aberto para autorização.\nApós concluir, volte para esta janela.", MsgInfo)
}

// Passo implementa core.Interacao: mostra o passo atual (demanda 018).
func (a *Acoes) Passo(p core.Passo) {
	if a.progresso != nil {
		a.progresso.passo(p)
	}
}

// EscolherPasta implementa core.Interacao com o seletor de pasta.
func (a *Acoes) EscolherPasta(remotoBase, tituloProvedor string) (string, bool) {
	if a.progresso != nil {
		a.progresso.esconder()
	}
	caminho := DialogoSeletorPastaRemota(a.jp.Janela(), a.g, remotoBase, tituloProvedor)
	if caminho == nil {
		return "", false
	}
	return *caminho, true
}

// Cofre age conforme o estado do cofre no core (demanda 018): tranca o
// destrancado; destranca o trancado ou o que não subiu/caiu ("Tentar de
// novo"); ignora o que está destrancando.
func (a *Acoes) Cofre(nome string) {
	switch a.g.EstadoDoCofre(nome).Estado {
	case core.EstadoMontado:
		a.trancar(nome)
	case core.EstadoMontando:
		return
	default:
		a.jp.LimparFalhaTrancar(nome)
		a.destrancar(nome)
	}
}

func (a *Acoes) destrancar(nome string) {
	a.jp.Mostrar()
	// "Destrancando…" aparece assim que a montagem começa (demanda 018).
	go a.jp.atualizarLogoApos(nome)
	ponto, err := a.g.Destrancar(nome, func() (string, bool) {
		senha := DialogoSenha(a.jp.Janela(), nome, "Desbloquear")
		return senha, senha != ""
	})
	switch {
	case errors.Is(err, core.ErrCancelado):
		return
	case err != nil:
		DialogoMensagem(a.jp.Janela(), "Erro ao Destrancar",
			fmt.Sprintf("Falha ao montar '%s':\n%s", nome, err), MsgErro)
	default:
		// Demanda 013: o Explorador abre o ponto real (letra ou pasta).
		texto := fmt.Sprintf("'%s' montado em %s\n\nO Explorador de Arquivos foi aberto.", nome, ponto)
		if err := core.AbrirExplorador(ponto); err != nil {
			// Demanda 009: a falha ao abrir o Explorador chega ao usuário.
			texto = fmt.Sprintf("'%s' montado em %s\n\nNão deu para abrir o Explorador: %v", nome, ponto, err)
		}
		DialogoMensagem(a.jp.Janela(), "Cofre Destrancado", texto, MsgInfo)
	}
	a.jp.ForcarAtualizacao()
}

// trancar: se a desmontagem não terminou, o card continua destrancado com
// "Não trancou: {motivo}" (demanda 022).
func (a *Acoes) trancar(nome string) {
	if err := a.g.Trancar(nome); err != nil {
		a.jp.MostrarFalhaTrancar(nome, err.Error())
		a.jp.Mostrar()
		return
	}
	a.jp.LimparFalhaTrancar(nome)
	DialogoMensagem(a.jp.Janela(), "Cofre Trancado", fmt.Sprintf("'%s' trancado.", nome), MsgInfo)
	a.jp.ForcarAtualizacao()
}

// NovoCofre roda o wizard e o caso de uso CriarCofre.
func (a *Acoes) NovoCofre() {
	a.jp.Mostrar()
	r := DialogoNovoCofre(a.jp.Janela())
	if !r.Sucesso {
		return
	}
	a.progresso = novoProgressoWizard(a.jp.Janela(), "Novo cofre", r.Dados.Provedor.Nome)
	remotoBase, err := a.g.CriarCofre(r.Dados, a)
	a.progresso.esconder()
	a.progresso = nil
	if err != nil {
		if !errors.Is(err, core.ErrCancelado) {
			DialogoMensagem(a.jp.Janela(), "Erro", mensagemErroPasso(err, r.Dados.Provedor.Nome), MsgErro)
		}
		return
	}
	DialogoMensagem(a.jp.Janela(), "Sucesso",
		fmt.Sprintf("Cofre '%s' criado com sucesso!\n\nProvedor: %s\nRemoto base: %s\n\nUse o botão 'Destrancar' para montar.", r.Dados.Nome, r.Dados.Provedor.Nome, remotoBase),
		MsgInfo)
	a.jp.ForcarAtualizacao()
}

// ImportarCofre roda o wizard e o caso de uso ConectarCofre.
func (a *Acoes) ImportarCofre() {
	a.jp.Mostrar()
	r := DialogoImportarCofre(a.jp.Janela())
	if !r.Sucesso {
		return
	}
	a.progresso = novoProgressoWizard(a.jp.Janela(), "Conectar cofre", r.Dados.Provedor.Nome)
	remotoBase, err := a.g.ConectarCofre(r.Dados, a)
	a.progresso.esconder()
	a.progresso = nil
	if err != nil {
		if !errors.Is(err, core.ErrCancelado) {
			DialogoMensagem(a.jp.Janela(), "Erro", mensagemErroPasso(err, r.Dados.Provedor.Nome), MsgErro)
		}
		return
	}
	DialogoMensagem(a.jp.Janela(), "Sucesso",
		fmt.Sprintf("Cofre '%s' importado com sucesso!\n\nProvedor: %s\nRemoto: %s\n\nUse o botão 'Destrancar' para montar.", r.Dados.Nome, r.Dados.Provedor.Nome, remotoBase),
		MsgInfo)
	a.jp.ForcarAtualizacao()
}

// AutoMontar monta os cofres marcados; cada falha aparece com o nome do
// cofre (demanda 009).
func (a *Acoes) AutoMontar() {
	for nome, err := range a.g.AutoMontar() {
		DialogoMensagem(a.jp.Janela(), "Erro ao Auto-montar",
			fmt.Sprintf("Falha ao montar '%s':\n%s", nome, err), MsgErro)
	}
}

// AlternarAutoIniciar liga ou desliga o auto-início e mostra o erro, se houver.
func (a *Acoes) AlternarAutoIniciar() {
	var err error
	if plataforma.VerificarAutoIniciar() {
		err = plataforma.RemoverAutoIniciar()
	} else {
		err = plataforma.AdicionarAutoIniciar()
	}
	if err != nil {
		a.jp.Mostrar()
		DialogoMensagem(a.jp.Janela(), "Erro no Auto-iniciar", err.Error(), MsgErro)
	}
}

// VerificarFuse diz se o WinFsp/FUSE está instalado.
func (a *Acoes) VerificarFuse() {
	info := plataforma.VerificarWinfsp()
	if info.Instalado {
		DialogoMensagem(a.jp.Janela(), "WinFsp/FUSE", "WinFsp/FUSE está instalado e funcionando.", MsgInfo)
		return
	}
	DialogoMensagem(a.jp.Janela(), "WinFsp/FUSE Ausente",
		fmt.Sprintf("%s\n\nBaixe em: %s", info.Motivo, info.UrlDownload), MsgAviso)
}

// Sobre mostra a versão e a licença.
func (a *Acoes) Sobre() {
	a.jp.Mostrar()
	DialogoMensagem(a.jp.Janela(), "RuntimeCrypto", fmt.Sprintf(
		"RuntimeCrypto — Cofre Criptografado na Nuvem\n\n"+
			"Versão %s\n"+
			"Escrito em Go — Binário nativo multiplataforma\n\n"+
			"Usa RClone + Crypt para criptografia ponta-a-ponta.\n"+
			"Seus arquivos são criptografados antes de enviados à nuvem\n"+
			"e descriptografados instantaneamente no seu PC.\n\n"+
			"Licença AGPL-3.0 — Copyright (c) Douglas Eufrauzino de Souza",
		core.Versao), MsgInfo)
}

// AvisosDaAbertura mostra o erro de vaults.json (demanda 007) e os avisos de
// vfs.json (demanda 012), se houver.
func (a *Acoes) AvisosDaAbertura() {
	if a.g.ErroCofres != nil {
		DialogoMensagem(a.jp.Janela(), "Erro ao ler os cofres", a.g.ErroCofres.Error(), MsgErro)
	}
	if avisos := a.g.Vfs.Avisos(); len(avisos) > 0 {
		// Demanda 012: valor inválido em vfs.json volta ao padrão com aviso.
		DialogoMensagem(a.jp.Janela(), "Configurações VFS", strings.Join(avisos, "\n"), MsgAviso)
	}
}
