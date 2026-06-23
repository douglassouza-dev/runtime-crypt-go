package core

import "sync"

// CacheSenhas gerencia o armazenamento volátil de senhas em memória.
// Senhas são mantidas apenas durante a sessão e limpas ao trancar ou sair.
// Armazenamento interno usa []byte para permitir zeroização explícita.
type CacheSenhas struct {
	cache map[string][]byte
	mu    sync.RWMutex
}

// NovoCacheSenhas cria uma nova instância do cache de senhas.
func NovoCacheSenhas() *CacheSenhas {
	return &CacheSenhas{
		cache: make(map[string][]byte),
	}
}

// Armazenar guarda a senha de um cofre no cache (cópia como []byte).
func (c *CacheSenhas) Armazenar(nomeCofre string, senha string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	b := make([]byte, len(senha))
	copy(b, senha)
	c.cache[nomeCofre] = b
}

// Obter retorna a senha de um cofre, ou string vazia se não encontrada.
func (c *CacheSenhas) Obter(nomeCofre string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	b, ok := c.cache[nomeCofre]
	if !ok {
		return ""
	}
	return string(b)
}

// Limpar remove a senha de um cofre do cache e zeroiza os bytes em memória.
func (c *CacheSenhas) Limpar(nomeCofre string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if b, ok := c.cache[nomeCofre]; ok {
		for i := range b {
			b[i] = 0
		}
		delete(c.cache, nomeCofre)
	}
}

// LimparTodas remove todas as senhas do cache e zeroiza os bytes em memória.
func (c *CacheSenhas) LimparTodas() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, b := range c.cache {
		for i := range b {
			b[i] = 0
		}
	}
	c.cache = make(map[string][]byte)
}

// Existe verifica se há senha armazenada para um cofre.
func (c *CacheSenhas) Existe(nomeCofre string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.cache[nomeCofre]
	return ok
}
