package core

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

// InfoMontagem armazena informações de uma montagem ativa.
type InfoMontagem struct {
	Processo  *os.Process
	Cmd       *exec.Cmd
	Remoto    string
	Letra     string
	finalizado chan struct{}
}

// StatusMontagem representa o estado público de uma montagem.
type StatusMontagem struct {
	Letra         string `json:"letra"`
	Remoto        string `json:"remoto"`
	Ativo         bool   `json:"ativo"`
	PontoMontagem string `json:"ponto_montagem"`
}

// GerenciadorMontagem controla montagens e desmontagens de unidades virtuais.
type GerenciadorMontagem struct {
	montagens  map[string]*InfoMontagem
	mu         sync.Mutex
	executavel string
	configVfs  *ConfigVfs
}

// NovoGerenciadorMontagem cria uma instância do gerenciador de montagens.
func NovoGerenciadorMontagem(executavel string, configVfs *ConfigVfs) *GerenciadorMontagem {
	return &GerenciadorMontagem{
		montagens:  make(map[string]*InfoMontagem),
		executavel: executavel,
		configVfs:  configVfs,
	}
}

// MontarUnidade monta um remoto crypt como unidade virtual.
func (g *GerenciadorMontagem) MontarUnidade(remoto string, letra string, senha string, configVfsOverride map[string]string) (bool, string, string) {
	if g.executavel == "" {
		return false, "RClone nao disponivel.", ""
	}

	if letra == "" {
		disponiveis := ObterLetrasDisponiveis(g.letrasOcupadas())
		if len(disponiveis) == 0 {
			return false, "Nenhuma letra de unidade disponivel.", ""
		}
		letra = disponiveis[0]
	}

	letra = strings.ToUpper(strings.TrimRight(letra, ":\\"))

	g.mu.Lock()
	if _, existe := g.montagens[letra]; existe {
		g.mu.Unlock()
		return false, fmt.Sprintf("A letra %s: ja esta em uso.", letra), ""
	}
	g.mu.Unlock()

	if !strings.HasSuffix(remoto, ":") {
		remoto = remoto + ":"
	}

	pontoMontagem := letra + ":"

	args := []string{"mount", remoto, pontoMontagem}
	args = append(args, g.configVfs.ConstruirArgs(configVfsOverride)...)
	args = append(args,
		"--volname", fmt.Sprintf("RuntimeCrypto (%s)", strings.TrimSuffix(remoto, ":")),
		"--no-checksum",
		"--no-modtime",
	)

	if runtime.GOOS == "windows" {
		args = append(args, "--network-mode", "--no-console")
	}

	cmd := exec.Command(g.executavel, args...)

	env := os.Environ()
	if senha != "" {
		env = append(env, "RCLONE_CONFIG_PASS="+senha)
	}
	cmd.Env = env

	if runtime.GOOS == "windows" {
		ocultarJanela(cmd)
	}

	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return false, fmt.Sprintf("Erro ao iniciar rclone: %s", err.Error()), ""
	}

	info := &InfoMontagem{
		Processo:  cmd.Process,
		Cmd:       cmd,
		Remoto:    remoto,
		Letra:     letra,
		finalizado: make(chan struct{}),
	}

	go func() {
		cmd.Wait()
		close(info.finalizado)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	intervalo := 200 * time.Millisecond

	for {
		select {
		case <-ctx.Done():
			cmd.Process.Kill()
			return false, "Timeout: A unidade nao ficou pronta em 45 segundos.", ""
		case <-info.finalizado:
			return false, "Falha ao montar: Processo encerrou inesperadamente.", ""
		default:
		}

		time.Sleep(intervalo)

		if caminhoExiste(letra + ":\\") {
			g.mu.Lock()
			if _, existe := g.montagens[letra]; existe {
				g.mu.Unlock()
				cmd.Process.Kill()
				return false, fmt.Sprintf("A letra %s: foi ocupada por outra montagem (concorrencia).", letra), ""
			}
			g.montagens[letra] = info
			g.mu.Unlock()
			return true, fmt.Sprintf("Unidade %s: montada com sucesso.", letra), letra
		}

		if intervalo < time.Second {
			intervalo = time.Duration(float64(intervalo) * 1.5)
			if intervalo > time.Second {
				intervalo = time.Second
			}
		}
	}
}

