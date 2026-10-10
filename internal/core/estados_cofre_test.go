package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Testes da demanda 018: o core decide o estado do cofre — desmontado,
// montando, montado ou falhou — e diz se a falha foi antes de subir ou
// depois (caiu).

func estadoDoRemoto(t *testing.T, g *GerenciadorMontagem, remoto string) EstadoRemoto {
	t.Helper()
	if e, ok := g.EstadosPorRemoto()[remoto]; ok {
		return e
	}
	return EstadoRemoto{Estado: EstadoDesmontado}
}

func TestEstadoVivoEPontoExisteMontado(t *testing.T) {
	g, ponto := novoMontadorComPasta(t)

	e := estadoDoRemoto(t, g, "cofre")

	if e.Estado != EstadoMontado || e.Caiu || e.Motivo != "" || e.Letra != "V" || e.PontoMontagem != ponto {
		t.Errorf("estado = %+v, quer montado em %s", e, ponto)
	}
}

func TestEstadoVivoEPontoAusenteFalhouCaiu(t *testing.T) {
	g, ponto := novoMontadorComPasta(t)
	if err := os.Remove(ponto); err != nil {
		t.Fatal(err)
	}

	e := estadoDoRemoto(t, g, "cofre")

	if e.Estado != EstadoFalhou || !e.Caiu || e.Motivo != MotivoPontoSumiu {
		t.Errorf("estado = %+v, quer falhou (caiu) com %q", e, MotivoPontoSumiu)
	}
}

func TestEstadoMortoEPontoExisteFalhouCaiu(t *testing.T) {
	g, _ := novoMontadorComPasta(t)
	p, err := os.FindProcess(pidDaMontagem(t, g, "V"))
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Kill(); err != nil {
		t.Fatal(err)
	}

	// O card confere a cada 3 s; o pronto-quando pede o estado em até 5 s.
	if !esperarAte(2*time.Second, func() bool { return estadoDoRemoto(t, g, "cofre").Estado == EstadoFalhou }) {
		t.Fatalf("estado = %+v, quer falhou", estadoDoRemoto(t, g, "cofre"))
	}
	if e := estadoDoRemoto(t, g, "cofre"); !e.Caiu || e.Motivo != MotivoProcessoTerminou {
		t.Errorf("estado = %+v, quer caiu com %q", e, MotivoProcessoTerminou)
	}
}

func TestEstadoSemProcessoDesmontado(t *testing.T) {
	g, _ := novoGerenciadorFalso(t)
	if ok, msg := g.Cofres.Adicionar("cofre", "drive", "Google Drive", "cofre_base:"); !ok {
		t.Fatal(msg)
	}

	lista := g.ListarCofres()

	if len(lista) != 1 || lista[0].Estado != EstadoDesmontado || lista[0].EstaMontado() || lista[0].Motivo != "" {
		t.Errorf("ListarCofres = %+v, quer desmontado", lista)
	}
}

