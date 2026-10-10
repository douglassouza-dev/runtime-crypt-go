//go:build linux

package plataforma

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// Demanda 027: o FUSE só conta como instalado com /dev/fuse e fusermount3
// (ou fusermount) no PATH.
func TestFuseDisponivel(t *testing.T) {
	dir := t.TempDir()
	dispositivo := filepath.Join(dir, "fuse")
	if err := os.WriteFile(dispositivo, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	so := func(nomes ...string) func(string) (string, error) {
		return func(n string) (string, error) {
			for _, x := range nomes {
				if x == n {
					return "/usr/bin/" + n, nil
				}
			}
			return "", errors.New("não achou")
		}
	}
	casos := []struct {
		nome        string
		dispositivo string
		procurar    func(string) (string, error)
		quer        bool
	}{
		{"tudo", dispositivo, so("fusermount3"), true},
		{"fusermount antigo", dispositivo, so("fusermount"), true},
		{"sem fusermount", dispositivo, so(), false},
		{"sem /dev/fuse", filepath.Join(dir, "nao-existe"), so("fusermount3"), false},
	}
	for _, c := range casos {
		if got := fuseDisponivel(c.dispositivo, c.procurar); got != c.quer {
			t.Errorf("%s: %v, quer %v", c.nome, got, c.quer)
		}
	}
}
