//go:build !windows

package core

import "os/exec"

// configurarOcultarJanela é um no-op em plataformas não-Windows.
func configurarOcultarJanela(cmd *exec.Cmd) {
	// Nada a fazer em Linux/macOS
}
