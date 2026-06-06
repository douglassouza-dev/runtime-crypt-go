package core

import (
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

	// Testar geração de argumentos CLI
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

	// Testar restauração de padrões
	cfg.Restaurar()
	valores = cfg.Obter()
	if valores["vfs_cache_mode"] != ConfiguracoesVfsPadrao["vfs_cache_mode"] {
		t.Errorf("Esperava vfs_cache_mode restaurado para padrão '%s', mas obteve '%s'", ConfiguracoesVfsPadrao["vfs_cache_mode"], valores["vfs_cache_mode"])
	}
}
