package core

import (
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"sync"
)

// StatusOAuth representa o estado atual do fluxo OAuth.
type StatusOAuth struct {
	URL       string `json:"url"`
	Token     string `json:"token"`
	Concluido bool   `json:"concluido"`
}

// GerenciadorOAuth controla o fluxo de autorização OAuth via rclone authorize.
type GerenciadorOAuth struct {
	processo *exec.Cmd
	token    string
	url      string
	mu       sync.Mutex
}

// NovoGerenciadorOAuth cria uma nova instância.
func NovoGerenciadorOAuth() *GerenciadorOAuth {
	return &GerenciadorOAuth{}
}

// Iniciar lança o processo rclone authorize e começa a parsear a saída.
func (g *GerenciadorOAuth) Iniciar(executavel string, tipo string) bool {
	g.Abortar()

	g.mu.Lock()
	g.token = ""
	g.url = ""
	g.mu.Unlock()

	cmd := exec.Command(executavel, "authorize", tipo)
	cmd.Stdout = nil
	cmd.Stderr = nil

	if runtime.GOOS == "windows" {
		ocultarJanela(cmd)
	}

	// Capturar stdout combinado
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return false
	}
	cmd.Stderr = cmd.Stdout // Combinar stderr com stdout

	if err := cmd.Start(); err != nil {
		return false
	}

	g.mu.Lock()
	g.processo = cmd
	g.mu.Unlock()

	// Goroutine para ler a saída
	go func() {
		buf := make([]byte, 4096)
		var bufferToken []string
		capturandoToken := false

		reURL := regexp.MustCompile(`https?://\S+`)
		reJSON := regexp.MustCompile(`(\{.*\})`)

		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				texto := string(buf[:n])
				for _, linha := range strings.Split(texto, "\n") {
					linhaStrip := strings.TrimSpace(linha)

					// Detectar URL de auth
					if strings.Contains(linhaStrip, "127.0.0.1") || strings.Contains(linhaStrip, "localhost") {
						match := reURL.FindString(linhaStrip)
						if match != "" {
							g.mu.Lock()
							g.url = strings.TrimRight(match, ".")
							g.mu.Unlock()
						}
					}

					// Detectar início do token
					if strings.Contains(linhaStrip, "Paste the following") {
						capturandoToken = true
						bufferToken = nil
						continue
					}

					// Capturar token JSON
					if capturandoToken {
						if strings.Contains(linhaStrip, "End paste") {
							tokenStr := strings.Join(bufferToken, "")
							match := reJSON.FindString(tokenStr)
							if match != "" {
								g.mu.Lock()
								g.token = match
								g.mu.Unlock()
							}
							break
						}
						if strings.HasPrefix(linhaStrip, "{") || len(bufferToken) > 0 {
							bufferToken = append(bufferToken, linhaStrip)
						}
					}
				}
			}
			if err != nil {
				break
			}
		}

		cmd.Wait()
	}()

	return true
}

// ObterStatus retorna o estado atual do fluxo OAuth.
func (g *GerenciadorOAuth) ObterStatus() StatusOAuth {
	g.mu.Lock()
	defer g.mu.Unlock()
	return StatusOAuth{
		URL:       g.url,
		Token:     g.token,
		Concluido: g.token != "",
	}
}

// Abortar encerra o processo OAuth se estiver em execução.
func (g *GerenciadorOAuth) Abortar() {
	g.mu.Lock()
	cmd := g.processo
	g.processo = nil
	g.mu.Unlock()

	if cmd != nil && cmd.Process != nil {
		cmd.Process.Kill()
		cmd.Wait()
	}
}

// ocultarJanela configura o comando para não criar janela visível (Windows).
func ocultarJanela(cmd *exec.Cmd) {
	// Implementação em plataforma_windows.go via SysProcAttr
	configurarOcultarJanela(cmd)
}

// AbrirNavegador abre uma URL no navegador padrão do sistema.
func AbrirNavegador(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

// AbrirExplorador abre o explorador de arquivos no caminho especificado.
func AbrirExplorador(caminho string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", caminho)
	case "darwin":
		cmd = exec.Command("open", caminho)
	default:
		cmd = exec.Command("xdg-open", caminho)
	}
	if runtime.GOOS == "windows" {
		ocultarJanela(cmd)
	}
	return cmd.Start()
}
