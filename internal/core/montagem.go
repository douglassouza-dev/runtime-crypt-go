package core

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
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

	// controle é ponteiro para InfoMontagem poder ser copiada (ObterMontagens)
	// sem copiar os atomics (demanda 010).
	controle *controleMontagem

	// rc fala com o controle remoto deste rclone (demanda 025).
	rc *clienteRC
}

// controleMontagem guarda as marcas da demanda 010 de uma montagem.
type controleMontagem struct {
	// desmontando é ligado por DesmontarUnidade antes de pedir o fim do
	// processo. Só então a saída do processo tira a montagem do mapa; uma
	// saída sem pedido deixa a montagem como "falhou".
	desmontando atomic.Bool

	// conferindo fica ligado enquanto um os.Stat do ponto de montagem está em
	// andamento, para um ponto travado não acumular goroutines.
	conferindo atomic.Bool

	// emCurso é a conferência do ponto em andamento. Quem chega enquanto ela
	// roda espera o mesmo resultado, em vez de ler "não respondeu" (demanda
	// 025: leituras ao mesmo tempo — tela, bandeja, Trancar — não podem ver
	// uma falha que não existe).
	mu      sync.Mutex
	emCurso *conferenciaPonto
}

// conferenciaPonto é um os.Stat do ponto em andamento.
type conferenciaPonto struct {
	pronto chan struct{}
	existe bool
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

// EstadoMontagem é o estado de um cofre/montagem (demandas 010 e 018). O core
// é o único lugar que decide o estado; a tela só lê e traduz para as frases
// dela.
type EstadoMontagem string

const (
	// EstadoDesmontado: não há processo do rclone para o cofre.
	EstadoDesmontado EstadoMontagem = "desmontado"
	// EstadoMontando: MontarUnidade está esperando a unidade aparecer.
	EstadoMontando EstadoMontagem = "montando"
	// EstadoMontado: o processo do rclone vive E o ponto de montagem existe.
	EstadoMontado EstadoMontagem = "montado"
	// EstadoFalhou: a montagem nunca subiu, ou subiu e uma das duas condições
	// deixou de valer. Motivo diz qual.
	EstadoFalhou EstadoMontagem = "falhou"
)

// Motivos de EstadoFalhou.
const (
	MotivoProcessoTerminou = "processo terminou"
	MotivoPontoSumiu       = "ponto de montagem sumiu"
	// MotivoPontoNaoResponde leva o limite da conferência (%s).
	MotivoPontoNaoResponde = "ponto de montagem nao respondeu em %s"
)

// StatusMontagem representa o estado público de uma montagem.
type StatusMontagem struct {
	Letra         string         `json:"letra"`
	Remoto        string         `json:"remoto"`
	Ativo         bool           `json:"ativo"` // Estado == EstadoMontado
	Estado        EstadoMontagem `json:"estado"`
	Motivo        string         `json:"motivo,omitempty"`
	PontoMontagem string         `json:"ponto_montagem"`
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

	// caminhoPonto dá o ponto de montagem de uma letra ("V" → "V:\\"). Os
	// testes trocam por uma pasta temporária (demanda 010).
	caminhoPonto func(letra string) string

	// limitePonto é quanto Status espera o os.Stat do ponto de montagem. Uma
	// unidade travada pode prender o os.Stat (demanda 010).
	limitePonto time.Duration

	// sinalizar pede o fim do processo: Interrupt (forcar=false) ou Kill
	// (forcar=true). Em produção é sinalizarProcesso; os testes trocam para
	// simular um rclone que não morre (demanda 002).
	sinalizar func(p *os.Process, forcar bool) error

	// esperaEncerrar é quanto se espera o processo terminar depois de cada
	// pedido, e o ponto de montagem sumir depois que ele terminou.
	esperaEncerrar time.Duration

	// montando guarda os remotos com MontarUnidade em andamento (demanda
	// 018). Um segundo pedido para o mesmo remoto é recusado.
	montando map[string]bool
	// falhasMontagem guarda, por remoto, o motivo curto da última tentativa
	// que nunca chegou a montar (demanda 018). Sai na próxima tentativa.
	falhasMontagem map[string]string

	// raizPontos é a pasta onde ficam os pontos de montagem fora do Windows
	// (demanda 013): <raizPontos>/<nome do cofre>. Os testes trocam.
	raizPontos string
	// desmontarPonto desfaz uma montagem FUSE que ficou presa depois de o
	// processo terminar (fusermount -u / umount). Os testes trocam.
	desmontarPonto func(caminho string) error

	// enviando guarda, por letra/pasta, quantos arquivos faltam subir
	// enquanto DesmontarUnidade espera o envio (demanda 025).
	enviando map[string]int
	// esperaSemEnvio é quanto se espera sem nenhum avanço no envio (fila e
	// bytes parados) antes de desistir de trancar.
	esperaSemEnvio time.Duration
	// intervaloEnvio é de quanto em quanto o rc é consultado.
	intervaloEnvio time.Duration
}

// EsperaSemEnvioPadrao: o write-back padrão é 5 s; uma fila parada por 2 min
// é envio que não vai andar sozinho (rede fora, nuvem recusando).
const EsperaSemEnvioPadrao = 2 * time.Minute

// EsperaEncerrarPadrao é a espera por pedido de encerramento: cobre o
// --vfs-write-back padrão (5 s) mais a desmontagem do WinFsp/FUSE.
const EsperaEncerrarPadrao = 5 * time.Second

// LimitePontoPadrao é a espera pelo os.Stat do ponto de montagem em Status.
// Um os.Stat numa unidade WinFsp saudável volta em milissegundos; o valor real
// com a unidade travada é desconhecido, por isso a demora é registrada no log
// (registrarDemoraPonto).
const LimitePontoPadrao = 2 * time.Second

// pontoDaLetra é o caminhoPonto de produção. No Windows a "letra" é a letra
// da unidade ("V" → "V:\\"); fora dele é a própria pasta (demanda 013).
func pontoDaLetra(letra string) string {
	if runtime.GOOS == "windows" {
		return letra + ":\\"
	}
	return letra
}

// normalizarPonto deixa a letra (Windows) ou a pasta (Linux/macOS) na forma
// usada como chave do mapa de montagens.
func normalizarPonto(letra string) string {
	if runtime.GOOS == "windows" {
		return strings.ToUpper(strings.TrimRight(letra, ":\\"))
	}
	if letra == "" {
		return ""
	}
	return filepath.Clean(letra)
}

// argumentoMount é o ponto como o `rclone mount` recebe: "V:" no Windows, a
// pasta fora dele.
func argumentoMount(letra string) string {
	if runtime.GOOS == "windows" {
		return letra + ":"
	}
	return letra
}

// PastaPontosPadrao é onde ficam os pontos de montagem fora do Windows
// (demanda 013): ~/RuntimeCrypto. Não é configurável por cofre.
func PastaPontosPadrao() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		home = os.TempDir()
	}
	return filepath.Join(home, "RuntimeCrypto")
}

