//go:build windows

package plataforma

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

const chaveAutoIniciar = `Software\Microsoft\Windows\CurrentVersion\Run`
const nomeRegistro = "RuntimeCrypto"

// VerificarAutoIniciar verifica se o programa está configurado para auto-iniciar com Windows.
func VerificarAutoIniciar() bool {
	chave, err := registry.OpenKey(
		registry.CURRENT_USER,
		chaveAutoIniciar,
		registry.QUERY_VALUE,
	)
	if err != nil {
		return false
	}
	defer chave.Close()

	_, _, err = chave.GetStringValue(nomeRegistro)
	return err == nil
}

// AdicionarAutoIniciar adiciona o programa ao auto-início do Windows.
func AdicionarAutoIniciar() error {
	chave, err := registry.OpenKey(
		registry.CURRENT_USER,
		chaveAutoIniciar,
		registry.SET_VALUE,
	)
	if err != nil {
		return fmt.Errorf("erro ao abrir registro: %w", err)
	}
	defer chave.Close()

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("erro ao obter caminho do executável: %w", err)
	}

	valor := fmt.Sprintf(`"%s"`, filepath.Clean(exe))
	return chave.SetStringValue(nomeRegistro, valor)
}

// RemoverAutoIniciar remove o programa do auto-início do Windows. Valor que
// já não existe não é erro; qualquer outra falha volta para quem chamou
// (demanda 009).
func RemoverAutoIniciar() error {
	chave, err := registry.OpenKey(
		registry.CURRENT_USER,
		chaveAutoIniciar,
		registry.SET_VALUE,
	)
	if errors.Is(err, registry.ErrNotExist) {
		return nil // sem a chave Run, não há o que remover
	}
	if err != nil {
		return fmt.Errorf("erro ao abrir registro: %w", err)
	}
	defer chave.Close()

	err = chave.DeleteValue(nomeRegistro)
	if errors.Is(err, registry.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("erro ao remover do registro: %w", err)
	}
	return nil
}

// InfoWinfsp contém informações sobre a instalação do WinFsp.
type InfoWinfsp struct {
	Instalado   bool   `json:"instalado"`
	Motivo      string `json:"motivo,omitempty"`
	UrlDownload string `json:"url_download,omitempty"`
}

// VerificarWinfsp verifica se o WinFsp está instalado no Windows.
func VerificarWinfsp() InfoWinfsp {
	// Método 1: DLL no System32
	systemRoot := os.Getenv("SystemRoot")
	if systemRoot == "" {
		systemRoot = `C:\Windows`
	}
	dllCaminho := filepath.Join(systemRoot, "System32", "winfsp-x64.dll")
	if _, err := os.Stat(dllCaminho); err == nil {
		return InfoWinfsp{Instalado: true}
	}

	// Método 2: Registro do Windows
	chave, err := registry.OpenKey(
		registry.LOCAL_MACHINE,
		`SOFTWARE\WOW6432Node\WinFsp`,
		registry.QUERY_VALUE,
	)
	if err == nil {
		defer chave.Close()
		caminho, _, err := chave.GetStringValue("InstallDir")
		if err == nil && caminho != "" {
			if _, err := os.Stat(caminho); err == nil {
				return InfoWinfsp{Instalado: true}
			}
		}
	}

	// Método 3: Program Files
	for _, base := range []string{os.Getenv("ProgramFiles"), os.Getenv("ProgramFiles(x86)")} {
		if base != "" {
			winfspDir := filepath.Join(base, "WinFsp")
			if info, err := os.Stat(winfspDir); err == nil && info.IsDir() {
				return InfoWinfsp{Instalado: true}
			}
		}
	}

	return InfoWinfsp{
		Instalado:   false,
		UrlDownload: "https://winfsp.dev/rel/",
		Motivo:      "WinFsp não encontrado. Necessário para montar unidades virtuais.",
	}
}
