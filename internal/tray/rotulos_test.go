package tray

import (
	"strings"
	"testing"
)

// Demanda 031: rótulos do menu com acento e "…" de um caractere.
func TestRotulosDoMenu(t *testing.T) {
	casos := map[string]string{
		RotuloNovoCofre:     "Novo Cofre…",
		RotuloConfiguracoes: "Configurações",
		RotuloConfigVfs:     "Configurações VFS…",
		RotuloAutoIniciar:   "Abrir ao ligar o computador",
	}
	for got, quer := range casos {
		if got != quer {
			t.Errorf("%q, quer %q", got, quer)
		}
		if strings.Contains(got, "...") {
			t.Errorf("%q usa três pontos", got)
		}
	}
}
