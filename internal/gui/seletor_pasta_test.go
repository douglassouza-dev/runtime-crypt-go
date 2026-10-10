package gui

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/eufrauzino/runtime-crypt-go/internal/core"
)

// Testes da demanda 009 no seletor de pasta: erro de listagem nunca vira
// "Nenhuma subpasta aqui." e "Tentar de novo" lista a mesma pasta de novo.

// listagemRoteirizada devolve, em ordem, as respostas dadas e guarda os
// caminhos pedidos.
type listagemRoteirizada struct {
	mu        sync.Mutex
	respostas []func() ([]string, error)
	pedidos   []string
}

func (r *listagemRoteirizada) listar(_ string, caminho string) ([]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.pedidos = append(r.pedidos, caminho)
	resp := r.respostas[0]
	if len(r.respostas) > 1 {
		r.respostas = r.respostas[1:]
	}
	return resp()
}

func (r *listagemRoteirizada) caminhos() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.pedidos...)
}

// novaListaDeTeste cria a lista com um canal que recebe um sinal a cada carga
// terminada.
func novaListaDeTeste(t *testing.T, r *listagemRoteirizada) (*listaPastas, chan struct{}) {
	t.Helper()
	test.NewTempApp(t)
	l := novaListaPastas("gdrive_base", "Google Drive", r.listar)
	pronta := make(chan struct{}, 4)
	l.carregada = func() { pronta <- struct{}{} }
	return l, pronta
}

func esperarCarga(t *testing.T, pronta chan struct{}) {
	t.Helper()
	select {
	case <-pronta:
	case <-time.After(5 * time.Second):
		t.Fatal("a lista não terminou de carregar")
	}
}

// textos devolve os textos visíveis na lista, e os botões achados.
func textos(objs []fyne.CanvasObject) ([]string, []*widget.Button) {
	var ts []string
	var bs []*widget.Button
	for _, o := range objs {
		switch v := o.(type) {
		case *canvas.Text:
			ts = append(ts, v.Text)
		case *widget.Label:
			ts = append(ts, v.Text)
		case *widget.Button:
			ts = append(ts, v.Text)
			bs = append(bs, v)
		case *fyne.Container:
			t2, b2 := textos(v.Objects)
			ts = append(ts, t2...)
			bs = append(bs, b2...)
		}
	}
	return ts, bs
}

func TestSeletorErroDeListagemMostraMotivoETentarDeNovo(t *testing.T) {
	r := &listagemRoteirizada{respostas: []func() ([]string, error){
		func() ([]string, error) {
			return nil, errors.New("couldn't list directory: invalid_grant")
		},
	}}
	l, pronta := novaListaDeTeste(t, r)

	l.carregar()
	esperarCarga(t, pronta)

	ts, bs := textos(l.lista.Objects)
	junto := strings.Join(ts, "|")
	if !strings.Contains(junto, "Não deu para listar as pastas: couldn't list directory: invalid_grant") {
		t.Errorf("linha de erro ausente; a lista mostra %q", ts)
	}
	if strings.Contains(junto, TextoPastaSemSubpastas) {
		t.Errorf("erro não pode aparecer como %q; a lista mostra %q", TextoPastaSemSubpastas, ts)
	}
	if len(bs) != 1 || bs[0].Text != "Tentar de novo" {
		t.Errorf("esperava só o botão \"Tentar de novo\"; há %q", ts)
	}
}

