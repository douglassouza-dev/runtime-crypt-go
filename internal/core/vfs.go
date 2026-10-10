package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// ConfigVfs gerencia as configurações VFS do RClone (cache, chunking, polling).
type ConfigVfs struct {
	config map[string]string
	mu     sync.RWMutex

	// arquivo é onde a configuração é gravada (demanda 012). Vazio: só em
	// memória (NovoConfigVfs).
	arquivo string
	// avisos lista o que foi descartado na leitura do arquivo.
	avisos []string
	// bloqueio impede gravar (demanda 031: pasta de configuração sem
	// gravação). Atualizar e Restaurar devolvem este erro.
	bloqueio error
}

// ArquivoVfs é o nome do arquivo da configuração VFS, ao lado de vaults.json.
const ArquivoVfs = "vfs.json"

// NovoConfigVfs cria uma nova instância com as configurações padrão, só em
// memória.
func NovoConfigVfs() *ConfigVfs {
	cfg := make(map[string]string)
	for k, v := range ConfiguracoesVfsPadrao {
		cfg[k] = v
	}
	return &ConfigVfs{config: cfg}
}

// NovoConfigVfsEm lê a configuração VFS de diretorio/vfs.json e grava ali a
// cada Atualizar ou Restaurar (demanda 012). Arquivo ausente: padrões, sem
// aviso. Arquivo ilegível, chave desconhecida ou valor que não passa em
// ValidarVfs: aquela parte fica no padrão e entra em Avisos().
func NovoConfigVfsEm(diretorio string) *ConfigVfs {
	c := NovoConfigVfs()
	c.arquivo = filepath.Join(diretorio, ArquivoVfs)

	dados, err := os.ReadFile(c.arquivo)
	if errors.Is(err, fs.ErrNotExist) {
		return c
	}
	if err != nil {
		c.avisos = append(c.avisos, fmt.Sprintf("%s nao pode ser lido (%v); usando os valores padrao.", c.arquivo, err))
		return c
	}
	var salvo map[string]string
	if err := json.Unmarshal(dados, &salvo); err != nil {
		c.avisos = append(c.avisos, fmt.Sprintf("%s esta corrompido (%v); usando os valores padrao.", c.arquivo, err))
		return c
	}

	chaves := make([]string, 0, len(salvo))
	for k := range salvo {
		chaves = append(chaves, k)
	}
	sort.Strings(chaves)
	for _, chave := range chaves {
		valor := salvo[chave]
		if err := ValidarVfs(chave, valor); err != nil {
			padrao, conhecida := ConfiguracoesVfsPadrao[chave]
			if !conhecida {
				c.avisos = append(c.avisos, fmt.Sprintf("%s: chave %q ignorada (%v).", ArquivoVfs, chave, err))
			} else {
				c.avisos = append(c.avisos, fmt.Sprintf("%s: %s=%q descartado (%v); usando %q.", ArquivoVfs, chave, valor, err, padrao))
			}
			continue
		}
		c.config[chave] = valor
	}
	return c
}

// Avisos devolve o que foi descartado na leitura de vfs.json.
func (c *ConfigVfs) Avisos() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return append([]string(nil), c.avisos...)
}

// gravar grava cfg no arquivo, com a escrita atômica de vaults.json (007).
// Sem arquivo (só em memória), não faz nada.
func (c *ConfigVfs) gravar(cfg map[string]string) error {
	if c.bloqueio != nil {
		return c.bloqueio
	}
	if c.arquivo == "" {
		return nil
	}
	dados, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := gravarAtomico(c.arquivo, dados); err != nil {
		return fmt.Errorf("nao foi possivel gravar %s: %w", c.arquivo, err)
	}
	return nil
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
	copia := make(map[string]string)
	for k, v := range c.config {
		copia[k] = v
	}
	if len(erros) > 0 {
		return copia, erros
	}

	// Demanda 012: grava primeiro; só aplica em memória se a gravação deu
	// certo, para memória e disco nunca divergirem.
	nova := make(map[string]string, len(copia))
	for k, v := range copia {
		nova[k] = v
	}
	for chave, valor := range novasConfig {
		nova[chave] = valor
	}
	if err := c.gravar(nova); err != nil {
		return copia, err
	}
	c.config = nova
	resultado := make(map[string]string, len(nova))
	for k, v := range nova {
		resultado[k] = v
	}
	return resultado, nil
}

// Restaurar reseta todas as configurações VFS para os valores padrão e grava
// (demanda 012). Se a gravação falha, nada muda e o erro volta.
func (c *ConfigVfs) Restaurar() (map[string]string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	nova := make(map[string]string)
	for k, v := range ConfiguracoesVfsPadrao {
		nova[k] = v
	}
	if err := c.gravar(nova); err != nil {
		copia := make(map[string]string)
		for k, v := range c.config {
			copia[k] = v
		}
		return copia, err
	}
	c.config = nova
	copia := make(map[string]string)
	for k, v := range c.config {
		copia[k] = v
	}
	return copia, nil
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

// BloquearGravacao faz Atualizar e Restaurar recusarem com err (demanda 031).
func (c *ConfigVfs) BloquearGravacao(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.bloqueio = err
}
