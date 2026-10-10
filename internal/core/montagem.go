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
	Processo *os.Process
	Cmd      *exec.Cmd
	Remoto   string
	Letra    string

	// fim é fechado quando cmd.Wait() devolve, ou seja, quando o processo do
	// rclone terminou. É a única fonte de "o processo vive" (demanda 001).
	fim chan struct{}
}

// vivo diz se o processo da montagem ainda não terminou.
func (i *InfoMontagem) vivo() bool {
	if i == nil || i.fim == nil {
		return false
	}
	select {
	case <-i.fim:
		return false
	default:
		return true
	}
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

	// pontoExiste diz se o ponto de montagem já apareceu. Em produção é
	// caminhoExiste; os testes trocam para não depender de WinFsp/FUSE.
	pontoExiste func(caminho string) bool

	// sinalizar pede o fim do processo: Interrupt (forcar=false) ou Kill
	// (forcar=true). Em produção é sinalizarProcesso; os testes trocam para
	// simular um rclone que não morre (demanda 002).
	sinalizar func(p *os.Process, forcar bool) error

	// esperaEncerrar é quanto se espera o processo terminar depois de cada
	// pedido, e o ponto de montagem sumir depois que ele terminou.
	esperaEncerrar time.Duration
}

// EsperaEncerrarPadrao é a espera por pedido de encerramento: cobre o
// --vfs-write-back padrão (5 s) mais a desmontagem do WinFsp/FUSE.
const EsperaEncerrarPadrao = 5 * time.Second

// sinalizarProcesso é o sinalizar de produção.
func sinalizarProcesso(p *os.Process, forcar bool) error {
	if forcar {
		return p.Kill()
	}
	return p.Signal(os.Interrupt)
}

