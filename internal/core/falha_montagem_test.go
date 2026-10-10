package core

import (
	"strings"
	"testing"
	"time"
)

// Testes da demanda 003.

// envFalsoStderr troca o que o rclone falso escreve no stderr quando
// RCLONE_FALSO_FALHA=1 (padrão: "CRITICAL: falha do rclone falso").
const envFalsoStderr = "RCLONE_FALSO_STDERR"

// semMontagemNemProcesso confere que, depois de MontarUnidade devolver false,
// não sobra processo do rclone (nem zumbi no Unix, ou seja, houve Wait) nem
// entrada no mapa.
func semMontagemNemProcesso(t *testing.T, g *GerenciadorMontagem, f *rcloneFalso) {
	t.Helper()
	cs := f.chamadas()
	if len(cs) == 0 {
		t.Fatal("o rclone falso não foi chamado")
	}
	for _, c := range cs {
		if processoVivoNoSO(c.Pid) {
			t.Errorf("processo %d (%v) ficou vivo depois de MontarUnidade devolver false", c.Pid, c.Args)
		}
	}
	g.mu.Lock()
	n := len(g.montagens)
	g.mu.Unlock()
	if n != 0 {
		t.Errorf("o mapa deveria ficar vazio; tem %d", n)
	}
}

func TestMontarUnidadeTempoEsgotadoNaoDeixaProcesso(t *testing.T) {
	g, f := novoMontadorFalso(t)
	g.pontoExiste = func(string) bool { return false } // a letra nunca aparece
	antes := limiteMontagem
	limiteMontagem = 500 * time.Millisecond
	t.Cleanup(func() { limiteMontagem = antes })

	ok, msg, _ := g.MontarUnidade("cofre", "V", "", nil)

	if ok || msg != "Não montou: a unidade não ficou pronta em 0.5 s." {
		t.Errorf("ok=%v msg=%q", ok, msg)
	}
	semMontagemNemProcesso(t, g, f)
}

func TestMontarUnidadeSaidaSemStderr(t *testing.T) {
	g, f := novoMontadorFalso(t)
	g.pontoExiste = func(string) bool { return false }
	f.falhar()
	t.Setenv(envFalsoStderr, " ")

	ok, msg, _ := g.MontarUnidade("cofre", "V", "", nil)

	if ok || !strings.Contains(msg, "sem mensagem") {
		t.Errorf("ok=%v msg=%q", ok, msg)
	}
	semMontagemNemProcesso(t, g, f)
}

func TestUltimasLinhasGuardaSoAsFinais(t *testing.T) {
	u := novasUltimasLinhas(3)
	u.Write([]byte("um\ndois\r\n\ntr"))
	u.Write([]byte("es\nquatro\ncin"))
	u.Write([]byte("co"))

	if got := u.Texto(); got != "tres\nquatro\ncinco" {
		t.Errorf("Texto = %q", got)
	}
}

func TestMotivoUnidadeNaoFicouPronta(t *testing.T) {
	if got := MotivoUnidadeNaoFicouPronta(LimiteMontagemPadrao); got != "a unidade não ficou pronta em 45 s" {
		t.Errorf("%q", got)
	}
}
