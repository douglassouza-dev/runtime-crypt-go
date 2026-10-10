package gui

import (
	"fmt"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/eufrauzino/runtime-crypt-go/internal/core"
)

// JanelaPrincipal é a janela principal do RuntimeCrypto.
type JanelaPrincipal struct {
	janela          fyne.Window
	gerenciador     *core.GerenciadorRClone
	containerCofres *fyne.Container
	lblVazio        *widget.Label
	estadoAnterior  map[string]string
	mu              sync.Mutex

	// falhasTrancar guarda, por cofre, o motivo do último Trancar que não
	// terminou (demanda 022). O card mostra "Não trancou: {motivo}".
	falhasTrancar map[string]string

	CallbackCofre         func(core.CofreStatus)
	CallbackNovoCofre     func()
	CallbackImportarCofre func()
	CallbackConfigVfs     func()
	CallbackVerificarFuse func()
	CallbackSobre         func()
	CallbackSair          func()
}

// NovaJanelaPrincipal cria e configura a janela principal.
func NovaJanelaPrincipal(app fyne.App, gerenciador *core.GerenciadorRClone) *JanelaPrincipal {
	janela := app.NewWindow("RuntimeCrypto")
	janela.Resize(fyne.NewSize(520, 640))

	jp := &JanelaPrincipal{
		janela:         janela,
		gerenciador:    gerenciador,
		estadoAnterior: make(map[string]string),
		falhasTrancar:  make(map[string]string),
	}

	jp.construirInterface()

	// Fechar (X) = esconder
	janela.SetCloseIntercept(func() {
		janela.Hide()
	})

	// Atualização periódica
	go jp.agendarAtualizacao()

	return jp
}

// construirInterface monta toda a UI da janela principal.
func (jp *JanelaPrincipal) construirInterface() {
	// === HEADER ===
	lblIcone := canvas.NewText("🔒", CorVerde)
	lblIcone.TextSize = 26

	lblTitulo := canvas.NewText("RuntimeCrypto", CorVerde)
	lblTitulo.TextSize = 18
	lblTitulo.TextStyle = fyne.TextStyle{Bold: true}

	lblSubtitulo := canvas.NewText(
		fmt.Sprintf("Cofre Criptografado na Nuvem  •  v%s", core.Versao),
		CorTextoSec,
	)
	lblSubtitulo.TextSize = 10

	headerTextos := container.NewVBox(lblTitulo, lblSubtitulo)
	headerConteudo := container.NewHBox(lblIcone, layout.NewSpacer(), headerTextos, layout.NewSpacer())

	headerBg := canvas.NewRectangle(CorCard)
	headerBg.SetMinSize(fyne.NewSize(0, 72))
	header := container.NewStack(headerBg, container.NewPadded(headerConteudo))

	// === ÁREA CENTRAL ===
	jp.lblVazio = widget.NewLabel("Nenhum cofre configurado.\n\nClique em '＋ Adicionar Cofre' para começar.")
	jp.lblVazio.Alignment = fyne.TextAlignCenter

	jp.containerCofres = container.NewVBox()

	// Botão novo cofre
	btnNovo := widget.NewButton("＋  Adicionar Cofre", func() {
		if jp.CallbackNovoCofre != nil {
			jp.CallbackNovoCofre()
		}
	})
	btnNovo.Importance = widget.MediumImportance

	// Botão importar
	btnImportar := widget.NewButton("📥  Importar Cofre Existente", func() {
		if jp.CallbackImportarCofre != nil {
			jp.CallbackImportarCofre()
		}
	})
	btnImportar.Importance = widget.LowImportance

	areaCentral := container.NewVBox(
		jp.containerCofres,
		btnNovo,
		btnImportar,
	)
	scrollCentral := container.NewVScroll(areaCentral)

	// === FOOTER ===
	btnConfig := widget.NewButton("⚙  Configurações", func() {
		if jp.CallbackConfigVfs != nil {
			jp.CallbackConfigVfs()
		}
	})
	btnConfig.Importance = widget.LowImportance

	btnSobre := widget.NewButton("ℹ  Sobre", func() {
		if jp.CallbackSobre != nil {
			jp.CallbackSobre()
		}
	})
	btnSobre.Importance = widget.LowImportance

	btnSair := widget.NewButton("✕  Sair", func() {
		if jp.CallbackSair != nil {
			jp.CallbackSair()
		}
	})
	btnSair.Importance = widget.DangerImportance

	footerBg := canvas.NewRectangle(CorCard)
	footerBg.SetMinSize(fyne.NewSize(0, 46))
	footerConteudo := container.NewHBox(
		btnConfig, btnSobre, layout.NewSpacer(), btnSair,
	)
	footer := container.NewStack(footerBg, container.NewPadded(footerConteudo))

	// === LAYOUT FINAL ===
	conteudo := container.NewBorder(header, footer, nil, nil, scrollCentral)
	jp.janela.SetContent(conteudo)

	// Primeira renderização
	jp.atualizarCofres()
}