// NovoGerenciadorMontagem cria uma instância do gerenciador de montagens.
func NovoGerenciadorMontagem(executavel string, configVfs *ConfigVfs) *GerenciadorMontagem {
	return &GerenciadorMontagem{
		montagens:   make(map[string]*InfoMontagem),
		executavel:  executavel,
		configVfs:   configVfs,
		pontoExiste: caminhoExiste,
		sinalizar:   sinalizarProcesso,

		esperaEncerrar: EsperaEncerrarPadrao,
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

	// O mount é um processo de vida longa: não leva tempo limite de execução.
	// A espera de 45 s para a unidade aparecer é tratada na demanda 003.
	cmd := exec.CommandContext(context.Background(), g.executavel, args...)

	env := os.Environ()
	if senha != "" {
		env = append(env, "RCLONE_CONFIG_PASS="+senha)
	}
	cmd.Env = env

	if runtime.GOOS == "windows" {
		ocultarJanela(cmd)
	}

	// Demanda 003: o stderr do rclone é guardado (últimas linhas) para a
	// mensagem de erro dizer o motivo real.
	saidaErro := novasUltimasLinhas(linhasErroMontagem)
	cmd.Stdout = nil
	cmd.Stderr = saidaErro

	if err := cmd.Start(); err != nil {
		return false, fmt.Sprintf("Erro ao iniciar rclone: %s", err.Error()), ""
	}

	// A goroutine acompanhar (demanda 001) começa já aqui: o único Wait do
	// processo também é o aviso de que ele saiu durante a espera.
	info := &InfoMontagem{
		Processo: cmd.Process,
		Cmd:      cmd,
		Remoto:   remoto,
		Letra:    letra,
		fim:      make(chan struct{}),
	}
	go g.acompanhar(info)

	limite := limiteMontagem
	prazo := time.NewTimer(limite)
	defer prazo.Stop()

	intervalo := 200 * time.Millisecond
	for {
		select {
		case <-info.fim:
			return false, mensagemFalhaMontagem(cmd, saidaErro), ""
		case <-prazo.C:
			cmd.Process.Kill()
			<-info.fim
			msg := fmt.Sprintf("Timeout: A unidade nao ficou pronta em %s.", limite)
			if linhas := saidaErro.Texto(); linhas != "" {
				msg += "\n\nSaida do rclone:\n" + linhas
			}
			return false, msg, ""
		case <-time.After(intervalo):
		}

		if g.pontoExiste(letra + ":\\") {
			break
		}

		if intervalo < time.Second {
			intervalo = time.Duration(float64(intervalo) * 1.5)
			if intervalo > time.Second {
				intervalo = time.Second
			}
		}
	}

	g.mu.Lock()
	g.montagens[letra] = info
	g.mu.Unlock()
	// Se o processo saiu entre o ponto aparecer e a entrada no mapa, a
	// acompanhar já passou e não tirou nada: tira aqui.
	if !info.vivo() {
		g.mu.Lock()
		if atual, ok := g.montagens[letra]; ok && atual == info {
			delete(g.montagens, letra)
		}
		g.mu.Unlock()
		return false, mensagemFalhaMontagem(cmd, saidaErro), ""
	}

	return true, fmt.Sprintf("Unidade %s: montada com sucesso.", letra), letra
}

// LimiteMontagemPadrao é quanto se espera a unidade aparecer. O WinFsp pode
// levar dezenas de segundos na primeira montagem do dia; a saída precoce do
// rclone não espera isso (demanda 003).
const LimiteMontagemPadrao = 45 * time.Second

// limiteMontagem começa com LimiteMontagemPadrao; só os testes encurtam.
var limiteMontagem = LimiteMontagemPadrao

// linhasErroMontagem é quantas linhas finais do stderr do rclone entram na
// mensagem de erro.
const linhasErroMontagem = 20

// mensagemFalhaMontagem explica a saída do rclone durante a montagem, com as
// últimas linhas do stderr dele. Só chamar depois de info.fim fechado.
func mensagemFalhaMontagem(cmd *exec.Cmd, saidaErro *ultimasLinhas) string {
	msg := "Falha ao montar: o rclone encerrou"
	if cmd.ProcessState != nil {
		msg += fmt.Sprintf(" (codigo %d)", cmd.ProcessState.ExitCode())
	}
	if linhas := saidaErro.Texto(); linhas != "" {
		return msg + ":\n" + linhas
	}
	return msg + " sem mensagem."
}

// ultimasLinhas é um io.Writer que guarda só as n últimas linhas escritas.
type ultimasLinhas struct {
	mu      sync.Mutex
	n       int
	linhas  []string
	parcial string
}

func novasUltimasLinhas(n int) *ultimasLinhas { return &ultimasLinhas{n: n} }

func (u *ultimasLinhas) Write(p []byte) (int, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	texto := u.parcial + string(p)
	partes := strings.Split(texto, "\n")
	u.parcial = partes[len(partes)-1]
	for _, l := range partes[:len(partes)-1] {
		l = strings.TrimRight(l, "\r")
		if strings.TrimSpace(l) == "" {
			continue
		}
		u.linhas = append(u.linhas, l)
		if len(u.linhas) > u.n {
			u.linhas = u.linhas[len(u.linhas)-u.n:]
		}
	}
	return len(p), nil
}

// Texto devolve as linhas guardadas, incluindo uma última sem quebra.
func (u *ultimasLinhas) Texto() string {
	u.mu.Lock()
	defer u.mu.Unlock()
	linhas := u.linhas
	if p := strings.TrimSpace(u.parcial); p != "" {
		linhas = append(append([]string(nil), linhas...), p)
		if len(linhas) > u.n {
			linhas = linhas[len(linhas)-u.n:]
		}
	}
	return strings.Join(linhas, "\n")
}

// acompanhar chama cmd.Wait() uma única vez para a montagem. Quando o processo
// termina, fecha info.fim e tira a montagem do mapa (se ela ainda for a mesma).
func (g *GerenciadorMontagem) acompanhar(info *InfoMontagem) {
	_ = info.Cmd.Wait()
	close(info.fim)

	g.mu.Lock()
	if atual, ok := g.montagens[info.Letra]; ok && atual == info {
		delete(g.montagens, info.Letra)
	}
	g.mu.Unlock()
}

// DesmontarUnidade desmonta uma unidade ativa (demanda 002).
//
// Primeiro pede o encerramento normal (Interrupt), para o rclone terminar de
// enviar o que está no write-back. No Windows, Interrupt não existe para
// processos (Signal devolve erro na hora) e o rclone roda sem console
// (CREATE_NO_WINDOW), então também não há Ctrl+C a mandar; lá o caminho vai
// direto para Kill, sem gastar a espera. Depois tenta Kill duas vezes.
//
// O fim do processo é sabido pela goroutine acompanhar da demanda 001 (único
// Wait). Só devolve true quando o processo terminou e o ponto de montagem
// sumiu; fora isso devolve false com o motivo.
func (g *GerenciadorMontagem) DesmontarUnidade(letra string) (bool, string) {
	letra = strings.ToUpper(strings.TrimRight(letra, ":\\"))

	g.mu.Lock()
	info, existe := g.montagens[letra]
	g.mu.Unlock()
	if !existe {
		return false, fmt.Sprintf("Nenhuma montagem ativa na letra %s:", letra)
	}

	terminou := false
	var motivos []string
	for tentativa := 0; tentativa < 3 && !terminou; tentativa++ {
		forcar := tentativa > 0
		if err := g.sinalizar(info.Processo, forcar); err != nil {
			if !forcar {
				// Interrupt não suportado (Windows): segue para Kill já.
				motivos = append(motivos, "Interrupt: "+err.Error())
				continue
			}
			motivos = append(motivos, "Kill: "+err.Error())
		}
		select {
		case <-info.fim:
			terminou = true
		case <-time.After(g.esperaEncerrar):
		}
	}

	if !terminou {
		msg := fmt.Sprintf("Unidade %s: o rclone (pid %d) nao terminou depois de Interrupt e duas tentativas de Kill. A unidade pode continuar aberta.", letra, info.Processo.Pid)
		if len(motivos) > 0 {
			msg += " (" + strings.Join(motivos, "; ") + ")"
		}
		return false, msg
	}

	// O processo terminou; acompanhar já tira a montagem do mapa. Garante
	// aqui também, para quem chama ver o mapa limpo ao voltar.
	g.mu.Lock()
	if atual, ok := g.montagens[letra]; ok && atual == info {
		delete(g.montagens, letra)
	}
	g.mu.Unlock()

	ponto := letra + ":\\"
	if !esperarCondicao(g.esperaEncerrar, func() bool { return !g.pontoExiste(ponto) }) {
		return false, fmt.Sprintf("Unidade %s: o rclone terminou, mas %s continua visivel. Confira no Explorador antes de considerar o cofre trancado.", letra, ponto)
	}

	return true, fmt.Sprintf("Unidade %s: desmontada com sucesso.", letra)
}

// esperarCondicao repete cond até ela valer ou o limite passar.
func esperarCondicao(limite time.Duration, cond func() bool) bool {
	fim := time.Now().Add(limite)
	for {
		if cond() {
			return true
		}
		if time.Now().After(fim) {
			return false
		}
		time.Sleep(50 * time.Millisecond)
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

// Status retorna as montagens cujo processo ainda vive. É só leitura: quem
// tira uma montagem do mapa é a goroutine acompanhar ou a desmontagem.
func (g *GerenciadorMontagem) Status() []StatusMontagem {
	g.mu.Lock()
	defer g.mu.Unlock()

	resultado := make([]StatusMontagem, 0)

	for letra, info := range g.montagens {
		if !info.vivo() {
			continue
		}
		resultado = append(resultado, StatusMontagem{
			Letra:         letra,
			Remoto:        info.Remoto,
			Ativo:         true,
			PontoMontagem: letra + ":\\",
		})
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

	// Verificar letras já em uso no SO
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
