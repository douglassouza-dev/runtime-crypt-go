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
	cofre := core.CofreStatus{Cofre: core.Cofre{Nome: "fotos"}, Montado: true, Letra: "V"}

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
	cofre := core.CofreStatus{Cofre: core.Cofre{Nome: "fotos"}, Montado: true, Letra: "V"}

	junto := strings.Join(textosDoCard(criarCardCofre(cofre, "", func() {})), "|")

	if strings.Contains(junto, "Não trancou") {
		t.Errorf("sem falha, o card não mostra \"Não trancou\": %q", junto)
	}
}
