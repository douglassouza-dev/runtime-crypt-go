package core

import (
	"os"
	"path/filepath"
	"testing"
)

// Testes da demanda 028: sair com cofre que caiu e arquivos que não subiram.

func TestPendentesAoSairListaOCofreQueCaiuComArquivos(t *testing.T) {
	g, cache, pid := montarComCache(t)
	metaNoCache(t, cache, "a.txt", metaSujo)
	metaNoCache(t, cache, "sub/b.txt", metaSujo)
	derrubar(t, g, pid)

	p := g.PendentesAoSair()
	if len(p) != 1 || p[0].Nome != "cofre" || p[0].N != 2 {
		t.Errorf("pendências = %+v, quer [{cofre 2}]", p)
	}
}

func TestPendentesAoSairVazioSemArquivosOuComRcloneVivo(t *testing.T) {
	g, cache, pid := montarComCache(t)
	metaNoCache(t, cache, "a.txt", metaSujo)
	// rclone vivo: o Encerrar espera pelo rc (025); não entra na lista.
	if p := g.PendentesAoSair(); len(p) != 0 {
		t.Errorf("com o rclone vivo, pendências = %+v", p)
	}
	metaNoCache(t, cache, "a.txt", metaLimpo)
	derrubar(t, g, pid)
	if p := g.PendentesAoSair(); len(p) != 0 {
		t.Errorf("sem arquivos sujos, pendências = %+v", p)
	}
}

func TestEnviarPendentesDestrancaEsperaETranca(t *testing.T) {
	g, f, cache, pid := montarComCacheEFalso(t)
	metaNoCache(t, cache, "a.txt", metaSujo)
	derrubar(t, g, pid)

	falhas := g.EnviarPendentes([]string{"cofre"}, func(string) (string, bool) {
		t.Error("a senha está na sessão; não deveria pedir")
		return "", false
	})

	if len(falhas) != 0 {
		t.Fatalf("falhas = %+v", falhas)
	}
	if e := g.EstadoDoCofre("cofre"); e.Estado != EstadoDesmontado {
		t.Errorf("estado = %+v, quer desmontado", e)
	}
	if g.Senhas.Obter("cofre") != "" {
		t.Error("trancado: a senha deveria ter saído")
	}
	var mount int
	for _, c := range f.chamadas() {
		if len(c.Args) > 0 && c.Args[0] == "mount" {
			mount++
		}
	}
	if mount != 2 {
		t.Errorf("esperava 2 mounts (o original e o de novo), houve %d", mount)
	}
}

func TestEnviarPendentesComEnvioQueFalhaNaoTranca(t *testing.T) {
	g, f, cache, pid := montarComCacheEFalso(t)
	metaNoCache(t, cache, "a.txt", metaSujo)
	derrubar(t, g, pid)
	filaFalsa(f, 0, 0, 1) // o envio falha na nova montagem

	falhas := g.EnviarPendentes([]string{"cofre"}, func(string) (string, bool) { return "", false })

	if len(falhas) != 1 || falhas[0].Etapa != EtapaTrancar || falhas[0].Err.Error() != "o envio de 1 arquivo falhou" {
		t.Fatalf("falhas = %+v", falhas)
	}
	if g.Senhas.Obter("cofre") == "" {
		t.Error("não trancou: a senha fica")
	}
}

func TestEnviarPendentesQueNaoDestrancaNaoTranca(t *testing.T) {
	g, f, cache, pid := montarComCacheEFalso(t)
	metaNoCache(t, cache, "a.txt", metaSujo)
	derrubar(t, g, pid)
	f.falhar()

	falhas := g.EnviarPendentes([]string{"cofre"}, func(string) (string, bool) { return "", false })

	if len(falhas) != 1 || falhas[0].Etapa != EtapaDestrancar {
		t.Fatalf("falhas = %+v", falhas)
	}
}

func TestSairSemEnviarDeixaOCacheComoEsta(t *testing.T) {
	g, cache, pid := montarComCache(t)
	metaNoCache(t, cache, "a.txt", metaSujo)
	derrubar(t, g, pid)

	g.Encerrar()

	dados, err := os.ReadFile(filepath.Join(cache, "vfsMeta", "cofre", "a.txt"))
	if err != nil || string(dados) != metaSujo {
		t.Errorf("o vfsMeta mudou: %q %v", dados, err)
	}
}
