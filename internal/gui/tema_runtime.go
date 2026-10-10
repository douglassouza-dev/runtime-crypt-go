package gui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// TemaRuntime é o tema escuro do RuntimeCrypto para Fyne. Delega ao tema dark
// padrão. Veio de main.go (demanda 017).
type TemaRuntime struct{}

func (t *TemaRuntime) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	return theme.DarkTheme().Color(name, theme.VariantDark)
}

func (t *TemaRuntime) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DarkTheme().Font(style)
}

func (t *TemaRuntime) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DarkTheme().Icon(name)
}

func (t *TemaRuntime) Size(name fyne.ThemeSizeName) float32 {
	return theme.DarkTheme().Size(name)
}
