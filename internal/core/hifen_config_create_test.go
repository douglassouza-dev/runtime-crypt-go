package core

import (
	"crypto/rand"
	"encoding/base64"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// Testes da demanda 029: valor que começa com "-" em `config create`.

// obscuroComHifen tem o formato do `rclone obscure` e começa com "-". É o
// valor que o rclone v1.75.2 devolveu no box e que quebrava a criação.
const obscuroComHifen = "-3RPpB-dM90yplYzO40JSrYR1a-xgwdS"

func TestCriarCryptComSenhaOfuscadaQueComecaComHifen(t *testing.T) {
	g, f := novoGerenciadorFalso(t)
	conf := confFalso(t, "")
	t.Setenv(envFalsoObscure, obscuroComHifen)

	ok, msg := g.CriarCrypt("cofre", "gdrive:pasta", "s1", "", nil)

	if !ok {
		t.Fatalf("CriarCrypt: %s", msg)
	}
	var create []string
	for _, c := range f.chamadas() {
		if argsContem(c.Args, "config", "create") {
			create = c.Args
		}
	}
	quer := []string{
		"config", "create", "--", "cofre", "crypt",
		"directory_name_encryption", "true",
		"filename_encryption", "standard",
		"password", obscuroComHifen,
		"password2", obscuroComHifen,
		"remote", "gdrive:pasta",
	}
	if !reflect.DeepEqual(create, quer) {
		t.Errorf("argv do config create:\n got %q\nquer %q", create, quer)
	}
	if texto := lerConf(t, conf); !strings.Contains(texto, "password = "+obscuroComHifen+"\n") {
		t.Errorf("o rclone.conf deveria guardar o valor exato:\n%s", texto)
	}
}

// O falso recusa como o rclone real; sem isso o teste acima passaria mesmo
// sem o "--".
func TestRcloneFalsoRecusaValorComHifenSemSeparador(t *testing.T) {
	f := novoRcloneFalso(t)
	confFalso(t, "")
	cmd := exec.Command(f.exe, "config", "create", "cofre", "crypt", "password", obscuroComHifen)
	saida, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("o falso aceitou %q sem \"--\": %s", obscuroComHifen, saida)
	}
	if !strings.Contains(string(saida), "unknown shorthand flag: '3'") {
		t.Errorf("saída = %q", saida)
	}
}

// valorObscuroAleatorio tem o formato do `rclone obscure`: base64 de URL sem
// "=" de 16 bytes de vetor aleatório mais o texto cifrado.
func valorObscuroAleatorio(t *testing.T, tamanhoSenha int) string {
	t.Helper()
	b := make([]byte, 16+tamanhoSenha)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		t.Fatal(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// Muitas senhas no formato do obscure passam pelo mesmo caminho do argv até
// o rclone.conf, sem subir um processo por senha (o falso roda aqui mesmo).
func TestConfigCreateMuitasSenhasOfuscadas(t *testing.T) {
	conf := filepath.Join(t.TempDir(), "rclone.conf")
	comHifen := 0
	for i := 0; i < 500; i++ {
		valor := valorObscuroAleatorio(t, 1+i%40)
		if i%50 == 0 {
			valor = "-" + valor[1:] // garante casos com hífen
		}
		if strings.HasPrefix(valor, "-") {
			comHifen++
		}
		args := argsConfigCreate("cofre", "crypt", map[string]string{"remote": "x:", "password": valor, "password2": valor})
		posicionais, codigo, ok := opcoesConfigCreateFalso(args)
		if !ok {
			t.Fatalf("valor %q recusado (código %d), argv %q", valor, codigo, args)
		}
		if codigo, _ := configIniFalso(conf, posicionais); codigo != 0 {
			t.Fatalf("config create falso saiu com %d para %q", codigo, valor)
		}
		secoes := lerIni(conf)
		if len(secoes) != 1 || !contemLinha(secoes[0].linhas, "password = "+valor) {
			t.Fatalf("valor %q não gravado como está: %+v", valor, secoes)
		}
	}
	if comHifen < 10 {
		t.Fatalf("só %d valores com hífen: o teste não exercitou o caso", comHifen)
	}
}

func contemLinha(linhas []string, quer string) bool {
	for _, l := range linhas {
		if l == quer {
			return true
		}
	}
	return false
}

func TestArgsConfigCreateOrdemFixaEValoresVaziosFora(t *testing.T) {
	got := argsConfigCreate("b", "local", map[string]string{"remote": "", "z": "1", "a": "2"})
	quer := []string{"config", "create", "--", "b", "local", "a", "2", "z", "1"}
	if !reflect.DeepEqual(got, quer) {
		t.Errorf("got %q, quer %q", got, quer)
	}
}

// Com RCLONE_REAL apontando para um rclone de verdade, confere a mesma linha
// de comando contra ele. Fora disso (CI) o teste é pulado.
func TestConfigCreateComHifenNoRcloneReal(t *testing.T) {
	real := os.Getenv("RCLONE_REAL")
	if real == "" {
		t.Skip("RCLONE_REAL não definido")
	}
	conf := filepath.Join(t.TempDir(), "rclone.conf")
	args := argsConfigCreate("t", "crypt", map[string]string{"remote": t.TempDir(), "password": obscuroComHifen, "password2": obscuroComHifen})
	cmd := exec.Command(real, args...)
	cmd.Env = append(os.Environ(), "RCLONE_CONFIG="+conf, envFalsoAtivo+"=")
	if saida, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("rclone real recusou %q: %v\n%s", args, err, saida)
	}
	texto := lerConf(t, conf)
	if !strings.Contains(texto, "password = "+obscuroComHifen+"\n") {
		t.Errorf("rclone.conf do rclone real:\n%s", texto)
	}
}
