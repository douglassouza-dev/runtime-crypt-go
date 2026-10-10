package core

import (
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// Testes da demanda 010: montado = processo vivo E ponto de montagem existe.
// O ponto de montagem é uma pasta temporária, conferida com o os.Stat de
// verdade (caminhoExiste), sem WinFsp nem FUSE.

func novoMontadorComPasta(t *testing.T) (*GerenciadorMontagem, string) {
	t.Helper()
	f := novoRcloneFalso(t)
	g := NovoGerenciadorMontagem(f.exe, NovoConfigVfs())
	raiz := t.TempDir()
	g.caminhoPonto = func(letra string) string { return filepath.Join(raiz, letra) }
	// A pasta temporária não some sozinha quando o rclone falso morre.
	g.esperaEncerrar = 200 * time.Millisecond
	// g.pontoExiste fica o de produção: os.Stat.
	ponto := g.caminhoPonto("V")
	if err := os.Mkdir(ponto, 0o755); err != nil {
		t.Fatal(err)
	}
	if ok, msg, _ := g.MontarUnidade("cofre", "V", "", nil); !ok {
		t.Fatal(msg)
	}
	t.Cleanup(func() { g.DesmontarTodas() })
	return g, ponto
}

func estadoUnico(t *testing.T, g *GerenciadorMontagem) StatusMontagem {
	t.Helper()
	st := g.Status()
	if len(st) != 1 {
		t.Fatalf("Status = %+v, quer uma montagem", st)
	}
	return st[0]
}

func TestSaudeProcessoVivoEPontoExistenteEMontado(t *testing.T) {
	g, ponto := novoMontadorComPasta(t)

	st := estadoUnico(t, g)

	if st.Estado != EstadoMontado || st.Motivo != "" || !st.Ativo || st.PontoMontagem != ponto {
		t.Errorf("Status = %+v, quer montado em %s", st, ponto)
	}
	if _, ok := g.ObterMontagens()["cofre"]; !ok {
		t.Error("montado deveria aparecer em ObterMontagens")
	}
}

func TestSaudeProcessoVivoEPontoInexistenteFalhou(t *testing.T) {
	g, ponto := novoMontadorComPasta(t)
	pid := pidDaMontagem(t, g, "V")

	if err := os.Remove(ponto); err != nil {
		t.Fatal(err)
	}
	st := estadoUnico(t, g)

	if st.Estado != EstadoFalhou || st.Motivo != "ponto de montagem sumiu" || st.Ativo {
		t.Errorf("Status = %+v, quer falhou com \"ponto de montagem sumiu\"", st)
	}
	if !processoVivoNoSO(pid) {
		t.Error("o processo deveria continuar vivo neste teste")
	}
	if len(g.ObterMontagens()) != 0 || g.ObterLetraPorRemoto("cofre") != "" {
		t.Error("falhou não pode aparecer como montado")
	}
}

func TestSaudeProcessoMortoEPontoExistenteFalhou(t *testing.T) {
	g, _ := novoMontadorComPasta(t)
	p, err := os.FindProcess(pidDaMontagem(t, g, "V"))
	if err != nil {
		t.Fatal(err)
	}

	if err := p.Kill(); err != nil {
		t.Fatal(err)
	}
	if !esperarAte(5*time.Second, func() bool { return !estadoUnico(t, g).Ativo }) {
		t.Fatal("o processo morreu e a montagem continua montada")
	}
	st := estadoUnico(t, g)

	if st.Estado != EstadoFalhou || st.Motivo != "processo terminou" {
		t.Errorf("Status = %+v, quer falhou com \"processo terminou\"", st)
	}
}

func TestSaudePontoQueNaoRespondeFalhaSemTravarStatus(t *testing.T) {
	g, _ := novoMontadorComPasta(t)
	g.limitePonto = 100 * time.Millisecond
	solta := make(chan struct{})
	defer close(solta)
	var chamadas atomic.Int32
	g.pontoExiste = func(string) bool { chamadas.Add(1); <-solta; return true }

	inicio := time.Now()
	st := estadoUnico(t, g)
	st2 := estadoUnico(t, g) // a conferência anterior ainda não voltou

	if d := time.Since(inicio); d > time.Second {
		t.Errorf("Status levou %v com o ponto travado", d)
	}
	if st.Estado != EstadoFalhou || !strings.Contains(st.Motivo, "nao respondeu") {
		t.Errorf("Status = %+v, quer falhou com \"nao respondeu\"", st)
	}
	if st2.Estado != EstadoFalhou || chamadas.Load() != 1 {
		t.Errorf("segunda leitura: %+v, %d conferências em andamento (quer 1)", st2, chamadas.Load())
	}
}

func TestSaudeDestrancarDeNovoSubstituiAFalha(t *testing.T) {
	g, ponto := novoMontadorComPasta(t)
	pidAntigo := pidDaMontagem(t, g, "V")
	os.Remove(ponto) // falhou: ponto de montagem sumiu, processo vivo
	if estadoUnico(t, g).Estado != EstadoFalhou {
		t.Fatal("esperava falhou")
	}
	// O ponto volta quando o novo rclone sobe, depois que o antigo saiu.
	go func() {
		esperarAte(10*time.Second, func() bool { return !processoVivoNoSO(pidAntigo) })
		os.Mkdir(ponto, 0o755)
	}()

	ok, msg, _ := g.MontarUnidade("cofre", "V", "", nil)

	if !ok {
		t.Fatalf("destrancar de novo: %q", msg)
	}
	if processoVivoNoSO(pidAntigo) {
		t.Errorf("o rclone da montagem que falhou (pid %d) deveria ter sido encerrado", pidAntigo)
	}
	if st := estadoUnico(t, g); st.Estado != EstadoMontado {
		t.Errorf("Status = %+v", st)
	}
}
