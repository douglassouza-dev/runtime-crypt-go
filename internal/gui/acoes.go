package gui

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync/atomic"

	"github.com/eufrauzino/runtime-crypt-go/internal/core"
	"github.com/eufrauzino/runtime-crypt-go/internal/gui/frases"
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

	// Demanda 028: peças de Sair, trocadas nos testes.
	saindo          atomic.Bool
	pendentesAoSair func() []core.PendenciaCofre
	enviarPendentes func(nomes []string) []core.FalhaEnvio
	perguntarSair   func(texto string) (enviar bool)
	avisar          func(titulo, texto string)
}

// NovasAcoes cria as ações da janela jp.
func NovasAcoes(g *core.GerenciadorRClone, jp *JanelaPrincipal) *Acoes {
	a := &Acoes{g: g, jp: jp}
	a.pendentesAoSair = g.PendentesAoSair
	a.enviarPendentes = func(nomes []string) []core.FalhaEnvio {
		return g.EnviarPendentes(nomes, func(nome string) (string, bool) {
			senha := DialogoSenha(jp.Janela(), nome, "Desbloquear", func(s string) error { return g.ConferirSenha(nome, s) })
			return senha, senha != ""
		})
	}
	a.perguntarSair = func(texto string) bool { return DialogoSairComPendencias(jp.Janela(), texto) }
	a.avisar = func(titulo, texto string) { DialogoMensagem(jp.Janela(), titulo, texto, MsgErro) }
	return a
}

// TituloNaoSaiu é o título do aviso quando "Enviar agora" não trancou tudo.
const TituloNaoSaiu = "Não deu para enviar"

