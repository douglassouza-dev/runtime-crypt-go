package core

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Testes da demanda 006.

// confFalso cria um rclone.conf temporário e aponta RCLONE_CONFIG para ele.
func confFalso(t *testing.T, conteudo string) string {
	t.Helper()
	caminho := filepath.Join(t.TempDir(), "rclone.conf")
	if err := os.WriteFile(caminho, []byte(conteudo), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("RCLONE_CONFIG", caminho)
	return caminho
}

func houveConfigCreate(f *rcloneFalso) bool {
	for _, c := range f.chamadas() {
		if argsContem(c.Args, "config", "create") {
			return true
		}
	}
	return false
}

func TestCriarCofreComNomeDeRemotoExistenteNaoTocaNoConf(t *testing.T) {
	for _, existente := range []string{"teste", "teste_base", "TESTE"} {
		t.Run(existente, func(t *testing.T) {
			g, f := novoGerenciadorFalso(t)
			conteudo := "[" + existente + "]\ntype = crypt\nremote = gdrive:cofre\npassword = senha-antiga\n\n[gdrive]\ntype = drive\n"
			conf := confFalso(t, conteudo)

			c, err := g.IniciarCriacaoCofre("teste")

			if err == nil || c != nil {
				t.Fatal("criar cofre 'teste' deveria falhar")
			}
			if !strings.Contains(err.Error(), existente) {
				t.Errorf("a mensagem deveria citar o remoto existente: %v", err)
			}
			depois, _ := os.ReadFile(conf)
			if !bytes.Equal(depois, []byte(conteudo)) {
				t.Errorf("rclone.conf mudou:\n%s", depois)
			}
			if houveConfigCreate(f) {
				t.Error("nenhum `config create` deveria ter rodado")
			}
		})
	}
}

func TestCriarCofreComNomeDeCofreExistente(t *testing.T) {
	g, f := novoGerenciadorFalso(t)
	confFalso(t, "")
	if ok, msg := g.Cofres.Adicionar("teste", "drive", "Google Drive", "teste_base:"); !ok {
		t.Fatal(msg)
	}

	if _, err := g.IniciarCriacaoCofre("teste"); err == nil {
		t.Fatal("deveria recusar nome de cofre que já está em vaults.json")
	}
	if len(f.chamadas()) != 0 {
		t.Errorf("não deveria chamar o rclone: %+v", f.chamadas())
	}
}

func TestFalhaEmCriarCryptNaoDeixaRemotoBase(t *testing.T) {
	g, _ := novoGerenciadorFalso(t)
	conf := confFalso(t, "[gdrive]\ntype = drive\n")
	t.Setenv(envFalsoFalhaTipo, "crypt")

	c, err := g.IniciarCriacaoCofre("teste")
	if err != nil {
		t.Fatal(err)
	}
	if ok, msg := c.CriarRemoto("teste_base", "local", map[string]string{"remote": "x"}); !ok {
		t.Fatal(msg)
	}
	if dados, _ := os.ReadFile(conf); !strings.Contains(string(dados), "[teste_base]") {
		t.Fatalf("o passo 1 deveria ter criado teste_base:\n%s", dados)
	}
	if ok, msg := c.CriarCrypt("teste_base:", "s", "", nil); ok || !strings.Contains(msg, "falha simulada") {
		t.Fatalf("CriarCrypt deveria falhar: ok=%v msg=%q", ok, msg)
	}

	c.Desfazer()

	dados, _ := os.ReadFile(conf)
	if strings.Contains(string(dados), "teste_base") || strings.Contains(string(dados), "[teste]") {
		t.Errorf("rclone.conf ficou com remoto da tentativa:\n%s", dados)
	}
	if !strings.Contains(string(dados), "[gdrive]") {
		t.Errorf("o remoto que já existia sumiu:\n%s", dados)
	}
}

func TestConcluirFalhoDesfazOsRemotos(t *testing.T) {
	g, _ := novoGerenciadorFalso(t)
	conf := confFalso(t, "")

	c, err := g.IniciarCriacaoCofre("teste")
	if err != nil {
		t.Fatal(err)
	}
	c.CriarRemoto("teste_base", "local", map[string]string{"remote": "x"})
	c.CriarCrypt("teste_base:", "s", "", nil)
	// Outra janela grava um cofre com o mesmo nome no meio da tentativa.
	g.Cofres.Adicionar("teste", "drive", "Google Drive", "outro:")

	if ok, _ := c.Concluir("drive", "Google Drive", "teste_base:"); ok {
		t.Fatal("Concluir deveria falhar com o nome já em vaults.json")
	}
	if dados, _ := os.ReadFile(conf); strings.Contains(string(dados), "teste") {
		t.Errorf("Concluir falho deveria desfazer os remotos:\n%s", dados)
	}
}

func TestConcluirComSucessoNaoDesfaz(t *testing.T) {
	g, _ := novoGerenciadorFalso(t)
	conf := confFalso(t, "")

	c, _ := g.IniciarCriacaoCofre("teste")
	c.CriarRemoto("teste_base", "local", map[string]string{"remote": "x"})
	c.CriarCrypt("teste_base:", "s", "", nil)
	if ok, msg := c.Concluir("local_path", "Pasta Local", "teste_base:"); !ok {
		t.Fatal(msg)
	}
	c.Desfazer() // o defer de main.go faz isto

	dados, _ := os.ReadFile(conf)
	if !strings.Contains(string(dados), "[teste_base]") || !strings.Contains(string(dados), "[teste]") {
		t.Errorf("os remotos de um cofre concluído não podem sumir:\n%s", dados)
	}
}

func TestCriacaoSoCriaRemotosDoProprioCofre(t *testing.T) {
	g, f := novoGerenciadorFalso(t)
	confFalso(t, "")
	c, _ := g.IniciarCriacaoCofre("teste")

	if ok, _ := c.CriarRemoto("gdrive", "drive", nil); ok {
		t.Error("a tentativa do cofre 'teste' não pode criar o remoto 'gdrive'")
	}
	if houveConfigCreate(f) {
		t.Error("nenhum `config create` deveria ter rodado")
	}
}

func TestNomesInvalidosRecusadosAntesDoRclone(t *testing.T) {
	for _, nome := range []string{"a:b", "a/b", `a\b`, "", " teste", "teste ", "-teste", "a:", "/"} {
		t.Run(nome, func(t *testing.T) {
			g, f := novoGerenciadorFalso(t)

			if _, err := g.IniciarCriacaoCofre(nome); err == nil {
				t.Errorf("nome %q deveria ser recusado", nome)
			}
			if len(f.chamadas()) != 0 {
				t.Errorf("o rclone não deveria ser chamado: %+v", f.chamadas())
			}
		})
	}
}

func TestNomesValidos(t *testing.T) {
	for _, nome := range []string{"teste", "Meu Cofre", "fotos-2024", "ação", "a.b+c@d", "_x", "2024"} {
		if err := ValidarNomeCofre(nome); err != nil {
			t.Errorf("nome %q deveria valer: %v", nome, err)
		}
	}
}

func TestIniciarCriacaoSemConseguirListarRemotosRecusa(t *testing.T) {
	g, f := novoGerenciadorFalso(t)
	f.falhar()

	if _, err := g.IniciarCriacaoCofre("teste"); err == nil {
		t.Fatal("sem conseguir conferir os remotos, a criação não pode seguir")
	}
	if houveConfigCreate(f) {
		t.Error("nenhum `config create` deveria ter rodado")
	}
}
