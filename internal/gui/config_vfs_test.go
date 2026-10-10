package gui

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2/test"

	"github.com/eufrauzino/runtime-crypt-go/internal/core"
)

// Demanda 011: "tudo" em modo de cache e Salvar mostra erro e não muda o valor.
func TestPainelVfsValorInvalidoMostraErroENaoMuda(t *testing.T) {
	test.NewTempApp(t)
	vfs := core.NovoConfigVfs()
	form := novoFormVfs(vfs)

	form.entries["vfs_cache_mode"].SetText("tudo")
	form.entries["vfs_cache_max_age"].SetText("2h") // válido, mas não pode entrar sozinho
	salvou := form.salvar()

	if salvou {
		t.Fatal("salvar deveria recusar")
	}
	if !form.lblErro.Visible() || !strings.Contains(form.lblErro.Text, `"tudo" não é modo de cache`) {
		t.Errorf("o painel deveria mostrar o motivo; mostra %q (visível=%v)", form.lblErro.Text, form.lblErro.Visible())
	}
	cfg := vfs.Obter()
	if cfg["vfs_cache_mode"] != core.ConfiguracoesVfsPadrao["vfs_cache_mode"] || cfg["vfs_cache_max_age"] != core.ConfiguracoesVfsPadrao["vfs_cache_max_age"] {
		t.Errorf("nada deveria mudar no core: %v", cfg)
	}
}

// Demanda 011: "Restaurar Padrões" seguido de "Cancelar" mantém os valores.
func TestPainelVfsRestaurarSemSalvarNaoMuda(t *testing.T) {
	test.NewTempApp(t)
	vfs := core.NovoConfigVfs()
	if _, err := vfs.Atualizar(map[string]string{"vfs_cache_mode": "writes"}); err != nil {
		t.Fatal(err)
	}
	form := novoFormVfs(vfs)

	form.restaurarPadroes()
	// Cancelar só fecha o painel: não chama salvar.

	if form.entries["vfs_cache_mode"].Text != "full" {
		t.Errorf("o campo deveria mostrar o padrão; mostra %q", form.entries["vfs_cache_mode"].Text)
	}
	if v := vfs.Obter()["vfs_cache_mode"]; v != "writes" {
		t.Errorf("sem Salvar, o core deveria continuar com writes; tem %q", v)
	}
}

func TestPainelVfsSalvarValidoAplica(t *testing.T) {
	test.NewTempApp(t)
	vfs := core.NovoConfigVfs()
	form := novoFormVfs(vfs)

	form.entries["vfs_cache_mode"].SetText("minimal")

	if !form.salvar() {
		t.Fatalf("salvar recusou: %q", form.lblErro.Text)
	}
	if v := vfs.Obter()["vfs_cache_mode"]; v != "minimal" {
		t.Errorf("vfs_cache_mode = %q", v)
	}
	if form.lblErro.Visible() {
		t.Error("sem erro, a linha de erro fica escondida")
	}
}
