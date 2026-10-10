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

// TextoFalhaSenha é a frase embaixo do campo de senha para o erro de
// conferir (demanda 030, aprovadas pela UI). ok=false: o erro não é de
// conferência e o diálogo fecha.
func TextoFalhaSenha(err error) (string, bool) {
	if errors.Is(err, core.ErrSenhaErrada) {
		return TextoSenhaErrada, true
	}
	var e *core.ErroConferirSenha
	if errors.As(err, &e) {
		return e.Error(), true
	}
	return "", false
}

// DialogoSenha exibe um diálogo modal para entrada de senha de cofre e
// devolve a senha ("" se cancelado).
//
// Demanda 030: com conferir, a senha é conferida antes de o diálogo fechar.
// Se a senha não confere (core.ErrSenhaErrada) ou não dá para conferir
// (*core.ErroConferirSenha), o diálogo fica aberto com a frase embaixo do
// campo (TextoFalhaSenha), o campo focado e o texto selecionado; o cofre
// continua trancado. Sem limite de tentativas. Qualquer outro erro fecha o
// diálogo e devolve a senha: core.Destrancar confere de novo e devolve o erro.
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
	lblErro      *widget.Label
	btnCancelar  *widget.Button
	btnConfirmar *widget.Button
	popup        *widget.PopUp
	conteudo     *fyne.Container
	conferir     func(string) error
	fim          func(string)
	encerrado    bool

	// naTela leva o resultado da conferência para a thread da tela. Em
	// produção é fyne.Do; o teste troca por uma fila que ele mesmo roda,
	// porque o driver de teste do Fyne roda fyne.Do na goroutine de quem
	// chama.
	naTela func(func())
}

// TextoCampoSenha é o placeholder do campo de senha (aprovado na 031).
const TextoCampoSenha = "Senha"

func novoDialogoSenha(janelaPai fyne.Window, nomeCofre string, acao string, conferir func(string) error, fim func(string)) *dialogoSenha {
	d := &dialogoSenha{janela: janelaPai, conferir: conferir, fim: fim, naTela: fyne.Do}

	d.entrySenha = widget.NewPasswordEntry()
	d.entrySenha.SetPlaceHolder(TextoCampoSenha)

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
	// As frases de configuração passam de uma linha: o rótulo quebra.
	d.lblErro = widget.NewLabel("")
	d.lblErro.Importance = widget.DangerImportance
	d.lblErro.Wrapping = fyne.TextWrapWord
	d.lblErro.Hide()

	d.btnCancelar = widget.NewButton("Cancelar", func() { d.encerrar("") })
	d.btnConfirmar = widget.NewButton(acao, d.confirmar)
	d.btnConfirmar.Importance = widget.HighImportance
	d.entrySenha.OnSubmitted = func(string) { d.confirmar() }

	d.conteudo = container.NewVBox(
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

	padded := container.NewPadded(d.conteudo)
	padded.Resize(fyne.NewSize(400, 330))

	d.popup = widget.NewModalPopUp(padded, janelaPai.Canvas())
	d.popup.Resize(fyne.NewSize(400, 330))
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
	if texto, ok := TextoFalhaSenha(err); ok {
		d.lblErro.SetText(texto)
		d.lblErro.Show()
		d.conteudo.Refresh() // a frase entra no layout embaixo do campo
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
