// Package frases (em internal/gui, mas sem Fyne: a bandeja também usa) tem as frases que a tela mostra para o estado de um cofre
// (demanda 018). Card e bandeja usam as mesmas; o core decide o estado e
// este pacote só traduz. Não importa Fyne nem systray.
package frases

import (
	"fmt"
	"strings"
	"unicode/utf16"

	"github.com/eufrauzino/runtime-crypt-go/internal/core"
)

// Frases do estado do cofre.
const (
	Trancado     = "Trancado"
	Destrancando = "Destrancando…"
	// Enviando: o Trancar espera a VFS enviar os arquivos (demanda 025).
	// %s é core.Arquivos: "1 arquivo", "3 arquivos".
	Enviando      = "Enviando %s…"
	DestrancadoEm = "Destrancado • %s"
	// NaoDestrancou: a montagem nunca subiu.
	NaoDestrancou = "Não destrancou: %s"
	// Caiu: a montagem chegou a subir e depois caiu.
	Caiu = "Caiu: %s"

	// Motivos de queda, já na língua da tela.
	MotivoRcloneParou = "o rclone parou"
	// %s é core.RotuloPonto: "a unidade V:\" ou "a pasta ~/RuntimeCrypto/x".
	MotivoPontoSumiu   = "%s sumiu"
	MotivoSemResposta  = "%s não respondeu"
	TextoTentarDeNovo  = "Tentar de novo"
	TextoDestrancar    = "Destrancar"
	TextoTrancar       = "Trancar"
	tituloBandeja      = "RuntimeCrypto"
	tooltipSemCofres   = "RuntimeCrypto — nenhum cofre"
	limiteTooltipUTF16 = 127 // NOTIFYICONDATA.szTip tem 128 posições
)

// Ponto é o ponto de montagem como a tela escreve: `V:\` no Windows,
// `~/RuntimeCrypto/{nome}` fora dele (demanda 013).
func Ponto(c core.CofreStatus) string {
	if c.PontoMontagem != "" {
		return core.TextoPonto(c.PontoMontagem)
	}
	if c.Letra != "" {
		return c.Letra + `:\`
	}
	return ""
}

// Motivo traduz o motivo do core para a tela. Na queda, "o rclone parou" tem
// precedência: o core já devolve só o motivo do processo quando os dois
// aconteceram.
func Motivo(c core.CofreStatus) string {
	if !c.Caiu {
		return c.Motivo
	}
	switch {
	case c.Motivo == core.MotivoProcessoTerminou:
		return MotivoRcloneParou
	case c.Motivo == core.MotivoPontoSumiu:
		return fmt.Sprintf(MotivoPontoSumiu, rotulo(c))
	case strings.HasPrefix(c.Motivo, strings.SplitN(core.MotivoPontoNaoResponde, "%", 2)[0]):
		return fmt.Sprintf(MotivoSemResposta, rotulo(c))
	}
	return c.Motivo
}

// rotulo é "a unidade V:\" ou "a pasta ~/RuntimeCrypto/x".
func rotulo(c core.CofreStatus) string {
	if c.PontoMontagem != "" {
		return core.RotuloPonto(c.PontoMontagem)
	}
	return core.RotuloPonto(Ponto(c))
}

// DoCofre é a frase do estado do cofre, igual no card e na bandeja.
func DoCofre(c core.CofreStatus) string {
	switch c.Estado {
	case core.EstadoMontando:
		return Destrancando
	case core.EstadoMontado:
		if c.Enviando > 0 {
			return fmt.Sprintf(Enviando, core.Arquivos(c.Enviando))
		}
		if p := Ponto(c); p != "" {
			return fmt.Sprintf(DestrancadoEm, p)
		}
		return strings.TrimSuffix(fmt.Sprintf(DestrancadoEm, ""), " • ")
	case core.EstadoFalhou:
		if c.Caiu {
			return fmt.Sprintf(Caiu, Motivo(c))
		}
		return fmt.Sprintf(NaoDestrancou, Motivo(c))
	}
	return Trancado
}

// Botao é o texto do botão do card para o estado.
func Botao(c core.CofreStatus) string {
	switch c.Estado {
	case core.EstadoMontado:
		return TextoTrancar
	case core.EstadoFalhou:
		return TextoTentarDeNovo
	}
	return TextoDestrancar
}

// Tooltip resume quantos cofres há em cada estado, com as frases do card.
// Ex.: "RuntimeCrypto\nDestrancado • V:\ (1)\nTrancado (2)".
func Tooltip(cofres []core.CofreStatus) string {
	if len(cofres) == 0 {
		return tooltipSemCofres
	}
	ordem := []core.EstadoMontagem{core.EstadoMontado, core.EstadoMontando, core.EstadoFalhou, core.EstadoDesmontado}
	var linhas []string
	for _, estado := range ordem {
		contagem := map[string]int{}
		var frasesEmOrdem []string
		for _, c := range cofres {
			if c.Estado != estado {
				continue
			}
			f := DoCofre(c)
			if contagem[f] == 0 {
				frasesEmOrdem = append(frasesEmOrdem, f)
			}
			contagem[f]++
		}
		for _, f := range frasesEmOrdem {
			linhas = append(linhas, fmt.Sprintf("%s (%d)", f, contagem[f]))
		}
	}
	return cortar(tituloBandeja+"\n"+strings.Join(linhas, "\n"), limiteTooltipUTF16)
}

// cortar limita o texto a n unidades UTF-16, terminando em "…" se cortou.
func cortar(texto string, n int) string {
	if len(utf16.Encode([]rune(texto))) <= n {
		return texto
	}
	runas := []rune(texto)
	for len(utf16.Encode(runas))+1 > n {
		runas = runas[:len(runas)-1]
	}
	return string(runas) + "…"
}
