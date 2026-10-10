package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/theme"

	"github.com/eufrauzino/runtime-crypt-go/internal/core"
	"github.com/eufrauzino/runtime-crypt-go/internal/gui"
	"github.com/eufrauzino/runtime-crypt-go/internal/plataforma"
	"github.com/eufrauzino/runtime-crypt-go/internal/tray"
)

//go:embed assets/icone.ico
var iconeBytes []byte

func main() {
	// Inicializar gerenciador principal
	gerenciador := core.NovoGerenciador()

	// Canal de ações do tray
	canalAcoes := make(chan tray.AcaoTray, 32)

	// Criar aplicação Fyne
	aplicacao := app.NewWithID("com.eufrauzino.runtime-crypto")
	aplicacao.Settings().SetTheme(&temaRuntime{})

	// Criar janela principal
	janelaPrincipal := gui.NovaJanelaPrincipal(aplicacao, gerenciador)

	// Configurar callbacks da janela principal
	janelaPrincipal.CallbackCofre = func(cofre core.CofreStatus) {
		go acaoCofre(gerenciador, janelaPrincipal, cofre)
	}
	janelaPrincipal.CallbackNovoCofre = func() {
		go acaoNovoCofre(gerenciador, janelaPrincipal)
	}
	janelaPrincipal.CallbackImportarCofre = func() {
		go acaoImportarCofre(gerenciador, janelaPrincipal)
	}
	janelaPrincipal.CallbackConfigVfs = func() {
		go gui.DialogoConfigVfs(janelaPrincipal.Janela(), gerenciador)
	}
	janelaPrincipal.CallbackVerificarFuse = func() {
		go acaoVerificarFuse(janelaPrincipal)
	}
	janelaPrincipal.CallbackSobre = func() {
		gui.DialogoMensagem(
			janelaPrincipal.Janela(),
			"RuntimeCrypto",
			fmt.Sprintf(
				"RuntimeCrypto — Cofre Criptografado na Nuvem\n\n"+
					"Versão %s\n"+
					"Escrito em Go — Binário nativo multiplataforma\n\n"+
					"Usa RClone + Crypt para criptografia ponta-a-ponta.\n"+
					"Seus arquivos são criptografados antes de enviados à nuvem\n"+
					"e descriptografados instantaneamente no seu PC.\n\n"+
					"Licença AGPL-3.0 — Copyright (c) Douglas Eufrauzino de Souza",
				core.Versao,
			),
			gui.MsgInfo,
		)
	}
	janelaPrincipal.CallbackSair = func() {
		gerenciador.Encerrar()
		aplicacao.Quit()
	}

	// Iniciar tray em goroutine
	gerenciadorTray := tray.NovoGerenciadorTray(gerenciador, canalAcoes, iconeBytes)
	go gerenciadorTray.Iniciar(nil, nil)

	// Processar ações do tray em goroutine
	go func() {
		for acao := range canalAcoes {
			switch acao.Tipo {
			case tray.AcaoMostrarJanela:
				janelaPrincipal.Mostrar()
			case tray.AcaoCofre:
				cofre := gerenciador.Cofres.Obter(acao.Dados)
				if cofre != nil {
					status := core.CofreStatus{Cofre: *cofre}
					go acaoCofre(gerenciador, janelaPrincipal, status)
				}
			case tray.AcaoNovoCofre:
				go acaoNovoCofre(gerenciador, janelaPrincipal)
			case tray.AcaoConfigVfs:
				janelaPrincipal.Mostrar()
				go gui.DialogoConfigVfs(janelaPrincipal.Janela(), gerenciador)
			case tray.AcaoVerificarFuse:
				go acaoVerificarFuse(janelaPrincipal)
			case tray.AcaoSobre:
				janelaPrincipal.Mostrar()
				janelaPrincipal.CallbackSobre()
			case tray.AcaoAutoIniciar:
				go alternarAutoIniciar(janelaPrincipal)
			case tray.AcaoSair:
				gerenciador.Encerrar()
				aplicacao.Quit()
			}
		}
	}()

	// Auto-montar cofres configurados
	go func() {
		time.Sleep(1500 * time.Millisecond)
		autoMontarCofres(gerenciador, janelaPrincipal)
	}()

	// Mostrar janela e iniciar mainloop
	janelaPrincipal.Mostrar()
	if gerenciador.ErroCofres != nil {
		// Demanda 007: vaults.json ilegível não é sobrescrito; o usuário vê o motivo.
		go gui.DialogoMensagem(janelaPrincipal.Janela(), "Erro ao ler os cofres",
			gerenciador.ErroCofres.Error(), gui.MsgErro)
	}
	aplicacao.Run()
}

