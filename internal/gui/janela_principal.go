package gui

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/eufrauzino/runtime-crypt-go/internal/core"
	"github.com/eufrauzino/runtime-crypt-go/internal/gui/frases"
)

// JanelaPrincipal é a janela principal do RuntimeCrypto.
type JanelaPrincipal struct {
	janela          fyne.Window
	gerenciador     *core.GerenciadorRClone
	containerCofres *fyne.Container
	lblVazio        *widget.Label
	// montou diz que a lista já foi montada uma vez.
	montou         bool
	estadoAnterior map[string]string
	mu             sync.Mutex

	// falhasTrancar guarda, por cofre, o motivo do último Trancar que não
	// terminou (demanda 022). O card mostra "Não trancou: {motivo}".
	falhasTrancar map[string]string

	CallbackCofre func(core.CofreStatus)
	// CallbackTrancar é o botão Trancar do cofre que caiu (demanda 026).
	CallbackTrancar       func(core.CofreStatus)
	CallbackNovoCofre     func()
	CallbackImportarCofre func()
	CallbackConfigVfs     func()
	CallbackVerificarFuse func()
	CallbackSobre         func()
	CallbackSair          func()

	// Demanda 031: guardados para os testes conferirem o modo só leitura.
	btnNovo, btnImportar, btnConfig *widget.Button
	faixa                           fyne.CanvasObject
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
	jp.lblVazio = widget.NewLabel(textoVazio(jp.gerenciador.SomenteLeitura()))
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
	jp.btnNovo, jp.btnImportar = btnNovo, btnImportar

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
	jp.btnConfig = btnConfig

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
	// Demanda 031: sem pasta de configuração, faixa fixa no topo e o que
	// grava fica desabilitado. Destrancar e trancar seguem normais.
	var topo fyne.CanvasObject = header
	if jp.gerenciador.SomenteLeitura() {
		jp.faixa = faixaSomenteLeitura(pastaDoErro(jp.gerenciador.ErroPastaConfig))
		topo = container.NewVBox(header, jp.faixa)
		btnNovo.Disable()
		btnImportar.Disable()
		btnConfig.Disable()
	}
	conteudo := container.NewBorder(topo, footer, nil, nil, scrollCentral)
	jp.janela.SetContent(conteudo)

	// Primeira renderização
	jp.atualizarCofres()
}

// textoVazio é o texto da janela sem cofres. No modo só leitura (031) não
// há a dica de Adicionar Cofre, que está desabilitado.
func textoVazio(somenteLeitura bool) string {
	if somenteLeitura {
		return "Nenhum cofre ainda."
	}
	return "Nenhum cofre configurado.\n\nClique em '＋ Adicionar Cofre' para começar."
}

// pastaDoErro é a pasta que não pôde ser usada, "" se não se sabe.
func pastaDoErro(err error) string {
	var e *core.ErroPastaConfig
	if errors.As(err, &e) {
		return e.Pasta
	}
	return ""
}

// faixaSomenteLeitura é a faixa fixa do topo quando a pasta de configuração
// não pode ser usada (031), sem título.
func faixaSomenteLeitura(pasta string) fyne.CanvasObject {
	texto := widget.NewLabel(frases.SomenteLeitura(pasta))
	texto.Wrapping = fyne.TextWrapWord
	texto.TextStyle = fyne.TextStyle{Bold: true}
	fundo := canvas.NewRectangle(CorAviso)
	return container.NewStack(fundo, container.NewPadded(texto))
}

