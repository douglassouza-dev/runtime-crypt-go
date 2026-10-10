package core

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Testes da demanda 026: cofre que caiu não conta como trancado, e o que
// ficou no cache da VFS sem subir impede o Trancar.

// montarComCache monta "cofre" com --cache-dir num diretório do teste e
// devolve o gerenciador, a pasta de cache e o pid do rclone falso.
func montarComCache(t *testing.T) (*GerenciadorRClone, string, int) {
	t.Helper()
	g, _, cache, pid := montarComCacheEFalso(t)
	return g, cache, pid
}

// montarComCacheEFalso é montarComCache devolvendo também o rclone falso.
func montarComCacheEFalso(t *testing.T) (*GerenciadorRClone, *rcloneFalso, string, int) {
	t.Helper()
	g, f := novoGerenciadorFalso(t)
	if ok, msg := g.Cofres.Adicionar("cofre", "drive", "Google Drive", "cofre_base:"); !ok {
		t.Fatal(msg)
	}
	cache := t.TempDir()
	g.Senhas.Armazenar("cofre", "segredo")
	if ok, msg, _ := g.Montagens.MontarUnidade("cofre", "V", "segredo", map[string]string{"cache_dir": cache}); !ok {
		t.Fatal(msg)
	}
	g.Montagens.intervaloEnvio = 20 * time.Millisecond
	t.Cleanup(g.Montagens.DesmontarTodas)
	return g, f, cache, pidDaMontagem(t, g.Montagens, normalizarPonto("V"))
}

// metaNoCache grava o registro da VFS de um arquivo do cofre.
func metaNoCache(t *testing.T, cache, caminho, conteudo string) {
	t.Helper()
	p := filepath.Join(cache, "vfsMeta", "cofre", filepath.FromSlash(caminho))
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(conteudo), 0o600); err != nil {
		t.Fatal(err)
	}
}

const (
	metaSujo  = `{"ModTime":"2026-10-10T09:12:54-03:00","Size":3,"Fingerprint":"","Dirty":true}`
	metaLimpo = `{"ModTime":"2026-10-10T09:12:54-03:00","Size":3,"Fingerprint":"x","Dirty":false}`
)

func derrubar(t *testing.T, g *GerenciadorRClone, pid int) {
	t.Helper()
	matarProcesso(t, pid)
	if !esperarAte(5*time.Second, func() bool {
		e := g.EstadoDoCofre("cofre")
		return e.Estado == EstadoFalhou && e.Caiu
	}) {
		t.Fatalf("o cofre deveria ter caído: %+v", g.EstadoDoCofre("cofre"))
	}
}

func TestTrancarCofreQueCaiuComArquivosQueNaoSubiramRecusa(t *testing.T) {
	g, cache, pid := montarComCache(t)
	metaNoCache(t, cache, "a.txt", metaSujo)
	metaNoCache(t, cache, "sub/b.txt", metaSujo)
	metaNoCache(t, cache, "c.txt", metaLimpo)
	derrubar(t, g, pid)

	err := g.Trancar("cofre")

	var ns *ErroNaoSubiram
	if !errors.As(err, &ns) || ns.N != 2 || err.Error() != "2 arquivos ainda não subiram" {
		t.Fatalf("Trancar = %v, quer 2 arquivos ainda não subiram", err)
	}
	if g.Senhas.Obter("cofre") == "" {
		t.Error("a senha não pode sair: o cofre não trancou")
	}
	if e := g.EstadoDoCofre("cofre"); e.Estado != EstadoFalhou || !e.Caiu {
		t.Errorf("o cofre deveria continuar caído: %+v", e)
	}

	// Singular.
	metaNoCache(t, cache, "sub/b.txt", metaLimpo)
	if err := g.Trancar("cofre"); err == nil || err.Error() != "1 arquivo ainda não subiu" {
		t.Errorf("Trancar = %v, quer 1 arquivo ainda não subiu", err)
	}
}

func TestTrancarCofreQueCaiuSemPendentesTranca(t *testing.T) {
	g, cache, pid := montarComCache(t)
	metaNoCache(t, cache, "c.txt", metaLimpo)
	derrubar(t, g, pid)

	if err := g.Trancar("cofre"); err != nil {
		t.Fatalf("Trancar: %v", err)
	}
	if e := g.EstadoDoCofre("cofre"); e.Estado != EstadoDesmontado {
		t.Errorf("estado = %+v, quer desmontado", e)
	}
	if g.Senhas.Obter("cofre") != "" {
		t.Error("trancado: a senha deveria ter saído")
	}
}

// O defeito antigo: Trancar via o cofre caído como desmontado, apagava a
// senha e voltava nil sem desmontar nada.
func TestTrancarNaoTrataCofreQueCaiuComoDesmontado(t *testing.T) {
	g, cache, pid := montarComCache(t)
	metaNoCache(t, cache, "a.txt", "{corrompido") // na dúvida, conta como pendente
	derrubar(t, g, pid)

	if err := g.Trancar("cofre"); err == nil {
		t.Fatal("Trancar não pode dar certo com o cofre caído e arquivo pendente")
	}
	if g.Senhas.Obter("cofre") == "" {
		t.Error("a senha saiu com o cofre caído")
	}
	if !g.Montagens.temMontagem("cofre") {
		t.Error("a montagem caída sumiu do mapa")
	}
}

func TestDestrancarDeNovoDepoisDaQuedaUsaASenhaDaSessao(t *testing.T) {
	g, cache, pid := montarComCache(t)
	metaNoCache(t, cache, "a.txt", metaSujo)
	derrubar(t, g, pid)
	if err := g.Trancar("cofre"); err == nil {
		t.Fatal("Trancar deveria recusar")
	}

	_, err := g.Destrancar("cofre", func() (string, bool) {
		t.Error("não deveria pedir a senha: ela ficou na sessão")
		return "", false
	})
	if err != nil {
		t.Fatalf("Destrancar de novo: %v", err)
	}
	if e := g.EstadoDoCofre("cofre"); e.Estado != EstadoMontado {
		t.Errorf("estado = %+v, quer montado", e)
	}
}

func TestPastaCacheRclone(t *testing.T) {
	t.Setenv("RCLONE_CACHE_DIR", "")
	base, err := os.UserCacheDir()
	if err != nil {
		base = os.TempDir()
	}
	padrao := filepath.Join(base, "rclone")
	casos := []struct {
		nome      string
		args, env []string
		quer      string
	}{
		{"argumento", []string{"mount", "x:", "p", "--cache-dir", "/c1"}, []string{"RCLONE_CACHE_DIR=/c2"}, "/c1"},
		{"argumento com =", []string{"--cache-dir=/c1"}, nil, "/c1"},
		{"ambiente", []string{"mount"}, []string{"A=b", "RCLONE_CACHE_DIR=/c2"}, "/c2"},
		{"padrão", []string{"mount"}, []string{"A=b"}, padrao},
	}
	for _, c := range casos {
		if got := pastaCacheRclone(c.args, c.env); got != c.quer {
			t.Errorf("%s: %q, quer %q", c.nome, got, c.quer)
		}
	}
}

func TestPendentesSemCacheEZero(t *testing.T) {
	n, err := pendentesNoCacheVfs(t.TempDir(), "cofre:")
	if err != nil || n != 0 {
		t.Errorf("sem cache: n=%d err=%v", n, err)
	}
}
