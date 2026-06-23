package core

import (
	"testing"
)

func TestConfigVfs(t *testing.T) {
	cfg := NovoConfigVfs()

	valores := cfg.Obter()
	for k, v := range ConfiguracoesVfsPadrao {
		if valores[k] != v {
			t.Errorf("Esperava valor padrão '%s' para chave '%s', mas obteve '%s'", v, k, valores[k])
		}
	}

	novasConfig := map[string]string{
		"vfs_cache_mode": "full",
		"chave_invalida": "invalido",
	}
	cfg.Atualizar(novasConfig)

	valores = cfg.Obter()
	if valores["vfs_cache_mode"] != "full" {
		t.Errorf("Esperava vfs_cache_mode atualizado para 'full', mas obteve '%s'", valores["vfs_cache_mode"])
	}

	if _, existe := valores["chave_invalida"]; existe {
		t.Errorf("Não esperava que a chave inválida fosse aceita no config")
	}

	args := cfg.ConstruirArgs(nil)
	temCacheMode := false
	for i, arg := range args {
		if arg == "--vfs-cache-mode" && i+1 < len(args) && args[i+1] == "full" {
			temCacheMode = true
			break
		}
	}
	if !temCacheMode {
		t.Errorf("Esperava encontrar '--vfs-cache-mode full' nos argumentos CLI gerados: %v", args)
	}

	cfg.Restaurar()
	valores = cfg.Obter()
	if valores["vfs_cache_mode"] != ConfiguracoesVfsPadrao["vfs_cache_mode"] {
		t.Errorf("Esperava vfs_cache_mode restaurado para padrão '%s', mas obteve '%s'", ConfiguracoesVfsPadrao["vfs_cache_mode"], valores["vfs_cache_mode"])
	}
}

func TestValidarValorVfs(t *testing.T) {
	testes := []struct {
		chave    string
		valor    string
		esperado bool
	}{
		{"vfs_cache_mode", "full", true},
		{"vfs_cache_mode", "off", true},
		{"vfs_cache_mode", "minimal", true},
		{"vfs_cache_mode", "writes", true},
		{"vfs_cache_mode", "garbage", false},
		{"vfs_cache_mode", "FULL", false},
		{"vfs_cache_mode", "", true},
		{"vfs_cache_max_size", "10G", true},
		{"vfs_cache_max_size", "512M", true},
		{"vfs_cache_max_size", "256Ki", true},
		{"vfs_cache_max_size", "-1", false},
		{"vfs_cache_max_size", "abc", false},
		{"vfs_read_chunk_size", "0", true},
		{"vfs_read_ahead", "256M", true},
		{"vfs_read_ahead", "0", true},
		{"buffer_size", "32M", true},
		{"vfs_disk_space_total_size", "1T", true},
		{"vfs_disk_space_total_size", "100G", true},
		{"vfs_cache_max_age", "1h", true},
		{"vfs_cache_max_age", "30m", true},
		{"vfs_cache_max_age", "10ms", true},
		{"vfs_cache_max_age", "xyz", false},
		{"dir_cache_time", "30m", true},
		{"dir_cache_time", "5s", true},
		{"poll_interval", "10s", true},
		{"poll_interval", "1m", true},
		{"poll_interval", "abc", false},
		{"attr_timeout", "1m", true},
		{"attr_timeout", "10s", true},
		{"vfs_write_back", "5s", true},
		{"vfs_write_back", "-1s", false},
		{"cache_dir", "/tmp/cache", true},
		{"cache_dir", "", true},
		{"chave_inexistente", "algo", true},
	}

	for _, tt := range testes {
		err := validarValorVfs(tt.chave, tt.valor)
		valido := err == nil
		if valido != tt.esperado {
			status := "válido"
			if !tt.esperado {
				status = "inválido"
			}
			errMsg := ""
			if err != nil {
				errMsg = " (erro: " + err.Error() + ")"
			}
			t.Errorf("%s='%s': esperava %s, mas foi considerado %s%s", tt.chave, tt.valor, status, map[bool]string{true: "válido", false: "inválido"}[valido], errMsg)
		}
	}
}

func TestAtualizarRejeitaValoresInvalidos(t *testing.T) {
	cfg := NovoConfigVfs()

	cfg.Atualizar(map[string]string{
		"vfs_cache_mode":    "invalido",
		"vfs_cache_max_size": "-1G",
		"vfs_cache_max_age":  "xyz",
	})

	valores := cfg.Obter()

	if valores["vfs_cache_mode"] != ConfiguracoesVfsPadrao["vfs_cache_mode"] {
		t.Errorf("vfs_cache_mode não deveria ter sido alterado para valor inválido: %s", valores["vfs_cache_mode"])
	}
	if valores["vfs_cache_max_size"] != ConfiguracoesVfsPadrao["vfs_cache_max_size"] {
		t.Errorf("vfs_cache_max_size não deveria ter sido alterado para valor inválido: %s", valores["vfs_cache_max_size"])
	}
	if valores["vfs_cache_max_age"] != ConfiguracoesVfsPadrao["vfs_cache_max_age"] {
		t.Errorf("vfs_cache_max_age não deveria ter sido alterado para valor inválido: %s", valores["vfs_cache_max_age"])
	}
}

func TestPresetsVfs(t *testing.T) {
	if len(PresetsVfs) == 0 {
		t.Error("Esperava pelo menos um preset VFS definido")
	}

	nomes := make(map[string]bool)
	for _, preset := range PresetsVfs {
		if preset.Nome == "" {
			t.Error("Preset com nome vazio encontrado")
		}
		if preset.Descricao == "" {
			t.Errorf("Preset '%s' sem descrição", preset.Nome)
		}
		if len(preset.Valores) == 0 {
			t.Errorf("Preset '%s' sem valores", preset.Nome)
		}
		if nomes[preset.Nome] {
			t.Errorf("Preset duplicado: '%s'", preset.Nome)
		}
		nomes[preset.Nome] = true

		for chave, valor := range preset.Valores {
			if err := validarValorVfs(chave, valor); err != nil {
				t.Errorf("Preset '%s': valor inválido para '%s'='%s': %s", preset.Nome, chave, valor, err.Error())
			}
		}
	}
}
