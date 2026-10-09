package core

import (
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
)

// Testes da demanda 002: trancar só informa sucesso quando a desmontagem
// aconteceu.

func TestDesmontarProcessoQueNaoMorreDevolveFalseComMotivo(t *testing.T) {
	g, _ := novoMontadorFalso(t)
	if ok, msg, _ := g.MontarUnidade("cofre", "V", "", nil); !ok {
		t.Fatal(msg)
	}
	pid := pidDaMontagem(t, g, "V")
	var pedidos []bool
	// Um rclone que ignora Interrupt e Kill.
	g.sinalizar = func(_ *os.Process, forcar bool) error {
		pedidos = append(pedidos, forcar)
		return nil
	}
	g.esperaEncerrar = 200 * time.Millisecond

	ok, msg := g.DesmontarUnidade("V")

	if ok {
		t.Fatalf("DesmontarUnidade deveria devolver false: %q", msg)
	}
	if !strings.Contains(msg, "nao terminou") || !strings.Contains(msg, "V") {
		t.Errorf("a mensagem deveria dizer o motivo: %q", msg)
	}
	if len(pedidos) != 3 || pedidos[0] || !pedidos[1] || !pedidos[2] {
		t.Errorf("esperava Interrupt, Kill, Kill; houve %v", pedidos)
	}
	g.mu.Lock()
	_, noMapa := g.montagens["V"]
	g.mu.Unlock()
	if !noMapa {
		t.Error("com o processo vivo, a montagem não pode sair do mapa")
	}
	if !processoVivoNoSO(pid) {
		t.Error("o processo deveria continuar vivo neste teste")
	}

	// Limpeza com o sinal de verdade.
	g.sinalizar = sinalizarProcesso
	if ok, msg := g.DesmontarUnidade("V"); !ok {
		t.Errorf("limpeza: %q", msg)
	}
}

func TestDesmontarProcessoQueTerminaComInterruptDevolveTrue(t *testing.T) {
	g, _ := novoMontadorFalso(t)
	if ok, msg, _ := g.MontarUnidade("cofre", "V", "", nil); !ok {
		t.Fatal(msg)
	}
	pid := pidDaMontagem(t, g, "V")
	var pedidos []bool
	g.sinalizar = func(p *os.Process, forcar bool) error {
		pedidos = append(pedidos, forcar)
		return sinalizarProcesso(p, forcar)
	}

	ok, msg := g.DesmontarUnidade("V")

	if !ok {
		t.Fatalf("ok=false msg=%q", msg)
	}
	if processoVivoNoSO(pid) {
		t.Errorf("processo %d continua vivo", pid)
	}
	g.mu.Lock()
	n := len(g.montagens)
	g.mu.Unlock()
	if n != 0 {
		t.Errorf("a montagem deveria sair do mapa; sobrou %d", n)
	}
	if len(pedidos) == 0 || pedidos[0] {
		t.Errorf("o primeiro pedido deveria ser Interrupt; houve %v", pedidos)
	}
	if osWindows() {
		// Interrupt não existe para processos no Windows: o segundo pedido é Kill, sem espera.
		if len(pedidos) != 2 || !pedidos[1] {
			t.Errorf("no Windows esperava Interrupt (recusado) e Kill; houve %v", pedidos)
		}
	} else if len(pedidos) != 1 {
		t.Errorf("fora do Windows o Interrupt basta; houve %v", pedidos)
	}
}

func TestDesmontarComPontoQueNaoSomeDevolveFalse(t *testing.T) {
	g, _ := novoMontadorFalso(t)
	if ok, msg, _ := g.MontarUnidade("cofre", "V", "", nil); !ok {
		t.Fatal(msg)
	}
	pid := pidDaMontagem(t, g, "V")
	g.pontoExiste = func(string) bool { return true } // a letra não some
	g.esperaEncerrar = 300 * time.Millisecond

	ok, msg := g.DesmontarUnidade("V")

	if ok || !strings.Contains(msg, "continua visivel") {
		t.Errorf("ok=%v msg=%q", ok, msg)
	}
	if processoVivoNoSO(pid) {
		t.Errorf("processo %d continua vivo", pid)
	}
}

func TestDesmontarNoWindowsNaoGastaEsperaComInterrupt(t *testing.T) {
	g, _ := novoMontadorFalso(t)
	if ok, msg, _ := g.MontarUnidade("cofre", "V", "", nil); !ok {
		t.Fatal(msg)
	}
	// Simula o Windows: Interrupt é recusado na hora.
	g.sinalizar = func(p *os.Process, forcar bool) error {
		if !forcar {
			return os.ErrInvalid
		}
		return p.Kill()
	}
	inicio := time.Now()

	ok, msg := g.DesmontarUnidade("V")

	if !ok {
		t.Fatalf("msg=%q", msg)
	}
	if d := time.Since(inicio); d > 3*time.Second {
		t.Errorf("Interrupt recusado não deveria custar a espera de %v; levou %v", g.esperaEncerrar, d)
	}
}

// Mesmo critério de `rg -n "DesmontarUnidade\(nome\)" main.go`, sem
// depender do rg instalado.
func TestTravarCofreNaoPassaNomeComoLetra(t *testing.T) {
	dados, err := os.ReadFile("../../main.go")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(dados), "DesmontarUnidade(nome)") {
		t.Error("main.go ainda passa o nome do cofre como letra")
	}
}

func osWindows() bool { return runtime.GOOS == "windows" }
