package core

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"
)

// Limites de tempo das chamadas ao rclone (demanda 008). Cada constante diz
// por que tem aquele valor. As variáveis minúsculas logo abaixo começam com
// estes valores; só os testes as encurtam.
const (
	// LimiteVersaoPadrao vale para `rclone --version`, que não sai da máquina.
	// NovoGerenciador pode chamar três vezes antes de a janela abrir, então o
	// pior caso fica em 30 s.
	LimiteVersaoPadrao = 10 * time.Second

	// LimiteObscurePadrao vale para `rclone obscure`, conta local e instantânea.
	LimiteObscurePadrao = 10 * time.Second

	// LimiteConfigLocalPadrao vale para `config dump`, `config delete` e
	// `listremotes`, que só leem ou gravam o rclone.conf. Se passar disso, o
	// rclone está parado esperando algo (por exemplo, a senha de um
	// rclone.conf cifrado no stdin).
	LimiteConfigLocalPadrao = 15 * time.Second

	// LimiteConfigCreatePadrao vale para `config create`. Alguns provedores
	// (OneDrive, Drive compartilhado) consultam a API do provedor ao criar o
	// remoto, então o limite cobre uma ida e volta lenta pela rede.
	LimiteConfigCreatePadrao = 60 * time.Second

	// LimiteListagemPadrao vale para listar pastas do remoto, que vai ao
	// provedor. É o mesmo valor que `lsd` já tinha antes desta demanda.
	LimiteListagemPadrao = 30 * time.Second

	// LimiteAuthorizePadrao vale para `rclone authorize`, que espera o usuário
	// entrar na conta pelo navegador. É maior que a espera de 120 s de
	// main.go, que continua sendo quem desiste primeiro pela tela; este limite
	// só garante que o processo nunca fica vivo para sempre.
	LimiteAuthorizePadrao = 5 * time.Minute

	// esperaPipes é quanto Wait espera os pipes fecharem depois que o processo
	// morreu (exec.Cmd.WaitDelay), para um neto que herdou o stdout não
	// prender a chamada.
	esperaPipes = 2 * time.Second
)

var (
	limiteVersao       = LimiteVersaoPadrao
	limiteObscure      = LimiteObscurePadrao
	limiteConfigLocal  = LimiteConfigLocalPadrao
	limiteConfigCreate = LimiteConfigCreatePadrao
	limiteListagem     = LimiteListagemPadrao
	limiteAuthorize    = LimiteAuthorizePadrao
)

// ErrTempoEsgotado é devolvido (embrulhado) quando uma chamada ao rclone passa
// do limite e é encerrada.
var ErrTempoEsgotado = errors.New("tempo esgotado")

// chamadaRclone descreve uma execução curta do rclone.
type chamadaRclone struct {
	executavel string
	args       []string
	limite     time.Duration
	stdin      io.Reader
	// combinada junta stderr com stdout na saída (CombinedOutput).
	combinada bool
	// ocultar pede CREATE_NO_WINDOW no Windows.
	ocultar bool
}

// rodar executa o rclone com tempo limite. O processo sempre passa por Wait,
// inclusive quando é morto por tempo esgotado (exec.CommandContext mata e
// Output/CombinedOutput esperam).
func (c chamadaRclone) rodar() ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.limite)
	defer cancel()

	cmd := exec.CommandContext(ctx, c.executavel, c.args...)
	cmd.WaitDelay = esperaPipes
	if c.stdin != nil {
		cmd.Stdin = c.stdin
	}
	if c.ocultar && runtime.GOOS == "windows" {
		configurarOcultarJanela(cmd)
	}

	var saida []byte
	var err error
	if c.combinada {
		saida, err = cmd.CombinedOutput()
	} else {
		saida, err = cmd.Output()
	}
	if ctx.Err() == context.DeadlineExceeded {
		return saida, erroTempoEsgotado(c.limite, c.args)
	}
	return saida, err
}

// erroTempoEsgotado monta a mensagem com o subcomando, sem os argumentos que
// podem levar token ou senha ofuscada.
func erroTempoEsgotado(limite time.Duration, args []string) error {
	return fmt.Errorf("%w depois de %s: rclone %s", ErrTempoEsgotado, limite, descreverComando(args))
}

// descreverComando devolve só o subcomando do rclone ("config create",
// "lsd", "authorize"), nunca os valores.
func descreverComando(args []string) string {
	if len(args) == 0 {
		return ""
	}
	if args[0] == "config" && len(args) > 1 {
		return "config " + args[1]
	}
	return strings.TrimSpace(args[0])
}

// reDataLog é o prefixo de data e hora que o rclone põe em cada linha de log.
var reDataLog = regexp.MustCompile(`^\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}(\.\d+)? `)

// erroComMotivoDoRclone troca "exit status 1" por um *ErroRclone com o
// stderr classificado (demanda 027); o texto original vai para o log. Tempo
// esgotado e erros sem stderr voltam como estão (demanda 009).
func erroComMotivoDoRclone(err error) error {
	var saida *exec.ExitError
	if err == nil || !errors.As(err, &saida) {
		return err
	}
	texto := strings.TrimSpace(strings.ReplaceAll(string(saida.Stderr), "\r\n", "\n"))
	if texto == "" {
		return err
	}
	return novoErroRclone("falhou", texto, err)
}