// acaoCofre processa o clique em um cofre (destrancar ou trancar).
func acaoCofre(gerenciador *core.GerenciadorRClone, jp *gui.JanelaPrincipal, cofre core.CofreStatus) {
	montagens := gerenciador.Montagens.ObterMontagens()
	_, montado := montagens[cofre.Nome]

	if montado {
		_ = travarCofre(gerenciador, jp, cofre.Nome) // o erro já está no card
	} else {
		jp.LimparFalhaTrancar(cofre.Nome)
		destravarCofre(gerenciador, jp, cofre.Nome)
	}
}

// destravarCofre solicita senha e monta o cofre.
func destravarCofre(gerenciador *core.GerenciadorRClone, jp *gui.JanelaPrincipal, nome string) {
	jp.Mostrar()

	senha := gerenciador.Senhas.Obter(nome)
	if senha == "" {
		senha = gui.DialogoSenha(jp.Janela(), nome, "Desbloquear")
	}
	if senha == "" {
		return
	}

	gerenciador.Senhas.Armazenar(nome, senha)
	sucesso, msg, letra := gerenciador.Montagens.MontarUnidade(nome, "", senha, nil)

	if sucesso && letra != "" {
		texto := fmt.Sprintf("'%s' montado em %s:\\\n\nO Explorador de Arquivos foi aberto.", nome, letra)
		if err := core.AbrirExplorador(letra + ":\\"); err != nil {
			// Demanda 009: a falha ao abrir o Explorador chega ao usuário.
			texto = fmt.Sprintf("'%s' montado em %s:\\\n\nNão deu para abrir o Explorador: %v", nome, letra, err)
		}
		gui.DialogoMensagem(jp.Janela(), "Cofre Destrancado", texto, gui.MsgInfo)
	} else {
		gerenciador.Senhas.Limpar(nome)
		gui.DialogoMensagem(
			jp.Janela(),
			"Erro ao Destrancar",
			fmt.Sprintf("Falha ao montar '%s':\n%s", nome, msg),
			gui.MsgErro,
		)
	}

	jp.ForcarAtualizacao()
}

// travarCofre desmonta o cofre (demanda 022). Se a desmontagem não terminou,
// o card continua destrancado com "Não trancou: {motivo}" e a senha da
// sessão fica; ela só sai quando o cofre chega a "Trancado" (core.Trancar).
func travarCofre(gerenciador *core.GerenciadorRClone, jp *gui.JanelaPrincipal, nome string) error {
	if err := gerenciador.Trancar(nome); err != nil {
		jp.MostrarFalhaTrancar(nome, err.Error())
		jp.Mostrar()
		return err
	}
	jp.LimparFalhaTrancar(nome)
	gui.DialogoMensagem(jp.Janela(), "Cofre Trancado", fmt.Sprintf("'%s' trancado.", nome), gui.MsgInfo)
	jp.ForcarAtualizacao()
	return nil
}

