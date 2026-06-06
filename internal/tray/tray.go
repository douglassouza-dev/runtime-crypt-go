package tray

import (
	"fmt"

	"github.com/getlantern/systray"

	"github.com/eufrauzino/runtime-crypto/internal/core"
)

// TipoAcaoTray identifica o tipo de ação enviada pelo tray.
type TipoAcaoTray string

const (
	AcaoMostrarJanela TipoAcaoTray = "mostrar_janela"
	AcaoCofre         TipoAcaoTray = "cofre"
	AcaoNovoCofre     TipoAcaoTray = "novo_cofre"
	AcaoConfigVfs     TipoAcaoTray = "config_vfs"
	AcaoVerificarFuse TipoAcaoTray = "verificar_winfsp"
	AcaoSobre         TipoAcaoTray = "sobre"
	AcaoSair          TipoAcaoTray = "sair"
	AcaoAutoIniciar   TipoAcaoTray = "auto_iniciar"
)

// AcaoTray é uma ação emitida pelo tray para processamento pela main loop.
type AcaoTray struct {
	Tipo  TipoAcaoTray
	Dados string
}

// GerenciadorTray controla o ícone na bandeja do sistema.
type GerenciadorTray struct {
	gerenciador *core.GerenciadorRClone
	canalAcoes  chan AcaoTray
	iconeBytes  []byte
}

// NovoGerenciadorTray cria uma instância do gerenciador de tray.
func NovoGerenciadorTray(gerenciador *core.GerenciadorRClone, canalAcoes chan AcaoTray, iconeBytes []byte) *GerenciadorTray {
	return &GerenciadorTray{
		gerenciador: gerenciador,
		canalAcoes:  canalAcoes,
		iconeBytes:  iconeBytes,
	}
}

// Iniciar configura e inicia o system tray (bloqueante — deve ser chamado na main thread).
func (g *GerenciadorTray) Iniciar(aoIniciar func(), aoEncerrar func()) {
	systray.Run(func() {
		g.aoIniciar()
		if aoIniciar != nil {
			aoIniciar()
		}
	}, func() {
		if aoEncerrar != nil {
			aoEncerrar()
		}
	})
}

// aoIniciar configura o ícone e menu inicial do tray.
func (g *GerenciadorTray) aoIniciar() {
	systray.SetIcon(g.iconeBytes)
	systray.SetTitle("RuntimeCrypto")
	systray.SetTooltip("RuntimeCrypto — Cofre Criptografado na Nuvem")

	// Menu item principal
	mAbrir := systray.AddMenuItem("Abrir RuntimeCrypto", "Abrir janela principal")
	systray.AddSeparator()

	mNovo := systray.AddMenuItem("Novo Cofre...", "Criar novo cofre")
	systray.AddSeparator()

	mConfig := systray.AddMenuItem("Configuracoes", "")
	mAutoIniciar := mConfig.AddSubMenuItem("Auto-iniciar com Windows", "")
	mConfigVfs := mConfig.AddSubMenuItem("Configuracoes VFS...", "")
	mVerificarFuse := mConfig.AddSubMenuItem("Verificar WinFsp/FUSE", "")

	systray.AddSeparator()
	mSobre := systray.AddMenuItem("Sobre", "Sobre o RuntimeCrypto")
	mSair := systray.AddMenuItem("Sair", "Encerrar programa")

	// Goroutine para processar cliques do menu
	go func() {
		for {
			select {
			case <-mAbrir.ClickedCh:
				g.canalAcoes <- AcaoTray{Tipo: AcaoMostrarJanela}
			case <-mNovo.ClickedCh:
				g.canalAcoes <- AcaoTray{Tipo: AcaoNovoCofre}
			case <-mAutoIniciar.ClickedCh:
				g.canalAcoes <- AcaoTray{Tipo: AcaoAutoIniciar}
			case <-mConfigVfs.ClickedCh:
				g.canalAcoes <- AcaoTray{Tipo: AcaoConfigVfs}
			case <-mVerificarFuse.ClickedCh:
				g.canalAcoes <- AcaoTray{Tipo: AcaoVerificarFuse}
			case <-mSobre.ClickedCh:
				g.canalAcoes <- AcaoTray{Tipo: AcaoSobre}
			case <-mSair.ClickedCh:
				g.canalAcoes <- AcaoTray{Tipo: AcaoSair}
			}
		}
	}()
}

// AtualizarTooltip atualiza o tooltip do tray com informação de cofres.
func (g *GerenciadorTray) AtualizarTooltip() {
	cofres := g.gerenciador.ListarCofres()
	montados := 0
	for _, c := range cofres {
		if c.Montado {
			montados++
		}
	}

	if montados > 0 {
		systray.SetTooltip(fmt.Sprintf("RuntimeCrypto — %d cofre(s) destrancado(s)", montados))
	} else {
		systray.SetTooltip("RuntimeCrypto — Todos os cofres trancados")
	}
}
