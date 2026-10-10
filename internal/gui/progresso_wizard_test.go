package gui

import (
	"errors"
	"strings"
	"testing"

	"fyne.io/fyne/v2/test"

	"github.com/eufrauzino/runtime-crypt-go/internal/core"
)

// Demanda 018: o wizard mostra o passo e, no erro, o passo em que parou.
func TestProgressoMostraOPasso(t *testing.T) {
	a := test.NewTempApp(t)
	w := a.NewWindow("x")
	p := novoProgressoWizard(w, "Novo cofre")

	p.passo(core.PassoAutorizando)

	if p.lbl.Text != "Autorizando no navegador…" {
		t.Errorf("passo = %q", p.lbl.Text)
	}
	p.passo(core.PassoGravando)
	if p.lbl.Text != "Gravando o cofre…" {
		t.Errorf("passo = %q", p.lbl.Text)
	}
	p.esconder()
}

func TestMensagemDeErroDizOPasso(t *testing.T) {
	err := &core.ErroPasso{Passo: core.PassoAutorizando, Err: errors.New("o rclone authorize terminou sem token: access_denied")}

	msg := mensagemErroPasso(err)

	if !strings.Contains(msg, "Autorizando no navegador…") || !strings.Contains(msg, "access_denied") {
		t.Errorf("mensagem = %q", msg)
	}
	if mensagemErroPasso(errors.New("x")) != "x" {
		t.Error("erro sem passo vai como está")
	}
}

// Acoes implementa a interface do core.
var _ core.Interacao = (*Acoes)(nil)
