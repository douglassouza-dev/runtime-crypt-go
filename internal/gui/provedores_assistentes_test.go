package gui

import (
	"reflect"
	"testing"

	"github.com/eufrauzino/runtime-crypt-go/internal/core"
)

func ids(ps []core.Provedor) []string {
	var r []string
	for _, p := range ps {
		r = append(r, p.Id)
	}
	return r
}

// S3 e Pasta Local não aparecem em nenhum dos dois assistentes.
func TestAssistentesSemS3NemPastaLocal(t *testing.T) {
	quer := []string{"drive", "onedrive", "dropbox"}
	if got := ids(ProvedoresNovoCofre()); !reflect.DeepEqual(got, quer) {
		t.Errorf("Criar Novo Cofre = %v, quer %v", got, quer)
	}
	if got := ids(ProvedoresImportar()); !reflect.DeepEqual(got, quer) {
		t.Errorf("Importar Cofre Existente = %v, quer %v", got, quer)
	}
}

// "Verificar WinFsp/FUSE" usa a frase e o botão da 027.
func TestAvisoVerificarDriver(t *testing.T) {
	casos := []struct {
		goos string
		quer AvisoDriver
	}{
		{"windows", AvisoDriver{Titulo: TituloDriverAusente, Texto: "Não montou: falta instalar o WinFsp.", Tipo: MsgAviso, Botao: "Baixar WinFsp", Url: "https://winfsp.dev/rel/"}},
		{"darwin", AvisoDriver{Titulo: TituloDriverAusente, Texto: "Não montou: falta instalar o macFUSE.", Tipo: MsgAviso, Botao: "Baixar macFUSE", Url: "https://macfuse.github.io/"}},
		{"linux", AvisoDriver{Titulo: TituloDriverAusente, Texto: "Não montou: falta instalar o FUSE.\n\nInstale o pacote fuse3 pelo gerenciador do seu sistema.", Tipo: MsgAviso}},
	}
	for _, c := range casos {
		if got := AvisoVerificarDriver(false, c.goos); got != c.quer {
			t.Errorf("%s: %+v, quer %+v", c.goos, got, c.quer)
		}
	}
	ok := AvisoVerificarDriver(true, "windows")
	if ok.Texto != "WinFsp/FUSE está instalado e funcionando." || ok.Botao != "" || ok.Tipo != MsgInfo {
		t.Errorf("instalado: %+v", ok)
	}
}
