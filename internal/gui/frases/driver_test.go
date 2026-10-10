package frases

import (
	"runtime"
	"testing"

	"github.com/eufrauzino/runtime-crypt-go/internal/core"
)

// Demanda 027: falta o driver de montagem.

func semDriver() core.CofreStatus {
	return core.CofreStatus{
		Cofre:  core.Cofre{Nome: "a"},
		Estado: core.EstadoFalhou,
		Motivo: "falta instalar o FUSE",
		Rclone: &core.ErroRclone{Falha: core.FalhaDriver},
	}
}

func TestTextoDriverAusentePorSistema(t *testing.T) {
	casos := map[string]string{
		"windows": "Não montou: falta instalar o WinFsp.",
		"linux":   "Não montou: falta instalar o FUSE.",
		"darwin":  "Não montou: falta instalar o macFUSE.",
	}
	for goos, quer := range casos {
		if got := TextoDriverAusente(goos); got != quer {
			t.Errorf("%s: %q, quer %q", goos, got, quer)
		}
	}
}

func TestBotaoDriverPorSistema(t *testing.T) {
	casos := []struct{ goos, rotulo, url string }{
		{"windows", "Baixar WinFsp", "https://winfsp.dev/rel/"},
		{"darwin", "Baixar macFUSE", "https://macfuse.github.io/"},
		{"linux", "", ""}, // no Linux, só o texto
	}
	for _, c := range casos {
		if r, u := botaoDriver(c.goos); r != c.rotulo || u != c.url {
			t.Errorf("%s: (%q, %q), quer (%q, %q)", c.goos, r, u, c.rotulo, c.url)
		}
	}
}

func TestDoCofreSemDriver(t *testing.T) {
	c := semDriver()
	if got := DoCofre(c); got != TextoDriverAusente(runtime.GOOS) {
		t.Errorf("DoCofre = %q", got)
	}
	if Botao(c) != TextoTentarDeNovo {
		t.Errorf("Botao = %q", Botao(c))
	}
	r, u := BotaoBaixarDriver(c)
	if rr, uu := botaoDriver(runtime.GOOS); r != rr || u != uu {
		t.Errorf("BotaoBaixarDriver = (%q, %q)", r, u)
	}
}

// Só a falha de driver de um cofre que não subiu tem a frase e o botão.
func TestDriverAusenteSoNaFalhaDeDriver(t *testing.T) {
	outra := semDriver()
	outra.Rclone = &core.ErroRclone{Falha: core.FalhaConexao}
	caiu := semDriver()
	caiu.Caiu = true
	semRclone := semDriver()
	semRclone.Rclone = nil
	for nome, c := range map[string]core.CofreStatus{"outra falha": outra, "caiu": caiu, "sem rclone": semRclone} {
		if DriverAusente(c) {
			t.Errorf("%s: não deveria ser driver ausente", nome)
		}
		if r, u := BotaoBaixarDriver(c); r != "" || u != "" {
			t.Errorf("%s: botão (%q, %q)", nome, r, u)
		}
	}
}
