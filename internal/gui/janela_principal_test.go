package gui

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/eufrauzino/runtime-crypt-go/internal/core"
)

// textosDoCard junta os textos visíveis de um card.
func textosDoCard(o fyne.CanvasObject) []string {
	switch v := o.(type) {
	case *canvas.Text:
		return []string{v.Text}
	case *widget.Label:
		return []string{v.Text}
	case *widget.Button:
		return []string{"[" + v.Text + "]"}
	case *fyne.Container:
		var ts []string
		for _, f := range v.Objects {
			ts = append(ts, textosDoCard(f)...)
		}
		return ts
	}
	return nil
}

// Demanda 022: o Trancar que falha deixa o card destrancado, com o botão
// Trancar e a linha "Não trancou: {motivo}".
func TestCardComTrancarQueFalhouContinuaDestrancadoComMotivo(t *testing.T) {
	test.NewTempApp(t)
	cofre := core.CofreStatus{Cofre: core.Cofre{Nome: "fotos"}, Estado: core.EstadoMontado, Letra: "V"}

	card := criarCardCofre(cofre, "o rclone nao terminou", func() {})

	junto := strings.Join(textosDoCard(card), "|")
	if !strings.Contains(junto, "Destrancado") || !strings.Contains(junto, `V:\`) {
		t.Errorf("o card deveria continuar destrancado em V:\\; mostra %q", junto)
	}
	if !strings.Contains(junto, "Não trancou: o rclone nao terminou") {
		t.Errorf("falta a linha \"Não trancou\"; o card mostra %q", junto)
	}
	if !strings.Contains(junto, "[Trancar]") {
		t.Errorf("o botão deveria continuar \"Trancar\"; o card mostra %q", junto)
	}
}

func TestCardSemFalhaNaoMostraNaoTrancou(t *testing.T) {
	test.NewTempApp(t)
	cofre := core.CofreStatus{Cofre: core.Cofre{Nome: "fotos"}, Estado: core.EstadoMontado, Letra: "V"}

	junto := strings.Join(textosDoCard(criarCardCofre(cofre, "", func() {})), "|")

	if strings.Contains(junto, "Não trancou") {
		t.Errorf("sem falha, o card não mostra \"Não trancou\": %q", junto)
	}
}

// botaoDoCard acha o botão de ação do card.
func botaoDoCard(o fyne.CanvasObject) *widget.Button {
	switch v := o.(type) {
	case *widget.Button:
		return v
	case *fyne.Container:
		for _, f := range v.Objects {
			if b := botaoDoCard(f); b != nil {
				return b
			}
		}
	}
	return nil
}

// Demanda 018: cada estado tem a frase e o botão certos.
func TestCardMostraOEstadoDoCore(t *testing.T) {
	test.NewTempApp(t)
	destrancando := core.CofreStatus{Cofre: core.Cofre{Nome: "a"}, Estado: core.EstadoMontando}
	naoSubiu := core.CofreStatus{Cofre: core.Cofre{Nome: "b"}, Estado: core.EstadoFalhou, Motivo: "sem winfsp"}
	caiu := core.CofreStatus{Cofre: core.Cofre{Nome: "c"}, Estado: core.EstadoFalhou, Caiu: true, Motivo: core.MotivoProcessoTerminou, Letra: "V"}
	trancado := core.CofreStatus{Cofre: core.Cofre{Nome: "d"}, Estado: core.EstadoDesmontado}

	casos := []struct {
		c          core.CofreStatus
		frase      string
		botao      string
		desabilita bool
	}{
		{destrancando, "Destrancando…", "Destrancar", true},
		{naoSubiu, "Não destrancou: sem winfsp", "Tentar de novo", false},
		{caiu, "Caiu: o rclone parou", "Tentar de novo", false},
		{trancado, "Trancado", "Destrancar", false},
	}
	for _, c := range casos {
		card := criarCardCofre(c.c, "", func() {})
		junto := strings.Join(textosDoCard(card), "|")
		if !strings.Contains(junto, c.frase) {
			t.Errorf("%s: card mostra %q, falta %q", c.c.Nome, junto, c.frase)
		}
		b := botaoDoCard(card)
		if b == nil || b.Text != c.botao || b.Disabled() != c.desabilita {
			t.Errorf("%s: botão = %+v, quer %q desabilitado=%v", c.c.Nome, b, c.botao, c.desabilita)
		}
	}
}

// Destrancando: o clique no botão desabilitado não chama a ação.
func TestCardDestrancandoIgnoraClique(t *testing.T) {
	test.NewTempApp(t)
	cliques := 0
	card := criarCardCofre(core.CofreStatus{Cofre: core.Cofre{Nome: "a"}, Estado: core.EstadoMontando}, "", func() { cliques++ })

	test.Tap(botaoDoCard(card))

	if cliques != 0 {
		t.Errorf("%d cliques chegaram à ação", cliques)
	}
}

// Demanda 025: enquanto envia, o card diz quantos arquivos faltam e o botão
// não aceita outro clique.
func TestCardEnviandoDesabilitaBotao(t *testing.T) {
	test.NewTempApp(t)
	c := core.CofreStatus{Cofre: core.Cofre{Nome: "a"}, Estado: core.EstadoMontado, Letra: "V", PontoMontagem: `V:\`, Enviando: 2}
	cliques := 0
	card := criarCardCofre(c, "", func() { cliques++ })

	junto := strings.Join(textosDoCard(card), "|")
	if !strings.Contains(junto, "Enviando 2 arquivos…") {
		t.Errorf("card = %q", junto)
	}
	b := botaoDoCard(card)
	test.Tap(b)
	if !b.Disabled() || cliques != 0 {
		t.Errorf("botão deveria estar desabilitado (cliques=%d)", cliques)
	}
}
