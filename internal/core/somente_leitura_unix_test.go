//go:build !windows

package core

import (
	"os"
	"testing"
)

// No Unix a permissão da pasta impede criar o temporário de verdade. No
// Windows o atributo de pasta não impede; lá o mesmo contrato é conferido por
// TestAtualizarSemConseguirGravarDevolveErro e
// TestAtualizarComPastaInvalidaDevolveErro.
func TestAtualizarEmDiretorioSomenteLeituraDevolveErro(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("como root a permissão da pasta não impede a escrita")
	}
	g, dir, antes := novoCofresComUm(t)
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o755) })

	conferirAtualizarFalha(t, g, dir, antes)
}
