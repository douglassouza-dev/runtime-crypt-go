package gui

import (
	"errors"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/eufrauzino/runtime-crypt-go/internal/core"
)

// Demanda 018: depois do wizard, a criação/conexão mostra em que passo está
// (autorizando, criando remoto, gravando) e, se der errado, o passo que parou.

// TextoPasso é como a tela escreve cada passo de core.Passo.
func TextoPasso(p core.Passo) string {
	switch p {
	case core.PassoAutorizando:
		return "Autorizando no navegador…"
	case core.PassoCriandoRemoto:
		return "Criando o remoto…"
	case core.PassoGravando:
		return "Gravando o cofre…"
	}
	return string(p)
}

// TextoErroNoPasso abre a mensagem de erro de um passo.
const TextoErroNoPasso = "Parou em \"%s\"\n\n%s"

// mensagemErroPasso diz em que passo o caso de uso parou, quando ele diz.
func mensagemErroPasso(err error) string {
	var ep *core.ErroPasso
	if errors.As(err, &ep) {
		return fmt.Sprintf(TextoErroNoPasso, TextoPasso(ep.Passo), ep.Err.Error())
	}
	return err.Error()
}

// progressoWizard é o diálogo sem botões que mostra o passo atual.
type progressoWizard struct {
	lbl *widget.Label
	dlg *dialog.CustomDialog
}

func novoProgressoWizard(janela fyne.Window, titulo string) *progressoWizard {
	p := &progressoWizard{lbl: widget.NewLabel("")}
	fyne.DoAndWait(func() {
		p.dlg = dialog.NewCustomWithoutButtons(titulo, p.lbl, janela)
	})
	return p
}

// passo mostra o diálogo com o texto do passo (na thread da tela).
func (p *progressoWizard) passo(ps core.Passo) {
	fyne.Do(func() {
		p.lbl.SetText(TextoPasso(ps))
		p.dlg.Show()
	})
}

// esconder tira o diálogo da frente (ex.: antes do seletor de pasta).
func (p *progressoWizard) esconder() {
	fyne.Do(p.dlg.Hide)
}
