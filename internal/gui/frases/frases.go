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
	MotivoPontoSumiu  = "%s sumiu"
	MotivoSemResposta = "%s não respondeu"
	TextoTentarDeNovo = "Tentar de novo"
	// TextoDestrancarDeNovo é o botão do cofre que caiu (demanda 026).
	TextoDestrancarDeNovo = "Destrancar de novo"
	TextoDestrancar       = "Destrancar"
	TextoTrancar          = "Trancar"
	tituloBandeja         = "RuntimeCrypto"
	tooltipSemCofres      = "RuntimeCrypto — nenhum cofre"
	limiteTooltipUTF16    = 127 // NOTIFYICONDATA.szTip tem 128 posições
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
		// Demanda 027: falha do rclone vira a frase fixa, com o provedor.
		if c.Rclone != nil {
			return core.TextoFalhaRclone(c.Rclone.Falha, c.ProvedorNome)
		}
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
		if c.Caiu {
			return TextoDestrancarDeNovo
		}
		return TextoTentarDeNovo
	}
	return TextoDestrancar
}

// BotaoTrancar diz se o card tem também o botão Trancar ao lado do
// principal: só no cofre que caiu (demanda 026).
func BotaoTrancar(c core.CofreStatus) bool {
	return c.Estado == core.EstadoFalhou && c.Caiu
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

// Demanda 028: sair com cofre que caiu e arquivos que não subiram.
const (
	TextoEnviarAgora = "Enviar agora"
	TextoSair        = "Sair"
	// TituloSair é o título do diálogo (aprovado pela UI).
	TituloSair      = "Arquivos que ainda não subiram"
	sobemDeNovo     = "Eles sobem quando você destrancar de novo."
	sobeDeNovo      = "Ele sobe quando você destrancar de novo."
	naoTrancouEnvio = "%s: Não trancou: %s"
	naoDestrancouEn = "%s: Não destrancou: %s"
)

// Pendencia é a linha de um cofre: "2 arquivos de X" ou "1 arquivo de X".
// O "ainda não subiram" está no título do diálogo.
func Pendencia(p core.PendenciaCofre) string {
	return fmt.Sprintf("%s de %s", core.Arquivos(p.N), p.Nome)
}

// AvisoAoSair é o texto do diálogo (aprovado pela UI): uma linha por cofre,
// com um cofre ou vários, e no fim "Eles sobem quando você destrancar de
// novo." Só quando há um arquivo ao todo (um cofre, 1 arquivo) o fim vai para
// o singular, "Ele sobe…" (aprovado), pela regra de singular da 026/028.
func AvisoAoSair(ps []core.PendenciaCofre) string {
	linhas := make([]string, 0, len(ps)+1)
	for _, p := range ps {
		linhas = append(linhas, Pendencia(p))
	}
	fim := sobemDeNovo
	if len(ps) == 1 && ps[0].N == 1 {
		fim = sobeDeNovo
	}
	return strings.Join(append(linhas, fim), "\n")
}

// FalhaAoEnviar é a linha de um cofre que "Enviar agora" não trancou, com a
// frase da 022/025.
func FalhaAoEnviar(f core.FalhaEnvio) string {
	if f.Etapa == core.EtapaDestrancar {
		return fmt.Sprintf(naoDestrancouEn, f.Nome, f.Err.Error())
	}
	return fmt.Sprintf(naoTrancouEnvio, f.Nome, f.Err.Error())
}
