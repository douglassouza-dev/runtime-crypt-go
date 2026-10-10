package core

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
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
	TextoRcloneAusente      = "Não destrancou: o rclone não está instalado."
)

// TextoRcloneNaoInstalado é a frase das ações que precisam do rclone e não o
// acharam (criar, conectar, montar), no mesmo estilo da 030. Era
// "RClone nao disponivel.".
const TextoRcloneNaoInstalado = "O rclone não está instalado."

// MotivoNaoConferiu diz por que a senha não pôde ser comparada.
type MotivoNaoConferiu int

const (
	// NaoLeuConfiguracao: `rclone config dump` rodou e falhou (tempo
	// esgotado, erro do rclone, saída ilegível).
	NaoLeuConfiguracao MotivoNaoConferiu = iota
	// ConfigIncompleta: o rclone.conf foi lido, mas o remoto do cofre não
	// está lá, não é crypt, não tem `password` ou o valor não se revela.
	// Conectar o cofre de novo grava o remoto inteiro.
	ConfigIncompleta
	// RcloneAusente: o binário do rclone não foi achado (nenhum rclone na
	// abertura, exec.ErrNotFound no PATH ou os.ErrNotExist no caminho dele).
	RcloneAusente
)

// ErroConferirSenha: não deu para comparar a senha. O cofre não destranca.
// Error() é a frase da tela; Err vai para o log.
type ErroConferirSenha struct {
	Motivo MotivoNaoConferiu
	Err    error
}

func (e *ErroConferirSenha) Error() string {
	switch e.Motivo {
	case ConfigIncompleta:
		return TextoConfigIncompleta
	case RcloneAusente:
		return TextoRcloneAusente
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
	if rcloneAusente(err) {
		return naoConferiu(nome, RcloneAusente, err)
	}
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

// rcloneAusente diz se err é o binário do rclone que não existe: nenhum
// rclone achado na abertura (ErrRcloneIndisponivel), o nome não está no PATH
// (exec.ErrNotFound) ou o arquivo no caminho dele sumiu (os.ErrNotExist, que
// no Windows também cobre ERROR_FILE_NOT_FOUND e ERROR_PATH_NOT_FOUND). Não
// olha o texto da mensagem. A saída do rclone que roda e falha vem como
// *exec.ExitError ou *ErroRclone, que não embrulham nenhum desses.
func rcloneAusente(err error) bool {
	return errors.Is(err, ErrRcloneIndisponivel) || errors.Is(err, exec.ErrNotFound) || errors.Is(err, os.ErrNotExist)
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
