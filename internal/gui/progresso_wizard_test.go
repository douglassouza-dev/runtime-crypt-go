package gui

import (
	"errors"
	"strings"
	"testing"

	"fyne.io/fyne/v2/test"

	"github.com/eufrauzino/runtime-crypt-go/internal/core"
)

// Demanda 018: o wizard mostra o passo e, no erro, o passo que não deu certo.
func TestProgressoMostraOPasso(t *testing.T) {
	a := test.NewTempApp(t)
	w := a.NewWindow("x")
	p := novoProgressoWizard(w, "Novo cofre", "Google Drive")

	for passo, quer := range map[core.Passo]string{
		core.PassoAutorizando:   "Autorizando no navegador…",
		core.PassoCriandoRemoto: "Configurando o Google Drive…",
		core.PassoGravando:      "Gravando o cofre…",
	} {
		p.passo(passo)
		if p.lbl.Text != quer {
			t.Errorf("passo %s = %q, quer %q", passo, p.lbl.Text, quer)
		}
		if strings.Contains(strings.ToLower(p.lbl.Text), "remoto") {
			t.Errorf("a tela não fala em remoto: %q", p.lbl.Text)
		}
	}
	p.esconder()
}

func TestMensagemDeErroDizOPasso(t *testing.T) {
	casos := map[core.Passo]string{
		core.PassoAutorizando:   "Não deu para autorizar no navegador: x",
		core.PassoCriandoRemoto: "Não deu para configurar o Dropbox: x",
		core.PassoGravando:      "Não deu para gravar o cofre: x",
	}
	for passo, quer := range casos {
		err := &core.ErroPasso{Passo: passo, Err: errors.New("x")}
		if got := mensagemErroPasso(err, "Dropbox"); got != quer {
			t.Errorf("%s: %q, quer %q", passo, got, quer)
		}
	}
	if mensagemErroPasso(errors.New("x"), "Dropbox") != "x" {
		t.Error("erro sem passo vai como está")
	}
}

// Acoes implementa a interface do core.
var _ core.Interacao = (*Acoes)(nil)
