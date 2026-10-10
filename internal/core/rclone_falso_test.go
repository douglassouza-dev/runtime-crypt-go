package core

// Este arquivo transforma o próprio binário de teste em um rclone falso.
//
// Quando o binário de teste roda com RCLONE_FALSO_ATIVO=1 no ambiente, ele não
// executa os testes: age como rclone (ou como xdg-open, open, cmd, explorer),
// grava a chamada em RCLONE_FALSO_DIR/chamadas.log e responde conforme os
// arquivos e variáveis abaixo. Assim `go test ./internal/core/...` roda sem
// rclone instalado.
//
//   RCLONE_FALSO_DIR/dump.json  saída de `config dump` (e base de `listremotes`)
//   RCLONE_FALSO_DIR/lsd.txt    saída de `lsd`
//   RCLONE_FALSO_DIR/lsjson.json saída de `lsjson` (padrão: [])
//   RCLONE_FALSO_FALHA=1        qualquer comando escreve no stderr e sai com 1
//   RCLONE_FALSO_TOKEN=<json>   `authorize` entrega este token e sai
//   RCLONE_CONFIG=<arquivo>     com esta variável, `config create|delete|dump`
//                               e `listremotes` usam este arquivo INI, como o
//                               rclone real (create em nome existente
//                               substitui a seção, igual ao rclone v1.60)
//   RCLONE_FALSO_FALHA_TIPO=<t> `config create <nome> <t>` falha
//   RCLONE_FALSO_DORME=<dur>    qualquer comando dorme este tempo antes de
//                               responder (ex.: 10m), para testar tempo limite
//
// `mount` e `authorize` sem token ficam vivos até serem encerrados (no máximo
// 2 min, para nunca deixar processo órfão depois dos testes).

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"
)

const (
	envFalsoAtivo = "RCLONE_FALSO_ATIVO"
	envFalsoDir   = "RCLONE_FALSO_DIR"
	envFalsoFalha = "RCLONE_FALSO_FALHA"
	envFalsoToken = "RCLONE_FALSO_TOKEN"
	envFalsoDorme = "RCLONE_FALSO_DORME"
	envFalsoFalhaTipo = "RCLONE_FALSO_FALHA_TIPO"

	vidaMaximaFalso = 2 * time.Minute
)

func TestMain(m *testing.M) {
	if os.Getenv(envFalsoAtivo) == "1" {
		os.Exit(rodarRcloneFalso(os.Args[0], os.Args[1:]))
	}
	// Todo processo filho iniciado pelos testes herda esta variável e vira o
	// rclone falso.
	os.Setenv(envFalsoAtivo, "1")
	// Com -race, cada filho esperaria 1 s ao sair (atexit_sleep_ms padrão).
	os.Setenv("GORACE", strings.TrimSpace(os.Getenv("GORACE")+" atexit_sleep_ms=0"))
	os.Exit(m.Run())
}

// chamadaFalsa é uma linha de chamadas.log.
type chamadaFalsa struct {
	Programa    string   `json:"programa"`
	Args        []string `json:"args"`
	Stdin       string   `json:"stdin,omitempty"`
	TemSenhaEnv bool     `json:"tem_senha_env"`
	Pid         int      `json:"pid"`
}

