package gui

import (
	"errors"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/eufrauzino/runtime-crypt-go/internal/core"
)

// Demanda 028: Sair com cofre que caiu e arquivos que não subiram.

type roteiroSair struct {
	pendentes  []core.PendenciaCofre
	enviar     bool
	falhas     []core.FalhaEnvio
	perguntou  string
	enviados   []string
	avisos     []string
	encerrados int
}

func acoesParaSair(t *testing.T, r *roteiroSair) *Acoes {
	t.Helper()
	app := test.NewTempApp(t)
	g := core.NovoGerenciadorEm(t.TempDir(), "")
	jp := NovaJanelaPrincipal(app, g)
	a := NovasAcoes(g, jp)
	a.pendentesAoSair = func() []core.PendenciaCofre { return r.pendentes }
	a.perguntarSair = func(texto string) bool { r.perguntou = texto; return r.enviar }
	a.enviarPendentes = func(nomes []string) []core.FalhaEnvio { r.enviados = nomes; return r.falhas }
	a.avisar = func(titulo, texto string) { r.avisos = append(r.avisos, titulo+": "+texto) }
	return a
}

func TestSairSemPendenciasSaiSemPerguntar(t *testing.T) {
	r := &roteiroSair{}
	acoesParaSair(t, r).Sair(func() { r.encerrados++ })
	if r.encerrados != 1 || r.perguntou != "" {
		t.Errorf("encerrados=%d perguntou=%q", r.encerrados, r.perguntou)
	}
}

func TestSairEscolhendoSairNaoEnvia(t *testing.T) {
	r := &roteiroSair{pendentes: []core.PendenciaCofre{{Nome: "fotos", N: 2}}}
	acoesParaSair(t, r).Sair(func() { r.encerrados++ })
	if r.perguntou != "2 arquivos de fotos ainda não subiram. Eles sobem quando você destrancar de novo." {
		t.Errorf("pergunta = %q", r.perguntou)
	}
	if r.encerrados != 1 || r.enviados != nil {
		t.Errorf("encerrados=%d enviados=%v", r.encerrados, r.enviados)
	}
}

func TestSairEnviarAgoraEnviaETerminaSaindo(t *testing.T) {
	r := &roteiroSair{enviar: true, pendentes: []core.PendenciaCofre{{Nome: "docs", N: 1}, {Nome: "fotos", N: 3}}}
	acoesParaSair(t, r).Sair(func() { r.encerrados++ })
	if strings.Join(r.enviados, ",") != "docs,fotos" || r.encerrados != 1 || len(r.avisos) != 0 {
		t.Errorf("enviados=%v encerrados=%d avisos=%v", r.enviados, r.encerrados, r.avisos)
	}
}

func TestSairEnviarAgoraQueFalhaNaoSai(t *testing.T) {
	r := &roteiroSair{enviar: true, pendentes: []core.PendenciaCofre{{Nome: "fotos", N: 1}},
		falhas: []core.FalhaEnvio{{Nome: "fotos", Etapa: core.EtapaTrancar, Err: errors.New("o envio de 1 arquivo falhou")}}}
	a := acoesParaSair(t, r)
	a.Sair(func() { r.encerrados++ })
	if r.encerrados != 0 {
		t.Error("com falha, o app não pode sair")
	}
	if len(r.avisos) != 1 || r.avisos[0] != "Não deu para enviar: fotos: Não trancou: o envio de 1 arquivo falhou" {
		t.Errorf("avisos = %q", r.avisos)
	}
	if a.jp.falhasTrancar["fotos"] != "o envio de 1 arquivo falhou" {
		t.Errorf("o card deveria mostrar Não trancou: %q", a.jp.falhasTrancar)
	}
}

// O diálogo tem o texto e só os dois botões; "Enviar agora" é o principal.
func TestDialogoSairTemSoEnviarAgoraESair(t *testing.T) {
	test.NewTempApp(t)
	enviou, saiu := 0, 0
	c := conteudoSair("1 arquivo de fotos ainda não subiu. Ele sobe quando você destrancar de novo.", func() { enviou++ }, func() { saiu++ })
	var bs []*widget.Button
	var textos []string
	var achar func(o fyne.CanvasObject)
	achar = func(o fyne.CanvasObject) {
		switch v := o.(type) {
		case *widget.Button:
			bs = append(bs, v)
		case *widget.Label:
			textos = append(textos, v.Text)
		case *fyne.Container:
			for _, f := range v.Objects {
				achar(f)
			}
		}
	}
	achar(c)
	if len(bs) != 2 || bs[0].Text != "Enviar agora" || bs[1].Text != "Sair" || bs[0].Importance != widget.HighImportance {
		t.Fatalf("botões = %+v", bs)
	}
	if !strings.Contains(strings.Join(textos, "|"), "1 arquivo de fotos ainda não subiu.") {
		t.Errorf("textos = %q", textos)
	}
	test.Tap(bs[0])
	test.Tap(bs[1])
	if enviou != 1 || saiu != 1 {
		t.Errorf("enviou=%d saiu=%d", enviou, saiu)
	}
}
