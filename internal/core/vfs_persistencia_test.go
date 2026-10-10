package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Testes da demanda 012: a configuração VFS vai para vfs.json, ao lado de
// vaults.json, e volta na próxima abertura.

func TestVfsAtualizarEReabrirDevolveOsValoresSalvos(t *testing.T) {
	dir := t.TempDir()
	cfg := NovoConfigVfsEm(dir)

	if _, err := cfg.Atualizar(map[string]string{"vfs_cache_max_age": "6h", "vfs_cache_mode": "writes"}); err != nil {
		t.Fatal(err)
	}
	reaberto := NovoConfigVfsEm(dir)

	v := reaberto.Obter()
	if v["vfs_cache_max_age"] != "6h" || v["vfs_cache_mode"] != "writes" {
		t.Errorf("depois de reabrir: vfs_cache_max_age=%q vfs_cache_mode=%q", v["vfs_cache_max_age"], v["vfs_cache_mode"])
	}
	if len(reaberto.Avisos()) != 0 {
		t.Errorf("sem avisos esperados: %v", reaberto.Avisos())
	}
	if _, err := os.Stat(filepath.Join(dir, ArquivoVfs)); err != nil {
		t.Errorf("vfs.json deveria estar ao lado de vaults.json: %v", err)
	}
}

func TestVfsArquivoComValorInvalidoUsaPadraoEAvisa(t *testing.T) {
	dir := t.TempDir()
	dados, _ := json.Marshal(map[string]string{
		"vfs_cache_mode":    "tudo",
		"vfs_cache_max_age": "6h",
		"chave_estranha":    "x",
	})
	if err := os.WriteFile(filepath.Join(dir, ArquivoVfs), dados, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := NovoConfigVfsEm(dir)

	v := cfg.Obter()
	if v["vfs_cache_mode"] != ConfiguracoesVfsPadrao["vfs_cache_mode"] {
		t.Errorf("valor inválido deveria voltar ao padrão; vfs_cache_mode=%q", v["vfs_cache_mode"])
	}
	if v["vfs_cache_max_age"] != "6h" {
		t.Errorf("o valor válido do arquivo deveria valer; vfs_cache_max_age=%q", v["vfs_cache_max_age"])
	}
	avisos := strings.Join(cfg.Avisos(), "\n")
	if !strings.Contains(avisos, "vfs_cache_mode") || !strings.Contains(avisos, "tudo") || !strings.Contains(avisos, "chave_estranha") {
		t.Errorf("avisos = %q", avisos)
	}
}

func TestVfsArquivoIlegivelUsaPadraoEAvisa(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ArquivoVfs), []byte("{nao é json"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := NovoConfigVfsEm(dir)

	if cfg.Obter()["vfs_cache_mode"] != ConfiguracoesVfsPadrao["vfs_cache_mode"] || len(cfg.Avisos()) != 1 {
		t.Errorf("config=%v avisos=%v", cfg.Obter(), cfg.Avisos())
	}
}

func TestVfsSemArquivoUsaPadraoSemAviso(t *testing.T) {
	cfg := NovoConfigVfsEm(t.TempDir())
	if len(cfg.Avisos()) != 0 || cfg.Obter()["vfs_cache_mode"] != ConfiguracoesVfsPadrao["vfs_cache_mode"] {
		t.Errorf("config=%v avisos=%v", cfg.Obter(), cfg.Avisos())
	}
}

// Se a gravação falha, nada muda em memória e o erro volta.
func TestVfsFalhaAoGravarNaoAplica(t *testing.T) {
	dir := t.TempDir()
	cfg := NovoConfigVfsEm(dir)
	original := renomearArquivo
	renomearArquivo = func(string, string) error { return os.ErrPermission }
	defer func() { renomearArquivo = original }()

	_, err := cfg.Atualizar(map[string]string{"vfs_cache_mode": "writes"})

	if err == nil {
		t.Fatal("esperava erro de gravação")
	}
	if v := cfg.Obter()["vfs_cache_mode"]; v != ConfiguracoesVfsPadrao["vfs_cache_mode"] {
		t.Errorf("sem gravar, a memória não pode mudar: %q", v)
	}
}

func TestVfsRestaurarGrava(t *testing.T) {
	dir := t.TempDir()
	cfg := NovoConfigVfsEm(dir)
	if _, err := cfg.Atualizar(map[string]string{"vfs_cache_mode": "writes"}); err != nil {
		t.Fatal(err)
	}

	if _, err := cfg.Restaurar(); err != nil {
		t.Fatal(err)
	}

	if v := NovoConfigVfsEm(dir).Obter()["vfs_cache_mode"]; v != ConfiguracoesVfsPadrao["vfs_cache_mode"] {
		t.Errorf("depois de Restaurar e reabrir: %q", v)
	}
}

func TestNovoGerenciadorLeVfsDoDiretorioDoApp(t *testing.T) {
	f := novoRcloneFalso(t)
	dir := t.TempDir()
	if _, err := NovoGerenciadorEm(dir, f.exe).Vfs.Atualizar(map[string]string{"dir_cache_time": "10m"}); err != nil {
		t.Fatal(err)
	}

	if v := NovoGerenciadorEm(dir, f.exe).Vfs.Obter()["dir_cache_time"]; v != "10m" {
		t.Errorf("dir_cache_time = %q", v)
	}
}
