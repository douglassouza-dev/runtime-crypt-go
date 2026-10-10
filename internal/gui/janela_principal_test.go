package gui

import (
	"runtime"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/eufrauzino/runtime-crypt-go/internal/core"
	"github.com/eufrauzino/runtime-crypt-go/internal/gui/frases"
)

// textosDoCard junta os textos visíveis de um card.
func textosDoCard(o fyne.CanvasObject) []string {
	switch v := o.(type) {
	case *canvas.Text:
		return []string{v.Text}
	case *widget.Label:
		return []string{v.Text}
	case *widget.Button:
		return []string{"[" + v.Text + "]"}
	case *fyne.Container:
		var ts []string
		for _, f := range v.Objects {
			ts = append(ts, textosDoCard(f)...)
		}
		return ts
	}
	return nil
}

// Demanda 022: o Trancar que falha deixa o card destrancado, com o botão
// Trancar e a linha "Não trancou: {motivo}".
func TestCardComTrancarQueFalhouContinuaDestrancadoComMotivo(t *testing.T) {
	test.NewTempApp(t)
	cofre := core.CofreStatus{Cofre: core.Cofre{Nome: "fotos"}, Estado: core.EstadoMontado, Letra: "V"}

	card := criarCardCofre(cofre, "o rclone nao terminou", func() {}, func() {})

	junto := strings.Join(textosDoCard(card), "|")
	if !strings.Contains(junto, "Destrancado") || !strings.Contains(junto, `V:\`) {
		t.Errorf("o card deveria continuar destrancado em V:\\; mostra %q", junto)
	}
	if !strings.Contains(junto, "Não trancou: o rclone nao terminou") {
		t.Errorf("falta a linha \"Não trancou\"; o card mostra %q", junto)
	}
	if !strings.Contains(junto, "[Trancar]") {
		t.Errorf("o botão deveria continuar \"Trancar\"; o card mostra %q", junto)
	}
}

func TestCardSemFalhaNaoMostraNaoTrancou(t *testing.T) {
	test.NewTempApp(t)
	cofre := core.CofreStatus{Cofre: core.Cofre{Nome: "fotos"}, Estado: core.EstadoMontado, Letra: "V"}

	junto := strings.Join(textosDoCard(criarCardCofre(cofre, "", func() {}, func() {})), "|")

	if strings.Contains(junto, "Não trancou") {
		t.Errorf("sem falha, o card não mostra \"Não trancou\": %q", junto)
	}
}

// botaoDoCard acha o botão de ação do card.
func botaoDoCard(o fyne.CanvasObject) *widget.Button {
	switch v := o.(type) {
	case *widget.Button:
		return v
	case *fyne.Container:
		for _, f := range v.Objects {
			if b := botaoDoCard(f); b != nil {
				return b
			}
		}
	}
	return nil
}

// Demanda 018: cada estado tem a frase e o botão certos.
func TestCardMostraOEstadoDoCore(t *testing.T) {
	test.NewTempApp(t)
	destrancando := core.CofreStatus{Cofre: core.Cofre{Nome: "a"}, Estado: core.EstadoMontando}
	naoSubiu := core.CofreStatus{Cofre: core.Cofre{Nome: "b"}, Estado: core.EstadoFalhou, Motivo: "sem winfsp"}
	caiu := core.CofreStatus{Cofre: core.Cofre{Nome: "c"}, Estado: core.EstadoFalhou, Caiu: true, Motivo: core.MotivoProcessoTerminou, Letra: "V"}
	trancado := core.CofreStatus{Cofre: core.Cofre{Nome: "d"}, Estado: core.EstadoDesmontado}

	casos := []struct {
		c          core.CofreStatus
		frase      string
		botao      string
		desabilita bool
	}{
		{destrancando, "Destrancando…", "Destrancar", true},
		{naoSubiu, "Não destrancou: sem winfsp", "Tentar de novo", false},
		{caiu, "Caiu: o rclone parou", "Destrancar de novo", false},
		{trancado, "Trancado", "Destrancar", false},
	}
	for _, c := range casos {
		card := criarCardCofre(c.c, "", func() {}, func() {})
		junto := strings.Join(textosDoCard(card), "|")
		if !strings.Contains(junto, c.frase) {
			t.Errorf("%s: card mostra %q, falta %q", c.c.Nome, junto, c.frase)
		}
		b := botaoDoCard(card)
		if b == nil || b.Text != c.botao || b.Disabled() != c.desabilita {
			t.Errorf("%s: botão = %+v, quer %q desabilitado=%v", c.c.Nome, b, c.botao, c.desabilita)
		}
	}
}

// Destrancando: o clique no botão desabilitado não chama a ação.
func TestCardDestrancandoIgnoraClique(t *testing.T) {
	test.NewTempApp(t)
	cliques := 0
	card := criarCardCofre(core.CofreStatus{Cofre: core.Cofre{Nome: "a"}, Estado: core.EstadoMontando}, "", func() { cliques++ }, func() {})

	test.Tap(botaoDoCard(card))

	if cliques != 0 {
		t.Errorf("%d cliques chegaram à ação", cliques)
	}
}

// Demanda 025: enquanto envia, o card diz quantos arquivos faltam e o botão
// não aceita outro clique.
func TestCardEnviandoDesabilitaBotao(t *testing.T) {
	test.NewTempApp(t)
	c := core.CofreStatus{Cofre: core.Cofre{Nome: "a"}, Estado: core.EstadoMontado, Letra: "V", PontoMontagem: `V:\`, Enviando: 2}
	cliques := 0
	card := criarCardCofre(c, "", func() { cliques++ }, func() {})

	junto := strings.Join(textosDoCard(card), "|")
	if !strings.Contains(junto, "Enviando 2 arquivos…") {
		t.Errorf("card = %q", junto)
	}
	b := botaoDoCard(card)
	test.Tap(b)
	if !b.Disabled() || cliques != 0 {
		t.Errorf("botão deveria estar desabilitado (cliques=%d)", cliques)
	}
}

// Demanda 026: o card que caiu tem "Destrancar de novo" e Trancar; o Trancar
// recusado mostra quantos arquivos não subiram.
func TestCardQueCaiuTemDestrancarDeNovoETrancar(t *testing.T) {
	test.NewTempApp(t)
	c := core.CofreStatus{Cofre: core.Cofre{Nome: "c"}, Estado: core.EstadoFalhou, Caiu: true, Motivo: core.MotivoProcessoTerminou, Letra: "V"}
	destrancou, trancou := 0, 0
	card := criarCardCofre(c, "1 arquivo ainda não subiu", func() { destrancou++ }, func() { trancou++ })

	junto := strings.Join(textosDoCard(card), "|")
	for _, parte := range []string{"[Destrancar de novo]|[Trancar]", "Caiu: o rclone parou", "Não trancou: 1 arquivo ainda não subiu"} {
		if !strings.Contains(junto, parte) {
			t.Errorf("faltou %q; o card mostra %q", parte, junto)
		}
	}
	var bs []*widget.Button
	var achar func(o fyne.CanvasObject)
	achar = func(o fyne.CanvasObject) {
		switch v := o.(type) {
		case *widget.Button:
			bs = append(bs, v)
		case *fyne.Container:
			for _, f := range v.Objects {
				achar(f)
			}
		}
	}
	achar(card)
	if len(bs) != 2 {
		t.Fatalf("esperava 2 botões, há %d", len(bs))
	}
	test.Tap(bs[0])
	test.Tap(bs[1])
	if destrancou != 1 || trancou != 1 {
		t.Errorf("cliques: destrancar=%d trancar=%d", destrancou, trancou)
	}
}

// Não destrancou continua só com "Tentar de novo".
func TestCardQueNaoSubiuSoTemTentarDeNovo(t *testing.T) {
	test.NewTempApp(t)
	c := core.CofreStatus{Cofre: core.Cofre{Nome: "b"}, Estado: core.EstadoFalhou, Motivo: "sem winfsp"}
	junto := strings.Join(textosDoCard(criarCardCofre(c, "", func() {}, func() {})), "|")
	if !strings.Contains(junto, "[Tentar de novo]") || strings.Contains(junto, "[Trancar]") {
		t.Errorf("card = %q", junto)
	}
}

// Demanda 027: sem o driver, o card diz qual falta e, no Windows e no macOS,
// tem o botão que abre o site oficial. No Linux, só o texto.
func TestCardSemDriver(t *testing.T) {
	test.NewTempApp(t)
	var abriu []string
	antes := abrirSiteDriver
	abrirSiteDriver = func(url string) error { abriu = append(abriu, url); return nil }
	t.Cleanup(func() { abrirSiteDriver = antes })

	c := core.CofreStatus{Cofre: core.Cofre{Nome: "d"}, Estado: core.EstadoFalhou, Motivo: "falta instalar o FUSE",
		Rclone: &core.ErroRclone{Falha: core.FalhaDriver}}
	card := criarCardCofre(c, "", func() {}, func() {})
	junto := strings.Join(textosDoCard(card), "|")
	if !strings.Contains(junto, frases.TextoDriverAusente(runtime.GOOS)) {
		t.Errorf("faltou a frase; o card mostra %q", junto)
	}

	var bs []*widget.Button
	var achar func(o fyne.CanvasObject)
	achar = func(o fyne.CanvasObject) {
		switch v := o.(type) {
		case *widget.Button:
			bs = append(bs, v)
		case *fyne.Container:
			for _, f := range v.Objects {
				achar(f)
			}
		}
	}
	achar(card)

	rotulo, url := frases.BotaoBaixarDriver(c)
	if url == "" {
		if len(bs) != 1 || runtime.GOOS != "linux" {
			t.Errorf("%s: esperava só Tentar de novo; o card mostra %q", runtime.GOOS, junto)
		}
		return
	}
	if len(bs) != 2 || bs[1].Text != rotulo {
		t.Fatalf("esperava [Tentar de novo] e [%s]; o card mostra %q", rotulo, junto)
	}
	test.Tap(bs[1])
	if len(abriu) != 1 || abriu[0] != url {
		t.Errorf("abriu %v, quer %s", abriu, url)
	}
}
