package gui

import (
	"errors"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/eufrauzino/runtime-crypt-go/internal/core"
)

// Demanda 030: senha errada fica no diálogo, com "Senha errada." embaixo do
// campo, o campo focado e o texto selecionado.

// telaTeste faz o papel da thread da tela: o resultado da conferência entra
// na fila e o teste o roda na própria goroutine.
type telaTeste chan func()

// rodar espera o próximo resultado da conferência e o aplica na tela.
func (f telaTeste) rodar(t *testing.T) {
	t.Helper()
	select {
	case fn := <-f:
		fn()
	case <-time.After(5 * time.Second):
		t.Fatal("a conferência não voltou")
	}
}

func novoDialogoSenhaTeste(t *testing.T, conferir func(string) error) (*dialogoSenha, fyne.Window, chan string, telaTeste) {
	t.Helper()
	a := test.NewTempApp(t)
	w := a.NewWindow("x")
	w.Resize(fyne.NewSize(600, 500))
	saida := make(chan string, 2)
	d := novoDialogoSenha(w, "fotos", "Desbloquear", conferir, func(s string) { saida <- s })
	tela := make(telaTeste, 4)
	d.naTela = func(fn func()) { tela <- fn }
	d.mostrar()
	return d, w, saida, tela
}

func TestDialogoSenhaErradaFicaAbertoComAFrase(t *testing.T) {
	tentativas := 0
	d, w, saida, tela := novoDialogoSenhaTeste(t, func(s string) error {
		tentativas++
		if s == "certa" {
			return nil
		}
		return core.ErrSenhaErrada
	})

	if d.lblErro.Visible() {
		t.Fatal("a frase de erro não aparece antes de tentar")
	}
	test.Type(d.entrySenha, "errada")
	test.Tap(d.btnConfirmar)
	if !d.btnConfirmar.Disabled() {
		t.Error("enquanto confere, o botão fica desligado")
	}
	tela.rodar(t)

	if !d.lblErro.Visible() || d.lblErro.Text != "Senha errada." {
		t.Errorf("frase = %q (visível %v)", d.lblErro.Text, d.lblErro.Visible())
	}
	if !d.popup.Visible() {
		t.Error("o diálogo deveria continuar aberto")
	}
	if w.Canvas().Focused() != d.entrySenha {
		t.Error("o campo de senha deveria estar com o foco")
	}
	if got := d.entrySenha.SelectedText(); got != "errada" {
		t.Errorf("texto selecionado = %q, quer o texto todo", got)
	}
	if d.btnConfirmar.Disabled() {
		t.Error("depois de conferir o botão volta")
	}
	select {
	case s := <-saida:
		t.Fatalf("o diálogo devolveu %q com a senha errada", s)
	default:
	}

	// Sem limite: erra de novo, depois acerta. Digitar com o texto
	// selecionado troca o texto todo.
	test.Tap(d.btnConfirmar)
	tela.rodar(t)
	if tentativas != 2 || !d.lblErro.Visible() {
		t.Fatalf("segunda tentativa: %d, frase visível %v", tentativas, d.lblErro.Visible())
	}
	test.Type(d.entrySenha, "certa")
	if d.entrySenha.Text != "certa" {
		t.Fatalf("o texto selecionado deveria ser trocado: %q", d.entrySenha.Text)
	}
	test.Tap(d.btnConfirmar)
	tela.rodar(t)

	select {
	case s := <-saida:
		if s != "certa" {
			t.Errorf("devolveu %q", s)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("a senha certa deveria fechar o diálogo")
	}
	if d.popup.Visible() {
		t.Error("o diálogo deveria ter fechado")
	}
}

func TestDialogoSenhaOutroErroFechaEDevolveASenha(t *testing.T) {
	d, _, saida, tela := novoDialogoSenhaTeste(t, func(string) error {
		return &core.ErroConferirSenha{Err: errors.New("sem rclone.conf")}
	})
	test.Type(d.entrySenha, "qualquer")
	test.Tap(d.btnConfirmar)
	tela.rodar(t)
	select {
	case s := <-saida:
		if s != "qualquer" {
			t.Errorf("devolveu %q", s)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("outro erro deveria fechar o diálogo")
	}
	if d.lblErro.Visible() {
		t.Error("outro erro não mostra 'Senha errada.'")
	}
}

func TestDialogoSenhaCancelarEVazio(t *testing.T) {
	chamou := false
	d, _, saida, _ := novoDialogoSenhaTeste(t, func(string) error { chamou = true; return nil })
	test.Tap(d.btnConfirmar) // vazio: nada acontece
	test.Tap(d.btnCancelar)
	if s := <-saida; s != "" {
		t.Errorf("cancelar devolveu %q", s)
	}
	if chamou {
		t.Error("senha vazia não deveria ser conferida")
	}
}

func TestDialogoSenhaEnterConfere(t *testing.T) {
	d, _, saida, tela := novoDialogoSenhaTeste(t, func(string) error { return nil })
	test.Type(d.entrySenha, "certa")
	d.entrySenha.TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
	tela.rodar(t)
	select {
	case s := <-saida:
		if s != "certa" {
			t.Errorf("devolveu %q", s)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Enter deveria confirmar")
	}
}
