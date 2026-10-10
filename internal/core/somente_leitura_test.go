package core

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// Demanda 031: no modo só leitura, destrancar e trancar cofres que já existem
// funcionam e não gravam nada na pasta de configuração nem na do executável.

const cofreExistente = `[{"nome":"cofre","provedor_id":"drive","provedor_nome":"Google Drive","remoto_base":"cofre_base:"}]`
const vfsExistente = `{"vfs_cache_mode":"writes"}`

// fotoPasta guarda nome, tamanho e data de cada arquivo da pasta (sem descer).
func fotoPasta(t *testing.T, dir string) map[string]string {
	t.Helper()
	es, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		t.Fatal(err)
	}
	m := map[string]string{}
	for _, e := range es {
		info, err := e.Info()
		if err != nil {
			t.Fatal(err)
		}
		m[e.Name()] = info.ModTime().Format(time.RFC3339Nano) + " " + info.Mode().String() + " " + fmt.Sprint(info.Size())
	}
	return m
}

func mesmaFoto(t *testing.T, nome string, antes, depois map[string]string) {
	t.Helper()
	if len(antes) != len(depois) {
		t.Errorf("%s: antes %v, depois %v", nome, antes, depois)
		return
	}
	for k, v := range antes {
		if depois[k] != v {
			t.Errorf("%s: %s mudou (%q → %q)", nome, k, v, depois[k])
		}
	}
}

type montagemSomenteLeitura struct {
	nome          string
	preparar      func(t *testing.T) (antiga, config string)
	soUnixSemRoot bool
}

func TestSomenteLeituraDestrancarETrancarNaoGravam(t *testing.T) {
	casos := []montagemSomenteLeitura{
		{nome: "pasta nova impossível de criar; lê da pasta do executável", preparar: func(t *testing.T) (string, string) {
			base := filepath.Join(t.TempDir(), "nao-e-pasta")
			if err := os.WriteFile(base, nil, 0o600); err != nil {
				t.Fatal(err)
			}
			antes := pastaConfigBase
			pastaConfigBase = func() (string, error) { return base, nil }
			t.Cleanup(func() { pastaConfigBase = antes })
			antiga := t.TempDir()
			gravarComData(t, filepath.Join(antiga, ArquivoCofres), cofreExistente)
			gravarComData(t, filepath.Join(antiga, ArquivoVfs), vfsExistente)
			return antiga, filepath.Dir(base)
		}},
		{nome: "pasta nova sem gravação; lê dela", soUnixSemRoot: true, preparar: func(t *testing.T) (string, string) {
			nova := basePastaConfig(t)
			gravarComData(t, filepath.Join(nova, ArquivoCofres), cofreExistente)
			gravarComData(t, filepath.Join(nova, ArquivoVfs), vfsExistente)
			if err := os.Chmod(nova, 0o500); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { os.Chmod(nova, 0o700) })
			return t.TempDir(), nova
		}},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			if c.soUnixSemRoot && (runtime.GOOS == "windows" || os.Geteuid() == 0) {
				t.Skip("permissão de pasta não vale aqui")
			}
			antiga, config := c.preparar(t)
			f := novoRcloneFalso(t)
			// Desde a 030 destrancar confere a senha no `config dump` (só lê).
			// "segredo" ofuscada pelo rclone 1.75.2.
			f.escrever("dump.json", `{"cofre":{"type":"crypt","remote":"cofre_base:","password":"d_zwfc_mykmNxc5ydxnoZGw0NiwI0JM"}}`)
			g := novoGerenciadorPadrao(antiga, f.exe)
			g.Montagens.pontoExiste = pontoPeloFalso(f)
			g.Montagens.raizPontos = t.TempDir()
			t.Cleanup(g.Montagens.DesmontarTodas)
			if !g.SomenteLeitura() {
				t.Fatalf("deveria estar só leitura: %+v", g.RegistroPastaConfig)
			}
			if g.Cofres.Obter("cofre") == nil {
				t.Fatal("o cofre existente deveria ser lido")
			}
			fotoAntiga, fotoConfig := fotoPasta(t, antiga), fotoPasta(t, config)

			if _, err := g.Destrancar("cofre", func() (string, bool) { return "segredo", true }); err != nil {
				t.Fatalf("Destrancar: %v", err)
			}
			if e := g.EstadoDoCofre("cofre"); e.Estado != EstadoMontado {
				t.Fatalf("estado = %+v", e)
			}
			if err := g.Trancar("cofre"); err != nil {
				t.Fatalf("Trancar: %v", err)
			}
			if g.EstaMontado("cofre") || g.Senhas.Existe("cofre") {
				t.Error("deveria estar trancado e sem senha")
			}
			// O fluxo da 028 (destrancar de novo e trancar) é o mesmo caminho.
			if falhas := g.EnviarPendentes([]string{"cofre"}, func(string) (string, bool) { return "segredo", true }); len(falhas) != 0 {
				t.Errorf("EnviarPendentes: %+v", falhas)
			}
			g.Encerrar()

			conferiu := false
			for _, ch := range f.chamadas() {
				if strings.Contains(strings.Join(ch.Args, " "), "config dump") {
					conferiu = true
				}
			}
			if !conferiu {
				t.Error("a senha deveria ser conferida no config dump também no modo só leitura")
			}
			mesmaFoto(t, "pasta do executável", fotoAntiga, fotoPasta(t, antiga))
			mesmaFoto(t, "pasta de configuração", fotoConfig, fotoPasta(t, config))
		})
	}
}

// Criar ou importar cofre recusa antes de chamar o rclone (nada no rclone.conf).
func TestSomenteLeituraNaoComecaCriacao(t *testing.T) {
	base := filepath.Join(t.TempDir(), "nao-e-pasta")
	if err := os.WriteFile(base, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	antes := pastaConfigBase
	pastaConfigBase = func() (string, error) { return base, nil }
	t.Cleanup(func() { pastaConfigBase = antes })
	f := novoRcloneFalso(t)
	g := novoGerenciadorPadrao(t.TempDir(), f.exe)

	_, err := g.IniciarCriacaoCofre("novo")

	var e *ErroPastaConfig
	if !errors.As(err, &e) {
		t.Fatalf("err = %v", err)
	}
	if cs := f.chamadas(); len(cs) != 0 {
		t.Errorf("o rclone não deveria ser chamado: %v", cs)
	}
}
