package core

import (
	"errors"
	"os"
	"path/filepath"
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
	g.pontoExiste = pontoPeloFalso(f)
	g.raizPontos = t.TempDir()
	// Sem fusermount/umount de verdade nos testes.
	g.desmontarPonto = func(string) error { return errors.New("sem FUSE no teste") }
	return g, f
}

// pontoDeTeste devolve como pedir um ponto de montagem, a chave que o mapa
// usa e o argumento do `rclone mount`: no Windows a letra "v:" (V, V:); fora
// dele uma pasta (demanda 013).
func pontoDeTeste(t *testing.T) (pedido, chave, arg string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		return "v:", "V", "V:"
	}
	p := filepath.Join(t.TempDir(), "cofre")
	return p, p, p
}

// pontoPeloFalso simula o WinFsp/FUSE: o ponto "X:\" existe enquanto houver
// um `rclone mount ... X:` do falso vivo, ou enquanto o mount ainda não
// chegou a se registrar (montando).
func pontoPeloFalso(f *rcloneFalso) func(string) bool {
	return func(caminho string) bool {
		alvo := strings.TrimSuffix(caminho, "\\")
		achou := false
		for _, c := range f.chamadas() {
			if len(c.Args) > 2 && c.Args[0] == "mount" && c.Args[2] == alvo {
				achou = true
				if processoVivoNoSO(c.Pid) {
					return true
				}
			}
		}
		return !achou
	}
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
	pedido, chave, arg := pontoDeTeste(t)

	ok, msg, letra := g.MontarUnidade("cofre", pedido, "senha", nil)

	if !ok || letra != chave || !strings.Contains(msg, chave) {
		t.Fatalf("ok=%v msg=%q letra=%q", ok, msg, letra)
	}
	cs := f.esperarChamadas(1, 5*time.Second)
	if len(cs) != 1 {
		t.Fatalf("chamadas = %+v", cs)
	}
	args := cs[0].Args
	for _, seq := range [][]string{
		{"mount", "cofre:", arg},
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
	g.DesmontarUnidade(chave)
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

// Sem letra pedida: no Windows vem uma letra livre do sistema; fora dele, a
// pasta <raizPontos>/<cofre>, criada antes de montar (demanda 013).
func TestMontarUnidadeSemLetraEscolheOPonto(t *testing.T) {
	g, f := novoMontadorFalso(t)

	ok, msg, letra := g.MontarUnidade("cofre", "", "", nil)

	if !ok {
		t.Fatalf("ok=%v msg=%q", ok, msg)
	}
	defer g.DesmontarUnidade(letra)
	cs := f.esperarChamadas(1, 5*time.Second)
	if len(cs) != 1 || !argsContem(cs[0].Args, "mount", "cofre:", argumentoMount(letra)) {
		t.Fatalf("chamadas = %+v", cs)
	}
	if runtime.GOOS == "windows" {
		if len(letra) != 1 || !strings.Contains(strings.Join(LetrasPreferidas, ""), letra) {
			t.Errorf("letra = %q, quer uma de %v", letra, LetrasPreferidas)
		}
		return
	}
	if letra != filepath.Join(g.raizPontos, "cofre") {
		t.Errorf("ponto = %q, quer %s", letra, filepath.Join(g.raizPontos, "cofre"))
	}
	if info, err := os.Stat(letra); err != nil || !info.IsDir() {
		t.Errorf("a pasta do ponto deveria existir: %v", err)
	}
}

func TestMontarUnidadeProcessoQueSaiFalhaRapido(t *testing.T) {
	g, f := novoMontadorFalso(t)
	g.pontoExiste = func(string) bool { return false }
	f.falhar()
	t.Setenv(envFalsoStderr, "CRITICAL: teste")
	registro := capturarLog(t)

	inicio := time.Now()
	ok, msg, _ := g.MontarUnidade("cofre", "V", "", nil)

	// Demanda 027: a mensagem não traz o texto do rclone; o log traz.
	if ok || time.Since(inicio) > 2*time.Second || msg != "Falha ao montar: o rclone falhou, detalhes no log" {
		t.Errorf("ok=%v em %v msg=%q", ok, time.Since(inicio), msg)
	}
	if !strings.Contains(registro.String(), "CRITICAL: teste") {
		t.Errorf("o log deveria trazer o texto do rclone: %q", registro.String())
	}
	semMontagemNemProcesso(t, g, f)
}

func TestDesmontarUnidade(t *testing.T) {
	g, _ := novoMontadorFalso(t)
	pedido, chave, _ := pontoDeTeste(t)
	if ok, msg, _ := g.MontarUnidade("cofre", pedido, "", nil); !ok {
		t.Fatal(msg)
	}
	pid := pidDaMontagem(t, g, chave)

	// Com barra no fim, como o usuário escreveria.
	ok, msg := g.DesmontarUnidade(pedido + string(filepath.Separator))

	if !ok || !strings.Contains(msg, chave) {
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
	if ok, msg := g.DesmontarUnidade("Q"); ok || !strings.Contains(msg, "Nenhuma montagem ativa em Q") {
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

	if len(st) != 1 || st[0].Letra != "V" || st[0].Remoto != "cofre:" || !st[0].Ativo || st[0].PontoMontagem != pontoDaLetra("V") {
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
