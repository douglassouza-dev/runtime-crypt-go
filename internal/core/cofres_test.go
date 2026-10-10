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

// lerOuFalhar lê o arquivo e falha o teste se não conseguir (para uma falha
// de leitura não parecer "o arquivo mudou").
func lerOuFalhar(t *testing.T, caminho string) []byte {
	t.Helper()
	dados, err := os.ReadFile(caminho)
	if err != nil {
		t.Fatalf("ler %s: %v", caminho, err)
	}
	return dados
}

// conferirAtualizarFalha confere o contrato de Atualizar quando a gravação
// falha: devolve erro, a memória volta e vaults.json fica igual.
func conferirAtualizarFalha(t *testing.T, g *GerenciadorCofres, dir string, antes []byte) {
	t.Helper()
	err := g.Atualizar("a", map[string]interface{}{"auto_montar": true})

	if err == nil {
		t.Fatal("Atualizar deveria devolver erro quando não consegue gravar")
	}
	if g.Obter("a").AutoMontar {
		t.Error("com a gravação falhando, a memória deveria voltar ao valor anterior")
	}
	if depois := lerOuFalhar(t, filepath.Join(dir, ArquivoCofres)); !bytes.Equal(antes, depois) {
		t.Errorf("vaults.json mudou apesar do erro:\nantes  %q\ndepois %q", antes, depois)
	}
}

func novoCofresComUm(t *testing.T) (*GerenciadorCofres, string, []byte) {
	t.Helper()
	dir := t.TempDir()
	g, err := NovoGerenciadorCofres(dir)
	if err != nil {
		t.Fatal(err)
	}
	if ok, msg := g.Adicionar("a", "drive", "Google Drive", "a_base:"); !ok {
		t.Fatal(msg)
	}
	return g, dir, lerOuFalhar(t, filepath.Join(dir, ArquivoCofres))
}

// Vale em Linux e Windows: a troca do arquivo temporário pelo final falha
// como falharia numa pasta sem permissão de escrita.
func TestAtualizarSemConseguirGravarDevolveErro(t *testing.T) {
	g, dir, antes := novoCofresComUm(t)
	original := renomearArquivo
	renomearArquivo = func(_, _ string) error { return os.ErrPermission }
	t.Cleanup(func() { renomearArquivo = original })

	conferirAtualizarFalha(t, g, dir, antes)

	entradas, _ := os.ReadDir(dir)
	if len(entradas) != 1 {
		t.Errorf("o temporário deveria ter sido apagado; a pasta tem %d entradas", len(entradas))
	}
}

// Vale em Linux e Windows: a pasta do app sumiu e um arquivo comum ficou no
// lugar dela, então não dá para criar o temporário.
func TestAtualizarComPastaInvalidaDevolveErro(t *testing.T) {
	g, dir, antes := novoCofresComUm(t)
	g.diretorioApp = filepath.Join(dir, ArquivoCofres) // um arquivo, não uma pasta

	err := g.Atualizar("a", map[string]interface{}{"auto_montar": true})

	if err == nil {
		t.Fatal("Atualizar deveria devolver erro")
	}
	if depois := lerOuFalhar(t, filepath.Join(dir, ArquivoCofres)); !bytes.Equal(antes, depois) {
		t.Errorf("vaults.json mudou apesar do erro")
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

// Demanda 024: abrir o gerenciador três vezes com o mesmo vaults.json
// ilegível deixa uma cópia .corrompido só.
func TestMesmoVaultsJsonIlegivelGuardaUmaCopiaSo(t *testing.T) {
	dir := t.TempDir()
	caminho := filepath.Join(dir, ArquivoCofres)
	if err := os.WriteFile(caminho, []byte("{ilegível"), 0o644); err != nil {
		t.Fatal(err)
	}

	var mensagens []string
	for i := 0; i < 3; i++ {
		_, err := NovoGerenciadorCofres(dir)
		if err == nil {
			t.Fatalf("abertura %d: esperava erro de arquivo corrompido", i+1)
		}
		mensagens = append(mensagens, err.Error())
	}

	copias, _ := filepath.Glob(caminho + ".corrompido-*")
	if len(copias) != 1 {
		t.Fatalf("esperava uma cópia só, achei %v", copias)
	}
	for i, m := range mensagens {
		if !strings.Contains(m, copias[0]) {
			t.Errorf("abertura %d deveria apontar a cópia que já existe: %q", i+1, m)
		}
	}
}

// Demanda 024: conteúdo ilegível diferente ganha cópia nova, mesmo no mesmo
// segundo.
func TestVaultsJsonIlegivelDiferenteGuardaCopiaNova(t *testing.T) {
	dir := t.TempDir()
	caminho := filepath.Join(dir, ArquivoCofres)

	for _, conteudo := range []string{"{primeiro", "{segundo"} {
		if err := os.WriteFile(caminho, []byte(conteudo), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := NovoGerenciadorCofres(dir); err == nil {
			t.Fatal("esperava erro de arquivo corrompido")
		}
	}

	copias, _ := filepath.Glob(caminho + ".corrompido-*")
	if len(copias) != 2 {
		t.Fatalf("esperava duas cópias, achei %v", copias)
	}
	vistos := map[string]bool{}
	for _, c := range copias {
		dados, _ := os.ReadFile(c)
		vistos[string(dados)] = true
	}
	if !vistos["{primeiro"] || !vistos["{segundo"] {
		t.Errorf("cada conteúdo deveria ter a sua cópia; achei %v", vistos)
	}
}
