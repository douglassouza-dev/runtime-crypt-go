package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
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

	// erroCarga guarda o motivo de vaults.json não ter sido lido. Enquanto ele
	// existir, nada é gravado por cima do arquivo (demanda 007).
	erroCarga error
}

// NovoGerenciadorCofres cria uma instância e carrega os cofres do disco.
//
// Arquivo ausente vira lista vazia, sem erro. Qualquer outro problema
// (sem permissão, JSON inválido) devolve o gerenciador com a lista vazia e o
// erro: nesse estado Adicionar, Remover e Atualizar recusam e o arquivo fica
// como está. Uma cópia do arquivo ilegível é guardada ao lado, com outro nome.
func NovoGerenciadorCofres(diretorioApp string) (*GerenciadorCofres, error) {
	g := &GerenciadorCofres{
		diretorioApp: diretorioApp,
		cofres:       []Cofre{},
	}
	if err := g.carregar(); err != nil {
		g.erroCarga = err
		return g, err
	}
	return g, nil
}

// ErroCarga devolve o motivo de vaults.json não ter sido lido, ou nil.
func (g *GerenciadorCofres) ErroCarga() error {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.erroCarga
}

// caminhoArquivo retorna o caminho completo do vaults.json.
func (g *GerenciadorCofres) caminhoArquivo() string {
	return filepath.Join(g.diretorioApp, ArquivoCofres)
}

// carregar lê os cofres do disco. Só "arquivo não existe" vira lista vazia
// sem erro.
func (g *GerenciadorCofres) carregar() error {
	caminho := g.caminhoArquivo()
	dados, err := os.ReadFile(caminho)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("nao foi possivel ler %s: %w", caminho, err)
	}
	var cofres []Cofre
	if err := json.Unmarshal(dados, &cofres); err != nil {
		copia := g.preservarIlegivel(dados)
		return fmt.Errorf("%s esta corrompido (%v); nada sera gravado por cima. Copia guardada em %s", caminho, err, copia)
	}
	if cofres == nil {
		cofres = []Cofre{}
	}
	g.cofres = cofres
	return nil
}

// preservarIlegivel grava uma cópia do conteúdo ilegível ao lado do original,
// com a data no nome. Devolve o caminho da cópia, ou o motivo de não ter
// conseguido.
func (g *GerenciadorCofres) preservarIlegivel(dados []byte) string {
	copia := g.caminhoArquivo() + ".corrompido-" + time.Now().Format("20060102-150405")
	if err := os.WriteFile(copia, dados, 0o600); err != nil {
		return "(nenhuma copia: " + err.Error() + ")"
	}
	return copia
}

// salvar persiste os cofres no disco. Grava num arquivo temporário na mesma
// pasta e renomeia por cima, para uma queda no meio nunca deixar vaults.json
// pela metade.
func (g *GerenciadorCofres) salvar() error {
	if g.erroCarga != nil {
		return fmt.Errorf("gravacao bloqueada: %w", g.erroCarga)
	}
	dados, err := json.MarshalIndent(g.cofres, "", "  ")
	if err != nil {
		return err
	}
	return gravarAtomico(g.caminhoArquivo(), dados)
}

// gravarAtomico grava dados em caminho via arquivo temporário + rename.
func gravarAtomico(caminho string, dados []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(caminho), filepath.Base(caminho)+".*.tmp")
	if err != nil {
		return err
	}
	nomeTmp := tmp.Name()
	falhou := true
	defer func() {
		if falhou {
			tmp.Close()
			os.Remove(nomeTmp)
		}
	}()

	if _, err := tmp.Write(dados); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(nomeTmp, 0o644); err != nil {
		return err
	}
	if err := os.Rename(nomeTmp, caminho); err != nil {
		return err
	}
	falhou = false
	return nil
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

	if g.erroCarga != nil {
		return false, "Erro ao salvar: gravacao bloqueada: " + g.erroCarga.Error()
	}

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
		g.cofres = g.cofres[:len(g.cofres)-1]
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
			antes := append([]Cofre(nil), g.cofres...)
			g.cofres = append(g.cofres[:i], g.cofres[i+1:]...)
			if err := g.salvar(); err != nil {
				g.cofres = antes
				return false, "Erro ao salvar: " + err.Error()
			}
			return true, "Cofre '" + nome + "' removido."
		}
	}
	return false, "Cofre '" + nome + "' nao encontrado."
}

// Atualizar modifica campos de um cofre existente e devolve o erro da
// gravação. Cofre inexistente devolve ErrCofreNaoEncontrado.
func (g *GerenciadorCofres) Atualizar(nome string, campos map[string]interface{}) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	for i, c := range g.cofres {
		if c.Nome == nome {
			anterior := g.cofres[i]
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
			if err := g.salvar(); err != nil {
				g.cofres[i] = anterior
				return err
			}
			return nil
		}
	}
	return fmt.Errorf("%w: %s", ErrCofreNaoEncontrado, nome)
}

// ErrCofreNaoEncontrado é devolvido por Atualizar quando o nome não existe.
var ErrCofreNaoEncontrado = errors.New("cofre nao encontrado")

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