// atualizarCofres reconstroi os cards dos cofres.
func (jp *JanelaPrincipal) atualizarCofres() {
	jp.mu.Lock()
	defer jp.mu.Unlock()

	cofres := jp.gerenciador.ListarCofres()

	// Gerar estado para comparação
	estadoNovo := make(map[string]string)
	for _, c := range cofres {
		chave := fmt.Sprintf("%s_%v_%s_%s", c.Nome, c.Montado, c.Letra, jp.falhasTrancar[c.Nome])
		estadoNovo[c.Nome] = chave
	}

	// Se não mudou, não reconstroi
	if len(estadoNovo) == len(jp.estadoAnterior) {
		igual := true
		for k, v := range estadoNovo {
			if jp.estadoAnterior[k] != v {
				igual = false
				break
			}
		}
		if igual {
			return
		}
	}
	jp.estadoAnterior = estadoNovo

	// Limpar
	jp.containerCofres.RemoveAll()

	if len(cofres) == 0 {
		jp.containerCofres.Add(jp.lblVazio)
	} else {
		for _, cofre := range cofres {
			c := cofre // captura para closure
			card := criarCardCofre(c, jp.falhasTrancar[c.Nome], func() {
				if jp.CallbackCofre != nil {
					jp.CallbackCofre(c)
				}
			})
			jp.containerCofres.Add(card)
		}
	}

	jp.containerCofres.Refresh()
}

// agendarAtualizacao atualiza os cards a cada 3 segundos.
func (jp *JanelaPrincipal) agendarAtualizacao() {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if jp.janela.Canvas() != nil {
			jp.atualizarCofres()
		}
	}
}

// Mostrar traz a janela para frente.
func (jp *JanelaPrincipal) Mostrar() {
	jp.janela.Show()
	jp.janela.RequestFocus()
	jp.ForcarAtualizacao()
}

// Esconder esconde a janela (minimiza pro tray).
func (jp *JanelaPrincipal) Esconder() {
	jp.janela.Hide()
}

// ForcarAtualizacao força reconstrução imediata dos cards.
func (jp *JanelaPrincipal) ForcarAtualizacao() {
	jp.mu.Lock()
	jp.estadoAnterior = make(map[string]string)
	jp.mu.Unlock()
	jp.atualizarCofres()
}

// MostrarFalhaTrancar põe "Não trancou: {motivo}" no card do cofre
// (demanda 022).
func (jp *JanelaPrincipal) MostrarFalhaTrancar(nome, motivo string) {
	jp.mu.Lock()
	jp.falhasTrancar[nome] = motivo
	jp.mu.Unlock()
	jp.ForcarAtualizacao()
}

// LimparFalhaTrancar tira a linha "Não trancou" do card do cofre.
func (jp *JanelaPrincipal) LimparFalhaTrancar(nome string) {
	jp.mu.Lock()
	_, havia := jp.falhasTrancar[nome]
	delete(jp.falhasTrancar, nome)
	jp.mu.Unlock()
	if havia {
		jp.ForcarAtualizacao()
	}
}

// Janela retorna a referência à fyne.Window.
func (jp *JanelaPrincipal) Janela() fyne.Window {
	return jp.janela
}

// TextoNaoTrancou é a linha do card quando o Trancar não terminou (demanda 022).
const TextoNaoTrancou = "Não trancou: %s"

// criarCardCofre cria um widget de card para um cofre. falhaTrancar, quando
// não é vazio, é o motivo do último Trancar que não terminou.
func criarCardCofre(cofre core.CofreStatus, falhaTrancar string, aoClicar func()) fyne.CanvasObject {
	// Indicador de cor do provedor
	corProv := ObterCorProvedor(cofre.ProvedorId)
	indicador := canvas.NewRectangle(corProv)
	indicador.SetMinSize(fyne.NewSize(4, 60))

	// Textos
	lblNome := canvas.NewText(cofre.Nome, CorTexto)
	lblNome.TextSize = 14
	lblNome.TextStyle = fyne.TextStyle{Bold: true}

	lblProvedor := canvas.NewText(cofre.ProvedorNome, CorTextoSec)
	lblProvedor.TextSize = 11

	var textoStatus string
	var corStatus = CorTrancado
	if cofre.Montado {
		if cofre.Letra != "" {
			textoStatus = fmt.Sprintf("● Destrancado  •  %s:\\", cofre.Letra)
		} else {
			textoStatus = "● Destrancado"
		}
		corStatus = CorMontado
	} else {
		textoStatus = "● Trancado"
	}

	lblStatus := canvas.NewText(textoStatus, corStatus)
	lblStatus.TextSize = 10

	infoContainer := container.NewVBox(lblNome, lblProvedor, lblStatus)

	// Botão de ação
	var textoBotao string
	if cofre.Montado {
		textoBotao = "Trancar"
	} else {
		textoBotao = "Destrancar"
	}

	btnAcao := widget.NewButton(textoBotao, aoClicar)
	if cofre.Montado {
		btnAcao.Importance = widget.MediumImportance
	} else {
		btnAcao.Importance = widget.HighImportance
	}

	// Layout do card
	cardConteudo := container.NewHBox(
		indicador,
		layout.NewSpacer(),
		infoContainer,
		layout.NewSpacer(),
		btnAcao,
	)

	cardBg := canvas.NewRectangle(CorCard)
	cardBg.CornerRadius = 10
	cardBg.SetMinSize(fyne.NewSize(0, 76))

	// A linha "Não trancou" fica embaixo, com a largura toda do card, para o
	// motivo quebrar em linhas.
	var corpo fyne.CanvasObject = cardConteudo
	if falhaTrancar != "" {
		lblFalha := widget.NewLabel(fmt.Sprintf(TextoNaoTrancou, falhaTrancar))
		lblFalha.Wrapping = fyne.TextWrapWord
		lblFalha.Importance = widget.DangerImportance
		corpo = container.NewVBox(cardConteudo, lblFalha)
	}

	return container.NewStack(cardBg, container.NewPadded(corpo))
}