func rodarRcloneFalso(arg0 string, args []string) int {
	dir := os.Getenv(envFalsoDir)
	programa := strings.TrimSuffix(strings.ToLower(filepath.Base(arg0)), ".exe")

	c := chamadaFalsa{
		Programa:    programa,
		Args:        args,
		TemSenhaEnv: os.Getenv("RCLONE_CONFIG_PASS") != "",
		Pid:         os.Getpid(),
	}
	if len(args) > 0 && args[0] == "obscure" {
		dados, _ := io.ReadAll(os.Stdin)
		c.Stdin = string(dados)
	}
	registrarChamada(dir, c)

	if d, err := time.ParseDuration(os.Getenv(envFalsoDorme)); err == nil && d > 0 {
		time.Sleep(d)
	}

	if os.Getenv(envFalsoFalha) == "1" {
		fmt.Fprintln(os.Stderr, "CRITICAL: falha do rclone falso")
		return 1
	}

	if len(args) == 0 {
		return 0
	}

	if conf := os.Getenv("RCLONE_CONFIG"); conf != "" {
		if codigo, tratado := configIniFalso(conf, args); tratado {
			return codigo
		}
	}

	switch args[0] {
	case "--version":
		fmt.Println("rclone v0.0.0-falso")
	case "obscure":
		fmt.Println("obs(" + c.Stdin + ")")
	case "config":
		if len(args) > 1 && args[1] == "dump" {
			fmt.Print(lerArquivo(dir, "dump.json", "{}"))
		}
	case "listremotes":
		var cfg map[string]any
		_ = json.Unmarshal([]byte(lerArquivo(dir, "dump.json", "{}")), &cfg)
		nomes := make([]string, 0, len(cfg))
		for n := range cfg {
			nomes = append(nomes, n)
		}
		sort.Strings(nomes)
		for _, n := range nomes {
			fmt.Println(n + ":")
		}
	case "lsd":
		fmt.Print(lerArquivo(dir, "lsd.txt", ""))
	case "lsjson":
		fmt.Print(lerArquivo(dir, "lsjson.json", "[]"))
	case "mount":
		time.Sleep(vidaMaximaFalso)
	case "authorize":
		saida := "If your browser doesn't open automatically go to the following link: http://127.0.0.1:53682/auth?state=falso\n" +
			"Log in and authorize rclone for access\nWaiting for code...\n"
		if token := os.Getenv(envFalsoToken); token != "" {
			saida += "Got code\nPaste the following into your remote machine --->\n" + token + "\n<---End paste\n"
			os.Stdout.WriteString(saida)
			return 0
		}
		os.Stdout.WriteString(saida)
		time.Sleep(vidaMaximaFalso)
	}
	return 0
}

// secaoIni é uma seção [nome] do rclone.conf falso.
type secaoIni struct {
	nome   string
	linhas []string // "chave = valor"
}

func lerIni(caminho string) []secaoIni {
	dados, _ := os.ReadFile(caminho)
	var secoes []secaoIni
	for _, linha := range strings.Split(strings.ReplaceAll(string(dados), "\r\n", "\n"), "\n") {
		linha = strings.TrimSpace(linha)
		if strings.HasPrefix(linha, "[") && strings.HasSuffix(linha, "]") {
			secoes = append(secoes, secaoIni{nome: linha[1 : len(linha)-1]})
		} else if linha != "" && len(secoes) > 0 {
			secoes[len(secoes)-1].linhas = append(secoes[len(secoes)-1].linhas, linha)
		}
	}
	return secoes
}

func gravarIni(caminho string, secoes []secaoIni) {
	var b strings.Builder
	for _, s := range secoes {
		b.WriteString("[" + s.nome + "]\n")
		for _, l := range s.linhas {
			b.WriteString(l + "\n")
		}
		b.WriteString("\n")
	}
	os.WriteFile(caminho, []byte(b.String()), 0o600)
}

// configIniFalso trata os comandos de configuração sobre o arquivo INI.
func configIniFalso(caminho string, args []string) (int, bool) {
	secoes := lerIni(caminho)
	achar := func(nome string) int {
		for i, s := range secoes {
			if s.nome == nome {
				return i
			}
		}
		return -1
	}
	switch {
	case args[0] == "listremotes":
		for _, s := range secoes {
			fmt.Println(s.nome + ":")
		}
		return 0, true
	case len(args) >= 2 && args[0] == "config" && args[1] == "dump":
		m := map[string]map[string]string{}
		for _, s := range secoes {
			kv := map[string]string{}
			for _, l := range s.linhas {
				if k, v, ok := strings.Cut(l, " = "); ok {
					kv[k] = v
				}
			}
			m[s.nome] = kv
		}
		saida, _ := json.Marshal(m)
		fmt.Print(string(saida))
		return 0, true
	case len(args) >= 4 && args[0] == "config" && args[1] == "create":
		nome, tipo := args[2], args[3]
		if t := os.Getenv(envFalsoFalhaTipo); t != "" && t == tipo {
			fmt.Fprintln(os.Stderr, "CRITICAL: falha simulada em config create "+tipo)
			return 1, true
		}
		sec := secaoIni{nome: nome, linhas: []string{"type = " + tipo}}
		for i := 4; i+1 < len(args); i += 2 {
			sec.linhas = append(sec.linhas, args[i]+" = "+args[i+1])
		}
		if i := achar(nome); i >= 0 {
			secoes[i] = sec
		} else {
			secoes = append(secoes, sec)
		}
		gravarIni(caminho, secoes)
		return 0, true
	case len(args) >= 3 && args[0] == "config" && args[1] == "delete":
		if i := achar(args[2]); i >= 0 {
			secoes = append(secoes[:i], secoes[i+1:]...)
			gravarIni(caminho, secoes)
		}
		return 0, true
	}
	return 0, false
}

