package core

import (
	"os"
	"strings"
	"testing"
	"time"
)

// Testes da demanda 022: a senha da sessão só sai quando o cofre trancou.

func montarCofreComSenha(t *testing.T) *GerenciadorRClone {
	t.Helper()
	g, _ := novoGerenciadorFalso(t)
	g.Senhas.Armazenar("cofre", "segredo")
	if ok, msg, _ := g.Montagens.MontarUnidade("cofre", "V", "segredo", nil); !ok {
		t.Fatal(msg)
	}
	return g
}

func TestTrancarQueFalhaMantemASenhaEDevolveErro(t *testing.T) {
	g := montarCofreComSenha(t)
	// Um rclone que não termina: ignora Interrupt e Kill.
	g.Montagens.sinalizar = func(*os.Process, bool) error { return nil }
	g.Montagens.esperaEncerrar = 200 * time.Millisecond

	err := g.Trancar("cofre")

	if err == nil {
		t.Fatal("Trancar deveria devolver erro quando a desmontagem não termina")
	}
	if !strings.Contains(err.Error(), "nao terminou") {
		t.Errorf("o erro deveria trazer o motivo: %q", err)
	}
	if g.Senhas.Obter("cofre") != "segredo" {
		t.Error("com o cofre ainda montado, a senha da sessão tem de ficar")
	}
	if g.Montagens.ObterLetraPorRemoto("cofre") != "V" {
		t.Error("o cofre deveria continuar montado em V")
	}

	// Limpeza com o sinal de verdade.
	g.Montagens.sinalizar = sinalizarProcesso
	if err := g.Trancar("cofre"); err != nil {
		t.Errorf("limpeza: %v", err)
	}
}

func TestTrancarQueTerminaApagaASenha(t *testing.T) {
	g := montarCofreComSenha(t)

	if err := g.Trancar("cofre"); err != nil {
		t.Fatalf("Trancar: %v", err)
	}

	if g.Senhas.Existe("cofre") {
		t.Error("com o cofre trancado, a senha da sessão deveria sair")
	}
	if g.Montagens.ObterLetraPorRemoto("cofre") != "" {
		t.Error("o cofre não deveria continuar montado")
	}
}

func TestTrancarCofreQueNaoEstaMontadoApagaASenha(t *testing.T) {
	g, _ := novoGerenciadorFalso(t)
	g.Senhas.Armazenar("cofre", "segredo")

	if err := g.Trancar("cofre"); err != nil {
		t.Fatalf("cofre já trancado não é erro: %v", err)
	}
	if g.Senhas.Existe("cofre") {
		t.Error("a senha deveria sair")
	}
}
