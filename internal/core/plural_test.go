package core

import "testing"

func TestArquivosSingularEPlural(t *testing.T) {
	for n, quer := range map[int]string{0: "0 arquivos", 1: "1 arquivo", 2: "2 arquivos", 10: "10 arquivos"} {
		if got := Arquivos(n); got != quer {
			t.Errorf("Arquivos(%d) = %q, quer %q", n, got, quer)
		}
	}
}
