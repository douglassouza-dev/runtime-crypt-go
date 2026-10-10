package gui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/eufrauzino/runtime-crypt-go/internal/core"
)

// TextoPastaSemSubpastas é a linha mostrada quando a pasta não tem subpastas.
const TextoPastaSemSubpastas = "Nenhuma subpasta aqui."

// Linha e botão mostrados quando a listagem falha (demanda 009). Erro nunca
// aparece como TextoPastaSemSubpastas.
const (
	TextoErroListagem = "Não deu para listar as pastas: %s"
	TextoTentarDeNovo = "Tentar de novo"
)

// linhaErroListagem é a linha de erro com o motivo e o botão que lista de
// novo a mesma pasta.
func linhaErroListagem(err error, tentarDeNovo func()) fyne.CanvasObject {
	lbl := widget.NewLabel(fmt.Sprintf(TextoErroListagem, err.Error()))
	lbl.Wrapping = fyne.TextWrapWord
	lbl.Importance = widget.DangerImportance
	btn := widget.NewButton(TextoTentarDeNovo, tentarDeNovo)
	return container.NewBorder(nil, nil, nil, btn, lbl)
}

// listaPastas é a lista do seletor: mostra as subpastas de caminho no remoto,
// a linha de pasta vazia ou a linha de erro com "Tentar de novo".
type listaPastas struct {
	nomeRemoto string
	listar     func(nomeRemoto, caminho string) ([]string, error)
	lista      *fyne.Container
	lblCaminho *canvas.Text
	caminho    string

	// carregada, se não for nil, é chamada no fim de cada carga, depois de a
	// lista ser atualizada. Os testes usam para esperar a goroutine.
	carregada func()
}

func novaListaPastas(nomeRemoto string, listar func(nomeRemoto, caminho string) ([]string, error)) *listaPastas {
	lblCaminho := canvas.NewText("  "+nomeRemoto+":/", CorTexto)
	lblCaminho.TextSize = 12
	return &listaPastas{
		nomeRemoto: nomeRemoto,
		listar:     listar,
		lista:      container.NewVBox(),
		lblCaminho: lblCaminho,
	}
}

// entrar desce para a subpasta nome e lista de novo.
func (l *listaPastas) entrar(nome string) {
	if l.caminho != "" {
		l.caminho = l.caminho + "/" + nome
	} else {
		l.caminho = nome
	}
	l.carregar()
}

// voltar sobe uma pasta e lista de novo.
func (l *listaPastas) voltar() {
	if l.caminho == "" {
		return
	}
	partes := splitCaminho(l.caminho)
	if len(partes) <= 1 {
		l.caminho = ""
	} else {
		l.caminho = joinCaminho(partes[:len(partes)-1])
	}
	l.carregar()
}

// carregar lista l.caminho numa goroutine e troca o conteúdo da lista pelo
// resultado.
func (l *listaPastas) carregar() {
	l.lista.RemoveAll()

	lblCarregando := canvas.NewText("🔄 Carregando pastas...", CorTextoSec)
	lblCarregando.TextSize = 12
	l.lista.Add(lblCarregando)
	l.lista.Refresh()

	exibicao := l.caminho
	if exibicao == "" {
		exibicao = "/"
	}
	l.lblCaminho.Text = "  " + l.nomeRemoto + ":" + exibicao
	l.lblCaminho.Refresh()

	caminho := l.caminho
	go func() {
		dirs, err := l.listar(l.nomeRemoto, caminho)

		// Atualizar UI na thread Fyne
		l.lista.RemoveAll()

		if err != nil {
			// Demanda 009: o erro aparece com o motivo, e "Tentar de novo"
			// lista de novo a mesma pasta (l.caminho não muda).
			l.lista.Add(linhaErroListagem(err, l.carregar))
		} else if len(dirs) == 0 {
			// Uma linha só, sem botão: a pasta atual continua escolhível
			// pelo caminho do topo e por "Selecionar esta pasta" (demanda 021).
			lblVazio := canvas.NewText(TextoPastaSemSubpastas, CorTextoSec)
			lblVazio.TextSize = 12
			l.lista.Add(lblVazio)
		} else {
			for _, nomePasta := range dirs {
				np := nomePasta // captura
				btn := widget.NewButton("📁  "+np, func() { l.entrar(np) })
				btn.Importance = widget.LowImportance
				btn.Alignment = widget.ButtonAlignLeading
				l.lista.Add(btn)
			}
		}
		l.lista.Refresh()
		if l.carregada != nil {
			l.carregada()
		}
	}()
}

