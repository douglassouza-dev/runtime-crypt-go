//go:build darwin

package plataforma

import (
	"fmt"
	"os"
	"path/filepath"
)

// VerificarAutoIniciar verifica se o LaunchAgent plist existe.
func VerificarAutoIniciar() bool {
	caminho := caminhoLaunchAgent()
	_, err := os.Stat(caminho)
	return err == nil
}

// AdicionarAutoIniciar cria um LaunchAgent plist para auto-iniciar.
func AdicionarAutoIniciar() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}

	conteudo := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>com.runtime-crypto.app</string>
	<key>ProgramArguments</key>
	<array>
		<string>%s</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
</dict>
</plist>
`, exe)

	caminho := caminhoLaunchAgent()
	dir := filepath.Dir(caminho)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(caminho, []byte(conteudo), 0644)
}

// RemoverAutoIniciar remove o LaunchAgent plist.
func RemoverAutoIniciar() error {
	caminho := caminhoLaunchAgent()
	err := os.Remove(caminho)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// InfoWinfsp contém informações sobre o macFUSE (equivalente macOS do WinFsp).
type InfoWinfsp struct {
	Instalado   bool   `json:"instalado"`
	Motivo      string `json:"motivo,omitempty"`
	UrlDownload string `json:"url_download,omitempty"`
}

// VerificarWinfsp verifica se o macFUSE está disponível.
func VerificarWinfsp() InfoWinfsp {
	// macFUSE instala em /Library/Filesystems/macfuse.fs
	if _, err := os.Stat("/Library/Filesystems/macfuse.fs"); err == nil {
		return InfoWinfsp{Instalado: true}
	}
	// FUSE-T (alternativa)
	if _, err := os.Stat("/Library/Filesystems/fuse-t.fs"); err == nil {
		return InfoWinfsp{Instalado: true}
	}
	return InfoWinfsp{
		Instalado:   false,
		Motivo:      "macFUSE nao encontrado. Necessario para montar unidades virtuais.",
		UrlDownload: "https://osxfuse.github.io/",
	}
}

func caminhoLaunchAgent() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", "com.runtime-crypto.app.plist")
}
