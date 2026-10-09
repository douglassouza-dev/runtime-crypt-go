//go:build !windows

package core

import (
	"os"
	"testing"
)

// somenteLeitura tira a permissão de escrita da pasta até o fim do teste.
func somenteLeitura(t *testing.T, dir string) {
	t.Helper()
	if os.Geteuid() == 0 {
		t.Skip("como root a permissão da pasta não impede a escrita")
	}
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o755) })
}
