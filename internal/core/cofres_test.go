package core

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Testes da demanda 007: vaults.json não é sobrescrito em silêncio.

func TestVaultsJsonCorrompidoNaoESobrescrito(t *testing.T) {
	dir := t.TempDir()
	caminho := filepath.Join(dir, ArquivoCofres)
	original := []byte("{quebrado")
	if err := os.WriteFile(caminho, original, 0o644); err != nil {
		t.Fatal(err)
	}

	g, err := NovoGerenciadorCofres(dir)

	if err == nil {
		t.Fatal("NovoGerenciadorCofres deveria devolver erro com JSON inválido")
	}
	if g == nil || g.ErroCarga() == nil {
		t.Fatal("o gerenciador deveria vir com ErroCarga preenchido")
	}
	if ok, msg := g.Adicionar("novo", "drive", "Google Drive", "novo_base:"); ok {
		t.Errorf("Adicionar deveria recusar: %q", msg)
	}
	if ok, _ := g.Remover("x"); ok {
		t.Error("Remover deveria recusar")
	}
	depois, errLeitura := os.ReadFile(caminho)
	if errLeitura != nil {
		t.Fatal(errLeitura)
	}
	if !bytes.Equal(depois, original) {
		t.Errorf("vaults.json mudou: %q", depois)
	}
	copias, _ := filepath.Glob(caminho + ".corrompido-*")
	if len(copias) != 1 {
		t.Fatalf("esperava uma cópia preservada, achei %v", copias)
	}
	if dados, _ := os.ReadFile(copias[0]); !bytes.Equal(dados, original) {
		t.Errorf("a cópia não tem o conteúdo original: %q", dados)
	}
	if !strings.Contains(err.Error(), copias[0]) {
		t.Errorf("a mensagem deveria dizer onde está a cópia: %v", err)
	}
}

func TestVaultsJsonAusenteViraListaVazia(t *testing.T) {
	dir := t.TempDir()

	g, err := NovoGerenciadorCofres(dir)

	if err != nil {
		t.Fatalf("arquivo ausente não é erro: %v", err)
	}
	if n := len(g.Listar(nil, NovoCacheSenhas())); n != 0 {
		t.Errorf("lista deveria vir vazia, veio %d", n)
	}
	if ok, msg := g.Adicionar("a", "drive", "Google Drive", "a_base:"); !ok {
		t.Fatalf("Adicionar: %s", msg)
	}
	g2, err := NovoGerenciadorCofres(dir)
	if err != nil || g2.Obter("a") == nil {
		t.Errorf("o cofre gravado deveria voltar na leitura: err=%v", err)
	}
}

func TestAtualizarEmDiretorioSomenteLeituraDevolveErro(t *testing.T) {
	dir := t.TempDir()
	g, err := NovoGerenciadorCofres(dir)
	if err != nil {
		t.Fatal(err)
	}
	if ok, msg := g.Adicionar("a", "drive", "Google Drive", "a_base:"); !ok {
		t.Fatal(msg)
	}
	antes, _ := os.ReadFile(filepath.Join(dir, ArquivoCofres))

	somenteLeitura(t, dir)
	err = g.Atualizar("a", map[string]interface{}{"auto_montar": true})

	if err == nil {
		t.Fatal("Atualizar deveria devolver erro em diretório somente leitura")
	}
	if g.Obter("a").AutoMontar {
		t.Error("com a gravação falhando, a memória deveria voltar ao valor anterior")
	}
	depois, _ := os.ReadFile(filepath.Join(dir, ArquivoCofres))
	if !bytes.Equal(antes, depois) {
		t.Error("vaults.json mudou apesar do erro")
	}
}

func TestAtualizarCofreInexistente(t *testing.T) {
	g, _ := NovoGerenciadorCofres(t.TempDir())
	if err := g.Atualizar("nao", map[string]interface{}{"auto_montar": true}); !errors.Is(err, ErrCofreNaoEncontrado) {
		t.Errorf("err = %v", err)
	}
}

func TestAtualizarGrava(t *testing.T) {
	dir := t.TempDir()
	g, _ := NovoGerenciadorCofres(dir)
	g.Adicionar("a", "drive", "Google Drive", "a_base:")

	if err := g.Atualizar("a", map[string]interface{}{"auto_montar": true, "caminho_cripto": "x"}); err != nil {
		t.Fatal(err)
	}

	g2, _ := NovoGerenciadorCofres(dir)
	if c := g2.Obter("a"); c == nil || !c.AutoMontar || c.CaminhoCripto != "x" {
		t.Errorf("releitura = %+v", c)
	}
}

func TestSalvarNaoDeixaTemporario(t *testing.T) {
	dir := t.TempDir()
	g, _ := NovoGerenciadorCofres(dir)
	g.Adicionar("a", "drive", "Google Drive", "a_base:")
	g.Adicionar("b", "drive", "Google Drive", "b_base:")

	entradas, _ := os.ReadDir(dir)
	if len(entradas) != 1 || entradas[0].Name() != ArquivoCofres {
		var nomes []string
		for _, e := range entradas {
			nomes = append(nomes, e.Name())
		}
		t.Errorf("a pasta deveria ter só %s, tem %v", ArquivoCofres, nomes)
	}
}

func TestVaultsJsonSemPermissaoDeLeitura(t *testing.T) {
	dir := t.TempDir()
	// Um diretório no lugar do arquivo: ReadFile falha com erro que não é
	// "não existe", em qualquer sistema.
	if err := os.Mkdir(filepath.Join(dir, ArquivoCofres), 0o755); err != nil {
		t.Fatal(err)
	}

	g, err := NovoGerenciadorCofres(dir)

	if err == nil {
		t.Fatal("erro de leitura que não é 'não existe' deveria voltar")
	}
	if ok, _ := g.Adicionar("a", "drive", "Google Drive", "a_base:"); ok {
		t.Error("Adicionar deveria recusar")
	}
}