func registrarChamada(dir string, c chamadaFalsa) {
	if dir == "" {
		return
	}
	linha, _ := json.Marshal(c)
	f, err := os.OpenFile(filepath.Join(dir, "chamadas.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	f.Write(append(linha, '\n'))
}

func lerArquivo(dir, nome, padrao string) string {
	dados, err := os.ReadFile(filepath.Join(dir, nome))
	if err != nil {
		return padrao
	}
	return string(dados)
}

// rcloneFalso é o lado do teste: prepara o diretório do falso e lê as chamadas.
type rcloneFalso struct {
	t   *testing.T
	dir string
	exe string
}

func novoRcloneFalso(t *testing.T) *rcloneFalso {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	f := &rcloneFalso{t: t, dir: t.TempDir(), exe: exe}
	t.Setenv(envFalsoDir, f.dir)
	t.Setenv(envFalsoFalha, "")
	t.Setenv(envFalsoToken, "")
	t.Setenv(envFalsoDorme, "")
	t.Setenv(envFalsoFalhaTipo, "")
	t.Setenv("RCLONE_CONFIG", "")
	t.Cleanup(f.matarSobreviventes)
	return f
}

func (f *rcloneFalso) falhar() { f.t.Setenv(envFalsoFalha, "1") }

// dormir faz toda chamada seguinte ao falso esperar d antes de responder.
func (f *rcloneFalso) dormir(d time.Duration) { f.t.Setenv(envFalsoDorme, d.String()) }

func (f *rcloneFalso) escrever(nome, conteudo string) {
	f.t.Helper()
	if err := os.WriteFile(filepath.Join(f.dir, nome), []byte(conteudo), 0o644); err != nil {
		f.t.Fatal(err)
	}
}

func (f *rcloneFalso) chamadas() []chamadaFalsa {
	f.t.Helper()
	arq, err := os.Open(filepath.Join(f.dir, "chamadas.log"))
	if err != nil {
		return nil
	}
	defer arq.Close()
	var lista []chamadaFalsa
	sc := bufio.NewScanner(arq)
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	for sc.Scan() {
		var c chamadaFalsa
		if err := json.Unmarshal(sc.Bytes(), &c); err != nil {
			f.t.Fatalf("linha inválida em chamadas.log: %q", sc.Text())
		}
		lista = append(lista, c)
	}
	return lista
}

// esperarChamadas espera até n chamadas registradas (para processos iniciados
// com Start, que podem ainda não ter escrito no log).
func (f *rcloneFalso) esperarChamadas(n int, limite time.Duration) []chamadaFalsa {
	f.t.Helper()
	fim := time.Now().Add(limite)
	for {
		cs := f.chamadas()
		if len(cs) >= n || time.Now().After(fim) {
			return cs
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// copiarComo coloca uma cópia do falso em dir com o nome dado (para testes que
// procuram o programa no PATH).
func (f *rcloneFalso) copiarComo(dir, nome string) string {
	f.t.Helper()
	if runtime.GOOS == "windows" && !strings.HasSuffix(nome, ".exe") {
		nome += ".exe"
	}
	dados, err := os.ReadFile(f.exe)
	if err != nil {
		f.t.Fatal(err)
	}
	destino := filepath.Join(dir, nome)
	if err := os.WriteFile(destino, dados, 0o755); err != nil {
		f.t.Fatal(err)
	}
	return destino
}

// matarSobreviventes encerra qualquer processo falso que ainda esteja vivo ao
// fim do teste, para um teste que falhou não deixar órfãos.
func (f *rcloneFalso) matarSobreviventes() {
	for _, c := range f.chamadas() {
		if processoVivoNoSO(c.Pid) {
			if p, err := os.FindProcess(c.Pid); err == nil {
				p.Kill()
			}
		}
	}
}

// argsContem confere se a sequência esperada aparece em args.
func argsContem(args []string, seq ...string) bool {
	for i := 0; i+len(seq) <= len(args); i++ {
		ok := true
		for j := range seq {
			if args[i+j] != seq[j] {
				ok = false
				break
			}
		}
		if ok {
			return true
		}
	}
	return false
}

// esperarAte repete cond até ela valer ou o limite passar.
func esperarAte(limite time.Duration, cond func() bool) bool {
	fim := time.Now().Add(limite)
	for time.Now().Before(fim) {
		if cond() {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return cond()
}
