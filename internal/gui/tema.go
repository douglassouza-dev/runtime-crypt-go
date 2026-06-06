package gui

import "image/color"

// Paleta de cores do RuntimeCrypto (tema escuro com acento verde).
var (
	CorBg           = color.NRGBA{R: 13, G: 27, B: 42, A: 255}    // #0d1b2a
	CorCard         = color.NRGBA{R: 27, G: 40, B: 56, A: 255}    // #1b2838
	CorVerde        = color.NRGBA{R: 16, G: 185, B: 129, A: 255}  // #10b981
	CorVerdeHover   = color.NRGBA{R: 13, G: 150, B: 104, A: 255}  // #0d9668
	CorTexto        = color.NRGBA{R: 224, G: 224, B: 224, A: 255} // #e0e0e0
	CorTextoSec     = color.NRGBA{R: 136, G: 153, B: 170, A: 255} // #8899aa
	CorErro         = color.NRGBA{R: 239, G: 68, B: 68, A: 255}   // #ef4444
	CorAviso        = color.NRGBA{R: 245, G: 158, B: 11, A: 255}  // #f59e0b
	CorEntryBg      = color.NRGBA{R: 17, G: 29, B: 46, A: 255}    // #111d2e
	CorBorda        = color.NRGBA{R: 27, G: 58, B: 42, A: 255}    // #1b3a2a
	CorBtnSec       = color.NRGBA{R: 58, G: 74, B: 90, A: 255}    // #3a4a5a
	CorBtnSecHover  = color.NRGBA{R: 74, G: 90, B: 106, A: 255}   // #4a5a6a
	CorMontado      = CorVerde
	CorTrancado     = CorErro
	CorFundoHover   = color.NRGBA{R: 34, G: 51, B: 68, A: 255}    // #223344
)

// CoresProvedores mapeia o ID do provedor para sua cor identificadora.
var CoresProvedores = map[string]color.NRGBA{
	"drive":      {R: 52, G: 168, B: 83, A: 255},   // #34A853
	"onedrive":   {R: 0, G: 120, B: 212, A: 255},    // #0078D4
	"dropbox":    {R: 0, G: 97, B: 255, A: 255},     // #0061FF
	"s3":         {R: 255, G: 153, B: 0, A: 255},    // #FF9900
	"local_path": {R: 16, G: 185, B: 129, A: 255},   // #10b981
}

// ObterCorProvedor retorna a cor de um provedor, ou verde padrão.
func ObterCorProvedor(provedorId string) color.NRGBA {
	if cor, ok := CoresProvedores[provedorId]; ok {
		return cor
	}
	return CorVerde
}
