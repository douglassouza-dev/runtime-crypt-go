package core

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

// Demanda 025: cada `rclone mount` sobe com o controle remoto (rc) do rclone
// só em 127.0.0.1, numa porta livre e com usuário e senha aleatórios. Antes
// de encerrar o processo, Trancar pergunta ao rc quantos envios da VFS faltam
// (vfs/stats) e espera a fila zerar.
//
// Usuário e senha vão no ambiente (RCLONE_RC_USER/RCLONE_RC_PASS), não nos
// argumentos: argumentos aparecem na lista de processos (demanda 005).

// clienteRC fala com o rc de uma montagem.
type clienteRC struct {
	endereco string // 127.0.0.1:porta
	usuario  string
	senha    string
	http     *http.Client
}

// limiteChamadaRC é o tempo limite de cada chamada ao rc.
const limiteChamadaRC = 5 * time.Second

// esperaRCSubir é quanto se insiste quando o rc ainda não aceita conexão
// (o processo acabou de subir).
var esperaRCSubir = 3 * time.Second

func textoAleatorio() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// novoClienteRC reserva uma porta livre em 127.0.0.1 e sorteia as
// credenciais. A porta é liberada antes de o rclone usá-la; outro programa
// pegar a mesma porta nesse meio-tempo faz o mount falhar com o motivo.
func novoClienteRC() (*clienteRC, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	endereco := l.Addr().String()
	l.Close()
	usuario, err := textoAleatorio()
	if err != nil {
		return nil, err
	}
	senha, err := textoAleatorio()
	if err != nil {
		return nil, err
	}
	return &clienteRC{
		endereco: endereco,
		usuario:  usuario,
		senha:    senha,
		http:     &http.Client{Timeout: limiteChamadaRC},
	}, nil
}

// args são os argumentos do mount para ligar o rc.
func (c *clienteRC) args() []string {
	return []string{"--rc", "--rc-addr", c.endereco}
}

// env leva as credenciais do rc para o processo do mount.
func (c *clienteRC) env() []string {
	return []string{"RCLONE_RC_USER=" + c.usuario, "RCLONE_RC_PASS=" + c.senha}
}

// chamar faz POST <metodo> com corpo {} e decodifica a resposta em saida.
// Conexão recusada é repetida por esperaRCSubir.
func (c *clienteRC) chamar(metodo string, saida any) error {
	fim := time.Now().Add(esperaRCSubir)
	for {
		err := c.chamarUmaVez(metodo, saida)
		var opErr *net.OpError
		if err == nil || !errors.As(err, &opErr) || time.Now().After(fim) {
			return err
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (c *clienteRC) chamarUmaVez(metodo string, saida any) error {
	req, err := http.NewRequest(http.MethodPost, "http://"+c.endereco+"/"+metodo, bytes.NewBufferString("{}"))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.usuario, c.senha)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	corpo, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("rc %s: %s", metodo, resp.Status)
	}
	return json.Unmarshal(corpo, saida)
}

// envioVfs é o que interessa de vfs/stats.
type envioVfs struct {
	DiskCache struct {
		UploadsInProgress int `json:"uploadsInProgress"`
		UploadsQueued     int `json:"uploadsQueued"`
		ErroredFiles      int `json:"erroredFiles"`
	} `json:"diskCache"`
}

// pendentes é quantos arquivos ainda não subiram.
func (e envioVfs) pendentes() int {
	return e.DiskCache.UploadsInProgress + e.DiskCache.UploadsQueued
}

func (c *clienteRC) envio() (envioVfs, error) {
	var e envioVfs
	err := c.chamar("vfs/stats", &e)
	return e, err
}

// bytesEnviados é o total de bytes transferidos (core/stats), para saber se
// um envio grande ainda anda.
func (c *clienteRC) bytesEnviados() (int64, error) {
	var s struct {
		Bytes int64 `json:"bytes"`
	}
	err := c.chamar("core/stats", &s)
	return s.Bytes, err
}