// pastaDoCofre é o ponto de montagem padrão do remoto fora do Windows. Uma
// barra no nome vira "_", para o ponto ficar dentro de raizPontos.
func (g *GerenciadorMontagem) pastaDoCofre(remoto string) string {
	nome := strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' {
			return '_'
		}
		return r
	}, strings.TrimSuffix(remoto, ":"))
	if nome == "" || nome == "." || nome == ".." {
		nome = "_" + nome
	}
	return filepath.Join(g.raizPontos, nome)
}

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
		pontoExiste: pontoAtivo,
		sinalizar:   sinalizarProcesso,

		caminhoPonto: pontoDaLetra,
		limitePonto:  LimitePontoPadrao,

		esperaEncerrar: EsperaEncerrarPadrao,

		montando:       make(map[string]bool),
		falhasMontagem: make(map[string]string),

		raizPontos:     PastaPontosPadrao(),
		desmontarPonto: desmontarPontoFuse,

		enviando:       make(map[string]int),
		esperaSemEnvio: EsperaSemEnvioPadrao,
		intervaloEnvio: 250 * time.Millisecond,
	}
}

// MsgJaDestrancando é a recusa de um segundo MontarUnidade do mesmo remoto
// enquanto o primeiro espera (demanda 018).
const MsgJaDestrancando = "Este cofre ja esta sendo destrancado."

