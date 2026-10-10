package core

import (
	"runtime"
	"testing"
)

// Testes da demanda 027: falta o driver de montagem.

func TestDriverDeMontagem(t *testing.T) {
	casos := []struct{ goos, nome, url string }{
		{"windows", "WinFsp", "https://winfsp.dev/rel/"},
		{"darwin", "macFUSE", "https://macfuse.github.io/"},
		{"linux", "FUSE", ""},
	}
	for _, c := range casos {
		if nome, url := DriverDeMontagem(c.goos); nome != c.nome || url != c.url {
			t.Errorf("%s: (%q, %q), quer (%q, %q)", c.goos, nome, url, c.nome, c.url)
		}
	}
}

// Sem o driver, a montagem é recusada antes de iniciar o rclone, e o estado
// guarda a falha classificada.
func TestMontarSemDriverNaoIniciaORclone(t *testing.T) {
	g, f := novoMontadorFalso(t)
	g.DriverInstalado = func() bool { return false }
	registro := capturarLog(t)

	ok, msg, _ := g.MontarUnidade("cofre", "V", "", nil)

	if ok {
		t.Fatal("deveria recusar")
	}
	if cs := f.chamadas(); len(cs) != 0 {
		t.Errorf("o rclone não deveria ser chamado; chamadas: %v", cs)
	}
	nome, _ := DriverDeMontagem(runtime.GOOS)
	quer := "falta instalar o " + nome
	if msg != "Falha ao montar: "+quer {
		t.Errorf("msg = %q", msg)
	}
	e := estadoDoRemoto(t, g, "cofre")
	if e.Estado != EstadoFalhou || e.Caiu || e.Rclone == nil || e.Rclone.Falha != FalhaDriver || e.Motivo != quer {
		t.Errorf("estado = %+v, quer falhou com FalhaDriver", e)
	}
	if registro.String() == "" {
		t.Error("a recusa deveria ir para o log")
	}
}

// Com o driver presente, a verificação não atrapalha.
func TestMontarComDriverSegueNormal(t *testing.T) {
	g, f := novoMontadorFalso(t)
	chamou := false
	g.DriverInstalado = func() bool { chamou = true; return true }
	t.Cleanup(g.DesmontarTodas)

	if ok, msg, _ := g.MontarUnidade("cofre", "V", "", nil); !ok {
		t.Fatal(msg)
	}
	if !chamou || len(f.chamadas()) == 0 {
		t.Errorf("chamou a verificação=%v, chamadas do rclone=%d", chamou, len(f.chamadas()))
	}
}

// A verificação pode errar para "instalado"; aí o rclone falha e o texto
// dele (amostra real do Linux) leva à mesma falha.
func TestMontarDriverFaltandoNoStderrDoRclone(t *testing.T) {
	g, f := novoMontadorFalso(t)
	g.pontoExiste = func(string) bool { return false }
	f.falhar()
	t.Setenv(envFalsoStderr, `2026/10/10 10:30:14 CRITICAL: Fatal error: failed to mount FUSE fs: fusermount: exec: "fusermount3": executable file not found in $PATH`)

	if ok, _, _ := g.MontarUnidade("cofre", "V", "", nil); ok {
		t.Fatal("deveria falhar")
	}
	e := estadoDoRemoto(t, g, "cofre")
	if e.Rclone == nil || e.Rclone.Falha != FalhaDriver {
		t.Errorf("estado = %+v, quer FalhaDriver", e)
	}
	semMontagemNemProcesso(t, g, f)
}
