package core

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLogVaiParaOArquivoETrocaNoLimite(t *testing.T) {
	dir := t.TempDir()
	caminho := filepath.Join(dir, NomeArquivoLog)
	a, err := abrirArquivoLog(caminho, 100)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()

	a.Write([]byte(strings.Repeat("a", 60) + "\n"))
	a.Write([]byte(strings.Repeat("b", 60) + "\n")) // passa de 100: troca

	atual, _ := os.ReadFile(caminho)
	antigo, _ := os.ReadFile(caminho + ".1")
	if !strings.HasPrefix(string(atual), "bbb") || !strings.HasPrefix(string(antigo), "aaa") {
		t.Errorf("atual=%q antigo=%q", atual, antigo)
	}
}

func TestAbrirLogDesviaOLogDoApp(t *testing.T) {
	antes := log.Writer()
	t.Cleanup(func() { log.SetOutput(antes) })
	dir := t.TempDir()
	c, err := AbrirLog(dir)
	if err != nil {
		t.Fatal(err)
	}
	novoErroRclone("mount", "CRITICAL: texto original do rclone", nil)
	c.Close()
	log.SetOutput(antes)

	dados, _ := os.ReadFile(filepath.Join(dir, NomeArquivoLog))
	if !strings.Contains(string(dados), "rclone mount: CRITICAL: texto original do rclone") {
		t.Errorf("log = %q", dados)
	}
}

// Demanda 027: se a pasta do vaults.json não aceita escrita, AbrirLog só
// devolve o erro: o log do app não muda de destino (não há outra pasta de
// reserva) e quem chama segue sem arquivo de log.
func TestAbrirLogEmPastaSemEscritaNaoTrocaOLog(t *testing.T) {
	arquivo := filepath.Join(t.TempDir(), "nao-e-pasta")
	if err := os.WriteFile(arquivo, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	antes := log.Writer()
	t.Cleanup(func() { log.SetOutput(antes) })

	fechar, err := AbrirLog(filepath.Join(arquivo, "config"))

	if err == nil {
		fechar.Close()
		t.Fatal("deveria falhar: a pasta não pode ser criada")
	}
	if log.Writer() != antes {
		t.Error("o log não deveria mudar de destino")
	}
}

// O log fica na mesma pasta do vaults.json (o DiretorioApp do gerenciador).
func TestLogFicaAoLadoDoVaultsJson(t *testing.T) {
	dir := t.TempDir()
	g := NovoGerenciadorEm(dir, "rclone-que-nao-existe")
	antes := log.Writer()
	t.Cleanup(func() { log.SetOutput(antes) })
	fechar, err := AbrirLog(g.DiretorioApp)
	if err != nil {
		t.Fatal(err)
	}
	log.Print("teste")
	fechar.Close()
	if g.Cofres.caminhoArquivo() != filepath.Join(dir, ArquivoCofres) {
		t.Fatalf("vaults.json em %s", g.Cofres.caminhoArquivo())
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(g.Cofres.caminhoArquivo()), NomeArquivoLog)); err != nil {
		t.Errorf("o log não está ao lado do vaults.json: %v", err)
	}
}