func TestSeletorTentarDeNovoListaAMesmaPastaETrocaOErroPelaLista(t *testing.T) {
	r := &listagemRoteirizada{respostas: []func() ([]string, error){
		func() ([]string, error) { return []string{"fotos"}, nil },
		func() ([]string, error) { return nil, errors.New("tempo esgotado") },
		func() ([]string, error) { return []string{"2024", "2025"}, nil },
	}}
	l, pronta := novaListaDeTeste(t, r)
	l.carregar()
	esperarCarga(t, pronta)

	// Entra em "fotos"; a listagem dela falha.
	_, bs := textos(l.lista.Objects)
	test.Tap(bs[0])
	esperarCarga(t, pronta)
	_, bs = textos(l.lista.Objects)
	if len(bs) != 1 || bs[0].Text != TextoTentarDeNovo {
		t.Fatalf("esperava a linha de erro com %q", TextoTentarDeNovo)
	}

	test.Tap(bs[0])
	esperarCarga(t, pronta)

	if got := r.caminhos(); strings.Join(got, "|") != "|fotos|fotos" {
		t.Errorf("caminhos pedidos = %q; \"Tentar de novo\" deveria listar \"fotos\" de novo", got)
	}
	ts, _ := textos(l.lista.Objects)
	junto := strings.Join(ts, "|")
	if strings.Contains(junto, "Não deu para listar") || !strings.Contains(junto, "2024") || !strings.Contains(junto, "2025") {
		t.Errorf("a lista deveria tomar o lugar da linha de erro; mostra %q", ts)
	}
}

func TestSeletorPastaVaziaContinuaNenhumaSubpastaAqui(t *testing.T) {
	r := &listagemRoteirizada{respostas: []func() ([]string, error){
		func() ([]string, error) { return []string{}, nil },
	}}
	l, pronta := novaListaDeTeste(t, r)

	l.carregar()
	esperarCarga(t, pronta)

	ts, bs := textos(l.lista.Objects)
	if strings.Join(ts, "|") != TextoPastaSemSubpastas || len(bs) != 0 {
		t.Errorf("pasta vazia mostra %q", ts)
	}
}

// Demanda 027: o topo mostra "{Provedor} /{caminho}", nunca "{nome}_base:".
func TestTopoDoSeletorMostraOProvedor(t *testing.T) {
	r := &listagemRoteirizada{respostas: []func() ([]string, error){
		func() ([]string, error) { return []string{"a"}, nil },
		func() ([]string, error) { return []string{"b"}, nil },
		func() ([]string, error) { return nil, nil },
	}}
	l, pronta := novaListaDeTeste(t, r)
	l.carregar()
	esperarCarga(t, pronta)
	if got := l.lblCaminho.Text; got != "  Google Drive /" {
		t.Errorf("topo = %q", got)
	}
	l.entrar("a")
	esperarCarga(t, pronta)
	l.entrar("b")
	esperarCarga(t, pronta)
	if got := l.lblCaminho.Text; got != "  Google Drive /a/b" || strings.Contains(got, "_base") {
		t.Errorf("topo = %q", got)
	}
}

// Demanda 027: erro do rclone no seletor aparece com a frase fixa.
func TestSeletorErroDoRcloneSemIngles(t *testing.T) {
	r := &listagemRoteirizada{respostas: []func() ([]string, error){
		func() ([]string, error) {
			return nil, &core.ErroRclone{Falha: core.FalhaConexao, Saida: "dial tcp: i/o timeout"}
		},
	}}
	l, pronta := novaListaDeTeste(t, r)
	l.carregar()
	esperarCarga(t, pronta)
	ts := strings.Join(textosDoCard(l.lista), "|")
	if !strings.Contains(ts, "Não deu para listar as pastas: sem conexão com o Google Drive") || strings.Contains(ts, "dial") {
		t.Errorf("lista = %q", ts)
	}
}

// Demanda 031 (cópia aprovada): enquanto lista, "Carregando pastas…", sem
// emoji e com reticências de um caractere.
func TestSeletorCarregandoPastas(t *testing.T) {
	soltar := make(chan struct{})
	r := &listagemRoteirizada{respostas: []func() ([]string, error){
		func() ([]string, error) { <-soltar; return []string{"a"}, nil },
	}}
	l, pronta := novaListaDeTeste(t, r)
	l.carregar()
	ts, _ := textos(l.lista.Objects)
	close(soltar)
	esperarCarga(t, pronta)
	if len(ts) != 1 || ts[0] != "Carregando pastas…" || TextoCarregandoPastas != "Carregando pastas…" {
		t.Errorf("durante a carga: %q", ts)
	}
}
