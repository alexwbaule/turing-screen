package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// myTheme é a nossa implementação de tema customizado.
type myTheme struct{}

// Garante que myTheme implementa a interface fyne.Theme
var _ fyne.Theme = (*myTheme)(nil)

// NewCustomTheme é o nosso construtor de tema.
func NewCustomTheme() fyne.Theme {
	return &myTheme{}
}

// Color define as cores para os diferentes elementos da UI.
func (t *myTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	// Definimos nossas cores base
	darkBg := color.NRGBA{R: 0x2d, G: 0x2d, B: 0x2d, A: 0xff}
	lightText := color.NRGBA{R: 0xf0, G: 0xf0, B: 0xf0, A: 0xff}
	primaryColor := color.NRGBA{R: 0x00, G: 0x7b, B: 0xff, A: 0xff} // Um azul vibrante

	switch name {
	// Cores de fundo
	case theme.ColorNameBackground:
		return darkBg
	case theme.ColorNameInputBackground:
		return color.NRGBA{R: 0x40, G: 0x40, B: 0x40, A: 0xff}
	case theme.ColorNameMenuBackground:
		return darkBg
	case theme.ColorNameOverlayBackground:
		return color.NRGBA{R: 0x3d, G: 0x3d, B: 0x3d, A: 0xff}

	// Cor do texto principal
	case theme.ColorNameForeground:
		return lightText

	// Cor de destaque (botões, etc.)
	case theme.ColorNamePrimary:
		return primaryColor
	case theme.ColorNameHover:
		return color.NRGBA{R: 0x40, G: 0x40, B: 0x40, A: 0xff} // Hover um pouco mais claro que o fundo

	// Cor dos placeholders e textos secundários
	case theme.ColorNamePlaceHolder:
		return color.NRGBA{R: 0x88, G: 0x88, B: 0x88, A: 0xff}
	case theme.ColorNameDisabled:
		return color.NRGBA{R: 0x55, G: 0x55, B: 0x55, A: 0xff}

	// Cor dos separadores
	case theme.ColorNameSeparator:
		return color.NRGBA{R: 0x55, G: 0x55, B: 0x55, A: 0xff}

	default:
		// Para todas as outras cores, usamos o tema escuro padrão do Fyne
		return theme.DarkTheme().Color(name, variant)
	}
}

// Icon, Font, e Size usarão os padrões do tema escuro.
func (t *myTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DarkTheme().Icon(name)
}
func (t *myTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DarkTheme().Font(style)
}
func (t *myTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DarkTheme().Size(name)
}
