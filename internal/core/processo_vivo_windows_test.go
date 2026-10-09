//go:build windows

package core

import "golang.org/x/sys/windows"

const stillActive = 259

// processoVivoNoSO pergunta ao Windows se o processo ainda não terminou.
func processoVivoNoSO(pid int) bool {
	if pid <= 0 {
		return false
	}
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer windows.CloseHandle(h)
	var codigo uint32
	if err := windows.GetExitCodeProcess(h, &codigo); err != nil {
		return false
	}
	return codigo == stillActive
}
