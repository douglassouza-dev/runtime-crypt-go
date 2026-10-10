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

// TextoPasso é como a tela escreve cada passo de core.Passo. provedor é o
// nome do provedor do wizard (ex.: "Google Drive"). A palavra "remoto" não
// aparece na tela.
func TextoPasso(p core.Passo, provedor string) string {
	switch p {
	case core.PassoAutorizando:
		return "Autorizando no navegador…"
	case core.PassoCriandoRemoto:
		return fmt.Sprintf("Configurando o %s…", provedor)
	case core.PassoGravando:
		return "Gravando o cofre…"
	}
	return string(p)
}

// verboPasso é o passo no infinitivo, para a mensagem de erro.
func verboPasso(p core.Passo, provedor string) string {
	switch p {
	case core.PassoAutorizando:
		return "autorizar no navegador"
	case core.PassoCriandoRemoto:
		return fmt.Sprintf("configurar o %s", provedor)
	case core.PassoGravando:
		return "gravar o cofre"
	}
	return string(p)
}

// TextoErroNoPasso é a mensagem de um passo que não deu certo.
const TextoErroNoPasso = "Não deu para %s: %s"

// mensagemErroPasso diz em que passo o caso de uso parou, quando ele diz.
func mensagemErroPasso(err error, provedor string) string {
	var ep *core.ErroPasso
	if errors.As(err, &ep) {
		return fmt.Sprintf(TextoErroNoPasso, verboPasso(ep.Passo, provedor), ep.Err.Error())
	}
	return err.Error()
}

// progressoWizard é o diálogo sem botões que mostra o passo atual.
type progressoWizard struct {
	lbl      *widget.Label
	dlg      *dialog.CustomDialog
	provedor string
}

func novoProgressoWizard(janela fyne.Window, titulo, provedor string) *progressoWizard {
	p := &progressoWizard{lbl: widget.NewLabel(""), provedor: provedor}
	fyne.DoAndWait(func() {
		p.dlg = dialog.NewCustomWithoutButtons(titulo, p.lbl, janela)
	})
	return p
}

// passo mostra o diálogo com o texto do passo (na thread da tela).
func (p *progressoWizard) passo(ps core.Passo) {
	fyne.Do(func() {
		p.lbl.SetText(TextoPasso(ps, p.provedor))
		p.dlg.Show()
	})
}

// esconder tira o diálogo da frente (ex.: antes do seletor de pasta).
func (p *progressoWizard) esconder() {
	fyne.Do(p.dlg.Hide)
}