// DialogoSeletorPastaRemota exibe um navegador visual de pastas em um remoto rclone.
// Retorna o caminho selecionado ou "" se cancelado.
func DialogoSeletorPastaRemota(janelaPai fyne.Window, gerenciador *core.GerenciadorRClone, nomeRemoto string, tituloProvedor string) *string {
	resultado := make(chan *string, 1)

	// Header
	lblIcone := canvas.NewText("📂", CorVerde)
	lblIcone.TextSize = 28
	lblIcone.Alignment = fyne.TextAlignCenter

	lblTitulo := canvas.NewText("Selecionar Pasta do Cofre", CorVerde)
	lblTitulo.TextSize = 16
	lblTitulo.TextStyle = fyne.TextStyle{Bold: true}
	lblTitulo.Alignment = fyne.TextAlignCenter

	// Barra de caminho e lista de pastas
	pastas := novaListaPastas(nomeRemoto, gerenciador.ListarDiretoriosRemoto)
	lblCaminho := pastas.lblCaminho
	scrollLista := container.NewVScroll(pastas.lista)
	scrollLista.SetMinSize(fyne.NewSize(0, 280))

	var dialogo *widget.PopUp

	// Botão voltar
	btnVoltar := widget.NewButton("⬆ Voltar", pastas.voltar)

	// Dica
	lblDica := canvas.NewText("💡 Clique para entrar em uma pasta.\nClique 'Selecionar esta pasta' para usar a pasta atual.", CorTextoSec)
	lblDica.TextSize = 10

	// Botões
	btnCancelar := widget.NewButton("Cancelar", func() {
		resultado <- nil
		if dialogo != nil {
			dialogo.Hide()
		}
	})

	btnSelecionar := widget.NewButton("✓  Selecionar esta pasta", func() {
		cam := pastas.caminho
		resultado <- &cam
		if dialogo != nil {
			dialogo.Hide()
		}
	})
	btnSelecionar.Importance = widget.HighImportance

	// Provedor
	var headerExtra fyne.CanvasObject
	if tituloProvedor != "" {
		lblProv := canvas.NewText("Navegando em: "+tituloProvedor, CorTextoSec)
		lblProv.TextSize = 11
		lblProv.Alignment = fyne.TextAlignCenter
		headerExtra = lblProv
	} else {
		headerExtra = layout.NewSpacer()
	}

	conteudo := container.NewVBox(
		lblIcone,
		lblTitulo,
		headerExtra,
		widget.NewSeparator(),
		container.NewHBox(lblCaminho, layout.NewSpacer(), btnVoltar),
		scrollLista,
		lblDica,
		layout.NewSpacer(),
		container.NewHBox(btnCancelar, layout.NewSpacer(), btnSelecionar),
	)

	padded := container.NewPadded(conteudo)

	dialogo = widget.NewModalPopUp(padded, janelaPai.Canvas())
	dialogo.Resize(fyne.NewSize(520, 540))
	dialogo.Show()

	// Carregar inicial
	pastas.carregar()

	return <-resultado
}

// splitCaminho divide um caminho "/" em partes.
func splitCaminho(caminho string) []string {
	var partes []string
	atual := ""
	for _, c := range caminho {
		if c == '/' {
			if atual != "" {
				partes = append(partes, atual)
				atual = ""
			}
		} else {
			atual += string(c)
		}
	}
	if atual != "" {
		partes = append(partes, atual)
	}
	return partes
}

// joinCaminho junta partes com "/".
func joinCaminho(partes []string) string {
	resultado := ""
	for i, p := range partes {
		if i > 0 {
			resultado += "/"
		}
		resultado += p
	}
	return resultado
}
