package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Cofre representa os metadados persistidos de um cofre criptografado.
type Cofre struct {
	Nome          string `json:"nome"`
	ProvedorId    string `json:"provedor_id"`
	ProvedorNome  string `json:"provedor_nome"`
	RemotoBase    string `json:"remoto_base"`
	CaminhoCripto string `json:"caminho_cripto"`
	CriadoEm      string `json:"criado_em"`
	AutoMontar    bool   `json:"auto_montar"`
}

// CofreStatus é um Cofre enriquecido com estado em tempo real.
type CofreStatus struct {
	Cofre
	Montado  bool   `json:"montado"`
	Letra    string `json:"letra"`
	TemSenha bool   `json:"tem_senha"`
}

// GerenciadorCofres é responsável pelo CRUD de cofres e persistência em JSON.
type GerenciadorCofres struct {
	cofres       []Cofre
	diretorioApp string
	mu           sync.RWMutex
}

// NovoGerenciadorCofres cria uma instância e carrega os cofres do disco.
func NovoGerenciadorCofres(diretorioApp string) *GerenciadorCofres {
	g := &GerenciadorCofres{
		diretorioApp: diretorioApp,
	}
	g.carregar()
	return g
}

// caminhoArquivo retorna o caminho completo do vaults.json.
func (g *GerenciadorCofres) caminhoArquivo() string {
	return filepath.Join(g.diretorioApp, ArquivoCofres)
}

// carregar lê os cofres do disco.
func (g *GerenciadorCofres) carregar() {
	caminho := g.caminhoArquivo()
	dados, err := os.ReadFile(caminho)
	if err != nil {
		g.cofres = []Cofre{}
		return
	}
	var cofres []Cofre
	if err := json.Unmarshal(dados, &cofres); err != nil {
		g.cofres = []Cofre{}
		return
	}
	g.cofres = cofres
}

// salvar persiste os cofres no disco.
func (g *GerenciadorCofres) salvar() error {
	dados, err := json.MarshalIndent(g.cofres, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(g.caminhoArquivo(), dados, 0644)
}

// Listar retorna todos os cofres com status enriquecido.
// Recebe as montagens ativas e o cache de senhas para preencher o status.
func (g *GerenciadorCofres) Listar(montagens map[string]InfoMontagem, senhas *CacheSenhas) []CofreStatus {
	g.mu.RLock()
	defer g.mu.RUnlock()

	resultado := make([]CofreStatus, 0, len(g.cofres))
	for _, cofre := range g.cofres {
		m, montado := montagens[cofre.Nome]
		status := CofreStatus{
			Cofre:    cofre,
			Montado:  montado,
			TemSenha: senhas.Existe(cofre.Nome),
		}
		if montado {
			status.Letra = m.Letra
		}
		resultado = append(resultado, status)
	}
	return resultado
}

// Adicionar cria um novo cofre.
func (g *GerenciadorCofres) Adicionar(nome, provedorId, provedorNome, remotoBase string) (bool, string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	for _, c := range g.cofres {
		if c.Nome == nome {
			return false, "Ja existe um cofre com o nome '" + nome + "'."
		}
	}

	cofre := Cofre{
		Nome:         nome,
		ProvedorId:   provedorId,
		ProvedorNome: provedorNome,
		RemotoBase:   remotoBase,
		CriadoEm:     time.Now().Format("2006-01-02 15:04:05"),
		AutoMontar:   false,
	}
	g.cofres = append(g.cofres, cofre)
	if err := g.salvar(); err != nil {
		return false, "Erro ao salvar: " + err.Error()
	}
	return true, "Cofre '" + nome + "' adicionado."
}

// Remover exclui um cofre pelo nome.
func (g *GerenciadorCofres) Remover(nome string) (bool, string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	for i, c := range g.cofres {
		if c.Nome == nome {
			g.cofres = append(g.cofres[:i], g.cofres[i+1:]...)
			if err := g.salvar(); err != nil {
				return false, "Erro ao salvar: " + err.Error()
			}
			return true, "Cofre '" + nome + "' removido."
		}
	}
	return false, "Cofre '" + nome + "' nao encontrado."
}

// Atualizar modifica campos de um cofre existente.
func (g *GerenciadorCofres) Atualizar(nome string, campos map[string]interface{}) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	for i, c := range g.cofres {
		if c.Nome == nome {
			if v, ok := campos["auto_montar"]; ok {
				if bv, ok := v.(bool); ok {
					g.cofres[i].AutoMontar = bv
				}
			}
			if v, ok := campos["caminho_cripto"]; ok {
				if sv, ok := v.(string); ok {
					g.cofres[i].CaminhoCripto = sv
				}
			}
			g.salvar()
			return true
		}
	}
	return false
}

// Obter retorna uma cópia do cofre pelo nome, ou nil se não encontrado.
func (g *GerenciadorCofres) Obter(nome string) *Cofre {
	g.mu.RLock()
	defer g.mu.RUnlock()

	for _, c := range g.cofres {
		if c.Nome == nome {
			copia := c
			return &copia
		}
	}
	return nil
}
