package core

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
)

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

// Atualizar valida todas as chaves e só então aplica (demanda 011). Se alguma
// chave for desconhecida ou tiver valor inválido, nada é aplicado e o erro é
// um ErrosVfs com o motivo de cada chave. Devolve a configuração resultante.
func (c *ConfigVfs) Atualizar(novasConfig map[string]string) (map[string]string, error) {
	erros := ErrosVfs{}
	for chave, valor := range novasConfig {
		if err := ValidarVfs(chave, valor); err != nil {
			erros[chave] = err
		}
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if len(erros) == 0 {
		for chave, valor := range novasConfig {
			c.config[chave] = valor
		}
	}
	copia := make(map[string]string)
	for k, v := range c.config {
		copia[k] = v
	}
	if len(erros) > 0 {
		return copia, erros
	}
	return copia, nil
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

// ErrosVfs junta o motivo de cada chave recusada por Atualizar.
type ErrosVfs map[string]error

func (e ErrosVfs) Error() string {
	chaves := make([]string, 0, len(e))
	for k := range e {
		chaves = append(chaves, k)
	}
	sort.Strings(chaves)
	partes := make([]string, 0, len(chaves))
	for _, k := range chaves {
		partes = append(partes, k+": "+e[k].Error())
	}
	return strings.Join(partes, "; ")
}

// ModosCacheVfs são os valores aceitos em vfs_cache_mode.
var ModosCacheVfs = []string{"off", "minimal", "writes", "full"}

// chavesTamanhoVfs usam o formato de tamanho do rclone (SizeSuffix).
var chavesTamanhoVfs = map[string]bool{
	"vfs_cache_max_size":        true,
	"vfs_read_chunk_size":       true,
	"vfs_read_chunk_size_limit": true,
	"vfs_read_ahead":            true,
	"buffer_size":               true,
	"vfs_disk_space_total_size": true,
}

// chavesDuracaoVfs usam o formato de duração do rclone (fs.Duration).
var chavesDuracaoVfs = map[string]bool{
	"vfs_cache_max_age": true,
	"dir_cache_time":    true,
	"poll_interval":     true,
	"attr_timeout":      true,
	"vfs_write_back":    true,
}

// reTamanhoVfs: número com sufixo opcional b, k, m, g, t, p (com "i" e "B"
// opcionais: 10G, 10GiB, 512M, 1.5G), ou "off". Sem sufixo o rclone lê KiB.
var reTamanhoVfs = regexp.MustCompile(`^(?i:off|[0-9]+(\.[0-9]+)?([bkmgtp](i?b)?)?)$`)

// reDuracaoVfs: sequência de número e unidade do rclone (300ms, 1h30m, 2d, 1w),
// número sem unidade (o rclone lê segundos) ou "off".
var reDuracaoVfs = regexp.MustCompile(`^(off|[0-9]+(\.[0-9]+)?|([0-9]+(\.[0-9]+)?(ns|us|µs|ms|s|m|h|d|w|M|y))+)$`)

// ValidarVfs confere uma chave VFS antes de ela ir para o rclone (demanda 011).
// cache_dir vazio significa "usar o padrão do rclone".
func ValidarVfs(chave, valor string) error {
	if _, existe := ConfiguracoesVfsPadrao[chave]; !existe {
		return errors.New("chave desconhecida")
	}
	if chave == "cache_dir" {
		return validarPastaCache(valor)
	}
	if strings.TrimSpace(valor) == "" {
		return errors.New("não pode ficar vazio")
	}
	switch {
	case chave == "vfs_cache_mode":
		for _, m := range ModosCacheVfs {
			if valor == m {
				return nil
			}
		}
		return fmt.Errorf("%q não é modo de cache; use off, minimal, writes ou full", valor)
	case chavesTamanhoVfs[chave]:
		if !reTamanhoVfs.MatchString(valor) {
			return fmt.Errorf("%q não é tamanho; use número com K, M, G ou T, por exemplo 512M ou 10G", valor)
		}
	case chavesDuracaoVfs[chave]:
		if !reDuracaoVfs.MatchString(valor) {
			return fmt.Errorf("%q não é duração; use número com s, m, h ou d, por exemplo 30s, 5m ou 1h", valor)
		}
	}
	return nil
}

// validarPastaCache aceita vazio (padrão do rclone) ou uma pasta que existe e
// aceita gravação.
func validarPastaCache(caminho string) error {
	if caminho == "" {
		return nil
	}
	info, err := os.Stat(caminho)
	if err != nil {
		return fmt.Errorf("a pasta %q não existe ou não pode ser lida", caminho)
	}
	if !info.IsDir() {
		return fmt.Errorf("%q não é pasta", caminho)
	}
	f, err := os.CreateTemp(caminho, ".runtimecrypt-teste-*")
	if err != nil {
		return fmt.Errorf("não dá para gravar em %q", caminho)
	}
	nome := f.Name()
	f.Close()
	os.Remove(nome)
	return nil
}