// atualizarCofres reconstroi os cards dos cofres.
func (jp *JanelaPrincipal) atualizarCofres() {
	jp.mu.Lock()
	defer jp.mu.Unlock()

	cofres := jp.gerenciador.ListarCofres()

	// Gerar estado para comparação
	estadoNovo := make(map[string]string)
	for _, c := range cofres {
		chave := fmt.Sprintf("%s|%s|%v|%s|%s", c.Nome, frases.DoCofre(c), c.Estado == core.EstadoMontando, frases.Botao(c), jp.falhasTrancar[c.Nome])
		estadoNovo[c.Nome] = chave
	}

	// Se não mudou, não reconstroi. A primeira vez sempre monta: sem cofres,
	// os dois mapas vazios pareciam iguais e o texto de vazio nunca aparecia.
	if jp.montou && len(estadoNovo) == len(jp.estadoAnterior) {
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
	jp.montou = true

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
			}, func() {
				if jp.CallbackTrancar != nil {
					jp.CallbackTrancar(c)
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

// atualizarLogoApos atualiza o card assim que o core tira o cofre de
// trancado (demanda 018), sem esperar o próximo ciclo de 3 s. Se a senha
// demorar mais que esperaDestrancando, o ciclo normal cuida.
func (jp *JanelaPrincipal) atualizarLogoApos(nome string) {
	fim := time.Now().Add(esperaDestrancando)
	for time.Now().Before(fim) {
		if jp.gerenciador.EstadoDoCofre(nome).Estado != core.EstadoDesmontado {
			jp.ForcarAtualizacao()
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}

const esperaDestrancando = 30 * time.Second

// acompanharTrancar atualiza o card enquanto o Trancar espera o envio
// (demanda 025): "Enviando {n} arquivos…" muda a cada arquivo enviado.
func (jp *JanelaPrincipal) acompanharTrancar(feito <-chan struct{}) {
	t := time.NewTicker(300 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-feito:
			return
		case <-t.C:
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
// não é vazio, é o motivo do último Trancar que não terminou. aoTrancar é o
// segundo botão, Trancar, que só o cofre que caiu tem (demanda 026).
func criarCardCofre(cofre core.CofreStatus, falhaTrancar string, aoClicar, aoTrancar func()) fyne.CanvasObject {
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

	// Demanda 018: a frase do estado vem de internal/frases, a mesma da
	// bandeja. Uma falha de montagem tem o motivo, que pode ser longo: vai
	// para a linha de baixo, com quebra.
	fraseEstado := frases.DoCofre(cofre)
	infoContainer := container.NewVBox(lblNome, lblProvedor)
	if cofre.Estado != core.EstadoFalhou {
		corStatus := CorTrancado
		switch cofre.Estado {
		case core.EstadoMontado:
			corStatus = CorMontado
		case core.EstadoMontando:
			corStatus = CorAviso
		}
		lblStatus := canvas.NewText(fraseEstado, corStatus)
		lblStatus.TextSize = 10
		infoContainer.Add(lblStatus)
	}

	// Botão de ação: Destrancar, Trancar ou Tentar de novo. Destrancando
	// desabilita o botão; um segundo clique não começa outro rclone.
	btnAcao := widget.NewButton(frases.Botao(cofre), aoClicar)
	if cofre.EstaMontado() {
		btnAcao.Importance = widget.MediumImportance
	} else {
		btnAcao.Importance = widget.HighImportance
	}
	if cofre.Estado == core.EstadoMontando || cofre.Enviando > 0 {
		btnAcao.Disable()
	}

	// Demanda 026: no cofre que caiu, o principal é "Destrancar de novo" e
	// Trancar fica ao lado.
	var botoes fyne.CanvasObject = btnAcao
	if frases.BotaoTrancar(cofre) {
		btnTrancar := widget.NewButton(frases.TextoTrancar, aoTrancar)
		btnTrancar.Importance = widget.MediumImportance
		botoes = container.NewHBox(btnAcao, btnTrancar)
	}
	// Demanda 027: sem o driver de montagem, "Baixar WinFsp"/"Baixar macFUSE"
	// abre o site oficial.
	if rotulo, url := frases.BotaoBaixarDriver(cofre); url != "" {
		btnBaixar := widget.NewButton(rotulo, func() {
			if err := abrirSiteDriver(url); err != nil {
				log.Printf("não deu para abrir %s: %v", url, err)
			}
		})
		btnBaixar.Importance = widget.MediumImportance
		botoes = container.NewHBox(btnAcao, btnBaixar)
	}

	// Layout do card
	cardConteudo := container.NewHBox(
		indicador,
		layout.NewSpacer(),
		infoContainer,
		layout.NewSpacer(),
		botoes,
	)

	cardBg := canvas.NewRectangle(CorCard)
	cardBg.CornerRadius = 10
	cardBg.SetMinSize(fyne.NewSize(0, 76))

	// As linhas de motivo ("Não destrancou", "Caiu", "Não trancou") ficam
	// embaixo, com a largura toda do card, para o motivo quebrar em linhas.
	var linhas []string
	if cofre.Estado == core.EstadoFalhou {
		linhas = append(linhas, fraseEstado)
	}
	if falhaTrancar != "" {
		linhas = append(linhas, fmt.Sprintf(TextoNaoTrancou, falhaTrancar))
	}
	var corpo fyne.CanvasObject = cardConteudo
	if len(linhas) > 0 {
		caixa := container.NewVBox(cardConteudo)
		for _, l := range linhas {
			lbl := widget.NewLabel(l)
			lbl.Wrapping = fyne.TextWrapWord
			lbl.Importance = widget.DangerImportance
			caixa.Add(lbl)
		}
		corpo = caixa
	}

	return container.NewStack(cardBg, container.NewPadded(corpo))
}

// abrirSiteDriver abre o site do driver de montagem. Os testes trocam.
var abrirSiteDriver = core.AbrirNavegador
