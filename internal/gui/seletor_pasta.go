package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/eufrauzino/runtime-crypto/internal/core"
)

// DialogoSeletorPastaRemota exibe um navegador visual de pastas em um remoto rclone.
// Retorna o caminho selecionado ou "" se cancelado.
func DialogoSeletorPastaRemota(janelaPai fyne.Window, gerenciador *core.GerenciadorRClone, nomeRemoto string, tituloProvedor string) *string {
	resultado := make(chan *string, 1)
	caminhoAtual := ""

	// Header
	lblIcone := canvas.NewText("📂", CorVerde)
	lblIcone.TextSize = 28
	lblIcone.Alignment = fyne.TextAlignCenter

	lblTitulo := canvas.NewText("Selecionar Pasta do Cofre", CorVerde)
	lblTitulo.TextSize = 16
	lblTitulo.TextStyle = fyne.TextStyle{Bold: true}
	lblTitulo.Alignment = fyne.TextAlignCenter

	// Barra de caminho
	lblCaminho := canvas.NewText("  "+nomeRemoto+":/", CorTexto)
	lblCaminho.TextSize = 12

	// Lista de pastas
	containerLista := container.NewVBox()
	scrollLista := container.NewVScroll(containerLista)
	scrollLista.SetMinSize(fyne.NewSize(0, 280))

	var dialogo *widget.PopUp

	// Função de carregamento
	var carregarPastas func()
	carregarPastas = func() {
		containerLista.RemoveAll()

		// Label "carregando"
		lblCarregando := canvas.NewText("🔄 Carregando pastas...", CorTextoSec)
		lblCarregando.TextSize = 12
		containerLista.Add(lblCarregando)
		containerLista.Refresh()

		// Atualizar caminho
		exibicao := caminhoAtual
		if exibicao == "" {
			exibicao = "/"
		}
		lblCaminho.Text = "  " + nomeRemoto + ":" + exibicao
		lblCaminho.Refresh()

		// Carregar em goroutine
		go func() {
			dirs := gerenciador.ListarDiretoriosRemoto(nomeRemoto, caminhoAtual)

			// Atualizar UI na thread Fyne
			containerLista.RemoveAll()

			if len(dirs) == 0 {
				lblVazio := canvas.NewText("📁 Nenhuma subpasta encontrada.\nVocê pode selecionar esta pasta.", CorTextoSec)
				lblVazio.TextSize = 12
				containerLista.Add(lblVazio)
			} else {
				for _, nomePasta := range dirs {
					np := nomePasta // captura
					btn := widget.NewButton("📁  "+np, func() {
						// Entrar na pasta (duplo clique simulado com clique normal)
						if caminhoAtual != "" {
							caminhoAtual = caminhoAtual + "/" + np
						} else {
							caminhoAtual = np
						}
						carregarPastas()
					})
					btn.Importance = widget.LowImportance
					btn.Alignment = widget.ButtonAlignLeading
					containerLista.Add(btn)
				}
			}
			containerLista.Refresh()
		}()
	}

	// Botão voltar
	btnVoltar := widget.NewButton("⬆ Voltar", func() {
		if caminhoAtual == "" {
			return
		}
		partes := splitCaminho(caminhoAtual)
		if len(partes) <= 1 {
			caminhoAtual = ""
		} else {
			caminhoAtual = joinCaminho(partes[:len(partes)-1])
		}
		carregarPastas()
	})

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
		cam := caminhoAtual
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
	carregarPastas()

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
