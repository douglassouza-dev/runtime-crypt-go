package core

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// Testes da demanda 030: senha errada não destranca.

// Valores gerados no box pelo rclone real (`printf '<senha>' | rclone obscure -`).
var obscurosDoRcloneReal = []struct {
	versao, obscuro, senha string
}{
	{"v1.60.1", "YfV_ZF3fQK-3h1cbc_es0t0-9bfYKSUaqqqL", "senha certa"},
	{"v1.75.2", "E30F4zW8waPvZfsG7oNoP-ygFeyVNGja2VLo", "senha certa"},
	{"v1.60.1, começa com hífen", "-0npvSMCM3iD3Hj8Kncu75bPm259ehsYDHgfZvwABRkM", "çãé ünï 🔐"},
}

func TestRevelarObscuroIgualAoRcloneReal(t *testing.T) {
	for _, c := range obscurosDoRcloneReal {
		got, err := revelarObscuro(c.obscuro)
		if err != nil || got != c.senha {
			t.Errorf("%s: revelarObscuro(%q) = %q, %v; quer %q", c.versao, c.obscuro, got, err, c.senha)
		}
	}
}

func TestRevelarObscuroRecusaValorInvalido(t *testing.T) {
	for _, v := range []string{"", "curto", "não é base64!", "obs(senha)"} {
		if _, err := revelarObscuro(v); err == nil {
			t.Errorf("revelarObscuro(%q) deveria falhar", v)
		}
	}
}

// cofreComSenha prepara o rclone falso com o remoto crypt "cofre" cuja
// senha gravada é "senha certa" (valor do rclone v1.75.2).
func cofreComSenha(t *testing.T) (*GerenciadorRClone, *rcloneFalso) {
	t.Helper()
	g, f := novoGerenciadorFalso(t)
	f.escrever("dump.json", fmt.Sprintf(`{"cofre":{"type":"crypt","remote":"cofre_base:","password":%q,"password2":%q},"cofre_base":{"type":"drive"}}`,
		obscurosDoRcloneReal[1].obscuro, obscurosDoRcloneReal[1].obscuro))
	if ok, msg := g.Cofres.Adicionar("cofre", "drive", "Google Drive", "cofre_base:"); !ok {
		t.Fatal(msg)
	}
	t.Cleanup(g.Montagens.DesmontarTodas)
	return g, f
}

func chamadasMount(f *rcloneFalso) []chamadaFalsa {
	var ms []chamadaFalsa
	for _, c := range f.chamadas() {
		if len(c.Args) > 0 && c.Args[0] == "mount" {
			ms = append(ms, c)
		}
	}
	return ms
}

func TestDestrancarComSenhaCertaMonta(t *testing.T) {
	g, f := cofreComSenha(t)

	ponto, err := g.Destrancar("cofre", func() (string, bool) { return "senha certa", true })

	if err != nil {
		t.Fatalf("Destrancar: %v", err)
	}
	if ponto == "" || len(chamadasMount(f)) != 1 {
		t.Errorf("ponto = %q, mounts = %d", ponto, len(chamadasMount(f)))
	}
	if e := g.EstadoDoCofre("cofre"); e.Estado != EstadoMontado {
		t.Errorf("estado = %+v, quer montado", e)
	}
	if g.Senhas.Obter("cofre") != "senha certa" {
		t.Error("a senha conferida deveria ficar na sessão")
	}
}

func TestDestrancarComSenhaErradaNaoMonta(t *testing.T) {
	for _, errada := range []string{"senha errada", "senha cert", "senha certa ", "SENHA CERTA"} {
		t.Run(errada, func(t *testing.T) {
			g, f := cofreComSenha(t)

			_, err := g.Destrancar("cofre", func() (string, bool) { return errada, true })

			if !errors.Is(err, ErrSenhaErrada) {
				t.Fatalf("erro = %v, quer ErrSenhaErrada", err)
			}
			if err.Error() != "senha errada" {
				t.Errorf("frase = %q", err.Error())
			}
			if n := len(chamadasMount(f)); n != 0 {
				t.Errorf("houve %d mount com senha errada", n)
			}
			for _, c := range f.chamadas() {
				if processoVivoNoSO(c.Pid) {
					t.Errorf("rclone %v ficou vivo", c.Args)
				}
			}
			if e := g.EstadoDoCofre("cofre"); e.Estado != EstadoDesmontado || e.Motivo != "" {
				t.Errorf("estado = %+v, quer desmontado sem motivo (Trancado)", e)
			}
			if g.Senhas.Existe("cofre") {
				t.Error("a senha errada não pode ficar na sessão")
			}
		})
	}
}

