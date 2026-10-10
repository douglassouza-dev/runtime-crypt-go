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

// TextoNaoConferiuSenha é a frase da tela quando não deu para comparar
// (proposta, aguarda aprovação da UI).
const TextoNaoConferiuSenha = "não deu para conferir a senha"

// ErroConferirSenha: não deu para comparar a senha (remoto sumiu do
// rclone.conf, sem `password`, valor que não se revela, `config dump` falhou).
// O cofre não destranca. Error() é a frase da tela; Err vai para o log.
type ErroConferirSenha struct {
	Err error
}

func (e *ErroConferirSenha) Error() string { return TextoNaoConferiuSenha }
func (e *ErroConferirSenha) Unwrap() error { return e.Err }

func naoConferiu(nome string, err error) error {
	log.Printf("conferir senha de %s: %v", nome, err)
	return &ErroConferirSenha{Err: err}
}

// ConferirSenha compara senha com o `password` do remoto crypt nome. Volta
// nil se confere, ErrSenhaErrada se não confere ou está vazia, e
// *ErroConferirSenha se não deu para comparar.
func (g *GerenciadorRClone) ConferirSenha(nome, senha string) error {
	if senha == "" {
		return ErrSenhaErrada
	}
	cfg, err := g.ObterConfigRemoto(nome)
	if err != nil {
		return naoConferiu(nome, err)
	}
	if tipo, _ := cfg["type"].(string); tipo != "crypt" {
		return naoConferiu(nome, fmt.Errorf("remoto do tipo %q, não crypt", tipo))
	}
	obscura, _ := cfg["password"].(string)
	if obscura == "" {
		return naoConferiu(nome, errors.New("remoto crypt sem password"))
	}
	gravada, err := revelarObscuro(obscura)
	if err != nil {
		return naoConferiu(nome, err)
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
