package gui

import (
	"errors"
	"strings"
	"testing"

	"fyne.io/fyne/v2/test"

	"github.com/eufrauzino/runtime-crypt-go/internal/core"
)

// Demanda 018: o wizard mostra o passo e, no erro, o passo que não deu certo.
func TestProgressoMostraOPasso(t *testing.T) {
	a := test.NewTempApp(t)
	w := a.NewWindow("x")
	p := novoProgressoWizard(w, "Novo cofre", "Google Drive")

	for passo, quer := range map[core.Passo]string{
		core.PassoAutorizando:   "Autorizando no navegador…",
		core.PassoCriandoRemoto: "Configurando o Google Drive…",
		core.PassoGravando:      "Gravando o cofre…",
	} {
		p.passo(passo)
		if p.lbl.Text != quer {
			t.Errorf("passo %s = %q, quer %q", passo, p.lbl.Text, quer)
		}
		if strings.Contains(strings.ToLower(p.lbl.Text), "remoto") {
			t.Errorf("a tela não fala em remoto: %q", p.lbl.Text)
		}
	}
	p.esconder()
}

func TestMensagemDeErroDizOPasso(t *testing.T) {
	casos := map[core.Passo]string{
		core.PassoAutorizando:   "Não deu para autorizar no navegador: x",
		core.PassoCriandoRemoto: "Não deu para configurar o Dropbox: x",
		core.PassoGravando:      "Não deu para gravar o cofre: x",
	}
	for passo, quer := range casos {
		err := &core.ErroPasso{Passo: passo, Err: errors.New("x")}
		if got := mensagemErroPasso(err, "Dropbox"); got != quer {
			t.Errorf("%s: %q, quer %q", passo, got, quer)
		}
	}
	if mensagemErroPasso(errors.New("x"), "Dropbox") != "x" {
		t.Error("erro sem passo vai como está")
	}
}

// Acoes implementa a interface do core.
var _ core.Interacao = (*Acoes)(nil)

// 018: erros do core que diriam "remoto" chegam à tela com outro texto.
func TestErrosDoCoreSemRemoto(t *testing.T) {
	casos := map[string]error{
		"O nome 'docs_base' já está em uso no rclone. Escolha outro nome para o cofre.":                      &core.ErroNomeNoRclone{Nome: "docs_base"},
		"Não deu para conferir a configuração do rclone: sem rclone.conf":                                    &core.ErroConferirRclone{Err: errors.New("sem rclone.conf")},
		"Não deu para gravar o cofre: O nome 'x' já está em uso no rclone. Escolha outro nome para o cofre.": &core.ErroPasso{Passo: core.PassoGravando, Err: &core.ErroNomeNoRclone{Nome: "x"}},
	}
	for quer, err := range casos {
		got := mensagemErroPasso(err, "Google Drive")
		if got != quer {
			t.Errorf("mensagem = %q, quer %q", got, quer)
		}
		if strings.Contains(strings.ToLower(got), "remoto") {
			t.Errorf("a tela não pode dizer remoto: %q", got)
		}
	}
}

// 018: o diálogo de sucesso mostra nome do cofre, provedor e pasta, sem "Remoto".
func TestDialogoDeSucessoSemRemoto(t *testing.T) {
	casos := []struct{ titulo, remotoBase, pasta string }{
		{TextoCofreCriado, "docs_base:", "Pasta: /\n"},
		{TextoCofreImportado, "docs_base:Backup/cofre", "Pasta: /Backup/cofre\n"},
	}
	for _, c := range casos {
		got := TextoCofrePronto(c.titulo, "docs", "Google Drive", c.remotoBase)
		for _, parte := range []string{c.titulo, "Nome do cofre: docs\n", "Provedor: Google Drive\n", c.pasta} {
			if !strings.Contains(got, parte) {
				t.Errorf("faltou %q em %q", parte, got)
			}
		}
		if strings.Contains(strings.ToLower(got), "remoto") || strings.Contains(got, "_base") {
			t.Errorf("diálogo mostra o remoto: %q", got)
		}
	}
}

// Demanda 027: falha do rclone no wizard vira a frase fixa com o provedor.
func TestErroDoRcloneNoWizard(t *testing.T) {
	err := &core.ErroPasso{Passo: core.PassoCriandoRemoto, Err: &core.ErroRclone{Falha: core.FalhaAutorizacao, Saida: `oauth2: "invalid_grant"`}}
	if got := mensagemErroPasso(err, "OneDrive"); got != "Não deu para configurar o OneDrive: autorização expirou" {
		t.Errorf("mensagem = %q", got)
	}
	conf := &core.ErroConferirRclone{Err: &core.ErroRclone{Falha: core.FalhaOutra, Saida: "CRITICAL: x"}}
	if got := mensagemErroPasso(conf, "OneDrive"); got != "Não deu para conferir a configuração do rclone: o rclone falhou, detalhes no log" {
		t.Errorf("mensagem = %q", got)
	}
}
