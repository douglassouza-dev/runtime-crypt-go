package core

import (
	"os"
	"testing"
	"time"
)

// Demanda 001: o core sabe se o processo do rclone vive pelo resultado de
// cmd.Wait(), não por sinal de teste.

func TestMontagemVivaContinuaListadaEMortaSaiEmAte2s(t *testing.T) {
	// Given: um processo de longa duração (o rclone falso, que é o próprio
	// binário de teste em modo `mount`) registrado como montagem.
	g, _ := novoMontadorFalso(t)
	if ok, msg, _ := g.MontarUnidade("cofre", "V", "", nil); !ok {
		t.Fatal(msg)
	}
	pid := pidDaMontagem(t, g, "V")

	// Then: 5 leituras seguidas continuam vendo a montagem.
	for i := 1; i <= 5; i++ {
		m, ok := g.ObterMontagens()["cofre"]
		if !ok || m.Letra != "V" {
			t.Fatalf("leitura %d: ObterMontagens() = %+v, esperava cofre em V", i, g.ObterMontagens())
		}
	}

	// When: o processo é morto por fora do core.
	p, err := os.FindProcess(pid)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Kill(); err != nil {
		t.Fatalf("matar %d: %v", pid, err)
	}

	// Then: em até 2 s a montagem sai da lista.
	inicio := time.Now()
	saiu := esperarAte(2*time.Second, func() bool {
		_, ok := g.ObterMontagens()["cofre"]
		return !ok
	})
	if !saiu {
		t.Fatalf("depois de 2 s ObterMontagens() ainda lista o cofre: %+v", g.ObterMontagens())
	}
	t.Logf("saiu da lista em %v", time.Since(inicio))

	// Demanda 010: a montagem morta não some em silêncio. Ela fica no Status
	// como falhou, com o motivo, até alguém trancar ou destrancar de novo.
	st := g.Status()
	if len(st) != 1 || st[0].Estado != EstadoFalhou || st[0].Motivo != MotivoProcessoTerminou {
		t.Errorf("Status = %+v, quer V falhou com %q", st, MotivoProcessoTerminou)
	}
}

func TestStatusNaoApagaMontagemAoLer(t *testing.T) {
	g, _ := novoMontadorFalso(t)
	if ok, msg, _ := g.MontarUnidade("cofre", "V", "", nil); !ok {
		t.Fatal(msg)
	}
	defer g.DesmontarUnidade("V")

	for i := 0; i < 5; i++ {
		g.Status()
		g.ObterLetraPorRemoto("cofre")
	}

	g.mu.Lock()
	n := len(g.montagens)
	g.mu.Unlock()
	if n != 1 {
		t.Fatalf("mapa tem %d montagens depois de ler 10 vezes, esperava 1", n)
	}
}

func TestLetraLiberadaDepoisQueOProcessoMorre(t *testing.T) {
	g, _ := novoMontadorFalso(t)
	if ok, msg, _ := g.MontarUnidade("cofre", "V", "", nil); !ok {
		t.Fatal(msg)
	}
	p, _ := os.FindProcess(pidDaMontagem(t, g, "V"))
	p.Kill()
	if !esperarAte(2*time.Second, func() bool { return len(g.ObterMontagens()) == 0 }) {
		t.Fatal("montagem morta continua em ObterMontagens")
	}

	ok, msg, _ := g.MontarUnidade("cofre", "V", "", nil)

	if !ok {
		t.Fatalf("a letra V deveria estar livre de novo: %q", msg)
	}
	// Demanda 010: a nova montagem toma o lugar da que falhou.
	if st := g.Status(); len(st) != 1 || st[0].Estado != EstadoMontado {
		t.Errorf("Status = %+v, quer só a nova montagem, montada", st)
	}
	g.DesmontarUnidade("V")
}
