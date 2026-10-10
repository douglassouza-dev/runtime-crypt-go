package core

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigVfs(t *testing.T) {
	cfg := NovoConfigVfs()

	// Verificar se os valores padrão foram carregados
	valores := cfg.Obter()
	for k, v := range ConfiguracoesVfsPadrao {
		if valores[k] != v {
			t.Errorf("Esperava valor padrão '%s' para chave '%s', mas obteve '%s'", v, k, valores[k])
		}
	}

	// Testar atualização com chave válida
	if _, err := cfg.Atualizar(map[string]string{"vfs_cache_mode": "writes"}); err != nil {
		t.Fatalf("Atualizar: %v", err)
	}

	valores = cfg.Obter()
	if valores["vfs_cache_mode"] != "writes" {
		t.Errorf("Esperava vfs_cache_mode atualizado para 'writes', mas obteve '%s'", valores["vfs_cache_mode"])
	}

	// Testar geração de argumentos CLI
	args := cfg.ConstruirArgs(nil)
	temCacheMode := false
	for i, arg := range args {
		if arg == "--vfs-cache-mode" && i+1 < len(args) && args[i+1] == "writes" {
			temCacheMode = true
			break
		}
	}
	if !temCacheMode {
		t.Errorf("Esperava encontrar '--vfs-cache-mode writes' nos argumentos CLI gerados: %v", args)
	}

	// Testar restauração de padrões
	cfg.Restaurar()
	valores = cfg.Obter()
	if valores["vfs_cache_mode"] != ConfiguracoesVfsPadrao["vfs_cache_mode"] {
		t.Errorf("Esperava vfs_cache_mode restaurado para padrão '%s', mas obteve '%s'", ConfiguracoesVfsPadrao["vfs_cache_mode"], valores["vfs_cache_mode"])
	}
}

// Demanda 011: cada uma das 13 chaves com pelo menos um valor válido e um
// inválido.
func TestValidarVfsTabela(t *testing.T) {
	pastaBoa := t.TempDir()
	arquivo := filepath.Join(t.TempDir(), "arquivo.txt")
	if err := os.WriteFile(arquivo, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	casos := []struct {
		chave     string
		validos   []string
		invalidos []string
	}{
		{"vfs_cache_mode", []string{"off", "minimal", "writes", "full"}, []string{"tudo", "FULL", "", "full "}},
		{"vfs_cache_max_size", []string{"10G", "512M", "1.5G", "10GiB", "off", "1024"}, []string{"10 G", "dez", "10X", "", "-1G"}},
		{"vfs_cache_max_age", []string{"1h", "30m", "1h30m", "2d", "off", "3600"}, []string{"1 hora", "1x", "h", ""}},
		{"vfs_read_chunk_size", []string{"8M", "128k"}, []string{"8MB/s", ""}},
		{"vfs_read_chunk_size_limit", []string{"512M", "off", "0"}, []string{"ilimitado", ""}},
		{"vfs_read_ahead", []string{"16M", "0"}, []string{"16 M", ""}},
		{"buffer_size", []string{"16M", "1G"}, []string{"grande", ""}},
		{"dir_cache_time", []string{"30m", "5m0s"}, []string{"30 min", ""}},
		{"poll_interval", []string{"30s", "0", "1m", "30"}, []string{"30 s", ""}},
		{"attr_timeout", []string{"1m", "500ms"}, []string{"1min", ""}},
		{"vfs_write_back", []string{"5s", "1m"}, []string{"cinco", ""}},
		{"vfs_disk_space_total_size", []string{"1T", "500G"}, []string{"1 tera", ""}},
		{"cache_dir", []string{"", pastaBoa}, []string{filepath.Join(pastaBoa, "nao-existe"), arquivo}},
	}
	if len(casos) != len(ConfiguracoesVfsPadrao) {
		t.Fatalf("a tabela cobre %d chaves; ConfiguracoesVfsPadrao tem %d", len(casos), len(ConfiguracoesVfsPadrao))
	}
	for _, c := range casos {
		if _, ok := ConfiguracoesVfsPadrao[c.chave]; !ok {
			t.Errorf("chave %q não existe no padrão", c.chave)
		}
		if len(c.validos) == 0 || len(c.invalidos) == 0 {
			t.Errorf("%s: falta valor válido ou inválido", c.chave)
		}
		for _, v := range c.validos {
			if err := ValidarVfs(c.chave, v); err != nil {
				t.Errorf("%s=%q deveria valer: %v", c.chave, v, err)
			}
		}
		for _, v := range c.invalidos {
			if err := ValidarVfs(c.chave, v); err == nil {
				t.Errorf("%s=%q deveria ser recusado", c.chave, v)
			}
		}
	}
}

func TestPadroesVfsSaoValidos(t *testing.T) {
	for k, v := range ConfiguracoesVfsPadrao {
		if err := ValidarVfs(k, v); err != nil {
			t.Errorf("padrão %s=%q recusado: %v", k, v, err)
		}
	}
}

// Demanda 011: com um valor inválido, Atualizar não aplica nada, nem as
// chaves válidas, e diz qual chave falhou.
func TestAtualizarComValorInvalidoNaoAplicaNada(t *testing.T) {
	cfg := NovoConfigVfs()
	antes := cfg.Obter()

	_, err := cfg.Atualizar(map[string]string{
		"vfs_cache_mode":    "tudo",
		"vfs_cache_max_age": "2h",
		"chave_invalida":    "x",
	})

	var erros ErrosVfs
	if !errors.As(err, &erros) {
		t.Fatalf("esperava ErrosVfs, veio %v", err)
	}
	if erros["vfs_cache_mode"] == nil || erros["chave_invalida"] == nil || erros["vfs_cache_max_age"] != nil {
		t.Errorf("erros por chave = %v", erros)
	}
	if !strings.Contains(err.Error(), "vfs_cache_mode") {
		t.Errorf("a mensagem deveria citar a chave: %q", err)
	}
	depois := cfg.Obter()
	for k, v := range antes {
		if depois[k] != v {
			t.Errorf("%s mudou de %q para %q", k, v, depois[k])
		}
	}
}

// Demanda 011: cache_dir vazio limpa o valor e o rclone usa a pasta padrão.
func TestCacheDirVazioUsaPadraoDoRclone(t *testing.T) {
	cfg := NovoConfigVfs()
	if _, err := cfg.Atualizar(map[string]string{"cache_dir": t.TempDir()}); err != nil {
		t.Fatal(err)
	}

	if _, err := cfg.Atualizar(map[string]string{"cache_dir": ""}); err != nil {
		t.Fatalf("cache_dir vazio deveria valer: %v", err)
	}

	if v := cfg.Obter()["cache_dir"]; v != "" {
		t.Errorf("cache_dir = %q, quer vazio", v)
	}
	for _, a := range cfg.ConstruirArgs(nil) {
		if a == "--cache-dir" {
			t.Error("com cache_dir vazio, --cache-dir não vai para o rclone")
		}
	}
}
