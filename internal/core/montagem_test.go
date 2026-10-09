package core

import (
	"runtime"
	"strings"
	"testing"
	"time"
)

// Testes de caracterização de montagem.go (demanda 016).

func novoMontadorFalso(t *testing.T) (*GerenciadorMontagem, *rcloneFalso) {
	t.Helper()
	f := novoRcloneFalso(t)
	g := NovoGerenciadorMontagem(f.exe, NovoConfigVfs())
	g.pontoExiste = func(string) bool { return true }
	return g, f
}

func pidDaMontagem(t *testing.T, g *GerenciadorMontagem, letra string) int {
	t.Helper()
	g.mu.Lock()
	defer g.mu.Unlock()
	info, ok := g.montagens[letra]
	if !ok || info.Processo == nil {
		t.Fatalf("montagem %s não está no mapa", letra)
	}
	return info.Processo.Pid
}

func TestNovoGerenciadorMontagemComecaVazio(t *testing.T) {
	g := NovoGerenciadorMontagem("rclone", NovoConfigVfs())
	if len(g.Status()) != 0 || len(g.ObterMontagens()) != 0 {
		t.Error("gerenciador novo deveria começar sem montagens")
	}
	if g.pontoExiste == nil {
		t.Error("pontoExiste deveria ter o padrão caminhoExiste")
	}
}

func TestMontarUnidadeSucesso(t *testing.T) {
	g, f := novoMontadorFalso(t)

	ok, msg, letra := g.MontarUnidade("cofre", "v:", "senha", nil)

	if !ok || letra != "V" || !strings.Contains(msg, "V:") {
		t.Fatalf("ok=%v msg=%q letra=%q", ok, msg, letra)
	}
	cs := f.esperarChamadas(1, 5*time.Second)
	if len(cs) != 1 {
		t.Fatalf("chamadas = %+v", cs)
	}
	args := cs[0].Args
	for _, seq := range [][]string{
		{"mount", "cofre:", "V:"},
		{"--volname", "RuntimeCrypto (cofre)"},
		{"--no-checksum"},
		{"--no-modtime"},
		{"--vfs-cache-mode", "full"},
	} {
		if !argsContem(args, seq...) {
			t.Errorf("faltou %v em %v", seq, args)
		}
	}
	if runtime.GOOS == "windows" && !argsContem(args, "--network-mode", "--no-console") {
		t.Errorf("no Windows deveria ter --network-mode --no-console: %v", args)
	}
	if !cs[0].TemSenhaEnv {
		t.Error("hoje a senha vai em RCLONE_CONFIG_PASS no ambiente do mount")
	}
	g.DesmontarUnidade("V")
}

func TestMontarUnidadeRecusa(t *testing.T) {
	semRclone := NovoGerenciadorMontagem("", NovoConfigVfs())
	if ok, msg, _ := semRclone.MontarUnidade("cofre", "V", "", nil); ok || !strings.Contains(msg, "nao disponivel") {
		t.Errorf("sem rclone: ok=%v msg=%q", ok, msg)
	}

	g, _ := novoMontadorFalso(t)
	if ok, msg, _ := g.MontarUnidade("cofre", "V", "", nil); !ok {
		t.Fatal(msg)
	}
	defer g.DesmontarUnidade("V")
	if ok, msg, _ := g.MontarUnidade("outro", "V", "", nil); ok || !strings.Contains(msg, "ja esta em uso") {
		t.Errorf("letra repetida: ok=%v msg=%q", ok, msg)
	}
}

func TestMontarUnidadeSemLetraForaDoWindows(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no Windows a letra livre vem do sistema")
	}
	g, f := novoMontadorFalso(t)

	ok, msg, _ := g.MontarUnidade("cofre", "", "", nil)

	if ok || !strings.Contains(msg, "Nenhuma letra") {
		t.Errorf("ok=%v msg=%q", ok, msg)
	}
	if n := len(f.chamadas()); n != 0 {
		t.Errorf("não deveria iniciar o rclone; houve %d chamadas", n)
	}
}

