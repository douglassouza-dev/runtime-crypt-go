//go:build linux

package core

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

// Testes da demanda 013 no Linux: o ponto de montagem é uma pasta.

func TestPastaPontosPadraoFicaNoHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	if got := PastaPontosPadrao(); got != filepath.Join(home, "RuntimeCrypto") {
		t.Errorf("PastaPontosPadrao = %q", got)
	}
	g := NovoGerenciadorMontagem("rclone", NovoConfigVfs())
	if g.raizPontos != filepath.Join(home, "RuntimeCrypto") {
		t.Errorf("raizPontos = %q", g.raizPontos)
	}
}

func TestPastaDoCofreFicaDentroDaRaiz(t *testing.T) {
	g := &GerenciadorMontagem{raizPontos: "/raiz"}
	casos := map[string]string{
		"fotos:":     "/raiz/fotos",
		"a/b:":       "/raiz/a_b",
		`..\..\x:`:   `/raiz/.._.._x`,
		"..:":        "/raiz/_..",
		"Meus Docs:": "/raiz/Meus Docs",
	}
	for remoto, quer := range casos {
		if got := g.pastaDoCofre(remoto); got != quer {
			t.Errorf("pastaDoCofre(%q) = %q, quer %q", remoto, got, quer)
		}
	}
}

func TestPontoAtivoSoParaMontagem(t *testing.T) {
	pasta := t.TempDir()
	if pontoAtivo(pasta) {
		t.Error("pasta comum não é ponto de montagem")
	}
	if pontoAtivo(filepath.Join(pasta, "nao-existe")) {
		t.Error("pasta inexistente não é ponto de montagem")
	}
	// /proc é sempre outra montagem no Linux.
	if !pontoAtivo("/proc") {
		t.Error("/proc deveria contar como ponto de montagem")
	}
}

func TestPastaCriadaAntesDeMontarERemovidaDepois(t *testing.T) {
	g, f := novoMontadorFalso(t)
	ponto := filepath.Join(g.raizPontos, "cofre")
	var existiaNoMount atomic.Bool
	falso := g.pontoExiste
	g.pontoExiste = func(c string) bool {
		if c == ponto && len(f.chamadas()) > 0 {
			if _, err := os.Stat(ponto); err == nil {
				existiaNoMount.Store(true)
			}
		}
		return falso(c)
	}

	ok, msg, letra := g.MontarUnidade("cofre", "", "", nil)
	if !ok || letra != ponto {
		t.Fatalf("ok=%v msg=%q ponto=%q", ok, msg, letra)
	}
	if !existiaNoMount.Load() {
		t.Error("a pasta deveria existir quando o rclone mount roda")
	}
	if st := g.Status(); len(st) != 1 || st[0].PontoMontagem != ponto {
		t.Errorf("Status = %+v, quer ponto %s", st, ponto)
	}

	if ok, msg := g.DesmontarUnidade(letra); !ok {
		t.Fatal(msg)
	}
	if _, err := os.Stat(ponto); !os.IsNotExist(err) {
		t.Errorf("a pasta vazia deveria sair depois de desmontar: %v", err)
	}
}

func TestPastaComArquivosNaoERemovida(t *testing.T) {
	g, _ := novoMontadorFalso(t)
	ponto := filepath.Join(g.raizPontos, "cofre")
	if err := os.MkdirAll(ponto, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ponto, "ficou.txt"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if ok, msg, _ := g.MontarUnidade("cofre", "", "", nil); !ok {
		t.Fatal(msg)
	}
	if ok, msg := g.DesmontarUnidade(ponto); !ok {
		t.Fatal(msg)
	}
	if _, err := os.Stat(filepath.Join(ponto, "ficou.txt")); err != nil {
		t.Errorf("arquivo da pasta sumiu: %v", err)
	}
}

// O processo terminou mas a montagem FUSE ficou: fusermount -u solta.
func TestDesmontarSoltaMontagemPresa(t *testing.T) {
	g, _ := novoMontadorFalso(t)
	g.esperaEncerrar = 200 * time.Millisecond
	ponto := filepath.Join(g.raizPontos, "cofre")
	ok, msg, _ := g.MontarUnidade("cofre", "", "", nil)
	if !ok {
		t.Fatal(msg)
	}
	var presa atomic.Bool
	presa.Store(true)
	g.pontoExiste = func(c string) bool { return c == ponto && presa.Load() }
	var soltou atomic.Value
	g.desmontarPonto = func(c string) error { soltou.Store(c); presa.Store(false); return nil }

	if ok, msg := g.DesmontarUnidade(ponto); !ok {
		t.Fatalf("Desmontar: %q", msg)
	}
	if soltou.Load() != ponto {
		t.Errorf("desmontarPonto chamado com %v, quer %s", soltou.Load(), ponto)
	}
}
