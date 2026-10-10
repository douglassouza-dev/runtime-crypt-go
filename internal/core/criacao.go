package core

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Demanda 006: criar ou conectar um cofre não pode sobrescrever um remoto que
// já existe, e uma tentativa que falha no meio não deixa remotos órfãos.

// reNomeCofre segue as regras de nome de remoto do rclone: letras, números,
// espaço e os sinais _ . + @ -, sem começar por hífen ou espaço. Fica de fora
// tudo o que o rclone usa como separador (":", "/", "\").
var reNomeCofre = regexp.MustCompile(`^[\p{L}\p{N}_.+@][\p{L}\p{N}_.+@ -]*$`)

// ValidarNomeCofre confere o nome antes de qualquer chamada ao rclone.
func ValidarNomeCofre(nome string) error {
	if nome == "" {
		return errors.New("O nome do cofre não pode ficar vazio.")
	}
	if strings.TrimSpace(nome) != nome {
		return errors.New("O nome do cofre não pode começar nem terminar com espaço.")
	}
	if !reNomeCofre.MatchString(nome) {
		return fmt.Errorf("Nome '%s' inválido: use letras, números, espaço e _ . + @ -, sem começar por hífen.", nome)
	}
	return nil
}

// NomeRemotoBase é o remoto que guarda os dados cifrados do cofre (ADR-0003).
func NomeRemotoBase(nome string) string { return nome + "_base" }

// CriacaoCofre acompanha uma tentativa de criar ou conectar um cofre e
// lembra os remotos que ela criou, para desfazer se algo falhar.
type CriacaoCofre struct {
	g         *GerenciadorRClone
	nome      string
	criados   []string
	concluida bool
}

// ErroNomeNoRclone: o nome (ou `<nome>_base`) já existe no rclone.conf. Tipo
// próprio para a tela trocar o texto sem a palavra "remoto" (018).
type ErroNomeNoRclone struct{ Nome string }

func (e *ErroNomeNoRclone) Error() string {
	return fmt.Sprintf("Já existe um remoto '%s' no rclone. Escolha outro nome para o cofre.", e.Nome)
}

// ErroConferirRclone: não deu para listar o rclone.conf antes de criar.
type ErroConferirRclone struct{ Err error }

func (e *ErroConferirRclone) Error() string {
	return fmt.Sprintf("Não foi possível conferir os remotos do rclone: %v", e.Err)
}

func (e *ErroConferirRclone) Unwrap() error { return e.Err }

// IniciarCriacaoCofre valida o nome e confere, antes de qualquer `config
// create`, que nem `<nome>` nem `<nome>_base` existem em vaults.json ou no
// rclone.conf.
func (g *GerenciadorRClone) IniciarCriacaoCofre(nome string) (*CriacaoCofre, error) {
	// Demanda 031: sem pasta de configuração, nem o rclone.conf é tocado.
	if g.SomenteLeitura() {
		return nil, g.ErroPastaConfig
	}
	if err := ValidarNomeCofre(nome); err != nil {
		return nil, err
	}
	if !g.EstaDisponivel() {
		return nil, errors.New(TextoRcloneNaoInstalado)
	}
	base := NomeRemotoBase(nome)
	for _, n := range []string{nome, base} {
		if g.Cofres.Obter(n) != nil {
			return nil, fmt.Errorf("Já existe um cofre com o nome '%s'.", n)
		}
	}
	remotos, err := g.listarTodosRemotos()
	if err != nil {
		return nil, &ErroConferirRclone{Err: err}
	}
	for _, r := range remotos {
		r = strings.TrimSuffix(r, ":")
		for _, n := range []string{nome, base} {
			// Sem diferenciar maiúsculas: no Windows dois nomes que só mudam
			// na caixa confundem quem lê o rclone.conf.
			if strings.EqualFold(r, n) {
				return nil, &ErroNomeNoRclone{Nome: r}
			}
		}
	}
	return &CriacaoCofre{g: g, nome: nome}, nil
}

// Nome devolve o nome do cofre desta tentativa.
func (c *CriacaoCofre) Nome() string { return c.nome }

// CriarRemoto cria um remoto desta tentativa. O nome entra na lista de
// desfazer mesmo se o rclone falhar, porque o nome foi conferido livre e uma
// falha no meio pode ter gravado algo.
// Devolve *ErroRclone quando o rclone recusou (demanda 027).
func (c *CriacaoCofre) CriarRemoto(nomeRemoto, tipo string, params map[string]string) error {
	if nomeRemoto != c.nome && nomeRemoto != NomeRemotoBase(c.nome) {
		return fmt.Errorf("Remoto '%s' não pertence ao cofre '%s'.", nomeRemoto, c.nome)
	}
	c.lembrar(nomeRemoto)
	return c.g.criarRemoto(nomeRemoto, tipo, params)
}

// CriarCrypt cria o remoto crypt `<nome>` desta tentativa.
func (c *CriacaoCofre) CriarCrypt(remotoBase, senha, senha2 string, configCrypt map[string]string) error {
	c.lembrar(c.nome)
	return c.g.criarCrypt(c.nome, remotoBase, senha, senha2, configCrypt)
}

// Concluir grava o cofre em vaults.json. Se falhar, desfaz os remotos.
func (c *CriacaoCofre) Concluir(provedorId, provedorNome, remotoBase string) (bool, string) {
	ok, msg := c.g.Cofres.Adicionar(c.nome, provedorId, provedorNome, remotoBase)
	if !ok {
		c.Desfazer()
		return false, msg
	}
	c.concluida = true
	return true, msg
}

// Desfazer remove os remotos criados nesta tentativa, do último para o
// primeiro. Depois de Concluir com sucesso não faz nada. Devolve os erros de
// remoção, se houver.
func (c *CriacaoCofre) Desfazer() []string {
	if c == nil || c.concluida {
		return nil
	}
	var erros []string
	for i := len(c.criados) - 1; i >= 0; i-- {
		if ok, msg := c.g.RemoverRemoto(c.criados[i]); !ok {
			erros = append(erros, msg)
		}
	}
	c.criados = nil
	return erros
}

func (c *CriacaoCofre) lembrar(nome string) {
	for _, n := range c.criados {
		if n == nome {
			return
		}
	}
	c.criados = append(c.criados, nome)
}
