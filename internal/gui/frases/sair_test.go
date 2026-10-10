package frases

import (
	"errors"
	"strings"
	"testing"

	"github.com/eufrauzino/runtime-crypt-go/internal/core"
)

// Demanda 028: o aviso ao sair, no singular, no plural e com vários cofres.
func TestAvisoAoSair(t *testing.T) {
	casos := []struct {
		ps   []core.PendenciaCofre
		quer string
	}{
		{[]core.PendenciaCofre{{Nome: "fotos", N: 2}}, "2 arquivos de fotos ainda não subiram. Eles sobem quando você destrancar de novo."},
		{[]core.PendenciaCofre{{Nome: "fotos", N: 1}}, "1 arquivo de fotos ainda não subiu. Ele sobe quando você destrancar de novo."},
		{[]core.PendenciaCofre{{Nome: "docs", N: 1}, {Nome: "fotos", N: 3}}, "1 arquivo de docs ainda não subiu.\n3 arquivos de fotos ainda não subiram.\nEles sobem quando você destrancar de novo."},
	}
	for _, c := range casos {
		got := AvisoAoSair(c.ps)
		if got != c.quer {
			t.Errorf("AvisoAoSair(%+v) =\n%q\nquer\n%q", c.ps, got, c.quer)
		}
		if strings.Contains(strings.ToLower(got), "perd") {
			t.Errorf("o aviso não pode falar em perda: %q", got)
		}
	}
}

func TestFalhaAoEnviar(t *testing.T) {
	d := core.FalhaEnvio{Nome: "fotos", Etapa: core.EtapaDestrancar, Err: errors.New("sem winfsp")}
	tr := core.FalhaEnvio{Nome: "docs", Etapa: core.EtapaTrancar, Err: &core.ErroNaoSubiram{N: 1}}
	if got := FalhaAoEnviar(d); got != "fotos: Não destrancou: sem winfsp" {
		t.Errorf("%q", got)
	}
	if got := FalhaAoEnviar(tr); got != "docs: Não trancou: 1 arquivo ainda não subiu" {
		t.Errorf("%q", got)
	}
}
