package core

import (
	"bytes"
	"log"
	"strings"
	"sync"
	"testing"
)

// capturarLog desvia o log do pacote para um buffer durante o teste.
func capturarLog(t *testing.T) *bufferSeguro {
	t.Helper()
	b := &bufferSeguro{}
	antes := log.Writer()
	log.SetOutput(b)
	t.Cleanup(func() { log.SetOutput(antes) })
	return b
}

type bufferSeguro struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (b *bufferSeguro) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.Write(p)
}

func (b *bufferSeguro) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.String()
}

// Demanda 027: amostras do stderr do rclone 1.60. "real" = capturada no box
// (rclone v1.60.1, Linux) em 2026-10-10; "suposta" = escrita a partir do
// código do rclone ou de relatos, sem como reproduzir aqui.
func TestClassificarSaidaRclone(t *testing.T) {
	casos := []struct {
		origem string
		saida  string
		quer   FalhaRclone
	}{
		// Pasta que não existe.
		{"real: lsjson numa pasta local que não existe", "2026/10/10 09:27:55 ERROR : : error listing: directory not found\n2026/10/10 09:27:55 Failed to lsjson with 2 errors: last error was: error in ListJSON: directory not found", FalhaPastaNaoExiste},
		{"real: lsjson num crypt sobre pasta que não existe", "2026/10/10 09:27:55 Failed to lsjson with 2 errors: last error was: error in ListJSON: directory not found", FalhaPastaNaoExiste},
		{"suposta: arquivo que não existe", "2026/10/10 10:00:00 Failed to cat: object not found", FalhaPastaNaoExiste},
		{"suposta: Dropbox", `Failed to lsjson: error in ListJSON: path/not_found/..`, FalhaPastaNaoExiste},

		// Sem conexão.
		{"real: webdav com conexão recusada", `2026/10/10 09:28:00 Failed to lsjson with 2 errors: last error was: error in ListJSON: couldn't list files: Propfind "http://127.0.0.1:1/": dial tcp 127.0.0.1:1: connect: connection refused`, FalhaConexao},
		{"real: webdav sem resposta", `2026/10/10 09:31:38 ERROR : : error listing: couldn't list files: Propfind "http://host-que-nao-existe.invalid/": dial tcp 198.18.0.1:80: i/o timeout`, FalhaConexao},
		{"suposta: sem DNS", `Failed to create file system for "g:": couldn't find root directory ID: Get "https://www.googleapis.com/drive/v3/files/root": dial tcp: lookup www.googleapis.com: no such host`, FalhaConexao},
		{"suposta: sem rede ao renovar o token (rede tem precedência)", `couldn't fetch token - maybe it has expired? - refresh with "rclone config reconnect g:": Post "https://oauth2.googleapis.com/token": dial tcp: lookup oauth2.googleapis.com: no such host`, FalhaConexao},
		{"suposta: rede fora", "dial tcp 142.250.0.1:443: connect: network is unreachable", FalhaConexao},

		// Autorização expirou.
		{"real: Google Drive com token inválido (lsjson e mount)", `2026/10/10 09:31:43 Failed to create file system for "gd:": couldn't find root directory ID: Get "https://www.googleapis.com/drive/v3/files/root?alt=json&fields=id&prettyPrint=false&supportsAllDrives=true": couldn't fetch token - maybe it has expired? - refresh with "rclone config reconnect gd:": oauth2: "invalid_grant" "Bad Request"`, FalhaAutorizacao},
		{"suposta: Google com 401", "googleapi: Error 401: Invalid Credentials, authError", FalhaAutorizacao},
		{"suposta: OneDrive", `oauth2: cannot fetch token: 400 Bad Request Response: {"error":"invalid_grant","error_description":"AADSTS700082: The refresh token has expired"}`, FalhaAutorizacao},

		// Senha errada.
		{"suposta: rclone.conf cifrado com senha errada", "2026/10/10 10:00:00 Failed to load config file \"rclone.conf\": unable to decrypt configuration: Couldn't decrypt configuration, most likely wrong password.", FalhaSenha},
		{"suposta: crypt com senha errada num bloco", "ERROR : a.txt: Failed to copy: failed to authenticate decrypted block - bad password?", FalhaSenha},

		// O resto.
		{"real: crypt com senha errada num arquivo pequeno (não dá para saber)", "2026/10/10 09:32:25 ERROR : s.txt: Failed to send to output: unexpected EOF\n2026/10/10 09:32:25 Failed to cat: unexpected EOF", FalhaOutra},
		{"real: aviso do mount webdav", "2026/10/10 09:31:54 NOTICE: webdav root '': --vfs-cache-mode writes or full is recommended for this remote as it can't stream", FalhaOutra},
		{"suposta: WinFsp ausente", "2026/10/09 10:00:00 CRITICAL: Fatal error: cannot find winfsp", FalhaOutra},
		{"suposta: remoto sem seção", `Failed to create file system for "x:": didn't find section in config file`, FalhaOutra},
		{"vazia", "", FalhaOutra},
	}
	for _, c := range casos {
		if got := ClassificarSaidaRclone(c.saida); got != c.quer {
			t.Errorf("%s: %v, quer %v", c.origem, got, c.quer)
		}
	}
}

func TestTextoFalhaRclone(t *testing.T) {
	casos := map[FalhaRclone]string{
		FalhaSenha:          "senha errada",
		FalhaConexao:        "sem conexão com o Google Drive",
		FalhaAutorizacao:    "autorização expirou",
		FalhaPastaNaoExiste: "a pasta não existe no Google Drive",
		FalhaOutra:          "o rclone falhou, detalhes no log",
	}
	for f, quer := range casos {
		if got := TextoFalhaRclone(f, "Google Drive"); got != quer {
			t.Errorf("%v: %q, quer %q", f, got, quer)
		}
	}
	if got := TextoFalhaRclone(FalhaConexao, ""); got != "sem conexão com o provedor" {
		t.Errorf("sem provedor: %q", got)
	}
}

func TestNovoErroRcloneGravaOOriginalSoNoLog(t *testing.T) {
	registro := capturarLog(t)
	e := novoErroRclone("mount", "Failed: dial tcp: connection refused", nil)
	if strings.Contains(e.Error(), "dial") || e.Error() != "sem conexão com o provedor" {
		t.Errorf("Error() = %q", e.Error())
	}
	if !strings.Contains(registro.String(), "rclone mount: Failed: dial tcp: connection refused") {
		t.Errorf("log = %q", registro.String())
	}
}
