package tray

import (
	"sync"
	"time"

	"github.com/getlantern/systray"

	"github.com/eufrauzino/runtime-crypt-go/internal/core"
	"github.com/eufrauzino/runtime-crypt-go/internal/gui/frases"
	"github.com/eufrauzino/runtime-crypt-go/internal/plataforma"
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
	// RotuloAutoIniciar é o mesmo em todos os sistemas.
	RotuloAutoIniciar = "Abrir ao ligar o computador"
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

	// autoIniciarLigado lê o estado real do auto-início (registro no
	// Windows, .desktop no Linux, LaunchAgent no macOS). Os testes trocam.
	autoIniciarLigado func() bool
	// itemAutoIniciar é o item "Abrir ao ligar o computador", com a marca.
	// O menu o cria na goroutine da bandeja; o clique o atualiza em outra.
	muAutoIniciar   sync.Mutex
	itemAutoIniciar itemMarcavel
}

// itemMarcavel é o que o item com marca do systray oferece.
type itemMarcavel interface {
	Check()
	Uncheck()
}

// NovoGerenciadorTray cria uma instância do gerenciador de tray.
func NovoGerenciadorTray(gerenciador *core.GerenciadorRClone, canalAcoes chan AcaoTray, iconeBytes []byte) *GerenciadorTray {
	return &GerenciadorTray{
		gerenciador:       gerenciador,
		canalAcoes:        canalAcoes,
		iconeBytes:        iconeBytes,
		autoIniciarLigado: plataforma.VerificarAutoIniciar,
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
	// A marca mostra se o auto-início está ligado, lido do sistema.
	mAutoIniciar := mConfig.AddSubMenuItemCheckbox(RotuloAutoIniciar, "", g.autoIniciarLigado())
	g.muAutoIniciar.Lock()
	g.itemAutoIniciar = mAutoIniciar
	g.muAutoIniciar.Unlock()
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

// AtualizarAutoIniciar lê de novo o estado real do auto-início e põe ou tira
// a marca de "Abrir ao ligar o computador". Chamado depois de cada clique, já
// com o auto-início trocado (ou não, se a troca falhou). Devolve o estado.
func (g *GerenciadorTray) AtualizarAutoIniciar() bool {
	ligado := g.autoIniciarLigado()
	g.muAutoIniciar.Lock()
	defer g.muAutoIniciar.Unlock()
	if g.itemAutoIniciar != nil {
		marcar(g.itemAutoIniciar, ligado)
	}
	return ligado
}

func marcar(item itemMarcavel, ligado bool) {
	if ligado {
		item.Check()
	} else {
		item.Uncheck()
	}
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
