package core

import (
	"context"
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
	// Erro vem preenchido quando o rclone authorize foi encerrado por tempo
	// esgotado (demanda 008).
	Erro string `json:"erro,omitempty"`
}

// GerenciadorOAuth controla o fluxo de autorização OAuth via rclone authorize.
type GerenciadorOAuth struct {
	processo *exec.Cmd
	// fim é fechado pela goroutine de leitura depois da única espera (Wait) do
	// processo atual (demanda 020). Abortar espera este canal.
	fim   chan struct{}
	token string
	url   string
	erro  string
	mu    sync.Mutex
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
	g.erro = ""
	g.mu.Unlock()

	// O contexto mata o authorize se o usuário nunca terminar o login.
	limite := limiteAuthorize
	ctx, cancelar := context.WithTimeout(context.Background(), limite)
	cmd := exec.CommandContext(ctx, executavel, "authorize", tipo)
	cmd.WaitDelay = esperaPipes
	cmd.Stdout = nil
	cmd.Stderr = nil

	if runtime.GOOS == "windows" {
		ocultarJanela(cmd)
	}

	// Capturar stdout combinado
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancelar()
		return false
	}
	cmd.Stderr = cmd.Stdout // Combinar stderr com stdout

	if err := cmd.Start(); err != nil {
		cancelar()
		return false
	}

	fim := make(chan struct{})
	g.mu.Lock()
	g.processo = cmd
	g.fim = fim
	g.mu.Unlock()

	// Goroutine para ler a saída. É a única que chama Wait no processo.
	go func() {
		defer close(fim)
		buf := make([]byte, 4096)
		var bufferToken []string
		capturandoToken := false
		ultimaLinha := ""

		reURL := regexp.MustCompile(`https?://\S+`)
		reJSON := regexp.MustCompile(`(\{.*\})`)

		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				texto := string(buf[:n])
				for _, linha := range strings.Split(texto, "\n") {
					linhaStrip := strings.TrimSpace(linha)
					if linhaStrip != "" {
						ultimaLinha = linhaStrip
					}

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
		g.mu.Lock()
		switch {
		case ctx.Err() == context.DeadlineExceeded:
			g.erro = erroTempoEsgotado(limite, []string{"authorize"}).Error()
		case g.token == "":
			// Demanda 018: login recusado ou cancelado no navegador. O
			// authorize sai sem token; quem espera fica sabendo na hora, em
			// vez de esperar o tempo todo.
			g.erro = "o rclone authorize terminou sem token"
			if l := reDataLog.ReplaceAllString(ultimaLinha, ""); l != "" {
				g.erro += ": " + l
			}
		}
		g.mu.Unlock()
		cancelar()
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
		Erro:      g.erro,
	}
}

// Abortar encerra o processo OAuth se estiver em execução.
func (g *GerenciadorOAuth) Abortar() {
	g.mu.Lock()
	cmd := g.processo
	fim := g.fim
	g.processo = nil
	g.fim = nil
	g.mu.Unlock()

	if cmd != nil && cmd.Process != nil {
		cmd.Process.Kill()
		// O Wait é da goroutine de leitura de Iniciar; aqui só se espera o
		// aviso de que ele voltou.
		if fim != nil {
			<-fim
		}
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
