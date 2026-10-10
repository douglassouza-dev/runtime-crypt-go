package gui

import (
	"errors"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/eufrauzino/runtime-crypt-go/internal/core"
	"github.com/eufrauzino/runtime-crypt-go/internal/gui/frases"
)

// TextoSenhaErrada aparece embaixo do campo quando a senha não confere
// (demanda 030, aprovado pela UI).
const TextoSenhaErrada = "Senha errada."

// DialogoSenha exibe um diálogo modal para entrada de senha de cofre e
// devolve a senha ("" se cancelado).
//
// Demanda 030: com conferir, a senha é conferida antes de o diálogo fechar.
// Se conferir volta core.ErrSenhaErrada, o diálogo fica aberto com
// TextoSenhaErrada embaixo do campo, o campo focado e o texto selecionado.
// Sem limite de tentativas. Outro erro de conferir fecha o diálogo e devolve
// a senha: quem chama (core.Destrancar) confere de novo e mostra o erro.
func DialogoSenha(janelaPai fyne.Window, nomeCofre string, acao string, conferir func(string) error) string {
	resultado := make(chan string, 1)
	d := novoDialogoSenha(janelaPai, nomeCofre, acao, conferir, func(senha string) { resultado <- senha })
	d.mostrar()
	return <-resultado
}

// dialogoSenha é o diálogo de DialogoSenha, separado para o teste com o
// driver de teste do Fyne.
type dialogoSenha struct {
	janela       fyne.Window
	entrySenha   *widget.Entry
	lblErro      *canvas.Text
	btnCancelar  *widget.Button
	btnConfirmar *widget.Button
	popup        *widget.PopUp
	conferir     func(string) error
	fim          func(string)
	encerrado    bool

	// naTela leva o resultado da conferência para a thread da tela. Em
	// produção é fyne.Do; o teste troca por uma fila que ele mesmo roda,
	// porque o driver de teste do Fyne roda fyne.Do na goroutine de quem
	// chama.
	naTela func(func())
}

func novoDialogoSenha(janelaPai fyne.Window, nomeCofre string, acao string, conferir func(string) error, fim func(string)) *dialogoSenha {
	d := &dialogoSenha{janela: janelaPai, conferir: conferir, fim: fim, naTela: fyne.Do}

	d.entrySenha = widget.NewPasswordEntry()
	d.entrySenha.SetPlaceHolder("Digite a senha...")

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

	// Demanda 030: embaixo do campo, escondido até a senha não conferir.
	d.lblErro = canvas.NewText("", CorErro)
	d.lblErro.TextSize = 12
	d.lblErro.Hide()

	d.btnCancelar = widget.NewButton("Cancelar", func() { d.encerrar("") })
	d.btnConfirmar = widget.NewButton(acao, d.confirmar)
	d.btnConfirmar.Importance = widget.HighImportance
	d.entrySenha.OnSubmitted = func(string) { d.confirmar() }

	conteudo := container.NewVBox(
		lblIcone,
		lblTitulo,
		lblNome,
		widget.NewSeparator(),
		lblCampo,
		d.entrySenha,
		d.lblErro,
		layout.NewSpacer(),
		container.NewHBox(d.btnCancelar, layout.NewSpacer(), d.btnConfirmar),
	)

	padded := container.NewPadded(conteudo)
	padded.Resize(fyne.NewSize(400, 300))

	d.popup = widget.NewModalPopUp(padded, janelaPai.Canvas())
	d.popup.Resize(fyne.NewSize(400, 300))
	return d
}

func (d *dialogoSenha) mostrar() {
	d.popup.Show()
	d.janela.Canvas().Focus(d.entrySenha)
}

// encerrar fecha o diálogo e entrega o resultado uma vez só.
func (d *dialogoSenha) encerrar(senha string) {
	if d.encerrado {
		return
	}
	d.encerrado = true
	d.popup.Hide()
	d.fim(senha)
}

// confirmar roda na thread da tela (clique ou Enter). A conferência chama o
// rclone, então vai para uma goroutine; o resultado volta pela fyne.Do.
func (d *dialogoSenha) confirmar() {
	senha := d.entrySenha.Text
	if senha == "" || d.encerrado || d.btnConfirmar.Disabled() {
		return
	}
	if d.conferir == nil {
		d.encerrar(senha)
		return
	}
	d.btnConfirmar.Disable()
	d.btnCancelar.Disable()
	conferir, naTela := d.conferir, d.naTela
	go func() {
		err := conferir(senha)
		naTela(func() { d.depoisDeConferir(senha, err) })
	}()
}

func (d *dialogoSenha) depoisDeConferir(senha string, err error) {
	d.btnConfirmar.Enable()
	d.btnCancelar.Enable()
	if errors.Is(err, core.ErrSenhaErrada) {
		d.lblErro.Text = TextoSenhaErrada
		d.lblErro.Show()
		d.lblErro.Refresh()
		d.janela.Canvas().Focus(d.entrySenha)
		d.entrySenha.TypedShortcut(&fyne.ShortcutSelectAll{})
		return
	}
	d.encerrar(senha)
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

// conteudoSair monta o diálogo da demanda 028: o texto e os dois botões,
// "Enviar agora" (principal) e "Sair". Não há botão de fechar.
func conteudoSair(texto string, aoEnviar, aoSair func()) fyne.CanvasObject {
	lblTitulo := canvas.NewText(frases.TituloSair, CorAviso)
	lblTitulo.TextSize = 14
	lblTitulo.TextStyle = fyne.TextStyle{Bold: true}
	lblTitulo.Alignment = fyne.TextAlignCenter

	lblTexto := widget.NewLabel(texto)
	lblTexto.Wrapping = fyne.TextWrapWord

	btnEnviar := widget.NewButton(frases.TextoEnviarAgora, aoEnviar)
	btnEnviar.Importance = widget.HighImportance
	btnSair := widget.NewButton(frases.TextoSair, aoSair)

	return container.NewPadded(container.NewVBox(
		lblTitulo,
		widget.NewSeparator(),
		lblTexto,
		layout.NewSpacer(),
		container.NewCenter(container.NewHBox(btnEnviar, btnSair)),
	))
}

// DialogoSairComPendencias mostra o diálogo da 028 e espera a escolha:
// true = "Enviar agora", false = "Sair".
func DialogoSairComPendencias(janelaPai fyne.Window, texto string) bool {
	escolha := make(chan bool, 1)
	var dialogo *widget.PopUp
	fechar := func(enviar bool) {
		escolha <- enviar
		if dialogo != nil {
			dialogo.Hide()
		}
	}
	fyne.DoAndWait(func() {
		conteudo := conteudoSair(texto, func() { fechar(true) }, func() { fechar(false) })
		dialogo = widget.NewModalPopUp(conteudo, janelaPai.Canvas())
		dialogo.Resize(fyne.NewSize(460, 280))
		dialogo.Show()
	})
	return <-escolha
}