// Durante a espera pela unidade: montando, e um segundo pedido não inicia
// outro rclone.
func TestEstadoDuranteAEsperaMontandoESegundoPedidoRecusado(t *testing.T) {
	f := novoRcloneFalso(t)
	g := NovoGerenciadorMontagem(f.exe, NovoConfigVfs())
	raiz := t.TempDir()
	g.caminhoPonto = func(letra string) string { return filepath.Join(raiz, letra) }
	g.pontoExiste = caminhoExiste // a pasta temporária faz o papel da unidade
	g.esperaEncerrar = 200 * time.Millisecond
	t.Cleanup(g.DesmontarTodas)

	feito := make(chan bool, 1)
	go func() {
		ok, _, _ := g.MontarUnidade("cofre", "V", "", nil)
		feito <- ok
	}()

	if !esperarAte(5*time.Second, func() bool { return estadoDoRemoto(t, g, "cofre").Estado == EstadoMontando }) {
		t.Fatalf("estado = %+v, quer montando", estadoDoRemoto(t, g, "cofre"))
	}
	f.esperarChamadas(1, 5*time.Second)

	ok, msg, _ := g.MontarUnidade("cofre", "W", "", nil)
	if ok || msg != MsgJaDestrancando {
		t.Errorf("segundo pedido: ok=%v msg=%q, quer recusa %q", ok, msg, MsgJaDestrancando)
	}
	mounts := 0
	for _, c := range f.chamadas() {
		if len(c.Args) > 0 && c.Args[0] == "mount" {
			mounts++
		}
	}
	if mounts != 1 {
		t.Errorf("%d `rclone mount` iniciados, quer 1", mounts)
	}

	// A unidade aparece: montado.
	if err := os.Mkdir(filepath.Join(raiz, "V"), 0o755); err != nil {
		t.Fatal(err)
	}
	select {
	case ok := <-feito:
		if !ok {
			t.Fatal("a montagem deveria subir")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("MontarUnidade não voltou")
	}
	if e := estadoDoRemoto(t, g, "cofre"); e.Estado != EstadoMontado {
		t.Errorf("estado = %+v, quer montado", e)
	}
}

// A montagem que nunca subiu: falhou sem Caiu, com o motivo curto (última
// linha do stderr, sem a data).
func TestEstadoMontagemQueNuncaSubiuFalhouSemCaiu(t *testing.T) {
	f := novoRcloneFalso(t)
	g := NovoGerenciadorMontagem(f.exe, NovoConfigVfs())
	raiz := t.TempDir()
	g.caminhoPonto = func(letra string) string { return filepath.Join(raiz, letra) }
	g.pontoExiste = caminhoExiste // a pasta temporária faz o papel da unidade
	f.falhar()
	t.Setenv("RCLONE_FALSO_STDERR", "2026/10/09 10:00:00 CRITICAL: Fatal error: cannot find winfsp")

	if ok, _, _ := g.MontarUnidade("cofre", "V", "", nil); ok {
		t.Fatal("deveria falhar")
	}
	e := estadoDoRemoto(t, g, "cofre")

	if e.Estado != EstadoFalhou || e.Caiu || e.Motivo != "CRITICAL: Fatal error: cannot find winfsp" {
		t.Errorf("estado = %+v, quer falhou (nunca subiu) com a última linha do rclone", e)
	}

	// Tentar de novo tira a falha: se agora sobe, fica montado.
	t.Setenv(envFalsoFalha, "")
	if err := os.Mkdir(filepath.Join(raiz, "V"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(g.DesmontarTodas)
	if ok, msg, _ := g.MontarUnidade("cofre", "V", "", nil); !ok {
		t.Fatal(msg)
	}
	if e := estadoDoRemoto(t, g, "cofre"); e.Estado != EstadoMontado || e.Motivo != "" {
		t.Errorf("estado = %+v, quer montado", e)
	}
}

func TestEstadoListarCofresUsaOCore(t *testing.T) {
	g, f := novoGerenciadorFalso(t)
	raiz := t.TempDir()
	g.Montagens.caminhoPonto = func(letra string) string { return filepath.Join(raiz, letra) }
	g.Montagens.pontoExiste = caminhoExiste
	g.Montagens.esperaEncerrar = 200 * time.Millisecond
	t.Cleanup(g.Montagens.DesmontarTodas)
	_ = f
	for _, n := range []string{"a", "b"} {
		if ok, msg := g.Cofres.Adicionar(n, "drive", "Google Drive", n+"_base:"); !ok {
			t.Fatal(msg)
		}
	}
	os.Mkdir(filepath.Join(raiz, "V"), 0o755)
	if ok, msg, _ := g.Montagens.MontarUnidade("a", "V", "", nil); !ok {
		t.Fatal(msg)
	}

	estados := map[string]CofreStatus{}
	for _, c := range g.ListarCofres() {
		estados[c.Nome] = c
	}

	if a := estados["a"]; a.Estado != EstadoMontado || a.Letra != "V" || !strings.HasSuffix(a.PontoMontagem, "V") {
		t.Errorf("a = %+v, quer montado em V", a)
	}
	if b := estados["b"]; b.Estado != EstadoDesmontado {
		t.Errorf("b = %+v, quer desmontado", b)
	}
}
