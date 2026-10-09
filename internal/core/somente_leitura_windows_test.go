//go:build windows

package core

import (
	"os/exec"
	"os/user"
	"testing"
)

// somenteLeitura nega ao usuário atual criar arquivos e gravar na pasta (no
// Windows o atributo "somente leitura" de pasta não impede escrita).
func somenteLeitura(t *testing.T, dir string) {
	t.Helper()
	u, err := user.Current()
	if err != nil {
		t.Fatal(err)
	}
	regra := u.Username + ":(OI)(CI)(W,D)"
	if saida, err := exec.Command("icacls", dir, "/deny", regra).CombinedOutput(); err != nil {
		t.Fatalf("icacls /deny: %v: %s", err, saida)
	}
	t.Cleanup(func() {
		exec.Command("icacls", dir, "/remove:d", u.Username).Run()
	})
}
