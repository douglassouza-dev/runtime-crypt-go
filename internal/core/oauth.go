package core

import (
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
)

// StatusOAuth representa o estado atual do fluxo OAuth.
type StatusOAuth struct {
	URL       string `json:"url"`
	Token     string `json:"token"`
	Concluido bool   `json:"concluido"`
}

// GerenciadorOAuth controla o fluxo de autorização OAuth via rclone authorize.
type GerenciadorOAuth struct {
	processo  *exec.Cmd
	token     string
	url       string
	mu        sync.Mutex
	finalizado chan struct{}
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
	finalizado := make(chan struct{})
	g.finalizado = finalizado
	g.mu.Unlock()

	cmd := exec.Command(executavel, "authorize", tipo)
	cmd.Stdout = nil
	cmd.Stderr = nil

	if runtime.GOOS == "windows" {
		ocultarJanela(cmd)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		g.mu.Lock()
		g.finalizado = nil
		g.mu.Unlock()
		return false
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		g.mu.Lock()
		g.finalizado = nil
		g.mu.Unlock()
		return false
	}

	g.mu.Lock()
	g.processo = cmd
	g.mu.Unlock()

	go func() {
		defer close(finalizado)

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

					if strings.Contains(linhaStrip, "127.0.0.1") || strings.Contains(linhaStrip, "localhost") {
						match := reURL.FindString(linhaStrip)
						if match != "" {
							g.mu.Lock()
							g.url = strings.TrimRight(match, ".")
							g.mu.Unlock()
						}
					}

					if strings.Contains(linhaStrip, "Paste the following") {
						capturandoToken = true
						bufferToken = nil
						continue
					}

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

// Abortar encerra o processo OAuth e aguarda a goroutine de leitura finalizar.
func (g *GerenciadorOAuth) Abortar() {
	g.mu.Lock()
	cmd := g.processo
	finalizado := g.finalizado
	g.processo = nil
	g.finalizado = nil
	g.mu.Unlock()

	if cmd != nil && cmd.Process != nil {
		cmd.Process.Kill()
	}

	if finalizado != nil {
		select {
		case <-finalizado:
		case <-time.After(5 * time.Second):
		}
	}
}

// ocultarJanela configura o comando para não criar janela visível (Windows).
func ocultarJanela(cmd *exec.Cmd) {
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
	if runtime.GOOS == "windows" {
		ocultarJanela(cmd)
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