// Sair é o pedido de sair do app (demanda 028). Se algum cofre caiu com
// arquivos que não subiram, pergunta antes: "Enviar agora" destranca de novo,
// espera o envio (025), tranca e só então sai; se algo falha, mostra a frase
// da 022/025 e não sai. "Sair" sai e deixa o cache como está. Sem pendência,
// sai direto. encerrar é o fim de verdade (Encerrar + Quit).
func (a *Acoes) Sair(encerrar func()) {
	if !a.saindo.CompareAndSwap(false, true) {
		return // já há um pedido de sair em andamento
	}
	defer a.saindo.Store(false)

	pendentes := a.pendentesAoSair()
	if len(pendentes) == 0 {
		encerrar()
		return
	}
	a.jp.Mostrar()
	if !a.perguntarSair(frases.AvisoAoSair(pendentes)) {
		encerrar()
		return
	}
	nomes := make([]string, 0, len(pendentes))
	for _, p := range pendentes {
		nomes = append(nomes, p.Nome)
	}
	feito := make(chan struct{})
	go a.jp.acompanharTrancar(feito)
	falhas := a.enviarPendentes(nomes)
	close(feito)
	if len(falhas) > 0 {
		linhas := make([]string, 0, len(falhas))
		for _, f := range falhas {
			linhas = append(linhas, frases.FalhaAoEnviar(f))
			if f.Etapa == core.EtapaTrancar {
				a.jp.MostrarFalhaTrancar(f.Nome, f.Err.Error())
			}
		}
		a.jp.ForcarAtualizacao()
		a.avisar(TituloNaoSaiu, strings.Join(linhas, "\n"))
		return
	}
	encerrar()
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
// destrancado; destranca o trancado, o que não subiu ("Tentar de novo") e o
// que caiu ("Destrancar de novo", 026: o rclone retoma os envios que ficaram
// no cache); ignora o que está destrancando.
func (a *Acoes) Cofre(nome string) {
	e := a.g.EstadoDoCofre(nome)
	switch e.Estado {
	case core.EstadoMontado:
		if e.Enviando > 0 {
			return // já está trancando, esperando o envio
		}
		a.trancar(nome)
	case core.EstadoMontando:
		return
	default:
		a.jp.LimparFalhaTrancar(nome)
		a.destrancar(nome)
	}
}

// Trancar é o botão Trancar do cofre que caiu (demanda 026). Também serve
// para o montado, como o clique no botão principal.
func (a *Acoes) Trancar(nome string) {
	e := a.g.EstadoDoCofre(nome)
	switch {
	case e.Estado == core.EstadoFalhou && e.Caiu:
		a.trancar(nome)
	case e.Estado == core.EstadoMontado && e.Enviando == 0:
		a.trancar(nome)
	}
}

// provedorDe é o nome do provedor do cofre, para as frases de erro (027).
func (a *Acoes) provedorDe(nome string) string {
	if c := a.g.Cofres.Obter(nome); c != nil {
		return c.ProvedorNome
	}
	return ""
}

func (a *Acoes) destrancar(nome string) {
	a.jp.Mostrar()
	// "Destrancando…" aparece assim que a montagem começa (demanda 018).
	go a.jp.atualizarLogoApos(nome)
	ponto, err := a.g.Destrancar(nome, func() (string, bool) {
		// Demanda 030: senha errada fica no diálogo, com "Senha errada."
		// embaixo do campo; o cofre continua trancado.
		senha := DialogoSenha(a.jp.Janela(), nome, "Desbloquear", func(s string) error { return a.g.ConferirSenha(nome, s) })
		return senha, senha != ""
	})
	switch {
	case errors.Is(err, core.ErrCancelado):
		return
	case errDriver(err):
		// Demanda 027: a mesma frase do card.
		DialogoMensagem(a.jp.Janela(), "Erro ao Destrancar", frases.TextoDriverAusente(runtime.GOOS), MsgErro)
	case err != nil:
		DialogoMensagem(a.jp.Janela(), "Erro ao Destrancar",
			fmt.Sprintf("Falha ao montar '%s':\n%s", nome, TextoErro(err, a.provedorDe(nome))), MsgErro)
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
	feito := make(chan struct{})
	go a.jp.acompanharTrancar(feito)
	err := a.g.Trancar(nome)
	close(feito)
	if err != nil {
		a.jp.MostrarFalhaTrancar(nome, err.Error())
		a.jp.Mostrar()
		return
	}
	a.jp.LimparFalhaTrancar(nome)
	DialogoMensagem(a.jp.Janela(), "Cofre Trancado", fmt.Sprintf("'%s' trancado.", nome), MsgInfo)
	a.jp.ForcarAtualizacao()
}

// Diálogo de sucesso dos wizards, com as palavras do wizard: nome do cofre,
// provedor e pasta. Sem "remoto" (018).
const (
	TextoCofreCriado    = "Cofre criado com sucesso!"
	TextoCofreImportado = "Cofre importado com sucesso!"
	textoCofrePronto    = "%s\n\nNome do cofre: %s\nProvedor: %s\nPasta: %s\n\nUse o botão 'Destrancar' para montar."
)

// TextoCofrePronto monta o diálogo de sucesso. A pasta vem de remotoBase
// ("x_base:Docs/cofre" → "/Docs/cofre"), escrita como o seletor de pasta
// escreve; sem pasta escolhida é a raiz, "/".
func TextoCofrePronto(titulo, nome, provedor, remotoBase string) string {
	pasta := remotoBase
	if i := strings.Index(pasta, ":"); i >= 0 {
		pasta = pasta[i+1:]
	}
	if !strings.HasPrefix(pasta, "/") {
		pasta = "/" + pasta
	}
	return fmt.Sprintf(textoCofrePronto, titulo, nome, provedor, pasta)
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
		TextoCofrePronto(TextoCofreCriado, r.Dados.Nome, r.Dados.Provedor.Nome, remotoBase),
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
		TextoCofrePronto(TextoCofreImportado, r.Dados.Nome, r.Dados.Provedor.Nome, remotoBase),
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

// errDriver diz se err é a falta do driver de montagem (demanda 027).
func errDriver(err error) bool {
	var rc *core.ErroRclone
	return errors.As(err, &rc) && rc.Falha == core.FalhaDriver
}
