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