func TestMontarUnidadeProcessoQueSaiFalhaRapido(t *testing.T) {
	g, f := novoMontadorFalso(t)
	g.pontoExiste = func(string) bool { return false }
	f.falhar()
	t.Setenv(envFalsoStderr, "CRITICAL: teste")

	inicio := time.Now()
	ok, msg, _ := g.MontarUnidade("cofre", "V", "", nil)

	if ok || time.Since(inicio) > 2*time.Second || !strings.Contains(msg, "CRITICAL: teste") {
		t.Errorf("ok=%v em %v msg=%q", ok, time.Since(inicio), msg)
	}
	if !strings.Contains(msg, "codigo 1") {
		t.Errorf("a mensagem deveria trazer o código de saída: %q", msg)
	}
	semMontagemNemProcesso(t, g, f)
}

func TestDesmontarUnidade(t *testing.T) {
	g, _ := novoMontadorFalso(t)
	if ok, msg, _ := g.MontarUnidade("cofre", "V", "", nil); !ok {
		t.Fatal(msg)
	}
	pid := pidDaMontagem(t, g, "V")

	ok, msg := g.DesmontarUnidade("v:\\")

	if !ok || !strings.Contains(msg, "V:") {
		t.Errorf("ok=%v msg=%q", ok, msg)
	}
	if processoVivoNoSO(pid) {
		t.Errorf("processo %d continua vivo", pid)
	}
	g.mu.Lock()
	n := len(g.montagens)
	g.mu.Unlock()
	if n != 0 {
		t.Errorf("mapa deveria ficar vazio, tem %d", n)
	}
}

func TestDesmontarUnidadeInexistente(t *testing.T) {
	g, _ := novoMontadorFalso(t)
	if ok, msg := g.DesmontarUnidade("Q"); ok || !strings.Contains(msg, "Q:") {
		t.Errorf("ok=%v msg=%q", ok, msg)
	}
}

func TestDesmontarTodas(t *testing.T) {
	g, _ := novoMontadorFalso(t)
	var pids []int
	for _, l := range []string{"V", "W"} {
		if ok, msg, _ := g.MontarUnidade("c"+l, l, "", nil); !ok {
			t.Fatal(msg)
		}
		pids = append(pids, pidDaMontagem(t, g, l))
	}

	g.DesmontarTodas()

	for _, pid := range pids {
		if processoVivoNoSO(pid) {
			t.Errorf("processo %d continua vivo", pid)
		}
	}
}

func TestStatusMantemMontagemViva(t *testing.T) {
	g, _ := novoMontadorFalso(t)
	if ok, msg, _ := g.MontarUnidade("cofre", "V", "", nil); !ok {
		t.Fatal(msg)
	}
	defer g.DesmontarUnidade("V")

	st := g.Status()

	if len(st) != 1 || st[0].Letra != "V" || st[0].Remoto != "cofre:" || !st[0].Ativo || st[0].PontoMontagem != "V:\\" {
		t.Errorf("Status = %+v", st)
	}
}

func TestObterMontagensEObterLetraPorRemoto(t *testing.T) {
	g, _ := novoMontadorFalso(t)
	if ok, msg, _ := g.MontarUnidade("cofre", "V", "", nil); !ok {
		t.Fatal(msg)
	}
	defer g.DesmontarUnidade("V")

	if m, ok := g.ObterMontagens()["cofre"]; !ok || m.Letra != "V" {
		t.Errorf("ObterMontagens = %+v", g.ObterMontagens())
	}
	if l := g.ObterLetraPorRemoto("cofre:"); l != "V" {
		t.Errorf("ObterLetraPorRemoto = %q", l)
	}
}

func TestObterMontagensEObterLetraPorRemotoSemMontagem(t *testing.T) {
	g, _ := novoMontadorFalso(t)
	if len(g.ObterMontagens()) != 0 {
		t.Error("ObterMontagens deveria vir vazio")
	}
	if l := g.ObterLetraPorRemoto("cofre"); l != "" {
		t.Errorf("ObterLetraPorRemoto = %q", l)
	}
}

func TestObterLetrasDisponiveis(t *testing.T) {
	got := ObterLetrasDisponiveis([]string{"v", "W"})
	if runtime.GOOS != "windows" {
		if got != nil {
			t.Errorf("fora do Windows deveria ser nil, veio %v", got)
		}
		return
	}
	for _, l := range got {
		if l == "V" || l == "W" {
			t.Errorf("letra ocupada %s voltou como livre: %v", l, got)
		}
	}
}
