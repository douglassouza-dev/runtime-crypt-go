package core

import "sync"

// CacheSenhas gerencia o armazenamento volátil de senhas em memória.
// Senhas são mantidas apenas durante a sessão e limpas ao trancar ou sair.
type CacheSenhas struct {
	cache map[string]string
	mu    sync.RWMutex
}

// NovoCacheSenhas cria uma nova instância do cache de senhas.
func NovoCacheSenhas() *CacheSenhas {
	return &CacheSenhas{
		cache: make(map[string]string),
	}
}

// Armazenar guarda a senha de um cofre no cache.
func (c *CacheSenhas) Armazenar(nomeCofre string, senha string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[nomeCofre] = senha
}

// Obter retorna a senha de um cofre, ou string vazia se não encontrada.
func (c *CacheSenhas) Obter(nomeCofre string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cache[nomeCofre]
}

// Limpar remove a senha de um cofre do cache.
func (c *CacheSenhas) Limpar(nomeCofre string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.cache, nomeCofre)
}

// LimparTodas remove todas as senhas do cache.
func (c *CacheSenhas) LimparTodas() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = make(map[string]string)
}

// Existe verifica se há senha armazenada para um cofre.
func (c *CacheSenhas) Existe(nomeCofre string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.cache[nomeCofre]
	return ok
}
