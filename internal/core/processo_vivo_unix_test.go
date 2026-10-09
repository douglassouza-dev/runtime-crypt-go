//go:build !windows

package core

import "syscall"

// processoVivoNoSO pergunta ao sistema se o pid existe. Um zumbi conta como
// vivo; os testes só usam isto depois que o core já chamou Wait.
func processoVivoNoSO(pid int) bool {
	if pid <= 0 {
		return false
	}
	return syscall.Kill(pid, 0) == nil
}
