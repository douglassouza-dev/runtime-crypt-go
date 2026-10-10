package core

import (
	"fmt"
	"log"
	"strings"
)

// Demanda 027: o texto em inglês do rclone não chega à tela. Ele vai só para
// o log; a tela recebe uma frase de um conjunto fixo.

// FalhaRclone é o tipo de falha que a tela sabe dizer.
type FalhaRclone int

const (
	// FalhaOutra: tudo que não é um dos casos abaixo.
	FalhaOutra FalhaRclone = iota
	FalhaSenha
	FalhaConexao
	FalhaAutorizacao
	FalhaPastaNaoExiste
)

// padroesFalhaRclone é a tabela de ClassificarSaidaRclone, na ordem em que é
// conferida. Conexão vem antes de autorização: "couldn't fetch token" sem
// rede traz também o erro de rede, e a causa é a rede.
var padroesFalhaRclone = []struct {
	falha   FalhaRclone
	trechos []string
}{
	{FalhaSenha, []string{
		"most likely wrong password",
		"bad password",
		"failed to authenticate decrypted block",
	}},
	{FalhaConexao, []string{
		"no such host",
		"connection refused",
		"i/o timeout",
		"network is unreachable",
		"connection reset by peer",
		"tls handshake timeout",
		"temporary failure in name resolution",
		"no route to host",
	}},
	{FalhaAutorizacao, []string{
		"invalid_grant",
		"couldn't fetch token",
		"token expired",
		"token has been expired or revoked",
		"invalid credentials",
		"401 unauthorized",
	}},
	{FalhaPastaNaoExiste, []string{
		"directory not found",
		"dir not found",
		"object not found",
		"path/not_found",
	}},
}

// ClassificarSaidaRclone leva o que o rclone escreveu a um tipo de falha. É a
// única função que olha o texto do rclone para decidir o que a tela diz.
func ClassificarSaidaRclone(saida string) FalhaRclone {
	s := strings.ToLower(saida)
	for _, p := range padroesFalhaRclone {
		for _, t := range p.trechos {
			if strings.Contains(s, t) {
				return p.falha
			}
		}
	}
	return FalhaOutra
}

// TextoFalhaRclone é a frase da tela para a falha. provedor vazio vira
// "provedor".
func TextoFalhaRclone(f FalhaRclone, provedor string) string {
	if provedor == "" {
		provedor = "provedor"
	}
	switch f {
	case FalhaSenha:
		return "senha errada"
	case FalhaConexao:
		return fmt.Sprintf("sem conexão com o %s", provedor)
	case FalhaAutorizacao:
		return "autorização expirou"
	case FalhaPastaNaoExiste:
		return fmt.Sprintf("a pasta não existe no %s", provedor)
	}
	return "o rclone falhou, detalhes no log"
}

// ErroRclone é uma falha do rclone já classificada. Error() é a frase da
// tela sem o nome do provedor; a tela usa TextoFalhaRclone com o nome. Saida
// é o texto original (só para o log e os testes).
type ErroRclone struct {
	Falha FalhaRclone
	Saida string
	err   error
}

func (e *ErroRclone) Error() string { return TextoFalhaRclone(e.Falha, "") }
func (e *ErroRclone) Unwrap() error { return e.err }

// novoErroRclone classifica a saída e grava o texto original no log.
func novoErroRclone(contexto, saida string, err error) *ErroRclone {
	saida = strings.TrimSpace(saida)
	log.Printf("rclone %s: %s", contexto, saida)
	return &ErroRclone{Falha: ClassificarSaidaRclone(saida), Saida: saida, err: err}
}