// acaoNovoCofre executa o fluxo de criação de novo cofre.
func acaoNovoCofre(gerenciador *core.GerenciadorRClone, jp *gui.JanelaPrincipal) {
	jp.Mostrar()
	resultado := gui.DialogoNovoCofre(jp.Janela())
	if !resultado.Sucesso {
		return
	}

	prov := resultado.Provedor
	nome := resultado.Nome
	senha := resultado.Senha
	nomeBase := core.NomeRemotoBase(nome)

	// Demanda 006: o nome é conferido antes de qualquer chamada ao rclone, e o
	// que esta tentativa criar é removido se ela não chegar ao fim.
	criacao, err := gerenciador.IniciarCriacaoCofre(nome)
	if err != nil {
		gui.DialogoMensagem(jp.Janela(), "Erro", err.Error(), gui.MsgErro)
		return
	}
	defer criacao.Desfazer()

	if prov.OAuth {
		sucesso := gerenciador.OAuth.Iniciar(gerenciador.Executavel, prov.Id)
		if !sucesso {
			gui.DialogoMensagem(jp.Janela(), "Erro OAuth", "Falha ao iniciar autenticação.", gui.MsgErro)
			return
		}

		time.Sleep(1 * time.Second)
		status := gerenciador.OAuth.ObterStatus()
		if status.URL != "" {
			avisarAutorizacao(jp, status.URL)
		}

		// Polling de token
		for i := 0; i < 120; i++ {
			time.Sleep(1 * time.Second)
			status = gerenciador.OAuth.ObterStatus()
			if status.Concluido {
				var tokenData map[string]interface{}
				if err := json.Unmarshal([]byte(status.Token), &tokenData); err != nil {
					gui.DialogoMensagem(jp.Janela(), "Erro OAuth", "Token OAuth inválido.", gui.MsgErro)
					return
				}
				sucesso, msg := criacao.CriarRemoto(nomeBase, prov.Id, map[string]string{
					"token": status.Token,
				})
				if !sucesso {
					gui.DialogoMensagem(jp.Janela(), "Erro", msg, gui.MsgErro)
					return
				}
				break
			}
		}

		if !gerenciador.OAuth.ObterStatus().Concluido {
			gerenciador.OAuth.Abortar()
			gui.DialogoMensagem(jp.Janela(), "Timeout", "Autorização não concluída em 2 minutos.", gui.MsgErro)
			return
		}
	} else if prov.Id == "local_path" {
		sucesso, msg := criacao.CriarRemoto(nomeBase, "local", map[string]string{"remote": ""})
		if !sucesso {
			gui.DialogoMensagem(jp.Janela(), "Erro", msg, gui.MsgErro)
			return
		}
	}

	remotoBase := nomeBase + ":"
	sucesso, msg := criacao.CriarCrypt(remotoBase, senha, senha, nil)
	if !sucesso {
		gui.DialogoMensagem(jp.Janela(), "Erro", msg, gui.MsgErro)
		return
	}

	sucesso, msg = criacao.Concluir(prov.Id, prov.Nome, remotoBase)
	if sucesso {
		gerenciador.Senhas.Armazenar(nome, senha)
		gui.DialogoMensagem(jp.Janela(), "Sucesso",
			fmt.Sprintf("Cofre '%s' criado com sucesso!\n\nProvedor: %s\nRemoto base: %s\n\nUse o botão 'Destrancar' para montar.", nome, prov.Nome, remotoBase),
			gui.MsgInfo,
		)
		jp.ForcarAtualizacao()
	} else {
		gui.DialogoMensagem(jp.Janela(), "Erro", msg, gui.MsgErro)
	}
}

// acaoImportarCofre executa o fluxo de importação de cofre existente.
func acaoImportarCofre(gerenciador *core.GerenciadorRClone, jp *gui.JanelaPrincipal) {
	jp.Mostrar()
	resultado := gui.DialogoImportarCofre(jp.Janela())
	if !resultado.Sucesso {
		return
	}

	prov := resultado.Provedor
	nome := resultado.Nome
	senha := resultado.Senha
	senha2 := resultado.Senha2
	nomeBase := core.NomeRemotoBase(nome)

	if prov.Id == "local_path" {
		// Para local: usar diálogo nativo de pasta do Fyne
		// TODO: Integrar com dialog.ShowFolderOpen
		gui.DialogoMensagem(jp.Janela(), "Info", "Selecione a pasta no explorador.", gui.MsgInfo)
		return
	}

	// Demanda 006: mesmo cuidado de acaoNovoCofre.
	criacao, err := gerenciador.IniciarCriacaoCofre(nome)
	if err != nil {
		gui.DialogoMensagem(jp.Janela(), "Erro", err.Error(), gui.MsgErro)
		return
	}
	defer criacao.Desfazer()

	if prov.OAuth {
		sucesso := gerenciador.OAuth.Iniciar(gerenciador.Executavel, prov.Id)
		if !sucesso {
			gui.DialogoMensagem(jp.Janela(), "Erro OAuth", "Falha ao iniciar autenticação.", gui.MsgErro)
			return
		}

		time.Sleep(1 * time.Second)
		status := gerenciador.OAuth.ObterStatus()
		if status.URL != "" {
			avisarAutorizacao(jp, status.URL)
		}

		for i := 0; i < 120; i++ {
			time.Sleep(1 * time.Second)
			status = gerenciador.OAuth.ObterStatus()
			if status.Concluido {
				sucesso, msg := criacao.CriarRemoto(nomeBase, prov.Id, map[string]string{
					"token": status.Token,
				})
				if !sucesso {
					gui.DialogoMensagem(jp.Janela(), "Erro", msg, gui.MsgErro)
					return
				}
				break
			}
		}

		if !gerenciador.OAuth.ObterStatus().Concluido {
			gerenciador.OAuth.Abortar()
			gui.DialogoMensagem(jp.Janela(), "Timeout", "Autorização não concluída em 2 minutos.", gui.MsgErro)
			return
		}
	}

	// Selecionar pasta remota
	caminho := gui.DialogoSeletorPastaRemota(jp.Janela(), gerenciador, nomeBase, prov.Nome)
	if caminho == nil {
		return // o defer de criacao.Desfazer remove o remoto base
	}

	var remotoBase string
	if *caminho != "" {
		remotoBase = nomeBase + ":" + *caminho
	} else {
		remotoBase = nomeBase + ":"
	}

	sucesso, msg := criacao.CriarCrypt(remotoBase, senha, senha2, nil)
	if !sucesso {
		gui.DialogoMensagem(jp.Janela(), "Erro", msg, gui.MsgErro)
		return
	}

	sucesso, msg = criacao.Concluir(prov.Id, prov.Nome, remotoBase)
	if sucesso {
		gerenciador.Senhas.Armazenar(nome, senha)
		gui.DialogoMensagem(jp.Janela(), "Sucesso",
			fmt.Sprintf("Cofre '%s' importado com sucesso!\n\nProvedor: %s\nRemoto: %s\n\nUse o botão 'Destrancar' para montar.", nome, prov.Nome, remotoBase),
			gui.MsgInfo,
		)
		jp.ForcarAtualizacao()
	} else {
		gui.DialogoMensagem(jp.Janela(), "Erro", msg, gui.MsgErro)
	}
}

