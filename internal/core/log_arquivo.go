package core

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
)

// Demanda 027: "detalhes no log" precisa de um log. No Windows, com
// -H windowsgui, o stderr do app não vai para lugar nenhum; o log passa a ir
// para runtimecrypto.log na pasta do vaults.json, que troca para .log.1 ao
// passar do limite. Se a pasta não aceita escrita, AbrirLog devolve o erro e
// o app segue sem arquivo de log (sem pasta de reserva). Senhas não são escritas no log em lugar nenhum do core.

// NomeArquivoLog é o nome do log dentro da pasta do app.
const NomeArquivoLog = "runtimecrypto.log"

// LimiteArquivoLog é o tamanho a partir do qual o log troca de arquivo.
const LimiteArquivoLog = 1 << 20

// arquivoLog é um io.Writer que grava em caminho e, ao passar de limite,
// renomeia o arquivo para caminho+".1" (apagando o .1 anterior) e começa
// outro.
type arquivoLog struct {
	mu      sync.Mutex
	caminho string
	limite  int64
	f       *os.File
	tamanho int64
}

func abrirArquivoLog(caminho string, limite int64) (*arquivoLog, error) {
	a := &arquivoLog{caminho: caminho, limite: limite}
	if err := a.abrir(); err != nil {
		return nil, err
	}
	return a, nil
}

func (a *arquivoLog) abrir() error {
	f, err := os.OpenFile(a.caminho, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return err
	}
	a.f, a.tamanho = f, info.Size()
	return nil
}

func (a *arquivoLog) Write(p []byte) (int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.tamanho > 0 && a.tamanho+int64(len(p)) > a.limite {
		a.f.Close()
		_ = os.Remove(a.caminho + ".1")
		_ = os.Rename(a.caminho, a.caminho+".1")
		if err := a.abrir(); err != nil {
			return 0, err
		}
	}
	n, err := a.f.Write(p)
	a.tamanho += int64(n)
	return n, err
}

func (a *arquivoLog) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.f.Close()
}

// AbrirLog manda o log do app para {dir}/runtimecrypto.log. Devolve quem
// fecha o arquivo no fim.
func AbrirLog(dir string) (io.Closer, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	a, err := abrirArquivoLog(filepath.Join(dir, NomeArquivoLog), LimiteArquivoLog)
	if err != nil {
		return nil, err
	}
	log.SetOutput(a)
	return a, nil
}