func TestDestrancarComSenhaVaziaNaoChamaORclone(t *testing.T) {
	g, f := cofreComSenha(t)
	antes := len(f.chamadas())

	_, err := g.Destrancar("cofre", func() (string, bool) { return "", true })

	if !errors.Is(err, ErrCancelado) {
		t.Fatalf("erro = %v, quer ErrCancelado", err)
	}
	if len(f.chamadas()) != antes {
		t.Error("senha vazia não deveria chamar o rclone")
	}
	if err := g.ConferirSenha("cofre", ""); !errors.Is(err, ErrSenhaErrada) {
		t.Errorf("ConferirSenha vazia = %v", err)
	}
}

func TestDestrancarSemComoConferirNaoMonta(t *testing.T) {
	casos := map[string]string{
		"remoto inexistente": `{"outro":{"type":"crypt","password":"x"}}`,
		"crypt sem password": `{"cofre":{"type":"crypt","remote":"cofre_base:"}}`,
		"password ilegível":  `{"cofre":{"type":"crypt","remote":"cofre_base:","password":"obs(senha certa)"}}`,
		"não é crypt":        `{"cofre":{"type":"drive"}}`,
	}
	for nome, dump := range casos {
		t.Run(nome, func(t *testing.T) {
			g, f := novoGerenciadorFalso(t)
			f.escrever("dump.json", dump)

			_, err := g.Destrancar("cofre", func() (string, bool) { return "senha certa", true })

			var e *ErroConferirSenha
			if !errors.As(err, &e) {
				t.Fatalf("erro = %v, quer *ErroConferirSenha", err)
			}
			if err.Error() != TextoNaoConferiuSenha {
				t.Errorf("frase = %q", err.Error())
			}
			if len(chamadasMount(f)) != 0 {
				t.Error("não deveria montar")
			}
			if g.Senhas.Existe("cofre") {
				t.Error("a senha não pode ficar na sessão")
			}
		})
	}
}

func TestDestrancarComConfigDumpFalhandoNaoMonta(t *testing.T) {
	g, f := cofreComSenha(t)
	f.falhar()

	_, err := g.Destrancar("cofre", func() (string, bool) { return "senha certa", true })

	var e *ErroConferirSenha
	if !errors.As(err, &e) {
		t.Fatalf("erro = %v, quer *ErroConferirSenha", err)
	}
	if len(chamadasMount(f)) != 0 {
		t.Error("não deveria montar")
	}
}

// Sem limite de tentativas: depois de errar, a certa destranca.
func TestDestrancarDepoisDeErrarComASenhaCerta(t *testing.T) {
	g, f := cofreComSenha(t)
	for i := 0; i < 5; i++ {
		if _, err := g.Destrancar("cofre", func() (string, bool) { return "errada", true }); !errors.Is(err, ErrSenhaErrada) {
			t.Fatalf("tentativa %d: %v", i, err)
		}
	}
	if _, err := g.Destrancar("cofre", func() (string, bool) { return "senha certa", true }); err != nil {
		t.Fatalf("senha certa depois de errar: %v", err)
	}
	if len(chamadasMount(f)) != 1 {
		t.Errorf("mounts = %d, quer 1", len(chamadasMount(f)))
	}
}

// Com RCLONE_REAL apontando para um rclone de verdade: o cofre é criado pelo
// rclone real (obscure + config create) e a conferência lê o `config dump`
// dele. Fora disso (CI) o teste é pulado.
func TestConferirSenhaComORcloneReal(t *testing.T) {
	real := os.Getenv("RCLONE_REAL")
	if real == "" {
		t.Skip("RCLONE_REAL não definido")
	}
	t.Setenv("RCLONE_CONFIG", filepath.Join(t.TempDir(), "rclone.conf"))
	g := NovoGerenciadorEm(t.TempDir(), real)
	// Sem a 029, 1 em 64 criações falha com senha ofuscada começando em "-".
	var msg string
	ok := false
	for i := 0; i < 5 && !ok; i++ {
		ok, msg = g.CriarCrypt("cofre", t.TempDir(), "senha certa", "", nil)
	}
	if !ok {
		t.Fatalf("CriarCrypt com o rclone real: %s", msg)
	}
	if err := g.ConferirSenha("cofre", "senha certa"); err != nil {
		t.Errorf("senha certa: %v", err)
	}
	if err := g.ConferirSenha("cofre", "senha errada"); !errors.Is(err, ErrSenhaErrada) {
		t.Errorf("senha errada: %v", err)
	}
}
