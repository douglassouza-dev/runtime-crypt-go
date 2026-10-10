package tray

import (
	"time"

	"github.com/getlantern/systray"

	"github.com/eufrauzino/runtime-crypt-go/internal/core"
	"github.com/eufrauzino/runtime-crypt-go/internal/gui/frases"
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

// Rótulos do menu da bandeja com acento e reticências de um caractere
// (demanda 031, aprovado pela UI).
const (
	RotuloNovoCofre     = "Novo Cofre…"
	RotuloConfiguracoes = "Configurações"
	RotuloConfigVfs     = "Configurações VFS…"
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
	go g.acompanharCofres()

	// Menu item principal
	mAbrir := systray.AddMenuItem("Abrir RuntimeCrypto", "Abrir janela principal")
	systray.AddSeparator()

	mNovo := systray.AddMenuItem(RotuloNovoCofre, "Criar novo cofre")
	systray.AddSeparator()

	mConfig := systray.AddMenuItem(RotuloConfiguracoes, "")
	mAutoIniciar := mConfig.AddSubMenuItem("Auto-iniciar com Windows", "")
	mConfigVfs := mConfig.AddSubMenuItem(RotuloConfigVfs, "")
	mVerificarFuse := mConfig.AddSubMenuItem("Verificar WinFsp/FUSE", "")
	// Demanda 031: sem pasta de configuração, o que grava fica desabilitado.
	if g.gerenciador.SomenteLeitura() {
		mNovo.Disable()
		mConfigVfs.Disable()
	}

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

// AtualizarTooltip põe no tooltip o resumo dos cofres, com as mesmas frases
// do card (demanda 018). Devolve o texto posto.
func (g *GerenciadorTray) AtualizarTooltip() string {
	texto := frases.Tooltip(g.gerenciador.ListarCofres())
	systray.SetTooltip(texto)
	return texto
}

// intervaloTooltip é de quanto em quanto o tooltip confere os cofres.
const intervaloTooltip = 2 * time.Second

// acompanharCofres atualiza o tooltip a cada mudança de estado dos cofres.
func (g *GerenciadorTray) acompanharCofres() {
	anterior := g.AtualizarTooltip()
	ticker := time.NewTicker(intervaloTooltip)
	defer ticker.Stop()
	for range ticker.C {
		if texto := frases.Tooltip(g.gerenciador.ListarCofres()); texto != anterior {
			systray.SetTooltip(texto)
			anterior = texto
		}
	}
}
