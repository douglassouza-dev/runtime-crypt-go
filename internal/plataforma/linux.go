//go:build linux

package plataforma

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// VerificarAutoIniciar verifica se o .desktop de autostart existe.
func VerificarAutoIniciar() bool {
	caminho := caminhoAutostart()
	_, err := os.Stat(caminho)
	return err == nil
}

// AdicionarAutoIniciar cria um arquivo .desktop para auto-iniciar.
func AdicionarAutoIniciar() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}

	conteudo := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=RuntimeCrypto
Exec=%s
Hidden=false
NoDisplay=false
X-GNOME-Autostart-enabled=true
Comment=Cofre Criptografado na Nuvem
`, exe)

	caminho := caminhoAutostart()
	dir := filepath.Dir(caminho)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(caminho, []byte(conteudo), 0644)
}

// RemoverAutoIniciar remove o arquivo .desktop de autostart.
func RemoverAutoIniciar() error {
	caminho := caminhoAutostart()
	err := os.Remove(caminho)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// InfoWinfsp contém informações sobre o FUSE (equivalente Linux do WinFsp).
type InfoWinfsp struct {
	Instalado   bool   `json:"instalado"`
	Motivo      string `json:"motivo,omitempty"`
	UrlDownload string `json:"url_download,omitempty"`
}

// VerificarWinfsp verifica se o FUSE está disponível no Linux: o rclone mount
// precisa de /dev/fuse e do fusermount3 (ou fusermount) no PATH.
func VerificarWinfsp() InfoWinfsp {
	if fuseDisponivel("/dev/fuse", exec.LookPath) {
		return InfoWinfsp{Instalado: true}
	}
	return InfoWinfsp{
		Instalado:   false,
		Motivo:      "FUSE não encontrado. Instale o pacote fuse3 pelo gerenciador do seu sistema.",
		UrlDownload: "https://github.com/libfuse/libfuse",
	}
}

func caminhoAutostart() string {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		home, _ := os.UserHomeDir()
		configDir = filepath.Join(home, ".config")
	}
	return filepath.Join(configDir, "autostart", "runtime-crypto.desktop")
}

// fuseDisponivel é o teste de VerificarWinfsp com o dispositivo e a busca no
// PATH trocáveis (testes).
func fuseDisponivel(dispositivo string, procurar func(string) (string, error)) bool {
	if _, err := os.Stat(dispositivo); err != nil {
		return false
	}
	for _, nome := range []string{"fusermount3", "fusermount"} {
		if _, err := procurar(nome); err == nil {
			return true
		}
	}
	return false
}
