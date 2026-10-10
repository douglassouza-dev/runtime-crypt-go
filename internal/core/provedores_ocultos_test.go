package core

import (
	"fmt"
	"reflect"
	"testing"
)

// S3 e Pasta Local saem dos assistentes até funcionarem, mas o código fica.

func TestProvedoresDosAssistentesSemS3NemPastaLocal(t *testing.T) {
	var ids []string
	for _, p := range ProvedoresDosAssistentes() {
		ids = append(ids, p.Id)
	}
	if quer := []string{"drive", "onedrive", "dropbox"}; !reflect.DeepEqual(ids, quer) {
		t.Errorf("assistentes = %v, quer %v", ids, quer)
	}
	for _, id := range []string{"s3", "local_path"} {
		if ObterProvedor(id) == nil {
			t.Errorf("%s sumiu de Provedores; só deveria sair dos assistentes", id)
		}
	}
}

// Um cofre de S3 ou de Pasta Local que já está em vaults.json continua
// listando depois de reabrir o app e continua destrancando e trancando.
func TestCofresOcultosQueJaExistemContinuamFuncionando(t *testing.T) {
	for _, c := range []struct{ nome, id, provedor, base string }{
		{"meu-s3", "s3", "Amazon S3 / MinIO", "meu-s3_base:bucket"},
		{"minha-pasta", "local_path", "Pasta Local", "minha-pasta_base:"},
	} {
		t.Run(c.id, func(t *testing.T) {
			f := novoRcloneFalso(t)
			dir := t.TempDir()
			g := NovoGerenciadorEm(dir, f.exe)
			if ok, msg := g.Cofres.Adicionar(c.nome, c.id, c.provedor, c.base); !ok {
				t.Fatal(msg)
			}

			// Reabrir: lê o vaults.json gravado.
			g = NovoGerenciadorEm(dir, f.exe)
			g.Montagens.pontoExiste = pontoPeloFalso(f)
			g.Montagens.raizPontos = t.TempDir()
			t.Cleanup(g.Montagens.DesmontarTodas)
			lista := g.ListarCofres()
			if len(lista) != 1 || lista[0].Nome != c.nome || lista[0].ProvedorId != c.id || lista[0].ProvedorNome != c.provedor {
				t.Fatalf("ListarCofres = %+v", lista)
			}

			f.escrever("dump.json", fmt.Sprintf(`{%q:{"type":"crypt","remote":%q,"password":%q}}`,
				c.nome, c.base, obscurosDoRcloneReal[1].obscuro))
			if _, err := g.Destrancar(c.nome, func() (string, bool) { return "senha certa", true }); err != nil {
				t.Fatalf("Destrancar: %v", err)
			}
			if e := g.EstadoDoCofre(c.nome); e.Estado != EstadoMontado {
				t.Fatalf("estado = %+v, quer montado", e)
			}
			if err := g.Trancar(c.nome); err != nil {
				t.Fatalf("Trancar: %v", err)
			}
			if e := g.EstadoDoCofre(c.nome); e.Estado != EstadoDesmontado {
				t.Errorf("estado = %+v, quer desmontado", e)
			}
		})
	}
}
