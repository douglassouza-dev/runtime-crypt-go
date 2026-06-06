//go:build windows

package core

import (
	"os/exec"
	"syscall"
)

// configurarOcultarJanela define CREATE_NO_WINDOW para processos no Windows.
func configurarOcultarJanela(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
}
