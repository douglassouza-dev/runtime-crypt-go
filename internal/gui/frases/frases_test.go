package frases

import (
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
	semResposta.Caiu, semResposta.Motivo, semResposta.Letra = true, "ponto de montagem nao respondeu em 2s", "X"

	casos := []struct {
		c            core.CofreStatus
		frase, botao string
	}{
		{cofre(core.EstadoDesmontado), "Trancado", "Destrancar"},
		{cofre(core.EstadoMontando), "Destrancando…", "Destrancar"},
		{montado, `Destrancado • V:\`, "Trancar"},
		{naoSubiu, "Não destrancou: CRITICAL: cannot find winfsp", "Tentar de novo"},
		{parou, "Caiu: o rclone parou", "Tentar de novo"},
		{sumiu, `Caiu: a unidade W:\ sumiu`, "Tentar de novo"},
		{semResposta, `Caiu: a unidade X:\ não respondeu`, "Tentar de novo"},
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
