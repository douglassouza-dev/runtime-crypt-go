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

// fraseEmbaixoDoCampo confere que a frase entrou no layout logo abaixo do
// campo (sem o Refresh do contêiner ela ficava no topo do diálogo, por cima
// do ícone).
func fraseEmbaixoDoCampo(t *testing.T, d *dialogoSenha) {
	t.Helper()
	campo, frase := d.entrySenha, d.lblErro
	if frase.Position().Y < campo.Position().Y+campo.Size().Height {
		t.Errorf("a frase (y=%v) não está embaixo do campo (y=%v, altura %v)", frase.Position().Y, campo.Position().Y, campo.Size().Height)
	}
	if frase.Size().Height <= 0 || frase.Size().Width <= 0 {
		t.Errorf("a frase não tem tamanho: %v", frase.Size())
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
	fraseEmbaixoDoCampo(t, d)
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

// As duas frases de configuração ficam embaixo do campo, como "Senha
// errada.", e o diálogo continua aberto.
func TestDialogoSenhaSemConferirMostraAFraseEFicaAberto(t *testing.T) {
	casos := map[string]*core.ErroConferirSenha{
		"Não destrancou: não deu para ler a configuração do rclone.":                           {Motivo: core.NaoLeuConfiguracao, Err: errors.New("exit status 1")},
		"Não destrancou: a configuração deste cofre está incompleta. Conecte o cofre de novo.": {Motivo: core.ConfigIncompleta, Err: errors.New("sem password")},
	}
	for quer, erro := range casos {
		t.Run(quer, func(t *testing.T) {
			d, w, saida, tela := novoDialogoSenhaTeste(t, func(string) error { return erro })
			test.Type(d.entrySenha, "qualquer")
			test.Tap(d.btnConfirmar)
			tela.rodar(t)

			if !d.lblErro.Visible() || d.lblErro.Text != quer {
				t.Errorf("frase = %q (visível %v)", d.lblErro.Text, d.lblErro.Visible())
			}
			fraseEmbaixoDoCampo(t, d)
			if !d.popup.Visible() {
				t.Error("o diálogo deveria continuar aberto")
			}
			if w.Canvas().Focused() != d.entrySenha {
				t.Error("o campo de senha deveria estar com o foco")
			}
			if got := d.entrySenha.SelectedText(); got != "qualquer" {
				t.Errorf("texto selecionado = %q", got)
			}
			select {
			case s := <-saida:
				t.Fatalf("o diálogo devolveu %q", s)
			default:
			}
			test.Tap(d.btnCancelar)
			if s := <-saida; s != "" {
				t.Errorf("cancelar devolveu %q", s)
			}
		})
	}
}

// Um erro que não é de conferência fecha o diálogo e devolve a senha:
// Destrancar confere de novo e devolve o erro.
func TestDialogoSenhaOutroErroFechaEDevolveASenha(t *testing.T) {
	d, _, saida, tela := novoDialogoSenhaTeste(t, func(string) error { return errors.New("outro") })
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
		t.Error("outro erro não mostra frase embaixo do campo")
	}
}

func TestTextoFalhaSenha(t *testing.T) {
	casos := []struct {
		err  error
		quer string
		ok   bool
	}{
		{core.ErrSenhaErrada, "Senha errada.", true},
		{&core.ErroConferirSenha{Motivo: core.NaoLeuConfiguracao}, "Não destrancou: não deu para ler a configuração do rclone.", true},
		{&core.ErroConferirSenha{Motivo: core.ConfigIncompleta}, "Não destrancou: a configuração deste cofre está incompleta. Conecte o cofre de novo.", true},
		{errors.New("x"), "", false},
		{nil, "", false},
	}
	for _, c := range casos {
		if got, ok := TextoFalhaSenha(c.err); got != c.quer || ok != c.ok {
			t.Errorf("TextoFalhaSenha(%v) = %q, %v", c.err, got, ok)
		}
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
