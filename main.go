package main

import (
	_ "embed"
	"time"

	"fyne.io/fyne/v2/app"

	"github.com/eufrauzino/runtime-crypt-go/internal/core"
	"github.com/eufrauzino/runtime-crypt-go/internal/gui"
	"github.com/eufrauzino/runtime-crypt-go/internal/plataforma"
	"github.com/eufrauzino/runtime-crypt-go/internal/tray"
)

// main só monta as peças (demanda 017): os casos de uso moram em
// internal/core e a ligação com a tela em internal/gui (Acoes).

//go:embed assets/icone.ico
var iconeBytes []byte

func main() {
	gerenciador := core.NovoGerenciador()
	// Demanda 027: o texto do rclone vai para o log, que precisa existir.
	if registro, err := core.AbrirLog(gerenciador.DiretorioApp); err == nil {
		defer registro.Close()
	}
	// Demanda 027: sem o driver de montagem, o card diz qual falta.
	gerenciador.Montagens.DriverInstalado = func() bool { return plataforma.VerificarWinfsp().Instalado }
	canalAcoes := make(chan tray.AcaoTray, 32)

	aplicacao := app.NewWithID("com.eufrauzino.runtime-crypto")
	aplicacao.Settings().SetTheme(&gui.TemaRuntime{})

	janela := gui.NovaJanelaPrincipal(aplicacao, gerenciador)
	acoes := gui.NovasAcoes(gerenciador, janela)
	// Demanda 028: antes de sair, avisa de arquivos que não subiram.
	sair := func() {
		go acoes.Sair(func() {
			gerenciador.Encerrar()
			aplicacao.Quit()
		})
	}

	janela.CallbackCofre = func(cofre core.CofreStatus) { go acoes.Cofre(cofre.Nome) }
	janela.CallbackTrancar = func(cofre core.CofreStatus) { go acoes.Trancar(cofre.Nome) }
	janela.CallbackNovoCofre = func() { go acoes.NovoCofre() }
	janela.CallbackImportarCofre = func() { go acoes.ImportarCofre() }
	janela.CallbackConfigVfs = func() { go gui.DialogoConfigVfs(janela.Janela(), gerenciador) }
	janela.CallbackVerificarFuse = func() { go acoes.VerificarFuse() }
	janela.CallbackSobre = func() { go acoes.Sobre() }
	janela.CallbackSair = sair

	bandeja := tray.NovoGerenciadorTray(gerenciador, canalAcoes, iconeBytes)
	go bandeja.Iniciar(nil, nil)
	go atenderBandeja(canalAcoes, janela, acoes, gerenciador, sair)

	go func() {
		time.Sleep(1500 * time.Millisecond)
		acoes.AutoMontar()
	}()

	janela.Mostrar()
	go acoes.AvisosDaAbertura()
	aplicacao.Run()
}

// atenderBandeja traduz cada clique da bandeja numa ação.
func atenderBandeja(canal <-chan tray.AcaoTray, janela *gui.JanelaPrincipal, acoes *gui.Acoes, gerenciador *core.GerenciadorRClone, sair func()) {
	for acao := range canal {
		switch acao.Tipo {
		case tray.AcaoMostrarJanela:
			janela.Mostrar()
		case tray.AcaoCofre:
			if gerenciador.Cofres.Obter(acao.Dados) != nil {
				go acoes.Cofre(acao.Dados)
			}
		case tray.AcaoNovoCofre:
			go acoes.NovoCofre()
		case tray.AcaoConfigVfs:
			janela.Mostrar()
			go gui.DialogoConfigVfs(janela.Janela(), gerenciador)
		case tray.AcaoVerificarFuse:
			go acoes.VerificarFuse()
		case tray.AcaoSobre:
			go acoes.Sobre()
		case tray.AcaoAutoIniciar:
			go acoes.AlternarAutoIniciar()
		case tray.AcaoSair:
			sair()
		}
	}
}