// MontarUnidade monta um remoto crypt como unidade virtual.
//
// Demanda 018: enquanto espera, o remoto fica no estado montando, e um
// segundo pedido para ele é recusado sem iniciar outro rclone. Se a montagem
// não sobe, o motivo curto fica guardado (estado falhou, nunca montou).
func (g *GerenciadorMontagem) MontarUnidade(remoto string, letra string, senha string, configVfsOverride map[string]string) (bool, string, string) {
	if !strings.HasSuffix(remoto, ":") {
		remoto = remoto + ":"
	}

	g.mu.Lock()
	if g.montando[remoto] {
		g.mu.Unlock()
		return false, MsgJaDestrancando, ""
	}
	g.montando[remoto] = true
	delete(g.falhasMontagem, remoto)
	g.mu.Unlock()

	ok, msg, letraMontada, motivo := g.montar(remoto, letra, senha, configVfsOverride)

	// A falha entra antes de o remoto sair de montando: a tela nunca vê
	// desmontado entre os dois.
	g.mu.Lock()
	if !ok && motivo != "" {
		g.falhasMontagem[remoto] = motivo
	}
	delete(g.montando, remoto)
	g.mu.Unlock()
	return ok, msg, letraMontada
}

// montar é o corpo de MontarUnidade. motivo é a frase curta de por que não
// montou ("" quando montou, ou quando o remoto já estava montado).
func (g *GerenciadorMontagem) montar(remoto string, letra string, senha string, configVfsOverride map[string]string) (ok bool, msg string, letraMontada string, motivo string) {
	if g.executavel == "" {
		return false, "RClone nao disponivel.", "", "o rclone não está disponível"
	}

	// Demanda 010: uma montagem que falhou continua no mapa até alguém agir.
	// Destrancar de novo é agir: a falha deste remoto (e a da letra pedida)
	// sai antes, e o processo que sobrou, se sobrou, é encerrado.
	g.limparFalhas(remoto, letra)

	// O que sobrou deste remoto no mapa está saudável: não monta duas vezes.
	g.mu.Lock()
	for _, info := range g.montagens {
		if info.Remoto == remoto {
			g.mu.Unlock()
			return false, fmt.Sprintf("Este cofre ja esta montado em %s", TextoPonto(g.caminhoPonto(info.Letra))), "", ""
		}
	}
	g.mu.Unlock()

	criarPasta := false
	if letra == "" && runtime.GOOS != "windows" {
		// Demanda 013: fora do Windows o ponto padrão é uma pasta, criada
		// aqui e removida (se vazia) depois de desmontar.
		letra = g.pastaDoCofre(remoto)
		criarPasta = true
	}
	if letra == "" {
		disponiveis := ObterLetrasDisponiveis(g.letrasOcupadas())
		if len(disponiveis) == 0 {
			return false, "Nenhuma letra de unidade disponivel.", "", "nenhuma letra de unidade livre"
		}
		letra = disponiveis[0]
	}

	letra = normalizarPonto(letra)

	g.mu.Lock()
	if _, existe := g.montagens[letra]; existe {
		g.mu.Unlock()
		rotulo := RotuloPonto(g.caminhoPonto(letra))
		return false, fmt.Sprintf("%s ja esta em uso.", maiuscula(rotulo)), "", rotulo + " já está em uso"
	}
	g.mu.Unlock()

	if criarPasta {
		if err := os.MkdirAll(letra, 0o700); err != nil {
			return false, fmt.Sprintf("Nao deu para criar a pasta %s: %v", TextoPonto(letra), err), "", fmt.Sprintf("não deu para criar a pasta %s", TextoPonto(letra))
		}
		// Se não montar, a pasta criada (vazia) sai.
		defer func() {
			if !ok {
				g.removerPastaVazia(letra)
			}
		}()
	}

	pontoMontagem := argumentoMount(letra)

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

	// Demanda 025: o rc do rclone, só em 127.0.0.1, para Trancar saber se
	// ainda há envio pendente.
	rc, err := novoClienteRC()
	if err != nil {
		return false, fmt.Sprintf("Nao deu para preparar o controle do rclone: %v", err), "", "não deu para preparar o controle do rclone: " + err.Error()
	}
	args = append(args, rc.args()...)

	// O mount é um processo de vida longa: não leva tempo limite de execução.
	// A espera de 45 s para a unidade aparecer é tratada na demanda 003.
	cmd := exec.CommandContext(context.Background(), g.executavel, args...)

	env := append(os.Environ(), rc.env()...)
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
		return false, fmt.Sprintf("Erro ao iniciar rclone: %s", err.Error()), "", "o rclone não iniciou: " + err.Error()
	}

	// A goroutine acompanhar (demanda 001) começa já aqui: o único Wait do
	// processo também é o aviso de que ele saiu durante a espera.
	info := &InfoMontagem{
		Processo: cmd.Process,
		Cmd:      cmd,
		Remoto:   remoto,
		Letra:    letra,
		fim:      make(chan struct{}),
		controle: &controleMontagem{},
		rc:       rc,
	}
	go g.acompanhar(info)

	limite := limiteMontagem
	prazo := time.NewTimer(limite)
	defer prazo.Stop()

	intervalo := 200 * time.Millisecond
	for {
		select {
		case <-info.fim:
			return false, mensagemFalhaMontagem(cmd, saidaErro), "", motivoFalhaMontagem(cmd, saidaErro)
		case <-prazo.C:
			cmd.Process.Kill()
			<-info.fim
			msg := fmt.Sprintf("Timeout: A unidade nao ficou pronta em %s.", limite)
			if linhas := saidaErro.Texto(); linhas != "" {
				msg += "\n\nSaida do rclone:\n" + linhas
			}
			return false, msg, "", fmt.Sprintf("a unidade não ficou pronta em %s", limite)
		case <-time.After(intervalo):
		}

		if g.pontoExiste(g.caminhoPonto(letra)) {
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
		return false, mensagemFalhaMontagem(cmd, saidaErro), "", motivoFalhaMontagem(cmd, saidaErro)
	}

	return true, fmt.Sprintf("%s foi montada.", maiuscula(RotuloPonto(g.caminhoPonto(letra)))), letra, ""
}

