package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// DialogoSenha exibe um diálogo modal para entrada de senha de cofre.
func DialogoSenha(janelaPai fyne.Window, nomeCofre string, acao string) string {
	resultado := make(chan string, 1)

	entrySenha := widget.NewPasswordEntry()
	entrySenha.SetPlaceHolder("Digite a senha...")

	// Ícone
	lblIcone := canvas.NewText("🔐", CorVerde)
	lblIcone.TextSize = 36
	lblIcone.Alignment = fyne.TextAlignCenter

	// Título
	lblTitulo := canvas.NewText(acao+" cofre", CorVerde)
	lblTitulo.TextSize = 16
	lblTitulo.TextStyle = fyne.TextStyle{Bold: true}
	lblTitulo.Alignment = fyne.TextAlignCenter

	lblNome := canvas.NewText(nomeCofre, CorTextoSec)
	lblNome.TextSize = 13
	lblNome.Alignment = fyne.TextAlignCenter

	lblCampo := canvas.NewText("Senha do cofre:", CorTextoSec)
	lblCampo.TextSize = 12

	var dialogo *widget.PopUp

	btnCancelar := widget.NewButton("Cancelar", func() {
		resultado <- ""
		if dialogo != nil {
			dialogo.Hide()
		}
	})

	btnConfirmar := widget.NewButton(acao, func() {
		senha := entrySenha.Text
		if senha != "" {
			resultado <- senha
			if dialogo != nil {
				dialogo.Hide()
			}
		}
	})
	btnConfirmar.Importance = widget.HighImportance

	conteudo := container.NewVBox(
		lblIcone,
		lblTitulo,
		lblNome,
		widget.NewSeparator(),
		lblCampo,
		entrySenha,
		layout.NewSpacer(),
		container.NewHBox(btnCancelar, layout.NewSpacer(), btnConfirmar),
	)

	padded := container.NewPadded(conteudo)
	padded.Resize(fyne.NewSize(400, 280))

	dialogo = widget.NewModalPopUp(padded, janelaPai.Canvas())
	dialogo.Resize(fyne.NewSize(400, 280))
	dialogo.Show()

	janelaPai.Canvas().Focus(entrySenha)

	return <-resultado
}

// TipoMensagem define o tipo visual de uma mensagem.
type TipoMensagem string

const (
	MsgInfo  TipoMensagem = "info"
	MsgErro  TipoMensagem = "erro"
	MsgAviso TipoMensagem = "aviso"
)

// DialogoMensagem exibe um diálogo modal de mensagem (info/erro/aviso).
func DialogoMensagem(janelaPai fyne.Window, titulo string, mensagem string, tipo TipoMensagem) {
	resultado := make(chan struct{}, 1)

	icones := map[TipoMensagem]string{
		MsgInfo:  "✅",
		MsgErro:  "❌",
		MsgAviso: "⚠️",
	}

	cores := map[TipoMensagem]fyne.CanvasObject{
		MsgInfo:  canvas.NewText(titulo, CorVerde),
		MsgErro:  canvas.NewText(titulo, CorErro),
		MsgAviso: canvas.NewText(titulo, CorAviso),
	}

	icone := icones[tipo]
	if icone == "" {
		icone = "ℹ️"
	}

	lblIcone := canvas.NewText(icone, CorTexto)
	lblIcone.TextSize = 32
	lblIcone.Alignment = fyne.TextAlignCenter

	lblTitulo := cores[tipo]
	if t, ok := lblTitulo.(*canvas.Text); ok {
		t.TextSize = 14
		t.TextStyle = fyne.TextStyle{Bold: true}
		t.Alignment = fyne.TextAlignCenter
	}

	lblMensagem := widget.NewLabel(mensagem)
	lblMensagem.Wrapping = fyne.TextWrapWord
	lblMensagem.Alignment = fyne.TextAlignCenter

	var dialogo *widget.PopUp

	btnOk := widget.NewButton("OK", func() {
		resultado <- struct{}{}
		if dialogo != nil {
			dialogo.Hide()
		}
	})
	btnOk.Importance = widget.HighImportance

	conteudo := container.NewVBox(
		lblIcone,
		lblTitulo,
		widget.NewSeparator(),
		lblMensagem,
		layout.NewSpacer(),
		container.NewCenter(btnOk),
	)

	padded := container.NewPadded(conteudo)

	dialogo = widget.NewModalPopUp(padded, janelaPai.Canvas())
	dialogo.Resize(fyne.NewSize(420, 280))
	dialogo.Show()

	<-resultado
}
