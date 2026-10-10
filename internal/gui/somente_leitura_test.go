package gui

import (
	"errors"
	"strings"
	"testing"

	"fyne.io/fyne/v2/test"

	"github.com/eufrauzino/runtime-crypt-go/internal/core"
	"github.com/eufrauzino/runtime-crypt-go/internal/gui/frases"
)

// Demanda 031: no modo só leitura a janela tem a faixa fixa no topo e o que
// grava fica desabilitado.
func TestJanelaSomenteLeituraTemFaixaEBotoesDesabilitados(t *testing.T) {
	app := test.NewTempApp(t)
	g := core.NovoGerenciadorEm(t.TempDir(), "rclone-que-nao-existe")
	g.DiretorioConfig = "/home/x/.config/RuntimeCrypto"
	g.ErroPastaConfig = &core.ErroPastaConfig{Pasta: g.DiretorioConfig, Err: errors.New("sem permissão")}

	jp := NovaJanelaPrincipal(app, g)

	if jp.faixa == nil {
		t.Fatal("faltou a faixa")
	}
	quer := "Mudanças não serão salvas: não deu para gravar em /home/x/.config/RuntimeCrypto."
	if junto := strings.Join(textosDoCard(jp.faixa), "|"); junto != quer {
		t.Errorf("faixa = %q, quer %q", junto, quer)
	}
	for nome, b := range map[string]interface{ Disabled() bool }{"novo": jp.btnNovo, "importar": jp.btnImportar, "configurações": jp.btnConfig} {
		if !b.Disabled() {
			t.Errorf("%s deveria estar desabilitado", nome)
		}
	}
	// A faixa está no conteúdo da janela, não num diálogo.
	if !strings.Contains(strings.Join(textosDoCard(jp.Janela().Content()), "|"), quer) {
		t.Error("a faixa não está na janela")
	}
}

func TestJanelaNormalSemFaixa(t *testing.T) {
	app := test.NewTempApp(t)
	g := core.NovoGerenciadorEm(t.TempDir(), "rclone-que-nao-existe")
	jp := NovaJanelaPrincipal(app, g)
	if jp.faixa != nil || jp.btnNovo.Disabled() || jp.btnImportar.Disabled() || jp.btnConfig.Disabled() {
		t.Error("sem o modo só leitura, sem faixa e com tudo habilitado")
	}
}

func TestFaixaSemCaminho(t *testing.T) {
	if got := frases.SomenteLeitura(""); got != "Mudanças não serão salvas: não deu para gravar na pasta de configuração." {
		t.Errorf("%q", got)
	}
}

// Demanda 031: janela vazia no modo só leitura só diz "Nenhum cofre ainda.";
// no normal, mantém a dica de Adicionar Cofre.
func TestJanelaVaziaPorModo(t *testing.T) {
	app := test.NewTempApp(t)

	g := core.NovoGerenciadorEm(t.TempDir(), "rclone-que-nao-existe")
	g.ErroPastaConfig = &core.ErroPastaConfig{Err: errors.New("sem pasta")}
	jp := NovaJanelaPrincipal(app, g)
	if got := jp.lblVazio.Text; got != "Nenhum cofre ainda." {
		t.Errorf("só leitura: %q", got)
	}
	if junto := strings.Join(textosDoCard(jp.containerCofres), "|"); junto != "Nenhum cofre ainda." {
		t.Errorf("só leitura, a lista mostra %q", junto)
	}
	// Sem o caminho no erro, a faixa usa a variante sem pasta.
	if junto := strings.Join(textosDoCard(jp.faixa), "|"); junto != "Mudanças não serão salvas: não deu para gravar na pasta de configuração." {
		t.Errorf("faixa = %q", junto)
	}

	g2 := core.NovoGerenciadorEm(t.TempDir(), "rclone-que-nao-existe")
	jp2 := NovaJanelaPrincipal(app, g2)
	if junto := strings.Join(textosDoCard(jp2.containerCofres), "|"); junto != "Nenhum cofre configurado.\n\nClique em '＋ Adicionar Cofre' para começar." {
		t.Errorf("normal, a lista mostra %q", junto)
	}
}