// motivoFalhaMontagem é a versão curta de mensagemFalhaMontagem, para o card
// (demanda 018): a última linha do stderr do rclone, sem a data, ou o código
// de saída. Só chamar depois de info.fim fechado.
func motivoFalhaMontagem(cmd *exec.Cmd, saidaErro *ultimasLinhas) string {
	linhas := strings.Split(saidaErro.Texto(), "\n")
	for i := len(linhas) - 1; i >= 0; i-- {
		if l := strings.TrimSpace(reDataLog.ReplaceAllString(linhas[i], "")); l != "" {
			return l
		}
	}
	if cmd.ProcessState != nil {
		return fmt.Sprintf("o rclone encerrou (código %d)", cmd.ProcessState.ExitCode())
	}
	return "o rclone encerrou"
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

// acompanhar chama cmd.Wait() uma única vez para a montagem e fecha info.fim
// quando o processo termina. Se o fim foi pedido por DesmontarUnidade, tira a
// montagem do mapa. Se não foi, a montagem fica no mapa e Status passa a
// dizer "falhou: processo terminou" (demanda 010).
func (g *GerenciadorMontagem) acompanhar(info *InfoMontagem) {
	_ = info.Cmd.Wait()
	close(info.fim)

	if !info.controle.desmontando.Load() {
		return
	}
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
	letra = normalizarPonto(letra)

	g.mu.Lock()
	info, existe := g.montagens[letra]
	g.mu.Unlock()
	if !existe {
		return false, fmt.Sprintf("Nenhuma montagem ativa em %s", TextoPonto(g.caminhoPonto(letra)))
	}

	// Demanda 025: só encerra o rclone depois de a VFS enviar tudo. Se o
	// envio falha, a montagem fica como está.
	if err := g.esperarEnvio(info); err != nil {
		return false, err.Error()
	}

	info.controle.desmontando.Store(true)
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
		msg := fmt.Sprintf("O rclone (pid %d) nao terminou depois de Interrupt e duas tentativas de Kill; %s pode continuar aberta.", info.Processo.Pid, RotuloPonto(g.caminhoPonto(letra)))
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

	ponto := g.caminhoPonto(letra)
	livre := esperarCondicao(g.esperaEncerrar, func() bool { return !g.pontoExiste(ponto) })
	if !livre && runtime.GOOS != "windows" {
		// Demanda 013: o processo saiu e a montagem FUSE ficou (ex.: rclone
		// morto com Kill). fusermount -u / umount solta a pasta.
		if err := g.desmontarPonto(ponto); err != nil {
			motivos = append(motivos, err.Error())
		}
		livre = esperarCondicao(g.esperaEncerrar, func() bool { return !g.pontoExiste(ponto) })
	}
	if !livre {
		var msg string
		if EhUnidade(ponto) {
			msg = fmt.Sprintf("O rclone terminou, mas %s continua visivel. Confira no Explorador antes de considerar o cofre trancado.", RotuloPonto(ponto))
		} else {
			msg = fmt.Sprintf("O rclone terminou, mas %s continua montada. Confira antes de considerar o cofre trancado.", RotuloPonto(ponto))
		}
		if len(motivos) > 0 {
			msg += " (" + strings.Join(motivos, "; ") + ")"
		}
		return false, msg
	}
	if runtime.GOOS != "windows" {
		g.removerPastaVazia(ponto)
	}

	return true, fmt.Sprintf("%s foi desmontada.", maiuscula(RotuloPonto(ponto)))
}

// esperarEnvio espera a VFS do rclone terminar de enviar os arquivos
// gravados (demanda 025). Volta nil quando a fila zera ou quando o processo
// já não existe (aí não há o que esperar: o que ficou no cache sobe na
// próxima montagem). Volta erro quando o rc não responde com o processo vivo,
// quando o rclone diz que um envio falhou, ou quando nada anda por
// esperaSemEnvio.
func (g *GerenciadorMontagem) esperarEnvio(info *InfoMontagem) error {
	if info.rc == nil {
		return nil
	}
	defer g.marcarEnviando(info.Letra, 0)
	ultimoPendente, ultimosBytes := -1, int64(-1)
	ultimoAvanco := time.Now()
	for {
		if !info.vivo() {
			return nil
		}
		e, err := info.rc.envio()
		if err != nil {
			if !info.vivo() {
				return nil
			}
			return fmt.Errorf("não deu para conferir o envio dos arquivos (%v)", err)
		}
		if n := e.DiskCache.ErroredFiles; n > 0 {
			return fmt.Errorf("o envio de %s falhou", Arquivos(n))
		}
		pendente := e.pendentes()
		if pendente == 0 {
			return nil
		}
		g.marcarEnviando(info.Letra, pendente)

		bytes, _ := info.rc.bytesEnviados()
		if pendente != ultimoPendente || bytes != ultimosBytes {
			ultimoPendente, ultimosBytes = pendente, bytes
			ultimoAvanco = time.Now()
		}
		if time.Since(ultimoAvanco) > g.esperaSemEnvio {
			return fmt.Errorf("o envio de %s parou por %s", Arquivos(pendente), g.esperaSemEnvio)
		}
		time.Sleep(g.intervaloEnvio)
	}
}

func (g *GerenciadorMontagem) marcarEnviando(letra string, n int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if n == 0 {
		delete(g.enviando, letra)
		return
	}
	g.enviando[letra] = n
}

// removerPastaVazia tira a pasta do ponto de montagem se ela estiver vazia e
// dentro de raizPontos (demanda 013). Pasta com arquivos fica.
func (g *GerenciadorMontagem) removerPastaVazia(ponto string) {
	if filepath.Dir(filepath.Clean(ponto)) != filepath.Clean(g.raizPontos) {
		return
	}
	_ = os.Remove(ponto) // só remove pasta vazia
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

// Status retorna as montagens do mapa com o estado de cada uma (demanda 010):
// montado só quando o processo vive E o ponto de montagem existe. Se uma das
// duas falha, o estado é falhou e Motivo diz qual. É só leitura: quem tira
// uma montagem do mapa é a desmontagem, ou uma nova montagem do mesmo remoto.
func (g *GerenciadorMontagem) Status() []StatusMontagem {
	g.mu.Lock()
	infos := make([]*InfoMontagem, 0, len(g.montagens))
	for _, info := range g.montagens {
		infos = append(infos, info)
	}
	g.mu.Unlock()

	// O os.Stat de cada ponto roda fora do lock: um ponto travado não pode
	// travar quem só quer montar ou desmontar outra letra.
	resultado := make([]StatusMontagem, 0, len(infos))
	for _, info := range infos {
		estado, motivo := g.estadoDe(info)
		resultado = append(resultado, StatusMontagem{
			Letra:         info.Letra,
			Remoto:        info.Remoto,
			Ativo:         estado == EstadoMontado,
			Estado:        estado,
			Motivo:        motivo,
			PontoMontagem: g.caminhoPonto(info.Letra),
		})
	}
	sort.Slice(resultado, func(i, j int) bool { return resultado[i].Letra < resultado[j].Letra })
	return resultado
}

// estadoDe aplica a regra montado = processo vivo E ponto de montagem existe.
func (g *GerenciadorMontagem) estadoDe(info *InfoMontagem) (EstadoMontagem, string) {
	if !info.vivo() {
		return EstadoFalhou, MotivoProcessoTerminou
	}
	existe, respondeu := g.conferirPonto(info)
	if !respondeu {
		return EstadoFalhou, fmt.Sprintf(MotivoPontoNaoResponde, g.limitePonto)
	}
	if !existe {
		// Quando o rclone morre, a unidade some antes de o Wait voltar. Dá um
		// tempo curto para o fim do processo aparecer: processo terminou tem
		// precedência sobre ponto sumiu.
		select {
		case <-info.fim:
			return EstadoFalhou, MotivoProcessoTerminou
		case <-time.After(esperaFimAntesDePontoSumiu):
		}
		return EstadoFalhou, MotivoPontoSumiu
	}
	return EstadoMontado, ""
}

// esperaFimAntesDePontoSumiu é quanto estadoDe espera o fim do processo antes
// de dizer "ponto de montagem sumiu" (precedência de "processo terminou").
const esperaFimAntesDePontoSumiu = time.Second

// conferirPonto roda pontoExiste com limite de tempo. respondeu=false quando
// o limite passou. Se já há uma conferência desta montagem em andamento,
// espera o resultado dela (não se empilha outra goroutine num ponto travado).
func (g *GerenciadorMontagem) conferirPonto(info *InfoMontagem) (existe bool, respondeu bool) {
	c := info.controle
	c.mu.Lock()
	conf := c.emCurso
	caminho := g.caminhoPonto(info.Letra)
	inicio := time.Now()
	if conf == nil {
		conf = &conferenciaPonto{pronto: make(chan struct{})}
		c.emCurso = conf
		c.conferindo.Store(true)
		go func() {
			conf.existe = g.pontoExiste(caminho)
			c.mu.Lock()
			c.emCurso = nil
			c.mu.Unlock()
			c.conferindo.Store(false)
			close(conf.pronto)
		}()
	}
	c.mu.Unlock()

	select {
	case <-conf.pronto:
		registrarDemoraPonto(caminho, time.Since(inicio), true)
		return conf.existe, true
	case <-time.After(g.limitePonto):
		registrarDemoraPonto(caminho, time.Since(inicio), false)
		return false, false
	}
}

// demoraPontoNoLog é a partir de quanto a conferência do ponto vai para o log.
const demoraPontoNoLog = 500 * time.Millisecond

// registrarDemoraPonto escreve no log as conferências lentas do ponto de
// montagem. É a medição pedida pela demanda 010: o comportamento do os.Stat
// numa unidade WinFsp travada ainda não foi observado.
func registrarDemoraPonto(caminho string, demora time.Duration, respondeu bool) {
	if !respondeu {
		log.Printf("montagem: os.Stat(%q) nao respondeu em %v", caminho, demora)
		return
	}
	if demora >= demoraPontoNoLog {
		log.Printf("montagem: os.Stat(%q) levou %v", caminho, demora)
	}
}

// limparFalhas desmonta as montagens que falharam deste remoto e a da letra
// pedida, se ela falhou. Montagens saudáveis ficam como estão.
func (g *GerenciadorMontagem) limparFalhas(remoto string, letra string) {
	letra = normalizarPonto(letra)
	for _, st := range g.Status() {
		if st.Estado != EstadoFalhou {
			continue
		}
		if st.Remoto == remoto || (letra != "" && st.Letra == letra) {
			g.DesmontarUnidade(st.Letra)
		}
	}
}

// ObterMontagens retorna um mapa nome_remoto → InfoMontagem das montagens no
// estado montado. As que falharam ficam de fora (a tela usa
// EstadosPorRemoto para mostrá-las).
func (g *GerenciadorMontagem) ObterMontagens() map[string]InfoMontagem {
	status := g.Status()
	resultado := make(map[string]InfoMontagem)
	for _, s := range status {
		if s.Estado != EstadoMontado {
			continue
		}
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
		if s.Estado == EstadoMontado && strings.TrimSuffix(s.Remoto, ":") == nomeRemoto {
			return s.Letra
		}
	}
	return ""
}

// EstadoRemoto é o estado de um remoto para a tela (demanda 018).
type EstadoRemoto struct {
	Estado EstadoMontagem
	// Motivo vem com EstadoFalhou. Quando Caiu, é um dos Motivo* deste
	// arquivo; quando não, é a frase curta de por que não montou.
	Motivo string
	// Caiu diz que a montagem chegou a montado e depois caiu. Falso quando
	// ela nunca subiu.
	Caiu          bool
	Letra         string
	PontoMontagem string
	// Enviando é quantos arquivos faltam subir enquanto Trancar espera
	// (demanda 025); 0 fora disso.
	Enviando int
}

// EstadosPorRemoto devolve nome_remoto → estado de todo remoto que não está
// desmontado. Ordem de precedência: montando (uma nova tentativa em
// andamento), depois o que está no mapa (montado ou caiu), depois a última
// tentativa que não subiu.
func (g *GerenciadorMontagem) EstadosPorRemoto() map[string]EstadoRemoto {
	resultado := make(map[string]EstadoRemoto)
	g.mu.Lock()
	for remoto, motivo := range g.falhasMontagem {
		resultado[strings.TrimSuffix(remoto, ":")] = EstadoRemoto{Estado: EstadoFalhou, Motivo: motivo}
	}
	g.mu.Unlock()

	status := g.Status()
	g.mu.Lock()
	enviando := make(map[string]int, len(g.enviando))
	for l, n := range g.enviando {
		enviando[l] = n
	}
	g.mu.Unlock()
	for _, s := range status {
		e := EstadoRemoto{Estado: s.Estado, Letra: s.Letra, PontoMontagem: s.PontoMontagem}
		if s.Estado == EstadoMontado {
			e.Enviando = enviando[s.Letra]
		}
		if s.Estado == EstadoFalhou {
			e.Motivo = s.Motivo
			e.Caiu = true
		}
		resultado[strings.TrimSuffix(s.Remoto, ":")] = e
	}

	g.mu.Lock()
	for remoto := range g.montando {
		resultado[strings.TrimSuffix(remoto, ":")] = EstadoRemoto{Estado: EstadoMontando}
	}
	g.mu.Unlock()
	return resultado
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
