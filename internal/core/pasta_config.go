package core

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Demanda 031: vaults.json, vfs.json e runtimecrypto.log saem da pasta do
// executável (que pode não aceitar gravação, ex.: Program Files) e vão para
// os.UserConfigDir()/RuntimeCrypto. O rclone continua sendo procurado na
// pasta do executável.

// NomePastaConfig é o nome da pasta do app dentro de os.UserConfigDir(): o
// mesmo nome do app na bandeja, em ~/RuntimeCrypto e no --volname.
const NomePastaConfig = "RuntimeCrypto"

// ArquivosMigrados são os arquivos copiados uma vez da pasta antiga. O log não
// entra: o novo começa na pasta nova e os antigos ficam onde estão.
var ArquivosMigrados = []string{ArquivoCofres, ArquivoVfs}

// pastaConfigBase é os.UserConfigDir. Os testes trocam.
var pastaConfigBase = os.UserConfigDir

// CaminhoPastaConfig é os.UserConfigDir()/RuntimeCrypto.
func CaminhoPastaConfig() (string, error) {
	base, err := pastaConfigBase()
	if err != nil {
		return "", err
	}
	if base == "" {
		return "", errors.New("pasta de configuração do usuário vazia")
	}
	return filepath.Join(base, NomePastaConfig), nil
}

// ErroPastaConfig: a pasta de configuração não pôde ser criada, gravada ou
// receber a cópia dos arquivos antigos. Error() é a frase da tela.
type ErroPastaConfig struct {
	Pasta string
	Err   error
}

func (e *ErroPastaConfig) Error() string { return TextoPastaConfig(e.Pasta) }
func (e *ErroPastaConfig) Unwrap() error { return e.Err }

// TextoPastaConfig é a faixa fixa do topo da janela nesse modo (aprovada) e
// também o motivo das ações recusadas. As duas variantes estão aprovadas.
func TextoPastaConfig(pasta string) string {
	if pasta == "" {
		return "Mudanças não serão salvas: não deu para gravar na pasta de configuração."
	}
	return fmt.Sprintf("Mudanças não serão salvas: não deu para gravar em %s.", pasta)
}

// PastaConfig é o resultado de preparar a pasta de configuração.
type PastaConfig struct {
	// Dir é a pasta de configuração (vazio se nem o caminho deu para saber).
	Dir string
	// Err (*ErroPastaConfig) diz que a pasta não pode ser usada: os arquivos
	// são lidos de onde existirem e nada é gravado.
	Err error
	// Leitura diz, por arquivo de ArquivosMigrados, de qual pasta ler.
	Leitura map[string]string
	// Registro são as linhas para o log (o log só abre depois).
	Registro []string
}

// prepararPastaConfig cria a pasta e confere que ela aceita gravação com um
// arquivo temporário, que é apagado em seguida.
func prepararPastaConfig(pasta string) error {
	if err := os.MkdirAll(pasta, 0o700); err != nil {
		return err
	}
	f, err := os.CreateTemp(pasta, ".teste-gravacao-*")
	if err != nil {
		return err
	}
	nome := f.Name()
	f.Close()
	return os.Remove(nome)
}

// migrarArquivo copia antiga/nome para nova/nome se nova/nome ainda não
// existe e antiga/nome existe. A cópia é um temporário na pasta nova,
// renomeado no fim. O antigo nunca é apagado nem regravado.
func migrarArquivo(antiga, nova, nome string) (copiou bool, err error) {
	destino := filepath.Join(nova, nome)
	if _, err := os.Lstat(destino); err == nil {
		return false, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return false, err
	}
	dados, err := os.ReadFile(filepath.Join(antiga, nome))
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := gravarAtomico(destino, dados); err != nil {
		return false, err
	}
	return true, nil
}

// pastaDeLeitura é nova se nova/nome existe; senão antiga (pasta nova
// inutilizável: o antigo é só lido).
func pastaDeLeitura(antiga, nova, nome string) string {
	if nova != "" {
		if _, err := os.Lstat(filepath.Join(nova, nome)); err == nil {
			return nova
		}
	}
	return antiga
}

// PrepararPastaConfig resolve a pasta de configuração, cria, confere a
// gravação e copia uma vez os arquivos de antiga (a pasta do executável).
func PrepararPastaConfig(antiga string) PastaConfig {
	nova, err := CaminhoPastaConfig()
	if err == nil {
		return prepararEm(antiga, nova)
	}
	p := PastaConfig{Err: &ErroPastaConfig{Err: err}, Leitura: map[string]string{}}
	for _, nome := range ArquivosMigrados {
		p.Leitura[nome] = antiga
	}
	p.Registro = append(p.Registro, fmt.Sprintf("pasta de configuração: não deu para saber o caminho: %v", err))
	return p
}

func prepararEm(antiga, nova string) PastaConfig {
	p := PastaConfig{Dir: nova, Leitura: map[string]string{}}
	falhou := func(err error) PastaConfig {
		p.Err = &ErroPastaConfig{Pasta: nova, Err: err}
		for _, nome := range ArquivosMigrados {
			p.Leitura[nome] = pastaDeLeitura(antiga, nova, nome)
		}
		p.Registro = append(p.Registro, fmt.Sprintf("pasta de configuração %s: %v; nada será gravado", nova, err))
		return p
	}
	if err := prepararPastaConfig(nova); err != nil {
		return falhou(err)
	}
	mesma := mesmaPasta(antiga, nova)
	for _, nome := range ArquivosMigrados {
		p.Leitura[nome] = nova
		if mesma {
			continue
		}
		copiou, err := migrarArquivo(antiga, nova, nome)
		if err != nil {
			return falhou(fmt.Errorf("não deu para copiar %s de %s: %w", nome, antiga, err))
		}
		if copiou {
			p.Registro = append(p.Registro, fmt.Sprintf("pasta de configuração: %s copiado de %s para %s (o antigo fica como está)", nome, antiga, nova))
		}
	}
	return p
}

func mesmaPasta(a, b string) bool {
	ia, errA := os.Stat(a)
	ib, errB := os.Stat(b)
	if errA != nil || errB != nil {
		return filepath.Clean(a) == filepath.Clean(b)
	}
	return os.SameFile(ia, ib)
}