// acaoVerificarFuse verifica se WinFsp/FUSE está instalado.
func acaoVerificarFuse(jp *gui.JanelaPrincipal) {
	info := plataforma.VerificarWinfsp()
	if info.Instalado {
		gui.DialogoMensagem(jp.Janela(), "WinFsp/FUSE", "WinFsp/FUSE está instalado e funcionando.", gui.MsgInfo)
	} else {
		gui.DialogoMensagem(jp.Janela(), "WinFsp/FUSE Ausente",
			fmt.Sprintf("%s\n\nBaixe em: %s", info.Motivo, info.UrlDownload), gui.MsgAviso)
	}
}

// autoMontarCofres monta automaticamente cofres configurados para auto-montagem.
// Uma falha aparece para o usuário com o nome do cofre (demanda 009).
func autoMontarCofres(gerenciador *core.GerenciadorRClone, jp *gui.JanelaPrincipal) {
	for _, cofre := range gerenciador.ListarCofres() {
		if cofre.AutoMontar && cofre.TemSenha && !cofre.Montado {
			senha := gerenciador.Senhas.Obter(cofre.Nome)
			if senha != "" {
				if ok, msg, _ := gerenciador.Montagens.MontarUnidade(cofre.Nome, "", senha, nil); !ok {
					gui.DialogoMensagem(jp.Janela(), "Erro ao Auto-montar",
						fmt.Sprintf("Falha ao montar '%s':\n%s", cofre.Nome, msg), gui.MsgErro)
				}
			}
		}
	}
}

// alternarAutoIniciar liga ou desliga o auto-início e mostra o erro, se houver
// (demanda 009).
func alternarAutoIniciar(jp *gui.JanelaPrincipal) {
	var err error
	if plataforma.VerificarAutoIniciar() {
		err = plataforma.RemoverAutoIniciar()
	} else {
		err = plataforma.AdicionarAutoIniciar()
	}
	if err != nil {
		jp.Mostrar()
		gui.DialogoMensagem(jp.Janela(), "Erro no Auto-iniciar", err.Error(), gui.MsgErro)
	}
}

// avisarAutorizacao abre o navegador na URL do OAuth. Se não abrir, mostra a
// URL para o usuário abrir à mão (demanda 009).
func avisarAutorizacao(jp *gui.JanelaPrincipal, url string) {
	if err := core.AbrirNavegador(url); err != nil {
		gui.DialogoMensagem(jp.Janela(), "Autorização",
			fmt.Sprintf("Não deu para abrir o navegador: %v\n\nAbra este endereço no navegador:\n%s\n\nApós concluir, volte para esta janela.", err, url), gui.MsgAviso)
		return
	}
	gui.DialogoMensagem(jp.Janela(), "Autorização",
		"O navegador foi aberto para autorização.\nApós concluir, volte para esta janela.", gui.MsgInfo)
}

// temaRuntime implementa o tema escuro do RuntimeCrypto para Fyne.
// Delega ao tema dark padrão, customizando apenas o que precisamos.
type temaRuntime struct{}

func (t *temaRuntime) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	// Usar o tema dark padrão como base
	return theme.DarkTheme().Color(name, theme.VariantDark)
}

func (t *temaRuntime) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DarkTheme().Font(style)
}

func (t *temaRuntime) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DarkTheme().Icon(name)
}

func (t *temaRuntime) Size(name fyne.ThemeSizeName) float32 {
	return theme.DarkTheme().Size(name)
}