// DesmontarUnidade desmonta uma unidade ativa.
func (g *GerenciadorMontagem) DesmontarUnidade(letra string) (bool, string) {
	letra = strings.ToUpper(strings.TrimRight(letra, ":\\"))

	g.mu.Lock()
	info, existe := g.montagens[letra]
	if !existe {
		g.mu.Unlock()
		return false, fmt.Sprintf("Nenhuma montagem ativa na letra %s:", letra)
	}
	g.mu.Unlock()

	if !estaFinalizado(info) {
		if runtime.GOOS != "windows" {
			info.Processo.Signal(os.Interrupt)

			select {
			case <-info.finalizado:
				goto removido
			case <-time.After(5 * time.Second):
			}
		}

		info.Processo.Kill()
		select {
		case <-info.finalizado:
		case <-time.After(5 * time.Second):
			residual := estaFinalizado(info)
			g.mu.Lock()
			if !residual {
				g.mu.Unlock()
				return false, fmt.Sprintf("Nao foi possivel encerrar o processo em %s:. Tente novamente.", letra)
			}
			delete(g.montagens, letra)
			g.mu.Unlock()
			return true, fmt.Sprintf("Unidade %s: desmontada (processo residual limpo).", letra)
		}
	}

removido:
	g.mu.Lock()
	delete(g.montagens, letra)
	g.mu.Unlock()

	return true, fmt.Sprintf("Unidade %s: desmontada com sucesso.", letra)
}

// estaFinalizado verifica se o canal finalizado está fechado (não-bloqueante).
func estaFinalizado(info *InfoMontagem) bool {
	select {
	case <-info.finalizado:
		return true
	default:
		return false
	}
}

// DesmontarTodas desmonta todas as unidades ativas.
func (g *GerenciadorMontagem) DesmontarTodas() {
	g.mu.Lock()
	letras := make([]string, 0, len(g.montagens))
	for letra := range g.montagens {
		letras = append(letras, letra)
	}
	g.mu.Unlock()

	for _, letra := range letras {
		g.DesmontarUnidade(letra)
	}
}

// Status retorna as montagens ativas, removendo as que já encerraram.
func (g *GerenciadorMontagem) Status() []StatusMontagem {
	g.mu.Lock()
	defer g.mu.Unlock()

	resultado := make([]StatusMontagem, 0)
	letrasRemover := make([]string, 0)

	for letra, info := range g.montagens {
		if estaFinalizado(info) {
			letrasRemover = append(letrasRemover, letra)
			continue
		}
		resultado = append(resultado, StatusMontagem{
			Letra:         letra,
			Remoto:        info.Remoto,
			Ativo:         true,
			PontoMontagem: letra + ":\\",
		})
	}

	for _, l := range letrasRemover {
		delete(g.montagens, l)
	}

	return resultado
}

// ObterMontagens retorna um mapa nome_remoto → InfoMontagem das montagens ativas.
func (g *GerenciadorMontagem) ObterMontagens() map[string]InfoMontagem {
	status := g.Status()
	resultado := make(map[string]InfoMontagem)
	for _, s := range status {
		nomeRemoto := strings.TrimSuffix(s.Remoto, ":")
		resultado[nomeRemoto] = InfoMontagem{
			Remoto: s.Remoto,
			Letra:  s.Letra,
		}
	}
	return resultado
}

// ObterLetraPorRemoto retorna a letra de montagem de um remoto, ou "" se não montado.
func (g *GerenciadorMontagem) ObterLetraPorRemoto(nomeRemoto string) string {
	nomeRemoto = strings.TrimSuffix(nomeRemoto, ":")
	for _, s := range g.Status() {
		if strings.TrimSuffix(s.Remoto, ":") == nomeRemoto {
			return s.Letra
		}
	}
	return ""
}

// letrasOcupadas retorna as letras atualmente em uso pelas montagens.
func (g *GerenciadorMontagem) letrasOcupadas() []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	ocupadas := make([]string, 0, len(g.montagens))
	for letra := range g.montagens {
		ocupadas = append(ocupadas, letra)
	}
	return ocupadas
}

// ObterLetrasDisponiveis retorna as letras de unidade livres no sistema.
func ObterLetrasDisponiveis(ocupadasExtra []string) []string {
	if runtime.GOOS != "windows" {
		return nil
	}

	ocupadas := make(map[string]bool)
	for _, l := range ocupadasExtra {
		ocupadas[strings.ToUpper(l)] = true
	}

	for _, letra := range "ABCDEFGHIJKLMNOPQRSTUVWXYZ" {
		l := string(letra)
		if caminhoExiste(l + ":\\") {
			ocupadas[l] = true
		}
	}

	disponiveis := make([]string, 0)
	for _, l := range LetrasPreferidas {
		if !ocupadas[l] {
			disponiveis = append(disponiveis, l)
		}
	}
	return disponiveis
}

// caminhoExiste verifica se um caminho existe no sistema de arquivos.
func caminhoExiste(caminho string) bool {
	_, err := os.Stat(caminho)
	return err == nil
}
