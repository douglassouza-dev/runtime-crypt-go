package core

import "sync"

// ConfigVfs gerencia as configurações VFS do RClone (cache, chunking, polling).
type ConfigVfs struct {
	config map[string]string
	mu     sync.RWMutex
}

// NovoConfigVfs cria uma nova instância com as configurações padrão.
func NovoConfigVfs() *ConfigVfs {
	cfg := make(map[string]string)
	for k, v := range ConfiguracoesVfsPadrao {
		cfg[k] = v
	}
	return &ConfigVfs{config: cfg}
}

// Obter retorna uma cópia das configurações VFS atuais.
func (c *ConfigVfs) Obter() map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	copia := make(map[string]string)
	for k, v := range c.config {
		copia[k] = v
	}
	return copia
}

// Atualizar atualiza as configurações VFS com os valores fornecidos.
// Apenas chaves válidas (existentes no padrão) são aceitas.
func (c *ConfigVfs) Atualizar(novasConfig map[string]string) map[string]string {
	c.mu.Lock()
	defer c.mu.Unlock()
	for chave, valor := range novasConfig {
		if _, existe := ConfiguracoesVfsPadrao[chave]; existe && valor != "" {
			c.config[chave] = valor
		}
	}
	copia := make(map[string]string)
	for k, v := range c.config {
		copia[k] = v
	}
	return copia
}

// Restaurar reseta todas as configurações VFS para os valores padrão.
func (c *ConfigVfs) Restaurar() map[string]string {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.config = make(map[string]string)
	for k, v := range ConfiguracoesVfsPadrao {
		c.config[k] = v
	}
	copia := make(map[string]string)
	for k, v := range c.config {
		copia[k] = v
	}
	return copia
}

// mapaFlagsVfs mapeia chaves de config para flags CLI do rclone.
var mapaFlagsVfs = map[string]string{
	"vfs_cache_mode":            "--vfs-cache-mode",
	"vfs_cache_max_size":        "--vfs-cache-max-size",
	"vfs_cache_max_age":         "--vfs-cache-max-age",
	"vfs_read_chunk_size":       "--vfs-read-chunk-size",
	"vfs_read_chunk_size_limit": "--vfs-read-chunk-size-limit",
	"vfs_read_ahead":            "--vfs-read-ahead",
	"buffer_size":               "--buffer-size",
	"dir_cache_time":            "--dir-cache-time",
	"poll_interval":             "--poll-interval",
	"attr_timeout":              "--attr-timeout",
	"vfs_write_back":            "--vfs-write-back",
	"vfs_disk_space_total_size": "--vfs-disk-space-total-size",
	"cache_dir":                 "--cache-dir",
}

// ConstruirArgs gera a lista de argumentos CLI do rclone a partir das configurações.
// Aceita um override opcional para sobrescrever valores específicos.
func (c *ConfigVfs) ConstruirArgs(override map[string]string) []string {
	c.mu.RLock()
	cfg := make(map[string]string)
	for k, v := range c.config {
		cfg[k] = v
	}
	c.mu.RUnlock()

	if override != nil {
		for k, v := range override {
			if v != "" {
				cfg[k] = v
			}
		}
	}

	var args []string
	for chave, flag := range mapaFlagsVfs {
		valor, existe := cfg[chave]
		if existe && valor != "" {
			args = append(args, flag, valor)
		}
	}
	return args
}
