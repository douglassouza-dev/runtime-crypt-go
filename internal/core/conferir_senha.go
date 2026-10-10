package core

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"strings"
)

// Demanda 030: a senha digitada ao destrancar é comparada com a senha do
// crypt gravada (ofuscada) no rclone.conf. Não muda onde a senha fica; isso é
// a demanda 004 e o ADR-0006.

// ErrSenhaErrada: a senha digitada não é a do cofre.
var ErrSenhaErrada = errors.New("senha errada")

// Frases da tela quando não deu para comparar a senha (demanda 030,
// aprovadas pela UI). Aparecem embaixo do campo de senha.
const (
	TextoNaoLeuConfiguracao = "Não destrancou: não deu para ler a configuração do rclone."
	TextoConfigIncompleta   = "Não destrancou: a configuração deste cofre está incompleta. Conecte o cofre de novo."
)

// MotivoNaoConferiu diz por que a senha não pôde ser comparada.
type MotivoNaoConferiu int

const (
	// NaoLeuConfiguracao: `rclone config dump` falhou (rclone ausente,
	// tempo esgotado, rclone.conf ilegível).
	NaoLeuConfiguracao MotivoNaoConferiu = iota
	// ConfigIncompleta: o rclone.conf foi lido, mas o remoto do cofre não
	// está lá, não é crypt, não tem `password` ou o valor não se revela.
	// Conectar o cofre de novo grava o remoto inteiro.
	ConfigIncompleta
)

// ErroConferirSenha: não deu para comparar a senha. O cofre não destranca.
// Error() é a frase da tela; Err vai para o log.
type ErroConferirSenha struct {
	Motivo MotivoNaoConferiu
	Err    error
}

func (e *ErroConferirSenha) Error() string {
	if e.Motivo == ConfigIncompleta {
		return TextoConfigIncompleta
	}
	return TextoNaoLeuConfiguracao
}
func (e *ErroConferirSenha) Unwrap() error { return e.Err }

func naoConferiu(nome string, motivo MotivoNaoConferiu, err error) error {
	log.Printf("conferir senha de %s: %v", nome, err)
	return &ErroConferirSenha{Motivo: motivo, Err: err}
}

// ConferirSenha compara senha com o `password` do remoto crypt nome. Volta
// nil se confere, ErrSenhaErrada se não confere ou está vazia, e
// *ErroConferirSenha se não deu para comparar.
func (g *GerenciadorRClone) ConferirSenha(nome, senha string) error {
	if senha == "" {
		return ErrSenhaErrada
	}
	cfg, err := g.ObterConfigRemoto(nome)
	if errors.Is(err, ErrRemotoNaoEncontrado) {
		return naoConferiu(nome, ConfigIncompleta, err)
	}
	if err != nil {
		return naoConferiu(nome, NaoLeuConfiguracao, err)
	}
	if tipo, _ := cfg["type"].(string); tipo != "crypt" {
		return naoConferiu(nome, ConfigIncompleta, fmt.Errorf("remoto do tipo %q, não crypt", tipo))
	}
	obscura, _ := cfg["password"].(string)
	if obscura == "" {
		return naoConferiu(nome, ConfigIncompleta, errors.New("remoto crypt sem password"))
	}
	gravada, err := revelarObscuro(obscura)
	if err != nil {
		return naoConferiu(nome, ConfigIncompleta, err)
	}
	if subtle.ConstantTimeCompare([]byte(gravada), []byte(senha)) != 1 {
		return ErrSenhaErrada
	}
	return nil
}

// chaveObscure é a chave fixa e pública que o rclone usa em `rclone obscure`
// (fs/config/obscure/obscure.go). Ofuscar não é cifrar: qualquer um com o
// rclone.conf revela a senha (ADR-0005).
var chaveObscure = []byte{
	0x9c, 0x93, 0x5b, 0x48, 0x73, 0x0a, 0x55, 0x4d,
	0x6b, 0xfd, 0x7c, 0x63, 0xc8, 0x86, 0xa9, 0x2b,
	0xd3, 0x90, 0x19, 0x8e, 0xb8, 0x12, 0x8a, 0xfb,
	0xf4, 0xde, 0x16, 0x2b, 0x8b, 0x95, 0xf6, 0x38,
}

// revelarObscuro faz o mesmo que `rclone reveal`: base64 de URL sem "=",
// 16 bytes de vetor e o resto em AES-CTR. Fica em Go para o valor não ir nos
// argumentos de um processo (demanda 005) e porque `rclone reveal` recusa
// valor que começa com "-" (demanda 029).
func revelarObscuro(valor string) (string, error) {
	dados, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(valor))
	if err != nil {
		return "", fmt.Errorf("valor ofuscado ilegível: %w", err)
	}
	if len(dados) < aes.BlockSize {
		return "", errors.New("valor ofuscado curto demais")
	}
	bloco, err := aes.NewCipher(chaveObscure)
	if err != nil {
		return "", err
	}
	texto := dados[aes.BlockSize:]
	cipher.NewCTR(bloco, dados[:aes.BlockSize]).XORKeyStream(texto, texto)
	return string(texto), nil
}
