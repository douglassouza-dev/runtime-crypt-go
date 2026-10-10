package frases

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/eufrauzino/runtime-crypt-go/internal/core"
)

func cofre(estado core.EstadoMontagem) core.CofreStatus {
	return core.CofreStatus{Cofre: core.Cofre{Nome: "fotos"}, Estado: estado}
}

func TestFrasesDoCofre(t *testing.T) {
	montado := cofre(core.EstadoMontado)
	montado.Letra, montado.PontoMontagem = "V", `V:\`
	naoSubiu := cofre(core.EstadoFalhou)
	naoSubiu.Motivo = "CRITICAL: cannot find winfsp"
	parou := cofre(core.EstadoFalhou)
	parou.Caiu, parou.Motivo, parou.Letra = true, core.MotivoProcessoTerminou, "V"
	sumiu := cofre(core.EstadoFalhou)
	sumiu.Caiu, sumiu.Motivo, sumiu.Letra, sumiu.PontoMontagem = true, core.MotivoPontoSumiu, "W", `W:\`
	semResposta := cofre(core.EstadoFalhou)
	semResposta.Caiu, semResposta.Motivo, semResposta.Letra = true, "ponto de montagem não respondeu em 2s", "X"

	casos := []struct {
		c            core.CofreStatus
		frase, botao string
	}{
		{cofre(core.EstadoDesmontado), "Trancado", "Destrancar"},
		{cofre(core.EstadoMontando), "Destrancando…", "Destrancar"},
		{montado, `Destrancado • V:\`, "Trancar"},
		{naoSubiu, "Não destrancou: CRITICAL: cannot find winfsp", "Tentar de novo"},
		{parou, "Caiu: o rclone parou", "Destrancar de novo"},
		{sumiu, `Caiu: a unidade W:\ sumiu`, "Destrancar de novo"},
		{semResposta, `Caiu: a unidade X:\ não respondeu`, "Destrancar de novo"},
	}
	for _, c := range casos {
		if got := DoCofre(c.c); got != c.frase {
			t.Errorf("DoCofre(%+v) = %q, quer %q", c.c, got, c.frase)
		}
		if got := Botao(c.c); got != c.botao {
			t.Errorf("Botao(%s) = %q, quer %q", c.frase, got, c.botao)
		}
	}
}

// Nenhuma frase mostra os nomes internos do core.
func TestFrasesNaoMostramNomesDoCore(t *testing.T) {
	for _, e := range []core.EstadoMontagem{core.EstadoDesmontado, core.EstadoMontando, core.EstadoMontado, core.EstadoFalhou} {
		c := cofre(e)
		c.Letra = "V"
		c.Caiu = true
		c.Motivo = core.MotivoProcessoTerminou
		f := DoCofre(c)
		for _, interno := range []core.EstadoMontagem{core.EstadoDesmontado, core.EstadoMontando, core.EstadoMontado, core.EstadoFalhou} {
			if strings.Contains(f, string(interno)) {
				t.Errorf("%q mostra %q", f, interno)
			}
		}
	}
}

func TestTooltipResumeComAsFrasesDoCard(t *testing.T) {
	a := cofre(core.EstadoDesmontado)
	b := cofre(core.EstadoDesmontado)
	c := cofre(core.EstadoMontado)
	c.Letra = "V"

	antes := Tooltip([]core.CofreStatus{a, b})
	depois := Tooltip([]core.CofreStatus{a, b, c})

	if antes != "RuntimeCrypto\nTrancado (2)" {
		t.Errorf("antes = %q", antes)
	}
	if depois != "RuntimeCrypto\nDestrancado • V:\\ (1)\nTrancado (2)" {
		t.Errorf("depois = %q", depois)
	}
	if !strings.Contains(depois, DoCofre(c)) {
		t.Error("o tooltip deveria usar a frase do card")
	}
}

func TestTooltipCabeNoLimiteDoWindows(t *testing.T) {
	var cofres []core.CofreStatus
	for i := 0; i < 20; i++ {
		c := cofre(core.EstadoFalhou)
		c.Motivo = strings.Repeat("motivo longo ", 3) + string(rune('a'+i))
		cofres = append(cofres, c)
	}
	tip := Tooltip(cofres)
	if n := len(utf16.Encode([]rune(tip))); n > 127 {
		t.Errorf("tooltip com %d unidades UTF-16, limite 127", n)
	}
	if !strings.HasSuffix(tip, "…") {
		t.Errorf("tooltip cortado deveria terminar em …: %q", tip)
	}
}

// Demanda 013: fora do Windows toda frase mostra a pasta, nunca `X:\`.
func TestFrasesComPasta(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		t.Fatalf("sem home: %v", err)
	}
	ponto := filepath.Join(home, "RuntimeCrypto", "fotos")
	mostrado := "~" + ponto[len(strings.TrimRight(home, `/\`)):] // ~/RuntimeCrypto/fotos
	if runtime.GOOS != "windows" && mostrado != "~/RuntimeCrypto/fotos" {
		t.Fatalf("pasta mostrada = %q", mostrado)
	}
	com := func(estado core.EstadoMontagem, caiu bool, motivo string) core.CofreStatus {
		c := cofre(estado)
		c.Letra, c.PontoMontagem, c.Caiu, c.Motivo = ponto, ponto, caiu, motivo
		return c
	}

	casos := map[string]core.CofreStatus{
		"Destrancado • " + mostrado:                    com(core.EstadoMontado, false, ""),
		"Caiu: a pasta " + mostrado + " sumiu":         com(core.EstadoFalhou, true, core.MotivoPontoSumiu),
		"Caiu: a pasta " + mostrado + " não respondeu": com(core.EstadoFalhou, true, "ponto de montagem não respondeu em 2s"),
		"Caiu: o rclone parou":                         com(core.EstadoFalhou, true, core.MotivoProcessoTerminou),
	}
	for quer, c := range casos {
		got := DoCofre(c)
		if got != quer {
			t.Errorf("DoCofre = %q, quer %q", got, quer)
		}
		if strings.Contains(got, `:\`) || strings.Contains(got, "unidade") {
			t.Errorf("frase de pasta com cara de unidade: %q", got)
		}
	}
	if tip := Tooltip([]core.CofreStatus{com(core.EstadoMontado, false, "")}); !strings.Contains(tip, "Destrancado • "+mostrado) {
		t.Errorf("tooltip = %q", tip)
	}
}

// No Windows: a unidade, como antes.
func TestFrasesComUnidade(t *testing.T) {
	c := cofre(core.EstadoFalhou)
	c.Letra, c.PontoMontagem, c.Caiu, c.Motivo = "V", `V:\`, true, "ponto de montagem não respondeu em 2s"
	if got := DoCofre(c); got != `Caiu: a unidade V:\ não respondeu` {
		t.Errorf("DoCofre = %q", got)
	}
	c.Estado, c.Caiu, c.Motivo = core.EstadoMontado, false, ""
	if got := DoCofre(c); got != `Destrancado • V:\` {
		t.Errorf("DoCofre = %q", got)
	}
}

// Demanda 025: enquanto o Trancar espera o envio.
func TestFraseEnviando(t *testing.T) {
	c := cofre(core.EstadoMontado)
	c.Letra, c.PontoMontagem, c.Enviando = "V", `V:\`, 3
	if got := DoCofre(c); got != "Enviando 3 arquivos…" {
		t.Errorf("DoCofre = %q", got)
	}
	if tip := Tooltip([]core.CofreStatus{c}); !strings.Contains(tip, "Enviando 3 arquivos…") {
		t.Errorf("tooltip = %q", tip)
	}
	c.Enviando = 1
	if got := DoCofre(c); got != "Enviando 1 arquivo…" {
		t.Errorf("singular: %q", got)
	}
	c.Enviando = 0
	if got := DoCofre(c); got != `Destrancado • V:\` {
		t.Errorf("sem envio: %q", got)
	}
}

// Demanda 026: só o cofre que caiu tem Trancar ao lado de "Destrancar de novo".
func TestBotaoTrancarSoNoCofreQueCaiu(t *testing.T) {
	caiu := core.CofreStatus{Estado: core.EstadoFalhou, Caiu: true}
	naoSubiu := core.CofreStatus{Estado: core.EstadoFalhou}
	montado := core.CofreStatus{Estado: core.EstadoMontado}
	if !BotaoTrancar(caiu) || BotaoTrancar(naoSubiu) || BotaoTrancar(montado) {
		t.Error("Trancar extra só no cofre que caiu")
	}
}

// Demanda 027: o motivo de "não destrancou" que veio do rclone é a frase fixa,
// com o provedor; o texto em inglês não aparece.
func TestNaoDestrancouComFalhaDoRclone(t *testing.T) {
	c := core.CofreStatus{Cofre: core.Cofre{Nome: "a", ProvedorNome: "Google Drive"}, Estado: core.EstadoFalhou,
		Motivo: "sem conexão com o provedor",
		Rclone: &core.ErroRclone{Falha: core.FalhaConexao, Saida: "dial tcp: connection refused"}}
	if got := DoCofre(c); got != "Não destrancou: sem conexão com o Google Drive" {
		t.Errorf("DoCofre = %q", got)
	}
	c.Rclone = &core.ErroRclone{Falha: core.FalhaOutra, Saida: "CRITICAL: cannot find winfsp"}
	if got := DoCofre(c); got != "Não destrancou: o rclone falhou, detalhes no log" {
		t.Errorf("DoCofre = %q", got)
	}
}
